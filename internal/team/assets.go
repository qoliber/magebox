// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package team

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// AssetClient handles downloading assets from remote storage
type AssetClient struct {
	team     *Team
	sftp     *sftp.Client
	ssh      *ssh.Client
	progress func(DownloadProgress)
}

// NewAssetClient creates a new asset client for a team
func NewAssetClient(team *Team, progressFn func(DownloadProgress)) *AssetClient {
	return &AssetClient{
		team:     team,
		progress: progressFn,
	}
}

// Connect establishes connection to the asset storage
func (a *AssetClient) Connect() error {
	switch a.team.Assets.Provider {
	case AssetSFTP:
		return a.connectSFTP()
	case AssetFTP:
		return fmt.Errorf("FTP support not yet implemented - please use SFTP")
	default:
		return fmt.Errorf("unsupported asset provider: %s", a.team.Assets.Provider)
	}
}

// Close closes the connection
func (a *AssetClient) Close() error {
	if a.sftp != nil {
		a.sftp.Close()
	}
	if a.ssh != nil {
		a.ssh.Close()
	}
	return nil
}

// connectSFTP establishes an SFTP connection
func (a *AssetClient) connectSFTP() error {
	config := a.team.Assets
	port := config.GetDefaultPort()

	// Build SSH config
	sshConfig := &ssh.ClientConfig{
		User:            config.Username,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: proper host key verification
		Timeout:         30 * time.Second,
	}

	// Try SSH agent first (most convenient)
	if agentConn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		agentClient := agent.NewClient(agentConn)
		sshConfig.Auth = append(sshConfig.Auth, ssh.PublicKeysCallback(agentClient.Signers))
	}

	// Try SSH key authentication
	keyPath := a.team.GetAssetKeyPath()
	if key, err := os.ReadFile(keyPath); err == nil {
		signer, err := ssh.ParsePrivateKey(key)
		if err == nil {
			sshConfig.Auth = append(sshConfig.Auth, ssh.PublicKeys(signer))
		}
	}

	// Try password authentication
	if password := a.team.GetAssetPassword(); password != "" {
		sshConfig.Auth = append(sshConfig.Auth, ssh.Password(password))
	}

	if len(sshConfig.Auth) == 0 {
		return fmt.Errorf("no authentication method available - set MAGEBOX_%s_ASSET_KEY or MAGEBOX_%s_ASSET_PASS",
			strings.ToUpper(a.team.Name), strings.ToUpper(a.team.Name))
	}

	// Connect
	addr := fmt.Sprintf("%s:%d", config.Host, port)
	conn, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", addr, err)
	}
	a.ssh = conn

	// Create SFTP client
	client, err := sftp.NewClient(conn)
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to create SFTP client: %w", err)
	}
	a.sftp = client

	return nil
}

// Download downloads a file from the asset storage
func (a *AssetClient) Download(remotePath, localPath string) error {
	if a.sftp == nil {
		return fmt.Errorf("not connected")
	}

	// Build full remote path
	fullRemotePath := filepath.Join(a.team.Assets.Path, remotePath)

	// Open remote file
	remoteFile, err := a.sftp.Open(fullRemotePath)
	if err != nil {
		return fmt.Errorf("failed to open remote file %s: %w", fullRemotePath, err)
	}
	defer remoteFile.Close()

	// Get file info for size
	stat, err := remoteFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat remote file: %w", err)
	}
	totalSize := stat.Size()

	// Ensure local directory exists
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("failed to create local directory: %w", err)
	}

	// Check for existing partial download
	var localFile *os.File
	var startOffset int64

	if existingInfo, err := os.Stat(localPath + ".partial"); err == nil {
		// Resume partial download
		startOffset = existingInfo.Size()
		if startOffset < totalSize {
			localFile, err = os.OpenFile(localPath+".partial", os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				startOffset = 0
			} else {
				// Seek remote file
				if _, err := remoteFile.Seek(startOffset, io.SeekStart); err != nil {
					localFile.Close()
					startOffset = 0
					localFile = nil
				}
			}
		}
	}

	if localFile == nil {
		// Start fresh download
		startOffset = 0
		localFile, err = os.Create(localPath + ".partial")
		if err != nil {
			return fmt.Errorf("failed to create local file: %w", err)
		}
	}
	defer localFile.Close()

	// Create progress tracking writer
	writer := &progressWriter{
		writer:     localFile,
		total:      totalSize,
		downloaded: startOffset,
		startTime:  time.Now(),
		filename:   filepath.Base(remotePath),
		callback:   a.progress,
	}

	// Copy with progress
	_, err = io.Copy(writer, remoteFile)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// Rename partial to final
	localFile.Close()
	if err := os.Rename(localPath+".partial", localPath); err != nil {
		return fmt.Errorf("failed to finalize download: %w", err)
	}

	return nil
}

// FileExists checks if a remote file exists
func (a *AssetClient) FileExists(remotePath string) bool {
	if a.sftp == nil {
		return false
	}
	fullPath := filepath.Join(a.team.Assets.Path, remotePath)
	_, err := a.sftp.Stat(fullPath)
	return err == nil
}

// GetFileSize returns the size of a remote file
func (a *AssetClient) GetFileSize(remotePath string) (int64, error) {
	if a.sftp == nil {
		return 0, fmt.Errorf("not connected")
	}
	fullPath := filepath.Join(a.team.Assets.Path, remotePath)
	info, err := a.sftp.Stat(fullPath)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// progressWriter wraps an io.Writer to track download progress
type progressWriter struct {
	writer     io.Writer
	total      int64
	downloaded int64
	startTime  time.Time
	filename   string
	callback   func(DownloadProgress)
	lastReport time.Time
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n, err := pw.writer.Write(p)
	pw.downloaded += int64(n)

	// Report progress at most every 100ms
	if time.Since(pw.lastReport) > 100*time.Millisecond {
		pw.lastReport = time.Now()
		if pw.callback != nil {
			elapsed := time.Since(pw.startTime).Seconds()
			speed := float64(pw.downloaded) / elapsed
			remaining := float64(pw.total-pw.downloaded) / speed

			pw.callback(DownloadProgress{
				Filename:   pw.filename,
				TotalBytes: pw.total,
				Downloaded: pw.downloaded,
				Speed:      speed,
				Percentage: float64(pw.downloaded) / float64(pw.total) * 100,
				ETA:        formatDuration(time.Duration(remaining) * time.Second),
			})
		}
	}

	return n, err
}

// formatDuration formats a duration as human-readable string
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}

// FormatBytes formats bytes as human-readable string
func FormatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// FormatSpeed formats bytes/second as human-readable string
func FormatSpeed(bytesPerSecond float64) string {
	return FormatBytes(int64(bytesPerSecond)) + "/s"
}

// TestConnection tests the asset storage connection
func (a *AssetClient) TestConnection() error {
	config := a.team.Assets
	port := config.GetDefaultPort()
	addr := fmt.Sprintf("%s:%d", config.Host, port)

	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("cannot reach %s: %w", addr, err)
	}
	conn.Close()
	return nil
}

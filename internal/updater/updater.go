package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	// GitHubOwner is the GitHub repository owner
	GitHubOwner = "qoliber"
	// GitHubRepo is the GitHub repository name
	GitHubRepo = "magebox"
	// GitHubAPIURL is the GitHub API base URL
	GitHubAPIURL = "https://api.github.com"
)

// Updater handles self-update functionality
type Updater struct {
	currentVersion string
	httpClient     *http.Client
}

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	TagName     string         `json:"tag_name"`
	Name        string         `json:"name"`
	Body        string         `json:"body"`
	Draft       bool           `json:"draft"`
	Prerelease  bool           `json:"prerelease"`
	PublishedAt string         `json:"published_at"`
	Assets      []ReleaseAsset `json:"assets"`
}

// ReleaseAsset represents a release asset (binary file)
type ReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
	ContentType        string `json:"content_type"`
}

// UpdateResult contains the result of an update check
type UpdateResult struct {
	CurrentVersion  string
	LatestVersion   string
	UpdateAvailable bool
	ReleaseNotes    string
	DownloadURL     string
	AssetName       string
}

// NewUpdater creates a new updater instance
func NewUpdater(currentVersion string) *Updater {
	return &Updater{
		currentVersion: currentVersion,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CheckForUpdate checks if a newer version is available
func (u *Updater) CheckForUpdate() (*UpdateResult, error) {
	release, err := u.getLatestRelease()
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %w", err)
	}

	result := &UpdateResult{
		CurrentVersion:  u.currentVersion,
		LatestVersion:   release.TagName,
		UpdateAvailable: u.isNewerVersion(release.TagName),
		ReleaseNotes:    release.Body,
	}

	// Find the appropriate asset for this platform
	assetName := u.getAssetName()
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			result.DownloadURL = asset.BrowserDownloadURL
			result.AssetName = asset.Name
			break
		}
	}

	return result, nil
}

// Update downloads and installs the latest version
func (u *Updater) Update(result *UpdateResult) error {
	if result.DownloadURL == "" {
		return fmt.Errorf("no download URL available for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Resolve symlinks
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	// Download new binary to temp file
	tmpFile, err := u.downloadBinary(result.DownloadURL)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}
	defer os.Remove(tmpFile)

	// Make temp file executable
	if err := os.Chmod(tmpFile, 0755); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	// Check if we can write to the install directory
	installDir := filepath.Dir(execPath)
	needsSudo := !isWritable(installDir)

	if needsSudo {
		// Use sudo to install
		return u.installWithSudo(tmpFile, execPath)
	}

	// Direct install without sudo
	backupPath := execPath + ".backup"
	if err := os.Rename(execPath, backupPath); err != nil {
		return fmt.Errorf("failed to backup current binary: %w", err)
	}

	if err := os.Rename(tmpFile, execPath); err != nil {
		// Try to restore backup
		_ = os.Rename(backupPath, execPath)
		return fmt.Errorf("failed to install new binary: %w", err)
	}

	// Remove backup
	_ = os.Remove(backupPath)

	return nil
}

// isWritable checks if a directory is writable by the current user
func isWritable(path string) bool {
	testFile := filepath.Join(path, ".magebox-write-test")
	f, err := os.Create(testFile)
	if err != nil {
		return false
	}
	f.Close()
	os.Remove(testFile)
	return true
}

// installWithSudo installs the binary using sudo
func (u *Updater) installWithSudo(tmpFile, execPath string) error {
	backupPath := execPath + ".backup"

	// Backup current binary with sudo
	cmd := exec.Command("sudo", "mv", execPath, backupPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to backup current binary: %w", err)
	}

	// Move new binary into place with sudo
	cmd = exec.Command("sudo", "mv", tmpFile, execPath)
	if err := cmd.Run(); err != nil {
		// Try to restore backup
		_ = exec.Command("sudo", "mv", backupPath, execPath).Run()
		return fmt.Errorf("failed to install new binary: %w", err)
	}

	// Set permissions with sudo
	cmd = exec.Command("sudo", "chmod", "+x", execPath)
	_ = cmd.Run()

	// Remove backup with sudo
	_ = exec.Command("sudo", "rm", "-f", backupPath).Run()

	return nil
}

// getLatestRelease fetches the latest release from GitHub
func (u *Updater) getLatestRelease() (*GitHubRelease, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", GitHubAPIURL, GitHubOwner, GitHubRepo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "MageBox-Updater")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no releases found")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse release data: %w", err)
	}

	return &release, nil
}

// downloadBinary downloads a binary and returns the temp file path
func (u *Updater) downloadBinary(url string) (string, error) {
	resp, err := u.httpClient.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Create temp file
	tmpFile, err := os.CreateTemp("", "magebox-update-*")
	if err != nil {
		return "", err
	}

	// Copy download to temp file
	_, err = io.Copy(tmpFile, resp.Body)
	tmpFile.Close()

	if err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}

// isNewerVersion checks if the given version is newer than current
func (u *Updater) isNewerVersion(version string) bool {
	// Strip 'v' prefix if present
	current := strings.TrimPrefix(u.currentVersion, "v")
	latest := strings.TrimPrefix(version, "v")

	// Parse and compare version parts
	currentParts := parseVersion(current)
	latestParts := parseVersion(latest)

	// Compare each part
	for i := 0; i < len(currentParts) || i < len(latestParts); i++ {
		var c, l int
		if i < len(currentParts) {
			c = currentParts[i]
		}
		if i < len(latestParts) {
			l = latestParts[i]
		}

		if l > c {
			return true
		}
		if l < c {
			return false
		}
	}

	return false // versions are equal
}

// parseVersion parses a version string into numeric parts
func parseVersion(version string) []int {
	parts := strings.Split(version, ".")
	result := make([]int, len(parts))

	for i, p := range parts {
		// Extract numeric part (ignore suffixes like -beta)
		numStr := strings.Split(p, "-")[0]
		var num int
		_, _ = fmt.Sscanf(numStr, "%d", &num)
		result[i] = num
	}

	return result
}

// getAssetName returns the expected asset name for this platform
func (u *Updater) getAssetName() string {
	os := runtime.GOOS
	arch := runtime.GOARCH

	// Map Go arch names to common release names
	if arch == "amd64" {
		arch = "amd64"
	} else if arch == "arm64" {
		arch = "arm64"
	}

	return fmt.Sprintf("magebox-%s-%s", os, arch)
}

// GetPlatformInfo returns info about the current platform
func GetPlatformInfo() string {
	return fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
}

// ListReleases lists recent releases
func (u *Updater) ListReleases(limit int) ([]GitHubRelease, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases?per_page=%d", GitHubAPIURL, GitHubOwner, GitHubRepo, limit)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "MageBox-Updater")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var releases []GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to parse releases: %w", err)
	}

	return releases, nil
}

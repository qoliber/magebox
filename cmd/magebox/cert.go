/**
 * Created by Qoliber
 *
 * @category    Qoliber
 * @package     MageBox
 * @author      Jakub Winkler <jwinkler@qoliber.com>
 */

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"qoliber/magebox/internal/cli"
	"qoliber/magebox/internal/teamserver"
)

var (
	certQuiet bool
)

var certCmd = &cobra.Command{
	Use:   "cert",
	Short: "Manage SSH certificates",
	Long: `Manage SSH certificates for team server authentication.

Certificates are time-limited (default 24 hours) and must be renewed periodically.
When a certificate expires, you'll need to renew it before SSH connections work.

Examples:
  magebox cert renew              # Renew certificate
  magebox cert show               # Show certificate info
  magebox cert expiry             # Show when certificate expires`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var certRenewCmd = &cobra.Command{
	Use:   "renew",
	Short: "Renew SSH certificate",
	Long: `Renew your SSH certificate from the team server.

The certificate is automatically saved to ~/.magebox/keys/<server>-cert.pub

Examples:
  magebox cert renew
  magebox cert renew --quiet    # Quiet mode (for scripts/cron)`,
	RunE: runCertRenew,
}

var certShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show certificate information",
	Long:  `Display information about your current SSH certificate.`,
	RunE:  runCertShow,
}

var certExpiryCmd = &cobra.Command{
	Use:   "expiry",
	Short: "Show certificate expiry",
	Long:  `Show when your SSH certificate expires.`,
	RunE:  runCertExpiry,
}

func init() {
	certRenewCmd.Flags().BoolVar(&certQuiet, "quiet", false, "Quiet mode (suppress output)")

	certCmd.AddCommand(certRenewCmd)
	certCmd.AddCommand(certShowCmd)
	certCmd.AddCommand(certExpiryCmd)
	rootCmd.AddCommand(certCmd)
}

func runCertRenew(cmd *cobra.Command, args []string) error {
	// Load team server config
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".magebox", "teamserver", "client.json")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("not connected to a team server. Run 'magebox server join' first")
		}
		return fmt.Errorf("failed to read config: %w", err)
	}

	var config struct {
		ServerURL    string `json:"server_url"`
		SessionToken string `json:"session_token"`
		KeyFile      string `json:"key_file"`
		KeyPath      string `json:"key_path"` // Fallback for older configs
	}
	if err := json.Unmarshal(configData, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Use KeyPath as fallback for KeyFile
	keyFile := config.KeyFile
	if keyFile == "" {
		keyFile = config.KeyPath
	}
	if keyFile == "" {
		return fmt.Errorf("no key file found in config. Re-join the team server")
	}

	// Request certificate renewal
	req, err := http.NewRequest("POST", config.ServerURL+"/api/cert/renew", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+config.SessionToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to contact server: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		var errResp teamserver.ErrorResponse
		if json.Unmarshal(body, &errResp) == nil {
			return fmt.Errorf("certificate renewal failed: %s", errResp.Error)
		}
		return fmt.Errorf("certificate renewal failed: %s", resp.Status)
	}

	var certResp teamserver.CertRenewResponse
	if err := json.Unmarshal(body, &certResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Save certificate to file
	certFile := keyFile + "-cert.pub"
	if err := os.WriteFile(certFile, []byte(certResp.Certificate+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to save certificate: %w", err)
	}

	if !certQuiet {
		cli.PrintSuccess("Certificate renewed successfully!")
		cli.PrintInfo("Valid until: %s", certResp.ValidUntil.Format("2006-01-02 15:04:05"))
		cli.PrintInfo("Principals: %v", certResp.Principals)
		cli.PrintInfo("Saved to: %s", certFile)
	}

	return nil
}

func runCertShow(cmd *cobra.Command, args []string) error {
	// Load team server config
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".magebox", "teamserver", "client.json")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("not connected to a team server. Run 'magebox server join' first")
		}
		return fmt.Errorf("failed to read config: %w", err)
	}

	var config struct {
		KeyFile string `json:"key_file"`
		KeyPath string `json:"key_path"`
	}
	if err := json.Unmarshal(configData, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	keyFile := config.KeyFile
	if keyFile == "" {
		keyFile = config.KeyPath
	}
	if keyFile == "" {
		return fmt.Errorf("no key file found in config")
	}

	// Read certificate file
	certFile := keyFile + "-cert.pub"
	certData, err := os.ReadFile(certFile)
	if err != nil {
		if os.IsNotExist(err) {
			cli.PrintWarning("No certificate found at %s", certFile)
			cli.PrintInfo("Run 'magebox cert renew' to get a new certificate")
			return nil
		}
		return fmt.Errorf("failed to read certificate: %w", err)
	}

	cli.PrintTitle("SSH Certificate")
	fmt.Println()

	// Try to parse and display certificate info using ssh-keygen if available
	// For now, just show the file info
	info, err := os.Stat(certFile)
	if err == nil {
		cli.PrintInfo("File: %s", certFile)
		cli.PrintInfo("Size: %d bytes", info.Size())
		cli.PrintInfo("Modified: %s", info.ModTime().Format("2006-01-02 15:04:05"))
	}

	// Show first line of certificate
	lines := bytes.Split(certData, []byte("\n"))
	if len(lines) > 0 {
		certType := "unknown"
		if bytes.HasPrefix(lines[0], []byte("ssh-ed25519-cert-v01@openssh.com")) {
			certType = "ssh-ed25519-cert-v01@openssh.com (Ed25519 certificate)"
		} else if bytes.HasPrefix(lines[0], []byte("ssh-rsa-cert-v01@openssh.com")) {
			certType = "ssh-rsa-cert-v01@openssh.com (RSA certificate)"
		}
		cli.PrintInfo("Type: %s", certType)
	}

	fmt.Println()
	cli.PrintInfo("Run 'ssh-keygen -L -f %s' for detailed certificate information", certFile)

	return nil
}

func runCertExpiry(cmd *cobra.Command, args []string) error {
	// Load team server config
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".magebox", "teamserver", "client.json")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("not connected to a team server. Run 'magebox server join' first")
		}
		return fmt.Errorf("failed to read config: %w", err)
	}

	var config struct {
		ServerURL    string `json:"server_url"`
		SessionToken string `json:"session_token"`
	}
	if err := json.Unmarshal(configData, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Request certificate info
	req, err := http.NewRequest("GET", config.ServerURL+"/api/cert/info", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+config.SessionToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to contact server: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		var errResp teamserver.ErrorResponse
		if json.Unmarshal(body, &errResp) == nil {
			return fmt.Errorf("failed to get certificate info: %s", errResp.Error)
		}
		return fmt.Errorf("failed to get certificate info: %s", resp.Status)
	}

	var infoResp teamserver.CertInfoResponse
	if err := json.Unmarshal(body, &infoResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !infoResp.HasCertificate {
		cli.PrintWarning("No valid certificate")
		if infoResp.IsExpired {
			cli.PrintInfo("Run 'magebox cert renew' to get a new certificate")
		}
		return nil
	}

	cli.PrintTitle("Certificate Status")
	fmt.Println()

	if infoResp.ValidUntil != nil {
		now := time.Now()
		if now.After(*infoResp.ValidUntil) {
			cli.PrintError("Certificate EXPIRED at %s", infoResp.ValidUntil.Format("2006-01-02 15:04:05"))
			cli.PrintInfo("Run 'magebox cert renew' to get a new certificate")
		} else {
			remaining := infoResp.ValidUntil.Sub(now)
			cli.PrintSuccess("Certificate valid until %s", infoResp.ValidUntil.Format("2006-01-02 15:04:05"))
			cli.PrintInfo("Expires in: %s", formatDuration(remaining))
		}
	}

	if len(infoResp.Principals) > 0 {
		cli.PrintInfo("Principals: %v", infoResp.Principals)
	}

	return nil
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d seconds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%d minutes", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		return fmt.Sprintf("%d hours %d minutes", hours, minutes)
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	return fmt.Sprintf("%d days %d hours", days, hours)
}

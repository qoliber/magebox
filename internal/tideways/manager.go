// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package tideways

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"qoliber/magebox/internal/platform"
	"qoliber/magebox/internal/xdebug"
)

// Manager handles Tideways installation and configuration
type Manager struct {
	platform    *platform.Platform
	xdebugMgr   *xdebug.Manager
	credentials *Credentials
}

// NewManager creates a new Tideways manager
func NewManager(p *platform.Platform, creds *Credentials) *Manager {
	return &Manager{
		platform:    p,
		xdebugMgr:   xdebug.NewManager(p),
		credentials: creds,
	}
}

// IsDaemonInstalled checks if Tideways daemon is installed
func (m *Manager) IsDaemonInstalled() bool {
	return platform.CommandExists("tideways-daemon")
}

// IsDaemonRunning checks if Tideways daemon is running
func (m *Manager) IsDaemonRunning() bool {
	switch m.platform.Type {
	case platform.Darwin:
		cmd := exec.Command("pgrep", "-f", "tideways-daemon")
		return cmd.Run() == nil
	case platform.Linux:
		cmd := exec.Command("systemctl", "is-active", "tideways-daemon")
		output, err := cmd.Output()
		if err != nil {
			return false
		}
		return strings.TrimSpace(string(output)) == "active"
	}
	return false
}

// IsExtensionInstalled checks if Tideways PHP extension is installed for a version
func (m *Manager) IsExtensionInstalled(phpVersion string) bool {
	iniPath := m.getExtensionIniPath(phpVersion)
	if iniPath == "" {
		return false
	}
	_, err := os.Stat(iniPath)
	return err == nil
}

// IsExtensionEnabled checks if Tideways PHP extension is enabled for a version
func (m *Manager) IsExtensionEnabled(phpVersion string) bool {
	iniPath := m.getExtensionIniPath(phpVersion)
	if iniPath == "" {
		return false
	}

	content, err := os.ReadFile(iniPath)
	if err != nil {
		return false
	}

	// Check if extension line is not commented out
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "extension=tideways") {
			return true
		}
	}
	return false
}

// GetStatus returns the current Tideways status
func (m *Manager) GetStatus(phpVersions []string) *Status {
	status := &Status{
		DaemonInstalled:    m.IsDaemonInstalled(),
		DaemonRunning:      m.IsDaemonRunning(),
		ExtensionInstalled: make(map[string]bool),
		ExtensionEnabled:   make(map[string]bool),
		Configured:         m.credentials != nil && m.credentials.APIKey != "",
	}

	for _, v := range phpVersions {
		status.ExtensionInstalled[v] = m.IsExtensionInstalled(v)
		status.ExtensionEnabled[v] = m.IsExtensionEnabled(v)
	}

	return status
}

// Enable enables Tideways for a PHP version (and disables Xdebug)
func (m *Manager) Enable(phpVersion string) error {
	if !m.IsExtensionInstalled(phpVersion) {
		return fmt.Errorf("tideways extension not installed for PHP %s", phpVersion)
	}

	// Disable Xdebug first to avoid conflicts (non-fatal if it fails)
	if m.xdebugMgr.IsEnabled(phpVersion) {
		_ = m.xdebugMgr.Disable(phpVersion) // Ignore errors - xdebug may not have ini file
	}

	iniPath := m.getExtensionIniPath(phpVersion)

	// Use sudo sed to uncomment extension line (same approach as xdebug manager)
	args := m.sedInPlaceArgs(`s/^;\(extension=tideways.*\)$/\1/`, iniPath)
	cmd := exec.Command("sudo", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable tideways: %w", err)
	}

	return nil
}

// Disable disables Tideways for a PHP version
func (m *Manager) Disable(phpVersion string) error {
	iniPath := m.getExtensionIniPath(phpVersion)
	if iniPath == "" {
		return nil
	}

	// Use sudo sed to comment out extension line (same approach as xdebug manager)
	args := m.sedInPlaceArgs(`s/^\(extension=tideways.*\)$/;\1/`, iniPath)
	cmd := exec.Command("sudo", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to disable tideways: %w", err)
	}

	return nil
}

// StartDaemon starts the Tideways daemon
func (m *Manager) StartDaemon() error {
	switch m.platform.Type {
	case platform.Darwin:
		cmd := exec.Command("brew", "services", "start", "tideways-daemon")
		return cmd.Run()
	case platform.Linux:
		cmd := exec.Command("sudo", "systemctl", "start", "tideways-daemon")
		return cmd.Run()
	}
	return fmt.Errorf("unsupported platform")
}

// StopDaemon stops the Tideways daemon
func (m *Manager) StopDaemon() error {
	switch m.platform.Type {
	case platform.Darwin:
		cmd := exec.Command("brew", "services", "stop", "tideways-daemon")
		return cmd.Run()
	case platform.Linux:
		cmd := exec.Command("sudo", "systemctl", "stop", "tideways-daemon")
		return cmd.Run()
	}
	return fmt.Errorf("unsupported platform")
}

// RestartDaemon restarts the Tideways daemon
func (m *Manager) RestartDaemon() error {
	switch m.platform.Type {
	case platform.Darwin:
		cmd := exec.Command("brew", "services", "restart", "tideways-daemon")
		return cmd.Run()
	case platform.Linux:
		cmd := exec.Command("sudo", "systemctl", "restart", "tideways-daemon")
		return cmd.Run()
	}
	return fmt.Errorf("unsupported platform")
}

// WriteExtensionConfig writes the tideways.api_key and tideways.environment
// directives to the PHP extension ini file for the given PHP version. Existing
// lines for these directives (commented or uncommented) are replaced in place;
// otherwise they are appended. The Tideways PHP extension requires
// tideways.api_key to be set in php.ini in order to transmit traces, and
// tideways.environment controls which Tideways environment bucket the traces
// land in — on a developer machine we don't want that to default to the
// server-side `production` bucket.
func (m *Manager) WriteExtensionConfig(phpVersion string) error {
	if m.credentials == nil || m.credentials.APIKey == "" {
		return fmt.Errorf("tideways API key not configured")
	}

	iniPath := m.getExtensionIniPath(phpVersion)
	if iniPath == "" {
		return fmt.Errorf("unsupported platform")
	}

	if _, err := os.Stat(iniPath); err != nil {
		return fmt.Errorf("tideways extension ini not found at %s (install the extension first)", iniPath)
	}

	existing, err := os.ReadFile(iniPath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", iniPath, err)
	}

	newContent := rewriteIniDirective(string(existing), "tideways.api_key", m.credentials.APIKey)
	if m.credentials.Environment != "" {
		newContent = rewriteIniDirective(newContent, "tideways.environment", m.credentials.Environment)
	}

	// Write through sudo tee because mods-available files are root-owned.
	cmd := exec.Command("sudo", "tee", iniPath)
	cmd.Stdin = strings.NewReader(newContent)
	cmd.Stdout = nil
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to write tideways config to %s: %w", iniPath, err)
	}

	return nil
}

// rewriteIniDirective returns a copy of existing with the given ini directive
// set to value. An existing line for the directive (commented or uncommented)
// is replaced in place; otherwise the directive is appended. The result is
// guaranteed to end with exactly one trailing newline.
func rewriteIniDirective(existing, directive, value string) string {
	newLine := fmt.Sprintf("%s=%s", directive, value)

	lines := strings.Split(existing, "\n")
	replaced := false
	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " \t;")
		if strings.HasPrefix(trimmed, directive) {
			lines[i] = newLine
			replaced = true
		}
	}
	if !replaced {
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			// Reuse the trailing empty element so we end up with exactly one
			// newline at the end of the file.
			lines[len(lines)-1] = newLine
		} else {
			lines = append(lines, newLine)
		}
	}
	out := strings.Join(lines, "\n")
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return out
}

// ImportCLIToken imports the access token for the `tideways` CLI command by
// running `tideways import <token>`. This is a separate credential from the
// PHP extension API key — it is used by the commandline tool for operations
// like `tideways run`, `tideways event create`, and `tideways tracepoint
// create`. See https://app.tideways.io/user/cli-import-settings.
func (m *Manager) ImportCLIToken() error {
	if m.credentials == nil || m.credentials.AccessToken == "" {
		return fmt.Errorf("tideways CLI access token not configured")
	}

	if !platform.CommandExists("tideways") {
		return fmt.Errorf("tideways CLI not found in PATH")
	}

	cmd := exec.Command("tideways", "import", m.credentials.AccessToken)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to import tideways CLI token: %w", err)
	}
	return nil
}

// sedInPlaceArgs returns the correct sed arguments for in-place editing on the current platform.
// macOS BSD sed requires "sed -i ” <expr> <file>" while GNU sed uses "sed -i <expr> <file>".
func (m *Manager) sedInPlaceArgs(expr, file string) []string {
	if m.platform.Type == platform.Darwin {
		return []string{"sed", "-i", "", expr, file}
	}
	return []string{"sed", "-i", expr, file}
}

// getExtensionIniPath returns the path to the Tideways PHP extension ini file
func (m *Manager) getExtensionIniPath(phpVersion string) string {
	switch m.platform.Type {
	case platform.Darwin:
		base := "/usr/local"
		if m.platform.IsAppleSilicon {
			base = "/opt/homebrew"
		}
		confDir := filepath.Join(base, "etc", "php", phpVersion, "conf.d")
		patterns := []string{
			filepath.Join(confDir, "ext-tideways.ini"),
			filepath.Join(confDir, "tideways.ini"),
			filepath.Join(confDir, "zz-tideways.ini"),
		}
		for _, p := range patterns {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
		return filepath.Join(confDir, "ext-tideways.ini")

	case platform.Linux:
		remiVersion := strings.ReplaceAll(phpVersion, ".", "")
		switch m.platform.LinuxDistro {
		case platform.DistroFedora:
			return fmt.Sprintf("/etc/opt/remi/php%s/php.d/zz-tideways.ini", remiVersion)
		default:
			return fmt.Sprintf("/etc/php/%s/mods-available/tideways.ini", phpVersion)
		}
	}
	return ""
}

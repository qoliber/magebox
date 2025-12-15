// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package tideways

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/qoliber/magebox/internal/platform"
	"github.com/qoliber/magebox/internal/xdebug"
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
	lines := strings.Split(string(content), "\\n")
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
	content, err := os.ReadFile(iniPath)
	if err != nil {
		return fmt.Errorf("failed to read tideways ini: %w", err)
	}

	// Uncomment extension line if commented
	lines := strings.Split(string(content), "\\n")
	modified := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, ";extension=tideways") {
			lines[i] = strings.TrimPrefix(trimmed, ";")
			modified = true
		}
	}

	if modified {
		if err := os.WriteFile(iniPath, []byte(strings.Join(lines, "\\n")), 0644); err != nil {
			return fmt.Errorf("failed to write tideways ini: %w", err)
		}
	}

	return nil
}

// Disable disables Tideways for a PHP version
func (m *Manager) Disable(phpVersion string) error {
	iniPath := m.getExtensionIniPath(phpVersion)
	if iniPath == "" {
		return nil
	}

	content, err := os.ReadFile(iniPath)
	if err != nil {
		return nil
	}

	// Comment out extension line
	lines := strings.Split(string(content), "\\n")
	modified := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "extension=tideways") {
			lines[i] = ";" + trimmed
			modified = true
		}
	}

	if modified {
		if err := os.WriteFile(iniPath, []byte(strings.Join(lines, "\\n")), 0644); err != nil {
			return fmt.Errorf("failed to write tideways ini: %w", err)
		}
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

// ConfigureDaemon configures the Tideways daemon with API key
func (m *Manager) ConfigureDaemon() error {
	if m.credentials == nil || m.credentials.APIKey == "" {
		return fmt.Errorf("tideways credentials not configured")
	}

	configPath := m.getDaemonConfigPath()
	if configPath == "" {
		return fmt.Errorf("unsupported platform")
	}

	// Create config directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write daemon config
	config := fmt.Sprintf("api_key=%s\\n", m.credentials.APIKey)
	if err := os.WriteFile(configPath, []byte(config), 0600); err != nil {
		return fmt.Errorf("failed to write daemon config: %w", err)
	}

	return nil
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

// getDaemonConfigPath returns the path to the Tideways daemon config
func (m *Manager) getDaemonConfigPath() string {
	switch m.platform.Type {
	case platform.Darwin:
		base := "/usr/local"
		if m.platform.IsAppleSilicon {
			base = "/opt/homebrew"
		}
		return filepath.Join(base, "etc", "tideways", "daemon.conf")
	case platform.Linux:
		return "/etc/tideways/daemon.conf"
	}
	return ""
}

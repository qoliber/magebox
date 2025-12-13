// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package blackfire

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/qoliber/magebox/internal/platform"
	"github.com/qoliber/magebox/internal/xdebug"
)

// Manager handles Blackfire installation and configuration
type Manager struct {
	platform    *platform.Platform
	xdebugMgr   *xdebug.Manager
	credentials *Credentials
}

// NewManager creates a new Blackfire manager
func NewManager(p *platform.Platform, creds *Credentials) *Manager {
	return &Manager{
		platform:    p,
		xdebugMgr:   xdebug.NewManager(p),
		credentials: creds,
	}
}

// IsAgentInstalled checks if Blackfire agent is installed
func (m *Manager) IsAgentInstalled() bool {
	return platform.CommandExists("blackfire-agent") || platform.CommandExists("blackfire")
}

// IsAgentRunning checks if Blackfire agent is running
func (m *Manager) IsAgentRunning() bool {
	switch m.platform.Type {
	case platform.Darwin:
		cmd := exec.Command("pgrep", "-f", "blackfire-agent")
		return cmd.Run() == nil
	case platform.Linux:
		cmd := exec.Command("systemctl", "is-active", "blackfire-agent")
		output, err := cmd.Output()
		if err != nil {
			return false
		}
		return strings.TrimSpace(string(output)) == "active"
	}
	return false
}

// IsExtensionInstalled checks if Blackfire PHP extension is installed for a version
func (m *Manager) IsExtensionInstalled(phpVersion string) bool {
	iniPath := m.getExtensionIniPath(phpVersion)
	if iniPath == "" {
		return false
	}
	_, err := os.Stat(iniPath)
	return err == nil
}

// IsExtensionEnabled checks if Blackfire PHP extension is enabled for a version
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
		if strings.HasPrefix(trimmed, "extension=blackfire") {
			return true
		}
	}
	return false
}

// GetStatus returns the current Blackfire status
func (m *Manager) GetStatus(phpVersions []string) *Status {
	status := &Status{
		AgentInstalled:     m.IsAgentInstalled(),
		AgentRunning:       m.IsAgentRunning(),
		ExtensionInstalled: make(map[string]bool),
		ExtensionEnabled:   make(map[string]bool),
		Configured:         m.credentials != nil && m.credentials.ServerID != "",
	}

	for _, v := range phpVersions {
		status.ExtensionInstalled[v] = m.IsExtensionInstalled(v)
		status.ExtensionEnabled[v] = m.IsExtensionEnabled(v)
	}

	return status
}

// Enable enables Blackfire for a PHP version (and disables Xdebug)
func (m *Manager) Enable(phpVersion string) error {
	if !m.IsExtensionInstalled(phpVersion) {
		return fmt.Errorf("blackfire extension not installed for PHP %s", phpVersion)
	}

	// Disable Xdebug first to avoid conflicts
	if m.xdebugMgr.IsEnabled(phpVersion) {
		if err := m.xdebugMgr.Disable(phpVersion); err != nil {
			return fmt.Errorf("failed to disable xdebug: %w", err)
		}
	}

	iniPath := m.getExtensionIniPath(phpVersion)
	content, err := os.ReadFile(iniPath)
	if err != nil {
		return fmt.Errorf("failed to read blackfire ini: %w", err)
	}

	// Uncomment extension line if commented
	lines := strings.Split(string(content), "\n")
	modified := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, ";extension=blackfire") {
			lines[i] = strings.TrimPrefix(trimmed, ";")
			modified = true
		}
	}

	if modified {
		if err := os.WriteFile(iniPath, []byte(strings.Join(lines, "\n")), 0644); err != nil {
			return fmt.Errorf("failed to write blackfire ini: %w", err)
		}
	}

	return nil
}

// Disable disables Blackfire for a PHP version
func (m *Manager) Disable(phpVersion string) error {
	iniPath := m.getExtensionIniPath(phpVersion)
	if iniPath == "" {
		return nil // Not installed, nothing to disable
	}

	content, err := os.ReadFile(iniPath)
	if err != nil {
		return nil // File doesn't exist
	}

	// Comment out extension line
	lines := strings.Split(string(content), "\n")
	modified := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "extension=blackfire") {
			lines[i] = ";" + trimmed
			modified = true
		}
	}

	if modified {
		if err := os.WriteFile(iniPath, []byte(strings.Join(lines, "\n")), 0644); err != nil {
			return fmt.Errorf("failed to write blackfire ini: %w", err)
		}
	}

	return nil
}

// StartAgent starts the Blackfire agent
func (m *Manager) StartAgent() error {
	switch m.platform.Type {
	case platform.Darwin:
		cmd := exec.Command("brew", "services", "start", "blackfire")
		return cmd.Run()
	case platform.Linux:
		cmd := exec.Command("sudo", "systemctl", "start", "blackfire-agent")
		return cmd.Run()
	}
	return fmt.Errorf("unsupported platform")
}

// StopAgent stops the Blackfire agent
func (m *Manager) StopAgent() error {
	switch m.platform.Type {
	case platform.Darwin:
		cmd := exec.Command("brew", "services", "stop", "blackfire")
		return cmd.Run()
	case platform.Linux:
		cmd := exec.Command("sudo", "systemctl", "stop", "blackfire-agent")
		return cmd.Run()
	}
	return fmt.Errorf("unsupported platform")
}

// RestartAgent restarts the Blackfire agent
func (m *Manager) RestartAgent() error {
	switch m.platform.Type {
	case platform.Darwin:
		cmd := exec.Command("brew", "services", "restart", "blackfire")
		return cmd.Run()
	case platform.Linux:
		cmd := exec.Command("sudo", "systemctl", "restart", "blackfire-agent")
		return cmd.Run()
	}
	return fmt.Errorf("unsupported platform")
}

// ConfigureAgent configures the Blackfire agent with credentials
func (m *Manager) ConfigureAgent() error {
	if m.credentials == nil || m.credentials.ServerID == "" {
		return fmt.Errorf("blackfire credentials not configured")
	}

	// Use blackfire-agent -register for configuration
	cmd := exec.Command("blackfire-agent", "-register")
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("BLACKFIRE_SERVER_ID=%s", m.credentials.ServerID),
		fmt.Sprintf("BLACKFIRE_SERVER_TOKEN=%s", m.credentials.ServerToken),
	)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// ConfigureCLI configures the Blackfire CLI with client credentials
func (m *Manager) ConfigureCLI() error {
	if m.credentials == nil || m.credentials.ClientID == "" {
		return fmt.Errorf("blackfire client credentials not configured")
	}

	cmd := exec.Command("blackfire", "client:config",
		"--client-id", m.credentials.ClientID,
		"--client-token", m.credentials.ClientToken,
	)
	return cmd.Run()
}

// getExtensionIniPath returns the path to the Blackfire PHP extension ini file
func (m *Manager) getExtensionIniPath(phpVersion string) string {
	switch m.platform.Type {
	case platform.Darwin:
		base := "/usr/local"
		if m.platform.IsAppleSilicon {
			base = "/opt/homebrew"
		}
		confDir := filepath.Join(base, "etc", "php", phpVersion, "conf.d")
		patterns := []string{
			filepath.Join(confDir, "ext-blackfire.ini"),
			filepath.Join(confDir, "blackfire.ini"),
			filepath.Join(confDir, "zz-blackfire.ini"),
		}
		for _, p := range patterns {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
		// Return default path for creation
		return filepath.Join(confDir, "ext-blackfire.ini")

	case platform.Linux:
		remiVersion := strings.ReplaceAll(phpVersion, ".", "")
		switch m.platform.LinuxDistro {
		case platform.DistroFedora:
			// Remi repository path
			return fmt.Sprintf("/etc/opt/remi/php%s/php.d/zz-blackfire.ini", remiVersion)
		default:
			// Debian/Ubuntu path
			return fmt.Sprintf("/etc/php/%s/mods-available/blackfire.ini", phpVersion)
		}
	}
	return ""
}

// PromptCredentials prompts the user for Blackfire credentials
func PromptCredentials() (*Credentials, error) {
	reader := bufio.NewReader(os.Stdin)
	creds := &Credentials{}

	fmt.Println("Enter your Blackfire credentials (from https://blackfire.io/my/settings/credentials)")
	fmt.Println()

	fmt.Print("Server ID: ")
	serverID, _ := reader.ReadString('\n')
	creds.ServerID = strings.TrimSpace(serverID)

	fmt.Print("Server Token: ")
	serverToken, _ := reader.ReadString('\n')
	creds.ServerToken = strings.TrimSpace(serverToken)

	fmt.Print("Client ID (for CLI, optional): ")
	clientID, _ := reader.ReadString('\n')
	creds.ClientID = strings.TrimSpace(clientID)

	fmt.Print("Client Token (for CLI, optional): ")
	clientToken, _ := reader.ReadString('\n')
	creds.ClientToken = strings.TrimSpace(clientToken)

	if creds.ServerID == "" || creds.ServerToken == "" {
		return nil, fmt.Errorf("server ID and token are required")
	}

	return creds, nil
}

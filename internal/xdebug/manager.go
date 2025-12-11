// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package xdebug

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/qoliber/magebox/internal/platform"
)

// Manager handles Xdebug enable/disable operations
type Manager struct {
	platform *platform.Platform
}

// NewManager creates a new Xdebug manager
func NewManager(p *platform.Platform) *Manager {
	return &Manager{platform: p}
}

// IsInstalled checks if Xdebug is installed for a specific PHP version
func (m *Manager) IsInstalled(phpVersion string) bool {
	iniFile := m.getXdebugIniPath(phpVersion)
	if iniFile == "" {
		return false
	}
	_, err := os.Stat(iniFile)
	return err == nil
}

// IsEnabled checks if Xdebug is enabled for a specific PHP version
func (m *Manager) IsEnabled(phpVersion string) bool {
	iniFile := m.getXdebugIniPath(phpVersion)
	if iniFile == "" {
		return false
	}

	content, err := os.ReadFile(iniFile)
	if err != nil {
		return false
	}

	// Check if the zend_extension line is not commented out
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "zend_extension") && strings.Contains(trimmed, "xdebug") {
			return true
		}
	}

	return false
}

// Enable enables Xdebug for a specific PHP version
func (m *Manager) Enable(phpVersion string) error {
	iniFile := m.getXdebugIniPath(phpVersion)
	if iniFile == "" {
		return fmt.Errorf("xdebug ini file not found for PHP %s", phpVersion)
	}

	content, err := os.ReadFile(iniFile)
	if err != nil {
		return fmt.Errorf("failed to read xdebug ini: %w", err)
	}

	// Uncomment the zend_extension line if commented
	lines := strings.Split(string(content), "\n")
	modified := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, ";zend_extension") && strings.Contains(trimmed, "xdebug") {
			lines[i] = strings.TrimPrefix(trimmed, ";")
			modified = true
		}
	}

	if modified {
		if err := os.WriteFile(iniFile, []byte(strings.Join(lines, "\n")), 0644); err != nil {
			return fmt.Errorf("failed to write xdebug ini: %w", err)
		}
	}

	// Also ensure xdebug.mode is set for development
	m.ensureXdebugConfig(phpVersion)

	return nil
}

// Disable disables Xdebug for a specific PHP version
func (m *Manager) Disable(phpVersion string) error {
	iniFile := m.getXdebugIniPath(phpVersion)
	if iniFile == "" {
		return fmt.Errorf("xdebug ini file not found for PHP %s", phpVersion)
	}

	content, err := os.ReadFile(iniFile)
	if err != nil {
		return fmt.Errorf("failed to read xdebug ini: %w", err)
	}

	// Comment out the zend_extension line
	lines := strings.Split(string(content), "\n")
	modified := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "zend_extension") && strings.Contains(trimmed, "xdebug") {
			lines[i] = ";" + trimmed
			modified = true
		}
	}

	if modified {
		if err := os.WriteFile(iniFile, []byte(strings.Join(lines, "\n")), 0644); err != nil {
			return fmt.Errorf("failed to write xdebug ini: %w", err)
		}
	}

	return nil
}

// getXdebugIniPath returns the path to the xdebug.ini file for a specific PHP version
func (m *Manager) getXdebugIniPath(phpVersion string) string {
	switch m.platform.Type {
	case platform.Darwin:
		base := "/usr/local"
		if m.platform.IsAppleSilicon {
			base = "/opt/homebrew"
		}

		// Check conf.d directory for xdebug ini
		confDir := filepath.Join(base, "etc", "php", phpVersion, "conf.d")
		patterns := []string{
			filepath.Join(confDir, "ext-xdebug.ini"),
			filepath.Join(confDir, "20-xdebug.ini"),
			filepath.Join(confDir, "xdebug.ini"),
		}

		for _, pattern := range patterns {
			if _, err := os.Stat(pattern); err == nil {
				return pattern
			}
		}

		// Check if xdebug.so exists but no ini file (need to create one)
		xdebugSo := m.findXdebugSo(phpVersion)
		if xdebugSo != "" {
			// Create the ini file
			iniPath := filepath.Join(confDir, "ext-xdebug.ini")
			return iniPath
		}

	case platform.Linux:
		confDir := filepath.Join("/etc", "php", phpVersion, "mods-available")
		iniPath := filepath.Join(confDir, "xdebug.ini")
		if _, err := os.Stat(iniPath); err == nil {
			return iniPath
		}
	}

	return ""
}

// findXdebugSo finds the xdebug.so extension file for a specific PHP version
func (m *Manager) findXdebugSo(phpVersion string) string {
	switch m.platform.Type {
	case platform.Darwin:
		base := "/usr/local"
		if m.platform.IsAppleSilicon {
			base = "/opt/homebrew"
		}

		// Look in Cellar for the extension
		cellarPath := filepath.Join(base, "Cellar", "php@"+phpVersion)
		matches, err := filepath.Glob(filepath.Join(cellarPath, "*", "pecl", "*", "xdebug.so"))
		if err == nil && len(matches) > 0 {
			return matches[0]
		}

		// Check lib directory
		libPath := filepath.Join(base, "lib", "php", "pecl", "*", "xdebug.so")
		matches, err = filepath.Glob(libPath)
		if err == nil && len(matches) > 0 {
			return matches[0]
		}
	}

	return ""
}

// ensureXdebugConfig ensures xdebug configuration is set for development
func (m *Manager) ensureXdebugConfig(phpVersion string) error {
	iniFile := m.getXdebugIniPath(phpVersion)
	if iniFile == "" {
		return nil
	}

	content, err := os.ReadFile(iniFile)
	if err != nil {
		return err
	}

	// Check if xdebug.mode is already configured
	if strings.Contains(string(content), "xdebug.mode") {
		return nil
	}

	// Append default xdebug configuration
	config := `
; MageBox Xdebug configuration
xdebug.mode=debug
xdebug.start_with_request=trigger
xdebug.client_host=127.0.0.1
xdebug.client_port=9003
xdebug.idekey=PHPSTORM
`

	f, err := os.OpenFile(iniFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(config)
	return err
}

// Install installs Xdebug for a specific PHP version
func (m *Manager) Install(phpVersion string) error {
	switch m.platform.Type {
	case platform.Darwin:
		// Use PECL to install xdebug
		phpBin := m.platform.PHPBinary(phpVersion)
		peclBin := filepath.Dir(phpBin) + "/pecl"

		if _, err := os.Stat(peclBin); os.IsNotExist(err) {
			return fmt.Errorf("PECL not found for PHP %s", phpVersion)
		}

		cmd := exec.Command(peclBin, "install", "xdebug")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()

	case platform.Linux:
		cmd := exec.Command("sudo", "apt-get", "install", "-y", fmt.Sprintf("php%s-xdebug", phpVersion))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	return fmt.Errorf("unsupported platform")
}

// Status returns the current Xdebug status for a PHP version
type Status struct {
	Installed bool
	Enabled   bool
	Mode      string
	IniPath   string
}

// GetStatus returns the current Xdebug status for a PHP version
func (m *Manager) GetStatus(phpVersion string) Status {
	status := Status{
		Installed: m.IsInstalled(phpVersion),
		Enabled:   m.IsEnabled(phpVersion),
		IniPath:   m.getXdebugIniPath(phpVersion),
	}

	if status.Installed && status.Enabled {
		status.Mode = m.getXdebugMode(phpVersion)
	}

	return status
}

// getXdebugMode returns the current xdebug.mode setting
func (m *Manager) getXdebugMode(phpVersion string) string {
	iniFile := m.getXdebugIniPath(phpVersion)
	if iniFile == "" {
		return ""
	}

	f, err := os.Open(iniFile)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "xdebug.mode") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}

	return "debug" // default mode
}

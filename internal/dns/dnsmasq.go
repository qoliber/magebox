package dns

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/qoliber/magebox/internal/platform"
)

//go:embed templates/dnsmasq.conf.tmpl
var dnsmasqConfigTemplate string

// DnsmasqManager manages dnsmasq configuration for wildcard DNS resolution
type DnsmasqManager struct {
	platform *platform.Platform
}

// NewDnsmasqManager creates a new dnsmasq manager
func NewDnsmasqManager(p *platform.Platform) *DnsmasqManager {
	return &DnsmasqManager{platform: p}
}

// IsInstalled checks if dnsmasq is installed
func (m *DnsmasqManager) IsInstalled() bool {
	return platform.CommandExists("dnsmasq")
}

// IsConfigured checks if dnsmasq is configured for MageBox
func (m *DnsmasqManager) IsConfigured() bool {
	configPath := m.getConfigPath()
	_, err := os.Stat(configPath)
	return err == nil
}

// IsRunning checks if dnsmasq is running
func (m *DnsmasqManager) IsRunning() bool {
	cmd := exec.Command("pgrep", "dnsmasq")
	return cmd.Run() == nil
}

// Configure sets up dnsmasq to resolve *.test to localhost
func (m *DnsmasqManager) Configure() error {
	// Create config directory if needed
	configDir := m.getConfigDir()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// On Linux, enable conf-dir in dnsmasq.conf if commented out
	if m.platform.Type == platform.Linux {
		if err := m.enableDnsmasqConfDir(); err != nil {
			return fmt.Errorf("failed to enable dnsmasq.d: %w", err)
		}
	}

	// Write dnsmasq config
	config := m.generateConfig()
	configPath := m.getConfigPath()

	if err := m.writeConfigWithSudo(configPath, config); err != nil {
		return fmt.Errorf("failed to write dnsmasq config: %w", err)
	}

	// On macOS, set up the resolver
	if m.platform.Type == platform.Darwin {
		if err := m.setupMacOSResolver(); err != nil {
			return fmt.Errorf("failed to setup macOS resolver: %w", err)
		}
	}

	// On Linux with systemd-resolved, configure it to use dnsmasq for .test
	if m.platform.Type == platform.Linux {
		if err := m.setupSystemdResolved(); err != nil {
			return fmt.Errorf("failed to setup systemd-resolved: %w", err)
		}
	}

	return nil
}

// Start starts dnsmasq service
func (m *DnsmasqManager) Start() error {
	switch m.platform.Type {
	case platform.Darwin:
		cmd := exec.Command("sudo", "brew", "services", "start", "dnsmasq")
		return cmd.Run()
	case platform.Linux:
		cmd := exec.Command("sudo", "systemctl", "start", "dnsmasq")
		return cmd.Run()
	}
	return fmt.Errorf("unsupported platform")
}

// Stop stops dnsmasq service
func (m *DnsmasqManager) Stop() error {
	switch m.platform.Type {
	case platform.Darwin:
		cmd := exec.Command("sudo", "brew", "services", "stop", "dnsmasq")
		return cmd.Run()
	case platform.Linux:
		cmd := exec.Command("sudo", "systemctl", "stop", "dnsmasq")
		return cmd.Run()
	}
	return fmt.Errorf("unsupported platform")
}

// Restart restarts dnsmasq service
func (m *DnsmasqManager) Restart() error {
	switch m.platform.Type {
	case platform.Darwin:
		cmd := exec.Command("sudo", "brew", "services", "restart", "dnsmasq")
		return cmd.Run()
	case platform.Linux:
		cmd := exec.Command("sudo", "systemctl", "restart", "dnsmasq")
		return cmd.Run()
	}
	return fmt.Errorf("unsupported platform")
}

// Enable enables dnsmasq to start on boot
func (m *DnsmasqManager) Enable() error {
	switch m.platform.Type {
	case platform.Darwin:
		// brew services start already enables it
		return nil
	case platform.Linux:
		cmd := exec.Command("sudo", "systemctl", "enable", "dnsmasq")
		return cmd.Run()
	}
	return fmt.Errorf("unsupported platform")
}

// Remove removes MageBox dnsmasq configuration
func (m *DnsmasqManager) Remove() error {
	configPath := m.getConfigPath()

	if _, err := os.Stat(configPath); err == nil {
		cmd := exec.Command("sudo", "rm", configPath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to remove dnsmasq config: %w", err)
		}
	}

	// On macOS, also remove the resolver
	if m.platform.Type == platform.Darwin {
		resolverPath := "/etc/resolver/test"
		if _, err := os.Stat(resolverPath); err == nil {
			cmd := exec.Command("sudo", "rm", resolverPath)
			_ = cmd.Run() // Ignore errors - resolver may not exist
		}
	}

	return nil
}

// InstallCommand returns the command to install dnsmasq
func (m *DnsmasqManager) InstallCommand() string {
	switch m.platform.Type {
	case platform.Darwin:
		return "brew install dnsmasq"
	case platform.Linux:
		switch m.platform.LinuxDistro {
		case platform.DistroFedora:
			return "sudo dnf install -y dnsmasq"
		case platform.DistroArch:
			return "sudo pacman -S dnsmasq"
		default:
			return "sudo apt install -y dnsmasq"
		}
	}
	return ""
}

// getConfigDir returns the dnsmasq config directory
func (m *DnsmasqManager) getConfigDir() string {
	switch m.platform.Type {
	case platform.Darwin:
		if runtime.GOARCH == "arm64" {
			return "/opt/homebrew/etc/dnsmasq.d"
		}
		return "/usr/local/etc/dnsmasq.d"
	case platform.Linux:
		return "/etc/dnsmasq.d"
	}
	return "/etc/dnsmasq.d"
}

// getConfigPath returns the MageBox dnsmasq config file path
func (m *DnsmasqManager) getConfigPath() string {
	return filepath.Join(m.getConfigDir(), "magebox.conf")
}

// generateConfig generates the dnsmasq configuration
func (m *DnsmasqManager) generateConfig() string {
	return dnsmasqConfigTemplate
}

// writeConfigWithSudo writes config file using sudo
func (m *DnsmasqManager) writeConfigWithSudo(path, content string) error {
	// Write to temp file first
	tmpFile, err := os.CreateTemp("", "dnsmasq-*.conf")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		return err
	}
	tmpFile.Close()

	// Copy to destination with sudo
	cmd := exec.Command("sudo", "cp", tmpPath, path)
	return cmd.Run()
}

// setupMacOSResolver sets up the macOS resolver for .test domain
func (m *DnsmasqManager) setupMacOSResolver() error {
	// Create /etc/resolver directory
	cmd := exec.Command("sudo", "mkdir", "-p", "/etc/resolver")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create resolver directory: %w", err)
	}

	// Create resolver config for .test
	resolverContent := "nameserver 127.0.0.1\n"

	tmpFile, err := os.CreateTemp("", "resolver-test-*")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.WriteString(resolverContent); err != nil {
		tmpFile.Close()
		return err
	}
	tmpFile.Close()

	// Copy to /etc/resolver/test
	cmd = exec.Command("sudo", "cp", tmpPath, "/etc/resolver/test")
	return cmd.Run()
}

// TestResolution tests if DNS resolution is working for .test domains
func (m *DnsmasqManager) TestResolution(domain string) bool {
	cmd := exec.Command("dig", "+short", domain, "@127.0.0.1")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), "127.0.0.1")
}

// Status returns the current dnsmasq status
type DnsmasqStatus struct {
	Installed  bool
	Configured bool
	Running    bool
	TestDomain string
	Resolving  bool
}

// GetStatus returns the current dnsmasq status
func (m *DnsmasqManager) GetStatus() DnsmasqStatus {
	status := DnsmasqStatus{
		Installed:  m.IsInstalled(),
		Configured: m.IsConfigured(),
		Running:    m.IsRunning(),
		TestDomain: "test.test",
	}

	if status.Running {
		status.Resolving = m.TestResolution(status.TestDomain)
	}

	return status
}

// enableDnsmasqConfDir enables the conf-dir directive in /etc/dnsmasq.conf
// This is needed on Fedora/RHEL where it's commented out by default
func (m *DnsmasqManager) enableDnsmasqConfDir() error {
	dnsmasqConf := "/etc/dnsmasq.conf"

	// Read current config
	content, err := os.ReadFile(dnsmasqConf)
	if err != nil {
		return err
	}

	// Check if conf-dir is already enabled
	if strings.Contains(string(content), "conf-dir=/etc/dnsmasq.d\n") &&
		!strings.Contains(string(content), "#conf-dir=/etc/dnsmasq.d") {
		return nil // Already enabled
	}

	// Use sed to uncomment the conf-dir line
	cmd := exec.Command("sudo", "sed", "-i", "s|#conf-dir=/etc/dnsmasq.d|conf-dir=/etc/dnsmasq.d|", dnsmasqConf)
	return cmd.Run()
}

// setupSystemdResolved configures systemd-resolved to use dnsmasq for .test domains
// This is needed on modern Linux distros (Fedora, Ubuntu 18.04+) that use systemd-resolved
func (m *DnsmasqManager) setupSystemdResolved() error {
	// Check if systemd-resolved is running
	cmd := exec.Command("systemctl", "is-active", "systemd-resolved")
	if err := cmd.Run(); err != nil {
		// systemd-resolved not running, skip this step
		return nil
	}

	// Create config directory
	confDir := "/etc/systemd/resolved.conf.d"
	cmd = exec.Command("sudo", "mkdir", "-p", confDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create resolved.conf.d: %w", err)
	}

	// Write resolved config for .test domain
	resolvedConfig := `[Resolve]
DNS=127.0.0.1
Domains=~test
`

	tmpFile, err := os.CreateTemp("", "resolved-magebox-*.conf")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.WriteString(resolvedConfig); err != nil {
		tmpFile.Close()
		return err
	}
	tmpFile.Close()

	// Copy to destination with sudo
	confPath := filepath.Join(confDir, "magebox.conf")
	cmd = exec.Command("sudo", "cp", tmpPath, confPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to write resolved config: %w", err)
	}

	// Restart systemd-resolved
	cmd = exec.Command("sudo", "systemctl", "restart", "systemd-resolved")
	return cmd.Run()
}

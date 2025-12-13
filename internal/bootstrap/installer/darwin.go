// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package installer

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/qoliber/magebox/internal/platform"
)

// DarwinInstaller handles installation on macOS
type DarwinInstaller struct {
	BaseInstaller
}

// NewDarwinInstaller creates a new macOS installer
func NewDarwinInstaller(p *platform.Platform) *DarwinInstaller {
	return &DarwinInstaller{
		BaseInstaller: BaseInstaller{Platform: p},
	}
}

// Platform returns the platform type
func (d *DarwinInstaller) Platform() platform.Type {
	return platform.Darwin
}

// Distro returns empty for Darwin
func (d *DarwinInstaller) Distro() platform.LinuxDistro {
	return ""
}

// ValidateOSVersion checks if the macOS version is supported
func (d *DarwinInstaller) ValidateOSVersion() (OSVersionInfo, error) {
	info := OSVersionInfo{
		Name: "macOS",
	}

	// Get macOS version using sw_vers
	cmd := exec.Command("sw_vers", "-productVersion")
	output, err := cmd.Output()
	if err != nil {
		return info, fmt.Errorf("failed to detect macOS version: %w", err)
	}

	fullVersion := strings.TrimSpace(string(output))
	info.Version = fullVersion

	// Extract major version
	parts := strings.Split(fullVersion, ".")
	if len(parts) < 1 {
		return info, fmt.Errorf("invalid macOS version format: %s", fullVersion)
	}
	majorVersion := parts[0]

	// Check if supported
	supportedVersions := SupportedVersions[platform.Darwin]["macos"]
	for _, v := range supportedVersions {
		if majorVersion == v {
			info.Supported = true
			break
		}
	}

	// Add codename based on major version
	switch majorVersion {
	case "12":
		info.Codename = "Monterey"
	case "13":
		info.Codename = "Ventura"
	case "14":
		info.Codename = "Sonoma"
	case "15":
		info.Codename = "Sequoia"
	}

	if !info.Supported {
		info.Message = fmt.Sprintf("macOS %s may work but is not officially tested", fullVersion)
	}

	return info, nil
}

// InstallPrerequisites installs system prerequisites via Homebrew
func (d *DarwinInstaller) InstallPrerequisites() error {
	// Check if Homebrew is installed
	if !d.CommandExists("brew") {
		return fmt.Errorf("Homebrew is not installed. Install from https://brew.sh")
	}

	// Update Homebrew
	return d.RunCommand("brew update")
}

// InstallPHP installs a specific PHP version via Homebrew
func (d *DarwinInstaller) InstallPHP(version string) error {
	return d.RunCommand(fmt.Sprintf("brew install php@%s", version))
}

// InstallNginx installs Nginx via Homebrew
func (d *DarwinInstaller) InstallNginx() error {
	return d.RunCommand("brew install nginx")
}

// InstallMkcert installs mkcert via Homebrew
func (d *DarwinInstaller) InstallMkcert() error {
	return d.RunCommand("brew install mkcert nss")
}

// InstallDocker returns Docker installation instructions
func (d *DarwinInstaller) InstallDocker() string {
	return "brew install --cask docker"
}

// InstallDnsmasq installs dnsmasq via Homebrew
func (d *DarwinInstaller) InstallDnsmasq() error {
	return d.RunCommand("brew install dnsmasq")
}

// InstallMultitail installs multitail via Homebrew
func (d *DarwinInstaller) InstallMultitail() error {
	return d.RunCommand("brew install multitail")
}

// InstallXdebug installs Xdebug for a specific PHP version via PECL
func (d *DarwinInstaller) InstallXdebug(version string) error {
	base := "/usr/local"
	if d.BaseInstaller.Platform.IsAppleSilicon {
		base = "/opt/homebrew"
	}
	peclBin := fmt.Sprintf("%s/opt/php@%s/bin/pecl", base, version)
	return d.RunCommand(fmt.Sprintf("%s install xdebug 2>/dev/null || true", peclBin))
}

// ConfigurePHPFPM configures PHP-FPM on macOS
// On macOS, PHP-FPM is typically started via brew services
func (d *DarwinInstaller) ConfigurePHPFPM(versions []string) error {
	for _, v := range versions {
		// Stop any running PHP-FPM service first (ignore errors)
		_ = d.RunCommandSilent(fmt.Sprintf("brew services stop php@%s 2>/dev/null || true", v))

		// Start PHP-FPM service
		if err := d.RunCommand(fmt.Sprintf("brew services start php@%s", v)); err != nil {
			return fmt.Errorf("failed to start PHP %s FPM: %w", v, err)
		}
	}
	return nil
}

// ConfigureNginx configures Nginx on macOS
func (d *DarwinInstaller) ConfigureNginx() error {
	// Nginx on macOS via Homebrew typically doesn't require special configuration
	// The main bootstrap process handles adding MageBox includes
	return nil
}

// ConfigureSudoers is a no-op on macOS (not needed for Homebrew services)
func (d *DarwinInstaller) ConfigureSudoers() error {
	// On macOS, Homebrew services run as current user, no sudo needed
	return nil
}

// SetupDNS configures DNS resolution for .test domains on macOS
func (d *DarwinInstaller) SetupDNS() error {
	// Create /etc/resolver directory
	if err := d.RunSudo("mkdir", "-p", "/etc/resolver"); err != nil {
		return fmt.Errorf("failed to create resolver directory: %w", err)
	}

	// Create resolver config for .test domain
	resolverContent := "nameserver 127.0.0.1\n"
	if err := d.WriteFile("/etc/resolver/test", resolverContent); err != nil {
		return fmt.Errorf("failed to write resolver config: %w", err)
	}

	return nil
}

// PackageManager returns "brew" for macOS
func (d *DarwinInstaller) PackageManager() string {
	return "brew"
}

// InstallCommand returns the Homebrew install command format
func (d *DarwinInstaller) InstallCommand(packages ...string) string {
	return fmt.Sprintf("brew install %s", strings.Join(packages, " "))
}

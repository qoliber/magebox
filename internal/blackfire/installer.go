// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package blackfire

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"qoliber/magebox/internal/platform"
)

// Installer handles Blackfire installation across platforms
type Installer struct {
	platform *platform.Platform
}

// NewInstaller creates a new Blackfire installer
func NewInstaller(p *platform.Platform) *Installer {
	return &Installer{platform: p}
}

// InstallAgent installs the Blackfire agent
func (i *Installer) InstallAgent() error {
	switch i.platform.Type {
	case platform.Darwin:
		return i.installAgentDarwin()
	case platform.Linux:
		return i.installAgentLinux()
	default:
		return fmt.Errorf("unsupported platform")
	}
}

// InstallExtension installs the Blackfire PHP extension for a specific version
func (i *Installer) InstallExtension(phpVersion string) error {
	switch i.platform.Type {
	case platform.Darwin:
		return i.installExtensionDarwin(phpVersion)
	case platform.Linux:
		return i.installExtensionLinux(phpVersion)
	default:
		return fmt.Errorf("unsupported platform")
	}
}

// installAgentDarwin installs Blackfire agent on macOS using Homebrew
func (i *Installer) installAgentDarwin() error {
	// Add Blackfire tap if not already added
	cmd := exec.Command("brew", "tap", "blackfireio/homebrew-blackfire")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add blackfire tap: %w", err)
	}

	// Install blackfire
	cmd = exec.Command("brew", "install", "blackfire")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install blackfire: %w", err)
	}

	return nil
}

// installAgentLinux installs Blackfire agent on Linux
func (i *Installer) installAgentLinux() error {
	switch i.platform.LinuxDistro {
	case platform.DistroFedora:
		return i.installAgentFedora()
	case platform.DistroDebian:
		return i.installAgentDebian()
	case platform.DistroArch:
		return i.installAgentArch()
	default:
		return fmt.Errorf("unsupported Linux distribution: %s", i.platform.LinuxDistro)
	}
}

// installAgentFedora installs Blackfire on Fedora/RHEL using dnf
func (i *Installer) installAgentFedora() error {
	// Add Blackfire repository
	repoContent := `[blackfire]
name=blackfire
baseurl=http://packages.blackfire.io/fedora/$releasever/$basearch
repo_gpgcheck=1
gpgcheck=0
enabled=1
gpgkey=https://packages.blackfire.io/gpg.key
sslverify=1
sslcacert=/etc/pki/tls/certs/ca-bundle.crt
metadata_expire=86400
`
	repoPath := "/etc/yum.repos.d/blackfire.repo"

	// Write repo file using sudo
	cmd := exec.Command("sudo", "tee", repoPath)
	cmd.Stdin = strings.NewReader(repoContent)
	cmd.Stdout = nil
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add blackfire repository: %w", err)
	}

	// Install blackfire agent
	cmd = exec.Command("sudo", "dnf", "install", "-y", "blackfire")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install blackfire: %w", err)
	}

	return nil
}

// installAgentDebian installs Blackfire on Debian/Ubuntu using apt
func (i *Installer) installAgentDebian() error {
	// Install prerequisites
	cmd := exec.Command("sudo", "apt-get", "install", "-y", "wget", "gnupg")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install prerequisites: %w", err)
	}

	// Add Blackfire GPG key
	cmd = exec.Command("bash", "-c", "wget -q -O - https://packages.blackfire.io/gpg.key | sudo gpg --dearmor -o /usr/share/keyrings/blackfire-archive-keyring.gpg")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add blackfire GPG key: %w", err)
	}

	// Add Blackfire repository
	repoLine := "deb [arch=amd64 signed-by=/usr/share/keyrings/blackfire-archive-keyring.gpg] http://packages.blackfire.io/debian any main"
	cmd = exec.Command("bash", "-c", fmt.Sprintf("echo '%s' | sudo tee /etc/apt/sources.list.d/blackfire.list", repoLine))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add blackfire repository: %w", err)
	}

	// Update apt and install
	cmd = exec.Command("sudo", "apt-get", "update")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update apt: %w", err)
	}

	cmd = exec.Command("sudo", "apt-get", "install", "-y", "blackfire")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install blackfire: %w", err)
	}

	return nil
}

// installAgentArch installs Blackfire on Arch Linux
func (i *Installer) installAgentArch() error {
	// Blackfire is available from AUR
	fmt.Println("Blackfire is available from AUR. Installing via yay/paru...")

	// Try yay first, then paru
	if platform.CommandExists("yay") {
		cmd := exec.Command("yay", "-S", "--noconfirm", "blackfire")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	if platform.CommandExists("paru") {
		cmd := exec.Command("paru", "-S", "--noconfirm", "blackfire")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	return fmt.Errorf("no AUR helper found (yay or paru). Please install blackfire manually from AUR")
}

// installExtensionDarwin installs Blackfire PHP extension on macOS
func (i *Installer) installExtensionDarwin(phpVersion string) error {
	// Install the PHP-specific Blackfire extension formula
	// e.g., blackfire-php82 for PHP 8.2
	// Homebrew automatically creates the ini file in conf.d
	formulaVersion := strings.ReplaceAll(phpVersion, ".", "")
	formula := fmt.Sprintf("blackfire-php%s", formulaVersion)

	cmd := exec.Command("brew", "install", formula)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install %s: %w", formula, err)
	}

	// Homebrew automatically configures the extension via conf.d/ext-blackfire.ini
	return nil
}

// installExtensionLinux installs Blackfire PHP extension on Linux
func (i *Installer) installExtensionLinux(phpVersion string) error {
	switch i.platform.LinuxDistro {
	case platform.DistroFedora:
		return i.installExtensionFedora(phpVersion)
	case platform.DistroDebian:
		return i.installExtensionDebian(phpVersion)
	case platform.DistroArch:
		return i.installExtensionArch(phpVersion)
	default:
		return fmt.Errorf("unsupported Linux distribution")
	}
}

// installExtensionFedora installs Blackfire PHP extension on Fedora
func (i *Installer) installExtensionFedora(phpVersion string) error {
	remiVersion := strings.ReplaceAll(phpVersion, ".", "")
	packageName := fmt.Sprintf("blackfire-php%s", remiVersion)

	cmd := exec.Command("sudo", "dnf", "install", "-y", packageName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install %s: %w", packageName, err)
	}

	return nil
}

// installExtensionDebian installs Blackfire PHP extension on Debian/Ubuntu
func (i *Installer) installExtensionDebian(phpVersion string) error {
	packageName := fmt.Sprintf("blackfire-php%s", strings.ReplaceAll(phpVersion, ".", ""))

	cmd := exec.Command("sudo", "apt-get", "install", "-y", packageName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install %s: %w", packageName, err)
	}

	return nil
}

// installExtensionArch installs Blackfire PHP extension on Arch
func (i *Installer) installExtensionArch(phpVersion string) error {
	// For Arch, blackfire-php is in AUR
	if platform.CommandExists("yay") {
		cmd := exec.Command("yay", "-S", "--noconfirm", "blackfire-php")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	if platform.CommandExists("paru") {
		cmd := exec.Command("paru", "-S", "--noconfirm", "blackfire-php")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	return fmt.Errorf("no AUR helper found (yay or paru). Please install blackfire-php manually from AUR")
}

// UninstallExtension removes the Blackfire PHP extension for a specific version
func (i *Installer) UninstallExtension(phpVersion string) error {
	iniPath := i.getExtensionIniPath(phpVersion)
	if iniPath == "" {
		return nil
	}

	if _, err := os.Stat(iniPath); os.IsNotExist(err) {
		return nil
	}

	return os.Remove(iniPath)
}

// getExtensionIniPath returns the path to the Blackfire ini file
func (i *Installer) getExtensionIniPath(phpVersion string) string {
	switch i.platform.Type {
	case platform.Darwin:
		base := "/usr/local"
		if i.platform.IsAppleSilicon {
			base = "/opt/homebrew"
		}
		return filepath.Join(base, "etc", "php", phpVersion, "conf.d", "ext-blackfire.ini")
	case platform.Linux:
		remiVersion := strings.ReplaceAll(phpVersion, ".", "")
		switch i.platform.LinuxDistro {
		case platform.DistroFedora:
			return fmt.Sprintf("/etc/opt/remi/php%s/php.d/zz-blackfire.ini", remiVersion)
		default:
			return fmt.Sprintf("/etc/php/%s/mods-available/blackfire.ini", phpVersion)
		}
	}
	return ""
}

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
)

// Installer handles Tideways installation across platforms
type Installer struct {
	platform *platform.Platform
}

// NewInstaller creates a new Tideways installer
func NewInstaller(p *platform.Platform) *Installer {
	return &Installer{platform: p}
}

// InstallDaemon installs the Tideways daemon
func (i *Installer) InstallDaemon() error {
	switch i.platform.Type {
	case platform.Darwin:
		return i.installDaemonDarwin()
	case platform.Linux:
		return i.installDaemonLinux()
	default:
		return fmt.Errorf("unsupported platform")
	}
}

// InstallExtension installs the Tideways PHP extension for a specific version
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

// installDaemonDarwin installs Tideways daemon on macOS using Homebrew
func (i *Installer) installDaemonDarwin() error {
	// Add Tideways tap
	cmd := exec.Command("brew", "tap", "tideways/homebrew-profiler")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add tideways tap: %w", err)
	}

	// Install tideways-daemon
	cmd = exec.Command("brew", "install", "tideways-daemon")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install tideways-daemon: %w", err)
	}

	return nil
}

// installDaemonLinux installs Tideways daemon on Linux
func (i *Installer) installDaemonLinux() error {
	switch i.platform.LinuxDistro {
	case platform.DistroFedora:
		return i.installDaemonFedora()
	case platform.DistroDebian:
		return i.installDaemonDebian()
	case platform.DistroArch:
		return i.installDaemonArch()
	default:
		return fmt.Errorf("unsupported Linux distribution: %s", i.platform.LinuxDistro)
	}
}

// installDaemonFedora installs Tideways on Fedora/RHEL
func (i *Installer) installDaemonFedora() error {
	// Add Tideways repository
	repoContent := `[tideways]
name=Tideways
baseurl=https://packages.tideways.com/rpm-repo/
gpgcheck=0
enabled=1
`
	repoPath := "/etc/yum.repos.d/tideways.repo"

	cmd := exec.Command("sudo", "tee", repoPath)
	cmd.Stdin = strings.NewReader(repoContent)
	cmd.Stdout = nil
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add tideways repository: %w", err)
	}

	// Install daemon
	cmd = exec.Command("sudo", "dnf", "install", "-y", "tideways-daemon")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install tideways-daemon: %w", err)
	}

	return nil
}

// installDaemonDebian installs Tideways on Debian/Ubuntu
func (i *Installer) installDaemonDebian() error {
	// Add GPG key
	cmd := exec.Command("bash", "-c", "curl -sSL 'https://packages.tideways.com/key.gpg' | sudo gpg --dearmor -o /usr/share/keyrings/tideways-archive-keyring.gpg")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add tideways GPG key: %w", err)
	}

	// Add repository
	repoLine := "deb [signed-by=/usr/share/keyrings/tideways-archive-keyring.gpg] https://packages.tideways.com/apt-packages any-version main"
	cmd = exec.Command("bash", "-c", fmt.Sprintf("echo '%s' | sudo tee /etc/apt/sources.list.d/tideways.list", repoLine))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add tideways repository: %w", err)
	}

	// Update and install
	cmd = exec.Command("sudo", "apt-get", "update")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update apt: %w", err)
	}

	cmd = exec.Command("sudo", "apt-get", "install", "-y", "tideways-daemon")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install tideways-daemon: %w", err)
	}

	return nil
}

// installDaemonArch installs Tideways on Arch Linux
func (i *Installer) installDaemonArch() error {
	// Tideways is available from AUR
	if platform.CommandExists("yay") {
		cmd := exec.Command("yay", "-S", "--noconfirm", "tideways-daemon")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	if platform.CommandExists("paru") {
		cmd := exec.Command("paru", "-S", "--noconfirm", "tideways-daemon")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	return fmt.Errorf("no AUR helper found (yay or paru). Please install tideways-daemon manually from AUR")
}

// installExtensionDarwin installs Tideways PHP extension on macOS
func (i *Installer) installExtensionDarwin(phpVersion string) error {
	// Install tideways-php extension via Homebrew
	cmd := exec.Command("brew", "install", "tideways-php")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install tideways-php: %w", err)
	}

	// Create ini file
	base := "/usr/local"
	if i.platform.IsAppleSilicon {
		base = "/opt/homebrew"
	}

	confDir := filepath.Join(base, "etc", "php", phpVersion, "conf.d")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		return fmt.Errorf("failed to create conf.d directory: %w", err)
	}

	extPath := i.findExtensionPath(phpVersion)
	if extPath == "" {
		return fmt.Errorf("tideways extension not found for PHP %s", phpVersion)
	}

	iniContent := fmt.Sprintf("; Tideways PHP extension\nextension=%s\n", extPath)
	iniPath := filepath.Join(confDir, "ext-tideways.ini")

	if err := os.WriteFile(iniPath, []byte(iniContent), 0644); err != nil {
		return fmt.Errorf("failed to create tideways ini: %w", err)
	}

	return nil
}

// installExtensionLinux installs Tideways PHP extension on Linux
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

// installExtensionFedora installs Tideways PHP extension on Fedora
func (i *Installer) installExtensionFedora(phpVersion string) error {
	remiVersion := strings.ReplaceAll(phpVersion, ".", "")
	packageName := fmt.Sprintf("tideways-php%s", remiVersion)

	cmd := exec.Command("sudo", "dnf", "install", "-y", packageName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install %s: %w", packageName, err)
	}

	return nil
}

// installExtensionDebian installs Tideways PHP extension on Debian/Ubuntu
func (i *Installer) installExtensionDebian(phpVersion string) error {
	packageName := fmt.Sprintf("tideways-php-%s", phpVersion)

	cmd := exec.Command("sudo", "apt-get", "install", "-y", packageName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install %s: %w", packageName, err)
	}

	return nil
}

// installExtensionArch installs Tideways PHP extension on Arch
func (i *Installer) installExtensionArch(phpVersion string) error {
	if platform.CommandExists("yay") {
		cmd := exec.Command("yay", "-S", "--noconfirm", "tideways-php")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	if platform.CommandExists("paru") {
		cmd := exec.Command("paru", "-S", "--noconfirm", "tideways-php")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	return fmt.Errorf("no AUR helper found (yay or paru). Please install tideways-php manually from AUR")
}

// findExtensionPath finds the path to the Tideways extension for a PHP version
func (i *Installer) findExtensionPath(phpVersion string) string {
	switch i.platform.Type {
	case platform.Darwin:
		base := "/usr/local"
		if i.platform.IsAppleSilicon {
			base = "/opt/homebrew"
		}
		patterns := []string{
			filepath.Join(base, "lib", "php", "pecl", phpVersion, "tideways.so"),
			filepath.Join(base, "Cellar", "tideways-php", "*", "lib", "php", phpVersion, "tideways.so"),
		}
		for _, p := range patterns {
			matches, _ := filepath.Glob(p)
			if len(matches) > 0 {
				return matches[0]
			}
		}
	case platform.Linux:
		patterns := []string{
			fmt.Sprintf("/usr/lib/php/%s/tideways.so", phpVersion),
			fmt.Sprintf("/usr/lib64/php/%s/tideways.so", phpVersion),
			"/usr/lib/tideways/tideways.so",
		}
		for _, p := range patterns {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}
	return ""
}

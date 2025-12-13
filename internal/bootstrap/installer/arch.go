// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package installer

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/qoliber/magebox/internal/platform"
)

// ArchInstaller handles installation on Arch Linux and derivatives
type ArchInstaller struct {
	BaseInstaller
}

// NewArchInstaller creates a new Arch Linux installer
func NewArchInstaller(p *platform.Platform) *ArchInstaller {
	return &ArchInstaller{
		BaseInstaller: BaseInstaller{Platform: p},
	}
}

// Platform returns the platform type
func (a *ArchInstaller) Platform() platform.Type {
	return platform.Linux
}

// Distro returns Arch
func (a *ArchInstaller) Distro() platform.LinuxDistro {
	return platform.DistroArch
}

// ValidateOSVersion checks the Arch Linux version (rolling release)
func (a *ArchInstaller) ValidateOSVersion() (OSVersionInfo, error) {
	info := OSVersionInfo{
		Name:      "Arch Linux",
		Version:   "rolling",
		Supported: true, // Arch is always rolling, so always "supported"
	}

	// Read /etc/os-release for distribution name
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return info, nil // Continue even if we can't read
	}

	content := string(data)
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "NAME=") {
			info.Name = strings.Trim(strings.TrimPrefix(line, "NAME="), "\"")
		}
		if strings.HasPrefix(line, "BUILD_ID=") {
			// Arch uses BUILD_ID for date-based version
			info.Version = strings.Trim(strings.TrimPrefix(line, "BUILD_ID="), "\"")
		}
	}

	info.Message = "Arch Linux is a rolling release - ensure system is up-to-date"
	return info, nil
}

// InstallPrerequisites installs system prerequisites
func (a *ArchInstaller) InstallPrerequisites() error {
	// Update package database
	if err := a.RunSudo("pacman", "-Sy"); err != nil {
		return fmt.Errorf("failed to update pacman: %w", err)
	}

	// Install basic tools
	return a.RunSudo("pacman", "-S", "--noconfirm", "curl", "git", "unzip", "base-devel")
}

// InstallPHP installs PHP via pacman
// Note: Arch doesn't have versioned PHP packages, only the current version
// For multiple PHP versions, users need AUR helpers or manual compilation
func (a *ArchInstaller) InstallPHP(version string) error {
	// Arch only has one PHP version in official repos
	// We'll install the standard PHP package
	packages := []string{
		"php",
		"php-fpm",
		"php-gd",
		"php-intl",
		"php-sodium",
	}

	args := append([]string{"pacman", "-S", "--noconfirm"}, packages...)
	if err := a.RunSudo(args...); err != nil {
		return err
	}

	// Note: For multiple PHP versions on Arch, users need AUR packages like php74, php81, etc.
	// We'll inform the user about this limitation
	fmt.Println("  Note: Arch Linux only has one PHP version in official repos.")
	fmt.Println("  For multiple PHP versions, consider using AUR packages or phpenv.")

	return nil
}

// InstallNginx installs Nginx
func (a *ArchInstaller) InstallNginx() error {
	return a.RunSudo("pacman", "-S", "--noconfirm", "nginx")
}

// InstallMkcert installs mkcert
func (a *ArchInstaller) InstallMkcert() error {
	return a.RunSudo("pacman", "-S", "--noconfirm", "mkcert", "nss")
}

// InstallDocker returns Docker installation instructions
func (a *ArchInstaller) InstallDocker() string {
	return "sudo pacman -S docker docker-compose && sudo systemctl enable --now docker && sudo usermod -aG docker $USER"
}

// InstallDnsmasq installs dnsmasq
func (a *ArchInstaller) InstallDnsmasq() error {
	return a.RunSudo("pacman", "-S", "--noconfirm", "dnsmasq")
}

// ConfigurePHPFPM configures PHP-FPM on Arch Linux
func (a *ArchInstaller) ConfigurePHPFPM(versions []string) error {
	// Create log directory
	if err := a.RunSudo("mkdir", "-p", "/var/log/magebox"); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}
	if err := a.RunSudo("chmod", "755", "/var/log/magebox"); err != nil {
		return fmt.Errorf("failed to set log directory permissions: %w", err)
	}

	// On Arch, PHP-FPM service is just "php-fpm"
	fpmConf := "/etc/php/php-fpm.conf"
	logFile := "/var/log/magebox/php-fpm.log"

	// Update error_log path if config exists
	if a.FileExists(fpmConf) {
		if err := a.RunSudo("sed", "-i", fmt.Sprintf("s|^error_log = .*|error_log = %s|", logFile), fpmConf); err != nil {
			return fmt.Errorf("failed to configure PHP-FPM logs: %w", err)
		}
	}

	// Enable and start service
	if err := a.RunSudo("systemctl", "enable", "php-fpm"); err != nil {
		return fmt.Errorf("failed to enable php-fpm: %w", err)
	}
	if err := a.RunSudo("systemctl", "restart", "php-fpm"); err != nil {
		return fmt.Errorf("failed to restart php-fpm: %w", err)
	}

	return nil
}

// ConfigureNginx configures Nginx on Arch Linux
func (a *ArchInstaller) ConfigureNginx() error {
	// Get current user
	currentUser := os.Getenv("USER")
	if currentUser == "" {
		currentUser = os.Getenv("LOGNAME")
	}

	if currentUser != "" {
		// Configure nginx to run as current user (for cert access)
		nginxConf := "/etc/nginx/nginx.conf"
		if err := a.RunSudo("sed", "-i", fmt.Sprintf("s/^user .*/user %s;/", currentUser), nginxConf); err != nil {
			return fmt.Errorf("failed to configure nginx user: %w", err)
		}
	}

	// Enable nginx on boot
	if err := a.RunSudo("systemctl", "enable", "nginx"); err != nil {
		return err
	}

	return nil
}

// ConfigureSudoers sets up passwordless sudo for services
func (a *ArchInstaller) ConfigureSudoers() error {
	currentUser := os.Getenv("USER")
	if currentUser == "" {
		currentUser = os.Getenv("LOGNAME")
	}
	if currentUser == "" {
		return fmt.Errorf("could not determine current user")
	}

	sudoersFile := "/etc/sudoers.d/magebox"
	if a.FileExists(sudoersFile) {
		return nil // Already configured
	}

	sudoersContent := fmt.Sprintf(`# MageBox - Allow %[1]s to control nginx and php-fpm without password
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl start nginx
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl stop nginx
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl reload nginx
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart nginx
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/nginx -s reload
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/nginx -t
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl start php-fpm
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl stop php-fpm
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl reload php-fpm
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart php-fpm
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/cp /tmp/magebox-* /etc/nginx/nginx.conf
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/mkdir -p /etc/nginx/*
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/rm /etc/nginx/*
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/ln -s *
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/sed -i *
`, currentUser)

	// Write sudoers file
	if err := a.WriteFile(sudoersFile, sudoersContent); err != nil {
		return fmt.Errorf("failed to write sudoers file: %w", err)
	}

	// Set correct permissions
	if err := a.RunSudo("chmod", "0440", sudoersFile); err != nil {
		return fmt.Errorf("failed to set sudoers permissions: %w", err)
	}

	return nil
}

// SetupDNS configures DNS resolution for .test domains
func (a *ArchInstaller) SetupDNS() error {
	// Create MageBox dnsmasq config
	configDir := "/etc/dnsmasq.d"
	if err := a.RunSudo("mkdir", "-p", configDir); err != nil {
		return err
	}

	mageboxConf := `# MageBox - Resolve *.test to localhost
address=/test/127.0.0.1
`
	if err := a.WriteFile("/etc/dnsmasq.d/magebox.conf", mageboxConf); err != nil {
		return err
	}

	// Enable conf-dir in dnsmasq.conf
	dnsmasqConf := "/etc/dnsmasq.conf"
	// Arch may not have conf-dir enabled by default
	if err := a.RunCommand("grep -q 'conf-dir=/etc/dnsmasq.d' /etc/dnsmasq.conf || echo 'conf-dir=/etc/dnsmasq.d' | sudo tee -a /etc/dnsmasq.conf"); err != nil {
		return fmt.Errorf("failed to configure dnsmasq.d: %w", err)
	}
	_ = dnsmasqConf // suppress unused warning

	// Check if systemd-resolved is running
	cmd := exec.Command("systemctl", "is-active", "systemd-resolved")
	if cmd.Run() == nil {
		// Configure systemd-resolved to use dnsmasq for .test
		resolvedDir := "/etc/systemd/resolved.conf.d"
		if err := a.RunSudo("mkdir", "-p", resolvedDir); err != nil {
			return err
		}

		resolvedConfig := `[Resolve]
DNS=127.0.0.1
Domains=~test
`
		if err := a.WriteFile(resolvedDir+"/magebox.conf", resolvedConfig); err != nil {
			return err
		}

		// Restart systemd-resolved
		if err := a.RunSudo("systemctl", "restart", "systemd-resolved"); err != nil {
			return err
		}
	}

	// Enable and start dnsmasq
	if err := a.RunSudo("systemctl", "enable", "dnsmasq"); err != nil {
		return err
	}
	if err := a.RunSudo("systemctl", "restart", "dnsmasq"); err != nil {
		return err
	}

	return nil
}

// PackageManager returns "pacman" for Arch
func (a *ArchInstaller) PackageManager() string {
	return "pacman"
}

// InstallCommand returns the pacman install command format
func (a *ArchInstaller) InstallCommand(packages ...string) string {
	return fmt.Sprintf("sudo pacman -S %s", strings.Join(packages, " "))
}

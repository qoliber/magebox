// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package installer

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"qoliber/magebox/internal/config"
	"qoliber/magebox/internal/platform"
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
		"php-imagick",
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

// InstallMultitail installs multitail
func (a *ArchInstaller) InstallMultitail() error {
	return a.RunSudo("pacman", "-S", "--noconfirm", "multitail")
}

// InstallXdebug installs Xdebug for a specific PHP version
func (a *ArchInstaller) InstallXdebug(version string) error {
	// On Arch, xdebug is available from community repo
	return a.RunSudo("pacman", "-S", "--noconfirm", "xdebug")
}

// InstallImagick installs ImageMagick PHP extension for a specific PHP version
func (a *ArchInstaller) InstallImagick(version string) error {
	// On Arch, imagick is available from community repo
	return a.RunSudo("pacman", "-S", "--noconfirm", "php-imagick")
}

// InstallSodium installs the sodium PHP extension for a specific PHP version
// Required for Argon2i password hashing in Magento
func (a *ArchInstaller) InstallSodium(version string) error {
	return a.RunSudo("pacman", "-S", "--noconfirm", "php-sodium")
}

// ConfigurePHPFPM configures PHP-FPM on Arch Linux
func (a *ArchInstaller) ConfigurePHPFPM(versions []string) error {
	// On Arch, PHP-FPM service is just "php-fpm" (single version)
	// Note: We use default log paths to avoid permission issues

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Arch Linux paths (single PHP version)
	fpmConfPath := "/etc/php/php-fpm.conf"
	wwwConfPath := "/etc/php/php-fpm.d/www.conf"

	// Disable default www.conf pool - MageBox manages its own pools
	if a.FileExists(wwwConfPath) {
		disabledPath := wwwConfPath + ".disabled"
		if !a.FileExists(disabledPath) {
			if err := a.RunSudo("mv", wwwConfPath, disabledPath); err != nil {
				fmt.Printf("  Warning: could not disable default www.conf pool: %v\n", err)
			}
		}
	}

	// Arch uses PHP version from system, so we'll use the first version or default to current
	phpVersion := "8.3" // Default for Arch's current PHP
	if len(versions) > 0 {
		phpVersion = versions[0]
	}
	mageboxPoolsInclude := fmt.Sprintf("include=%s/.magebox/php/pools/%s/*.conf", homeDir, phpVersion)

	// Add MageBox pools include to php-fpm.conf if not already present
	if a.FileExists(fpmConfPath) {
		checkCmd := exec.Command("grep", "-q", mageboxPoolsInclude, fpmConfPath)
		if checkCmd.Run() != nil {
			if err := a.RunSudo("sh", "-c", fmt.Sprintf("echo '%s' >> %s", mageboxPoolsInclude, fpmConfPath)); err != nil {
				return fmt.Errorf("failed to add MageBox pools include to %s: %w", fpmConfPath, err)
			}
		}
	}

	// Create MageBox pools directory
	poolsDir := fmt.Sprintf("%s/.magebox/php/pools/%s", homeDir, phpVersion)
	if err := os.MkdirAll(poolsDir, 0755); err != nil {
		return fmt.Errorf("failed to create pools directory %s: %w", poolsDir, err)
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

		// Increase worker_connections for better performance (default 1024 is too low for Magento)
		_ = a.RunSudo("sed", "-i", "s/worker_connections.*/worker_connections 4096;/", nginxConf)
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
# Blackfire profiler
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl start blackfire-agent
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl stop blackfire-agent
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart blackfire-agent
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl enable blackfire-agent
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/pacman -S --noconfirm *blackfire*
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/pacman -S --noconfirm *tideways*
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

// ConfigureSELinux is a no-op on Arch (SELinux typically not used)
func (a *ArchInstaller) ConfigureSELinux() error {
	return nil
}

// SetupDNS configures DNS resolution for local domains
func (a *ArchInstaller) SetupDNS() error {
	// Get configured TLD
	homeDir, _ := os.UserHomeDir()
	globalCfg, _ := config.LoadGlobalConfig(homeDir)
	tld := globalCfg.GetTLD()

	// Create MageBox dnsmasq config
	configDir := "/etc/dnsmasq.d"
	if err := a.RunSudo("mkdir", "-p", configDir); err != nil {
		return err
	}

	mageboxConf := fmt.Sprintf(`# MageBox - Resolve *.%s to localhost
address=/%s/127.0.0.1
address=/%s/::1
listen-address=127.0.0.2
port=53
bind-interfaces
`, tld, tld, tld)
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
		// Configure systemd-resolved to use dnsmasq for the TLD
		resolvedDir := "/etc/systemd/resolved.conf.d"
		if err := a.RunSudo("mkdir", "-p", resolvedDir); err != nil {
			return err
		}

		resolvedConfig := fmt.Sprintf(`[Resolve]
DNS=127.0.0.2
Domains=~%s
`, tld)
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

// ConfigurePHPINI sets Magento-friendly PHP INI defaults for Arch Linux
func (a *ArchInstaller) ConfigurePHPINI(versions []string) error {
	// Arch Linux uses a single PHP version at /etc/php/php.ini
	iniPath := "/etc/php/php.ini"

	if !a.FileExists(iniPath) {
		return nil
	}

	// Set memory_limit=-1 for CLI (unlimited for Magento compile/deploy)
	if err := a.RunSudo("sed", "-i", "s/^memory_limit = .*/memory_limit = -1/", iniPath); err != nil {
		return fmt.Errorf("failed to set memory_limit in %s: %w", iniPath, err)
	}

	// Set max_execution_time for long-running CLI scripts
	if err := a.RunSudo("sed", "-i", "s/^max_execution_time = .*/max_execution_time = 18000/", iniPath); err != nil {
		return fmt.Errorf("failed to set max_execution_time in %s: %w", iniPath, err)
	}

	return nil
}

// InstallBlackfire installs Blackfire agent and PHP extension
func (a *ArchInstaller) InstallBlackfire(versions []string) error {
	// On Arch, Blackfire needs to be installed via AUR or manually
	// Install via pecl for the PHP extension
	fmt.Println("  Note: Blackfire agent must be installed from AUR (blackfire-agent) or manually.")
	fmt.Println("  Installing PHP extension via pecl...")

	// Install blackfire extension via pecl (ignore errors if it fails)
	_ = a.RunCommandSilent("pecl install blackfire 2>/dev/null || true")

	return nil
}

// InstallTideways installs Tideways PHP extension
func (a *ArchInstaller) InstallTideways(versions []string) error {
	// On Arch, install Tideways via pecl
	fmt.Println("  Installing Tideways PHP extension via pecl...")

	// Install tideways extension via pecl (ignore errors if it fails)
	_ = a.RunCommandSilent("pecl install tideways 2>/dev/null || true")

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

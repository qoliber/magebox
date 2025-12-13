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

// UbuntuInstaller handles installation on Ubuntu/Debian
type UbuntuInstaller struct {
	BaseInstaller
}

// NewUbuntuInstaller creates a new Ubuntu/Debian installer
func NewUbuntuInstaller(p *platform.Platform) *UbuntuInstaller {
	return &UbuntuInstaller{
		BaseInstaller: BaseInstaller{Platform: p},
	}
}

// Platform returns the platform type
func (u *UbuntuInstaller) Platform() platform.Type {
	return platform.Linux
}

// Distro returns Debian
func (u *UbuntuInstaller) Distro() platform.LinuxDistro {
	return platform.DistroDebian
}

// ValidateOSVersion checks if the Ubuntu/Debian version is supported
func (u *UbuntuInstaller) ValidateOSVersion() (OSVersionInfo, error) {
	info := OSVersionInfo{}

	// Read /etc/os-release
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return info, fmt.Errorf("failed to read /etc/os-release: %w", err)
	}

	content := string(data)
	isUbuntu := strings.Contains(strings.ToLower(content), "id=ubuntu")

	// Parse values
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "VERSION_ID=") {
			info.Version = strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), "\"")
		}
		if strings.HasPrefix(line, "NAME=") {
			info.Name = strings.Trim(strings.TrimPrefix(line, "NAME="), "\"")
		}
		if strings.HasPrefix(line, "VERSION_CODENAME=") {
			info.Codename = strings.Trim(strings.TrimPrefix(line, "VERSION_CODENAME="), "\"")
		}
	}

	// Check if supported
	var supportedVersions []string
	if isUbuntu {
		supportedVersions = SupportedVersions[platform.Linux]["ubuntu"]
	} else {
		supportedVersions = SupportedVersions[platform.Linux]["debian"]
	}

	for _, v := range supportedVersions {
		if info.Version == v {
			info.Supported = true
			break
		}
	}

	if !info.Supported {
		info.Message = fmt.Sprintf("%s %s may work but is not officially tested", info.Name, info.Version)
	}

	return info, nil
}

// InstallPrerequisites installs system prerequisites
func (u *UbuntuInstaller) InstallPrerequisites() error {
	// Update package lists
	if err := u.RunSudo("apt", "update"); err != nil {
		return fmt.Errorf("failed to update apt: %w", err)
	}

	// Install basic tools
	if err := u.RunSudo("apt", "install", "-y", "curl", "git", "unzip", "software-properties-common"); err != nil {
		return err
	}

	// Add Ondrej PPA for PHP
	if err := u.RunCommand("sudo add-apt-repository -y ppa:ondrej/php"); err != nil {
		return fmt.Errorf("failed to add Ondrej PPA: %w", err)
	}

	// Update after adding PPA
	return u.RunSudo("apt", "update")
}

// InstallPHP installs a specific PHP version via Ondrej PPA
func (u *UbuntuInstaller) InstallPHP(version string) error {
	// Install PHP with all required extensions
	packages := []string{
		fmt.Sprintf("php%s-fpm", version),
		fmt.Sprintf("php%s-cli", version),
		fmt.Sprintf("php%s-common", version),
		fmt.Sprintf("php%s-mysql", version),
		fmt.Sprintf("php%s-xml", version),
		fmt.Sprintf("php%s-curl", version),
		fmt.Sprintf("php%s-mbstring", version),
		fmt.Sprintf("php%s-zip", version),
		fmt.Sprintf("php%s-gd", version),
		fmt.Sprintf("php%s-intl", version),
		fmt.Sprintf("php%s-bcmath", version),
		fmt.Sprintf("php%s-soap", version),
		fmt.Sprintf("php%s-opcache", version),
		fmt.Sprintf("php%s-sodium", version),
	}

	args := append([]string{"apt", "install", "-y"}, packages...)
	return u.RunSudo(args...)
}

// InstallNginx installs Nginx
func (u *UbuntuInstaller) InstallNginx() error {
	return u.RunSudo("apt", "install", "-y", "nginx")
}

// InstallMkcert installs mkcert
func (u *UbuntuInstaller) InstallMkcert() error {
	return u.RunSudo("apt", "install", "-y", "mkcert", "libnss3-tools")
}

// InstallDocker returns Docker installation instructions
func (u *UbuntuInstaller) InstallDocker() string {
	return "curl -fsSL https://get.docker.com | sudo sh && sudo usermod -aG docker $USER"
}

// InstallDnsmasq installs dnsmasq
func (u *UbuntuInstaller) InstallDnsmasq() error {
	return u.RunSudo("apt", "install", "-y", "dnsmasq")
}

// ConfigurePHPFPM configures PHP-FPM on Ubuntu/Debian
func (u *UbuntuInstaller) ConfigurePHPFPM(versions []string) error {
	// Create log directory
	if err := u.RunSudo("mkdir", "-p", "/var/log/magebox"); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}
	if err := u.RunSudo("chmod", "755", "/var/log/magebox"); err != nil {
		return fmt.Errorf("failed to set log directory permissions: %w", err)
	}

	for _, v := range versions {
		serviceName := fmt.Sprintf("php%s-fpm", v)
		fpmConf := fmt.Sprintf("/etc/php/%s/fpm/php-fpm.conf", v)
		logFile := fmt.Sprintf("/var/log/magebox/php%s-fpm.log", strings.ReplaceAll(v, ".", ""))

		// Update error_log path if config exists
		if u.FileExists(fpmConf) {
			if err := u.RunSudo("sed", "-i", fmt.Sprintf("s|^error_log = .*|error_log = %s|", logFile), fpmConf); err != nil {
				return fmt.Errorf("failed to configure PHP %s FPM logs: %w", v, err)
			}
		}

		// Enable and start service
		if err := u.RunSudo("systemctl", "enable", serviceName); err != nil {
			return fmt.Errorf("failed to enable %s: %w", serviceName, err)
		}
		if err := u.RunSudo("systemctl", "restart", serviceName); err != nil {
			return fmt.Errorf("failed to restart %s: %w", serviceName, err)
		}
	}

	return nil
}

// ConfigureNginx configures Nginx on Ubuntu/Debian
func (u *UbuntuInstaller) ConfigureNginx() error {
	// Get current user
	currentUser := os.Getenv("USER")
	if currentUser == "" {
		currentUser = os.Getenv("LOGNAME")
	}

	if currentUser != "" {
		// Configure nginx to run as current user (for cert access)
		nginxConf := "/etc/nginx/nginx.conf"
		if err := u.RunSudo("sed", "-i", fmt.Sprintf("s/^user .*/user %s;/", currentUser), nginxConf); err != nil {
			return fmt.Errorf("failed to configure nginx user: %w", err)
		}
	}

	// Enable nginx on boot
	if err := u.RunSudo("systemctl", "enable", "nginx"); err != nil {
		return err
	}

	return nil
}

// ConfigureSudoers sets up passwordless sudo for services
func (u *UbuntuInstaller) ConfigureSudoers() error {
	currentUser := os.Getenv("USER")
	if currentUser == "" {
		currentUser = os.Getenv("LOGNAME")
	}
	if currentUser == "" {
		return fmt.Errorf("could not determine current user")
	}

	sudoersFile := "/etc/sudoers.d/magebox"
	if u.FileExists(sudoersFile) {
		return nil // Already configured
	}

	sudoersContent := fmt.Sprintf(`# MageBox - Allow %[1]s to control nginx and php-fpm without password
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl start nginx
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl stop nginx
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl reload nginx
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart nginx
%[1]s ALL=(ALL) NOPASSWD: /usr/sbin/nginx -s reload
%[1]s ALL=(ALL) NOPASSWD: /usr/sbin/nginx -t
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl start php*-fpm
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl stop php*-fpm
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl reload php*-fpm
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart php*-fpm
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/cp /tmp/magebox-* /etc/nginx/nginx.conf
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/mkdir -p /etc/nginx/*
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/rm /etc/nginx/*
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/ln -s *
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/sed -i *
`, currentUser)

	// Write sudoers file
	if err := u.WriteFile(sudoersFile, sudoersContent); err != nil {
		return fmt.Errorf("failed to write sudoers file: %w", err)
	}

	// Set correct permissions
	if err := u.RunSudo("chmod", "0440", sudoersFile); err != nil {
		return fmt.Errorf("failed to set sudoers permissions: %w", err)
	}

	return nil
}

// SetupDNS configures DNS resolution for .test domains
func (u *UbuntuInstaller) SetupDNS() error {
	// Create MageBox dnsmasq config
	configDir := "/etc/dnsmasq.d"
	if err := u.RunSudo("mkdir", "-p", configDir); err != nil {
		return err
	}

	mageboxConf := `# MageBox - Resolve *.test to localhost
address=/test/127.0.0.1
`
	if err := u.WriteFile("/etc/dnsmasq.d/magebox.conf", mageboxConf); err != nil {
		return err
	}

	// Check if systemd-resolved is running (common on Ubuntu 18.04+)
	cmd := exec.Command("systemctl", "is-active", "systemd-resolved")
	if cmd.Run() == nil {
		// Configure systemd-resolved to use dnsmasq for .test
		resolvedDir := "/etc/systemd/resolved.conf.d"
		if err := u.RunSudo("mkdir", "-p", resolvedDir); err != nil {
			return err
		}

		resolvedConfig := `[Resolve]
DNS=127.0.0.1
Domains=~test
`
		if err := u.WriteFile(resolvedDir+"/magebox.conf", resolvedConfig); err != nil {
			return err
		}

		// Restart systemd-resolved
		if err := u.RunSudo("systemctl", "restart", "systemd-resolved"); err != nil {
			return err
		}
	}

	// Enable and start dnsmasq
	if err := u.RunSudo("systemctl", "enable", "dnsmasq"); err != nil {
		return err
	}
	if err := u.RunSudo("systemctl", "restart", "dnsmasq"); err != nil {
		return err
	}

	return nil
}

// PackageManager returns "apt" for Ubuntu/Debian
func (u *UbuntuInstaller) PackageManager() string {
	return "apt"
}

// InstallCommand returns the apt install command format
func (u *UbuntuInstaller) InstallCommand(packages ...string) string {
	return fmt.Sprintf("sudo apt install -y %s", strings.Join(packages, " "))
}

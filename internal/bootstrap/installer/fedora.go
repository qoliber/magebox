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

// FedoraInstaller handles installation on Fedora/RHEL/CentOS
type FedoraInstaller struct {
	BaseInstaller
}

// NewFedoraInstaller creates a new Fedora installer
func NewFedoraInstaller(p *platform.Platform) *FedoraInstaller {
	return &FedoraInstaller{
		BaseInstaller: BaseInstaller{Platform: p},
	}
}

// Platform returns the platform type
func (f *FedoraInstaller) Platform() platform.Type {
	return platform.Linux
}

// Distro returns Fedora
func (f *FedoraInstaller) Distro() platform.LinuxDistro {
	return platform.DistroFedora
}

// ValidateOSVersion checks if the Fedora version is supported
func (f *FedoraInstaller) ValidateOSVersion() (OSVersionInfo, error) {
	info := OSVersionInfo{
		Name: "Fedora",
	}

	// Read /etc/os-release
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return info, fmt.Errorf("failed to read /etc/os-release: %w", err)
	}

	content := string(data)

	// Parse VERSION_ID
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
	supportedVersions := SupportedVersions[platform.Linux]["fedora"]
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
func (f *FedoraInstaller) InstallPrerequisites() error {
	// Install basic tools
	if err := f.RunSudo("dnf", "install", "-y", "curl", "git", "unzip"); err != nil {
		return err
	}

	// Enable Remi repository for PHP
	if !f.FileExists("/etc/yum.repos.d/remi.repo") {
		// Get Fedora version for Remi repo URL
		info, err := f.ValidateOSVersion()
		if err != nil {
			return fmt.Errorf("failed to get Fedora version: %w", err)
		}

		remiURL := fmt.Sprintf("https://rpms.remirepo.net/fedora/remi-release-%s.rpm", info.Version)
		if err := f.RunSudo("dnf", "install", "-y", remiURL); err != nil {
			return fmt.Errorf("failed to install Remi repository: %w", err)
		}
	}

	return nil
}

// InstallPHP installs a specific PHP version via Remi repository
func (f *FedoraInstaller) InstallPHP(version string) error {
	// Remi uses php82, php83 format (no dot)
	remiVersion := strings.ReplaceAll(version, ".", "")

	// Install PHP with all required extensions
	packages := []string{
		fmt.Sprintf("php%s-php-fpm", remiVersion),
		fmt.Sprintf("php%s-php-cli", remiVersion),
		fmt.Sprintf("php%s-php-common", remiVersion),
		fmt.Sprintf("php%s-php-mysqlnd", remiVersion),
		fmt.Sprintf("php%s-php-xml", remiVersion),
		fmt.Sprintf("php%s-php-mbstring", remiVersion),
		fmt.Sprintf("php%s-php-zip", remiVersion),
		fmt.Sprintf("php%s-php-gd", remiVersion),
		fmt.Sprintf("php%s-php-intl", remiVersion),
		fmt.Sprintf("php%s-php-bcmath", remiVersion),
		fmt.Sprintf("php%s-php-soap", remiVersion),
		fmt.Sprintf("php%s-php-opcache", remiVersion),
		fmt.Sprintf("php%s-php-sodium", remiVersion),
	}

	args := append([]string{"dnf", "install", "-y"}, packages...)
	return f.RunSudo(args...)
}

// InstallNginx installs Nginx
func (f *FedoraInstaller) InstallNginx() error {
	return f.RunSudo("dnf", "install", "-y", "nginx")
}

// InstallMkcert installs mkcert
func (f *FedoraInstaller) InstallMkcert() error {
	return f.RunSudo("dnf", "install", "-y", "mkcert", "nss-tools")
}

// InstallDocker returns Docker installation instructions
func (f *FedoraInstaller) InstallDocker() string {
	return "sudo dnf install -y dnf-plugins-core && sudo dnf config-manager --add-repo https://download.docker.com/linux/fedora/docker-ce.repo && sudo dnf install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin && sudo systemctl enable --now docker && sudo usermod -aG docker $USER"
}

// InstallDnsmasq installs dnsmasq
func (f *FedoraInstaller) InstallDnsmasq() error {
	return f.RunSudo("dnf", "install", "-y", "dnsmasq")
}

// InstallMultitail installs multitail
func (f *FedoraInstaller) InstallMultitail() error {
	return f.RunSudo("dnf", "install", "-y", "multitail")
}

// ConfigurePHPFPM configures PHP-FPM on Fedora
func (f *FedoraInstaller) ConfigurePHPFPM(versions []string) error {
	// Create log directory
	if err := f.RunSudo("mkdir", "-p", "/var/log/magebox"); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}
	if err := f.RunSudo("chmod", "755", "/var/log/magebox"); err != nil {
		return fmt.Errorf("failed to set log directory permissions: %w", err)
	}

	for _, v := range versions {
		remiVersion := strings.ReplaceAll(v, ".", "")
		serviceName := fmt.Sprintf("php%s-php-fpm", remiVersion)
		fpmConf := fmt.Sprintf("/etc/opt/remi/php%s/php-fpm.conf", remiVersion)
		logFile := fmt.Sprintf("/var/log/magebox/php%s-fpm.log", remiVersion)

		// Update error_log path
		if f.FileExists(fpmConf) {
			if err := f.RunSudo("sed", "-i", fmt.Sprintf("s|^error_log = .*|error_log = %s|", logFile), fpmConf); err != nil {
				return fmt.Errorf("failed to configure PHP %s FPM logs: %w", v, err)
			}
		}

		// Enable and start service
		if err := f.RunSudo("systemctl", "enable", serviceName); err != nil {
			return fmt.Errorf("failed to enable %s: %w", serviceName, err)
		}
		if err := f.RunSudo("systemctl", "restart", serviceName); err != nil {
			return fmt.Errorf("failed to restart %s: %w", serviceName, err)
		}
	}

	return nil
}

// ConfigureNginx configures Nginx on Fedora
func (f *FedoraInstaller) ConfigureNginx() error {
	// Get current user
	currentUser := os.Getenv("USER")
	if currentUser == "" {
		currentUser = os.Getenv("LOGNAME")
	}

	if currentUser != "" {
		// Configure nginx to run as current user (for cert access)
		nginxConf := "/etc/nginx/nginx.conf"
		if err := f.RunSudo("sed", "-i", fmt.Sprintf("s/^user .*/user %s;/", currentUser), nginxConf); err != nil {
			return fmt.Errorf("failed to configure nginx user: %w", err)
		}
	}

	// Enable nginx on boot
	if err := f.RunSudo("systemctl", "enable", "nginx"); err != nil {
		return err
	}

	return nil
}

// ConfigureSudoers sets up passwordless sudo for services
func (f *FedoraInstaller) ConfigureSudoers() error {
	currentUser := os.Getenv("USER")
	if currentUser == "" {
		currentUser = os.Getenv("LOGNAME")
	}
	if currentUser == "" {
		return fmt.Errorf("could not determine current user")
	}

	sudoersFile := "/etc/sudoers.d/magebox"
	if f.FileExists(sudoersFile) {
		return nil // Already configured
	}

	sudoersContent := fmt.Sprintf(`# MageBox - Allow %[1]s to control nginx and php-fpm without password
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl start nginx
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl stop nginx
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl reload nginx
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart nginx
%[1]s ALL=(ALL) NOPASSWD: /usr/sbin/nginx -s reload
%[1]s ALL=(ALL) NOPASSWD: /usr/sbin/nginx -t
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/nginx -s reload
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/nginx -t
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/nginx
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl start php*-php-fpm
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl stop php*-php-fpm
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl reload php*-php-fpm
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart php*-php-fpm
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/cp /tmp/magebox-* /etc/nginx/nginx.conf
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/mkdir -p /etc/nginx/*
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/rm /etc/nginx/*
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/ln -s *
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/sed -i *
`, currentUser)

	// Write sudoers file
	if err := f.WriteFile(sudoersFile, sudoersContent); err != nil {
		return fmt.Errorf("failed to write sudoers file: %w", err)
	}

	// Set correct permissions
	if err := f.RunSudo("chmod", "0440", sudoersFile); err != nil {
		return fmt.Errorf("failed to set sudoers permissions: %w", err)
	}

	return nil
}

// SetupDNS configures DNS resolution for .test domains
func (f *FedoraInstaller) SetupDNS() error {
	// Enable conf-dir in dnsmasq.conf
	dnsmasqConf := "/etc/dnsmasq.conf"
	if err := f.RunSudo("sed", "-i", "s|#conf-dir=/etc/dnsmasq.d|conf-dir=/etc/dnsmasq.d|", dnsmasqConf); err != nil {
		return fmt.Errorf("failed to enable dnsmasq.d: %w", err)
	}

	// Create MageBox dnsmasq config
	configDir := "/etc/dnsmasq.d"
	if err := f.RunSudo("mkdir", "-p", configDir); err != nil {
		return err
	}

	mageboxConf := `# MageBox - Resolve *.test to localhost
address=/test/127.0.0.1
`
	if err := f.WriteFile("/etc/dnsmasq.d/magebox.conf", mageboxConf); err != nil {
		return err
	}

	// Check if systemd-resolved is running
	cmd := exec.Command("systemctl", "is-active", "systemd-resolved")
	if cmd.Run() == nil {
		// Configure systemd-resolved to use dnsmasq for .test
		resolvedDir := "/etc/systemd/resolved.conf.d"
		if err := f.RunSudo("mkdir", "-p", resolvedDir); err != nil {
			return err
		}

		resolvedConfig := `[Resolve]
DNS=127.0.0.1
Domains=~test
`
		if err := f.WriteFile(resolvedDir+"/magebox.conf", resolvedConfig); err != nil {
			return err
		}

		// Restart systemd-resolved
		if err := f.RunSudo("systemctl", "restart", "systemd-resolved"); err != nil {
			return err
		}
	}

	// Enable and start dnsmasq
	if err := f.RunSudo("systemctl", "enable", "dnsmasq"); err != nil {
		return err
	}
	if err := f.RunSudo("systemctl", "restart", "dnsmasq"); err != nil {
		return err
	}

	return nil
}

// PackageManager returns "dnf" for Fedora
func (f *FedoraInstaller) PackageManager() string {
	return "dnf"
}

// InstallCommand returns the dnf install command format
func (f *FedoraInstaller) InstallCommand(packages ...string) string {
	return fmt.Sprintf("sudo dnf install -y %s", strings.Join(packages, " "))
}

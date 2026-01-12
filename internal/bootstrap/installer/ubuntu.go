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
	// Note: sodium is included in php-common on Ubuntu/Debian
	packages := []string{
		fmt.Sprintf("php%s-fpm", version),
		fmt.Sprintf("php%s-cli", version),
		fmt.Sprintf("php%s-common", version),
		fmt.Sprintf("php%s-opcache", version),
		fmt.Sprintf("php%s-zip", version),
		fmt.Sprintf("php%s-curl", version),
		fmt.Sprintf("php%s-mbstring", version),
		fmt.Sprintf("php%s-xml", version),
		fmt.Sprintf("php%s-bcmath", version),
		fmt.Sprintf("php%s-gd", version),
		fmt.Sprintf("php%s-intl", version),
		fmt.Sprintf("php%s-mysql", version),
		fmt.Sprintf("php%s-soap", version),
		fmt.Sprintf("php%s-imagick", version),
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

// InstallMultitail installs multitail
func (u *UbuntuInstaller) InstallMultitail() error {
	return u.RunSudo("apt", "install", "-y", "multitail")
}

// InstallXdebug installs Xdebug for a specific PHP version
func (u *UbuntuInstaller) InstallXdebug(version string) error {
	return u.RunSudo("apt", "install", "-y", fmt.Sprintf("php%s-xdebug", version))
}

// InstallImagick installs ImageMagick PHP extension for a specific PHP version
func (u *UbuntuInstaller) InstallImagick(version string) error {
	return u.RunSudo("apt", "install", "-y", fmt.Sprintf("php%s-imagick", version))
}

// InstallSodium installs the sodium PHP extension for a specific PHP version
// Required for Argon2i password hashing in Magento
// Note: On Ubuntu/Debian with Ondrej PPA, sodium may be in php-common or separate package
func (u *UbuntuInstaller) InstallSodium(version string) error {
	return u.RunSudo("apt", "install", "-y", fmt.Sprintf("php%s-sodium", version))
}

// ConfigurePHPFPM configures PHP-FPM on Ubuntu/Debian
func (u *UbuntuInstaller) ConfigurePHPFPM(versions []string) error {
	// Get home directory for MageBox pools path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	for _, v := range versions {
		serviceName := fmt.Sprintf("php%s-fpm", v)
		fpmConfPath := fmt.Sprintf("/etc/php/%s/fpm/php-fpm.conf", v)
		wwwConfPath := fmt.Sprintf("/etc/php/%s/fpm/pool.d/www.conf", v)
		mageboxPoolsInclude := fmt.Sprintf("include=%s/.magebox/php/pools/%s/*.conf", homeDir, v)

		// Disable default www.conf pool - MageBox manages its own pools
		// This prevents "Permission denied" errors on the default socket path
		if u.FileExists(wwwConfPath) {
			disabledPath := wwwConfPath + ".disabled"
			if !u.FileExists(disabledPath) {
				if err := u.RunSudo("mv", wwwConfPath, disabledPath); err != nil {
					// Non-fatal, just warn
					fmt.Printf("  Warning: could not disable default www.conf pool: %v\n", err)
				}
			}
		}

		// Add MageBox pools include to php-fpm.conf if not already present
		if u.FileExists(fpmConfPath) {
			// Check if include already exists
			checkCmd := exec.Command("grep", "-q", mageboxPoolsInclude, fpmConfPath)
			if checkCmd.Run() != nil {
				// Include not found, add it
				if err := u.RunSudo("sh", "-c", fmt.Sprintf("echo '%s' >> %s", mageboxPoolsInclude, fpmConfPath)); err != nil {
					return fmt.Errorf("failed to add MageBox pools include to %s: %w", fpmConfPath, err)
				}
			}
		}

		// Create MageBox pools directory if it doesn't exist
		poolsDir := fmt.Sprintf("%s/.magebox/php/pools/%s", homeDir, v)
		if err := os.MkdirAll(poolsDir, 0755); err != nil {
			return fmt.Errorf("failed to create pools directory %s: %w", poolsDir, err)
		}

		// Enable and start service
		// Note: We use default log paths to avoid permission issues
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

		// Increase worker_connections for better performance (default 1024 is too low for Magento)
		_ = u.RunSudo("sed", "-i", "s/worker_connections.*/worker_connections 4096;/", nginxConf)
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
# Blackfire profiler
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl start blackfire-agent
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl stop blackfire-agent
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart blackfire-agent
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl enable blackfire-agent
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/apt install -y blackfire*
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/apt install -y tideways*
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

// ConfigureSELinux is a no-op on Ubuntu (SELinux typically not used)
func (u *UbuntuInstaller) ConfigureSELinux() error {
	return nil
}

// SetupDNS configures DNS resolution for local domains
func (u *UbuntuInstaller) SetupDNS() error {
	// Get configured TLD
	homeDir, _ := os.UserHomeDir()
	globalCfg, _ := config.LoadGlobalConfig(homeDir)
	tld := globalCfg.GetTLD()

	// Create MageBox dnsmasq config
	configDir := "/etc/dnsmasq.d"
	if err := u.RunSudo("mkdir", "-p", configDir); err != nil {
		return err
	}

	mageboxConf := fmt.Sprintf(`# MageBox - Resolve *.%s to localhost
address=/%s/127.0.0.1
address=/%s/::1
listen-address=127.0.0.2
port=53
bind-interfaces
`, tld, tld, tld)
	if err := u.WriteFile("/etc/dnsmasq.d/magebox.conf", mageboxConf); err != nil {
		return err
	}

	// Check if systemd-resolved is running (common on Ubuntu 18.04+)
	cmd := exec.Command("systemctl", "is-active", "systemd-resolved")
	if cmd.Run() == nil {
		// Configure systemd-resolved to use dnsmasq for the TLD
		resolvedDir := "/etc/systemd/resolved.conf.d"
		if err := u.RunSudo("mkdir", "-p", resolvedDir); err != nil {
			return err
		}

		resolvedConfig := fmt.Sprintf(`[Resolve]
DNS=127.0.0.2
Domains=~%s
`, tld)
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

// ConfigurePHPINI sets Magento-friendly PHP INI defaults for Ubuntu/Debian
func (u *UbuntuInstaller) ConfigurePHPINI(versions []string) error {
	for _, version := range versions {
		// Ubuntu/Debian with Ondrej PPA uses /etc/php/8.2/cli/php.ini
		iniPath := fmt.Sprintf("/etc/php/%s/cli/php.ini", version)

		if !u.FileExists(iniPath) {
			continue
		}

		// Set memory_limit=-1 for CLI (unlimited for Magento compile/deploy)
		if err := u.RunSudo("sed", "-i", "s/^memory_limit = .*/memory_limit = -1/", iniPath); err != nil {
			return fmt.Errorf("failed to set memory_limit in %s: %w", iniPath, err)
		}

		// Set max_execution_time for long-running CLI scripts
		if err := u.RunSudo("sed", "-i", "s/^max_execution_time = .*/max_execution_time = 18000/", iniPath); err != nil {
			return fmt.Errorf("failed to set max_execution_time in %s: %w", iniPath, err)
		}
	}
	return nil
}

// InstallBlackfire installs Blackfire agent and PHP extension for all versions
func (u *UbuntuInstaller) InstallBlackfire(versions []string) error {
	// Add Blackfire GPG key and repository
	if err := u.RunCommand("curl -sSL https://packages.blackfire.io/gpg.key | sudo gpg --dearmor -o /usr/share/keyrings/blackfire-archive-keyring.gpg"); err != nil {
		return fmt.Errorf("failed to add Blackfire GPG key: %w", err)
	}

	repoLine := "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/blackfire-archive-keyring.gpg] https://packages.blackfire.io/debian any main"
	if err := u.RunCommand(fmt.Sprintf("echo '%s' | sudo tee /etc/apt/sources.list.d/blackfire.list", repoLine)); err != nil {
		return fmt.Errorf("failed to add Blackfire repository: %w", err)
	}

	// Update apt cache
	if err := u.RunSudo("apt", "update"); err != nil {
		return fmt.Errorf("failed to update apt: %w", err)
	}

	// Install Blackfire agent
	if err := u.RunSudo("apt", "install", "-y", "blackfire"); err != nil {
		return fmt.Errorf("failed to install Blackfire agent: %w", err)
	}

	// Install Blackfire PHP extension for each version
	for _, version := range versions {
		pkgName := fmt.Sprintf("blackfire-php%s", strings.ReplaceAll(version, ".", ""))
		if err := u.RunSudo("apt", "install", "-y", pkgName); err != nil {
			// Don't fail if extension not available for this PHP version
			continue
		}
	}

	return nil
}

// InstallTideways installs Tideways PHP extension for all versions
func (u *UbuntuInstaller) InstallTideways(versions []string) error {
	// Add Tideways GPG key and repository
	if err := u.RunCommand("curl -sSL https://packages.tideways.com/key.gpg | sudo gpg --dearmor -o /usr/share/keyrings/tideways-archive-keyring.gpg"); err != nil {
		return fmt.Errorf("failed to add Tideways GPG key: %w", err)
	}

	repoLine := "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/tideways-archive-keyring.gpg] https://packages.tideways.com/apt-packages any-version main"
	if err := u.RunCommand(fmt.Sprintf("echo '%s' | sudo tee /etc/apt/sources.list.d/tideways.list", repoLine)); err != nil {
		return fmt.Errorf("failed to add Tideways repository: %w", err)
	}

	// Update apt cache
	if err := u.RunSudo("apt", "update"); err != nil {
		return fmt.Errorf("failed to update apt: %w", err)
	}

	// Install Tideways PHP extension for each version
	for _, version := range versions {
		pkgName := fmt.Sprintf("tideways-php-%s", version)
		if err := u.RunSudo("apt", "install", "-y", pkgName); err != nil {
			// Don't fail if extension not available for this PHP version
			continue
		}
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

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
		fmt.Sprintf("php%s-php-pecl-imagick-im7", remiVersion), // ImageMagick 7 on Fedora
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

// InstallXdebug installs Xdebug for a specific PHP version
func (f *FedoraInstaller) InstallXdebug(version string) error {
	remiVersion := strings.ReplaceAll(version, ".", "")
	return f.RunSudo("dnf", "install", "-y", fmt.Sprintf("php%s-php-xdebug", remiVersion))
}

// InstallImagick installs ImageMagick PHP extension for a specific PHP version
func (f *FedoraInstaller) InstallImagick(version string) error {
	remiVersion := strings.ReplaceAll(version, ".", "")
	return f.RunSudo("dnf", "install", "-y", fmt.Sprintf("php%s-php-pecl-imagick-im7", remiVersion))
}

// InstallSodium installs the sodium PHP extension for a specific PHP version
// Required for Argon2i password hashing in Magento
func (f *FedoraInstaller) InstallSodium(version string) error {
	remiVersion := strings.ReplaceAll(version, ".", "")
	return f.RunSudo("dnf", "install", "-y", fmt.Sprintf("php%s-php-sodium", remiVersion))
}

// ConfigurePHPFPM configures PHP-FPM on Fedora
func (f *FedoraInstaller) ConfigurePHPFPM(versions []string) error {
	// Get home directory for MageBox pools path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	for _, v := range versions {
		remiVersion := strings.ReplaceAll(v, ".", "")
		fpmConfPath := fmt.Sprintf("/etc/opt/remi/php%s/php-fpm.conf", remiVersion)
		wwwConfPath := fmt.Sprintf("/etc/opt/remi/php%s/php-fpm.d/www.conf", remiVersion)
		mageboxPoolsInclude := fmt.Sprintf("include=%s/.magebox/php/pools/%s/*.conf", homeDir, v)

		// Disable default www.conf pool - MageBox manages its own pools
		// This prevents "Permission denied" errors on the default socket path
		if f.FileExists(wwwConfPath) {
			disabledPath := wwwConfPath + ".disabled"
			if !f.FileExists(disabledPath) {
				if err := f.RunSudo("mv", wwwConfPath, disabledPath); err != nil {
					// Non-fatal, just warn
					fmt.Printf("  Warning: could not disable default www.conf pool: %v\n", err)
				}
			}
		}

		// Add MageBox pools include to php-fpm.conf if not already present
		if f.FileExists(fpmConfPath) {
			// Check if include already exists
			checkCmd := exec.Command("grep", "-q", mageboxPoolsInclude, fpmConfPath)
			if checkCmd.Run() != nil {
				// Include not found, add it
				if err := f.RunSudo("sh", "-c", fmt.Sprintf("echo '%s' >> %s", mageboxPoolsInclude, fpmConfPath)); err != nil {
					return fmt.Errorf("failed to add MageBox pools include to %s: %w", fpmConfPath, err)
				}
			}
		}

		// Create MageBox pools directory if it doesn't exist
		poolsDir := fmt.Sprintf("%s/.magebox/php/pools/%s", homeDir, v)
		if err := os.MkdirAll(poolsDir, 0755); err != nil {
			return fmt.Errorf("failed to create pools directory %s: %w", poolsDir, err)
		}

		// NOTE: On Fedora, MageBox manages PHP-FPM directly to avoid SELinux httpd_t
		// restrictions on user home directories. Do not enable/start systemd here.
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

		// Increase worker_connections for better performance (default 1024 is too low for Magento)
		_ = f.RunSudo("sed", "-i", "s/worker_connections.*/worker_connections 4096;/", nginxConf)
	}

	// Fix nginx directory permissions for client body uploads and caching
	// This prevents "Permission denied" errors when uploading files via POST
	// Since nginx runs as current user, it needs ownership of the entire nginx lib directory
	if currentUser != "" {
		_ = f.RunSudo("chown", "-R", currentUser+":"+currentUser, "/var/lib/nginx/")
		_ = f.RunSudo("chmod", "-R", "755", "/var/lib/nginx/")

		// Create tmpfiles.d config for persistent permissions across reboots/restarts
		// Without this, systemd recreates /var/lib/nginx/tmp with wrong permissions
		tmpfilesContent := fmt.Sprintf(`d /var/lib/nginx/tmp 0755 %s %s -
d /var/lib/nginx/tmp/client_body 0755 %s %s -
d /var/lib/nginx/tmp/fastcgi 0755 %s %s -
d /var/lib/nginx/tmp/proxy 0755 %s %s -
d /var/lib/nginx/tmp/scgi 0755 %s %s -
d /var/lib/nginx/tmp/uwsgi 0755 %s %s -
`, currentUser, currentUser, currentUser, currentUser, currentUser, currentUser, currentUser, currentUser, currentUser, currentUser, currentUser, currentUser)
		_ = f.WriteFile("/etc/tmpfiles.d/nginx-magebox.conf", tmpfilesContent)
		_ = f.RunSudo("systemd-tmpfiles", "--create", "/etc/tmpfiles.d/nginx-magebox.conf")

		// Restore SELinux context after changing ownership
		if f.CommandExists("restorecon") {
			_ = f.RunSudo("restorecon", "-Rv", "/var/lib/nginx/")
		}
	}

	// Enable nginx on boot
	if err := f.RunSudo("systemctl", "enable", "nginx"); err != nil {
		return err
	}

	return nil
}

// ConfigureSELinux configures SELinux for nginx proxy and config access
func (f *FedoraInstaller) ConfigureSELinux() error {
	// Check if SELinux is enabled
	if !f.CommandExists("getenforce") {
		return nil // SELinux not installed
	}

	// SELinux configuration for MageBox development environment
	// - httpd_can_network_connect: Allow nginx to proxy to Docker containers
	// - httpd_read_user_content: Allow nginx to serve files from home directories
	// - httpd_enable_homedirs: Allow PHP-FPM/nginx to access user home directories
	// - httpd_unified: Allow PHP to write to any httpd-accessible location (dev mode)
	if f.CommandExists("setsebool") {
		_ = f.RunSudo("setsebool", "-P", "httpd_can_network_connect", "on")
		_ = f.RunSudo("setsebool", "-P", "httpd_read_user_content", "on")
		_ = f.RunSudo("setsebool", "-P", "httpd_enable_homedirs", "on")
		_ = f.RunSudo("setsebool", "-P", "httpd_unified", "on")
	}

	// Get home directory for SELinux context on MageBox configs
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil // Skip if can't get home dir
	}

	mageboxDir := homeDir + "/.magebox"

	// Create run directory if it doesn't exist
	_ = os.MkdirAll(mageboxDir+"/run", 0755)

	// Set persistent SELinux fcontext rules using semanage (survives restorecon)
	if f.CommandExists("semanage") {
		// Allow httpd to access PHP-FPM sockets in ~/.magebox/run
		_ = f.RunSudo("semanage", "fcontext", "-a", "-t", "httpd_var_run_t", mageboxDir+"/run(/.*)?")
		_ = f.RunSudo("restorecon", "-Rv", mageboxDir+"/run")

		// Allow httpd to access nginx configs in ~/.magebox/nginx
		_ = f.RunSudo("semanage", "fcontext", "-a", "-t", "httpd_config_t", mageboxDir+"/nginx(/.*)?")
		_ = f.RunSudo("restorecon", "-Rv", mageboxDir+"/nginx")

		// Allow httpd to access certs in ~/.magebox/certs
		_ = f.RunSudo("semanage", "fcontext", "-a", "-t", "httpd_config_t", mageboxDir+"/certs(/.*)?")
		_ = f.RunSudo("restorecon", "-Rv", mageboxDir+"/certs")

		// Allow httpd to write log files in ~/.magebox/logs
		_ = f.RunSudo("semanage", "fcontext", "-a", "-t", "httpd_log_t", mageboxDir+"/logs(/.*)?")
		_ = f.RunSudo("restorecon", "-Rv", mageboxDir+"/logs")

		// Fix Remi PHP-FPM run directories (use /opt path due to Fedora equivalency rule)
		// This allows PHP-FPM to create PID files in /var/opt/remi/php*/run/
		for _, v := range []string{"81", "82", "83", "84", "85"} {
			remiRunPath := fmt.Sprintf("/opt/remi/php%s/run(/.*)?", v)
			_ = f.RunSudo("semanage", "fcontext", "-a", "-t", "httpd_var_run_t", remiRunPath)
			_ = f.RunSudo("restorecon", "-Rv", fmt.Sprintf("/var/opt/remi/php%s/run", v))
		}
	} else if f.CommandExists("chcon") {
		// Fallback to chcon if semanage not available (temporary, won't survive restorecon)
		_ = f.RunSudo("chcon", "-R", "-t", "httpd_var_run_t", mageboxDir+"/run")
		_ = f.RunSudo("chcon", "-R", "-t", "httpd_config_t", mageboxDir+"/nginx")
		_ = f.RunSudo("chcon", "-R", "-t", "httpd_config_t", mageboxDir+"/certs")
		_ = f.RunSudo("chcon", "-R", "-t", "httpd_log_t", mageboxDir+"/logs")
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
# Allow editing /etc/hosts for DNS entries
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/tee -a /etc/hosts
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/sed -i * /etc/hosts
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/cp /tmp/magebox-hosts-* /etc/hosts
# Blackfire profiler
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl start blackfire-agent
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl stop blackfire-agent
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart blackfire-agent
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl enable blackfire-agent
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/dnf install -y blackfire*
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/dnf install -y tideways*
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

// SetupDNS configures DNS resolution for local domains
func (f *FedoraInstaller) SetupDNS() error {
	// Get configured TLD
	homeDir, _ := os.UserHomeDir()
	globalCfg, _ := config.LoadGlobalConfig(homeDir)
	tld := globalCfg.GetTLD()

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

	mageboxConf := fmt.Sprintf(`# MageBox - Resolve *.%s to localhost
address=/%s/127.0.0.1
address=/%s/::1
listen-address=127.0.0.2
port=53
bind-interfaces
`, tld, tld, tld)
	if err := f.WriteFile("/etc/dnsmasq.d/magebox.conf", mageboxConf); err != nil {
		return err
	}

	// Check if systemd-resolved is running
	cmd := exec.Command("systemctl", "is-active", "systemd-resolved")
	if cmd.Run() == nil {
		// Configure systemd-resolved to use dnsmasq for the TLD
		resolvedDir := "/etc/systemd/resolved.conf.d"
		if err := f.RunSudo("mkdir", "-p", resolvedDir); err != nil {
			return err
		}

		resolvedConfig := fmt.Sprintf(`[Resolve]
DNS=127.0.0.2
Domains=~%s
`, tld)
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

// ConfigurePHPINI sets Magento-friendly PHP INI defaults for Fedora Remi
func (f *FedoraInstaller) ConfigurePHPINI(versions []string) error {
	for _, version := range versions {
		remiVersion := strings.ReplaceAll(version, ".", "")
		iniPath := fmt.Sprintf("/etc/opt/remi/php%s/php.ini", remiVersion)

		if !f.FileExists(iniPath) {
			continue
		}

		// Set memory_limit=-1 for CLI (unlimited for Magento compile/deploy)
		if err := f.RunSudo("sed", "-i", "s/^memory_limit = .*/memory_limit = -1/", iniPath); err != nil {
			return fmt.Errorf("failed to set memory_limit in %s: %w", iniPath, err)
		}

		// Set max_execution_time for long-running CLI scripts
		if err := f.RunSudo("sed", "-i", "s/^max_execution_time = .*/max_execution_time = 18000/", iniPath); err != nil {
			return fmt.Errorf("failed to set max_execution_time in %s: %w", iniPath, err)
		}
	}
	return nil
}

// InstallBlackfire installs Blackfire agent and PHP extension for all versions
func (f *FedoraInstaller) InstallBlackfire(versions []string) error {
	// Import GPG key first to avoid "skipped OpenPGP checks" warning
	if err := f.RunSudo("rpm", "--import", "https://packages.blackfire.io/gpg.key"); err != nil {
		// Non-fatal, continue anyway
		fmt.Println("  Warning: could not import Blackfire GPG key")
	}

	// Add Blackfire repository
	repoURL := "https://packages.blackfire.io/fedora/blackfire.repo"
	if err := f.RunSudo("sh", "-c", fmt.Sprintf("curl -sSL %s -o /etc/yum.repos.d/blackfire.repo", repoURL)); err != nil {
		return fmt.Errorf("failed to add Blackfire repository: %w", err)
	}

	// Install Blackfire agent and PHP extension
	// Note: Fedora repo has single 'blackfire-php' package (not versioned like Ubuntu)
	if err := f.RunSudo("dnf", "install", "-y", "blackfire", "blackfire-php"); err != nil {
		return fmt.Errorf("failed to install Blackfire: %w", err)
	}

	return nil
}

// InstallTideways installs Tideways PHP extension for all versions
func (f *FedoraInstaller) InstallTideways(versions []string) error {
	// Import GPG key first
	_ = f.RunCommandSilent("sudo rpm --import https://packages.tideways.com/key.gpg 2>/dev/null")

	// dnf5 (Fedora 41+) has issues with cloudsmith repos, so we download RPMs directly
	// Latest versions as of Dec 2025
	phpPkg := "https://packages.tideways.com/rpm-packages/x86_64/tideways-php-5.0.44-1.x86_64.rpm"
	daemonPkg := "https://packages.tideways.com/rpm-packages/x86_64/tideways-daemon-1.9.48-1.x86_64.rpm"
	cliPkg := "https://packages.tideways.com/rpm-packages/x86_64/tideways-cli-1.3.24-1.x86_64.rpm"

	// Install directly from URLs (dnf handles this well)
	_ = f.RunCommandSilent(fmt.Sprintf("sudo dnf install -y -q %s %s %s 2>/dev/null", phpPkg, daemonPkg, cliPkg))

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

package platform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Type represents the operating system type
type Type string

// LinuxDistro represents the Linux distribution family
type LinuxDistro string

const (
	// Darwin represents macOS
	Darwin Type = "darwin"
	// Linux represents Linux
	Linux Type = "linux"
	// Unknown represents an unsupported platform
	Unknown Type = "unknown"
)

const (
	// DistroDebian represents Debian/Ubuntu family (uses apt)
	DistroDebian LinuxDistro = "debian"
	// DistroFedora represents Fedora/RHEL/CentOS family (uses dnf)
	DistroFedora LinuxDistro = "fedora"
	// DistroArch represents Arch Linux family (uses pacman)
	DistroArch LinuxDistro = "arch"
	// DistroUnknown represents an unknown distro
	DistroUnknown LinuxDistro = "unknown"
)

// Platform contains information about the current platform
type Platform struct {
	Type           Type
	Arch           string
	HomeDir        string
	IsAppleSilicon bool
	LinuxDistro    LinuxDistro
}

// Detect detects the current platform
func Detect() (*Platform, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	p := &Platform{
		Type:    getType(),
		Arch:    runtime.GOARCH,
		HomeDir: homeDir,
	}

	p.IsAppleSilicon = p.Type == Darwin && p.Arch == "arm64"

	// Detect Linux distribution
	if p.Type == Linux {
		p.LinuxDistro = detectLinuxDistro()
	}

	return p, nil
}

// getType returns the platform type based on GOOS
func getType() Type {
	switch runtime.GOOS {
	case "darwin":
		return Darwin
	case "linux":
		return Linux
	default:
		return Unknown
	}
}

// detectLinuxDistro detects the Linux distribution family
func detectLinuxDistro() LinuxDistro {
	// Read /etc/os-release to determine distro
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return DistroUnknown
	}

	content := strings.ToLower(string(data))

	// Check for Fedora/RHEL/CentOS family
	if strings.Contains(content, "id=fedora") ||
		strings.Contains(content, "id=rhel") ||
		strings.Contains(content, "id=centos") ||
		strings.Contains(content, "id=rocky") ||
		strings.Contains(content, "id=almalinux") {
		return DistroFedora
	}

	// Check for Debian/Ubuntu family
	if strings.Contains(content, "id=debian") ||
		strings.Contains(content, "id=ubuntu") ||
		strings.Contains(content, "id=linuxmint") ||
		strings.Contains(content, "id=pop") {
		return DistroDebian
	}

	// Check for Arch family
	if strings.Contains(content, "id=arch") ||
		strings.Contains(content, "id=manjaro") ||
		strings.Contains(content, "id=endeavouros") {
		return DistroArch
	}

	return DistroUnknown
}

// IsSupported returns true if the platform is supported
func (p *Platform) IsSupported() bool {
	return p.Type == Darwin || p.Type == Linux
}

// MageBoxDir returns the path to the MageBox configuration directory
func (p *Platform) MageBoxDir() string {
	return filepath.Join(p.HomeDir, ".magebox")
}

// NginxConfigDir returns the path to the Nginx configuration directory
func (p *Platform) NginxConfigDir() string {
	switch p.Type {
	case Darwin:
		if p.IsAppleSilicon {
			return "/opt/homebrew/etc/nginx"
		}
		return "/usr/local/etc/nginx"
	case Linux:
		return "/etc/nginx"
	default:
		return ""
	}
}

// NginxBinary returns the path to the Nginx binary
func (p *Platform) NginxBinary() string {
	switch p.Type {
	case Darwin:
		if p.IsAppleSilicon {
			return "/opt/homebrew/bin/nginx"
		}
		return "/usr/local/bin/nginx"
	case Linux:
		return "/usr/sbin/nginx"
	default:
		return ""
	}
}

// PHPFPMConfigDir returns the base path for PHP-FPM pool configurations
func (p *Platform) PHPFPMConfigDir(version string) string {
	normalizedVersion := normalizeVersion(version)
	switch p.Type {
	case Darwin:
		base := "/usr/local"
		if p.IsAppleSilicon {
			base = "/opt/homebrew"
		}
		return filepath.Join(base, "etc", "php", normalizedVersion, "php-fpm.d")
	case Linux:
		switch p.LinuxDistro {
		case DistroFedora:
			// Remi uses php82, php83, etc. format
			remiVersion := strings.ReplaceAll(normalizedVersion, ".", "")
			return fmt.Sprintf("/etc/opt/remi/php%s/php-fpm.d", remiVersion)
		default:
			// Debian/Ubuntu uses php8.2, php8.3 format
			return filepath.Join("/etc", "php", normalizedVersion, "fpm", "pool.d")
		}
	default:
		return ""
	}
}

// PHPFPMBinary returns the path to the PHP-FPM binary for a specific version
// On macOS, uses Cellar path directly (more reliable than opt symlinks)
func (p *Platform) PHPFPMBinary(version string) string {
	normalizedVersion := normalizeVersion(version)
	switch p.Type {
	case Darwin:
		base := "/usr/local"
		if p.IsAppleSilicon {
			base = "/opt/homebrew"
		}
		// Use Cellar path with glob to find actual installation
		cellarPath := filepath.Join(base, "Cellar", "php@"+normalizedVersion)
		matches, err := filepath.Glob(filepath.Join(cellarPath, "*", "sbin", "php-fpm"))
		if err == nil && len(matches) > 0 {
			return matches[0]
		}
		// Fallback to opt symlink (for backwards compatibility)
		return filepath.Join(base, "opt", "php@"+normalizedVersion, "sbin", "php-fpm")
	case Linux:
		switch p.LinuxDistro {
		case DistroFedora:
			// Remi uses php82-php-fpm format, binary is at /usr/bin/php-fpm82 or /opt/remi/php82/root/usr/sbin/php-fpm
			remiVersion := strings.ReplaceAll(normalizedVersion, ".", "")
			return fmt.Sprintf("/opt/remi/php%s/root/usr/sbin/php-fpm", remiVersion)
		default:
			// Debian/Ubuntu uses php-fpm8.2 format
			return fmt.Sprintf("/usr/sbin/php-fpm%s", normalizedVersion)
		}
	default:
		return ""
	}
}

// PHPBinary returns the path to the PHP CLI binary for a specific version
// On macOS, uses Cellar path directly (more reliable than opt symlinks)
func (p *Platform) PHPBinary(version string) string {
	normalizedVersion := normalizeVersion(version)
	switch p.Type {
	case Darwin:
		base := "/usr/local"
		if p.IsAppleSilicon {
			base = "/opt/homebrew"
		}
		// Use Cellar path with glob to find actual installation
		cellarPath := filepath.Join(base, "Cellar", "php@"+normalizedVersion)
		matches, err := filepath.Glob(filepath.Join(cellarPath, "*", "bin", "php"))
		if err == nil && len(matches) > 0 {
			return matches[0]
		}
		// Fallback to opt symlink (for backwards compatibility)
		return filepath.Join(base, "opt", "php@"+normalizedVersion, "bin", "php")
	case Linux:
		switch p.LinuxDistro {
		case DistroFedora:
			// Remi installs as php82, php83, etc. in /usr/bin/
			remiVersion := strings.ReplaceAll(normalizedVersion, ".", "")
			return fmt.Sprintf("/usr/bin/php%s", remiVersion)
		default:
			// Debian/Ubuntu uses php8.2, php8.3 format
			return fmt.Sprintf("/usr/bin/php%s", normalizedVersion)
		}
	default:
		return ""
	}
}

// VarnishBinary returns the path to the Varnish binary
func (p *Platform) VarnishBinary() string {
	switch p.Type {
	case Darwin:
		if p.IsAppleSilicon {
			return "/opt/homebrew/sbin/varnishd"
		}
		return "/usr/local/sbin/varnishd"
	case Linux:
		return "/usr/sbin/varnishd"
	default:
		return ""
	}
}

// VarnishConfigDir returns the path to the Varnish configuration directory
func (p *Platform) VarnishConfigDir() string {
	switch p.Type {
	case Darwin:
		if p.IsAppleSilicon {
			return "/opt/homebrew/etc/varnish"
		}
		return "/usr/local/etc/varnish"
	case Linux:
		return "/etc/varnish"
	default:
		return ""
	}
}

// HostsFilePath returns the path to the hosts file
func (p *Platform) HostsFilePath() string {
	return "/etc/hosts"
}

// PHPInstallCommand returns the command to install a specific PHP version
func (p *Platform) PHPInstallCommand(version string) string {
	normalizedVersion := normalizeVersion(version)                // e.g., "8.2"
	remiVersion := strings.ReplaceAll(normalizedVersion, ".", "") // e.g., "82" for Remi packages
	switch p.Type {
	case Darwin:
		return fmt.Sprintf("brew install php@%s", normalizedVersion)
	case Linux:
		switch p.LinuxDistro {
		case DistroFedora:
			// Fedora/RHEL uses Remi repository with format php82, php83, etc.
			return fmt.Sprintf("sudo dnf install -y php%s-php-fpm php%s-php-cli php%s-php-common php%s-php-mysqlnd php%s-php-xml php%s-php-mbstring php%s-php-zip php%s-php-gd php%s-php-intl php%s-php-bcmath php%s-php-soap php%s-php-opcache",
				remiVersion, remiVersion, remiVersion, remiVersion,
				remiVersion, remiVersion, remiVersion, remiVersion,
				remiVersion, remiVersion, remiVersion, remiVersion)
		case DistroArch:
			// Arch doesn't have versioned PHP packages by default
			return "sudo pacman -S php php-fpm php-gd php-intl php-sodium"
		default:
			// Debian/Ubuntu uses Ondrej PPA with format php8.2, php8.3, etc.
			return fmt.Sprintf("sudo add-apt-repository -y ppa:ondrej/php && sudo apt install -y php%s-fpm php%s-cli php%s-common php%s-mysql php%s-xml php%s-curl php%s-mbstring php%s-zip php%s-gd php%s-intl php%s-bcmath php%s-soap",
				normalizedVersion, normalizedVersion, normalizedVersion, normalizedVersion,
				normalizedVersion, normalizedVersion, normalizedVersion, normalizedVersion,
				normalizedVersion, normalizedVersion, normalizedVersion, normalizedVersion)
		}
	default:
		return ""
	}
}

// NginxInstallCommand returns the command to install Nginx
func (p *Platform) NginxInstallCommand() string {
	switch p.Type {
	case Darwin:
		return "brew install nginx"
	case Linux:
		switch p.LinuxDistro {
		case DistroFedora:
			return "sudo dnf install -y nginx"
		case DistroArch:
			return "sudo pacman -S nginx"
		default:
			return "sudo apt install -y nginx"
		}
	default:
		return ""
	}
}

// VarnishInstallCommand returns the command to install Varnish
func (p *Platform) VarnishInstallCommand() string {
	switch p.Type {
	case Darwin:
		return "brew install varnish"
	case Linux:
		switch p.LinuxDistro {
		case DistroFedora:
			return "sudo dnf install -y varnish"
		case DistroArch:
			return "sudo pacman -S varnish"
		default:
			return "sudo apt install -y varnish"
		}
	default:
		return ""
	}
}

// MkcertInstallCommand returns the command to install mkcert
func (p *Platform) MkcertInstallCommand() string {
	switch p.Type {
	case Darwin:
		return "brew install mkcert nss"
	case Linux:
		switch p.LinuxDistro {
		case DistroFedora:
			return "sudo dnf install -y mkcert nss-tools"
		case DistroArch:
			return "sudo pacman -S mkcert nss"
		default:
			return "sudo apt install -y mkcert libnss3-tools"
		}
	default:
		return ""
	}
}

// DockerInstallCommand returns the command to install Docker
func (p *Platform) DockerInstallCommand() string {
	switch p.Type {
	case Darwin:
		return "brew install --cask docker"
	case Linux:
		// Docker's convenience script works on most distros
		return "curl -fsSL https://get.docker.com | sudo sh"
	default:
		return ""
	}
}

// PackageManager returns the package manager command for this platform
func (p *Platform) PackageManager() string {
	switch p.Type {
	case Darwin:
		return "brew"
	case Linux:
		switch p.LinuxDistro {
		case DistroFedora:
			return "dnf"
		case DistroArch:
			return "pacman"
		default:
			return "apt"
		}
	default:
		return ""
	}
}

// CommandExists checks if a command exists in the system PATH
func CommandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// BinaryExists checks if a binary exists at the specified path
func BinaryExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// normalizeVersion normalizes a PHP version string (e.g., "8.2" stays "8.2", "82" becomes "8.2")
func normalizeVersion(version string) string {
	version = strings.TrimPrefix(version, "php")
	version = strings.TrimPrefix(version, "PHP")

	// If it's already in "X.Y" format, return as-is
	if strings.Contains(version, ".") {
		return version
	}

	// Convert "82" to "8.2"
	if len(version) == 2 {
		return string(version[0]) + "." + string(version[1])
	}

	return version
}

// GetInstalledPHPVersions returns a list of installed PHP versions
func (p *Platform) GetInstalledPHPVersions() []string {
	versions := []string{"8.1", "8.2", "8.3", "8.4"}
	installed := make([]string, 0)

	for _, v := range versions {
		if BinaryExists(p.PHPBinary(v)) || BinaryExists(p.PHPFPMBinary(v)) {
			installed = append(installed, v)
		}
	}

	return installed
}

// IsNginxInstalled checks if Nginx is installed
func (p *Platform) IsNginxInstalled() bool {
	return BinaryExists(p.NginxBinary()) || CommandExists("nginx")
}

// IsVarnishInstalled checks if Varnish is installed
func (p *Platform) IsVarnishInstalled() bool {
	return BinaryExists(p.VarnishBinary()) || CommandExists("varnishd")
}

// IsMkcertInstalled checks if mkcert is installed
func (p *Platform) IsMkcertInstalled() bool {
	return CommandExists("mkcert")
}

// IsDockerInstalled checks if Docker is installed
func (p *Platform) IsDockerInstalled() bool {
	return CommandExists("docker")
}

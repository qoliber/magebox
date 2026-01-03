package platform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"qoliber/magebox/internal/verbose"
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
	DistroName     string // Actual distro name (e.g., "endeavouros", "rocky")
	DistroTested   bool   // Whether this specific distro has been tested
}

// Detect detects the current platform
func Detect() (*Platform, error) {
	verbose.Debug("Detecting platform...")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	p := &Platform{
		Type:    getType(),
		Arch:    runtime.GOARCH,
		HomeDir: homeDir,
	}

	verbose.Debug("Platform type: %s, arch: %s", p.Type, p.Arch)
	verbose.Debug("Home directory: %s", homeDir)

	p.IsAppleSilicon = p.Type == Darwin && p.Arch == "arm64"
	if p.IsAppleSilicon {
		verbose.Debug("Apple Silicon detected")
	}

	// Detect Linux distribution
	if p.Type == Linux {
		p.LinuxDistro, p.DistroName, p.DistroTested = detectLinuxDistro()
		verbose.Debug("Linux distro: %s (family: %s, tested: %v)", p.DistroName, p.LinuxDistro, p.DistroTested)
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

// Tested distributions - these have been verified to work with MageBox
var testedDistros = map[string]bool{
	"fedora":  true,
	"ubuntu":  true,
	"debian":  true,
	"arch":    true,
	"rocky":   true,
	"rhel":    true,
	"centos":  true,
	"manjaro": true,
}

// detectLinuxDistro detects the Linux distribution family
// Returns: distro family, distro name, and whether it's been tested
func detectLinuxDistro() (LinuxDistro, string, bool) {
	verbose.Debug("Reading /etc/os-release...")

	// Read /etc/os-release to determine distro
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		verbose.Debug("Failed to read /etc/os-release: %v", err)
		return DistroUnknown, "unknown", false
	}

	// Parse os-release into key=value map
	osRelease := parseOSRelease(string(data))

	id := strings.ToLower(osRelease["ID"])
	idLike := strings.ToLower(osRelease["ID_LIKE"])
	prettyName := osRelease["PRETTY_NAME"]
	if prettyName == "" {
		prettyName = id
	}

	verbose.Debug("os-release: ID=%s, ID_LIKE=%s, PRETTY_NAME=%s", id, idLike, prettyName)

	// Check if this specific distro has been tested
	tested := testedDistros[id]
	verbose.Debug("Distro %s tested: %v", id, tested)

	// Check for Fedora/RHEL/CentOS family
	if id == "fedora" || id == "rhel" || id == "centos" || id == "rocky" || id == "almalinux" ||
		strings.Contains(idLike, "fedora") || strings.Contains(idLike, "rhel") {
		verbose.Debug("Matched Fedora/RHEL family")
		return DistroFedora, prettyName, tested
	}

	// Check for Debian/Ubuntu family
	if id == "debian" || id == "ubuntu" || id == "linuxmint" || id == "pop" ||
		strings.Contains(idLike, "debian") || strings.Contains(idLike, "ubuntu") {
		verbose.Debug("Matched Debian/Ubuntu family")
		return DistroDebian, prettyName, tested
	}

	// Check for Arch family
	if id == "arch" || id == "manjaro" || id == "endeavouros" || id == "garuda" || id == "artix" ||
		strings.Contains(idLike, "arch") {
		verbose.Debug("Matched Arch family")
		return DistroArch, prettyName, tested
	}

	verbose.Debug("No known distro family matched")
	return DistroUnknown, prettyName, false
}

// parseOSRelease parses /etc/os-release content into a map
func parseOSRelease(content string) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := parts[0]
			value := strings.Trim(parts[1], "\"'")
			result[key] = value
		}
	}
	return result
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

package php

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/qoliber/magebox/internal/platform"
)

// SupportedVersions lists all PHP versions that MageBox supports
var SupportedVersions = []string{"8.1", "8.2", "8.3", "8.4", "8.5"}

// Version represents an installed PHP version with its paths
type Version struct {
	Version    string
	PHPBinary  string
	FPMBinary  string
	FPMConfig  string
	Installed  bool
	FPMRunning bool
}

// Detector handles PHP version detection
type Detector struct {
	platform *platform.Platform
}

// NewDetector creates a new PHP detector
func NewDetector(p *platform.Platform) *Detector {
	return &Detector{platform: p}
}

// DetectAll detects all installed PHP versions
func (d *Detector) DetectAll() []Version {
	versions := make([]Version, 0, len(SupportedVersions))

	for _, v := range SupportedVersions {
		version := d.Detect(v)
		versions = append(versions, version)
	}

	return versions
}

// DetectInstalled returns only installed PHP versions
func (d *Detector) DetectInstalled() []Version {
	all := d.DetectAll()
	installed := make([]Version, 0)

	for _, v := range all {
		if v.Installed {
			installed = append(installed, v)
		}
	}

	return installed
}

// Detect checks if a specific PHP version is installed
func (d *Detector) Detect(version string) Version {
	v := Version{
		Version:   normalizeVersion(version),
		PHPBinary: d.platform.PHPBinary(version),
		FPMBinary: d.platform.PHPFPMBinary(version),
		FPMConfig: d.platform.PHPFPMConfigDir(version),
	}

	// Check if PHP binary exists
	v.Installed = platform.BinaryExists(v.PHPBinary) || platform.BinaryExists(v.FPMBinary)

	// If not found at expected path, try to find via command
	if !v.Installed {
		v.Installed = d.detectViaCommand(version)
	}

	// Check if FPM is running
	if v.Installed {
		v.FPMRunning = d.isFPMRunning(version)
	}

	return v
}

// detectViaCommand tries to detect PHP version via php command
func (d *Detector) detectViaCommand(version string) bool {
	// Try php{version} command
	cmd := exec.Command(fmt.Sprintf("php%s", normalizeVersion(version)), "-v")
	if err := cmd.Run(); err == nil {
		return true
	}

	return false
}

// isFPMRunning checks if PHP-FPM is running for the given version
func (d *Detector) isFPMRunning(version string) bool {
	normalized := normalizeVersion(version)

	// Check via pgrep
	cmd := exec.Command("pgrep", "-f", fmt.Sprintf("php-fpm.*%s", normalized))
	if err := cmd.Run(); err == nil {
		return true
	}

	// Check via systemctl on Linux
	if d.platform.Type == platform.Linux {
		cmd := exec.Command("systemctl", "is-active", fmt.Sprintf("php%s-fpm", normalized))
		output, err := cmd.Output()
		if err == nil && strings.TrimSpace(string(output)) == "active" {
			return true
		}
	}

	return false
}

// IsVersionInstalled checks if a specific version is installed
func (d *Detector) IsVersionInstalled(version string) bool {
	v := d.Detect(version)
	return v.Installed
}

// GetActualVersion runs php -v and parses the actual version string
func (d *Detector) GetActualVersion(version string) (string, error) {
	binary := d.platform.PHPBinary(version)

	// Try the expected binary path first
	if platform.BinaryExists(binary) {
		return d.runVersionCommand(binary)
	}

	// Try via command name
	cmdName := fmt.Sprintf("php%s", normalizeVersion(version))
	if platform.CommandExists(cmdName) {
		return d.runVersionCommand(cmdName)
	}

	return "", fmt.Errorf("PHP %s not found", version)
}

// runVersionCommand executes php -v and parses the version
func (d *Detector) runVersionCommand(binary string) (string, error) {
	cmd := exec.Command(binary, "-v")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Parse "PHP 8.2.15 (cli) ..." format
	re := regexp.MustCompile(`PHP (\d+\.\d+\.\d+)`)
	matches := re.FindStringSubmatch(string(output))
	if len(matches) < 2 {
		return "", fmt.Errorf("could not parse PHP version from output")
	}

	return matches[1], nil
}

// RequiredExtensions returns the list of required PHP extensions for Magento
func RequiredExtensions() []string {
	return []string{
		"bcmath",
		"ctype",
		"curl",
		"dom",
		"fileinfo",
		"gd",
		"hash",
		"iconv",
		"intl",
		"json",
		"libxml",
		"mbstring",
		"openssl",
		"pcre",
		"pdo_mysql",
		"simplexml",
		"soap",
		"sockets",
		"sodium",
		"spl",
		"tokenizer",
		"xml",
		"xmlwriter",
		"xsl",
		"zip",
		"zlib",
	}
}

// CheckExtensions checks which required extensions are missing
func (d *Detector) CheckExtensions(version string) (installed []string, missing []string, err error) {
	binary := d.platform.PHPBinary(version)

	// Get list of loaded extensions
	cmd := exec.Command(binary, "-m")
	output, err := cmd.Output()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get PHP modules: %w", err)
	}

	loadedExtensions := make(map[string]bool)
	for _, line := range strings.Split(string(output), "\n") {
		ext := strings.TrimSpace(strings.ToLower(line))
		if ext != "" && !strings.HasPrefix(ext, "[") {
			loadedExtensions[ext] = true
		}
	}

	required := RequiredExtensions()
	for _, ext := range required {
		if loadedExtensions[strings.ToLower(ext)] {
			installed = append(installed, ext)
		} else {
			missing = append(missing, ext)
		}
	}

	return installed, missing, nil
}

// Recommendation represents an installation recommendation
type Recommendation struct {
	Version        string
	InstallCommand string
	IsInstalled    bool
	IsFPMRunning   bool
}

// GetRecommendation returns installation recommendation for a PHP version
func (d *Detector) GetRecommendation(version string) Recommendation {
	v := d.Detect(version)

	return Recommendation{
		Version:        v.Version,
		InstallCommand: d.platform.PHPInstallCommand(version),
		IsInstalled:    v.Installed,
		IsFPMRunning:   v.FPMRunning,
	}
}

// normalizeVersion normalizes a PHP version string
func normalizeVersion(version string) string {
	version = strings.TrimPrefix(version, "php")
	version = strings.TrimPrefix(version, "PHP")

	if strings.Contains(version, ".") {
		return version
	}

	if len(version) == 2 {
		return string(version[0]) + "." + string(version[1])
	}

	return version
}

// FormatNotInstalledMessage formats a message for when PHP is not installed
func FormatNotInstalledMessage(version string, p *platform.Platform) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("✗ PHP %s not found\n\n", version))
	sb.WriteString("  Install it with:\n")

	switch p.Type {
	case platform.Darwin:
		sb.WriteString(fmt.Sprintf("    brew install php@%s\n", version))
	case platform.Linux:
		sb.WriteString(fmt.Sprintf("    sudo add-apt-repository ppa:ondrej/php\n"))
		sb.WriteString(fmt.Sprintf("    sudo apt install php%s-fpm php%s-cli php%s-common \\\n", version, version, version))
		sb.WriteString(fmt.Sprintf("      php%s-mysql php%s-xml php%s-curl php%s-mbstring \\\n", version, version, version, version))
		sb.WriteString(fmt.Sprintf("      php%s-zip php%s-gd php%s-intl php%s-bcmath php%s-soap\n", version, version, version, version, version))
	}

	sb.WriteString("\n  Then run: magebox start\n")

	return sb.String()
}

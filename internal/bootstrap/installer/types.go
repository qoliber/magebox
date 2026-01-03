// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package installer

import (
	"qoliber/magebox/internal/platform"
)

// SupportedVersions defines the OS versions supported by MageBox
var SupportedVersions = map[platform.Type]map[string][]string{
	platform.Darwin: {
		"macos": {"12", "13", "14", "15"}, // Monterey, Ventura, Sonoma, Sequoia
	},
	platform.Linux: {
		"fedora": {"38", "39", "40", "41", "42", "43"}, // Fedora 38-43
		"ubuntu": {"20.04", "22.04", "24.04"},          // LTS versions
		"debian": {"11", "12"},                         // Bullseye, Bookworm
		"arch":   {"rolling"},                          // Arch is rolling release
	},
}

// PHPVersions defines PHP versions to install for Magento compatibility
var PHPVersions = []string{"8.1", "8.2", "8.3", "8.4", "8.5"}

// RequiredPHPExtensions defines required PHP extensions for Magento
var RequiredPHPExtensions = []string{
	"bcmath", "cli", "common", "curl", "fpm", "gd", "intl",
	"mbstring", "mysql", "opcache", "soap", "sodium", "xml", "zip",
}

// OSVersionInfo contains OS version details
type OSVersionInfo struct {
	Name      string // e.g., "macOS", "Fedora", "Ubuntu"
	Version   string // e.g., "14.0", "40", "24.04"
	Codename  string // e.g., "Sonoma", "Noble"
	Supported bool
	Message   string // Warning or info message
}

// Installer is the interface for platform-specific installation
type Installer interface {
	// Platform returns the platform this installer handles
	Platform() platform.Type

	// Distro returns the Linux distribution (empty for Darwin)
	Distro() platform.LinuxDistro

	// ValidateOSVersion checks if the current OS version is supported
	ValidateOSVersion() (OSVersionInfo, error)

	// InstallPrerequisites installs system prerequisites (curl, git, etc.)
	InstallPrerequisites() error

	// InstallPHP installs a specific PHP version
	InstallPHP(version string) error

	// InstallNginx installs Nginx
	InstallNginx() error

	// InstallMkcert installs mkcert for SSL certificates
	InstallMkcert() error

	// InstallDocker installs Docker (instructions only - too complex for auto-install)
	InstallDocker() string

	// InstallDnsmasq installs dnsmasq for DNS resolution
	InstallDnsmasq() error

	// InstallMultitail installs multitail for log viewing
	InstallMultitail() error

	// InstallXdebug installs Xdebug for a specific PHP version
	InstallXdebug(version string) error

	// InstallImagick installs ImageMagick PHP extension for a specific PHP version
	InstallImagick(version string) error

	// InstallSodium installs the sodium PHP extension for a specific PHP version
	// This is required for Argon2i password hashing in Magento
	InstallSodium(version string) error

	// InstallBlackfire installs Blackfire agent and PHP extension for all versions
	InstallBlackfire(versions []string) error

	// InstallTideways installs Tideways PHP extension for all versions
	InstallTideways(versions []string) error

	// ConfigurePHPFPM configures PHP-FPM for the platform
	ConfigurePHPFPM(versions []string) error

	// ConfigureNginx configures Nginx for MageBox
	ConfigureNginx() error

	// ConfigureSudoers sets up passwordless sudo for services (Linux only)
	ConfigureSudoers() error

	// ConfigureSELinux configures SELinux for nginx (Fedora only)
	ConfigureSELinux() error

	// ConfigureShellPath adds ~/.magebox/bin to the user's shell PATH
	ConfigureShellPath() error

	// ConfigurePHPINI sets Magento-friendly PHP INI defaults (memory_limit, etc.)
	ConfigurePHPINI(versions []string) error

	// SetupDNS configures DNS resolution for .test domains
	SetupDNS() error

	// PackageManager returns the package manager name
	PackageManager() string

	// InstallCommand returns the install command format
	InstallCommand(packages ...string) string
}

// InstallResult tracks the result of an installation step
type InstallResult struct {
	Step    string
	Success bool
	Error   error
	Message string
}

// BootstrapProgress tracks overall bootstrap progress
type BootstrapProgress struct {
	CurrentStep  int
	TotalSteps   int
	StepName     string
	Results      []InstallResult
	Warnings     []string
	Errors       []string
	PHPInstalled []string
}

// NewProgress creates a new bootstrap progress tracker
func NewProgress(totalSteps int) *BootstrapProgress {
	return &BootstrapProgress{
		CurrentStep:  0,
		TotalSteps:   totalSteps,
		Results:      make([]InstallResult, 0),
		Warnings:     make([]string, 0),
		Errors:       make([]string, 0),
		PHPInstalled: make([]string, 0),
	}
}

// AddResult adds an installation result
func (p *BootstrapProgress) AddResult(step string, success bool, err error, message string) {
	p.Results = append(p.Results, InstallResult{
		Step:    step,
		Success: success,
		Error:   err,
		Message: message,
	})
	if err != nil {
		p.Errors = append(p.Errors, step+": "+err.Error())
	}
}

// AddWarning adds a warning message
func (p *BootstrapProgress) AddWarning(msg string) {
	p.Warnings = append(p.Warnings, msg)
}

// HasErrors returns true if there were any errors
func (p *BootstrapProgress) HasErrors() bool {
	return len(p.Errors) > 0
}

// HasWarnings returns true if there were any warnings
func (p *BootstrapProgress) HasWarnings() bool {
	return len(p.Warnings) > 0
}

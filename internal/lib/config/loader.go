// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package config

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
	"qoliber/magebox/internal/lib"
)

// Loader loads and parses installer configurations
type Loader struct {
	paths     *lib.Paths
	variables *Variables
}

// NewLoader creates a new config loader
func NewLoader(paths *lib.Paths) *Loader {
	return &Loader{
		paths:     paths,
		variables: NewVariables(),
	}
}

// DefaultLoader creates a loader with default paths
func DefaultLoader() (*Loader, error) {
	paths, err := lib.DefaultPaths()
	if err != nil {
		return nil, err
	}
	return NewLoader(paths), nil
}

// Variables returns the variables instance for customization
func (l *Loader) Variables() *Variables {
	return l.variables
}

// LoadInstaller loads an installer config by name (e.g., "fedora", "ubuntu")
func (l *Loader) LoadInstaller(name string) (*InstallerConfig, error) {
	// Get the installer path (checks local override first)
	path := l.paths.InstallerPath(name)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read installer config %s: %w", name, err)
	}

	var config InstallerConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse installer config %s: %w", name, err)
	}

	return &config, nil
}

// LoadCurrentPlatform loads the installer config for the current platform
func (l *Loader) LoadCurrentPlatform() (*InstallerConfig, error) {
	name := l.DetectPlatform()
	if name == "" {
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return l.LoadInstaller(name)
}

// DetectPlatform returns the installer name for the current platform
func (l *Loader) DetectPlatform() string {
	switch runtime.GOOS {
	case "darwin":
		return "darwin"
	case "linux":
		return l.detectLinuxDistro()
	default:
		return ""
	}
}

// detectLinuxDistro detects the Linux distribution
func (l *Loader) detectLinuxDistro() string {
	// Read /etc/os-release
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return ""
	}

	content := string(data)

	// Check for known distros
	if strings.Contains(content, "ID=fedora") ||
		strings.Contains(content, "ID=\"fedora\"") {
		return "fedora"
	}
	if strings.Contains(content, "ID=ubuntu") ||
		strings.Contains(content, "ID=\"ubuntu\"") {
		return "ubuntu"
	}
	if strings.Contains(content, "ID=debian") ||
		strings.Contains(content, "ID=\"debian\"") {
		return "ubuntu" // Use Ubuntu config for Debian
	}
	if strings.Contains(content, "ID=arch") ||
		strings.Contains(content, "ID=\"arch\"") {
		return "arch"
	}
	if strings.Contains(content, "ID=manjaro") {
		return "arch" // Use Arch config for Manjaro
	}

	return ""
}

// GetOSVersion returns the OS version from /etc/os-release
func (l *Loader) GetOSVersion() string {
	if runtime.GOOS == "darwin" {
		// macOS version detection would need sw_vers
		return ""
	}

	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "VERSION_ID=") {
			version := strings.TrimPrefix(line, "VERSION_ID=")
			version = strings.Trim(version, "\"")
			return version
		}
	}

	return ""
}

// ExpandConfig expands all variables in the config
// This creates a copy with all ${var} patterns replaced
func (l *Loader) ExpandConfig(config *InstallerConfig) *InstallerConfig {
	// For now, we expand strings as needed when accessing them
	// A full deep copy with expansion would be complex
	// Instead, callers should use l.Expand() on specific strings
	return config
}

// Expand expands variables in a string using the loader's variables
func (l *Loader) Expand(s string) string {
	return l.variables.Expand(s)
}

// ExpandSlice expands variables in a slice of strings
func (l *Loader) ExpandSlice(slice []string) []string {
	return l.variables.ExpandSlice(slice)
}

// SetupPHPVariables sets up PHP-related variables based on config and version
func (l *Loader) SetupPHPVariables(config *InstallerConfig, phpVersion string) {
	l.variables.SetPHPVersion(phpVersion)

	// Compute phpPrefix from version_format
	// e.g., "php${versionNoDot}" with version 8.2 -> "php82"
	versionNoDot := strings.ReplaceAll(phpVersion, ".", "")
	prefix := strings.ReplaceAll(config.PHP.VersionFormat, "${versionNoDot}", versionNoDot)
	prefix = strings.ReplaceAll(prefix, "${phpVersion}", phpVersion)
	l.variables.SetPHPPrefix(prefix)
}

// SetupOSVariables sets up OS-related variables
func (l *Loader) SetupOSVariables() {
	osVersion := l.GetOSVersion()
	if osVersion != "" {
		l.variables.SetOSVersion(osVersion)
	}
}

// IsLibInstalled returns true if the lib is installed
func (l *Loader) IsLibInstalled() bool {
	return l.paths.Exists()
}

// GetPHPPackages returns the expanded list of PHP packages for a version
func (l *Loader) GetPHPPackages(config *InstallerConfig, phpVersion string) []string {
	l.SetupPHPVariables(config, phpVersion)

	var packages []string
	packages = append(packages, l.ExpandSlice(config.PHP.Packages.Core)...)
	packages = append(packages, l.ExpandSlice(config.PHP.Packages.Extensions)...)

	return packages
}

// GetPHPBinary returns the expanded PHP binary path for a version
func (l *Loader) GetPHPBinary(config *InstallerConfig, phpVersion string) string {
	l.SetupPHPVariables(config, phpVersion)
	return l.Expand(config.PHP.Paths.Binary)
}

// GetPHPFPMService returns the expanded FPM service name for a version
func (l *Loader) GetPHPFPMService(config *InstallerConfig, phpVersion string) string {
	l.SetupPHPVariables(config, phpVersion)
	return l.Expand(config.PHP.Services.FPM.Name)
}

// GetSudoersRules returns the expanded sudoers rules
func (l *Loader) GetSudoersRules(config *InstallerConfig) []string {
	return l.ExpandSlice(config.Sudoers.Rules)
}

// GetSELinuxContexts returns SELinux contexts with expanded paths
func (l *Loader) GetSELinuxContexts(config *InstallerConfig) []SELinuxContext {
	contexts := make([]SELinuxContext, len(config.SELinux.Contexts))
	for i, ctx := range config.SELinux.Contexts {
		contexts[i] = SELinuxContext{
			Path:    l.Expand(ctx.Path),
			Type:    ctx.Type,
			Pattern: l.Expand(ctx.Pattern),
		}
	}
	return contexts
}

// GetNginxFixCommands returns expanded commands for an nginx fix
func (l *Loader) GetNginxFixCommands(config *InstallerConfig, fixName string) []string {
	fix, ok := config.Nginx.Fixes[fixName]
	if !ok {
		return nil
	}
	return l.ExpandSlice(fix.Commands)
}

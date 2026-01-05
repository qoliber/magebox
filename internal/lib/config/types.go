// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package config

// InstallerConfig represents the full installer configuration from YAML
type InstallerConfig struct {
	SchemaVersion  string          `yaml:"schema_version"`
	Meta           Meta            `yaml:"meta"`
	PackageManager PackageManager  `yaml:"package_manager"`
	Prerequisites  Prerequisites   `yaml:"prerequisites"`
	Repositories   map[string]Repo `yaml:"repositories"`
	PHP            PHPConfig       `yaml:"php"`
	Nginx          NginxConfig     `yaml:"nginx"`
	Tools          ToolsConfig     `yaml:"tools"`
	DNS            DNSConfig       `yaml:"dns"`
	SELinux        SELinuxConfig   `yaml:"selinux"`
	Sudoers        SudoersConfig   `yaml:"sudoers"`
	Profiling      ProfilingConfig `yaml:"profiling"`
	Paths          PathsConfig     `yaml:"paths"` // macOS specific
}

// Meta contains platform metadata
type Meta struct {
	Platform          string      `yaml:"platform"`
	Distro            string      `yaml:"distro"`
	DisplayName       string      `yaml:"display_name"`
	SupportedVersions interface{} `yaml:"supported_versions"` // Can be []string or []VersionInfo
}

// GetSupportedVersions returns supported versions as strings
func (m *Meta) GetSupportedVersions() []string {
	switch v := m.SupportedVersions.(type) {
	case []interface{}:
		var versions []string
		for _, item := range v {
			switch t := item.(type) {
			case string:
				versions = append(versions, t)
			case map[string]interface{}:
				if ver, ok := t["version"].(string); ok {
					versions = append(versions, ver)
				}
			}
		}
		return versions
	case []string:
		return v
	default:
		return nil
	}
}

// VersionInfo contains OS version details (macOS)
type VersionInfo struct {
	Version  string `yaml:"version"`
	Codename string `yaml:"codename"`
}

// PackageManager defines package manager commands
type PackageManager struct {
	Name                string `yaml:"name"`
	Install             string `yaml:"install"`
	Update              string `yaml:"update"`
	Check               string `yaml:"check"`
	NotInstalledMessage string `yaml:"not_installed_message"`
}

// Prerequisites defines required packages
type Prerequisites struct {
	Check          string   `yaml:"check"`
	InstallMessage string   `yaml:"install_message"`
	Packages       []string `yaml:"packages"`
}

// Repo defines a package repository
type Repo struct {
	Name    string `yaml:"name"`
	Check   string `yaml:"check"`
	Install string `yaml:"install"`
}

// PHPConfig defines PHP installation settings
type PHPConfig struct {
	VersionFormat string            `yaml:"version_format"`
	Versions      []string          `yaml:"versions"`
	Packages      PHPPackages       `yaml:"packages"`
	Optional      map[string]any    `yaml:"optional"`
	Paths         PHPPaths          `yaml:"paths"`
	INISettings   map[string]string `yaml:"ini_settings"`
	Services      PHPServices       `yaml:"services"`
}

// PHPPackages defines PHP packages to install
type PHPPackages struct {
	Formula    string   `yaml:"formula"`    // macOS
	Core       []string `yaml:"core"`
	Extensions []string `yaml:"extensions"`
}

// PHPPaths defines PHP path locations
type PHPPaths struct {
	Binary    string `yaml:"binary"`
	FPMBinary string `yaml:"fpm_binary"`
	INI       string `yaml:"ini"`
	FPMConfig string `yaml:"fpm_config"`
}

// PHPServices defines PHP-FPM service commands
type PHPServices struct {
	FPM ServiceCommands `yaml:"fpm"`
}

// ServiceCommands defines systemd/service commands
type ServiceCommands struct {
	Name    string `yaml:"name"`
	Enable  string `yaml:"enable"`
	Start   string `yaml:"start"`
	Stop    string `yaml:"stop"`
	Restart string `yaml:"restart"`
	Reload  string `yaml:"reload"`
	Test    string `yaml:"test"`
}

// NginxConfig defines Nginx installation settings
type NginxConfig struct {
	Packages      []string                `yaml:"packages"`
	Paths         NginxPaths              `yaml:"paths"`
	Services      NginxServices           `yaml:"services"`
	Configuration NginxConfiguration      `yaml:"configuration"`
	Fixes         map[string]NginxFix     `yaml:"fixes"`
}

// NginxPaths defines Nginx path locations
type NginxPaths struct {
	Binary string `yaml:"binary"`
	Config string `yaml:"config"`
	TmpDir string `yaml:"tmp_dir"`
}

// NginxServices defines Nginx service commands
type NginxServices struct {
	Nginx ServiceCommands `yaml:"nginx"`
}

// NginxConfiguration defines Nginx configuration commands
type NginxConfiguration struct {
	SetUser string `yaml:"set_user"`
}

// NginxFix defines a fix for Nginx issues
type NginxFix struct {
	Description string   `yaml:"description"`
	Commands    []string `yaml:"commands"`
}

// ToolsConfig defines tool installations
type ToolsConfig struct {
	Mkcert    ToolConfig `yaml:"mkcert"`
	Dnsmasq   ToolConfig `yaml:"dnsmasq"`
	Multitail ToolConfig `yaml:"multitail"`
	Docker    ToolConfig `yaml:"docker"`
}

// ToolConfig defines a single tool installation
type ToolConfig struct {
	Packages            []string `yaml:"packages"`
	InstallInstructions string   `yaml:"install_instructions"`
}

// DNSConfig defines DNS configuration
type DNSConfig struct {
	Dnsmasq         DnsmasqConfig         `yaml:"dnsmasq"`
	SystemdResolved SystemdResolvedConfig `yaml:"systemd_resolved"`
	Resolver        ResolverConfig        `yaml:"resolver"` // macOS
}

// DnsmasqConfig defines dnsmasq configuration
type DnsmasqConfig struct {
	ConfigDir      string              `yaml:"config_dir"`
	MainConfig     string              `yaml:"main_config"`
	Setup          []SetupStep         `yaml:"setup"`
	ConfigTemplate string              `yaml:"config_template"`
	Services       DnsmasqServices     `yaml:"services"`
}

// SetupStep defines a setup command
type SetupStep struct {
	Description string `yaml:"description"`
	Command     string `yaml:"command"`
}

// DnsmasqServices defines dnsmasq service commands
type DnsmasqServices struct {
	Enable  string `yaml:"enable"`
	Restart string `yaml:"restart"`
}

// SystemdResolvedConfig defines systemd-resolved configuration
type SystemdResolvedConfig struct {
	Check          string `yaml:"check"`
	ConfigDir      string `yaml:"config_dir"`
	ConfigTemplate string `yaml:"config_template"`
	Restart        string `yaml:"restart"`
}

// ResolverConfig defines macOS resolver configuration
type ResolverConfig struct {
	ConfigDir      string      `yaml:"config_dir"`
	ConfigFile     string      `yaml:"config_file"`
	ConfigTemplate string      `yaml:"config_template"`
	Setup          []SetupStep `yaml:"setup"`
}

// SELinuxConfig defines SELinux configuration
type SELinuxConfig struct {
	Enabled   bool              `yaml:"enabled"`
	Check     string            `yaml:"check"`
	Booleans  []SELinuxBoolean  `yaml:"booleans"`
	Contexts  []SELinuxContext  `yaml:"contexts"`
	Commands  SELinuxCommands   `yaml:"commands"`
}

// SELinuxBoolean defines an SELinux boolean setting
type SELinuxBoolean struct {
	Name    string `yaml:"name"`
	Value   string `yaml:"value"`
	Command string `yaml:"command"`
}

// SELinuxContext defines an SELinux context
type SELinuxContext struct {
	Path    string `yaml:"path"`
	Type    string `yaml:"type"`
	Pattern string `yaml:"pattern"`
}

// SELinuxCommands defines SELinux command templates
type SELinuxCommands struct {
	Semanage       string `yaml:"semanage"`
	Restorecon     string `yaml:"restorecon"`
	ChconFallback  string `yaml:"chcon_fallback"`
}

// SudoersConfig defines sudoers configuration
type SudoersConfig struct {
	Enabled     bool     `yaml:"enabled"`
	File        string   `yaml:"file"`
	Permissions string   `yaml:"permissions"`
	Rules       []string `yaml:"rules"`
}

// ProfilingConfig defines profiling tools configuration
type ProfilingConfig struct {
	Blackfire BlackfireConfig `yaml:"blackfire"`
	Tideways  TidewaysConfig  `yaml:"tideways"`
}

// BlackfireConfig defines Blackfire profiler configuration
type BlackfireConfig struct {
	GPGKey       string              `yaml:"gpg_key"`
	GPGImport    string              `yaml:"gpg_import"`
	Repository   BlackfireRepository `yaml:"repository"`
	Packages     interface{}         `yaml:"packages"` // Can be []string or map[string]string
	Tap          string              `yaml:"tap"`          // macOS
	TapInstall   string              `yaml:"tap_install"`  // macOS
	PHPExtension PHPExtension        `yaml:"php_extension"` // macOS
}

// GetPackages returns packages as a slice of strings
func (b *BlackfireConfig) GetPackages() []string {
	switch v := b.Packages.(type) {
	case []interface{}:
		var packages []string
		for _, item := range v {
			if s, ok := item.(string); ok {
				packages = append(packages, s)
			}
		}
		return packages
	case []string:
		return v
	case map[string]interface{}:
		var packages []string
		for _, val := range v {
			if s, ok := val.(string); ok {
				packages = append(packages, s)
			}
		}
		return packages
	default:
		return nil
	}
}

// BlackfireRepository defines Blackfire repo configuration
type BlackfireRepository struct {
	URL     string `yaml:"url"`
	Install string `yaml:"install"`
}

// PHPExtension defines PHP extension installation (macOS pecl)
type PHPExtension struct {
	InstallMethod string `yaml:"install_method"`
	Install       string `yaml:"install"`
}

// TidewaysConfig defines Tideways profiler configuration
type TidewaysConfig struct {
	GPGKey       string            `yaml:"gpg_key"`
	GPGImport    string            `yaml:"gpg_import"`
	Packages     map[string]string `yaml:"packages"`
	Install      string            `yaml:"install"`
	PHPExtension PHPExtension      `yaml:"php_extension"` // macOS
}

// PathsConfig defines platform-specific paths (macOS)
type PathsConfig struct {
	AppleSilicon MacOSPaths `yaml:"apple_silicon"`
	Intel        MacOSPaths `yaml:"intel"`
}

// MacOSPaths defines macOS Homebrew paths
type MacOSPaths struct {
	Prefix string `yaml:"prefix"`
	PHP    string `yaml:"php"`
	Pecl   string `yaml:"pecl"`
}

package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// GlobalConfig represents the global MageBox configuration stored in ~/.magebox/config.yaml
type GlobalConfig struct {
	// DNSMode specifies how DNS resolution is handled: "dnsmasq" or "hosts"
	DNSMode string `yaml:"dns_mode,omitempty"`

	// DefaultPHP is the default PHP version for new projects
	DefaultPHP string `yaml:"default_php,omitempty"`

	// DefaultServices are the default services for new projects
	DefaultServices DefaultServices `yaml:"default_services,omitempty"`

	// Portainer enables/disables Portainer Docker UI
	Portainer bool `yaml:"portainer,omitempty"`

	// TLD is the top-level domain for local development (default: "test")
	TLD string `yaml:"tld,omitempty"`

	// Editor is the preferred editor for opening files
	Editor string `yaml:"editor,omitempty"`

	// AutoStart enables automatic service startup
	AutoStart bool `yaml:"auto_start,omitempty"`
}

// DefaultServices represents default service configurations
type DefaultServices struct {
	MySQL      string `yaml:"mysql,omitempty"`
	MariaDB    string `yaml:"mariadb,omitempty"`
	Redis      bool   `yaml:"redis,omitempty"`
	OpenSearch string `yaml:"opensearch,omitempty"`
	RabbitMQ   bool   `yaml:"rabbitmq,omitempty"`
	Mailpit    bool   `yaml:"mailpit,omitempty"`
}

// GlobalConfigPath returns the path to the global config file
func GlobalConfigPath(homeDir string) string {
	return filepath.Join(homeDir, ".magebox", "config.yaml")
}

// LoadGlobalConfig loads the global configuration
func LoadGlobalConfig(homeDir string) (*GlobalConfig, error) {
	configPath := GlobalConfigPath(homeDir)

	// Return defaults if config doesn't exist
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return DefaultGlobalConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read global config: %w", err)
	}

	var config GlobalConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse global config: %w", err)
	}

	// Apply defaults for unset values
	config.applyDefaults()

	return &config, nil
}

// SaveGlobalConfig saves the global configuration
func SaveGlobalConfig(homeDir string, config *GlobalConfig) error {
	configPath := GlobalConfigPath(homeDir)

	// Ensure directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Add header comment
	content := "# MageBox Global Configuration\n"
	content += "# This file is managed by MageBox. Edit with care.\n\n"
	content += string(data)

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// DefaultGlobalConfig returns a GlobalConfig with sensible defaults
func DefaultGlobalConfig() *GlobalConfig {
	return &GlobalConfig{
		DNSMode:    "hosts", // Use /etc/hosts by default (works without extra setup)
		DefaultPHP: "8.2",
		DefaultServices: DefaultServices{
			MySQL: "8.0",
			Redis: true,
		},
		Portainer: false,
		TLD:       "test",
		AutoStart: false,
	}
}

// applyDefaults applies default values for any unset fields
func (c *GlobalConfig) applyDefaults() {
	defaults := DefaultGlobalConfig()

	if c.DNSMode == "" {
		c.DNSMode = defaults.DNSMode
	}
	if c.DefaultPHP == "" {
		c.DefaultPHP = defaults.DefaultPHP
	}
	if c.TLD == "" {
		c.TLD = defaults.TLD
	}
}

// UseDnsmasq returns true if dnsmasq should be used for DNS
func (c *GlobalConfig) UseDnsmasq() bool {
	return c.DNSMode == "dnsmasq"
}

// UseHosts returns true if /etc/hosts should be used for DNS
func (c *GlobalConfig) UseHosts() bool {
	return c.DNSMode == "hosts" || c.DNSMode == ""
}

// GetTLD returns the configured TLD with fallback to default
func (c *GlobalConfig) GetTLD() string {
	if c.TLD == "" {
		return "test"
	}
	return c.TLD
}

// InitGlobalConfig creates the initial global config file with defaults
func InitGlobalConfig(homeDir string) error {
	configPath := GlobalConfigPath(homeDir)

	// Don't overwrite existing config
	if _, err := os.Stat(configPath); err == nil {
		return nil
	}

	return SaveGlobalConfig(homeDir, DefaultGlobalConfig())
}

// GlobalConfigExists checks if the global config file exists
func GlobalConfigExists(homeDir string) bool {
	configPath := GlobalConfigPath(homeDir)
	_, err := os.Stat(configPath)
	return err == nil
}

// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

//go:embed versions.yaml
var embeddedVersionsYAML string

// VersionsConfig represents the entire versions.yaml file
type VersionsConfig struct {
	SchemaVersion string          `yaml:"schema_version"`
	Defaults      VersionDefaults `yaml:"defaults"`
	Magento       DistroConfig    `yaml:"magento"`
	MageOS        DistroConfig    `yaml:"mageos"`
}

// VersionDefaults contains default settings for quick install
type VersionDefaults struct {
	PHP          string `yaml:"php"`
	MySQL        string `yaml:"mysql"`
	OpenSearch   string `yaml:"opensearch"`
	Distribution string `yaml:"distribution"`
}

// DistroConfig contains configuration for a distribution (Magento or MageOS)
type DistroConfig struct {
	Package  string         `yaml:"package"`
	Versions []VersionEntry `yaml:"versions"`
}

// VersionEntry represents a single version of Magento/MageOS
type VersionEntry struct {
	Version string   `yaml:"version"`
	Name    string   `yaml:"name"`
	PHP     []string `yaml:"php"`
	Default bool     `yaml:"default,omitempty"`
	Base    string   `yaml:"base,omitempty"` // For MageOS: which Magento version it's based on
}

// LoadVersions loads the versions configuration
// It checks for a custom file first, then falls back to embedded
func LoadVersions(mageboxDir string) (*VersionsConfig, error) {
	// Check for custom versions file in yaml/config/
	customPath := filepath.Join(mageboxDir, "yaml", "config", "versions.yaml")
	if data, err := os.ReadFile(customPath); err == nil {
		var config VersionsConfig
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse custom versions.yaml: %w", err)
		}
		return &config, nil
	}

	// Check for local override
	localPath := filepath.Join(mageboxDir, "yaml-local", "config", "versions.yaml")
	if data, err := os.ReadFile(localPath); err == nil {
		var config VersionsConfig
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse local versions.yaml: %w", err)
		}
		return &config, nil
	}

	// Fall back to embedded
	return LoadEmbeddedVersions()
}

// LoadEmbeddedVersions loads the embedded versions configuration
func LoadEmbeddedVersions() (*VersionsConfig, error) {
	var config VersionsConfig
	if err := yaml.Unmarshal([]byte(embeddedVersionsYAML), &config); err != nil {
		return nil, fmt.Errorf("failed to parse embedded versions.yaml: %w", err)
	}
	return &config, nil
}

// GetMagentoVersions returns a list of Magento versions
func (c *VersionsConfig) GetMagentoVersions() []VersionEntry {
	return c.Magento.Versions
}

// GetMageOSVersions returns a list of MageOS versions
func (c *VersionsConfig) GetMageOSVersions() []VersionEntry {
	return c.MageOS.Versions
}

// GetDefaultMagentoVersion returns the default Magento version
func (c *VersionsConfig) GetDefaultMagentoVersion() *VersionEntry {
	for i := range c.Magento.Versions {
		if c.Magento.Versions[i].Default {
			return &c.Magento.Versions[i]
		}
	}
	if len(c.Magento.Versions) > 0 {
		return &c.Magento.Versions[0]
	}
	return nil
}

// GetDefaultMageOSVersion returns the default MageOS version
func (c *VersionsConfig) GetDefaultMageOSVersion() *VersionEntry {
	for i := range c.MageOS.Versions {
		if c.MageOS.Versions[i].Default {
			return &c.MageOS.Versions[i]
		}
	}
	if len(c.MageOS.Versions) > 0 {
		return &c.MageOS.Versions[0]
	}
	return nil
}

// GetMagentoPackage returns the Magento composer package name
func (c *VersionsConfig) GetMagentoPackage() string {
	return c.Magento.Package
}

// GetMageOSPackage returns the MageOS composer package name
func (c *VersionsConfig) GetMageOSPackage() string {
	return c.MageOS.Package
}

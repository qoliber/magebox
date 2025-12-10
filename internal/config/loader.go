package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	// ConfigFileName is the main configuration file name
	ConfigFileName = ".magebox"
	// LocalConfigFileName is the local override configuration file name
	LocalConfigFileName = ".magebox.local"
)

// Loader handles loading and merging configuration files
type Loader struct {
	basePath string
}

// NewLoader creates a new configuration loader
func NewLoader(basePath string) *Loader {
	return &Loader{basePath: basePath}
}

// Load loads and merges the configuration from .magebox and .magebox.local
func (l *Loader) Load() (*Config, error) {
	mainConfigPath := filepath.Join(l.basePath, ConfigFileName)
	localConfigPath := filepath.Join(l.basePath, LocalConfigFileName)

	// Load main config (required)
	mainConfig, err := l.loadFile(mainConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &ConfigNotFoundError{Path: mainConfigPath}
		}
		return nil, fmt.Errorf("failed to load %s: %w", ConfigFileName, err)
	}

	// Load local config (optional)
	localConfig, err := l.loadFile(localConfigPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load %s: %w", LocalConfigFileName, err)
	}

	// Merge configs (local overrides main)
	config := l.merge(mainConfig, localConfig)

	// Validate the merged config
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// loadFile loads a single configuration file
func (l *Loader) loadFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, &ParseError{Path: path, Err: err}
	}

	return &config, nil
}

// merge merges two configurations, with local taking precedence
func (l *Loader) merge(main, local *Config) *Config {
	if local == nil {
		return main
	}

	result := *main

	// Override simple fields if set in local
	if local.Name != "" {
		result.Name = local.Name
	}
	if local.PHP != "" {
		result.PHP = local.PHP
	}
	if len(local.Domains) > 0 {
		result.Domains = local.Domains
	}

	// Merge services
	result.Services = l.mergeServices(main.Services, local.Services)

	// Merge env vars
	if result.Env == nil {
		result.Env = make(map[string]string)
	}
	for k, v := range main.Env {
		result.Env[k] = v
	}
	for k, v := range local.Env {
		result.Env[k] = v
	}

	// Merge commands
	if result.Commands == nil {
		result.Commands = make(map[string]Command)
	}
	for k, v := range main.Commands {
		result.Commands[k] = v
	}
	for k, v := range local.Commands {
		result.Commands[k] = v
	}

	return &result
}

// mergeServices merges service configurations
func (l *Loader) mergeServices(main, local Services) Services {
	result := main

	if local.MySQL != nil {
		result.MySQL = local.MySQL
	}
	if local.MariaDB != nil {
		result.MariaDB = local.MariaDB
	}
	if local.Redis != nil {
		result.Redis = local.Redis
	}
	if local.OpenSearch != nil {
		result.OpenSearch = local.OpenSearch
	}
	if local.Elasticsearch != nil {
		result.Elasticsearch = local.Elasticsearch
	}
	if local.RabbitMQ != nil {
		result.RabbitMQ = local.RabbitMQ
	}
	if local.Mailpit != nil {
		result.Mailpit = local.Mailpit
	}

	return result
}

// ConfigNotFoundError indicates the configuration file was not found
type ConfigNotFoundError struct {
	Path string
}

func (e *ConfigNotFoundError) Error() string {
	return fmt.Sprintf("configuration file not found: %s\n\nRun 'magebox init' to create one", e.Path)
}

// ParseError indicates a YAML parsing error
type ParseError struct {
	Path string
	Err  error
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("failed to parse %s: %v", e.Path, e.Err)
}

// LoadFromPath is a convenience function to load config from a path
func LoadFromPath(path string) (*Config, error) {
	loader := NewLoader(path)
	return loader.Load()
}

// LoadFromCurrentDir loads config from the current working directory
func LoadFromCurrentDir() (*Config, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}
	return LoadFromPath(cwd)
}

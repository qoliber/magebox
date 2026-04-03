package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// ConfigFileName is the main configuration file name
	ConfigFileName = ".magebox.yaml"
	// ConfigFileNameLegacy is the legacy configuration file name (for backward compatibility)
	ConfigFileNameLegacy = ".magebox"
	// LocalConfigFileName is the local override configuration file name
	LocalConfigFileName = ".magebox.local.yaml"
	// LocalConfigFileNameLegacy is the legacy local configuration file name (for backward compatibility)
	LocalConfigFileNameLegacy = ".magebox.local"
)

// Loader handles loading and merging configuration files
type Loader struct {
	basePath string
}

// NewLoader creates a new configuration loader
func NewLoader(basePath string) *Loader {
	return &Loader{basePath: basePath}
}

// Load loads and merges the configuration from .magebox.yaml (or .magebox for backward compatibility) and .magebox.local
func (l *Loader) Load() (*Config, error) {
	mainConfigPath := filepath.Join(l.basePath, ConfigFileName)
	legacyConfigPath := filepath.Join(l.basePath, ConfigFileNameLegacy)
	localConfigPath := filepath.Join(l.basePath, LocalConfigFileName)

	visited := make(map[string]bool)

	// Try to load new format first, fall back to legacy
	mainConfig, err := l.loadFileWithIncludes(mainConfigPath, visited)
	if err != nil && os.IsNotExist(err) {
		// Try legacy format
		mainConfig, err = l.loadFileWithIncludes(legacyConfigPath, visited)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, &ConfigNotFoundError{Path: mainConfigPath}
			}
			return nil, fmt.Errorf("failed to load %s: %w", ConfigFileNameLegacy, err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to load %s: %w", ConfigFileName, err)
	}

	// Load local config (optional) - try new format first, fall back to legacy
	legacyLocalConfigPath := filepath.Join(l.basePath, LocalConfigFileNameLegacy)
	localConfig, err := l.loadFileWithIncludes(localConfigPath, visited)
	if err != nil && os.IsNotExist(err) {
		// Try legacy format
		localConfig, err = l.loadFileWithIncludes(legacyLocalConfigPath, visited)
	}
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load local config: %w", err)
	}

	// Merge configs (local overrides main)
	config := l.merge(mainConfig, localConfig)

	// Validate the merged config
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// loadFileWithIncludes loads a config file and recursively processes include_config entries.
// visited tracks absolute paths that have already been loaded to detect circular includes.
func (l *Loader) loadFileWithIncludes(path string, visited map[string]bool) (*Config, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path %s: %w", path, err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}

	if visited[absPath] {
		return nil, fmt.Errorf("circular include detected: %s", path)
	}
	visited[absPath] = true

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, &ParseError{Path: path, Err: err}
	}

	if len(config.IncludeConfig) == 0 {
		return &config, nil
	}

	// Process include_config entries: accumulate included configs in order, then
	// merge the current file's own fields on top so they take final precedence.
	baseDir := filepath.Dir(absPath)
	base := &Config{}

	for _, includePath := range config.IncludeConfig {
		paths, err := l.resolveIncludePaths(baseDir, includePath)
		if err != nil {
			return nil, fmt.Errorf("include_config %q: %w", includePath, err)
		}

		for _, resolvedPath := range paths {
			included, err := l.loadFileWithIncludes(resolvedPath, visited)
			if err != nil {
				return nil, fmt.Errorf("include_config %q: %w", resolvedPath, err)
			}
			base = l.merge(base, included)
		}
	}

	// Merge the current file's own values on top; clear IncludeConfig so it is
	// not treated as a field to propagate into the merged result.
	own := config
	own.IncludeConfig = nil
	return l.merge(base, &own), nil
}

// resolveIncludePaths resolves an include_config entry to one or more file paths.
// If the resolved path is a directory, all .yaml/.yml files inside it are returned,
// sorted by file name.
func (l *Loader) resolveIncludePaths(baseDir, includePath string) ([]string, error) {
	resolved := includePath
	if !filepath.IsAbs(includePath) {
		resolved = filepath.Join(baseDir, includePath)
	}

	info, err := os.Stat(resolved)
	if err != nil {
		return nil, fmt.Errorf("cannot access %q: %w", includePath, err)
	}

	if !info.IsDir() {
		return []string{resolved}, nil
	}

	// Directory: collect all .yaml/.yml files (os.ReadDir returns them sorted by name)
	entries, err := os.ReadDir(resolved)
	if err != nil {
		return nil, fmt.Errorf("cannot read directory %q: %w", includePath, err)
	}

	var paths []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		paths = append(paths, filepath.Join(resolved, entry.Name()))
	}

	return paths, nil
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
	if local.Type != "" {
		result.Type = local.Type
	}
	if local.PHP != "" {
		result.PHP = local.PHP
	}
	if len(local.Domains) > 0 {
		result.Domains = local.Domains
	}
	if local.ComposeFile != "" {
		result.ComposeFile = local.ComposeFile
	}
	if local.Isolated {
		result.Isolated = local.Isolated
	}
	if local.Testing != nil {
		result.Testing = local.Testing
	}
	if local.Sandbox != nil {
		result.Sandbox = local.Sandbox
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

	// Merge PHP INI overrides
	if result.PHPINI == nil {
		result.PHPINI = make(map[string]string)
	}
	for k, v := range main.PHPINI {
		result.PHPINI[k] = v
	}
	for k, v := range local.PHPINI {
		result.PHPINI[k] = v
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
	if local.Varnish != nil {
		result.Varnish = local.Varnish
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

// SaveToPath saves the config to the specified path
func SaveToPath(cfg *Config, path string) error {
	configPath := filepath.Join(path, ConfigFileName)

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(cfg); err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	enc.Close()

	if err := os.WriteFile(configPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// LocalConfig represents local config overrides that can be saved independently
type LocalConfig struct {
	PHP    string            `yaml:"php,omitempty"`
	PHPINI map[string]string `yaml:"php_ini,omitempty"`
	Env    map[string]string `yaml:"env,omitempty"`
}

// LoadLocalConfig loads only the local config file
func LoadLocalConfig(basePath string) (*LocalConfig, error) {
	localConfigPath := filepath.Join(basePath, LocalConfigFileName)

	data, err := os.ReadFile(localConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &LocalConfig{}, nil
		}
		return nil, err
	}

	var local LocalConfig
	if err := yaml.Unmarshal(data, &local); err != nil {
		return nil, err
	}

	return &local, nil
}

// SaveLocalConfig saves only the local config file
func SaveLocalConfig(basePath string, local *LocalConfig) error {
	localConfigPath := filepath.Join(basePath, LocalConfigFileName)

	data, err := yaml.Marshal(local)
	if err != nil {
		return fmt.Errorf("failed to marshal local config: %w", err)
	}

	if err := os.WriteFile(localConfigPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write local config file: %w", err)
	}

	return nil
}

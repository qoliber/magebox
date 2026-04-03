package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Project types
const (
	ProjectTypeMagento = "magento"
	ProjectTypeLaravel = "laravel"
)

// Config represents the merged configuration from .magebox and .magebox.local
type Config struct {
	Name          string             `yaml:"name"`
	Type          string             `yaml:"type,omitempty"` // Project type: "magento" (default) or "laravel"
	Domains       []Domain           `yaml:"domains"`
	PHP           string             `yaml:"php"`
	PHPINI        map[string]string  `yaml:"php_ini,omitempty"`
	Isolated      bool               `yaml:"isolated,omitempty"` // Use dedicated PHP-FPM master for this project
	Services      Services           `yaml:"services"`
	Env           map[string]string  `yaml:"env,omitempty"`
	Commands      map[string]Command `yaml:"commands,omitempty"`
	Testing       *TestingConfig     `yaml:"testing,omitempty"`
	ComposeFile   string             `yaml:"compose_file,omitempty"` // Path to project-specific docker-compose.yml
	Sandbox       *SandboxConfig     `yaml:"sandbox,omitempty"`
	IncludeConfig []string           `yaml:"include_config,omitempty"` // Paths to additional config files or directories to merge
}

// GetType returns the project type, defaulting to "magento"
func (c *Config) GetType() string {
	if c.Type == "" {
		return ProjectTypeMagento
	}
	return c.Type
}

// IsMagento returns true if this is a Magento project
func (c *Config) IsMagento() bool {
	return c.GetType() == ProjectTypeMagento
}

// IsLaravel returns true if this is a Laravel project
func (c *Config) IsLaravel() bool {
	return c.GetType() == ProjectTypeLaravel
}

// GetDefaultRoot returns the default document root for this project type
func (c *Config) GetDefaultRoot() string {
	if c.GetType() == ProjectTypeLaravel {
		return "public"
	}
	return "pub"
}

// TestingConfig represents the testing configuration
type TestingConfig struct {
	PHPUnit     *PHPUnitTestConfig     `yaml:"phpunit,omitempty"`
	Integration *IntegrationTestConfig `yaml:"integration,omitempty"`
	PHPStan     *PHPStanTestConfig     `yaml:"phpstan,omitempty"`
	PHPCS       *PHPCSTestConfig       `yaml:"phpcs,omitempty"`
	PHPMD       *PHPMDTestConfig       `yaml:"phpmd,omitempty"`
}

// PHPUnitTestConfig represents PHPUnit configuration
type PHPUnitTestConfig struct {
	Enabled   bool   `yaml:"enabled,omitempty"`
	Config    string `yaml:"config,omitempty"`
	TestSuite string `yaml:"testsuite,omitempty"`
}

// IntegrationTestConfig represents Magento integration test configuration
type IntegrationTestConfig struct {
	Enabled   bool   `yaml:"enabled,omitempty"`
	Config    string `yaml:"config,omitempty"`
	DBHost    string `yaml:"db_host,omitempty"`
	DBPort    int    `yaml:"db_port,omitempty"`
	DBName    string `yaml:"db_name,omitempty"`
	DBUser    string `yaml:"db_user,omitempty"`
	DBPass    string `yaml:"db_pass,omitempty"`
	Tmpfs     bool   `yaml:"tmpfs,omitempty"`      // Run MySQL in RAM for faster tests
	TmpfsSize string `yaml:"tmpfs_size,omitempty"` // RAM size (default: 1g)
	KeepAlive bool   `yaml:"keep_alive,omitempty"` // Keep container running after tests
}

// PHPStanTestConfig represents PHPStan configuration
type PHPStanTestConfig struct {
	Enabled bool     `yaml:"enabled,omitempty"`
	Level   int      `yaml:"level,omitempty"`
	Config  string   `yaml:"config,omitempty"`
	Paths   []string `yaml:"paths,omitempty"`
}

// PHPCSTestConfig represents PHP_CodeSniffer configuration
type PHPCSTestConfig struct {
	Enabled  bool     `yaml:"enabled,omitempty"`
	Standard string   `yaml:"standard,omitempty"`
	Config   string   `yaml:"config,omitempty"`
	Paths    []string `yaml:"paths,omitempty"`
}

// PHPMDTestConfig represents PHP Mess Detector configuration
type PHPMDTestConfig struct {
	Enabled bool     `yaml:"enabled,omitempty"`
	Ruleset string   `yaml:"ruleset,omitempty"`
	Config  string   `yaml:"config,omitempty"`
	Paths   []string `yaml:"paths,omitempty"`
}

// SandboxConfig configures the bubblewrap sandbox for AI coding agents
type SandboxConfig struct {
	// DefaultTool is the default command to run (default: "claude")
	DefaultTool string `yaml:"default_tool,omitempty"`
	// ExtraROBinds are additional read-only bind mounts
	ExtraROBinds []string `yaml:"extra_ro_binds,omitempty"`
	// ExtraBinds are additional read-write bind mounts
	ExtraBinds []string `yaml:"extra_binds,omitempty"`
	// ToolProfiles overrides for known tool configs
	ToolProfiles map[string]SandboxToolProfile `yaml:"tool_profiles,omitempty"`
}

// SandboxToolProfile defines sandbox config for a specific AI tool
type SandboxToolProfile struct {
	// Command is the binary name or path to execute
	Command string `yaml:"command,omitempty"`
	// Args are default arguments for this tool
	Args []string `yaml:"args,omitempty"`
	// ConfigDirs are directories to bind read-write (e.g., ~/.claude)
	ConfigDirs []string `yaml:"config_dirs,omitempty"`
	// ConfigFiles are individual files to bind read-write
	ConfigFiles []string `yaml:"config_files,omitempty"`
}

// Command represents a custom command that can be run via "magebox run <name>"
type Command struct {
	Description string `yaml:"description,omitempty"`
	Run         string `yaml:"run"`
}

// UnmarshalYAML allows commands to be defined as string or object
func (c *Command) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// First try as string
	var raw interface{}
	if err := unmarshal(&raw); err != nil {
		return err
	}

	switch v := raw.(type) {
	case string:
		// commands:
		//   deploy: "php bin/magento deploy:mode:set production"
		c.Run = v
		return nil
	case map[string]interface{}:
		// commands:
		//   deploy:
		//     description: "Deploy to production mode"
		//     run: "php bin/magento deploy:mode:set production"
		if run, ok := v["run"].(string); ok {
			c.Run = run
		}
		if desc, ok := v["description"].(string); ok {
			c.Description = desc
		}
		return nil
	default:
		return nil
	}
}

// Domain represents a domain configuration
type Domain struct {
	Host        string `yaml:"host"`
	Root        string `yaml:"root,omitempty"`
	SSL         *bool  `yaml:"ssl,omitempty"`
	MageRunCode string `yaml:"mage_run_code,omitempty"` // Magento store/website code for multi-store setup
	MageRunType string `yaml:"mage_run_type,omitempty"` // "store" or "website" (default: "store")
}

// Services represents the services configuration
type Services struct {
	MySQL         *ServiceConfig `yaml:"mysql,omitempty"`
	MariaDB       *ServiceConfig `yaml:"mariadb,omitempty"`
	Redis         *ServiceConfig `yaml:"redis,omitempty"`
	Valkey        *ServiceConfig `yaml:"valkey,omitempty"`
	OpenSearch    *ServiceConfig `yaml:"opensearch,omitempty"`
	Elasticsearch *ServiceConfig `yaml:"elasticsearch,omitempty"`
	RabbitMQ      *ServiceConfig `yaml:"rabbitmq,omitempty"`
	Mailpit       *ServiceConfig `yaml:"mailpit,omitempty"`
	Varnish       *ServiceConfig `yaml:"varnish,omitempty"`
	PhpMyAdmin    *ServiceConfig `yaml:"phpmyadmin,omitempty"`
}

// ServiceConfig represents a service configuration
// Can be specified as just a version string "8.0" or as an object with more options
type ServiceConfig struct {
	Enabled bool   `yaml:"-"`
	Version string `yaml:"version,omitempty"`
	Port    int    `yaml:"port,omitempty"`
	Memory  string `yaml:"memory,omitempty"` // RAM allocation (e.g., "2g", "1024m")
}

// UnmarshalYAML implements custom unmarshaling to handle both string and object formats
func (s *ServiceConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// First unmarshal to interface{} to check the type
	var raw interface{}
	if err := unmarshal(&raw); err != nil {
		return err
	}

	switch v := raw.(type) {
	case bool:
		// redis: true or redis: false
		s.Enabled = v
		return nil
	case string:
		// mysql: "8.0"
		s.Enabled = true
		s.Version = v
		return nil
	case map[string]interface{}:
		// mysql:
		//   version: "8.0"
		//   port: 3307
		//   memory: "2g"
		s.Enabled = true
		if version, ok := v["version"].(string); ok {
			s.Version = version
		}
		if port, ok := v["port"].(int); ok {
			s.Port = port
		}
		if memory, ok := v["memory"].(string); ok {
			s.Memory = memory
		}
		return nil
	default:
		s.Enabled = true
		return nil
	}
}

// MarshalYAML implements custom marshaling to preserve the original format.
// - If only Enabled is set (no version/port/memory), marshals as `true`
// - If only version is set, marshals as the version string `"8.0"`
// - Otherwise marshals as an object
func (s ServiceConfig) MarshalYAML() (interface{}, error) {
	if s.Version == "" && s.Port == 0 && s.Memory == "" {
		return s.Enabled, nil
	}
	if s.Port == 0 && s.Memory == "" {
		return s.Version, nil
	}
	// Return as struct — use an alias to avoid infinite recursion
	type plain ServiceConfig
	return plain(s), nil
}

// GetRoot returns the document root, defaulting to "pub"
// Use GetRootForType when the project type is known
func (d *Domain) GetRoot() string {
	if d.Root == "" {
		return "pub"
	}
	return d.Root
}

// GetRootForType returns the document root, using the project type default if not explicitly set
func (d *Domain) GetRootForType(projectType string) string {
	if d.Root != "" {
		return d.Root
	}
	if projectType == ProjectTypeLaravel {
		return "public"
	}
	return "pub"
}

// webrootCandidates lists common webroot directories in priority order.
// Magento-style roots come first, then generic ones.
var webrootCandidates = []string{"pub", "public", "web", "htdocs", "httpdocs"}

// GetRootForProject returns the document root, auto-discovering it from the
// project directory when no explicit root is configured. It checks for common
// webroot directories (pub, public, web, htdocs, httpdocs) and returns the
// first one that exists. Falls back to the project-type default.
func (d *Domain) GetRootForProject(projectPath, projectType string) string {
	if d.Root != "" {
		return d.Root
	}
	if detected := discoverRoot(projectPath); detected != "" {
		return detected
	}
	if projectType == ProjectTypeLaravel {
		return "public"
	}
	return "pub"
}

// discoverRoot checks for common webroot directories inside projectPath
// and returns the first match, or "" if none exist.
func discoverRoot(projectPath string) string {
	for _, candidate := range webrootCandidates {
		info, err := os.Stat(filepath.Join(projectPath, candidate))
		if err == nil && info.IsDir() {
			return candidate
		}
	}
	return ""
}

// IsSSLEnabled returns whether SSL is enabled, defaulting to true
func (d *Domain) IsSSLEnabled() bool {
	if d.SSL == nil {
		return true
	}
	return *d.SSL
}

// GetStoreCode returns the Magento store code, defaulting to "default"
func (d *Domain) GetStoreCode() string {
	if d.MageRunCode == "" {
		return "default"
	}
	return d.MageRunCode
}

// GetMageRunType returns the Magento run type, defaulting to "store"
func (d *Domain) GetMageRunType() string {
	if d.MageRunType == "" {
		return "store"
	}
	return d.MageRunType
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Name == "" {
		return &ValidationError{Field: "name", Message: "name is required"}
	}
	if len(c.Domains) == 0 {
		return &ValidationError{Field: "domains", Message: "at least one domain is required"}
	}
	for i, d := range c.Domains {
		if d.Host == "" {
			return &ValidationError{Field: "domains", Message: "domain host is required", Index: i}
		}
	}
	if c.PHP == "" {
		return &ValidationError{Field: "php", Message: "php version is required"}
	}
	return nil
}

// DatabaseName returns the sanitized database name
// MySQL doesn't handle hyphens well in database names, so we replace them with underscores
func (c *Config) DatabaseName() string {
	return strings.ReplaceAll(c.Name, "-", "_")
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Message string
	Index   int
}

func (e *ValidationError) Error() string {
	if e.Index > 0 {
		return fmt.Sprintf("%s[%d]: %s", e.Field, e.Index, e.Message)
	}
	return e.Field + ": " + e.Message
}

// HasMySQL returns true if MySQL service is configured
func (s *Services) HasMySQL() bool {
	return s.MySQL != nil && s.MySQL.Enabled
}

// HasMariaDB returns true if MariaDB service is configured
func (s *Services) HasMariaDB() bool {
	return s.MariaDB != nil && s.MariaDB.Enabled
}

// HasRedis returns true if Redis service is configured
func (s *Services) HasRedis() bool {
	return s.Redis != nil && s.Redis.Enabled
}

// HasValkey returns true if Valkey service is configured
func (s *Services) HasValkey() bool {
	return s.Valkey != nil && s.Valkey.Enabled
}

// HasCacheService returns true if any Redis-compatible cache service (Redis or Valkey) is configured
func (s *Services) HasCacheService() bool {
	return s.HasRedis() || s.HasValkey()
}

// GetCacheServiceName returns the Docker compose service name for the cache service
func (s *Services) GetCacheServiceName() string {
	if s.HasValkey() {
		return "valkey"
	}
	return "redis"
}

// GetCacheServiceDisplayName returns a human-readable name for the cache service
func (s *Services) GetCacheServiceDisplayName() string {
	if s.HasValkey() {
		return "Valkey"
	}
	return "Redis"
}

// HasOpenSearch returns true if OpenSearch service is configured
func (s *Services) HasOpenSearch() bool {
	return s.OpenSearch != nil && s.OpenSearch.Enabled
}

// HasElasticsearch returns true if Elasticsearch service is configured
func (s *Services) HasElasticsearch() bool {
	return s.Elasticsearch != nil && s.Elasticsearch.Enabled
}

// HasRabbitMQ returns true if RabbitMQ service is configured
func (s *Services) HasRabbitMQ() bool {
	return s.RabbitMQ != nil && s.RabbitMQ.Enabled
}

// HasMailpit returns true if Mailpit service is configured
func (s *Services) HasMailpit() bool {
	return s.Mailpit != nil && s.Mailpit.Enabled
}

// HasVarnish returns true if Varnish service is configured
func (s *Services) HasVarnish() bool {
	return s.Varnish != nil && s.Varnish.Enabled
}

// HasPhpMyAdmin returns true if phpMyAdmin service is configured
func (s *Services) HasPhpMyAdmin() bool {
	return s.PhpMyAdmin != nil && s.PhpMyAdmin.Enabled
}

// GetDatabaseService returns the configured database service (MySQL or MariaDB)
func (s *Services) GetDatabaseService() *ServiceConfig {
	if s.HasMySQL() {
		return s.MySQL
	}
	if s.HasMariaDB() {
		return s.MariaDB
	}
	return nil
}

// GetSearchService returns the configured search service (OpenSearch or Elasticsearch)
func (s *Services) GetSearchService() *ServiceConfig {
	if s.HasOpenSearch() {
		return s.OpenSearch
	}
	if s.HasElasticsearch() {
		return s.Elasticsearch
	}
	return nil
}

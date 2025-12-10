package php

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/qoliber/magebox/internal/platform"
)

// PoolGenerator generates PHP-FPM pool configurations
type PoolGenerator struct {
	platform *platform.Platform
	poolsDir string
	runDir   string
}

// PoolConfig contains all data needed to generate a PHP-FPM pool
type PoolConfig struct {
	ProjectName     string
	PHPVersion      string
	SocketPath      string
	User            string
	Group           string
	MaxChildren     int
	StartServers    int
	MinSpareServers int
	MaxSpareServers int
	MaxRequests     int
	Env             map[string]string
}

// NewPoolGenerator creates a new pool generator
func NewPoolGenerator(p *platform.Platform) *PoolGenerator {
	return &PoolGenerator{
		platform: p,
		poolsDir: filepath.Join(p.MageBoxDir(), "php", "pools"),
		runDir:   filepath.Join(p.MageBoxDir(), "run"),
	}
}

// Generate generates a PHP-FPM pool configuration for a project
func (g *PoolGenerator) Generate(projectName, phpVersion string, env map[string]string) error {
	// Ensure directories exist
	if err := os.MkdirAll(g.poolsDir, 0755); err != nil {
		return fmt.Errorf("failed to create pools directory: %w", err)
	}
	if err := os.MkdirAll(g.runDir, 0755); err != nil {
		return fmt.Errorf("failed to create run directory: %w", err)
	}

	cfg := PoolConfig{
		ProjectName:     projectName,
		PHPVersion:      phpVersion,
		SocketPath:      g.GetSocketPath(projectName, phpVersion),
		User:            getCurrentUser(),
		Group:           getCurrentGroup(),
		MaxChildren:     10,
		StartServers:    2,
		MinSpareServers: 1,
		MaxSpareServers: 3,
		MaxRequests:     500,
		Env:             env,
	}

	content, err := g.renderPool(cfg)
	if err != nil {
		return fmt.Errorf("failed to render pool config: %w", err)
	}

	poolFile := filepath.Join(g.poolsDir, fmt.Sprintf("%s.conf", projectName))
	if err := os.WriteFile(poolFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write pool file: %w", err)
	}

	return nil
}

// Remove removes the pool configuration for a project
func (g *PoolGenerator) Remove(projectName string) error {
	poolFile := filepath.Join(g.poolsDir, fmt.Sprintf("%s.conf", projectName))
	if err := os.Remove(poolFile); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// GetSocketPath returns the socket path for a project
func (g *PoolGenerator) GetSocketPath(projectName, phpVersion string) string {
	return filepath.Join(g.runDir, fmt.Sprintf("%s-php%s.sock", projectName, phpVersion))
}

// PoolsDir returns the pools directory path
func (g *PoolGenerator) PoolsDir() string {
	return g.poolsDir
}

// RunDir returns the run directory path
func (g *PoolGenerator) RunDir() string {
	return g.runDir
}

// ListPools returns all pool configuration files
func (g *PoolGenerator) ListPools() ([]string, error) {
	pattern := filepath.Join(g.poolsDir, "*.conf")
	return filepath.Glob(pattern)
}

// renderPool renders the pool template
func (g *PoolGenerator) renderPool(cfg PoolConfig) (string, error) {
	tmpl, err := template.New("pool").Parse(poolTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, cfg); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// getCurrentUser returns the current user
func getCurrentUser() string {
	user := os.Getenv("USER")
	if user == "" {
		user = os.Getenv("LOGNAME")
	}
	if user == "" {
		user = "www-data"
	}
	return user
}

// getCurrentGroup returns the current user's primary group
func getCurrentGroup() string {
	// On macOS, the group is typically "staff", on Linux it's the username
	user := getCurrentUser()
	if user == "www-data" {
		return "www-data"
	}
	// For development, use the user as the group
	return user
}

// PHP-FPM pool template
const poolTemplate = `; MageBox generated pool for {{.ProjectName}}
; Do not edit manually - regenerated on magebox start

[{{.ProjectName}}]

user = {{.User}}
group = {{.Group}}

listen = {{.SocketPath}}
listen.owner = {{.User}}
listen.group = {{.Group}}
listen.mode = 0660

pm = dynamic
pm.max_children = {{.MaxChildren}}
pm.start_servers = {{.StartServers}}
pm.min_spare_servers = {{.MinSpareServers}}
pm.max_spare_servers = {{.MaxSpareServers}}
pm.max_requests = {{.MaxRequests}}

pm.status_path = /status

catch_workers_output = yes
decorate_workers_output = no

php_admin_value[error_log] = /var/log/php-fpm/{{.ProjectName}}-error.log
php_admin_flag[log_errors] = on

; Magento recommended settings
php_value[memory_limit] = 756M
php_value[max_execution_time] = 18000
php_value[max_input_time] = 600
php_value[max_input_vars] = 10000
php_value[post_max_size] = 64M
php_value[upload_max_filesize] = 64M
php_value[session.gc_maxlifetime] = 86400

; OPcache settings
php_admin_value[opcache.enable] = 1
php_admin_value[opcache.memory_consumption] = 512
php_admin_value[opcache.max_accelerated_files] = 130986
php_admin_value[opcache.validate_timestamps] = 1
php_admin_value[opcache.consistency_checks] = 0
php_admin_value[opcache.interned_strings_buffer] = 20

; Realpath cache
php_admin_value[realpath_cache_size] = 10M
php_admin_value[realpath_cache_ttl] = 7200

{{range $key, $value := .Env}}
env[{{$key}}] = {{$value}}
{{end}}
`

// FPMController manages PHP-FPM service for a specific version
type FPMController struct {
	platform *platform.Platform
	version  string
}

// NewFPMController creates a new PHP-FPM controller
func NewFPMController(p *platform.Platform, version string) *FPMController {
	return &FPMController{
		platform: p,
		version:  version,
	}
}

// Reload reloads PHP-FPM configuration
func (c *FPMController) Reload() error {
	switch c.platform.Type {
	case platform.Darwin:
		cmd := exec.Command("brew", "services", "restart", fmt.Sprintf("php@%s", c.version))
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to reload php-fpm: %w\nOutput: %s", err, output)
		}
	case platform.Linux:
		cmd := exec.Command("sudo", "systemctl", "reload", fmt.Sprintf("php%s-fpm", c.version))
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to reload php-fpm: %w\nOutput: %s", err, output)
		}
	}
	return nil
}

// Start starts PHP-FPM
func (c *FPMController) Start() error {
	switch c.platform.Type {
	case platform.Darwin:
		cmd := exec.Command("brew", "services", "start", fmt.Sprintf("php@%s", c.version))
		return cmd.Run()
	case platform.Linux:
		cmd := exec.Command("sudo", "systemctl", "start", fmt.Sprintf("php%s-fpm", c.version))
		return cmd.Run()
	}
	return fmt.Errorf("unsupported platform")
}

// Stop stops PHP-FPM
func (c *FPMController) Stop() error {
	switch c.platform.Type {
	case platform.Darwin:
		cmd := exec.Command("brew", "services", "stop", fmt.Sprintf("php@%s", c.version))
		return cmd.Run()
	case platform.Linux:
		cmd := exec.Command("sudo", "systemctl", "stop", fmt.Sprintf("php%s-fpm", c.version))
		return cmd.Run()
	}
	return fmt.Errorf("unsupported platform")
}

// IsRunning checks if PHP-FPM is running
func (c *FPMController) IsRunning() bool {
	switch c.platform.Type {
	case platform.Darwin:
		cmd := exec.Command("pgrep", "-f", fmt.Sprintf("php-fpm.*%s", c.version))
		return cmd.Run() == nil
	case platform.Linux:
		cmd := exec.Command("systemctl", "is-active", fmt.Sprintf("php%s-fpm", c.version))
		output, err := cmd.Output()
		if err != nil {
			return false
		}
		return string(output) == "active\n"
	}
	return false
}

// GetIncludeDirective returns the include path for the pools directory
func (g *PoolGenerator) GetIncludeDirective() string {
	return g.poolsDir + "/*.conf"
}

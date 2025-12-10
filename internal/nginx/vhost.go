package nginx

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/qoliber/magebox/internal/config"
	"github.com/qoliber/magebox/internal/platform"
	"github.com/qoliber/magebox/internal/ssl"
)

//go:embed templates/vhost.conf.tmpl
var vhostTemplate string

// Template variables available in vhost.conf.tmpl:
// - ProjectName: Name of the project (e.g., "mystore")
// - Domain: Domain name (e.g., "mystore.test")
// - DocumentRoot: Absolute path to document root (e.g., "/var/www/mystore/pub")
// - PHPVersion: PHP version (e.g., "8.2")
// - PHPSocketPath: Path to PHP-FPM socket (e.g., "/tmp/magebox/mystore-php8.2.sock")
// - SSLEnabled: Boolean indicating if SSL is enabled
// - SSLCertFile: Path to SSL certificate file (only if SSLEnabled=true)
// - SSLKeyFile: Path to SSL key file (only if SSLEnabled=true)
// - UseVarnish: Boolean indicating if Varnish is enabled (currently unused)
// - VarnishPort: Varnish port number (currently unused)

// VhostGenerator generates Nginx vhost configurations
type VhostGenerator struct {
	platform   *platform.Platform
	sslManager *ssl.Manager
	vhostsDir  string
}

// VhostConfig contains all data needed to generate a vhost
type VhostConfig struct {
	ProjectName   string
	Domain        string
	DocumentRoot  string
	PHPVersion    string
	PHPSocketPath string
	SSLEnabled    bool
	SSLCertFile   string
	SSLKeyFile    string
	UseVarnish    bool
	VarnishPort   int
}

// NewVhostGenerator creates a new vhost generator
func NewVhostGenerator(p *platform.Platform, sslMgr *ssl.Manager) *VhostGenerator {
	return &VhostGenerator{
		platform:   p,
		sslManager: sslMgr,
		vhostsDir:  filepath.Join(p.MageBoxDir(), "nginx", "vhosts"),
	}
}

// Generate generates a vhost configuration for a project
func (g *VhostGenerator) Generate(cfg *config.Config, projectPath string) error {
	// Ensure vhosts directory exists
	if err := os.MkdirAll(g.vhostsDir, 0755); err != nil {
		return fmt.Errorf("failed to create vhosts directory: %w", err)
	}

	for _, domain := range cfg.Domains {
		vhostCfg := VhostConfig{
			ProjectName:   cfg.Name,
			Domain:        domain.Host,
			DocumentRoot:  filepath.Join(projectPath, domain.GetRoot()),
			PHPVersion:    cfg.PHP,
			PHPSocketPath: g.getPHPSocketPath(cfg.Name, cfg.PHP),
			SSLEnabled:    domain.IsSSLEnabled(),
			UseVarnish:    false, // TODO: Add varnish support
			VarnishPort:   6081,
		}

		// Get SSL cert paths if SSL is enabled
		if vhostCfg.SSLEnabled {
			certPaths := g.sslManager.GetCertPaths(ssl.ExtractBaseDomain(domain.Host))
			vhostCfg.SSLCertFile = certPaths.CertFile
			vhostCfg.SSLKeyFile = certPaths.KeyFile
		}

		content, err := g.renderVhost(vhostCfg)
		if err != nil {
			return fmt.Errorf("failed to render vhost for %s: %w", domain.Host, err)
		}

		vhostFile := filepath.Join(g.vhostsDir, fmt.Sprintf("%s-%s.conf", cfg.Name, sanitizeDomain(domain.Host)))
		if err := os.WriteFile(vhostFile, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write vhost file: %w", err)
		}
	}

	return nil
}

// Remove removes vhost configurations for a project
func (g *VhostGenerator) Remove(projectName string) error {
	pattern := filepath.Join(g.vhostsDir, projectName+"-*.conf")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	for _, file := range matches {
		if err := os.Remove(file); err != nil {
			return fmt.Errorf("failed to remove %s: %w", file, err)
		}
	}

	return nil
}

// getPHPSocketPath returns the PHP-FPM socket path for a project
func (g *VhostGenerator) getPHPSocketPath(projectName, phpVersion string) string {
	return filepath.Join("/tmp/magebox", fmt.Sprintf("%s-php%s.sock", projectName, phpVersion))
}

// renderVhost renders the vhost template
func (g *VhostGenerator) renderVhost(cfg VhostConfig) (string, error) {
	tmpl, err := template.New("vhost").Parse(vhostTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, cfg); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// VhostsDir returns the vhosts directory path
func (g *VhostGenerator) VhostsDir() string {
	return g.vhostsDir
}

// ListVhosts returns all vhost files
func (g *VhostGenerator) ListVhosts() ([]string, error) {
	pattern := filepath.Join(g.vhostsDir, "*.conf")
	return filepath.Glob(pattern)
}

// sanitizeDomain converts a domain to a safe filename
func sanitizeDomain(domain string) string {
	return domain // Domains are already safe for filenames
}

// Controller manages Nginx service
type Controller struct {
	platform *platform.Platform
}

// NewController creates a new Nginx controller
func NewController(p *platform.Platform) *Controller {
	return &Controller{platform: p}
}

// Reload reloads Nginx configuration
func (c *Controller) Reload() error {
	cmd := exec.Command("sudo", "nginx", "-s", "reload")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to reload nginx: %w\nOutput: %s", err, output)
	}
	return nil
}

// Test tests Nginx configuration
func (c *Controller) Test() error {
	cmd := exec.Command("sudo", "nginx", "-t")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("nginx configuration test failed: %w\nOutput: %s", err, output)
	}
	return nil
}

// Start starts Nginx
func (c *Controller) Start() error {
	switch c.platform.Type {
	case platform.Darwin:
		cmd := exec.Command("brew", "services", "start", "nginx")
		return cmd.Run()
	case platform.Linux:
		cmd := exec.Command("sudo", "systemctl", "start", "nginx")
		return cmd.Run()
	}
	return fmt.Errorf("unsupported platform")
}

// Stop stops Nginx
func (c *Controller) Stop() error {
	switch c.platform.Type {
	case platform.Darwin:
		cmd := exec.Command("brew", "services", "stop", "nginx")
		return cmd.Run()
	case platform.Linux:
		cmd := exec.Command("sudo", "systemctl", "stop", "nginx")
		return cmd.Run()
	}
	return fmt.Errorf("unsupported platform")
}

// IsRunning checks if Nginx is running
func (c *Controller) IsRunning() bool {
	cmd := exec.Command("pgrep", "nginx")
	return cmd.Run() == nil
}

// GetIncludeDirective returns the include directive to add to nginx.conf
func (g *VhostGenerator) GetIncludeDirective() string {
	return fmt.Sprintf("include %s/*.conf;", g.vhostsDir)
}

// SetupNginxConfig ensures nginx.conf includes MageBox vhosts directory using symlinks
func (c *Controller) SetupNginxConfig() error {
	// Determine nginx servers directory based on platform
	var nginxServersDir string
	switch c.platform.Type {
	case platform.Darwin:
		// Check Homebrew ARM first, then Intel
		if _, err := os.Stat("/opt/homebrew/etc/nginx/servers"); err == nil {
			nginxServersDir = "/opt/homebrew/etc/nginx/servers"
		} else {
			nginxServersDir = "/usr/local/etc/nginx/servers"
		}
	case platform.Linux:
		nginxServersDir = "/etc/nginx/sites-enabled"
	default:
		return fmt.Errorf("unsupported platform")
	}

	// Ensure nginx servers directory exists
	if _, err := os.Stat(nginxServersDir); os.IsNotExist(err) {
		// Try to create it with sudo
		cmd := exec.Command("sudo", "mkdir", "-p", nginxServersDir)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create nginx servers directory: %w", err)
		}
	}

	// Create symlink from nginx servers dir to our magebox vhosts dir
	mageboxVhostsDir := filepath.Join(c.platform.MageBoxDir(), "nginx", "vhosts")
	symlinkPath := filepath.Join(nginxServersDir, "magebox")

	// Check if symlink already exists
	if linkTarget, err := os.Readlink(symlinkPath); err == nil {
		// Symlink exists, check if it points to the right place
		if linkTarget == mageboxVhostsDir {
			return nil // Already configured correctly
		}
		// Remove old symlink
		cmd := exec.Command("sudo", "rm", symlinkPath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to remove old symlink: %w", err)
		}
	}

	// Create symlink with sudo
	cmd := exec.Command("sudo", "ln", "-s", mageboxVhostsDir, symlinkPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	return nil
}

// GetNginxConfPath returns the path to nginx.conf
func (c *Controller) GetNginxConfPath() string {
	switch c.platform.Type {
	case platform.Darwin:
		if _, err := os.Stat("/opt/homebrew/etc/nginx/nginx.conf"); err == nil {
			return "/opt/homebrew/etc/nginx/nginx.conf"
		}
		return "/usr/local/etc/nginx/nginx.conf"
	case platform.Linux:
		return "/etc/nginx/nginx.conf"
	default:
		return "/etc/nginx/nginx.conf"
	}
}

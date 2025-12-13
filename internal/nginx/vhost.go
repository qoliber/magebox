package nginx

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/qoliber/magebox/internal/config"
	"github.com/qoliber/magebox/internal/platform"
	"github.com/qoliber/magebox/internal/ssl"
)

//go:embed templates/vhost.conf.tmpl
var vhostTemplate string

//go:embed templates/proxy.conf.tmpl
var proxyTemplate string

//go:embed templates/upstream.conf.tmpl
var upstreamTemplate string

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
	HTTPPort      int    // 80 on Linux, 8080 on macOS (port forwarding)
	HTTPSPort     int    // 443 on Linux, 8443 on macOS (port forwarding)
	StoreCode     string // Magento store code for multi-store setup (default: "default")
}

// ProxyConfig contains data needed to generate a proxy vhost
type ProxyConfig struct {
	Name        string
	Domain      string
	ProxyHost   string
	ProxyPort   int
	SSLEnabled  bool
	SSLCertFile string
	SSLKeyFile  string
	HTTPPort    int
	HTTPSPort   int
}

// UpstreamConfig contains data needed to generate an upstream config
type UpstreamConfig struct {
	ProjectName   string
	PHPSocketPath string
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

	// Generate upstream config (once per project, not per domain)
	upstreamCfg := UpstreamConfig{
		ProjectName:   cfg.Name,
		PHPSocketPath: g.getPHPSocketPath(cfg.Name, cfg.PHP),
	}
	if err := g.generateUpstream(upstreamCfg); err != nil {
		return fmt.Errorf("failed to generate upstream config: %w", err)
	}

	// Determine ports based on platform
	// macOS uses port forwarding (80->8080, 443->8443), Linux uses standard ports
	httpPort := 80
	httpsPort := 443
	if g.platform.Type == platform.Darwin {
		httpPort = 8080
		httpsPort = 8443
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
			HTTPPort:      httpPort,
			HTTPSPort:     httpsPort,
			StoreCode:     domain.GetStoreCode(),
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

// generateUpstream generates the upstream config file for a project
func (g *VhostGenerator) generateUpstream(cfg UpstreamConfig) error {
	content, err := g.renderUpstream(cfg)
	if err != nil {
		return err
	}

	upstreamFile := filepath.Join(g.vhostsDir, fmt.Sprintf("%s-upstream.conf", cfg.ProjectName))
	return os.WriteFile(upstreamFile, []byte(content), 0644)
}

// renderUpstream renders the upstream template
func (g *VhostGenerator) renderUpstream(cfg UpstreamConfig) (string, error) {
	tmpl, err := template.New("upstream").Parse(upstreamTemplate)
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

// GenerateProxyVhost generates a proxy vhost configuration for a service
func (g *VhostGenerator) GenerateProxyVhost(cfg ProxyConfig) error {
	// Ensure vhosts directory exists
	if err := os.MkdirAll(g.vhostsDir, 0755); err != nil {
		return fmt.Errorf("failed to create vhosts directory: %w", err)
	}

	// Set ports based on platform
	cfg.HTTPPort = 80
	cfg.HTTPSPort = 443
	if g.platform.Type == platform.Darwin {
		cfg.HTTPPort = 8080
		cfg.HTTPSPort = 8443
	}

	// Generate SSL certificate if enabled
	if cfg.SSLEnabled && g.sslManager != nil {
		certPaths, err := g.sslManager.GenerateCert(cfg.Domain)
		if err != nil {
			return fmt.Errorf("failed to generate SSL certificate for %s: %w", cfg.Domain, err)
		}
		cfg.SSLCertFile = certPaths.CertFile
		cfg.SSLKeyFile = certPaths.KeyFile
	}

	content, err := g.renderProxyVhost(cfg)
	if err != nil {
		return fmt.Errorf("failed to render proxy vhost for %s: %w", cfg.Domain, err)
	}

	vhostFile := filepath.Join(g.vhostsDir, fmt.Sprintf("%s.conf", cfg.Name))
	if err := os.WriteFile(vhostFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write proxy vhost file: %w", err)
	}

	return nil
}

// renderProxyVhost renders the proxy vhost template
func (g *VhostGenerator) renderProxyVhost(cfg ProxyConfig) (string, error) {
	tmpl, err := template.New("proxy").Parse(proxyTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, cfg); err != nil {
		return "", err
	}

	return buf.String(), nil
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

// Restart restarts Nginx (needed to pick up new listen ports)
func (c *Controller) Restart() error {
	switch c.platform.Type {
	case platform.Darwin:
		cmd := exec.Command("brew", "services", "restart", "nginx")
		return cmd.Run()
	case platform.Linux:
		cmd := exec.Command("sudo", "systemctl", "restart", "nginx")
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

// SetupNginxConfig ensures nginx includes MageBox vhosts
// On Fedora/RHEL: adds include line to nginx.conf (conf.d/*.conf doesn't support nested dirs)
// On Debian/Ubuntu: creates symlink in sites-enabled
// On macOS: creates symlink in servers directory
func (c *Controller) SetupNginxConfig() error {
	mageboxVhostsDir := filepath.Join(c.platform.MageBoxDir(), "nginx", "vhosts")
	includeDirective := fmt.Sprintf("include %s/*.conf;", mageboxVhostsDir)

	switch c.platform.Type {
	case platform.Darwin:
		// macOS: use symlink approach (Homebrew nginx supports it)
		return c.setupNginxSymlink(mageboxVhostsDir)

	case platform.Linux:
		switch c.platform.LinuxDistro {
		case platform.DistroFedora, platform.DistroArch:
			// Fedora/Arch: add include line directly to nginx.conf
			// because conf.d/*.conf doesn't recurse into symlinked directories
			return c.addIncludeToNginxConf(includeDirective)
		default:
			// Debian/Ubuntu: use sites-enabled symlink
			return c.setupNginxSymlink(mageboxVhostsDir)
		}
	}

	return fmt.Errorf("unsupported platform")
}

// addIncludeToNginxConf adds an include directive to nginx.conf
func (c *Controller) addIncludeToNginxConf(includeDirective string) error {
	nginxConf := c.GetNginxConfPath()

	// Read current config
	content, err := os.ReadFile(nginxConf)
	if err != nil {
		return fmt.Errorf("failed to read nginx.conf: %w", err)
	}

	// Check if include already exists
	if strings.Contains(string(content), includeDirective) {
		return nil // Already configured
	}

	// Find the http { block and add include after the last existing include in conf.d
	// We'll add it right after "include /etc/nginx/conf.d/*.conf;"
	marker := "include /etc/nginx/conf.d/*.conf;"
	if !strings.Contains(string(content), marker) {
		return fmt.Errorf("could not find conf.d include in nginx.conf")
	}

	newContent := strings.Replace(
		string(content),
		marker,
		marker+"\n    "+includeDirective+" # MageBox vhosts",
		1,
	)

	// Write to temp file
	tmpFile, err := os.CreateTemp("", "nginx-conf-*")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.WriteString(newContent); err != nil {
		tmpFile.Close()
		return err
	}
	tmpFile.Close()

	// Copy to nginx.conf with sudo
	cmd := exec.Command("sudo", "cp", tmpPath, nginxConf)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// setupNginxSymlink creates a symlink for platforms that support it
func (c *Controller) setupNginxSymlink(mageboxVhostsDir string) error {
	var nginxServersDir string
	switch c.platform.Type {
	case platform.Darwin:
		if _, err := os.Stat("/opt/homebrew/etc/nginx/servers"); err == nil {
			nginxServersDir = "/opt/homebrew/etc/nginx/servers"
		} else {
			nginxServersDir = "/usr/local/etc/nginx/servers"
		}
	case platform.Linux:
		nginxServersDir = "/etc/nginx/sites-enabled"
	default:
		return fmt.Errorf("unsupported platform for symlink")
	}

	// Ensure nginx servers directory exists
	if _, err := os.Stat(nginxServersDir); os.IsNotExist(err) {
		cmd := exec.Command("sudo", "mkdir", "-p", nginxServersDir)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create nginx servers directory: %w", err)
		}
	}

	symlinkPath := filepath.Join(nginxServersDir, "magebox")

	// Check if symlink already exists
	if linkTarget, err := os.Readlink(symlinkPath); err == nil {
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

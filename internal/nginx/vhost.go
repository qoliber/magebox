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

	"qoliber/magebox/internal/config"
	"qoliber/magebox/internal/lib"
	"qoliber/magebox/internal/php"
	"qoliber/magebox/internal/platform"
	"qoliber/magebox/internal/ssl"
)

//go:embed templates/vhost.conf.tmpl
var vhostTemplateEmbed string

//go:embed templates/proxy.conf.tmpl
var proxyTemplateEmbed string

//go:embed templates/upstream.conf.tmpl
var upstreamTemplateEmbed string

func init() {
	// Register embedded templates as fallbacks
	lib.RegisterFallbackTemplate(lib.TemplateNginx, "vhost.conf.tmpl", vhostTemplateEmbed)
	lib.RegisterFallbackTemplate(lib.TemplateNginx, "proxy.conf.tmpl", proxyTemplateEmbed)
	lib.RegisterFallbackTemplate(lib.TemplateNginx, "upstream.conf.tmpl", upstreamTemplateEmbed)
}

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
	ProjectPath   string // Absolute path to project root (for config file references)
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
	BackendPort   int    // Backend port for Varnish (always 8080 when Varnish enabled)
	StoreCode     string // Magento store code for multi-store setup (default: "default")
	MageRunType   string // Magento run type: "store" or "website" (default: "store")
	AccessLog     string // Path to access log file
	ErrorLog      string // Path to error log file
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

	// Ensure nginx logs directory exists
	logsDir := filepath.Join(g.platform.MageBoxDir(), "logs", "nginx")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("failed to create nginx logs directory: %w", err)
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
		// Backend port for Varnish is always 8080
		backendPort := httpPort
		if cfg.Services.HasVarnish() {
			backendPort = 8080
		}

		// Generate sanitized domain name for log files
		sanitizedDomain := sanitizeDomain(domain.Host)

		vhostCfg := VhostConfig{
			ProjectName:   cfg.Name,
			ProjectPath:   projectPath,
			Domain:        domain.Host,
			DocumentRoot:  filepath.Join(projectPath, domain.GetRoot()),
			PHPVersion:    cfg.PHP,
			PHPSocketPath: g.getPHPSocketPath(cfg.Name, cfg.PHP),
			SSLEnabled:    domain.IsSSLEnabled(),
			UseVarnish:    cfg.Services.HasVarnish(),
			VarnishPort:   6081,
			HTTPPort:      httpPort,
			HTTPSPort:     httpsPort,
			BackendPort:   backendPort,
			StoreCode:     domain.GetStoreCode(),
			MageRunType:   domain.GetMageRunType(),
			AccessLog:     filepath.Join(logsDir, fmt.Sprintf("%s-access.log", sanitizedDomain)),
			ErrorLog:      filepath.Join(logsDir, fmt.Sprintf("%s-error.log", sanitizedDomain)),
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
// If the project has isolation enabled, returns the isolated socket path
func (g *VhostGenerator) getPHPSocketPath(projectName, phpVersion string) string {
	// Check if project has isolation enabled
	isolatedController := php.NewIsolatedFPMController(g.platform)
	if isolatedController.IsIsolated(projectName) {
		return isolatedController.GetSocketPath(projectName, phpVersion)
	}
	// Default to shared pool socket
	return filepath.Join(g.platform.MageBoxDir(), "run", fmt.Sprintf("%s-php%s.sock", projectName, phpVersion))
}

// renderVhost renders the vhost template
func (g *VhostGenerator) renderVhost(cfg VhostConfig) (string, error) {
	tmplContent, err := lib.GetTemplate(lib.TemplateNginx, "vhost.conf.tmpl")
	if err != nil {
		return "", err
	}

	tmpl, err := template.New("vhost").Parse(tmplContent)
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
	tmplContent, err := lib.GetTemplate(lib.TemplateNginx, "upstream.conf.tmpl")
	if err != nil {
		return "", err
	}

	tmpl, err := template.New("upstream").Parse(tmplContent)
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
	tmplContent, err := lib.GetTemplate(lib.TemplateNginx, "proxy.conf.tmpl")
	if err != nil {
		return "", err
	}

	tmpl, err := template.New("proxy").Parse(tmplContent)
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
	switch c.platform.Type {
	case platform.Darwin:
		// On macOS, use nginx -s reload directly (more reliable than brew services)
		cmd := exec.Command("nginx", "-s", "reload")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to reload nginx: %w\nOutput: %s", err, output)
		}
		return nil
	case platform.Linux:
		// Use nginx -s reload directly (more reliable than systemctl reload)
		cmd := exec.Command("sudo", "nginx", "-s", "reload")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to reload nginx: %w\nOutput: %s", err, output)
		}
		return nil
	}
	return fmt.Errorf("unsupported platform")
}

// Test tests Nginx configuration
func (c *Controller) Test() error {
	var cmd *exec.Cmd
	switch c.platform.Type {
	case platform.Darwin:
		cmd = exec.Command("nginx", "-t")
	default:
		cmd = exec.Command("sudo", "nginx", "-t")
	}
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
		// macOS: add explicit include to nginx.conf
		// The default "include servers/*;" doesn't work with symlinked directories
		return c.addIncludeToNginxConfDarwin(includeDirective)

	case platform.Linux:
		// All Linux distros: add include line directly to nginx.conf
		// Symlink approach doesn't work because "include sites-enabled/*" or "include conf.d/*"
		// tries to load the directory instead of *.conf files inside
		return c.addIncludeToNginxConf(includeDirective)
	}

	return fmt.Errorf("unsupported platform")
}

// addIncludeToNginxConfDarwin adds an include directive to macOS nginx.conf
func (c *Controller) addIncludeToNginxConfDarwin(includeDirective string) error {
	nginxConf := c.GetNginxConfPath()

	// Read current config
	content, err := os.ReadFile(nginxConf)
	if err != nil {
		return fmt.Errorf("failed to read nginx.conf: %w", err)
	}

	// Check if include already exists
	if strings.Contains(string(content), includeDirective) {
		// Already configured, but still remove invalid "include servers/*;" if present
		newContent := string(content)
		newContent = strings.Replace(newContent, "include servers/*;", "# include servers/*; # Disabled by MageBox (invalid: loads directories)", 1)
		if newContent != string(content) {
			if err := os.WriteFile(nginxConf, []byte(newContent), 0644); err != nil {
				return fmt.Errorf("failed to write nginx.conf: %w", err)
			}
		}
		return nil
	}

	// Replace "include servers/*;" with our specific include
	// The default "include servers/*;" is invalid as it tries to load directories
	newContent := string(content)
	marker := "include servers/*;"
	if strings.Contains(newContent, marker) {
		newContent = strings.Replace(newContent, marker, includeDirective+" # MageBox vhosts", 1)
	} else {
		// Try to find closing brace of http block and add before it
		// This handles fresh nginx installs or custom configs
		if strings.Contains(newContent, "http {") {
			// Find last } in the file (closing http block)
			lastBrace := strings.LastIndex(newContent, "}")
			if lastBrace > 0 {
				newContent = newContent[:lastBrace] + "    " + includeDirective + " # MageBox vhosts\n" + newContent[lastBrace:]
			} else {
				return fmt.Errorf("could not find http block closing brace in nginx.conf")
			}
		} else {
			return fmt.Errorf("could not find http block in nginx.conf")
		}
	}

	// Write back to nginx.conf (no sudo needed on macOS with Homebrew)
	if err := os.WriteFile(nginxConf, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write nginx.conf: %w", err)
	}

	return nil
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

	var newContent string
	contentStr := string(content)

	// Try multiple markers in order of preference
	markers := []string{
		"include /etc/nginx/conf.d/*.conf;",
		"include /etc/nginx/sites-enabled/*;",
		"include sites-enabled/*;",
	}

	found := false
	for _, marker := range markers {
		if strings.Contains(contentStr, marker) {
			newContent = strings.Replace(
				contentStr,
				marker,
				marker+"\n    "+includeDirective+" # MageBox vhosts",
				1,
			)
			found = true
			break
		}
	}

	// Fallback: find http block closing brace and add before it
	if !found {
		// Look for the last } in the http block
		httpStart := strings.Index(contentStr, "http {")
		if httpStart == -1 {
			httpStart = strings.Index(contentStr, "http{")
		}
		if httpStart != -1 {
			// Find the matching closing brace by counting
			depth := 0
			lastBrace := -1
			for i := httpStart; i < len(contentStr); i++ {
				if contentStr[i] == '{' {
					depth++
				} else if contentStr[i] == '}' {
					depth--
					if depth == 0 {
						lastBrace = i
						break
					}
				}
			}
			if lastBrace != -1 {
				newContent = contentStr[:lastBrace] + "\n    " + includeDirective + " # MageBox vhosts\n" + contentStr[lastBrace:]
				found = true
			}
		}
	}

	if !found {
		return fmt.Errorf("could not find suitable location in nginx.conf to add MageBox include. Please add manually:\n    %s", includeDirective)
	}

	// Write to temp file (use magebox- prefix to match sudoers whitelist)
	tmpFile, err := os.CreateTemp("", "magebox-nginx-*")
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

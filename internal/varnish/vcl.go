package varnish

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/qoliber/magebox/internal/config"
	"github.com/qoliber/magebox/internal/platform"
)

// VCLGenerator generates Varnish VCL configurations
type VCLGenerator struct {
	platform  *platform.Platform
	vclDir    string
}

// BackendConfig represents a backend configuration for VCL
type BackendConfig struct {
	Name       string
	Host       string
	Port       int
	ProbeURL   string
	ProbeInterval string
}

// VCLConfig contains all data needed to generate a VCL file
type VCLConfig struct {
	Backends     []BackendConfig
	DefaultBackend string
	GracePeriod  string
	PurgeACL     []string
}

// NewVCLGenerator creates a new VCL generator
func NewVCLGenerator(p *platform.Platform) *VCLGenerator {
	return &VCLGenerator{
		platform: p,
		vclDir:   filepath.Join(p.MageBoxDir(), "varnish"),
	}
}

// Generate generates the VCL configuration for all projects
func (g *VCLGenerator) Generate(configs []*config.Config) error {
	// Ensure VCL directory exists
	if err := os.MkdirAll(g.vclDir, 0755); err != nil {
		return fmt.Errorf("failed to create VCL directory: %w", err)
	}

	// Build VCL config from all projects
	vclCfg := g.buildVCLConfig(configs)

	// Render VCL
	content, err := g.renderVCL(vclCfg)
	if err != nil {
		return fmt.Errorf("failed to render VCL: %w", err)
	}

	// Write main VCL file
	vclFile := filepath.Join(g.vclDir, "default.vcl")
	if err := os.WriteFile(vclFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write VCL file: %w", err)
	}

	return nil
}

// buildVCLConfig builds the VCL configuration from project configs
func (g *VCLGenerator) buildVCLConfig(configs []*config.Config) VCLConfig {
	vclCfg := VCLConfig{
		Backends:     make([]BackendConfig, 0),
		GracePeriod:  "300s",
		PurgeACL:     []string{"localhost", "127.0.0.1", "::1"},
	}

	for _, cfg := range configs {
		// Each project gets a backend pointing to Nginx
		backend := BackendConfig{
			Name:          sanitizeName(cfg.Name),
			Host:          "127.0.0.1",
			Port:          80,
			ProbeURL:      "/health_check.php",
			ProbeInterval: "5s",
		}
		vclCfg.Backends = append(vclCfg.Backends, backend)

		// First project is default backend
		if vclCfg.DefaultBackend == "" {
			vclCfg.DefaultBackend = backend.Name
		}
	}

	// If no projects, create a default backend
	if len(vclCfg.Backends) == 0 {
		vclCfg.Backends = append(vclCfg.Backends, BackendConfig{
			Name: "default",
			Host: "127.0.0.1",
			Port: 80,
		})
		vclCfg.DefaultBackend = "default"
	}

	return vclCfg
}

// renderVCL renders the VCL template
func (g *VCLGenerator) renderVCL(cfg VCLConfig) (string, error) {
	tmpl, err := template.New("vcl").Parse(vclTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, cfg); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// VCLDir returns the VCL directory path
func (g *VCLGenerator) VCLDir() string {
	return g.vclDir
}

// VCLFilePath returns the path to the main VCL file
func (g *VCLGenerator) VCLFilePath() string {
	return filepath.Join(g.vclDir, "default.vcl")
}

// sanitizeName converts a project name to a valid VCL identifier
func sanitizeName(name string) string {
	// Replace non-alphanumeric characters with underscores
	result := make([]byte, 0, len(name))
	for _, c := range []byte(name) {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			result = append(result, c)
		} else {
			result = append(result, '_')
		}
	}
	return string(result)
}

// Magento 2 optimized VCL template
const vclTemplate = `# MageBox Varnish VCL - Magento 2 Optimized
# Generated automatically - do not edit manually

vcl 4.1;

import std;

# Backend definitions
{{range .Backends}}
backend {{.Name}} {
    .host = "{{.Host}}";
    .port = "{{.Port}}";
    .first_byte_timeout = 600s;
    .between_bytes_timeout = 600s;
    {{if .ProbeURL}}
    .probe = {
        .url = "{{.ProbeURL}}";
        .timeout = 2s;
        .interval = {{.ProbeInterval}};
        .window = 10;
        .threshold = 5;
    }
    {{end}}
}
{{end}}

# ACL for purge requests
acl purge {
{{range .PurgeACL}}
    "{{.}}";
{{end}}
}

sub vcl_init {
    return (ok);
}

sub vcl_recv {
    # Set backend based on Host header
    {{range .Backends}}
    # Backend: {{.Name}}
    {{end}}
    set req.backend_hint = {{.DefaultBackend}};

    # Normalize the host header
    if (req.http.host ~ ":[0-9]+") {
        set req.http.host = regsub(req.http.host, ":[0-9]+", "");
    }

    # Handle PURGE requests
    if (req.method == "PURGE") {
        if (!client.ip ~ purge) {
            return (synth(405, "Method not allowed"));
        }
        if (!req.http.X-Magento-Tags-Pattern && !req.http.X-Pool) {
            return (purge);
        }
        if (req.http.X-Magento-Tags-Pattern) {
            ban("obj.http.X-Magento-Tags ~ " + req.http.X-Magento-Tags-Pattern);
        }
        if (req.http.X-Pool) {
            ban("obj.http.X-Pool ~ " + req.http.X-Pool);
        }
        return (synth(200, "Purged"));
    }

    # Handle BAN requests
    if (req.method == "BAN") {
        if (!client.ip ~ purge) {
            return (synth(405, "Method not allowed"));
        }
        if (req.http.X-Magento-Tags-Pattern) {
            ban("obj.http.X-Magento-Tags ~ " + req.http.X-Magento-Tags-Pattern);
            return (synth(200, "Banned"));
        }
        return (synth(400, "X-Magento-Tags-Pattern header required"));
    }

    # Only cache GET and HEAD requests
    if (req.method != "GET" && req.method != "HEAD") {
        return (pass);
    }

    # Bypass health check requests
    if (req.url ~ "^/(pub/)?(health_check|info)\.php$") {
        return (pass);
    }

    # Normalize Accept-Encoding header
    if (req.http.Accept-Encoding) {
        if (req.http.Accept-Encoding ~ "gzip") {
            set req.http.Accept-Encoding = "gzip";
        } elsif (req.http.Accept-Encoding ~ "deflate") {
            set req.http.Accept-Encoding = "deflate";
        } else {
            unset req.http.Accept-Encoding;
        }
    }

    # Remove marketing tracking parameters
    if (req.url ~ "(\?|&)(gclid|cx|ie|cof|siteurl|zanpid|origin|fbclid|mc_[a-z]+|utm_[a-z]+|_ga|_bta_[a-z]+)=") {
        set req.url = regsuball(req.url, "(gclid|cx|ie|cof|siteurl|zanpid|origin|fbclid|mc_[a-z]+|utm_[a-z]+|_ga|_bta_[a-z]+)=[^&]+&?", "");
        set req.url = regsub(req.url, "(\?|&)$", "");
    }

    # Don't cache requests with authorization
    if (req.http.Authorization) {
        return (pass);
    }

    # Don't cache admin, checkout, customer areas
    if (req.url ~ "^/(pub/)?(index\.php/)?admin" ||
        req.url ~ "^/(pub/)?(index\.php/)?checkout" ||
        req.url ~ "^/(pub/)?(index\.php/)?customer") {
        return (pass);
    }

    # Don't cache if frontend or admin cookies are set
    if (req.http.cookie ~ "frontend=" || req.http.cookie ~ "adminhtml=") {
        return (pass);
    }

    return (hash);
}

sub vcl_hash {
    # Hash based on URL and host
    hash_data(req.url);
    if (req.http.host) {
        hash_data(req.http.host);
    } else {
        hash_data(server.ip);
    }

    # Hash based on SSL
    if (req.http.X-Forwarded-Proto) {
        hash_data(req.http.X-Forwarded-Proto);
    }

    # Hash based on store/currency cookies for Magento
    if (req.http.cookie ~ "X-Magento-Vary=") {
        hash_data(regsub(req.http.cookie, "^.*?X-Magento-Vary=([^;]+);*.*$", "\1"));
    }

    return (lookup);
}

sub vcl_hit {
    # Allow PURGE from localhost
    if (req.method == "PURGE") {
        return (synth(200, "Purged"));
    }
    return (deliver);
}

sub vcl_miss {
    return (fetch);
}

sub vcl_backend_response {
    # Serve stale content if backend is sick
    set beresp.grace = {{.GracePeriod}};

    # Validate response
    if (beresp.status >= 500 && beresp.status < 600) {
        # Don't cache server errors
        set beresp.ttl = 0s;
        set beresp.uncacheable = true;
        return (deliver);
    }

    # Check for Magento TTL header
    if (beresp.http.X-Magento-Cache-Control) {
        set beresp.ttl = std.duration(
            regsub(beresp.http.X-Magento-Cache-Control, "max-age=([0-9]+).*", "\1s"),
            0s
        );
    }

    # Cache static content longer
    if (bereq.url ~ "\.(css|js|jpg|jpeg|png|gif|ico|svg|woff|woff2|ttf|eot)(\?.*)?$") {
        set beresp.ttl = 86400s;
        unset beresp.http.Set-Cookie;
    }

    # Don't cache responses with Set-Cookie
    if (beresp.http.Set-Cookie) {
        set beresp.uncacheable = true;
        return (deliver);
    }

    # Don't cache private responses
    if (beresp.http.Cache-Control ~ "private" ||
        beresp.http.Cache-Control ~ "no-cache" ||
        beresp.http.Cache-Control ~ "no-store") {
        set beresp.uncacheable = true;
        return (deliver);
    }

    return (deliver);
}

sub vcl_deliver {
    # Debug headers (remove in production)
    if (resp.http.X-Magento-Debug) {
        if (obj.hits > 0) {
            set resp.http.X-Cache = "HIT";
            set resp.http.X-Cache-Hits = obj.hits;
        } else {
            set resp.http.X-Cache = "MISS";
        }
    } else {
        # Remove internal headers
        unset resp.http.X-Magento-Debug;
        unset resp.http.X-Magento-Tags;
        unset resp.http.X-Powered-By;
        unset resp.http.Server;
        unset resp.http.X-Varnish;
        unset resp.http.Via;
    }

    return (deliver);
}

sub vcl_synth {
    if (resp.status == 750) {
        set resp.http.Location = req.http.X-Redirect-Url;
        set resp.status = 301;
        return (deliver);
    }
    return (deliver);
}
`

// Controller manages Varnish service
type Controller struct {
	platform *platform.Platform
	vclFile  string
}

// NewController creates a new Varnish controller
func NewController(p *platform.Platform, vclFile string) *Controller {
	return &Controller{
		platform: p,
		vclFile:  vclFile,
	}
}

// Reload reloads Varnish configuration
func (c *Controller) Reload() error {
	// Use varnishadm to reload VCL
	cmd := exec.Command("varnishadm", "vcl.load", "reload", c.vclFile)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to load VCL: %w", err)
	}

	cmd = exec.Command("varnishadm", "vcl.use", "reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to activate VCL: %w", err)
	}

	return nil
}

// Start starts Varnish
func (c *Controller) Start() error {
	switch c.platform.Type {
	case platform.Darwin:
		cmd := exec.Command("brew", "services", "start", "varnish")
		return cmd.Run()
	case platform.Linux:
		cmd := exec.Command("sudo", "systemctl", "start", "varnish")
		return cmd.Run()
	}
	return fmt.Errorf("unsupported platform")
}

// Stop stops Varnish
func (c *Controller) Stop() error {
	switch c.platform.Type {
	case platform.Darwin:
		cmd := exec.Command("brew", "services", "stop", "varnish")
		return cmd.Run()
	case platform.Linux:
		cmd := exec.Command("sudo", "systemctl", "stop", "varnish")
		return cmd.Run()
	}
	return fmt.Errorf("unsupported platform")
}

// IsRunning checks if Varnish is running
func (c *Controller) IsRunning() bool {
	cmd := exec.Command("pgrep", "varnishd")
	return cmd.Run() == nil
}

// Purge sends a purge request to Varnish
func (c *Controller) Purge(host, url string) error {
	cmd := exec.Command("curl", "-X", "PURGE", "-H", "Host: "+host, "http://127.0.0.1:6081"+url)
	return cmd.Run()
}

// Ban sends a ban request to Varnish
func (c *Controller) Ban(pattern string) error {
	cmd := exec.Command("curl", "-X", "BAN", "-H", "X-Magento-Tags-Pattern: "+pattern, "http://127.0.0.1:6081/")
	return cmd.Run()
}

// FlushAll flushes all cached content
func (c *Controller) FlushAll() error {
	cmd := exec.Command("varnishadm", "ban", "req.url", "~", ".")
	return cmd.Run()
}

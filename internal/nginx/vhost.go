package nginx

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/qoliber/magebox/internal/config"
	"github.com/qoliber/magebox/internal/platform"
	"github.com/qoliber/magebox/internal/ssl"
)

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

// Nginx vhost template for Magento 2
const vhostTemplate = `# MageBox generated vhost for {{.ProjectName}} - {{.Domain}}
# Do not edit manually - regenerated on magebox start

upstream fastcgi_backend_{{.ProjectName}} {
    server unix:{{.PHPSocketPath}};
}

{{if .SSLEnabled}}
server {
    listen 8080;
    server_name {{.Domain}};
    return 301 https://$host$request_uri;
}

server {
    listen 8443 ssl http2;
    server_name {{.Domain}};

    ssl_certificate {{.SSLCertFile}};
    ssl_certificate_key {{.SSLKeyFile}};
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;
{{else}}
server {
    listen 8080;
    server_name {{.Domain}};
{{end}}

    set $MAGE_ROOT {{.DocumentRoot}};
    set $MAGE_MODE developer;

    root $MAGE_ROOT;
    index index.php;

    autoindex off;
    charset UTF-8;
    error_page 404 403 = /errors/404.php;

    # Deny access to sensitive files
    location /.user.ini {
        deny all;
    }

    location / {
        try_files $uri $uri/ /index.php$is_args$args;
    }

    location /pub/ {
        location ~ ^/pub/media/(downloadable|customer|import|custom_options|theme_customization/.*\.xml) {
            deny all;
        }
        alias $MAGE_ROOT/pub/;
        add_header X-Frame-Options "SAMEORIGIN";
    }

    location /static/ {
        expires max;

        location ~ ^/static/version\d*/ {
            rewrite ^/static/version\d*/(.*)$ /static/$1 last;
        }

        location ~* \.(ico|jpg|jpeg|png|gif|svg|svgz|webp|avif|avifs|js|css|eot|ttf|otf|woff|woff2|html|json|webmanifest)$ {
            add_header Cache-Control "public";
            add_header X-Frame-Options "SAMEORIGIN";
            expires +1y;

            if (!-f $request_filename) {
                rewrite ^/static/(version\d*/)?(.*)$ /static.php?resource=$2 last;
            }
        }
        location ~* \.(zip|gz|gzip|bz2|csv|xml)$ {
            add_header Cache-Control "no-store";
            add_header X-Frame-Options "SAMEORIGIN";
            expires off;

            if (!-f $request_filename) {
               rewrite ^/static/(version\d*/)?(.*)$ /static.php?resource=$2 last;
            }
        }
        if (!-f $request_filename) {
            rewrite ^/static/(version\d*/)?(.*)$ /static.php?resource=$2 last;
        }
        add_header X-Frame-Options "SAMEORIGIN";
    }

    location /media/ {
        try_files $uri $uri/ /get.php$is_args$args;

        location ~ ^/media/theme_customization/.*\.xml {
            deny all;
        }

        location ~* \.(ico|jpg|jpeg|png|gif|svg|svgz|webp|avif|avifs|js|css|eot|ttf|otf|woff|woff2)$ {
            add_header Cache-Control "public";
            add_header X-Frame-Options "SAMEORIGIN";
            expires +1y;
            try_files $uri $uri/ /get.php$is_args$args;
        }
        location ~* \.(zip|gz|gzip|bz2|csv|xml)$ {
            add_header Cache-Control "no-store";
            add_header X-Frame-Options "SAMEORIGIN";
            expires off;
            try_files $uri $uri/ /get.php$is_args$args;
        }
        add_header X-Frame-Options "SAMEORIGIN";
    }

    location /media/customer/ {
        deny all;
    }

    location /media/downloadable/ {
        deny all;
    }

    location /media/import/ {
        deny all;
    }

    location /media/custom_options/ {
        deny all;
    }

    location /errors/ {
        location ~* \.xml$ {
            deny all;
        }
    }

    location ~ \.php$ {
        try_files $uri =404;
        fastcgi_pass fastcgi_backend_{{.ProjectName}};
        fastcgi_buffers 16 16k;
        fastcgi_buffer_size 32k;

        fastcgi_param PHP_FLAG "session.auto_start=off \n suhosin.session.cryptua=off";
        fastcgi_param PHP_VALUE "memory_limit=756M \n max_execution_time=18000";
        fastcgi_read_timeout 600s;
        fastcgi_connect_timeout 600s;

        fastcgi_index index.php;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
        include fastcgi_params;
    }

    gzip on;
    gzip_disable "msie6";
    gzip_comp_level 6;
    gzip_min_length 1100;
    gzip_buffers 16 8k;
    gzip_proxied any;
    gzip_types
        text/plain
        text/css
        text/js
        text/xml
        text/javascript
        application/javascript
        application/x-javascript
        application/json
        application/xml
        application/xml+rss
        image/svg+xml;
    gzip_vary on;

    location ~* (\.php$|\.phtml$|\.htaccess$|\.git) {
        deny all;
    }
}
`

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

package varnish

import (
	"bytes"
	_ "embed"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/qoliber/magebox/internal/config"
	"github.com/qoliber/magebox/internal/platform"
)

//go:embed templates/default.vcl.tmpl
var vclTemplate string

// Template variables available in default.vcl.tmpl:
// - Backends: Array of backend configurations
//   - Name: Backend name (sanitized project name)
//   - Host: Backend host (e.g., "127.0.0.1")
//   - Port: Backend port (e.g., 80)
//   - ProbeURL: Health check URL (e.g., "/health_check.php")
//   - ProbeInterval: Health check interval (e.g., "5s")
// - DefaultBackend: Name of the default backend to use
// - GracePeriod: Grace period for serving stale content (e.g., "300s")
// - PurgeACL: Array of IP addresses/ranges allowed to purge (e.g., ["localhost", "127.0.0.1"])

// VCLGenerator generates Varnish VCL configurations
type VCLGenerator struct {
	platform *platform.Platform
	vclDir   string
}

// BackendConfig represents a backend configuration for VCL
type BackendConfig struct {
	Name          string
	Host          string
	Port          int
	ProbeURL      string
	ProbeInterval string
}

// VCLConfig contains all data needed to generate a VCL file
type VCLConfig struct {
	Backends       []BackendConfig
	DefaultBackend string
	GracePeriod    string
	PurgeACL       []string
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
		Backends:    make([]BackendConfig, 0),
		GracePeriod: "300s",
		PurgeACL:    []string{"localhost", "127.0.0.1", "::1", "host.docker.internal"},
	}

	// Backend host - detect host IP for Docker to reach nginx
	backendHost := getHostIP()
	backendPort := 8080 // Nginx listens on 8080

	for _, cfg := range configs {
		// Each project gets a backend pointing to Nginx
		backend := BackendConfig{
			Name:          sanitizeName(cfg.Name),
			Host:          backendHost,
			Port:          backendPort,
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
			Host: backendHost,
			Port: backendPort,
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

// Reload reloads Varnish configuration by restarting the container
func (c *Controller) Reload() error {
	cmd := exec.Command("docker", "restart", "magebox-varnish")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to restart Varnish: %w", err)
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

// IsRunning checks if Varnish Docker container is running
func (c *Controller) IsRunning() bool {
	cmd := exec.Command("docker", "ps", "--filter", "name=magebox-varnish", "--filter", "status=running", "-q")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(output) > 0
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
	cmd := exec.Command("docker", "exec", "magebox-varnish", "varnishadm", "ban", "req.url", "~", ".")
	return cmd.Run()
}

// getHostIP returns the IP address that Docker containers use to reach the host
func getHostIP() string {
	// Try to get the preferred outbound IP (LAN IP)
	// This is more reliable than host.docker.internal for some Docker runtimes
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "host.docker.internal"
	}
	defer func() { _ = conn.Close() }()

	localAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return "host.docker.internal"
	}
	return localAddr.IP.String()
}

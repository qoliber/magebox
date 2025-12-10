package nginx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/qoliber/magebox/internal/config"
	"github.com/qoliber/magebox/internal/platform"
	"github.com/qoliber/magebox/internal/ssl"
)

func setupTestGenerator(t *testing.T) (*VhostGenerator, string) {
	tmpDir := t.TempDir()
	p := &platform.Platform{
		Type:    platform.Linux,
		HomeDir: tmpDir,
	}
	sslMgr := ssl.NewManager(p)
	g := NewVhostGenerator(p, sslMgr)
	return g, tmpDir
}

func TestNewVhostGenerator(t *testing.T) {
	g, tmpDir := setupTestGenerator(t)

	if g == nil {
		t.Error("NewVhostGenerator should not return nil")
	}

	expectedDir := filepath.Join(tmpDir, ".magebox", "nginx", "vhosts")
	if g.vhostsDir != expectedDir {
		t.Errorf("vhostsDir = %v, want %v", g.vhostsDir, expectedDir)
	}
}

func TestVhostGenerator_VhostsDir(t *testing.T) {
	g, tmpDir := setupTestGenerator(t)

	expected := filepath.Join(tmpDir, ".magebox", "nginx", "vhosts")
	if got := g.VhostsDir(); got != expected {
		t.Errorf("VhostsDir() = %v, want %v", got, expected)
	}
}

func TestVhostGenerator_Generate(t *testing.T) {
	g, tmpDir := setupTestGenerator(t)

	projectPath := filepath.Join(tmpDir, "projects", "mystore")
	cfg := &config.Config{
		Name: "mystore",
		Domains: []config.Domain{
			{Host: "mystore.test", Root: "pub"},
		},
		PHP: "8.2",
	}

	err := g.Generate(cfg, projectPath)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check that vhost file was created
	vhostFile := filepath.Join(g.vhostsDir, "mystore-mystore.test.conf")
	if _, err := os.Stat(vhostFile); os.IsNotExist(err) {
		t.Error("Vhost file should have been created")
	}

	// Read and verify content
	content, err := os.ReadFile(vhostFile)
	if err != nil {
		t.Fatalf("Failed to read vhost file: %v", err)
	}

	contentStr := string(content)

	// Verify essential parts are present
	checks := []string{
		"server_name mystore.test",
		"fastcgi_backend_mystore",
		projectPath + "/pub",
		".sock",
	}

	for _, check := range checks {
		if !strings.Contains(contentStr, check) {
			t.Errorf("Vhost content should contain %q", check)
		}
	}
}

func TestVhostGenerator_GenerateMultipleDomains(t *testing.T) {
	g, tmpDir := setupTestGenerator(t)

	projectPath := filepath.Join(tmpDir, "projects", "mystore")
	cfg := &config.Config{
		Name: "mystore",
		Domains: []config.Domain{
			{Host: "mystore.test"},
			{Host: "api.mystore.test"},
		},
		PHP: "8.2",
	}

	err := g.Generate(cfg, projectPath)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check that both vhost files were created
	files := []string{
		filepath.Join(g.vhostsDir, "mystore-mystore.test.conf"),
		filepath.Join(g.vhostsDir, "mystore-api.mystore.test.conf"),
	}

	for _, file := range files {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("Vhost file %s should have been created", file)
		}
	}
}

func TestVhostGenerator_Remove(t *testing.T) {
	g, tmpDir := setupTestGenerator(t)

	projectPath := filepath.Join(tmpDir, "projects", "mystore")
	cfg := &config.Config{
		Name: "mystore",
		Domains: []config.Domain{
			{Host: "mystore.test"},
			{Host: "api.mystore.test"},
		},
		PHP: "8.2",
	}

	// Generate vhosts first
	if err := g.Generate(cfg, projectPath); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Remove them
	if err := g.Remove("mystore"); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	// Verify they're gone
	files, err := g.ListVhosts()
	if err != nil {
		t.Fatalf("ListVhosts failed: %v", err)
	}

	for _, file := range files {
		if strings.Contains(file, "mystore") {
			t.Errorf("Vhost file %s should have been removed", file)
		}
	}
}

func TestVhostGenerator_ListVhosts(t *testing.T) {
	g, _ := setupTestGenerator(t)

	// Create vhosts directory and some files
	if err := os.MkdirAll(g.vhostsDir, 0755); err != nil {
		t.Fatalf("Failed to create vhosts dir: %v", err)
	}

	files := []string{"project1-domain.test.conf", "project2-other.test.conf"}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(g.vhostsDir, f), []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	vhosts, err := g.ListVhosts()
	if err != nil {
		t.Fatalf("ListVhosts failed: %v", err)
	}

	if len(vhosts) != 2 {
		t.Errorf("ListVhosts returned %d files, want 2", len(vhosts))
	}
}

func TestVhostGenerator_GetIncludeDirective(t *testing.T) {
	g, tmpDir := setupTestGenerator(t)

	expected := "include " + filepath.Join(tmpDir, ".magebox", "nginx", "vhosts") + "/*.conf;"
	if got := g.GetIncludeDirective(); got != expected {
		t.Errorf("GetIncludeDirective() = %v, want %v", got, expected)
	}
}

func TestVhostGenerator_getPHPSocketPath(t *testing.T) {
	g, _ := setupTestGenerator(t)

	expected := "/tmp/magebox/mystore-php8.2.sock"
	if got := g.getPHPSocketPath("mystore", "8.2"); got != expected {
		t.Errorf("getPHPSocketPath() = %v, want %v", got, expected)
	}
}

func TestRenderVhost_SSLEnabled(t *testing.T) {
	g, tmpDir := setupTestGenerator(t)

	cfg := VhostConfig{
		ProjectName:   "mystore",
		Domain:        "mystore.test",
		DocumentRoot:  "/var/www/mystore/pub",
		PHPVersion:    "8.2",
		PHPSocketPath: filepath.Join(tmpDir, ".magebox", "run", "mystore-php8.2.sock"),
		SSLEnabled:    true,
		SSLCertFile:   "/path/to/cert.pem",
		SSLKeyFile:    "/path/to/key.pem",
	}

	content, err := g.renderVhost(cfg)
	if err != nil {
		t.Fatalf("renderVhost failed: %v", err)
	}

	// Should contain SSL configuration
	checks := []string{
		"listen 8443 ssl",
		"ssl_certificate /path/to/cert.pem",
		"ssl_certificate_key /path/to/key.pem",
		"return 301 https://",
	}

	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("SSL vhost should contain %q", check)
		}
	}
}

func TestRenderVhost_SSLDisabled(t *testing.T) {
	g, tmpDir := setupTestGenerator(t)

	cfg := VhostConfig{
		ProjectName:   "mystore",
		Domain:        "mystore.test",
		DocumentRoot:  "/var/www/mystore/pub",
		PHPVersion:    "8.2",
		PHPSocketPath: filepath.Join(tmpDir, ".magebox", "run", "mystore-php8.2.sock"),
		SSLEnabled:    false,
	}

	content, err := g.renderVhost(cfg)
	if err != nil {
		t.Fatalf("renderVhost failed: %v", err)
	}

	// Should NOT contain SSL configuration
	if strings.Contains(content, "listen 443") {
		t.Error("Non-SSL vhost should not contain listen 443")
	}
	if strings.Contains(content, "ssl_certificate") {
		t.Error("Non-SSL vhost should not contain ssl_certificate")
	}
}

func TestNewController(t *testing.T) {
	p := &platform.Platform{Type: platform.Linux}
	c := NewController(p)

	if c == nil {
		t.Error("NewController should not return nil")
	}
}

func TestSanitizeDomain(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"mystore.test", "mystore.test"},
		{"api.mystore.test", "api.mystore.test"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := sanitizeDomain(tt.input); got != tt.expected {
				t.Errorf("sanitizeDomain(%v) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestVhostTemplateValidity(t *testing.T) {
	// Test that the embedded template parses correctly
	tmpl, err := template.New("vhost").Parse(vhostTemplate)
	if err != nil {
		t.Fatalf("Vhost template parsing failed: %v", err)
	}

	if tmpl == nil {
		t.Error("Parsed template should not be nil")
	}

	// Verify template contains expected sections
	templateStr := vhostTemplate
	expectedSections := []string{
		"upstream fastcgi_backend_{{.ProjectName}}",
		"server_name {{.Domain}}",
		"root $MAGE_ROOT",
		"{{if .SSLEnabled}}",
		"ssl_certificate {{.SSLCertFile}}",
		"fastcgi_pass fastcgi_backend_{{.ProjectName}}",
		"location /static/",
		"location /media/",
	}

	for _, section := range expectedSections {
		if !strings.Contains(templateStr, section) {
			t.Errorf("Template should contain section: %s", section)
		}
	}
}

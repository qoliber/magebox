package varnish

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"qoliber/magebox/internal/config"
	"qoliber/magebox/internal/platform"
)

func setupTestVCLGenerator(t *testing.T) (*VCLGenerator, string) {
	tmpDir := t.TempDir()
	p := &platform.Platform{
		Type:    platform.Linux,
		HomeDir: tmpDir,
	}
	g := NewVCLGenerator(p)
	return g, tmpDir
}

func TestNewVCLGenerator(t *testing.T) {
	g, tmpDir := setupTestVCLGenerator(t)

	if g == nil {
		t.Fatal("NewVCLGenerator should not return nil")
	}

	expectedDir := filepath.Join(tmpDir, ".magebox", "varnish")
	if g.vclDir != expectedDir {
		t.Errorf("vclDir = %v, want %v", g.vclDir, expectedDir)
	}
}

func TestVCLGenerator_VCLDir(t *testing.T) {
	g, tmpDir := setupTestVCLGenerator(t)

	expected := filepath.Join(tmpDir, ".magebox", "varnish")
	if got := g.VCLDir(); got != expected {
		t.Errorf("VCLDir() = %v, want %v", got, expected)
	}
}

func TestVCLGenerator_VCLFilePath(t *testing.T) {
	g, tmpDir := setupTestVCLGenerator(t)

	expected := filepath.Join(tmpDir, ".magebox", "varnish", "default.vcl")
	if got := g.VCLFilePath(); got != expected {
		t.Errorf("VCLFilePath() = %v, want %v", got, expected)
	}
}

func TestVCLGenerator_Generate(t *testing.T) {
	g, _ := setupTestVCLGenerator(t)

	configs := []*config.Config{
		{
			Name: "mystore",
			Domains: []config.Domain{
				{Host: "mystore.test"},
			},
			PHP: "8.2",
		},
	}

	err := g.Generate(configs)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check that VCL file was created
	vclFile := g.VCLFilePath()
	if _, err := os.Stat(vclFile); os.IsNotExist(err) {
		t.Error("VCL file should have been created")
	}

	// Read and verify content
	content, err := os.ReadFile(vclFile)
	if err != nil {
		t.Fatalf("Failed to read VCL file: %v", err)
	}

	contentStr := string(content)

	// Verify essential VCL elements
	checks := []string{
		"vcl 4.1",
		"backend mystore",
		"acl purge",
		"sub vcl_recv",
		"sub vcl_hash",
		"sub vcl_backend_response",
		"sub vcl_deliver",
		"X-Magento-Tags",
	}

	for _, check := range checks {
		if !strings.Contains(contentStr, check) {
			t.Errorf("VCL content should contain %q", check)
		}
	}
}

func TestVCLGenerator_GenerateMultipleProjects(t *testing.T) {
	g, _ := setupTestVCLGenerator(t)

	configs := []*config.Config{
		{
			Name: "store1",
			Domains: []config.Domain{
				{Host: "store1.test"},
			},
			PHP: "8.2",
		},
		{
			Name: "store2",
			Domains: []config.Domain{
				{Host: "store2.test"},
			},
			PHP: "8.3",
		},
	}

	err := g.Generate(configs)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content, err := os.ReadFile(g.VCLFilePath())
	if err != nil {
		t.Fatalf("Failed to read VCL file: %v", err)
	}

	contentStr := string(content)

	// Both backends should be present
	if !strings.Contains(contentStr, "backend store1") {
		t.Error("VCL should contain backend store1")
	}
	if !strings.Contains(contentStr, "backend store2") {
		t.Error("VCL should contain backend store2")
	}
}

func TestVCLGenerator_GenerateEmptyConfigs(t *testing.T) {
	g, _ := setupTestVCLGenerator(t)

	// Generate with no configs - should create default backend
	err := g.Generate([]*config.Config{})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content, err := os.ReadFile(g.VCLFilePath())
	if err != nil {
		t.Fatalf("Failed to read VCL file: %v", err)
	}

	contentStr := string(content)

	// Should have default backend
	if !strings.Contains(contentStr, "backend default") {
		t.Error("VCL should contain default backend when no configs provided")
	}
}

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"mystore", "mystore"},
		{"my-store", "my_store"},
		{"my_store", "my_store"},
		{"my.store", "my_store"},
		{"MyStore123", "MyStore123"},
		{"store@123", "store_123"},
		{"store with spaces", "store_with_spaces"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := sanitizeName(tt.input); got != tt.expected {
				t.Errorf("sanitizeName(%v) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestVCLGenerator_buildVCLConfig(t *testing.T) {
	g, _ := setupTestVCLGenerator(t)

	configs := []*config.Config{
		{Name: "store1"},
		{Name: "store2"},
	}

	vclCfg := g.buildVCLConfig(configs)

	// Should have 2 backends
	if len(vclCfg.Backends) != 2 {
		t.Errorf("Expected 2 backends, got %d", len(vclCfg.Backends))
	}

	// First backend should be default
	if vclCfg.DefaultBackend != "store1" {
		t.Errorf("DefaultBackend = %v, want store1", vclCfg.DefaultBackend)
	}

	// Should have purge ACL
	if len(vclCfg.PurgeACL) == 0 {
		t.Error("PurgeACL should not be empty")
	}

	// Grace period should be set
	if vclCfg.GracePeriod == "" {
		t.Error("GracePeriod should not be empty")
	}
}

func TestBackendConfig(t *testing.T) {
	backend := BackendConfig{
		Name:          "mystore",
		Host:          "127.0.0.1",
		Port:          80,
		ProbeURL:      "/health_check.php",
		ProbeInterval: "5s",
	}

	if backend.Name != "mystore" {
		t.Error("Name should be set")
	}
	if backend.Host != "127.0.0.1" {
		t.Error("Host should be 127.0.0.1")
	}
	if backend.Port != 80 {
		t.Error("Port should be 80")
	}
}

func TestVCLConfig(t *testing.T) {
	cfg := VCLConfig{
		Backends: []BackendConfig{
			{Name: "store1", Host: "127.0.0.1", Port: 80},
		},
		DefaultBackend: "store1",
		GracePeriod:    "300s",
		PurgeACL:       []string{"localhost", "127.0.0.1"},
	}

	if len(cfg.Backends) != 1 {
		t.Error("Should have 1 backend")
	}
	if cfg.DefaultBackend != "store1" {
		t.Error("DefaultBackend should be store1")
	}
}

func TestNewController(t *testing.T) {
	p := &platform.Platform{Type: platform.Linux}
	c := NewController(p, "/path/to/default.vcl")

	if c == nil {
		t.Fatal("NewController should not return nil")
	}
	if c.vclFile != "/path/to/default.vcl" {
		t.Errorf("vclFile = %v, want /path/to/default.vcl", c.vclFile)
	}
}

func TestVCLTemplate_MagentoSpecific(t *testing.T) {
	g, _ := setupTestVCLGenerator(t)

	configs := []*config.Config{
		{Name: "magento"},
	}

	err := g.Generate(configs)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content, err := os.ReadFile(g.VCLFilePath())
	if err != nil {
		t.Fatalf("Failed to read VCL file: %v", err)
	}

	contentStr := string(content)

	// Check for Magento-specific VCL features
	magentoFeatures := []string{
		"X-Magento-Tags",
		"X-Magento-Cache-Control",
		"X-Magento-Vary",
		"X-Magento-Debug",
		"admin",
		"checkout",
		"customer",
		"frontend=",
		"adminhtml=",
		"PURGE",
		"BAN",
	}

	for _, feature := range magentoFeatures {
		if !strings.Contains(contentStr, feature) {
			t.Errorf("VCL should contain Magento feature %q", feature)
		}
	}
}

func TestVCLTemplate_CacheControl(t *testing.T) {
	g, _ := setupTestVCLGenerator(t)

	configs := []*config.Config{
		{Name: "store"},
	}

	err := g.Generate(configs)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content, err := os.ReadFile(g.VCLFilePath())
	if err != nil {
		t.Fatalf("Failed to read VCL file: %v", err)
	}

	contentStr := string(content)

	// Check for cache control features
	cacheFeatures := []string{
		"beresp.ttl",
		"beresp.grace",
		"beresp.uncacheable",
		"Cache-Control",
		"Set-Cookie",
		"no-cache",
		"private",
	}

	for _, feature := range cacheFeatures {
		if !strings.Contains(contentStr, feature) {
			t.Errorf("VCL should contain cache control feature %q", feature)
		}
	}
}

func TestVCLTemplate_StaticContent(t *testing.T) {
	g, _ := setupTestVCLGenerator(t)

	configs := []*config.Config{
		{Name: "store"},
	}

	err := g.Generate(configs)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content, err := os.ReadFile(g.VCLFilePath())
	if err != nil {
		t.Fatalf("Failed to read VCL file: %v", err)
	}

	contentStr := string(content)

	// Check for static content handling
	staticExtensions := []string{"css", "js", "jpg", "png", "gif", "svg", "woff"}
	for _, ext := range staticExtensions {
		if !strings.Contains(contentStr, ext) {
			t.Errorf("VCL should handle .%s files", ext)
		}
	}
}

func TestVCLTemplateValidity(t *testing.T) {
	// Test that the embedded template parses correctly
	tmpl, err := template.New("vcl").Parse(vclTemplateEmbed)
	if err != nil {
		t.Fatalf("VCL template parsing failed: %v", err)
	}

	if tmpl == nil {
		t.Error("Parsed template should not be nil")
	}

	// Verify template contains expected sections
	templateStr := vclTemplateEmbed
	expectedSections := []string{
		"vcl 4.1",
		"{{range .Backends}}",
		"backend {{.Name}}",
		"acl purge",
		"{{range .PurgeACL}}",
		"sub vcl_recv",
		"sub vcl_hash",
		"sub vcl_backend_response",
		"beresp.grace = {{.GracePeriod}}",
		"set req.backend_hint = {{.DefaultBackend}}",
		"X-Magento-Tags",
	}

	for _, section := range expectedSections {
		if !strings.Contains(templateStr, section) {
			t.Errorf("Template should contain section: %s", section)
		}
	}
}

package php

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"qoliber/magebox/internal/platform"
)

func setupTestPoolGenerator(t *testing.T) (*PoolGenerator, string) {
	tmpDir := t.TempDir()
	p := &platform.Platform{
		Type:    platform.Linux,
		HomeDir: tmpDir,
	}
	g := NewPoolGenerator(p)
	return g, tmpDir
}

func TestNewPoolGenerator(t *testing.T) {
	g, tmpDir := setupTestPoolGenerator(t)

	if g == nil {
		t.Fatal("NewPoolGenerator should not return nil")
	}

	expectedPoolsDir := filepath.Join(tmpDir, ".magebox", "php", "pools")
	if g.PoolsDir() != expectedPoolsDir {
		t.Errorf("PoolsDir() = %v, want %v", g.PoolsDir(), expectedPoolsDir)
	}

	expectedRunDir := filepath.Join(tmpDir, ".magebox", "run")
	if g.RunDir() != expectedRunDir {
		t.Errorf("RunDir() = %v, want %v", g.RunDir(), expectedRunDir)
	}
}

func TestPoolGenerator_PoolsDir(t *testing.T) {
	g, tmpDir := setupTestPoolGenerator(t)

	expected := filepath.Join(tmpDir, ".magebox", "php", "pools")
	if got := g.PoolsDir(); got != expected {
		t.Errorf("PoolsDir() = %v, want %v", got, expected)
	}
}

func TestPoolGenerator_RunDir(t *testing.T) {
	g, tmpDir := setupTestPoolGenerator(t)

	expected := filepath.Join(tmpDir, ".magebox", "run")
	if got := g.RunDir(); got != expected {
		t.Errorf("RunDir() = %v, want %v", got, expected)
	}
}

func TestPoolGenerator_GetSocketPath(t *testing.T) {
	g, tmpDir := setupTestPoolGenerator(t)

	expected := filepath.Join(tmpDir, ".magebox", "run", "mystore-php8.2.sock")
	if got := g.GetSocketPath("mystore", "8.2"); got != expected {
		t.Errorf("GetSocketPath() = %v, want %v", got, expected)
	}
}

func TestPoolGenerator_Generate(t *testing.T) {
	g, _ := setupTestPoolGenerator(t)

	env := map[string]string{
		"MAGE_MODE": "developer",
	}

	phpIni := map[string]string{}

	err := g.Generate("mystore", "/tmp/mystore", "8.2", env, phpIni, false)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check that pool file was created
	poolFile := filepath.Join(g.PoolsDirForVersion("8.2"), "mystore.conf")
	if _, err := os.Stat(poolFile); os.IsNotExist(err) {
		t.Error("Pool file should have been created")
	}

	// Read and verify content
	content, err := os.ReadFile(poolFile)
	if err != nil {
		t.Fatalf("Failed to read pool file: %v", err)
	}

	contentStr := string(content)

	// Verify essential parts are present
	checks := []string{
		"[mystore]",
		"listen =",
		".sock",
		"pm = dynamic",
		"memory_limit",
		"MAGE_MODE",
	}

	for _, check := range checks {
		if !strings.Contains(contentStr, check) {
			t.Errorf("Pool content should contain %q", check)
		}
	}
}

func TestPoolGenerator_GenerateWithoutEnv(t *testing.T) {
	g, _ := setupTestPoolGenerator(t)

	err := g.Generate("mystore", "/tmp/mystore", "8.3", nil, nil, false)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check that pool file was created in version-specific directory
	poolFile := filepath.Join(g.PoolsDirForVersion("8.3"), "mystore.conf")
	if _, err := os.Stat(poolFile); os.IsNotExist(err) {
		t.Error("Pool file should have been created")
	}
}

func TestPoolGenerator_Remove(t *testing.T) {
	g, _ := setupTestPoolGenerator(t)

	// Generate pool first
	if err := g.Generate("mystore", "/tmp/mystore", "8.2", nil, nil, false); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Remove it
	if err := g.Remove("mystore"); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	// Verify it's gone
	poolFile := filepath.Join(g.PoolsDirForVersion("8.2"), "mystore.conf")
	if _, err := os.Stat(poolFile); !os.IsNotExist(err) {
		t.Error("Pool file should have been removed")
	}
}

func TestPoolGenerator_RemoveNonExistent(t *testing.T) {
	g, _ := setupTestPoolGenerator(t)

	// Should not error when removing non-existent pool
	if err := g.Remove("nonexistent"); err != nil {
		t.Errorf("Remove should not error for non-existent pool: %v", err)
	}
}

func TestPoolGenerator_ListPools(t *testing.T) {
	g, _ := setupTestPoolGenerator(t)

	// Create some pool files
	if err := g.Generate("project1", "/tmp/project1", "8.2", nil, nil, false); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if err := g.Generate("project2", "/tmp/project2", "8.3", nil, nil, false); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	pools, err := g.ListPools()
	if err != nil {
		t.Fatalf("ListPools failed: %v", err)
	}

	if len(pools) != 2 {
		t.Errorf("ListPools returned %d pools, want 2", len(pools))
	}
}

func TestPoolGenerator_GetIncludeDirective(t *testing.T) {
	g, tmpDir := setupTestPoolGenerator(t)

	// GetIncludeDirective returns pattern for all versions
	expected := filepath.Join(tmpDir, ".magebox", "php", "pools") + "/*/*.conf"
	if got := g.GetIncludeDirective(); got != expected {
		t.Errorf("GetIncludeDirective() = %v, want %v", got, expected)
	}

	// GetIncludeDirectiveForVersion returns pattern for specific version
	expectedVersion := filepath.Join(tmpDir, ".magebox", "php", "pools", "8.2") + "/*.conf"
	if got := g.GetIncludeDirectiveForVersion("8.2"); got != expectedVersion {
		t.Errorf("GetIncludeDirectiveForVersion() = %v, want %v", got, expectedVersion)
	}
}

func TestPoolConfig_Defaults(t *testing.T) {
	g, _ := setupTestPoolGenerator(t)

	// Generate and read back to verify defaults
	if err := g.Generate("testproject", "/tmp/testproject", "8.2", nil, nil, false); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	poolFile := filepath.Join(g.PoolsDirForVersion("8.2"), "testproject.conf")
	content, err := os.ReadFile(poolFile)
	if err != nil {
		t.Fatalf("Failed to read pool file: %v", err)
	}

	contentStr := string(content)

	// Check default values (updated for better Magento performance)
	checks := []string{
		"pm.max_children = 25",
		"pm.start_servers = 4",
		"pm.min_spare_servers = 2",
		"pm.max_spare_servers = 6",
		"pm.max_requests = 1000",
		"memory_limit] = 768M",
	}

	for _, check := range checks {
		if !strings.Contains(contentStr, check) {
			t.Errorf("Pool content should contain default %q", check)
		}
	}
}

func TestNewFPMController(t *testing.T) {
	p := &platform.Platform{Type: platform.Linux}
	c := NewFPMController(p, "8.2")

	if c == nil {
		t.Fatal("NewFPMController should not return nil")
	}
	if c.version != "8.2" {
		t.Errorf("version = %v, want 8.2", c.version)
	}
}

func TestGetCurrentUser(t *testing.T) {
	user := getCurrentUser()
	if user == "" {
		t.Error("getCurrentUser should not return empty string")
	}
}

func TestGetCurrentGroup(t *testing.T) {
	group := getCurrentGroup()
	if group == "" {
		t.Error("getCurrentGroup should not return empty string")
	}
}

func TestPoolTemplateValidity(t *testing.T) {
	// Test that the embedded template parses correctly
	tmpl, err := template.New("pool").Parse(poolTemplateEmbed)
	if err != nil {
		t.Fatalf("Pool template parsing failed: %v", err)
	}

	if tmpl == nil {
		t.Error("Parsed template should not be nil")
	}

	// Verify template contains expected sections
	templateStr := poolTemplateEmbed
	expectedSections := []string{
		"[{{.ProjectName}}]",
		"listen = {{.SocketPath}}",
		"pm = dynamic",
		"php_admin_value[error_log] = {{.LogPath}}",
		"{{range $key, $value := .PHPINI}}",
		"{{range $key, $value := .Env}}",
	}

	for _, section := range expectedSections {
		if !strings.Contains(templateStr, section) {
			t.Errorf("Template should contain section: %s", section)
		}
	}
}

func TestRenderPool_WithEnv(t *testing.T) {
	g, _ := setupTestPoolGenerator(t)

	cfg := PoolConfig{
		ProjectName:     "mystore",
		PHPVersion:      "8.2",
		SocketPath:      "/tmp/mystore.sock",
		User:            "testuser",
		Group:           "testgroup",
		MaxChildren:     10,
		StartServers:    2,
		MinSpareServers: 1,
		MaxSpareServers: 3,
		MaxRequests:     500,
		Env: map[string]string{
			"MAGE_MODE":  "developer",
			"REDIS_HOST": "localhost",
		},
		PHPINI: map[string]string{},
	}

	content, err := g.renderPool(cfg)
	if err != nil {
		t.Fatalf("renderPool failed: %v", err)
	}

	// Check env vars are rendered
	if !strings.Contains(content, "env[MAGE_MODE] = developer") {
		t.Error("Pool should contain MAGE_MODE env var")
	}
	if !strings.Contains(content, "env[REDIS_HOST] = localhost") {
		t.Error("Pool should contain REDIS_HOST env var")
	}
}

func TestRenderPool_WithPHPINI(t *testing.T) {
	g, _ := setupTestPoolGenerator(t)

	cfg := PoolConfig{
		ProjectName:     "mystore",
		PHPVersion:      "8.2",
		SocketPath:      "/tmp/mystore.sock",
		User:            "testuser",
		Group:           "testgroup",
		MaxChildren:     10,
		StartServers:    2,
		MinSpareServers: 1,
		MaxSpareServers: 3,
		MaxRequests:     500,
		Env:             map[string]string{},
		PHPINI: map[string]string{
			"opcache.enable":     "0",
			"display_errors":     "On",
			"xdebug.mode":        "debug",
			"max_execution_time": "3600",
		},
	}

	content, err := g.renderPool(cfg)
	if err != nil {
		t.Fatalf("renderPool failed: %v", err)
	}

	// Check PHP INI directives are rendered
	checks := []string{
		"php_admin_value[opcache.enable] = 0",
		"php_admin_value[display_errors] = On",
		"php_admin_value[xdebug.mode] = debug",
		"php_admin_value[max_execution_time] = 3600",
	}

	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("Pool should contain PHP INI directive: %q", check)
		}
	}
}

func TestGenerate_WithPHPINI(t *testing.T) {
	g, _ := setupTestPoolGenerator(t)

	phpIni := map[string]string{
		"opcache.enable": "0",
		"display_errors": "On",
	}

	err := g.Generate("testproject", "/tmp/testproject", "8.2", nil, phpIni, false)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Read the generated file
	poolFile := filepath.Join(g.PoolsDirForVersion("8.2"), "testproject.conf")
	content, err := os.ReadFile(poolFile)
	if err != nil {
		t.Fatalf("Failed to read pool file: %v", err)
	}

	contentStr := string(content)

	// Verify PHP INI overrides are present
	if !strings.Contains(contentStr, "php_admin_value[opcache.enable] = 0") {
		t.Error("Pool should contain opcache.enable override")
	}
	if !strings.Contains(contentStr, "php_admin_value[display_errors] = On") {
		t.Error("Pool should contain display_errors override")
	}
}

func TestGenerate_WithMailpit(t *testing.T) {
	g, tmpDir := setupTestPoolGenerator(t)

	err := g.Generate("testproject", "/tmp/testproject", "8.2", nil, nil, true)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Read the generated file
	poolFile := filepath.Join(g.PoolsDirForVersion("8.2"), "testproject.conf")
	content, err := os.ReadFile(poolFile)
	if err != nil {
		t.Fatalf("Failed to read pool file: %v", err)
	}

	contentStr := string(content)

	// Verify Mailpit configuration is present
	if !strings.Contains(contentStr, "sendmail_path") {
		t.Error("Pool should contain sendmail_path when Mailpit is enabled")
	}
	if !strings.Contains(contentStr, "smtp_port] = 1025") {
		t.Error("Pool should contain smtp_port = 1025 when Mailpit is enabled")
	}
	if !strings.Contains(contentStr, "env[MAILPIT_HOST]") {
		t.Error("Pool should contain MAILPIT_HOST env var when Mailpit is enabled")
	}
	if !strings.Contains(contentStr, "env[MAILPIT_PORT]") {
		t.Error("Pool should contain MAILPIT_PORT env var when Mailpit is enabled")
	}

	// Verify mailpit-sendmail script was created
	sendmailPath := filepath.Join(tmpDir, ".magebox", "bin", "mailpit-sendmail")
	if _, err := os.Stat(sendmailPath); os.IsNotExist(err) {
		t.Error("mailpit-sendmail script should have been created")
	}
}

func TestMailpitConstants(t *testing.T) {
	if MailpitSMTPHost != "127.0.0.1" {
		t.Errorf("MailpitSMTPHost = %v, want 127.0.0.1", MailpitSMTPHost)
	}
	if MailpitSMTPPort != 1025 {
		t.Errorf("MailpitSMTPPort = %v, want 1025", MailpitSMTPPort)
	}
	if MailpitWebHost != "127.0.0.1" {
		t.Errorf("MailpitWebHost = %v, want 127.0.0.1", MailpitWebHost)
	}
	if MailpitWebPort != 8025 {
		t.Errorf("MailpitWebPort = %v, want 8025", MailpitWebPort)
	}
}

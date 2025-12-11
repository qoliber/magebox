package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qoliber/magebox/internal/config"
	"github.com/qoliber/magebox/internal/platform"
)

func setupTestManager(t *testing.T) (*Manager, string) {
	tmpDir := t.TempDir()
	p := &platform.Platform{
		Type:    platform.Linux,
		HomeDir: tmpDir,
	}
	m := NewManager(p)
	return m, tmpDir
}

func TestNewManager(t *testing.T) {
	m, _ := setupTestManager(t)

	if m == nil {
		t.Fatal("NewManager should not return nil")
	}
	if m.platform == nil {
		t.Error("platform should not be nil")
	}
	if m.sslManager == nil {
		t.Error("sslManager should not be nil")
	}
	if m.vhostGenerator == nil {
		t.Error("vhostGenerator should not be nil")
	}
	if m.poolGenerator == nil {
		t.Error("poolGenerator should not be nil")
	}
	if m.composeGen == nil {
		t.Error("composeGen should not be nil")
	}
	if m.hostsManager == nil {
		t.Error("hostsManager should not be nil")
	}
	if m.phpDetector == nil {
		t.Error("phpDetector should not be nil")
	}
}

func TestManager_Init(t *testing.T) {
	m, tmpDir := setupTestManager(t)

	projectPath := filepath.Join(tmpDir, "myproject")
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	err := m.Init(projectPath, "mystore")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Check that .magebox.yaml file was created
	configPath := filepath.Join(projectPath, config.ConfigFileName)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("%s file should have been created", config.ConfigFileName)
	}

	// Read and verify content
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	contentStr := string(content)
	checks := []string{
		"name: mystore",
		"host: mystore.test",
		"php:",
		"mysql:",
		"redis:",
	}

	for _, check := range checks {
		if !strings.Contains(contentStr, check) {
			t.Errorf("Config should contain %q", check)
		}
	}
}

func TestManager_InitAlreadyExists(t *testing.T) {
	m, tmpDir := setupTestManager(t)

	projectPath := filepath.Join(tmpDir, "myproject")
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	// Create first time
	if err := m.Init(projectPath, "mystore"); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Try to create again - should fail
	err := m.Init(projectPath, "mystore")
	if err == nil {
		t.Errorf("Init should fail when %s already exists", config.ConfigFileName)
	}
}

func TestManager_ValidateConfig(t *testing.T) {
	m, tmpDir := setupTestManager(t)

	projectPath := filepath.Join(tmpDir, "myproject")
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	// Create a valid .magebox file
	configContent := `name: mystore
domains:
  - host: mystore.test
php: "8.2"
services:
  mysql: "8.0"
`
	if err := os.WriteFile(filepath.Join(projectPath, config.ConfigFileName), []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, warnings, err := m.ValidateConfig(projectPath)
	if err != nil {
		t.Fatalf("ValidateConfig failed: %v", err)
	}

	if cfg == nil {
		t.Fatal("Config should not be nil")
	}
	if cfg.Name != "mystore" {
		t.Errorf("Name = %v, want mystore", cfg.Name)
	}

	// Should have warning about PHP not being installed (in test environment)
	hasPhpWarning := false
	for _, w := range warnings {
		if strings.Contains(w, "PHP") && strings.Contains(w, "not installed") {
			hasPhpWarning = true
			break
		}
	}
	if !hasPhpWarning {
		t.Log("Expected PHP not installed warning (may pass if PHP is installed)")
	}
}

func TestManager_ValidateConfigMissingFile(t *testing.T) {
	m, tmpDir := setupTestManager(t)

	projectPath := filepath.Join(tmpDir, "nonexistent")

	_, _, err := m.ValidateConfig(projectPath)
	if err == nil {
		t.Errorf("ValidateConfig should fail for missing %s file", config.ConfigFileName)
	}
}

func TestStartResult(t *testing.T) {
	result := &StartResult{
		ProjectPath: "/path/to/project",
		PHPVersion:  "8.2",
		Domains:     []string{"mystore.test"},
		Services:    []string{"PHP-FPM 8.2", "Nginx", "MySQL 8.0"},
		Errors:      make([]error, 0),
		Warnings:    make([]string, 0),
	}

	if result.ProjectPath != "/path/to/project" {
		t.Error("ProjectPath should be set")
	}
	if len(result.Services) != 3 {
		t.Errorf("Services count = %d, want 3", len(result.Services))
	}
}

func TestProjectStatus(t *testing.T) {
	status := &ProjectStatus{
		Name:       "mystore",
		Path:       "/path/to/project",
		PHPVersion: "8.2",
		Domains:    []string{"mystore.test", "api.mystore.test"},
		Services: map[string]ServiceStatus{
			"nginx": {Name: "Nginx", IsRunning: true},
			"mysql": {Name: "MySQL 8.0", IsRunning: true, Port: 33080},
		},
	}

	if status.Name != "mystore" {
		t.Error("Name should be set")
	}
	if len(status.Domains) != 2 {
		t.Errorf("Domains count = %d, want 2", len(status.Domains))
	}
	if !status.Services["nginx"].IsRunning {
		t.Error("Nginx should be running")
	}
}

func TestServiceStatus(t *testing.T) {
	status := ServiceStatus{
		Name:      "MySQL 8.0",
		IsRunning: true,
		Port:      33080,
	}

	if status.Name != "MySQL 8.0" {
		t.Error("Name should be set")
	}
	if !status.IsRunning {
		t.Error("IsRunning should be true")
	}
	if status.Port != 33080 {
		t.Errorf("Port = %d, want 33080", status.Port)
	}
}

func TestPHPNotInstalledError(t *testing.T) {
	p := &platform.Platform{Type: platform.Linux}
	err := &PHPNotInstalledError{
		Version:  "8.3",
		Platform: p,
	}

	msg := err.Error()

	if !strings.Contains(msg, "PHP 8.3") {
		t.Error("Error should mention PHP version")
	}
	if !strings.Contains(msg, "not found") {
		t.Error("Error should mention not found")
	}
	if !strings.Contains(msg, "Install") {
		t.Error("Error should contain install instructions")
	}
}

func TestManager_getStartedServices(t *testing.T) {
	m, tmpDir := setupTestManager(t)

	projectPath := filepath.Join(tmpDir, "myproject")
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	// Create a .magebox file
	configContent := `name: mystore
domains:
  - host: mystore.test
php: "8.2"
services:
  mysql: "8.0"
  redis: true
  opensearch: "2.12"
`
	if err := os.WriteFile(filepath.Join(projectPath, config.ConfigFileName), []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, _, err := m.ValidateConfig(projectPath)
	if err != nil {
		t.Fatalf("ValidateConfig failed: %v", err)
	}

	services := m.getStartedServices(cfg)

	// Should include PHP-FPM, Nginx, MySQL, Redis, OpenSearch
	expectedServices := []string{"PHP-FPM", "Nginx", "MySQL", "Redis", "OpenSearch"}
	for _, expected := range expectedServices {
		found := false
		for _, svc := range services {
			if strings.Contains(svc, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Services should include %s", expected)
		}
	}
}

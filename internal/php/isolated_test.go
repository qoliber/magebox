package php

import (
	"os"
	"path/filepath"
	"testing"

	"qoliber/magebox/internal/platform"
)

func setupTestIsolatedController(t *testing.T) (*IsolatedFPMController, string) {
	tmpDir := t.TempDir()
	p := &platform.Platform{
		Type:    platform.Linux,
		HomeDir: tmpDir,
	}
	c := NewIsolatedFPMController(p)
	return c, tmpDir
}

func TestNewIsolatedFPMController(t *testing.T) {
	c, _ := setupTestIsolatedController(t)

	if c == nil {
		t.Fatal("NewIsolatedFPMController should not return nil")
	}

	if c.registry == nil {
		t.Error("registry should not be nil")
	}
}

func TestIsolatedRegistry_LoadEmpty(t *testing.T) {
	c, _ := setupTestIsolatedController(t)

	projects, err := c.registry.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(projects) != 0 {
		t.Errorf("Expected 0 projects, got %d", len(projects))
	}
}

func TestIsolatedRegistry_AddAndGet(t *testing.T) {
	c, tmpDir := setupTestIsolatedController(t)

	// Ensure magebox directory exists
	if err := os.MkdirAll(filepath.Join(tmpDir, ".magebox"), 0755); err != nil {
		t.Fatalf("Failed to create magebox dir: %v", err)
	}

	project := &IsolatedProject{
		ProjectName: "testproject",
		ProjectPath: "/tmp/testproject",
		PHPVersion:  "8.3",
		SocketPath:  "/tmp/test.sock",
		PIDPath:     "/tmp/test.pid",
		ConfigPath:  "/tmp/test.conf",
		Settings: map[string]string{
			"opcache.enable": "0",
		},
	}

	if err := c.registry.Add(project); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Get it back
	retrieved, err := c.registry.Get("testproject")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Retrieved project should not be nil")
	}

	if retrieved.ProjectName != "testproject" {
		t.Errorf("ProjectName = %v, want testproject", retrieved.ProjectName)
	}

	if retrieved.PHPVersion != "8.3" {
		t.Errorf("PHPVersion = %v, want 8.3", retrieved.PHPVersion)
	}
}

func TestIsolatedRegistry_Remove(t *testing.T) {
	c, tmpDir := setupTestIsolatedController(t)

	// Ensure magebox directory exists
	if err := os.MkdirAll(filepath.Join(tmpDir, ".magebox"), 0755); err != nil {
		t.Fatalf("Failed to create magebox dir: %v", err)
	}

	project := &IsolatedProject{
		ProjectName: "testproject",
		ProjectPath: "/tmp/testproject",
		PHPVersion:  "8.3",
	}

	if err := c.registry.Add(project); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Remove it
	if err := c.registry.Remove("testproject"); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	// Verify it's gone
	retrieved, err := c.registry.Get("testproject")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved != nil {
		t.Error("Retrieved project should be nil after removal")
	}
}

func TestIsolatedRegistry_List(t *testing.T) {
	c, tmpDir := setupTestIsolatedController(t)

	// Ensure magebox directory exists
	if err := os.MkdirAll(filepath.Join(tmpDir, ".magebox"), 0755); err != nil {
		t.Fatalf("Failed to create magebox dir: %v", err)
	}

	// Add multiple projects
	projects := []*IsolatedProject{
		{ProjectName: "project1", PHPVersion: "8.2"},
		{ProjectName: "project2", PHPVersion: "8.3"},
		{ProjectName: "project3", PHPVersion: "8.3"},
	}

	for _, p := range projects {
		if err := c.registry.Add(p); err != nil {
			t.Fatalf("Add failed: %v", err)
		}
	}

	// List them
	list, err := c.registry.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 3 {
		t.Errorf("Expected 3 projects, got %d", len(list))
	}
}

func TestIsolatedFPMController_GetSocketPath_NotIsolated(t *testing.T) {
	c, tmpDir := setupTestIsolatedController(t)

	// Ensure magebox directory exists
	if err := os.MkdirAll(filepath.Join(tmpDir, ".magebox"), 0755); err != nil {
		t.Fatalf("Failed to create magebox dir: %v", err)
	}

	// Not isolated project should return shared socket path
	socketPath := c.GetSocketPath("myproject", "8.3")
	expected := filepath.Join(tmpDir, ".magebox", "run", "myproject-php8.3.sock")

	if socketPath != expected {
		t.Errorf("GetSocketPath = %v, want %v", socketPath, expected)
	}
}

func TestIsolatedFPMController_GetSocketPath_Isolated(t *testing.T) {
	c, tmpDir := setupTestIsolatedController(t)

	// Ensure magebox directory exists
	if err := os.MkdirAll(filepath.Join(tmpDir, ".magebox"), 0755); err != nil {
		t.Fatalf("Failed to create magebox dir: %v", err)
	}

	// Register an isolated project
	project := &IsolatedProject{
		ProjectName: "myproject",
		PHPVersion:  "8.3",
		SocketPath:  filepath.Join(tmpDir, ".magebox", "run", "myproject-isolated-php8.3.sock"),
	}
	if err := c.registry.Add(project); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Should return isolated socket path
	socketPath := c.GetSocketPath("myproject", "8.3")
	expected := filepath.Join(tmpDir, ".magebox", "run", "myproject-isolated-php8.3.sock")

	if socketPath != expected {
		t.Errorf("GetSocketPath = %v, want %v", socketPath, expected)
	}
}

func TestIsolatedFPMController_IsIsolated(t *testing.T) {
	c, tmpDir := setupTestIsolatedController(t)

	// Ensure magebox directory exists
	if err := os.MkdirAll(filepath.Join(tmpDir, ".magebox"), 0755); err != nil {
		t.Fatalf("Failed to create magebox dir: %v", err)
	}

	// Not isolated
	if c.IsIsolated("nonexistent") {
		t.Error("IsIsolated should return false for non-existent project")
	}

	// Add isolated project
	project := &IsolatedProject{ProjectName: "myproject"}
	if err := c.registry.Add(project); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Now it should be isolated
	if !c.IsIsolated("myproject") {
		t.Error("IsIsolated should return true for registered project")
	}
}

func TestIsolatedFPMConfig_Template(t *testing.T) {
	// Test that the embedded template parses correctly
	if isolatedFPMTemplateEmbed == "" {
		t.Error("isolatedFPMTemplateEmbed should not be empty")
	}

	// Verify template contains expected sections
	expectedSections := []string{
		"[global]",
		"pid = {{.PIDPath}}",
		"error_log = {{.ErrorLogPath}}",
		"[{{.ProjectName}}]",
		"listen = {{.SocketPath}}",
		"pm = dynamic",
		"{{range $key, $value := .SystemSettings}}",
		"{{range $key, $value := .PoolSettings}}",
	}

	for _, section := range expectedSections {
		if !containsString(isolatedFPMTemplateEmbed, section) {
			t.Errorf("Template should contain section: %s", section)
		}
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package lib

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewPaths(t *testing.T) {
	homeDir := "/home/testuser"
	paths := NewPaths(homeDir)

	if paths.MageBoxDir != "/home/testuser/.magebox" {
		t.Errorf("Expected MageBoxDir to be /home/testuser/.magebox, got %s", paths.MageBoxDir)
	}

	if paths.LibDir != "/home/testuser/.magebox/yaml" {
		t.Errorf("Expected LibDir to be /home/testuser/.magebox/yaml, got %s", paths.LibDir)
	}

	if paths.TemplatesDir != "/home/testuser/.magebox/yaml/templates" {
		t.Errorf("Expected TemplatesDir to be /home/testuser/.magebox/yaml/templates, got %s", paths.TemplatesDir)
	}

	if paths.InstallersDir != "/home/testuser/.magebox/yaml/installers" {
		t.Errorf("Expected InstallersDir to be /home/testuser/.magebox/yaml/installers, got %s", paths.InstallersDir)
	}
}

func TestNewPathsWithCustomLib(t *testing.T) {
	homeDir := "/home/testuser"
	customLib := "/custom/lib/path"
	paths := NewPathsWithCustomLib(homeDir, customLib)

	if paths.MageBoxDir != "/home/testuser/.magebox" {
		t.Errorf("Expected MageBoxDir to be /home/testuser/.magebox, got %s", paths.MageBoxDir)
	}

	if paths.LibDir != "/custom/lib/path" {
		t.Errorf("Expected LibDir to be /custom/lib/path, got %s", paths.LibDir)
	}

	if paths.TemplatesDir != "/custom/lib/path/templates" {
		t.Errorf("Expected TemplatesDir to be /custom/lib/path/templates, got %s", paths.TemplatesDir)
	}

	if paths.InstallersDir != "/custom/lib/path/installers" {
		t.Errorf("Expected InstallersDir to be /custom/lib/path/installers, got %s", paths.InstallersDir)
	}

	// Local overrides should still be in the standard location
	if paths.LocalTemplatesDir != "/home/testuser/.magebox/yaml-local/templates" {
		t.Errorf("Expected LocalTemplatesDir to be /home/testuser/.magebox/yaml-local/templates, got %s", paths.LocalTemplatesDir)
	}
}

func TestTemplatePath_LocalOverride(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a custom lib structure
	libDir := filepath.Join(tmpDir, "lib")
	localDir := filepath.Join(tmpDir, "lib-local")

	// Create directories
	os.MkdirAll(filepath.Join(libDir, "templates", "nginx"), 0755)
	os.MkdirAll(filepath.Join(localDir, "templates", "nginx"), 0755)

	// Create a template in lib
	libTemplate := filepath.Join(libDir, "templates", "nginx", "vhost.conf.tmpl")
	os.WriteFile(libTemplate, []byte("lib template"), 0644)

	// Create paths manually
	paths := &Paths{
		MageBoxDir:        tmpDir,
		LibDir:            libDir,
		LocalDir:          localDir,
		TemplatesDir:      filepath.Join(libDir, "templates"),
		LocalTemplatesDir: filepath.Join(localDir, "templates"),
	}

	// Without local override, should return lib path
	result := paths.TemplatePath("nginx", "vhost.conf.tmpl")
	if result != libTemplate {
		t.Errorf("Expected %s, got %s", libTemplate, result)
	}

	// Create local override
	localTemplate := filepath.Join(localDir, "templates", "nginx", "vhost.conf.tmpl")
	os.WriteFile(localTemplate, []byte("local override"), 0644)

	// With local override, should return local path
	result = paths.TemplatePath("nginx", "vhost.conf.tmpl")
	if result != localTemplate {
		t.Errorf("Expected local override %s, got %s", localTemplate, result)
	}
}

func TestTemplateLoader_LoadFromLib(t *testing.T) {
	tmpDir := t.TempDir()

	// Create lib structure
	libDir := filepath.Join(tmpDir, "lib")
	os.MkdirAll(filepath.Join(libDir, "templates", "nginx"), 0755)

	// Create a template
	templateContent := "server { listen 80; }"
	templatePath := filepath.Join(libDir, "templates", "nginx", "vhost.conf.tmpl")
	os.WriteFile(templatePath, []byte(templateContent), 0644)

	// Create paths
	paths := &Paths{
		MageBoxDir:        tmpDir,
		LibDir:            libDir,
		LocalDir:          filepath.Join(tmpDir, "lib-local"),
		TemplatesDir:      filepath.Join(libDir, "templates"),
		LocalTemplatesDir: filepath.Join(tmpDir, "lib-local", "templates"),
	}

	loader := NewTemplateLoader(paths)

	// Load template
	content, err := loader.LoadTemplate("nginx", "vhost.conf.tmpl")
	if err != nil {
		t.Fatalf("Failed to load template: %v", err)
	}

	if content != templateContent {
		t.Errorf("Expected '%s', got '%s'", templateContent, content)
	}
}

func TestTemplateLoader_FallbackToEmbedded(t *testing.T) {
	tmpDir := t.TempDir()

	// Create paths with non-existent lib
	paths := &Paths{
		MageBoxDir:        tmpDir,
		LibDir:            filepath.Join(tmpDir, "nonexistent"),
		LocalDir:          filepath.Join(tmpDir, "lib-local"),
		TemplatesDir:      filepath.Join(tmpDir, "nonexistent", "templates"),
		LocalTemplatesDir: filepath.Join(tmpDir, "lib-local", "templates"),
	}

	loader := NewTemplateLoader(paths)

	// Register a fallback
	fallbackContent := "embedded fallback content"
	loader.RegisterFallback("nginx", "vhost.conf.tmpl", fallbackContent)

	// Load template - should use fallback since lib doesn't exist
	content, err := loader.LoadTemplate("nginx", "vhost.conf.tmpl")
	if err != nil {
		t.Fatalf("Failed to load template: %v", err)
	}

	if content != fallbackContent {
		t.Errorf("Expected fallback '%s', got '%s'", fallbackContent, content)
	}
}

func TestTemplateLoader_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	paths := &Paths{
		MageBoxDir:        tmpDir,
		LibDir:            filepath.Join(tmpDir, "nonexistent"),
		LocalDir:          filepath.Join(tmpDir, "lib-local"),
		TemplatesDir:      filepath.Join(tmpDir, "nonexistent", "templates"),
		LocalTemplatesDir: filepath.Join(tmpDir, "lib-local", "templates"),
	}

	loader := NewTemplateLoader(paths)

	// Try to load a template that doesn't exist and has no fallback
	_, err := loader.LoadTemplate("nginx", "nonexistent.tmpl")
	if err == nil {
		t.Error("Expected error for non-existent template")
	}
}

func TestPaths_Exists(t *testing.T) {
	tmpDir := t.TempDir()

	// Create lib directory
	libDir := filepath.Join(tmpDir, "lib")
	os.MkdirAll(libDir, 0755)

	paths := &Paths{
		LibDir: libDir,
	}

	if !paths.Exists() {
		t.Error("Expected Exists() to return true for existing lib directory")
	}

	// Test with non-existent directory
	paths.LibDir = filepath.Join(tmpDir, "nonexistent")
	if paths.Exists() {
		t.Error("Expected Exists() to return false for non-existent lib directory")
	}
}

func TestPaths_IsGitRepo(t *testing.T) {
	tmpDir := t.TempDir()

	// Create lib directory with .git
	libDir := filepath.Join(tmpDir, "lib")
	os.MkdirAll(filepath.Join(libDir, ".git"), 0755)

	paths := &Paths{
		LibDir: libDir,
	}

	if !paths.IsGitRepo() {
		t.Error("Expected IsGitRepo() to return true for directory with .git")
	}

	// Test without .git
	libDir2 := filepath.Join(tmpDir, "lib2")
	os.MkdirAll(libDir2, 0755)
	paths.LibDir = libDir2

	if paths.IsGitRepo() {
		t.Error("Expected IsGitRepo() to return false for directory without .git")
	}
}

func TestGetCustomLibPath(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .magebox directory
	mageboxDir := filepath.Join(tmpDir, ".magebox")
	os.MkdirAll(mageboxDir, 0755)

	// Test without config file
	result := getCustomLibPath(tmpDir)
	if result != "" {
		t.Errorf("Expected empty string without config, got %s", result)
	}

	// Create config with lib_path
	configContent := "lib_path: /custom/lib/path\n"
	configPath := filepath.Join(mageboxDir, "config.yaml")
	os.WriteFile(configPath, []byte(configContent), 0644)

	result = getCustomLibPath(tmpDir)
	if result != "/custom/lib/path" {
		t.Errorf("Expected /custom/lib/path, got %s", result)
	}

	// Test with empty lib_path
	configContent = "dns_mode: dnsmasq\n"
	os.WriteFile(configPath, []byte(configContent), 0644)

	result = getCustomLibPath(tmpDir)
	if result != "" {
		t.Errorf("Expected empty string for empty lib_path, got %s", result)
	}
}

func TestGlobalLoaderFallback(t *testing.T) {
	// Test the global loader fallback registration
	loader := GetGlobalLoader()
	if loader == nil {
		t.Fatal("GetGlobalLoader returned nil")
	}

	// Register a test fallback
	testContent := "test template content"
	RegisterFallbackTemplate("test", "test.tmpl", testContent)

	// Load the template
	content, err := GetTemplate("test", "test.tmpl")
	if err != nil {
		t.Fatalf("Failed to get template: %v", err)
	}

	if content != testContent {
		t.Errorf("Expected '%s', got '%s'", testContent, content)
	}
}

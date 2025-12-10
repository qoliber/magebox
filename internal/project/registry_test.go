package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/qoliber/magebox/internal/platform"
)

func TestNewProjectDiscovery(t *testing.T) {
	p := &platform.Platform{
		Type:    platform.Linux,
		HomeDir: "/home/test",
	}

	d := NewProjectDiscovery(p)
	if d == nil {
		t.Fatal("NewProjectDiscovery should not return nil")
	}
}

func TestProjectDiscovery_DiscoverProjects_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	p := &platform.Platform{
		Type:    platform.Linux,
		HomeDir: tmpDir,
	}

	d := NewProjectDiscovery(p)
	projects, err := d.DiscoverProjects()

	if err != nil {
		t.Fatalf("DiscoverProjects failed: %v", err)
	}

	if len(projects) != 0 {
		t.Errorf("Expected 0 projects, got %d", len(projects))
	}
}

func TestProjectDiscovery_DiscoverProjects_WithVhosts(t *testing.T) {
	tmpDir := t.TempDir()

	// Create vhosts directory
	vhostsDir := filepath.Join(tmpDir, ".magebox", "nginx", "vhosts")
	os.MkdirAll(vhostsDir, 0755)

	// Create a project directory with .magebox file
	projectDir := filepath.Join(tmpDir, "myproject")
	os.MkdirAll(filepath.Join(projectDir, "pub"), 0755)
	os.WriteFile(filepath.Join(projectDir, ".magebox"), []byte(`
name: myproject
domains:
  - host: myproject.test
php: "8.2"
services:
  mysql: "8.0"
`), 0644)

	// Create vhost file
	vhostContent := `server {
    listen 80;
    server_name myproject.test;
    root ` + projectDir + `/pub;
}
`
	os.WriteFile(filepath.Join(vhostsDir, "myproject.conf"), []byte(vhostContent), 0644)

	p := &platform.Platform{
		Type:    platform.Linux,
		HomeDir: tmpDir,
	}

	d := NewProjectDiscovery(p)
	projects, err := d.DiscoverProjects()

	if err != nil {
		t.Fatalf("DiscoverProjects failed: %v", err)
	}

	if len(projects) != 1 {
		t.Fatalf("Expected 1 project, got %d", len(projects))
	}

	project := projects[0]
	if project.Name != "myproject" {
		t.Errorf("Name = %v, want myproject", project.Name)
	}
	if project.Path != projectDir {
		t.Errorf("Path = %v, want %v", project.Path, projectDir)
	}
	if !project.HasConfig {
		t.Error("HasConfig should be true")
	}
	if project.PHPVersion != "8.2" {
		t.Errorf("PHPVersion = %v, want 8.2", project.PHPVersion)
	}
}

func TestProjectDiscovery_CountProjects(t *testing.T) {
	tmpDir := t.TempDir()
	p := &platform.Platform{
		Type:    platform.Linux,
		HomeDir: tmpDir,
	}

	d := NewProjectDiscovery(p)
	count := d.CountProjects()

	if count != 0 {
		t.Errorf("CountProjects = %d, want 0", count)
	}
}

func TestProjectInfo(t *testing.T) {
	info := ProjectInfo{
		Name:       "test-project",
		Path:       "/var/www/test",
		Domains:    []string{"test.test", "api.test.test"},
		PHPVersion: "8.3",
		ConfigFile: "/var/www/test/.magebox",
		HasConfig:  true,
	}

	if info.Name != "test-project" {
		t.Error("Name mismatch")
	}
	if len(info.Domains) != 2 {
		t.Error("Should have 2 domains")
	}
	if !info.HasConfig {
		t.Error("HasConfig should be true")
	}
}

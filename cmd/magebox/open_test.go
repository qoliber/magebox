package main

import (
	"os"
	"path/filepath"
	"testing"

	"qoliber/magebox/internal/config"
)

func TestSuffixHostBeforeTLD(t *testing.T) {
	tests := []struct {
		name   string
		host   string
		suffix string
		want   string
	}{
		{"localhost tld", "shop.localhost", "b2b-case", "shop.b2b-case.localhost"},
		{"nested subdomain", "store.magento.localhost", "qa", "store.magento.qa.localhost"},
		{"public tld", "www.example.com", "b2b-case", "www.example.b2b-case.com"},
		{"bare tld prepends", "localhost", "b2b-case", "b2b-case.localhost"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := suffixHostBeforeTLD(tt.host, tt.suffix); got != tt.want {
				t.Errorf("suffixHostBeforeTLD(%q, %q) = %q, want %q", tt.host, tt.suffix, got, tt.want)
			}
		})
	}
}

func TestWorktreeLocalConfig(t *testing.T) {
	src := []byte(`# project config
name: mystore
php: "8.3"
domains:
  - host: mystore.localhost
    ssl: true
  - host: admin.mystore.localhost
services:
  mysql: "8.0"
`)

	out, err := worktreeLocalConfig(src, "b2b-case")
	if err != nil {
		t.Fatalf("worktreeLocalConfig: %v", err)
	}

	// Round-trip through the real loader to validate the merged result.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, config.ConfigFileName), src, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, config.LocalConfigFileName), out, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.LoadFromPath(dir)
	if err != nil {
		t.Fatalf("LoadFromPath: %v", err)
	}

	if cfg.Name != "mystore.b2b-case" {
		t.Errorf("name = %q, want %q", cfg.Name, "mystore.b2b-case")
	}
	if len(cfg.Domains) != 2 {
		t.Fatalf("got %d domains, want 2", len(cfg.Domains))
	}
	if cfg.Domains[0].Host != "mystore.b2b-case.localhost" {
		t.Errorf("domain[0].Host = %q, want %q", cfg.Domains[0].Host, "mystore.b2b-case.localhost")
	}
	if cfg.Domains[1].Host != "admin.mystore.b2b-case.localhost" {
		t.Errorf("domain[1].Host = %q, want %q", cfg.Domains[1].Host, "admin.mystore.b2b-case.localhost")
	}
	// Untouched fields are preserved so the worktree is a complete, valid project.
	if cfg.PHP != "8.3" {
		t.Errorf("php = %q, want %q", cfg.PHP, "8.3")
	}
	if !cfg.Services.HasMySQL() {
		t.Error("expected MySQL service to be preserved")
	}
}

func TestPrepareWorktree(t *testing.T) {
	base := t.TempDir()
	worktree := filepath.Join(base, ".claude", "worktrees", "b2b-case")
	if err := os.MkdirAll(worktree, 0755); err != nil {
		t.Fatal(err)
	}
	src := []byte("name: mystore\nphp: \"8.3\"\ndomains:\n  - host: mystore.localhost\nservices:\n  mysql: \"8.0\"\n")
	if err := os.WriteFile(filepath.Join(worktree, config.ConfigFileName), src, 0644); err != nil {
		t.Fatal(err)
	}

	got, err := prepareWorktree(base, "b2b-case")
	if err != nil {
		t.Fatalf("prepareWorktree: %v", err)
	}
	if got != worktree {
		t.Errorf("worktree path = %q, want %q", got, worktree)
	}

	cfg, err := config.LoadFromPath(worktree)
	if err != nil {
		t.Fatalf("LoadFromPath: %v", err)
	}
	if cfg.Name != "mystore.b2b-case" {
		t.Errorf("name = %q, want %q", cfg.Name, "mystore.b2b-case")
	}
	if cfg.Domains[0].Host != "mystore.b2b-case.localhost" {
		t.Errorf("host = %q, want %q", cfg.Domains[0].Host, "mystore.b2b-case.localhost")
	}

	// Re-running is idempotent: the override is derived from .magebox.yaml each time.
	if _, err := prepareWorktree(base, "b2b-case"); err != nil {
		t.Fatalf("second prepareWorktree: %v", err)
	}
	cfg2, err := config.LoadFromPath(worktree)
	if err != nil {
		t.Fatalf("LoadFromPath after rerun: %v", err)
	}
	if cfg2.Name != "mystore.b2b-case" {
		t.Errorf("name after rerun = %q, want %q", cfg2.Name, "mystore.b2b-case")
	}

	// A missing worktree is reported as an error.
	if _, err := prepareWorktree(base, "does-not-exist"); err == nil {
		t.Error("expected error for missing worktree")
	}
}

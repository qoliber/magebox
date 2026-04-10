// Copyright (c) qoliber

package main

import (
	"os"
	"path/filepath"
	"testing"

	"qoliber/magebox/internal/config"
)

// TestWriteProjectTidewaysAPIKey_NewFile verifies that writing the key to a
// project that has no .magebox.local.yaml yet creates the file with a
// single php_ini entry.
func TestWriteProjectTidewaysAPIKey_NewFile(t *testing.T) {
	tmpDir := t.TempDir()

	if err := writeProjectTidewaysAPIKey(tmpDir, "abc123"); err != nil {
		t.Fatalf("writeProjectTidewaysAPIKey failed: %v", err)
	}

	local, err := config.LoadLocalConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadLocalConfig failed: %v", err)
	}
	if got := local.PHPINI["tideways.api_key"]; got != "abc123" {
		t.Errorf("tideways.api_key = %q, want abc123", got)
	}

	// The file should actually exist on disk.
	if _, err := os.Stat(filepath.Join(tmpDir, config.LocalConfigFileName)); err != nil {
		t.Errorf("%s was not created: %v", config.LocalConfigFileName, err)
	}
}

// TestWriteProjectTidewaysAPIKey_MergeExisting verifies that pre-existing
// php_ini entries in .magebox.local.yaml are preserved when we add the key.
// We must not clobber memory_limit, xdebug settings, or anything else a
// developer has set locally.
func TestWriteProjectTidewaysAPIKey_MergeExisting(t *testing.T) {
	tmpDir := t.TempDir()

	initial := &config.LocalConfig{
		PHPINI: map[string]string{
			"memory_limit":         "2G",
			"xdebug.mode":          "debug",
			"tideways.api_key":     "old-value",
			"tideways.sample_rate": "25",
		},
		Env: map[string]string{
			"APP_ENV": "dev",
		},
	}
	if err := config.SaveLocalConfig(tmpDir, initial); err != nil {
		t.Fatalf("SaveLocalConfig failed: %v", err)
	}

	if err := writeProjectTidewaysAPIKey(tmpDir, "new-value"); err != nil {
		t.Fatalf("writeProjectTidewaysAPIKey failed: %v", err)
	}

	local, err := config.LoadLocalConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadLocalConfig failed: %v", err)
	}

	if got := local.PHPINI["tideways.api_key"]; got != "new-value" {
		t.Errorf("tideways.api_key = %q, want new-value", got)
	}
	// Other entries must be preserved.
	if got := local.PHPINI["memory_limit"]; got != "2G" {
		t.Errorf("memory_limit = %q, want 2G (pre-existing entry dropped)", got)
	}
	if got := local.PHPINI["xdebug.mode"]; got != "debug" {
		t.Errorf("xdebug.mode = %q, want debug (pre-existing entry dropped)", got)
	}
	if got := local.PHPINI["tideways.sample_rate"]; got != "25" {
		t.Errorf("tideways.sample_rate = %q, want 25 (pre-existing entry dropped)", got)
	}
	if got := local.Env["APP_ENV"]; got != "dev" {
		t.Errorf("env APP_ENV = %q, want dev (env section dropped)", got)
	}
}

// TestWriteProjectTidewaysAPIKey_NilPHPINI covers the case where the file
// exists but has no php_ini section yet — the function must allocate the
// map before assigning into it (otherwise we panic on nil map write).
func TestWriteProjectTidewaysAPIKey_NilPHPINI(t *testing.T) {
	tmpDir := t.TempDir()

	initial := &config.LocalConfig{
		Env: map[string]string{"FOO": "bar"},
	}
	if err := config.SaveLocalConfig(tmpDir, initial); err != nil {
		t.Fatalf("SaveLocalConfig failed: %v", err)
	}

	if err := writeProjectTidewaysAPIKey(tmpDir, "abc"); err != nil {
		t.Fatalf("writeProjectTidewaysAPIKey failed: %v", err)
	}

	local, err := config.LoadLocalConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadLocalConfig failed: %v", err)
	}
	if got := local.PHPINI["tideways.api_key"]; got != "abc" {
		t.Errorf("tideways.api_key = %q, want abc", got)
	}
	if got := local.Env["FOO"]; got != "bar" {
		t.Errorf("env FOO = %q, want bar", got)
	}
}

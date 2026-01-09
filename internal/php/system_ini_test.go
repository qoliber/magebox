package php

import (
	"os"
	"path/filepath"
	"testing"

	"qoliber/magebox/internal/platform"
)

func TestIsSystemSetting(t *testing.T) {
	tests := []struct {
		setting  string
		isSystem bool
	}{
		// System settings
		{"opcache.preload", true},
		{"opcache.preload_user", true},
		{"opcache.jit", true},
		{"opcache.jit_buffer_size", true},
		{"opcache.memory_consumption", true},
		{"opcache.interned_strings_buffer", true},
		{"opcache.max_accelerated_files", true},
		{"opcache.huge_code_pages", true},
		{"opcache.enable_cli", true},

		// Pool settings (not system)
		{"opcache.enable", false},
		{"opcache.validate_timestamps", false},
		{"opcache.revalidate_freq", false},
		{"opcache.save_comments", false},
		{"memory_limit", false},
		{"max_execution_time", false},
		{"realpath_cache_size", false},
	}

	for _, tc := range tests {
		t.Run(tc.setting, func(t *testing.T) {
			result := IsSystemSetting(tc.setting)
			if result != tc.isSystem {
				t.Errorf("IsSystemSetting(%q) = %v, want %v", tc.setting, result, tc.isSystem)
			}
		})
	}
}

func TestSeparateSettings(t *testing.T) {
	settings := map[string]string{
		// System settings
		"opcache.preload":            "/path/to/preload.php",
		"opcache.preload_user":       "www-data",
		"opcache.jit":                "tracing",
		"opcache.jit_buffer_size":    "128M",
		"opcache.memory_consumption": "512",

		// Pool settings
		"opcache.enable":              "1",
		"opcache.validate_timestamps": "0",
		"memory_limit":                "768M",
		"max_execution_time":          "18000",
	}

	system, pool := SeparateSettings(settings)

	// Check system settings
	expectedSystem := map[string]string{
		"opcache.preload":            "/path/to/preload.php",
		"opcache.preload_user":       "www-data",
		"opcache.jit":                "tracing",
		"opcache.jit_buffer_size":    "128M",
		"opcache.memory_consumption": "512",
	}

	if len(system) != len(expectedSystem) {
		t.Errorf("system settings count = %d, want %d", len(system), len(expectedSystem))
	}

	for k, v := range expectedSystem {
		if system[k] != v {
			t.Errorf("system[%q] = %q, want %q", k, system[k], v)
		}
	}

	// Check pool settings
	expectedPool := map[string]string{
		"opcache.enable":              "1",
		"opcache.validate_timestamps": "0",
		"memory_limit":                "768M",
		"max_execution_time":          "18000",
	}

	if len(pool) != len(expectedPool) {
		t.Errorf("pool settings count = %d, want %d", len(pool), len(expectedPool))
	}

	for k, v := range expectedPool {
		if pool[k] != v {
			t.Errorf("pool[%q] = %q, want %q", k, pool[k], v)
		}
	}
}

func TestSystemINIManager(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "magebox-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a mock platform
	p := &platform.Platform{
		Type:    platform.Linux,
		HomeDir: tmpDir,
	}

	mgr := NewSystemINIManager(p)

	// Test with no settings
	owner, err := mgr.GetCurrentOwner("8.3")
	if err != nil {
		t.Errorf("GetCurrentOwner with no settings: %v", err)
	}
	if owner != nil {
		t.Errorf("Expected nil owner, got %+v", owner)
	}

	// Write settings for project A
	settings := map[string]string{
		"opcache.preload":         "/path/to/preload.php",
		"opcache.jit":             "tracing",
		"opcache.jit_buffer_size": "128M",
	}

	prevOwner, err := mgr.WriteSystemINI("8.3", "project-a", "/path/to/project-a", settings)
	if err != nil {
		t.Fatalf("WriteSystemINI failed: %v", err)
	}
	if prevOwner != nil {
		t.Errorf("Expected nil previous owner for first write, got %+v", prevOwner)
	}

	// Verify the INI file was created
	iniPath := mgr.GetSystemINIPath("8.3")
	if _, err := os.Stat(iniPath); os.IsNotExist(err) {
		t.Errorf("INI file not created at %s", iniPath)
	}

	// Verify owner
	owner, err = mgr.GetCurrentOwner("8.3")
	if err != nil {
		t.Fatalf("GetCurrentOwner failed: %v", err)
	}
	if owner == nil {
		t.Fatal("Expected owner, got nil")
	}
	if owner.ProjectName != "project-a" {
		t.Errorf("owner.ProjectName = %q, want %q", owner.ProjectName, "project-a")
	}
	if owner.ProjectPath != "/path/to/project-a" {
		t.Errorf("owner.ProjectPath = %q, want %q", owner.ProjectPath, "/path/to/project-a")
	}
	if len(owner.Settings) != 3 {
		t.Errorf("owner.Settings count = %d, want 3", len(owner.Settings))
	}

	// Write settings for project B (should return project A as previous owner)
	settingsB := map[string]string{
		"opcache.preload": "/path/to/other/preload.php",
		"opcache.jit":     "function",
	}

	prevOwner, err = mgr.WriteSystemINI("8.3", "project-b", "/path/to/project-b", settingsB)
	if err != nil {
		t.Fatalf("WriteSystemINI for project B failed: %v", err)
	}
	if prevOwner == nil {
		t.Error("Expected previous owner (project-a), got nil")
	} else if prevOwner.ProjectName != "project-a" {
		t.Errorf("prevOwner.ProjectName = %q, want %q", prevOwner.ProjectName, "project-a")
	}

	// Verify new owner
	owner, err = mgr.GetCurrentOwner("8.3")
	if err != nil {
		t.Fatalf("GetCurrentOwner failed: %v", err)
	}
	if owner.ProjectName != "project-b" {
		t.Errorf("owner.ProjectName = %q, want %q", owner.ProjectName, "project-b")
	}

	// Clear settings for project B
	err = mgr.ClearSystemINI("8.3", "project-b")
	if err != nil {
		t.Fatalf("ClearSystemINI failed: %v", err)
	}

	// Verify files are removed
	if _, err := os.Stat(iniPath); !os.IsNotExist(err) {
		t.Error("INI file should be removed after ClearSystemINI")
	}

	owner, err = mgr.GetCurrentOwner("8.3")
	if err != nil {
		t.Fatalf("GetCurrentOwner after clear failed: %v", err)
	}
	if owner != nil {
		t.Errorf("Expected nil owner after clear, got %+v", owner)
	}
}

func TestGetSystemSettingsList(t *testing.T) {
	list := GetSystemSettingsList()

	if len(list) == 0 {
		t.Error("GetSystemSettingsList returned empty list")
	}

	// Check that it's sorted
	for i := 1; i < len(list); i++ {
		if list[i] < list[i-1] {
			t.Errorf("List not sorted: %q comes after %q", list[i], list[i-1])
		}
	}

	// Check some expected settings are present
	expectedSettings := []string{
		"opcache.preload",
		"opcache.jit",
		"opcache.memory_consumption",
	}

	for _, expected := range expectedSettings {
		found := false
		for _, s := range list {
			if s == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected setting %q not found in list", expected)
		}
	}
}

func TestFormatOwnerWarning(t *testing.T) {
	previous := &SystemINIOwner{
		ProjectName: "old-project",
		ProjectPath: "/path/to/old",
		Settings: map[string]string{
			"opcache.jit":     "tracing",
			"opcache.preload": "/old/preload.php",
		},
	}

	newSettings := map[string]string{
		"opcache.jit":             "function",
		"opcache.preload":         "/new/preload.php",
		"opcache.jit_buffer_size": "256M",
	}

	warning := FormatOwnerWarning(previous, "new-project", newSettings)

	// Check that warning contains expected info
	if warning == "" {
		t.Error("FormatOwnerWarning returned empty string")
	}

	expectedStrings := []string{
		"old-project",
		"new-project",
		"/path/to/old",
		"opcache.jit",
		"tracing",
		"function",
	}

	for _, expected := range expectedStrings {
		if !contains(warning, expected) {
			t.Errorf("Warning should contain %q", expected)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestGetPHPScanDir(t *testing.T) {
	tests := []struct {
		name       string
		platform   platform.Type
		distro     platform.LinuxDistro
		phpVersion string
		expected   string
	}{
		{
			name:       "Fedora PHP 8.3",
			platform:   platform.Linux,
			distro:     platform.DistroFedora,
			phpVersion: "8.3",
			expected:   "/etc/opt/remi/php83/php.d",
		},
		{
			name:       "Debian PHP 8.2",
			platform:   platform.Linux,
			distro:     platform.DistroDebian,
			phpVersion: "8.2",
			expected:   "/etc/php/8.2/fpm/conf.d",
		},
		{
			name:       "Arch",
			platform:   platform.Linux,
			distro:     platform.DistroArch,
			phpVersion: "8.3",
			expected:   "/etc/php/conf.d",
		},
		{
			name:       "macOS",
			platform:   platform.Darwin,
			phpVersion: "8.3",
			expected:   "/opt/homebrew/etc/php/8.3/conf.d",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir, _ := os.MkdirTemp("", "test-*")
			defer os.RemoveAll(tmpDir)

			p := &platform.Platform{
				Type:        tc.platform,
				LinuxDistro: tc.distro,
				HomeDir:     tmpDir,
			}

			mgr := NewSystemINIManager(p)
			result := mgr.GetPHPScanDir(tc.phpVersion)

			if result != tc.expected {
				t.Errorf("GetPHPScanDir() = %q, want %q", result, tc.expected)
			}
		})
	}
}

func TestPoolGeneratorWithSystemSettings(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "magebox-pool-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create required directories
	poolsDir := filepath.Join(tmpDir, ".magebox", "php", "pools", "8.3")
	runDir := filepath.Join(tmpDir, ".magebox", "run")
	logsDir := filepath.Join(tmpDir, ".magebox", "logs", "php-fpm")
	os.MkdirAll(poolsDir, 0755)
	os.MkdirAll(runDir, 0755)
	os.MkdirAll(logsDir, 0755)

	// Create a mock platform
	p := &platform.Platform{
		Type:    platform.Linux,
		HomeDir: tmpDir,
	}

	gen := NewPoolGenerator(p)

	// Generate with mixed settings
	phpIni := map[string]string{
		// System settings
		"opcache.preload":            "/path/to/preload.php",
		"opcache.jit":                "tracing",
		"opcache.memory_consumption": "512",

		// Pool settings
		"opcache.validate_timestamps": "0",
		"memory_limit":                "768M",
	}

	result, err := gen.GenerateWithResult("test-project", "/path/to/project", "8.3", nil, phpIni, false)
	if err != nil {
		t.Fatalf("GenerateWithResult failed: %v", err)
	}

	// Check that system settings were separated
	// Note: defaults include opcache.memory_consumption, opcache.interned_strings_buffer, opcache.max_accelerated_files
	// which are also system settings, so we expect 5 total (3 from phpIni + 2 from defaults that are system)
	if len(result.SystemSettings) < 3 {
		t.Errorf("SystemSettings count = %d, want at least 3", len(result.SystemSettings))
	}

	// Verify our custom system settings are present
	if result.SystemSettings["opcache.preload"] != "/path/to/preload.php" {
		t.Errorf("SystemSettings missing opcache.preload")
	}
	if result.SystemSettings["opcache.jit"] != "tracing" {
		t.Errorf("SystemSettings missing opcache.jit")
	}

	// Pool settings should include defaults + custom pool settings (not system ones)
	if result.PoolSettings["opcache.validate_timestamps"] != "0" {
		t.Errorf("PoolSettings should contain opcache.validate_timestamps")
	}

	// System settings should not be in pool settings
	if _, exists := result.PoolSettings["opcache.preload"]; exists {
		t.Error("PoolSettings should not contain opcache.preload (system setting)")
	}

	// System INI path should be set
	if result.SystemINIPath == "" {
		t.Error("SystemINIPath should be set when system settings are present")
	}

	// Verify system INI file was created
	if _, err := os.Stat(result.SystemINIPath); os.IsNotExist(err) {
		t.Errorf("System INI file not created at %s", result.SystemINIPath)
	}

	// Verify pool file was created
	if result.PoolPath == "" {
		t.Error("PoolPath should be set")
	}
	if _, err := os.Stat(result.PoolPath); os.IsNotExist(err) {
		t.Errorf("Pool file not created at %s", result.PoolPath)
	}
}

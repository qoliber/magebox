package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGlobalConfigPath(t *testing.T) {
	homeDir := "/home/test"
	expected := "/home/test/.magebox/config.yaml"

	if got := GlobalConfigPath(homeDir); got != expected {
		t.Errorf("GlobalConfigPath() = %v, want %v", got, expected)
	}
}

func TestDefaultGlobalConfig(t *testing.T) {
	config := DefaultGlobalConfig()

	if config.DNSMode != "dnsmasq" {
		t.Errorf("DNSMode = %v, want dnsmasq", config.DNSMode)
	}
	if config.DefaultPHP != "8.2" {
		t.Errorf("DefaultPHP = %v, want 8.2", config.DefaultPHP)
	}
	if config.TLD != "test" {
		t.Errorf("TLD = %v, want test", config.TLD)
	}
	if config.Portainer {
		t.Error("Portainer should be false by default")
	}
	if config.PhpMyAdmin {
		t.Error("PhpMyAdmin should be false by default")
	}
	if !config.DefaultServices.Redis {
		t.Error("Redis should be true by default")
	}
	if config.DefaultServices.MySQL != "8.0" {
		t.Errorf("MySQL = %v, want 8.0", config.DefaultServices.MySQL)
	}
}

func TestLoadGlobalConfig_NotExists(t *testing.T) {
	tmpDir := t.TempDir()

	config, err := LoadGlobalConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadGlobalConfig failed: %v", err)
	}

	// Should return defaults
	if config.DNSMode != "dnsmasq" {
		t.Errorf("DNSMode = %v, want dnsmasq", config.DNSMode)
	}
}

func TestLoadGlobalConfig_Exists(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".magebox")
	os.MkdirAll(configDir, 0755)

	configContent := `
dns_mode: dnsmasq
default_php: "8.3"
tld: local
portainer: false
`
	os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configContent), 0644)

	config, err := LoadGlobalConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadGlobalConfig failed: %v", err)
	}

	if config.DNSMode != "dnsmasq" {
		t.Errorf("DNSMode = %v, want dnsmasq", config.DNSMode)
	}
	if config.DefaultPHP != "8.3" {
		t.Errorf("DefaultPHP = %v, want 8.3", config.DefaultPHP)
	}
	if config.TLD != "local" {
		t.Errorf("TLD = %v, want local", config.TLD)
	}
	if config.Portainer {
		t.Error("Portainer should be false")
	}
}

func TestSaveGlobalConfig(t *testing.T) {
	tmpDir := t.TempDir()

	config := &GlobalConfig{
		DNSMode:    "dnsmasq",
		DefaultPHP: "8.4",
		TLD:        "dev",
		Portainer:  true,
	}

	err := SaveGlobalConfig(tmpDir, config)
	if err != nil {
		t.Fatalf("SaveGlobalConfig failed: %v", err)
	}

	// Verify file exists
	configPath := GlobalConfigPath(tmpDir)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file should exist")
	}

	// Reload and verify
	loaded, err := LoadGlobalConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadGlobalConfig failed: %v", err)
	}

	if loaded.DNSMode != "dnsmasq" {
		t.Errorf("DNSMode = %v, want dnsmasq", loaded.DNSMode)
	}
	if loaded.DefaultPHP != "8.4" {
		t.Errorf("DefaultPHP = %v, want 8.4", loaded.DefaultPHP)
	}
}

func TestGlobalConfig_UseDnsmasq(t *testing.T) {
	config := &GlobalConfig{DNSMode: "dnsmasq"}
	if !config.UseDnsmasq() {
		t.Error("UseDnsmasq should return true")
	}

	config.DNSMode = "hosts"
	if config.UseDnsmasq() {
		t.Error("UseDnsmasq should return false")
	}
}

func TestGlobalConfig_UseHosts(t *testing.T) {
	config := &GlobalConfig{DNSMode: "hosts"}
	if !config.UseHosts() {
		t.Error("UseHosts should return true")
	}

	config.DNSMode = ""
	if !config.UseHosts() {
		t.Error("UseHosts should return true for empty")
	}

	config.DNSMode = "dnsmasq"
	if config.UseHosts() {
		t.Error("UseHosts should return false")
	}
}

func TestGlobalConfig_GetTLD(t *testing.T) {
	config := &GlobalConfig{TLD: "local"}
	if got := config.GetTLD(); got != "local" {
		t.Errorf("GetTLD() = %v, want local", got)
	}

	config.TLD = ""
	if got := config.GetTLD(); got != "test" {
		t.Errorf("GetTLD() = %v, want test (default)", got)
	}
}

func TestGlobalConfigExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Should not exist initially
	if GlobalConfigExists(tmpDir) {
		t.Error("Config should not exist initially")
	}

	// Create config
	SaveGlobalConfig(tmpDir, DefaultGlobalConfig())

	// Should exist now
	if !GlobalConfigExists(tmpDir) {
		t.Error("Config should exist after save")
	}
}

func TestInitGlobalConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Init should create config
	err := InitGlobalConfig(tmpDir)
	if err != nil {
		t.Fatalf("InitGlobalConfig failed: %v", err)
	}

	if !GlobalConfigExists(tmpDir) {
		t.Error("Config should exist after init")
	}

	// Init again should not fail (shouldn't overwrite)
	err = InitGlobalConfig(tmpDir)
	if err != nil {
		t.Fatalf("Second InitGlobalConfig failed: %v", err)
	}
}

func TestGlobalConfig_applyDefaults(t *testing.T) {
	config := &GlobalConfig{}
	config.applyDefaults()

	if config.DNSMode != "dnsmasq" {
		t.Errorf("DNSMode = %v, want dnsmasq", config.DNSMode)
	}
	if config.DefaultPHP != "8.2" {
		t.Errorf("DefaultPHP = %v, want 8.2", config.DefaultPHP)
	}
	if config.TLD != "test" {
		t.Errorf("TLD = %v, want test", config.TLD)
	}
}

func TestGlobalConfig_TidewaysCredentials(t *testing.T) {
	// Unset env vars so they don't leak into the test from the shell.
	t.Setenv("TIDEWAYS_CLI_TOKEN", "")
	t.Setenv("TIDEWAYS_ENVIRONMENT", "")

	cfg := &GlobalConfig{
		Profiling: ProfilingConfig{
			Tideways: TidewaysCredentials{
				AccessToken: "cli-token",
				Environment: "local_fromfile",
			},
		},
	}

	if !cfg.HasTidewaysAccessToken() {
		t.Error("HasTidewaysAccessToken should return true when access token is set")
	}

	// Without env var overrides, stored Environment should pass through.
	got := cfg.GetTidewaysCredentials()
	if got.Environment != "local_fromfile" {
		t.Errorf("Environment = %q, want local_fromfile", got.Environment)
	}
	// APIKey is never surfaced — it lives per project.
	if got.APIKey != "" {
		t.Errorf("APIKey leaked from GetTidewaysCredentials: %q", got.APIKey)
	}

	empty := &GlobalConfig{}
	if empty.HasTidewaysAccessToken() {
		t.Error("HasTidewaysAccessToken should return false for empty config")
	}

	// When no Environment is stored and no env var is set, we fall back to
	// the local_<username> default — never to an empty string.
	emptyGot := empty.GetTidewaysCredentials()
	if emptyGot.Environment == "" {
		t.Error("GetTidewaysCredentials should fall back to DefaultTidewaysEnvironment, got empty")
	}
	if emptyGot.Environment != DefaultTidewaysEnvironment() {
		t.Errorf("Environment = %q, want %q", emptyGot.Environment, DefaultTidewaysEnvironment())
	}

	// Environment variables should take precedence.
	t.Setenv("TIDEWAYS_CLI_TOKEN", "env-cli-token")
	t.Setenv("TIDEWAYS_ENVIRONMENT", "env-environment")

	got = cfg.GetTidewaysCredentials()
	if got.AccessToken != "env-cli-token" {
		t.Errorf("AccessToken = %q, want env-cli-token", got.AccessToken)
	}
	if got.Environment != "env-environment" {
		t.Errorf("Environment = %q, want env-environment", got.Environment)
	}
}

// TestGlobalConfig_HasLegacyTidewaysAPIKey verifies we can detect stale
// api_key entries from older MageBox versions so the config command can
// migrate them away.
func TestGlobalConfig_HasLegacyTidewaysAPIKey(t *testing.T) {
	withLegacy := &GlobalConfig{
		Profiling: ProfilingConfig{
			Tideways: TidewaysCredentials{APIKey: "stale"},
		},
	}
	if !withLegacy.HasLegacyTidewaysAPIKey() {
		t.Error("HasLegacyTidewaysAPIKey should return true when stale api_key is present")
	}

	clean := &GlobalConfig{
		Profiling: ProfilingConfig{
			Tideways: TidewaysCredentials{AccessToken: "t", Environment: "local_x"},
		},
	}
	if clean.HasLegacyTidewaysAPIKey() {
		t.Error("HasLegacyTidewaysAPIKey should return false for clean config")
	}
}

func TestDefaultTidewaysEnvironment(t *testing.T) {
	got := DefaultTidewaysEnvironment()
	if got == "" {
		t.Fatal("DefaultTidewaysEnvironment returned empty string")
	}
	if len(got) <= len("local_") || got[:len("local_")] != "local_" {
		t.Errorf("DefaultTidewaysEnvironment() = %q, want local_<username>", got)
	}
}

func TestGlobalConfig_TidewaysRoundTrip(t *testing.T) {
	// Make sure access_token and environment survive a save/load cycle.
	// api_key is intentionally not part of the persisted schema anymore.
	tmpDir := t.TempDir()

	cfg := &GlobalConfig{
		Profiling: ProfilingConfig{
			Tideways: TidewaysCredentials{
				AccessToken: "def456",
				Environment: "local_tester",
			},
		},
	}
	if err := SaveGlobalConfig(tmpDir, cfg); err != nil {
		t.Fatalf("SaveGlobalConfig failed: %v", err)
	}

	loaded, err := LoadGlobalConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadGlobalConfig failed: %v", err)
	}

	if loaded.Profiling.Tideways.AccessToken != "def456" {
		t.Errorf("AccessToken = %q, want def456", loaded.Profiling.Tideways.AccessToken)
	}
	if loaded.Profiling.Tideways.Environment != "local_tester" {
		t.Errorf("Environment = %q, want local_tester", loaded.Profiling.Tideways.Environment)
	}
}

// TestGlobalConfig_TidewaysLegacyAPIKeyStillUnmarshals verifies that a
// config.yaml written by an older MageBox version (which stored an api_key)
// still loads cleanly — the field is kept on the struct (marked deprecated)
// purely so we can detect and migrate it via HasLegacyTidewaysAPIKey.
func TestGlobalConfig_TidewaysLegacyAPIKeyStillUnmarshals(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := tmpDir + "/.magebox"
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	yaml := `profiling:
  tideways:
    api_key: legacy-project-key
    access_token: good-cli-token
    environment: local_tester
`
	if err := os.WriteFile(configDir+"/config.yaml", []byte(yaml), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	loaded, err := LoadGlobalConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadGlobalConfig failed: %v", err)
	}
	if !loaded.HasLegacyTidewaysAPIKey() {
		t.Error("HasLegacyTidewaysAPIKey should return true after loading a legacy config")
	}
	if loaded.Profiling.Tideways.AccessToken != "good-cli-token" {
		t.Errorf("AccessToken not loaded: %q", loaded.Profiling.Tideways.AccessToken)
	}
}

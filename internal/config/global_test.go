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

	if config.DNSMode != "hosts" {
		t.Errorf("DNSMode = %v, want hosts", config.DNSMode)
	}
	if config.DefaultPHP != "8.2" {
		t.Errorf("DefaultPHP = %v, want 8.2", config.DefaultPHP)
	}
	if config.TLD != "test" {
		t.Errorf("TLD = %v, want test", config.TLD)
	}
	if !config.Portainer {
		t.Error("Portainer should be true by default")
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
	if config.DNSMode != "hosts" {
		t.Errorf("DNSMode = %v, want hosts", config.DNSMode)
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

	if config.DNSMode != "hosts" {
		t.Errorf("DNSMode = %v, want hosts", config.DNSMode)
	}
	if config.DefaultPHP != "8.2" {
		t.Errorf("DefaultPHP = %v, want 8.2", config.DefaultPHP)
	}
	if config.TLD != "test" {
		t.Errorf("TLD = %v, want test", config.TLD)
	}
}

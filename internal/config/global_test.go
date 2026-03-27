package config

import (
	"os"
	"path/filepath"
	"testing"

	"qoliber/magebox/internal/remote"
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

func TestGlobalConfig_EnvironmentsWithRootPath(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".magebox")
	os.MkdirAll(configDir, 0755)

	configContent := `
environments:
  - name: staging
    user: app
    host: staging.example.com
    root_path: /data/web/project/current/
  - name: production
    user: app
    host: prod.example.com
    port: 2222
    root_path: /data/web/magento/current/
`
	os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configContent), 0644)

	config, err := LoadGlobalConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadGlobalConfig failed: %v", err)
	}

	if len(config.Environments) != 2 {
		t.Fatalf("len(Environments) = %d, want 2", len(config.Environments))
	}

	staging, err := config.GetEnvironment("staging")
	if err != nil {
		t.Fatalf("GetEnvironment(staging) failed: %v", err)
	}
	if staging.RootPath != "/data/web/project/current/" {
		t.Errorf("staging.RootPath = %q, want %q", staging.RootPath, "/data/web/project/current/")
	}

	prod, err := config.GetEnvironment("production")
	if err != nil {
		t.Fatalf("GetEnvironment(production) failed: %v", err)
	}
	if prod.RootPath != "/data/web/magento/current/" {
		t.Errorf("production.RootPath = %q, want %q", prod.RootPath, "/data/web/magento/current/")
	}
	if prod.Port != 2222 {
		t.Errorf("production.Port = %d, want 2222", prod.Port)
	}
}

func TestGlobalConfig_SaveLoadWithRootPath(t *testing.T) {
	tmpDir := t.TempDir()

	config := DefaultGlobalConfig()
	config.Environments = append(config.Environments, remote.Environment{
		Name:     "staging",
		User:     "deploy",
		Host:     "staging.example.com",
		RootPath: "/data/web/current/",
	})

	if err := SaveGlobalConfig(tmpDir, config); err != nil {
		t.Fatalf("SaveGlobalConfig failed: %v", err)
	}

	loaded, err := LoadGlobalConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadGlobalConfig failed: %v", err)
	}

	if len(loaded.Environments) != 1 {
		t.Fatalf("len(Environments) = %d, want 1", len(loaded.Environments))
	}
	if loaded.Environments[0].RootPath != "/data/web/current/" {
		t.Errorf("RootPath = %q, want %q", loaded.Environments[0].RootPath, "/data/web/current/")
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

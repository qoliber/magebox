package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qoliber/magebox/internal/config"
)

func TestNewEnvGenerator(t *testing.T) {
	cfg := &config.Config{
		Name: "testproject",
	}
	g := newEnvGenerator("/path/to/project", cfg)

	if g == nil {
		t.Fatal("newEnvGenerator should not return nil")
	}
	if g.projectPath != "/path/to/project" {
		t.Errorf("projectPath = %v, want /path/to/project", g.projectPath)
	}
	if g.config != cfg {
		t.Error("config should be set")
	}
}

func TestEnvGenerator_GenerateRandomPrefix(t *testing.T) {
	cfg := &config.Config{Name: "testproject"}
	g := newEnvGenerator("/path/to/project", cfg)

	prefix1 := g.generateRandomPrefix(3)
	prefix2 := g.generateRandomPrefix(3)

	if len(prefix1) != 3 {
		t.Errorf("prefix length = %d, want 3", len(prefix1))
	}
	if len(prefix2) != 3 {
		t.Errorf("prefix length = %d, want 3", len(prefix2))
	}
}

func TestEnvGenerator_GenerateCryptKey(t *testing.T) {
	cfg := &config.Config{Name: "testproject"}
	g := newEnvGenerator("/path/to/project", cfg)

	key := g.generateCryptKey()

	if len(key) != 32 {
		t.Errorf("crypt key length = %d, want 32", len(key))
	}
	// Key should be hex characters only
	for _, c := range key {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("crypt key contains invalid character: %c", c)
		}
	}
}

func TestEnvGenerator_GetMageMode_Default(t *testing.T) {
	cfg := &config.Config{Name: "testproject"}
	g := newEnvGenerator("/path/to/project", cfg)

	mode := g.getMageMode()

	if mode != "developer" {
		t.Errorf("getMageMode() = %v, want developer", mode)
	}
}

func TestEnvGenerator_GetMageMode_Custom(t *testing.T) {
	cfg := &config.Config{
		Name: "testproject",
		Env: map[string]string{
			"MAGE_MODE": "production",
		},
	}
	g := newEnvGenerator("/path/to/project", cfg)

	mode := g.getMageMode()

	if mode != "production" {
		t.Errorf("getMageMode() = %v, want production", mode)
	}
}

func TestEnvGenerator_GetDatabasePort_MySQL(t *testing.T) {
	tests := []struct {
		version      string
		expectedPort string
	}{
		{"8.0", "33080"},
		{"8.4", "33084"},
	}

	for _, tt := range tests {
		t.Run("MySQL_"+tt.version, func(t *testing.T) {
			cfg := &config.Config{
				Name: "testproject",
				Services: config.Services{
					MySQL: &config.ServiceConfig{
						Enabled: true,
						Version: tt.version,
					},
				},
			}
			g := newEnvGenerator("/path/to/project", cfg)

			port := g.getDatabasePort()

			if port != tt.expectedPort {
				t.Errorf("getDatabasePort() = %v, want %v", port, tt.expectedPort)
			}
		})
	}
}

func TestEnvGenerator_GetDatabasePort_MariaDB(t *testing.T) {
	tests := []struct {
		version      string
		expectedPort string
	}{
		{"10.6", "33110"},
		{"11.4", "33111"},
	}

	for _, tt := range tests {
		t.Run("MariaDB_"+tt.version, func(t *testing.T) {
			cfg := &config.Config{
				Name: "testproject",
				Services: config.Services{
					MariaDB: &config.ServiceConfig{
						Enabled: true,
						Version: tt.version,
					},
				},
			}
			g := newEnvGenerator("/path/to/project", cfg)

			port := g.getDatabasePort()

			if port != tt.expectedPort {
				t.Errorf("getDatabasePort() = %v, want %v", port, tt.expectedPort)
			}
		})
	}
}

func TestEnvGenerator_BuildTemplateData_Basic(t *testing.T) {
	cfg := &config.Config{
		Name: "testproject",
		Services: config.Services{
			MySQL: &config.ServiceConfig{
				Enabled: true,
				Version: "8.0",
			},
		},
	}
	g := newEnvGenerator("/path/to/project", cfg)

	data := g.buildTemplateData()

	if data.ProjectName != "testproject" {
		t.Errorf("ProjectName = %v, want testproject", data.ProjectName)
	}
	if data.MageMode != "developer" {
		t.Errorf("MageMode = %v, want developer", data.MageMode)
	}
	if data.DatabaseName != "testproject" {
		t.Errorf("DatabaseName = %v, want testproject", data.DatabaseName)
	}
	if data.DatabasePort != "33080" {
		t.Errorf("DatabasePort = %v, want 33080", data.DatabasePort)
	}
	if data.DatabaseUser != "root" {
		t.Errorf("DatabaseUser = %v, want root", data.DatabaseUser)
	}
	if data.DatabasePassword != "magebox" {
		t.Errorf("DatabasePassword = %v, want magebox", data.DatabasePassword)
	}
	if !data.HasMailpit {
		t.Error("HasMailpit should always be true")
	}
}

func TestEnvGenerator_BuildTemplateData_WithRedis(t *testing.T) {
	cfg := &config.Config{
		Name: "testproject",
		Services: config.Services{
			MySQL: &config.ServiceConfig{
				Enabled: true,
				Version: "8.0",
			},
			Redis: &config.ServiceConfig{
				Enabled: true,
			},
		},
	}
	g := newEnvGenerator("/path/to/project", cfg)

	data := g.buildTemplateData()

	if !data.HasRedis {
		t.Error("HasRedis should be true")
	}
	if data.RedisHost != "127.0.0.1" {
		t.Errorf("RedisHost = %v, want 127.0.0.1", data.RedisHost)
	}
	if data.RedisPort != "6379" {
		t.Errorf("RedisPort = %v, want 6379", data.RedisPort)
	}
	if data.RedisSessionDB != "2" {
		t.Errorf("RedisSessionDB = %v, want 2", data.RedisSessionDB)
	}
	if data.RedisCacheDB != "0" {
		t.Errorf("RedisCacheDB = %v, want 0", data.RedisCacheDB)
	}
	if data.RedisPageCacheDB != "1" {
		t.Errorf("RedisPageCacheDB = %v, want 1", data.RedisPageCacheDB)
	}
}

func TestEnvGenerator_BuildTemplateData_WithVarnish(t *testing.T) {
	cfg := &config.Config{
		Name: "testproject",
		Services: config.Services{
			MySQL: &config.ServiceConfig{
				Enabled: true,
				Version: "8.0",
			},
			Varnish: &config.ServiceConfig{
				Enabled: true,
			},
		},
	}
	g := newEnvGenerator("/path/to/project", cfg)

	data := g.buildTemplateData()

	if !data.HasVarnish {
		t.Error("HasVarnish should be true")
	}
}

func TestEnvGenerator_RenderTemplate_Basic(t *testing.T) {
	cfg := &config.Config{
		Name: "testproject",
		Services: config.Services{
			MySQL: &config.ServiceConfig{
				Enabled: true,
				Version: "8.0",
			},
		},
	}
	g := newEnvGenerator("/path/to/project", cfg)

	data := g.buildTemplateData()
	content, err := g.renderTemplate(data)

	if err != nil {
		t.Fatalf("renderTemplate failed: %v", err)
	}

	// Check basic structure
	if !strings.Contains(content, "<?php") {
		t.Error("env.php should start with <?php")
	}
	if !strings.Contains(content, "return [") {
		t.Error("env.php should contain return [")
	}
	if !strings.Contains(content, "];") {
		t.Error("env.php should end with ];")
	}

	// Check backend config
	if !strings.Contains(content, "'backend'") {
		t.Error("env.php should contain backend config")
	}
	if !strings.Contains(content, "'frontName' => 'admin'") {
		t.Error("env.php should contain admin frontName")
	}

	// Check database config
	if !strings.Contains(content, "'db'") {
		t.Error("env.php should contain db config")
	}
	if !strings.Contains(content, "'dbname' => 'testproject'") {
		t.Error("env.php should contain correct database name")
	}
	if !strings.Contains(content, "33080") {
		t.Error("env.php should contain MySQL 8.0 port 33080")
	}

	// Check MAGE_MODE defaults to developer
	if !strings.Contains(content, "'MAGE_MODE' => 'developer'") {
		t.Error("env.php should default to developer mode")
	}

	// Check cache types
	if !strings.Contains(content, "'cache_types'") {
		t.Error("env.php should contain cache_types")
	}

	// Check file session (no Redis)
	if !strings.Contains(content, "'save' => 'files'") {
		t.Error("env.php should use file session without Redis")
	}
}

func TestEnvGenerator_RenderTemplate_WithRedis(t *testing.T) {
	cfg := &config.Config{
		Name: "testproject",
		Services: config.Services{
			MySQL: &config.ServiceConfig{
				Enabled: true,
				Version: "8.0",
			},
			Redis: &config.ServiceConfig{
				Enabled: true,
			},
		},
	}
	g := newEnvGenerator("/path/to/project", cfg)

	data := g.buildTemplateData()
	content, err := g.renderTemplate(data)

	if err != nil {
		t.Fatalf("renderTemplate failed: %v", err)
	}

	// Check Redis session config
	if !strings.Contains(content, "'save' => 'redis'") {
		t.Error("env.php should configure Redis session")
	}
	if !strings.Contains(content, "'database' => '2'") {
		t.Error("env.php should use database 2 for sessions")
	}

	// Check Redis cache config (double backslashes in PHP output)
	if !strings.Contains(content, "Magento\\\\Framework\\\\Cache\\\\Backend\\\\Redis") {
		t.Error("env.php should use Redis cache backend")
	}
	if !strings.Contains(content, "'database' => '0'") {
		t.Error("env.php should use database 0 for default cache")
	}
	if !strings.Contains(content, "'database' => '1'") {
		t.Error("env.php should use database 1 for page cache")
	}
}

func TestEnvGenerator_RenderTemplate_WithVarnish(t *testing.T) {
	cfg := &config.Config{
		Name: "testproject",
		Services: config.Services{
			MySQL: &config.ServiceConfig{
				Enabled: true,
				Version: "8.0",
			},
			Varnish: &config.ServiceConfig{
				Enabled: true,
			},
		},
	}
	g := newEnvGenerator("/path/to/project", cfg)

	data := g.buildTemplateData()
	content, err := g.renderTemplate(data)

	if err != nil {
		t.Fatalf("renderTemplate failed: %v", err)
	}

	// Check Varnish http_cache_hosts config
	if !strings.Contains(content, "'http_cache_hosts'") {
		t.Error("env.php should contain http_cache_hosts for Varnish")
	}
	if !strings.Contains(content, "'port' => '6081'") {
		t.Error("env.php should contain Varnish port 6081")
	}
}

func TestEnvGenerator_RenderTemplate_WithMailpit(t *testing.T) {
	cfg := &config.Config{
		Name: "testproject",
		Services: config.Services{
			MySQL: &config.ServiceConfig{
				Enabled: true,
				Version: "8.0",
			},
		},
	}
	g := newEnvGenerator("/path/to/project", cfg)

	data := g.buildTemplateData()
	content, err := g.renderTemplate(data)

	if err != nil {
		t.Fatalf("renderTemplate failed: %v", err)
	}

	// Mailpit is always enabled
	if !strings.Contains(content, "'system'") {
		t.Error("env.php should contain system config for Mailpit")
	}
	if !strings.Contains(content, "'smtp'") {
		t.Error("env.php should contain smtp config")
	}
	if !strings.Contains(content, "'port' => '1025'") {
		t.Error("env.php should contain Mailpit SMTP port 1025")
	}
}

func TestEnvGenerator_RenderTemplate_CustomMageMode(t *testing.T) {
	cfg := &config.Config{
		Name: "testproject",
		Env: map[string]string{
			"MAGE_MODE": "production",
		},
		Services: config.Services{
			MySQL: &config.ServiceConfig{
				Enabled: true,
				Version: "8.0",
			},
		},
	}
	g := newEnvGenerator("/path/to/project", cfg)

	data := g.buildTemplateData()
	content, err := g.renderTemplate(data)

	if err != nil {
		t.Fatalf("renderTemplate failed: %v", err)
	}

	if !strings.Contains(content, "'MAGE_MODE' => 'production'") {
		t.Error("env.php should use custom MAGE_MODE from config")
	}
}

func TestEnvGenerator_RenderTemplate_MariaDB(t *testing.T) {
	cfg := &config.Config{
		Name: "testproject",
		Services: config.Services{
			MariaDB: &config.ServiceConfig{
				Enabled: true,
				Version: "10.6",
			},
		},
	}
	g := newEnvGenerator("/path/to/project", cfg)

	data := g.buildTemplateData()
	content, err := g.renderTemplate(data)

	if err != nil {
		t.Fatalf("renderTemplate failed: %v", err)
	}

	// MariaDB 10.6 should use port 33110
	if !strings.Contains(content, "33110") {
		t.Error("env.php should contain MariaDB 10.6 port 33110")
	}
}

func TestEnvGenerator_Generate(t *testing.T) {
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "testproject")

	// Create app/etc directory
	appEtcDir := filepath.Join(projectPath, "app", "etc")
	if err := os.MkdirAll(appEtcDir, 0755); err != nil {
		t.Fatalf("failed to create app/etc dir: %v", err)
	}

	cfg := &config.Config{
		Name: "testproject",
		Services: config.Services{
			MySQL: &config.ServiceConfig{
				Enabled: true,
				Version: "8.0",
			},
		},
	}
	g := newEnvGenerator(projectPath, cfg)

	if err := g.Generate(); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check that env.php was created
	envPath := filepath.Join(appEtcDir, "env.php")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		t.Error("env.php should have been created")
	}

	// Read and verify content
	content, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("failed to read env.php: %v", err)
	}

	if !strings.Contains(string(content), "<?php") {
		t.Error("env.php should contain PHP opening tag")
	}
	if !strings.Contains(string(content), "'dbname' => 'testproject'") {
		t.Error("env.php should contain correct database name")
	}
}

func TestEnvPHPTemplate_ValidSyntax(t *testing.T) {
	// Test that the embedded template parses correctly
	cfg := &config.Config{
		Name: "testproject",
		Services: config.Services{
			MySQL: &config.ServiceConfig{
				Enabled: true,
				Version: "8.0",
			},
		},
	}
	g := newEnvGenerator("/path/to/project", cfg)

	data := g.buildTemplateData()
	_, err := g.renderTemplate(data)

	if err != nil {
		t.Fatalf("Template should parse and render without error: %v", err)
	}
}

func TestEnvPHPTemplate_AllConditionalBranches(t *testing.T) {
	// Test all combinations of conditionals
	tests := []struct {
		name       string
		hasRedis   bool
		hasVarnish bool
	}{
		{"NoServices", false, false},
		{"OnlyRedis", true, false},
		{"OnlyVarnish", false, true},
		{"AllServices", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Name: "testproject",
				Services: config.Services{
					MySQL: &config.ServiceConfig{
						Enabled: true,
						Version: "8.0",
					},
				},
			}
			if tt.hasRedis {
				cfg.Services.Redis = &config.ServiceConfig{Enabled: true}
			}
			if tt.hasVarnish {
				cfg.Services.Varnish = &config.ServiceConfig{Enabled: true}
			}

			g := newEnvGenerator("/path/to/project", cfg)
			data := g.buildTemplateData()
			content, err := g.renderTemplate(data)

			if err != nil {
				t.Fatalf("renderTemplate failed: %v", err)
			}

			// Verify conditionals rendered correctly
			if tt.hasRedis {
				if !strings.Contains(content, "'save' => 'redis'") {
					t.Error("Should have Redis session")
				}
			} else {
				if !strings.Contains(content, "'save' => 'files'") {
					t.Error("Should have file session")
				}
			}

			if tt.hasVarnish {
				if !strings.Contains(content, "'http_cache_hosts'") {
					t.Error("Should have Varnish config")
				}
			} else {
				if strings.Contains(content, "'http_cache_hosts'") {
					t.Error("Should NOT have Varnish config")
				}
			}
		})
	}
}

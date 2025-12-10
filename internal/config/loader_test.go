package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoader_Load(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		dir := t.TempDir()
		configContent := `
name: mystore
domains:
  - host: mystore.test
php: "8.2"
services:
  mysql: "8.0"
  redis: true
`
		err := os.WriteFile(filepath.Join(dir, ".magebox"), []byte(configContent), 0644)
		if err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		loader := NewLoader(dir)
		config, err := loader.Load()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if config.Name != "mystore" {
			t.Errorf("Name = %v, want mystore", config.Name)
		}
		if config.PHP != "8.2" {
			t.Errorf("PHP = %v, want 8.2", config.PHP)
		}
		if len(config.Domains) != 1 {
			t.Errorf("len(Domains) = %v, want 1", len(config.Domains))
		}
		if config.Domains[0].Host != "mystore.test" {
			t.Errorf("Domains[0].Host = %v, want mystore.test", config.Domains[0].Host)
		}
		if !config.Services.HasMySQL() {
			t.Error("expected MySQL to be enabled")
		}
		if config.Services.MySQL.Version != "8.0" {
			t.Errorf("MySQL.Version = %v, want 8.0", config.Services.MySQL.Version)
		}
		if !config.Services.HasRedis() {
			t.Error("expected Redis to be enabled")
		}
	})

	t.Run("config not found", func(t *testing.T) {
		dir := t.TempDir()
		loader := NewLoader(dir)
		_, err := loader.Load()
		if err == nil {
			t.Error("expected error, got nil")
		}
		if _, ok := err.(*ConfigNotFoundError); !ok {
			t.Errorf("expected ConfigNotFoundError, got %T", err)
		}
	})

	t.Run("invalid yaml", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(filepath.Join(dir, ".magebox"), []byte("invalid: yaml: content:"), 0644)
		if err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		loader := NewLoader(dir)
		_, err = loader.Load()
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("validation error", func(t *testing.T) {
		dir := t.TempDir()
		// Missing required fields
		configContent := `
domains:
  - host: mystore.test
`
		err := os.WriteFile(filepath.Join(dir, ".magebox"), []byte(configContent), 0644)
		if err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		loader := NewLoader(dir)
		_, err = loader.Load()
		if err == nil {
			t.Error("expected validation error, got nil")
		}
	})
}

func TestLoader_LoadWithLocalOverride(t *testing.T) {
	t.Run("local overrides php version", func(t *testing.T) {
		dir := t.TempDir()

		mainConfig := `
name: mystore
domains:
  - host: mystore.test
php: "8.2"
services:
  mysql: "8.0"
`
		localConfig := `
php: "8.3"
`
		err := os.WriteFile(filepath.Join(dir, ".magebox"), []byte(mainConfig), 0644)
		if err != nil {
			t.Fatalf("failed to write main config: %v", err)
		}
		err = os.WriteFile(filepath.Join(dir, ".magebox.local"), []byte(localConfig), 0644)
		if err != nil {
			t.Fatalf("failed to write local config: %v", err)
		}

		loader := NewLoader(dir)
		config, err := loader.Load()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// PHP should be overridden
		if config.PHP != "8.3" {
			t.Errorf("PHP = %v, want 8.3", config.PHP)
		}
		// Other fields should remain from main
		if config.Name != "mystore" {
			t.Errorf("Name = %v, want mystore", config.Name)
		}
		if !config.Services.HasMySQL() {
			t.Error("expected MySQL to be enabled")
		}
	})

	t.Run("local overrides services", func(t *testing.T) {
		dir := t.TempDir()

		mainConfig := `
name: mystore
domains:
  - host: mystore.test
php: "8.2"
services:
  mysql: "8.0"
`
		localConfig := `
services:
  mysql:
    version: "8.0"
    port: 3307
`
		err := os.WriteFile(filepath.Join(dir, ".magebox"), []byte(mainConfig), 0644)
		if err != nil {
			t.Fatalf("failed to write main config: %v", err)
		}
		err = os.WriteFile(filepath.Join(dir, ".magebox.local"), []byte(localConfig), 0644)
		if err != nil {
			t.Fatalf("failed to write local config: %v", err)
		}

		loader := NewLoader(dir)
		config, err := loader.Load()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if config.Services.MySQL.Port != 3307 {
			t.Errorf("MySQL.Port = %v, want 3307", config.Services.MySQL.Port)
		}
	})

	t.Run("local merges env vars", func(t *testing.T) {
		dir := t.TempDir()

		mainConfig := `
name: mystore
domains:
  - host: mystore.test
php: "8.2"
env:
  MAGE_MODE: developer
  DB_HOST: localhost
`
		localConfig := `
env:
  MAGE_MODE: production
  XDEBUG_ENABLE: "1"
`
		err := os.WriteFile(filepath.Join(dir, ".magebox"), []byte(mainConfig), 0644)
		if err != nil {
			t.Fatalf("failed to write main config: %v", err)
		}
		err = os.WriteFile(filepath.Join(dir, ".magebox.local"), []byte(localConfig), 0644)
		if err != nil {
			t.Fatalf("failed to write local config: %v", err)
		}

		loader := NewLoader(dir)
		config, err := loader.Load()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Overridden value
		if config.Env["MAGE_MODE"] != "production" {
			t.Errorf("MAGE_MODE = %v, want production", config.Env["MAGE_MODE"])
		}
		// Preserved from main
		if config.Env["DB_HOST"] != "localhost" {
			t.Errorf("DB_HOST = %v, want localhost", config.Env["DB_HOST"])
		}
		// Added from local
		if config.Env["XDEBUG_ENABLE"] != "1" {
			t.Errorf("XDEBUG_ENABLE = %v, want 1", config.Env["XDEBUG_ENABLE"])
		}
	})
}

func TestLoader_FullConfig(t *testing.T) {
	dir := t.TempDir()

	configContent := `
name: mystore
domains:
  - host: mystore.test
    root: pub
    ssl: true
  - host: api.mystore.test
    root: pub/api
    ssl: false
php: "8.2"
services:
  mysql:
    version: "8.0"
    port: 3306
  redis: "7.2"
  opensearch: "2.12"
  mailpit: true
env:
  MAGE_MODE: developer
commands:
  build: |
    composer install
    bin/magento setup:upgrade
`
	err := os.WriteFile(filepath.Join(dir, ".magebox"), []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	loader := NewLoader(dir)
	config, err := loader.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all fields
	if config.Name != "mystore" {
		t.Errorf("Name = %v, want mystore", config.Name)
	}

	if len(config.Domains) != 2 {
		t.Fatalf("len(Domains) = %v, want 2", len(config.Domains))
	}

	// First domain
	if config.Domains[0].Host != "mystore.test" {
		t.Errorf("Domains[0].Host = %v, want mystore.test", config.Domains[0].Host)
	}
	if config.Domains[0].GetRoot() != "pub" {
		t.Errorf("Domains[0].GetRoot() = %v, want pub", config.Domains[0].GetRoot())
	}
	if !config.Domains[0].IsSSLEnabled() {
		t.Error("Domains[0] should have SSL enabled")
	}

	// Second domain
	if config.Domains[1].Host != "api.mystore.test" {
		t.Errorf("Domains[1].Host = %v, want api.mystore.test", config.Domains[1].Host)
	}
	if config.Domains[1].GetRoot() != "pub/api" {
		t.Errorf("Domains[1].GetRoot() = %v, want pub/api", config.Domains[1].GetRoot())
	}
	if config.Domains[1].IsSSLEnabled() {
		t.Error("Domains[1] should have SSL disabled")
	}

	// Services
	if !config.Services.HasMySQL() {
		t.Error("expected MySQL")
	}
	if config.Services.MySQL.Version != "8.0" {
		t.Errorf("MySQL.Version = %v, want 8.0", config.Services.MySQL.Version)
	}
	if config.Services.MySQL.Port != 3306 {
		t.Errorf("MySQL.Port = %v, want 3306", config.Services.MySQL.Port)
	}

	if !config.Services.HasRedis() {
		t.Error("expected Redis")
	}
	if config.Services.Redis.Version != "7.2" {
		t.Errorf("Redis.Version = %v, want 7.2", config.Services.Redis.Version)
	}

	if !config.Services.HasOpenSearch() {
		t.Error("expected OpenSearch")
	}

	if !config.Services.HasMailpit() {
		t.Error("expected Mailpit")
	}

	// Env
	if config.Env["MAGE_MODE"] != "developer" {
		t.Errorf("MAGE_MODE = %v, want developer", config.Env["MAGE_MODE"])
	}

	// Commands
	if cmd, ok := config.Commands["build"]; !ok {
		t.Error("expected build command")
	} else if cmd.Run == "" {
		t.Error("expected build command to have Run field")
	}
}

func TestLoader_Commands(t *testing.T) {
	t.Run("string command syntax", func(t *testing.T) {
		dir := t.TempDir()
		configContent := `
name: mystore
domains:
  - host: mystore.test
php: "8.2"
commands:
  deploy: "php bin/magento deploy:mode:set production"
  reindex: "php bin/magento indexer:reindex"
`
		err := os.WriteFile(filepath.Join(dir, ".magebox"), []byte(configContent), 0644)
		if err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		loader := NewLoader(dir)
		config, err := loader.Load()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(config.Commands) != 2 {
			t.Errorf("len(Commands) = %v, want 2", len(config.Commands))
		}

		deploy := config.Commands["deploy"]
		if deploy.Run != "php bin/magento deploy:mode:set production" {
			t.Errorf("deploy.Run = %v, want php bin/magento deploy:mode:set production", deploy.Run)
		}
		if deploy.Description != "" {
			t.Errorf("deploy.Description = %v, want empty", deploy.Description)
		}
	})

	t.Run("object command syntax", func(t *testing.T) {
		dir := t.TempDir()
		configContent := `
name: mystore
domains:
  - host: mystore.test
php: "8.2"
commands:
  deploy:
    description: "Deploy to production mode"
    run: "php bin/magento deploy:mode:set production"
  reindex:
    description: "Reindex all Magento indexes"
    run: "php bin/magento indexer:reindex"
`
		err := os.WriteFile(filepath.Join(dir, ".magebox"), []byte(configContent), 0644)
		if err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		loader := NewLoader(dir)
		config, err := loader.Load()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(config.Commands) != 2 {
			t.Errorf("len(Commands) = %v, want 2", len(config.Commands))
		}

		deploy := config.Commands["deploy"]
		if deploy.Run != "php bin/magento deploy:mode:set production" {
			t.Errorf("deploy.Run = %v, want php bin/magento deploy:mode:set production", deploy.Run)
		}
		if deploy.Description != "Deploy to production mode" {
			t.Errorf("deploy.Description = %v, want 'Deploy to production mode'", deploy.Description)
		}

		reindex := config.Commands["reindex"]
		if reindex.Run != "php bin/magento indexer:reindex" {
			t.Errorf("reindex.Run = %v, want php bin/magento indexer:reindex", reindex.Run)
		}
		if reindex.Description != "Reindex all Magento indexes" {
			t.Errorf("reindex.Description = %v, want 'Reindex all Magento indexes'", reindex.Description)
		}
	})

	t.Run("mixed command syntax", func(t *testing.T) {
		dir := t.TempDir()
		configContent := `
name: mystore
domains:
  - host: mystore.test
php: "8.2"
commands:
  deploy: "php bin/magento deploy:mode:set production"
  setup:
    description: "Full project setup"
    run: "composer install && php bin/magento setup:upgrade"
`
		err := os.WriteFile(filepath.Join(dir, ".magebox"), []byte(configContent), 0644)
		if err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		loader := NewLoader(dir)
		config, err := loader.Load()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// String syntax
		deploy := config.Commands["deploy"]
		if deploy.Run != "php bin/magento deploy:mode:set production" {
			t.Errorf("deploy.Run = %v, want php bin/magento deploy:mode:set production", deploy.Run)
		}
		if deploy.Description != "" {
			t.Errorf("deploy.Description should be empty for string syntax")
		}

		// Object syntax
		setup := config.Commands["setup"]
		if setup.Run != "composer install && php bin/magento setup:upgrade" {
			t.Errorf("setup.Run = %v", setup.Run)
		}
		if setup.Description != "Full project setup" {
			t.Errorf("setup.Description = %v, want 'Full project setup'", setup.Description)
		}
	})
}

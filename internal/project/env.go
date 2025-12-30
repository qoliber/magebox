package project

import (
	"bytes"
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/qoliber/magebox/internal/config"
)

//go:embed templates/env.php.tmpl
var envPHPTemplate string

// EnvPHPData contains all variables available in env.php.tmpl
type EnvPHPData struct {
	// Project
	ProjectName   string
	MageMode      string
	CryptKey      string
	CacheIDPrefix string

	// Database
	DatabaseHost     string
	DatabasePort     string
	DatabaseName     string
	DatabaseUser     string
	DatabasePassword string

	// Service flags (for conditionals)
	HasRedis   bool
	HasVarnish bool
	HasMailpit bool

	// Redis configuration
	RedisHost        string
	RedisPort        string
	RedisSessionDB   string
	RedisCacheDB     string
	RedisPageCacheDB string

	// Mailpit configuration
	MailpitHost string
	MailpitPort string
}

// envGenerator generates Magento 2 app/etc/env.php configuration
type envGenerator struct {
	projectPath string
	config      *config.Config
}

// newEnvGenerator creates a new env.php generator
func newEnvGenerator(projectPath string, cfg *config.Config) *envGenerator {
	return &envGenerator{
		projectPath: projectPath,
		config:      cfg,
	}
}

// Generate creates the env.php file
func (g *envGenerator) Generate() error {
	envPath := filepath.Join(g.projectPath, "app", "etc", "env.php")

	// Build template data
	data := g.buildTemplateData()

	// Render template
	content, err := g.renderTemplate(data)
	if err != nil {
		return fmt.Errorf("failed to render env.php template: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(envPath), 0755); err != nil {
		return fmt.Errorf("failed to create app/etc directory: %w", err)
	}

	// Write the file
	if err := os.WriteFile(envPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write env.php: %w", err)
	}

	return nil
}

// buildTemplateData constructs the data structure for the template
func (g *envGenerator) buildTemplateData() EnvPHPData {
	data := EnvPHPData{
		// Project settings
		ProjectName:   g.config.Name,
		MageMode:      g.getMageMode(),
		CryptKey:      g.generateCryptKey(),
		CacheIDPrefix: g.generateRandomPrefix(3) + "_",

		// Database defaults
		DatabaseHost:     "127.0.0.1",
		DatabasePort:     g.getDatabasePort(),
		DatabaseName:     g.config.DatabaseName(),
		DatabaseUser:     "root",
		DatabasePassword: "magebox",

		// Service flags
		HasRedis:   g.config.Services.HasRedis(),
		HasVarnish: g.config.Services.HasVarnish(),
		HasMailpit: true, // Always enabled for local dev safety

		// Redis configuration
		RedisHost:        "127.0.0.1",
		RedisPort:        "6379",
		RedisSessionDB:   "2",
		RedisCacheDB:     "0",
		RedisPageCacheDB: "1",

		// Mailpit configuration
		MailpitHost: "127.0.0.1",
		MailpitPort: "1025",
	}

	return data
}

// getMageMode returns the MAGE_MODE from config or defaults to "developer"
func (g *envGenerator) getMageMode() string {
	if g.config.Env != nil {
		if mode, ok := g.config.Env["MAGE_MODE"]; ok {
			return mode
		}
	}
	return "developer"
}

// getDatabasePort returns the appropriate database port based on service config
func (g *envGenerator) getDatabasePort() string {
	if g.config.Services.HasMySQL() {
		version := g.config.Services.MySQL.Version
		// Map version to port: 8.0 -> 33080, 8.4 -> 33084
		versionNoDots := strings.ReplaceAll(version, ".", "")
		if len(versionNoDots) >= 2 {
			return fmt.Sprintf("330%s", versionNoDots[:2])
		}
		return "33080" // Default MySQL 8.0 port
	}

	if g.config.Services.HasMariaDB() {
		version := g.config.Services.MariaDB.Version
		versionNoDots := strings.ReplaceAll(version, ".", "")
		// MariaDB ports: 10.6 -> 33110, 11.4 -> 33111
		if len(versionNoDots) >= 2 {
			return fmt.Sprintf("331%s", versionNoDots[:2])
		}
		return "33106" // Default MariaDB port
	}

	return "33080" // Default fallback
}

// renderTemplate renders the env.php template with the given data
func (g *envGenerator) renderTemplate(data EnvPHPData) (string, error) {
	tmpl, err := template.New("env.php").Parse(envPHPTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// generateRandomPrefix generates a random alphanumeric prefix
func (g *envGenerator) generateRandomPrefix(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "mb"
	}
	return hex.EncodeToString(bytes)[:length]
}

// generateCryptKey generates a random 32-char hex crypt key
func (g *envGenerator) generateCryptKey() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "0000000000000000000000000000000000000000"
	}
	return hex.EncodeToString(bytes)
}

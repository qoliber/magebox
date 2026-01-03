package docker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"qoliber/magebox/internal/config"
	"qoliber/magebox/internal/platform"
	"gopkg.in/yaml.v3"
)

func setupTestComposeGenerator(t *testing.T) (*ComposeGenerator, string) {
	tmpDir := t.TempDir()
	p := &platform.Platform{
		Type:    platform.Linux,
		HomeDir: tmpDir,
	}
	g := NewComposeGenerator(p)
	return g, tmpDir
}

func TestNewComposeGenerator(t *testing.T) {
	g, tmpDir := setupTestComposeGenerator(t)

	if g == nil {
		t.Fatal("NewComposeGenerator should not return nil")
	}

	expectedDir := filepath.Join(tmpDir, ".magebox", "docker")
	if g.composeDir != expectedDir {
		t.Errorf("composeDir = %v, want %v", g.composeDir, expectedDir)
	}
}

func TestComposeGenerator_ComposeDir(t *testing.T) {
	g, tmpDir := setupTestComposeGenerator(t)

	expected := filepath.Join(tmpDir, ".magebox", "docker")
	if got := g.ComposeDir(); got != expected {
		t.Errorf("ComposeDir() = %v, want %v", got, expected)
	}
}

func TestComposeGenerator_ComposeFilePath(t *testing.T) {
	g, tmpDir := setupTestComposeGenerator(t)

	expected := filepath.Join(tmpDir, ".magebox", "docker", "docker-compose.yml")
	if got := g.ComposeFilePath(); got != expected {
		t.Errorf("ComposeFilePath() = %v, want %v", got, expected)
	}
}

func TestComposeGenerator_GenerateGlobalServices(t *testing.T) {
	g, _ := setupTestComposeGenerator(t)

	configs := []*config.Config{
		{
			Name: "project1",
			Services: config.Services{
				MySQL: &config.ServiceConfig{Enabled: true, Version: "8.0"},
				Redis: &config.ServiceConfig{Enabled: true},
			},
		},
	}

	err := g.GenerateGlobalServices(configs)
	if err != nil {
		t.Fatalf("GenerateGlobalServices failed: %v", err)
	}

	// Check that compose file was created
	composeFile := g.ComposeFilePath()
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		t.Error("Compose file should have been created")
	}

	// Read and verify content
	content, err := os.ReadFile(composeFile)
	if err != nil {
		t.Fatalf("Failed to read compose file: %v", err)
	}

	var compose ComposeConfig
	if err := yaml.Unmarshal(content, &compose); err != nil {
		t.Fatalf("Failed to parse compose file: %v", err)
	}

	// Should have MySQL 8.0 and Redis
	if _, ok := compose.Services["mysql80"]; !ok {
		t.Error("Compose should contain mysql80 service")
	}
	if _, ok := compose.Services["redis"]; !ok {
		t.Error("Compose should contain redis service")
	}
}

func TestComposeGenerator_GenerateMultipleVersions(t *testing.T) {
	g, _ := setupTestComposeGenerator(t)

	configs := []*config.Config{
		{
			Name: "project1",
			Services: config.Services{
				MySQL: &config.ServiceConfig{Enabled: true, Version: "8.0"},
			},
		},
		{
			Name: "project2",
			Services: config.Services{
				MySQL: &config.ServiceConfig{Enabled: true, Version: "5.7"},
			},
		},
	}

	err := g.GenerateGlobalServices(configs)
	if err != nil {
		t.Fatalf("GenerateGlobalServices failed: %v", err)
	}

	content, err := os.ReadFile(g.ComposeFilePath())
	if err != nil {
		t.Fatalf("Failed to read compose file: %v", err)
	}

	var compose ComposeConfig
	if err := yaml.Unmarshal(content, &compose); err != nil {
		t.Fatalf("Failed to parse compose file: %v", err)
	}

	// Should have both MySQL versions
	if _, ok := compose.Services["mysql80"]; !ok {
		t.Error("Compose should contain mysql80 service")
	}
	if _, ok := compose.Services["mysql57"]; !ok {
		t.Error("Compose should contain mysql57 service")
	}
}

func TestComposeGenerator_GenerateAllServices(t *testing.T) {
	g, _ := setupTestComposeGenerator(t)

	configs := []*config.Config{
		{
			Name: "fullproject",
			Services: config.Services{
				MySQL:      &config.ServiceConfig{Enabled: true, Version: "8.0"},
				Redis:      &config.ServiceConfig{Enabled: true},
				OpenSearch: &config.ServiceConfig{Enabled: true, Version: "2.12"},
				RabbitMQ:   &config.ServiceConfig{Enabled: true},
				Mailpit:    &config.ServiceConfig{Enabled: true},
			},
		},
	}

	err := g.GenerateGlobalServices(configs)
	if err != nil {
		t.Fatalf("GenerateGlobalServices failed: %v", err)
	}

	content, err := os.ReadFile(g.ComposeFilePath())
	if err != nil {
		t.Fatalf("Failed to read compose file: %v", err)
	}

	var compose ComposeConfig
	if err := yaml.Unmarshal(content, &compose); err != nil {
		t.Fatalf("Failed to parse compose file: %v", err)
	}

	expectedServices := []string{"mysql80", "redis", "opensearch212", "rabbitmq", "mailpit"}
	for _, svc := range expectedServices {
		if _, ok := compose.Services[svc]; !ok {
			t.Errorf("Compose should contain %s service", svc)
		}
	}
}

func TestComposeGenerator_getMySQLPort(t *testing.T) {
	g, _ := setupTestComposeGenerator(t)

	tests := []struct {
		version  string
		expected int
	}{
		{"5.7", 33057},
		{"8.0", 33080},
		{"8.4", 33084},
		{"unknown", 33080}, // default
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			if got := g.getMySQLPort(tt.version); got != tt.expected {
				t.Errorf("getMySQLPort(%v) = %v, want %v", tt.version, got, tt.expected)
			}
		})
	}
}

func TestComposeGenerator_getMariaDBPort(t *testing.T) {
	g, _ := setupTestComposeGenerator(t)

	tests := []struct {
		version  string
		expected int
	}{
		{"10.4", 33104},
		{"10.6", 33106},
		{"11.4", 33114},
		{"unknown", 33106}, // default
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			if got := g.getMariaDBPort(tt.version); got != tt.expected {
				t.Errorf("getMariaDBPort(%v) = %v, want %v", tt.version, got, tt.expected)
			}
		})
	}
}

func TestComposeService_MySQL(t *testing.T) {
	g, _ := setupTestComposeGenerator(t)

	svcCfg := &config.ServiceConfig{
		Enabled: true,
		Version: "8.0",
	}
	svc := g.getMySQLService(svcCfg)

	if svc.Image != "mysql:8.0" {
		t.Errorf("Image = %v, want mysql:8.0", svc.Image)
	}
	if len(svc.Ports) != 1 || !strings.Contains(svc.Ports[0], "33080:3306") {
		t.Errorf("Ports = %v, want 33080:3306", svc.Ports)
	}
	if svc.Environment["MYSQL_ROOT_PASSWORD"] != DefaultDBRootPassword {
		t.Errorf("MYSQL_ROOT_PASSWORD should be %s", DefaultDBRootPassword)
	}
	if svc.Restart != "unless-stopped" {
		t.Error("Restart should be unless-stopped")
	}
	if svc.HealthCheck == nil {
		t.Error("HealthCheck should not be nil")
	}
}

func TestComposeService_MySQL_WithMemory(t *testing.T) {
	g, _ := setupTestComposeGenerator(t)

	svcCfg := &config.ServiceConfig{
		Enabled: true,
		Version: "8.0",
		Memory:  "1g",
	}
	svc := g.getMySQLService(svcCfg)

	if svc.Environment["MYSQL_INNODB_BUFFER_POOL_SIZE"] != "1g" {
		t.Error("MYSQL_INNODB_BUFFER_POOL_SIZE should be 1g when memory is specified")
	}
}

func TestComposeService_Redis(t *testing.T) {
	g, _ := setupTestComposeGenerator(t)

	svc := g.getRedisService()

	if !strings.Contains(svc.Image, "redis") {
		t.Errorf("Image = %v, should contain redis", svc.Image)
	}
	if len(svc.Ports) != 1 || svc.Ports[0] != "6379:6379" {
		t.Errorf("Ports = %v, want [6379:6379]", svc.Ports)
	}
}

func TestComposeService_OpenSearch(t *testing.T) {
	g, _ := setupTestComposeGenerator(t)

	svcCfg := &config.ServiceConfig{
		Enabled: true,
		Version: "2.12",
		Memory:  "2g",
	}
	svc := g.getOpenSearchService(svcCfg)

	if !strings.Contains(svc.Image, "opensearch") {
		t.Errorf("Image = %v, should contain opensearch", svc.Image)
	}
	if svc.Environment["DISABLE_SECURITY_PLUGIN"] != "true" {
		t.Error("DISABLE_SECURITY_PLUGIN should be true")
	}
	if svc.Environment["OPENSEARCH_JAVA_OPTS"] != "-Xms2g -Xmx2g" {
		t.Errorf("OPENSEARCH_JAVA_OPTS = %v, want -Xms2g -Xmx2g", svc.Environment["OPENSEARCH_JAVA_OPTS"])
	}
	if !strings.Contains(svc.Command, "analysis-icu") {
		t.Error("Command should install analysis-icu plugin")
	}
	if !strings.Contains(svc.Command, "analysis-phonetic") {
		t.Error("Command should install analysis-phonetic plugin")
	}
}

func TestComposeService_RabbitMQ(t *testing.T) {
	g, _ := setupTestComposeGenerator(t)

	svc := g.getRabbitMQService()

	if !strings.Contains(svc.Image, "rabbitmq") {
		t.Errorf("Image = %v, should contain rabbitmq", svc.Image)
	}
	if len(svc.Ports) != 2 {
		t.Errorf("RabbitMQ should expose 2 ports, got %d", len(svc.Ports))
	}
}

func TestComposeService_Mailpit(t *testing.T) {
	g, _ := setupTestComposeGenerator(t)

	svc := g.getMailpitService()

	if !strings.Contains(svc.Image, "mailpit") {
		t.Errorf("Image = %v, should contain mailpit", svc.Image)
	}
	if len(svc.Ports) != 2 {
		t.Errorf("Mailpit should expose 2 ports, got %d", len(svc.Ports))
	}
}

func TestNewDockerController(t *testing.T) {
	c := NewDockerController("/path/to/docker-compose.yml")

	if c == nil {
		t.Fatal("NewDockerController should not return nil")
	}
	if c.composeFile != "/path/to/docker-compose.yml" {
		t.Errorf("composeFile = %v, want /path/to/docker-compose.yml", c.composeFile)
	}
}

func TestComposeGenerator_collectRequiredServices(t *testing.T) {
	g, _ := setupTestComposeGenerator(t)

	configs := []*config.Config{
		{
			Name: "project1",
			Services: config.Services{
				MySQL: &config.ServiceConfig{Enabled: true, Version: "8.0"},
				Redis: &config.ServiceConfig{Enabled: true},
			},
		},
		{
			Name: "project2",
			Services: config.Services{
				MariaDB:    &config.ServiceConfig{Enabled: true, Version: "10.6"},
				OpenSearch: &config.ServiceConfig{Enabled: true, Version: "2.19.4"},
			},
		},
	}

	rs := g.collectRequiredServices(configs)

	if rs.mysql["8.0"] == nil {
		t.Error("Should require MySQL 8.0")
	}
	if rs.mariadb["10.6"] == nil {
		t.Error("Should require MariaDB 10.6")
	}
	if !rs.redis {
		t.Error("Should require Redis")
	}
	if rs.opensearch["2.19.4"] == nil {
		t.Error("Should require OpenSearch 2.19.4")
	}
}

func TestComposeConfig_Networks(t *testing.T) {
	g, _ := setupTestComposeGenerator(t)

	configs := []*config.Config{
		{
			Name: "project1",
			Services: config.Services{
				Redis: &config.ServiceConfig{Enabled: true},
			},
		},
	}

	err := g.GenerateGlobalServices(configs)
	if err != nil {
		t.Fatalf("GenerateGlobalServices failed: %v", err)
	}

	content, err := os.ReadFile(g.ComposeFilePath())
	if err != nil {
		t.Fatalf("Failed to read compose file: %v", err)
	}

	var compose ComposeConfig
	if err := yaml.Unmarshal(content, &compose); err != nil {
		t.Fatalf("Failed to parse compose file: %v", err)
	}

	if _, ok := compose.Networks["magebox"]; !ok {
		t.Error("Compose should have magebox network")
	}
}

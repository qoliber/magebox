package docker

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"qoliber/magebox/internal/config"
	"qoliber/magebox/internal/platform"

	"gopkg.in/yaml.v3"
)

// setupMockDockerHub starts a lightweight httptest server that mimics the Docker Hub
// tags API for the given namespace/image/version combinations. The provided tags map
// is keyed by "namespace/image:major.minor" and the value is the slice of full tags
// that the mock should return for that query. Returns a cleanup function that restores
// the original values of dockerHubAPIBase and clears resolvedTags.
func setupMockDockerHub(t *testing.T, tags map[string][]string) func() {
	t.Helper()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// URL pattern: /v2/repositories/{namespace}/{image}/tags?name={prefix}.&...
		const repoPrefix = "/v2/repositories/"
		if !strings.HasPrefix(r.URL.Path, repoPrefix) {
			http.NotFound(w, r)
			return
		}
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, repoPrefix), "/")
		if len(parts) < 3 {
			http.NotFound(w, r)
			return
		}
		namespace := parts[0]
		image := parts[1]
		prefix := strings.TrimSuffix(r.URL.Query().Get("name"), ".")
		key := namespace + "/" + image + ":" + prefix

		tagNames, ok := tags[key]
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"results": []interface{}{}})
			return
		}

		type tagResult struct {
			Name string `json:"name"`
		}
		results := make([]tagResult, len(tagNames))
		for i, name := range tagNames {
			results[i] = tagResult{Name: name}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"results": results})
	}))

	prevBase := dockerHubAPIBase
	dockerHubAPIBase = ts.URL

	return func() {
		ts.Close()
		dockerHubAPIBase = prevBase
		// Clear in-memory cache so other tests are not affected.
		resolvedTags.Range(func(k, _ interface{}) bool {
			resolvedTags.Delete(k)
			return true
		})
	}
}

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
	svc := g.getMySQLService(svcCfg, false)

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

func TestComposeService_MySQL_WithStandardPort(t *testing.T) {
	g, _ := setupTestComposeGenerator(t)

	svcCfg := &config.ServiceConfig{
		Enabled: true,
		Version: "8.0",
	}
	svc := g.getMySQLService(svcCfg, true)

	if len(svc.Ports) != 2 {
		t.Fatalf("Ports = %v, want 2 port mappings (version-specific + standard)", svc.Ports)
	}
	if !strings.Contains(svc.Ports[0], "33080:3306") {
		t.Errorf("Ports[0] = %v, want 33080:3306", svc.Ports[0])
	}
	if !strings.Contains(svc.Ports[1], "3306:3306") {
		t.Errorf("Ports[1] = %v, want 3306:3306", svc.Ports[1])
	}
}

func TestComposeService_MySQL_WithMemory(t *testing.T) {
	g, _ := setupTestComposeGenerator(t)

	svcCfg := &config.ServiceConfig{
		Enabled: true,
		Version: "8.0",
		Memory:  "1g",
	}
	svc := g.getMySQLService(svcCfg, false)

	if svc.Environment["MYSQL_INNODB_BUFFER_POOL_SIZE"] != "1g" {
		t.Error("MYSQL_INNODB_BUFFER_POOL_SIZE should be 1g when memory is specified")
	}
}

func TestComposeService_MySQL_WithCustomConfig(t *testing.T) {
	g, tmpDir := setupTestComposeGenerator(t)

	// Create the custom config file
	cnfDir := filepath.Join(tmpDir, ".magebox", "docker")
	if err := os.MkdirAll(cnfDir, 0755); err != nil {
		t.Fatal(err)
	}
	cnfPath := filepath.Join(cnfDir, "mysql-custom.cnf")
	if err := os.WriteFile(cnfPath, []byte("[mysqld]\nskip-log-bin\n"), 0644); err != nil {
		t.Fatal(err)
	}

	svcCfg := &config.ServiceConfig{
		Enabled: true,
		Version: "8.0",
	}
	svc := g.getMySQLService(svcCfg, false)

	if len(svc.Volumes) != 2 {
		t.Fatalf("Volumes = %v, want 2 entries (data + custom config)", svc.Volumes)
	}
	if !strings.Contains(svc.Volumes[1], "mysql-custom.cnf:/etc/mysql/conf.d/custom.cnf:ro") {
		t.Errorf("Volumes[1] = %v, want custom cnf mount", svc.Volumes[1])
	}
}

func TestComposeService_MySQL_WithoutCustomConfig(t *testing.T) {
	g, _ := setupTestComposeGenerator(t)

	svcCfg := &config.ServiceConfig{
		Enabled: true,
		Version: "8.0",
	}
	svc := g.getMySQLService(svcCfg, false)

	if len(svc.Volumes) != 1 {
		t.Errorf("Volumes = %v, want only data volume when no custom config exists", svc.Volumes)
	}
}

func TestComposeService_MariaDB_WithCustomConfig(t *testing.T) {
	g, tmpDir := setupTestComposeGenerator(t)

	// Create the custom config file
	cnfDir := filepath.Join(tmpDir, ".magebox", "docker")
	if err := os.MkdirAll(cnfDir, 0755); err != nil {
		t.Fatal(err)
	}
	cnfPath := filepath.Join(cnfDir, "mariadb-custom.cnf")
	if err := os.WriteFile(cnfPath, []byte("[mysqld]\nskip-log-bin\n"), 0644); err != nil {
		t.Fatal(err)
	}

	svcCfg := &config.ServiceConfig{
		Enabled: true,
		Version: "10.6",
	}
	svc := g.getMariaDBService(svcCfg, false)

	if len(svc.Volumes) != 2 {
		t.Fatalf("Volumes = %v, want 2 entries (data + custom config)", svc.Volumes)
	}
	if !strings.Contains(svc.Volumes[1], "mariadb-custom.cnf:/etc/mysql/conf.d/custom.cnf:ro") {
		t.Errorf("Volumes[1] = %v, want custom cnf mount", svc.Volumes[1])
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
	svc := g.getOpenSearchService(svcCfg, false)

	if !strings.Contains(svc.Image, "opensearch") {
		t.Errorf("Image = %v, should contain opensearch", svc.Image)
	}
	if len(svc.Ports) != 1 || !strings.Contains(svc.Ports[0], "9252:9200") {
		t.Errorf("Ports = %v, want [9252:9200]", svc.Ports)
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

func TestComposeService_OpenSearch_WithStandardPort(t *testing.T) {
	g, _ := setupTestComposeGenerator(t)

	svcCfg := &config.ServiceConfig{
		Enabled: true,
		Version: "2.19.4",
	}
	svc := g.getOpenSearchService(svcCfg, true)

	if len(svc.Ports) != 2 {
		t.Fatalf("Ports = %v, want 2 port mappings (version-specific + standard)", svc.Ports)
	}
	if !strings.Contains(svc.Ports[0], "9259:9200") {
		t.Errorf("Ports[0] = %v, want 9259:9200", svc.Ports[0])
	}
	if !strings.Contains(svc.Ports[1], "9200:9200") {
		t.Errorf("Ports[1] = %v, want 9200:9200", svc.Ports[1])
	}
}

func TestComposeService_Elasticsearch_WithStandardPort(t *testing.T) {
	g, _ := setupTestComposeGenerator(t)

	svcCfg := &config.ServiceConfig{
		Enabled: true,
		Version: "7.17",
	}
	svc := g.getElasticsearchService(svcCfg, true)

	if len(svc.Ports) != 2 {
		t.Fatalf("Ports = %v, want 2 port mappings (version-specific + standard)", svc.Ports)
	}
	if !strings.Contains(svc.Ports[0], "9657:9200") {
		t.Errorf("Ports[0] = %v, want 9657:9200", svc.Ports[0])
	}
	if !strings.Contains(svc.Ports[1], "9200:9200") {
		t.Errorf("Ports[1] = %v, want 9200:9200", svc.Ports[1])
	}
}

func TestGetOpenSearchPort(t *testing.T) {
	tests := []struct {
		version  string
		expected int
	}{
		{"1.3", 9223},
		{"2.5", 9245},
		{"2.11", 9251},
		{"2.12", 9252},
		{"2.19", 9259},
		{"2.19.4", 9259}, // patch version should be stripped
		{"3.0", 9260},
		{"3.3", 9263},
		{"4.0", 9280}, // unknown version uses formula
		{"1.0", 9220}, // unknown version uses formula
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			if got := GetOpenSearchPort(tt.version); got != tt.expected {
				t.Errorf("GetOpenSearchPort(%v) = %v, want %v", tt.version, got, tt.expected)
			}
		})
	}
}

func TestGetElasticsearchPort(t *testing.T) {
	tests := []struct {
		version  string
		expected int
	}{
		{"7.6", 9646},
		{"7.17", 9657},
		{"8.0", 9660},
		{"8.11", 9671},
		{"8.17", 9677},
		{"8.11.3", 9671}, // patch version should be stripped
		{"9.0", 9680},    // unknown version uses formula
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			if got := GetElasticsearchPort(tt.version); got != tt.expected {
				t.Errorf("GetElasticsearchPort(%v) = %v, want %v", tt.version, got, tt.expected)
			}
		})
	}
}

func TestResolveElasticsearchVersion(t *testing.T) {
	cleanup := setupMockDockerHub(t, map[string][]string{
		"library/elasticsearch:7.17": {"7.17.27", "7.17.28", "7.17.26"},
		"library/elasticsearch:8.11": {"8.11.3", "8.11.4", "8.11.2"},
		"library/elasticsearch:8.17": {"8.17.4", "8.17.3"},
		"library/elasticsearch:7.6":  {"7.6.2", "7.6.1"},
	})
	defer cleanup()

	tests := []struct {
		version  string
		expected string
	}{
		// major.minor inputs should resolve to highest patch version
		{"7.17", "7.17.28"},
		{"8.11", "8.11.4"},
		{"8.17", "8.17.4"},
		{"7.6", "7.6.2"},
		// already full versions pass through unchanged (no Hub query)
		{"7.17.28", "7.17.28"},
		{"8.11.4", "8.11.4"},
		// unknown major.minor (not in mock) passes through unchanged
		{"9.0", "9.0"},
		{"10.5", "10.5"},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			if got := ResolveElasticsearchVersion(tt.version); got != tt.expected {
				t.Errorf("ResolveElasticsearchVersion(%v) = %v, want %v", tt.version, got, tt.expected)
			}
		})
	}
}

func TestResolveOpenSearchVersion(t *testing.T) {
	cleanup := setupMockDockerHub(t, map[string][]string{
		"opensearchproject/opensearch:2.19": {"2.19.1", "2.19.2", "2.19.0"},
		"opensearchproject/opensearch:1.3":  {"1.3.19", "1.3.20", "1.3.18"},
		"opensearchproject/opensearch:2.5":  {"2.5.0"},
		"opensearchproject/opensearch:3.0":  {"3.0.0"},
		"opensearchproject/opensearch:3.3":  {"3.3.0"},
	})
	defer cleanup()

	tests := []struct {
		version  string
		expected string
	}{
		// major.minor inputs should resolve to highest patch version
		{"2.19", "2.19.2"},
		{"1.3", "1.3.20"},
		{"2.5", "2.5.0"},
		{"3.0", "3.0.0"},
		{"3.3", "3.3.0"},
		// already full versions pass through unchanged (no Hub query)
		{"2.19.4", "2.19.4"},
		{"1.3.20", "1.3.20"},
		// unknown major.minor (not in mock) passes through unchanged
		{"4.0", "4.0"},
		{"5.1", "5.1"},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			if got := ResolveOpenSearchVersion(tt.version); got != tt.expected {
				t.Errorf("ResolveOpenSearchVersion(%v) = %v, want %v", tt.version, got, tt.expected)
			}
		})
	}
}

func TestComposeService_Elasticsearch_ImageResolvesVersion(t *testing.T) {
	cleanup := setupMockDockerHub(t, map[string][]string{
		"library/elasticsearch:7.17": {"7.17.26", "7.17.28", "7.17.27"},
	})
	defer cleanup()

	g, _ := setupTestComposeGenerator(t)

	// When user specifies major.minor only, image should use resolved full version
	svcCfg := &config.ServiceConfig{
		Enabled: true,
		Version: "7.17",
	}
	svc := g.getElasticsearchService(svcCfg, false)

	if svc.Image != "elasticsearch:7.17.28" {
		t.Errorf("Image = %v, want elasticsearch:7.17.28 (resolved full version)", svc.Image)
	}
	// Container name should still use the user-specified version
	if svc.ContainerName != "magebox-elasticsearch-7.17" {
		t.Errorf("ContainerName = %v, want magebox-elasticsearch-7.17", svc.ContainerName)
	}
}

func TestComposeService_OpenSearch_ImageResolvesVersion(t *testing.T) {
	cleanup := setupMockDockerHub(t, map[string][]string{
		"opensearchproject/opensearch:2.19": {"2.19.0", "2.19.2", "2.19.1"},
	})
	defer cleanup()

	g, _ := setupTestComposeGenerator(t)

	// When user specifies major.minor only, image should use resolved full version
	svcCfg := &config.ServiceConfig{
		Enabled: true,
		Version: "2.19",
	}
	svc := g.getOpenSearchService(svcCfg, false)

	if svc.Image != "opensearchproject/opensearch:2.19.2" {
		t.Errorf("Image = %v, want opensearchproject/opensearch:2.19.2 (resolved full version)", svc.Image)
	}
	// Container name should still use the user-specified version
	if svc.ContainerName != "magebox-opensearch-2.19" {
		t.Errorf("ContainerName = %v, want magebox-opensearch-2.19", svc.ContainerName)
	}
}

func TestResolveDockerTagVersion_FallbackOnHTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	prevBase := dockerHubAPIBase
	dockerHubAPIBase = ts.URL
	defer func() {
		dockerHubAPIBase = prevBase
		resolvedTags.Range(func(k, _ interface{}) bool {
			resolvedTags.Delete(k)
			return true
		})
	}()

	// Should fall back to the input version when the Hub returns an error
	got := ResolveElasticsearchVersion("8.11")
	if got != "8.11" {
		t.Errorf("ResolveElasticsearchVersion on HTTP error = %v, want 8.11 (unchanged)", got)
	}
}

func TestCompareVersionStrings(t *testing.T) {
	tests := []struct {
		a, b string
		want int // sign only: negative, zero, positive
	}{
		{"7.17.28", "7.17.27", 1},
		{"7.17.27", "7.17.28", -1},
		{"7.17.28", "7.17.28", 0},
		{"8.0.0", "7.17.28", 1},
		{"7.17.9", "7.17.28", -1}, // numeric vs string: 9 < 28
	}
	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			got := compareVersionStrings(tt.a, tt.b)
			switch {
			case tt.want > 0 && got <= 0:
				t.Errorf("compareVersionStrings(%q, %q) = %d, want > 0", tt.a, tt.b, got)
			case tt.want < 0 && got >= 0:
				t.Errorf("compareVersionStrings(%q, %q) = %d, want < 0", tt.a, tt.b, got)
			case tt.want == 0 && got != 0:
				t.Errorf("compareVersionStrings(%q, %q) = %d, want 0", tt.a, tt.b, got)
			}
		})
	}
}

func TestComposeService_Valkey(t *testing.T) {
	g, _ := setupTestComposeGenerator(t)

	svc := g.getValkeyService()

	if !strings.Contains(svc.Image, "valkey") {
		t.Errorf("Image = %v, should contain valkey", svc.Image)
	}
	if svc.ContainerName != "magebox-valkey" {
		t.Errorf("ContainerName = %v, want magebox-valkey", svc.ContainerName)
	}
	if len(svc.Ports) != 1 || svc.Ports[0] != "6379:6379" {
		t.Errorf("Ports = %v, want [6379:6379]", svc.Ports)
	}
	if svc.HealthCheck == nil {
		t.Error("HealthCheck should not be nil")
	}
	if svc.HealthCheck != nil && svc.HealthCheck.Test[1] != "valkey-cli" {
		t.Errorf("HealthCheck test = %v, should use valkey-cli", svc.HealthCheck.Test)
	}
}

func TestComposeGenerator_GenerateWithValkey(t *testing.T) {
	g, _ := setupTestComposeGenerator(t)

	configs := []*config.Config{
		{
			Name: "valkeyproject",
			Services: config.Services{
				MySQL:  &config.ServiceConfig{Enabled: true, Version: "8.0"},
				Valkey: &config.ServiceConfig{Enabled: true},
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

	// Should have Valkey, not Redis
	if _, ok := compose.Services["valkey"]; !ok {
		t.Error("Compose should contain valkey service")
	}
	if _, ok := compose.Services["redis"]; ok {
		t.Error("Compose should NOT contain redis service when valkey is configured")
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

func TestComposeService_PhpMyAdmin(t *testing.T) {
	g, _ := setupTestComposeGenerator(t)

	svc := g.getPhpMyAdminService("magebox-mysql-8.0", 8036)

	if svc.Image != "phpmyadmin:latest" {
		t.Errorf("Image = %v, want phpmyadmin:latest", svc.Image)
	}
	if svc.ContainerName != "magebox-phpmyadmin" {
		t.Errorf("ContainerName = %v, want magebox-phpmyadmin", svc.ContainerName)
	}
	if len(svc.Ports) != 1 || svc.Ports[0] != "8036:80" {
		t.Errorf("Ports = %v, want [8036:80]", svc.Ports)
	}
	if svc.Environment["PMA_ARBITRARY"] != "1" {
		t.Error("PMA_ARBITRARY should be 1")
	}
	if svc.Environment["PMA_HOST"] != "magebox-mysql-8.0" {
		t.Errorf("PMA_HOST = %v, want magebox-mysql-8.0", svc.Environment["PMA_HOST"])
	}
	if svc.Environment["PMA_USER"] != "root" {
		t.Error("PMA_USER should be root")
	}
	if svc.Environment["PMA_PASSWORD"] != DefaultDBRootPassword {
		t.Errorf("PMA_PASSWORD should be %s", DefaultDBRootPassword)
	}
	if svc.Restart != "unless-stopped" {
		t.Error("Restart should be unless-stopped")
	}
}

func TestComposeService_PhpMyAdmin_NoDBHost(t *testing.T) {
	g, _ := setupTestComposeGenerator(t)

	svc := g.getPhpMyAdminService("", 8036)

	if _, ok := svc.Environment["PMA_HOST"]; ok {
		t.Error("PMA_HOST should not be set when dbHost is empty")
	}
	if svc.Environment["PMA_ARBITRARY"] != "1" {
		t.Error("PMA_ARBITRARY should still be 1")
	}
}

func TestComposeService_PhpMyAdmin_CustomPort(t *testing.T) {
	g, _ := setupTestComposeGenerator(t)

	svc := g.getPhpMyAdminService("magebox-mysql-8.0", 9090)

	if len(svc.Ports) != 1 || svc.Ports[0] != "9090:80" {
		t.Errorf("Ports = %v, want [9090:80]", svc.Ports)
	}
}

func TestRequiredServices_firstDBHost(t *testing.T) {
	tests := []struct {
		name           string
		rs             requiredServices
		defaultMySQL   string
		defaultMariaDB string
		want           string
	}{
		{
			name: "prefers default MySQL version",
			rs: requiredServices{
				mysql:   map[string]*config.ServiceConfig{"5.7": {}, "8.0": {}},
				mariadb: map[string]*config.ServiceConfig{},
			},
			defaultMySQL: "8.0",
			want:         "magebox-mysql-8.0",
		},
		{
			name: "falls back to any MySQL when default not present",
			rs: requiredServices{
				mysql:   map[string]*config.ServiceConfig{"5.7": {}},
				mariadb: map[string]*config.ServiceConfig{},
			},
			defaultMySQL: "8.0",
			want:         "magebox-mysql-5.7",
		},
		{
			name: "prefers default MariaDB version",
			rs: requiredServices{
				mysql:   map[string]*config.ServiceConfig{},
				mariadb: map[string]*config.ServiceConfig{"10.4": {}, "10.6": {}},
			},
			defaultMariaDB: "10.6",
			want:           "magebox-mariadb-10.6",
		},
		{
			name: "MySQL takes priority over MariaDB",
			rs: requiredServices{
				mysql:   map[string]*config.ServiceConfig{"8.0": {}},
				mariadb: map[string]*config.ServiceConfig{"10.6": {}},
			},
			defaultMySQL:   "8.0",
			defaultMariaDB: "10.6",
			want:           "magebox-mysql-8.0",
		},
		{
			name: "returns empty when no databases",
			rs: requiredServices{
				mysql:   map[string]*config.ServiceConfig{},
				mariadb: map[string]*config.ServiceConfig{},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.rs.firstDBHost(tt.defaultMySQL, tt.defaultMariaDB)
			if got != tt.want {
				t.Errorf("firstDBHost() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestComposeGenerator_GenerateWithPhpMyAdmin(t *testing.T) {
	g, tmpDir := setupTestComposeGenerator(t)

	// Create global config with phpMyAdmin enabled
	configDir := filepath.Join(tmpDir, ".magebox")
	os.MkdirAll(configDir, 0755)
	os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("phpmyadmin: true\n"), 0644)

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

	content, err := os.ReadFile(g.ComposeFilePath())
	if err != nil {
		t.Fatalf("Failed to read compose file: %v", err)
	}

	var compose ComposeConfig
	if err := yaml.Unmarshal(content, &compose); err != nil {
		t.Fatalf("Failed to parse compose file: %v", err)
	}

	if _, ok := compose.Services["phpmyadmin"]; !ok {
		t.Error("Compose should contain phpmyadmin service when enabled in global config")
	}
}

func TestComposeGenerator_GenerateWithPhpMyAdminFromProject(t *testing.T) {
	g, _ := setupTestComposeGenerator(t)

	configs := []*config.Config{
		{
			Name: "project1",
			Services: config.Services{
				MySQL:      &config.ServiceConfig{Enabled: true, Version: "8.0"},
				PhpMyAdmin: &config.ServiceConfig{Enabled: true},
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

	if _, ok := compose.Services["phpmyadmin"]; !ok {
		t.Error("Compose should contain phpmyadmin service when enabled in project config")
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

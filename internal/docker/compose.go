package docker

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/qoliber/magebox/internal/config"
	"github.com/qoliber/magebox/internal/platform"
	"gopkg.in/yaml.v3"
)

// ComposeGenerator generates Docker Compose configurations for global services
type ComposeGenerator struct {
	platform   *platform.Platform
	composeDir string
}

// ComposeConfig represents a Docker Compose configuration
type ComposeConfig struct {
	Version  string                       `yaml:"version,omitempty"`
	Services map[string]ComposeService    `yaml:"services"`
	Networks map[string]ComposeNetwork    `yaml:"networks,omitempty"`
	Volumes  map[string]ComposeVolume     `yaml:"volumes,omitempty"`
}

// ComposeService represents a service in Docker Compose
type ComposeService struct {
	Image       string            `yaml:"image"`
	Ports       []string          `yaml:"ports,omitempty"`
	Environment map[string]string `yaml:"environment,omitempty"`
	Volumes     []string          `yaml:"volumes,omitempty"`
	Networks    []string          `yaml:"networks,omitempty"`
	Restart     string            `yaml:"restart,omitempty"`
	HealthCheck *HealthCheck      `yaml:"healthcheck,omitempty"`
	Command     string            `yaml:"command,omitempty"`
}

// ComposeNetwork represents a network in Docker Compose
type ComposeNetwork struct {
	Driver string `yaml:"driver,omitempty"`
}

// ComposeVolume represents a volume in Docker Compose
type ComposeVolume struct {
	Driver string `yaml:"driver,omitempty"`
}

// HealthCheck represents a health check configuration
type HealthCheck struct {
	Test     []string `yaml:"test"`
	Interval string   `yaml:"interval"`
	Timeout  string   `yaml:"timeout"`
	Retries  int      `yaml:"retries"`
}

// ServiceInfo contains information about a running service
type ServiceInfo struct {
	Name      string
	Image     string
	Port      int
	IsRunning bool
}

// NewComposeGenerator creates a new Docker Compose generator
func NewComposeGenerator(p *platform.Platform) *ComposeGenerator {
	return &ComposeGenerator{
		platform:   p,
		composeDir: filepath.Join(p.MageBoxDir(), "docker"),
	}
}

// GenerateGlobalServices generates the global docker-compose.yml for shared services
func (g *ComposeGenerator) GenerateGlobalServices(configs []*config.Config) error {
	if err := os.MkdirAll(g.composeDir, 0755); err != nil {
		return fmt.Errorf("failed to create compose directory: %w", err)
	}

	compose := ComposeConfig{
		Services: make(map[string]ComposeService),
		Networks: map[string]ComposeNetwork{
			"magebox": {Driver: "bridge"},
		},
		Volumes: make(map[string]ComposeVolume),
	}

	// Collect all required services from all projects
	requiredServices := g.collectRequiredServices(configs)

	// Add MySQL services
	for version := range requiredServices.mysql {
		serviceName := fmt.Sprintf("mysql%s", strings.ReplaceAll(version, ".", ""))
		compose.Services[serviceName] = g.getMySQLService(version)
		compose.Volumes[fmt.Sprintf("mysql%s_data", strings.ReplaceAll(version, ".", ""))] = ComposeVolume{}
	}

	// Add MariaDB services
	for version := range requiredServices.mariadb {
		serviceName := fmt.Sprintf("mariadb%s", strings.ReplaceAll(version, ".", ""))
		compose.Services[serviceName] = g.getMariaDBService(version)
		compose.Volumes[fmt.Sprintf("mariadb%s_data", strings.ReplaceAll(version, ".", ""))] = ComposeVolume{}
	}

	// Add Redis if needed
	if requiredServices.redis {
		compose.Services["redis"] = g.getRedisService()
	}

	// Add OpenSearch services
	for version := range requiredServices.opensearch {
		serviceName := fmt.Sprintf("opensearch%s", strings.ReplaceAll(version, ".", ""))
		compose.Services[serviceName] = g.getOpenSearchService(version)
		compose.Volumes[fmt.Sprintf("opensearch%s_data", strings.ReplaceAll(version, ".", ""))] = ComposeVolume{}
	}

	// Add Elasticsearch services
	for version := range requiredServices.elasticsearch {
		serviceName := fmt.Sprintf("elasticsearch%s", strings.ReplaceAll(version, ".", ""))
		compose.Services[serviceName] = g.getElasticsearchService(version)
		compose.Volumes[fmt.Sprintf("elasticsearch%s_data", strings.ReplaceAll(version, ".", ""))] = ComposeVolume{}
	}

	// Add RabbitMQ if needed
	if requiredServices.rabbitmq {
		compose.Services["rabbitmq"] = g.getRabbitMQService()
		compose.Volumes["rabbitmq_data"] = ComposeVolume{}
	}

	// Add Mailpit if needed
	if requiredServices.mailpit {
		compose.Services["mailpit"] = g.getMailpitService()
	}

	// Write compose file
	data, err := yaml.Marshal(compose)
	if err != nil {
		return fmt.Errorf("failed to marshal compose config: %w", err)
	}

	composeFile := filepath.Join(g.composeDir, "docker-compose.yml")
	if err := os.WriteFile(composeFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write compose file: %w", err)
	}

	return nil
}

// requiredServices tracks which services are needed
type requiredServices struct {
	mysql         map[string]bool
	mariadb       map[string]bool
	redis         bool
	opensearch    map[string]bool
	elasticsearch map[string]bool
	rabbitmq      bool
	mailpit       bool
}

// collectRequiredServices collects all required services from configs
func (g *ComposeGenerator) collectRequiredServices(configs []*config.Config) requiredServices {
	rs := requiredServices{
		mysql:         make(map[string]bool),
		mariadb:       make(map[string]bool),
		opensearch:    make(map[string]bool),
		elasticsearch: make(map[string]bool),
	}

	for _, cfg := range configs {
		if cfg.Services.HasMySQL() {
			rs.mysql[cfg.Services.MySQL.Version] = true
		}
		if cfg.Services.HasMariaDB() {
			rs.mariadb[cfg.Services.MariaDB.Version] = true
		}
		if cfg.Services.HasRedis() {
			rs.redis = true
		}
		if cfg.Services.HasOpenSearch() {
			rs.opensearch[cfg.Services.OpenSearch.Version] = true
		}
		if cfg.Services.HasElasticsearch() {
			rs.elasticsearch[cfg.Services.Elasticsearch.Version] = true
		}
		if cfg.Services.HasRabbitMQ() {
			rs.rabbitmq = true
		}
		if cfg.Services.HasMailpit() {
			rs.mailpit = true
		}
	}

	return rs
}

// getMySQLService returns a MySQL service configuration
func (g *ComposeGenerator) getMySQLService(version string) ComposeService {
	port := g.getMySQLPort(version)
	return ComposeService{
		Image: fmt.Sprintf("mysql:%s", version),
		Ports: []string{fmt.Sprintf("%d:3306", port)},
		Environment: map[string]string{
			"MYSQL_ROOT_PASSWORD": "magebox",
		},
		Volumes: []string{
			fmt.Sprintf("mysql%s_data:/var/lib/mysql", strings.ReplaceAll(version, ".", "")),
		},
		Networks: []string{"magebox"},
		Restart:  "unless-stopped",
		HealthCheck: &HealthCheck{
			Test:     []string{"CMD", "mysqladmin", "ping", "-h", "localhost", "-uroot", "-pmagebox"},
			Interval: "10s",
			Timeout:  "5s",
			Retries:  5,
		},
	}
}

// getMariaDBService returns a MariaDB service configuration
func (g *ComposeGenerator) getMariaDBService(version string) ComposeService {
	port := g.getMariaDBPort(version)
	return ComposeService{
		Image: fmt.Sprintf("mariadb:%s", version),
		Ports: []string{fmt.Sprintf("%d:3306", port)},
		Environment: map[string]string{
			"MYSQL_ROOT_PASSWORD": "magebox",
		},
		Volumes: []string{
			fmt.Sprintf("mariadb%s_data:/var/lib/mysql", strings.ReplaceAll(version, ".", "")),
		},
		Networks: []string{"magebox"},
		Restart:  "unless-stopped",
		HealthCheck: &HealthCheck{
			Test:     []string{"CMD", "healthcheck.sh", "--connect", "--innodb_initialized"},
			Interval: "10s",
			Timeout:  "5s",
			Retries:  5,
		},
	}
}

// getRedisService returns a Redis service configuration
func (g *ComposeGenerator) getRedisService() ComposeService {
	return ComposeService{
		Image:    "redis:7-alpine",
		Ports:    []string{"6379:6379"},
		Networks: []string{"magebox"},
		Restart:  "unless-stopped",
		HealthCheck: &HealthCheck{
			Test:     []string{"CMD", "redis-cli", "ping"},
			Interval: "10s",
			Timeout:  "5s",
			Retries:  5,
		},
	}
}

// getOpenSearchService returns an OpenSearch service configuration
func (g *ComposeGenerator) getOpenSearchService(version string) ComposeService {
	port := g.getOpenSearchPort(version)
	return ComposeService{
		Image: fmt.Sprintf("opensearchproject/opensearch:%s", version),
		Ports: []string{fmt.Sprintf("%d:9200", port)},
		Environment: map[string]string{
			"discovery.type":                         "single-node",
			"DISABLE_SECURITY_PLUGIN":                "true",
			"OPENSEARCH_JAVA_OPTS":                   "-Xms512m -Xmx512m",
			"cluster.routing.allocation.disk.threshold_enabled": "false",
		},
		Volumes: []string{
			fmt.Sprintf("opensearch%s_data:/usr/share/opensearch/data", strings.ReplaceAll(version, ".", "")),
		},
		Networks: []string{"magebox"},
		Restart:  "unless-stopped",
	}
}

// getElasticsearchService returns an Elasticsearch service configuration
func (g *ComposeGenerator) getElasticsearchService(version string) ComposeService {
	port := g.getElasticsearchPort(version)
	return ComposeService{
		Image: fmt.Sprintf("elasticsearch:%s", version),
		Ports: []string{fmt.Sprintf("%d:9200", port)},
		Environment: map[string]string{
			"discovery.type":        "single-node",
			"xpack.security.enabled": "false",
			"ES_JAVA_OPTS":          "-Xms512m -Xmx512m",
		},
		Volumes: []string{
			fmt.Sprintf("elasticsearch%s_data:/usr/share/elasticsearch/data", strings.ReplaceAll(version, ".", "")),
		},
		Networks: []string{"magebox"},
		Restart:  "unless-stopped",
	}
}

// getRabbitMQService returns a RabbitMQ service configuration
func (g *ComposeGenerator) getRabbitMQService() ComposeService {
	return ComposeService{
		Image: "rabbitmq:3-management-alpine",
		Ports: []string{
			"5672:5672",
			"15672:15672",
		},
		Environment: map[string]string{
			"RABBITMQ_DEFAULT_USER": "magebox",
			"RABBITMQ_DEFAULT_PASS": "magebox",
		},
		Volumes: []string{
			"rabbitmq_data:/var/lib/rabbitmq",
		},
		Networks: []string{"magebox"},
		Restart:  "unless-stopped",
	}
}

// getMailpitService returns a Mailpit service configuration
func (g *ComposeGenerator) getMailpitService() ComposeService {
	return ComposeService{
		Image: "axllent/mailpit:latest",
		Ports: []string{
			"1025:1025",
			"8025:8025",
		},
		Networks: []string{"magebox"},
		Restart:  "unless-stopped",
	}
}

// getPortainerService returns a Portainer service configuration
func (g *ComposeGenerator) getPortainerService() ComposeService {
	return ComposeService{
		Image: "portainer/portainer-ce:latest",
		Ports: []string{
			"9000:9000",
			"9443:9443",
		},
		Volumes: []string{
			"/var/run/docker.sock:/var/run/docker.sock",
			"portainer_data:/data",
		},
		Networks: []string{"magebox"},
		Restart:  "unless-stopped",
	}
}

// Port mapping functions to avoid conflicts
func (g *ComposeGenerator) getMySQLPort(version string) int {
	ports := map[string]int{
		"5.7": 33057,
		"8.0": 33080,
		"8.4": 33084,
	}
	if port, ok := ports[version]; ok {
		return port
	}
	return 33080 // default
}

func (g *ComposeGenerator) getMariaDBPort(version string) int {
	ports := map[string]int{
		"10.4": 33104,
		"10.5": 33105,
		"10.6": 33106,
		"10.11": 33111,
		"11.0": 33110,
		"11.4": 33114,
	}
	if port, ok := ports[version]; ok {
		return port
	}
	return 33106 // default
}

func (g *ComposeGenerator) getOpenSearchPort(version string) int {
	// Use 9200 for the first version, 9201, 9202, etc. for others
	return 9200
}

func (g *ComposeGenerator) getElasticsearchPort(version string) int {
	ports := map[string]int{
		"7.17": 9217,
		"8.0":  9280,
		"8.11": 9281,
	}
	if port, ok := ports[version]; ok {
		return port
	}
	return 9200
}

// ComposeDir returns the compose directory path
func (g *ComposeGenerator) ComposeDir() string {
	return g.composeDir
}

// ComposeFilePath returns the path to the docker-compose.yml file
func (g *ComposeGenerator) ComposeFilePath() string {
	return filepath.Join(g.composeDir, "docker-compose.yml")
}

// DockerController manages Docker Compose operations
type DockerController struct {
	composeFile string
}

// NewDockerController creates a new Docker controller
func NewDockerController(composeFile string) *DockerController {
	return &DockerController{composeFile: composeFile}
}

// Up starts all services
func (c *DockerController) Up() error {
	cmd := exec.Command("docker", "compose", "-f", c.composeFile, "up", "-d")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Down stops all services
func (c *DockerController) Down() error {
	cmd := exec.Command("docker", "compose", "-f", c.composeFile, "down")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// StartService starts a specific service
func (c *DockerController) StartService(serviceName string) error {
	cmd := exec.Command("docker", "compose", "-f", c.composeFile, "up", "-d", serviceName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// StopService stops a specific service
func (c *DockerController) StopService(serviceName string) error {
	cmd := exec.Command("docker", "compose", "-f", c.composeFile, "stop", serviceName)
	return cmd.Run()
}

// IsServiceRunning checks if a service is running
func (c *DockerController) IsServiceRunning(serviceName string) bool {
	cmd := exec.Command("docker", "compose", "-f", c.composeFile, "ps", "-q", serviceName)
	output, err := cmd.Output()
	return err == nil && len(strings.TrimSpace(string(output))) > 0
}

// Exec executes a command in a running container
func (c *DockerController) Exec(serviceName string, command ...string) error {
	args := append([]string{"compose", "-f", c.composeFile, "exec", serviceName}, command...)
	cmd := exec.Command("docker", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// CreateDatabase creates a database in the MySQL/MariaDB service
func (c *DockerController) CreateDatabase(serviceName, dbName string) error {
	cmd := exec.Command("docker", "compose", "-f", c.composeFile, "exec", "-T", serviceName,
		"mysql", "-uroot", "-pmagebox", "-e", fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", dbName))
	return cmd.Run()
}

// DatabaseExists checks if a database exists
func (c *DockerController) DatabaseExists(serviceName, dbName string) bool {
	cmd := exec.Command("docker", "compose", "-f", c.composeFile, "exec", "-T", serviceName,
		"mysql", "-uroot", "-pmagebox", "-e", fmt.Sprintf("SHOW DATABASES LIKE '%s'", dbName))
	output, err := cmd.Output()
	return err == nil && strings.Contains(string(output), dbName)
}

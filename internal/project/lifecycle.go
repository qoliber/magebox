package project

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/qoliber/magebox/internal/config"
	"github.com/qoliber/magebox/internal/dns"
	"github.com/qoliber/magebox/internal/docker"
	"github.com/qoliber/magebox/internal/nginx"
	"github.com/qoliber/magebox/internal/php"
	"github.com/qoliber/magebox/internal/platform"
	"github.com/qoliber/magebox/internal/ssl"
)

// Manager manages project lifecycle
type Manager struct {
	platform       *platform.Platform
	sslManager     *ssl.Manager
	vhostGenerator *nginx.VhostGenerator
	poolGenerator  *php.PoolGenerator
	composeGen     *docker.ComposeGenerator
	hostsManager   *dns.HostsManager
	phpDetector    *php.Detector
}

// NewManager creates a new project manager
func NewManager(p *platform.Platform) *Manager {
	sslMgr := ssl.NewManager(p)
	return &Manager{
		platform:       p,
		sslManager:     sslMgr,
		vhostGenerator: nginx.NewVhostGenerator(p, sslMgr),
		poolGenerator:  php.NewPoolGenerator(p),
		composeGen:     docker.NewComposeGenerator(p),
		hostsManager:   dns.NewHostsManager(p),
		phpDetector:    php.NewDetector(p),
	}
}

// StartResult contains the result of a start operation
type StartResult struct {
	Config      *config.Config
	ProjectPath string
	Domains     []string
	PHPVersion  string
	Services    []string
	Errors      []error
	Warnings    []string
}

// Start starts a project
func (m *Manager) Start(projectPath string) (*StartResult, error) {
	result := &StartResult{
		ProjectPath: projectPath,
		Errors:      make([]error, 0),
		Warnings:    make([]string, 0),
	}

	// Load configuration
	cfg, err := config.LoadFromPath(projectPath)
	if err != nil {
		return nil, err
	}
	result.Config = cfg
	result.PHPVersion = cfg.PHP

	// Extract domains
	for _, d := range cfg.Domains {
		result.Domains = append(result.Domains, d.Host)
	}

	// Check PHP version is installed
	if !m.phpDetector.IsVersionInstalled(cfg.PHP) {
		return nil, &PHPNotInstalledError{
			Version:  cfg.PHP,
			Platform: m.platform,
		}
	}

	// Generate SSL certificates
	if err := m.generateSSLCerts(cfg); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("SSL: %v", err))
	}

	// Generate PHP-FPM pool
	if err := m.poolGenerator.Generate(cfg.Name, cfg.PHP, cfg.Env, cfg.PHPINI); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("PHP-FPM pool: %w", err))
	}

	// Generate Nginx vhost
	if err := m.vhostGenerator.Generate(cfg, projectPath); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("nginx vhost: %w", err))
	}

	// Add domains to /etc/hosts only if using hosts mode (not dnsmasq)
	globalCfg, err := config.LoadGlobalConfig(m.platform.HomeDir)
	if err == nil && globalCfg.UseHosts() {
		if err := m.hostsManager.AddDomains(result.Domains); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("DNS: %v", err))
		}
	}

	// Generate and start Docker services
	if err := m.startDockerServices(cfg); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("docker: %w", err))
	}

	// Create database if needed
	if err := m.ensureDatabase(cfg); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Database: %v", err))
	}

	// Collect started services
	result.Services = m.getStartedServices(cfg)

	return result, nil
}

// Stop stops a project
func (m *Manager) Stop(projectPath string) error {
	cfg, err := config.LoadFromPath(projectPath)
	if err != nil {
		return err
	}

	// Remove Nginx vhost
	if err := m.vhostGenerator.Remove(cfg.Name); err != nil {
		return fmt.Errorf("failed to remove nginx vhost: %w", err)
	}

	// Remove PHP-FPM pool
	if err := m.poolGenerator.Remove(cfg.Name); err != nil {
		return fmt.Errorf("failed to remove php-fpm pool: %w", err)
	}

	// Remove domains from /etc/hosts only if using hosts mode (not dnsmasq)
	globalCfg, err := config.LoadGlobalConfig(m.platform.HomeDir)
	if err == nil && globalCfg.UseHosts() {
		domains := make([]string, 0, len(cfg.Domains))
		for _, d := range cfg.Domains {
			domains = append(domains, d.Host)
		}
		if err := m.hostsManager.RemoveDomains(domains); err != nil {
			return fmt.Errorf("failed to remove dns entries: %w", err)
		}
	}

	return nil
}

// Status returns the status of a project
func (m *Manager) Status(projectPath string) (*ProjectStatus, error) {
	cfg, err := config.LoadFromPath(projectPath)
	if err != nil {
		return nil, err
	}

	status := &ProjectStatus{
		Name:       cfg.Name,
		Path:       projectPath,
		PHPVersion: cfg.PHP,
		Domains:    make([]string, 0, len(cfg.Domains)),
		Services:   make(map[string]ServiceStatus),
	}

	for _, d := range cfg.Domains {
		status.Domains = append(status.Domains, d.Host)
	}

	// Check PHP-FPM
	fpmController := php.NewFPMController(m.platform, cfg.PHP)
	status.Services["php-fpm"] = ServiceStatus{
		Name:      fmt.Sprintf("PHP-FPM %s", cfg.PHP),
		IsRunning: fpmController.IsRunning(),
	}

	// Check Nginx
	nginxController := nginx.NewController(m.platform)
	status.Services["nginx"] = ServiceStatus{
		Name:      "Nginx",
		IsRunning: nginxController.IsRunning(),
	}

	// Check Docker services
	dockerController := docker.NewDockerController(m.composeGen.ComposeFilePath())
	if cfg.Services.HasMySQL() {
		serviceName := fmt.Sprintf("mysql%s", cfg.Services.MySQL.Version)
		status.Services["mysql"] = ServiceStatus{
			Name:      fmt.Sprintf("MySQL %s", cfg.Services.MySQL.Version),
			IsRunning: dockerController.IsServiceRunning(serviceName),
		}
	}
	if cfg.Services.HasRedis() {
		status.Services["redis"] = ServiceStatus{
			Name:      "Redis",
			IsRunning: dockerController.IsServiceRunning("redis"),
		}
	}
	if cfg.Services.HasOpenSearch() {
		status.Services["opensearch"] = ServiceStatus{
			Name:      fmt.Sprintf("OpenSearch %s", cfg.Services.OpenSearch.Version),
			IsRunning: dockerController.IsServiceRunning("opensearch"),
		}
	}

	return status, nil
}

// generateSSLCerts generates SSL certificates for all domains
func (m *Manager) generateSSLCerts(cfg *config.Config) error {
	for _, domain := range cfg.Domains {
		if domain.IsSSLEnabled() {
			baseDomain := ssl.ExtractBaseDomain(domain.Host)
			if !m.sslManager.CertExists(baseDomain) {
				if _, err := m.sslManager.GenerateCert(baseDomain); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// startDockerServices starts Docker services needed by the project
func (m *Manager) startDockerServices(cfg *config.Config) error {
	// Generate compose file with this project's requirements
	if err := m.composeGen.GenerateGlobalServices([]*config.Config{cfg}); err != nil {
		return err
	}

	// Start the services
	dockerController := docker.NewDockerController(m.composeGen.ComposeFilePath())
	return dockerController.Up()
}

// ensureDatabase creates the database if it doesn't exist
func (m *Manager) ensureDatabase(cfg *config.Config) error {
	dbService := cfg.Services.GetDatabaseService()
	if dbService == nil {
		return nil
	}

	dockerController := docker.NewDockerController(m.composeGen.ComposeFilePath())

	// Determine service name
	var serviceName string
	if cfg.Services.HasMySQL() {
		serviceName = fmt.Sprintf("mysql%s", cfg.Services.MySQL.Version)
	} else if cfg.Services.HasMariaDB() {
		serviceName = fmt.Sprintf("mariadb%s", cfg.Services.MariaDB.Version)
	}

	if serviceName == "" {
		return nil
	}

	// Wait for service to be healthy (simplified - in production would use proper health check)
	if !dockerController.IsServiceRunning(serviceName) {
		return fmt.Errorf("database service %s is not running", serviceName)
	}

	// Create database
	return dockerController.CreateDatabase(serviceName, cfg.Name)
}

// getStartedServices returns a list of started service names
func (m *Manager) getStartedServices(cfg *config.Config) []string {
	services := []string{
		fmt.Sprintf("PHP-FPM %s", cfg.PHP),
		"Nginx",
	}

	if cfg.Services.HasMySQL() {
		services = append(services, fmt.Sprintf("MySQL %s", cfg.Services.MySQL.Version))
	}
	if cfg.Services.HasMariaDB() {
		services = append(services, fmt.Sprintf("MariaDB %s", cfg.Services.MariaDB.Version))
	}
	if cfg.Services.HasRedis() {
		services = append(services, "Redis")
	}
	if cfg.Services.HasOpenSearch() {
		services = append(services, fmt.Sprintf("OpenSearch %s", cfg.Services.OpenSearch.Version))
	}
	if cfg.Services.HasElasticsearch() {
		services = append(services, fmt.Sprintf("Elasticsearch %s", cfg.Services.Elasticsearch.Version))
	}
	if cfg.Services.HasRabbitMQ() {
		services = append(services, "RabbitMQ")
	}
	if cfg.Services.HasMailpit() {
		services = append(services, "Mailpit")
	}

	return services
}

// ProjectStatus represents the status of a project
type ProjectStatus struct {
	Name       string
	Path       string
	PHPVersion string
	Domains    []string
	Services   map[string]ServiceStatus
}

// ServiceStatus represents the status of a service
type ServiceStatus struct {
	Name      string
	IsRunning bool
	Port      int
}

// PHPNotInstalledError indicates PHP is not installed
type PHPNotInstalledError struct {
	Version  string
	Platform *platform.Platform
}

func (e *PHPNotInstalledError) Error() string {
	return php.FormatNotInstalledMessage(e.Version, e.Platform)
}

// Init initializes a new .magebox.yaml file in the given directory
func (m *Manager) Init(projectPath string, projectName string) error {
	configPath := filepath.Join(projectPath, config.ConfigFileName)

	// Check if file already exists
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("%s file already exists", config.ConfigFileName)
	}

	// Derive domain from project name
	domain := projectName + ".test"

	content := fmt.Sprintf(`name: %s
domains:
  - host: %s
php: "8.2"
services:
  mysql: "8.0"
  redis: true
`, projectName, domain)

	return os.WriteFile(configPath, []byte(content), 0644)
}

// ValidateConfig validates a project configuration
func (m *Manager) ValidateConfig(projectPath string) (*config.Config, []string, error) {
	cfg, err := config.LoadFromPath(projectPath)
	if err != nil {
		return nil, nil, err
	}

	warnings := make([]string, 0)

	// Check if PHP version is installed
	if !m.phpDetector.IsVersionInstalled(cfg.PHP) {
		warnings = append(warnings, fmt.Sprintf("PHP %s is not installed", cfg.PHP))
	}

	// Check if mkcert is installed for SSL
	for _, domain := range cfg.Domains {
		if domain.IsSSLEnabled() && !m.sslManager.IsMkcertInstalled() {
			warnings = append(warnings, "mkcert is not installed, SSL certificates cannot be generated")
			break
		}
	}

	return cfg, warnings, nil
}

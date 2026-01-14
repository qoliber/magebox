package php

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"qoliber/magebox/internal/lib"
	"qoliber/magebox/internal/platform"
)

//go:embed templates/isolated-fpm.conf.tmpl
var isolatedFPMTemplateEmbed string

func init() {
	lib.RegisterFallbackTemplate(lib.TemplatePHP, "isolated-fpm.conf.tmpl", isolatedFPMTemplateEmbed)
}

// IsolatedProject represents an isolated PHP-FPM master for a project
type IsolatedProject struct {
	ProjectName string            `json:"project_name"`
	ProjectPath string            `json:"project_path"`
	PHPVersion  string            `json:"php_version"`
	SocketPath  string            `json:"socket_path"`
	PIDPath     string            `json:"pid_path"`
	ConfigPath  string            `json:"config_path"`
	Settings    map[string]string `json:"settings"`
	CreatedAt   time.Time         `json:"created_at"`
}

// IsolatedRegistry manages the registry of isolated PHP-FPM projects
type IsolatedRegistry struct {
	platform     *platform.Platform
	registryPath string
}

// NewIsolatedRegistry creates a new isolated registry
func NewIsolatedRegistry(p *platform.Platform) *IsolatedRegistry {
	return &IsolatedRegistry{
		platform:     p,
		registryPath: filepath.Join(p.MageBoxDir(), "isolated-projects.json"),
	}
}

// Load loads the registry from disk
func (r *IsolatedRegistry) Load() (map[string]*IsolatedProject, error) {
	projects := make(map[string]*IsolatedProject)

	data, err := os.ReadFile(r.registryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return projects, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, &projects); err != nil {
		return nil, err
	}

	return projects, nil
}

// Save saves the registry to disk
func (r *IsolatedRegistry) Save(projects map[string]*IsolatedProject) error {
	data, err := json.MarshalIndent(projects, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(r.registryPath, data, 0644)
}

// Get returns an isolated project by name
func (r *IsolatedRegistry) Get(projectName string) (*IsolatedProject, error) {
	projects, err := r.Load()
	if err != nil {
		return nil, err
	}

	project, ok := projects[projectName]
	if !ok {
		return nil, nil
	}

	return project, nil
}

// Add adds or updates an isolated project
func (r *IsolatedRegistry) Add(project *IsolatedProject) error {
	projects, err := r.Load()
	if err != nil {
		return err
	}

	projects[project.ProjectName] = project
	return r.Save(projects)
}

// Remove removes an isolated project
func (r *IsolatedRegistry) Remove(projectName string) error {
	projects, err := r.Load()
	if err != nil {
		return err
	}

	delete(projects, projectName)
	return r.Save(projects)
}

// List returns all isolated projects
func (r *IsolatedRegistry) List() ([]*IsolatedProject, error) {
	projects, err := r.Load()
	if err != nil {
		return nil, err
	}

	list := make([]*IsolatedProject, 0, len(projects))
	for _, p := range projects {
		list = append(list, p)
	}

	return list, nil
}

// IsolatedFPMController manages isolated PHP-FPM master processes
type IsolatedFPMController struct {
	platform *platform.Platform
	registry *IsolatedRegistry
}

// NewIsolatedFPMController creates a new isolated FPM controller
func NewIsolatedFPMController(p *platform.Platform) *IsolatedFPMController {
	return &IsolatedFPMController{
		platform: p,
		registry: NewIsolatedRegistry(p),
	}
}

// GetRegistry returns the isolated registry
func (c *IsolatedFPMController) GetRegistry() *IsolatedRegistry {
	return c.registry
}

// IsolatedFPMConfig contains config data for isolated PHP-FPM master
type IsolatedFPMConfig struct {
	ProjectName     string
	ProjectPath     string
	PHPVersion      string
	PIDPath         string
	ErrorLogPath    string
	SocketPath      string
	User            string
	Group           string
	MaxChildren     int
	StartServers    int
	MinSpareServers int
	MaxSpareServers int
	MaxRequests     int
	SystemSettings  map[string]string // PHP_INI_SYSTEM settings (opcache, preload, etc.)
	PoolSettings    map[string]string // PHP_INI_PERDIR settings
	Env             map[string]string
}

// getIsolatedSocketPath returns the socket path for an isolated project
func (c *IsolatedFPMController) getIsolatedSocketPath(projectName, phpVersion string) string {
	return filepath.Join(c.platform.MageBoxDir(), "run", fmt.Sprintf("%s-isolated-php%s.sock", projectName, phpVersion))
}

// getIsolatedPIDPath returns the PID file path for an isolated project
func (c *IsolatedFPMController) getIsolatedPIDPath(projectName, phpVersion string) string {
	return filepath.Join(c.platform.MageBoxDir(), "run", fmt.Sprintf("%s-isolated-php%s.pid", projectName, phpVersion))
}

// getIsolatedConfigPath returns the config file path for an isolated project
func (c *IsolatedFPMController) getIsolatedConfigPath(projectName, phpVersion string) string {
	return filepath.Join(c.platform.MageBoxDir(), "php", "isolated", fmt.Sprintf("%s-php%s.conf", projectName, phpVersion))
}

// getIsolatedLogPath returns the log file path for an isolated project
func (c *IsolatedFPMController) getIsolatedLogPath(projectName, phpVersion string) string {
	return filepath.Join(c.platform.MageBoxDir(), "logs", "php-fpm", fmt.Sprintf("%s-isolated-error.log", projectName))
}

// Enable enables isolation for a project
func (c *IsolatedFPMController) Enable(projectName, projectPath, phpVersion string, systemSettings map[string]string) (*IsolatedProject, error) {
	// Check if already isolated
	existing, err := c.registry.Get(projectName)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		// Update settings and restart
		existing.Settings = systemSettings
		existing.PHPVersion = phpVersion
		if err := c.Stop(projectName); err != nil {
			return nil, fmt.Errorf("failed to stop existing isolated master: %w", err)
		}
	}

	// Create isolated project entry
	project := &IsolatedProject{
		ProjectName: projectName,
		ProjectPath: projectPath,
		PHPVersion:  phpVersion,
		SocketPath:  c.getIsolatedSocketPath(projectName, phpVersion),
		PIDPath:     c.getIsolatedPIDPath(projectName, phpVersion),
		ConfigPath:  c.getIsolatedConfigPath(projectName, phpVersion),
		Settings:    systemSettings,
		CreatedAt:   time.Now(),
	}

	// Generate config and start
	if err := c.generateConfig(project); err != nil {
		return nil, fmt.Errorf("failed to generate isolated config: %w", err)
	}

	if err := c.start(project); err != nil {
		return nil, fmt.Errorf("failed to start isolated master: %w", err)
	}

	// Register in registry
	if err := c.registry.Add(project); err != nil {
		return nil, fmt.Errorf("failed to register isolated project: %w", err)
	}

	return project, nil
}

// Disable disables isolation for a project
func (c *IsolatedFPMController) Disable(projectName string) error {
	project, err := c.registry.Get(projectName)
	if err != nil {
		return err
	}
	if project == nil {
		return fmt.Errorf("project %s is not isolated", projectName)
	}

	// Stop the isolated master
	if err := c.Stop(projectName); err != nil {
		return fmt.Errorf("failed to stop isolated master: %w", err)
	}

	// Remove config file
	if err := os.Remove(project.ConfigPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove config: %w", err)
	}

	// Remove from registry
	if err := c.registry.Remove(projectName); err != nil {
		return fmt.Errorf("failed to unregister: %w", err)
	}

	return nil
}

// generateConfig generates the isolated PHP-FPM config file
func (c *IsolatedFPMController) generateConfig(project *IsolatedProject) error {
	// Ensure directories exist
	configDir := filepath.Dir(project.ConfigPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	runDir := filepath.Dir(project.PIDPath)
	if err := os.MkdirAll(runDir, 0755); err != nil {
		return err
	}

	logDir := filepath.Dir(c.getIsolatedLogPath(project.ProjectName, project.PHPVersion))
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	// Separate system vs pool settings
	systemSettings, poolSettings := SeparateSettings(project.Settings)

	cfg := IsolatedFPMConfig{
		ProjectName:     project.ProjectName,
		ProjectPath:     project.ProjectPath,
		PHPVersion:      project.PHPVersion,
		PIDPath:         project.PIDPath,
		ErrorLogPath:    c.getIsolatedLogPath(project.ProjectName, project.PHPVersion),
		SocketPath:      project.SocketPath,
		User:            getCurrentUser(),
		Group:           getCurrentGroup(),
		MaxChildren:     25,
		StartServers:    4,
		MinSpareServers: 2,
		MaxSpareServers: 6,
		MaxRequests:     1000,
		SystemSettings:  systemSettings,
		PoolSettings:    poolSettings,
		Env:             make(map[string]string),
	}

	// Load template
	tmplContent, err := lib.GetTemplate(lib.TemplatePHP, "isolated-fpm.conf.tmpl")
	if err != nil {
		return fmt.Errorf("failed to load isolated template: %w", err)
	}

	tmpl, err := template.New("isolated-fpm").Parse(tmplContent)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, cfg); err != nil {
		return err
	}

	return os.WriteFile(project.ConfigPath, buf.Bytes(), 0644)
}

// start starts an isolated PHP-FPM master
func (c *IsolatedFPMController) start(project *IsolatedProject) error {
	// Clean up stale socket
	_ = os.Remove(project.SocketPath)

	binary := c.platform.PHPFPMBinary(project.PHPVersion)
	if binary == "" {
		return fmt.Errorf("PHP-FPM binary not found for version %s", project.PHPVersion)
	}

	cmd := exec.Command(binary, "-y", project.ConfigPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start: %w\nOutput: %s", err, output)
	}

	return nil
}

// Stop stops an isolated PHP-FPM master
func (c *IsolatedFPMController) Stop(projectName string) error {
	project, err := c.registry.Get(projectName)
	if err != nil {
		return err
	}
	if project == nil {
		return nil // Not isolated, nothing to stop
	}

	pid, err := c.getPID(project.PIDPath)
	if err != nil {
		// Not running
		return nil
	}

	// Send SIGTERM
	cmd := exec.Command("kill", fmt.Sprintf("%d", pid))
	if err := cmd.Run(); err != nil {
		// Try SIGKILL
		cmd = exec.Command("kill", "-9", fmt.Sprintf("%d", pid))
		_ = cmd.Run()
	}

	// Remove PID file and socket
	_ = os.Remove(project.PIDPath)
	_ = os.Remove(project.SocketPath)

	return nil
}

// Restart restarts an isolated PHP-FPM master
func (c *IsolatedFPMController) Restart(projectName string) error {
	project, err := c.registry.Get(projectName)
	if err != nil {
		return err
	}
	if project == nil {
		return fmt.Errorf("project %s is not isolated", projectName)
	}

	if err := c.Stop(projectName); err != nil {
		return err
	}

	return c.start(project)
}

// IsRunning checks if an isolated master is running
func (c *IsolatedFPMController) IsRunning(projectName string) bool {
	project, err := c.registry.Get(projectName)
	if err != nil || project == nil {
		return false
	}

	pid, err := c.getPID(project.PIDPath)
	if err != nil {
		return false
	}

	cmd := exec.Command("kill", "-0", fmt.Sprintf("%d", pid))
	return cmd.Run() == nil
}

// IsIsolated checks if a project is isolated
func (c *IsolatedFPMController) IsIsolated(projectName string) bool {
	project, err := c.registry.Get(projectName)
	return err == nil && project != nil
}

// GetSocketPath returns the socket path for a project (isolated or shared)
func (c *IsolatedFPMController) GetSocketPath(projectName, phpVersion string) string {
	project, err := c.registry.Get(projectName)
	if err == nil && project != nil {
		return project.SocketPath
	}
	// Fall back to shared socket path
	return filepath.Join(c.platform.MageBoxDir(), "run", fmt.Sprintf("%s-php%s.sock", projectName, phpVersion))
}

// GetStatus returns status info for an isolated project
func (c *IsolatedFPMController) GetStatus(projectName string) (map[string]interface{}, error) {
	project, err := c.registry.Get(projectName)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, nil
	}

	status := map[string]interface{}{
		"project_name": project.ProjectName,
		"php_version":  project.PHPVersion,
		"socket_path":  project.SocketPath,
		"pid_path":     project.PIDPath,
		"config_path":  project.ConfigPath,
		"settings":     project.Settings,
		"created_at":   project.CreatedAt,
		"running":      c.IsRunning(projectName),
	}

	if c.IsRunning(projectName) {
		pid, _ := c.getPID(project.PIDPath)
		status["pid"] = pid
	}

	return status, nil
}

// getPID reads PID from file
func (c *IsolatedFPMController) getPID(pidPath string) (int, error) {
	data, err := os.ReadFile(pidPath)
	if err != nil {
		return 0, err
	}

	var pid int
	if _, err := fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &pid); err != nil {
		return 0, err
	}

	return pid, nil
}

// StartAllIsolated starts all registered isolated masters
func (c *IsolatedFPMController) StartAllIsolated() error {
	projects, err := c.registry.List()
	if err != nil {
		return err
	}

	for _, project := range projects {
		if !c.IsRunning(project.ProjectName) {
			if err := c.start(project); err != nil {
				return fmt.Errorf("failed to start %s: %w", project.ProjectName, err)
			}
		}
	}

	return nil
}

// StopAllIsolated stops all running isolated masters
func (c *IsolatedFPMController) StopAllIsolated() error {
	projects, err := c.registry.List()
	if err != nil {
		return err
	}

	for _, project := range projects {
		if err := c.Stop(project.ProjectName); err != nil {
			return fmt.Errorf("failed to stop %s: %w", project.ProjectName, err)
		}
	}

	return nil
}

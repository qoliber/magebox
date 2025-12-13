package php

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	"github.com/qoliber/magebox/internal/platform"
)

//go:embed templates/pool.conf.tmpl
var poolTemplate string

//go:embed templates/php-fpm.conf.tmpl
var fpmConfigTemplate string

//go:embed templates/mailpit-sendmail.sh
var mailpitSendmailScript string

// Mailpit SMTP configuration constants
const (
	MailpitSMTPHost = "127.0.0.1"
	MailpitSMTPPort = 1025
	MailpitWebHost  = "127.0.0.1"
	MailpitWebPort  = 8025
)

// Template variables available in pool.conf.tmpl:
// - ProjectName: Name of the project (e.g., "mystore")
// - PHPVersion: PHP version (e.g., "8.2")
// - SocketPath: Path to PHP-FPM socket (e.g., "/tmp/magebox/mystore-php8.2.sock")
// - LogPath: Path to PHP-FPM error log (e.g., "~/.magebox/logs/php-fpm/mystore-error.log")
// - User: System user running PHP-FPM (e.g., "jakub")
// - Group: System group running PHP-FPM (e.g., "staff")
// - MaxChildren: Maximum number of child processes
// - StartServers: Number of child processes created on startup
// - MinSpareServers: Minimum number of idle server processes
// - MaxSpareServers: Maximum number of idle server processes
// - MaxRequests: Number of requests each child process should execute before respawning
// - Env: Map of environment variables to set (e.g., {"MAGE_MODE": "developer"})
// - PHPINI: Map of PHP INI overrides (e.g., {"opcache.enable": "0"})

// PoolGenerator generates PHP-FPM pool configurations
type PoolGenerator struct {
	platform *platform.Platform
	poolsDir string
	runDir   string
}

// PoolConfig contains all data needed to generate a PHP-FPM pool
type PoolConfig struct {
	ProjectName     string
	PHPVersion      string
	SocketPath      string
	LogPath         string
	User            string
	Group           string
	MaxChildren     int
	StartServers    int
	MinSpareServers int
	MaxSpareServers int
	MaxRequests     int
	Env             map[string]string
	PHPINI          map[string]string
	HasMailpit      bool
	SendmailPath    string
}

// NewPoolGenerator creates a new pool generator
func NewPoolGenerator(p *platform.Platform) *PoolGenerator {
	return &PoolGenerator{
		platform: p,
		poolsDir: filepath.Join(p.MageBoxDir(), "php", "pools"),
		runDir:   "/tmp/magebox",
	}
}

// Generate generates a PHP-FPM pool configuration for a project
func (g *PoolGenerator) Generate(projectName, phpVersion string, env map[string]string, phpIni map[string]string, hasMailpit bool) error {
	// Ensure directories exist
	if err := os.MkdirAll(g.poolsDir, 0755); err != nil {
		return fmt.Errorf("failed to create pools directory: %w", err)
	}
	if err := os.MkdirAll(g.runDir, 0755); err != nil {
		return fmt.Errorf("failed to create run directory: %w", err)
	}

	// Create logs directory
	logsDir := filepath.Join(g.platform.MageBoxDir(), "logs", "php-fpm")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Setup Mailpit sendmail wrapper if Mailpit is enabled
	var sendmailPath string
	if hasMailpit {
		var err error
		sendmailPath, err = g.setupMailpitSendmail()
		if err != nil {
			return fmt.Errorf("failed to setup mailpit sendmail: %w", err)
		}

		// Add Mailpit environment variables
		if env == nil {
			env = make(map[string]string)
		}
		env["MAILPIT_HOST"] = MailpitSMTPHost
		env["MAILPIT_PORT"] = fmt.Sprintf("%d", MailpitSMTPPort)
	}

	cfg := PoolConfig{
		ProjectName:     projectName,
		PHPVersion:      phpVersion,
		SocketPath:      g.GetSocketPath(projectName, phpVersion),
		LogPath:         filepath.Join(logsDir, projectName+"-error.log"),
		User:            getCurrentUser(),
		Group:           getCurrentGroup(),
		MaxChildren:     10,
		StartServers:    2,
		MinSpareServers: 1,
		MaxSpareServers: 3,
		MaxRequests:     500,
		Env:             env,
		PHPINI:          phpIni,
		HasMailpit:      hasMailpit,
		SendmailPath:    sendmailPath,
	}

	content, err := g.renderPool(cfg)
	if err != nil {
		return fmt.Errorf("failed to render pool config: %w", err)
	}

	poolFile := filepath.Join(g.poolsDir, fmt.Sprintf("%s.conf", projectName))
	if err := os.WriteFile(poolFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write pool file: %w", err)
	}

	return nil
}

// setupMailpitSendmail creates the Mailpit sendmail wrapper script
func (g *PoolGenerator) setupMailpitSendmail() (string, error) {
	binDir := filepath.Join(g.platform.MageBoxDir(), "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create bin directory: %w", err)
	}

	sendmailPath := filepath.Join(binDir, "mailpit-sendmail")

	// Write the sendmail wrapper script
	if err := os.WriteFile(sendmailPath, []byte(mailpitSendmailScript), 0755); err != nil {
		return "", fmt.Errorf("failed to write mailpit-sendmail script: %w", err)
	}

	return sendmailPath, nil
}

// Remove removes the pool configuration for a project
func (g *PoolGenerator) Remove(projectName string) error {
	poolFile := filepath.Join(g.poolsDir, fmt.Sprintf("%s.conf", projectName))
	if err := os.Remove(poolFile); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// GetSocketPath returns the socket path for a project
func (g *PoolGenerator) GetSocketPath(projectName, phpVersion string) string {
	return filepath.Join(g.runDir, fmt.Sprintf("%s-php%s.sock", projectName, phpVersion))
}

// PoolsDir returns the pools directory path
func (g *PoolGenerator) PoolsDir() string {
	return g.poolsDir
}

// RunDir returns the run directory path
func (g *PoolGenerator) RunDir() string {
	return g.runDir
}

// ListPools returns all pool configuration files
func (g *PoolGenerator) ListPools() ([]string, error) {
	pattern := filepath.Join(g.poolsDir, "*.conf")
	return filepath.Glob(pattern)
}

// renderPool renders the pool template
func (g *PoolGenerator) renderPool(cfg PoolConfig) (string, error) {
	tmpl, err := template.New("pool").Parse(poolTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, cfg); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// getCurrentUser returns the current user
func getCurrentUser() string {
	user := os.Getenv("USER")
	if user == "" {
		user = os.Getenv("LOGNAME")
	}
	if user == "" {
		user = "www-data"
	}
	return user
}

// getCurrentGroup returns the current user's primary group
func getCurrentGroup() string {
	// On macOS, the group is typically "staff"
	// On Linux, it's usually the username or www-data
	user := getCurrentUser()
	if user == "www-data" {
		return "www-data"
	}

	// Check platform - on macOS use "staff"
	if runtime.GOOS == "darwin" {
		return "staff"
	}

	// On Linux, use the username as the group
	return user
}

// FPMConfig contains data for generating the master php-fpm.conf
type FPMConfig struct {
	PHPVersion   string
	PIDPath      string
	ErrorLogPath string
	PoolsDir     string
}

// FPMController manages PHP-FPM service for a specific version
type FPMController struct {
	platform *platform.Platform
	version  string
}

// NewFPMController creates a new PHP-FPM controller
func NewFPMController(p *platform.Platform, version string) *FPMController {
	return &FPMController{
		platform: p,
		version:  version,
	}
}

// getPIDPath returns the path to the PID file for this PHP version
func (c *FPMController) getPIDPath() string {
	return filepath.Join(c.platform.MageBoxDir(), "run", fmt.Sprintf("php-fpm-%s.pid", c.version))
}

// getConfigPath returns the path to the php-fpm.conf for this PHP version
func (c *FPMController) getConfigPath() string {
	return filepath.Join(c.platform.MageBoxDir(), "php", fmt.Sprintf("php-fpm-%s.conf", c.version))
}

// getErrorLogPath returns the path to the error log for this PHP version
func (c *FPMController) getErrorLogPath() string {
	return filepath.Join(c.platform.MageBoxDir(), "logs", "php-fpm", fmt.Sprintf("php%s-error.log", c.version))
}

// getPoolsDir returns the path to the pools directory
func (c *FPMController) getPoolsDir() string {
	return filepath.Join(c.platform.MageBoxDir(), "php", "pools")
}

// getBinaryPath returns the path to php-fpm binary for this version
func (c *FPMController) getBinaryPath() string {
	switch c.platform.Type {
	case platform.Darwin:
		// Check ARM Homebrew first
		armPath := fmt.Sprintf("/opt/homebrew/opt/php@%s/sbin/php-fpm", c.version)
		if _, err := os.Stat(armPath); err == nil {
			return armPath
		}
		// Fall back to Intel Homebrew
		return fmt.Sprintf("/usr/local/opt/php@%s/sbin/php-fpm", c.version)
	case platform.Linux:
		return fmt.Sprintf("/usr/sbin/php-fpm%s", c.version)
	}
	return "php-fpm"
}

// GenerateConfig generates the master php-fpm.conf for MageBox
func (c *FPMController) GenerateConfig() error {
	// Ensure directories exist
	configDir := filepath.Dir(c.getConfigPath())
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	runDir := filepath.Dir(c.getPIDPath())
	if err := os.MkdirAll(runDir, 0755); err != nil {
		return fmt.Errorf("failed to create run directory: %w", err)
	}

	logDir := filepath.Dir(c.getErrorLogPath())
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	poolsDir := c.getPoolsDir()
	if err := os.MkdirAll(poolsDir, 0755); err != nil {
		return fmt.Errorf("failed to create pools directory: %w", err)
	}

	cfg := FPMConfig{
		PHPVersion:   c.version,
		PIDPath:      c.getPIDPath(),
		ErrorLogPath: c.getErrorLogPath(),
		PoolsDir:     poolsDir,
	}

	tmpl, err := template.New("fpm").Parse(fpmConfigTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse fpm config template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, cfg); err != nil {
		return fmt.Errorf("failed to execute fpm config template: %w", err)
	}

	return os.WriteFile(c.getConfigPath(), buf.Bytes(), 0644)
}

// Reload reloads PHP-FPM configuration by sending SIGUSR2
func (c *FPMController) Reload() error {
	pid, err := c.getPID()
	if err != nil {
		// Not running, start instead
		return c.Start()
	}

	// Send SIGUSR2 to reload
	cmd := exec.Command("kill", "-USR2", fmt.Sprintf("%d", pid))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reload php-fpm: %w", err)
	}
	return nil
}

// Start starts the dedicated MageBox PHP-FPM process
func (c *FPMController) Start() error {
	// Check if already running
	if c.IsRunning() {
		return nil
	}

	// Generate config
	if err := c.GenerateConfig(); err != nil {
		return fmt.Errorf("failed to generate php-fpm config: %w", err)
	}

	// Start php-fpm with our custom config
	binary := c.getBinaryPath()
	configPath := c.getConfigPath()

	cmd := exec.Command(binary, "-y", configPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start php-fpm: %w\nOutput: %s", err, output)
	}

	return nil
}

// Stop stops the MageBox PHP-FPM process
func (c *FPMController) Stop() error {
	pid, err := c.getPID()
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

	// Remove PID file
	_ = os.Remove(c.getPIDPath())
	return nil
}

// IsRunning checks if MageBox PHP-FPM is running for this version
func (c *FPMController) IsRunning() bool {
	pid, err := c.getPID()
	if err != nil {
		return false
	}

	// Check if process exists
	cmd := exec.Command("kill", "-0", fmt.Sprintf("%d", pid))
	return cmd.Run() == nil
}

// getPID reads the PID from the PID file
func (c *FPMController) getPID() (int, error) {
	pidPath := c.getPIDPath()
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

// GetIncludeDirective returns the include path for the pools directory
func (g *PoolGenerator) GetIncludeDirective() string {
	return g.poolsDir + "/*.conf"
}

// SetupFPMConfig creates symlinks from PHP-FPM conf.d directories to MageBox pools
func (g *PoolGenerator) SetupFPMConfig(version string) error {
	// Determine PHP-FPM conf.d directory based on platform and version
	var fpmConfDir string
	switch g.platform.Type {
	case platform.Darwin:
		// Check Homebrew ARM first, then Intel
		armPath := fmt.Sprintf("/opt/homebrew/etc/php/%s/php-fpm.d", version)
		intelPath := fmt.Sprintf("/usr/local/etc/php/%s/php-fpm.d", version)
		if _, err := os.Stat(armPath); err == nil {
			fpmConfDir = armPath
		} else {
			fpmConfDir = intelPath
		}
	case platform.Linux:
		fpmConfDir = fmt.Sprintf("/etc/php/%s/fpm/pool.d", version)
	default:
		return fmt.Errorf("unsupported platform")
	}

	// Ensure PHP-FPM conf.d directory exists
	if _, err := os.Stat(fpmConfDir); os.IsNotExist(err) {
		return fmt.Errorf("PHP-FPM config directory not found: %s", fpmConfDir)
	}

	// Create symlink from PHP-FPM conf.d to our pools directory
	symlinkPath := filepath.Join(fpmConfDir, "magebox")

	// Check if symlink already exists
	if linkTarget, err := os.Readlink(symlinkPath); err == nil {
		// Symlink exists, check if it points to the right place
		if linkTarget == g.poolsDir {
			return nil // Already configured correctly
		}
		// Remove old symlink
		cmd := exec.Command("sudo", "rm", symlinkPath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to remove old PHP-FPM symlink: %w", err)
		}
	}

	// Create symlink with sudo
	cmd := exec.Command("sudo", "ln", "-s", g.poolsDir, symlinkPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create PHP-FPM symlink: %w", err)
	}

	return nil
}

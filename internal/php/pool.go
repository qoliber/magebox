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

	"qoliber/magebox/internal/lib"
	"qoliber/magebox/internal/platform"
)

//go:embed templates/pool.conf.tmpl
var poolTemplateEmbed string

//go:embed templates/php-fpm.conf.tmpl
var fpmConfigTemplateEmbed string

//go:embed templates/mailpit-sendmail.sh
var mailpitSendmailScriptEmbed string

func init() {
	// Register embedded templates as fallbacks
	lib.RegisterFallbackTemplate(lib.TemplatePHP, "pool.conf.tmpl", poolTemplateEmbed)
	lib.RegisterFallbackTemplate(lib.TemplatePHP, "php-fpm.conf.tmpl", fpmConfigTemplateEmbed)
	lib.RegisterFallbackTemplate(lib.TemplatePHP, "mailpit-sendmail.sh", mailpitSendmailScriptEmbed)
}

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
	platform     *platform.Platform
	basePoolsDir string // Base pools directory (version subdirs will be created)
	runDir       string
}

// PoolConfig contains all data needed to generate a PHP-FPM pool
type PoolConfig struct {
	ProjectName     string
	ProjectPath     string
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

// defaultPHPINI returns the default PHP INI settings for Magento
func defaultPHPINI() map[string]string {
	return map[string]string{
		// OPcache settings
		"opcache.enable":                  "1",
		"opcache.memory_consumption":      "512",
		"opcache.max_accelerated_files":   "130986",
		"opcache.validate_timestamps":     "1",
		"opcache.consistency_checks":      "0",
		"opcache.interned_strings_buffer": "20",
		// Realpath cache
		"realpath_cache_size": "10M",
		"realpath_cache_ttl":  "7200",
	}
}

// mergePHPINI merges custom PHPINI settings over defaults
// Custom settings override defaults
func mergePHPINI(custom map[string]string) map[string]string {
	merged := defaultPHPINI()
	for k, v := range custom {
		merged[k] = v
	}
	return merged
}

// GetMergedPHPINI returns the merged PHP INI settings (defaults + custom)
// Exported for use by CLI commands
func GetMergedPHPINI(custom map[string]string) map[string]string {
	return mergePHPINI(custom)
}

// GetDefaultPHPINI returns the default PHP INI settings
// Exported for use by CLI commands
func GetDefaultPHPINI() map[string]string {
	return defaultPHPINI()
}

// NewPoolGenerator creates a new pool generator
func NewPoolGenerator(p *platform.Platform) *PoolGenerator {
	return &PoolGenerator{
		platform:     p,
		basePoolsDir: filepath.Join(p.MageBoxDir(), "php", "pools"),
		runDir:       filepath.Join(p.MageBoxDir(), "run"),
	}
}

// getVersionPoolsDir returns the version-specific pools directory
func (g *PoolGenerator) getVersionPoolsDir(phpVersion string) string {
	return filepath.Join(g.basePoolsDir, phpVersion)
}

// Generate generates a PHP-FPM pool configuration for a project
func (g *PoolGenerator) Generate(projectName, projectPath, phpVersion string, env map[string]string, phpIni map[string]string, hasMailpit bool) error {
	// Ensure version-specific pools directory exists
	versionPoolsDir := g.getVersionPoolsDir(phpVersion)
	if err := os.MkdirAll(versionPoolsDir, 0755); err != nil {
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
		ProjectPath:     projectPath,
		PHPVersion:      phpVersion,
		SocketPath:      g.GetSocketPath(projectName, phpVersion),
		LogPath:         filepath.Join(logsDir, projectName+"-error.log"),
		User:            getCurrentUser(),
		Group:           getCurrentGroup(),
		MaxChildren:     25,
		StartServers:    4,
		MinSpareServers: 2,
		MaxSpareServers: 6,
		MaxRequests:     1000,
		Env:             env,
		PHPINI:          mergePHPINI(phpIni),
		HasMailpit:      hasMailpit,
		SendmailPath:    sendmailPath,
	}

	content, err := g.renderPool(cfg)
	if err != nil {
		return fmt.Errorf("failed to render pool config: %w", err)
	}

	poolFile := filepath.Join(versionPoolsDir, fmt.Sprintf("%s.conf", projectName))
	if err := os.WriteFile(poolFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write pool file: %w", err)
	}

	// Clean up old pool configs from other PHP versions (project can only use one version)
	if err := g.removeOldVersionPools(projectName, phpVersion); err != nil {
		// Log but don't fail - old pools won't cause issues, just clutter
		fmt.Printf("[WARN] Failed to remove old version pool configs: %v\n", err)
	}

	return nil
}

// removeOldVersionPools removes pool configs for a project from other PHP version directories
func (g *PoolGenerator) removeOldVersionPools(projectName, currentVersion string) error {
	entries, err := os.ReadDir(g.basePoolsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	poolFileName := fmt.Sprintf("%s.conf", projectName)
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == currentVersion {
			continue
		}
		oldPoolFile := filepath.Join(g.basePoolsDir, entry.Name(), poolFileName)
		if _, err := os.Stat(oldPoolFile); err == nil {
			if err := os.Remove(oldPoolFile); err != nil {
				return err
			}
		}
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

	// Load sendmail script from lib (with embedded fallback)
	script, err := lib.GetTemplate(lib.TemplatePHP, "mailpit-sendmail.sh")
	if err != nil {
		return "", fmt.Errorf("failed to load mailpit-sendmail script: %w", err)
	}

	// Write the sendmail wrapper script
	if err := os.WriteFile(sendmailPath, []byte(script), 0755); err != nil {
		return "", fmt.Errorf("failed to write mailpit-sendmail script: %w", err)
	}

	return sendmailPath, nil
}

// Remove removes the pool configuration for a project from all version directories
func (g *PoolGenerator) Remove(projectName string) error {
	entries, err := os.ReadDir(g.basePoolsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	poolFileName := fmt.Sprintf("%s.conf", projectName)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		poolFile := filepath.Join(g.basePoolsDir, entry.Name(), poolFileName)
		if err := os.Remove(poolFile); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// GetSocketPath returns the socket path for a project
func (g *PoolGenerator) GetSocketPath(projectName, phpVersion string) string {
	return filepath.Join(g.runDir, fmt.Sprintf("%s-php%s.sock", projectName, phpVersion))
}

// PoolsDir returns the base pools directory path
func (g *PoolGenerator) PoolsDir() string {
	return g.basePoolsDir
}

// PoolsDirForVersion returns the version-specific pools directory path
func (g *PoolGenerator) PoolsDirForVersion(phpVersion string) string {
	return g.getVersionPoolsDir(phpVersion)
}

// RunDir returns the run directory path
func (g *PoolGenerator) RunDir() string {
	return g.runDir
}

// ListPools returns all pool configuration files across all versions
func (g *PoolGenerator) ListPools() ([]string, error) {
	pattern := filepath.Join(g.basePoolsDir, "*", "*.conf")
	return filepath.Glob(pattern)
}

// ListPoolsForVersion returns pool configuration files for a specific PHP version
func (g *PoolGenerator) ListPoolsForVersion(phpVersion string) ([]string, error) {
	pattern := filepath.Join(g.getVersionPoolsDir(phpVersion), "*.conf")
	return filepath.Glob(pattern)
}

// renderPool renders the pool template
func (g *PoolGenerator) renderPool(cfg PoolConfig) (string, error) {
	// Load template from lib (with embedded fallback)
	tmplContent, err := lib.GetTemplate(lib.TemplatePHP, "pool.conf.tmpl")
	if err != nil {
		return "", fmt.Errorf("failed to load pool template: %w", err)
	}

	tmpl, err := template.New("pool").Parse(tmplContent)
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

// useSystemd returns true if we should use systemd for PHP-FPM management
// NOTE: On Fedora/RHEL with SELinux, using systemd causes PHP-FPM to run with
// httpd_t context which restricts access to user home directories. For this
// reason, we disable systemd on Fedora and use direct process control.
// Debian/Ubuntu use AppArmor which doesn't have this restriction.
func (c *FPMController) useSystemd() bool {
	if runtime.GOOS != "linux" {
		return false
	}

	// Fedora/RHEL: SELinux httpd_t context blocks access to user_home_t
	// Use direct process management instead
	if c.platform.LinuxDistro == platform.DistroFedora {
		return false
	}

	// Arch: single PHP version, direct management is simpler
	if c.platform.LinuxDistro == platform.DistroArch {
		return false
	}

	// Debian/Ubuntu: AppArmor is permissive, systemd works fine
	if c.platform.LinuxDistro == platform.DistroDebian {
		// Check if systemctl exists and service is available
		if _, err := exec.LookPath("systemctl"); err != nil {
			return false
		}

		serviceName := c.getSystemdServiceName()
		cmd := exec.Command("systemctl", "list-unit-files", serviceName+".service", "--no-pager")
		output, err := cmd.Output()
		if err != nil {
			return false
		}
		return strings.Contains(string(output), serviceName)
	}

	// Unknown distro: use direct management to be safe
	return false
}

// getSystemdServiceName returns the systemd service name for this PHP version
func (c *FPMController) getSystemdServiceName() string {
	versionNoDot := strings.ReplaceAll(c.version, ".", "")

	switch c.platform.LinuxDistro {
	case platform.DistroFedora:
		// Remi: php83-php-fpm
		return fmt.Sprintf("php%s-php-fpm", versionNoDot)
	case platform.DistroArch:
		// Arch: php-fpm (single version)
		return "php-fpm"
	default:
		// Debian/Ubuntu: php8.3-fpm
		return fmt.Sprintf("php%s-fpm", c.version)
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
	return filepath.Join(c.platform.MageBoxDir(), "php", "pools", c.version)
}

// getBinaryPath returns the path to php-fpm binary for this version
func (c *FPMController) getBinaryPath() string {
	// Use platform's distribution-aware path detection
	return c.platform.PHPFPMBinary(c.version)
}

// cleanupStaleSockets removes stale socket files for this PHP version
// This prevents "Another FPM instance seems to already listen" errors when
// PHP-FPM was killed without cleaning up its sockets
func (c *FPMController) cleanupStaleSockets() {
	runDir := filepath.Join(c.platform.MageBoxDir(), "run")
	pattern := filepath.Join(runDir, fmt.Sprintf("*-php%s.sock", c.version))

	sockets, err := filepath.Glob(pattern)
	if err != nil {
		return
	}

	for _, sock := range sockets {
		_ = os.Remove(sock)
	}
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

	// Load template from lib (with embedded fallback)
	tmplContent, err := lib.GetTemplate(lib.TemplatePHP, "php-fpm.conf.tmpl")
	if err != nil {
		return fmt.Errorf("failed to load fpm config template: %w", err)
	}

	tmpl, err := template.New("fpm").Parse(tmplContent)
	if err != nil {
		return fmt.Errorf("failed to parse fpm config template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, cfg); err != nil {
		return fmt.Errorf("failed to execute fpm config template: %w", err)
	}

	return os.WriteFile(c.getConfigPath(), buf.Bytes(), 0644)
}

// Reload reloads PHP-FPM configuration
func (c *FPMController) Reload() error {
	if c.useSystemd() {
		// Use systemctl reload on Linux
		serviceName := c.getSystemdServiceName()
		cmd := exec.Command("sudo", "systemctl", "reload", serviceName)
		if output, err := cmd.CombinedOutput(); err != nil {
			// If reload fails, try restart
			cmd = exec.Command("sudo", "systemctl", "restart", serviceName)
			if restartOutput, restartErr := cmd.CombinedOutput(); restartErr != nil {
				return fmt.Errorf("failed to reload php-fpm: %w\nReload output: %s\nRestart output: %s", restartErr, output, restartOutput)
			}
		}
		return nil
	}

	// macOS: use PID-based reload
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

// Start starts PHP-FPM service
func (c *FPMController) Start() error {
	if c.useSystemd() {
		// Use systemctl start on Linux
		serviceName := c.getSystemdServiceName()

		// Check if already running
		if c.IsRunning() {
			return nil
		}

		cmd := exec.Command("sudo", "systemctl", "start", serviceName)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to start php-fpm (%s): %w\nOutput: %s", serviceName, err, output)
		}
		return nil
	}

	// macOS: start dedicated MageBox PHP-FPM process
	if c.IsRunning() {
		return nil
	}

	// Generate config
	if err := c.GenerateConfig(); err != nil {
		return fmt.Errorf("failed to generate php-fpm config: %w", err)
	}

	// Clean up stale socket files before starting
	// This prevents "Another FPM instance seems to already listen" errors
	c.cleanupStaleSockets()

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

// Stop stops PHP-FPM service (Note: on Linux with systemd, we don't stop the shared service)
func (c *FPMController) Stop() error {
	if c.useSystemd() {
		// On Linux, we don't stop the systemd service because it's shared
		// Just reload to unload the removed pool
		if c.IsRunning() {
			serviceName := c.getSystemdServiceName()
			cmd := exec.Command("sudo", "systemctl", "reload", serviceName)
			_ = cmd.Run() // Ignore errors, pool file is already removed
		}
		return nil
	}

	// macOS: stop dedicated process
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

// IsRunning checks if PHP-FPM is running for this version
func (c *FPMController) IsRunning() bool {
	if c.useSystemd() {
		// Use systemctl is-active on Linux
		serviceName := c.getSystemdServiceName()
		cmd := exec.Command("systemctl", "is-active", "--quiet", serviceName)
		return cmd.Run() == nil
	}

	// macOS: check PID file
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

// GetIncludeDirective returns the include path for all pools (deprecated - use GetIncludeDirectiveForVersion)
func (g *PoolGenerator) GetIncludeDirective() string {
	return g.basePoolsDir + "/*/*.conf"
}

// GetIncludeDirectiveForVersion returns the include path for version-specific pools
func (g *PoolGenerator) GetIncludeDirectiveForVersion(phpVersion string) string {
	return g.getVersionPoolsDir(phpVersion) + "/*.conf"
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

	// Create symlink from PHP-FPM conf.d to our version-specific pools directory
	symlinkPath := filepath.Join(fpmConfDir, "magebox")
	versionPoolsDir := g.getVersionPoolsDir(version)

	// Ensure version pools directory exists
	if err := os.MkdirAll(versionPoolsDir, 0755); err != nil {
		return fmt.Errorf("failed to create version pools directory: %w", err)
	}

	// Check if symlink already exists
	if linkTarget, err := os.Readlink(symlinkPath); err == nil {
		// Symlink exists, check if it points to the right place
		if linkTarget == versionPoolsDir {
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
	cmd := exec.Command("sudo", "ln", "-s", versionPoolsDir, symlinkPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create PHP-FPM symlink: %w", err)
	}

	return nil
}

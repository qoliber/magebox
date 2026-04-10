// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package tideways

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"qoliber/magebox/internal/platform"
	"qoliber/magebox/internal/xdebug"
)

// Manager handles Tideways installation and configuration
type Manager struct {
	platform    *platform.Platform
	xdebugMgr   *xdebug.Manager
	credentials *Credentials
}

// NewManager creates a new Tideways manager
func NewManager(p *platform.Platform, creds *Credentials) *Manager {
	return &Manager{
		platform:    p,
		xdebugMgr:   xdebug.NewManager(p),
		credentials: creds,
	}
}

// IsDaemonInstalled checks if Tideways daemon is installed
func (m *Manager) IsDaemonInstalled() bool {
	return platform.CommandExists("tideways-daemon")
}

// IsDaemonRunning checks if Tideways daemon is running
func (m *Manager) IsDaemonRunning() bool {
	switch m.platform.Type {
	case platform.Darwin:
		cmd := exec.Command("pgrep", "-f", "tideways-daemon")
		return cmd.Run() == nil
	case platform.Linux:
		cmd := exec.Command("systemctl", "is-active", "tideways-daemon")
		output, err := cmd.Output()
		if err != nil {
			return false
		}
		return strings.TrimSpace(string(output)) == "active"
	}
	return false
}

// IsExtensionInstalled checks if Tideways PHP extension is installed for a version
func (m *Manager) IsExtensionInstalled(phpVersion string) bool {
	iniPath := m.getExtensionIniPath(phpVersion)
	if iniPath == "" {
		return false
	}
	_, err := os.Stat(iniPath)
	return err == nil
}

// IsExtensionEnabled checks if Tideways PHP extension is enabled for a version
func (m *Manager) IsExtensionEnabled(phpVersion string) bool {
	iniPath := m.getExtensionIniPath(phpVersion)
	if iniPath == "" {
		return false
	}

	content, err := os.ReadFile(iniPath)
	if err != nil {
		return false
	}

	// Check if extension line is not commented out
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "extension=tideways") {
			return true
		}
	}
	return false
}

// GetStatus returns the current Tideways status
func (m *Manager) GetStatus(phpVersions []string) *Status {
	status := &Status{
		DaemonInstalled:    m.IsDaemonInstalled(),
		DaemonRunning:      m.IsDaemonRunning(),
		ExtensionInstalled: make(map[string]bool),
		ExtensionEnabled:   make(map[string]bool),
	}

	for _, v := range phpVersions {
		status.ExtensionInstalled[v] = m.IsExtensionInstalled(v)
		status.ExtensionEnabled[v] = m.IsExtensionEnabled(v)
	}

	return status
}

// Enable enables Tideways for a PHP version (and disables Xdebug)
func (m *Manager) Enable(phpVersion string) error {
	if !m.IsExtensionInstalled(phpVersion) {
		return fmt.Errorf("tideways extension not installed for PHP %s", phpVersion)
	}

	// Disable Xdebug first to avoid conflicts (non-fatal if it fails)
	if m.xdebugMgr.IsEnabled(phpVersion) {
		_ = m.xdebugMgr.Disable(phpVersion) // Ignore errors - xdebug may not have ini file
	}

	iniPath := m.getExtensionIniPath(phpVersion)

	// Use sudo sed to uncomment extension line (same approach as xdebug manager)
	args := m.sedInPlaceArgs(`s/^;\(extension=tideways.*\)$/\1/`, iniPath)
	cmd := exec.Command("sudo", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable tideways: %w", err)
	}

	return nil
}

// Disable disables Tideways for a PHP version
func (m *Manager) Disable(phpVersion string) error {
	iniPath := m.getExtensionIniPath(phpVersion)
	if iniPath == "" {
		return nil
	}

	// Use sudo sed to comment out extension line (same approach as xdebug manager)
	args := m.sedInPlaceArgs(`s/^\(extension=tideways.*\)$/;\1/`, iniPath)
	cmd := exec.Command("sudo", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to disable tideways: %w", err)
	}

	return nil
}

// StartDaemon starts the Tideways daemon
func (m *Manager) StartDaemon() error {
	switch m.platform.Type {
	case platform.Darwin:
		cmd := exec.Command("brew", "services", "start", "tideways-daemon")
		return cmd.Run()
	case platform.Linux:
		cmd := exec.Command("sudo", "systemctl", "start", "tideways-daemon")
		return cmd.Run()
	}
	return fmt.Errorf("unsupported platform")
}

// StopDaemon stops the Tideways daemon
func (m *Manager) StopDaemon() error {
	switch m.platform.Type {
	case platform.Darwin:
		cmd := exec.Command("brew", "services", "stop", "tideways-daemon")
		return cmd.Run()
	case platform.Linux:
		cmd := exec.Command("sudo", "systemctl", "stop", "tideways-daemon")
		return cmd.Run()
	}
	return fmt.Errorf("unsupported platform")
}

// RestartDaemon restarts the Tideways daemon
func (m *Manager) RestartDaemon() error {
	switch m.platform.Type {
	case platform.Darwin:
		cmd := exec.Command("brew", "services", "restart", "tideways-daemon")
		return cmd.Run()
	case platform.Linux:
		cmd := exec.Command("sudo", "systemctl", "restart", "tideways-daemon")
		return cmd.Run()
	}
	return fmt.Errorf("unsupported platform")
}

// CleanLegacyExtensionDirectives strips any stale `tideways.api_key` and
// `tideways.environment` lines from the PHP extension ini file for the
// given PHP version. These were written by earlier MageBox versions:
//
//   - tideways.api_key: was written globally, but the Tideways API key is
//     per Tideways project, so it now lives in each project's .magebox.yaml
//     under php_ini.tideways.api_key (rendered into the FPM pool config).
//   - tideways.environment: is not a real PHP extension directive at all —
//     environment is a daemon-level setting applied via systemd drop-in.
//
// Both stale lines are harmless (the extension ignores the unknown
// directive, and the FPM pool's php_admin_value wins over the mods-available
// api_key) but they are confusing when debugging, so we evict them on every
// `magebox tideways config` run.
//
// Returns (cleaned, wasModified, err). wasModified is true if the file
// actually changed, so the caller can decide whether to reload PHP-FPM.
// Does nothing if the ini file does not exist.
func (m *Manager) CleanLegacyExtensionDirectives(phpVersion string) (bool, error) {
	iniPath := m.getExtensionIniPath(phpVersion)
	if iniPath == "" {
		return false, fmt.Errorf("unsupported platform")
	}

	if _, err := os.Stat(iniPath); err != nil {
		// Not installed for this PHP version — nothing to clean.
		return false, nil
	}

	existing, err := os.ReadFile(iniPath)
	if err != nil {
		return false, fmt.Errorf("failed to read %s: %w", iniPath, err)
	}

	cleaned := stripIniDirective(string(existing), "tideways.api_key")
	cleaned = stripIniDirective(cleaned, "tideways.environment")
	if cleaned == string(existing) {
		return false, nil
	}
	// Preserve exactly one trailing newline.
	if !strings.HasSuffix(cleaned, "\n") {
		cleaned += "\n"
	}

	// Write through sudo tee because mods-available files are root-owned.
	cmd := exec.Command("sudo", "tee", iniPath)
	cmd.Stdin = strings.NewReader(cleaned)
	cmd.Stdout = nil
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("failed to clean %s: %w", iniPath, err)
	}

	return true, nil
}

// stripIniDirective removes every line (commented or uncommented) that sets
// the given directive. Used to evict stale directives from earlier MageBox
// versions. See CleanLegacyExtensionDirectives.
func stripIniDirective(existing, directive string) string {
	lines := strings.Split(existing, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimLeft(line, " \t;")
		if strings.HasPrefix(trimmed, directive) {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// daemonEnvironmentDropInDir is the systemd drop-in directory used to inject
// TIDEWAYS_ENVIRONMENT into the tideways-daemon process environment. Exposed
// as a package variable so tests can point it at a temp dir.
const daemonEnvironmentDropInDir = "/etc/systemd/system/tideways-daemon.service.d"
const daemonEnvironmentDropInFile = "magebox-environment.conf"

// WriteDaemonEnvironment configures the Tideways daemon to label traces with
// the given environment name. The environment is a daemon-level setting, not
// a PHP extension setting — the extension transmits traces to the local
// daemon, and the daemon stamps them with whatever --env it was started with
// (default "production"). We install a systemd drop-in that sets
// TIDEWAYS_ENVIRONMENT in the daemon process environment, which the daemon
// reads at startup, and then restart the daemon. On macOS daemon env
// configuration is left manual — there is no clean, non-destructive way to
// inject env vars into a brew-services plist.
//
// See https://support.tideways.com/documentation/setup/configuration/environments.html
func (m *Manager) WriteDaemonEnvironment(environment string) error {
	if environment == "" {
		return fmt.Errorf("environment is empty")
	}

	switch m.platform.Type {
	case platform.Linux:
		content := renderDaemonEnvironmentDropIn(environment)
		dropInPath := filepath.Join(daemonEnvironmentDropInDir, daemonEnvironmentDropInFile)

		// Create drop-in directory and write the file via sudo.
		mkdir := exec.Command("sudo", "mkdir", "-p", daemonEnvironmentDropInDir)
		mkdir.Stdin = os.Stdin
		mkdir.Stdout = os.Stdout
		mkdir.Stderr = os.Stderr
		if err := mkdir.Run(); err != nil {
			return fmt.Errorf("failed to create %s: %w", daemonEnvironmentDropInDir, err)
		}

		tee := exec.Command("sudo", "tee", dropInPath)
		tee.Stdin = strings.NewReader(content)
		tee.Stdout = nil
		tee.Stderr = os.Stderr
		if err := tee.Run(); err != nil {
			return fmt.Errorf("failed to write %s: %w", dropInPath, err)
		}

		reload := exec.Command("sudo", "systemctl", "daemon-reload")
		reload.Stdin = os.Stdin
		reload.Stdout = os.Stdout
		reload.Stderr = os.Stderr
		if err := reload.Run(); err != nil {
			return fmt.Errorf("systemctl daemon-reload failed: %w", err)
		}

		restart := exec.Command("sudo", "systemctl", "restart", "tideways-daemon")
		restart.Stdin = os.Stdin
		restart.Stdout = os.Stdout
		restart.Stderr = os.Stderr
		if err := restart.Run(); err != nil {
			return fmt.Errorf("systemctl restart tideways-daemon failed: %w", err)
		}

		return nil

	case platform.Darwin:
		return fmt.Errorf("automatic daemon environment configuration is not supported on macOS yet — " +
			"set TIDEWAYS_ENVIRONMENT in the brew services plist manually")
	}
	return fmt.Errorf("unsupported platform")
}

// renderDaemonEnvironmentDropIn returns the systemd drop-in file contents
// that inject TIDEWAYS_ENVIRONMENT into the tideways-daemon process
// environment. Pure so it can be tested without sudo.
func renderDaemonEnvironmentDropIn(environment string) string {
	return fmt.Sprintf(`# Managed by MageBox — see 'magebox tideways config'.
# This drop-in labels all traces collected by the local tideways-daemon
# with the given environment name so local development traces don't land
# in the 'production' bucket on app.tideways.io.
[Service]
Environment="TIDEWAYS_ENVIRONMENT=%s"
`, environment)
}

// ImportCLIToken imports the access token for the `tideways` CLI command by
// running `tideways import <token>`. This is a separate credential from the
// PHP extension API key — it is used by the commandline tool for operations
// like `tideways run`, `tideways event create`, and `tideways tracepoint
// create`. See https://app.tideways.io/user/cli-import-settings.
func (m *Manager) ImportCLIToken() error {
	if m.credentials == nil || m.credentials.AccessToken == "" {
		return fmt.Errorf("tideways CLI access token not configured")
	}

	if !platform.CommandExists("tideways") {
		return fmt.Errorf("tideways CLI not found in PATH")
	}

	cmd := exec.Command("tideways", "import", m.credentials.AccessToken)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to import tideways CLI token: %w", err)
	}
	return nil
}

// sedInPlaceArgs returns the correct sed arguments for in-place editing on the current platform.
// macOS BSD sed requires "sed -i ” <expr> <file>" while GNU sed uses "sed -i <expr> <file>".
func (m *Manager) sedInPlaceArgs(expr, file string) []string {
	if m.platform.Type == platform.Darwin {
		return []string{"sed", "-i", "", expr, file}
	}
	return []string{"sed", "-i", expr, file}
}

// getExtensionIniPath returns the path to the Tideways PHP extension ini file
func (m *Manager) getExtensionIniPath(phpVersion string) string {
	switch m.platform.Type {
	case platform.Darwin:
		base := "/usr/local"
		if m.platform.IsAppleSilicon {
			base = "/opt/homebrew"
		}
		confDir := filepath.Join(base, "etc", "php", phpVersion, "conf.d")
		patterns := []string{
			filepath.Join(confDir, "ext-tideways.ini"),
			filepath.Join(confDir, "tideways.ini"),
			filepath.Join(confDir, "zz-tideways.ini"),
		}
		for _, p := range patterns {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
		return filepath.Join(confDir, "ext-tideways.ini")

	case platform.Linux:
		remiVersion := strings.ReplaceAll(phpVersion, ".", "")
		switch m.platform.LinuxDistro {
		case platform.DistroFedora:
			return fmt.Sprintf("/etc/opt/remi/php%s/php.d/zz-tideways.ini", remiVersion)
		default:
			return fmt.Sprintf("/etc/php/%s/mods-available/tideways.ini", phpVersion)
		}
	}
	return ""
}

// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package portforward

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"qoliber/magebox/internal/verbose"
)

const (
	launchDaemonPlist = "/Library/LaunchDaemons/com.magebox.portforward.plist"
	launchDaemonLabel = "com.magebox.portforward"

	// Legacy pf files — cleaned up during upgrade
	legacyPfRulesFile    = "/etc/pf.anchors/com.magebox"
	legacyPfHelperScript = "/usr/local/bin/magebox-pf-restore"
)

// Manager handles port forwarding setup
type Manager struct {
	platform string
}

// NewManager creates a new port forwarding manager
func NewManager() *Manager {
	return &Manager{
		platform: runtime.GOOS,
	}
}

// launchDaemonVersion is incremented when the plist content changes
// This ensures existing users get updates when they run bootstrap
const launchDaemonVersion = "7"

// findMageboxBinary returns the path to the magebox binary for use in the LaunchDaemon
func findMageboxBinary() string {
	// Check common installation paths
	candidates := []string{
		"/usr/local/bin/magebox",
		"/opt/homebrew/bin/magebox",
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Try to find via current executable
	if exe, err := os.Executable(); err == nil {
		return exe
	}

	return "/usr/local/bin/magebox"
}

// EnsureRulesActive checks if port forwarding is active and starts the daemon
// if needed. This is a lightweight operation safe to call on every
// "magebox start" so that reboots and sleep/wake cycles are self-healing.
// Returns true if forwarding was already active, false if it had to be restored.
func (m *Manager) EnsureRulesActive() (bool, error) {
	if m.platform != "darwin" {
		return true, nil // Not applicable on Linux
	}

	if !m.IsInstalled() {
		return false, fmt.Errorf("port forwarding not configured — run 'magebox bootstrap' first")
	}

	if m.AreRulesActive() {
		return true, nil
	}

	verbose.Debug("Port forwarding daemon not active, restarting...")

	// Try to kickstart the daemon
	if err := m.kickstartDaemon(); err != nil {
		return false, fmt.Errorf("failed to restart port forwarding daemon: %w", err)
	}

	// Wait briefly for the daemon to start listening
	for i := 0; i < 10; i++ {
		time.Sleep(200 * time.Millisecond)
		if m.AreRulesActive() {
			verbose.Debug("Port forwarding daemon started successfully")
			return false, nil
		}
	}

	return false, fmt.Errorf("port forwarding daemon not responding — check: sudo launchctl list %s", launchDaemonLabel)
}

// Setup installs the port forwarding LaunchDaemon
func (m *Manager) Setup() error {
	if m.platform != "darwin" {
		return fmt.Errorf("port forwarding is only supported on macOS")
	}

	verbose.Debug("Setting up macOS port forwarding...")

	// Clean up legacy pf-based approach
	m.cleanupLegacyPf()

	if m.IsInstalled() && !m.needsUpgrade() {
		verbose.Debug("Port forwarding daemon already installed and up to date")
		// Make sure it's running
		if !m.AreRulesActive() {
			_ = m.kickstartDaemon()
		}
		return nil
	}

	// Unload existing daemon if upgrading
	if m.IsInstalled() {
		verbose.Debug("Upgrading port forwarding daemon...")
		_ = m.unloadLaunchDaemon()
	}

	verbose.Debug("Installing port forwarding daemon (requires sudo)...")

	// Create LaunchDaemon plist
	if err := m.createLaunchDaemon(); err != nil {
		return fmt.Errorf("failed to create launch daemon: %w", err)
	}

	// Load the LaunchDaemon
	if err := m.loadLaunchDaemon(); err != nil {
		return fmt.Errorf("failed to load launch daemon: %w", err)
	}

	verbose.Debug("Port forwarding configured: 80 → 8080, 443 → 8443 (IPv4 + IPv6)")
	return nil
}

// IsInstalled checks if port forwarding is already configured
func (m *Manager) IsInstalled() bool {
	_, err := os.Stat(launchDaemonPlist)
	return err == nil
}

// needsUpgrade checks if the installed LaunchDaemon needs to be upgraded
func (m *Manager) needsUpgrade() bool {
	content, err := os.ReadFile(launchDaemonPlist)
	if err != nil {
		return true
	}

	versionMarker := fmt.Sprintf("MageBox-Version-%s", launchDaemonVersion)
	if !strings.Contains(string(content), versionMarker) {
		verbose.Debug("LaunchDaemon missing version %s marker", launchDaemonVersion)
		return true
	}

	return false
}

// Remove uninstalls port forwarding
func (m *Manager) Remove() error {
	if m.platform != "darwin" {
		return nil
	}

	fmt.Println("[INFO] Removing port forwarding configuration...")

	// Unload daemon
	if err := m.unloadLaunchDaemon(); err != nil {
		fmt.Printf("[WARN] Failed to unload launch daemon: %v\n", err)
	}

	// Remove plist
	cmd := exec.Command("sudo", "rm", "-f", launchDaemonPlist)
	if err := cmd.Run(); err != nil {
		fmt.Printf("[WARN] Failed to remove %s: %v\n", launchDaemonPlist, err)
	}

	// Clean up legacy files too
	m.cleanupLegacyPf()

	fmt.Println("[OK] Port forwarding removed")
	return nil
}

// cleanupLegacyPf removes leftover files from the old pf-based approach
func (m *Manager) cleanupLegacyPf() {
	legacyFiles := []string{legacyPfRulesFile, legacyPfHelperScript}
	for _, f := range legacyFiles {
		if _, err := os.Stat(f); err == nil {
			verbose.Debug("Removing legacy pf file: %s", f)
			cmd := exec.Command("sudo", "rm", "-f", f)
			_ = cmd.Run()
		}
	}

	// Remove magebox anchor from /etc/pf.conf if present
	content, err := os.ReadFile("/etc/pf.conf")
	if err != nil {
		return
	}
	pfConf := string(content)
	if !strings.Contains(pfConf, "com.magebox") {
		return
	}
	verbose.Debug("Removing legacy magebox entries from /etc/pf.conf")
	lines := strings.Split(pfConf, "\n")
	cleaned := make([]string, 0, len(lines))
	for _, line := range lines {
		if !strings.Contains(line, "com.magebox") {
			cleaned = append(cleaned, line)
		}
	}
	newContent := strings.Join(cleaned, "\n")
	tmpFile := "/tmp/pf.conf.magebox-cleanup"
	if err := os.WriteFile(tmpFile, []byte(newContent), 0644); err != nil {
		return
	}
	defer os.Remove(tmpFile)
	cmd := exec.Command("sudo", "cp", tmpFile, "/etc/pf.conf")
	_ = cmd.Run()
}

// createLaunchDaemon creates the LaunchDaemon plist that runs magebox _portforward
func (m *Manager) createLaunchDaemon() error {
	mageboxBin := findMageboxBinary()

	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<!-- MageBox-Version-%s -->
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.magebox.portforward</string>

    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>_portforward</string>
    </array>

    <key>RunAtLoad</key>
    <true/>

    <key>KeepAlive</key>
    <true/>

    <key>StandardOutPath</key>
    <string>/var/log/magebox-portforward.log</string>

    <key>StandardErrorPath</key>
    <string>/var/log/magebox-portforward.log</string>
</dict>
</plist>
`, launchDaemonVersion, mageboxBin)

	tmpFile := "/tmp/com.magebox.portforward.plist"
	if err := os.WriteFile(tmpFile, []byte(plist), 0644); err != nil {
		return err
	}

	cmd := exec.Command("sudo", "mv", tmpFile, launchDaemonPlist)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install launch daemon: %w", err)
	}

	cmd = exec.Command("sudo", "chown", "root:wheel", launchDaemonPlist)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set ownership: %w", err)
	}

	cmd = exec.Command("sudo", "chmod", "644", launchDaemonPlist)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	return nil
}

// loadLaunchDaemon loads the LaunchDaemon using both modern and legacy APIs
func (m *Manager) loadLaunchDaemon() error {
	cmd := exec.Command("sudo", "launchctl", "bootstrap", "system", launchDaemonPlist)
	if err := cmd.Run(); err != nil {
		verbose.Debug("launchctl bootstrap failed (trying legacy load): %v", err)
		cmd = exec.Command("sudo", "launchctl", "load", "-w", launchDaemonPlist)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	return nil
}

// unloadLaunchDaemon unloads the LaunchDaemon
func (m *Manager) unloadLaunchDaemon() error {
	cmd := exec.Command("sudo", "launchctl", "bootout", "system/"+launchDaemonLabel)
	if err := cmd.Run(); err != nil {
		verbose.Debug("launchctl bootout failed (trying legacy unload): %v", err)
		cmd = exec.Command("sudo", "launchctl", "unload", launchDaemonPlist)
		return cmd.Run()
	}
	return nil
}

// kickstartDaemon forces launchd to start the daemon immediately
func (m *Manager) kickstartDaemon() error {
	cmd := exec.Command("sudo", "launchctl", "kickstart", "-k", "system/"+launchDaemonLabel)
	return cmd.Run()
}

// Status checks if port forwarding is active
func (m *Manager) Status() error {
	if !m.IsInstalled() {
		fmt.Println("[INFO] Port forwarding is not configured")
		fmt.Println("       Run 'magebox bootstrap' to set it up")
		return nil
	}

	fmt.Println("[OK] Port forwarding is configured")
	fmt.Println("    LaunchDaemon: " + launchDaemonPlist)
	fmt.Println("    Mode: TCP proxy (magebox _portforward)")
	fmt.Println("    Forwarding: 80 → 8080, 443 → 8443")

	if m.isDaemonLoaded() {
		fmt.Println("[OK] Daemon is loaded")
	} else {
		fmt.Println("[WARN] Daemon is not loaded")
	}

	if m.AreRulesActive() {
		fmt.Println("[OK] Port forwarding is active")
	} else {
		fmt.Println("[WARN] Port forwarding is NOT active")
		fmt.Println("       Try: magebox bootstrap")
	}

	return nil
}

// isDaemonLoaded checks if the LaunchDaemon is loaded in launchd
func (m *Manager) isDaemonLoaded() bool {
	cmd := exec.Command("sudo", "launchctl", "list", launchDaemonLabel)
	return cmd.Run() == nil
}

// AreRulesActive checks if port forwarding is actually working by testing
// whether something is listening on the forwarded ports
func (m *Manager) AreRulesActive() bool {
	if m.platform != "darwin" {
		return true
	}
	// Try to connect to port 80 — if the proxy is running, it will accept
	conn, err := net.DialTimeout("tcp", "127.0.0.1:80", 500*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

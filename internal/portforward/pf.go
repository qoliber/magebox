// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package portforward

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"qoliber/magebox/internal/verbose"
)

const (
	pfRulesFile       = "/etc/pf.anchors/com.magebox"
	pfConfFile        = "/etc/pf.conf"
	pfHelperScript    = "/usr/local/bin/magebox-pf-restore"
	launchDaemonPlist = "/Library/LaunchDaemons/com.magebox.portforward.plist"
	launchDaemonLabel = "com.magebox.portforward"
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
const launchDaemonVersion = "5"

// EnsureRulesActive checks if pf port forwarding rules are active and restores
// them if needed. This is a lightweight operation safe to call on every
// "magebox start" so that reboots and sleep/wake cycles are self-healing.
// Returns true if rules were already active, false if they had to be restored.
func (m *Manager) EnsureRulesActive() (bool, error) {
	if m.platform != "darwin" {
		return true, nil // Not applicable on Linux
	}

	if !m.IsInstalled() {
		return false, fmt.Errorf("port forwarding not configured — run 'magebox bootstrap' first")
	}

	if m.areRulesActive() {
		return true, nil
	}

	verbose.Debug("PF rules not active, restoring...")

	// Ensure the anchor file exists and is up to date
	if _, err := os.Stat(pfRulesFile); os.IsNotExist(err) {
		if err := m.createPfRules(); err != nil {
			return false, fmt.Errorf("failed to recreate pf rules: %w", err)
		}
	}

	// Ensure the anchor is referenced in pf.conf
	if err := m.addAnchorToPfConf(); err != nil {
		return false, fmt.Errorf("failed to add anchor to pf.conf: %w", err)
	}

	// Reload pf (enables it if disabled, or just reloads rules)
	if err := m.reloadPfRules(); err != nil {
		return false, fmt.Errorf("failed to reload pf rules: %w", err)
	}

	// Verify rules are now active
	if !m.areRulesActive() {
		return false, fmt.Errorf("pf rules still not active after reload — try: sudo pfctl -ef /etc/pf.conf")
	}

	verbose.Debug("PF rules restored successfully")
	return false, nil
}

// Setup installs the port forwarding rules and LaunchDaemon
func (m *Manager) Setup() error {
	if m.platform != "darwin" {
		return fmt.Errorf("port forwarding is only supported on macOS")
	}

	verbose.Debug("Setting up macOS port forwarding...")

	// Check if already installed
	if m.IsInstalled() {
		verbose.Debug("Port forwarding plist exists, checking version...")

		// Always ensure pf rules and config are up to date
		if err := m.createPfRules(); err != nil {
			return fmt.Errorf("failed to update pf rules: %w", err)
		}
		if err := m.reloadPfRules(); err != nil {
			return fmt.Errorf("failed to reload pf rules: %w", err)
		}

		// Check if we need to upgrade the LaunchDaemon
		if m.needsUpgrade() {
			verbose.Debug("LaunchDaemon needs upgrade, reinstalling...")
			_ = m.unloadLaunchDaemon()
			if err := m.createHelperScript(); err != nil {
				return fmt.Errorf("failed to create helper script: %w", err)
			}
			if err := m.createLaunchDaemon(); err != nil {
				return fmt.Errorf("failed to upgrade launch daemon: %w", err)
			}
			if err := m.loadLaunchDaemon(); err != nil {
				return fmt.Errorf("failed to load upgraded launch daemon: %w", err)
			}
			verbose.Debug("LaunchDaemon upgraded to version %s", launchDaemonVersion)
		}

		verbose.Debug("Port forwarding configured and active")
		return nil
	}

	verbose.Debug("Installing port forwarding (requires sudo)...")

	// Create pf rules file (anchor)
	if err := m.createPfRules(); err != nil {
		return fmt.Errorf("failed to create pf rules: %w", err)
	}

	// Add anchor to /etc/pf.conf if not present
	if err := m.addAnchorToPfConf(); err != nil {
		return fmt.Errorf("failed to add anchor to pf.conf: %w", err)
	}

	// Create helper script
	if err := m.createHelperScript(); err != nil {
		return fmt.Errorf("failed to create helper script: %w", err)
	}

	// Create LaunchDaemon plist
	if err := m.createLaunchDaemon(); err != nil {
		return fmt.Errorf("failed to create launch daemon: %w", err)
	}

	// Load the LaunchDaemon
	if err := m.loadLaunchDaemon(); err != nil {
		return fmt.Errorf("failed to load launch daemon: %w", err)
	}

	// Reload pf rules immediately
	if err := m.reloadPfRules(); err != nil {
		return fmt.Errorf("failed to reload pf rules: %w", err)
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
		return true // Can't read, needs reinstall
	}

	// Check for version marker in plist
	versionMarker := fmt.Sprintf("MageBox-Version-%s", launchDaemonVersion)
	if !strings.Contains(string(content), versionMarker) {
		verbose.Debug("LaunchDaemon missing version %s marker", launchDaemonVersion)
		return true
	}

	// Verify helper script exists
	if _, err := os.Stat(pfHelperScript); os.IsNotExist(err) {
		verbose.Debug("Helper script missing at %s", pfHelperScript)
		return true
	}

	return false
}

// Remove uninstalls port forwarding rules
func (m *Manager) Remove() error {
	if m.platform != "darwin" {
		return nil
	}

	fmt.Println("[INFO] Removing port forwarding configuration...")

	// Unload LaunchDaemon
	if err := m.unloadLaunchDaemon(); err != nil {
		fmt.Printf("[WARN] Failed to unload launch daemon: %v\n", err)
	}

	// Remove files
	files := []string{
		launchDaemonPlist,
		pfRulesFile,
		pfHelperScript,
	}

	for _, file := range files {
		cmd := exec.Command("sudo", "rm", "-f", file)
		if err := cmd.Run(); err != nil {
			fmt.Printf("[WARN] Failed to remove %s: %v\n", file, err)
		}
	}

	fmt.Println("[OK] Port forwarding removed")
	return nil
}

// createPfRules creates the pf (packet filter) rules file
func (m *Manager) createPfRules() error {
	rules := `# MageBox port forwarding rules
# Forward privileged ports to unprivileged ports (IPv4 and IPv6)
rdr pass on lo0 inet proto tcp from any to any port 80 -> 127.0.0.1 port 8080
rdr pass on lo0 inet proto tcp from any to any port 443 -> 127.0.0.1 port 8443
rdr pass on lo0 inet6 proto tcp from any to ::1 port 80 -> ::1 port 8080
rdr pass on lo0 inet6 proto tcp from any to ::1 port 443 -> ::1 port 8443
`

	// Ensure directory exists
	dir := filepath.Dir(pfRulesFile)
	cmd := exec.Command("sudo", "mkdir", "-p", dir)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create pf.anchors directory: %w", err)
	}

	// Write rules file
	tmpFile := "/tmp/com.magebox.pf"
	if err := os.WriteFile(tmpFile, []byte(rules), 0644); err != nil {
		return err
	}

	cmd = exec.Command("sudo", "mv", tmpFile, pfRulesFile)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install pf rules: %w", err)
	}

	return nil
}

// createHelperScript creates the pf restore helper script that the LaunchDaemon calls.
// Using a script file instead of an inline plist command allows proper logging,
// error handling, and easier debugging when things go wrong.
func (m *Manager) createHelperScript() error {
	script := `#!/bin/sh
# MageBox PF Port Forwarding Restore Script
# Called by com.magebox.portforward LaunchDaemon on boot, sleep/wake, and periodically.
# Logs to /var/log/magebox-portforward.log

LOG="/var/log/magebox-portforward.log"
ANCHOR="com.magebox"
ANCHOR_FILE="/etc/pf.anchors/com.magebox"
PF_CONF="/etc/pf.conf"

log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') [magebox-pf] $1" >> "$LOG"
}

# Wait briefly for system to settle (boot / wake)
sleep 2

# Check if our anchor file exists
if [ ! -f "$ANCHOR_FILE" ]; then
    log "ERROR: anchor file missing: $ANCHOR_FILE"
    exit 1
fi

# Check if our rules are already active
if /sbin/pfctl -a "$ANCHOR" -sn 2>/dev/null | grep -q "port = 80"; then
    exit 0
fi

log "PF rules not active, restoring..."

# Check if pf is enabled
if /sbin/pfctl -s info 2>/dev/null | grep -q "Status: Enabled"; then
    # pf is enabled but our anchor rules are missing — reload just our anchor
    log "pf enabled, loading anchor rules..."
    if /sbin/pfctl -a "$ANCHOR" -f "$ANCHOR_FILE" 2>>"$LOG"; then
        log "Anchor rules loaded successfully"
    else
        # Anchor-only load failed, try full reload
        log "Anchor load failed, trying full pf.conf reload..."
        /sbin/pfctl -f "$PF_CONF" 2>>"$LOG"
    fi
else
    # pf is not enabled (typical after reboot) — enable it with full config
    log "pf disabled, enabling with full config..."
    /sbin/pfctl -ef "$PF_CONF" 2>>"$LOG"
fi

# Verify
if /sbin/pfctl -a "$ANCHOR" -sn 2>/dev/null | grep -q "port = 80"; then
    log "PF rules restored successfully"
else
    log "WARNING: PF rules still not active after restore attempt"
    exit 1
fi
`

	tmpFile := "/tmp/magebox-pf-restore.sh"
	if err := os.WriteFile(tmpFile, []byte(script), 0755); err != nil {
		return err
	}

	cmd := exec.Command("sudo", "mv", tmpFile, pfHelperScript)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install helper script: %w", err)
	}

	cmd = exec.Command("sudo", "chmod", "755", pfHelperScript)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set script permissions: %w", err)
	}

	return nil
}

// createLaunchDaemon creates the LaunchDaemon plist
func (m *Manager) createLaunchDaemon() error {
	// The LaunchDaemon calls the helper script which handles all the logic.
	// Triggers to keep rules active:
	// - RunAtLoad: runs on boot
	// - KeepAlive with NetworkState: re-triggers on network state change (sleep/wake)
	// - WatchPaths: re-triggers when pf config or network preferences change
	// - StartInterval: periodic fallback check every 30 seconds
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<!-- MageBox-Version-%s -->
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.magebox.portforward</string>

    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/magebox-pf-restore</string>
    </array>

    <key>RunAtLoad</key>
    <true/>

    <key>StartInterval</key>
    <integer>30</integer>

    <key>KeepAlive</key>
    <dict>
        <key>NetworkState</key>
        <true/>
    </dict>

    <key>WatchPaths</key>
    <array>
        <string>/etc/pf.conf</string>
        <string>/etc/pf.anchors</string>
        <string>/Library/Preferences/SystemConfiguration</string>
    </array>

    <key>ThrottleInterval</key>
    <integer>5</integer>

    <key>StandardOutPath</key>
    <string>/var/log/magebox-portforward.log</string>

    <key>StandardErrorPath</key>
    <string>/var/log/magebox-portforward.log</string>
</dict>
</plist>
`, launchDaemonVersion)

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

	// Set correct permissions
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
	// Try modern API first (macOS 10.10+)
	cmd := exec.Command("sudo", "launchctl", "bootstrap", "system", launchDaemonPlist)
	if err := cmd.Run(); err != nil {
		verbose.Debug("launchctl bootstrap failed (trying legacy load): %v", err)
		// Fall back to legacy API
		cmd = exec.Command("sudo", "launchctl", "load", "-w", launchDaemonPlist)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	return nil
}

// unloadLaunchDaemon unloads the LaunchDaemon using both modern and legacy APIs
func (m *Manager) unloadLaunchDaemon() error {
	// Try modern API first
	cmd := exec.Command("sudo", "launchctl", "bootout", "system/"+launchDaemonLabel)
	if err := cmd.Run(); err != nil {
		verbose.Debug("launchctl bootout failed (trying legacy unload): %v", err)
		// Fall back to legacy API
		cmd = exec.Command("sudo", "launchctl", "unload", launchDaemonPlist)
		return cmd.Run()
	}
	return nil
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
	fmt.Println("    PF Rules: " + pfRulesFile)
	fmt.Println("    Forwarding: 80 → 8080, 443 → 8443")

	// Check if loaded
	cmd := exec.Command("sudo", "launchctl", "list", launchDaemonLabel)
	if err := cmd.Run(); err != nil {
		fmt.Println("[WARN] LaunchDaemon is not loaded")
	} else {
		fmt.Println("[OK] LaunchDaemon is active")
	}

	// Check if rules are actually active
	if m.areRulesActive() {
		fmt.Println("[OK] PF rules are active")
	} else {
		fmt.Println("[WARN] PF rules are NOT active - port forwarding may not work")
		fmt.Println("       Try: sudo pfctl -ef /etc/pf.conf")
	}

	return nil
}

// AreRulesActive returns whether the pf redirect rules are currently loaded (exported)
func (m *Manager) AreRulesActive() bool {
	if m.platform != "darwin" {
		return true // Not applicable
	}
	return m.areRulesActive()
}

// areRulesActive checks if the MageBox pf rules are currently loaded
func (m *Manager) areRulesActive() bool {
	// Check if pf is enabled and our rdr (redirect) rules are loaded
	// Note: rdr rules are NAT rules, shown with -sn not -sr
	cmd := exec.Command("sudo", "pfctl", "-a", "com.magebox", "-sn")
	output, err := cmd.Output()
	if err != nil {
		verbose.Debug("Failed to get pf NAT rules: %v", err)
		return false
	}

	// Look for our redirect rules in the output
	rules := string(output)
	hasPort80 := strings.Contains(rules, "port = 80") || strings.Contains(rules, "port 80 ->")
	hasPort443 := strings.Contains(rules, "port = 443") || strings.Contains(rules, "port 443 ->")

	verbose.Debug("PF rules check: port80=%v, port443=%v", hasPort80, hasPort443)
	return hasPort80 && hasPort443
}

// addAnchorToPfConf adds MageBox anchor references to /etc/pf.conf
func (m *Manager) addAnchorToPfConf() error {
	verbose.Debug("Checking if anchor is in /etc/pf.conf...")

	content, err := os.ReadFile(pfConfFile)
	if err != nil {
		return fmt.Errorf("failed to read pf.conf: %w", err)
	}

	pfConf := string(content)

	if strings.Contains(pfConf, "com.magebox") {
		verbose.Debug("Anchor already present in pf.conf")
		return nil
	}

	verbose.Debug("Adding MageBox anchor to pf.conf...")

	newContent := m.insertAnchorIntoPfConf(pfConf)

	if err := m.writePfConfWithBackup(newContent); err != nil {
		return err
	}

	verbose.Debug("Added MageBox anchor to pf.conf")
	return nil
}

// insertAnchorIntoPfConf inserts the MageBox anchor lines into pf.conf content
// It adds: rdr-anchor "com.magebox" (near other rdr-anchor lines or before first rule)
// And: load anchor "com.magebox" from "/etc/pf.anchors/com.magebox" (at end)
func (m *Manager) insertAnchorIntoPfConf(pfConf string) string {
	lines := strings.Split(pfConf, "\n")
	result := make([]string, 0, len(lines)+2)

	rdrAnchorAdded := false
	lastRdrAnchorIdx := -1

	// First pass: find the last rdr-anchor line
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "rdr-anchor") {
			lastRdrAnchorIdx = i
		}
	}

	// Second pass: build the new content
	for i, line := range lines {
		result = append(result, line)

		// Add our rdr-anchor after the last existing rdr-anchor
		if i == lastRdrAnchorIdx && !rdrAnchorAdded {
			result = append(result, `rdr-anchor "com.magebox"`)
			rdrAnchorAdded = true
		}
	}

	// If no rdr-anchor existed, insert before first non-comment, non-empty line
	if !rdrAnchorAdded {
		result = m.insertRdrAnchorAtStart(result)
	}

	// Add load anchor at the end
	result = append(result, `load anchor "com.magebox" from "/etc/pf.anchors/com.magebox"`)
	result = append(result, "") // Trailing newline

	return strings.Join(result, "\n")
}

// insertRdrAnchorAtStart inserts the rdr-anchor line before the first rule
func (m *Manager) insertRdrAnchorAtStart(lines []string) []string {
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			// Insert before this line
			newLines := make([]string, 0, len(lines)+1)
			newLines = append(newLines, lines[:i]...)
			newLines = append(newLines, `rdr-anchor "com.magebox"`)
			newLines = append(newLines, lines[i:]...)
			return newLines
		}
	}
	// Fallback: prepend
	return append([]string{`rdr-anchor "com.magebox"`}, lines...)
}

// writePfConfWithBackup writes the new pf.conf content with a backup
func (m *Manager) writePfConfWithBackup(content string) error {
	tmpFile := "/tmp/pf.conf.magebox"
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	defer os.Remove(tmpFile)

	// Backup original
	cmd := exec.Command("sudo", "cp", pfConfFile, pfConfFile+".magebox.bak")
	if err := cmd.Run(); err != nil {
		verbose.Debug("Warning: failed to backup pf.conf: %v", err)
	}

	// Copy new file
	cmd = exec.Command("sudo", "cp", tmpFile, pfConfFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update pf.conf: %w", err)
	}

	return nil
}

// isPfEnabled checks if pf is currently enabled
func (m *Manager) isPfEnabled() bool {
	cmd := exec.Command("sudo", "pfctl", "-s", "info")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), "Status: Enabled")
}

// reloadPfRules reloads the pf configuration
func (m *Manager) reloadPfRules() error {
	verbose.Debug("Reloading pf rules...")

	// Check if pf is already enabled
	if m.isPfEnabled() {
		verbose.Debug("pf is already enabled, just reloading rules...")
		// Just reload rules without trying to enable
		cmd := exec.Command("sudo", "pfctl", "-f", pfConfFile)
		output, err := cmd.CombinedOutput()
		if err != nil {
			verbose.Debug("pfctl output: %s", string(output))
			return fmt.Errorf("pfctl failed: %w", err)
		}
	} else {
		verbose.Debug("pf is not enabled, enabling and loading rules...")
		// Enable pf and load rules
		cmd := exec.Command("sudo", "pfctl", "-ef", pfConfFile)
		output, err := cmd.CombinedOutput()
		if err != nil {
			verbose.Debug("pfctl output: %s", string(output))
			return fmt.Errorf("pfctl failed: %w", err)
		}
	}

	verbose.Debug("PF rules reloaded successfully")
	return nil
}

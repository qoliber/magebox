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
	launchDaemonPlist = "/Library/LaunchDaemons/com.magebox.portforward.plist"
	launchDaemonLabel = "com.magebox.portforward"
)

// launchDaemonVersion is incremented when the plist content changes
// This ensures existing users get updates when they run bootstrap
const launchDaemonVersion = "5"

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

// Setup installs the port forwarding rules and LaunchDaemon
// Standalone anchor approach — does NOT modify /etc/pf.conf, making us
// immune to macOS updates that reset the system pf configuration.
func (m *Manager) Setup() error {
	if m.platform != "darwin" {
		return fmt.Errorf("port forwarding is only supported on macOS")
	}

	verbose.Debug("Setting up macOS port forwarding (standalone anchor)...")

	// Ensure pf rules anchor file is up to date
	if err := m.createPfRules(); err != nil {
		return fmt.Errorf("failed to create pf rules: %w", err)
	}

	// Ensure LaunchDaemon is installed at the latest version
	if m.IsInstalled() && !m.needsUpgrade() {
		verbose.Debug("LaunchDaemon already at version %s", launchDaemonVersion)
	} else {
		if m.IsInstalled() {
			verbose.Debug("LaunchDaemon needs upgrade, reinstalling...")
			_ = m.unloadLaunchDaemon()
		}
		if err := m.createLaunchDaemon(); err != nil {
			return fmt.Errorf("failed to create launch daemon: %w", err)
		}
		if err := m.loadLaunchDaemon(); err != nil {
			return fmt.Errorf("failed to load launch daemon: %w", err)
		}
	}

	// Activate rules immediately (no reboot required)
	if err := m.activateRules(); err != nil {
		return fmt.Errorf("failed to activate rules: %w", err)
	}

	verbose.Debug("Port forwarding configured: 80 → 8080, 443 → 8443 (IPv4 + IPv6)")
	return nil
}

// EnsureActive verifies port forwarding is active and attempts self-repair
// if it's not. Designed to be called from `magebox start` as a safety net.
// Returns nil on success or when platform is not macOS.
func (m *Manager) EnsureActive() error {
	if m.platform != "darwin" {
		return nil
	}

	if m.areRulesActive() {
		verbose.Debug("PF rules are active")
		return nil
	}

	verbose.Debug("PF rules not active, attempting self-repair...")

	// If files are missing, we need a full Setup — bail out so user can run bootstrap
	if !m.IsInstalled() {
		return fmt.Errorf("port forwarding not installed (run: magebox bootstrap)")
	}
	if _, err := os.Stat(pfRulesFile); err != nil {
		return fmt.Errorf("pf rules file missing at %s (run: magebox bootstrap)", pfRulesFile)
	}

	// Try to activate the existing anchor
	if err := m.activateRules(); err != nil {
		return fmt.Errorf("failed to activate rules: %w", err)
	}

	if !m.areRulesActive() {
		return fmt.Errorf("rules still not active after reload (run: magebox doctor)")
	}

	verbose.Debug("PF rules self-repaired")
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

	versionMarker := fmt.Sprintf("MageBox-Version-%s", launchDaemonVersion)
	if !strings.Contains(string(content), versionMarker) {
		verbose.Debug("LaunchDaemon missing version %s marker", launchDaemonVersion)
		return true
	}

	return false
}

// Remove uninstalls port forwarding rules and disables pf-related changes
func (m *Manager) Remove() error {
	if m.platform != "darwin" {
		return nil
	}

	fmt.Println("[INFO] Removing port forwarding configuration...")

	// Unload LaunchDaemon
	if err := m.unloadLaunchDaemon(); err != nil {
		fmt.Printf("[WARN] Failed to unload launch daemon: %v\n", err)
	}

	// Flush our anchor (best effort)
	_ = exec.Command("sudo", "pfctl", "-a", "com.magebox", "-F", "all").Run()

	// Remove our files
	files := []string{launchDaemonPlist, pfRulesFile}
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

	dir := filepath.Dir(pfRulesFile)
	cmd := exec.Command("sudo", "mkdir", "-p", dir)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create pf.anchors directory: %w", err)
	}

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

// createLaunchDaemon creates the LaunchDaemon plist
// The daemon script is self-healing:
//  1. Ensures pf is enabled
//  2. Loads our standalone anchor (no pf.conf dependency)
//  3. Logs what it did with timestamps
func (m *Manager) createLaunchDaemon() error {
	// The script runs on boot, wake, and every 30s. It's idempotent:
	// always enables pf + loads the anchor, regardless of current state.
	script := `sleep 2
TS=$(date '+%Y-%m-%d %H:%M:%S')
# Ensure pf is enabled (ignore "already enabled" errors)
pfctl -E 2>/dev/null || true
# Load our standalone anchor
if pfctl -a com.magebox -f /etc/pf.anchors/com.magebox 2>/dev/null; then
  echo "[$TS] magebox pf rules loaded"
else
  echo "[$TS] magebox pf rules FAILED to load" >&2
  exit 1
fi
exit 0`

	// Escape XML entities in script
	scriptEscaped := strings.ReplaceAll(script, "&", "&amp;")
	scriptEscaped = strings.ReplaceAll(scriptEscaped, "<", "&lt;")
	scriptEscaped = strings.ReplaceAll(scriptEscaped, ">", "&gt;")

	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<!-- MageBox-Version-%s -->
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.magebox.portforward</string>

    <key>ProgramArguments</key>
    <array>
        <string>/bin/sh</string>
        <string>-c</string>
        <string>%s</string>
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
        <string>/etc/pf.anchors</string>
        <string>/Library/Preferences/SystemConfiguration</string>
    </array>

    <key>ThrottleInterval</key>
    <integer>5</integer>

    <key>StandardOutPath</key>
    <string>/var/log/magebox-portforward.log</string>

    <key>StandardErrorPath</key>
    <string>/var/log/magebox-portforward-error.log</string>
</dict>
</plist>
`, launchDaemonVersion, scriptEscaped)

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
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set ownership: %w", err)
	}

	cmd = exec.Command("sudo", "chmod", "644", launchDaemonPlist)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	return nil
}

// loadLaunchDaemon loads the LaunchDaemon
func (m *Manager) loadLaunchDaemon() error {
	cmd := exec.Command("sudo", "launchctl", "load", "-w", launchDaemonPlist)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// unloadLaunchDaemon unloads the LaunchDaemon
func (m *Manager) unloadLaunchDaemon() error {
	cmd := exec.Command("sudo", "launchctl", "unload", launchDaemonPlist)
	return cmd.Run()
}

// IsLaunchDaemonLoaded checks if the LaunchDaemon is currently loaded
func (m *Manager) IsLaunchDaemonLoaded() bool {
	cmd := exec.Command("sudo", "launchctl", "list", launchDaemonLabel)
	return cmd.Run() == nil
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

	if m.IsLaunchDaemonLoaded() {
		fmt.Println("[OK] LaunchDaemon is active")
	} else {
		fmt.Println("[WARN] LaunchDaemon is not loaded")
	}

	if m.areRulesActive() {
		fmt.Println("[OK] PF rules are active")
	} else {
		fmt.Println("[WARN] PF rules are NOT active - run: magebox doctor")
	}

	return nil
}

// areRulesActive checks if the MageBox pf rules are currently loaded
func (m *Manager) areRulesActive() bool {
	cmd := exec.Command("sudo", "pfctl", "-a", "com.magebox", "-sn")
	output, err := cmd.Output()
	if err != nil {
		verbose.Debug("Failed to get pf NAT rules: %v", err)
		return false
	}

	rules := string(output)
	hasPort80 := strings.Contains(rules, "port = 80") || strings.Contains(rules, "port 80 ->")
	hasPort443 := strings.Contains(rules, "port = 443") || strings.Contains(rules, "port 443 ->")

	verbose.Debug("PF rules check: port80=%v, port443=%v", hasPort80, hasPort443)
	return hasPort80 && hasPort443
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

// activateRules enables pf and loads the standalone anchor.
// Does NOT touch /etc/pf.conf — loads the anchor directly via `pfctl -a`.
func (m *Manager) activateRules() error {
	verbose.Debug("Activating MageBox pf anchor...")

	if !m.isPfEnabled() {
		verbose.Debug("pf is not enabled, enabling...")
		cmd := exec.Command("sudo", "pfctl", "-E")
		output, _ := cmd.CombinedOutput()
		verbose.Debug("pfctl -E output: %s", string(output))
	}

	// Load our standalone anchor — no dependency on /etc/pf.conf
	cmd := exec.Command("sudo", "pfctl", "-a", "com.magebox", "-f", pfRulesFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pfctl anchor load failed: %w (output: %s)", err, string(output))
	}

	verbose.Debug("MageBox pf anchor activated")
	return nil
}

// DoctorReport contains the result of a full diagnostic check
type DoctorReport struct {
	Platform            string
	PfRulesFileExists   bool
	PfRulesFileValid    bool
	LaunchDaemonExists  bool
	LaunchDaemonLoaded  bool
	LaunchDaemonVersion string
	PfEnabled           bool
	RulesActive         bool
	Issues              []string
}

// Diagnose performs a full diagnostic check of port forwarding state
func (m *Manager) Diagnose() *DoctorReport {
	r := &DoctorReport{Platform: m.platform}

	if m.platform != "darwin" {
		r.Issues = append(r.Issues, "not running on macOS — port forwarding not applicable")
		return r
	}

	// Check pf rules file
	if data, err := os.ReadFile(pfRulesFile); err == nil {
		r.PfRulesFileExists = true
		content := string(data)
		if strings.Contains(content, "port 80 -> 127.0.0.1 port 8080") &&
			strings.Contains(content, "port 443 -> 127.0.0.1 port 8443") {
			r.PfRulesFileValid = true
		} else {
			r.Issues = append(r.Issues, fmt.Sprintf("pf rules file at %s has unexpected content", pfRulesFile))
		}
	} else {
		r.Issues = append(r.Issues, fmt.Sprintf("pf rules file missing: %s", pfRulesFile))
	}

	// Check LaunchDaemon plist
	if data, err := os.ReadFile(launchDaemonPlist); err == nil {
		r.LaunchDaemonExists = true
		content := string(data)
		for _, v := range []string{"5", "4", "3", "2", "1"} {
			marker := fmt.Sprintf("MageBox-Version-%s", v)
			if strings.Contains(content, marker) {
				r.LaunchDaemonVersion = v
				break
			}
		}
		if r.LaunchDaemonVersion != launchDaemonVersion {
			r.Issues = append(r.Issues, fmt.Sprintf("LaunchDaemon is version %q, latest is %q", r.LaunchDaemonVersion, launchDaemonVersion))
		}
	} else {
		r.Issues = append(r.Issues, fmt.Sprintf("LaunchDaemon plist missing: %s", launchDaemonPlist))
	}

	r.LaunchDaemonLoaded = m.IsLaunchDaemonLoaded()
	if r.LaunchDaemonExists && !r.LaunchDaemonLoaded {
		r.Issues = append(r.Issues, "LaunchDaemon exists but is not loaded")
	}

	r.PfEnabled = m.isPfEnabled()
	if !r.PfEnabled {
		r.Issues = append(r.Issues, "pf is not enabled")
	}

	r.RulesActive = m.areRulesActive()
	if !r.RulesActive {
		r.Issues = append(r.Issues, "MageBox pf rules are not active (80/443 not being forwarded)")
	}

	return r
}

// Heal attempts to repair broken port forwarding state
// Returns a list of actions taken.
func (m *Manager) Heal() ([]string, error) {
	if m.platform != "darwin" {
		return nil, fmt.Errorf("port forwarding is only supported on macOS")
	}

	var actions []string

	// Recreate rules file if missing or invalid
	if _, err := os.Stat(pfRulesFile); err != nil {
		if err := m.createPfRules(); err != nil {
			return actions, fmt.Errorf("failed to create pf rules: %w", err)
		}
		actions = append(actions, "recreated pf rules file")
	}

	// Recreate LaunchDaemon if missing or out of date
	if !m.IsInstalled() || m.needsUpgrade() {
		if m.IsInstalled() {
			_ = m.unloadLaunchDaemon()
			actions = append(actions, "unloaded outdated LaunchDaemon")
		}
		if err := m.createLaunchDaemon(); err != nil {
			return actions, fmt.Errorf("failed to create LaunchDaemon: %w", err)
		}
		actions = append(actions, "installed LaunchDaemon")
	}

	// Ensure LaunchDaemon is loaded
	if !m.IsLaunchDaemonLoaded() {
		if err := m.loadLaunchDaemon(); err != nil {
			return actions, fmt.Errorf("failed to load LaunchDaemon: %w", err)
		}
		actions = append(actions, "loaded LaunchDaemon")
	}

	// Activate rules
	if err := m.activateRules(); err != nil {
		return actions, fmt.Errorf("failed to activate rules: %w", err)
	}
	actions = append(actions, "activated pf rules")

	return actions, nil
}

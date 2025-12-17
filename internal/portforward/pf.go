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

	"github.com/qoliber/magebox/internal/verbose"
)

const (
	pfRulesFile       = "/etc/pf.anchors/com.magebox"
	pfConfFile        = "/etc/pf.conf"
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

// Setup installs the port forwarding rules and LaunchDaemon
func (m *Manager) Setup() error {
	if m.platform != "darwin" {
		return fmt.Errorf("port forwarding is only supported on macOS")
	}

	verbose.Debug("Setting up macOS port forwarding...")

	// Check if already installed and active
	if m.IsInstalled() {
		verbose.Debug("Port forwarding plist exists, checking if rules are active...")
		// Even if installed, verify rules are active
		if m.areRulesActive() {
			verbose.Debug("Port forwarding already configured and active")
			return nil
		}
		verbose.Debug("Rules not active, reloading...")
		// Just reload pf rules if plist exists but rules not active
		return m.reloadPfRules()
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

	verbose.Debug("Port forwarding configured: 80 → 8080, 443 → 8443")
	return nil
}

// IsInstalled checks if port forwarding is already configured
func (m *Manager) IsInstalled() bool {
	_, err := os.Stat(launchDaemonPlist)
	return err == nil
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
# Forward privileged ports to unprivileged ports
rdr pass on lo0 inet proto tcp from any to any port 80 -> 127.0.0.1 port 8080
rdr pass on lo0 inet proto tcp from any to any port 443 -> 127.0.0.1 port 8443
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

// createLaunchDaemon creates the LaunchDaemon plist
func (m *Manager) createLaunchDaemon() error {
	// Load the main pf.conf which now includes our anchor
	plist := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.magebox.portforward</string>

    <key>ProgramArguments</key>
    <array>
        <string>/bin/sh</string>
        <string>-c</string>
        <string>pfctl -s info | grep -q "Status: Enabled" &amp;&amp; pfctl -f /etc/pf.conf || pfctl -ef /etc/pf.conf</string>
    </array>

    <key>RunAtLoad</key>
    <true/>

    <key>KeepAlive</key>
    <false/>

    <key>StandardOutPath</key>
    <string>/var/log/magebox-portforward.log</string>

    <key>StandardErrorPath</key>
    <string>/var/log/magebox-portforward-error.log</string>
</dict>
</plist>
`

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

// areRulesActive checks if the MageBox pf rules are currently loaded
func (m *Manager) areRulesActive() bool {
	// Check if pf is enabled and our rules are loaded
	cmd := exec.Command("sudo", "pfctl", "-sr")
	output, err := cmd.Output()
	if err != nil {
		verbose.Debug("Failed to get pf rules: %v", err)
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

	// Read current pf.conf
	content, err := os.ReadFile(pfConfFile)
	if err != nil {
		return fmt.Errorf("failed to read pf.conf: %w", err)
	}

	pfConf := string(content)

	// Check if our anchor is already present
	if strings.Contains(pfConf, "com.magebox") {
		verbose.Debug("Anchor already present in pf.conf")
		return nil
	}

	verbose.Debug("Adding MageBox anchor to pf.conf...")

	// We need to add two lines:
	// 1. rdr-anchor "com.magebox" - before any rdr rules
	// 2. load anchor "com.magebox" from "/etc/pf.anchors/com.magebox" - at the end

	// Find the right place to insert rdr-anchor (after other rdr-anchor lines or at start)
	lines := strings.Split(pfConf, "\n")
	var newLines []string
	rdrAnchorAdded := false
	loadAnchorAdded := false

	for _, line := range lines {
		// Add rdr-anchor after existing rdr-anchor lines
		if !rdrAnchorAdded && strings.HasPrefix(strings.TrimSpace(line), "rdr-anchor") {
			newLines = append(newLines, line)
			continue
		}

		// If we see a non-rdr-anchor line and haven't added ours yet, add it
		if !rdrAnchorAdded && !strings.HasPrefix(strings.TrimSpace(line), "rdr-anchor") && strings.TrimSpace(line) != "" && !strings.HasPrefix(strings.TrimSpace(line), "#") {
			// Check if there were any rdr-anchor lines before
			hasRdrAnchor := false
			for _, prev := range newLines {
				if strings.HasPrefix(strings.TrimSpace(prev), "rdr-anchor") {
					hasRdrAnchor = true
					break
				}
			}
			if hasRdrAnchor || strings.HasPrefix(strings.TrimSpace(line), "rdr") || strings.HasPrefix(strings.TrimSpace(line), "nat") {
				newLines = append(newLines, `rdr-anchor "com.magebox"`)
				rdrAnchorAdded = true
			}
		}

		newLines = append(newLines, line)

		// Add load anchor after existing load anchor lines
		if strings.HasPrefix(strings.TrimSpace(line), "load anchor") && !loadAnchorAdded {
			// Continue to collect all load anchor lines
			continue
		}
	}

	// If we still haven't added rdr-anchor, add it near the beginning
	if !rdrAnchorAdded {
		// Find first non-comment, non-empty line
		for i, line := range newLines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
				// Insert before this line
				newLines = append(newLines[:i], append([]string{`rdr-anchor "com.magebox"`}, newLines[i:]...)...)
				break
			}
		}
	}

	// Add load anchor at the end with trailing newline
	newLines = append(newLines, `load anchor "com.magebox" from "/etc/pf.anchors/com.magebox"`)
	newLines = append(newLines, "") // Ensure trailing newline

	newContent := strings.Join(newLines, "\n")

	// Write to temp file and move with sudo
	tmpFile := "/tmp/pf.conf.magebox"
	if err := os.WriteFile(tmpFile, []byte(newContent), 0644); err != nil {
		return err
	}

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

	verbose.Debug("Added MageBox anchor to pf.conf")
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

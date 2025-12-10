/**
 * Created by Qoliber
 *
 * @category    Qoliber
 * @package     Qoliber_MageBox
 * @author      Jakub Winkler <jwinkler@qoliber.com>
 */

package portforward

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const (
	pfRulesFile      = "/etc/pf.anchors/com.magebox"
	pfConfFragment   = "/etc/pf.anchors/com.magebox.conf"
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

	// Check if already installed
	if m.IsInstalled() {
		fmt.Println("[INFO] Port forwarding already configured")
		return nil
	}

	fmt.Println("[INFO] Setting up port forwarding (requires sudo)...")
	fmt.Println("       This allows Nginx to run on ports 8080/8443 as your user")
	fmt.Println("       while being accessible on standard ports 80/443")

	// Create pf rules file
	if err := m.createPfRules(); err != nil {
		return fmt.Errorf("failed to create pf rules: %w", err)
	}

	// Create LaunchDaemon plist
	if err := m.createLaunchDaemon(); err != nil {
		return fmt.Errorf("failed to create launch daemon: %w", err)
	}

	// Load the LaunchDaemon
	if err := m.loadLaunchDaemon(); err != nil {
		return fmt.Errorf("failed to load launch daemon: %w", err)
	}

	fmt.Println("[OK] Port forwarding configured successfully")
	fmt.Println("     80 → 8080, 443 → 8443")

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
		pfConfFragment,
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
        <string>pfctl -ef /etc/pf.anchors/com.magebox 2&gt;&amp;1</string>
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

	return nil
}

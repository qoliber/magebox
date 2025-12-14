// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/qoliber/magebox/internal/blackfire"
	"github.com/qoliber/magebox/internal/cli"
	"github.com/qoliber/magebox/internal/config"
	"github.com/qoliber/magebox/internal/php"
	"github.com/qoliber/magebox/internal/xdebug"
)

var blackfireCmd = &cobra.Command{
	Use:   "blackfire",
	Short: "Manage Blackfire PHP profiler",
	Long: `Enable, disable, or check status of Blackfire profiler.

Use 'magebox blackfire on' to enable Blackfire for the current project's PHP version.
Use 'magebox blackfire off' to disable Blackfire.
Use 'magebox blackfire status' to check current Blackfire status.
Use 'magebox blackfire install' to install Blackfire agent and PHP extension.
Use 'magebox blackfire config' to configure Blackfire credentials.`,
	RunE: runBlackfireStatus,
}

var blackfireOnCmd = &cobra.Command{
	Use:   "on",
	Short: "Enable Blackfire",
	Long: `Enables Blackfire for the project's PHP version and restarts PHP-FPM.
Xdebug will be automatically disabled to avoid conflicts.`,
	RunE: runBlackfireOn,
}

var blackfireOffCmd = &cobra.Command{
	Use:   "off",
	Short: "Disable Blackfire",
	Long:  `Disables Blackfire for the project's PHP version and restarts PHP-FPM.`,
	RunE:  runBlackfireOff,
}

var blackfireStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Blackfire status",
	Long:  `Shows the current Blackfire status including agent and extension state.`,
	RunE:  runBlackfireStatus,
}

var blackfireInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install Blackfire",
	Long:  `Installs the Blackfire agent and PHP extension for the project's PHP version.`,
	RunE:  runBlackfireInstall,
}

var blackfireConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure Blackfire credentials",
	Long: `Configures Blackfire credentials for the agent and CLI.
Credentials are stored in ~/.magebox/config.yaml.`,
	RunE: runBlackfireConfig,
}

// Flags for blackfire config
var (
	blackfireServerID    string
	blackfireServerToken string
	blackfireClientID    string
	blackfireClientToken string
)

func init() {
	// Add flags for non-interactive config
	blackfireConfigCmd.Flags().StringVar(&blackfireServerID, "server-id", "", "Blackfire Server ID")
	blackfireConfigCmd.Flags().StringVar(&blackfireServerToken, "server-token", "", "Blackfire Server Token")
	blackfireConfigCmd.Flags().StringVar(&blackfireClientID, "client-id", "", "Blackfire Client ID (optional, for CLI)")
	blackfireConfigCmd.Flags().StringVar(&blackfireClientToken, "client-token", "", "Blackfire Client Token (optional, for CLI)")

	blackfireCmd.AddCommand(blackfireOnCmd)
	blackfireCmd.AddCommand(blackfireOffCmd)
	blackfireCmd.AddCommand(blackfireStatusCmd)
	blackfireCmd.AddCommand(blackfireInstallCmd)
	blackfireCmd.AddCommand(blackfireConfigCmd)
	rootCmd.AddCommand(blackfireCmd)
}

func runBlackfireOn(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	p, err := getPlatform()
	if err != nil {
		return err
	}

	phpVersion, err := getProjectPHPVersion(cwd)
	if err != nil {
		return err
	}

	cli.PrintTitle("Enabling Blackfire")
	fmt.Printf("PHP version: %s\n", cli.Highlight(phpVersion))
	fmt.Println()

	mgr := blackfire.NewManager(p, nil)

	// Check if extension is installed
	if !mgr.IsExtensionInstalled(phpVersion) {
		cli.PrintWarning("Blackfire extension not installed for PHP %s", phpVersion)
		cli.PrintInfo("Install with: magebox blackfire install")
		return nil
	}

	// Check if agent is installed
	if !mgr.IsAgentInstalled() {
		cli.PrintWarning("Blackfire agent is not installed")
		cli.PrintInfo("Install with: magebox blackfire install")
		return nil
	}

	// Check and save Xdebug state before disabling
	xdebugMgr := xdebug.NewManager(p)
	xdebugWasEnabled := xdebugMgr.IsEnabled(phpVersion)
	if xdebugWasEnabled {
		fmt.Print("Disabling Xdebug (conflicts with Blackfire)... ")
		if err := xdebugMgr.Disable(phpVersion); err != nil {
			fmt.Println(cli.Warning("failed"))
		} else {
			fmt.Println(cli.Success("done"))
		}
		// Save state so we can restore it when Blackfire is disabled
		saveXdebugState(p.HomeDir, phpVersion, true)
	}

	// Enable Blackfire
	fmt.Print("Enabling Blackfire... ")
	if err := mgr.Enable(phpVersion); err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to enable Blackfire: %w", err)
	}
	fmt.Println(cli.Success("done"))

	// Start agent if not running
	if !mgr.IsAgentRunning() {
		fmt.Print("Starting Blackfire agent... ")
		if err := mgr.StartAgent(); err != nil {
			fmt.Println(cli.Warning("failed (may need manual start)"))
		} else {
			fmt.Println(cli.Success("done"))
		}
	}

	// Restart PHP-FPM
	fmt.Print("Restarting PHP-FPM... ")
	fpmCtrl := php.NewFPMController(p, phpVersion)
	if err := fpmCtrl.Reload(); err != nil {
		fmt.Println(cli.Warning("failed (may need manual restart)"))
	} else {
		fmt.Println(cli.Success("done"))
	}

	fmt.Println()
	cli.PrintSuccess("Blackfire enabled!")
	fmt.Println()
	cli.PrintInfo("Run 'blackfire curl https://your-site.test' to profile a request")

	return nil
}

func runBlackfireOff(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	p, err := getPlatform()
	if err != nil {
		return err
	}

	phpVersion, err := getProjectPHPVersion(cwd)
	if err != nil {
		return err
	}

	cli.PrintTitle("Disabling Blackfire")
	fmt.Printf("PHP version: %s\n", cli.Highlight(phpVersion))
	fmt.Println()

	mgr := blackfire.NewManager(p, nil)

	// Disable Blackfire
	fmt.Print("Disabling Blackfire... ")
	if err := mgr.Disable(phpVersion); err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to disable Blackfire: %w", err)
	}
	fmt.Println(cli.Success("done"))

	// Restore Xdebug if it was enabled before
	if loadXdebugState(p.HomeDir, phpVersion) {
		xdebugMgr := xdebug.NewManager(p)
		fmt.Print("Restoring Xdebug (was enabled before)... ")
		if err := xdebugMgr.Enable(phpVersion); err != nil {
			fmt.Println(cli.Warning("failed"))
		} else {
			fmt.Println(cli.Success("done"))
		}
		// Clear the saved state
		clearXdebugState(p.HomeDir, phpVersion)
	}

	// Restart PHP-FPM
	fmt.Print("Restarting PHP-FPM... ")
	fpmCtrl := php.NewFPMController(p, phpVersion)
	if err := fpmCtrl.Reload(); err != nil {
		fmt.Println(cli.Warning("failed (may need manual restart)"))
	} else {
		fmt.Println(cli.Success("done"))
	}

	fmt.Println()
	cli.PrintSuccess("Blackfire disabled!")

	return nil
}

func runBlackfireStatus(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	p, err := getPlatform()
	if err != nil {
		return err
	}

	phpVersion, err := getProjectPHPVersion(cwd)
	if err != nil {
		return err
	}

	globalCfg, err := config.LoadGlobalConfig(p.HomeDir)
	if err != nil {
		return err
	}

	cli.PrintTitle("Blackfire Status")
	fmt.Printf("PHP version: %s\n", cli.Highlight(phpVersion))
	fmt.Println()

	mgr := blackfire.NewManager(p, nil)
	status := mgr.GetStatus([]string{phpVersion})

	// Agent status
	fmt.Println("Agent:")
	fmt.Printf("  Installed: %s\n", formatBool(status.AgentInstalled))
	fmt.Printf("  Running:   %s\n", formatBool(status.AgentRunning))
	fmt.Println()

	// Extension status
	fmt.Println("PHP Extension:")
	fmt.Printf("  Installed: %s\n", formatBool(status.ExtensionInstalled[phpVersion]))
	fmt.Printf("  Enabled:   %s\n", formatBool(status.ExtensionEnabled[phpVersion]))
	fmt.Println()

	// Credentials status
	fmt.Println("Credentials:")
	fmt.Printf("  Server configured: %s\n", formatBool(globalCfg.HasBlackfireCredentials()))
	fmt.Printf("  Client configured: %s\n", formatBool(globalCfg.HasBlackfireClientCredentials()))
	fmt.Println()

	// Helpful tips
	if !status.AgentInstalled || !status.ExtensionInstalled[phpVersion] {
		cli.PrintInfo("Install with: magebox blackfire install")
	} else if !status.ExtensionEnabled[phpVersion] {
		cli.PrintInfo("Enable with: magebox blackfire on")
	} else {
		cli.PrintInfo("Disable with: magebox blackfire off")
	}

	if !globalCfg.HasBlackfireCredentials() {
		cli.PrintWarning("Configure credentials with: magebox blackfire config")
	}

	return nil
}

func runBlackfireInstall(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	p, err := getPlatform()
	if err != nil {
		return err
	}

	phpVersion, err := getProjectPHPVersion(cwd)
	if err != nil {
		return err
	}

	cli.PrintTitle("Installing Blackfire")
	fmt.Printf("PHP version: %s\n", cli.Highlight(phpVersion))
	fmt.Println()

	installer := blackfire.NewInstaller(p)

	// Install agent if not installed
	mgr := blackfire.NewManager(p, nil)
	if !mgr.IsAgentInstalled() {
		fmt.Print("Installing Blackfire agent... ")
		if err := installer.InstallAgent(); err != nil {
			fmt.Println(cli.Error("failed"))
			return fmt.Errorf("failed to install agent: %w", err)
		}
		fmt.Println(cli.Success("done"))
	} else {
		fmt.Println("Blackfire agent already installed " + cli.Success("✓"))
	}

	// Install extension if not installed
	if !mgr.IsExtensionInstalled(phpVersion) {
		fmt.Printf("Installing Blackfire PHP extension for %s... ", phpVersion)
		if err := installer.InstallExtension(phpVersion); err != nil {
			fmt.Println(cli.Error("failed"))
			return fmt.Errorf("failed to install extension: %w", err)
		}
		fmt.Println(cli.Success("done"))
	} else {
		fmt.Printf("Blackfire PHP extension already installed for %s %s\n", phpVersion, cli.Success("✓"))
	}

	fmt.Println()
	cli.PrintSuccess("Blackfire installed!")
	fmt.Println()
	cli.PrintInfo("Configure credentials with: magebox blackfire config")
	cli.PrintInfo("Enable profiling with: magebox blackfire on")

	return nil
}

func runBlackfireConfig(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	// Load existing config
	globalCfg, err := config.LoadGlobalConfig(p.HomeDir)
	if err != nil {
		return err
	}

	existingCreds := globalCfg.GetBlackfireCredentials()

	// Check if we have enough flags for non-interactive mode
	nonInteractive := blackfireServerID != "" && blackfireServerToken != ""

	var serverID, serverToken, clientID, clientToken string

	if nonInteractive {
		// Use flag values
		serverID = blackfireServerID
		serverToken = blackfireServerToken
		clientID = blackfireClientID
		clientToken = blackfireClientToken
	} else {
		// Interactive mode
		cli.PrintTitle("Configure Blackfire")
		fmt.Println("Get your credentials from: https://blackfire.io/my/settings/credentials")
		fmt.Println()

		reader := bufio.NewReader(os.Stdin)

		// Prompt for Server ID
		fmt.Print("Server ID")
		if existingCreds.ServerID != "" {
			fmt.Printf(" [%s]", maskCredential(existingCreds.ServerID))
		}
		fmt.Print(": ")
		serverID, _ = reader.ReadString('\n')
		serverID = strings.TrimSpace(serverID)
		if serverID == "" && existingCreds.ServerID != "" {
			serverID = existingCreds.ServerID
		}

		// Prompt for Server Token
		fmt.Print("Server Token")
		if existingCreds.ServerToken != "" {
			fmt.Printf(" [%s]", maskCredential(existingCreds.ServerToken))
		}
		fmt.Print(": ")
		serverToken, _ = reader.ReadString('\n')
		serverToken = strings.TrimSpace(serverToken)
		if serverToken == "" && existingCreds.ServerToken != "" {
			serverToken = existingCreds.ServerToken
		}

		// Prompt for Client ID (optional)
		fmt.Print("Client ID (for CLI, optional)")
		if existingCreds.ClientID != "" {
			fmt.Printf(" [%s]", maskCredential(existingCreds.ClientID))
		}
		fmt.Print(": ")
		clientID, _ = reader.ReadString('\n')
		clientID = strings.TrimSpace(clientID)
		if clientID == "" && existingCreds.ClientID != "" {
			clientID = existingCreds.ClientID
		}

		// Prompt for Client Token (optional)
		fmt.Print("Client Token (for CLI, optional)")
		if existingCreds.ClientToken != "" {
			fmt.Printf(" [%s]", maskCredential(existingCreds.ClientToken))
		}
		fmt.Print(": ")
		clientToken, _ = reader.ReadString('\n')
		clientToken = strings.TrimSpace(clientToken)
		if clientToken == "" && existingCreds.ClientToken != "" {
			clientToken = existingCreds.ClientToken
		}
	}

	// Validate required fields
	if serverID == "" || serverToken == "" {
		return fmt.Errorf("server ID and token are required")
	}

	// Save to global config
	globalCfg.Profiling.Blackfire = config.BlackfireCredentials{
		ServerID:    serverID,
		ServerToken: serverToken,
		ClientID:    clientID,
		ClientToken: clientToken,
	}

	fmt.Println()
	fmt.Print("Saving credentials... ")
	if err := config.SaveGlobalConfig(p.HomeDir, globalCfg); err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to save config: %w", err)
	}
	fmt.Println(cli.Success("done"))

	// Configure agent if installed
	mgr := blackfire.NewManager(p, &blackfire.Credentials{
		ServerID:    serverID,
		ServerToken: serverToken,
		ClientID:    clientID,
		ClientToken: clientToken,
	})

	if mgr.IsAgentInstalled() {
		fmt.Print("Configuring Blackfire agent... ")
		if err := mgr.ConfigureAgent(); err != nil {
			fmt.Println(cli.Warning("failed (configure manually)"))
		} else {
			fmt.Println(cli.Success("done"))
		}

		// Restart agent
		fmt.Print("Restarting Blackfire agent... ")
		if err := mgr.RestartAgent(); err != nil {
			fmt.Println(cli.Warning("failed (restart manually)"))
		} else {
			fmt.Println(cli.Success("done"))
		}
	}

	// Configure CLI if credentials provided
	if clientID != "" && clientToken != "" {
		fmt.Print("Configuring Blackfire CLI... ")
		if err := mgr.ConfigureCLI(); err != nil {
			fmt.Println(cli.Warning("failed (configure manually)"))
		} else {
			fmt.Println(cli.Success("done"))
		}
	}

	fmt.Println()
	cli.PrintSuccess("Blackfire configured!")
	fmt.Println()
	cli.PrintInfo("Enable profiling with: magebox blackfire on")

	return nil
}

// maskCredential masks a credential for display (shows first 4 and last 4 chars)
func maskCredential(s string) string {
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "****" + s[len(s)-4:]
}

// xdebugStateFile returns the path to the Xdebug state file for a PHP version
func xdebugStateFile(homeDir, phpVersion string) string {
	return filepath.Join(homeDir, ".magebox", "run", fmt.Sprintf("xdebug-state-%s", phpVersion))
}

// saveXdebugState saves the Xdebug state (was enabled) for a PHP version
func saveXdebugState(homeDir, phpVersion string, wasEnabled bool) {
	if !wasEnabled {
		return
	}
	stateFile := xdebugStateFile(homeDir, phpVersion)
	// Ensure directory exists
	_ = os.MkdirAll(filepath.Dir(stateFile), 0755)
	_ = os.WriteFile(stateFile, []byte("enabled"), 0644)
}

// loadXdebugState loads the Xdebug state for a PHP version
func loadXdebugState(homeDir, phpVersion string) bool {
	stateFile := xdebugStateFile(homeDir, phpVersion)
	_, err := os.Stat(stateFile)
	return err == nil
}

// clearXdebugState clears the saved Xdebug state for a PHP version
func clearXdebugState(homeDir, phpVersion string) {
	stateFile := xdebugStateFile(homeDir, phpVersion)
	_ = os.Remove(stateFile)
}

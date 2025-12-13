// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/qoliber/magebox/internal/cli"
	"github.com/qoliber/magebox/internal/config"
	"github.com/qoliber/magebox/internal/php"
	"github.com/qoliber/magebox/internal/tideways"
	"github.com/qoliber/magebox/internal/xdebug"
)

var tidewaysCmd = &cobra.Command{
	Use:   "tideways",
	Short: "Manage Tideways PHP profiler",
	Long: `Enable, disable, or check status of Tideways profiler.

Use 'magebox tideways on' to enable Tideways for the current project's PHP version.
Use 'magebox tideways off' to disable Tideways.
Use 'magebox tideways status' to check current Tideways status.
Use 'magebox tideways install' to install Tideways daemon and PHP extension.
Use 'magebox tideways config' to configure Tideways API key.`,
	RunE: runTidewaysStatus,
}

var tidewaysOnCmd = &cobra.Command{
	Use:   "on",
	Short: "Enable Tideways",
	Long: `Enables Tideways for the project's PHP version and restarts PHP-FPM.
Xdebug will be automatically disabled to avoid conflicts.`,
	RunE: runTidewaysOn,
}

var tidewaysOffCmd = &cobra.Command{
	Use:   "off",
	Short: "Disable Tideways",
	Long:  `Disables Tideways for the project's PHP version and restarts PHP-FPM.`,
	RunE:  runTidewaysOff,
}

var tidewaysStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Tideways status",
	Long:  `Shows the current Tideways status including daemon and extension state.`,
	RunE:  runTidewaysStatus,
}

var tidewaysInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install Tideways",
	Long:  `Installs the Tideways daemon and PHP extension for the project's PHP version.`,
	RunE:  runTidewaysInstall,
}

var tidewaysConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure Tideways API key",
	Long: `Configures the Tideways API key for the daemon.
The API key is stored in ~/.magebox/config.yaml.`,
	RunE: runTidewaysConfig,
}

func init() {
	tidewaysCmd.AddCommand(tidewaysOnCmd)
	tidewaysCmd.AddCommand(tidewaysOffCmd)
	tidewaysCmd.AddCommand(tidewaysStatusCmd)
	tidewaysCmd.AddCommand(tidewaysInstallCmd)
	tidewaysCmd.AddCommand(tidewaysConfigCmd)
	rootCmd.AddCommand(tidewaysCmd)
}

func runTidewaysOn(cmd *cobra.Command, args []string) error {
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

	cli.PrintTitle("Enabling Tideways")
	fmt.Printf("PHP version: %s\n", cli.Highlight(phpVersion))
	fmt.Println()

	mgr := tideways.NewManager(p, nil)

	// Check if extension is installed
	if !mgr.IsExtensionInstalled(phpVersion) {
		cli.PrintWarning("Tideways extension not installed for PHP %s", phpVersion)
		cli.PrintInfo("Install with: magebox tideways install")
		return nil
	}

	// Check if daemon is installed
	if !mgr.IsDaemonInstalled() {
		cli.PrintWarning("Tideways daemon is not installed")
		cli.PrintInfo("Install with: magebox tideways install")
		return nil
	}

	// Disable Xdebug first
	xdebugMgr := xdebug.NewManager(p)
	if xdebugMgr.IsEnabled(phpVersion) {
		fmt.Print("Disabling Xdebug (conflicts with Tideways)... ")
		if err := xdebugMgr.Disable(phpVersion); err != nil {
			fmt.Println(cli.Warning("failed"))
		} else {
			fmt.Println(cli.Success("done"))
		}
	}

	// Enable Tideways
	fmt.Print("Enabling Tideways... ")
	if err := mgr.Enable(phpVersion); err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to enable Tideways: %w", err)
	}
	fmt.Println(cli.Success("done"))

	// Start daemon if not running
	if !mgr.IsDaemonRunning() {
		fmt.Print("Starting Tideways daemon... ")
		if err := mgr.StartDaemon(); err != nil {
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
	cli.PrintSuccess("Tideways enabled!")
	fmt.Println()
	cli.PrintInfo("View your traces at https://app.tideways.io")

	return nil
}

func runTidewaysOff(cmd *cobra.Command, args []string) error {
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

	cli.PrintTitle("Disabling Tideways")
	fmt.Printf("PHP version: %s\n", cli.Highlight(phpVersion))
	fmt.Println()

	mgr := tideways.NewManager(p, nil)

	// Disable Tideways
	fmt.Print("Disabling Tideways... ")
	if err := mgr.Disable(phpVersion); err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to disable Tideways: %w", err)
	}
	fmt.Println(cli.Success("done"))

	// Restart PHP-FPM
	fmt.Print("Restarting PHP-FPM... ")
	fpmCtrl := php.NewFPMController(p, phpVersion)
	if err := fpmCtrl.Reload(); err != nil {
		fmt.Println(cli.Warning("failed (may need manual restart)"))
	} else {
		fmt.Println(cli.Success("done"))
	}

	fmt.Println()
	cli.PrintSuccess("Tideways disabled!")

	return nil
}

func runTidewaysStatus(cmd *cobra.Command, args []string) error {
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

	cli.PrintTitle("Tideways Status")
	fmt.Printf("PHP version: %s\n", cli.Highlight(phpVersion))
	fmt.Println()

	mgr := tideways.NewManager(p, nil)
	status := mgr.GetStatus([]string{phpVersion})

	// Daemon status
	fmt.Println("Daemon:")
	fmt.Printf("  Installed: %s\n", formatBool(status.DaemonInstalled))
	fmt.Printf("  Running:   %s\n", formatBool(status.DaemonRunning))
	fmt.Println()

	// Extension status
	fmt.Println("PHP Extension:")
	fmt.Printf("  Installed: %s\n", formatBool(status.ExtensionInstalled[phpVersion]))
	fmt.Printf("  Enabled:   %s\n", formatBool(status.ExtensionEnabled[phpVersion]))
	fmt.Println()

	// Credentials status
	fmt.Println("Credentials:")
	fmt.Printf("  API Key configured: %s\n", formatBool(globalCfg.HasTidewaysCredentials()))
	fmt.Println()

	// Helpful tips
	if !status.DaemonInstalled || !status.ExtensionInstalled[phpVersion] {
		cli.PrintInfo("Install with: magebox tideways install")
	} else if !status.ExtensionEnabled[phpVersion] {
		cli.PrintInfo("Enable with: magebox tideways on")
	} else {
		cli.PrintInfo("Disable with: magebox tideways off")
	}

	if !globalCfg.HasTidewaysCredentials() {
		cli.PrintWarning("Configure API key with: magebox tideways config")
	}

	return nil
}

func runTidewaysInstall(cmd *cobra.Command, args []string) error {
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

	cli.PrintTitle("Installing Tideways")
	fmt.Printf("PHP version: %s\n", cli.Highlight(phpVersion))
	fmt.Println()

	installer := tideways.NewInstaller(p)

	// Install daemon if not installed
	mgr := tideways.NewManager(p, nil)
	if !mgr.IsDaemonInstalled() {
		fmt.Print("Installing Tideways daemon... ")
		if err := installer.InstallDaemon(); err != nil {
			fmt.Println(cli.Error("failed"))
			return fmt.Errorf("failed to install daemon: %w", err)
		}
		fmt.Println(cli.Success("done"))
	} else {
		fmt.Println("Tideways daemon already installed " + cli.Success("✓"))
	}

	// Install extension if not installed
	if !mgr.IsExtensionInstalled(phpVersion) {
		fmt.Printf("Installing Tideways PHP extension for %s... ", phpVersion)
		if err := installer.InstallExtension(phpVersion); err != nil {
			fmt.Println(cli.Error("failed"))
			return fmt.Errorf("failed to install extension: %w", err)
		}
		fmt.Println(cli.Success("done"))
	} else {
		fmt.Printf("Tideways PHP extension already installed for %s %s\n", phpVersion, cli.Success("✓"))
	}

	fmt.Println()
	cli.PrintSuccess("Tideways installed!")
	fmt.Println()
	cli.PrintInfo("Configure API key with: magebox tideways config")
	cli.PrintInfo("Enable profiling with: magebox tideways on")

	return nil
}

func runTidewaysConfig(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	cli.PrintTitle("Configure Tideways")
	fmt.Println("Get your API key from: https://app.tideways.io/settings/api")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	// Load existing config
	globalCfg, err := config.LoadGlobalConfig(p.HomeDir)
	if err != nil {
		return err
	}

	existingCreds := globalCfg.GetTidewaysCredentials()

	// Prompt for API key
	fmt.Print("API Key")
	if existingCreds.APIKey != "" {
		fmt.Printf(" [%s]", maskCredential(existingCreds.APIKey))
	}
	fmt.Print(": ")
	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" && existingCreds.APIKey != "" {
		apiKey = existingCreds.APIKey
	}

	// Validate
	if apiKey == "" {
		return fmt.Errorf("API key is required")
	}

	// Save to global config
	globalCfg.Profiling.Tideways = config.TidewaysCredentials{
		APIKey: apiKey,
	}

	fmt.Println()
	fmt.Print("Saving credentials... ")
	if err := config.SaveGlobalConfig(p.HomeDir, globalCfg); err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to save config: %w", err)
	}
	fmt.Println(cli.Success("done"))

	// Configure daemon if installed
	mgr := tideways.NewManager(p, &tideways.Credentials{
		APIKey: apiKey,
	})

	if mgr.IsDaemonInstalled() {
		fmt.Print("Configuring Tideways daemon... ")
		if err := mgr.ConfigureDaemon(); err != nil {
			fmt.Println(cli.Warning("failed (configure manually)"))
		} else {
			fmt.Println(cli.Success("done"))
		}

		// Restart daemon
		fmt.Print("Restarting Tideways daemon... ")
		if err := mgr.RestartDaemon(); err != nil {
			fmt.Println(cli.Warning("failed (restart manually)"))
		} else {
			fmt.Println(cli.Success("done"))
		}
	}

	fmt.Println()
	cli.PrintSuccess("Tideways configured!")
	fmt.Println()
	cli.PrintInfo("Enable profiling with: magebox tideways on")

	return nil
}

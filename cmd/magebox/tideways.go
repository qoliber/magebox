// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"qoliber/magebox/internal/cli"
	"qoliber/magebox/internal/config"
	"qoliber/magebox/internal/php"
	"qoliber/magebox/internal/tideways"
	"qoliber/magebox/internal/xdebug"
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
	Short: "Configure global Tideways settings",
	Long: `Configures the globally-scoped Tideways settings: the CLI access token
(for the 'tideways' commandline tool) and the daemon environment label
(which stamps all traces from this machine).

The Tideways API key is *per Tideways project*, not per user or machine,
so it is not configured here. Set it per project in .magebox.yaml under
php_ini.tideways.api_key — MageBox renders that into the project's FPM
pool config as a php_admin_value, scoping the key to that one project.`,
	RunE: runTidewaysConfig,
}

// Flags for tideways config
var (
	tidewaysAccessToken string
	tidewaysEnvironment string
)

func init() {
	// Add flags for non-interactive config
	tidewaysConfigCmd.Flags().StringVar(&tidewaysAccessToken, "access-token", "", "Tideways CLI access token (for the tideways commandline tool)")
	tidewaysConfigCmd.Flags().StringVar(&tidewaysEnvironment, "environment", "", "Tideways environment label (default: local_$USER)")

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

	// Load the project config so we can surface whether this project has
	// tideways.api_key set in its php_ini. It is optional — if there is no
	// project here we just skip the per-project line.
	projectCfg, _ := config.LoadFromPath(cwd)

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

	// Credentials / labels
	creds := globalCfg.GetTidewaysCredentials()
	fmt.Println("Configuration:")
	if projectCfg != nil {
		projectKey := projectCfg.PHPINI["tideways.api_key"]
		fmt.Printf("  Project API key (php_ini): %s\n", formatBool(projectKey != ""))
	} else {
		fmt.Printf("  Project API key (php_ini): %s (no project config in cwd)\n", cli.Warning("?"))
	}
	fmt.Printf("  CLI access token:          %s\n", formatBool(globalCfg.HasTidewaysAccessToken()))
	fmt.Printf("  Environment (daemon):      %s\n", cli.Highlight(creds.Environment))
	fmt.Println()

	// Helpful tips
	if !status.DaemonInstalled || !status.ExtensionInstalled[phpVersion] {
		cli.PrintInfo("Install with: magebox tideways install")
	} else if !status.ExtensionEnabled[phpVersion] {
		cli.PrintInfo("Enable with: magebox tideways on")
	} else {
		cli.PrintInfo("Disable with: magebox tideways off")
	}

	if projectCfg != nil && projectCfg.PHPINI["tideways.api_key"] == "" {
		cli.PrintWarning("This project has no Tideways API key set — add it to .magebox.yaml:")
		cli.PrintWarning("  php_ini:")
		cli.PrintWarning("    tideways.api_key: your-project-specific-key")
	}
	if globalCfg.HasLegacyTidewaysAPIKey() {
		cli.PrintWarning("Legacy global api_key detected in ~/.magebox/config.yaml —")
		cli.PrintWarning("  run 'magebox tideways config' to migrate it away.")
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

	// Load existing config
	globalCfg, err := config.LoadGlobalConfig(p.HomeDir)
	if err != nil {
		return err
	}

	existingCreds := globalCfg.GetTidewaysCredentials()
	hadLegacyAPIKey := globalCfg.HasLegacyTidewaysAPIKey()

	// Non-interactive mode is triggered by passing at least one flag.
	nonInteractive := tidewaysAccessToken != "" || tidewaysEnvironment != ""

	var accessToken, environment string

	if nonInteractive {
		accessToken = tidewaysAccessToken
		if accessToken == "" {
			accessToken = existingCreds.AccessToken
		}
		environment = tidewaysEnvironment
		if environment == "" {
			environment = existingCreds.Environment
		}
	} else {
		cli.PrintTitle("Configure Tideways")
		fmt.Println("This command configures the *global* Tideways settings for this")
		fmt.Println("machine: the CLI access token and the daemon environment label.")
		fmt.Println()
		fmt.Println("The Tideways API key is per-project, not global — add it to each")
		fmt.Println("project's .magebox.yaml (or .magebox.local.yaml):")
		fmt.Println()
		fmt.Println("    php_ini:")
		fmt.Println("      tideways.api_key: your-project-specific-key")
		fmt.Println()
		fmt.Println("Find the key on the project's Installation page in the Tideways")
		fmt.Println("dashboard: https://app.tideways.io/o/<organization>/<project>/installation")
		fmt.Println()
		fmt.Println("The Access Token is a personal token for the `tideways` CLI command,")
		fmt.Println("generated at https://app.tideways.io/user/cli-import-settings.")
		fmt.Println()
		fmt.Println("The Environment labels traces from this machine on the Tideways")
		fmt.Println("daemon side — default is local_<username> so local traces don't")
		fmt.Println("land in the `production` bucket. On Linux MageBox installs a systemd")
		fmt.Println("drop-in to pass TIDEWAYS_ENVIRONMENT to the daemon and restarts it.")
		fmt.Println()

		reader := bufio.NewReader(os.Stdin)

		// Prompt for Access Token (optional — only needed for the CLI tool)
		fmt.Print("Access Token (for tideways CLI, optional)")
		if existingCreds.AccessToken != "" {
			fmt.Printf(" [%s]", maskCredential(existingCreds.AccessToken))
		}
		fmt.Print(": ")
		accessToken, _ = reader.ReadString('\n')
		accessToken = strings.TrimSpace(accessToken)
		if accessToken == "" && existingCreds.AccessToken != "" {
			accessToken = existingCreds.AccessToken
		}

		// Prompt for Environment (defaults to local_$USER)
		envDefault := existingCreds.Environment
		if envDefault == "" {
			envDefault = config.DefaultTidewaysEnvironment()
		}
		fmt.Printf("Environment [%s]: ", envDefault)
		environment, _ = reader.ReadString('\n')
		environment = strings.TrimSpace(environment)
		if environment == "" {
			environment = envDefault
		}
	}

	if environment == "" {
		environment = config.DefaultTidewaysEnvironment()
	}

	// Save to global config. Explicitly clear the deprecated api_key field so
	// a stale value from an older MageBox version is removed from disk.
	globalCfg.Profiling.Tideways = config.TidewaysCredentials{
		AccessToken: accessToken,
		Environment: environment,
	}

	fmt.Println()
	if hadLegacyAPIKey {
		cli.PrintWarning("A legacy 'profiling.tideways.api_key' was found in ~/.magebox/config.yaml.")
		cli.PrintWarning("The API key is per project — move it to each project's .magebox.yaml")
		cli.PrintWarning("under 'php_ini.tideways.api_key'. MageBox is removing it from the global config.")
	}
	fmt.Print("Saving credentials... ")
	if err := config.SaveGlobalConfig(p.HomeDir, globalCfg); err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to save config: %w", err)
	}
	fmt.Println(cli.Success("done"))

	mgr := tideways.NewManager(p, &tideways.Credentials{
		AccessToken: accessToken,
		Environment: environment,
	})

	// Migration: strip any tideways.api_key / tideways.environment lines that
	// older MageBox versions wrote to the global PHP extension ini. These are
	// harmless (the FPM pool's php_admin_value wins) but confusing. Reload
	// PHP-FPM afterward if the file actually changed.
	cleanedAny := false
	for _, v := range p.GetInstalledPHPVersions() {
		if !mgr.IsExtensionInstalled(v) {
			continue
		}
		changed, cleanErr := mgr.CleanLegacyExtensionDirectives(v)
		if cleanErr != nil {
			cli.PrintWarning("  could not clean stale Tideways directives from PHP %s: %v", v, cleanErr)
			continue
		}
		if changed {
			fmt.Printf("Removed legacy Tideways directives from PHP %s ini %s\n", v, cli.Success("✓"))
			cleanedAny = true
		}
	}

	// Configure the daemon environment label. This is a *daemon-level*
	// setting — the PHP extension transmits traces to the local daemon, and
	// the daemon stamps them with whatever --env (or TIDEWAYS_ENVIRONMENT)
	// it was started with. On Linux we install a systemd drop-in and
	// restart the daemon. On macOS this is left manual.
	fmt.Printf("Configuring Tideways daemon environment (%s)... ", cli.Highlight(environment))
	if err := mgr.WriteDaemonEnvironment(environment); err != nil {
		fmt.Println(cli.Warning("skipped"))
		cli.PrintWarning("  %v", err)
	} else {
		fmt.Println(cli.Success("done"))
	}

	// Reload PHP-FPM for the current project's PHP version if we cleaned
	// anything, so the stale directives are dropped from the running workers.
	if cleanedAny {
		if cwd, cwdErr := getCwd(); cwdErr == nil {
			if phpVersion, phpErr := getProjectPHPVersion(cwd); phpErr == nil && mgr.IsExtensionInstalled(phpVersion) {
				fmt.Printf("Reloading PHP-FPM %s... ", phpVersion)
				fpmCtrl := php.NewFPMController(p, phpVersion)
				if err := fpmCtrl.Reload(); err != nil {
					fmt.Println(cli.Warning("failed (reload manually)"))
				} else {
					fmt.Println(cli.Success("done"))
				}
			}
		}
	}

	// Import the CLI access token if provided. The `tideways` CLI stores it
	// locally after a successful import.
	if accessToken != "" {
		fmt.Print("Importing Tideways CLI access token... ")
		if err := mgr.ImportCLIToken(); err != nil {
			fmt.Println(cli.Warning("skipped"))
			cli.PrintWarning("  %v", err)
			cli.PrintInfo("  Run 'tideways import <token>' manually once the CLI is installed")
		} else {
			fmt.Println(cli.Success("done"))
		}
	}

	fmt.Println()
	cli.PrintSuccess("Tideways configured!")
	fmt.Println()
	cli.PrintInfo("Set the API key per project in .magebox.yaml under php_ini.tideways.api_key")
	cli.PrintInfo("Enable profiling with: magebox tideways on")

	return nil
}

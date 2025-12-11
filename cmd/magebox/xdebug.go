// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/qoliber/magebox/internal/cli"
	"github.com/qoliber/magebox/internal/config"
	"github.com/qoliber/magebox/internal/php"
	"github.com/qoliber/magebox/internal/xdebug"
)

var xdebugCmd = &cobra.Command{
	Use:   "xdebug",
	Short: "Manage Xdebug for PHP",
	Long: `Enable, disable, or check status of Xdebug for PHP debugging.

Use 'magebox xdebug on' to enable Xdebug for the current project's PHP version.
Use 'magebox xdebug off' to disable Xdebug.
Use 'magebox xdebug status' to check current Xdebug status.`,
	RunE: runXdebugStatus,
}

var xdebugOnCmd = &cobra.Command{
	Use:   "on",
	Short: "Enable Xdebug",
	Long:  `Enables Xdebug for the project's PHP version and restarts PHP-FPM.`,
	RunE:  runXdebugOn,
}

var xdebugOffCmd = &cobra.Command{
	Use:   "off",
	Short: "Disable Xdebug",
	Long:  `Disables Xdebug for the project's PHP version and restarts PHP-FPM.`,
	RunE:  runXdebugOff,
}

var xdebugStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Xdebug status",
	Long:  `Shows the current Xdebug status for the project's PHP version.`,
	RunE:  runXdebugStatus,
}

func runXdebugOn(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	p, err := getPlatform()
	if err != nil {
		return err
	}

	// Get PHP version from project config
	phpVersion, err := getProjectPHPVersion(cwd)
	if err != nil {
		return err
	}

	cli.PrintTitle("Enabling Xdebug")
	fmt.Printf("PHP version: %s\n", cli.Highlight(phpVersion))
	fmt.Println()

	mgr := xdebug.NewManager(p)

	// Check if installed
	if !mgr.IsInstalled(phpVersion) {
		cli.PrintWarning("Xdebug is not installed for PHP %s", phpVersion)
		cli.PrintInfo("Install with: pecl install xdebug")
		return nil
	}

	// Enable Xdebug
	fmt.Print("Enabling Xdebug... ")
	if err := mgr.Enable(phpVersion); err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to enable Xdebug: %w", err)
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
	cli.PrintSuccess("Xdebug enabled!")
	fmt.Println()
	fmt.Println("Configuration:")
	fmt.Printf("  Mode:        %s\n", cli.Highlight("debug"))
	fmt.Printf("  Client Host: %s\n", cli.Highlight("127.0.0.1"))
	fmt.Printf("  Client Port: %s\n", cli.Highlight("9003"))
	fmt.Printf("  IDE Key:     %s\n", cli.Highlight("PHPSTORM"))
	fmt.Println()
	cli.PrintInfo("Add XDEBUG_TRIGGER=1 to enable debugging per-request")

	return nil
}

func runXdebugOff(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	p, err := getPlatform()
	if err != nil {
		return err
	}

	// Get PHP version from project config
	phpVersion, err := getProjectPHPVersion(cwd)
	if err != nil {
		return err
	}

	cli.PrintTitle("Disabling Xdebug")
	fmt.Printf("PHP version: %s\n", cli.Highlight(phpVersion))
	fmt.Println()

	mgr := xdebug.NewManager(p)

	// Check if installed
	if !mgr.IsInstalled(phpVersion) {
		cli.PrintInfo("Xdebug is not installed for PHP %s", phpVersion)
		return nil
	}

	// Disable Xdebug
	fmt.Print("Disabling Xdebug... ")
	if err := mgr.Disable(phpVersion); err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to disable Xdebug: %w", err)
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
	cli.PrintSuccess("Xdebug disabled!")
	cli.PrintInfo("Xdebug overhead removed - PHP will run faster")

	return nil
}

func runXdebugStatus(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	p, err := getPlatform()
	if err != nil {
		return err
	}

	// Get PHP version from project config
	phpVersion, err := getProjectPHPVersion(cwd)
	if err != nil {
		return err
	}

	cli.PrintTitle("Xdebug Status")
	fmt.Printf("PHP version: %s\n", cli.Highlight(phpVersion))
	fmt.Println()

	mgr := xdebug.NewManager(p)
	status := mgr.GetStatus(phpVersion)

	fmt.Printf("Installed: %s\n", formatBool(status.Installed))
	fmt.Printf("Enabled:   %s\n", formatBool(status.Enabled))

	if status.Installed && status.Enabled {
		fmt.Printf("Mode:      %s\n", cli.Highlight(status.Mode))
	}

	if status.IniPath != "" {
		fmt.Printf("INI Path:  %s\n", cli.Highlight(status.IniPath))
	}

	fmt.Println()

	if !status.Installed {
		cli.PrintInfo("Install Xdebug with: pecl install xdebug")
	} else if !status.Enabled {
		cli.PrintInfo("Enable with: magebox xdebug on")
	} else {
		cli.PrintInfo("Disable with: magebox xdebug off")
	}

	return nil
}

// getProjectPHPVersion gets the PHP version from the project config
func getProjectPHPVersion(cwd string) (string, error) {
	cfg, err := config.LoadFromPath(cwd)
	if err != nil {
		return "", fmt.Errorf("no project config found - run 'magebox init' first")
	}

	if cfg.PHP == "" {
		return "", fmt.Errorf("no PHP version specified in project config")
	}

	return cfg.PHP, nil
}

// formatBool formats a boolean as a colored yes/no
func formatBool(b bool) string {
	if b {
		return cli.Success("yes")
	}
	return cli.Warning("no")
}

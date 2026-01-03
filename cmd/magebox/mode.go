// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"qoliber/magebox/internal/blackfire"
	"qoliber/magebox/internal/cli"
	"qoliber/magebox/internal/config"
	"qoliber/magebox/internal/php"
	"qoliber/magebox/internal/xdebug"
)

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Switch to development mode",
	Long: `Switches the project to development mode optimized for debugging:

  - OPcache: Disabled (immediate code changes)
  - Xdebug: Enabled (step debugging)
  - Blackfire: Disabled (conflicts with Xdebug)

This mode is optimized for active development with debugging support.
Use 'magebox prod' to switch back to production-like mode.`,
	RunE: runDevMode,
}

var prodCmd = &cobra.Command{
	Use:   "prod",
	Short: "Switch to production mode",
	Long: `Switches the project to production-like mode optimized for performance:

  - OPcache: Enabled (faster execution)
  - Xdebug: Disabled (no overhead)
  - Blackfire: Disabled (enable manually when needed)

This mode is optimized for testing production-like performance.
Use 'magebox dev' to switch back to development mode.`,
	RunE: runProdMode,
}

func init() {
	rootCmd.AddCommand(devCmd)
	rootCmd.AddCommand(prodCmd)
}

func runDevMode(cmd *cobra.Command, args []string) error {
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

	cli.PrintTitle("Switching to Development Mode")
	fmt.Printf("PHP version: %s\n", cli.Highlight(phpVersion))
	fmt.Println()

	// Load or create local config
	localCfg, err := config.LoadLocalConfig(cwd)
	if err != nil {
		return fmt.Errorf("failed to load local config: %w", err)
	}

	// Initialize PHPINI if nil
	if localCfg.PHPINI == nil {
		localCfg.PHPINI = make(map[string]string)
	}

	// Disable OPcache
	fmt.Print("Disabling OPcache... ")
	localCfg.PHPINI["opcache.enable"] = "0"
	fmt.Println(cli.Success("done"))

	// Save local config
	if err := config.SaveLocalConfig(cwd, localCfg); err != nil {
		return fmt.Errorf("failed to save local config: %w", err)
	}

	// Disable Blackfire first (conflicts with Xdebug)
	blackfireMgr := blackfire.NewManager(p, nil)
	if blackfireMgr.IsExtensionEnabled(phpVersion) {
		fmt.Print("Disabling Blackfire (conflicts with Xdebug)... ")
		if err := blackfireMgr.Disable(phpVersion); err != nil {
			fmt.Println(cli.Warning("failed"))
		} else {
			fmt.Println(cli.Success("done"))
		}
	}

	// Enable Xdebug
	xdebugMgr := xdebug.NewManager(p)
	if xdebugMgr.IsInstalled(phpVersion) {
		if !xdebugMgr.IsEnabled(phpVersion) {
			fmt.Print("Enabling Xdebug... ")
			if err := xdebugMgr.Enable(phpVersion); err != nil {
				fmt.Println(cli.Warning("failed"))
			} else {
				fmt.Println(cli.Success("done"))
			}
		} else {
			fmt.Println("Xdebug already enabled " + cli.Success("✓"))
		}
	} else {
		cli.PrintWarning("Xdebug is not installed for PHP %s", phpVersion)
	}

	// Regenerate pool config with updated settings
	cfg, err := config.LoadFromPath(cwd)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Print("Regenerating PHP-FPM pool... ")
	poolGen := php.NewPoolGenerator(p)
	if err := poolGen.Generate(cfg.Name, cwd, cfg.PHP, cfg.Env, cfg.PHPINI, true); err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to regenerate pool: %w", err)
	}
	fmt.Println(cli.Success("done"))

	// Reload PHP-FPM
	fmt.Print("Reloading PHP-FPM... ")
	fpmCtrl := php.NewFPMController(p, phpVersion)
	if err := fpmCtrl.Reload(); err != nil {
		fmt.Println(cli.Warning("failed (may need manual restart)"))
	} else {
		fmt.Println(cli.Success("done"))
	}

	fmt.Println()
	cli.PrintSuccess("Development mode enabled!")
	fmt.Println()
	fmt.Println("Configuration:")
	fmt.Printf("  OPcache:   %s\n", cli.Warning("disabled"))
	fmt.Printf("  Xdebug:    %s\n", cli.Success("enabled"))
	fmt.Printf("  Blackfire: %s\n", cli.Warning("disabled"))
	fmt.Println()
	cli.PrintInfo("Code changes will take effect immediately")
	cli.PrintInfo("Add XDEBUG_TRIGGER=1 to enable debugging per-request")

	return nil
}

func runProdMode(cmd *cobra.Command, args []string) error {
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

	cli.PrintTitle("Switching to Production Mode")
	fmt.Printf("PHP version: %s\n", cli.Highlight(phpVersion))
	fmt.Println()

	// Load or create local config
	localCfg, err := config.LoadLocalConfig(cwd)
	if err != nil {
		return fmt.Errorf("failed to load local config: %w", err)
	}

	// Initialize PHPINI if nil
	if localCfg.PHPINI == nil {
		localCfg.PHPINI = make(map[string]string)
	}

	// Enable OPcache (remove the disable override, defaults to enabled in pool.conf)
	fmt.Print("Enabling OPcache... ")
	delete(localCfg.PHPINI, "opcache.enable")
	// Also clean up empty PHPINI to keep config clean
	if len(localCfg.PHPINI) == 0 {
		localCfg.PHPINI = nil
	}
	fmt.Println(cli.Success("done"))

	// Save local config
	if err := config.SaveLocalConfig(cwd, localCfg); err != nil {
		return fmt.Errorf("failed to save local config: %w", err)
	}

	// Disable Xdebug
	xdebugMgr := xdebug.NewManager(p)
	if xdebugMgr.IsEnabled(phpVersion) {
		fmt.Print("Disabling Xdebug... ")
		if err := xdebugMgr.Disable(phpVersion); err != nil {
			fmt.Println(cli.Warning("failed"))
		} else {
			fmt.Println(cli.Success("done"))
		}
	} else {
		fmt.Println("Xdebug already disabled " + cli.Success("✓"))
	}

	// Disable Blackfire
	blackfireMgr := blackfire.NewManager(p, nil)
	if blackfireMgr.IsExtensionEnabled(phpVersion) {
		fmt.Print("Disabling Blackfire... ")
		if err := blackfireMgr.Disable(phpVersion); err != nil {
			fmt.Println(cli.Warning("failed"))
		} else {
			fmt.Println(cli.Success("done"))
		}
	}

	// Regenerate pool config with updated settings
	cfg, err := config.LoadFromPath(cwd)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Print("Regenerating PHP-FPM pool... ")
	poolGen := php.NewPoolGenerator(p)
	if err := poolGen.Generate(cfg.Name, cwd, cfg.PHP, cfg.Env, cfg.PHPINI, true); err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to regenerate pool: %w", err)
	}
	fmt.Println(cli.Success("done"))

	// Reload PHP-FPM
	fmt.Print("Reloading PHP-FPM... ")
	fpmCtrl := php.NewFPMController(p, phpVersion)
	if err := fpmCtrl.Reload(); err != nil {
		fmt.Println(cli.Warning("failed (may need manual restart)"))
	} else {
		fmt.Println(cli.Success("done"))
	}

	fmt.Println()
	cli.PrintSuccess("Production mode enabled!")
	fmt.Println()
	fmt.Println("Configuration:")
	fmt.Printf("  OPcache:   %s\n", cli.Success("enabled"))
	fmt.Printf("  Xdebug:    %s\n", cli.Warning("disabled"))
	fmt.Printf("  Blackfire: %s\n", cli.Warning("disabled"))
	fmt.Println()
	cli.PrintInfo("OPcache will cache PHP bytecode for faster execution")
	cli.PrintInfo("Enable Blackfire with: magebox blackfire on")

	return nil
}

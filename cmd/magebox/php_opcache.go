package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"qoliber/magebox/internal/cli"
	"qoliber/magebox/internal/config"
	"qoliber/magebox/internal/php"
	"qoliber/magebox/internal/project"
)

var opcacheCmd = &cobra.Command{
	Use:   "opcache",
	Short: "Manage OPcache settings",
	Long:  "Enable, disable, or clear PHP OPcache for the current project",
}

var opcacheEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable OPcache",
	Long:  "Enable OPcache for the current project (sets opcache.enable=1)",
	RunE:  runOpcacheEnable,
}

var opcacheDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable OPcache",
	Long:  "Disable OPcache for the current project (sets opcache.enable=0)",
	RunE:  runOpcacheDisable,
}

var opcacheClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear OPcache",
	Long:  "Clear OPcache by reloading PHP-FPM",
	RunE:  runOpcacheClear,
}

var opcacheStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show OPcache status",
	Long:  "Show current OPcache settings for the project",
	RunE:  runOpcacheStatus,
}

func init() {
	phpCmd.AddCommand(opcacheCmd)
	opcacheCmd.AddCommand(opcacheEnableCmd)
	opcacheCmd.AddCommand(opcacheDisableCmd)
	opcacheCmd.AddCommand(opcacheClearCmd)
	opcacheCmd.AddCommand(opcacheStatusCmd)
}

func runOpcacheEnable(cmd *cobra.Command, args []string) error {
	return setOpcacheSetting("1", "enabled")
}

func runOpcacheDisable(cmd *cobra.Command, args []string) error {
	return setOpcacheSetting("0", "disabled")
}

func setOpcacheSetting(value, description string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	p, err := getPlatform()
	if err != nil {
		return err
	}

	cfg, ok := loadProjectConfig(cwd)
	if !ok {
		return nil
	}

	// Update php_ini in local config
	localCfg, err := config.LoadLocalConfig(cwd)
	if err != nil {
		// Start with empty local config
		localCfg = &config.LocalConfig{
			PHPINI: make(map[string]string),
		}
	}

	if localCfg.PHPINI == nil {
		localCfg.PHPINI = make(map[string]string)
	}

	localCfg.PHPINI["opcache.enable"] = value

	if err := config.SaveLocalConfig(cwd, localCfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	cli.PrintSuccess("OPcache %s", description)

	// Restart to apply changes
	fmt.Println("Reloading PHP-FPM to apply changes...")
	fpmController := php.NewFPMController(p, cfg.PHP)

	// Regenerate pool config with new settings
	mgr := project.NewManager(p)
	if err := mgr.RegenerateConfigs(cwd); err != nil {
		cli.PrintWarning("Failed to regenerate configs: %v", err)
	}

	if err := fpmController.Reload(); err != nil {
		cli.PrintWarning("Failed to reload PHP-FPM: %v", err)
	} else {
		cli.PrintSuccess("PHP-FPM reloaded")
	}

	return nil
}

func runOpcacheClear(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	p, err := getPlatform()
	if err != nil {
		return err
	}

	cfg, ok := loadProjectConfig(cwd)
	if !ok {
		return nil
	}

	fmt.Println("Reloading PHP-FPM to clear OPcache...")
	fpmController := php.NewFPMController(p, cfg.PHP)

	if err := fpmController.Reload(); err != nil {
		return fmt.Errorf("failed to reload PHP-FPM: %w", err)
	}

	cli.PrintSuccess("OPcache cleared (PHP-FPM reloaded)")
	return nil
}

func runOpcacheStatus(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	cfg, ok := loadProjectConfig(cwd)
	if !ok {
		return nil
	}

	// Get merged settings (defaults + custom)
	merged := php.GetMergedPHPINI(cfg.PHPINI)

	cli.PrintTitle("OPcache Settings")
	fmt.Println()

	// Show opcache-related settings
	opcacheSettings := []string{
		"opcache.enable",
		"opcache.enable_cli",
		"opcache.memory_consumption",
		"opcache.max_accelerated_files",
		"opcache.validate_timestamps",
		"opcache.revalidate_freq",
		"opcache.interned_strings_buffer",
		"opcache.preload",
		"opcache.preload_user",
	}

	for _, key := range opcacheSettings {
		if value, exists := merged[key]; exists {
			fmt.Printf("  %s%s%s = %s\n", cli.Dim, key, cli.Reset, cli.Highlight(value))
		}
	}

	// Show custom overrides
	fmt.Println()
	fmt.Println(cli.Header("Custom Overrides"))
	hasCustom := false
	for _, key := range opcacheSettings {
		if value, exists := cfg.PHPINI[key]; exists {
			fmt.Printf("  %s%s%s = %s\n", cli.Dim, key, cli.Reset, cli.Highlight(value))
			hasCustom = true
		}
	}
	if !hasCustom {
		fmt.Println("  (none)")
	}

	return nil
}

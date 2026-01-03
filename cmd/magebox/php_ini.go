// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"qoliber/magebox/internal/cli"
	"qoliber/magebox/internal/config"
	"qoliber/magebox/internal/php"
	"qoliber/magebox/internal/project"
)

var phpIniCmd = &cobra.Command{
	Use:   "ini",
	Short: "Manage PHP INI settings",
	Long: `Manage PHP INI settings for the current project.

Settings are stored in .magebox.local.yaml and override defaults.
Changes are applied immediately by restarting PHP-FPM.`,
}

var phpIniSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a PHP INI value",
	Long: `Set a PHP INI value for the current project.

Examples:
  mbox php ini set opcache.validate_timestamps 0
  mbox php ini set memory_limit 2G
  mbox php ini set opcache.preload /path/to/preload.php`,
	Args: cobra.ExactArgs(2),
	RunE: runPhpIniSet,
}

var phpIniListCmd = &cobra.Command{
	Use:   "list",
	Short: "List PHP INI settings",
	Long:  `List all PHP INI settings for the current project (merged from defaults and overrides).`,
	RunE:  runPhpIniList,
}

var phpIniUnsetCmd = &cobra.Command{
	Use:   "unset <key>",
	Short: "Remove a PHP INI override",
	Long: `Remove a PHP INI override from .magebox.local.yaml.

This reverts the setting to its default value.

Example:
  mbox php ini unset opcache.validate_timestamps`,
	Args: cobra.ExactArgs(1),
	RunE: runPhpIniUnset,
}

var phpIniGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a PHP INI value",
	Long: `Get the current value of a PHP INI setting.

Example:
  mbox php ini get opcache.validate_timestamps`,
	Args: cobra.ExactArgs(1),
	RunE: runPhpIniGet,
}

func init() {
	phpCmd.AddCommand(phpIniCmd)
	phpIniCmd.AddCommand(phpIniSetCmd)
	phpIniCmd.AddCommand(phpIniListCmd)
	phpIniCmd.AddCommand(phpIniUnsetCmd)
	phpIniCmd.AddCommand(phpIniGetCmd)
}

func runPhpIniSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	cwd, err := getCwd()
	if err != nil {
		return err
	}

	// Load current config to verify project exists
	_, ok := loadProjectConfig(cwd)
	if !ok {
		return nil
	}

	// Load local config
	localCfg, err := config.LoadLocalConfig(cwd)
	if err != nil {
		return fmt.Errorf("failed to load local config: %w", err)
	}

	// Initialize PHPINI map if nil
	if localCfg.PHPINI == nil {
		localCfg.PHPINI = make(map[string]string)
	}

	// Set the value
	localCfg.PHPINI[key] = value

	// Save local config
	if err := config.SaveLocalConfig(cwd, localCfg); err != nil {
		return fmt.Errorf("failed to save local config: %w", err)
	}

	cli.PrintSuccess("Set %s = %s", key, value)
	fmt.Println()

	// Show config file path
	localConfigPath := filepath.Join(cwd, config.LocalConfigFileName)
	fmt.Printf("Saved to: %s\n", color.CyanString(localConfigPath))
	fmt.Println()

	// Restart PHP-FPM to apply changes
	return restartPHPFPM(cwd)
}

func runPhpIniList(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	// Load merged config
	cfg, ok := loadProjectConfig(cwd)
	if !ok {
		return nil
	}

	// Load local config to show which are overrides
	localCfg, err := config.LoadLocalConfig(cwd)
	if err != nil {
		localCfg = &config.LocalConfig{}
	}

	// Get merged PHPINI (defaults + overrides)
	mergedIni := php.GetMergedPHPINI(cfg.PHPINI)

	fmt.Println(cli.Header("PHP INI Settings"))
	fmt.Println()

	// Show config file paths
	mainConfigPath := filepath.Join(cwd, config.ConfigFileName)
	localConfigPath := filepath.Join(cwd, config.LocalConfigFileName)
	fmt.Printf("Main config:  %s\n", color.CyanString(mainConfigPath))
	fmt.Printf("Local config: %s\n", color.CyanString(localConfigPath))
	fmt.Println()

	// Sort keys for consistent output
	keys := make([]string, 0, len(mergedIni))
	for k := range mergedIni {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Group by category
	categories := make(map[string][]string)
	for _, k := range keys {
		parts := strings.SplitN(k, ".", 2)
		category := "other"
		if len(parts) > 1 {
			category = parts[0]
		}
		categories[category] = append(categories[category], k)
	}

	// Sort categories
	catKeys := make([]string, 0, len(categories))
	for c := range categories {
		catKeys = append(catKeys, c)
	}
	sort.Strings(catKeys)

	for _, cat := range catKeys {
		fmt.Printf("%s:\n", color.YellowString(cat))
		for _, k := range categories[cat] {
			v := mergedIni[k]
			// Check if this is an override
			isOverride := false
			if localCfg.PHPINI != nil {
				if _, ok := localCfg.PHPINI[k]; ok {
					isOverride = true
				}
			}
			if cfg.PHPINI != nil {
				if _, ok := cfg.PHPINI[k]; ok {
					isOverride = true
				}
			}

			if isOverride {
				fmt.Printf("  %s = %s %s\n", k, color.GreenString(v), color.CyanString("(override)"))
			} else {
				fmt.Printf("  %s = %s\n", k, v)
			}
		}
	}

	return nil
}

func runPhpIniUnset(cmd *cobra.Command, args []string) error {
	key := args[0]

	cwd, err := getCwd()
	if err != nil {
		return err
	}

	// Load current config to verify project exists
	_, ok := loadProjectConfig(cwd)
	if !ok {
		return nil
	}

	// Load local config
	localCfg, err := config.LoadLocalConfig(cwd)
	if err != nil {
		return fmt.Errorf("failed to load local config: %w", err)
	}

	// Check if key exists
	if localCfg.PHPINI == nil {
		cli.PrintWarning("No PHP INI overrides in local config")
		return nil
	}

	if _, exists := localCfg.PHPINI[key]; !exists {
		cli.PrintWarning("%s is not set in local config", key)
		return nil
	}

	// Remove the key
	delete(localCfg.PHPINI, key)

	// Save local config
	if err := config.SaveLocalConfig(cwd, localCfg); err != nil {
		return fmt.Errorf("failed to save local config: %w", err)
	}

	cli.PrintSuccess("Removed %s from local config", key)
	fmt.Println()

	// Restart PHP-FPM to apply changes
	return restartPHPFPM(cwd)
}

func runPhpIniGet(cmd *cobra.Command, args []string) error {
	key := args[0]

	cwd, err := getCwd()
	if err != nil {
		return err
	}

	// Load merged config
	cfg, ok := loadProjectConfig(cwd)
	if !ok {
		return nil
	}

	// Get merged PHPINI (defaults + overrides)
	mergedIni := php.GetMergedPHPINI(cfg.PHPINI)

	if value, exists := mergedIni[key]; exists {
		fmt.Printf("%s = %s\n", key, value)
	} else {
		cli.PrintWarning("%s is not set", key)
	}

	return nil
}

// restartPHPFPM restarts PHP-FPM to apply INI changes
func restartPHPFPM(cwd string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	fmt.Print("Restarting PHP-FPM to apply changes... ")

	mgr := project.NewManager(p)

	// Stop and start to regenerate pool config
	if err := mgr.Stop(cwd); err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to stop: %w", err)
	}

	if _, err := mgr.Start(cwd); err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to start: %w", err)
	}

	fmt.Println(cli.Success("done"))
	return nil
}

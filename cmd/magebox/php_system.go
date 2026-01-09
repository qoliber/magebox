// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"

	"github.com/spf13/cobra"

	"qoliber/magebox/internal/cli"
	"qoliber/magebox/internal/php"
)

var phpSystemCmd = &cobra.Command{
	Use:   "system",
	Short: "Manage PHP system-level (PHP_INI_SYSTEM) settings",
	Long: `Manage PHP system-level settings that apply to ALL projects.

PHP_INI_SYSTEM settings (like opcache.preload, opcache.jit, opcache.memory_consumption)
can only be set globally in php.ini, not per-pool. These settings require a symlink
to be created in the PHP scan directory.

Use 'mbox php system' to view current settings and activation status.
Use 'mbox php system enable' to activate the settings.
Use 'mbox php system disable' to deactivate the settings.
Use 'mbox php system list' to show all PHP_INI_SYSTEM setting names.`,
	RunE: runPhpSystem,
}

var phpSystemEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable PHP system settings",
	Long:  `Create a symlink to activate MageBox PHP system settings.`,
	RunE:  runPhpSystemEnable,
}

var phpSystemDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable PHP system settings",
	Long:  `Remove the symlink to deactivate MageBox PHP system settings.`,
	RunE:  runPhpSystemDisable,
}

var phpSystemListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all PHP_INI_SYSTEM setting names",
	Long:  `Show all PHP INI settings that are classified as PHP_INI_SYSTEM and can only be set globally.`,
	RunE:  runPhpSystemList,
}

func init() {
	phpCmd.AddCommand(phpSystemCmd)
	phpSystemCmd.AddCommand(phpSystemEnableCmd)
	phpSystemCmd.AddCommand(phpSystemDisableCmd)
	phpSystemCmd.AddCommand(phpSystemListCmd)
}

func runPhpSystem(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	// Get PHP version from project or use default
	phpVersion := "8.3" // default
	cwd, err := getCwd()
	if err == nil {
		if cfg, ok := loadProjectConfig(cwd); ok {
			phpVersion = cfg.PHP
		}
	}

	sysMgr := php.NewSystemINIManager(p)
	owner, err := sysMgr.GetCurrentOwner(phpVersion)
	if err != nil {
		cli.PrintError("Failed to get system INI owner: %v", err)
		return nil
	}

	cli.PrintTitle("PHP %s System Settings (PHP_INI_SYSTEM)", phpVersion)
	fmt.Println()

	if owner == nil {
		cli.PrintInfo("No system settings configured")
		fmt.Println()
		fmt.Println("System settings are configured per-project in .magebox.yaml under php_ini.")
		fmt.Println("Settings like opcache.preload, opcache.jit, etc. are automatically")
		fmt.Println("separated and written to a system INI file.")
		return nil
	}

	// Show owner info
	fmt.Printf("Owner:      %s\n", cli.Highlight(owner.ProjectName))
	fmt.Printf("Path:       %s\n", owner.ProjectPath)
	fmt.Printf("Updated:    %s\n", owner.UpdatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Config:     %s\n", sysMgr.GetSystemINIPath(phpVersion))
	fmt.Println()

	// Show settings
	fmt.Println(cli.Header("Settings"))
	keys := make([]string, 0, len(owner.Settings))
	for k := range owner.Settings {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Printf("  %s = %s\n", key, owner.Settings[key])
	}
	fmt.Println()

	// Show activation status
	fmt.Println(cli.Header("Activation Status"))
	symlinkPath := sysMgr.GetSymlinkPath(phpVersion)
	if sysMgr.IsSymlinkActive(phpVersion) {
		fmt.Printf("  %s Active\n", cli.Success("✓"))
		fmt.Printf("  Symlink: %s\n", symlinkPath)
	} else {
		fmt.Printf("  %s Not active\n", cli.Error("✗"))
		fmt.Println()
		fmt.Println("To activate, run:")
		fmt.Printf("  mbox php system enable\n")
		fmt.Println()
		fmt.Println("Or manually:")
		fmt.Printf("  %s\n", sysMgr.GetEnableCommand(phpVersion))
	}

	return nil
}

func runPhpSystemEnable(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	// Get PHP version from project or use default
	phpVersion := "8.3"
	cwd, err := getCwd()
	if err == nil {
		if cfg, ok := loadProjectConfig(cwd); ok {
			phpVersion = cfg.PHP
		}
	}

	sysMgr := php.NewSystemINIManager(p)

	// Check if there are settings to enable
	owner, _ := sysMgr.GetCurrentOwner(phpVersion)
	if owner == nil {
		cli.PrintInfo("No system settings to enable for PHP %s", phpVersion)
		return nil
	}

	// Check if already active
	if sysMgr.IsSymlinkActive(phpVersion) {
		cli.PrintInfo("System settings already active for PHP %s", phpVersion)
		return nil
	}

	// Create symlink using sudo
	iniPath := sysMgr.GetSystemINIPath(phpVersion)
	symlinkPath := sysMgr.GetSymlinkPath(phpVersion)

	fmt.Printf("Creating symlink: %s -> %s\n", symlinkPath, iniPath)

	execCmd := exec.Command("sudo", "ln", "-sf", iniPath, symlinkPath)
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	if err := execCmd.Run(); err != nil {
		cli.PrintError("Failed to create symlink: %v", err)
		fmt.Println()
		fmt.Println("Try running manually:")
		fmt.Printf("  %s\n", sysMgr.GetEnableCommand(phpVersion))
		return nil
	}

	cli.PrintSuccess("System settings enabled for PHP %s", phpVersion)
	fmt.Println()
	fmt.Println("Restart PHP-FPM to apply changes:")
	fmt.Println("  mbox restart php")

	return nil
}

func runPhpSystemDisable(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	// Get PHP version from project or use default
	phpVersion := "8.3"
	cwd, err := getCwd()
	if err == nil {
		if cfg, ok := loadProjectConfig(cwd); ok {
			phpVersion = cfg.PHP
		}
	}

	sysMgr := php.NewSystemINIManager(p)

	// Check if active
	if !sysMgr.IsSymlinkActive(phpVersion) {
		cli.PrintInfo("System settings already disabled for PHP %s", phpVersion)
		return nil
	}

	// Remove symlink using sudo
	symlinkPath := sysMgr.GetSymlinkPath(phpVersion)

	fmt.Printf("Removing symlink: %s\n", symlinkPath)

	execCmd := exec.Command("sudo", "rm", "-f", symlinkPath)
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	if err := execCmd.Run(); err != nil {
		cli.PrintError("Failed to remove symlink: %v", err)
		fmt.Println()
		fmt.Println("Try running manually:")
		fmt.Printf("  %s\n", sysMgr.GetDisableCommand(phpVersion))
		return nil
	}

	cli.PrintSuccess("System settings disabled for PHP %s", phpVersion)
	fmt.Println()
	fmt.Println("Restart PHP-FPM to apply changes:")
	fmt.Println("  mbox restart php")

	return nil
}

func runPhpSystemList(cmd *cobra.Command, args []string) error {
	cli.PrintTitle("PHP_INI_SYSTEM Settings")
	fmt.Println()
	fmt.Println("These settings can only be set in php.ini, not per-pool:")
	fmt.Println()

	settings := php.GetSystemSettingsList()
	for _, s := range settings {
		fmt.Printf("  • %s\n", s)
	}
	fmt.Println()
	fmt.Printf("Total: %d settings\n", len(settings))

	return nil
}

// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"qoliber/magebox/internal/config"
	"qoliber/magebox/internal/lib"
	libconfig "qoliber/magebox/internal/lib/config"
	"github.com/spf13/cobra"
)

var libCmd = &cobra.Command{
	Use:   "lib",
	Short: "Manage MageBox configuration library",
	Long: `Manage the MageBox configuration library (magebox-lib).

The library contains YAML configuration files for different platforms
(Fedora, Ubuntu, Arch, macOS) that define how MageBox installs and
configures services.

The library is stored in ~/.magebox/yaml and can be updated independently
of the MageBox binary.`,
}

var libUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update the configuration library",
	Long:  `Pull the latest configuration files from the magebox-lib repository.`,
	RunE:  runLibUpdate,
}

var libStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show library status",
	Long:  `Show the current status of the configuration library including version and update availability.`,
	RunE:  runLibStatus,
}

var libResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset library to upstream",
	Long:  `Discard all local changes and reset the library to the upstream version.`,
	RunE:  runLibReset,
}

var libPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show library path",
	Long:  `Show the filesystem path to the configuration library.`,
	RunE:  runLibPath,
}

var libListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available installers",
	Long:  `List all available platform installer configurations.`,
	RunE:  runLibList,
}

var libTemplatesCmd = &cobra.Command{
	Use:   "templates",
	Short: "List available templates",
	Long:  `List all available configuration templates (nginx, PHP, Varnish, etc.).`,
	RunE:  runLibTemplates,
}

var libShowCmd = &cobra.Command{
	Use:   "show [platform]",
	Short: "Show installer config details",
	Long:  `Show the installer configuration for a platform with variable expansion.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runLibShow,
}

var libSetCmd = &cobra.Command{
	Use:   "set <path>",
	Short: "Set custom library path",
	Long: `Set a custom path for the MageBox configuration library.

This allows you to use your own templates and installer configurations
instead of the default ~/.magebox/yaml directory.

The path should contain:
  - templates/    (nginx, php, varnish, etc. templates)
  - installers/   (platform-specific YAML configs)

Example:
  mbox lib set ~/my-magebox-configs
  mbox lib set /path/to/custom/lib`,
	Args: cobra.ExactArgs(1),
	RunE: runLibSet,
}

var libUnsetCmd = &cobra.Command{
	Use:   "unset",
	Short: "Remove custom library path",
	Long:  `Remove the custom library path and revert to using the default ~/.magebox/yaml directory.`,
	RunE:  runLibUnset,
}

func init() {
	rootCmd.AddCommand(libCmd)
	libCmd.AddCommand(libUpdateCmd)
	libCmd.AddCommand(libStatusCmd)
	libCmd.AddCommand(libResetCmd)
	libCmd.AddCommand(libPathCmd)
	libCmd.AddCommand(libListCmd)
	libCmd.AddCommand(libTemplatesCmd)
	libCmd.AddCommand(libShowCmd)
	libCmd.AddCommand(libSetCmd)
	libCmd.AddCommand(libUnsetCmd)
}

func getLibManager() (*lib.Manager, error) {
	paths, err := lib.DefaultPaths()
	if err != nil {
		return nil, fmt.Errorf("failed to get paths: %w", err)
	}
	return lib.NewManager(paths), nil
}

func runLibUpdate(cmd *cobra.Command, args []string) error {
	manager, err := getLibManager()
	if err != nil {
		return err
	}

	fmt.Println("Updating MageBox library...")

	if err := manager.Update(); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	version := manager.GetVersion()
	color.Green("✓ Library updated to version %s", version)

	return nil
}

func runLibStatus(cmd *cobra.Command, args []string) error {
	manager, err := getLibManager()
	if err != nil {
		return err
	}

	status := manager.GetStatus()

	fmt.Println("MageBox Library Status")
	fmt.Println("──────────────────────")

	// Check if custom path is set
	homeDir, _ := os.UserHomeDir()
	globalCfg, _ := config.LoadGlobalConfig(homeDir)
	if globalCfg != nil && globalCfg.LibPath != "" {
		color.Cyan("Mode:     Custom path")
		fmt.Printf("Path:     %s\n", globalCfg.LibPath)
		fmt.Println()
	}

	if !status.Installed {
		color.Yellow("Library is not installed")
		fmt.Println("\nRun 'mbox bootstrap' or 'mbox lib update' to install.")
		return nil
	}

	// Version
	fmt.Printf("Version:  %s\n", status.Version)

	// Git info
	if status.IsGitRepo {
		fmt.Printf("Branch:   %s\n", status.Branch)
		fmt.Printf("Commit:   %s\n", status.CommitHash)

		// Local changes
		if status.HasLocalChanges {
			color.Yellow("Status:   Modified (has local changes)")
		} else {
			color.Green("Status:   Clean")
		}

		// Updates available
		if status.BehindRemote > 0 {
			color.Cyan("Updates:  %d commit(s) available", status.BehindRemote)
			fmt.Println("\nRun 'mbox lib update' to get the latest changes.")
		} else {
			fmt.Println("Updates:  Up to date")
		}
	} else {
		color.Yellow("Status:   Not a git repository (manual installation)")
	}

	// Path
	fmt.Printf("Path:     %s\n", manager.GetPath())

	return nil
}

func runLibReset(cmd *cobra.Command, args []string) error {
	manager, err := getLibManager()
	if err != nil {
		return err
	}

	fmt.Println("Resetting MageBox library to upstream...")
	fmt.Println("This will discard all local changes.")

	if err := manager.Reset(); err != nil {
		return fmt.Errorf("reset failed: %w", err)
	}

	version := manager.GetVersion()
	color.Green("✓ Library reset to version %s", version)

	return nil
}

func runLibPath(cmd *cobra.Command, args []string) error {
	paths, err := lib.DefaultPaths()
	if err != nil {
		return err
	}

	// Check if custom path is set
	homeDir, _ := os.UserHomeDir()
	globalCfg, _ := config.LoadGlobalConfig(homeDir)
	if globalCfg != nil && globalCfg.LibPath != "" {
		fmt.Println(paths.LibDir + " (custom)")
	} else {
		fmt.Println(paths.LibDir)
	}
	return nil
}

func runLibList(cmd *cobra.Command, args []string) error {
	manager, err := getLibManager()
	if err != nil {
		return err
	}

	installers, err := manager.ListInstallers()
	if err != nil {
		// Check if lib is not installed
		paths, _ := lib.DefaultPaths()
		if !paths.Exists() {
			color.Yellow("Library is not installed")
			fmt.Println("\nRun 'mbox bootstrap' or 'mbox lib update' to install.")
			return nil
		}
		return err
	}

	fmt.Println("Available Installers")
	fmt.Println("────────────────────")

	for _, name := range installers {
		fmt.Printf("  • %s\n", name)
	}

	// Check for local overrides
	paths, _ := lib.DefaultPaths()
	if entries, err := os.ReadDir(paths.LocalInstallersDir); err == nil && len(entries) > 0 {
		fmt.Println("\nLocal Overrides")
		fmt.Println("───────────────")
		for _, entry := range entries {
			if !entry.IsDir() {
				fmt.Printf("  • %s\n", entry.Name())
			}
		}
	}

	return nil
}

func runLibTemplates(cmd *cobra.Command, args []string) error {
	paths, err := lib.DefaultPaths()
	if err != nil {
		return err
	}

	if !paths.Exists() {
		color.Yellow("Library is not installed")
		fmt.Println("\nRun 'mbox bootstrap' or 'mbox lib update' to install.")
		return nil
	}

	fmt.Println("Available Templates")
	fmt.Println("───────────────────")

	// List templates by category
	for category, templates := range lib.TemplateNames {
		fmt.Printf("\n%s:\n", category)
		for _, tmpl := range templates {
			// Check if template exists
			exists := paths.TemplateExists(category, tmpl)
			if exists {
				fmt.Printf("  • %s\n", tmpl)
			} else {
				color.Red("  • %s (missing)\n", tmpl)
			}
		}

		// Check for local overrides in this category
		localCategoryDir := paths.LocalTemplatesDir + "/" + category
		if entries, err := os.ReadDir(localCategoryDir); err == nil && len(entries) > 0 {
			color.Cyan("  Local overrides:")
			for _, entry := range entries {
				if !entry.IsDir() {
					fmt.Printf("    • %s\n", entry.Name())
				}
			}
		}
	}

	return nil
}

func runLibShow(cmd *cobra.Command, args []string) error {
	loader, err := libconfig.DefaultLoader()
	if err != nil {
		return err
	}

	if !loader.IsLibInstalled() {
		color.Yellow("Library is not installed")
		fmt.Println("\nRun 'mbox bootstrap' or 'mbox lib update' to install.")
		return nil
	}

	// Determine platform
	var platform string
	if len(args) > 0 {
		platform = args[0]
	} else {
		platform = loader.DetectPlatform()
		if platform == "" {
			return fmt.Errorf("could not detect platform, please specify one")
		}
	}

	// Load config
	cfg, err := loader.LoadInstaller(platform)
	if err != nil {
		return fmt.Errorf("failed to load %s config: %w", platform, err)
	}

	// Setup variables
	loader.SetupOSVariables()

	// Display config info
	fmt.Printf("Installer Configuration: %s\n", platform)
	fmt.Println(strings.Repeat("─", 40))

	fmt.Printf("\nPlatform:     %s\n", cfg.Meta.Platform)
	fmt.Printf("Display Name: %s\n", cfg.Meta.DisplayName)
	fmt.Printf("Versions:     %s\n", strings.Join(cfg.Meta.GetSupportedVersions(), ", "))

	fmt.Printf("\nPackage Manager: %s\n", cfg.PackageManager.Name)
	fmt.Printf("  Install: %s\n", cfg.PackageManager.Install)

	fmt.Println("\nPHP Versions:", strings.Join(cfg.PHP.Versions, ", "))
	fmt.Printf("  Format: %s\n", cfg.PHP.VersionFormat)

	// Show expanded PHP packages for 8.3 as example
	if len(cfg.PHP.Versions) > 0 {
		exampleVersion := "8.3"
		packages := loader.GetPHPPackages(cfg, exampleVersion)
		fmt.Printf("\n  Packages (PHP %s):\n", exampleVersion)
		for _, pkg := range packages[:min(5, len(packages))] {
			fmt.Printf("    • %s\n", pkg)
		}
		if len(packages) > 5 {
			fmt.Printf("    ... and %d more\n", len(packages)-5)
		}

		binary := loader.GetPHPBinary(cfg, exampleVersion)
		fmt.Printf("  Binary: %s\n", binary)

		service := loader.GetPHPFPMService(cfg, exampleVersion)
		fmt.Printf("  FPM Service: %s\n", service)
	}

	fmt.Printf("\nNginx Packages: %s\n", strings.Join(cfg.Nginx.Packages, ", "))

	if cfg.SELinux.Enabled {
		color.Cyan("\nSELinux: enabled")
		fmt.Printf("  Booleans: %d\n", len(cfg.SELinux.Booleans))
		fmt.Printf("  Contexts: %d\n", len(cfg.SELinux.Contexts))
	} else {
		fmt.Println("\nSELinux: disabled")
	}

	if cfg.Sudoers.Enabled {
		color.Cyan("\nSudoers: enabled")
		fmt.Printf("  File: %s\n", cfg.Sudoers.File)
		fmt.Printf("  Rules: %d\n", len(cfg.Sudoers.Rules))
	}

	return nil
}

func runLibSet(cmd *cobra.Command, args []string) error {
	customPath := args[0]

	// Expand ~ to home directory
	if strings.HasPrefix(customPath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		customPath = filepath.Join(homeDir, customPath[2:])
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(customPath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Check if the path exists
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("path does not exist: %s", absPath)
		}
		return fmt.Errorf("failed to access path: %w", err)
	}

	// Check if it's a directory
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", absPath)
	}

	// Load and update global config
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	globalCfg, err := config.LoadGlobalConfig(homeDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	globalCfg.LibPath = absPath

	if err := config.SaveGlobalConfig(homeDir, globalCfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	color.Green("✓ Library path set to: %s", absPath)
	fmt.Println()

	// Show what's available in the custom path
	templatesDir := filepath.Join(absPath, "templates")
	installersDir := filepath.Join(absPath, "installers")

	if _, err := os.Stat(templatesDir); err == nil {
		fmt.Println("  Templates directory: " + color.GreenString("found"))
	} else {
		fmt.Println("  Templates directory: " + color.YellowString("not found"))
	}

	if _, err := os.Stat(installersDir); err == nil {
		fmt.Println("  Installers directory: " + color.GreenString("found"))
	} else {
		fmt.Println("  Installers directory: " + color.YellowString("not found"))
	}

	fmt.Println()
	fmt.Println("MageBox will now use templates and installers from this path.")
	fmt.Println("Run 'mbox lib unset' to revert to the default path.")

	return nil
}

func runLibUnset(cmd *cobra.Command, args []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	globalCfg, err := config.LoadGlobalConfig(homeDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if globalCfg.LibPath == "" {
		fmt.Println("No custom library path is set.")
		return nil
	}

	oldPath := globalCfg.LibPath
	globalCfg.LibPath = ""

	if err := config.SaveGlobalConfig(homeDir, globalCfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	color.Green("✓ Custom library path removed")
	fmt.Printf("  Previous: %s\n", oldPath)
	fmt.Println()

	// Show default path
	paths, _ := lib.DefaultPaths()
	fmt.Printf("MageBox will now use the default path: %s\n", paths.LibDir)

	return nil
}

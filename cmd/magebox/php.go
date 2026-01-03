package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"qoliber/magebox/internal/cli"
	"qoliber/magebox/internal/config"
	"qoliber/magebox/internal/php"
	"qoliber/magebox/internal/project"
)

var phpCmd = &cobra.Command{
	Use:   "php [version]",
	Short: "Switch PHP version",
	Long:  "Switches the PHP version for the current project (updates .magebox.local)",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runPhp,
}

func init() {
	rootCmd.AddCommand(phpCmd)
}

func runPhp(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	p, err := getPlatform()
	if err != nil {
		return err
	}

	// Load current config
	cfg, ok := loadProjectConfig(cwd)
	if !ok {
		return nil
	}

	if len(args) == 0 {
		// Show current PHP version
		fmt.Printf("Current PHP version: %s\n", cli.Highlight(cfg.PHP))

		// Show installed versions
		detector := php.NewDetector(p)
		installed := detector.DetectInstalled()
		if len(installed) > 0 {
			fmt.Println(cli.Header("Installed Versions"))
			for _, v := range installed {
				marker := "  "
				if v.Version == cfg.PHP {
					marker = cli.Success("")
				}
				fmt.Printf("%s %s\n", marker, v.Version)
			}
		}
		return nil
	}

	// Switch PHP version
	newVersion := args[0]

	// Check if version is installed
	detector := php.NewDetector(p)
	if !detector.IsVersionInstalled(newVersion) {
		cli.PrintError("PHP %s is not installed", newVersion)
		fmt.Println()
		fmt.Print(php.FormatNotInstalledMessage(newVersion, p))
		return nil
	}

	// Write to .magebox.local.yaml
	localConfigPath := filepath.Join(cwd, config.LocalConfigFileName)
	content := fmt.Sprintf("php: \"%s\"\n", newVersion)

	if err := os.WriteFile(localConfigPath, []byte(content), 0644); err != nil {
		cli.PrintError("Failed to write %s: %v", config.LocalConfigFileName, err)
		return nil
	}

	cli.PrintSuccess("Switched to PHP %s", newVersion)
	fmt.Println()

	// Reload config with new PHP version
	_, ok = loadProjectConfig(cwd)
	if !ok {
		return nil
	}

	// Restart the project to apply the PHP version change
	fmt.Println("Applying PHP version change...")
	fmt.Println()

	mgr := project.NewManager(p)

	// Stop current services
	fmt.Print("Stopping services... ")
	if err := mgr.Stop(cwd); err != nil {
		fmt.Println(cli.Error("failed"))
		cli.PrintWarning("Failed to stop: %v", err)
	} else {
		fmt.Println(cli.Success("done"))
	}

	// Start with new PHP version
	fmt.Printf("Starting with PHP %s... ", newVersion)
	result, err := mgr.Start(cwd)
	if err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to start: %w", err)
	}
	fmt.Println(cli.Success("done"))
	fmt.Println()

	cli.PrintSuccess("Project is now running with PHP %s", newVersion)
	if len(result.Domains) > 0 {
		fmt.Printf("  Domain: %s\n", cli.URL("https://"+result.Domains[0]))
	}

	return nil
}

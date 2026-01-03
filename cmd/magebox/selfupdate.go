package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"qoliber/magebox/internal/cli"
	"qoliber/magebox/internal/updater"
)

var selfUpdateCmd = &cobra.Command{
	Use:   "self-update",
	Short: "Update MageBox to latest version",
	Long: `Checks for and installs the latest MageBox version from GitHub.

Downloads the appropriate binary for your platform and replaces the current one.`,
	RunE: runSelfUpdate,
}

var selfUpdateCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check for updates",
	Long:  "Checks if a newer version of MageBox is available",
	RunE:  runSelfUpdateCheck,
}

func init() {
	selfUpdateCmd.AddCommand(selfUpdateCheckCmd)
	rootCmd.AddCommand(selfUpdateCmd)
}

func runSelfUpdate(cmd *cobra.Command, args []string) error {
	cli.PrintTitle("MageBox Self-Update")
	fmt.Println()

	u := updater.NewUpdater(version)

	fmt.Printf("Current version: %s\n", cli.Highlight(version))
	fmt.Printf("Platform: %s\n", updater.GetPlatformInfo())
	fmt.Println()

	cli.PrintInfo("Checking for updates...")

	result, err := u.CheckForUpdate()
	if err != nil {
		cli.PrintError("Failed to check for updates: %v", err)
		fmt.Println()
		cli.PrintInfo("Check your internet connection or try again later")
		return nil
	}

	if !result.UpdateAvailable {
		cli.PrintSuccess("You're already running the latest version!")
		return nil
	}

	fmt.Println()
	fmt.Printf("New version available: %s\n", cli.Highlight(result.LatestVersion))

	if result.DownloadURL == "" {
		cli.PrintError("No binary available for your platform (%s)", updater.GetPlatformInfo())
		fmt.Println()
		cli.PrintInfo("You can build from source: %s", cli.Command("go install qoliber/magebox@latest"))
		return nil
	}

	// Show release notes if available
	if result.ReleaseNotes != "" {
		fmt.Println(cli.Header("Release Notes"))
		// Truncate long release notes
		notes := result.ReleaseNotes
		if len(notes) > 500 {
			notes = notes[:500] + "..."
		}
		fmt.Println(notes)
		fmt.Println()
	}

	cli.PrintInfo("Downloading update...")

	if err := u.Update(result); err != nil {
		cli.PrintError("Failed to install update: %v", err)
		fmt.Println()
		cli.PrintInfo("You may need to run with sudo or check file permissions")
		return nil
	}

	cli.PrintSuccess("Updated to version %s!", result.LatestVersion)
	fmt.Println()
	cli.PrintInfo("Run %s to verify", cli.Command("magebox --version"))

	return nil
}

// runSelfUpdateCheck checks for updates without installing
func runSelfUpdateCheck(cmd *cobra.Command, args []string) error {
	cli.PrintTitle("Check for Updates")
	fmt.Println()

	fmt.Printf("Current version: %s\n", cli.Highlight(version))
	fmt.Printf("Platform: %s\n", updater.GetPlatformInfo())
	fmt.Println()

	u := updater.NewUpdater(version)

	cli.PrintInfo("Checking GitHub releases...")

	result, err := u.CheckForUpdate()
	if err != nil {
		cli.PrintError("Failed to check for updates: %v", err)
		return nil
	}

	fmt.Println()
	if result.UpdateAvailable {
		cli.PrintSuccess("Update available!")
		fmt.Printf("  Latest version: %s\n", cli.Highlight(result.LatestVersion))
		fmt.Println()
		cli.PrintInfo("Run %s to install", cli.Command("magebox self-update"))
	} else {
		cli.PrintSuccess("You're running the latest version!")
	}

	return nil
}

// Copyright (c) qoliber

package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"qoliber/magebox/internal/cli"
	"qoliber/magebox/internal/php"
)

var extCmd = &cobra.Command{
	Use:   "ext",
	Short: "Manage PHP extensions",
	Long: `Install, remove, list, or search PHP extensions for the current project's PHP version.

For system packages (apt/dnf/pacman/pecl):
  magebox ext install redis apcu
  magebox ext remove redis

For custom extensions via PIE (vendor/package format):
  magebox ext install noisebynorthwest/php-spx
  magebox ext remove noisebynorthwest/php-spx

Other commands:
  magebox ext list
  magebox ext search redis
  magebox ext pie          Install the PIE tool itself`,
	RunE: runExtList,
}

var extInstallCmd = &cobra.Command{
	Use:   "install <extension> [extension...]",
	Short: "Install PHP extensions",
	Long:  `Installs one or more PHP extensions for the current project's PHP version and restarts PHP-FPM.`,
	Args:  cobra.MinimumNArgs(1),
	RunE:  runExtInstall,
}

var extRemoveCmd = &cobra.Command{
	Use:   "remove <extension> [extension...]",
	Short: "Remove PHP extensions",
	Long:  `Removes one or more PHP extensions for the current project's PHP version and restarts PHP-FPM.`,
	Args:  cobra.MinimumNArgs(1),
	RunE:  runExtRemove,
}

var extListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed PHP extensions",
	Long:  `Lists all loaded PHP extensions for the current project's PHP version.`,
	RunE:  runExtList,
}

var extSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search available PHP extensions",
	Long:  `Searches for available PHP extension packages matching the query.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runExtSearch,
}

var extPieCmd = &cobra.Command{
	Use:   "pie",
	Short: "Install PIE (PHP Installer for Extensions)",
	Long: `Downloads and installs PIE, the official PHP extension installer.

PIE is needed to install custom extensions from Packagist using the
vendor/package format (e.g., magebox ext install noisebynorthwest/php-spx).

Browse available extensions at https://packagist.org/extensions`,
	RunE: runExtPie,
}

func init() {
	extCmd.AddCommand(extInstallCmd)
	extCmd.AddCommand(extRemoveCmd)
	extCmd.AddCommand(extListCmd)
	extCmd.AddCommand(extSearchCmd)
	extCmd.AddCommand(extPieCmd)
	rootCmd.AddCommand(extCmd)
}

func runExtInstall(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	// Detect installed PHP versions
	detector := php.NewDetector(p)
	installed := detector.DetectInstalled()
	if len(installed) == 0 {
		return fmt.Errorf("no PHP versions installed")
	}

	// Prompt for PHP version selection
	versions := promptPHPVersionSelection(installed)

	cli.PrintTitle("Installing PHP Extensions")
	fmt.Println()

	mgr := php.NewExtensionManager(p)

	// Check if any args are PIE packages (vendor/package format)
	hasPIE := false
	for _, ext := range args {
		if php.IsPIEPackage(ext) {
			hasPIE = true
			break
		}
	}

	// Ensure PIE is available if needed
	if hasPIE {
		if err := ensurePIE(mgr); err != nil {
			return err
		}
	}

	for _, phpVersion := range versions {
		fmt.Printf("PHP %s:\n", cli.Highlight(phpVersion))

		// Ensure php-dev package is installed for PIE (phpize/php-config needed)
		if hasPIE {
			fmt.Printf("  Ensuring PHP %s development tools... ", phpVersion)
			if err := mgr.EnsurePHPDev(phpVersion); err != nil {
				fmt.Println(cli.Error("failed"))
				cli.PrintError("  %v", err)
				cli.PrintInfo("  PIE requires php-dev (phpize, php-config) to compile extensions")
				fmt.Println()
				continue
			}
			fmt.Println(cli.Success("done"))
		}

		for _, ext := range args {
			if php.IsPIEPackage(ext) {
				pieCmd := mgr.PIEInstallCommand(ext, phpVersion)
				fmt.Printf("  Installing %s (%s)...\n", cli.Highlight(ext), pieCmd)

				if err := mgr.InstallViaPIE(ext, phpVersion); err != nil {
					cli.PrintError("  %v", err)
					continue
				}
				fmt.Printf("  %s\n", cli.Success("done"))
			} else {
				installCmd := mgr.InstallCommand(ext, phpVersion)
				fmt.Printf("  Installing %s (%s)... ", cli.Highlight(ext), installCmd)

				errs := mgr.Install([]string{ext}, phpVersion)
				if len(errs) > 0 {
					fmt.Println(cli.Error("failed"))
					cli.PrintError("  %v", errs[0])
					continue
				}
				fmt.Println(cli.Success("done"))
			}
		}

		// Restart PHP-FPM for this version
		fmt.Printf("  Restarting PHP-FPM %s... ", phpVersion)
		fpmCtrl := php.NewFPMController(p, phpVersion)
		if err := fpmCtrl.Reload(); err != nil {
			fmt.Println(cli.Warning("failed (may need manual restart)"))
		} else {
			fmt.Println(cli.Success("done"))
		}

		fmt.Println()
	}

	cli.PrintSuccess("Done!")

	return nil
}

func runExtRemove(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	// Detect installed PHP versions
	detector := php.NewDetector(p)
	installed := detector.DetectInstalled()
	if len(installed) == 0 {
		return fmt.Errorf("no PHP versions installed")
	}

	// Prompt for PHP version selection
	versions := promptPHPVersionSelection(installed)

	cli.PrintTitle("Removing PHP Extensions")
	fmt.Println()

	mgr := php.NewExtensionManager(p)

	// Check if any args are PIE packages
	hasPIE := false
	for _, ext := range args {
		if php.IsPIEPackage(ext) {
			hasPIE = true
			break
		}
	}

	if hasPIE && !mgr.IsPIEInstalled() {
		return fmt.Errorf("PIE is not installed - install with: magebox ext pie")
	}

	for _, phpVersion := range versions {
		fmt.Printf("PHP %s:\n", cli.Highlight(phpVersion))

		for _, ext := range args {
			if php.IsPIEPackage(ext) {
				fmt.Printf("  Removing %s...\n", cli.Highlight(ext))

				if err := mgr.RemoveViaPIE(ext, phpVersion); err != nil {
					cli.PrintError("  %v", err)
					continue
				}
				fmt.Printf("  %s\n", cli.Success("done"))
			} else {
				fmt.Printf("  Removing %s... ", cli.Highlight(ext))

				errs := mgr.Remove([]string{ext}, phpVersion)
				if len(errs) > 0 {
					fmt.Println(cli.Error("failed"))
					cli.PrintError("  %v", errs[0])
					continue
				}
				fmt.Println(cli.Success("done"))
			}
		}

		// Restart PHP-FPM for this version
		fmt.Printf("  Restarting PHP-FPM %s... ", phpVersion)
		fpmCtrl := php.NewFPMController(p, phpVersion)
		if err := fpmCtrl.Reload(); err != nil {
			fmt.Println(cli.Warning("failed (may need manual restart)"))
		} else {
			fmt.Println(cli.Success("done"))
		}

		fmt.Println()
	}

	cli.PrintSuccess("Done!")

	return nil
}

func runExtPie(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	mgr := php.NewExtensionManager(p)

	if mgr.IsPIEInstalled() {
		cli.PrintSuccess("PIE is already installed")
		cli.PrintInfo("Install extensions with: magebox ext install vendor/package")
		cli.PrintInfo("Browse extensions at: https://packagist.org/extensions")
		return nil
	}

	cli.PrintTitle("Installing PIE")
	fmt.Println("PIE (PHP Installer for Extensions) is the official PECL replacement.")
	fmt.Println("It installs PHP extensions from Packagist using vendor/package format.")
	fmt.Println()

	fmt.Print("Downloading and installing PIE... ")
	if err := mgr.InstallPIE(); err != nil {
		fmt.Println(cli.Error("failed"))
		return err
	}
	fmt.Println(cli.Success("done"))

	fmt.Println()
	cli.PrintSuccess("PIE installed!")
	fmt.Println()
	cli.PrintInfo("Install extensions with: magebox ext install vendor/package")
	cli.PrintInfo("Example: magebox ext install noisebynorthwest/php-spx")
	cli.PrintInfo("Browse extensions at: https://packagist.org/extensions")

	return nil
}

// ensurePIE checks if PIE is installed and offers to install it if not.
func ensurePIE(mgr *php.ExtensionManager) error {
	if mgr.IsPIEInstalled() {
		return nil
	}

	fmt.Println("PIE (PHP Installer for Extensions) is required for vendor/package extensions.")
	fmt.Print("Install PIE now? [Y/n]: ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input != "" && input != "y" && input != "yes" {
		return fmt.Errorf("PIE is required - install with: magebox ext pie")
	}

	fmt.Print("Downloading and installing PIE... ")
	if err := mgr.InstallPIE(); err != nil {
		fmt.Println(cli.Error("failed"))
		return err
	}
	fmt.Println(cli.Success("done"))
	fmt.Println()

	return nil
}

func runExtList(cmd *cobra.Command, args []string) error {
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

	cli.PrintTitle("PHP Extensions")
	fmt.Printf("PHP version: %s\n", cli.Highlight(phpVersion))
	fmt.Println()

	mgr := php.NewExtensionManager(p)
	extensions, err := mgr.List(phpVersion)
	if err != nil {
		return err
	}

	sort.Strings(extensions)

	for _, ext := range extensions {
		fmt.Printf("  %s\n", ext)
	}

	fmt.Println()
	fmt.Printf("%s extensions loaded\n", cli.Highlight(fmt.Sprintf("%d", len(extensions))))
	fmt.Println()
	cli.PrintInfo("Install more with: magebox ext install <name>")

	return nil
}

func runExtSearch(cmd *cobra.Command, args []string) error {
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

	query := strings.ToLower(args[0])

	cli.PrintTitle("Search PHP Extensions")
	fmt.Printf("PHP version: %s\n", cli.Highlight(phpVersion))
	fmt.Printf("Query: %s\n", cli.Highlight(query))
	fmt.Println()

	mgr := php.NewExtensionManager(p)
	results, err := mgr.Search(query, phpVersion)
	if err != nil {
		return err
	}

	if len(results) == 0 {
		cli.PrintWarning("No extensions found matching '%s'", query)
		return nil
	}

	// Get installed extensions to show status
	installed := make(map[string]bool)
	if loadedExts, err := mgr.List(phpVersion); err == nil {
		for _, ext := range loadedExts {
			installed[strings.ToLower(ext)] = true
		}
	}

	for _, line := range results {
		// Check if any installed extension name appears in the search result line
		status := cli.Warning("not installed")
		lineLower := strings.ToLower(line)
		for ext := range installed {
			if strings.Contains(lineLower, ext) {
				status = cli.Success("installed")
				break
			}
		}
		fmt.Printf("  %s [%s]\n", line, status)
	}

	fmt.Println()
	fmt.Printf("%s results\n", cli.Highlight(fmt.Sprintf("%d", len(results))))

	return nil
}

// promptPHPVersionSelection shows installed PHP versions and lets the user
// pick one or all. Returns the selected version string(s).
func promptPHPVersionSelection(installed []php.Version) []string {
	// Collect version strings
	versions := make([]string, 0, len(installed))
	for _, v := range installed {
		versions = append(versions, v.Version)
	}

	// Single version installed — no need to ask
	if len(versions) == 1 {
		return versions
	}

	// Show menu
	fmt.Println("Installed PHP versions:")
	fmt.Printf("  %s) %s (default)\n", cli.Highlight("0"), "All installed")
	for i, v := range versions {
		fmt.Printf("  %s) PHP %s\n", cli.Highlight(fmt.Sprintf("%d", i+1)), v)
	}
	fmt.Print("\nSelect PHP version [0]: ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	// Default: all
	if input == "" || input == "0" {
		return versions
	}

	// Parse selection
	idx, err := strconv.Atoi(input)
	if err != nil || idx < 1 || idx > len(versions) {
		fmt.Printf("Invalid selection, using all installed versions\n")
		return versions
	}

	return []string{versions[idx-1]}
}

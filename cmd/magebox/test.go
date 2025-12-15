// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/qoliber/magebox/internal/cli"
	"github.com/qoliber/magebox/internal/config"
	"github.com/qoliber/magebox/internal/testing"
)

// Test command flags
var (
	testFilter    string
	testSuite     string
	phpstanLevel  int
	phpcsStandard string
	phpmdRuleset  string
	// Integration test flags
	integrationTmpfs     bool
	integrationTmpfsSize string
	integrationMySQLVer  string
	integrationKeepAlive bool
)

// testCmd is the root test command
var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Run tests and code quality tools",
	Long: `Run PHPUnit tests and static analysis tools for your Magento project.

Available subcommands:
  setup        Install and configure testing tools
  unit         Run PHPUnit unit tests
  integration  Run Magento integration tests
  phpstan      Run PHPStan static analysis
  phpcs        Run PHP_CodeSniffer code style checks
  phpmd        Run PHP Mess Detector
  all          Run all tests except integration
  status       Show testing tools status`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

// testSetupCmd handles test setup
var testSetupCmd = &cobra.Command{
	Use:   "setup [unit|static|all]",
	Short: "Install and configure testing tools",
	Long: `Install testing tools via composer.

Available options:
  unit    - Install PHPUnit for unit testing
  static  - Install static analysis tools (PHPStan, PHPCS, PHPMD)
  all     - Install all testing tools (default)

Examples:
  magebox test setup           # Interactive setup wizard
  magebox test setup all       # Install all tools
  magebox test setup static    # Install static analysis tools only`,
	Run: runTestSetup,
}

// testUnitCmd runs PHPUnit unit tests
var testUnitCmd = &cobra.Command{
	Use:   "unit [filter]",
	Short: "Run PHPUnit unit tests",
	Long: `Run PHPUnit unit tests for your Magento project.

Examples:
  magebox test unit                    # Run all unit tests
  magebox test unit --filter=MyTest    # Run tests matching filter
  magebox test unit --testsuite=Unit   # Run specific test suite`,
	Run: runTestUnit,
}

// testIntegrationCmd runs Magento integration tests
var testIntegrationCmd = &cobra.Command{
	Use:   "integration [filter]",
	Short: "Run Magento integration tests",
	Long: `Run Magento integration tests.

Note: Integration tests require a separate database and may take a long time.

Tmpfs Mode:
  Use --tmpfs to run MySQL in RAM for much faster tests. This creates a
  dedicated test container named mysql-{version}-test (e.g., mysql-8-0-test).

Examples:
  magebox test integration                           # Run all integration tests
  magebox test integration --filter=Cart             # Run tests matching filter
  magebox test integration --tmpfs                   # Run with MySQL in RAM (fast!)
  magebox test integration --tmpfs --tmpfs-size=2g   # Use 2GB RAM for MySQL
  magebox test integration --tmpfs --keep-alive      # Keep container after tests
  magebox test integration --mysql-version=8.4       # Use specific MySQL version`,
	Run: runTestIntegration,
}

// testPHPStanCmd runs PHPStan analysis
var testPHPStanCmd = &cobra.Command{
	Use:   "phpstan [path...]",
	Short: "Run PHPStan static analysis",
	Long: `Run PHPStan static analysis on your code.

Examples:
  magebox test phpstan                    # Analyze default paths (app/code)
  magebox test phpstan --level=5          # Analyze at level 5
  magebox test phpstan app/code/MyModule  # Analyze specific path`,
	Run: runTestPHPStan,
}

// testPHPCSCmd runs PHP_CodeSniffer
var testPHPCSCmd = &cobra.Command{
	Use:   "phpcs [path...]",
	Short: "Run PHP_CodeSniffer code style checks",
	Long: `Run PHP_CodeSniffer to check coding standards.

Examples:
  magebox test phpcs                      # Check default paths with Magento2 standard
  magebox test phpcs --standard=PSR12     # Use PSR-12 standard
  magebox test phpcs app/code/MyModule    # Check specific path`,
	Run: runTestPHPCS,
}

// testPHPMDCmd runs PHP Mess Detector
var testPHPMDCmd = &cobra.Command{
	Use:   "phpmd [path...]",
	Short: "Run PHP Mess Detector",
	Long: `Run PHP Mess Detector to find potential problems.

Examples:
  magebox test phpmd                              # Analyze default paths
  magebox test phpmd --ruleset=cleancode,design   # Use specific rulesets
  magebox test phpmd app/code/MyModule            # Analyze specific path`,
	Run: runTestPHPMD,
}

// testAllCmd runs all tests except integration
var testAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Run all tests except integration",
	Long: `Run all tests and static analysis tools except integration tests.

This runs: unit tests, PHPStan, PHPCS, and PHPMD.
Integration tests are excluded because they require database setup.`,
	Run: runTestAll,
}

// testStatusCmd shows testing tools status
var testStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show testing tools status",
	Long:  `Show the installation and configuration status of all testing tools.`,
	Run:   runTestStatus,
}

func init() {
	// Add flags
	testUnitCmd.Flags().StringVarP(&testFilter, "filter", "f", "", "Filter tests by name")
	testUnitCmd.Flags().StringVarP(&testSuite, "testsuite", "t", "", "Test suite to run")

	testIntegrationCmd.Flags().StringVarP(&testFilter, "filter", "f", "", "Filter tests by name")
	testIntegrationCmd.Flags().StringVarP(&testSuite, "testsuite", "t", "", "Test suite to run")
	testIntegrationCmd.Flags().BoolVar(&integrationTmpfs, "tmpfs", false, "Run MySQL in RAM for faster tests")
	testIntegrationCmd.Flags().StringVar(&integrationTmpfsSize, "tmpfs-size", "1g", "RAM size for tmpfs MySQL (e.g., 1g, 2g)")
	testIntegrationCmd.Flags().StringVar(&integrationMySQLVer, "mysql-version", "8.0", "MySQL version for test container")
	testIntegrationCmd.Flags().BoolVar(&integrationKeepAlive, "keep-alive", false, "Keep test container running after tests")

	testPHPStanCmd.Flags().IntVarP(&phpstanLevel, "level", "l", -1, "PHPStan analysis level (0-9)")

	testPHPCSCmd.Flags().StringVarP(&phpcsStandard, "standard", "s", "", "Coding standard (Magento2, PSR12)")

	testPHPMDCmd.Flags().StringVarP(&phpmdRuleset, "ruleset", "r", "", "PHPMD ruleset (cleancode,codesize,design)")

	// Add subcommands
	testCmd.AddCommand(testSetupCmd)
	testCmd.AddCommand(testUnitCmd)
	testCmd.AddCommand(testIntegrationCmd)
	testCmd.AddCommand(testPHPStanCmd)
	testCmd.AddCommand(testPHPCSCmd)
	testCmd.AddCommand(testPHPMDCmd)
	testCmd.AddCommand(testAllCmd)
	testCmd.AddCommand(testStatusCmd)

	// Add to root
	rootCmd.AddCommand(testCmd)
}

// getTestManager creates a testing manager for the current project
func getTestManager() (*testing.Manager, error) {
	cwd, err := getCwd()
	if err != nil {
		return nil, err
	}

	cfg, ok := loadProjectConfig(cwd)
	if !ok {
		return nil, fmt.Errorf("no project configuration found")
	}

	p, err := getPlatform()
	if err != nil {
		return nil, err
	}

	return testing.NewManager(p, cfg.PHP, cwd), nil
}

func runTestSetup(cmd *cobra.Command, args []string) {
	mgr, err := getTestManager()
	if err != nil {
		cli.PrintError("%v", err)
		return
	}

	installer := testing.NewInstaller(mgr)

	// Determine what to install
	if len(args) > 0 {
		switch args[0] {
		case "unit":
			cli.PrintTitle("Installing PHPUnit")
			if err := installer.InstallPHPUnit(); err != nil {
				cli.PrintError("%v", err)
				return
			}
			cli.PrintSuccess("PHPUnit installed successfully")
			return

		case "static":
			cli.PrintTitle("Installing Static Analysis Tools")
			if err := installer.InstallPHPStan(); err != nil {
				cli.PrintError("%v", err)
				return
			}
			if err := installer.InstallPHPCS(); err != nil {
				cli.PrintError("%v", err)
				return
			}
			if err := installer.InstallPHPMD(); err != nil {
				cli.PrintError("%v", err)
				return
			}
			cli.PrintSuccess("Static analysis tools installed successfully")
			return

		case "all":
			cli.PrintTitle("Installing All Testing Tools")
			if err := installer.InstallAll(); err != nil {
				cli.PrintError("%v", err)
				return
			}
			cli.PrintSuccess("All testing tools installed successfully")
			return
		}
	}

	// Interactive setup
	cli.PrintTitle("Testing Tools Setup Wizard")
	fmt.Println()
	fmt.Println("Select the tools you want to install:")
	fmt.Println()

	opts := testing.SetupOptions{}

	// PHPUnit
	fmt.Print("  [1] PHPUnit (unit testing) [Y/n]: ")
	opts.InstallPHPUnit = askYesNo(true)

	// PHPStan
	fmt.Print("  [2] PHPStan (static analysis) [Y/n]: ")
	opts.InstallPHPStan = askYesNo(true)

	// PHPCS
	fmt.Print("  [3] PHP_CodeSniffer (code style) [Y/n]: ")
	opts.InstallPHPCS = askYesNo(true)

	// PHPMD
	fmt.Print("  [4] PHP Mess Detector [Y/n]: ")
	opts.InstallPHPMD = askYesNo(true)

	fmt.Println()

	if !opts.InstallPHPUnit && !opts.InstallPHPStan && !opts.InstallPHPCS && !opts.InstallPHPMD {
		cli.PrintWarning("No tools selected for installation")
		return
	}

	cli.PrintTitle("Installing Selected Tools")
	if err := installer.InstallSelected(opts); err != nil {
		cli.PrintError("%v", err)
		return
	}

	fmt.Println()
	cli.PrintSuccess("Testing tools installed successfully!")
	fmt.Println()
	fmt.Println("You can now run:")
	if opts.InstallPHPUnit {
		fmt.Printf("  %s - Run unit tests\n", cli.Command("magebox test unit"))
	}
	if opts.InstallPHPStan {
		fmt.Printf("  %s - Run PHPStan analysis\n", cli.Command("magebox test phpstan"))
	}
	if opts.InstallPHPCS {
		fmt.Printf("  %s - Run code style checks\n", cli.Command("magebox test phpcs"))
	}
	if opts.InstallPHPMD {
		fmt.Printf("  %s - Run mess detection\n", cli.Command("magebox test phpmd"))
	}
	fmt.Printf("  %s - Run all (except integration)\n", cli.Command("magebox test all"))
}

func runTestUnit(cmd *cobra.Command, args []string) {
	mgr, err := getTestManager()
	if err != nil {
		cli.PrintError("%v", err)
		os.Exit(1)
	}

	cwd, _ := getCwd()
	cfg, _ := loadProjectConfig(cwd)

	var phpunitCfg *config.PHPUnitTestConfig
	if cfg != nil && cfg.Testing != nil {
		phpunitCfg = cfg.Testing.PHPUnit
	}

	runner := testing.NewPHPUnitRunner(mgr, phpunitCfg)

	cli.PrintTitle("Running PHPUnit Unit Tests")
	fmt.Println()

	if err := runner.RunUnit(testFilter, testSuite); err != nil {
		os.Exit(1)
	}
}

func runTestIntegration(cmd *cobra.Command, args []string) {
	mgr, err := getTestManager()
	if err != nil {
		cli.PrintError("%v", err)
		os.Exit(1)
	}

	cwd, _ := getCwd()
	cfg, _ := loadProjectConfig(cwd)

	var phpunitCfg *config.PHPUnitTestConfig
	var integrationCfg *config.IntegrationTestConfig
	if cfg != nil && cfg.Testing != nil {
		phpunitCfg = cfg.Testing.PHPUnit
		integrationCfg = cfg.Testing.Integration
	}

	runner := testing.NewPHPUnitRunner(mgr, phpunitCfg)
	runner.SetIntegrationConfig(integrationCfg)

	cli.PrintTitle("Running Magento Integration Tests")

	// Check if tmpfs is enabled via flag or config
	useTmpfs := integrationTmpfs
	if !useTmpfs && integrationCfg != nil {
		useTmpfs = integrationCfg.Tmpfs
	}

	if useTmpfs {
		cli.PrintInfo("Using tmpfs MySQL container for faster tests")
	} else {
		cli.PrintWarning("Integration tests may take a long time and require database setup")
		cli.PrintInfo("Tip: Use --tmpfs to run MySQL in RAM for much faster tests")
	}
	fmt.Println()

	opts := testing.IntegrationOptions{
		Filter:    testFilter,
		TestSuite: testSuite,
		UseTmpfs:  useTmpfs,
		TmpfsSize: integrationTmpfsSize,
		MySQLVer:  integrationMySQLVer,
		KeepAlive: integrationKeepAlive,
	}

	if err := runner.RunIntegrationWithOptions(opts); err != nil {
		os.Exit(1)
	}
}

func runTestPHPStan(cmd *cobra.Command, args []string) {
	mgr, err := getTestManager()
	if err != nil {
		cli.PrintError("%v", err)
		os.Exit(1)
	}

	cwd, _ := getCwd()
	cfg, _ := loadProjectConfig(cwd)

	var phpstanCfg *config.PHPStanTestConfig
	if cfg != nil && cfg.Testing != nil {
		phpstanCfg = cfg.Testing.PHPStan
	}

	runner := testing.NewPHPStanRunner(mgr, phpstanCfg)

	cli.PrintTitle("Running PHPStan Static Analysis")
	fmt.Println()

	if err := runner.Run(args, phpstanLevel); err != nil {
		os.Exit(1)
	}
}

func runTestPHPCS(cmd *cobra.Command, args []string) {
	mgr, err := getTestManager()
	if err != nil {
		cli.PrintError("%v", err)
		os.Exit(1)
	}

	cwd, _ := getCwd()
	cfg, _ := loadProjectConfig(cwd)

	var phpcsCfg *config.PHPCSTestConfig
	if cfg != nil && cfg.Testing != nil {
		phpcsCfg = cfg.Testing.PHPCS
	}

	runner := testing.NewPHPCSRunner(mgr, phpcsCfg)

	cli.PrintTitle("Running PHP_CodeSniffer")
	fmt.Println()

	if err := runner.Run(args, phpcsStandard); err != nil {
		os.Exit(1)
	}
}

func runTestPHPMD(cmd *cobra.Command, args []string) {
	mgr, err := getTestManager()
	if err != nil {
		cli.PrintError("%v", err)
		os.Exit(1)
	}

	cwd, _ := getCwd()
	cfg, _ := loadProjectConfig(cwd)

	var phpmdCfg *config.PHPMDTestConfig
	if cfg != nil && cfg.Testing != nil {
		phpmdCfg = cfg.Testing.PHPMD
	}

	runner := testing.NewPHPMDRunner(mgr, phpmdCfg)

	cli.PrintTitle("Running PHP Mess Detector")
	fmt.Println()

	if err := runner.Run(args, phpmdRuleset); err != nil {
		os.Exit(1)
	}
}

func runTestAll(cmd *cobra.Command, args []string) {
	mgr, err := getTestManager()
	if err != nil {
		cli.PrintError("%v", err)
		os.Exit(1)
	}

	cwd, _ := getCwd()
	cfg, _ := loadProjectConfig(cwd)

	var (
		phpunitCfg *config.PHPUnitTestConfig
		phpstanCfg *config.PHPStanTestConfig
		phpcsCfg   *config.PHPCSTestConfig
		phpmdCfg   *config.PHPMDTestConfig
	)

	if cfg != nil && cfg.Testing != nil {
		phpunitCfg = cfg.Testing.PHPUnit
		phpstanCfg = cfg.Testing.PHPStan
		phpcsCfg = cfg.Testing.PHPCS
		phpmdCfg = cfg.Testing.PHPMD
	}

	hasError := false

	// Run unit tests
	status := mgr.GetStatus()
	if status.PHPUnit.Installed {
		cli.PrintTitle("1/4 - Running PHPUnit Unit Tests")
		fmt.Println()
		runner := testing.NewPHPUnitRunner(mgr, phpunitCfg)
		if err := runner.RunUnit("", ""); err != nil {
			cli.PrintError("Unit tests failed")
			hasError = true
		} else {
			cli.PrintSuccess("Unit tests passed")
		}
		fmt.Println()
	}

	// Run PHPStan
	if status.PHPStan.Installed {
		cli.PrintTitle("2/4 - Running PHPStan")
		fmt.Println()
		runner := testing.NewPHPStanRunner(mgr, phpstanCfg)
		if err := runner.Run(nil, -1); err != nil {
			cli.PrintError("PHPStan analysis failed")
			hasError = true
		} else {
			cli.PrintSuccess("PHPStan analysis passed")
		}
		fmt.Println()
	}

	// Run PHPCS
	if status.PHPCS.Installed {
		cli.PrintTitle("3/4 - Running PHP_CodeSniffer")
		fmt.Println()
		runner := testing.NewPHPCSRunner(mgr, phpcsCfg)
		if err := runner.Run(nil, ""); err != nil {
			cli.PrintError("Code style checks failed")
			hasError = true
		} else {
			cli.PrintSuccess("Code style checks passed")
		}
		fmt.Println()
	}

	// Run PHPMD
	if status.PHPMD.Installed {
		cli.PrintTitle("4/4 - Running PHP Mess Detector")
		fmt.Println()
		runner := testing.NewPHPMDRunner(mgr, phpmdCfg)
		if err := runner.Run(nil, ""); err != nil {
			cli.PrintError("Mess detection found issues")
			hasError = true
		} else {
			cli.PrintSuccess("No mess detected")
		}
		fmt.Println()
	}

	// Summary
	cli.PrintTitle("Summary")
	if hasError {
		cli.PrintError("Some checks failed")
		os.Exit(1)
	} else {
		cli.PrintSuccess("All checks passed!")
	}
}

func runTestStatus(cmd *cobra.Command, args []string) {
	mgr, err := getTestManager()
	if err != nil {
		cli.PrintError("%v", err)
		return
	}

	status := mgr.GetStatus()

	cli.PrintTitle("Testing Tools Status")
	fmt.Println()

	printToolStatus("PHPUnit", &status.PHPUnit)
	printToolStatus("Integration Tests", &status.Integration)
	printToolStatus("PHPStan", &status.PHPStan)
	printToolStatus("PHP_CodeSniffer", &status.PHPCS)
	printToolStatus("PHP Mess Detector", &status.PHPMD)

	fmt.Println()
	fmt.Println("Run", cli.Command("magebox test setup"), "to install missing tools")
}

func printToolStatus(name string, status *testing.ToolStatus) {
	installed := cli.Error("Not installed")
	if status.Installed {
		installed = cli.Success("Installed")
		if status.Version != "" {
			installed += fmt.Sprintf(" (%s)", status.Version)
		}
	}

	configured := ""
	if status.Installed {
		if status.Configured {
			configured = cli.Success("  Config: ") + status.ConfigPath
		} else {
			configured = cli.Warning("  No config file found")
		}
	}

	fmt.Printf("  %-20s %s\n", name+":", installed)
	if configured != "" {
		fmt.Println(configured)
	}
}

// askYesNo prompts for a yes/no answer with a default
func askYesNo(defaultYes bool) bool {
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input == "" {
		return defaultYes
	}
	return input == "y" || input == "yes"
}

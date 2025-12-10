package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/qoliber/magebox/internal/cli"
	"github.com/qoliber/magebox/internal/config"
	"github.com/qoliber/magebox/internal/dns"
	"github.com/qoliber/magebox/internal/docker"
	"github.com/qoliber/magebox/internal/nginx"
	"github.com/qoliber/magebox/internal/php"
	"github.com/qoliber/magebox/internal/platform"
	"github.com/qoliber/magebox/internal/project"
	"github.com/qoliber/magebox/internal/ssl"
	"github.com/qoliber/magebox/internal/updater"
	"github.com/qoliber/magebox/internal/varnish"
)

var version = "0.2.1"

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "magebox",
	Short: "MageBox - Modern Magento Development Environment",
	Long: `MageBox is a modern, fast development environment for Magento.

It uses native PHP-FPM, Nginx, and Varnish for maximum performance,
with Docker only for stateless services like MySQL, Redis, and OpenSearch.`,
	Version: version,
	Run: func(cmd *cobra.Command, args []string) {
		// Show logo when running without subcommand
		cli.PrintLogoSmall(version)
		fmt.Println()
		_ = cmd.Help()
	},
}

var initCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Initialize a new MageBox project",
	Long:  "Creates a .magebox configuration file in the current directory",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runInit,
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start project services",
	Long:  "Starts all services defined in .magebox for the current project",
	RunE:  runStart,
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop project services",
	Long:  "Stops all services for the current project",
	RunE:  runStop,
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show project status",
	Long:  "Shows the status of all services for the current project",
	RunE:  runStatus,
}

var phpCmd = &cobra.Command{
	Use:   "php [version]",
	Short: "Switch PHP version",
	Long:  "Switches the PHP version for the current project (updates .magebox.local)",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runPhp,
}

var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Open project shell",
	Long:  "Opens a shell with the correct PHP version in PATH",
	RunE:  runShell,
}

var cliCmd = &cobra.Command{
	Use:                "cli [args...]",
	Short:              "Run bin/magento command",
	Long:               "Runs a bin/magento command in the project context",
	DisableFlagParsing: true,
	RunE:               runCli,
}

var runCmd = &cobra.Command{
	Use:   "run <command>",
	Short: "Run custom project command",
	Long: `Runs a custom command defined in .magebox file.

Example .magebox commands:

  commands:
    deploy: "php bin/magento deploy:mode:set production"
    reindex:
      description: "Reindex all Magento indexes"
      run: "php bin/magento indexer:reindex"
    setup:
      description: "Run full setup"
      run: "php bin/magento setup:upgrade && php bin/magento cache:flush"

Then run with: magebox run deploy`,
	Args:              cobra.MinimumNArgs(1),
	RunE:              runCustomCommand,
	ValidArgsFunction: completeCustomCommands,
}

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Database operations",
	Long:  "Database management commands",
}

var dbImportCmd = &cobra.Command{
	Use:   "import [file]",
	Short: "Import database",
	Long:  "Imports a SQL file into the project database",
	Args:  cobra.ExactArgs(1),
	RunE:  runDbImport,
}

var dbExportCmd = &cobra.Command{
	Use:   "export [file]",
	Short: "Export database",
	Long:  "Exports the project database to a SQL file",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runDbExport,
}

var dbShellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Open database shell",
	Long:  "Opens a MySQL shell connected to the project database",
	RunE:  runDbShell,
}

var redisCmd = &cobra.Command{
	Use:   "redis",
	Short: "Redis operations",
	Long:  "Redis cache management commands",
}

var redisFlushCmd = &cobra.Command{
	Use:   "flush",
	Short: "Flush Redis cache",
	Long:  "Flushes all data from the Redis cache",
	RunE:  runRedisFlush,
}

var redisShellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Open Redis shell",
	Long:  "Opens a Redis CLI shell",
	RunE:  runRedisShell,
}

var redisInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show Redis info",
	Long:  "Shows Redis server information and statistics",
	RunE:  runRedisInfo,
}

var logsCmd = &cobra.Command{
	Use:   "logs [pattern]",
	Short: "Tail project logs",
	Long: `Tails log files from var/log directory.

Examples:
  magebox logs              # Tail all .log files
  magebox logs system.log   # Tail only system.log
  magebox logs "*.log"      # Tail all .log files
  magebox logs -f           # Follow mode (continuous)
  magebox logs -n 50        # Show last 50 lines`,
	Args: cobra.MaximumNArgs(1),
	RunE: runLogs,
}

var globalCmd = &cobra.Command{
	Use:   "global",
	Short: "Global service management",
	Long:  "Manage global MageBox services",
}

var globalStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start global services",
	Long:  "Starts Nginx, Varnish, and Docker services",
	RunE:  runGlobalStart,
}

var globalStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop global services",
	Long:  "Stops all global MageBox services",
	RunE:  runGlobalStop,
}

var globalStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show global status",
	Long:  "Shows status of all global services and registered projects",
	RunE:  runGlobalStatus,
}

var sslCmd = &cobra.Command{
	Use:   "ssl",
	Short: "SSL certificate management",
	Long:  "Manage SSL certificates for project domains",
}

var sslTrustCmd = &cobra.Command{
	Use:   "trust",
	Short: "Trust local CA",
	Long:  "Installs and trusts the local certificate authority",
	RunE:  runSslTrust,
}

var sslGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate certificates",
	Long:  "Generates SSL certificates for project domains",
	RunE:  runSslGenerate,
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install MageBox dependencies",
	Long:  "Checks and installs required dependencies for MageBox",
	RunE:  runInstall,
}

var varnishCmd = &cobra.Command{
	Use:   "varnish",
	Short: "Varnish cache management",
	Long:  "Manage Varnish full-page cache",
}

var varnishPurgeCmd = &cobra.Command{
	Use:   "purge [url]",
	Short: "Purge a URL from cache",
	Long:  "Purges a specific URL from Varnish cache",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runVarnishPurge,
}

var varnishFlushCmd = &cobra.Command{
	Use:   "flush",
	Short: "Flush all cache",
	Long:  "Flushes all content from Varnish cache",
	RunE:  runVarnishFlush,
}

var varnishStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Varnish status",
	Long:  "Shows Varnish cache statistics and status",
	RunE:  runVarnishStatus,
}

var dnsCmd = &cobra.Command{
	Use:   "dns",
	Short: "DNS configuration",
	Long:  "Manage DNS resolution for local domains",
}

var dnsSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Setup dnsmasq for wildcard DNS",
	Long: `Sets up dnsmasq to resolve *.test domains to localhost.

This eliminates the need to add each domain to /etc/hosts manually.
Requires dnsmasq to be installed first.`,
	RunE: runDnsSetup,
}

var dnsStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show DNS configuration status",
	Long:  "Shows current DNS resolution status and configuration",
	RunE:  runDnsStatus,
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "MageBox configuration",
	Long:  "View and modify MageBox global configuration",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long:  "Displays the current MageBox global configuration",
	RunE:  runConfigShow,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Sets a global configuration value.

Available keys:
  dns_mode     - DNS resolution mode: "hosts" or "dnsmasq"
  default_php  - Default PHP version for new projects (e.g., "8.2")
  tld          - Top-level domain for local dev (default: "test")
  portainer    - Enable Portainer Docker UI: "true" or "false"`,
	Args: cobra.ExactArgs(2),
	RunE: runConfigSet,
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize global configuration",
	Long:  "Creates the global configuration file with defaults",
	RunE:  runConfigInit,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all MageBox projects",
	Long:  "Lists all discovered MageBox projects from nginx vhosts",
	RunE:  runList,
}

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

var newCmd = &cobra.Command{
	Use:   "new [directory]",
	Short: "Create a new Magento/MageOS project",
	Long: `Creates a new Magento or MageOS project with interactive setup wizard.

This command will guide you through:
  1. Selecting Magento or MageOS distribution
  2. Choosing the version to install
  3. Configuring Composer authentication
  4. Selecting PHP version
  5. Choosing services (MySQL, Redis, OpenSearch, etc.)
  6. Optional sample data installation
  7. Database setup

Quick Mode (--quick):
  Skip all questions and install MageOS with sensible defaults:
  - MageOS 1.0.4 (latest stable, no auth required)
  - PHP 8.3
  - MySQL 8.0, Redis, OpenSearch
  - Sample data included
  - Domain: {directory}.test

Example:
  magebox new mystore              # Interactive wizard
  magebox new mystore --quick      # Quick install with defaults + sample data
  magebox new . --quick            # Quick install in current directory`,
	Args: cobra.ExactArgs(1),
	RunE: runNew,
}

// Log command flags
var (
	logsFollow bool
	logsLines  int
)

// New command flags
var (
	newQuick      bool
	newWithSample bool
)

func init() {
	// New command flags
	newCmd.Flags().BoolVarP(&newQuick, "quick", "q", false, "Quick install with defaults (MageOS + sample data)")
	newCmd.Flags().BoolVar(&newWithSample, "with-sample", false, "Include sample data (used with --quick)")

	// Logs command flags
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow log files for changes")
	logsCmd.Flags().IntVarP(&logsLines, "lines", "n", 20, "Number of lines to show initially")

	// Database subcommands
	dbCmd.AddCommand(dbImportCmd)
	dbCmd.AddCommand(dbExportCmd)
	dbCmd.AddCommand(dbShellCmd)

	// Redis subcommands
	redisCmd.AddCommand(redisFlushCmd)
	redisCmd.AddCommand(redisShellCmd)
	redisCmd.AddCommand(redisInfoCmd)

	// Global subcommands
	globalCmd.AddCommand(globalStartCmd)
	globalCmd.AddCommand(globalStopCmd)
	globalCmd.AddCommand(globalStatusCmd)

	// SSL subcommands
	sslCmd.AddCommand(sslTrustCmd)
	sslCmd.AddCommand(sslGenerateCmd)

	// Varnish subcommands
	varnishCmd.AddCommand(varnishPurgeCmd)
	varnishCmd.AddCommand(varnishFlushCmd)
	varnishCmd.AddCommand(varnishStatusCmd)

	// DNS subcommands
	dnsCmd.AddCommand(dnsSetupCmd)
	dnsCmd.AddCommand(dnsStatusCmd)

	// Config subcommands
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configInitCmd)

	// Self-update subcommands
	selfUpdateCmd.AddCommand(selfUpdateCheckCmd)

	// Root commands
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(phpCmd)
	rootCmd.AddCommand(shellCmd)
	rootCmd.AddCommand(cliCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(dbCmd)
	rootCmd.AddCommand(redisCmd)
	rootCmd.AddCommand(globalCmd)
	rootCmd.AddCommand(sslCmd)
	rootCmd.AddCommand(varnishCmd)
	rootCmd.AddCommand(dnsCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(selfUpdateCmd)
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(bootstrapCmd)
	rootCmd.AddCommand(newCmd)
}

// runInit initializes a new MageBox project
func runInit(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	// Determine project name
	var projectName string
	if len(args) > 0 {
		projectName = args[0]
	} else {
		// Use directory name
		projectName = filepath.Base(cwd)
		// Prompt for confirmation
		fmt.Printf("Project name [%s]: ", cli.Highlight(projectName))
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			projectName = input
		}
	}

	// Check if .magebox already exists
	configPath := filepath.Join(cwd, ".magebox")
	if _, err := os.Stat(configPath); err == nil {
		cli.PrintError(".magebox file already exists")
		return nil
	}

	p, err := getPlatform()
	if err != nil {
		return err
	}

	mgr := project.NewManager(p)
	if err := mgr.Init(cwd, projectName); err != nil {
		return err
	}

	cli.PrintSuccess("Created .magebox for project '%s'", projectName)
	fmt.Println()
	fmt.Printf("Domain: %s\n", cli.URL(projectName+".test"))
	fmt.Println()
	cli.PrintInfo("Next steps:")
	fmt.Println(cli.Bullet("Edit .magebox to customize your configuration"))
	fmt.Println(cli.Bullet("Run " + cli.Command("magebox start") + " to start your project"))

	return nil
}

// runStart starts all project services
func runStart(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	p, err := getPlatform()
	if err != nil {
		return err
	}

	cli.PrintTitle("Starting MageBox Services")
	fmt.Println()

	mgr := project.NewManager(p)

	// Validate first
	cfg, warnings, err := mgr.ValidateConfig(cwd)
	if err != nil {
		cli.PrintError("%v", err)
		return nil
	}

	// Show warnings
	for _, w := range warnings {
		cli.PrintWarning("%s", w)
	}

	// Start services
	result, err := mgr.Start(cwd)
	if err != nil {
		cli.PrintError("%v", err)
		return nil
	}

	// Show results
	fmt.Printf("Project: %s\n", cli.Highlight(cfg.Name))
	fmt.Printf("PHP:     %s\n", cli.Highlight(result.PHPVersion))
	fmt.Println()

	fmt.Println(cli.Header("Domains"))
	for _, d := range result.Domains {
		fmt.Printf("  %s\n", cli.URL("https://"+d))
	}
	fmt.Println()

	fmt.Println(cli.Header("Services"))
	for _, s := range result.Services {
		fmt.Printf("  %s %s\n", cli.Success(""), s)
	}
	fmt.Println()

	// Show warnings from start
	for _, w := range result.Warnings {
		cli.PrintWarning("%s", w)
	}

	// Show errors
	for _, e := range result.Errors {
		cli.PrintError("%v", e)
	}

	if len(result.Errors) == 0 {
		cli.PrintSuccess("Project started successfully!")
	}

	return nil
}

// runStop stops all project services
func runStop(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	p, err := getPlatform()
	if err != nil {
		return err
	}

	cli.PrintInfo("Stopping MageBox services...")

	mgr := project.NewManager(p)
	if err := mgr.Stop(cwd); err != nil {
		cli.PrintError("%v", err)
		return nil
	}

	cli.PrintSuccess("Project stopped successfully!")
	return nil
}

// runStatus shows project status
func runStatus(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	p, err := getPlatform()
	if err != nil {
		return err
	}

	mgr := project.NewManager(p)
	status, err := mgr.Status(cwd)
	if err != nil {
		cli.PrintError("%v", err)
		return nil
	}

	cli.PrintTitle("Project Status")
	fmt.Println()
	fmt.Printf("Project: %s\n", cli.Highlight(status.Name))
	fmt.Printf("Path:    %s\n", cli.Path(status.Path))
	fmt.Printf("PHP:     %s\n", cli.Highlight(status.PHPVersion))

	fmt.Println(cli.Header("Domains"))
	for _, d := range status.Domains {
		fmt.Printf("  %s\n", cli.URL(d))
	}

	fmt.Println(cli.Header("Services"))
	for _, svc := range status.Services {
		fmt.Printf("  %-20s %s\n", svc.Name, cli.Status(svc.IsRunning))
	}

	return nil
}

// runPhp shows or switches PHP version
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

	// Write to .magebox.local
	localConfigPath := filepath.Join(cwd, ".magebox.local")
	content := fmt.Sprintf("php: \"%s\"\n", newVersion)

	if err := os.WriteFile(localConfigPath, []byte(content), 0644); err != nil {
		cli.PrintError("Failed to write .magebox.local: %v", err)
		return nil
	}

	cli.PrintSuccess("Switched to PHP %s", newVersion)
	cli.PrintInfo("Run '%s' to apply the change", cli.Command("magebox start"))

	return nil
}

// runShell opens a shell with the correct PHP in PATH
func runShell(cmd *cobra.Command, args []string) error {
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

	// Get PHP binary path
	phpBin := p.PHPBinary(cfg.PHP)

	// Set up environment
	phpDir := filepath.Dir(phpBin)
	path := phpDir + ":" + os.Getenv("PATH")

	// Get shell
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}

	fmt.Printf("Opening shell with PHP %s\n", cfg.PHP)

	shellCmd := exec.Command(shell)
	shellCmd.Dir = cwd
	shellCmd.Env = append(os.Environ(), "PATH="+path)
	shellCmd.Stdin = os.Stdin
	shellCmd.Stdout = os.Stdout
	shellCmd.Stderr = os.Stderr

	return shellCmd.Run()
}

// runCli runs a bin/magento command
func runCli(cmd *cobra.Command, args []string) error {
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

	// Get PHP binary
	phpBin := p.PHPBinary(cfg.PHP)

	// Build command
	magentoCmd := append([]string{filepath.Join(cwd, "bin/magento")}, args...)

	execCmd := exec.Command(phpBin, magentoCmd...)
	execCmd.Dir = cwd
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	return execCmd.Run()
}

// runDbImport imports a database from a file
func runDbImport(cmd *cobra.Command, args []string) error {
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

	// Determine service name
	var serviceName string
	if cfg.Services.MySQL != nil && cfg.Services.MySQL.Enabled {
		serviceName = fmt.Sprintf("mysql%s", strings.ReplaceAll(cfg.Services.MySQL.Version, ".", ""))
	} else if cfg.Services.MariaDB != nil && cfg.Services.MariaDB.Enabled {
		serviceName = fmt.Sprintf("mariadb%s", strings.ReplaceAll(cfg.Services.MariaDB.Version, ".", ""))
	} else {
		cli.PrintError("No database service configured in .magebox")
		return nil
	}

	composeGen := docker.NewComposeGenerator(p)
	composeFile := composeGen.ComposeFilePath()

	sqlFile := args[0]
	fmt.Printf("Importing %s into database '%s'...\n", sqlFile, cfg.Name)

	// Use docker compose exec to import
	importCmd := exec.Command("docker", "compose", "-f", composeFile, "exec", "-T", serviceName,
		"mysql", "-uroot", "-pmagebox", cfg.Name)

	file, err := os.Open(sqlFile)
	if err != nil {
		return fmt.Errorf("failed to open SQL file: %w", err)
	}
	defer file.Close()

	importCmd.Stdin = file
	importCmd.Stdout = os.Stdout
	importCmd.Stderr = os.Stderr

	if err := importCmd.Run(); err != nil {
		return fmt.Errorf("import failed: %w", err)
	}

	fmt.Println("Import completed successfully!")
	return nil
}

// runDbExport exports the database to a file
func runDbExport(cmd *cobra.Command, args []string) error {
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

	// Determine output file
	var outputFile string
	if len(args) > 0 {
		outputFile = args[0]
	} else {
		outputFile = fmt.Sprintf("%s.sql", cfg.Name)
	}

	// Determine service name
	var serviceName string
	if cfg.Services.MySQL != nil && cfg.Services.MySQL.Enabled {
		serviceName = fmt.Sprintf("mysql%s", strings.ReplaceAll(cfg.Services.MySQL.Version, ".", ""))
	} else if cfg.Services.MariaDB != nil && cfg.Services.MariaDB.Enabled {
		serviceName = fmt.Sprintf("mariadb%s", strings.ReplaceAll(cfg.Services.MariaDB.Version, ".", ""))
	} else {
		return fmt.Errorf("no database service configured")
	}

	composeGen := docker.NewComposeGenerator(p)
	composeFile := composeGen.ComposeFilePath()

	fmt.Printf("Exporting database '%s' to %s...\n", cfg.Name, outputFile)

	// Use docker compose exec to export
	exportCmd := exec.Command("docker", "compose", "-f", composeFile, "exec", "-T", serviceName,
		"mysqldump", "-uroot", "-pmagebox", cfg.Name)

	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	exportCmd.Stdout = file
	exportCmd.Stderr = os.Stderr

	if err := exportCmd.Run(); err != nil {
		return fmt.Errorf("export failed: %w", err)
	}

	fmt.Println("Export completed successfully!")
	return nil
}

// runDbShell opens a database shell
func runDbShell(cmd *cobra.Command, args []string) error {
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

	// Determine service name
	var serviceName string
	if cfg.Services.MySQL != nil && cfg.Services.MySQL.Enabled {
		serviceName = fmt.Sprintf("mysql%s", strings.ReplaceAll(cfg.Services.MySQL.Version, ".", ""))
	} else if cfg.Services.MariaDB != nil && cfg.Services.MariaDB.Enabled {
		serviceName = fmt.Sprintf("mariadb%s", strings.ReplaceAll(cfg.Services.MariaDB.Version, ".", ""))
	} else {
		cli.PrintError("No database service configured in .magebox")
		return nil
	}

	composeGen := docker.NewComposeGenerator(p)
	composeFile := composeGen.ComposeFilePath()

	fmt.Printf("Connecting to database '%s'...\n", cfg.Name)

	shellCmd := exec.Command("docker", "compose", "-f", composeFile, "exec", serviceName,
		"mysql", "-uroot", "-pmagebox", cfg.Name)
	shellCmd.Stdin = os.Stdin
	shellCmd.Stdout = os.Stdout
	shellCmd.Stderr = os.Stderr

	return shellCmd.Run()
}

// runGlobalStart starts global services
func runGlobalStart(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	cli.PrintTitle("Starting Global Services")
	fmt.Println()

	// Check Docker is running
	dockerCmd := exec.Command("docker", "info")
	if dockerCmd.Run() != nil {
		cli.PrintError("Docker is not running. Please start Docker first.")
		return nil
	}

	// Start Nginx
	fmt.Print("  Nginx... ")
	nginxCtrl := nginx.NewController(p)
	if nginxCtrl.IsRunning() {
		fmt.Println(cli.Success("already running"))
	} else if err := nginxCtrl.Start(); err != nil {
		fmt.Println(cli.Error("failed: " + err.Error()))
	} else {
		fmt.Println(cli.Success("started"))
	}

	// Start Docker services
	composeGen := docker.NewComposeGenerator(p)
	composeFile := composeGen.ComposeFilePath()

	// Check if compose file exists
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		cli.PrintWarning("Docker services not configured. Run " + cli.Command("magebox bootstrap") + " first.")
	} else {
		fmt.Print("  Docker services... ")
		dockerCtrl := docker.NewDockerController(composeFile)
		if err := dockerCtrl.Up(); err != nil {
			fmt.Println(cli.Error("failed: " + err.Error()))
		} else {
			fmt.Println(cli.Success("started"))
		}

		// List running services
		if services, err := dockerCtrl.GetRunningServices(); err == nil && len(services) > 0 {
			for _, svc := range services {
				fmt.Printf("    %s %s\n", cli.Success("✓"), svc)
			}
		}
	}

	fmt.Println()
	cli.PrintSuccess("Global services started!")
	fmt.Println()
	cli.PrintInfo("Run " + cli.Command("magebox start") + " in your project directory to start project services.")

	return nil
}

// runGlobalStop stops all global services
func runGlobalStop(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	fmt.Println("Stopping global services...")

	// Stop Nginx
	fmt.Print("  Nginx... ")
	nginxCtrl := nginx.NewController(p)
	if err := nginxCtrl.Stop(); err != nil {
		fmt.Printf("failed: %v\n", err)
	} else {
		fmt.Println("stopped")
	}

	// Stop Docker services
	composeGen := docker.NewComposeGenerator(p)
	composeFile := composeGen.ComposeFilePath()
	if _, err := os.Stat(composeFile); err == nil {
		fmt.Print("  Docker services... ")
		dockerCtrl := docker.NewDockerController(composeFile)
		if err := dockerCtrl.Down(); err != nil {
			fmt.Printf("failed: %v\n", err)
		} else {
			fmt.Println("stopped")
		}
	}

	fmt.Println("\nGlobal services stopped!")
	return nil
}

// runGlobalStatus shows status of all global services
func runGlobalStatus(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	cli.PrintTitle("Global Services Status")
	fmt.Println()

	// Check Nginx
	nginxCtrl := nginx.NewController(p)
	fmt.Printf("  %-20s %s\n", "Nginx", cli.Status(nginxCtrl.IsRunning()))

	// Check Docker
	dockerInstalled := platform.CommandExists("docker")
	dockerRunning := false
	if dockerInstalled {
		dockerCmd := exec.Command("docker", "info")
		dockerRunning = dockerCmd.Run() == nil
	}
	if !dockerInstalled {
		fmt.Printf("  %-20s %s\n", "Docker", cli.StatusInstalled(false))
	} else {
		fmt.Printf("  %-20s %s\n", "Docker", cli.Status(dockerRunning))
	}

	// Check mkcert
	fmt.Printf("  %-20s %s\n", "mkcert", cli.StatusInstalled(platform.CommandExists("mkcert")))

	// List PHP versions
	fmt.Println(cli.Header("PHP Versions"))
	detector := php.NewDetector(p)
	for _, v := range php.SupportedVersions {
		version := detector.Detect(v)
		var status string
		if !version.Installed {
			status = cli.StatusInstalled(false)
		} else if version.FPMRunning {
			status = cli.Status(true)
		} else {
			status = cli.StatusInstalled(true)
		}
		fmt.Printf("  %-20s %s\n", "PHP "+v, status)
	}

	return nil
}

// runSslTrust trusts the local CA
func runSslTrust(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	sslMgr := ssl.NewManager(p)

	if !sslMgr.IsMkcertInstalled() {
		fmt.Println("mkcert is not installed")
		fmt.Println()
		fmt.Println("Install it with:")
		fmt.Printf("  %s\n", p.MkcertInstallCommand())
		return nil
	}

	fmt.Println("Installing and trusting local CA...")

	if err := sslMgr.EnsureCAInstalled(); err != nil {
		return err
	}

	fmt.Println("Local CA is now trusted!")
	fmt.Println("SSL certificates will be automatically generated when you run 'magebox start'")

	return nil
}

// runSslGenerate generates SSL certificates for project domains
func runSslGenerate(cmd *cobra.Command, args []string) error {
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

	sslMgr := ssl.NewManager(p)

	if !sslMgr.IsMkcertInstalled() {
		fmt.Println("mkcert is not installed")
		fmt.Println()
		fmt.Println("Install it with:")
		fmt.Printf("  %s\n", p.MkcertInstallCommand())
		fmt.Println()
		fmt.Println("Then run: magebox ssl:trust")
		return nil
	}

	fmt.Println("Generating SSL certificates...")

	for _, domain := range cfg.Domains {
		if domain.IsSSLEnabled() {
			baseDomain := ssl.ExtractBaseDomain(domain.Host)
			fmt.Printf("  %s... ", baseDomain)
			cert, err := sslMgr.GenerateCert(baseDomain)
			if err != nil {
				fmt.Printf("failed: %v\n", err)
				continue
			}
			fmt.Printf("done\n")
			fmt.Printf("    Cert: %s\n", cert.CertFile)
			fmt.Printf("    Key:  %s\n", cert.KeyFile)
		}
	}

	fmt.Println("\nSSL certificates generated!")
	return nil
}

// runInstall checks and installs MageBox dependencies
func runInstall(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	cli.PrintTitle("MageBox Installation Check")
	fmt.Println()

	allOk := true

	// Check Docker
	dockerInstalled := platform.CommandExists("docker")
	fmt.Printf("%-12s %s\n", "Docker:", cli.StatusInstalled(dockerInstalled))
	if !dockerInstalled {
		fmt.Printf("  Install: %s\n", cli.Command(p.DockerInstallCommand()))
		allOk = false
	}

	// Check Nginx
	nginxInstalled := p.IsNginxInstalled()
	fmt.Printf("%-12s %s\n", "Nginx:", cli.StatusInstalled(nginxInstalled))
	if !nginxInstalled {
		fmt.Printf("  Install: %s\n", cli.Command(p.NginxInstallCommand()))
		allOk = false
	}

	// Check mkcert
	mkcertInstalled := platform.CommandExists("mkcert")
	fmt.Printf("%-12s %s\n", "mkcert:", cli.StatusInstalled(mkcertInstalled))
	if !mkcertInstalled {
		fmt.Printf("  Install: %s\n", cli.Command(p.MkcertInstallCommand()))
		allOk = false
	}

	// Check PHP versions
	fmt.Println(cli.Header("PHP Versions"))
	detector := php.NewDetector(p)
	installedPHP := false
	for _, v := range php.SupportedVersions {
		version := detector.Detect(v)
		if version.Installed {
			fmt.Printf("  PHP %s: %s\n", v, cli.StatusInstalled(true))
			installedPHP = true
		}
	}
	if !installedPHP {
		cli.PrintWarning("No PHP versions installed!")
		fmt.Printf("  Install at least one version:\n")
		fmt.Printf("    %s\n", cli.Command(p.PHPInstallCommand("8.2")))
		allOk = false
	}

	fmt.Println()
	if allOk {
		cli.PrintSuccess("All dependencies are installed!")
		fmt.Println()
		cli.PrintInfo("Next steps:")
		fmt.Println(cli.Bullet("Run " + cli.Command("magebox ssl trust") + " to set up SSL"))
		fmt.Println(cli.Bullet("Run " + cli.Command("magebox init") + " in your project directory"))
		fmt.Println(cli.Bullet("Run " + cli.Command("magebox start") + " to start your project"))
	} else {
		cli.PrintWarning("Some dependencies are missing. Install them and run '%s' again.", cli.Command("magebox install"))
	}

	return nil
}

// runVarnishPurge purges a URL from Varnish cache
func runVarnishPurge(cmd *cobra.Command, args []string) error {
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

	vclGen := varnish.NewVCLGenerator(p)
	ctrl := varnish.NewController(p, vclGen.VCLFilePath())

	if !ctrl.IsRunning() {
		fmt.Println("Varnish is not running")
		return nil
	}

	// Determine URL to purge
	url := "/"
	if len(args) > 0 {
		url = args[0]
	}

	// Purge for each domain
	for _, domain := range cfg.Domains {
		fmt.Printf("Purging %s%s... ", domain.Host, url)
		if err := ctrl.Purge(domain.Host, url); err != nil {
			fmt.Printf("failed: %v\n", err)
		} else {
			fmt.Println("done")
		}
	}

	return nil
}

// runVarnishFlush flushes all Varnish cache
func runVarnishFlush(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	vclGen := varnish.NewVCLGenerator(p)
	ctrl := varnish.NewController(p, vclGen.VCLFilePath())

	if !ctrl.IsRunning() {
		fmt.Println("Varnish is not running")
		return nil
	}

	fmt.Print("Flushing all Varnish cache... ")
	if err := ctrl.FlushAll(); err != nil {
		return fmt.Errorf("failed: %w", err)
	}
	fmt.Println("done")

	return nil
}

// runVarnishStatus shows Varnish status
func runVarnishStatus(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	vclGen := varnish.NewVCLGenerator(p)
	ctrl := varnish.NewController(p, vclGen.VCLFilePath())

	fmt.Println("Varnish Status")
	fmt.Println("==============")

	if ctrl.IsRunning() {
		fmt.Println("Status: running")

		// Try to get stats
		statsCmd := exec.Command("varnishstat", "-1")
		output, err := statsCmd.Output()
		if err == nil {
			lines := strings.Split(string(output), "\n")
			fmt.Println()
			fmt.Println("Statistics:")
			for _, line := range lines {
				if strings.Contains(line, "cache_hit") ||
					strings.Contains(line, "cache_miss") ||
					strings.Contains(line, "client_req") {
					fmt.Printf("  %s\n", strings.TrimSpace(line))
				}
			}
		}
	} else {
		fmt.Println("Status: stopped")
	}

	return nil
}

// runCustomCommand runs a custom command defined in .magebox
func runCustomCommand(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	p, err := getPlatform()
	if err != nil {
		return err
	}

	cfg, cfgOk := loadProjectConfig(cwd)
	if !cfgOk {
		return nil
	}

	cmdName := args[0]

	// Check if command exists
	command, ok := cfg.Commands[cmdName]
	if !ok {
		// List available commands
		fmt.Printf("Command '%s' not found in .magebox\n\n", cmdName)
		if len(cfg.Commands) > 0 {
			fmt.Println("Available commands:")
			for name, cmd := range cfg.Commands {
				if cmd.Description != "" {
					fmt.Printf("  %-15s %s\n", name, cmd.Description)
				} else {
					fmt.Printf("  %s\n", name)
				}
			}
		} else {
			fmt.Println("No commands defined in .magebox")
			fmt.Println()
			fmt.Println("Add commands to your .magebox file:")
			fmt.Println()
			fmt.Println("  commands:")
			fmt.Println("    deploy: \"php bin/magento deploy:mode:set production\"")
			fmt.Println("    reindex:")
			fmt.Println("      description: \"Reindex all Magento indexes\"")
			fmt.Println("      run: \"php bin/magento indexer:reindex\"")
		}
		return nil
	}

	// Get PHP path
	detector := php.NewDetector(p)
	version := detector.Detect(cfg.PHP)
	if !version.Installed {
		return fmt.Errorf("PHP %s is not installed", cfg.PHP)
	}

	// Build command to run
	cmdToRun := command.Run

	// Append any additional arguments passed after the command name
	if len(args) > 1 {
		cmdToRun = cmdToRun + " " + strings.Join(args[1:], " ")
	}

	// Set up environment with correct PHP in PATH
	phpDir := filepath.Dir(version.PHPBinary)
	currentPath := os.Getenv("PATH")
	newPath := phpDir + string(os.PathListSeparator) + currentPath

	fmt.Printf("Running: %s\n\n", cmdToRun)

	// Execute command via shell
	shellCmd := exec.Command("bash", "-c", cmdToRun)
	shellCmd.Dir = cwd
	shellCmd.Stdin = os.Stdin
	shellCmd.Stdout = os.Stdout
	shellCmd.Stderr = os.Stderr
	shellCmd.Env = append(os.Environ(), "PATH="+newPath)

	// Add project env vars
	for key, value := range cfg.Env {
		shellCmd.Env = append(shellCmd.Env, key+"="+value)
	}

	return shellCmd.Run()
}

// completeCustomCommands provides tab completion for custom commands
func completeCustomCommands(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	cwd, err := getCwd()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	cfg, err := config.LoadFromPath(cwd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []string
	for name, command := range cfg.Commands {
		if strings.HasPrefix(name, toComplete) {
			if command.Description != "" {
				completions = append(completions, name+"\t"+command.Description)
			} else {
				completions = append(completions, name)
			}
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

// runLogs tails Magento log files
func runLogs(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	// Determine log directory
	logDir := filepath.Join(cwd, "var", "log")

	// Check if log directory exists
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		cli.PrintError("Log directory not found: %s", logDir)
		cli.PrintInfo("Make sure you're in a Magento project root directory")
		return nil
	}

	// Determine pattern
	pattern := "*.log"
	if len(args) > 0 {
		pattern = args[0]
	}

	cli.PrintTitle("MageBox Log Viewer")
	fmt.Printf("Directory: %s\n", cli.Path(logDir))
	fmt.Printf("Pattern: %s\n", pattern)

	// Create tailer
	tailer := cli.NewLogTailer(logDir, pattern, logsFollow, logsLines)

	// Handle Ctrl+C
	if logsFollow {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigChan
			tailer.Stop()
		}()
	}

	return tailer.Start()
}

// runRedisFlush flushes all Redis data
func runRedisFlush(cmd *cobra.Command, args []string) error {
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

	if !cfg.Services.HasRedis() {
		cli.PrintError("Redis is not configured in .magebox")
		return nil
	}

	composeGen := docker.NewComposeGenerator(p)
	composeFile := composeGen.ComposeFilePath()

	cli.PrintInfo("Flushing Redis cache...")

	// Run redis-cli FLUSHALL
	flushCmd := exec.Command("docker", "compose", "-f", composeFile, "exec", "-T", "redis",
		"redis-cli", "FLUSHALL")
	output, err := flushCmd.CombinedOutput()
	if err != nil {
		cli.PrintError("Failed to flush Redis: %v", err)
		return nil
	}

	if strings.TrimSpace(string(output)) == "OK" {
		cli.PrintSuccess("Redis cache flushed successfully")
	} else {
		fmt.Println(string(output))
	}

	return nil
}

// runRedisShell opens an interactive Redis CLI
func runRedisShell(cmd *cobra.Command, args []string) error {
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

	if !cfg.Services.HasRedis() {
		cli.PrintError("Redis is not configured in .magebox")
		return nil
	}

	composeGen := docker.NewComposeGenerator(p)
	composeFile := composeGen.ComposeFilePath()

	cli.PrintInfo("Connecting to Redis...")
	fmt.Println()

	// Open interactive redis-cli
	shellCmd := exec.Command("docker", "compose", "-f", composeFile, "exec", "redis", "redis-cli")
	shellCmd.Stdin = os.Stdin
	shellCmd.Stdout = os.Stdout
	shellCmd.Stderr = os.Stderr

	return shellCmd.Run()
}

// runRedisInfo shows Redis server information
func runRedisInfo(cmd *cobra.Command, args []string) error {
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

	if !cfg.Services.HasRedis() {
		cli.PrintError("Redis is not configured in .magebox")
		return nil
	}

	composeGen := docker.NewComposeGenerator(p)
	composeFile := composeGen.ComposeFilePath()

	cli.PrintTitle("Redis Information")
	fmt.Println()

	// Get Redis info
	infoCmd := exec.Command("docker", "compose", "-f", composeFile, "exec", "-T", "redis",
		"redis-cli", "INFO")
	output, err := infoCmd.Output()
	if err != nil {
		cli.PrintError("Failed to get Redis info: %v", err)
		return nil
	}

	// Parse and display key info
	lines := strings.Split(string(output), "\n")
	sections := []string{"Server", "Memory", "Stats", "Keyspace"}
	currentSection := ""

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Section header
		if strings.HasPrefix(line, "#") {
			sectionName := strings.TrimPrefix(line, "# ")
			for _, s := range sections {
				if sectionName == s {
					currentSection = sectionName
					fmt.Println(cli.Header(sectionName))
					break
				}
			}
			continue
		}

		// Only show lines from selected sections
		if currentSection == "" {
			continue
		}

		// Parse key:value
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := parts[0]
			value := parts[1]

			// Highlight important values
			switch key {
			case "redis_version", "used_memory_human", "connected_clients",
				"total_connections_received", "total_commands_processed",
				"keyspace_hits", "keyspace_misses":
				fmt.Printf("  %s: %s\n", key, cli.Highlight(value))
			default:
				if currentSection == "Keyspace" {
					fmt.Printf("  %s: %s\n", key, cli.Highlight(value))
				}
			}
		}
	}

	return nil
}

// runDnsSetup sets up dnsmasq for wildcard DNS resolution
func runDnsSetup(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	dnsMgr := dns.NewDnsmasqManager(p)

	// Check if dnsmasq is installed
	if !dnsMgr.IsInstalled() {
		cli.PrintError("dnsmasq is not installed")
		fmt.Println()
		fmt.Printf("Install it with: %s\n", cli.Command(dnsMgr.InstallCommand()))
		return nil
	}

	cli.PrintInfo("Setting up dnsmasq for *.test domain resolution...")

	// Configure dnsmasq
	if err := dnsMgr.Configure(); err != nil {
		cli.PrintError("Failed to configure dnsmasq: %v", err)
		return nil
	}

	// Start/restart dnsmasq
	if dnsMgr.IsRunning() {
		if err := dnsMgr.Restart(); err != nil {
			cli.PrintError("Failed to restart dnsmasq: %v", err)
			return nil
		}
	} else {
		if err := dnsMgr.Start(); err != nil {
			cli.PrintError("Failed to start dnsmasq: %v", err)
			return nil
		}
	}

	// Update global config
	homeDir, _ := os.UserHomeDir()
	globalCfg, _ := config.LoadGlobalConfig(homeDir)
	globalCfg.DNSMode = "dnsmasq"
	_ = config.SaveGlobalConfig(homeDir, globalCfg)

	cli.PrintSuccess("dnsmasq configured successfully!")
	fmt.Println()
	cli.PrintInfo("All *.test domains now resolve to 127.0.0.1")
	fmt.Println(cli.Bullet("No need to edit /etc/hosts for new projects"))
	fmt.Println(cli.Bullet("Test with: " + cli.Command("dig test.test @127.0.0.1")))

	return nil
}

// runDnsStatus shows DNS configuration status
func runDnsStatus(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	cli.PrintTitle("DNS Configuration Status")
	fmt.Println()

	// Check global config
	homeDir, _ := os.UserHomeDir()
	globalCfg, _ := config.LoadGlobalConfig(homeDir)

	fmt.Printf("DNS Mode:      %s\n", cli.Highlight(globalCfg.DNSMode))
	fmt.Printf("TLD:           %s\n", cli.Highlight(globalCfg.GetTLD()))

	// Check dnsmasq status
	dnsMgr := dns.NewDnsmasqManager(p)
	status := dnsMgr.GetStatus()

	fmt.Println(cli.Header("dnsmasq"))
	fmt.Printf("  %-14s %s\n", "Installed:", cli.StatusInstalled(status.Installed))
	fmt.Printf("  %-14s %s\n", "Configured:", cli.StatusInstalled(status.Configured))
	fmt.Printf("  %-14s %s\n", "Running:", cli.Status(status.Running))

	if status.Running {
		fmt.Printf("  %-14s %s\n", "Resolution:", cli.Status(status.Resolving))
		if !status.Resolving {
			cli.PrintWarning("DNS resolution test failed. Check dnsmasq configuration.")
		}
	}

	if !status.Installed {
		fmt.Println()
		cli.PrintInfo("To enable wildcard DNS:")
		fmt.Println(cli.Bullet("Install: " + cli.Command(dnsMgr.InstallCommand())))
		fmt.Println(cli.Bullet("Setup: " + cli.Command("magebox dns setup")))
	}

	return nil
}

// runConfigShow shows the current global configuration
func runConfigShow(cmd *cobra.Command, args []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	cfg, err := config.LoadGlobalConfig(homeDir)
	if err != nil {
		cli.PrintError("Failed to load config: %v", err)
		return nil
	}

	cli.PrintTitle("MageBox Global Configuration")
	fmt.Println()

	configPath := config.GlobalConfigPath(homeDir)
	if config.GlobalConfigExists(homeDir) {
		fmt.Printf("Config file: %s\n", cli.Path(configPath))
	} else {
		fmt.Printf("Config file: %s (using defaults)\n", cli.Subtitle("not created"))
	}
	fmt.Println()

	fmt.Printf("  %-14s %s\n", "dns_mode:", cli.Highlight(cfg.DNSMode))
	fmt.Printf("  %-14s %s\n", "default_php:", cli.Highlight(cfg.DefaultPHP))
	fmt.Printf("  %-14s %s\n", "tld:", cli.Highlight(cfg.TLD))
	fmt.Printf("  %-14s %s\n", "portainer:", cli.Highlight(fmt.Sprintf("%v", cfg.Portainer)))
	fmt.Printf("  %-14s %s\n", "auto_start:", cli.Highlight(fmt.Sprintf("%v", cfg.AutoStart)))

	fmt.Println(cli.Header("Default Services"))
	if cfg.DefaultServices.MySQL != "" {
		fmt.Printf("  %-14s %s\n", "mysql:", cli.Highlight(cfg.DefaultServices.MySQL))
	}
	if cfg.DefaultServices.MariaDB != "" {
		fmt.Printf("  %-14s %s\n", "mariadb:", cli.Highlight(cfg.DefaultServices.MariaDB))
	}
	fmt.Printf("  %-14s %s\n", "redis:", cli.Highlight(fmt.Sprintf("%v", cfg.DefaultServices.Redis)))
	if cfg.DefaultServices.OpenSearch != "" {
		fmt.Printf("  %-14s %s\n", "opensearch:", cli.Highlight(cfg.DefaultServices.OpenSearch))
	}
	fmt.Printf("  %-14s %s\n", "rabbitmq:", cli.Highlight(fmt.Sprintf("%v", cfg.DefaultServices.RabbitMQ)))
	fmt.Printf("  %-14s %s\n", "mailpit:", cli.Highlight(fmt.Sprintf("%v", cfg.DefaultServices.Mailpit)))

	return nil
}

// runConfigSet sets a configuration value
func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	cfg, err := config.LoadGlobalConfig(homeDir)
	if err != nil {
		cli.PrintError("Failed to load config: %v", err)
		return nil
	}

	switch key {
	case "dns_mode":
		if value != "hosts" && value != "dnsmasq" {
			cli.PrintError("Invalid value for dns_mode. Use 'hosts' or 'dnsmasq'")
			return nil
		}
		cfg.DNSMode = value
	case "default_php":
		cfg.DefaultPHP = value
	case "tld":
		cfg.TLD = value
	case "portainer":
		cfg.Portainer = (value == "true" || value == "1" || value == "yes")
	case "auto_start":
		cfg.AutoStart = (value == "true" || value == "1" || value == "yes")
	default:
		cli.PrintError("Unknown configuration key: %s", key)
		fmt.Println()
		cli.PrintInfo("Available keys: dns_mode, default_php, tld, portainer, auto_start")
		return nil
	}

	if err := config.SaveGlobalConfig(homeDir, cfg); err != nil {
		cli.PrintError("Failed to save config: %v", err)
		return nil
	}

	cli.PrintSuccess("Configuration updated: %s = %s", key, value)
	return nil
}

// runConfigInit initializes the global configuration
func runConfigInit(cmd *cobra.Command, args []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := config.GlobalConfigPath(homeDir)

	if config.GlobalConfigExists(homeDir) {
		cli.PrintWarning("Configuration already exists at %s", configPath)
		return nil
	}

	if err := config.InitGlobalConfig(homeDir); err != nil {
		cli.PrintError("Failed to initialize config: %v", err)
		return nil
	}

	cli.PrintSuccess("Created global configuration at %s", configPath)
	fmt.Println()
	cli.PrintInfo("Edit the file or use " + cli.Command("magebox config set <key> <value>"))
	return nil
}

// runList lists all discovered MageBox projects
func runList(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	cli.PrintTitle("MageBox Projects")
	fmt.Println()

	discovery := project.NewProjectDiscovery(p)
	projects, err := discovery.DiscoverProjects()
	if err != nil {
		cli.PrintError("Failed to discover projects: %v", err)
		return nil
	}

	if len(projects) == 0 {
		cli.PrintInfo("No projects found")
		fmt.Println()
		fmt.Println(cli.Bullet("Run " + cli.Command("magebox init") + " in a Magento project directory"))
		fmt.Println(cli.Bullet("Then run " + cli.Command("magebox start") + " to start services"))
		return nil
	}

	for i, proj := range projects {
		// Project header
		if proj.HasConfig {
			fmt.Printf("%s %s\n", cli.Success(""), cli.Highlight(proj.Name))
		} else {
			fmt.Printf("%s %s %s\n", cli.Warning(""), proj.Name, cli.Subtitle("(no .magebox file)"))
		}

		// Details
		fmt.Printf("    Path: %s\n", cli.Path(proj.Path))
		if proj.PHPVersion != "" {
			fmt.Printf("    PHP:  %s\n", proj.PHPVersion)
		}
		if len(proj.Domains) > 0 {
			fmt.Printf("    URLs: ")
			for j, domain := range proj.Domains {
				if j > 0 {
					fmt.Print(", ")
				}
				fmt.Print(cli.URL("https://" + domain))
			}
			fmt.Println()
		}

		if i < len(projects)-1 {
			fmt.Println()
		}
	}

	fmt.Println()
	fmt.Printf("Total: %d project(s)\n", len(projects))

	return nil
}

// runSelfUpdate updates MageBox to the latest version
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
		cli.PrintInfo("You can build from source: " + cli.Command("go install github.com/qoliber/magebox@latest"))
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
	cli.PrintInfo("Run " + cli.Command("magebox --version") + " to verify")

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
		cli.PrintInfo("Run " + cli.Command("magebox self-update") + " to install")
	} else {
		cli.PrintSuccess("You're running the latest version!")
	}

	return nil
}

// MagentoVersion represents a Magento/MageOS version
type MagentoVersion struct {
	Name        string
	Version     string
	Package     string
	PHPVersions []string
	Default     bool
}

// Distribution types
const (
	DistMagento = "magento"
	DistMageOS  = "mageos"
)

// Available Magento versions
var magentoVersions = []MagentoVersion{
	{Name: "Magento 2.4.7-p3 (Latest)", Version: "2.4.7-p3", Package: "magento/project-community-edition", PHPVersions: []string{"8.3", "8.2"}, Default: true},
	{Name: "Magento 2.4.7-p2", Version: "2.4.7-p2", Package: "magento/project-community-edition", PHPVersions: []string{"8.3", "8.2"}},
	{Name: "Magento 2.4.7-p1", Version: "2.4.7-p1", Package: "magento/project-community-edition", PHPVersions: []string{"8.3", "8.2"}},
	{Name: "Magento 2.4.7", Version: "2.4.7", Package: "magento/project-community-edition", PHPVersions: []string{"8.3", "8.2"}},
	{Name: "Magento 2.4.6-p7", Version: "2.4.6-p7", Package: "magento/project-community-edition", PHPVersions: []string{"8.2", "8.1"}},
	{Name: "Magento 2.4.6-p6", Version: "2.4.6-p6", Package: "magento/project-community-edition", PHPVersions: []string{"8.2", "8.1"}},
	{Name: "Magento 2.4.5-p9", Version: "2.4.5-p9", Package: "magento/project-community-edition", PHPVersions: []string{"8.1"}},
}

// Available MageOS versions
var mageosVersions = []MagentoVersion{
	{Name: "MageOS 1.0.4 (Latest)", Version: "1.0.4", Package: "mage-os/project-community-edition", PHPVersions: []string{"8.3", "8.2"}, Default: true},
	{Name: "MageOS 1.0.3", Version: "1.0.3", Package: "mage-os/project-community-edition", PHPVersions: []string{"8.3", "8.2"}},
	{Name: "MageOS 1.0.2", Version: "1.0.2", Package: "mage-os/project-community-edition", PHPVersions: []string{"8.3", "8.2"}},
	{Name: "MageOS 1.0.1", Version: "1.0.1", Package: "mage-os/project-community-edition", PHPVersions: []string{"8.2", "8.1"}},
}

// runNew creates a new Magento/MageOS project with interactive setup
func runNew(cmd *cobra.Command, args []string) error {
	targetDir := args[0]
	reader := bufio.NewReader(os.Stdin)

	p, err := getPlatform()
	if err != nil {
		return err
	}

	cli.PrintLogoSmall(version)
	fmt.Println()

	// Check prerequisites
	if !platform.CommandExists("composer") {
		cli.PrintError("Composer is not installed!")
		fmt.Println()
		cli.PrintInfo("Install Composer first:")
		fmt.Println("  curl -sS https://getcomposer.org/installer | php")
		fmt.Println("  sudo mv composer.phar /usr/local/bin/composer")
		return nil
	}

	// Quick mode - skip all questions, use sensible defaults
	if newQuick {
		return runNewQuick(targetDir, p)
	}

	cli.PrintTitle("Create New Magento/MageOS Project")
	fmt.Println()

	// Step 1: Choose distribution
	fmt.Println(cli.Header("Step 1: Choose Distribution"))
	fmt.Println()
	fmt.Println("  [1] Magento Open Source (Adobe)")
	fmt.Println("  [2] MageOS (Community Fork)")
	fmt.Println()
	fmt.Print("Select distribution [1]: ")

	distChoice, _ := reader.ReadString('\n')
	distChoice = strings.TrimSpace(distChoice)
	if distChoice == "" {
		distChoice = "1"
	}

	var distribution string
	var versions []MagentoVersion
	if distChoice == "2" {
		distribution = DistMageOS
		versions = mageosVersions
		fmt.Println("  → MageOS selected")
	} else {
		distribution = DistMagento
		versions = magentoVersions
		fmt.Println("  → Magento Open Source selected")
	}
	fmt.Println()

	// Step 2: Choose version
	fmt.Println(cli.Header("Step 2: Choose Version"))
	fmt.Println()
	defaultIdx := 0
	for i, v := range versions {
		marker := "  "
		if v.Default {
			marker = "→ "
			defaultIdx = i + 1
		}
		fmt.Printf("  [%d] %s%s\n", i+1, marker, v.Name)
	}
	fmt.Println()
	fmt.Printf("Select version [%d]: ", defaultIdx)

	versionChoice, _ := reader.ReadString('\n')
	versionChoice = strings.TrimSpace(versionChoice)
	if versionChoice == "" {
		versionChoice = fmt.Sprintf("%d", defaultIdx)
	}

	versionIdx := 0
	_, _ = fmt.Sscanf(versionChoice, "%d", &versionIdx)
	if versionIdx < 1 || versionIdx > len(versions) {
		versionIdx = defaultIdx
	}
	selectedVersion := versions[versionIdx-1]
	fmt.Printf("  → %s selected\n", selectedVersion.Name)
	fmt.Println()

	// Step 3: PHP Version
	fmt.Println(cli.Header("Step 3: PHP Version"))
	fmt.Println()
	fmt.Printf("  Compatible versions: %s\n", strings.Join(selectedVersion.PHPVersions, ", "))
	fmt.Println()
	for i, phpV := range selectedVersion.PHPVersions {
		marker := "  "
		if i == 0 {
			marker = "→ "
		}
		fmt.Printf("  [%d] %sPHP %s\n", i+1, marker, phpV)
	}
	fmt.Println()
	fmt.Print("Select PHP version [1]: ")

	phpChoice, _ := reader.ReadString('\n')
	phpChoice = strings.TrimSpace(phpChoice)
	if phpChoice == "" {
		phpChoice = "1"
	}

	phpIdx := 0
	_, _ = fmt.Sscanf(phpChoice, "%d", &phpIdx)
	if phpIdx < 1 || phpIdx > len(selectedVersion.PHPVersions) {
		phpIdx = 1
	}
	selectedPHP := selectedVersion.PHPVersions[phpIdx-1]
	fmt.Printf("  → PHP %s selected\n", selectedPHP)
	fmt.Println()

	// Check if PHP version is installed
	detector := php.NewDetector(p)
	phpVersion := detector.Detect(selectedPHP)
	if !phpVersion.Installed {
		cli.PrintWarning("PHP %s is not installed!", selectedPHP)
		fmt.Printf("  Install: %s\n", cli.Command(p.PHPInstallCommand(selectedPHP)))
		fmt.Println()
		fmt.Print("Continue anyway? [y/N]: ")
		continueChoice, _ := reader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(continueChoice)) != "y" {
			return nil
		}
	}

	// Step 4: Composer Authentication (for Magento only)
	var composerUser, composerPass string
	if distribution == DistMagento {
		fmt.Println(cli.Header("Step 4: Composer Authentication"))
		fmt.Println()
		fmt.Println("  Magento requires authentication keys from marketplace.magento.com")
		fmt.Println("  Get your keys at: " + cli.URL("https://marketplace.magento.com/customer/accessKeys/"))
		fmt.Println()

		// Check for existing auth.json
		homeDir, _ := os.UserHomeDir()
		authFile := filepath.Join(homeDir, ".composer", "auth.json")
		hasAuth := false
		if _, err := os.Stat(authFile); err == nil {
			// Check if repo.magento.com exists in auth.json
			authContent, _ := os.ReadFile(authFile)
			if strings.Contains(string(authContent), "repo.magento.com") {
				hasAuth = true
				fmt.Println("  " + cli.Success("✓") + " Found existing Composer authentication")
			}
		}

		if !hasAuth {
			fmt.Print("Public Key (username): ")
			composerUser, _ = reader.ReadString('\n')
			composerUser = strings.TrimSpace(composerUser)

			fmt.Print("Private Key (password): ")
			composerPass, _ = reader.ReadString('\n')
			composerPass = strings.TrimSpace(composerPass)

			if composerUser == "" || composerPass == "" {
				cli.PrintError("Composer keys are required for Magento installation")
				return nil
			}
		}
		fmt.Println()
	} else {
		fmt.Println(cli.Header("Step 4: Composer Authentication"))
		fmt.Println()
		fmt.Println("  " + cli.Success("✓") + " MageOS does not require authentication keys")
		fmt.Println()
	}

	// Step 5: Services
	fmt.Println(cli.Header("Step 5: Database & Services"))
	fmt.Println()

	// MySQL version
	fmt.Println("  Database:")
	fmt.Println("  [1] → MySQL 8.0 (recommended)")
	fmt.Println("  [2]   MySQL 8.4")
	fmt.Println("  [3]   MariaDB 10.6")
	fmt.Println("  [4]   MariaDB 11.4")
	fmt.Println()
	fmt.Print("Select database [1]: ")

	dbChoice, _ := reader.ReadString('\n')
	dbChoice = strings.TrimSpace(dbChoice)
	if dbChoice == "" {
		dbChoice = "1"
	}

	var dbService, dbVersion string
	switch dbChoice {
	case "2":
		dbService, dbVersion = "mysql", "8.4"
	case "3":
		dbService, dbVersion = "mariadb", "10.6"
	case "4":
		dbService, dbVersion = "mariadb", "11.4"
	default:
		dbService, dbVersion = "mysql", "8.0"
	}
	fmt.Printf("  → %s %s selected\n", titleCase(dbService), dbVersion)
	fmt.Println()

	// Search engine
	fmt.Println("  Search Engine:")
	fmt.Println("  [1] → OpenSearch 2.12 (recommended)")
	fmt.Println("  [2]   Elasticsearch 8.11")
	fmt.Println("  [3]   None (use MySQL for catalog search)")
	fmt.Println()
	fmt.Print("Select search engine [1]: ")

	searchChoice, _ := reader.ReadString('\n')
	searchChoice = strings.TrimSpace(searchChoice)
	if searchChoice == "" {
		searchChoice = "1"
	}

	var searchEngine, searchVersion string
	switch searchChoice {
	case "2":
		searchEngine, searchVersion = "elasticsearch", "8.11"
	case "3":
		searchEngine, searchVersion = "", ""
	default:
		searchEngine, searchVersion = "opensearch", "2.12"
	}
	if searchEngine != "" {
		fmt.Printf("  → %s %s selected\n", titleCase(searchEngine), searchVersion)
	} else {
		fmt.Println("  → No search engine (MySQL search)")
	}
	fmt.Println()

	// Additional services
	fmt.Println("  Additional Services:")
	fmt.Print("  Enable Redis cache? [Y/n]: ")
	redisChoice, _ := reader.ReadString('\n')
	enableRedis := strings.ToLower(strings.TrimSpace(redisChoice)) != "n"

	fmt.Print("  Enable RabbitMQ? [y/N]: ")
	rabbitChoice, _ := reader.ReadString('\n')
	enableRabbitMQ := strings.ToLower(strings.TrimSpace(rabbitChoice)) == "y"

	fmt.Print("  Enable Mailpit (email testing)? [Y/n]: ")
	mailChoice, _ := reader.ReadString('\n')
	enableMailpit := strings.ToLower(strings.TrimSpace(mailChoice)) != "n"
	fmt.Println()

	// Step 6: Sample Data
	fmt.Println(cli.Header("Step 6: Sample Data"))
	fmt.Println()
	fmt.Println("  Sample data includes demo products, categories, and CMS content.")
	fmt.Print("  Install sample data? [y/N]: ")
	sampleChoice, _ := reader.ReadString('\n')
	installSampleData := strings.ToLower(strings.TrimSpace(sampleChoice)) == "y"
	fmt.Println()

	// Step 7: Project Details
	fmt.Println(cli.Header("Step 7: Project Details"))
	fmt.Println()

	// Determine project name from directory
	var projectDir string
	if targetDir == "." {
		projectDir, _ = os.Getwd()
	} else {
		if filepath.IsAbs(targetDir) {
			projectDir = targetDir
		} else {
			cwd, _ := os.Getwd()
			projectDir = filepath.Join(cwd, targetDir)
		}
	}
	projectName := filepath.Base(projectDir)

	fmt.Printf("  Project directory: %s\n", cli.Highlight(projectDir))
	fmt.Printf("  Project name [%s]: ", projectName)
	nameInput, _ := reader.ReadString('\n')
	nameInput = strings.TrimSpace(nameInput)
	if nameInput != "" {
		projectName = nameInput
	}

	// Domain
	defaultDomain := projectName + ".test"
	fmt.Printf("  Domain [%s]: ", defaultDomain)
	domainInput, _ := reader.ReadString('\n')
	domainInput = strings.TrimSpace(domainInput)
	if domainInput == "" {
		domainInput = defaultDomain
	}
	fmt.Println()

	// Summary
	fmt.Println(cli.Header("Summary"))
	fmt.Println()
	fmt.Printf("  Distribution:    %s\n", cli.Highlight(distribution))
	fmt.Printf("  Version:         %s\n", cli.Highlight(selectedVersion.Version))
	fmt.Printf("  PHP:             %s\n", cli.Highlight(selectedPHP))
	fmt.Printf("  Database:        %s %s\n", cli.Highlight(dbService), cli.Highlight(dbVersion))
	if searchEngine != "" {
		fmt.Printf("  Search:          %s %s\n", cli.Highlight(searchEngine), cli.Highlight(searchVersion))
	}
	fmt.Printf("  Redis:           %s\n", cli.Status(enableRedis))
	fmt.Printf("  RabbitMQ:        %s\n", cli.Status(enableRabbitMQ))
	fmt.Printf("  Mailpit:         %s\n", cli.Status(enableMailpit))
	fmt.Printf("  Sample Data:     %s\n", cli.Status(installSampleData))
	fmt.Printf("  Project:         %s\n", cli.Highlight(projectName))
	fmt.Printf("  Domain:          %s\n", cli.URL("https://"+domainInput))
	fmt.Println()

	fmt.Print("Proceed with installation? [Y/n]: ")
	proceedChoice, _ := reader.ReadString('\n')
	if strings.ToLower(strings.TrimSpace(proceedChoice)) == "n" {
		fmt.Println("Installation canceled.")
		return nil
	}
	fmt.Println()

	// Create directory if needed
	if targetDir != "." {
		if err := os.MkdirAll(projectDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Set up Composer auth if needed
	if composerUser != "" && composerPass != "" {
		cli.PrintInfo("Configuring Composer authentication...")
		authCmd := exec.Command("composer", "config", "--global", "http-basic.repo.magento.com", composerUser, composerPass)
		if err := authCmd.Run(); err != nil {
			cli.PrintWarning("Failed to configure Composer auth: %v", err)
		}
	}

	// Run Composer create-project
	cli.PrintTitle("Installing " + selectedVersion.Name)
	fmt.Println()

	composerArgs := []string{
		"create-project",
		"--repository-url=https://repo.magento.com/",
		selectedVersion.Package,
		projectDir,
		selectedVersion.Version,
	}

	// MageOS doesn't need repo.magento.com
	if distribution == DistMageOS {
		composerArgs = []string{
			"create-project",
			selectedVersion.Package,
			projectDir,
			selectedVersion.Version,
		}
	}

	composerCmd := exec.Command("composer", composerArgs...)
	composerCmd.Stdout = os.Stdout
	composerCmd.Stderr = os.Stderr
	composerCmd.Stdin = os.Stdin

	fmt.Println(cli.Command("composer " + strings.Join(composerArgs, " ")))
	fmt.Println()

	if err := composerCmd.Run(); err != nil {
		cli.PrintError("Composer installation failed: %v", err)
		return err
	}
	fmt.Println()

	// Create .magebox file
	cli.PrintInfo("Creating MageBox configuration...")

	mageboxConfig := fmt.Sprintf(`name: %s
domains:
  - host: %s
    root: pub
    ssl: true
php: "%s"
services:
`, projectName, domainInput, selectedPHP)

	// Add database
	if dbService == "mysql" {
		mageboxConfig += fmt.Sprintf("  mysql: \"%s\"\n", dbVersion)
	} else {
		mageboxConfig += fmt.Sprintf("  mariadb: \"%s\"\n", dbVersion)
	}

	// Add search
	if searchEngine == "opensearch" {
		mageboxConfig += fmt.Sprintf("  opensearch: \"%s\"\n", searchVersion)
	} else if searchEngine == "elasticsearch" {
		mageboxConfig += fmt.Sprintf("  elasticsearch: \"%s\"\n", searchVersion)
	}

	// Add other services
	if enableRedis {
		mageboxConfig += "  redis: true\n"
	}
	if enableRabbitMQ {
		mageboxConfig += "  rabbitmq: true\n"
	}
	if enableMailpit {
		mageboxConfig += "  mailpit: true\n"
	}

	// Add common commands
	mageboxConfig += `
commands:
  setup:
    description: "Run Magento setup"
    run: "php bin/magento setup:upgrade && php bin/magento cache:flush"
  reindex:
    description: "Reindex all indexes"
    run: "php bin/magento indexer:reindex"
  deploy:
    description: "Deploy static content"
    run: "php bin/magento setup:static-content:deploy -f"
  cache:
    description: "Flush all caches"
    run: "php bin/magento cache:flush"
`

	mageboxFile := filepath.Join(projectDir, ".magebox")
	if err := os.WriteFile(mageboxFile, []byte(mageboxConfig), 0644); err != nil {
		cli.PrintWarning("Failed to create .magebox file: %v", err)
	} else {
		fmt.Printf("  Created %s\n", cli.Highlight(".magebox"))
	}

	// Install sample data if requested
	if installSampleData {
		fmt.Println()
		cli.PrintInfo("Installing sample data...")

		sampleCmd := exec.Command("composer", "require", "magento/module-bundle-sample-data",
			"magento/module-catalog-sample-data", "magento/module-catalog-rule-sample-data",
			"magento/module-cms-sample-data", "magento/module-configurable-sample-data",
			"magento/module-customer-sample-data", "magento/module-downloadable-sample-data",
			"magento/module-grouped-sample-data", "magento/module-msrp-sample-data",
			"magento/module-offline-shipping-sample-data", "magento/module-product-links-sample-data",
			"magento/module-review-sample-data", "magento/module-sales-rule-sample-data",
			"magento/module-sales-sample-data", "magento/module-swatches-sample-data",
			"magento/module-tax-sample-data", "magento/module-theme-sample-data",
			"magento/module-widget-sample-data", "magento/module-wishlist-sample-data",
			"magento/sample-data-media", "--no-update")
		sampleCmd.Dir = projectDir
		sampleCmd.Stdout = os.Stdout
		sampleCmd.Stderr = os.Stderr

		if err := sampleCmd.Run(); err != nil {
			cli.PrintWarning("Sample data require failed: %v", err)
		}

		// Run composer update
		updateCmd := exec.Command("composer", "update")
		updateCmd.Dir = projectDir
		updateCmd.Stdout = os.Stdout
		updateCmd.Stderr = os.Stderr
		_ = updateCmd.Run()
	}

	// Success!
	fmt.Println()
	cli.PrintTitle("Installation Complete!")
	fmt.Println()
	cli.PrintSuccess("Project created successfully!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println(cli.Bullet("cd " + cli.Highlight(projectDir)))
	fmt.Println(cli.Bullet("Run " + cli.Command("magebox start") + " to start services"))
	fmt.Println(cli.Bullet("Run " + cli.Command("magebox cli setup:install") + " to complete Magento setup"))
	fmt.Println()
	fmt.Println("After setup, access your store at: " + cli.URL("https://"+domainInput))
	fmt.Println()

	return nil
}

// runNewQuick creates a new MageOS project with sensible defaults (no questions)
func runNewQuick(targetDir string, p *platform.Platform) error {
	cli.PrintTitle("Quick Install - MageOS with Sample Data")
	fmt.Println()

	// Defaults for quick mode
	selectedVersion := mageosVersions[0] // MageOS 1.0.4 (latest)
	selectedPHP := "8.3"
	dbVersion := "8.0"
	searchVersion := "2.12"

	// Determine project name and directory
	var projectName string
	var projectDir string

	if targetDir == "." {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		projectDir = cwd
		projectName = filepath.Base(cwd)
	} else {
		projectDir, _ = filepath.Abs(targetDir)
		projectName = filepath.Base(targetDir)
	}

	// Domain from project name
	domainInput := projectName + ".test"

	// Check PHP availability
	detector := php.NewDetector(p)
	installedVersions := detector.DetectInstalled()

	phpFound := false
	for _, v := range installedVersions {
		if v.Version == selectedPHP {
			phpFound = true
			break
		}
	}

	if !phpFound {
		// Try to find any compatible version
		for _, compatiblePHP := range selectedVersion.PHPVersions {
			for _, v := range installedVersions {
				if v.Version == compatiblePHP {
					selectedPHP = compatiblePHP
					phpFound = true
					cli.PrintWarning("PHP 8.3 not found, using PHP %s instead", selectedPHP)
					break
				}
			}
			if phpFound {
				break
			}
		}
	}

	if !phpFound {
		cli.PrintError("No compatible PHP version found!")
		fmt.Println()
		cli.PrintInfo("Install PHP 8.3 first:")
		fmt.Println("  macOS:  brew install php@8.3")
		fmt.Println("  Ubuntu: sudo apt install php8.3-fpm php8.3-cli ...")
		return nil
	}

	// Show what we're going to install
	fmt.Println("Configuration:")
	fmt.Println(cli.Bullet("Distribution: " + cli.Highlight("MageOS")))
	fmt.Println(cli.Bullet("Version:      " + cli.Highlight(selectedVersion.Name)))
	fmt.Println(cli.Bullet("PHP:          " + cli.Highlight(selectedPHP)))
	fmt.Println(cli.Bullet("Database:     " + cli.Highlight("MySQL "+dbVersion)))
	fmt.Println(cli.Bullet("Search:       " + cli.Highlight("OpenSearch "+searchVersion)))
	fmt.Println(cli.Bullet("Services:     " + cli.Highlight("Redis, Mailpit")))
	fmt.Println(cli.Bullet("Sample Data:  " + cli.Highlight("Yes")))
	fmt.Println(cli.Bullet("Directory:    " + cli.Highlight(projectDir)))
	fmt.Println(cli.Bullet("Domain:       " + cli.Highlight(domainInput)))
	fmt.Println()

	// Create directory if needed
	if targetDir != "." {
		if err := os.MkdirAll(projectDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Run Composer create-project
	cli.PrintTitle("Installing " + selectedVersion.Name)
	fmt.Println()

	composerArgs := []string{
		"create-project",
		selectedVersion.Package,
		projectDir,
		selectedVersion.Version,
	}

	composerCmd := exec.Command("composer", composerArgs...)
	composerCmd.Stdout = os.Stdout
	composerCmd.Stderr = os.Stderr
	composerCmd.Stdin = os.Stdin

	fmt.Println(cli.Command("composer " + strings.Join(composerArgs, " ")))
	fmt.Println()

	if err := composerCmd.Run(); err != nil {
		cli.PrintError("Composer installation failed: %v", err)
		return err
	}
	fmt.Println()

	// Create .magebox file
	cli.PrintInfo("Creating MageBox configuration...")

	mageboxConfig := fmt.Sprintf(`name: %s
domains:
  - host: %s
    root: pub
    ssl: true
php: "%s"
services:
  mysql: "%s"
  opensearch: "%s"
  redis: true
  mailpit: true

commands:
  setup:
    description: "Run Magento setup"
    run: "php bin/magento setup:upgrade && php bin/magento cache:flush"
  reindex:
    description: "Reindex all indexes"
    run: "php bin/magento indexer:reindex"
  deploy:
    description: "Deploy static content"
    run: "php bin/magento setup:static-content:deploy -f"
  cache:
    description: "Flush all caches"
    run: "php bin/magento cache:flush"
`, projectName, domainInput, selectedPHP, dbVersion, searchVersion)

	mageboxFile := filepath.Join(projectDir, ".magebox")
	if err := os.WriteFile(mageboxFile, []byte(mageboxConfig), 0644); err != nil {
		cli.PrintWarning("Failed to create .magebox file: %v", err)
	} else {
		fmt.Printf("  Created %s\n", cli.Highlight(".magebox"))
	}

	// Install sample data (always in quick mode)
	fmt.Println()
	cli.PrintInfo("Installing sample data (this may take a while)...")

	sampleCmd := exec.Command("composer", "require", "magento/module-bundle-sample-data",
		"magento/module-catalog-sample-data", "magento/module-catalog-rule-sample-data",
		"magento/module-cms-sample-data", "magento/module-configurable-sample-data",
		"magento/module-customer-sample-data", "magento/module-downloadable-sample-data",
		"magento/module-grouped-sample-data", "magento/module-msrp-sample-data",
		"magento/module-offline-shipping-sample-data", "magento/module-product-links-sample-data",
		"magento/module-review-sample-data", "magento/module-sales-rule-sample-data",
		"magento/module-sales-sample-data", "magento/module-swatches-sample-data",
		"magento/module-tax-sample-data", "magento/module-theme-sample-data",
		"magento/module-widget-sample-data", "magento/module-wishlist-sample-data",
		"magento/sample-data-media", "--no-update")
	sampleCmd.Dir = projectDir
	sampleCmd.Stdout = os.Stdout
	sampleCmd.Stderr = os.Stderr

	if err := sampleCmd.Run(); err != nil {
		cli.PrintWarning("Sample data require failed: %v", err)
	}

	// Run composer update
	updateCmd := exec.Command("composer", "update")
	updateCmd.Dir = projectDir
	updateCmd.Stdout = os.Stdout
	updateCmd.Stderr = os.Stderr
	_ = updateCmd.Run()

	// Success!
	fmt.Println()
	cli.PrintTitle("Installation Complete!")
	fmt.Println()
	cli.PrintSuccess("MageOS project created successfully!")
	fmt.Println()

	// Print the full setup command
	dbPort := "33080" // MySQL 8.0 default port

	fmt.Println("Next steps:")
	fmt.Println()
	fmt.Println(cli.Bullet("1. Start services:"))
	fmt.Println("      cd " + cli.Highlight(projectDir))
	fmt.Println("      " + cli.Command("magebox start"))
	fmt.Println()
	fmt.Println(cli.Bullet("2. Install Magento:"))
	installCmd := fmt.Sprintf(`magebox cli setup:install \
    --base-url=https://%s \
    --db-host=127.0.0.1:%s \
    --db-name=%s \
    --db-user=root \
    --db-password=magebox \
    --search-engine=opensearch \
    --opensearch-host=127.0.0.1 \
    --opensearch-port=9200 \
    --admin-firstname=Admin \
    --admin-lastname=User \
    --admin-email=admin@example.com \
    --admin-user=admin \
    --admin-password=Admin123!`, domainInput, dbPort, projectName)
	fmt.Println("      " + cli.Command(installCmd))
	fmt.Println()
	fmt.Println(cli.Bullet("3. Deploy sample data:"))
	fmt.Println("      " + cli.Command("magebox cli setup:upgrade"))
	fmt.Println("      " + cli.Command("magebox cli indexer:reindex"))
	fmt.Println("      " + cli.Command("magebox cli cache:flush"))
	fmt.Println()
	fmt.Println("After setup, access your store at: " + cli.URL("https://"+domainInput))
	fmt.Println("Admin panel: " + cli.URL("https://"+domainInput+"/admin"))
	fmt.Println()

	return nil
}

// titleCase converts a string to title case (first letter uppercase)
// Used instead of deprecated strings.Title
func titleCase(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

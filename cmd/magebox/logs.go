package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"qoliber/magebox/internal/cli"
	"qoliber/magebox/internal/config"
	"qoliber/magebox/internal/docker"
	"qoliber/magebox/internal/platform"
)

var logsFollowFlag bool
var logsLinesFlag int

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View logs for Magento or MageBox services",
	Long: `View application and service logs.

Without a subcommand, opens Magento system.log and exception.log in multitail.

Service-specific logs:
  magebox logs php      # PHP-FPM error logs
  magebox logs nginx    # Nginx access/error logs
  magebox logs mysql    # MySQL/MariaDB container logs
  magebox logs redis    # Redis container logs
  magebox logs varnish  # Varnish logs

Press 'q' to quit, 'b' to scroll back in history (multitail views).
Use -f to follow (tail) file-based logs, or Ctrl+C to stop container log streams.`,
	RunE: runLogs,
}

var logsPhpCmd = &cobra.Command{
	Use:   "php",
	Short: "View PHP-FPM logs",
	Long: `Tails PHP-FPM error logs for the current project.

Log files are located in ~/.magebox/logs/php-fpm/.
Use -f to follow the log output in real-time.`,
	RunE: runLogsPhp,
}

var logsNginxCmd = &cobra.Command{
	Use:   "nginx",
	Short: "View Nginx logs",
	Long: `Opens Nginx access and error logs for the current project in a split-screen view.

Log files are located in ~/.magebox/logs/nginx/.
Without -f, uses multitail for split-screen viewing.
With -f, tails all matching log files.`,
	RunE: runLogsNginx,
}

var logsMysqlCmd = &cobra.Command{
	Use:   "mysql",
	Short: "View MySQL/MariaDB logs",
	Long: `Streams logs from the MySQL or MariaDB Docker container.

Uses docker compose logs to stream the container output.
Press Ctrl+C to stop.`,
	RunE: runLogsMysql,
}

var logsRedisCmd = &cobra.Command{
	Use:   "redis",
	Short: "View Redis logs",
	Long: `Streams logs from the Redis Docker container.

Uses docker compose logs to stream the container output.
Press Ctrl+C to stop.`,
	RunE: runLogsRedis,
}

var logsVarnishCmd = &cobra.Command{
	Use:   "varnish",
	Short: "View Varnish logs",
	Long: `Streams varnishlog output from the Varnish container.

Press Ctrl+C to stop.`,
	RunE: runLogsVarnish,
}

func init() {
	logsCmd.PersistentFlags().BoolVarP(&logsFollowFlag, "follow", "f", false, "Follow log output (tail -f)")
	logsCmd.PersistentFlags().IntVarP(&logsLinesFlag, "lines", "n", 100, "Number of lines to show")

	logsCmd.AddCommand(logsPhpCmd)
	logsCmd.AddCommand(logsNginxCmd)
	logsCmd.AddCommand(logsMysqlCmd)
	logsCmd.AddCommand(logsRedisCmd)
	logsCmd.AddCommand(logsVarnishCmd)
	rootCmd.AddCommand(logsCmd)
}

func runLogs(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	// Check if multitail is installed
	if !platform.CommandExists("multitail") {
		cli.PrintError("multitail is not installed")
		cli.PrintInfo("Run 'magebox bootstrap' to install it, or install manually:")
		fmt.Println("  brew install multitail  # macOS")
		fmt.Println("  sudo dnf install multitail  # Fedora")
		fmt.Println("  sudo apt install multitail  # Ubuntu/Debian")
		return nil
	}

	// Check for log directory
	logDir := filepath.Join(cwd, "var", "log")
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		cli.PrintError("Log directory not found: %s", logDir)
		cli.PrintInfo("Make sure you're in a Magento project root directory")
		return nil
	}

	systemLog := filepath.Join(logDir, "system.log")
	exceptionLog := filepath.Join(logDir, "exception.log")

	// Create log files if they don't exist (touch them)
	for _, logFile := range []string{systemLog, exceptionLog} {
		if _, err := os.Stat(logFile); os.IsNotExist(err) {
			f, err := os.Create(logFile)
			if err != nil {
				cli.PrintWarning("Could not create %s: %v", filepath.Base(logFile), err)
				continue
			}
			f.Close()
		}
	}

	fmt.Println("Watching: " + cli.Path(logDir))
	fmt.Println("Press 'q' to quit, 'b' to scroll back")
	fmt.Println()

	// Run multitail with 2 columns
	// -s 2: split into 2 columns
	// -n 200: show last 200 lines
	// -m 500: scrollback buffer for 'b' key
	multitailCmd := exec.Command("multitail",
		"-s", "2",
		"-n", "200",
		"-m", "500",
		systemLog,
		exceptionLog,
	)
	multitailCmd.Stdin = os.Stdin
	multitailCmd.Stdout = os.Stdout
	multitailCmd.Stderr = os.Stderr

	return multitailCmd.Run()
}

func runLogsPhp(cmd *cobra.Command, args []string) error {
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

	logsDir := filepath.Join(p.MageBoxDir(), "logs", "php-fpm")

	// Find log files matching the project name
	logFiles := findProjectLogFiles(logsDir, cfg.Name)
	if len(logFiles) == 0 {
		cli.PrintError("No PHP-FPM log files found for project %s", cli.Highlight(cfg.Name))
		cli.PrintInfo("Log directory: %s", cli.Path(logsDir))
		cli.PrintInfo("PHP-FPM logs are created when the project is started")
		return nil
	}

	if logsFollowFlag {
		return tailFiles(logFiles)
	}

	// Use multitail if available, otherwise fall back to tail
	if platform.CommandExists("multitail") {
		return multitailFiles(logFiles)
	}
	return tailFiles(logFiles)
}

func runLogsNginx(cmd *cobra.Command, args []string) error {
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

	logsDir := filepath.Join(p.MageBoxDir(), "logs", "nginx")

	// Find log files matching the project domains
	var logFiles []string
	for _, domain := range cfg.Domains {
		accessLog := filepath.Join(logsDir, fmt.Sprintf("%s-access.log", domain.Host))
		errorLog := filepath.Join(logsDir, fmt.Sprintf("%s-error.log", domain.Host))

		if _, err := os.Stat(accessLog); err == nil {
			logFiles = append(logFiles, accessLog)
		}
		if _, err := os.Stat(errorLog); err == nil {
			logFiles = append(logFiles, errorLog)
		}
	}

	if len(logFiles) == 0 {
		cli.PrintError("No Nginx log files found for project %s", cli.Highlight(cfg.Name))
		cli.PrintInfo("Log directory: %s", cli.Path(logsDir))
		cli.PrintInfo("Nginx logs are created when the project is started")
		return nil
	}

	if logsFollowFlag {
		return tailFiles(logFiles)
	}

	// Use multitail if available for split-screen view
	if platform.CommandExists("multitail") {
		return multitailFiles(logFiles)
	}
	return tailFiles(logFiles)
}

func runLogsMysql(cmd *cobra.Command, args []string) error {
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

	db, err := getDbInfo(cfg)
	if err != nil {
		cli.PrintError("%v", err)
		return nil
	}

	composeGen := docker.NewComposeGenerator(p)
	composeFile := composeGen.ComposeFilePath()

	// Determine compose service name
	serviceName := fmt.Sprintf("%s%s", db.Type, strings.ReplaceAll(db.Version, ".", ""))

	return streamDockerLogs(composeFile, serviceName, db.Type)
}

func runLogsRedis(cmd *cobra.Command, args []string) error {
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

	if !cfg.Services.HasCacheService() {
		cli.PrintError("Neither Redis nor Valkey is configured in %s", config.ConfigFileName)
		return nil
	}

	composeGen := docker.NewComposeGenerator(p)
	composeFile := composeGen.ComposeFilePath()

	svcName := cfg.Services.GetCacheServiceName()
	displayName := cfg.Services.GetCacheServiceDisplayName()
	return streamDockerLogs(composeFile, svcName, displayName)
}

func runLogsVarnish(cmd *cobra.Command, args []string) error {
	return runVarnishLogs(cmd, args)
}

// streamDockerLogs streams logs from a Docker Compose service
func streamDockerLogs(composeFile, serviceName, displayName string) error {
	logsArgs := []string{"logs"}
	logsArgs = append(logsArgs, "--tail", fmt.Sprintf("%d", logsLinesFlag))
	logsArgs = append(logsArgs, "-f")
	logsArgs = append(logsArgs, serviceName)

	cli.PrintInfo("Streaming %s logs (Ctrl+C to stop)...", displayName)
	fmt.Println()

	logCmd := docker.BuildComposeCmd(composeFile, logsArgs...)
	logCmd.Stdin = os.Stdin
	logCmd.Stdout = os.Stdout
	logCmd.Stderr = os.Stderr

	return logCmd.Run()
}

// findProjectLogFiles finds log files in a directory matching a project name prefix
func findProjectLogFiles(dir, projectName string) []string {
	var files []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return files
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasPrefix(entry.Name(), projectName) && strings.HasSuffix(entry.Name(), ".log") {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}

	return files
}

// tailFiles tails one or more files using tail -f
func tailFiles(files []string) error {
	args := []string{"-f", "-n", fmt.Sprintf("%d", logsLinesFlag)}
	args = append(args, files...)

	cli.PrintInfo("Tailing %d log file(s) (Ctrl+C to stop)...", len(files))
	for _, f := range files {
		fmt.Printf("  %s\n", cli.Path(f))
	}
	fmt.Println()

	tailCmd := exec.Command("tail", args...)
	tailCmd.Stdin = os.Stdin
	tailCmd.Stdout = os.Stdout
	tailCmd.Stderr = os.Stderr

	return tailCmd.Run()
}

// multitailFiles opens files in multitail split-screen view
func multitailFiles(files []string) error {
	cli.PrintInfo("Opening %d log file(s) in multitail...", len(files))
	for _, f := range files {
		fmt.Printf("  %s\n", cli.Path(f))
	}
	fmt.Println("Press 'q' to quit, 'b' to scroll back")
	fmt.Println()

	args := []string{"-n", "200", "-m", "500"}
	// -s splits into columns; multitail requires the value to be >= 2
	if len(files) >= 2 {
		args = append(args, "-s", "2")
	}
	args = append(args, files...)

	multitailCmd := exec.Command("multitail", args...)
	multitailCmd.Stdin = os.Stdin
	multitailCmd.Stdout = os.Stdout
	multitailCmd.Stderr = os.Stderr

	return multitailCmd.Run()
}

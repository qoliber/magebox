package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/qoliber/magebox/internal/cli"
	"github.com/qoliber/magebox/internal/platform"
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View Magento logs with multitail",
	Long: `Opens system.log and exception.log in a split-screen view using multitail.

The logs are displayed in 2 columns:
  - Left:  system.log
  - Right: exception.log

Press 'q' to quit, 'b' to scroll back in history.`,
	RunE: runLogs,
}

func init() {
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
	multitailCmd := exec.Command("multitail",
		"-s", "2",
		systemLog,
		exceptionLog,
	)
	multitailCmd.Stdin = os.Stdin
	multitailCmd.Stdout = os.Stdout
	multitailCmd.Stderr = os.Stderr

	return multitailCmd.Run()
}

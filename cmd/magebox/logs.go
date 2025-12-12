package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/qoliber/magebox/internal/cli"
)

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

var (
	logsFollow bool
	logsLines  int
)

func init() {
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow log files for changes")
	logsCmd.Flags().IntVarP(&logsLines, "lines", "n", 20, "Number of lines to show initially")
	rootCmd.AddCommand(logsCmd)
}

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

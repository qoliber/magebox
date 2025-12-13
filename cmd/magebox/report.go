package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"

	"github.com/qoliber/magebox/internal/cli"
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Watch Magento error reports",
	Long: `Watches the var/report directory for new error reports.

Displays the latest report and automatically shows new reports as they are created.
Press Ctrl+C to stop watching.`,
	RunE: runReport,
}

func init() {
	rootCmd.AddCommand(reportCmd)
}

func runReport(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	reportDir := filepath.Join(cwd, "var", "report")

	// Check if report directory exists
	if _, err := os.Stat(reportDir); os.IsNotExist(err) {
		cli.PrintInfo("Report directory not found: %s", reportDir)
		cli.PrintInfo("No error reports have been generated yet.")
		return nil
	}

	// Show the latest report first
	latestReport, err := getLatestReport(reportDir)
	if err != nil {
		cli.PrintInfo("No reports found in %s", reportDir)
	} else {
		displayReport(latestReport)
	}

	// Set up file watcher using fsnotify (Go library)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	defer func() { _ = watcher.Close() }()

	if err := watcher.Add(reportDir); err != nil {
		return fmt.Errorf("failed to watch directory: %w", err)
	}

	fmt.Println()
	cli.PrintInfo("Watching for new reports... (Ctrl+C to stop)")
	fmt.Println()

	// Watch for new files
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if event.Op&fsnotify.Create == fsnotify.Create {
				// Small delay to ensure file is fully written
				time.Sleep(100 * time.Millisecond)
				displayReport(event.Name)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			cli.PrintWarning("Watcher error: %v", err)
		}
	}
}

func getLatestReport(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	var files []os.DirEntry
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry)
		}
	}

	if len(files) == 0 {
		return "", fmt.Errorf("no files found")
	}

	// Sort by modification time (newest first)
	sort.Slice(files, func(i, j int) bool {
		infoI, _ := files[i].Info()
		infoJ, _ := files[j].Info()
		return infoI.ModTime().After(infoJ.ModTime())
	})

	return filepath.Join(dir, files[0].Name()), nil
}

func displayReport(path string) {
	content, err := os.ReadFile(path)
	if err != nil {
		cli.PrintError("Failed to read report: %v", err)
		return
	}

	filename := filepath.Base(path)
	info, _ := os.Stat(path)

	// Clear screen and show report
	fmt.Print("\033[H\033[2J")

	cli.PrintTitle("Magento Error Report")
	fmt.Println()
	fmt.Printf("File: %s\n", cli.Highlight(filename))
	if info != nil {
		fmt.Printf("Time: %s\n", cli.Path(info.ModTime().Format("2006-01-02 15:04:05")))
	}
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println()
	fmt.Println(string(content))
	fmt.Println()
	fmt.Println(strings.Repeat("-", 60))
}

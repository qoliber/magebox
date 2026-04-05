// Copyright (c) qoliber

package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"qoliber/magebox/internal/cli"
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch project files and clear affected Magento caches",
	Long: `Runs mage-os/magento-cache-clean in the foreground to watch the current
project and clear only the affected Magento cache types on file changes.

Requires cache-clean.js to be installed globally via:
  composer global require mage-os/magento-cache-clean

Press Ctrl-C to stop the watcher.`,
	RunE: runWatch,
}

func init() {
	rootCmd.AddCommand(watchCmd)
}

func runWatch(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	if _, ok := loadProjectConfig(cwd); !ok {
		return nil
	}

	bin, err := findCacheCleanBinary()
	if err != nil {
		return err
	}
	if bin == "" {
		if !promptInstallCacheClean() {
			cli.PrintInfo("Skipped. Install manually with:")
			fmt.Printf("  %s\n", cli.Command("composer global require mage-os/magento-cache-clean"))
			return nil
		}
		if err := installCacheCleanGlobally(); err != nil {
			return fmt.Errorf("failed to install cache-clean: %w", err)
		}
		bin, err = findCacheCleanBinary()
		if err != nil {
			return err
		}
		if bin == "" {
			cli.PrintError("cache-clean.js was installed but could not be located on PATH")
			cli.PrintInfo("Ensure your Composer global bin dir is on your PATH, e.g.:")
			fmt.Printf("  %s\n", cli.Command(`export PATH="$(composer global config bin-dir --absolute --quiet):$PATH"`))
			return nil
		}
	}

	cli.PrintInfo("Watching %s", cli.Path(cwd))
	cli.PrintInfo("Using %s", cli.Path(bin))
	fmt.Println()

	watcher := exec.Command(bin, "--watch", "--directory", cwd)
	watcher.Stdin = os.Stdin
	watcher.Stdout = os.Stdout
	watcher.Stderr = os.Stderr

	// Ctrl-C at the terminal is delivered to the whole foreground process
	// group, so the child receives SIGINT directly and exits cleanly.
	if err := watcher.Run(); err != nil {
		// Suppress exit-status noise when the user interrupted.
		if exitErr, ok := err.(*exec.ExitError); ok && !exitErr.Success() {
			return nil
		}
		return err
	}
	return nil
}

// findCacheCleanBinary resolves the global cache-clean.js binary, preferring
// PATH and falling back to the Composer global bin directory.
func findCacheCleanBinary() (string, error) {
	if p, err := exec.LookPath("cache-clean.js"); err == nil {
		return p, nil
	}

	// Fallback: ask composer where its global bin dir is.
	if _, err := exec.LookPath("composer"); err != nil {
		// No composer available at all — treat as "not installed".
		return "", nil
	}

	out, err := exec.Command("composer", "global", "config", "bin-dir", "--absolute", "--quiet").Output()
	if err != nil {
		return "", nil
	}
	binDir := strings.TrimSpace(string(out))
	if binDir == "" {
		return "", nil
	}
	candidate := filepath.Join(binDir, "cache-clean.js")
	if _, err := os.Stat(candidate); err == nil {
		return candidate, nil
	}
	return "", nil
}

func promptInstallCacheClean() bool {
	cli.PrintWarning("cache-clean.js is not installed globally")
	fmt.Println()
	cli.PrintInfo("MageBox can install it for you by running:")
	fmt.Printf("  %s\n", cli.Command("composer global require mage-os/magento-cache-clean"))
	fmt.Println()
	fmt.Print("Install now? [Y/n] ")

	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "" || answer == "y" || answer == "yes"
}

func installCacheCleanGlobally() error {
	if _, err := exec.LookPath("composer"); err != nil {
		return fmt.Errorf("composer not found on PATH")
	}
	cli.PrintInfo("Installing mage-os/magento-cache-clean globally...")
	fmt.Println()

	install := exec.Command("composer", "global", "require", "mage-os/magento-cache-clean")
	install.Stdin = os.Stdin
	install.Stdout = os.Stdout
	install.Stderr = os.Stderr
	if err := install.Run(); err != nil {
		return err
	}
	fmt.Println()
	cli.PrintSuccess("cache-clean.js installed")
	return nil
}

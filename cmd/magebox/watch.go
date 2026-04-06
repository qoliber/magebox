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

When a Hyvä theme with a Tailwind setup is detected, a second pane is opened
running "npm run watch" for the theme's Tailwind build (requires multitail).

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

	p, err := getPlatform()
	if err != nil {
		return err
	}

	cfg, ok := loadProjectConfig(cwd)
	if !ok {
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

	// Generate cache-clean-config.json from current env.php
	if err := generateCacheCleanConfig(bin, cwd, p.PHPBinary(cfg.PHP)); err != nil {
		cli.PrintWarning("Could not generate cache-clean config: %v", err)
	}

	// Detect Hyvä Tailwind directory
	hyvaTailwindDir := findHyvaTailwindDir(cwd)

	if hyvaTailwindDir != "" {
		return runWatchWithHyva(bin, cwd, hyvaTailwindDir)
	}

	return runWatchCacheCleanOnly(bin, cwd)
}

// runWatchCacheCleanOnly runs cache-clean.js directly in the foreground.
func runWatchCacheCleanOnly(bin, cwd string) error {
	cli.PrintInfo("Watching %s", cli.Path(cwd))
	cli.PrintInfo("Using %s", cli.Path(bin))
	fmt.Println()

	watcher := exec.Command(bin, "--watch", "--directory", cwd)
	watcher.Stdin = os.Stdin
	watcher.Stdout = os.Stdout
	watcher.Stderr = os.Stderr

	if err := watcher.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return nil
		}
		return err
	}
	return nil
}

// runWatchWithHyva runs cache-clean.js and npm run watch side by side in a tmux session.
func runWatchWithHyva(bin, cwd, hyvaTailwindDir string) error {
	if _, err := exec.LookPath("tmux"); err != nil {
		cli.PrintError("tmux is not installed")
		cli.PrintInfo("tmux is required to run cache-clean and Hyvä Tailwind watcher side by side")
		fmt.Println("  brew install tmux  # macOS")
		fmt.Println("  sudo dnf install tmux  # Fedora")
		fmt.Println("  sudo apt install tmux  # Ubuntu/Debian")
		return nil
	}

	relDir, err := filepath.Rel(cwd, hyvaTailwindDir)
	if err != nil {
		relDir = hyvaTailwindDir
	}

	cli.PrintInfo("Watching %s", cli.Path(cwd))
	cli.PrintInfo("Cache clean: %s", cli.Path(bin))
	cli.PrintInfo("Hyvä Tailwind: %s", cli.Path(relDir))
	fmt.Println()

	sessionName := "magebox-watch"

	// Kill any existing session with the same name
	_ = exec.Command("tmux", "kill-session", "-t", sessionName).Run()

	cacheCleanCmd := fmt.Sprintf("%s --watch --directory %s", bin, cwd)
	npmWatchCmd := fmt.Sprintf("npm --prefix %s run watch", hyvaTailwindDir)

	// Create a new detached tmux session with npm watch in the left pane
	if err := exec.Command("tmux", "new-session", "-d", "-s", sessionName, npmWatchCmd).Run(); err != nil {
		return fmt.Errorf("failed to create tmux session: %w", err)
	}

	// Split horizontally and run cache-clean in the right pane
	if err := exec.Command("tmux", "split-window", "-h", "-t", sessionName, cacheCleanCmd).Run(); err != nil {
		_ = exec.Command("tmux", "kill-session", "-t", sessionName).Run()
		return fmt.Errorf("failed to split tmux pane: %w", err)
	}

	// Attach to the session — this blocks until the user detaches or both panes exit
	attach := exec.Command("tmux", "attach-session", "-t", sessionName)
	attach.Stdin = os.Stdin
	attach.Stdout = os.Stdout
	attach.Stderr = os.Stderr

	if err := attach.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return nil
		}
		return err
	}
	return nil
}

// findHyvaTailwindDir looks for a Hyvä theme Tailwind directory under
// app/design/frontend/*/*/web/tailwind/ that contains a package.json.
func findHyvaTailwindDir(cwd string) string {
	designDir := filepath.Join(cwd, "app", "design", "frontend")
	if _, err := os.Stat(designDir); os.IsNotExist(err) {
		return ""
	}

	// Walk: app/design/frontend/<Vendor>/<Theme>/web/tailwind/package.json
	vendors, err := os.ReadDir(designDir)
	if err != nil {
		return ""
	}
	for _, vendor := range vendors {
		if !vendor.IsDir() {
			continue
		}
		themes, err := os.ReadDir(filepath.Join(designDir, vendor.Name()))
		if err != nil {
			continue
		}
		for _, theme := range themes {
			if !theme.IsDir() {
				continue
			}
			tailwindDir := filepath.Join(designDir, vendor.Name(), theme.Name(), "web", "tailwind")
			packageJSON := filepath.Join(tailwindDir, "package.json")
			if _, err := os.Stat(packageJSON); err == nil {
				return tailwindDir
			}
		}
	}
	return ""
}

// generateCacheCleanConfig runs the generate-cache-clean-config.php script
// that ships with mage-os/magento-cache-clean to produce var/cache-clean-config.json
// from the current app/etc/env.php. This ensures the watcher uses up-to-date config.
func generateCacheCleanConfig(cacheCleanBin, projectDir, phpBin string) error {
	genScript := findGenerateConfigScript(cacheCleanBin)
	if genScript == "" {
		return fmt.Errorf("generate-cache-clean-config.php not found")
	}

	cli.PrintInfo("Generating cache-clean config...")
	gen := exec.Command(phpBin, genScript, projectDir)
	gen.Stdout = os.Stdout
	gen.Stderr = os.Stderr
	return gen.Run()
}

// findGenerateConfigScript locates generate-cache-clean-config.php relative to
// the cache-clean.js binary. The Composer bin wrapper is a shell script that
// delegates to ../mage-os/magento-cache-clean/bin/cache-clean.js, so we resolve
// the real package bin directory by following that path.
func findGenerateConfigScript(cacheCleanBin string) string {
	binDir := filepath.Dir(cacheCleanBin)

	// Direct: script next to the binary (real package bin dir)
	candidate := filepath.Join(binDir, "generate-cache-clean-config.php")
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}

	// Composer proxy: vendor/bin/ → ../mage-os/magento-cache-clean/bin/
	candidate = filepath.Join(binDir, "..", "mage-os", "magento-cache-clean", "bin", "generate-cache-clean-config.php")
	resolved, err := filepath.Abs(candidate)
	if err != nil {
		return ""
	}
	if _, err := os.Stat(resolved); err == nil {
		return resolved
	}

	return ""
}

// findCacheCleanBinary resolves the global cache-clean.js binary, preferring
// PATH and falling back to the Composer global bin directory.
func findCacheCleanBinary() (string, error) {
	if p, err := exec.LookPath("cache-clean.js"); err == nil {
		return p, nil
	}

	// Fallback: ask composer where its global bin dir is.
	if _, err := exec.LookPath("composer"); err != nil {
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

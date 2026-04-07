// Copyright (c) qoliber

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"qoliber/magebox/internal/cli"
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch project files and clear affected Magento caches",
	Long: `Runs mage-os/magento-cache-clean in the foreground to watch the current
project and clear only the affected Magento cache types on file changes.

When a theme with a Tailwind build setup is detected (any theme under
app/design/frontend/ containing a package.json with a "watch" script),
a second pane is opened running "npm run watch" for that theme (requires tmux).
If multiple themes are found, you will be prompted to choose one.

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

	// Detect theme directories with an npm watch script
	themeDirs := findThemeWatchDirs(cwd)

	if len(themeDirs) == 1 {
		return runWatchWithTheme(bin, cwd, themeDirs[0])
	}
	if len(themeDirs) > 1 {
		selected, err := selectThemeWatchDir(cwd, themeDirs)
		if err != nil {
			return err
		}
		if selected != "" {
			return runWatchWithTheme(bin, cwd, selected)
		}
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

// runWatchWithTheme runs cache-clean.js and npm run watch side by side in a tmux session.
func runWatchWithTheme(bin, cwd, themeDir string) error {
	if _, err := exec.LookPath("tmux"); err != nil {
		cli.PrintError("tmux is not installed")
		cli.PrintInfo("tmux is required to run cache-clean and the theme watcher side by side")
		fmt.Println("  brew install tmux  # macOS")
		fmt.Println("  sudo dnf install tmux  # Fedora")
		fmt.Println("  sudo apt install tmux  # Ubuntu/Debian")
		return nil
	}

	relDir, err := filepath.Rel(cwd, themeDir)
	if err != nil {
		relDir = themeDir
	}

	cli.PrintInfo("Watching %s", cli.Path(cwd))
	cli.PrintInfo("Cache clean: %s", cli.Path(bin))
	cli.PrintInfo("Theme watcher: %s", cli.Path(relDir))
	fmt.Println()

	sessionName := "magebox-watch"

	// Kill any existing session with the same name
	_ = exec.Command("tmux", "kill-session", "-t", sessionName).Run()

	cacheCleanCmd := fmt.Sprintf("%s --watch --directory %s", bin, cwd)
	// Wrap npm watch so errors pause with a visible message instead of closing the pane.
	npmWatchCmd := fmt.Sprintf("npm --prefix %s run watch || { echo ''; echo 'npm run watch failed — press Enter to close'; read; }", themeDir)

	// Create a new detached tmux session with npm watch in the left pane
	if err := exec.Command("tmux", "new-session", "-d", "-s", sessionName, "sh", "-c", npmWatchCmd).Run(); err != nil {
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

// selectThemeWatchDir prompts the user to select a theme directory when multiple are found.
func selectThemeWatchDir(cwd string, dirs []string) (string, error) {
	options := make([]huh.Option[string], 0, len(dirs)+1)
	for _, dir := range dirs {
		relDir, err := filepath.Rel(cwd, dir)
		if err != nil {
			relDir = dir
		}
		options = append(options, huh.NewOption(relDir, dir))
	}
	options = append(options, huh.NewOption("None — only run cache-clean", ""))

	var selected string
	err := huh.NewSelect[string]().
		Title("Multiple themes with watch scripts found").
		Options(options...).
		Value(&selected).
		Run()
	if err != nil {
		return "", err
	}
	return selected, nil
}

// packageJSON represents the parts of package.json we care about.
type packageJSON struct {
	Scripts map[string]string `json:"scripts"`
}

// hasWatchScript checks if a package.json file contains a "watch" script.
func hasWatchScript(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return false
	}
	_, ok := pkg.Scripts["watch"]
	return ok
}

// findThemeWatchDirs finds directories under app/design/frontend/ that contain
// a package.json with a "watch" script. It walks the full theme tree to find
// package.json files regardless of their exact location within the theme
// (e.g. web/tailwind/, web/css/, or directly in the theme root).
func findThemeWatchDirs(cwd string) []string {
	designDir := filepath.Join(cwd, "app", "design", "frontend")
	if _, err := os.Stat(designDir); os.IsNotExist(err) {
		return nil
	}

	var dirs []string

	vendors, err := os.ReadDir(designDir)
	if err != nil {
		return nil
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
			themeDir := filepath.Join(designDir, vendor.Name(), theme.Name())
			// Walk the theme directory to find any package.json with a watch script
			_ = filepath.WalkDir(themeDir, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				// Skip node_modules and vendor directories
				if d.IsDir() && (d.Name() == "node_modules" || d.Name() == "vendor") {
					return filepath.SkipDir
				}
				if d.Name() == "package.json" && hasWatchScript(path) {
					dirs = append(dirs, filepath.Dir(path))
				}
				return nil
			})
		}
	}
	return dirs
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

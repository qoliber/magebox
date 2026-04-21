// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"qoliber/magebox/internal/platform"
)

// BaseInstaller provides common functionality for all platform installers
type BaseInstaller struct {
	Platform *platform.Platform
}

// RunCommand executes a command with shell handling
func (b *BaseInstaller) RunCommand(cmdStr string) error {
	cmd := exec.Command("sh", "-c", cmdStr)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RunCommandSilent executes a command without output
func (b *BaseInstaller) RunCommandSilent(cmdStr string) error {
	cmd := exec.Command("sh", "-c", cmdStr)
	return cmd.Run()
}

// RunSudo executes a command with sudo
func (b *BaseInstaller) RunSudo(args ...string) error {
	cmd := exec.Command("sudo", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RunSudoSilent executes a sudo command without output
func (b *BaseInstaller) RunSudoSilent(args ...string) error {
	cmd := exec.Command("sudo", args...)
	return cmd.Run()
}

// FileExists checks if a file exists
func (b *BaseInstaller) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// WriteFile writes content to a file using sudo
func (b *BaseInstaller) WriteFile(path, content string) error {
	// Write to temp file
	tmpFile, err := os.CreateTemp("", "magebox-*")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		return err
	}
	tmpFile.Close()

	// Copy to destination with sudo (use RunSudo to allow password prompt)
	return b.RunSudo("cp", tmpPath, path)
}

// CommandExists checks if a command is available
func (b *BaseInstaller) CommandExists(name string) bool {
	return platform.CommandExists(name)
}

// filterPackages partitions pkgs into those the availability predicate
// accepts (kept) and those it rejects (dropped), preserving input order.
func filterPackages(pkgs []string, available func(string) bool) (kept, dropped []string) {
	for _, p := range pkgs {
		if available(p) {
			kept = append(kept, p)
		} else {
			dropped = append(dropped, p)
		}
	}
	return
}

// IsRealComposerInstalled checks if the real Composer binary (not our wrapper) is installed
func (b *BaseInstaller) IsRealComposerInstalled() bool {
	wrapperDir := filepath.Join(b.Platform.MageBoxDir(), "bin")

	// Check common locations
	locations := []string{
		"/usr/local/bin/composer",
		"/usr/local/bin/composer.phar",
		"/opt/homebrew/bin/composer",
	}

	homeDir, _ := os.UserHomeDir()
	if homeDir != "" {
		locations = append(locations,
			filepath.Join(homeDir, ".composer", "composer.phar"),
			filepath.Join(homeDir, "composer.phar"),
		)
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return true
		}
	}

	// Search PATH, skipping our wrapper directory
	pathEnv := os.Getenv("PATH")
	for _, dir := range strings.Split(pathEnv, string(os.PathListSeparator)) {
		if dir == wrapperDir {
			continue
		}
		composerPath := filepath.Join(dir, "composer")
		if info, err := os.Stat(composerPath); err == nil && info.Mode()&0111 != 0 {
			return true
		}
	}

	return false
}

// InstallComposer downloads and installs the Composer PHP dependency manager
func (b *BaseInstaller) InstallComposer() error {
	if b.IsRealComposerInstalled() {
		return nil
	}

	// Find a PHP binary to use for the installer
	phpBin := b.findPHPBinary()
	if phpBin == "" {
		return fmt.Errorf("PHP is required to install Composer but was not found")
	}

	// Download composer installer to a temp file
	tmpDir := os.TempDir()
	setupPath := filepath.Join(tmpDir, "composer-setup.php")
	defer os.Remove(setupPath)

	downloadCmd := exec.Command(phpBin, "-r",
		fmt.Sprintf(`copy('https://getcomposer.org/installer', '%s');`, setupPath))
	if out, err := downloadCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to download Composer installer: %w (%s)", err, string(out))
	}

	// Run the installer to /usr/local/bin
	installCmd := exec.Command("sudo", phpBin, setupPath,
		"--install-dir=/usr/local/bin", "--filename=composer")
	installCmd.Stdin = os.Stdin
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("failed to install Composer: %w", err)
	}

	return nil
}

// findPHPBinary finds a usable PHP binary on the system
func (b *BaseInstaller) findPHPBinary() string {
	// Try versioned binaries first (these are always the real ones)
	for _, ver := range PHPVersions {
		bin := fmt.Sprintf("php%s", ver)
		if path, err := exec.LookPath(bin); err == nil {
			return path
		}
	}

	// Fall back to plain "php"
	if path, err := exec.LookPath("php"); err == nil {
		return path
	}

	return ""
}

// ConfigureShellPath adds ~/.magebox/bin to the user's shell PATH
// This is called during bootstrap to ensure the PHP wrapper is in PATH
func (b *BaseInstaller) ConfigureShellPath() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	mageboxBin := homeDir + "/.magebox/bin"
	pathLine := `export PATH="$HOME/.magebox/bin:$PATH"`
	marker := ".magebox/bin"

	// Create ~/.magebox/bin if it doesn't exist
	if err := os.MkdirAll(mageboxBin, 0755); err != nil {
		return fmt.Errorf("failed to create %s: %w", mageboxBin, err)
	}

	// Determine which shell config files to update based on current shell
	shell := os.Getenv("SHELL")
	shellName := ""
	if shell != "" {
		parts := strings.Split(shell, "/")
		shellName = parts[len(parts)-1]
	}

	var rcFiles []string

	// Check for shell-specific config files
	switch shellName {
	case "zsh":
		// zsh: add to .zshenv (always sourced, even by IDE terminals)
		zshenv := homeDir + "/.zshenv"
		if !b.FileExists(zshenv) {
			if f, err := os.Create(zshenv); err == nil {
				f.Close()
			}
		}
		rcFiles = append(rcFiles, zshenv)
		// Also add to .zprofile/.profile for login shells and non-zsh invocations.
		zprofile := homeDir + "/.zprofile"
		if b.FileExists(zprofile) {
			rcFiles = append(rcFiles, zprofile)
		}
		profile := homeDir + "/.profile"
		if b.FileExists(profile) {
			rcFiles = append(rcFiles, profile)
		}
	case "bash":
		// bash: prefer .bashrc
		bashrc := homeDir + "/.bashrc"
		if b.FileExists(bashrc) {
			rcFiles = append(rcFiles, bashrc)
		}
		// Also try .bash_profile for login shells (common on macOS)
		bashProfile := homeDir + "/.bash_profile"
		if b.FileExists(bashProfile) {
			rcFiles = append(rcFiles, bashProfile)
		}
	case "fish":
		// fish: use config.fish
		fishConfig := homeDir + "/.config/fish/config.fish"
		if b.FileExists(fishConfig) {
			// fish uses different syntax: set -gx PATH $HOME/.magebox/bin $PATH
			pathLine = `set -gx PATH $HOME/.magebox/bin $PATH`
			rcFiles = append(rcFiles, fishConfig)
		}
	default:
		// Unknown shell: try common files
		bashrc := homeDir + "/.bashrc"
		if b.FileExists(bashrc) {
			rcFiles = append(rcFiles, bashrc)
		}
		profile := homeDir + "/.profile"
		if b.FileExists(profile) {
			rcFiles = append(rcFiles, profile)
		}
	}

	// If no files found, return error
	if len(rcFiles) == 0 {
		return fmt.Errorf("no shell config file found")
	}

	configured := false
	for _, rcFile := range rcFiles {
		// Check if already configured
		content, err := os.ReadFile(rcFile)
		if err != nil {
			continue
		}

		if strings.Contains(string(content), marker) {
			configured = true
			continue // Already configured in this file
		}

		// Append PATH configuration
		f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			continue
		}

		entry := fmt.Sprintf("\n# MageBox PHP version wrapper (auto-selects PHP based on .magebox.yaml)\n%s\n", pathLine)
		_, err = f.WriteString(entry)
		f.Close()

		if err != nil {
			return fmt.Errorf("failed to update %s: %w", rcFile, err)
		}
		configured = true
	}

	if !configured {
		return fmt.Errorf("failed to configure any shell file")
	}

	return nil
}

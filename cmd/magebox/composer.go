// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/qoliber/magebox/internal/cli"
)

var composerCmd = &cobra.Command{
	Use:                "composer [args...]",
	Short:              "Run Composer with project's PHP version",
	Long:               "Runs Composer using the PHP version specified in .magebox.yaml",
	DisableFlagParsing: true,
	RunE:               runComposer,
}

func init() {
	rootCmd.AddCommand(composerCmd)
}

func runComposer(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	p, err := getPlatform()
	if err != nil {
		return err
	}

	// Load project config to get PHP version
	cfg, ok := loadProjectConfig(cwd)
	if !ok {
		return nil
	}

	// Get the PHP binary path for the project's PHP version
	phpBinary := p.PHPBinary(cfg.PHP)

	// Check if PHP binary exists
	if _, err := os.Stat(phpBinary); os.IsNotExist(err) {
		cli.PrintError("PHP %s is not installed at %s", cfg.PHP, phpBinary)
		return nil
	}

	// Find composer
	composerPath, err := findComposer()
	if err != nil {
		cli.PrintError("Composer not found: %v", err)
		fmt.Println()
		fmt.Println("Install composer: https://getcomposer.org/download/")
		return nil
	}

	// Build command: php composer.phar [args...]
	cmdArgs := append([]string{composerPath}, args...)

	// Execute composer with the project's PHP version
	execCmd := exec.Command(phpBinary, cmdArgs...)
	execCmd.Dir = cwd
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	// Set memory limit for composer
	execCmd.Env = append(os.Environ(), "COMPOSER_MEMORY_LIMIT=-1")

	err = execCmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				os.Exit(status.ExitStatus())
			}
		}
		return err
	}

	return nil
}

// findComposer locates the composer executable
func findComposer() (string, error) {
	// Check common locations
	locations := []string{
		"/usr/local/bin/composer",
		"/usr/bin/composer",
		"/opt/homebrew/bin/composer",
	}

	// Check home directory
	if home, err := os.UserHomeDir(); err == nil {
		locations = append(locations,
			home+"/.composer/composer.phar",
			home+"/.local/bin/composer",
			home+"/bin/composer",
		)
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc, nil
		}
	}

	// Try to find in PATH
	path, err := exec.LookPath("composer")
	if err == nil {
		return path, nil
	}

	return "", fmt.Errorf("composer not found in common locations or PATH")
}

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Open project shell",
	Long:  "Opens a shell with the correct PHP version in PATH",
	RunE:  runShell,
}

func init() {
	rootCmd.AddCommand(shellCmd)
}

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

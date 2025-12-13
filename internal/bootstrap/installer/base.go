// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package installer

import (
	"os"
	"os/exec"

	"github.com/qoliber/magebox/internal/platform"
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

	// Copy to destination with sudo
	return b.RunSudoSilent("cp", tmpPath, path)
}

// CommandExists checks if a command is available
func (b *BaseInstaller) CommandExists(name string) bool {
	return platform.CommandExists(name)
}

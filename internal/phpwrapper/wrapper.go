// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package phpwrapper

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/qoliber/magebox/internal/platform"
)

//go:embed templates/php.sh
var phpWrapperScript string

//go:embed templates/composer.sh
var composerWrapperScript string

const (
	WrapperScriptName         = "php"
	ComposerWrapperScriptName = "composer"
)

// Manager handles PHP wrapper installation and management
type Manager struct {
	platform *platform.Platform
}

// NewManager creates a new PHP wrapper manager
func NewManager(p *platform.Platform) *Manager {
	return &Manager{platform: p}
}

// GetWrapperPath returns the path where the wrapper should be installed
func (m *Manager) GetWrapperPath() string {
	return filepath.Join(m.platform.MageBoxDir(), "bin", WrapperScriptName)
}

// GenerateWrapper returns the PHP wrapper script content
func (m *Manager) GenerateWrapper() string {
	return phpWrapperScript
}

// Install creates and installs the PHP wrapper script
func (m *Manager) Install() error {
	binDir := filepath.Join(m.platform.MageBoxDir(), "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	wrapperPath := m.GetWrapperPath()
	content := m.GenerateWrapper()

	if err := os.WriteFile(wrapperPath, []byte(content), 0755); err != nil {
		return fmt.Errorf("failed to write wrapper script: %w", err)
	}

	return nil
}

// IsInstalled checks if the wrapper is already installed
func (m *Manager) IsInstalled() bool {
	wrapperPath := m.GetWrapperPath()
	info, err := os.Stat(wrapperPath)
	if err != nil {
		return false
	}
	return info.Mode().Perm()&0111 != 0 // Check if executable
}

// Uninstall removes the PHP wrapper script
func (m *Manager) Uninstall() error {
	wrapperPath := m.GetWrapperPath()
	if err := os.Remove(wrapperPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove wrapper script: %w", err)
	}
	return nil
}

// GetInstructions returns instructions for adding wrapper to PATH
func (m *Manager) GetInstructions() string {
	binDir := filepath.Join(m.platform.MageBoxDir(), "bin")

	shell := os.Getenv("SHELL")
	var rcFile string

	if filepath.Base(shell) == "zsh" {
		rcFile = "~/.zshrc"
	} else {
		rcFile = "~/.bashrc"
	}

	return fmt.Sprintf(`To use the PHP version wrapper, add this to your %s:

    export PATH="%s:$PATH"

Then reload your shell:

    source %s

After this, the 'php' command will automatically use the version specified in .magebox.yaml!`, rcFile, binDir, rcFile)
}

// GetComposerWrapperPath returns the path where the composer wrapper should be installed
func (m *Manager) GetComposerWrapperPath() string {
	return filepath.Join(m.platform.MageBoxDir(), "bin", ComposerWrapperScriptName)
}

// GenerateComposerWrapper returns the Composer wrapper script content
func (m *Manager) GenerateComposerWrapper() string {
	return composerWrapperScript
}

// InstallComposer creates and installs the Composer wrapper script
func (m *Manager) InstallComposer() error {
	binDir := filepath.Join(m.platform.MageBoxDir(), "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	wrapperPath := m.GetComposerWrapperPath()
	content := m.GenerateComposerWrapper()

	if err := os.WriteFile(wrapperPath, []byte(content), 0755); err != nil {
		return fmt.Errorf("failed to write composer wrapper script: %w", err)
	}

	return nil
}

// IsComposerInstalled checks if the composer wrapper is already installed
func (m *Manager) IsComposerInstalled() bool {
	wrapperPath := m.GetComposerWrapperPath()
	info, err := os.Stat(wrapperPath)
	if err != nil {
		return false
	}
	return info.Mode().Perm()&0111 != 0 // Check if executable
}

// UninstallComposer removes the Composer wrapper script
func (m *Manager) UninstallComposer() error {
	wrapperPath := m.GetComposerWrapperPath()
	if err := os.Remove(wrapperPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove composer wrapper script: %w", err)
	}
	return nil
}

// InstallAll installs both PHP and Composer wrappers
func (m *Manager) InstallAll() error {
	if err := m.Install(); err != nil {
		return err
	}
	return m.InstallComposer()
}

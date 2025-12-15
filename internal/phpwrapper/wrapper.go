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

//go:embed templates/blackfire.sh
var blackfireWrapperScript string

const (
	WrapperScriptName          = "php"
	ComposerWrapperScriptName  = "composer"
	BlackfireWrapperScriptName = "blackfire"
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

// GetBlackfireWrapperPath returns the path where the blackfire wrapper should be installed
func (m *Manager) GetBlackfireWrapperPath() string {
	return filepath.Join(m.platform.MageBoxDir(), "bin", BlackfireWrapperScriptName)
}

// GenerateBlackfireWrapper returns the Blackfire wrapper script content
func (m *Manager) GenerateBlackfireWrapper() string {
	return blackfireWrapperScript
}

// InstallBlackfire creates and installs the Blackfire wrapper script
func (m *Manager) InstallBlackfire() error {
	binDir := filepath.Join(m.platform.MageBoxDir(), "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	wrapperPath := m.GetBlackfireWrapperPath()
	content := m.GenerateBlackfireWrapper()

	if err := os.WriteFile(wrapperPath, []byte(content), 0755); err != nil {
		return fmt.Errorf("failed to write blackfire wrapper script: %w", err)
	}

	return nil
}

// IsBlackfireInstalled checks if the blackfire wrapper is already installed
func (m *Manager) IsBlackfireInstalled() bool {
	wrapperPath := m.GetBlackfireWrapperPath()
	info, err := os.Stat(wrapperPath)
	if err != nil {
		return false
	}
	return info.Mode().Perm()&0111 != 0 // Check if executable
}

// UninstallBlackfire removes the Blackfire wrapper script
func (m *Manager) UninstallBlackfire() error {
	wrapperPath := m.GetBlackfireWrapperPath()
	if err := os.Remove(wrapperPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove blackfire wrapper script: %w", err)
	}
	return nil
}

// InstallAll installs PHP, Composer, and Blackfire wrappers
func (m *Manager) InstallAll() error {
	if err := m.Install(); err != nil {
		return err
	}
	if err := m.InstallComposer(); err != nil {
		return err
	}
	return m.InstallBlackfire()
}

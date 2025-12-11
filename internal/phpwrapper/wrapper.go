/**
 * Created by Qoliber
 *
 * @category    Qoliber
 * @package     Qoliber_MageBox
 * @author      Jakub Winkler <jwinkler@qoliber.com>
 */

package phpwrapper

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/qoliber/magebox/internal/platform"
)

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

// GenerateWrapper creates the PHP wrapper script content
func (m *Manager) GenerateWrapper() string {
	return `#!/bin/bash
# MageBox PHP version wrapper
# Automatically uses the correct PHP version based on .magebox.yaml

find_config_file() {
    local dir="$PWD"
    while [[ "$dir" != "/" ]]; do
        if [[ -f "$dir/.magebox.yaml" ]]; then
            echo "$dir/.magebox.yaml"
            return 0
        elif [[ -f "$dir/.magebox.local.yaml" ]]; then
            echo "$dir/.magebox.local.yaml"
            return 0
        elif [[ -f "$dir/.magebox" ]]; then
            echo "$dir/.magebox"
            return 0
        elif [[ -f "$dir/.magebox.local" ]]; then
            echo "$dir/.magebox.local"
            return 0
        fi
        dir="$(dirname "$dir")"
    done
    return 1
}

get_php_version_from_config() {
    local config_file="$1"
    # Extract PHP version from YAML using grep
    php_version=$(grep "^php:" "$config_file" | head -n1 | sed 's/php:[[:space:]]*["'\'']\{0,1\}\([0-9.]*\)["'\'']\{0,1\}/\1/' | tr -d ' ')
    echo "$php_version"
}

find_php_binary() {
    local version="$1"
    local php_bin=""

    # macOS: Use Cellar path directly (more reliable than opt symlinks)
    # Apple Silicon
    php_bin=$(ls /opt/homebrew/Cellar/php@$version/*/bin/php 2>/dev/null | head -n1)
    if [[ -x "$php_bin" ]]; then
        echo "$php_bin"
        return 0
    fi

    # Intel Mac
    php_bin=$(ls /usr/local/Cellar/php@$version/*/bin/php 2>/dev/null | head -n1)
    if [[ -x "$php_bin" ]]; then
        echo "$php_bin"
        return 0
    fi

    # Linux: Try common paths
    if [[ -x "/usr/bin/php$version" ]]; then
        echo "/usr/bin/php$version"
        return 0
    fi

    return 1
}

# Try to find config file
config_file=$(find_config_file)

if [[ -n "$config_file" ]]; then
    # Get PHP version from config
    php_version=$(get_php_version_from_config "$config_file")

    if [[ -n "$php_version" ]]; then
        php_bin=$(find_php_binary "$php_version")
        if [[ -n "$php_bin" ]]; then
            exec "$php_bin" "$@"
        else
            echo "Error: PHP $php_version not found. Install with: brew install php@$php_version" >&2
            exit 1
        fi
    fi
fi

# Fallback to system PHP (no config file found)
if command -v /opt/homebrew/bin/php &> /dev/null; then
    exec /opt/homebrew/bin/php "$@"
elif command -v /usr/local/bin/php &> /dev/null; then
    exec /usr/local/bin/php "$@"
elif command -v /usr/bin/php &> /dev/null; then
    exec /usr/bin/php "$@"
else
    echo "Error: No PHP installation found" >&2
    exit 1
fi
`
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

// GenerateComposerWrapper creates the Composer wrapper script content
func (m *Manager) GenerateComposerWrapper() string {
	return `#!/bin/bash
# MageBox Composer wrapper
# Automatically uses the correct PHP version based on .magebox.yaml

find_config_file() {
    local dir="$PWD"
    while [[ "$dir" != "/" ]]; do
        if [[ -f "$dir/.magebox.yaml" ]]; then
            echo "$dir/.magebox.yaml"
            return 0
        elif [[ -f "$dir/.magebox.local.yaml" ]]; then
            echo "$dir/.magebox.local.yaml"
            return 0
        elif [[ -f "$dir/.magebox" ]]; then
            echo "$dir/.magebox"
            return 0
        elif [[ -f "$dir/.magebox.local" ]]; then
            echo "$dir/.magebox.local"
            return 0
        fi
        dir="$(dirname "$dir")"
    done
    return 1
}

get_php_version_from_config() {
    local config_file="$1"
    # Extract PHP version from YAML using grep
    php_version=$(grep "^php:" "$config_file" | head -n1 | sed 's/php:[[:space:]]*["'\'']\{0,1\}\([0-9.]*\)["'\'']\{0,1\}/\1/' | tr -d ' ')
    echo "$php_version"
}

find_php_binary() {
    local version="$1"
    local php_bin=""

    # macOS: Use Cellar path directly (more reliable than opt symlinks)
    # Apple Silicon
    php_bin=$(ls /opt/homebrew/Cellar/php@$version/*/bin/php 2>/dev/null | head -n1)
    if [[ -x "$php_bin" ]]; then
        echo "$php_bin"
        return 0
    fi

    # Intel Mac
    php_bin=$(ls /usr/local/Cellar/php@$version/*/bin/php 2>/dev/null | head -n1)
    if [[ -x "$php_bin" ]]; then
        echo "$php_bin"
        return 0
    fi

    # Linux: Try common paths
    if [[ -x "/usr/bin/php$version" ]]; then
        echo "/usr/bin/php$version"
        return 0
    fi

    return 1
}

find_real_composer() {
    # Find composer, excluding our wrapper
    local self_dir="$(dirname "$(readlink -f "$0" 2>/dev/null || echo "$0")")"
    local IFS=':'
    for dir in $PATH; do
        if [[ "$dir" != "$self_dir" && -x "$dir/composer" ]]; then
            # Skip if it's a bash script (another wrapper)
            if head -1 "$dir/composer" 2>/dev/null | grep -q "^#!/bin/bash"; then
                continue
            fi
            echo "$dir/composer"
            return 0
        fi
    done
    # Fallback to common locations
    if [[ -x "/opt/homebrew/bin/composer" ]]; then
        echo "/opt/homebrew/bin/composer"
        return 0
    elif [[ -x "/usr/local/bin/composer" ]]; then
        echo "/usr/local/bin/composer"
        return 0
    fi
    return 1
}

# Find the real composer binary
REAL_COMPOSER=$(find_real_composer)
if [[ -z "$REAL_COMPOSER" ]]; then
    echo "Error: Composer not found in PATH" >&2
    exit 1
fi

# Try to find config file
config_file=$(find_config_file)

if [[ -n "$config_file" ]]; then
    # Get PHP version from config
    php_version=$(get_php_version_from_config "$config_file")

    if [[ -n "$php_version" ]]; then
        php_bin=$(find_php_binary "$php_version")
        if [[ -n "$php_bin" ]]; then
            exec "$php_bin" "$REAL_COMPOSER" "$@"
        else
            echo "Error: PHP $php_version not found. Install with: brew install php@$php_version" >&2
            exit 1
        fi
    fi
fi

# Fallback - run composer with system PHP (no config file found)
if command -v /opt/homebrew/bin/php &> /dev/null; then
    exec /opt/homebrew/bin/php "$REAL_COMPOSER" "$@"
elif command -v /usr/local/bin/php &> /dev/null; then
    exec /usr/local/bin/php "$REAL_COMPOSER" "$@"
elif command -v /usr/bin/php &> /dev/null; then
    exec /usr/bin/php "$REAL_COMPOSER" "$@"
else
    # Last resort - let composer use its shebang
    exec "$REAL_COMPOSER" "$@"
fi
`
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

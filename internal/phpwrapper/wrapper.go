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
	WrapperScriptName = "php"
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

# Try to find config file
config_file=$(find_config_file)

if [[ -n "$config_file" ]]; then
    # Get PHP version from config
    php_version=$(get_php_version_from_config "$config_file")

    if [[ -n "$php_version" ]]; then
        # Try Homebrew path first (macOS)
        if [[ -x "/opt/homebrew/opt/php@$php_version/bin/php" ]]; then
            exec "/opt/homebrew/opt/php@$php_version/bin/php" "$@"
        elif [[ -x "/usr/local/opt/php@$php_version/bin/php" ]]; then
            exec "/usr/local/opt/php@$php_version/bin/php" "$@"
        # Try system paths (Linux)
        elif [[ -x "/usr/bin/php$php_version" ]]; then
            exec "/usr/bin/php$php_version" "$@"
        elif [[ -x "/usr/bin/php" ]]; then
            # Verify version matches
            system_version=$(/usr/bin/php -r "echo PHP_MAJOR_VERSION.'.'.PHP_MINOR_VERSION;")
            if [[ "$system_version" == "$php_version" ]]; then
                exec "/usr/bin/php" "$@"
            fi
        fi
    fi
fi

# Fallback to system PHP
if command -v /usr/bin/php &> /dev/null; then
    exec /usr/bin/php "$@"
elif command -v /opt/homebrew/bin/php &> /dev/null; then
    exec /opt/homebrew/bin/php "$@"
elif command -v /usr/local/bin/php &> /dev/null; then
    exec /usr/local/bin/php "$@"
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

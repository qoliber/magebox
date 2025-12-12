#!/bin/bash
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
    local version_no_dot="${version//./}"

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

    # Linux Debian/Ubuntu: /usr/bin/php8.2 (with dot)
    if [[ -x "/usr/bin/php$version" ]]; then
        echo "/usr/bin/php$version"
        return 0
    fi

    # Linux Fedora/RHEL Remi: /usr/bin/php82 (no dot)
    if [[ -x "/usr/bin/php$version_no_dot" ]]; then
        echo "/usr/bin/php$version_no_dot"
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

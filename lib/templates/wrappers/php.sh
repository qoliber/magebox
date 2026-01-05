#!/bin/bash
# MageBox PHP version wrapper
# Automatically uses the correct PHP version based on .magebox.yaml

find_project_dir() {
    local dir="$PWD"
    while [[ "$dir" != "/" ]]; do
        if [[ -f "$dir/.magebox.yaml" ]] || [[ -f "$dir/.magebox.local.yaml" ]] || \
           [[ -f "$dir/.magebox" ]] || [[ -f "$dir/.magebox.local" ]]; then
            echo "$dir"
            return 0
        fi
        dir="$(dirname "$dir")"
    done
    return 1
}

get_php_version_from_file() {
    local config_file="$1"
    if [[ -f "$config_file" ]]; then
        grep "^php:" "$config_file" | head -n1 | sed 's/php:[[:space:]]*["'\'']\{0,1\}\([0-9.]*\)["'\'']\{0,1\}/\1/' | tr -d ' '
    fi
}

get_php_version() {
    local project_dir="$1"
    local version=""

    # Check local override first (highest priority)
    version=$(get_php_version_from_file "$project_dir/.magebox.local.yaml")
    if [[ -n "$version" ]]; then
        echo "$version"
        return 0
    fi

    version=$(get_php_version_from_file "$project_dir/.magebox.local")
    if [[ -n "$version" ]]; then
        echo "$version"
        return 0
    fi

    # Fall back to main config
    version=$(get_php_version_from_file "$project_dir/.magebox.yaml")
    if [[ -n "$version" ]]; then
        echo "$version"
        return 0
    fi

    version=$(get_php_version_from_file "$project_dir/.magebox")
    if [[ -n "$version" ]]; then
        echo "$version"
        return 0
    fi

    return 1
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

    # Linux Fedora/RHEL: /usr/bin/php82 (without dot, symlink to Remi)
    if [[ -x "/usr/bin/php$version_no_dot" ]]; then
        echo "/usr/bin/php$version_no_dot"
        return 0
    fi

    # Linux Fedora/RHEL Remi direct: /opt/remi/php82/root/usr/bin/php
    if [[ -x "/opt/remi/php$version_no_dot/root/usr/bin/php" ]]; then
        echo "/opt/remi/php$version_no_dot/root/usr/bin/php"
        return 0
    fi

    return 1
}

# Try to find project directory
project_dir=$(find_project_dir)

if [[ -n "$project_dir" ]]; then
    # Get PHP version (local override takes priority over main config)
    php_version=$(get_php_version "$project_dir")

    if [[ -n "$php_version" ]]; then
        php_bin=$(find_php_binary "$php_version")
        if [[ -n "$php_bin" ]]; then
            # Set Magento-friendly defaults for CLI (unlimited memory for compile/deploy)
            exec "$php_bin" -d memory_limit=-1 "$@"
        else
            echo "Error: PHP $php_version not found. Install with: brew install php@$php_version" >&2
            exit 1
        fi
    fi
fi

# Fallback to system PHP (no config file found)
if command -v /opt/homebrew/bin/php &> /dev/null; then
    exec /opt/homebrew/bin/php -d memory_limit=-1 "$@"
elif command -v /usr/local/bin/php &> /dev/null; then
    exec /usr/local/bin/php -d memory_limit=-1 "$@"
elif command -v /usr/bin/php &> /dev/null; then
    exec /usr/bin/php -d memory_limit=-1 "$@"
else
    echo "Error: No PHP installation found" >&2
    exit 1
fi

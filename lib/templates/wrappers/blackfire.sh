#!/bin/bash
# MageBox Blackfire wrapper
# Automatically uses the correct PHP version for 'blackfire run' commands

find_config_file() {
    local dir="$PWD"
    while [[ "$dir" != "/" ]]; do
        if [[ -f "$dir/.magebox.yaml" ]]; then
            echo "$dir/.magebox.yaml"
            return 0
        elif [[ -f "$dir/.magebox.local.yaml" ]]; then
            echo "$dir/.magebox.local.yaml"
            return 0
        fi
        dir="$(dirname "$dir")"
    done
    return 1
}

get_php_version_from_config() {
    local config_file="$1"
    php_version=$(grep "^php:" "$config_file" | head -n1 | sed 's/php:[[:space:]]*["'\'']\{0,1\}\([0-9.]*\)["'\'']\{0,1\}/\1/' | tr -d ' ')
    echo "$php_version"
}

find_php_binary() {
    local version="$1"
    local version_no_dot="${version//./}"

    # macOS Apple Silicon
    php_bin=$(ls /opt/homebrew/Cellar/php@$version/*/bin/php 2>/dev/null | head -n1)
    [[ -x "$php_bin" ]] && echo "$php_bin" && return 0

    # macOS Intel
    php_bin=$(ls /usr/local/Cellar/php@$version/*/bin/php 2>/dev/null | head -n1)
    [[ -x "$php_bin" ]] && echo "$php_bin" && return 0

    # Linux Debian/Ubuntu
    [[ -x "/usr/bin/php$version" ]] && echo "/usr/bin/php$version" && return 0

    # Linux Fedora/RHEL Remi
    [[ -x "/usr/bin/php$version_no_dot" ]] && echo "/usr/bin/php$version_no_dot" && return 0

    return 1
}

# Find real blackfire binary
REAL_BLACKFIRE=""
for path in /usr/bin/blackfire /usr/local/bin/blackfire /opt/homebrew/bin/blackfire; do
    if [[ -x "$path" ]]; then
        REAL_BLACKFIRE="$path"
        break
    fi
done

if [[ -z "$REAL_BLACKFIRE" ]]; then
    echo "Error: blackfire not found" >&2
    exit 1
fi

# Check if this is a 'run' command with 'php'
if [[ "$1" == "run" ]] || [[ "$1" == "--ignore-exit-status" && "$2" == "run" ]]; then
    # Find config and get PHP version
    config_file=$(find_config_file)
    if [[ -n "$config_file" ]]; then
        php_version=$(get_php_version_from_config "$config_file")
        if [[ -n "$php_version" ]]; then
            php_bin=$(find_php_binary "$php_version")
            if [[ -n "$php_bin" ]]; then
                # Replace 'php' with project PHP in arguments
                args=()
                for arg in "$@"; do
                    if [[ "$arg" == "php" ]]; then
                        args+=("$php_bin" "-d" "memory_limit=-1")
                    else
                        args+=("$arg")
                    fi
                done
                exec "$REAL_BLACKFIRE" "${args[@]}"
            fi
        fi
    fi
fi

# Pass through to real blackfire
exec "$REAL_BLACKFIRE" "$@"

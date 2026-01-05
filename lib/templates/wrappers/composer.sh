#!/bin/bash
# MageBox Composer wrapper
# Uses the MageBox PHP wrapper to ensure correct PHP version from .magebox.yaml

SCRIPT_DIR="$(dirname "$(readlink -f "$0" 2>/dev/null || echo "$0")")"
PHP_WRAPPER="$SCRIPT_DIR/php"

# Find the real composer binary (not our wrapper)
find_real_composer() {
    local IFS=':'
    for dir in $PATH; do
        if [[ "$dir" != "$SCRIPT_DIR" && -x "$dir/composer" ]]; then
            # Skip bash script wrappers
            if head -1 "$dir/composer" 2>/dev/null | grep -q "^#!/bin/bash"; then
                continue
            fi
            echo "$dir/composer"
            return 0
        fi
    done
    # Fallback to common locations
    for loc in /opt/homebrew/bin/composer /usr/local/bin/composer /home/*/.composer/vendor/bin/composer; do
        if [[ -x "$loc" ]]; then
            echo "$loc"
            return 0
        fi
    done
    return 1
}

REAL_COMPOSER=$(find_real_composer)
if [[ -z "$REAL_COMPOSER" ]]; then
    echo "Error: Composer not found in PATH" >&2
    exit 1
fi

# Use the PHP wrapper which handles version detection from .magebox.yaml
exec "$PHP_WRAPPER" "$REAL_COMPOSER" "$@"

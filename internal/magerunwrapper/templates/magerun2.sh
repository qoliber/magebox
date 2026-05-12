#!/bin/bash
# MageBox magerun2 wrapper
# Automatically downloads and uses the correct n98-magerun2 version.

SCRIPT_DIR="$(dirname "$(readlink -f "$0" 2>/dev/null || echo "$0")")"
PHP_WRAPPER="$SCRIPT_DIR/php"

find_project_dir() {
    local dir="$PWD"
    while [[ "$dir" != "/" ]]; do
        if [[ -f "$dir/.magebox.yaml" ]] || [[ -f "$dir/.magebox.local.yaml" ]]; then
            echo "$dir"
            return 0
        fi
        dir="$(dirname "$dir")"
    done
    return 1
}

project_dir=$(find_project_dir)

resolve_args=()
if [[ -n "$project_dir" ]]; then
    resolve_args+=("--project-dir=$project_dir")
fi

if ! phar_path=$(magebox magerun-resolve "${resolve_args[@]}"); then
    exit 1
fi

if [[ -z "$phar_path" ]]; then
    echo "Error: Could not resolve n98-magerun2. Run 'magebox check' for details." >&2
    exit 1
fi

exec "$PHP_WRAPPER" "$phar_path" "$@"

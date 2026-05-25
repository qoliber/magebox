#!/usr/bin/env bash
#
# Run magebox start for Valet projects that have .magebox.yaml (nginx vhost + SSL).
# Part of MageBox — https://github.com/qoliber/magebox
#
# This file is sourced by valet-to-magebox.sh.
# It must not be executed directly.

# Reload nginx so newly-created vhosts take effect.
valet_tm_reload_nginx() {
    if ! command -v nginx >/dev/null 2>&1; then
        echo "Tip: run 'magebox global stop && magebox global start' to reload nginx."
        return 0
    fi

    if ! nginx -t >/dev/null 2>&1; then
        echo "Warning: nginx config test failed — skipping reload." >&2
        return 1
    fi

    if nginx -s reload 2>/dev/null; then
        echo "Reloaded nginx."
        return 0
    fi

    echo "Tip: run 'magebox global stop && magebox global start' if HTTPS still shows the wrong certificate."
}

# Start all discovered projects that have a .magebox.yaml.
valet_tm_start_all_projects() {
    if [[ "${VALET_TM_DRY_RUN:-0}" == "1" ]]; then
        echo "[dry-run] Would run 'magebox start' in each project with .magebox.yaml"
    else
        echo "Running magebox start for each project with .magebox.yaml (nginx vhost + SSL cert)..."
    fi
    echo ""

    if [[ "${VALET_TM_DRY_RUN:-0}" != "1" ]] && ! command -v magebox >/dev/null 2>&1; then
        echo "Error: magebox not found in PATH. Install MageBox and run magebox bootstrap." >&2
        return 1
    fi

    local roots root
    roots="$(valet_tm_discover_project_roots)"
    if [[ -z "${roots}" ]]; then
        echo "No Valet project roots found."
        return 0
    fi

    local started=0 skipped=0 failed=0
    while IFS= read -r root; do
        [[ -n "${root}" ]] || continue
        if [[ ! -f "${root}/.magebox.yaml" ]]; then
            skipped=$((skipped + 1))
            continue
        fi

        echo "${root}"
        if [[ "${VALET_TM_DRY_RUN:-0}" == "1" ]]; then
            echo "  [dry-run] would run: magebox start"
            started=$((started + 1))
            echo ""
            continue
        fi

        if (cd "${root}" && magebox start 2>&1); then
            echo "  started"
            started=$((started + 1))
        else
            echo "  failed (see output above)" >&2
            failed=$((failed + 1))
        fi
        echo ""
    done <<< "${roots}"

    if [[ "${VALET_TM_DRY_RUN:-0}" == "1" ]]; then
        echo "Done: ${started} project(s) would be started, ${skipped} skipped (no .magebox.yaml)."
        return 0
    fi

    echo "Done: ${started} started, ${failed} failed, ${skipped} skipped (no .magebox.yaml)."
    if [[ "${started}" -gt 0 ]]; then
        echo ""
        valet_tm_reload_nginx
    fi
    return 0
}

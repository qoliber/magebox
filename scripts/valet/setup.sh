#!/usr/bin/env bash
#
# Install valet-to-magebox (MySQL migration helper) and store credentials.
#
# Usage:
#   ./scripts/valet/setup.sh
#   ./scripts/valet/setup.sh --uninstall
#
# Project credential patching lives in valet-to-magebox:
#   valet-to-magebox --update-projects
#   valet-to-magebox --update-projects --dry-run
#   valet-to-magebox --start-projects
#   valet-to-magebox --update-projects --start-projects
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

SOURCE="${SCRIPT_DIR}/valet-to-magebox.sh"
INSTALL_LIB="${HOME}/.config/magebox/valet-lib"

setup_usage() {
    cat <<EOF
Valet → MageBox setup

Usage:
  $(basename "$0")              Install valet-to-magebox to ~/bin and save credentials
  $(basename "$0") --uninstall  Remove ~/bin/valet-to-magebox, lib, and credentials file
  $(basename "$0") --help

During install you will be asked for:
  - Homebrew MySQL credentials (source, usually port 3307)
  - MageBox MySQL credentials (target, usually port 33080)

Patch Valet project configs (Magento env.php, Laravel .env, WordPress wp-config.php):

  valet-to-magebox --update-projects
  valet-to-magebox --update-projects --dry-run
  valet-to-magebox --start-projects
  valet-to-magebox --update-projects --start-projects

Credentials are stored in:
  ${VALET_TO_MAGEBOX_CONFIG}
EOF
}

install_tool() {
    [[ -f "${SOURCE}" ]] || valet_tm_die "Source not found: ${SOURCE}"

    echo "=== Valet → MageBox MySQL tool ==="
    echo ""
    valet_tm_prompt_credentials

    mkdir -p "${HOME}/bin" "${INSTALL_LIB}"
    cp "${SCRIPT_DIR}/lib/common.sh" "${SCRIPT_DIR}/lib/patch-projects.sh" "${SCRIPT_DIR}/lib/generate-magebox-yaml.sh" "${SCRIPT_DIR}/lib/start-projects.sh" "${INSTALL_LIB}/"
    cp "${SOURCE}" "${VALET_TO_MAGEBOX_BIN}"
    chmod +x "${VALET_TO_MAGEBOX_BIN}"

    echo "Installed: ${VALET_TO_MAGEBOX_BIN}"
    echo "Installed lib: ${INSTALL_LIB}"
    echo ""
    echo "Run from anywhere:"
    echo "  valet-to-magebox --help"
    echo "  valet-to-magebox --list"
    echo "  valet-to-magebox --update-projects --dry-run"
    echo "  valet-to-magebox --start-projects"
    echo ""

    if [[ ":${PATH}:" != *":${HOME}/bin:"* ]]; then
        echo "Note: ~/bin is not in your PATH."
        echo "Add this to ~/.zshrc:"
        echo '  export PATH="${HOME}/bin:${PATH}"'
        echo ""
        echo "Then run: source ~/.zshrc"
        echo ""
    fi

    printf "Update database credentials in all Valet parked projects now? [y/N]: "
    read -r patch_now
    if [[ "${patch_now}" =~ ^[yY]$ ]]; then
        echo ""
        "${VALET_TO_MAGEBOX_BIN}" --update-projects
        echo ""
        printf "Run magebox start for each project (nginx SSL vhosts)? [y/N]: "
        read -r start_now
        if [[ "${start_now}" =~ ^[yY]$ ]]; then
            echo ""
            "${VALET_TO_MAGEBOX_BIN}" --start-projects
        else
            echo ""
            echo "Later, run:"
            echo "  valet-to-magebox --start-projects"
        fi
    else
        echo ""
        echo "Later, run:"
        echo "  valet-to-magebox --update-projects"
        echo "  valet-to-magebox --start-projects"
    fi
}

uninstall_tool() {
    if [[ -f "${VALET_TO_MAGEBOX_BIN}" ]]; then
        rm -f "${VALET_TO_MAGEBOX_BIN}"
        echo "Removed: ${VALET_TO_MAGEBOX_BIN}"
    else
        echo "Not installed: ${VALET_TO_MAGEBOX_BIN}"
    fi

    if [[ -d "${INSTALL_LIB}" ]]; then
        rm -rf "${INSTALL_LIB}"
        echo "Removed: ${INSTALL_LIB}"
    fi

    if [[ -f "${VALET_TO_MAGEBOX_CONFIG}" ]]; then
        rm -f "${VALET_TO_MAGEBOX_CONFIG}"
        echo "Removed: ${VALET_TO_MAGEBOX_CONFIG}"
    fi

    if [[ -f "${HOME}/bin/magebox-mysql-tool" ]]; then
        rm -f "${HOME}/bin/magebox-mysql-tool"
        echo "Removed legacy: ${HOME}/bin/magebox-mysql-tool"
    fi
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --uninstall|-u)
            uninstall_tool
            exit 0
            ;;
        --update-projects|--patch-projects)
            shift
            exec "${SOURCE}" --update-projects "$@"
            ;;
        --start-projects)
            shift
            exec "${SOURCE}" --start-projects "$@"
            ;;
        -h|--help)
            setup_usage
            exit 0
            ;;
        "")
            break
            ;;
        *)
            valet_tm_die "Unknown option: $1 (use --help)"
            ;;
    esac
done

install_tool

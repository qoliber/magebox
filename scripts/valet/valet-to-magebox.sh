#!/usr/bin/env bash
#
# List, delete, or move databases between Homebrew MySQL and MageBox Docker MySQL.
# Install and configure with: ./scripts/valet/setup.sh
#
# Usage:
#   valet-to-magebox --list
#   valet-to-magebox --list='acme_shop_2025*'
#   valet-to-magebox --delete=dbname
#   valet-to-magebox --delete='acme_shop_2025*'
#   valet-to-magebox --move=dbname
#   valet-to-magebox --move='acme_shop_2025*'
#   valet-to-magebox --move=dbname --move-to=other_name
#   valet-to-magebox --move=dbname --keep-brew
#   valet-to-magebox --update-projects
#   valet-to-magebox --update-projects --dry-run
#   valet-to-magebox --start-projects
#   valet-to-magebox --update-projects --start-projects
#
# Environment:
#   BREW_MYSQL_PORT=3307          Port for Brew MySQL (default: 3307, avoids MageBox on 3306)
#   BREW_MYSQL_DATADIR=...        Data directory (default: /opt/homebrew/var/mysql)
#   BREW_MYSQL_BIN=...            MySQL bin directory (default: /opt/homebrew/opt/mysql@8.0/bin)
#   BREW_MYSQL_USER=root          Client user (default: root, or from ~/.my.cnf)
#   BREW_MYSQL_PASSWORD=          Client password (default: empty)
#   BREW_MYSQL_AUTO_START=1       Start Brew MySQL if unreachable (default: 1)
#   MAGEBOX_MYSQL_HOST=127.0.0.1  MageBox MySQL host (default: 127.0.0.1)
#   MAGEBOX_MYSQL_PORT=33080      MageBox MySQL port (default: 33080)
#   MAGEBOX_MYSQL_USER=root       MageBox MySQL user (default: root)
#   MAGEBOX_MYSQL_PASSWORD=magebox MageBox MySQL password (default: magebox)
#   VALET_TO_MAGEBOX_CONFIG=...     Credentials file (default: ~/.config/magebox/valet-to-magebox.env)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LIB_DIR="${VALET_TO_MAGEBOX_LIB_DIR:-${SCRIPT_DIR}/lib}"
if [[ ! -f "${LIB_DIR}/common.sh" ]]; then
    LIB_DIR="${HOME}/.config/magebox/valet-lib"
fi
# shellcheck source=lib/common.sh
source "${LIB_DIR}/common.sh"
# shellcheck source=lib/patch-projects.sh
source "${LIB_DIR}/patch-projects.sh"
# shellcheck source=lib/generate-magebox-yaml.sh
source "${LIB_DIR}/generate-magebox-yaml.sh"
# shellcheck source=lib/start-projects.sh
source "${LIB_DIR}/start-projects.sh"

valet_tm_load_config

BREW_MYSQL_PORT="${BREW_MYSQL_PORT:-3307}"
BREW_MYSQL_DATADIR="${BREW_MYSQL_DATADIR:-/opt/homebrew/var/mysql}"
BREW_MYSQL_BIN="${BREW_MYSQL_BIN:-/opt/homebrew/opt/mysql@8.0/bin}"
BREW_MYSQL_USER="${BREW_MYSQL_USER:-root}"
BREW_MYSQL_PASSWORD="${BREW_MYSQL_PASSWORD:-}"
BREW_MYSQL_AUTO_START="${BREW_MYSQL_AUTO_START:-1}"
BREW_MYSQL_SOCKET="${BREW_MYSQL_SOCKET:-/tmp/mysql-brew-${BREW_MYSQL_PORT}.sock}"

MAGEBOX_MYSQL_HOST="${MAGEBOX_MYSQL_HOST:-127.0.0.1}"
MAGEBOX_MYSQL_PORT="${MAGEBOX_MYSQL_PORT:-33080}"
MAGEBOX_MYSQL_USER="${MAGEBOX_MYSQL_USER:-root}"
MAGEBOX_MYSQL_PASSWORD="${MAGEBOX_MYSQL_PASSWORD:-magebox}"

STARTED_MYSQLD=0
ACTION=""
LIST_PATTERN=""
DELETE_TARGET=""
MOVE_TARGET=""
MOVE_TO=""
FORCE=0
KEEP_BREW=0
VALET_TM_START_PROJECTS=0

die() {
    valet_tm_die "$@"
}

normalize_option_value() {
    local value="$1"
    value="${value#=}"
    printf '%s' "${value}"
}

usage() {
    cat <<'EOF'
Manage Homebrew MySQL databases and move them to MageBox MySQL.

Usage:
  valet-to-magebox --list
  valet-to-magebox --list='acme_shop_2025*'
  valet-to-magebox --delete=<database>
  valet-to-magebox --delete='acme_shop_2025*'
  valet-to-magebox --move=<database>
  valet-to-magebox --move='acme_shop_2025*'
  valet-to-magebox --move=<database> --move-to=<target>
  valet-to-magebox --move=<database> --keep-brew
  valet-to-magebox --update-projects
  valet-to-magebox --update-projects --dry-run
  valet-to-magebox --start-projects
  valet-to-magebox --update-projects --start-projects

Options:
  --list              Show all user databases on Brew MySQL
  --list=<pattern>    Show databases matching a glob pattern (* and ?)
  --delete=<name>     Drop database(s) on Brew MySQL (supports * wildcard)
  --move=<name>       Move database(s) to MageBox MySQL (supports * wildcard)
  --move-to=<name>    Target database name on MageBox (default: same as --move)
  --keep-brew         Keep the Brew database after a successful move
  --force             Skip confirmations (also overwrites existing MageBox database)
  --update-projects   Add .magebox.yaml (if missing) and patch DB/URLs in Valet projects
  --start-projects    Run magebox start for each project with .magebox.yaml (nginx SSL vhost)
  --dry-run, -n       With --update-projects or --start-projects: show planned changes only
  -h, --help          Show this help

Notes:
  - Brew MySQL uses port 3307 by default so MageBox (3306/33080) can keep running.
  - If Brew MySQL is not reachable, it is started temporarily (when BREW_MYSQL_AUTO_START=1).
  - After a successful move, the Brew database is removed unless --keep-brew is set.
  - System databases (mysql, sys, …) cannot be deleted or moved.
  - Wildcards: use * and ? (e.g. anfors_2025* matches anfors_20250816, anfors_20250925, …).
  - --move-to cannot be combined with a wildcard pattern (each database keeps its own name).
  - --update-projects adds .magebox.yaml (if missing), backs up config to *.valet, patches DB + https URLs.
  - --start-projects registers each site in nginx (fixes ERR_CERT_COMMON_NAME_INVALID for old Valet sites).

Environment:
  BREW_MYSQL_PORT, BREW_MYSQL_DATADIR, BREW_MYSQL_BIN, BREW_MYSQL_USER,
  BREW_MYSQL_PASSWORD, BREW_MYSQL_AUTO_START,
  MAGEBOX_MYSQL_HOST, MAGEBOX_MYSQL_PORT, MAGEBOX_MYSQL_USER, MAGEBOX_MYSQL_PASSWORD,
  VALET_TO_MAGEBOX_CONFIG

Credentials:
  Run ./scripts/valet/setup.sh to save Brew and MageBox MySQL credentials to
  ~/.config/magebox/valet-to-magebox.env (mode 600).
EOF
}

is_system_database() {
    case "$1" in
        mysql|information_schema|performance_schema|sys) return 0 ;;
        *) return 1 ;;
    esac
}

require_binaries() {
    [[ -x "${BREW_MYSQL_BIN}/mysql" ]] || die "mysql not found at ${BREW_MYSQL_BIN}/mysql (set BREW_MYSQL_BIN)"
    [[ -x "${BREW_MYSQL_BIN}/mysqldump" ]] || die "mysqldump not found at ${BREW_MYSQL_BIN}/mysqldump (set BREW_MYSQL_BIN)"
    [[ -d "${BREW_MYSQL_DATADIR}" ]] || die "datadir not found: ${BREW_MYSQL_DATADIR}"
}

escape_identifier() {
    printf '%s' "${1//\`/\\\`}"
}

magebox_exec() {
    local cmd=("${BREW_MYSQL_BIN}/mysql" --protocol=TCP -h"${MAGEBOX_MYSQL_HOST}" "-P${MAGEBOX_MYSQL_PORT}" "-u${MAGEBOX_MYSQL_USER}" --batch --skip-column-names)
    if [[ -n "${MAGEBOX_MYSQL_PASSWORD}" ]]; then
        cmd+=("-p${MAGEBOX_MYSQL_PASSWORD}")
    fi
    "${cmd[@]}" "$@"
}

magebox_can_connect() {
    magebox_exec -e "SELECT 1" >/dev/null 2>&1
}

magebox_database_exists() {
    local target="$1"
    local escaped
    escaped="$(escape_identifier "${target}")"
    [[ "$(magebox_exec -N -e "SELECT COUNT(*) FROM information_schema.schemata WHERE schema_name = '${escaped//\'/\\\'}';")" == "1" ]]
}

magebox_table_count() {
    local target="$1"
    local escaped
    escaped="$(escape_identifier "${target}")"
    magebox_exec -N -e "
        SELECT COUNT(*)
        FROM information_schema.tables
        WHERE table_schema = '${escaped//\'/\\\'}';
    "
}

brew_table_count() {
    local target="$1"
    local escaped
    escaped="$(escape_identifier "${target}")"
    mysql_exec -N -e "
        SELECT COUNT(*)
        FROM information_schema.tables
        WHERE table_schema = '${escaped//\'/\\\'}';
    "
}

port_is_listening() {
    lsof -nP -iTCP:"${BREW_MYSQL_PORT}" -sTCP:LISTEN >/dev/null 2>&1
}

mysql_exec() {
    local cmd=("${BREW_MYSQL_BIN}/mysql" --protocol=TCP -h127.0.0.1 "-P${BREW_MYSQL_PORT}" "-u${BREW_MYSQL_USER}" --batch --skip-column-names)
    if [[ -n "${BREW_MYSQL_PASSWORD}" ]]; then
        cmd+=("-p${BREW_MYSQL_PASSWORD}")
    fi
    "${cmd[@]}" "$@"
}

mysql_can_connect() {
    mysql_exec -e "SELECT 1" >/dev/null 2>&1
}

start_brew_mysqld() {
    if [[ "${BREW_MYSQL_AUTO_START}" != "1" ]]; then
        return 1
    fi

    if port_is_listening; then
        local attempts=0
        while ! mysql_can_connect; do
            attempts=$((attempts + 1))
            [[ "${attempts}" -lt 10 ]] || return 1
            sleep 1
        done
        return 0
    fi

    echo "Starting Homebrew MySQL on port ${BREW_MYSQL_PORT}…" >&2
    "${BREW_MYSQL_BIN}/mysqld_safe" \
        --datadir="${BREW_MYSQL_DATADIR}" \
        --port="${BREW_MYSQL_PORT}" \
        --socket="${BREW_MYSQL_SOCKET}" \
        --bind-address=127.0.0.1 \
        >/dev/null 2>&1 &

    local attempts=0
    until mysql_can_connect; do
        attempts=$((attempts + 1))
        if [[ "${attempts}" -ge 30 ]]; then
            return 1
        fi
        sleep 1
    done

    STARTED_MYSQLD=1
    echo "Brew MySQL is ready on port ${BREW_MYSQL_PORT}." >&2
    return 0
}

stop_brew_mysqld() {
    if [[ "${STARTED_MYSQLD}" != "1" ]]; then
        return 0
    fi

    echo "Stopping temporary Brew MySQL…" >&2
    mysql_exec -e "SHUTDOWN" >/dev/null 2>&1 || true
    STARTED_MYSQLD=0
}

ensure_connection() {
    if mysql_can_connect; then
        return 0
    fi

    start_brew_mysqld || die "Cannot connect to Brew MySQL on port ${BREW_MYSQL_PORT}. Start it manually or set BREW_MYSQL_AUTO_START=1."
}

fetch_user_databases() {
    mysql_exec -N -e "
        SELECT schema_name
        FROM information_schema.schemata
        WHERE schema_name NOT IN ('mysql','information_schema','performance_schema','sys','.')
        ORDER BY schema_name;
    "
}

has_glob_pattern() {
    [[ "$1" == *'*'* || "$1" == *'?'* ]]
}

database_matches_pattern() {
    local db="$1"
    local pattern="$2"
    [[ "${db}" == ${pattern} ]]
}

# Prints matching database names (one per line). Sets RESOLVED_DB_COUNT.
resolve_database_pattern() {
    local pattern="$1"
    local db
    RESOLVED_DATABASES=()
    RESOLVED_DB_COUNT=0

    [[ -n "${pattern}" ]] || die "Database pattern is empty."

    if is_system_database "${pattern}" && ! has_glob_pattern "${pattern}"; then
        die "Refusing to match system database '${pattern}'."
    fi

    while IFS= read -r db; do
        [[ -n "${db}" ]] || continue
        is_system_database "${db}" && continue
        if database_matches_pattern "${db}" "${pattern}"; then
            RESOLVED_DATABASES+=("${db}")
            RESOLVED_DB_COUNT=$((RESOLVED_DB_COUNT + 1))
        fi
    done <<< "$(fetch_user_databases)"

    if [[ "${RESOLVED_DB_COUNT}" -eq 0 ]]; then
        if has_glob_pattern "${pattern}"; then
            die "No Brew databases match pattern '${pattern}'."
        fi
        die "Database '${pattern}' not found on Brew MySQL."
    fi
}

print_resolved_databases() {
    local label="$1"
    local db
    printf '%s\n' "${label}"
    for db in "${RESOLVED_DATABASES[@]}"; do
        printf '  %s\n' "${db}"
    done
}

confirm_pattern_action() {
    local action="$1"
    local pattern="$2"

    if [[ "${FORCE}" == "1" ]]; then
        return 0
    fi

    print_resolved_databases "About to ${action} ${RESOLVED_DB_COUNT} Brew MySQL database(s):"
    echo "  (port ${BREW_MYSQL_PORT}, datadir ${BREW_MYSQL_DATADIR})"
    if has_glob_pattern "${pattern}"; then
        printf "Type the pattern to confirm: "
        read -r confirmation
        [[ "${confirmation}" == "${pattern}" ]] || die "Confirmation did not match. Aborted."
        return 0
    fi

    printf "Type the database name to confirm: "
    read -r confirmation
    [[ "${confirmation}" == "${pattern}" ]] || die "Confirmation did not match. Aborted."
}

cmd_list() {
    local pattern="${1:-}"

    ensure_connection

    local output
    if [[ -n "${pattern}" ]]; then
        resolve_database_pattern "${pattern}"
        output="$(printf '%s\n' "${RESOLVED_DATABASES[@]}")"
    else
        output="$(fetch_user_databases)"
    fi

    if [[ -z "${output}" ]]; then
        if [[ -n "${pattern}" ]]; then
            die "No databases match pattern '${pattern}'."
        fi
        echo "No user databases found on Brew MySQL (port ${BREW_MYSQL_PORT})."
        return 0
    fi

    local count
    count="$(printf '%s\n' "${output}" | wc -l | tr -d ' ')"

    if [[ -n "${pattern}" ]]; then
        printf '%s\n' "Brew MySQL databases matching '${pattern}' (port ${BREW_MYSQL_PORT}):"
    else
        printf '%s\n' "Brew MySQL databases (port ${BREW_MYSQL_PORT}, datadir ${BREW_MYSQL_DATADIR}):"
    fi
    printf '%s\n' "────────────────────────────────────────────────────────────"
    while IFS= read -r db; do
        [[ -n "${db}" ]] && printf '  %s\n' "${db}"
    done <<< "${output}"
    printf '\nTotal: %s database(s)\n' "${count}"
}

database_exists() {
    resolve_database_pattern "$1"
    [[ "${RESOLVED_DB_COUNT}" -eq 1 && "${RESOLVED_DATABASES[0]}" == "$1" ]]
}

delete_single() {
    local target="$1"
    mysql_exec -e "DROP DATABASE \`$(escape_identifier "${target}")\`;"
    echo "Deleted database '${target}'."
}

cmd_delete() {
    local pattern="$1"
    [[ -n "${pattern}" ]] || die "Missing database name. Use --delete=<name>"

    ensure_connection
    resolve_database_pattern "${pattern}"
    confirm_pattern_action "permanently delete" "${pattern}"

    local db
    for db in "${RESOLVED_DATABASES[@]}"; do
        delete_single "${db}"
    done

    if [[ "${RESOLVED_DB_COUNT}" -gt 1 ]]; then
        echo "Deleted ${RESOLVED_DB_COUNT} database(s)."
    fi
}

move_single() {
    local source="$1"
    local dest="${2:-${source}}"

    [[ -n "${source}" ]] || die "Missing source database name."
    [[ -n "${dest}" ]] || die "Target database name is empty."

    if is_system_database "${source}" || is_system_database "${dest}"; then
        die "Refusing to move system database."
    fi

    magebox_can_connect || die "Cannot connect to MageBox MySQL at ${MAGEBOX_MYSQL_HOST}:${MAGEBOX_MYSQL_PORT} (user ${MAGEBOX_MYSQL_USER})."

    if magebox_database_exists "${dest}"; then
        if [[ "${FORCE}" != "1" ]]; then
            die "Database '${dest}' already exists on MageBox. Use --force to overwrite."
        fi
        echo "Dropping existing MageBox database '${dest}'…" >&2
        magebox_exec -e "DROP DATABASE \`$(escape_identifier "${dest}")\`;"
    fi

    local brew_tables
    brew_tables="$(brew_table_count "${source}")"

    echo "Creating '${dest}' on MageBox…" >&2
    magebox_exec -e "CREATE DATABASE \`$(escape_identifier "${dest}")\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"

    echo "Dumping '${source}' from Brew MySQL…" >&2

    local dump_cmd=("${BREW_MYSQL_BIN}/mysqldump" --protocol=TCP -h127.0.0.1 "-P${BREW_MYSQL_PORT}" "-u${BREW_MYSQL_USER}" --single-transaction --routines --triggers --set-gtid-purged=OFF "${source}")
    local import_cmd=("${BREW_MYSQL_BIN}/mysql" --protocol=TCP -h"${MAGEBOX_MYSQL_HOST}" "-P${MAGEBOX_MYSQL_PORT}" "-u${MAGEBOX_MYSQL_USER}" "${dest}")
    if [[ -n "${BREW_MYSQL_PASSWORD}" ]]; then
        dump_cmd+=("-p${BREW_MYSQL_PASSWORD}")
    fi
    if [[ -n "${MAGEBOX_MYSQL_PASSWORD}" ]]; then
        import_cmd+=("-p${MAGEBOX_MYSQL_PASSWORD}")
    fi

    echo "Importing into MageBox as '${dest}'…" >&2
    if ! "${dump_cmd[@]}" | "${import_cmd[@]}"; then
        die "Import failed. Brew database '${source}' was not removed."
    fi

    local magebox_tables
    magebox_tables="$(magebox_table_count "${dest}")"

    echo "Import complete: ${brew_tables} table(s) on Brew → ${magebox_tables} table(s) on MageBox."

    if [[ "${KEEP_BREW}" == "1" ]]; then
        echo "Brew database '${source}' kept (--keep-brew)."
        return 0
    fi

    if [[ "${FORCE}" != "1" ]]; then
        printf "Remove '${source}' from Brew MySQL? [y/N]: "
        read -r remove_confirm
        [[ "${remove_confirm}" =~ ^[yY]$ ]] || {
            echo "Brew database '${source}' kept."
            return 0
        }
    fi

    mysql_exec -e "DROP DATABASE \`$(escape_identifier "${source}")\`;"
    echo "Removed '${source}' from Brew MySQL. Data is now on MageBox as '${dest}'."
}

cmd_move() {
    local pattern="$1"
    local dest_override="${2:-}"

    [[ -n "${pattern}" ]] || die "Missing database name. Use --move=<name>"

    if has_glob_pattern "${pattern}" && [[ -n "${dest_override}" && "${dest_override}" != "${pattern}" ]]; then
        die "Cannot use --move-to with a wildcard pattern. Each database is moved under its own name."
    fi

    ensure_connection
    resolve_database_pattern "${pattern}"

    if [[ "${RESOLVED_DB_COUNT}" -gt 1 ]]; then
        echo "Moving ${RESOLVED_DB_COUNT} database(s) to MageBox…" >&2
        confirm_pattern_action "move" "${pattern}"
        local db
        for db in "${RESOLVED_DATABASES[@]}"; do
            echo "" >&2
            echo "── ${db} ──" >&2
            move_single "${db}" "${db}"
        done
        echo ""
        echo "Moved ${RESOLVED_DB_COUNT} database(s) to MageBox."
        return 0
    fi

    local source="${RESOLVED_DATABASES[0]}"
    local dest="${dest_override:-${source}}"

    if [[ "${FORCE}" != "1" ]]; then
        echo "Move database:"
        echo "  From: Brew MySQL  ${source} (port ${BREW_MYSQL_PORT})"
        echo "  To:   MageBox MySQL ${dest} (${MAGEBOX_MYSQL_HOST}:${MAGEBOX_MYSQL_PORT})"
        if [[ "${KEEP_BREW}" == "1" ]]; then
            echo "  Brew copy will be kept after import (--keep-brew)."
        else
            echo "  Brew copy will be deleted after a successful import."
        fi
        printf "Type the source database name to confirm: "
        read -r confirmation
        [[ "${confirmation}" == "${source}" ]] || die "Confirmation did not match. Aborted."
    fi

    move_single "${source}" "${dest}"
}

parse_args() {
    if [[ $# -eq 0 ]]; then
        usage
        exit 1
    fi

    while [[ $# -gt 0 ]]; do
        if [[ "$1" == =* ]]; then
            die "Invalid argument '${1}'. Remove the leading '=' (use --delete=name, not =--delete=name)."
        fi

        case "$1" in
            -h|--help)
                usage
                exit 0
                ;;
            --list)
                ACTION="list"
                shift
                ;;
            --list=*)
                ACTION="list"
                LIST_PATTERN="$(normalize_option_value "${1#--list=}")"
                shift
                ;;
            --update-projects|--patch-projects)
                [[ -z "${ACTION}" || "${ACTION}" == "update-projects" ]] || die "Use only one action per run."
                ACTION="update-projects"
                shift
                ;;
            --start-projects)
                VALET_TM_START_PROJECTS=1
                if [[ -z "${ACTION}" ]]; then
                    ACTION="start-projects"
                fi
                shift
                ;;
            --dry-run|-n)
                export VALET_TM_DRY_RUN=1
                shift
                ;;
            --delete=*)
                [[ -z "${ACTION}" || "${ACTION}" == "delete" ]] || die "Use only one action per run."
                ACTION="delete"
                DELETE_TARGET="$(normalize_option_value "${1#--delete=}")"
                shift
                ;;
            --delete)
                [[ -z "${ACTION}" || "${ACTION}" == "delete" ]] || die "Use only one action per run."
                ACTION="delete"
                shift
                [[ $# -gt 0 ]] || die "Missing value for --delete"
                DELETE_TARGET="$(normalize_option_value "$1")"
                shift
                ;;
            --move=*)
                [[ -z "${ACTION}" || "${ACTION}" == "move" ]] || die "Use only one action per run."
                ACTION="move"
                MOVE_TARGET="$(normalize_option_value "${1#--move=}")"
                shift
                ;;
            --move)
                [[ -z "${ACTION}" || "${ACTION}" == "move" ]] || die "Use only one action per run."
                ACTION="move"
                shift
                [[ $# -gt 0 ]] || die "Missing value for --move"
                MOVE_TARGET="$(normalize_option_value "$1")"
                shift
                ;;
            --move-to=*)
                MOVE_TO="$(normalize_option_value "${1#--move-to=}")"
                shift
                ;;
            --move-to)
                shift
                [[ $# -gt 0 ]] || die "Missing value for --move-to"
                MOVE_TO="$(normalize_option_value "$1")"
                shift
                ;;
            --keep-brew)
                KEEP_BREW=1
                shift
                ;;
            --force)
                FORCE=1
                shift
                ;;
            *)
                die "Unknown argument: $1 (use --help)"
                ;;
        esac
    done

    [[ -n "${ACTION}" ]] || die "Specify --list, --delete=<name>, --move=<name>, --update-projects, or --start-projects"
}

main() {
    parse_args "$@"

    case "${ACTION}" in
        update-projects)
            if [[ "${VALET_TM_DRY_RUN}" != "1" ]]; then
                [[ -f "${VALET_TO_MAGEBOX_CONFIG}" ]] || die "No credentials file. Run ./scripts/valet/setup.sh first."
            fi
            valet_tm_patch_all_projects
            if [[ "${VALET_TM_START_PROJECTS}" == "1" ]]; then
                echo ""
                valet_tm_start_all_projects
            elif [[ "${VALET_TM_DRY_RUN}" != "1" ]]; then
                echo ""
                echo "SSL: run 'valet-to-magebox --start-projects' so each *.test site gets its own nginx vhost and certificate."
                echo "  Without magebox start, Chrome may show ERR_CERT_COMMON_NAME_INVALID (wrong default SSL server)."
            fi
            exit 0
            ;;
        start-projects)
            valet_tm_start_all_projects
            exit 0
            ;;
    esac

    require_binaries
    trap stop_brew_mysqld EXIT

    case "${ACTION}" in
        list)
            cmd_list "${LIST_PATTERN}"
            ;;
        delete)
            cmd_delete "${DELETE_TARGET}"
            ;;
        move)
            cmd_move "${MOVE_TARGET}" "${MOVE_TO:-${MOVE_TARGET}}"
            ;;
        *)
            die "Unknown action: ${ACTION}"
            ;;
    esac
}

main "$@"
exit 0

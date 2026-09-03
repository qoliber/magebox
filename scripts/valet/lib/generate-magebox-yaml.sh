#!/usr/bin/env bash
#
# Generate .magebox.yaml for Valet parked projects (when missing).
# Part of MageBox — https://github.com/qoliber/magebox
#
# This file is sourced by valet-to-magebox.sh.
# It must not be executed directly.

# --- Global defaults ---

valet_tm_read_magebox_global_defaults() {
    VALET_TM_DEFAULT_PHP="${VALET_TM_DEFAULT_PHP:-8.3}"
    VALET_TM_DEFAULT_MYSQL="${VALET_TM_DEFAULT_MYSQL:-8.0}"
    VALET_TM_DEFAULT_TLD="${VALET_TM_DEFAULT_TLD:-test}"
    VALET_TM_DEFAULT_REDIS="${VALET_TM_DEFAULT_REDIS:-1}"
    VALET_TM_DEFAULT_VALKEY="${VALET_TM_DEFAULT_VALKEY:-0}"
    VALET_TM_DEFAULT_OPENSEARCH="${VALET_TM_DEFAULT_OPENSEARCH:-}"
    VALET_TM_DEFAULT_ELASTICSEARCH="${VALET_TM_DEFAULT_ELASTICSEARCH:-}"
    VALET_TM_DEFAULT_MAILPIT="${VALET_TM_DEFAULT_MAILPIT:-1}"
    VALET_TM_OPENSEARCH_VERSION="${VALET_TM_OPENSEARCH_VERSION:-2.19}"
    VALET_TM_OPENSEARCH_MEMORY="${VALET_TM_OPENSEARCH_MEMORY:-2g}"
    VALET_TM_ELASTICSEARCH_VERSION="${VALET_TM_ELASTICSEARCH_VERSION:-8.11}"

    local global_cfg="${HOME}/.magebox/config.yaml"
    [[ -f "${global_cfg}" ]] || return 0
    command -v python3 >/dev/null 2>&1 || return 0

    # shellcheck disable=SC2046
    eval "$(python3 - "${global_cfg}" <<'PY'
import sys
try:
    import yaml
except ImportError:
    yaml = None

path = sys.argv[1]
data = {}
if yaml is not None:
    with open(path, encoding="utf-8") as fh:
        data = yaml.safe_load(fh) or {}
else:
    import re
    text = open(path, encoding="utf-8").read()
    for key, var in (
        ("default_php", "VALET_TM_DEFAULT_PHP"),
        ("tld", "VALET_TM_DEFAULT_TLD"),
    ):
        m = re.search(rf"^{key}:\s*[\"']?([^\"'\n]+)", text, re.M)
        if m:
            print(f'{var}="{m.group(1).strip()}"')
    m = re.search(r"^\s*mysql:\s*[\"']?([^\"'\n]+)", text, re.M)
    if m:
        print(f'VALET_TM_DEFAULT_MYSQL="{m.group(1).strip()}"')
    if re.search(r"^\s*redis:\s*true", text, re.M):
        print('VALET_TM_DEFAULT_REDIS=1')
    if re.search(r"^\s*valkey:\s*true", text, re.M):
        print('VALET_TM_DEFAULT_VALKEY=1')
    if re.search(r"^\s*mailpit:\s*true", text, re.M):
        print('VALET_TM_DEFAULT_MAILPIT=1')
    sys.exit(0)

defaults = data.get("default_services") or {}
php = data.get("default_php") or "8.3"
tld = data.get("tld") or "test"
print(f'VALET_TM_DEFAULT_PHP="{php}"')
print(f'VALET_TM_DEFAULT_TLD="{tld}"')
if defaults.get("mysql"):
    print(f'VALET_TM_DEFAULT_MYSQL="{defaults["mysql"]}"')
if defaults.get("redis"):
    print("VALET_TM_DEFAULT_REDIS=1")
if defaults.get("valkey"):
    print("VALET_TM_DEFAULT_VALKEY=1")
if defaults.get("opensearch"):
    os_ver = defaults["opensearch"]
    if isinstance(os_ver, dict):
        if os_ver.get("version"):
            print(f'VALET_TM_OPENSEARCH_VERSION="{os_ver["version"]}"')
        if os_ver.get("memory"):
            print(f'VALET_TM_OPENSEARCH_MEMORY="{os_ver["memory"]}"')
    else:
        print(f'VALET_TM_DEFAULT_OPENSEARCH="{os_ver}"')
        print(f'VALET_TM_OPENSEARCH_VERSION="{os_ver}"')
if defaults.get("elasticsearch"):
    es_ver = defaults["elasticsearch"]
    if isinstance(es_ver, dict) and es_ver.get("version"):
        print(f'VALET_TM_ELASTICSEARCH_VERSION="{es_ver["version"]}"')
    else:
        print(f'VALET_TM_DEFAULT_ELASTICSEARCH="{es_ver}"')
        print(f'VALET_TM_ELASTICSEARCH_VERSION="{es_ver}"')
if defaults.get("mailpit"):
    print("VALET_TM_DEFAULT_MAILPIT=1")
PY
)"
}

# --- Search engine detection ---

# Detect whether a Magento project uses OpenSearch or Elasticsearch.
# Prints "opensearch" or "elasticsearch" to stdout; empty if undetermined.
valet_tm_detect_magento_search_engine() {
    local root="$1"
    local env_file="${root}/app/etc/env.php"

    [[ -f "${env_file}" ]] || return 0
    command -v php >/dev/null 2>&1 || return 0

    php -r '
$config = @include $argv[1];
if (!is_array($config)) { exit(0); }
$engine = $config["system"]["default"]["catalog"]["search"]["engine"] ?? "";
$engine = is_string($engine) ? strtolower($engine) : "";
if ($engine !== "" && str_contains($engine, "elastic")) {
    echo "elasticsearch";
    exit(0);
}
echo "opensearch";
' "${env_file}" 2>/dev/null || true
}

# Write the search service YAML block to stdout.
valet_tm_write_search_service_yaml() {
    local engine="${1:-opensearch}"

    case "${engine}" in
        elasticsearch)
            printf '  elasticsearch: "%s"\n' "${VALET_TM_ELASTICSEARCH_VERSION}"
            ;;
        *)
            local version="${VALET_TM_OPENSEARCH_VERSION}"
            if [[ -n "${VALET_TM_DEFAULT_OPENSEARCH}" ]]; then
                version="${VALET_TM_DEFAULT_OPENSEARCH}"
            fi
            printf '  opensearch:\n'
            printf '    version: "%s"\n' "${version}"
            printf '    memory: "%s"\n' "${VALET_TM_OPENSEARCH_MEMORY}"
            ;;
    esac
}

# --- PHP version detection ---

valet_tm_magebox_project_name() {
    local root="$1"
    local name
    name="$(valet_tm_valet_site_name "${root}")"
    name="${name//\//.}"
    printf '%s' "${name}"
}

valet_tm_detect_php_version() {
    local root="$1"
    local composer="${root}/composer.json"

    [[ -f "${composer}" ]] || return 1
    command -v python3 >/dev/null 2>&1 || return 1

    python3 - "${composer}" <<'PY'
import json, re, sys

supported = {"8.1", "8.2", "8.3", "8.4"}
path = sys.argv[1]
with open(path, encoding="utf-8") as fh:
    data = json.load(fh)

def major_minor(value: str) -> str:
    m = re.search(r"(\d+)\.(\d+)", value or "")
    return f"{m.group(1)}.{m.group(2)}" if m else ""

candidates = []
platform = (data.get("config") or {}).get("platform") or {}
if platform.get("php"):
    candidates.append(platform["php"])
require = data.get("require") or {}
if require.get("php"):
    candidates.append(require["php"])

for raw in candidates:
    version = major_minor(raw)
    if version in supported:
        print(version)
        sys.exit(0)
sys.exit(1)
PY
}

# --- YAML generation ---

valet_tm_build_magebox_yaml() {
    local root="$1"
    local ptype="$2"
    local name host php doc_root

    valet_tm_read_magebox_global_defaults

    name="$(valet_tm_magebox_project_name "${root}")"
    host="${name}.${VALET_TM_DEFAULT_TLD}"

    php="$(valet_tm_detect_php_version "${root}" 2>/dev/null || true)"
    php="${php:-${VALET_TM_DEFAULT_PHP}}"

    case "${ptype}" in
        magento) doc_root="pub" ;;
        laravel) doc_root="public" ;;
        *) doc_root="." ;;
    esac

    {
        printf 'name: %s\n' "${name}"
        if [[ "${ptype}" == "laravel" ]]; then
            printf 'type: laravel\n'
        fi
        printf 'domains:\n'
        printf '  - host: %s\n' "${host}"
        printf '    root: %s\n' "${doc_root}"
        printf '    ssl: true\n'
        printf 'php: "%s"\n' "${php}"
        printf 'services:\n'
        if [[ -n "${VALET_TM_DEFAULT_MYSQL}" ]]; then
            printf '  mysql: "%s"\n' "${VALET_TM_DEFAULT_MYSQL}"
        fi
        if [[ "${VALET_TM_DEFAULT_VALKEY}" == "1" ]]; then
            printf '  valkey: true\n'
        elif [[ "${VALET_TM_DEFAULT_REDIS}" == "1" ]]; then
            printf '  redis: true\n'
        fi
        if [[ "${ptype}" == "magento" ]]; then
            local search_engine="opensearch"
            if [[ -n "${VALET_TM_DEFAULT_ELASTICSEARCH}" && -z "${VALET_TM_DEFAULT_OPENSEARCH}" ]]; then
                search_engine="elasticsearch"
            else
                local detected
                detected="$(valet_tm_detect_magento_search_engine "${root}" 2>/dev/null || true)"
                [[ -n "${detected}" ]] && search_engine="${detected}"
            fi
            valet_tm_write_search_service_yaml "${search_engine}"
        elif [[ -n "${VALET_TM_DEFAULT_OPENSEARCH}" ]]; then
            valet_tm_write_search_service_yaml "opensearch"
        elif [[ -n "${VALET_TM_DEFAULT_ELASTICSEARCH}" ]]; then
            valet_tm_write_search_service_yaml "elasticsearch"
        fi
        if [[ "${VALET_TM_DEFAULT_MAILPIT}" == "1" ]]; then
            printf '  mailpit: true\n'
        fi
    }
}

# --- Entrypoint ---

valet_tm_ensure_magebox_yaml() {
    local root="$1"
    local ptype="$2"
    local yaml="${root}/.magebox.yaml"

    case "${ptype}" in
        magento|laravel) ;;
        wordpress)
            echo "  skip .magebox.yaml: WordPress (MageBox has no WordPress type; patch DB/URLs only)"
            return 0
            ;;
        *)
            return 0
            ;;
    esac

    if [[ -f "${yaml}" ]]; then
        echo "  .magebox.yaml already exists"
        return 0
    fi

    local preview host_line
    preview="$(valet_tm_build_magebox_yaml "${root}" "${ptype}")"
    host_line="$(printf '%s\n' "${preview}" | grep -E '^  - host:' | head -1 | sed 's/.*host:[[:space:]]*//')"

    if [[ "${VALET_TM_DRY_RUN:-0}" == "1" ]]; then
        echo "  [dry-run] would create .magebox.yaml → host=${host_line}"
        return 0
    fi

    valet_tm_build_magebox_yaml "${root}" "${ptype}" >"${yaml}"
    echo "  created .magebox.yaml (host ${host_line})"
}

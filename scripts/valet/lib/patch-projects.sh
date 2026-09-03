#!/usr/bin/env bash
#
# Update database credentials and URLs in Valet parked / linked projects.
# Part of MageBox — https://github.com/qoliber/magebox
#
# This file is sourced by valet-to-magebox.sh.
# It must not be executed directly.

# --- Backup helper ---

# Copy the original credentials file to <filename>.valet before the first patch.
valet_tm_backup_credentials_file() {
    local file="$1"
    local backup="${file}.valet"

    [[ -f "${file}" ]] || return 1

    if [[ "${VALET_TM_DRY_RUN:-0}" == "1" ]]; then
        if [[ -f "${backup}" ]]; then
            echo "  [dry-run] backup exists: ${backup}"
        else
            echo "  [dry-run] would back up: ${file} → ${backup}"
        fi
        return 0
    fi

    if [[ -f "${backup}" ]]; then
        echo "  backup already exists: ${backup}"
        return 0
    fi

    cp -p "${file}" "${backup}"
    echo "  backed up: ${backup}"
}

# --- Magento patching ---

valet_tm_patch_magento() {
    local root="$1"
    local env_file="${root}/app/etc/env.php"
    local db_host="$2"
    local db_user="$3"
    local db_pass="$4"
    local base_url
    base_url="$(valet_tm_project_base_url "${root}")"

    if [[ ! -f "${env_file}" ]]; then
        echo "  skip: no app/etc/env.php" >&2
        return 1
    fi

    if ! command -v php >/dev/null 2>&1; then
        echo "  skip: php not found (needed to patch env.php)" >&2
        return 1
    fi

    valet_tm_backup_credentials_file "${env_file}" || return 1

    if [[ "${VALET_TM_DRY_RUN:-0}" == "1" ]]; then
        echo "  [dry-run] would set app/etc/env.php → host=${db_host}, user=${db_user}"
        echo "  [dry-run] would set web URLs: unsecure=http, secure=${base_url}, use_in_frontend=1"
        return 0
    fi

    VALET_TM_ENV_FILE="${env_file}" \
    VALET_TM_DB_HOST="${db_host}" \
    VALET_TM_DB_USER="${db_user}" \
    VALET_TM_DB_PASS="${db_pass}" \
    VALET_TM_BASE_URL="${base_url}" \
    php <<'PHP'
<?php
$path = getenv('VALET_TM_ENV_FILE');
$dbHost = getenv('VALET_TM_DB_HOST');
$dbUser = getenv('VALET_TM_DB_USER');
$dbPass = getenv('VALET_TM_DB_PASS');
$fallbackBase = getenv('VALET_TM_BASE_URL');

$config = include $path;
if (!is_array($config) || !isset($config['db']['connection']['default'])) {
    fwrite(STDERR, "  skip: unexpected env.php structure\n");
    exit(1);
}

$config['db']['connection']['default']['host'] = $dbHost;
$config['db']['connection']['default']['username'] = $dbUser;
$config['db']['connection']['default']['password'] = $dbPass;
unset($config['db']['connection']['default']['port']);

function valet_secure_magento_url(string $url, string $fallback): string
{
    if ($url === '') {
        return $fallback;
    }
    if (!preg_match('#^https?://#i', $url)) {
        $url = 'https://' . ltrim($url, '/');
    }
    $url = preg_replace('#^http://#i', 'https://', $url);
    return rtrim($url, '/') . '/';
}

function valet_patch_magento_urls(array &$node, string $fallback): int
{
    $count = 0;
    foreach ($node as $key => &$value) {
        if ($key === 'base_url' && is_string($value)) {
            $value = valet_secure_magento_url($value, $fallback);
            $count++;
        } elseif (is_array($value)) {
            $count += valet_patch_magento_urls($value, $fallback);
        }
    }
    unset($value);
    return $count;
}

function valet_apply_magento_ssl_settings(array &$config, string $secureBase): void
{
    $parsed = parse_url($secureBase);
    $host = $parsed['host'] ?? '';
    if ($host === '') {
        return;
    }
    $httpsBase = 'https://' . $host . '/';
    $httpBase = 'http://' . $host . '/';

    if (!isset($config['system']) || !is_array($config['system'])) {
        $config['system'] = [];
    }
    if (!isset($config['system']['default']) || !is_array($config['system']['default'])) {
        $config['system']['default'] = [];
    }
    if (!isset($config['system']['default']['web']) || !is_array($config['system']['default']['web'])) {
        $config['system']['default']['web'] = [];
    }

    $web = &$config['system']['default']['web'];
    if (!isset($web['unsecure']) || !is_array($web['unsecure'])) {
        $web['unsecure'] = [];
    }
    if (!isset($web['secure']) || !is_array($web['secure'])) {
        $web['secure'] = [];
    }

    $web['unsecure']['base_url'] = $httpBase;
    $web['secure']['base_url'] = $httpsBase;
    $web['secure']['use_in_frontend'] = '1';
    $web['secure']['use_in_adminhtml'] = '1';
    $web['secure']['enable_upgrade_insecure'] = '1';
    unset($web);

    if (!isset($config['system']['stores']) || !is_array($config['system']['stores'])) {
        return;
    }

    foreach ($config['system']['stores'] as &$store) {
        if (!is_array($store) || !isset($store['web']) || !is_array($store['web'])) {
            continue;
        }
        $storeHost = $host;
        if (isset($store['web']['secure']['base_url']) && is_string($store['web']['secure']['base_url'])) {
            $storeParsed = parse_url(valet_secure_magento_url($store['web']['secure']['base_url'], $httpsBase));
            if (!empty($storeParsed['host'])) {
                $storeHost = $storeParsed['host'];
            }
        }
        $storeHttps = 'https://' . $storeHost . '/';
        $storeHttp = 'http://' . $storeHost . '/';
        if (!isset($store['web']['unsecure']) || !is_array($store['web']['unsecure'])) {
            $store['web']['unsecure'] = [];
        }
        if (!isset($store['web']['secure']) || !is_array($store['web']['secure'])) {
            $store['web']['secure'] = [];
        }
        $store['web']['unsecure']['base_url'] = $storeHttp;
        $store['web']['secure']['base_url'] = $storeHttps;
        $store['web']['secure']['use_in_frontend'] = '1';
        $store['web']['secure']['use_in_adminhtml'] = '1';
    }
    unset($store);
}

$urlCount = valet_patch_magento_urls($config, $fallbackBase);
valet_apply_magento_ssl_settings($config, $fallbackBase);

$export = var_export($config, true);
file_put_contents($path, "<?php\nreturn " . $export . ";\n");
fwrite(STDERR, "  secured {$urlCount} base_url value(s); SSL flags enabled in env.php\n");
PHP
    valet_tm_magento_sync_ssl_db "${root}" "${base_url}"
    echo "  updated app/etc/env.php (host ${db_host}, SSL on)"
    echo "  tip: run 'magebox start' (or 'nginx -s reload') so vhosts/certs are picked up"
}

# --- Magento DB sync ---

valet_tm_magento_http_base_url() {
    local https_base="$1"
    local host
    host="$(printf '%s' "${https_base}" | sed -E 's#^https?://([^/]+)/?.*#\1#')"
    printf 'http://%s/' "${host}"
}

valet_tm_magento_sync_ssl_db() {
    local root="$1"
    local https_base="$2"
    local http_base dbname

    http_base="$(valet_tm_magento_http_base_url "${https_base}")"
    dbname=""
    if [[ -f "${root}/.magebox.yaml" ]]; then
        dbname="$(grep -E '^name:[[:space:]]*' "${root}/.magebox.yaml" | head -1 | sed -E 's/^name:[[:space:]]*//')"
    fi
    [[ -n "${dbname}" ]] || return 0

    valet_tm_load_config
    if ! command -v mysql >/dev/null 2>&1; then
        echo "  note: mysql CLI not found — run bin/magento cache:flush after patch"
        return 0
    fi

    local -a mysql_cmd=(mysql "-h${MAGEBOX_MYSQL_HOST:-127.0.0.1}" "-P${MAGEBOX_MYSQL_PORT:-33080}" "-u${MAGEBOX_MYSQL_USER:-root}")
    if [[ -n "${MAGEBOX_MYSQL_PASSWORD:-}" ]]; then
        mysql_cmd+=("-p${MAGEBOX_MYSQL_PASSWORD}")
    fi

    if ! "${mysql_cmd[@]}" -N -e "USE \`${dbname//\`/\\\`}\`; SELECT 1" >/dev/null 2>&1; then
        echo "  note: database '${dbname}' not found — skipped DB URL sync"
        return 0
    fi

    local sql
    sql="$(cat <<SQL
REPLACE INTO core_config_data (scope, scope_id, path, value) VALUES
('default', 0, 'web/unsecure/base_url', '${http_base}'),
('default', 0, 'web/secure/base_url', '${https_base}'),
('default', 0, 'web/secure/use_in_frontend', '1'),
('default', 0, 'web/secure/use_in_adminhtml', '1'),
('default', 0, 'web/secure/enable_upgrade_insecure', '1');
SQL
)"
    if "${mysql_cmd[@]}" "${dbname}" -e "${sql}" >/dev/null 2>&1; then
        echo "  synced SSL URL settings in database '${dbname}'"
    else
        echo "  note: could not sync database (run bin/magento cache:flush)"
    fi
}

# --- Laravel patching ---

valet_tm_upsert_dotenv_key() {
    local file="$1"
    local key="$2"
    local value="$3"

    [[ -f "${file}" ]] || return 1

    if command -v python3 >/dev/null 2>&1; then
        python3 - "${file}" "${key}" "${value}" <<'PY'
import re, sys

path, key, value = sys.argv[1], sys.argv[2], sys.argv[3]
with open(path, encoding="utf-8") as fh:
    lines = fh.readlines()

pattern = re.compile(rf"^{re.escape(key)}=")
out = []
found = False
for line in lines:
    if pattern.match(line):
        out.append(f"{key}={value}\n")
        found = True
    else:
        out.append(line)
if not found:
    if out and not out[-1].endswith("\n"):
        out[-1] += "\n"
    out.append(f"{key}={value}\n")
with open(path, "w", encoding="utf-8") as fh:
    fh.writelines(out)
PY
        return 0
    fi

    if grep -q "^${key}=" "${file}" 2>/dev/null; then
        local tmp
        tmp="$(mktemp)"
        while IFS= read -r line || [[ -n "${line}" ]]; do
            if [[ "${line}" == "${key}="* ]]; then
                printf '%s=%s\n' "${key}" "${value}"
            else
                printf '%s\n' "${line}"
            fi
        done <"${file}" >"${tmp}"
        mv "${tmp}" "${file}"
    else
        printf '%s=%s\n' "${key}" "${value}" >>"${file}"
    fi
}

valet_tm_patch_laravel() {
    local root="$1"
    local env_file="${root}/.env"
    local db_host="$2"
    local db_port="$3"
    local db_user="$4"
    local db_pass="$5"
    local app_url
    app_url="$(valet_tm_project_app_url "${root}")"

    [[ -f "${env_file}" ]] || {
        echo "  skip: no .env" >&2
        return 1
    }

    valet_tm_backup_credentials_file "${env_file}" || return 1

    if [[ "${VALET_TM_DRY_RUN:-0}" == "1" ]]; then
        echo "  [dry-run] would set .env → DB_HOST=${db_host}, DB_PORT=${db_port}, DB_USERNAME=${db_user}"
        echo "  [dry-run] would set APP_URL=${app_url} (https)"
        return 0
    fi

    valet_tm_upsert_dotenv_key "${env_file}" "DB_HOST" "${db_host}"
    valet_tm_upsert_dotenv_key "${env_file}" "DB_PORT" "${db_port}"
    valet_tm_upsert_dotenv_key "${env_file}" "DB_USERNAME" "${db_user}"
    valet_tm_upsert_dotenv_key "${env_file}" "DB_PASSWORD" "${db_pass}"

    if grep -q '^APP_URL=' "${env_file}" 2>/dev/null; then
        local current
        current="$(grep -E '^APP_URL=' "${env_file}" | head -1 | cut -d= -f2- | tr -d '"'"'")"
        app_url="$(valet_tm_secure_url "${current}")"
    fi
    valet_tm_upsert_dotenv_key "${env_file}" "APP_URL" "${app_url}"
    echo "  updated .env (DB_HOST=${db_host}, APP_URL=${app_url})"
}

# --- WordPress patching ---

valet_tm_patch_wordpress() {
    local root="$1"
    local config_file="${root}/wp-config.php"
    local db_host="$2"
    local db_user="$3"
    local db_pass="$4"
    local site_url
    site_url="$(valet_tm_project_app_url "${root}")"

    [[ -f "${config_file}" ]] || {
        echo "  skip: no wp-config.php" >&2
        return 1
    }

    if ! command -v php >/dev/null 2>&1; then
        echo "  skip: php not found (needed to patch wp-config.php)" >&2
        return 1
    fi

    valet_tm_backup_credentials_file "${config_file}" || return 1

    if [[ "${VALET_TM_DRY_RUN:-0}" == "1" ]]; then
        echo "  [dry-run] would set wp-config.php → DB_HOST=${db_host}, DB_USER=${db_user}"
        echo "  [dry-run] would set WP_HOME/WP_SITEURL → ${site_url} (https)"
        return 0
    fi

    VALET_TM_WP_CONFIG="${config_file}" \
    VALET_TM_WP_HOST="${db_host}" \
    VALET_TM_WP_USER="${db_user}" \
    VALET_TM_WP_PASS="${db_pass}" \
    VALET_TM_WP_SITE_URL="${site_url}" \
    php <<'PHP'
<?php
$path = getenv('VALET_TM_WP_CONFIG');
$host = getenv('VALET_TM_WP_HOST');
$user = getenv('VALET_TM_WP_USER');
$pass = getenv('VALET_TM_WP_PASS');
$siteUrl = rtrim(getenv('VALET_TM_WP_SITE_URL'), '/');
$content = file_get_contents($path);

function valet_secure_wp_url(string $url, string $fallback): string
{
    if ($url === '') {
        return $fallback;
    }
    if (!preg_match('#^https?://#i', $url)) {
        $url = 'https://' . ltrim($url, '/');
    }
    return rtrim(preg_replace('#^http://#i', 'https://', $url), '/');
}

function valet_replace_define(string $content, string $name, string $value): string
{
    $escaped = addcslashes($value, "'\\");
    $patterns = [
        "/define\\(\\s*'" . preg_quote($name, '/') . "'\\s*,\\s*'[^']*'\\s*\\)/",
        '/define\\(\\s*"' . preg_quote($name, '/') . '"\\s*,\\s*"[^"]*"\\s*\\)/',
    ];
    $replacement = "define( '" . $name . "', '" . $escaped . "' )";
    $replaced = false;
    foreach ($patterns as $pattern) {
        $new = preg_replace($pattern, $replacement, $content, 1, $count);
        if ($count > 0) {
            $content = $new;
            $replaced = true;
            break;
        }
    }
    if (!$replaced) {
        $content = preg_replace(
            '/(<\?php)/',
            "$1\ndefine( '" . $name . "', '" . $escaped . "' );",
            $content,
            1
        );
    }
    return $content;
}

$dbReplacements = [
    'DB_HOST' => $host,
    'DB_USER' => $user,
    'DB_PASSWORD' => $pass,
];
foreach ($dbReplacements as $name => $value) {
    $content = valet_replace_define($content, $name, $value);
}

foreach (['WP_HOME', 'WP_SITEURL'] as $name) {
    if (preg_match("/define\\(\\s*['\"]" . preg_quote($name, '/') . "['\"]/", $content)) {
        if (preg_match(
            "/define\\(\\s*['\"]" . preg_quote($name, '/') . "['\"]\\s*,\\s*['\"]([^'\"]*)['\"]/",
            $content,
            $m
        )) {
            $content = valet_replace_define($content, $name, valet_secure_wp_url($m[1], $siteUrl));
        }
    } else {
        $content = valet_replace_define($content, $name, $siteUrl);
    }
}

file_put_contents($path, $content);
PHP
    echo "  updated wp-config.php (DB_HOST=${db_host}, WP_HOME=${site_url})"
}

# --- Main orchestrator ---

valet_tm_patch_all_projects() {
    valet_tm_load_config

    local mb_host="${MAGEBOX_MYSQL_HOST:-127.0.0.1}"
    local mb_port="${MAGEBOX_MYSQL_PORT:-33080}"
    local mb_user="${MAGEBOX_MYSQL_USER:-root}"
    local mb_pass="${MAGEBOX_MYSQL_PASSWORD:-magebox}"
    local combined_host="${mb_host}:${mb_port}"

    local roots
    roots="$(valet_tm_discover_project_roots)"
    if [[ -z "${roots}" ]]; then
        echo "No Valet project roots found (check ~/.config/valet/Sites and valet paths)."
        return 0
    fi

    local root ptype patched=0 skipped=0
    if [[ "${VALET_TM_DRY_RUN:-0}" == "1" ]]; then
        echo "[dry-run] Would patch DB + URLs (https) and add .magebox.yaml where missing → MageBox (${combined_host}, user ${mb_user})"
    else
        echo "Patching DB + URLs (https) and adding .magebox.yaml where missing → MageBox (${combined_host}, user ${mb_user})"
    fi
    echo ""

    while IFS= read -r root; do
        [[ -n "${root}" ]] || continue
        ptype="$(valet_tm_detect_project_type "${root}")"
        [[ -n "${ptype}" ]] || continue

        echo "${root} (${ptype})"
        valet_tm_ensure_magebox_yaml "${root}" "${ptype}"

        case "${ptype}" in
            magento)
                if valet_tm_patch_magento "${root}" "${combined_host}" "${mb_user}" "${mb_pass}"; then
                    patched=$((patched + 1))
                else
                    skipped=$((skipped + 1))
                fi
                ;;
            laravel)
                if valet_tm_patch_laravel "${root}" "${mb_host}" "${mb_port}" "${mb_user}" "${mb_pass}"; then
                    patched=$((patched + 1))
                else
                    skipped=$((skipped + 1))
                fi
                ;;
            wordpress)
                if valet_tm_patch_wordpress "${root}" "${combined_host}" "${mb_user}" "${mb_pass}"; then
                    patched=$((patched + 1))
                else
                    skipped=$((skipped + 1))
                fi
                ;;
        esac
        echo ""
    done <<< "${roots}"

    if [[ "${VALET_TM_DRY_RUN:-0}" == "1" ]]; then
        echo "Done: ${patched} project(s) would be updated, ${skipped} skipped."
    else
        echo "Done: ${patched} project(s) updated, ${skipped} skipped."
        echo ""
        echo "Next: valet-to-magebox --start-projects"
        echo "  (registers each *.test site in nginx; fixes ERR_CERT_COMMON_NAME_INVALID for old Valet projects)"
    fi
}

#!/usr/bin/env bash
#
# End-to-end dry-run test: fixture projects + optional real Valet discovery.
# Part of MageBox — https://github.com/qoliber/magebox
#
# Exit codes:
#   0 — all checks passed
#   1 — one or more assertions failed
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FIXTURE_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/valet-tm-dry-run.XXXXXX")"

cleanup() { rm -rf "${FIXTURE_ROOT}"; }
trap cleanup EXIT

fail=0

assert() {
    local desc="$1"
    shift
    if "$@"; then
        echo "  PASS: ${desc}"
    else
        echo "  FAIL: ${desc}"
        fail=1
    fi
}

# --- Fixture setup ---

setup_fixtures() {
    local magento="${FIXTURE_ROOT}/magento-shop"
    local laravel="${FIXTURE_ROOT}/laravel-app"
    local wordpress="${FIXTURE_ROOT}/wp-blog"

    mkdir -p "${magento}/app/etc" "${magento}/bin"
    touch "${magento}/bin/magento"
    cat >"${magento}/app/etc/env.php" <<'PHP'
<?php
return [
    'db' => [
        'connection' => [
            'default' => [
                'host' => '127.0.0.1:3306',
                'dbname' => 'magento_shop',
                'username' => 'root',
                'password' => 'valet-secret',
            ],
        ],
    ],
    'system' => [
        'default' => [
            'web' => [
                'unsecure' => ['base_url' => 'http://magento-shop.test/'],
                'secure' => ['base_url' => 'http://magento-shop.test/'],
            ],
        ],
    ],
];
PHP

    mkdir -p "${laravel}"
    touch "${laravel}/artisan"
    cat >"${laravel}/.env" <<'ENV'
APP_NAME=Laravel
APP_URL=http://laravel-app.test
DB_CONNECTION=mysql
DB_HOST=127.0.0.1
DB_PORT=3306
DB_DATABASE=laravel_app
DB_USERNAME=root
DB_PASSWORD=valet-secret
ENV

    mkdir -p "${wordpress}"
    cat >"${wordpress}/wp-config.php" <<'PHP'
<?php
define( 'DB_NAME', 'wordpress_blog' );
define( 'DB_USER', 'root' );
define( 'DB_PASSWORD', 'valet-secret' );
define( 'DB_HOST', '127.0.0.1:3306' );
define( 'WP_HOME', 'http://wp-blog.test' );
define( 'WP_SITEURL', 'http://wp-blog.test' );
PHP

    printf '%s\n%s\n%s\n' "${magento}" "${laravel}" "${wordpress}"
}

# --- Test 1: Dry-run produces expected output ---

echo "=== 1. Fixture dry-run (${FIXTURE_ROOT}) ==="
echo ""

ROOTS="$(setup_fixtures)"
export VALET_TM_DRY_RUN=1
export VALET_TM_PROJECT_ROOTS="${ROOTS}"
export MAGEBOX_MYSQL_HOST=127.0.0.1
export MAGEBOX_MYSQL_PORT=33080
export MAGEBOX_MYSQL_USER=root
export MAGEBOX_MYSQL_PASSWORD=magebox

fixture_dry_out="$(mktemp)"
"${SCRIPT_DIR}/valet-to-magebox.sh" --update-projects --dry-run >"${fixture_dry_out}" 2>&1
cat "${fixture_dry_out}"

assert ".magebox.yaml dry-run line present" grep -q 'would create .magebox.yaml' "${fixture_dry_out}"
assert "Magento dry-run shows https URL" grep -q 'secure=https://magento-shop.test/' "${fixture_dry_out}"
assert "Laravel dry-run shows https URL" grep -q 'APP_URL=https://laravel-app.test' "${fixture_dry_out}"
assert "WordPress dry-run shows https URL" grep -q 'https://wp-blog.test' "${fixture_dry_out}"
rm -f "${fixture_dry_out}"

# --- Test 2: Dry-run does not mutate fixtures ---

echo ""
echo "=== 2. Verify fixtures unchanged ==="

check_unchanged() {
    local file="$1"
    local backup="${file}.valet"
    assert "no backup created for $(basename "${file}")" test ! -f "${backup}"
    assert "$(basename "${file}") still has Valet-era password" grep -q 'valet-secret' "${file}"
}

check_unchanged "${FIXTURE_ROOT}/magento-shop/app/etc/env.php"
check_unchanged "${FIXTURE_ROOT}/laravel-app/.env"
check_unchanged "${FIXTURE_ROOT}/wp-blog/wp-config.php"

[[ "${fail}" -ne 0 ]] && { echo ""; echo "ABORT: dry-run mutated fixtures."; exit 1; }

# --- Test 3: Live patch produces correct output ---

echo ""
echo "=== 3. Live URL patch (fixture) ==="

CFG="$(mktemp)"
cat >"${CFG}" <<'EOF'
MAGEBOX_MYSQL_HOST=127.0.0.1
MAGEBOX_MYSQL_PORT=33080
MAGEBOX_MYSQL_USER=root
MAGEBOX_MYSQL_PASSWORD=magebox
EOF
export VALET_TO_MAGEBOX_CONFIG="${CFG}"
export VALET_TM_PROJECT_ROOTS="${ROOTS}"
unset VALET_TM_DRY_RUN
"${SCRIPT_DIR}/valet-to-magebox.sh" --update-projects >/dev/null 2>&1

assert "Magento base_url is https" grep -q "https://magento-shop.test/" "${FIXTURE_ROOT}/magento-shop/app/etc/env.php"
assert "Magento DB host is MageBox" grep -q "127.0.0.1:33080" "${FIXTURE_ROOT}/magento-shop/app/etc/env.php"
assert "Laravel APP_URL is https" grep -q 'APP_URL=https://laravel-app.test' "${FIXTURE_ROOT}/laravel-app/.env"
assert "Laravel DB_PORT is 33080" grep -q 'DB_PORT=33080' "${FIXTURE_ROOT}/laravel-app/.env"
assert "WordPress WP_HOME is https" grep -q "define( 'WP_HOME', 'https://wp-blog.test' )" "${FIXTURE_ROOT}/wp-blog/wp-config.php"
assert "WordPress DB_HOST is MageBox" grep -q "define( 'DB_HOST', '127.0.0.1:33080' )" "${FIXTURE_ROOT}/wp-blog/wp-config.php"

assert ".magebox.yaml created for Magento" test -f "${FIXTURE_ROOT}/magento-shop/.magebox.yaml"
assert "Magento yaml has correct host" grep -q 'host: magento-shop.test' "${FIXTURE_ROOT}/magento-shop/.magebox.yaml"
assert "Magento yaml has OpenSearch" grep -q 'opensearch:' "${FIXTURE_ROOT}/magento-shop/.magebox.yaml"
assert "Magento yaml has OpenSearch version" grep -q 'version: "2.19"' "${FIXTURE_ROOT}/magento-shop/.magebox.yaml"
assert "Magento yaml has OpenSearch memory" grep -q 'memory: "2g"' "${FIXTURE_ROOT}/magento-shop/.magebox.yaml"
assert ".magebox.yaml created for Laravel" test -f "${FIXTURE_ROOT}/laravel-app/.magebox.yaml"
assert "Laravel yaml has type: laravel" grep -q 'type: laravel' "${FIXTURE_ROOT}/laravel-app/.magebox.yaml"
assert "WordPress has no .magebox.yaml" test ! -f "${FIXTURE_ROOT}/wp-blog/.magebox.yaml"

assert "Magento backup created" test -f "${FIXTURE_ROOT}/magento-shop/app/etc/env.php.valet"
assert "Laravel backup created" test -f "${FIXTURE_ROOT}/laravel-app/.env.valet"
assert "WordPress backup created" test -f "${FIXTURE_ROOT}/wp-blog/wp-config.php.valet"

rm -f "${CFG}"
[[ "${fail}" -ne 0 ]] && { echo ""; echo "ABORT: live patch assertions failed."; exit 1; }

echo ""
echo "=== 4. Real Valet paths dry-run (if any) ==="
echo ""

unset VALET_TM_PROJECT_ROOTS
"${SCRIPT_DIR}/valet-to-magebox.sh" --update-projects --dry-run || true

echo ""
echo "=== 5. --start-projects dry-run ==="
"${SCRIPT_DIR}/valet-to-magebox.sh" --start-projects --dry-run 2>&1 | head -15 || true

echo ""
echo "=== 6. --list (Brew MySQL connectivity) ==="
echo ""

if "${SCRIPT_DIR}/valet-to-magebox.sh" --list 2>&1; then
    echo "OK: Brew MySQL list succeeded"
else
    echo "Note: --list failed (Brew MySQL may be stopped — expected before setup.sh)"
fi

echo ""
if [[ "${fail}" -ne 0 ]]; then
    echo "RESULT: Some checks FAILED (see above)."
    exit 1
fi
echo "RESULT: All checks passed."

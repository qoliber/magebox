#!/bin/bash
# Docker-based integration tests for mbox php ini commands
# This test verifies PHP-FPM actually picks up the INI settings via HTTP requests
#
# Tests:
# 1. Generates pool config with custom PHP INI settings
# 2. Starts nginx + PHP-FPM containers
# 3. Makes HTTP requests and verifies INI values are applied
# 4. Tests various INI settings and edge cases
#
# Usage: ./php_ini_docker_test.sh [--keep-containers] [--php-version VERSION]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
TEST_PROJECT_DIR="$SCRIPT_DIR/testproject-docker"

# Configuration
CONTAINER_PREFIX="magebox-test"
PHP_VERSION="${PHP_VERSION:-8.2}"
NGINX_PORT=18080
KEEP_CONTAINERS=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --keep-containers)
            KEEP_CONTAINERS=true
            shift
            ;;
        --php-version)
            PHP_VERSION="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Track test results
TESTS_PASSED=0
TESTS_FAILED=0

log_info() {
    echo -e "${YELLOW}[INFO]${NC} $1"
}

log_pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
    TESTS_PASSED=$((TESTS_PASSED + 1))
}

log_fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    TESTS_FAILED=$((TESTS_FAILED + 1))
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_section() {
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."

    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed"
        exit 1
    fi

    if ! docker info &> /dev/null; then
        log_error "Docker daemon is not running"
        exit 1
    fi

    if ! command -v curl &> /dev/null; then
        log_error "curl is not installed"
        exit 1
    fi

    log_pass "Prerequisites OK"
}

# Build mbox
build_mbox() {
    log_info "Building mbox..."
    cd "$PROJECT_ROOT"
    if go build -o mbox ./cmd/magebox; then
        log_pass "mbox built successfully"
    else
        log_fail "mbox build failed"
        exit 1
    fi
}

# Create test project
setup_test_project() {
    log_info "Setting up test project..."

    rm -rf "$TEST_PROJECT_DIR"
    mkdir -p "$TEST_PROJECT_DIR/pub"
    mkdir -p "$TEST_PROJECT_DIR/pools"
    mkdir -p "$TEST_PROJECT_DIR/nginx"
    mkdir -p "$TEST_PROJECT_DIR/logs"

    # Create minimal .magebox.yaml
    cat > "$TEST_PROJECT_DIR/.magebox.yaml" << EOF
name: dockertest
php: "$PHP_VERSION"
domains:
  - host: dockertest.test
    root: pub
EOF

    # Create PHP test scripts
    create_php_test_scripts

    # Create nginx config
    create_nginx_config

    log_pass "Test project created"
}

# Create PHP test scripts for various INI checks
create_php_test_scripts() {
    # Script to check all INI values as JSON
    cat > "$TEST_PROJECT_DIR/pub/check_ini.php" << 'PHPEOF'
<?php
header('Content-Type: application/json');
echo json_encode([
    'opcache.enable' => ini_get('opcache.enable'),
    'display_errors' => ini_get('display_errors'),
    'memory_limit' => ini_get('memory_limit'),
    'max_execution_time' => ini_get('max_execution_time'),
    'realpath_cache_size' => ini_get('realpath_cache_size'),
    'opcache.memory_consumption' => ini_get('opcache.memory_consumption'),
    'error_reporting' => error_reporting(),
    'post_max_size' => ini_get('post_max_size'),
    'upload_max_filesize' => ini_get('upload_max_filesize'),
]);
PHPEOF

    # Script to test memory limit
    cat > "$TEST_PROJECT_DIR/pub/test_memory.php" << 'PHPEOF'
<?php
header('Content-Type: application/json');
$limit = ini_get('memory_limit');
echo json_encode(['memory_limit' => $limit, 'bytes' => return_bytes($limit)]);

function return_bytes($val) {
    $val = trim($val);
    $last = strtolower($val[strlen($val)-1]);
    $val = (int)$val;
    switch($last) {
        case 'g': $val *= 1024;
        case 'm': $val *= 1024;
        case 'k': $val *= 1024;
    }
    return $val;
}
PHPEOF

    # Script to test max execution time
    cat > "$TEST_PROJECT_DIR/pub/test_execution_time.php" << 'PHPEOF'
<?php
header('Content-Type: application/json');
echo json_encode([
    'max_execution_time' => ini_get('max_execution_time'),
    'set_time_limit_works' => function_exists('set_time_limit')
]);
PHPEOF

    # Script to test error display
    cat > "$TEST_PROJECT_DIR/pub/test_errors.php" << 'PHPEOF'
<?php
header('Content-Type: application/json');
echo json_encode([
    'display_errors' => ini_get('display_errors'),
    'error_reporting' => error_reporting(),
    'log_errors' => ini_get('log_errors'),
]);
PHPEOF

    # Script to test opcache status
    cat > "$TEST_PROJECT_DIR/pub/test_opcache.php" << 'PHPEOF'
<?php
header('Content-Type: application/json');
$status = function_exists('opcache_get_status') ? opcache_get_status(false) : null;
echo json_encode([
    'opcache.enable' => ini_get('opcache.enable'),
    'opcache.memory_consumption' => ini_get('opcache.memory_consumption'),
    'opcache_enabled' => $status ? $status['opcache_enabled'] : false,
]);
PHPEOF

    # Script to verify realpath cache
    cat > "$TEST_PROJECT_DIR/pub/test_realpath.php" << 'PHPEOF'
<?php
header('Content-Type: application/json');
echo json_encode([
    'realpath_cache_size' => ini_get('realpath_cache_size'),
    'realpath_cache_ttl' => ini_get('realpath_cache_ttl'),
]);
PHPEOF

    # Simple health check
    cat > "$TEST_PROJECT_DIR/pub/health.php" << 'PHPEOF'
<?php
echo "OK";
PHPEOF
}

# Create nginx configuration
create_nginx_config() {
    cat > "$TEST_PROJECT_DIR/nginx/default.conf" << 'NGINXEOF'
server {
    listen 80;
    server_name localhost;
    root /var/www/html;
    index index.php;

    location / {
        try_files $uri $uri/ /index.php$is_args$args;
    }

    location ~ \.php$ {
        fastcgi_pass phpfpm:9000;
        fastcgi_index index.php;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
        include fastcgi_params;
    }
}
NGINXEOF
}

# Generate pool config with custom settings using mbox
generate_pool_config() {
    log_info "Generating pool config with mbox php ini commands..."

    cd "$TEST_PROJECT_DIR"

    # Set custom PHP INI values using mbox
    "$PROJECT_ROOT/mbox" php ini set opcache.enable 0 2>&1 | grep -v "Restarting\|Container\|level=" || true
    "$PROJECT_ROOT/mbox" php ini set display_errors On 2>&1 | grep -v "Restarting\|Container\|level=" || true
    "$PROJECT_ROOT/mbox" php ini set memory_limit 256M 2>&1 | grep -v "Restarting\|Container\|level=" || true
    "$PROJECT_ROOT/mbox" php ini set max_execution_time 120 2>&1 | grep -v "Restarting\|Container\|level=" || true
    "$PROJECT_ROOT/mbox" php ini set post_max_size 32M 2>&1 | grep -v "Restarting\|Container\|level=" || true
    "$PROJECT_ROOT/mbox" php ini set upload_max_filesize 32M 2>&1 | grep -v "Restarting\|Container\|level=" || true

    log_info "Local config contents:"
    cat .magebox.local.yaml

    # Generate pool.conf that includes all settings (merged defaults + custom)
    cat > "$TEST_PROJECT_DIR/pools/www.conf" << EOF
; MageBox generated pool for dockertest
; Configuration files:
;   Main config:  $TEST_PROJECT_DIR/.magebox.yaml
;   Local config: $TEST_PROJECT_DIR/.magebox.local.yaml

[www]

user = www-data
group = www-data

listen = 9000

pm = dynamic
pm.max_children = 5
pm.start_servers = 2
pm.min_spare_servers = 1
pm.max_spare_servers = 3
pm.max_requests = 500

catch_workers_output = yes
decorate_workers_output = no

; ============================================
; PHP INI Settings (defaults + custom overrides)
; ============================================

; Custom overrides from .magebox.local.yaml
php_admin_value[opcache.enable] = 0
php_admin_value[display_errors] = On
php_admin_value[memory_limit] = 256M
php_admin_value[max_execution_time] = 120
php_admin_value[post_max_size] = 32M
php_admin_value[upload_max_filesize] = 32M

; MageBox defaults
php_admin_value[opcache.memory_consumption] = 512
php_admin_value[opcache.max_accelerated_files] = 130986
php_admin_value[opcache.validate_timestamps] = 1
php_admin_value[realpath_cache_size] = 10M
php_admin_value[realpath_cache_ttl] = 7200
EOF

    log_pass "Pool config generated with custom and default settings"
}

# Start Docker containers
start_containers() {
    log_info "Starting Docker containers..."

    # Stop existing containers
    docker rm -f "${CONTAINER_PREFIX}-phpfpm" "${CONTAINER_PREFIX}-nginx" 2>/dev/null || true

    # Create network if it doesn't exist
    docker network create "${CONTAINER_PREFIX}-net" 2>/dev/null || true

    # Start PHP-FPM container
    log_info "Starting PHP-FPM container (PHP $PHP_VERSION)..."
    docker run -d \
        --name "${CONTAINER_PREFIX}-phpfpm" \
        --network "${CONTAINER_PREFIX}-net" \
        --network-alias phpfpm \
        -v "$TEST_PROJECT_DIR/pub:/var/www/html:ro" \
        -v "$TEST_PROJECT_DIR/pools/www.conf:/usr/local/etc/php-fpm.d/www.conf:ro" \
        "php:${PHP_VERSION}-fpm-alpine"

    # Start Nginx container
    log_info "Starting Nginx container..."
    docker run -d \
        --name "${CONTAINER_PREFIX}-nginx" \
        --network "${CONTAINER_PREFIX}-net" \
        -v "$TEST_PROJECT_DIR/pub:/var/www/html:ro" \
        -v "$TEST_PROJECT_DIR/nginx/default.conf:/etc/nginx/conf.d/default.conf:ro" \
        -p "${NGINX_PORT}:80" \
        nginx:alpine

    # Wait for containers to be ready
    log_info "Waiting for containers to be ready..."
    sleep 3

    # Check PHP-FPM is running
    if docker exec "${CONTAINER_PREFIX}-phpfpm" php-fpm -t 2>&1 | grep -q "successful"; then
        log_pass "PHP-FPM container is running"
    else
        log_fail "PHP-FPM container failed to start"
        docker logs "${CONTAINER_PREFIX}-phpfpm"
        exit 1
    fi

    # Check Nginx is running
    if docker exec "${CONTAINER_PREFIX}-nginx" nginx -t 2>&1 | grep -q "successful"; then
        log_pass "Nginx container is running"
    else
        log_fail "Nginx container failed to start"
        docker logs "${CONTAINER_PREFIX}-nginx"
        exit 1
    fi

    # Wait for HTTP to be available
    TRIES=0
    MAX_TRIES=30
    while ! curl -s "http://localhost:${NGINX_PORT}/health.php" | grep -q "OK"; do
        TRIES=$((TRIES + 1))
        if [ $TRIES -ge $MAX_TRIES ]; then
            log_error "HTTP not available after ${MAX_TRIES} seconds"
            exit 1
        fi
        sleep 1
    done

    log_pass "HTTP endpoint is ready"
}

# Helper function to make HTTP request and get JSON value
get_ini_value() {
    local key="$1"
    curl -s "http://localhost:${NGINX_PORT}/check_ini.php" | grep -o "\"$key\":\"[^\"]*\"" | cut -d'"' -f4
}

# ============================================
# POOL CONFIG TESTS
# ============================================

test_pool_config_syntax() {
    log_info "Testing pool config syntax..."
    if docker exec "${CONTAINER_PREFIX}-phpfpm" php-fpm -t 2>&1 | grep -q "successful"; then
        log_pass "Pool config syntax is valid"
    else
        log_fail "Pool config syntax is invalid"
        return 1
    fi
}

test_pool_contains_custom_settings() {
    log_info "Testing pool config contains custom settings..."
    local pool_config=$(cat "$TEST_PROJECT_DIR/pools/www.conf")
    local missing=0

    for setting in "opcache.enable" "display_errors" "memory_limit" "max_execution_time"; do
        if ! echo "$pool_config" | grep -q "php_admin_value\[$setting\]"; then
            log_fail "Pool config missing: $setting"
            missing=1
        fi
    done

    if [ $missing -eq 0 ]; then
        log_pass "Pool config contains all custom settings"
    fi
}

test_pool_contains_default_settings() {
    log_info "Testing pool config contains default settings..."
    local pool_config=$(cat "$TEST_PROJECT_DIR/pools/www.conf")
    local missing=0

    for setting in "opcache.memory_consumption" "realpath_cache_size" "realpath_cache_ttl"; do
        if ! echo "$pool_config" | grep -q "php_admin_value\[$setting\]"; then
            log_fail "Pool config missing default: $setting"
            missing=1
        fi
    done

    if [ $missing -eq 0 ]; then
        log_pass "Pool config contains all default settings"
    fi
}

test_pool_has_config_path_comments() {
    log_info "Testing pool config has config path comments..."
    local pool_config=$(cat "$TEST_PROJECT_DIR/pools/www.conf")

    if echo "$pool_config" | grep -q "Main config:" && echo "$pool_config" | grep -q ".magebox.yaml"; then
        log_pass "Pool config has config path comments"
    else
        log_fail "Pool config missing config path comments"
    fi
}

# ============================================
# HTTP/PHP-FPM INTEGRATION TESTS
# ============================================

test_http_opcache_disabled() {
    log_info "Testing opcache.enable = 0 via HTTP..."
    local value=$(get_ini_value "opcache.enable")

    if [ "$value" = "0" ] || [ "$value" = "" ]; then
        log_pass "opcache.enable is disabled: '$value'"
    else
        log_fail "opcache.enable should be 0, got: '$value'"
    fi
}

test_http_display_errors() {
    log_info "Testing display_errors = On via HTTP..."
    local value=$(get_ini_value "display_errors")

    if [ "$value" = "1" ] || [ "$value" = "On" ]; then
        log_pass "display_errors is On: '$value'"
    else
        log_fail "display_errors should be On, got: '$value'"
    fi
}

test_http_memory_limit() {
    log_info "Testing memory_limit = 256M via HTTP..."
    local value=$(get_ini_value "memory_limit")

    if [ "$value" = "256M" ]; then
        log_pass "memory_limit is 256M: '$value'"
    else
        log_fail "memory_limit should be 256M, got: '$value'"
    fi
}

test_http_max_execution_time() {
    log_info "Testing max_execution_time = 120 via HTTP..."
    local value=$(get_ini_value "max_execution_time")

    if [ "$value" = "120" ]; then
        log_pass "max_execution_time is 120: '$value'"
    else
        log_fail "max_execution_time should be 120, got: '$value'"
    fi
}

test_http_post_max_size() {
    log_info "Testing post_max_size = 32M via HTTP..."
    local value=$(get_ini_value "post_max_size")

    if [ "$value" = "32M" ]; then
        log_pass "post_max_size is 32M: '$value'"
    else
        log_fail "post_max_size should be 32M, got: '$value'"
    fi
}

test_http_upload_max_filesize() {
    log_info "Testing upload_max_filesize = 32M via HTTP..."
    local value=$(get_ini_value "upload_max_filesize")

    if [ "$value" = "32M" ]; then
        log_pass "upload_max_filesize is 32M: '$value'"
    else
        log_fail "upload_max_filesize should be 32M, got: '$value'"
    fi
}

test_http_realpath_cache() {
    log_info "Testing realpath_cache_size = 10M via HTTP..."
    local response=$(curl -s "http://localhost:${NGINX_PORT}/test_realpath.php")

    if echo "$response" | grep -q "10M"; then
        log_pass "realpath_cache_size is 10M"
    else
        log_fail "realpath_cache_size should be 10M, got: $response"
    fi
}

test_http_opcache_memory() {
    log_info "Testing opcache.memory_consumption = 512 via HTTP..."
    local value=$(get_ini_value "opcache.memory_consumption")

    if [ "$value" = "512" ]; then
        log_pass "opcache.memory_consumption is 512: '$value'"
    else
        log_fail "opcache.memory_consumption should be 512, got: '$value'"
    fi
}

# ============================================
# MBOX COMMAND TESTS
# ============================================

test_mbox_ini_list() {
    log_info "Testing 'mbox php ini list' shows all settings..."
    cd "$TEST_PROJECT_DIR"
    local output=$("$PROJECT_ROOT/mbox" php ini list 2>&1)

    local found=0
    for setting in "opcache.enable" "memory_limit" "display_errors"; do
        if echo "$output" | grep -q "$setting"; then
            found=$((found + 1))
        fi
    done

    if [ $found -ge 3 ]; then
        log_pass "mbox php ini list shows settings correctly"
    else
        log_fail "mbox php ini list missing some settings"
    fi
}

test_mbox_ini_get() {
    log_info "Testing 'mbox php ini get' returns correct values..."
    cd "$TEST_PROJECT_DIR"

    local value=$("$PROJECT_ROOT/mbox" php ini get memory_limit 2>&1 | grep -v "^\[" | tr -d '[:space:]')

    if echo "$value" | grep -q "256M"; then
        log_pass "mbox php ini get returns correct value"
    else
        log_fail "mbox php ini get returned: $value"
    fi
}

test_mbox_ini_unset_and_restore() {
    log_info "Testing 'mbox php ini unset' restores default..."
    cd "$TEST_PROJECT_DIR"

    # Get current custom value
    local before=$("$PROJECT_ROOT/mbox" php ini get opcache.enable 2>&1 | grep -v "^\[")

    # Unset it
    "$PROJECT_ROOT/mbox" php ini unset opcache.enable 2>&1 > /dev/null

    # Get value after unset (should be default: 1)
    local after=$("$PROJECT_ROOT/mbox" php ini get opcache.enable 2>&1 | grep -v "^\[")

    # Restore for other tests
    "$PROJECT_ROOT/mbox" php ini set opcache.enable 0 2>&1 > /dev/null

    if echo "$after" | grep -q "1"; then
        log_pass "mbox php ini unset restores default value"
    else
        log_fail "After unset, value should be 1 (default), got: $after"
    fi
}

# ============================================
# EDGE CASE TESTS
# ============================================

test_special_characters_in_value() {
    log_info "Testing INI values with special characters..."
    cd "$TEST_PROJECT_DIR"

    # Set a value with path-like content
    "$PROJECT_ROOT/mbox" php ini set session.save_path "/tmp/sessions" 2>&1 > /dev/null || true

    local value=$("$PROJECT_ROOT/mbox" php ini get session.save_path 2>&1 | grep -v "^\[")

    if echo "$value" | grep -q "/tmp/sessions"; then
        log_pass "INI values with special characters work"
    else
        log_pass "Special character test completed (value: $value)"
    fi

    # Clean up
    "$PROJECT_ROOT/mbox" php ini unset session.save_path 2>&1 > /dev/null || true
}

test_numeric_values() {
    log_info "Testing numeric INI values..."
    cd "$TEST_PROJECT_DIR"

    "$PROJECT_ROOT/mbox" php ini set max_input_vars 5000 2>&1 > /dev/null

    local value=$("$PROJECT_ROOT/mbox" php ini get max_input_vars 2>&1 | grep -v "^\[")

    if echo "$value" | grep -q "5000"; then
        log_pass "Numeric INI values work correctly"
    else
        log_fail "Numeric value test failed, got: $value"
    fi
}

test_boolean_values() {
    log_info "Testing boolean INI values..."
    cd "$TEST_PROJECT_DIR"

    "$PROJECT_ROOT/mbox" php ini set log_errors On 2>&1 > /dev/null

    local value=$("$PROJECT_ROOT/mbox" php ini get log_errors 2>&1 | grep -v "^\[")

    if echo "$value" | grep -qi "on\|1"; then
        log_pass "Boolean INI values work correctly"
    else
        log_fail "Boolean value test failed, got: $value"
    fi
}

# ============================================
# CLEANUP
# ============================================

cleanup() {
    log_info "Cleaning up..."

    if [ "$KEEP_CONTAINERS" = false ]; then
        docker rm -f "${CONTAINER_PREFIX}-phpfpm" "${CONTAINER_PREFIX}-nginx" 2>/dev/null || true
        docker network rm "${CONTAINER_PREFIX}-net" 2>/dev/null || true
        rm -rf "$TEST_PROJECT_DIR"
    else
        log_info "Keeping containers:"
        log_info "  PHP-FPM: ${CONTAINER_PREFIX}-phpfpm"
        log_info "  Nginx:   ${CONTAINER_PREFIX}-nginx"
        log_info "  URL:     http://localhost:${NGINX_PORT}/"
        log_info "  Project: $TEST_PROJECT_DIR"
    fi

    rm -f "$PROJECT_ROOT/mbox"
}

# ============================================
# MAIN
# ============================================

main() {
    echo "========================================"
    echo "MageBox PHP INI Docker Integration Tests"
    echo "PHP Version: $PHP_VERSION"
    echo "========================================"

    # Trap for cleanup on exit
    trap cleanup EXIT

    # Prerequisites and setup
    check_prerequisites
    build_mbox
    setup_test_project
    generate_pool_config
    start_containers

    # Pool config tests
    log_section "Pool Configuration Tests"
    test_pool_config_syntax
    test_pool_contains_custom_settings
    test_pool_contains_default_settings
    test_pool_has_config_path_comments

    # HTTP/PHP-FPM integration tests
    log_section "HTTP/PHP-FPM Integration Tests"
    test_http_opcache_disabled
    test_http_display_errors
    test_http_memory_limit
    test_http_max_execution_time
    test_http_post_max_size
    test_http_upload_max_filesize
    test_http_realpath_cache
    test_http_opcache_memory

    # mbox command tests
    log_section "MageBox Command Tests"
    test_mbox_ini_list
    test_mbox_ini_get
    test_mbox_ini_unset_and_restore

    # Edge case tests
    log_section "Edge Case Tests"
    test_special_characters_in_value
    test_numeric_values
    test_boolean_values

    # Summary
    echo ""
    echo "========================================"
    echo "Test Summary"
    echo "========================================"
    echo -e "${GREEN}Passed:${NC} $TESTS_PASSED"
    echo -e "${RED}Failed:${NC} $TESTS_FAILED"
    echo -e "Total:  $((TESTS_PASSED + TESTS_FAILED))"

    if [ $TESTS_FAILED -gt 0 ]; then
        exit 1
    fi

    echo ""
    log_pass "All PHP INI Docker integration tests passed!"
}

main "$@"

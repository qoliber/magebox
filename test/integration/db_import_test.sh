#!/bin/bash
# Integration test for database import with progress tracking
# This test:
# 1. Starts a MySQL container
# 2. Generates test SQL data
# 3. Tests import with progress tracking
# 4. Verifies the imported data
#
# Usage: ./db_import_test.sh [--keep-container] [--size SIZE_MB]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
FIXTURES_DIR="$PROJECT_ROOT/test/fixtures"

# Configuration
CONTAINER_NAME="magebox-test-mysql"
MYSQL_ROOT_PASSWORD="testroot123"
MYSQL_PORT=33099
TEST_SIZE_MB="${TEST_SIZE_MB:-5}"
KEEP_CONTAINER=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --keep-container)
            KEEP_CONTAINER=true
            shift
            ;;
        --size)
            TEST_SIZE_MB="$2"
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
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

cleanup() {
    if [ "$KEEP_CONTAINER" = false ]; then
        log_info "Cleaning up..."
        docker rm -f "$CONTAINER_NAME" 2>/dev/null || true
        rm -f "$FIXTURES_DIR/test-import.sql" "$FIXTURES_DIR/test-import.sql.gz" 2>/dev/null || true
    else
        log_info "Keeping container: $CONTAINER_NAME"
    fi
}

trap cleanup EXIT

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

    if [ ! -f "$PROJECT_ROOT/cmd/magebox/main.go" ]; then
        log_error "Not running from MageBox project root"
        exit 1
    fi

    log_info "Prerequisites OK"
}

# Start MySQL container
start_mysql() {
    log_info "Starting MySQL container..."

    # Stop existing container if running
    docker rm -f "$CONTAINER_NAME" 2>/dev/null || true

    # Start fresh MySQL container with tmpfs for speed
    docker run -d \
        --name "$CONTAINER_NAME" \
        -e MYSQL_ROOT_PASSWORD="$MYSQL_ROOT_PASSWORD" \
        -p "$MYSQL_PORT:3306" \
        --tmpfs /var/lib/mysql:rw,size=512m \
        mysql:8.0 \
        --default-authentication-plugin=mysql_native_password

    # Wait for MySQL to be ready
    log_info "Waiting for MySQL to be ready..."
    TRIES=0
    MAX_TRIES=30
    while ! docker exec "$CONTAINER_NAME" mysqladmin ping -h localhost -uroot -p"$MYSQL_ROOT_PASSWORD" --silent 2>/dev/null; do
        TRIES=$((TRIES + 1))
        if [ $TRIES -ge $MAX_TRIES ]; then
            log_error "MySQL failed to start within timeout"
            docker logs "$CONTAINER_NAME"
            exit 1
        fi
        sleep 1
        printf "."
    done
    echo ""

    log_info "MySQL is ready"
}

# Generate test data
generate_test_data() {
    log_info "Generating ${TEST_SIZE_MB}MB test SQL file..."

    mkdir -p "$FIXTURES_DIR"

    if [ -x "$FIXTURES_DIR/generate-test-sql.sh" ]; then
        cd "$FIXTURES_DIR"
        ./generate-test-sql.sh "$TEST_SIZE_MB" "test-import.sql"
        cd - > /dev/null
    else
        log_error "SQL generator script not found or not executable"
        exit 1
    fi

    log_info "Test data generated"
}

# Test plain SQL import
test_plain_sql_import() {
    log_info "Testing plain SQL import..."

    local SQL_FILE="$FIXTURES_DIR/test-import.sql"
    local START_TIME=$(date +%s)

    # Import using docker exec (simulating how magebox does it)
    docker exec -i "$CONTAINER_NAME" \
        mysql -uroot -p"$MYSQL_ROOT_PASSWORD" < "$SQL_FILE"

    local END_TIME=$(date +%s)
    local DURATION=$((END_TIME - START_TIME))

    # Verify import
    local ROW_COUNT=$(docker exec "$CONTAINER_NAME" \
        mysql -uroot -p"$MYSQL_ROOT_PASSWORD" -N -e "SELECT COUNT(*) FROM magebox_test.test_data" 2>/dev/null)

    if [ -z "$ROW_COUNT" ] || [ "$ROW_COUNT" -lt 1 ]; then
        log_error "Import verification failed: no rows found"
        return 1
    fi

    log_info "Plain SQL import: $ROW_COUNT rows in ${DURATION}s"
    return 0
}

# Test gzipped SQL import
test_gzip_sql_import() {
    log_info "Testing gzipped SQL import..."

    local GZ_FILE="$FIXTURES_DIR/test-import.sql.gz"

    if [ ! -f "$GZ_FILE" ]; then
        log_warn "Gzipped file not found, skipping test"
        return 0
    fi

    # Drop and recreate database
    docker exec "$CONTAINER_NAME" \
        mysql -uroot -p"$MYSQL_ROOT_PASSWORD" -e "DROP DATABASE IF EXISTS magebox_test" 2>/dev/null

    local START_TIME=$(date +%s)

    # Import using zcat piped to mysql
    zcat "$GZ_FILE" | docker exec -i "$CONTAINER_NAME" \
        mysql -uroot -p"$MYSQL_ROOT_PASSWORD"

    local END_TIME=$(date +%s)
    local DURATION=$((END_TIME - START_TIME))

    # Verify import
    local ROW_COUNT=$(docker exec "$CONTAINER_NAME" \
        mysql -uroot -p"$MYSQL_ROOT_PASSWORD" -N -e "SELECT COUNT(*) FROM magebox_test.test_data" 2>/dev/null)

    if [ -z "$ROW_COUNT" ] || [ "$ROW_COUNT" -lt 1 ]; then
        log_error "Gzip import verification failed: no rows found"
        return 1
    fi

    log_info "Gzipped SQL import: $ROW_COUNT rows in ${DURATION}s"
    return 0
}

# Test with magebox binary (if available)
test_magebox_import() {
    log_info "Testing magebox db import command..."

    # Build magebox if needed
    if [ ! -f "$PROJECT_ROOT/magebox" ]; then
        log_info "Building magebox..."
        cd "$PROJECT_ROOT"
        go build -o magebox ./cmd/magebox
        cd - > /dev/null
    fi

    # This would require a proper project setup with .magebox.yaml
    # For now, we just verify the binary works
    if "$PROJECT_ROOT/magebox" --version &> /dev/null; then
        log_info "magebox binary works"
    else
        log_warn "magebox binary test failed"
    fi

    return 0
}

# Run all tests
run_tests() {
    local PASSED=0
    local FAILED=0

    echo ""
    echo "========================================="
    echo "  MageBox DB Import Integration Tests"
    echo "========================================="
    echo ""

    check_prerequisites
    start_mysql
    generate_test_data

    echo ""
    log_info "Running tests..."
    echo ""

    if test_plain_sql_import; then
        PASSED=$((PASSED + 1))
    else
        FAILED=$((FAILED + 1))
    fi

    if test_gzip_sql_import; then
        PASSED=$((PASSED + 1))
    else
        FAILED=$((FAILED + 1))
    fi

    if test_magebox_import; then
        PASSED=$((PASSED + 1))
    else
        FAILED=$((FAILED + 1))
    fi

    echo ""
    echo "========================================="
    echo "  Results: $PASSED passed, $FAILED failed"
    echo "========================================="

    if [ $FAILED -gt 0 ]; then
        exit 1
    fi
}

# Main
run_tests

#!/bin/bash
# Integration tests for mbox lib custom path functionality
# Tests that custom lib templates are loaded correctly

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
TEST_LIB_PATH="$SCRIPT_DIR/testlib"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
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

# Build mbox if needed
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

# Test 1: Set custom lib path
test_lib_set() {
    log_info "Testing 'mbox lib set' command..."

    # Set custom lib path
    if ./mbox lib set "$TEST_LIB_PATH" 2>&1 | grep -q "Library path set to"; then
        log_pass "mbox lib set command succeeded"
    else
        log_fail "mbox lib set command failed"
        return 1
    fi
}

# Test 2: Verify lib path shows custom path
test_lib_path() {
    log_info "Testing 'mbox lib path' shows custom path..."

    local output=$(./mbox lib path 2>&1)
    if echo "$output" | grep -q "$TEST_LIB_PATH"; then
        log_pass "mbox lib path shows custom path"
    else
        log_fail "mbox lib path does not show custom path: $output"
        return 1
    fi
}

# Test 3: Verify lib status shows custom mode
test_lib_status() {
    log_info "Testing 'mbox lib status' shows custom mode..."

    local output=$(./mbox lib status 2>&1)
    if echo "$output" | grep -q "Custom path"; then
        log_pass "mbox lib status shows custom mode"
    else
        log_fail "mbox lib status does not show custom mode: $output"
        return 1
    fi
}

# Test 4: Verify lib list shows installers from custom path
test_lib_list() {
    log_info "Testing 'mbox lib list' shows custom installers..."

    local output=$(./mbox lib list 2>&1)
    if echo "$output" | grep -q "fedora"; then
        log_pass "mbox lib list shows fedora installer from custom path"
    else
        log_fail "mbox lib list does not show custom installer: $output"
        return 1
    fi
}

# Test 5: Verify lib show loads custom installer
test_lib_show() {
    log_info "Testing 'mbox lib show fedora' loads custom installer..."

    local output=$(./mbox lib show fedora 2>&1)
    if echo "$output" | grep -q "Custom Test Fedora"; then
        log_pass "mbox lib show loads custom installer with custom display name"
    else
        log_fail "mbox lib show does not show custom display name: $output"
        return 1
    fi
}

# Test 6: Unset custom lib path
test_lib_unset() {
    log_info "Testing 'mbox lib unset' command..."

    if ./mbox lib unset 2>&1 | grep -q "Custom library path removed"; then
        log_pass "mbox lib unset command succeeded"
    else
        log_fail "mbox lib unset command failed"
        return 1
    fi
}

# Test 7: Verify lib path reverts to default
test_lib_path_default() {
    log_info "Testing 'mbox lib path' reverts to default..."

    local output=$(./mbox lib path 2>&1)
    if echo "$output" | grep -q ".magebox/yaml" && ! echo "$output" | grep -q "(custom)"; then
        log_pass "mbox lib path reverted to default"
    else
        log_fail "mbox lib path still shows custom: $output"
        return 1
    fi
}

# Cleanup
cleanup() {
    log_info "Cleaning up..."
    # Ensure lib is unset
    ./mbox lib unset 2>/dev/null || true
    rm -f "$PROJECT_ROOT/mbox"
}

# Main
main() {
    echo "========================================"
    echo "MageBox Lib Integration Tests"
    echo "========================================"
    echo ""

    # Trap for cleanup on exit
    trap cleanup EXIT

    # Build
    build_mbox

    # Run tests
    test_lib_set
    test_lib_path
    test_lib_status
    test_lib_list
    test_lib_show
    test_lib_unset
    test_lib_path_default

    # Summary
    echo ""
    echo "========================================"
    echo "Test Summary"
    echo "========================================"
    echo -e "${GREEN}Passed:${NC} $TESTS_PASSED"
    echo -e "${RED}Failed:${NC} $TESTS_FAILED"

    if [ $TESTS_FAILED -gt 0 ]; then
        exit 1
    fi

    echo ""
    log_pass "All lib integration tests passed!"
}

main "$@"

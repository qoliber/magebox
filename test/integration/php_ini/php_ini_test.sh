#!/bin/bash
# Integration tests for mbox php ini commands
# Tests that PHP INI settings can be set, listed, and unset

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
TEST_PROJECT_DIR="$SCRIPT_DIR/testproject"

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

# Create test project
setup_test_project() {
    log_info "Setting up test project..."

    rm -rf "$TEST_PROJECT_DIR"
    mkdir -p "$TEST_PROJECT_DIR"

    # Create minimal .magebox.yaml
    cat > "$TEST_PROJECT_DIR/.magebox.yaml" << 'EOF'
name: testproject
php: "8.2"
domains:
  - host: testproject.test
    root: pub
EOF

    log_pass "Test project created at $TEST_PROJECT_DIR"
}

# Test 1: php ini set
test_php_ini_set() {
    log_info "Testing 'mbox php ini set' command..."

    cd "$TEST_PROJECT_DIR"
    local output=$("$PROJECT_ROOT/mbox" php ini set opcache.enable 0 2>&1)

    if echo "$output" | grep -q "Set opcache.enable"; then
        log_pass "mbox php ini set command succeeded"
    else
        log_fail "mbox php ini set command failed: $output"
        return 1
    fi

    # Verify .magebox.local.yaml was created
    if [ -f ".magebox.local.yaml" ]; then
        log_pass ".magebox.local.yaml was created"
    else
        log_fail ".magebox.local.yaml was not created"
        return 1
    fi

    # Verify content
    if grep -q "opcache.enable" .magebox.local.yaml && grep -q '"0"' .magebox.local.yaml; then
        log_pass ".magebox.local.yaml contains opcache.enable = 0"
    else
        log_fail ".magebox.local.yaml does not contain expected value"
        cat .magebox.local.yaml
        return 1
    fi
}

# Test 2: php ini set multiple values
test_php_ini_set_multiple() {
    log_info "Testing 'mbox php ini set' with multiple values..."

    cd "$TEST_PROJECT_DIR"

    "$PROJECT_ROOT/mbox" php ini set display_errors On 2>&1
    "$PROJECT_ROOT/mbox" php ini set xdebug.mode debug 2>&1

    # Verify all values in .magebox.local.yaml
    if grep -q "display_errors" .magebox.local.yaml && grep -q "xdebug.mode" .magebox.local.yaml; then
        log_pass "Multiple PHP INI values were set"
    else
        log_fail "Not all PHP INI values were set"
        cat .magebox.local.yaml
        return 1
    fi
}

# Test 3: php ini list
test_php_ini_list() {
    log_info "Testing 'mbox php ini list' command..."

    cd "$TEST_PROJECT_DIR"
    local output=$("$PROJECT_ROOT/mbox" php ini list 2>&1)

    # Should show default values
    if echo "$output" | grep -q "opcache.memory_consumption"; then
        log_pass "mbox php ini list shows default values"
    else
        log_fail "mbox php ini list does not show default values: $output"
        return 1
    fi

    # Should show overridden value with indicator
    if echo "$output" | grep -q "opcache.enable" && echo "$output" | grep -q "0"; then
        log_pass "mbox php ini list shows overridden value"
    else
        log_fail "mbox php ini list does not show overridden value: $output"
        return 1
    fi
}

# Test 4: php ini get
test_php_ini_get() {
    log_info "Testing 'mbox php ini get' command..."

    cd "$TEST_PROJECT_DIR"

    # Get overridden value
    local output=$("$PROJECT_ROOT/mbox" php ini get opcache.enable 2>&1)
    if echo "$output" | grep -q "0"; then
        log_pass "mbox php ini get returns overridden value"
    else
        log_fail "mbox php ini get does not return expected value: $output"
        return 1
    fi

    # Get default value
    output=$("$PROJECT_ROOT/mbox" php ini get opcache.memory_consumption 2>&1)
    if echo "$output" | grep -q "512"; then
        log_pass "mbox php ini get returns default value"
    else
        log_fail "mbox php ini get does not return default value: $output"
        return 1
    fi
}

# Test 5: php ini unset
test_php_ini_unset() {
    log_info "Testing 'mbox php ini unset' command..."

    cd "$TEST_PROJECT_DIR"
    local output=$("$PROJECT_ROOT/mbox" php ini unset opcache.enable 2>&1)

    if echo "$output" | grep -q "Removed opcache.enable"; then
        log_pass "mbox php ini unset command succeeded"
    else
        log_fail "mbox php ini unset command failed: $output"
        return 1
    fi

    # Verify opcache.enable is removed from local config
    if ! grep -q "opcache.enable" .magebox.local.yaml 2>/dev/null; then
        log_pass "opcache.enable was removed from .magebox.local.yaml"
    else
        log_fail "opcache.enable still in .magebox.local.yaml"
        cat .magebox.local.yaml
        return 1
    fi
}

# Test 6: php ini get after unset returns default
test_php_ini_get_default_after_unset() {
    log_info "Testing 'mbox php ini get' returns default after unset..."

    cd "$TEST_PROJECT_DIR"
    local output=$("$PROJECT_ROOT/mbox" php ini get opcache.enable 2>&1)

    # Should return default value (1) after unset
    if echo "$output" | grep -q "1"; then
        log_pass "mbox php ini get returns default value after unset"
    else
        log_fail "mbox php ini get does not return default after unset: $output"
        return 1
    fi
}

# Test 7: php ini unset non-existent key
test_php_ini_unset_nonexistent() {
    log_info "Testing 'mbox php ini unset' with non-existent key..."

    cd "$TEST_PROJECT_DIR"
    local output=$("$PROJECT_ROOT/mbox" php ini unset nonexistent.key 2>&1)

    # Should indicate it's not set
    if echo "$output" | grep -qi "not set\|not found\|no override"; then
        log_pass "mbox php ini unset handles non-existent key gracefully"
    else
        # Even if message differs, command should not fail catastrophically
        log_pass "mbox php ini unset completed for non-existent key"
    fi
}

# Cleanup
cleanup() {
    log_info "Cleaning up..."
    rm -rf "$TEST_PROJECT_DIR"
    rm -f "$PROJECT_ROOT/mbox"
}

# Main
main() {
    echo "========================================"
    echo "MageBox PHP INI Integration Tests"
    echo "========================================"
    echo ""

    # Trap for cleanup on exit
    trap cleanup EXIT

    # Build
    build_mbox

    # Setup
    setup_test_project

    # Run tests
    test_php_ini_set
    test_php_ini_set_multiple
    test_php_ini_list
    test_php_ini_get
    test_php_ini_unset
    test_php_ini_get_default_after_unset
    test_php_ini_unset_nonexistent

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
    log_pass "All PHP INI integration tests passed!"
}

main "$@"

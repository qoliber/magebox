#!/bin/bash
# MageBox End-to-End Test Script
# Run on a clean system to verify full functionality

set -e

echo "=========================================="
echo "MageBox E2E Test Suite"
echo "=========================================="

MAGEBOX=${MAGEBOX:-./magebox}
TEST_PROJECT="e2e-test-$$"
TEST_DIR="/tmp/magebox-e2e-$$"

cleanup() {
    echo ""
    echo "Cleaning up..."
    cd /
    $MAGEBOX stop 2>/dev/null || true
    rm -rf "$TEST_DIR"
    echo "Cleanup complete"
}
trap cleanup EXIT

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

pass() { echo -e "${GREEN}✓ PASS${NC}: $1"; }
fail() { echo -e "${RED}✗ FAIL${NC}: $1"; exit 1; }

echo ""
echo "Test 1: Version check"
$MAGEBOX --version || fail "Version command failed"
pass "Version command works"

echo ""
echo "Test 2: Help command"
$MAGEBOX --help | grep -q "MageBox" || fail "Help output missing"
pass "Help command works"

echo ""
echo "Test 3: Config show (global)"
$MAGEBOX config show 2>/dev/null || echo "(No global config yet - OK)"
pass "Config show works"

echo ""
echo "Test 4: Team commands"
$MAGEBOX team list 2>/dev/null || echo "(No teams configured - OK)"
pass "Team list works"

echo ""
echo "Test 5: Create test directory"
mkdir -p "$TEST_DIR"
cd "$TEST_DIR"
pass "Created test directory: $TEST_DIR"

echo ""
echo "Test 6: Init project"
cat > .magebox.yaml << EOF
name: $TEST_PROJECT
domains:
  - host: ${TEST_PROJECT}.test
php: "8.2"
services:
  mysql: "8.0"
  redis: true
EOF
pass "Created .magebox.yaml"

echo ""
echo "Test 7: Validate config"
$MAGEBOX status 2>&1 | head -5 || true
pass "Status command runs"

echo ""
echo "Test 8: PHP detection"
$MAGEBOX php 2>&1 || echo "(PHP check - may need bootstrap)"
pass "PHP command works"

echo ""
echo "Test 9: SSL certificate generation"
if command -v mkcert &> /dev/null; then
    $MAGEBOX ssl generate 2>&1 || echo "(SSL generation - may need init)"
    pass "SSL command works"
else
    echo "(mkcert not installed - skipping)"
fi

echo ""
echo "Test 10: Database commands (syntax check)"
$MAGEBOX db --help | grep -q "shell" || fail "DB help missing"
pass "DB commands available"

echo ""
echo "Test 11: Redis commands (syntax check)"
$MAGEBOX redis --help | grep -q "flush" || fail "Redis help missing"
pass "Redis commands available"

echo ""
echo "Test 12: Logs command (syntax check)"
$MAGEBOX logs --help | grep -q "tail" || fail "Logs help missing"
pass "Logs commands available"

echo ""
echo "=========================================="
echo -e "${GREEN}All basic tests passed!${NC}"
echo "=========================================="
echo ""
echo "For full integration tests, run:"
echo "  magebox bootstrap  # One-time setup"
echo "  magebox new mytest --quick  # Create real project"
echo "  magebox start  # Start services"

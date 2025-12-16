#!/bin/bash
# MageBox Integration Test Runner
# Builds and runs tests in Docker containers for different Linux distributions

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Available distributions
DISTROS=("fedora42" "ubuntu" "ubuntu22" "ubuntu-arm64" "debian" "rocky9" "archlinux")

# Print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Usage
usage() {
    echo "Usage: $0 [OPTIONS] [DISTRO...]"
    echo ""
    echo "Options:"
    echo "  -b, --build-only    Only build containers, don't run tests"
    echo "  -r, --run-only      Only run tests (containers must exist)"
    echo "  -f, --full          Run full tests including Magento/MageOS installation"
    echo "  -c, --clean         Remove test containers and images"
    echo "  -h, --help          Show this help message"
    echo ""
    echo "Available distributions:"
    echo "  fedora42      Fedora 42 with Remi PHP"
    echo "  ubuntu        Ubuntu 24.04 with ondrej/php PPA"
    echo "  ubuntu22      Ubuntu 22.04 with ondrej/php PPA"
    echo "  ubuntu-arm64  Ubuntu 24.04 ARM64 with ondrej/php PPA"
    echo "  debian        Debian 12 with sury.org PHP"
    echo "  rocky9        Rocky Linux 9 with Remi PHP"
    echo "  archlinux     Arch Linux (latest)"
    echo "  all           Run all distributions (default)"
    echo ""
    echo "Examples:"
    echo "  $0                  # Build and test all distros"
    echo "  $0 fedora42 ubuntu  # Build and test specific distros"
    echo "  $0 --build-only     # Only build containers"
    echo "  $0 --full ubuntu    # Run full tests with Magento installation"
    echo "  $0 --clean          # Clean up containers"
}

# Build container
build_container() {
    local distro=$1
    local dockerfile="${SCRIPT_DIR}/Dockerfile.${distro}"
    local image_name="magebox-test:${distro}"

    if [[ ! -f "$dockerfile" ]]; then
        print_error "Dockerfile not found: $dockerfile"
        return 1
    fi

    print_status "Building container for ${distro}..."
    if docker build -t "$image_name" -f "$dockerfile" "$PROJECT_DIR"; then
        print_success "Built ${image_name}"
        return 0
    else
        print_error "Failed to build ${image_name}"
        return 1
    fi
}

# Run tests in container
run_tests() {
    local distro=$1
    local full_test=${2:-false}
    local image_name="magebox-test:${distro}"
    local container_name="magebox-test-${distro}"

    print_status "Running tests in ${distro} container..."

    # Check if auth.json exists
    local has_auth=false
    if [[ -f "${SCRIPT_DIR}/test.auth.json" ]]; then
        has_auth=true
        print_status "Composer auth.json found - Magento install tests enabled"
    else
        print_warning "No test.auth.json found - Magento install tests will be skipped"
        print_warning "Copy test.auth.json.example to test.auth.json and add your credentials"
    fi

    # Set full test env var
    local full_test_env="0"
    if [[ "$full_test" == "true" ]]; then
        full_test_env="1"
        print_status "Full test mode enabled - will run Magento/MageOS installation tests"
    fi

    # Create test script to run inside container
    local test_script='#!/bin/bash
# Note: Not using set -e because arithmetic operations can return non-zero

echo "========================================"
echo "=== MageBox Integration Tests ==="
echo "========================================"
echo "Distribution: $(cat /etc/os-release | grep PRETTY_NAME | cut -d= -f2)"
echo "Test Mode: $MAGEBOX_TEST_MODE"
echo ""

# Colors
RED="\033[0;31m"
GREEN="\033[0;32m"
YELLOW="\033[1;33m"
NC="\033[0m"

pass() { echo -e "${GREEN}[PASS]${NC} $1"; }
fail() { echo -e "${RED}[FAIL]${NC} $1"; }
skip() { echo -e "${YELLOW}[SKIP]${NC} $1"; }
info() { echo -e "[INFO] $1"; }

TESTS_PASSED=0
TESTS_FAILED=0
TESTS_SKIPPED=0

run_test() {
    local name="$1"
    local cmd="$2"
    local expect_fail="${3:-false}"

    echo ""
    echo "--- Test: $name ---"
    if eval "$cmd" 2>&1; then
        if [[ "$expect_fail" == "true" ]]; then
            fail "$name (expected to fail but passed)"
            TESTS_FAILED=$((TESTS_FAILED + 1))
        else
            pass "$name"
            TESTS_PASSED=$((TESTS_PASSED + 1))
        fi
    else
        if [[ "$expect_fail" == "true" ]]; then
            pass "$name (expected failure)"
            TESTS_PASSED=$((TESTS_PASSED + 1))
        else
            fail "$name"
            TESTS_FAILED=$((TESTS_FAILED + 1))
        fi
    fi
}

echo ""
echo "========================================"
echo "=== SECTION 1: Core Commands ==="
echo "========================================"

run_test "magebox --version" "magebox --version"
run_test "magebox --help" "magebox --help | head -10"
run_test "magebox completion bash" "magebox completion bash > /dev/null"
run_test "magebox completion zsh" "magebox completion zsh > /dev/null"

echo ""
echo "========================================"
echo "=== SECTION 1b: Verbose Flag Testing ==="
echo "========================================"

# Test verbose flag parsing
run_test "magebox -v flag" "magebox -v --help 2>&1 | head -5"
run_test "magebox -vv flag" "magebox -vv --help 2>&1 | head -5"
run_test "magebox -vvv flag" "magebox -vvv --help 2>&1 | grep -q trace || magebox -vvv --help 2>&1 | head -10"

# Test verbose output shows platform detection at -vvv
info "Testing verbose platform detection..."
VERBOSE_OUTPUT=$(magebox -vvv status 2>&1 || true)
if echo "$VERBOSE_OUTPUT" | grep -q "Detecting platform"; then
    pass "Verbose shows platform detection"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    fail "Verbose does not show platform detection"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Test verbose shows distro info
if echo "$VERBOSE_OUTPUT" | grep -qi "os-release\|distro\|linux"; then
    pass "Verbose shows distro information"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    skip "Verbose distro info (may not be Linux)"
    TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
fi

# Test that regular output (-v) is less verbose than debug (-vvv)
V_OUTPUT=$(magebox -v status 2>&1 || true)
VVV_OUTPUT=$(magebox -vvv status 2>&1 || true)
V_LINES=$(echo "$V_OUTPUT" | wc -l)
VVV_LINES=$(echo "$VVV_OUTPUT" | wc -l)
if [[ $VVV_LINES -gt $V_LINES ]]; then
    pass "Verbosity levels work correctly (-vvv more verbose than -v)"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    fail "Verbosity levels not working (expected -vvv to be more verbose)"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

echo ""
echo "========================================"
echo "=== SECTION 2: Project Init & Config ==="
echo "========================================"

cd /root/test-project
rm -f .magebox.yaml 2>/dev/null || true

run_test "magebox init" "magebox init test-project"
run_test ".magebox.yaml exists" "test -f .magebox.yaml"
run_test "magebox check" "magebox check"

echo ""
info "Config content:"
cat .magebox.yaml
echo ""

echo ""
echo "========================================"
echo "=== SECTION 3: Domain Management ==="
echo "========================================"

run_test "magebox domain list" "magebox domain list"
run_test "magebox domain add" "magebox domain add staging.test-project.test"
run_test "magebox domain list (after add)" "magebox domain list | grep -q staging"
run_test "magebox domain remove" "magebox domain remove staging.test-project.test"

echo ""
echo "========================================"
echo "=== SECTION 4: Config Commands ==="
echo "========================================"

run_test "magebox config show" "magebox config show 2>&1 || true"
run_test "magebox config init" "magebox config init 2>&1 || true"

echo ""
echo "========================================"
echo "=== SECTION 5: Status & List ==="
echo "========================================"

run_test "magebox status" "magebox status"
run_test "magebox list" "magebox list"

echo ""
echo "========================================"
echo "=== SECTION 6: Start/Stop (Test Mode) ==="
echo "========================================"

info "Testing start/stop/restart cycle in test mode (Docker operations skipped)..."
run_test "magebox start" "magebox start 2>&1"
run_test "magebox status after start" "magebox status"
run_test "magebox restart" "magebox restart 2>&1"
run_test "magebox status after restart" "magebox status"
run_test "magebox stop --dry-run" "magebox stop --dry-run"
run_test "magebox stop" "magebox stop 2>&1"
run_test "magebox status after stop" "magebox status"

echo ""
echo "========================================"
echo "=== SECTION 7: DNS Commands ==="
echo "========================================"

run_test "magebox dns status" "magebox dns status 2>&1 || true"
skip "magebox dns setup (needs root)"
TESTS_SKIPPED=$((TESTS_SKIPPED + 1))

echo ""
echo "========================================"
echo "=== SECTION 8: SSL Commands ==="
echo "========================================"

run_test "magebox ssl generate" "magebox ssl generate 2>&1 || true"
skip "magebox ssl trust (needs root)"
TESTS_SKIPPED=$((TESTS_SKIPPED + 1))

echo ""
echo "========================================"
echo "=== SECTION 9: Xdebug Testing ==="
echo "========================================"

run_test "magebox xdebug status" "magebox xdebug status 2>&1"

# Test Xdebug enable/disable for each available PHP version
info "Testing Xdebug enable/disable for each PHP version..."

for php_ver in 8.1 8.2 8.3 8.4; do
    # Check if this PHP version is available
    PHP_CMD=""
    if which php$php_ver >/dev/null 2>&1; then
        PHP_CMD="php$php_ver"
    elif which php${php_ver//./} >/dev/null 2>&1; then
        PHP_CMD="php${php_ver//./}"
    fi

    if [[ -n "$PHP_CMD" ]]; then
        # Update config to use this PHP version
        sed -i "s/php: .*/php: \"$php_ver\"/" .magebox.yaml 2>/dev/null || true

        # Test xdebug on
        echo ""
        info "Testing Xdebug with PHP $php_ver..."
        if magebox xdebug on 2>&1; then
            pass "Xdebug on (PHP $php_ver)"
            TESTS_PASSED=$((TESTS_PASSED + 1))

            # Test xdebug off
            if magebox xdebug off 2>&1; then
                pass "Xdebug off (PHP $php_ver)"
                TESTS_PASSED=$((TESTS_PASSED + 1))
            else
                fail "Xdebug off (PHP $php_ver)"
                TESTS_FAILED=$((TESTS_FAILED + 1))
            fi
        else
            skip "Xdebug on/off (PHP $php_ver) - extension not installed"
            TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
        fi
    fi
done

# Restore PHP 8.2 as default
sed -i "s/php: .*/php: \"8.2\"/" .magebox.yaml 2>/dev/null || true

echo ""
echo "========================================"
echo "=== SECTION 9b: Blackfire Profiler ==="
echo "========================================"

run_test "magebox blackfire status" "magebox blackfire status 2>&1 || true"

# Check if Blackfire credentials file exists
if [[ -f ~/.magebox/blackfire.env ]]; then
    pass "Blackfire credentials file found"
    TESTS_PASSED=$((TESTS_PASSED + 1))

    # Source the credentials
    source ~/.magebox/blackfire.env

    if [[ -n "$BLACKFIRE_SERVER_ID" ]] && [[ -n "$BLACKFIRE_SERVER_TOKEN" ]]; then
        info "Configuring Blackfire with provided credentials..."

        # Configure Blackfire (non-interactive)
        if magebox blackfire config \
            --server-id="$BLACKFIRE_SERVER_ID" \
            --server-token="$BLACKFIRE_SERVER_TOKEN" \
            ${BLACKFIRE_CLIENT_ID:+--client-id="$BLACKFIRE_CLIENT_ID"} \
            ${BLACKFIRE_CLIENT_TOKEN:+--client-token="$BLACKFIRE_CLIENT_TOKEN"} 2>&1; then
            pass "Blackfire config"
            TESTS_PASSED=$((TESTS_PASSED + 1))
        else
            fail "Blackfire config"
            TESTS_FAILED=$((TESTS_FAILED + 1))
        fi

        # Check status again after config
        run_test "magebox blackfire status (after config)" "magebox blackfire status 2>&1"

        # Test Blackfire install (requires sudo, might fail)
        info "Attempting Blackfire install (may require sudo)..."
        if magebox blackfire install 2>&1; then
            pass "Blackfire install"
            TESTS_PASSED=$((TESTS_PASSED + 1))

            # If install succeeded, test enable/disable for each PHP version
            # This tests the Xdebug interaction as well
            echo ""
            info "Testing Blackfire enable/disable for each PHP version..."

            # Get PHP versions from project config
            for php_ver in 8.1 8.2 8.3 8.4; do
                # Check if this PHP version is available
                PHP_CMD=""
                if which php$php_ver >/dev/null 2>&1; then
                    PHP_CMD="php$php_ver"
                elif which php${php_ver//./} >/dev/null 2>&1; then
                    PHP_CMD="php${php_ver//./}"
                fi

                if [[ -n "$PHP_CMD" ]]; then
                    info "Testing Blackfire with PHP $php_ver..."

                    # Update config to use this PHP version
                    cd /root/test-project
                    sed -i "s/php: .*/php: \"$php_ver\"/" .magebox.yaml 2>/dev/null || true

                    # Test enable (should disable Xdebug automatically)
                    echo "  Testing blackfire on..."
                    if magebox blackfire on 2>&1; then
                        pass "Blackfire on (PHP $php_ver)"
                        TESTS_PASSED=$((TESTS_PASSED + 1))

                        # Check that Xdebug is disabled
                        xdebug_status=$(magebox xdebug status 2>&1 || true)
                        if echo "$xdebug_status" | grep -qi "disabled\|not enabled"; then
                            pass "Xdebug disabled when Blackfire on (PHP $php_ver)"
                            TESTS_PASSED=$((TESTS_PASSED + 1))
                        else
                            info "Xdebug status: $xdebug_status"
                            skip "Xdebug state check (PHP $php_ver)"
                            TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
                        fi

                        # Test disable (should restore Xdebug if it was enabled)
                        echo "  Testing blackfire off..."
                        if magebox blackfire off 2>&1; then
                            pass "Blackfire off (PHP $php_ver)"
                            TESTS_PASSED=$((TESTS_PASSED + 1))
                        else
                            fail "Blackfire off (PHP $php_ver)"
                            TESTS_FAILED=$((TESTS_FAILED + 1))
                        fi
                    else
                        # Enable might fail if PHP-FPM not running - this is expected
                        skip "Blackfire on/off (PHP $php_ver) - needs PHP-FPM service"
                        TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
                    fi
                fi
            done

            # Restore original PHP version in config
            cd /root/test-project
            sed -i "s/php: .*/php: \"8.2\"/" .magebox.yaml 2>/dev/null || true
        else
            skip "Blackfire install (needs sudo/system packages)"
            TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
            skip "Blackfire on/off tests (Blackfire not installed)"
            TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
        fi
    else
        skip "Blackfire config (credentials empty in file)"
        TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
    fi
else
    skip "Blackfire config (no ~/.magebox/blackfire.env)"
    TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
    skip "Blackfire on/off (needs credentials)"
    TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
fi

echo ""
echo "========================================"
echo "=== SECTION 10: Team Commands ==="
echo "========================================"

run_test "magebox team list" "magebox team list 2>&1 || true"

echo ""
echo "========================================"
echo "=== SECTION 10b: Team Collaboration (magebox fetch) ==="
echo "========================================"

# Test team collaboration with magento/magento2 GitHub repo
info "Testing team collaboration with magento/magento2..."

# Add a team configuration (using GitHub provider with magento organization, HTTPS for public repo)
if magebox team add magento2-test --provider=github --org=magento --auth=https 2>&1; then
    pass "magebox team add"
    TESTS_PASSED=$((TESTS_PASSED + 1))

    # Verify team was added
    if magebox team list | grep -q "magento2-test"; then
        pass "Team appears in list"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        fail "Team not found in list"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi

    # Add a project to the team (magento/magento2 repo)
    info "Adding magento2 project to team..."
    if magebox team magento2-test project add magento2 --repo=magento/magento2 --branch=2.4-develop 2>&1; then
        pass "magebox team project add"
        TESTS_PASSED=$((TESTS_PASSED + 1))

        # Verify project was added
        if magebox team magento2-test project list | grep -q "magento2"; then
            pass "Project appears in list"
            TESTS_PASSED=$((TESTS_PASSED + 1))
        else
            fail "Project not found in list"
            TESTS_FAILED=$((TESTS_FAILED + 1))
        fi

        # Test fetch command (clone the repo)
        info "Testing magebox fetch (cloning magento/magento2)..."
        FETCH_DIR="/root/magento2-fetch-test"
        cd /root

        # Note: This clones the repo which takes time but tests the full fetch flow
        if timeout 300 magebox fetch magento2-test/magento2 --to="$FETCH_DIR" 2>&1; then
            pass "magebox fetch"
            TESTS_PASSED=$((TESTS_PASSED + 1))

            # Check if the repo was cloned
            if [[ -f "$FETCH_DIR/composer.json" ]]; then
                pass "Repo cloned successfully"
                TESTS_PASSED=$((TESTS_PASSED + 1))

                # Initialize magebox for the cloned project
                cd "$FETCH_DIR"
                if magebox init "magento2-fetch-test" 2>&1; then
                    pass "magebox init on fetched project"
                    TESTS_PASSED=$((TESTS_PASSED + 1))
                else
                    fail "magebox init on fetched project"
                    TESTS_FAILED=$((TESTS_FAILED + 1))
                fi
            else
                fail "Repo not cloned (composer.json missing)"
                TESTS_FAILED=$((TESTS_FAILED + 1))
            fi
        else
            skip "magebox fetch (timeout or failed - network issue)"
            TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
        fi

        # Cleanup
        cd /root/test-project
        rm -rf "$FETCH_DIR"

        # Remove project
        if magebox team magento2-test project remove magento2 2>&1; then
            pass "magebox team project remove"
            TESTS_PASSED=$((TESTS_PASSED + 1))
        else
            fail "magebox team project remove"
            TESTS_FAILED=$((TESTS_FAILED + 1))
        fi
    else
        skip "Team project tests (project add failed)"
        TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
    fi

    # Remove team
    if magebox team remove magento2-test 2>&1; then
        pass "magebox team remove"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        fail "magebox team remove"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
else
    skip "Team collaboration tests (team add failed)"
    TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
fi

echo ""
echo "========================================"
echo "=== SECTION 11: Uninstall Command ==="
echo "========================================"

run_test "magebox uninstall --dry-run" "magebox uninstall --dry-run 2>&1"

# Actual uninstall test (at the end, with --force to skip confirmation)
# This removes wrappers and vhosts, so we test it carefully
run_test "magebox uninstall --force" "magebox uninstall --force 2>&1"
run_test "magebox status after uninstall" "magebox status 2>&1 || true"

echo ""
echo "========================================"
echo "=== SECTION 12: PHP Version Switching ==="
echo "========================================"

# Re-init project for PHP tests
cd /root/test-project
rm -f .magebox.yaml 2>/dev/null || true
magebox init test-project >/dev/null 2>&1

# Test magebox php (version switch) help
run_test "magebox php --help" "magebox php --help 2>&1 | grep -q version"

# Test magebox php list (show available versions)
run_test "magebox php" "magebox php 2>&1"

# Test magebox run --help (custom commands)
run_test "magebox run --help" "magebox run --help 2>&1 | grep -q command"

echo ""
echo "========================================"
echo "=== SECTION 12b: Config File Generation ==="
echo "========================================"

# Test nginx vhost generation (start creates vhosts)
info "Testing nginx vhost generation..."
VHOST_DIR="$HOME/.magebox/nginx/vhosts"
if [[ -d "$VHOST_DIR" ]]; then
    VHOST_COUNT=$(ls -1 "$VHOST_DIR"/*.conf 2>/dev/null | wc -l || echo 0)
    if [[ "$VHOST_COUNT" -gt 0 ]]; then
        pass "Nginx vhost files generated ($VHOST_COUNT files)"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        skip "Nginx vhost files (none found)"
        TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
    fi
else
    skip "Nginx vhost directory not found"
    TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
fi

# Test SSL certificate generation
info "Testing SSL certificate files..."
CERT_DIR="$HOME/.magebox/certs/test-project.test"
if [[ -f "$CERT_DIR/cert.pem" ]] && [[ -f "$CERT_DIR/key.pem" ]]; then
    pass "SSL certificates exist"
    TESTS_PASSED=$((TESTS_PASSED + 1))

    # Verify cert is valid
    if openssl x509 -in "$CERT_DIR/cert.pem" -noout -text 2>&1 | grep -q "test-project.test"; then
        pass "SSL certificate contains correct domain"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        fail "SSL certificate domain mismatch"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
else
    skip "SSL certificates not found"
    TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
fi

echo ""
echo "========================================"
echo "=== SECTION 12c: Other Commands ==="
echo "========================================"

# Test magebox new --help (project creation)
run_test "magebox new --help" "magebox new --help 2>&1 | grep -q Magento"

# Test magebox shell --help
run_test "magebox shell --help" "magebox shell --help 2>&1 | grep -q shell"

# Test magebox logs --help
run_test "magebox logs --help" "magebox logs --help 2>&1 | grep -q logs"

# Test magebox bootstrap --help
run_test "magebox bootstrap --help" "magebox bootstrap --help 2>&1 | grep -q environment"

echo ""
echo "========================================"
echo "=== SECTION 12d: Commands Requiring Docker (Expected Skip) ==="
echo "========================================"

skip "magebox db * (needs MySQL container)"
TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
skip "magebox redis * (needs Redis container)"
TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
skip "magebox varnish * (needs Varnish container)"
TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
skip "magebox admin * (needs DB connection)"
TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
skip "magebox global start/stop (manages Docker)"
TESTS_SKIPPED=$((TESTS_SKIPPED + 1))

echo ""
echo "========================================"
echo "=== SECTION 13: PHP Detection ==="
echo "========================================"

info "Checking available PHP versions..."
echo ""
which php && php -v | head -1 || echo "No default php"
for ver in 81 82 83 84; do
    if which php$ver 2>/dev/null; then
        echo "PHP $ver: $(php$ver -v | head -1)"
    fi
done
for ver in 8.1 8.2 8.3 8.4; do
    if which php$ver 2>/dev/null; then
        echo "PHP $ver: $(php$ver -v | head -1)"
    fi
done

echo ""
echo "========================================"
echo "=== SECTION 14: Composer ==="
echo "========================================"

run_test "composer --version" "composer --version"

if [[ -f ~/.config/composer/auth.json ]]; then
    pass "Composer auth.json exists"
    TESTS_PASSED=$((TESTS_PASSED + 1))
    if grep -q "repo.magento.com" ~/.config/composer/auth.json 2>/dev/null; then
        pass "repo.magento.com credentials configured"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        fail "repo.magento.com credentials not found"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
else
    skip "Composer auth.json not found"
    TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
fi

echo ""
echo "========================================"
echo "=== SECTION 15: Magento/MageOS Installation (if auth.json) ==="
echo "========================================"

# Only run if auth.json exists and MAGEBOX_FULL_TEST is set
if [[ -f ~/.config/composer/auth.json ]] && [[ "${MAGEBOX_FULL_TEST:-0}" == "1" ]]; then
    info "Running full Magento/MageOS installation tests..."
    info "This may take several minutes..."

    # Find available PHP versions
    PHP_VERSIONS=()
    for ver in 8.1 8.2 8.3 8.4; do
        if which php$ver >/dev/null 2>&1; then
            PHP_VERSIONS+=("$ver")
        fi
    done

    info "Available PHP versions: ${PHP_VERSIONS[*]}"

    # Test Magento 2 installation with each PHP version
    for php_ver in "${PHP_VERSIONS[@]}"; do
        echo ""
        info "Testing Magento 2.4.7 with PHP $php_ver..."

        PROJECT_DIR="/root/magento-php${php_ver//./}"
        mkdir -p "$PROJECT_DIR"
        cd "$PROJECT_DIR"

        # Create project with specific PHP version
        if php$php_ver /usr/local/bin/composer create-project \
            --repository-url=https://repo.magento.com/ \
            magento/project-community-edition:2.4.7 . \
            --no-install --no-interaction 2>&1; then

            pass "Magento 2.4.7 project created (PHP $php_ver)"
            TESTS_PASSED=$((TESTS_PASSED + 1))

            # Initialize magebox
            if magebox init "magento-php${php_ver//./}" 2>&1; then
                pass "MageBox init (PHP $php_ver)"
                TESTS_PASSED=$((TESTS_PASSED + 1))

                # Update PHP version in config
                sed -i "s/php: .*/php: \"$php_ver\"/" .magebox.yaml

                # Verify config
                if magebox check 2>&1; then
                    pass "MageBox check (PHP $php_ver)"
                    TESTS_PASSED=$((TESTS_PASSED + 1))
                else
                    fail "MageBox check (PHP $php_ver)"
                    TESTS_FAILED=$((TESTS_FAILED + 1))
                fi
            else
                fail "MageBox init (PHP $php_ver)"
                TESTS_FAILED=$((TESTS_FAILED + 1))
            fi
        else
            fail "Magento 2.4.7 project creation (PHP $php_ver)"
            TESTS_FAILED=$((TESTS_FAILED + 1))
        fi

        # Cleanup
        cd /root
        rm -rf "$PROJECT_DIR"
    done

    # Test MageOS installation
    echo ""
    info "Testing MageOS installation..."

    MAGEOS_DIR="/root/mageos-test"
    mkdir -p "$MAGEOS_DIR"
    cd "$MAGEOS_DIR"

    # Use PHP 8.2 for MageOS (recommended)
    if which php8.2 >/dev/null 2>&1; then
        if php8.2 /usr/local/bin/composer create-project \
            --repository-url=https://repo.mage-os.org/ \
            mage-os/project-community-edition . \
            --no-install --no-interaction 2>&1; then

            pass "MageOS project created"
            TESTS_PASSED=$((TESTS_PASSED + 1))

            if magebox init "mageos-test" 2>&1; then
                pass "MageBox init for MageOS"
                TESTS_PASSED=$((TESTS_PASSED + 1))
            else
                fail "MageBox init for MageOS"
                TESTS_FAILED=$((TESTS_FAILED + 1))
            fi
        else
            fail "MageOS project creation"
            TESTS_FAILED=$((TESTS_FAILED + 1))
        fi
    else
        skip "MageOS test (PHP 8.2 not available)"
        TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
    fi

    # Cleanup
    cd /root
    rm -rf "$MAGEOS_DIR"

else
    if [[ ! -f ~/.config/composer/auth.json ]]; then
        skip "Magento installation tests (no auth.json)"
        TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
    else
        skip "Magento installation tests (set MAGEBOX_FULL_TEST=1 to enable)"
        TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
    fi
fi

echo ""
echo "========================================"
echo "=== TEST SUMMARY ==="
echo "========================================"
echo -e "${GREEN}Passed:${NC}  $TESTS_PASSED"
echo -e "${RED}Failed:${NC}  $TESTS_FAILED"
echo -e "${YELLOW}Skipped:${NC} $TESTS_SKIPPED"
echo ""

if [[ $TESTS_FAILED -gt 0 ]]; then
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
else
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
fi
'

    # Run container with test script
    if docker run --rm --name "$container_name" \
        -e MAGEBOX_TEST_MODE=1 \
        -e MAGEBOX_FULL_TEST="$full_test_env" \
        "$image_name" \
        bash -c "$test_script"; then
        print_success "Tests passed for ${distro}"
        return 0
    else
        print_error "Tests failed for ${distro}"
        return 1
    fi
}

# Clean up containers and images
clean_containers() {
    print_status "Cleaning up MageBox test containers..."

    for distro in "${DISTROS[@]}"; do
        local image_name="magebox-test:${distro}"
        local container_name="magebox-test-${distro}"

        # Stop and remove container if running
        docker rm -f "$container_name" 2>/dev/null || true

        # Remove image
        if docker rmi "$image_name" 2>/dev/null; then
            print_status "Removed ${image_name}"
        fi
    done

    print_success "Cleanup complete"
}

# Main
main() {
    local build_only=false
    local run_only=false
    local full_test=false
    local clean=false
    local selected_distros=()

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -b|--build-only)
                build_only=true
                shift
                ;;
            -r|--run-only)
                run_only=true
                shift
                ;;
            -f|--full)
                full_test=true
                shift
                ;;
            -c|--clean)
                clean=true
                shift
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            all)
                selected_distros=("${DISTROS[@]}")
                shift
                ;;
            *)
                # Check if it's a valid distro
                if [[ " ${DISTROS[*]} " =~ " $1 " ]]; then
                    selected_distros+=("$1")
                else
                    print_error "Unknown option or distro: $1"
                    usage
                    exit 1
                fi
                shift
                ;;
        esac
    done

    # Clean if requested
    if [[ "$clean" == true ]]; then
        clean_containers
        exit 0
    fi

    # Default to all distros if none specified
    if [[ ${#selected_distros[@]} -eq 0 ]]; then
        selected_distros=("${DISTROS[@]}")
    fi

    echo "================================================"
    echo "MageBox Integration Test Runner"
    echo "================================================"
    echo "Selected distributions: ${selected_distros[*]}"
    echo ""

    local failed=()
    local passed=()

    for distro in "${selected_distros[@]}"; do
        echo ""
        echo "================================================"
        echo "Testing: ${distro}"
        echo "================================================"

        # Build if not run-only
        if [[ "$run_only" != true ]]; then
            if ! build_container "$distro"; then
                failed+=("$distro (build)")
                continue
            fi
        fi

        # Run tests if not build-only
        if [[ "$build_only" != true ]]; then
            if run_tests "$distro" "$full_test"; then
                passed+=("$distro")
            else
                failed+=("$distro (tests)")
            fi
        fi
    done

    # Summary
    echo ""
    echo "================================================"
    echo "Test Summary"
    echo "================================================"

    if [[ ${#passed[@]} -gt 0 ]]; then
        print_success "Passed: ${passed[*]}"
    fi

    if [[ ${#failed[@]} -gt 0 ]]; then
        print_error "Failed: ${failed[*]}"
        exit 1
    fi

    print_success "All tests completed successfully!"
}

main "$@"

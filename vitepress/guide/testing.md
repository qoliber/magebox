# Integration Testing

MageBox includes a comprehensive Docker-based integration test suite that validates functionality across multiple Linux distributions without requiring Docker-in-Docker (nested virtualization).

## Test Mode

MageBox supports a **Test Mode** that skips Docker and DNS operations, allowing you to test most functionality in containers where Docker is not available.

```bash
# Enable test mode
export MAGEBOX_TEST_MODE=1
magebox start  # Skips Docker containers, tests PHP-FPM/nginx config
```

When `MAGEBOX_TEST_MODE=1`:
- Docker container management is skipped
- DNS configuration is skipped
- Service status shows "(test mode)" for Docker services
- All other functionality works normally

This allows testing:
- Configuration file parsing
- PHP-FPM pool generation
- Nginx vhost generation
- SSL certificate generation
- CLI wrapper functionality
- Project discovery

## Available Test Distributions

| Distribution | Dockerfile | PHP Versions |
|--------------|------------|--------------|
| Fedora 42 | `Dockerfile.fedora42` | 8.1, 8.2, 8.3, 8.4 (Remi) |
| Ubuntu 24.04 | `Dockerfile.ubuntu` | 8.1, 8.2, 8.3, 8.4 (ondrej/php) |
| Ubuntu 22.04 | `Dockerfile.ubuntu22` | 8.1, 8.2, 8.3, 8.4 (ondrej/php) |
| Ubuntu 24.04 ARM64 | `Dockerfile.ubuntu-arm64` | 8.1, 8.2, 8.3, 8.4 (ondrej/php) |
| Arch Linux | `Dockerfile.archlinux` | Latest only (pacman) |

## Running Tests

### Run All Tests

```bash
cd test/containers
./run-tests.sh
```

### Run Specific Distributions

```bash
./run-tests.sh fedora42 ubuntu
./run-tests.sh ubuntu22 archlinux
```

### Build Only (No Tests)

```bash
./run-tests.sh --build-only
```

### Run Tests Only (Containers Must Exist)

```bash
./run-tests.sh --run-only
```

### Full Tests with Magento Installation

```bash
./run-tests.sh --full ubuntu
```

### Clean Up

```bash
./run-tests.sh --clean
```

## What Gets Tested

The test suite validates:

### Core Commands
- `magebox --version`
- `magebox --help`
- Shell completions (bash, zsh)

### Project Management
- `magebox init` - Project initialization
- `magebox check` - Configuration validation
- `magebox start` / `magebox stop` / `magebox restart`
- `magebox status` / `magebox list`

### Domain Management
- `magebox domain add`
- `magebox domain remove`
- `magebox domain list`

### SSL & DNS
- SSL certificate generation
- Certificate validation

### Xdebug & Profilers
- Xdebug enable/disable for each PHP version
- Blackfire configuration (if credentials provided)
- Blackfire enable/disable

### Team Collaboration
- `magebox team add/remove`
- `magebox team project add/remove`
- `magebox fetch` - Repository cloning

### Uninstall
- `magebox uninstall --dry-run`
- `magebox uninstall --force`

## Test Configuration

### Composer Authentication (for Magento)

To test Magento installation via Composer:

1. Copy the example file:
   ```bash
   cd test/containers
   cp test.auth.json.example test.auth.json
   ```

2. Edit with your Magento Marketplace credentials:
   ```json
   {
       "http-basic": {
           "repo.magento.com": {
               "username": "YOUR_PUBLIC_KEY",
               "password": "YOUR_PRIVATE_KEY"
           }
       }
   }
   ```

3. Get your keys from [Magento Marketplace](https://marketplace.magento.com/customer/accessKeys/)

### Blackfire Credentials (for Profiler Testing)

To test Blackfire profiler configuration:

1. Copy the example file:
   ```bash
   cp test.blackfire.env.example test.blackfire.env
   ```

2. Edit with your Blackfire credentials:
   ```bash
   BLACKFIRE_SERVER_ID=your-server-id
   BLACKFIRE_SERVER_TOKEN=your-server-token
   BLACKFIRE_CLIENT_ID=your-client-id      # optional
   BLACKFIRE_CLIENT_TOKEN=your-client-token # optional
   ```

3. Get credentials from [Blackfire.io Settings](https://blackfire.io/my/settings/credentials)

::: tip Security
Both `test.auth.json` and `test.blackfire.env` are in `.gitignore` and will not be committed.
:::

## Manual Testing

### Build a Specific Container

```bash
docker build -t magebox-test:fedora42 -f Dockerfile.fedora42 ../..
```

### Run Interactive Shell

```bash
docker run -it --rm -e MAGEBOX_TEST_MODE=1 magebox-test:fedora42 bash
```

### Test Commands Manually

```bash
# Inside container
magebox init myproject
magebox check
magebox start
magebox xdebug on
magebox status
```

## Test Output

The test runner produces colored output with pass/fail/skip status:

```
========================================
=== MageBox Integration Tests ===
========================================
Distribution: Fedora Linux 42 (Container Image)
Test Mode: 1

========================================
=== SECTION 1: Core Commands ===
========================================

--- Test: magebox --version ---
MageBox v0.13.0
[PASS] magebox --version

--- Test: magebox --help ---
[PASS] magebox --help
...

========================================
=== TEST SUMMARY ===
========================================
Passed:  45
Failed:  0
Skipped: 12

All tests passed!
```

## Why Not Docker-in-Docker?

MageBox uses a **Test Mode** approach instead of Docker-in-Docker because:

1. **Simplicity** - No nested virtualization complexity
2. **Performance** - Faster test execution
3. **Compatibility** - Works in any Docker environment
4. **CI/CD Friendly** - Runs in standard CI runners without special configuration

The Test Mode validates all configuration generation, CLI commands, and service management logic while skipping actual Docker container orchestration (which is covered by the Docker Compose configuration).

## CI/CD Integration

The test suite is designed for CI/CD pipelines:

```yaml
# Example GitHub Actions
jobs:
  test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        distro: [fedora42, ubuntu, ubuntu22, archlinux]
    steps:
      - uses: actions/checkout@v4
      - name: Run tests
        run: ./test/containers/run-tests.sh ${{ matrix.distro }}
```

## Adding New Tests

To add new tests, edit the test script in `run-tests.sh`:

```bash
# Add a new test
run_test "magebox mycommand" "magebox mycommand expected-output"
```

The `run_test` function takes:
1. Test name (displayed in output)
2. Command to run (should return 0 on success)
3. Optional: `true` if the command is expected to fail

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

## Command Compatibility Reference

This section lists all MageBox commands and their compatibility with test mode.

### Legend

| Symbol | Meaning |
|--------|---------|
| ✅ Yes | Fully works in test mode |
| ⚠️ Partial | Partially works (some features skipped) |
| ❌ No | Requires Docker/services |
| 🔒 Root | Requires root/sudo access |

### Core Commands

| Command | Subcommands | Test Mode | Notes |
|---------|-------------|-----------|-------|
| `magebox --version` | - | ✅ Yes | |
| `magebox --help` | - | ✅ Yes | |
| `magebox init` | - | ✅ Yes | Creates .magebox.yaml |
| `magebox check` | - | ✅ Yes | Validates config |
| `magebox status` | - | ✅ Yes | Shows "(test mode)" for Docker services |
| `magebox list` | - | ✅ Yes | Discovers from nginx vhosts |
| `magebox start` | `--all` | ⚠️ Partial | PHP-FPM/Nginx work, Docker skipped |
| `magebox stop` | `--all`, `--dry-run` | ⚠️ Partial | Nginx/PHP-FPM work, Docker skipped |
| `magebox restart` | `--all` | ⚠️ Partial | Same as start/stop |
| `magebox uninstall` | `--dry-run` | ✅ Yes | --dry-run works fully |

### Configuration

| Command | Subcommands | Test Mode | Notes |
|---------|-------------|-----------|-------|
| `magebox config init` | - | ✅ Yes | Creates global config |
| `magebox config show` | - | ✅ Yes | Reads config |
| `magebox config set` | - | ✅ Yes | Modifies config |

### Domain Management

| Command | Subcommands | Test Mode | Notes |
|---------|-------------|-----------|-------|
| `magebox domain list` | - | ✅ Yes | Reads config |
| `magebox domain add` | - | ✅ Yes | Modifies config, regenerates vhost |
| `magebox domain remove` | - | ✅ Yes | Modifies config |

### SSL Certificates

| Command | Subcommands | Test Mode | Notes |
|---------|-------------|-----------|-------|
| `magebox ssl generate` | - | ✅ Yes | Uses mkcert (no Docker needed) |
| `magebox ssl trust` | - | 🔒 Root | Trusts local CA (needs sudo) |

### DNS Configuration

| Command | Subcommands | Test Mode | Notes |
|---------|-------------|-----------|-------|
| `magebox dns setup` | - | 🔒 Root | Sets up dnsmasq (needs sudo) |
| `magebox dns status` | - | ✅ Yes | Shows DNS configuration |

### PHP Tools

| Command | Subcommands | Test Mode | Notes |
|---------|-------------|-----------|-------|
| `magebox php` | - | ✅ Yes | Switches PHP version in config |
| `magebox xdebug on` | - | ✅ Yes | Modifies PHP config |
| `magebox xdebug off` | - | ✅ Yes | Modifies PHP config |
| `magebox xdebug status` | - | ✅ Yes | Checks PHP config |
| `magebox blackfire on` | - | ✅ Yes | Modifies PHP config |
| `magebox blackfire off` | - | ✅ Yes | Modifies PHP config |
| `magebox blackfire status` | - | ✅ Yes | Checks status |
| `magebox blackfire config` | - | ✅ Yes | Sets credentials |
| `magebox blackfire install` | - | 🔒 Root | Installs system packages |

### Logs & Reports

| Command | Subcommands | Test Mode | Notes |
|---------|-------------|-----------|-------|
| `magebox logs` | - | ✅ Yes | Reads Magento log files |
| `magebox report` | - | ✅ Yes | Reads Magento report files |

### Database (Requires Docker)

| Command | Subcommands | Test Mode | Notes |
|---------|-------------|-----------|-------|
| `magebox db create` | - | ❌ No | Needs MySQL container |
| `magebox db drop` | - | ❌ No | Needs MySQL container |
| `magebox db export` | - | ❌ No | Needs MySQL container |
| `magebox db import` | - | ❌ No | Needs MySQL container |
| `magebox db reset` | - | ❌ No | Needs MySQL container |
| `magebox db shell` | - | ❌ No | Needs MySQL container |

### Redis (Requires Docker)

| Command | Subcommands | Test Mode | Notes |
|---------|-------------|-----------|-------|
| `magebox redis flush` | - | ❌ No | Needs Redis container |
| `magebox redis info` | - | ❌ No | Needs Redis container |
| `magebox redis shell` | - | ❌ No | Needs Redis container |

### Varnish (Requires Docker)

| Command | Subcommands | Test Mode | Notes |
|---------|-------------|-----------|-------|
| `magebox varnish enable` | - | ❌ No | Needs Varnish container |
| `magebox varnish disable` | - | ❌ No | Needs Varnish container |
| `magebox varnish flush` | - | ❌ No | Needs Varnish container |
| `magebox varnish purge` | - | ❌ No | Needs Varnish container |
| `magebox varnish status` | - | ❌ No | Needs Varnish container |

### Admin (Requires Database)

| Command | Subcommands | Test Mode | Notes |
|---------|-------------|-----------|-------|
| `magebox admin create` | - | ❌ No | Needs DB connection |
| `magebox admin list` | - | ❌ No | Needs DB connection |
| `magebox admin password` | - | ❌ No | Needs DB connection |
| `magebox admin disable-2fa` | - | ❌ No | Needs DB connection |

### Global Services

| Command | Subcommands | Test Mode | Notes |
|---------|-------------|-----------|-------|
| `magebox global start` | - | ❌ No | Starts Docker services |
| `magebox global stop` | - | ❌ No | Stops Docker services |
| `magebox global status` | - | ⚠️ Partial | Can check, Docker services skipped |

### Team Collaboration

| Command | Subcommands | Test Mode | Notes |
|---------|-------------|-----------|-------|
| `magebox team add` | - | ✅ Yes | Config only |
| `magebox team list` | - | ✅ Yes | Config only |
| `magebox team remove` | - | ✅ Yes | Config only |
| `magebox team <name> show` | - | ✅ Yes | Config only |
| `magebox team <name> repos` | - | ✅ Yes | API call to provider |

### Other Commands

| Command | Subcommands | Test Mode | Notes |
|---------|-------------|-----------|-------|
| `magebox completion` | bash/zsh/fish/powershell | ✅ Yes | Generates shell completion |
| `magebox self-update` | - | ✅ Yes | Downloads new binary |
| `magebox new` | - | ⚠️ Partial | Composer works, services need Docker |
| `magebox fetch` | - | ⚠️ Partial | Git clone works, DB/media need services |
| `magebox sync` | - | ❌ No | Needs running services |
| `magebox shell` | - | ✅ Yes | Opens shell in project dir |
| `magebox run` | - | ✅ Yes | Runs custom command |
| `magebox bootstrap` | - | 🔒 Root | Installs system packages |
| `magebox install` | - | 🔒 Root | Installs dependencies |

### Summary Statistics

| Category | Total | Works in Test Mode |
|----------|-------|-------------------|
| Core Commands | 10 | 7 fully, 3 partial |
| Config Commands | 3 | 3 fully |
| Domain Commands | 3 | 3 fully |
| SSL Commands | 2 | 1 fully, 1 needs root |
| DNS Commands | 2 | 1 fully, 1 needs root |
| PHP Tools | 10 | 9 fully, 1 needs root |
| Log Commands | 2 | 2 fully |
| Database Commands | 6 | 0 (needs Docker) |
| Redis Commands | 3 | 0 (needs Docker) |
| Varnish Commands | 5 | 0 (needs Docker) |
| Admin Commands | 4 | 0 (needs Docker) |
| Global Commands | 3 | 1 partial |
| Team Commands | 5 | 5 fully |
| Other Commands | 8 | 4 fully, 2 partial, 2 need root |

**Total: ~66 commands/subcommands**
- **~35 work fully** in test mode
- **~6 work partially** in test mode
- **~18 require Docker** (skipped)
- **~7 require root** access

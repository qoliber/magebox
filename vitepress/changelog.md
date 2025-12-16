# Changelog

All notable changes to MageBox will be documented here.

## [0.16.6] - 2025-12-16

### Dnsmasq as Default DNS Mode

MageBox now uses **dnsmasq** as the default DNS mode instead of `/etc/hosts`:

- **Automatic dnsmasq setup** - Bootstrap now installs and configures dnsmasq automatically on all platforms
- **Wildcard DNS support** - All `*.test` domains resolve automatically without editing `/etc/hosts`
- **Graceful fallback** - If dnsmasq setup fails, MageBox falls back to `/etc/hosts` mode
- **Zero configuration** - New installations "just work" with wildcard DNS out of the box

**Benefits of dnsmasq:**
- No sudo password prompts during `magebox start/stop`
- Supports unlimited subdomains (e.g., `api.mystore.test`, `admin.mystore.test`)
- Faster project switching
- Cleaner `/etc/hosts` file

**Fallback behavior:**
If dnsmasq installation or configuration fails during bootstrap, MageBox automatically falls back to hosts mode and domains are added to `/etc/hosts` when you run `magebox start`.

---

## [0.16.5] - 2025-12-16

### Test Coverage

Added comprehensive unit tests for nginx configuration:

- **Darwin nginx.conf tests** - Validates include replacement on macOS
- **Linux nginx.conf tests** - Validates include addition on Linux distros
- **MageBoxDir tests** - Ensures correct home directory resolution for different users
- **Include directive tests** - Verifies correct path generation per user

---

## [0.16.4] - 2025-12-16

### Nginx Configuration Improvements

Improved nginx.conf modification for better reliability:

- **Replace instead of append** - Now replaces invalid `include servers/*;` with MageBox include instead of adding after it
- **Comment out invalid includes** - If MageBox is already configured but `include servers/*;` is still present, it gets commented out
- **Fresh install support** - Handles nginx installs without `include servers/*;` by adding include to http block
- **Better error handling** - Clearer error messages when nginx.conf structure is unexpected

---

## [0.16.3] - 2025-12-16

### Cross-Platform Nginx Configuration Fix

Extended the explicit nginx include approach to all platforms:

- **Unified approach** - All platforms (macOS and Linux) now use explicit include directives instead of symlinks
- **Removed symlink code** - Eliminated the `setupNginxSymlink()` function entirely
- **Linux compatibility** - Fixes the same "include tries to load directory" issue on Ubuntu/Debian that was fixed on macOS in v0.16.2

**Affected platforms:**
- macOS (Homebrew) - already fixed in v0.16.2
- Ubuntu/Debian - previously used symlinks in `/etc/nginx/sites-enabled/`
- Fedora/Arch - already used direct includes

This ensures consistent, reliable nginx configuration across all supported platforms.

---

## [0.16.2] - 2025-12-16

### macOS Nginx Configuration Fix

Fixed nginx vhost loading on macOS with Homebrew:

- **Explicit include directive** - Replaced symlink approach with direct include line in `nginx.conf`
- **Fixes `include servers/*` issue** - The default Homebrew nginx pattern `include servers/*;` was trying to load the MageBox symlinked directory as a file instead of loading `.conf` files from it
- **Cleaner integration** - Adds `include ~/.magebox/nginx/vhosts/*.conf;` directly after the `include servers/*;` line

This fixes the "unknown directive" error that occurred on fresh macOS installations when nginx tried to interpret the symlinked directory as a configuration file.

---

## [0.16.1] - 2025-12-16

### Bootstrap Reliability Fix

Fixed bootstrap failure on clean macOS installations:

- **Mailpit vhost generation moved earlier** - Now generates Mailpit vhost before nginx config test
- **Ensures vhosts directory is not empty** - Nginx config test no longer fails due to empty include directory
- **Smoother first-time setup** - Bootstrap completes successfully without manual intervention

### Enhanced `magebox check` Command

Added SSL infrastructure status to the check command:

- **mkcert status** - Shows if mkcert is installed and provides install command if missing
- **Local CA status** - Verifies if mkcert's local Certificate Authority is installed and trusted
- **Better SSL debugging** - Helps diagnose certificate issues before they cause problems

```bash
magebox check
# Now shows:
# SSL Certificates
#   mkcert          OK  Installed
#   Local CA        OK  Installed and trusted
#   myproject.test  OK  Valid certificate
```

---

## [0.16.0] - 2025-12-16

### Configurable TLD (Top-Level Domain)

You can now customize the top-level domain used for all MageBox projects:

```bash
# Change from .test to .local
magebox config set tld local

# Change to .dev
magebox config set tld dev
```

**Features:**
- All domain generation now uses the configured TLD from global config
- DNS automatically reconfigures when TLD changes
- dnsmasq and systemd-resolved configs use dynamic TLD
- macOS resolver file created at `/etc/resolver/<tld>`
- Default remains `.test` for compatibility

**Example workflow:**
```bash
# Set your preferred TLD
magebox config set tld local

# New projects will use .local
magebox init mystore
# Creates mystore.local

# Existing projects update on restart
magebox restart
```

::: tip
The `.test` TLD is recommended as it's reserved for testing and won't conflict with real domains.
:::

---

## [0.15.2] - 2025-12-16

### DNS Improvements

**Fixed systemd-resolved port conflict:**
- dnsmasq now listens on `127.0.0.2:53` on Linux to avoid conflicts with systemd-resolved
- Fixes issue on EndeavourOS and other Arch-based distros where dnsmasq and systemd-resolved competed for port 53
- macOS continues to use `127.0.0.1:53` (no change)

**Auto-configure dns_mode in bootstrap:**
- Bootstrap now automatically sets `dns_mode: dnsmasq` when dnsmasq is successfully configured
- Eliminates need to manually run `magebox config set dns_mode dnsmasq`
- `magebox start` no longer asks for sudo password when dnsmasq is working

---

## [0.15.1] - 2025-12-16

### macOS Fixes

**Port forwarding fix:**
- Fixed pf (packet filter) rules not working properly on macOS
- Now properly integrates with `/etc/pf.conf` using anchors
- LaunchDaemon loads main pf.conf instead of just the anchor file

**Docker Compose detection:**
- Fixed "unknown shorthand flag: 'f'" error with OrbStack/Colima
- Better detection using `docker compose ls` verification
- Proper fallback to standalone `docker-compose` when needed

---

## [0.15.0] - 2025-12-16

### Verbose Logging

New verbose flags for debugging and troubleshooting:

```bash
magebox -v start     # Basic - shows commands being run
magebox -vv start    # Detailed - shows command output
magebox -vvv start   # Debug - full debug info
```

**Features:**
- Three verbosity levels: `-v` (basic), `-vv` (detailed), `-vvv` (debug)
- Color-coded output: `[verbose]` (cyan), `[debug]` (yellow), `[trace]` (magenta)
- Platform and Linux distro detection logging
- Docker Compose command detection (V2 vs V1 fallback)
- Bootstrap process debugging
- PHP version detection

**Debug output includes:**
- MageBox version and verbosity level
- Environment variables (MAGEBOX_*, DOCKER_*, PATH, HOME)
- Platform type, architecture, home directory
- Linux distro family detection from `/etc/os-release`
- Docker Compose version detection

This makes it much easier to diagnose issues and report bugs with full context.

---

## [0.14.5] - 2025-12-16

### Expanded Linux Distribution Support

New test containers and improved compatibility:

- **Debian 12** - Added test container with sury.org PHP repository
- **Rocky Linux 9** - Added test container with Remi PHP repository
- **Improved distro detection** - Better support for derivative distributions:
  - Proper parsing of `/etc/os-release` (handles quoted values)
  - `ID_LIKE` support for derivatives (EndeavourOS, Pop!_OS, Garuda, etc.)
  - Warning for untested but compatible distros instead of hard failure

**Bug Fixes:**
- Fixed EndeavourOS bootstrap (quoted values in os-release)
- Fixed Ubuntu PHP installation (removed non-existent `php-sodium` package)
- Fixed OpenSearch version (updated from 2.12 to 2.19.4)
- Fixed self-update permissions (automatic sudo for /usr/local/bin)

---

## [0.14.4] - 2025-12-16

### Self-Hosted GitLab & Bitbucket Support

MageBox now supports self-hosted GitLab and Bitbucket instances:

```bash
# Self-hosted GitLab
magebox team add myteam \
  --provider=gitlab \
  --org=mygroup \
  --url=https://gitlab.mycompany.com \
  --auth=ssh

# Self-hosted Bitbucket Server
magebox team add myteam \
  --provider=bitbucket \
  --org=MYPROJECT \
  --url=https://bitbucket.mycompany.com \
  --auth=ssh
```

**Features:**
- Custom URL support for GitLab CE/EE and Bitbucket Server/Data Center
- `magebox team myteam repos` now works with self-hosted instances
- Clone URLs automatically use the custom host
- Better error messages when authentication is required

---

## [0.14.3] - 2025-12-16

### Bug Fix

Fixed installer hanging when running via `curl | bash`. The alias selection prompt now auto-detects non-interactive mode and uses the default `mbox` alias.

---

## [0.14.2] - 2025-12-16

### Bug Fix

Fixed installer checksum verification failure. The download info message was being captured with the filename, causing checksum verification to fail.

---

## [0.14.1] - 2025-12-15

### Short Command Aliases

The install script now offers interactive alias selection:

```bash
curl -fsSL https://get.magebox.dev | bash
```

```
Short Command Aliases

Create shorter command aliases for faster typing:

  1) mbox        - recommended, descriptive
  2) mb          - shortest (2 chars)
  3) Both        - create both aliases
  4) Skip        - use only 'magebox'

Choose [1-4, default: 1]:
```

Now you can use `mbox` (or `mb`) instead of `magebox`:

```bash
mbox start
mbox stop
mbox test all
```

### Installer Improvements

- Version number now displayed in banner
- ASCII logo matches CLI output

---

## [0.14.0] - 2025-12-15

### Testing & Code Quality Commands

New comprehensive testing commands to run PHPUnit, PHPStan, PHPCS, and PHPMD directly from MageBox:

```bash
# Install all testing tools
magebox test setup

# Run unit tests
magebox test unit
magebox test unit --filter=ProductTest

# Run integration tests with RAM-based MySQL (FAST!)
magebox test integration --tmpfs
magebox test integration --tmpfs --tmpfs-size=2g

# Run static analysis
magebox test phpstan --level=5
magebox test phpcs --standard=Magento2
magebox test phpmd --ruleset=cleancode,design

# Run all checks (except integration)
magebox test all

# Check tool status
magebox test status
```

**Features:**
- **`magebox test setup`** - Interactive wizard to install PHPUnit, PHPStan, PHPCS, PHPMD
- **`magebox test unit`** - Run PHPUnit unit tests with filter and testsuite options
- **`magebox test integration`** - Run Magento integration tests with tmpfs support
- **`magebox test phpstan`** - Run PHPStan static analysis (levels 0-9)
- **`magebox test phpcs`** - Run PHP_CodeSniffer with Magento2 or PSR12 standards
- **`magebox test phpmd`** - Run PHP Mess Detector with configurable rulesets
- **`magebox test all`** - Run all tests except integration (ideal for CI/CD)
- **`magebox test status`** - Show installed tools and configuration status

### Tmpfs MySQL for Integration Tests

Run MySQL entirely in RAM for **10-100x faster** integration tests:

```bash
# Fast integration tests with RAM-based MySQL
magebox test integration --tmpfs

# Allocate more RAM for larger test suites
magebox test integration --tmpfs --tmpfs-size=2g

# Keep container running for repeated test runs
magebox test integration --tmpfs --keep-alive

# Use specific MySQL version
magebox test integration --tmpfs --mysql-version=8.4
```

Container naming: `mysql-{version}-test` (e.g., `mysql-8-0-test`)

### PHPStan Magento Extension

Automatic support for [bitexpert/phpstan-magento](https://github.com/bitExpert/phpstan-magento):
- Factory method analysis for ObjectManager
- Auto-generates `phpstan.neon` with extension includes when installed

**Configuration in `.magebox.yaml`:**
```yaml
testing:
  phpstan:
    level: 1
    paths: ["app/code"]
  phpcs:
    standard: "Magento2"
    paths: ["app/code"]
  phpmd:
    ruleset: "cleancode,codesize,design"
  integration:
    tmpfs: true
    tmpfs_size: "2g"
```

See the new [Testing & Code Quality](/guide/testing-tools) documentation for full details.

---

## [0.13.3] - 2025-12-15

### Bug Fixes

- **Test containers**: Added missing Magento-required PHP extensions to all Dockerfiles
  - Ubuntu (24.04, 22.04, ARM64): `bcmath`, `gd`, `intl`, `mysql`, `soap`
  - Fedora 42: `bcmath`, `gd`, `intl`, `mysqlnd`, `soap`
  - Arch Linux: `php-gd`, `php-intl`, `php-sodium`

---

## [0.13.2] - 2025-12-15

### Development & Production Modes, Queue Management

Quickly switch between optimized development and production configurations:

```bash
# Development mode - debugging enabled
magebox dev
# OPcache: disabled (code changes immediately)
# Xdebug: enabled (step debugging)
# Blackfire: disabled

# Production mode - performance optimized
magebox prod
# OPcache: enabled (faster execution)
# Xdebug: disabled (no overhead)
# Blackfire: disabled (enable manually when needed)
```

**RabbitMQ Queue Management:**

```bash
# View queue status with message counts
magebox queue status

# Purge all queues (use with caution!)
magebox queue flush

# Run a specific queue consumer
magebox queue consumer product_action_attribute.update

# Run all consumers via cron
magebox queue consumer --all
```

**Features:**
- **`magebox dev`** - Switch to development mode (OPcache off, Xdebug on)
- **`magebox prod`** - Switch to production mode (OPcache on, debuggers off)
- **`magebox queue status`** - View RabbitMQ queue status
- **`magebox queue flush`** - Purge all messages from queues
- **`magebox queue consumer`** - Run Magento queue consumers
- **Config Fix** - PHP INI overrides now properly merge from local config

---

## [0.13.1] - 2025-12-15

### Database Snapshots & Security

Quick backup and restore for your databases:

```bash
# Create a snapshot (with optional name)
magebox db snapshot create           # Auto-named with timestamp
magebox db snapshot create mybackup  # Named snapshot

# List and manage snapshots
magebox db snapshot list             # Show all snapshots
magebox db snapshot restore mybackup # Restore from snapshot
magebox db snapshot delete mybackup  # Delete snapshot
```

**Features:**
- **`magebox db snapshot`** - Full snapshot management for quick backup/restore
- **Compressed storage** - Snapshots use gzip compression
- **Per-project snapshots** - Stored in `~/.magebox/snapshots/{project}/`
- **SSH Security** - SFTP connections now verify host keys against `~/.ssh/known_hosts`

---

## [0.13.0] - 2025-12-15

### Multi-Project Management, Restart & Test Infrastructure

- **`magebox start --all`** - Start all discovered MageBox projects at once
- **`magebox stop --all`** - Stop all running MageBox projects at once
- **`magebox restart`** - Restart all services for a project (stop + start)
- **`magebox restart --all`** - Restart all MageBox projects
- **`magebox uninstall`** - Clean uninstall of MageBox components
- **`--dry-run` flag** - Preview what would happen without making changes
- **Test Mode** (`MAGEBOX_TEST_MODE=1`) - Run MageBox in containers without Docker
- **Docker Integration Tests** - Comprehensive test suite for multiple distributions

```bash
# Manage all projects
magebox start --all    # Start all projects
magebox stop --all     # Stop all projects
magebox restart        # Restart current project
magebox restart --all  # Restart all projects

# Uninstall MageBox
magebox uninstall              # Interactive uninstall
magebox uninstall --force      # Skip confirmation
magebox uninstall --dry-run    # Preview uninstall

# Run integration tests
./test/containers/run-tests.sh              # All distributions
./test/containers/run-tests.sh ubuntu       # Single distro
./test/containers/run-tests.sh --full       # Include Magento install
```

---

## [0.12.14] - 2025-12-15

### Multi-Domain Store Code Fix

- **Fixed `mage_run_code` and `mage_run_type` not being passed to nginx** - Domain configs now correctly use `mage_run_code` and `mage_run_type` YAML keys
- **Dynamic `MAGE_RUN_TYPE`** - No longer hardcoded to `store`, now reads from domain config (supports `store` or `website`)

Example multi-store configuration:

```yaml
domains:
  - host: mystore.test
    root: pub
  - host: wholesale.test
    root: pub
    mage_run_code: wholesale
    mage_run_type: website
  - host: uk.mystore.test
    root: pub
    mage_run_code: uk_store
    mage_run_type: store
```

---

## [0.12.13] - 2025-12-15

### Xdebug Fedora/Remi Fix

- **Fixed Xdebug enable/disable on Fedora** - Now supports Remi PHP paths (`/etc/opt/remi/php{ver}/php.d/`)
- **Uses sudo sed** for Xdebug ini modifications (required on Fedora)
- **`magebox blackfire on` now properly disables Xdebug** on Fedora before enabling Blackfire

This fixes the 10x performance degradation when both Xdebug and Blackfire were accidentally enabled together.

---

## [0.12.12] - 2025-12-15

### CLI Wrappers & Profiler Improvements

Bootstrap now installs **three shell script wrappers** in `~/.magebox/bin/`:

| Wrapper | Path | Purpose |
|---------|------|---------|
| **php** | `~/.magebox/bin/php` | Automatically uses PHP version from `.magebox.yaml` |
| **composer** | `~/.magebox/bin/composer` | Runs Composer with project's PHP version |
| **blackfire** | `~/.magebox/bin/blackfire` | Uses project's PHP for `blackfire run` commands |

These wrappers walk up the directory tree to find `.magebox.yaml`, extract the PHP version, and execute the correct PHP binary for your platform (Homebrew Cellar on macOS, Ondrej PPA on Ubuntu, Remi on Fedora).

See the [CLI Wrappers guide](/guide/php-wrapper) for full documentation.

### Profiler Fixes

- **Fixed Blackfire agent configuration** - Uses `sudo sed` to update `/etc/blackfire/agent` credentials
- **Fixed Blackfire PHP extension on Fedora** - Uses single `blackfire-php` package (not versioned)
- **Fixed Tideways on Fedora 41+** - Downloads RPMs directly (dnf5/cloudsmith compatibility)
- **GPG key import** - Imports Blackfire and Tideways GPG keys before installing packages
- **Non-fatal xdebug disable** - Enabling Blackfire/Tideways no longer fails if xdebug ini is missing

---

## [0.12.11] - 2025-12-15

### Tideways Repository Fix & Profiler Sudoers

- **Fixed Tideways repository URL for Fedora** - Changed from `fedora/$releasever/$basearch` to just `$basearch`
- **Added passwordless sudo for Blackfire/Tideways**:
  - Fedora: `dnf install -y blackfire*`, `tideways*`
  - Ubuntu: `apt install -y blackfire*`, `tideways*`
  - Arch: `pacman -S --noconfirm *blackfire*`, `*tideways*`
- **Added systemctl sudoers for blackfire-agent** - start/stop/restart/enable

---

## [0.12.10] - 2025-12-14

### Blackfire & Tideways in Bootstrap

Bootstrap now automatically installs **Blackfire** and **Tideways** profilers for all PHP versions:

- **Fedora**: Adds Blackfire/Tideways repos, installs agent and PHP extensions
- **Ubuntu/Debian**: Adds repos with GPG keys, installs packages
- **macOS**: Uses Homebrew tap and pecl
- **Arch**: Uses pecl (agent must be installed from AUR)

After bootstrap, configure credentials with:
```bash
magebox blackfire config --server-id=XXX --server-token=XXX
magebox blackfire on
```

---

## [0.12.9] - 2025-12-14

### Varnish Integration Fix

Fixed Varnish backend connectivity on Linux:

- Use `host.docker.internal` instead of host LAN IP for Varnish backend
- Add dedicated backend port (8080) for Varnish on Linux
- Nginx now listens on port 8080 as backend when Varnish is enabled

The full Varnish flow now works: `Client → Nginx (443) → Varnish (6081) → Nginx (8080) → PHP-FPM`

---

## [0.12.8] - 2025-12-14

### PHP INI Configuration in Bootstrap

Bootstrap now **automatically configures PHP INI settings** for optimal Magento development:

- Sets `memory_limit = -1` (unlimited) for CLI
- Sets `max_execution_time = 18000` for long-running CLI scripts
- Works on all platforms: Fedora (Remi), Ubuntu (Ondrej PPA), macOS (Homebrew), Arch

### Fedora 43 Support

Added Fedora 43 to officially supported Linux distributions.

### PHP Memory Limit Improvements

- PHP-FPM pool now uses `memory_limit = 768M` (increased from 756M)
- PHP wrapper adds `-d memory_limit=-1` for unlimited CLI memory

---

## [0.12.7] - 2025-12-14

### PHP Memory Limit Fix

Fixed "Allowed memory size exhausted" errors during Magento compilation:

- PHP-FPM pool: increased to 768M
- PHP CLI wrapper: now passes `-d memory_limit=-1`

---

## [0.12.6] - 2025-12-14

### Fixed Bootstrap Sudoers Creation

Fixed a bug where the sudoers file (`/etc/sudoers.d/magebox`) was not created during bootstrap because `WriteFile()` used `RunSudoSilent()` which doesn't connect stdin for password prompts. Now uses `RunSudo()` to allow interactive password entry.

---

## [0.12.5] - 2025-12-14

### Simplified Composer Wrapper Script

The **Composer wrapper shell script** at `~/.magebox/bin/composer` now uses the PHP wrapper instead of duplicating version detection logic:
- Single source of truth for PHP version detection
- Reduced code complexity
- The wrapper finds the real Composer binary and executes it via `~/.magebox/bin/php`

### Removed `magebox composer` Command

The `magebox composer` command has been removed - it's no longer needed since the `~/.magebox/bin/composer` wrapper script handles automatic PHP version detection. Just use `composer` directly with `~/.magebox/bin` in your PATH.

See the [CLI Wrappers guide](/guide/php-wrapper) for details on how the wrapper system works.

---

## [0.12.4] - 2025-12-14

### Automatic PATH Configuration

Bootstrap now **automatically adds `~/.magebox/bin` to your PATH** - no manual configuration needed!

- Automatically updates shell config (`.zshrc`, `.bashrc`, `.bash_profile`)
- Supports zsh (macOS default), bash, and fish shells
- Creates `.zshrc` if it doesn't exist on macOS
- Shows reload instructions after bootstrap completes

After bootstrap, just run:
```bash
source ~/.zshrc  # or restart your terminal
```

---

## [0.12.3] - 2025-12-14

### New `magebox composer` Command

Run Composer using the PHP version configured in your project's `.magebox.yaml`:

```bash
magebox composer install
magebox composer require vendor/package
magebox composer update
```

The command automatically:
- Uses the correct PHP version for your project
- Sets `COMPOSER_MEMORY_LIMIT=-1` for large Magento projects
- Passes all arguments directly to Composer

---

## [0.12.2] - 2025-12-14

### Composer Install in Fetch Workflow

The `magebox fetch` command now automatically runs `composer install` after cloning a team project, ensuring dependencies are ready immediately.

### Enhanced Fedora SELinux Support

Bootstrap now configures **persistent** SELinux fcontext rules using `semanage`:

- `httpd_var_run_t` context for `~/.magebox/run/` (PHP-FPM sockets)
- `httpd_config_t` context for `~/.magebox/nginx/` and `~/.magebox/certs/`

These rules survive system updates and `restorecon` operations.

### Fixed PHP-FPM Socket Location

Moved PHP-FPM sockets from `/tmp/magebox/` to `~/.magebox/run/` to avoid nginx PrivateTmp isolation issues on systems with `PrivateTmp=yes` in nginx.service.

### Sudoers for /etc/hosts

Bootstrap now adds passwordless sudo rules for `/etc/hosts` modifications, eliminating prompts during `magebox start/stop`.

---

## [0.12.1] - 2025-12-14

### SELinux Support (Fedora)

MageBox bootstrap now automatically configures SELinux on Fedora:

- **Network connections** - Sets `httpd_can_network_connect` boolean so nginx can proxy to Docker containers
- **Config file access** - Configures `httpd_config_t` context on `~/.magebox/nginx/` and `~/.magebox/certs/`

### Simplified PHP-FPM Configuration

PHP-FPM configuration on Linux has been simplified:

- No longer modifies PHP-FPM config files
- Uses default repository log paths (avoids permission and SELinux issues)
- Reduces potential for configuration conflicts

### Documentation

- Added comprehensive SELinux troubleshooting guide
- Updated bootstrap documentation with SELinux configuration details
- Updated Linux installers documentation with SELinux tips

---

## [0.12.0] - 2025-12-14

### Installation Improvements

MageBox is now easier to install than ever:

- **Homebrew Tap**: Install with `brew install qoliber/magebox/magebox`
- **Install Script**: One-liner installation with `curl -fsSL https://get.magebox.dev | bash`

### Non-Interactive CLI Flags

All interactive commands now support CLI flags for scripting and automation:

- **`magebox team add`** now supports:
  - `--provider` - Repository provider (github/gitlab/bitbucket)
  - `--org` - Organization/namespace name
  - `--auth` - Authentication method (ssh/token)
  - `--asset-provider` - Asset storage provider (sftp/ftp)
  - `--asset-host`, `--asset-port`, `--asset-path`, `--asset-username` - Asset storage configuration

- **`magebox blackfire config`** now supports:
  - `--server-id`, `--server-token` - Server credentials
  - `--client-id`, `--client-token` - Client credentials

- **`magebox tideways config`** now supports:
  - `--api-key` - Tideways API key

### Bug Fixes

- Fixed dynamic team subcommand routing (`magebox team <teamname> <subcommand>` now works correctly)

### Other

- Improved CI workflow: removed deprecated macOS-13 runner
- All interactive commands fall back to prompts only when required flags are not provided

## [0.11.0] - 2025-12-14

### Team Collaboration

MageBox now supports team collaboration features for sharing project configurations across your team:

- **Team Management**
  - `magebox team add <name>` - Add a new team with repository and asset storage configuration
  - `magebox team list` - List all configured teams
  - `magebox team remove <name>` - Remove a team configuration
  - `magebox team <name> show` - Show team configuration details
  - `magebox team <name> repos` - Browse repositories in team namespace (with glob filtering)

- **Project Management**
  - `magebox team <name> project add` - Add a project to a team
  - `magebox team <name> project list` - List team projects
  - `magebox team <name> project remove` - Remove a project

- **Fetch Command** - One-command project setup:
  - `magebox fetch <team/project>` - Clone repo, download & import database, download & extract media
  - Supports `--branch`, `--no-db`, `--no-media`, `--db-only`, `--dry-run` flags

- **Sync Command** - Keep existing projects up to date:
  - `magebox sync` - Sync latest database and media from asset storage
  - Auto-detects project from git remote
  - Supports `--db`, `--media`, `--backup`, `--dry-run` flags

- **Repository Providers**: GitHub, GitLab, Bitbucket with SSH or token authentication
- **Asset Storage**: SFTP/FTP for database dumps and media files with progress tracking

Team configurations are stored in `~/.magebox/teams.yaml`.

## [0.10.12] - 2025-12-14

### Bug Fixes

- **Fixed Blackfire installation on macOS**: Improved detection and installation via Homebrew
- **Fixed Blackfire PHP extension detection**: Now correctly detects extension in PHP modules
- **Xdebug state restoration**: When disabling Blackfire, Xdebug is automatically re-enabled if it was previously active

### Improvements

- Better profiler state management between Xdebug and Blackfire

## [0.10.11] - 2025-12-13

### New Features

- **Xdebug auto-installation**: Bootstrap now installs Xdebug for all detected PHP versions
  - macOS: Uses `pecl install xdebug`
  - Fedora: Installs `php*-php-xdebug` package
  - Ubuntu/Debian: Installs `php*-xdebug` package
  - Arch: Installs from community repository

### Bug Fixes

- **Fixed service detection**: Improved container detection when compose file is stale
- **Fixed Elasticsearch detection**: Container names like `magebox-elasticsearch-8.17.0` now correctly detected

### Other

- Added MIT License
- Added MageBox logo and favicons

## [0.10.10] - 2025-12-13

### New Features

- **`magebox logs`**: View system.log and exception.log in 2-column split-screen using multitail
- **`magebox report`**: Watch var/report directory for new error reports with auto-refresh

## [0.10.9] - 2025-12-13

### New Features

- **multitail installation**: Bootstrap now installs `multitail` for viewing multiple log files

## [0.10.8] - 2025-12-13

### Bug Fixes

- **Fixed `magebox list` panic**: No longer crashes when parsing vhost files without root path
- **Fixed project discovery**: Now correctly parses `$MAGE_ROOT` variable from vhost files
- **Fixed duplicate domains**: Domains no longer appear twice in project listings

## [0.10.7] - 2025-12-13

### Bug Fixes

- **Fixed `magebox status` service detection**: MySQL, OpenSearch, and Elasticsearch now correctly detected as running
- **Added Elasticsearch to status**: Shows Elasticsearch service status when configured

## [0.10.6] - 2025-12-13

### Bug Fixes

- **Fixed `magebox check` SSL detection**: Now correctly looks for certificates in `~/.magebox/certs/{domain}/cert.pem`
- **Fixed `magebox check` vhost detection**: Now correctly detects `{project}-upstream.conf` pattern

## [0.10.5] - 2025-12-13

### Varnish Full-Page Cache Integration

Complete Varnish integration with automatic Nginx configuration:

- **Automatic Nginx proxy**: When Varnish is enabled, HTTPS traffic is automatically proxied through Varnish
- **Traffic flow**: Browser → Nginx (HTTPS) → Varnish (6081) → Nginx (HTTP) → PHP-FPM
- **Vhost regeneration**: `varnish enable/disable` automatically regenerates Nginx vhosts and reloads Nginx
- **SSL offloading**: Proper `Ssl-Offloaded` and `X-Forwarded-*` headers for Magento compatibility
- **No sudo required**: Nginx reload uses `brew services` on macOS, eliminating password prompts

### Commands

```bash
magebox varnish enable   # Enable Varnish, update Nginx config, reload
magebox varnish disable  # Disable Varnish, restore direct PHP-FPM routing
magebox varnish status   # Show cache statistics and backend health
magebox varnish flush    # Flush all cached content
magebox varnish purge /  # Purge specific URL from cache
```

## [0.10.4] - 2025-12-13

### Varnish Improvements

- **Docker-based Varnish management**: Varnish controller now uses Docker commands for reliable container management
- **Backend connectivity**: Added `host.docker.internal` support for Varnish to reach Nginx backend
- **Enhanced status command**: `magebox varnish status` now shows backend health and cache statistics (hits, misses, requests)
- **Improved documentation**: Added `varnish enable/disable` commands to README and reference docs

### PHP-FPM Lifecycle

- **Smart reload/start**: `magebox start` now reloads PHP-FPM if already running, starts if stopped
- Prevents unnecessary restarts when switching between projects

### Test Improvements

- Fixed nginx vhost tests for upstream separation architecture (from v0.10.3)
- Fixed platform and PHP detector tests for flexible path checking on different systems
- Tests now handle both Homebrew symlink and Cellar paths correctly

## [0.10.3] - 2025-12-13

### Dedicated MageBox PHP-FPM Process

MageBox now runs its own PHP-FPM process instead of relying on system-managed PHP-FPM services. This provides better isolation and control:

- **Per-version PHP-FPM**: Dedicated PHP-FPM process for each PHP version with MageBox-specific configuration
- **Custom php-fpm.conf**: New template at `~/.magebox/php/php-fpm-{version}.conf`
- **PID management**: Process tracking via `~/.magebox/run/php-fpm-{version}.pid`
- **Centralized logs**: Error logs at `~/.magebox/logs/php-fpm/php{version}-error.log`

### Multi-Domain Upstream Fix

- **Fixed duplicate upstream error**: Projects with multiple domains no longer cause nginx "duplicate upstream" errors
- **Separate upstream.conf**: Each project now has a dedicated `{project}-upstream.conf` file
- **Cleaner vhost templates**: Upstream configuration moved out of vhost.conf into its own file

### Lifecycle Improvements

- `magebox start` now properly starts PHP-FPM and reloads Nginx
- `magebox stop` now reloads PHP-FPM and Nginx after removing configurations
- Better process management without requiring `brew services` or `systemctl`

## [0.10.2] - 2025-12-13

### Database Management Commands

New commands for database lifecycle management:

- `magebox db create` - Create project database from config
- `magebox db drop` - Drop project database (with confirmation prompt)
- `magebox db reset` - Drop and recreate project database (with confirmation prompt)

### Improvements

- Database commands now use consistent password constants

## [0.10.1] - 2025-12-13

### Bug Fixes

- Fixed linting issues: unchecked error returns, unused functions, gofmt formatting

## [0.10.0] - 2025-12-13

### Blackfire Profiler Integration

Full Blackfire profiler support for performance analysis:

- `magebox blackfire on` - Enable Blackfire profiling
- `magebox blackfire off` - Disable Blackfire profiling
- `magebox blackfire status` - Show current Blackfire status
- `magebox blackfire install` - Install Blackfire agent and PHP extension
- `magebox blackfire config` - Configure Blackfire credentials

**Platform support:** macOS (Homebrew), Fedora (dnf), Ubuntu/Debian (apt), Arch (AUR)

### Tideways Profiler Integration

Full Tideways profiler support for application performance monitoring:

- `magebox tideways on` - Enable Tideways monitoring
- `magebox tideways off` - Disable Tideways monitoring
- `magebox tideways status` - Show current Tideways status
- `magebox tideways install` - Install Tideways daemon and PHP extension
- `magebox tideways config` - Configure Tideways API key

**Platform support:** macOS (Homebrew), Fedora (dnf), Ubuntu/Debian (apt), Arch (AUR)

### Global Profiling Credentials

- New `profiling` section in `~/.magebox/config.yaml` for secure credential storage
- Environment variable fallback support:
  - Blackfire: `BLACKFIRE_SERVER_ID`, `BLACKFIRE_SERVER_TOKEN`, `BLACKFIRE_CLIENT_ID`, `BLACKFIRE_CLIENT_TOKEN`
  - Tideways: `TIDEWAYS_API_KEY`
- Automatic Xdebug disable when enabling profilers to avoid conflicts

### New Packages

- `internal/blackfire/` - Blackfire manager, installer, and types
- `internal/tideways/` - Tideways manager, installer, and types

## [0.9.1] - 2025-12-13

### Additional Templates

Continued refactoring to template-based configuration for better maintainability:

- **`xdebug.ini.tmpl`** - Xdebug INI configuration with `XdebugConfig` struct:
  ```ini
  xdebug.mode={{.Mode}}
  xdebug.start_with_request={{.StartWithRequest}}
  xdebug.client_host={{.ClientHost}}
  xdebug.client_port={{.ClientPort}}
  xdebug.idekey={{.IdeKey}}
  ```

- **`not-installed-message.tmpl`** - Platform-aware PHP installation instructions:
  - macOS: `brew install php@8.2`
  - Fedora: `sudo dnf install -y php82-php-fpm...`
  - Ubuntu: `sudo apt install php8.2-fpm...`
  - Arch: `sudo pacman -S php php-fpm`

- **`systemd-resolved.conf.tmpl`** - DNS configuration for systemd-resolved
- **`not-installed-error.tmpl`** - mkcert installation instructions with platform-specific commands

### New Structs

- `XdebugConfig` - Customizable Xdebug settings (mode, client_host, client_port, idekey)
- `SystemdResolvedConfig` - DNS configuration settings

## [0.9.0] - 2025-12-13

### Platform-Specific Installers

- **New `internal/bootstrap/` package** with platform-specific installers:
  - `installer/darwin.go` - macOS (Homebrew) support
  - `installer/fedora.go` - Fedora/RHEL/CentOS (dnf + Remi) support
  - `installer/ubuntu.go` - Ubuntu/Debian (apt + Ondrej PPA) support
  - `installer/arch.go` - Arch Linux (pacman) support
- **OS version validation** during bootstrap
- **Explicit supported platform versions** in bootstrap help
- Bootstrap now uses `Installer` interface for clean platform abstraction

### Template-Based Configuration

- **Template-based configuration generation** using Go's `text/template` engine
- New template files:
  - `internal/project/templates/env.php.tmpl` - Magento env.php with conditionals
  - `internal/dns/templates/dnsmasq.conf.tmpl` - dnsmasq configuration
  - `internal/dns/templates/hosts-section.tmpl` - /etc/hosts entries
- `EnvPHPData` struct for clean template data separation
- Refactored `internal/project/env.go` from 310 lines to 195 lines using templates

### Mailpit Integration

- **Mailpit always enabled** for local development safety (prevents accidental real emails)
- Mailpit Docker service now always included in docker-compose.yml
- Mailpit PHP-FPM configuration (sendmail_path) always enabled

### Bug Fixes

- Fixed `ValidationError` index conversion bug (now works for indices > 9)
- Fixed missing Varnish merge in config loader
- Fixed PHP-FPM config path for Fedora/RHEL (Remi repository)
- Fixed silently ignored errors in bootstrap.go PHP-FPM setup
- Fixed MySQL/MariaDB memory configuration now properly used

### Code Quality

- Added constants for magic numbers in `cmd/magebox/new.go`
- Database credentials now use constants (`DefaultDBRootPassword`, etc.)
- Test coverage for `internal/project` improved from 19.4% to 63.7%

## [0.8.0] - 2025-12-13

### Documentation

- **New VitePress documentation site** with comprehensive guides
- Added dedicated service pages: Nginx, PHP-FPM, Database, Redis, OpenSearch, RabbitMQ, Mailpit, Varnish
- Architecture documentation with request flow diagrams
- Roadmap page for planned features

### Mailpit Integration

- Automatic Mailpit SMTP configuration in generated `env.php`
- Prevents accidental emails to real addresses during development

## [0.7.1] - 2025-12-12

### Domain Management

- **New `magebox domain` command** for managing project domains:
  - `magebox domain add <host>` - Add a domain with optional flags:
    - `--store-code` - Magento store code for multi-store setup
    - `--root` - Document root (default: `pub`)
    - `--ssl` - Enable/disable SSL (default: true)
  - `magebox domain remove <host>` - Remove a domain
  - `magebox domain list` - List all configured domains
- Store code support (`MAGE_RUN_CODE`) in nginx vhost configuration
- Automatic SSL certificate generation when adding domains

### Code Refactoring

- Split monolithic `main.go` (3550 lines) into 26 focused command files
- Each command now has its own file with self-contained init()
- Improved code maintainability and organization

## [0.7.0] - 2025-12-12

### Database Improvements

- Refactored db import/export/shell to use `docker exec` directly
- Added `--no-tablespaces` flag to mysqldump for better compatibility
- Added MySQL 5.7 support (port 33057)

### Bug Fixes

- Fixed OpenSearch/Elasticsearch plugin install loop on restart
- Fixed SSL tests for unified `~/.magebox/certs` path

### Changes

- Portainer disabled by default (enable with `magebox config set portainer true`)

## [0.6.1] - 2025-12-11

### Quick Mode Improvements

- Auto-execute Magento installation in `magebox new --quick` command
- Streamlined project creation workflow

## [0.6.0] - 2025-12-11

### Linux/Fedora Support Improvements

- **Nginx user configuration**: Automatically configures nginx to run as current user on Linux (required for SSL cert access)
- **PHP-FPM logging**: Centralized logging to `/var/log/magebox/` on Fedora/RHEL
- **Sudoers configuration**: Passwordless sudo for nginx/php-fpm control
- **DNS improvements**:
  - Linux: dnsmasq + systemd-resolved configuration for `.test` wildcard DNS
  - macOS: Creates `/etc/resolver/test` for wildcard resolution

### Bug Fixes

- Fixed errcheck linter warnings in Linux support code

## [0.5.2] - 2025-12-10

### Varnish Integration

- Added Varnish Docker container support
- Added `magebox varnish status`, `purge`, and `flush` commands
- VCL configuration generation for Magento

### Admin Commands

- Added admin user creation commands
- Improved project management

### Bug Fixes

- Fixed linter warnings (errcheck)
- Various code quality improvements

## [0.5.1] - 2025-12-10

### Bug Fixes

- Fixed errcheck linter warnings in multiple packages
- Improved error handling throughout the codebase

## [0.5.0] - 2025-12-10

### Varnish Docker Integration

- Full Varnish support with Docker container
- Cache management commands
- VCL template generation

## [0.4.0] - 2025-12-10

### Composer Templates

- Added Composer configuration templates
- Improved PHP Cellar detection on macOS
- Better Homebrew PHP version detection

## [0.3.2] - 2025-12-10

### OpenSearch/Elasticsearch Enhancements

- **ICU and Phonetic plugins**: Automatically installed for both OpenSearch and Elasticsearch
- **Memory configuration**: Added configurable RAM allocation (default: 1GB)
- **Updated versions**: Default OpenSearch updated to 2.19.4
- **Object format**: Search engine configuration now supports `version` and `memory` properties

### Project Creation Improvements

- Composer now runs with explicit PHP binary to ensure correct version
- Added RabbitMQ to quick mode (`magebox new --quick`)
- Full `setup:install` command shown after project creation with all service parameters
- Displays which PHP version and binary is being used during installation

### Code Cleanup

- Removed `magebox cli` command (use `php bin/magento` directly - PHP wrapper handles version)
- DNS cleanup skipped when using dnsmasq mode

## [0.3.1] - 2025-12-10

### Documentation

- Added comprehensive migration guide from Laravel Herd to MageBox
- Step-by-step instructions for cleaning up Herd configuration
- Troubleshooting section for common migration issues

## [0.3.0] - 2025-12-10

### Template Refactoring

- Extracted all templates from Go code into separate `.tmpl` files
- Added comprehensive template variable documentation
- Created README files for each template directory
- Added template validation tests
- Templates remain embedded in binary at compile time

## [0.2.3] - 2025-12-10

### PHP Version Management

- **Smart PHP wrapper shell script**: Added `~/.magebox/bin/php` shell script that automatically detects and uses correct PHP version per project
- The wrapper walks directory tree to find `.magebox.yaml` and uses configured PHP version
- Fixed `magebox php` command to properly restart services when switching versions
- Added PHP wrapper installation to bootstrap process

See the [CLI Wrappers guide](/guide/php-wrapper) for details.

### Bug Fixes

- Fixed PHP-FPM logs to use writable directory (`~/.magebox/logs/php-fpm/`)
- Fixed service lifecycle when switching PHP versions

## [0.2.2] - 2025-12-10

### Configuration

- **Renamed config files** for better editor support:
  - `.magebox` → `.magebox.yaml`
  - `.magebox.local` → `.magebox.local.yaml`
- Full backward compatibility maintained for legacy filenames

## [0.2.0] - 2025-12-09

### Features

- **Port forwarding (macOS)**: Nginx runs on 8080/8443, pf forwards from 80/443
- **No sudo after bootstrap**: All daily commands run as your user
- **PHP INI configuration**: Per-project PHP settings via `php_ini` in config
- **Xdebug support**: Easy configuration through `php_ini` settings

### Improvements

- Nginx configuration improvements
- Better PHP-FPM pool management
- Enhanced error handling

## [0.1.0] - 2024

### Initial Release

MageBox v0.1.0 is the first public release of the native Magento development environment.

#### Features

- **Native PHP/Nginx**: Run PHP-FPM and Nginx directly on your machine for maximum performance
- **Docker Services**: MySQL, MariaDB, Redis, OpenSearch, Elasticsearch, RabbitMQ, Mailpit, Varnish
- **Multi-PHP Support**: Switch PHP versions per project (8.1, 8.2, 8.3, 8.4)
- **Auto SSL**: Automatic HTTPS with mkcert integration
- **Project Discovery**: Automatic detection of MageBox projects
- **Custom Commands**: Define project-specific shortcuts
- **Database Tools**: Import, export, and shell access
- **Redis Tools**: Shell, flush, and info commands
- **Log Viewer**: Built-in log tailing with pattern matching
- **Self-Update**: Update MageBox from GitHub releases
- **New Project Wizard**: Interactive Magento/MageOS installation

#### Supported Platforms

- macOS (Intel and Apple Silicon)
- Linux (Ubuntu, Debian, Fedora)
- Windows WSL2 (Ubuntu, Fedora)

#### Database Support

- MySQL 5.7, 8.0, 8.4
- MariaDB 10.4, 10.6, 11.4

---

## Roadmap

Future improvements planned:

- Xdebug GUI integration
- Performance profiling tools
- Database snapshots
- Custom Docker service support
- Team configuration sharing
- IDE plugins

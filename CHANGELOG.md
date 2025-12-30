# Changelog

All notable changes to MageBox will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.5] - 2025-12-30

### Fixed

- **Database Name Sanitization** - MySQL database names now replace hyphens with underscores:
  - Project `m2-layout-xml-compiler` creates database `m2_layout_xml_compiler`
  - Added `DatabaseName()` method to Config for consistent sanitization
  - Applied to all database operations: create, import, export, shell, reset, snapshots

- **Search Plugins Volume Definition** - Fixed Docker Compose volume errors:
  - OpenSearch and Elasticsearch plugins volumes now properly defined
  - Fixes "undefined volume" error on `mbox start`

## [1.0.4] - 2025-12-30

### Fixed

- **ImageMagick PHP Extension** - Bootstrap now properly installs imagick for all PHP versions:
  - Fixed `InstallImagick` on Fedora to install `php*-php-pecl-imagick-im7`
  - Fixed `InstallImagick` on Ubuntu to install `php*-imagick`
  - Added `php-imagick` to Arch Linux default PHP packages
  - macOS already working via PECL

- **OpenSearch/Elasticsearch Reliability** - Plugins now persist across container restarts:
  - Added plugins volume to prevent re-downloading on every start
  - Fixes restart loops caused by temporary DNS/network issues

- **PHP-FPM Pool Isolation** - Fixed pool configuration path bug:
  - Each PHP version now uses isolated pool directory (`pools/8.1/`, `pools/8.3/`, etc.)
  - Prevents version conflicts when running multiple PHP versions

## [1.0.3] - 2025-12-29

### Added

- **New `magebox clone` Command** - Dedicated command for cloning team projects:
  - Clones repository from configured Git provider
  - Creates `.magebox.yaml` if not present
  - Runs `composer install` automatically
  - Optional `--fetch` flag to also download database and media

- **SELinux Fix for Fedora** - Bootstrap now sets `httpd_read_user_content` boolean:
  - Fixes "Permission denied" errors when nginx accesses files in home directories
  - Resolves issues with PHP-created files being inaccessible to nginx

### Changed

- **Simplified `magebox fetch` Command** - Now works on existing projects:
  - Reads project name from local `.magebox.yaml`
  - Searches team asset storage for matching files
  - Downloads and imports database by default
  - Use `--media` flag to also download media files
  - Use `--team` flag to specify team explicitly

### Improved

- **Better Error Messages** - Clearer guidance when project not found in asset storage
- **Unit Tests** - Added tests for Cloner and AssetFetcher
- **E2E Test Setup** - Added integration test framework with Docker SFTP

## [1.0.2] - 2025-12-26

### Fixed

- **macOS Port Forwarding Persistence** - Completely fixed port forwarding rules not persisting after sleep/restart:
  - Added `KeepAlive.NetworkState` trigger - rules now reload automatically when network comes up after wake
  - Fixed rule detection using `-sn` (NAT rules) instead of `-sr` (filter rules)
  - Added `sleep 2` delay to wait for network stability after wake
  - Reduced check interval from 60s to 30s for faster recovery
  - Added `ThrottleInterval: 5` to prevent rapid re-execution
  - LaunchDaemon now correctly reloads anchor rules directly instead of full pf.conf

- **LaunchDaemon Auto-Upgrade** - Running `magebox bootstrap` now automatically upgrades old LaunchDaemon versions:
  - Version marker (`<!-- MageBox-Version-X -->`) tracks plist version
  - Bootstrap detects outdated versions and reinstalls with new configuration
  - Existing users get sleep/wake fixes automatically

### Improved

- **Refactored `addAnchorToPfConf()`** - Split complex function into smaller, focused functions for better maintainability
- **Future-proofed `insertVersionDots()`** - Better handling of MariaDB 10.x/11.x version formats in Docker Compose

### Added

- **Makefile** - Added development tooling with `make lint`, `make test`, `make build` targets

## [1.0.1] - 2025-12-23

### Added

- **Per-Domain Nginx Logging** - Each domain now gets its own access and error logs stored in `~/.magebox/logs/nginx/`
  - Access logs: `~/.magebox/logs/nginx/<domain>-access.log`
  - Error logs: `~/.magebox/logs/nginx/<domain>-error.log`

- **Sodium PHP Extension** - Bootstrap now installs the `sodium` extension for all PHP versions. Required for Argon2i password hashing in Magento.

### Fixed

- **PHP Wrapper Local Override** - Fixed bug where PHP wrapper ignored `.magebox.local.yaml` PHP version. Local config now correctly takes priority over main config
- **PHP Version Switching** - Fixed critical bug where switching PHP versions (e.g., `mbox php 8.1`) would fail because all pool configs were in a single directory. Now pools are organized by version: `~/.magebox/php/pools/<version>/`
- **macOS Port Forwarding Reliability** - Added `StartInterval` to LaunchDaemon to re-apply pf rules every 60 seconds (catches resets by other apps like Little Snitch)

## [1.0.0] - 2025-12-22

### 🎉 First Stable Release

MageBox v1.0.0 marks the first stable release, ready for team collaboration and production use.

### Security Hardening

This release includes comprehensive security improvements following a full code audit:

- **Shell Injection Prevention** - SSH key deployment now uses base64 encoding instead of heredocs to prevent command injection
- **X-Forwarded-For Protection** - Proxy headers only trusted when connection is from configured `trusted_proxies`, with rightmost non-proxy IP extraction
- **SSH Host Key Verification** - TOFU (Trust On First Use) with fingerprint storage and verification for subsequent connections
- **TOTP Replay Attack Prevention** - Used codes are tracked and rejected within validity window (90 seconds)
- **HMAC Upgraded to SHA-256** - MFA now uses HMAC-SHA256 with 32-byte secrets (was SHA-1/20-byte)
- **URL Validation** - Server join command validates URLs and warns about private/local addresses
- **Security Headers** - Added HSTS, Content-Security-Policy, Referrer-Policy, Permissions-Policy, Cache-Control
- **Input Validation** - SSH public keys validated for format and base64 encoding; usernames sanitized

### Added

- `TrustedProxies` configuration option for secure proxy header handling
- `HostKey` field in Environment for SSH host key verification
- `ValidateCodeForUser()` MFA method with replay protection
- SSH key format validation (`validateSSHPublicKey`)
- Username sanitization (`sanitizeUsername`)

### Changed

- TOTP secret length increased from 20 to 32 bytes for SHA-256
- SSH key deployment uses base64 encoding for security
- Client IP extraction requires explicit proxy configuration

### Fixed

- **IDE Terminal PHP Wrappers** - Bootstrap now adds PATH to `.zshenv` for zsh users, which is sourced by ALL shell invocations including IDE terminals
- **Fedora/RHEL PHP Detection** - Added direct Remi PHP path (`/opt/remi/phpXX/root/usr/bin/php`) as fallback
- **macOS Port Forwarding Persistence** - Use WatchPaths instead of KeepAlive for reliable pf rule persistence across network changes

### Testing

- All unit tests pass (17 packages)
- Static analysis clean (`go vet`, `staticcheck`)
- No known vulnerabilities (`govulncheck`)

## [0.19.1] - 2025-12-19

### Added
- **Server-Side SSH Key Generation** - Team Server now generates Ed25519 SSH key pairs for users:
  - Keys are generated server-side when users join (no need to provide public key)
  - Private key is securely returned to client and stored in `~/.magebox/keys/`
  - Public key is automatically stored on the server for deployment
  - Each user gets a unique key pair per Team Server

- **New CLI Commands**:
  - `magebox ssh <environment>` - SSH into team server environments using stored keys
  - `magebox env sync` - Sync accessible environments from team server

- **Environment Sync API** - New `/api/environments` endpoint for clients to fetch accessible environments

### Changed
- `magebox server join` no longer requires `--key` flag - server generates the key
- Join response now includes `private_key`, `server_host`, and `environments` list

### Testing
- Added comprehensive unit tests for SSH key generation (`crypto_test.go`)
- Added E2E integration tests with Docker containers for actual SSH connections
- Tests cover key uniqueness, access grant/revoke, and environment sync

## [0.19.0] - 2025-12-18

### Added
- **Team Server** - Centralized team access management system for secure SSH key distribution:
  - **Project-Based Access Control** - Users granted access to projects containing multiple environments
  - **User Management** - Invite flow with admin approval, role-based permissions (admin, dev, readonly)
  - **SSH Key Distribution** - Automatic deployment/removal of SSH keys to environments
  - **Multi-Factor Authentication** - TOTP support (Google Authenticator compatible)
  - **Tamper-Evident Audit Logging** - Hash chain verification for compliance
  - **Email Notifications** - SMTP support for invites, security alerts
  - **Security Features** - AES-256-GCM encryption, Argon2id token hashing, IP lockout
  - **ISO 27001 Compliance** - Documentation with control mapping and recommended procedures

### New Commands
- `magebox server init` - Initialize team server with master key and admin token
- `magebox server start` - Start team server with TLS, SMTP, and security options
- `magebox server stop` - Stop running team server
- `magebox server status` - Check team server status
- `magebox server user add/list/show/remove` - User management
- `magebox server user grant/revoke` - Project access management
- `magebox server project add/list/show/remove` - Project management
- `magebox server env add/list/show/remove/sync` - Environment management
- `magebox server audit` - View and export audit logs with filtering
- `magebox server join` - Accept invitation and register SSH key

### Documentation
- New `docs/TEAMSERVER.md` with comprehensive documentation
- ISO 27001 compliance section with control mapping
- Recommended procedures for access review, onboarding/offboarding
- Docker deployment examples
- Integration testing guide

## [0.14.5] - 2025-12-16

### Added
- **Debian 12 and Rocky Linux 9 test containers** - Expanded CI testing coverage
- **Improved Linux distro detection** - Better support for derivative distributions:
  - Proper parsing of `/etc/os-release` (handles quoted values)
  - `ID_LIKE` support for derivatives (EndeavourOS, Pop!_OS, Garuda, etc.)
  - Warning for untested but compatible distros instead of hard failure

### Fixed
- **Docker Compose V1 fallback** - Auto-detects and uses `docker-compose` (standalone) when `docker compose` (V2) is not available
- **EndeavourOS bootstrap** - Fixed detection failing due to quoted values in os-release
- **Ubuntu PHP installation** - Removed non-existent `php-sodium` package (bundled in php-common)
- **OpenSearch version** - Updated from 2.12 to 2.19.4 (2.12 tag doesn't exist on Docker Hub)
- **Self-update permissions** - Automatic sudo when updating binary in /usr/local/bin

## [0.14.4] - 2025-12-16

### Added
- **Self-hosted GitLab/Bitbucket support** - New `--url` flag for `magebox team add` to specify custom instance URLs:
  - `magebox team add myteam --provider=gitlab --org=mygroup --url=https://gitlab.mycompany.com`
  - Supports both GitLab CE/EE and Bitbucket Server/Data Center
  - Clone URLs automatically use the custom host
- **Bitbucket Server API support** - Repository listing now works with self-hosted Bitbucket instances

### Fixed
- **Bitbucket authentication error** - Now shows helpful message when token is required for private repos

## [0.14.3] - 2025-12-16

### Fixed
- **Installer non-interactive mode** - Fixed alias prompt hanging when running via `curl | bash` by auto-detecting non-interactive mode and using default alias

## [0.14.2] - 2025-12-16

### Fixed
- **Installer checksum verification** - Fixed bug where download info message was captured with filename, causing checksum verification to fail

## [0.14.1] - 2025-12-15

### Added
- **Interactive alias selection** - Install script now prompts for short command alias:
  - `mbox` - recommended, descriptive (default)
  - `mb` - shortest (2 chars)
  - Both or skip options available
- **Version display in installer** - Banner now shows version number

### Changed
- Updated ASCII logo in installer to match CLI

## [0.14.0] - 2025-12-15

### Added
- **`magebox test`** - Comprehensive testing and code quality commands:
  - `magebox test setup` - Interactive wizard to install PHPUnit, PHPStan, PHPCS, PHPMD
  - `magebox test unit` - Run PHPUnit unit tests with filter and testsuite options
  - `magebox test integration` - Run Magento integration tests with tmpfs support
  - `magebox test phpstan` - Run PHPStan static analysis (levels 0-9)
  - `magebox test phpcs` - Run PHP_CodeSniffer with Magento2 or PSR12 standards
  - `magebox test phpmd` - Run PHP Mess Detector with configurable rulesets
  - `magebox test all` - Run all tests except integration (for CI/CD)
  - `magebox test status` - Show installed tools and their configuration status
- **Tmpfs MySQL for integration tests** - Run MySQL in RAM for 10-100x faster tests:
  - `--tmpfs` flag to enable RAM-based MySQL container
  - `--tmpfs-size` to configure RAM allocation (default: 1g)
  - `--mysql-version` to specify MySQL version (default: 8.0)
  - `--keep-alive` to keep container running after tests
  - Container naming: `mysql-{version}-test` (e.g., `mysql-8-0-test`)
- **PHPStan Magento extension support** - Automatic integration with `bitexpert/phpstan-magento`:
  - Factory method analysis for ObjectManager
  - Auto-generates `phpstan.neon` with extension includes
- **Testing configuration in `.magebox.yaml`** - Configure paths, levels, standards, and rulesets per project
- **Comprehensive testing documentation** - Added "Testing" section in navigation with detailed command reference

### Changed
- Added "Testing" link to VitePress navigation header

## [0.13.3] - 2025-12-15

### Fixed
- **Test containers**: Added missing Magento-required PHP extensions to all Dockerfiles:
  - Ubuntu (24.04, 22.04, ARM64): Added `bcmath`, `gd`, `intl`, `mysql`, `soap`
  - Fedora 42: Added `bcmath`, `gd`, `intl`, `mysqlnd`, `soap`
  - Arch Linux: Added `php-gd`, `php-intl`, `php-sodium`

## [0.13.2] - 2025-12-15

### Added
- **`magebox dev`** - Switch to development mode optimized for debugging:
  - OPcache: Disabled (immediate code changes)
  - Xdebug: Enabled (step debugging)
  - Blackfire: Disabled (conflicts with Xdebug)
  - Settings persisted in `.magebox.local.yaml`
- **`magebox prod`** - Switch to production mode optimized for performance:
  - OPcache: Enabled (faster execution)
  - Xdebug: Disabled (no overhead)
  - Blackfire: Disabled (enable manually when needed)
- **`magebox queue`** - RabbitMQ queue management for Magento:
  - `magebox queue status` - View queue status with message counts
  - `magebox queue flush` - Purge all queues (use with caution)
  - `magebox queue consumer [name]` - Run Magento queue consumers
  - `magebox queue consumer --all` - Start all consumers via cron
  - Uses RabbitMQ Management API for status/flush operations

### Fixed
- **Config Loader** - PHP INI overrides (`php_ini`) are now properly merged from local config

## [0.13.1] - 2025-12-15

### Added
- **`magebox db snapshot`** - Database snapshot management for quick backup/restore:
  - `magebox db snapshot create [name]` - Create a compressed snapshot
  - `magebox db snapshot restore <name>` - Restore from a snapshot
  - `magebox db snapshot list` - List all snapshots for the project
  - `magebox db snapshot delete <name>` - Delete a snapshot
  - Snapshots stored in `~/.magebox/snapshots/{project}/`
  - Automatic gzip compression for smaller files
- **HTTPS Auth for Teams** - New `--auth=https` option for public repositories:
  - Default auth method changed from `ssh` to `https`
  - Enables cloning public repos without SSH keys (e.g., `magento/magento2`)
  - SSH still available with `--auth=ssh` for private repos

### Security
- **SSH Host Key Verification** - SFTP connections now verify host keys against `~/.ssh/known_hosts` instead of accepting any key

## [0.13.0] - 2025-12-15

### Added
- **`magebox start --all`** - Start all discovered MageBox projects at once
- **`magebox stop --all`** - Stop all running MageBox projects at once
- **`magebox restart`** - Restart project services (stop + start)
- **`magebox uninstall`** - Clean uninstall of MageBox components:
  - Stops all running projects
  - Removes CLI wrappers (php, composer, blackfire)
  - Removes nginx vhost configurations
  - Use `--keep-vhosts` to preserve nginx configs
  - Use `--force` to skip confirmation
- **Test Mode** (`MAGEBOX_TEST_MODE=1`) - Run MageBox in containers without Docker:
  - Skips Docker operations for container-based testing
  - Useful for CI/CD and integration testing
- **Docker Integration Tests** - Comprehensive test suite for multiple distributions:
  - Fedora 42 (Remi PHP)
  - Ubuntu 24.04 (ondrej/php PPA)
  - Ubuntu 22.04 (ondrej/php PPA)
  - Ubuntu 24.04 ARM64 (ondrej/php PPA)
  - Arch Linux (latest PHP)
  - Tests: init, start/stop/restart, domains, SSL, Xdebug, Blackfire, team, uninstall
  - Run with: `./test/containers/run-tests.sh`

## [0.12.14] - 2025-12-15

### Fixed
- **Multi-domain store code** - Fixed `mage_run_code` and `mage_run_type` not being passed to nginx
- **Dynamic `MAGE_RUN_TYPE`** - No longer hardcoded to `store`, now reads from domain config (supports `store` or `website`)

## [0.12.13] - 2025-12-15

### Fixed
- **Xdebug enable/disable on Fedora** - Now supports Remi PHP paths (`/etc/opt/remi/php{ver}/php.d/`)
- **Uses sudo sed** for Xdebug ini modifications (required on Fedora)
- **`magebox blackfire on` now properly disables Xdebug** on Fedora before enabling Blackfire

## [0.12.12] - 2025-12-15

### Added
- **Blackfire CLI wrapper** - `~/.magebox/bin/blackfire` uses project's PHP for `blackfire run` commands
- Bootstrap now installs three shell script wrappers in `~/.magebox/bin/`:
  - `php` - Automatically uses PHP version from `.magebox.yaml`
  - `composer` - Runs Composer with project's PHP version
  - `blackfire` - Uses project's PHP for `blackfire run` commands

### Fixed
- **Blackfire agent configuration** - Uses `sudo sed` to update `/etc/blackfire/agent` credentials
- **Blackfire PHP extension on Fedora** - Uses single `blackfire-php` package (not versioned)
- **Tideways on Fedora 41+** - Downloads RPMs directly (dnf5/cloudsmith compatibility)
- **GPG key import** - Imports Blackfire and Tideways GPG keys before installing packages
- **Non-fatal xdebug disable** - Enabling Blackfire/Tideways no longer fails if xdebug ini is missing

## [0.12.11] - 2025-12-15

### Fixed
- **Tideways repository URL for Fedora** - Changed from `fedora/$releasever/$basearch` to just `$basearch`

### Added
- **Passwordless sudo for Blackfire/Tideways** installation and systemctl commands

## [0.12.10] - 2025-12-14

### Added
- **Blackfire & Tideways in Bootstrap** - Bootstrap now automatically installs profilers for all PHP versions:
  - Fedora: Adds Blackfire/Tideways repos, installs agent and PHP extensions
  - Ubuntu/Debian: Adds repos with GPG keys, installs packages
  - macOS: Uses Homebrew tap and pecl
  - Arch: Uses pecl (agent must be installed from AUR)

## [0.12.9] - 2025-12-14

### Fixed
- **Varnish backend connectivity on Linux** - Use `host.docker.internal` instead of host LAN IP
- **Varnish backend port** - Added dedicated backend port (8080) for Varnish on Linux
- Nginx now listens on port 8080 as backend when Varnish is enabled

## [0.12.8] - 2025-12-14

### Added
- **PHP INI configuration in Bootstrap** - Automatically configures PHP INI settings:
  - Sets `memory_limit = -1` (unlimited) for CLI
  - Sets `max_execution_time = 18000` for long-running CLI scripts
  - Works on all platforms: Fedora (Remi), Ubuntu (Ondrej PPA), macOS (Homebrew), Arch
- **Fedora 43 Support** - Added to officially supported Linux distributions

## [0.12.7] - 2025-12-14

### Changed
- **PHP memory limits** - Increased for Magento compatibility
  - PHP-FPM pool: 768M (was 756M)
  - PHP CLI wrapper: unlimited (`-1`) for commands like `setup:di:compile`

## [0.12.6] - 2024-12-14

### Fixed
- **Bootstrap sudoers creation** - Fixed silent failure when creating `/etc/sudoers.d/magebox`
  - `WriteFile` now uses `RunSudo` instead of `RunSudoSilent` to allow password prompt
  - Previously failed silently if no cached sudo session existed

## [0.12.5] - 2024-12-14

### Changed
- **Simplified Composer wrapper** - Now uses the PHP wrapper instead of duplicating version detection logic
  - Composer wrapper at `~/.magebox/bin/composer` calls the PHP wrapper
  - Reduced code duplication, single source of truth for PHP version detection

### Removed
- **Removed `magebox composer` command** - No longer needed since `~/.magebox/bin/composer` wrapper handles this automatically
  - Just use `composer` directly (with `~/.magebox/bin` in PATH)

## [0.12.4] - 2024-12-14

### Added
- **Automatic PATH configuration during bootstrap** - No longer need to manually add `~/.magebox/bin` to PATH
  - Bootstrap automatically adds PATH entry to shell config (`.zshrc`, `.bashrc`, `.bash_profile`)
  - Supports zsh (macOS default), bash, and fish shells
  - Creates `.zshrc` if it doesn't exist on macOS
  - Shows reload instructions after bootstrap completes

## [0.12.3] - 2024-12-14

### Added
- **`magebox composer` command** - Run Composer with the project's configured PHP version
  - Automatically uses PHP version from `.magebox.yaml`
  - Passes all arguments to Composer
  - Sets `COMPOSER_MEMORY_LIMIT=-1` for large projects

## [0.12.2] - 2024-12-14

### Added
- **Composer install in fetch workflow** - Automatically runs `composer install` after cloning a team project
- **Enhanced Fedora SELinux support** - Bootstrap now configures persistent SELinux fcontext rules using `semanage`:
  - `httpd_var_run_t` context for `~/.magebox/run/` (PHP-FPM sockets)
  - `httpd_config_t` context for `~/.magebox/nginx/` and `~/.magebox/certs/`
- **Sudoers rule for /etc/hosts** - Bootstrap adds passwordless sudo for hosts file modifications

### Fixed
- **PHP-FPM socket location** - Moved from `/tmp/magebox/` to `~/.magebox/run/` to avoid nginx PrivateTmp isolation
- **Fedora PHP-FPM binary path** - Fixed detection to use Remi paths (`/opt/remi/php*/root/usr/sbin/php-fpm`)

### Changed
- PHP-FPM pool generator now uses platform-aware binary path detection
- Nginx vhost generator uses `~/.magebox/run/` for socket paths

## [0.12.1] - 2024-12-14

### Added
- **SELinux support for Fedora** - Bootstrap automatically configures SELinux:
  - Sets `httpd_can_network_connect` boolean for nginx to proxy to Docker containers
  - Configures `httpd_config_t` context on `~/.magebox/nginx/` and `~/.magebox/certs/`
- Added `ConfigureSELinux()` method to installer interface

### Changed
- **Simplified PHP-FPM configuration** - No longer modifies PHP-FPM config files on Linux
  - Uses default repository log paths to avoid permission and SELinux issues
  - Reduces potential for configuration conflicts

### Documentation
- Added comprehensive SELinux troubleshooting guide
- Updated bootstrap documentation with SELinux configuration details
- Updated Linux installers documentation with SELinux tips

## [0.12.0] - 2024-12-14

### Added
- **CLI flags for non-interactive command execution**:
  - `magebox team add` now supports `--provider`, `--org`, `--auth` flags
  - `magebox team add` supports asset storage flags: `--asset-provider`, `--asset-host`, `--asset-port`, `--asset-path`, `--asset-username`
  - `magebox blackfire config` now supports `--server-id`, `--server-token`, `--client-id`, `--client-token` flags
  - `magebox tideways config` now supports `--api-key` flag
- **Homebrew tap** for easy installation: `brew install qoliber/magebox/magebox`
- **Install script** for curl-based installation: `curl -fsSL https://get.magebox.dev | bash`
- **GitHub Actions workflows**:
  - Automatic Homebrew formula updates on new releases
  - Install script deployment to documentation server

### Fixed
- Dynamic team subcommand routing (`magebox team <teamname> <subcommand>` now works correctly)

### Changed
- All interactive commands now fall back to prompts only when required flags are not provided
- Improved CI workflow: removed deprecated macOS-13 runner

## [0.11.0] - 2024-12-14

### Added
- **Team collaboration feature** - Share project configurations across teams:
  - `magebox team add <name>` - Add a new team with repository and asset storage config
  - `magebox team list` - List all configured teams
  - `magebox team remove <name>` - Remove a team configuration
  - `magebox team <name> show` - Show team configuration details
  - `magebox team <name> repos [--filter]` - Browse repositories in team namespace
  - `magebox team <name> project add/list/remove` - Manage team projects
- **Repository provider support**:
  - GitHub, GitLab, and Bitbucket integration
  - SSH and token-based authentication
  - Repository listing with filtering (glob patterns)
- **Asset storage support**:
  - SFTP/FTP for database dumps and media files
  - Progress tracking with download speed and ETA
  - Secure credential storage via environment variables
- **Fetch command** - `magebox fetch <project>`:
  - Clone repository from configured provider
  - Download and import database automatically
  - Download and extract media files
  - Support for `--branch`, `--no-db`, `--no-media`, `--dry-run` flags
- **Sync command** - `magebox sync`:
  - Sync latest database and media for existing projects
  - Auto-detect project from git remote
  - Support for `--db`, `--media`, `--backup`, `--dry-run` flags
- New packages: `internal/team/` with comprehensive test coverage
- Team configuration stored in `~/.magebox/teams.yaml`
- Documentation: `docs/teamwork.md` - Complete guide for team features

### Changed
- Fixed printf format string issues across multiple commands for cleaner linter output

## [0.10.12] - 2024-12-14

### Added
- Xdebug state restoration when disabling Blackfire
  - When enabling Blackfire, Xdebug state is saved if it was enabled
  - When disabling Blackfire, Xdebug is automatically restored to previous state
  - State stored in `~/.magebox/run/xdebug-state-{version}`

### Fixed
- Blackfire installation on macOS now installs PHP-specific formula (`blackfire-php82`, etc.)
- Blackfire extension detection updated for Homebrew's path format
- Blackfire agent detection now handles `blackfire agent:start` process name
- Service detection fallback for Elasticsearch/OpenSearch when compose file is stale

### Changed
- Blackfire enable/disable properly handles Homebrew's ini file format

## [0.10.11] - 2024-12-14

### Fixed
- Docker service detection now falls back to container name when compose file is stale
- Service names like `elasticsearch8170` properly map to container `magebox-elasticsearch-8.17.0`

### Changed
- Updated version display to 0.10.11

## [0.10.10] - 2024-12-13

### Added
- MageBox logo to README
- Complete CLI commands reference documentation
- Logs & Reports guide page

### Fixed
- Xdebug installation detection now uses `php -m` instead of file checks
- Skip xdebug pecl install if already installed (avoids confusing error messages)

## [0.10.2] - 2024-12-13

### Added
- `magebox db create` - Create project database from config
- `magebox db drop` - Drop project database (with confirmation)
- `magebox db reset` - Drop and recreate project database (with confirmation)
- Database commands now use `DefaultDBRootPassword` constant for consistency

### Changed
- Updated `db import`, `db export`, `db shell` to use password constant

## [0.10.1] - 2024-12-13

### Fixed
- Linting issues: unchecked error return, unused functions, gofmt formatting

## [0.10.0] - 2024-12-13

### Added
- **Blackfire profiler integration**:
  - `magebox blackfire on/off` - Enable/disable Blackfire profiling
  - `magebox blackfire status` - Show current Blackfire status
  - `magebox blackfire install` - Install Blackfire agent and PHP extension
  - `magebox blackfire config` - Configure Blackfire credentials
  - Platform support: macOS (Homebrew), Fedora (dnf), Ubuntu/Debian (apt), Arch (AUR)
  - Automatic Xdebug disable when enabling Blackfire to avoid conflicts
- **Tideways profiler integration**:
  - `magebox tideways on/off` - Enable/disable Tideways profiling
  - `magebox tideways status` - Show current Tideways status
  - `magebox tideways install` - Install Tideways daemon and PHP extension
  - `magebox tideways config` - Configure Tideways API key
  - Platform support: macOS (Homebrew), Fedora (dnf), Ubuntu/Debian (apt), Arch (AUR)
  - Automatic Xdebug disable when enabling Tideways to avoid conflicts
- **Global profiling credentials storage**:
  - New `profiling` section in `~/.magebox/config.yaml`
  - Secure credential storage (no credentials in per-project config)
  - Environment variable fallback (`BLACKFIRE_*`, `TIDEWAYS_API_KEY`)
- New packages: `internal/blackfire/`, `internal/tideways/`

### Changed
- `GlobalConfig` now includes `Profiling` configuration section
- Added helper methods for credential management with environment variable precedence

## [0.9.1] - 2024-12-13

### Added
- Additional template files for better maintainability:
  - `internal/dns/templates/systemd-resolved.conf.tmpl` - systemd-resolved configuration
  - `internal/xdebug/templates/xdebug.ini.tmpl` - Xdebug INI configuration
  - `internal/php/templates/not-installed-message.tmpl` - PHP installation instructions (platform-aware)
  - `internal/ssl/templates/not-installed-error.tmpl` - mkcert installation error
- `XdebugConfig` struct for customizable Xdebug settings
- `SystemdResolvedConfig` struct for DNS configuration

### Changed
- Refactored Xdebug configuration to use template
- Refactored PHP not-installed message to use template with platform detection
- Refactored systemd-resolved config generation to use template
- Refactored mkcert error message to use template

## [0.9.0] - 2024-12-13

### Added
- Template-based configuration generation using Go's `text/template` engine
- New template files:
  - `internal/project/templates/env.php.tmpl` - Magento env.php with conditionals
  - `internal/dns/templates/dnsmasq.conf.tmpl` - dnsmasq configuration
  - `internal/dns/templates/hosts-section.tmpl` - /etc/hosts entries
- `EnvPHPData` struct for clean template data separation
- Mailpit always enabled for local development safety (prevents accidental real emails)
- Comprehensive test coverage for env.php generation
- New `internal/bootstrap/` package with platform-specific installers:
  - `installer/darwin.go` - macOS (Homebrew) support
  - `installer/fedora.go` - Fedora/RHEL/CentOS (dnf + Remi) support
  - `installer/ubuntu.go` - Ubuntu/Debian (apt + Ondrej PPA) support
  - `installer/arch.go` - Arch Linux (pacman) support
- OS version validation during bootstrap
- Explicit supported platform versions in bootstrap help

### Changed
- Refactored `internal/project/env.go` from 310 lines of string builders to 195 lines using templates
- Refactored `internal/dns/dnsmasq.go` to use embedded template
- Refactored `internal/dns/hosts.go` `GenerateMageBoxSection()` to use template with `{{range}}`
- Mailpit Docker service now always included in docker-compose.yml
- Mailpit PHP-FPM configuration (sendmail_path) always enabled
- Refactored `cmd/magebox/bootstrap.go` to use new bootstrap package
- Bootstrap now uses `Installer` interface for clean platform abstraction

### Fixed
- `ValidationError` index conversion bug (now works for indices > 9)
- Missing Varnish merge in config loader
- PHP-FPM config path for Fedora/RHEL (Remi repository)
- Silently ignored errors in bootstrap.go PHP-FPM setup
- MySQL/MariaDB memory configuration now properly used

### Improved
- Added constants for magic numbers in `cmd/magebox/new.go`
- Database credentials now use constants (`DefaultDBRootPassword`, etc.)
- Test coverage for `internal/project` improved from 19.4% to 63.7%
- Bootstrap command is now more maintainable with separate installer files per platform

## [0.7.1] - 2024-12-12

### Changed
- Refactored commands into separate files for better maintainability

## [0.7.0] - 2024-12-11

### Changed
- Database operations refactor and fixes

## [0.6.1] - 2024-12-10

### Added
- Auto-execute installation in new `--quick` command

## [0.6.0] - 2024-12-09

### Added
- PHP wrapper for Fedora Remi repository support
- Nginx user configuration for Linux

### Fixed
- Removed sudo from SSL certificate generation

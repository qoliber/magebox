# Changelog

All notable changes to MageBox will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

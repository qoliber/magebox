# Changelog

All notable changes to MageBox will be documented here.

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

- **Smart PHP wrapper**: Added `~/.magebox/bin/php` that automatically detects and uses correct PHP version per project
- PHP wrapper walks directory tree to find `.magebox.yaml` and uses configured PHP version
- Fixed `magebox php` command to properly restart services when switching versions
- Added PHP wrapper installation to bootstrap process

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

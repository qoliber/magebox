# Changelog

All notable changes to MageBox will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

# Changelog

All notable changes to MageBox will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.16.0] - 2026-04-23

### Changed

- **macOS Port Forwarding: pf replaced with TCP proxy daemon** - The previous approach used macOS `pf` (packet filter) kernel rules to redirect ports 80→8080 and 443→8443. These rules were unreliable — Apple's own pf management would override custom anchors after reboot and sleep/wake, silently breaking all `.test` domains. MageBox now uses a persistent TCP proxy daemon (`magebox _portforward`) managed by launchd with `KeepAlive: true`. The daemon listens on ports 80/443 and forwards connections to nginx on 8080/8443. This eliminates all interaction with the macOS pf subsystem. Legacy pf anchor files (`/etc/pf.anchors/com.magebox`), helper scripts, and `/etc/pf.conf` modifications are automatically cleaned up on upgrade. ([#95](https://github.com/qoliber/magebox/issues/95))

### Added

- **Port forwarding self-healing on `magebox start`/`magebox restart`** - On macOS, `magebox start` and `magebox restart` verify that the port forwarding daemon is running and restart it if needed. Domains work immediately without needing `magebox bootstrap` again.
- **Port Forwarding Health Check** - `magebox check` now includes a Port Forwarding section on macOS that reports whether the LaunchDaemon is installed and port forwarding is active.

### Fixed

- **macOS PHP Detection for Unversioned Homebrew Formula** - When PHP was installed via `brew install php` (the unversioned/current formula), MageBox failed to find it because it only looked in `Cellar/php@8.4/`. The unversioned formula installs to `Cellar/php/8.4.x/` instead. Both the Go binary detection and the PHP wrapper script now check both paths.

## [1.15.1] - 2026-04-21

### Fixed

- **PHP-FPM Pool Group Lookup** - `getCurrentGroup()` on Linux previously returned the username verbatim, assuming the primary group name matched the username (USERGROUPS_ENAB convention). On systems where the primary group is renamed or shared (e.g. user `john` with primary group `john-doe`), the generated FPM pool contained `group = john` and `php-fpm` refused to start with `cannot get gid for group 'john'`. Because the pool file is regenerated on every `magebox start`, hand-patching didn't stick. The group is now resolved via `os/user` + GID lookup, and `getCurrentUser()` routes through `user.Current()` so both resolutions share one source of truth. ([#92](https://github.com/qoliber/magebox/pull/92))
- **PHP Install on Ubuntu with PHP 8.5** - PHP 8.5 ships OPcache built into `php8.5-cli`, so the Ondřej PPA no longer publishes a separate `php8.5-opcache` package. The hardcoded install list caused bootstrap to fail with `Unable to locate package php8.5-opcache`. MageBox now filters the extension list through `apt-cache show` before invoking `apt install`, printing a note for any skipped packages. Self-heals future packaging consolidations without requiring version-specific branches. ([#94](https://github.com/qoliber/magebox/pull/94))

## [1.15.0] - 2026-04-20

### Added

- **Auto-Start on `magebox open`** - `magebox open` now starts the project automatically when it is not running. It checks each required service (skipping optional Xdebug and Blackfire) and brings up anything that is down before opening the browser. If everything is already up, the browser opens immediately without extra output. ([#86](https://github.com/qoliber/magebox/pull/86))
- **MageOS 2.2.2 in Version Registry** - `magebox new` can now scaffold MageOS 2.2.2 projects (base: Magento 2.4.8). ([#89](https://github.com/qoliber/magebox/pull/89))

### Fixed

- **`magebox run` PHP Version Shadowing on Linux** - `magebox run` previously prepended `/usr/bin` (the directory of the versioned PHP binary on Linux) to `PATH`, which silently shadowed the `~/.magebox/bin/php` wrapper and made `php` in custom commands resolve to the system default (e.g. `/usr/bin/php` → PHP 8.4). With Magento 2.4.8-p3 and a mismatched system PHP, `bin/magento` degraded to minimal bootstrap mode and commands like `deploy:mode:set` were silently absent. `magebox run` now prepends `~/.magebox/bin` instead, so `php`, `composer`, and `blackfire` in custom commands consistently resolve to the project-aware wrappers and pick the PHP version from `.magebox.yaml`. ([#91](https://github.com/qoliber/magebox/pull/91))
- **Database Import Progress Bar Reaching 100%** - The database import progress bar now reaches 100% instead of stopping at the last tick before EOF (e.g. 98.9%). ([#87](https://github.com/qoliber/magebox/pull/87))

## [1.14.2] - 2026-04-10

### Changed

- **Tideways API Key Scope** - The Tideways API key is now configured per-project instead of globally. Because each Tideways API key is tied to a single Tideways project, storing it in `~/.magebox/config.yaml` either pinned every local Magento project to the same Tideways project or silently got shadowed when reconfigured. The key now lives in the project's `.magebox.local.yaml` under `php_ini.tideways.api_key` and is rendered into the FPM pool as `php_admin_value`, so multiple local projects can each report to their own Tideways project. `magebox tideways config` prompts for the key when run inside a project that doesn't have one set, and `magebox tideways status` reports whether the current project has a key configured. Legacy global `api_key` entries still unmarshal and are detected, surfaced as a migration warning, and removed on save. Access token and environment label remain global. ([#82](https://github.com/qoliber/magebox/pull/82))

### Added

- **`--project-api-key` Flag** - `magebox tideways config` accepts a new `--project-api-key` flag for non-interactive per-project API key configuration. Only applied when MageBox is run from inside a project. ([#82](https://github.com/qoliber/magebox/pull/82))

### Removed

- **`TIDEWAYS_API_KEY` Environment Variable** - The `TIDEWAYS_API_KEY` env var is no longer read by `magebox tideways config`, since the API key is no longer a global setting. Use `--project-api-key` or set `php_ini.tideways.api_key` in `.magebox.local.yaml` instead. ([#82](https://github.com/qoliber/magebox/pull/82))

## [1.14.1] - 2026-04-10

### Fixed

- **Tideways PHP Extension API Key** - `magebox tideways config` now writes the `tideways.api_key` directive into the PHP extension ini file for every installed PHP version, which the Tideways PHP extension requires to transmit traces. Previously MageBox only wrote a daemon config file, and traces never reached Tideways. ([#77](https://github.com/qoliber/magebox/pull/77))
- **Composer Version Fallback** - `magebox new` no longer errors when the daily-updated version registry contains Magento/MageOS versions not present in the hardcoded composer plugin constraint map. Unknown versions now fall back to sensible defaults instead of failing project creation. ([#80](https://github.com/qoliber/magebox/pull/80))
- **Self-Update Binary Sync** - `magebox self-update` now syncs the updated binary to all known locations (`~/.magebox/bin/magebox`, `~/.magebox/bin/mbox`, `/usr/local/bin/magebox`, `/usr/local/bin/mbox`) to prevent version mismatches between `magebox` and `mbox`. ([#80](https://github.com/qoliber/magebox/pull/80))
- **`magebox new ./` Path Handling** - `magebox new ./` now correctly resolves the project name to the current directory name instead of literally `.`. Paths are normalized via `filepath.Clean`. ([#80](https://github.com/qoliber/magebox/pull/80))

### Added

- **Tideways CLI Access Token** - `magebox tideways config` now also prompts for (and stores) a separate **Access Token** used by the `tideways` commandline tool (`tideways run`, `tideways event create`, `tideways tracepoint create`). When provided, MageBox runs `tideways import <token>` automatically. A new `--access-token` flag and `TIDEWAYS_CLI_TOKEN` environment variable were added. The API key and access token are two different credentials — both are now managed. ([#77](https://github.com/qoliber/magebox/pull/77))
- **Tideways Environment Label** - `magebox tideways config` now prompts for an environment label (defaulting to `local_<username>`) and writes a systemd drop-in for `tideways-daemon` on Linux so traces from developer machines are tagged with the local environment instead of the server-side default `production`. A `--environment` flag and `TIDEWAYS_ENVIRONMENT` env var are also supported. macOS prints a clear manual-config hint. ([#77](https://github.com/qoliber/magebox/pull/77))

### Changed

- **Troubleshooting Docs** - Updated troubleshooting documentation with entries for version mismatch between `magebox`/`mbox` binaries and unsupported Magento version errors. ([#80](https://github.com/qoliber/magebox/pull/80))
- **Tideways Docs** - Updated Tideways service and global-config guide pages to document the API key vs. access token vs. environment credentials and the per-project Installation URL as the API key source. ([#77](https://github.com/qoliber/magebox/pull/77))

## [1.14.0] - 2026-04-08

### Fixed

- **Xdebug Linux Support** - Fixed Xdebug detection on Linux by adding a Linux case to `findXdebugSo()` using `php-config` to locate the extension directory. Also writes full Xdebug configuration (mode, start_with_request, client_host, client_port, idekey) when enabling, and updated install hint to use `magebox ext install xdebug`. ([#75](https://github.com/qoliber/magebox/pull/75))
- **Watch Command Cache Clean** - The `magebox watch` command now checks for `cache-clean.js` in the project's `vendor/bin` directory before falling back to a global install. ([#73](https://github.com/qoliber/magebox/pull/73))

## [1.13.1] - 2026-04-07

### Changed

- **Watch Command Theme Detection** - The `magebox watch` command now detects any theme with an npm `watch` script instead of only Hyvä themes. Walks the full theme tree under `app/design/frontend/` and prompts the user to select when multiple themes are found. ([#70](https://github.com/qoliber/magebox/pull/70))
- **MageOS Version Registry** - Added MageOS 2.2.1 to the supported version registry. ([#69](https://github.com/qoliber/magebox/pull/69))

### Fixed

- **Expose Revert NULL Handling** - Fixed expose revert inserting literal string `'NULL'` instead of SQL `NULL` into `base_static_url` and `base_media_url` config entries, corrupting the database values after reverting an expose session. ([#71](https://github.com/qoliber/magebox/pull/71))
- **Tmux Theme Watcher** - Fixed tmux theme watcher pane closing immediately when npm watch fails. Errors now stay visible with a user-friendly message. ([#72](https://github.com/qoliber/magebox/pull/72))
- **FAQ: Composer Patches on macOS** - Added FAQ entry for composer patches hanging on macOS due to BSD patch interactive behavior. ([#63](https://github.com/qoliber/magebox/pull/63))

## [1.13.0] - 2026-04-06

### Added

- **Watch Command** - New `magebox watch` command that runs `mage-os/magento-cache-clean` to watch for file changes and selectively clear affected cache types. Automatically detects Hyvä themes and launches a split tmux session with both the cache watcher and Tailwind CSS watcher. ([#66](https://github.com/qoliber/magebox/pull/66))
- **Magerun2 Fallback** - Unknown commands are now automatically forwarded to magerun2 if available, so you can run commands like `magebox cache:flush` or `magebox setup:upgrade` directly without prefixing with `magerun2`. ([#67](https://github.com/qoliber/magebox/pull/67))

## [1.12.1] - 2026-04-04

### Fixed

- **Magento Version Matrix** - Added missing Magento patch versions (2.4.8-p4, 2.4.7-p9, 2.4.6-p14) to the composer.json generator, fixing "unsupported Magento version" errors during project creation.
- **Homebrew Install Instructions** - Updated brew install instructions to use tap + install pattern.
- **Release Workflow** - Added versioned Homebrew formulae to release workflow and fixed YAML syntax error in auto-update-versions workflow.

## [1.12.0] - 2026-04-04

### Added

- **Include Config Support** - Split `.magebox.yaml` across multiple files using `include_config` to organize large configurations into manageable pieces. ([#60](https://github.com/qoliber/magebox/pull/60))
- **Automated Magento/MageOS Version Updates** - New CI workflow that automatically checks for new Magento and MageOS releases and updates the supported versions list. ([#64](https://github.com/qoliber/magebox/pull/64))

### Fixed

- **RabbitMQ Credentials** - Fixed credential mismatch by using `guest/guest` as the default RabbitMQ credentials. ([#61](https://github.com/qoliber/magebox/pull/61))
- **Elasticsearch/OpenSearch Version Resolution** - Short version numbers (e.g., `8.17`) are now resolved to full patch versions (e.g., `8.17.0`) for Docker image compatibility. ([#62](https://github.com/qoliber/magebox/pull/62))

## [1.11.1] - 2026-04-02

### Added

- **Update Available Notification** - MageBox now checks for new versions in the background and displays a notification after command output when an update is available. Results are cached for 24 hours in `~/.magebox/version-check.json`. Skipped for `self-update` and dev builds.

### Fixed

- **GitHub Release Notes** - Release workflow now includes the changelog entry in GitHub release notes above the installation instructions.

## [1.11.0] - 2026-04-02

### Added

- **Sandbox Command** - New `magebox sandbox` command to run AI coding agents (Claude, Codex) inside a bubblewrap (bwrap) sandbox with restricted filesystem access. Supports configurable tool profiles, extra bind mounts, and a `--dry-run` mode. Linux only. ([#54](https://github.com/qoliber/magebox/pull/54))
- **IPv6 Support (macOS)** - Added IPv6 port forwarding rules and nginx listen directives on macOS, fixing "connection refused" errors when clients resolve `.test` domains to `::1` first. ([#52](https://github.com/qoliber/magebox/pull/52))

## [1.10.1] - 2026-03-29

### Fixed

- **GitHub Actions Warnings** - Fixed deprecation warnings in CI workflows. ([#51](https://github.com/qoliber/magebox/pull/51))
- **Homebrew Formula Audit** - Fixed "Do not define methods in blocks" audit errors by using `on_arm`/`on_intel` DSL and a class-level `def install`.

## [1.10.0] - 2026-03-29

### Added

- **Purge Command** - New `magebox purge` command to remove all generated code, preprocessed views, static content, and caches in parallel. Flushes Redis/Valkey and Varnish when configured. ([#47](https://github.com/qoliber/magebox/pull/47))
- **phpMyAdmin Service** - Add phpMyAdmin as a built-in service with `magebox phpmyadmin enable/disable/status/open` commands and per-project config support. ([#43](https://github.com/qoliber/magebox/pull/43))
- **Elasticvue Open Command** - New `magebox elasticvue open` command and fixed hardcoded port references. ([#44](https://github.com/qoliber/magebox/pull/44))
- **Mailpit Commands** - New `magebox mailpit open` and `magebox mailpit status` subcommands. ([#45](https://github.com/qoliber/magebox/pull/45))
- **Interactive TUI for Run Command** - `magebox run` without arguments now shows an interactive selector. Added `--list`/`-l` flag and implicit `magebox <name>` rewrite to `magebox run <name>`. ([#50](https://github.com/qoliber/magebox/pull/50))
- **Expose Request Logs** - `magebox expose` now displays incoming HTTP requests with color-coded status codes. ([#40](https://github.com/qoliber/magebox/pull/40))

### Fixed

- **PHP Wrapper Recursion** - Unset `MAGEBOX_PHP_WRAPPER` before running the PHP binary, preventing infinite recursion when tools like GrumPHP spawn child processes. ([#39](https://github.com/qoliber/magebox/pull/39))

## [1.9.0] - 2026-03-17

### Changed

- **Version-Specific Search Ports** - Refined port allocation for OpenSearch and Elasticsearch: moved Elasticsearch base from 9300 to 9500 to provide ample room for both search engines. Updated port documentation accordingly. ([#36](https://github.com/qoliber/magebox/pull/36))

### Removed

- **Magerun2-Overlapping Commands** - Removed `magebox admin`, `magebox mode`, `magebox queue`, and `magebox report` commands that duplicated functionality already provided by magerun2 (`admin:user:*`, `deploy:mode:set`, `queue:consumers:*`, `dev:report:count`). This keeps MageBox focused on environment management. ([#38](https://github.com/qoliber/magebox/pull/38))

## [1.8.0] - 2026-03-17

### Added

- **Valkey Support** - Valkey is now available as a Redis alternative, configurable per project with `valkey: true` in `.magebox.yaml`. Valkey is a Redis-compatible fork by the Linux Foundation that uses the same port (6379) and protocol. The `magebox redis` commands automatically detect which cache service is configured. ([#36](https://github.com/qoliber/magebox/pull/36))
- **Open Command** - New `magebox open` command to quickly open the project's first domain in the default browser. ([#35](https://github.com/qoliber/magebox/pull/35))
- **Version-Specific Search Ports** - OpenSearch and Elasticsearch now use unique host ports per version (e.g., OS 2.19 → 9259, ES 7.17 → 9457), preventing port conflicts when running multiple projects with different search engine versions. When only one search service is configured, the standard port 9200 is additionally mapped for backward compatibility. ([#35](https://github.com/qoliber/magebox/pull/35))
- **Magento 2.4.8-p4, 2.4.7-p9, 2.4.6-p14** - Added latest Magento Open Source patch releases to the version matrix.

## [1.7.3] - 2026-03-17

### Added

- **Unattended Bootstrap** - New `--unattended` flag for `magebox bootstrap` that auto-accepts all interactive prompts, enabling use in Ubuntu autoinstall and other automated provisioning contexts.
- **Bootstrap TLD Flag** - New `--tld` flag for `magebox bootstrap` to set the top-level domain during setup (e.g., `--tld local`), saving it to the global config before DNS and vhost configuration runs.

## [1.7.2] - 2026-03-17

### Added

- **Autostart Service** - New `magebox service` command with `install`, `uninstall`, and `status` subcommands. Installs a system service (systemd on Linux, launchd on macOS) that automatically starts all global services and projects on boot/login — no more manual `magebox start` after reboot.
- **Bootstrap Autostart Prompt** - `magebox bootstrap` now offers to install the autostart service at the end of setup.

## [1.7.1] - 2026-03-17

### Added

- **Varnish Log Streaming** - New `magebox varnish logs` and `magebox logs varnish` commands for streaming varnishlog output from the container.
- **Varnish Request Histogram** - New `magebox varnish hist` (alias: `history`) command showing a live histogram of request processing times via varnishhist.
- **Varnish Admin CLI** - New `magebox varnish admin` command for interactive or single-command varnishadm access (e.g., `magebox varnish admin backend.list`).
- **VCL Import Backend Host Check** - `magebox varnish vcl-import` now detects when backend `.host` is set to `localhost` or `127.0.0.1` and offers to rewrite it to `host.docker.internal` so Varnish in Docker can reach Nginx on the host.
- **Varnish Enable Health Check** - `magebox varnish enable` now verifies the container is healthy after starting and shows container logs on failure. Also detects when Varnish is enabled in config but not running.

### Fixed

- **Elasticvue Port Conflict** - Moved Elasticvue from port 8080 to 8090 to avoid conflict with the Varnish backend port (Nginx listens on 8080 for Varnish).
- **Global Start Compose Regeneration** - `magebox global start` now regenerates the docker-compose file before starting, so config changes take effect without requiring a full bootstrap.

## [1.7.0] - 2026-03-17

### Added

- **PHP Extension Management** - New `magebox ext` command with `install`, `remove`, `list`, and `search` subcommands for managing PHP extensions across all installed PHP versions. Automatically resolves platform-specific package names (apt/dnf/pacman/pecl) from a single canonical extension name. ([#31](https://github.com/qoliber/magebox/pull/31))
- **PIE Integration** - Install custom PHP extensions from Packagist using the `vendor/package` format (e.g., `magebox ext install noisebynorthwest/php-spx`). Uses [PIE](https://github.com/php/pie), the official PECL replacement from the PHP Foundation. PIE and PHP dev tools are installed automatically on first use.
- **PHP Version Selection** - Extension install and remove commands prompt to select which installed PHP version(s) to target, with "All installed" as the default.
- **Custom Docker Compose Support** - New `compose_file` config option to reference a project-level `docker-compose.yml`. On `magebox start`/`stop`, MageBox prompts to manage these containers and connects them to the MageBox Docker network for service discovery. ([#27](https://github.com/qoliber/magebox/pull/27))

## [1.6.0] - 2026-03-15

### Added

- **Service-Specific Log Tailing** - New `magebox logs php/nginx/mysql/redis` subcommands for viewing per-service logs. PHP-FPM and Nginx tail local log files with multitail support, while MySQL and Redis stream Docker container logs. Supports `-f` (follow) and `-n` (lines) flags. ([#26](https://github.com/qoliber/magebox/pull/26))
- **Expose / Share via Cloudflare Tunnels** - New `magebox expose` command to share local projects via public Cloudflare Tunnel URLs. Automatically backs up and updates Magento base URLs, nginx vhosts, and env.php, with full revert on Ctrl+C. ([#25](https://github.com/qoliber/magebox/pull/25))
- **Auto Document Root Discovery** - MageBox now automatically discovers the document root directory, removing the need to manually configure it in most cases. ([#24](https://github.com/qoliber/magebox/pull/24))
- **Elasticvue Integration** - New `magebox elasticvue enable/disable/status` commands for managing the Elasticvue web UI for OpenSearch/Elasticsearch. ([#23](https://github.com/qoliber/magebox/pull/23))
- **Database Process Monitor** - New `magebox db top` command for real-time MySQL/MariaDB process monitoring using innotop or mysqladmin processlist. ([#23](https://github.com/qoliber/magebox/pull/23))
- **VCL Import/Reset** - New `magebox varnish vcl-import` and `magebox varnish vcl-reset` commands for custom Varnish VCL management. ([#23](https://github.com/qoliber/magebox/pull/23))

## [1.5.0] - 2026-03-15

### Added

- **Hyvä Theme Support** - Install and activate the Hyvä theme with `magebox new --hyva`. Handles Composer authentication, package installation, and automatic theme activation after `setup:upgrade`. ([#21](https://github.com/qoliber/magebox/pull/21))
- **Automatic Composer Installation** - `magebox new` now detects if Composer is missing and installs it automatically. ([#13](https://github.com/qoliber/magebox/pull/13))
- **New Magento & MageOS Versions** - Added support for latest Magento and MageOS releases. ([#14](https://github.com/qoliber/magebox/pull/14))

### Changed

- **CGO-Free Builds** - Replaced `mattn/go-sqlite3` with `modernc.org/sqlite` for CGO-free builds, simplifying cross-compilation. ([#15](https://github.com/qoliber/magebox/pull/15))

### Fixed

- **IPv6 Nginx Listen Directives** - Added IPv6 (`[::]:port`) listen directives to nginx vhost templates on Linux. ([#20](https://github.com/qoliber/magebox/pull/20))
- **Quick Install DI Error** - Fixed dependency injection error during quick install by removing pre-generated `env.php` before `setup:install`. ([#18](https://github.com/qoliber/magebox/pull/18))
- **Tideways INI Modifications** - Use `sudo sed` for tideways PHP INI modifications to match the approach used for xdebug. ([#16](https://github.com/qoliber/magebox/pull/16))
- **Global Config Defaults** - Fixed `magebox init` to respect `default_php` and `default_services` from global configuration. ([#12](https://github.com/qoliber/magebox/pull/12))
- **Custom MySQL/MariaDB Config** - Fixed mounting of custom MySQL/MariaDB configuration files in docker-compose. ([#12](https://github.com/qoliber/magebox/pull/12))
- **macOS sed Compatibility** - Use platform-aware sed flags for xdebug enable/disable on macOS. ([#12](https://github.com/qoliber/magebox/pull/12))

## [1.4.0] - 2026-03-14

Special thanks to [Peter Jaap Blaakmeer](https://github.com/peterjaap) for his contributions to this release.

### Added

- **Laravel Project Type Support** - MageBox now supports Laravel projects with a dedicated vhost template, `--type laravel` project type, and proper nginx configuration for Laravel routing. ([#8](https://github.com/qoliber/magebox/pull/8) - [@peterjaap](https://github.com/peterjaap))
- **Uninstall `--purge-packages` Flag** - New flag to remove installed system packages (PHP, nginx, etc.) during uninstall, not just MageBox config and data.
- **Bootstrap MySQL Default Selection** - The bootstrap command now asks which MySQL version should be the default and exposes it on port 3306 alongside the version-specific port.

### Changed

- **Project Name Slash Handling** - Slashes in project names are now replaced with dots for cleaner domain and directory names.

### Fixed

- **Ubuntu 24.04 Bootstrap** - Resolved multiple bootstrap failures on Ubuntu 24.04, including package installation, service configuration, and PHP-FPM setup. ([#7](https://github.com/qoliber/magebox/pull/7) - [@peterjaap](https://github.com/peterjaap))
- **Nginx http2 Compatibility** - Use compatible `http2` syntax for nginx 1.24 (Ubuntu 24.04) which doesn't support the newer `http2 on` directive.
- **DNS Setup Permissions** - Use sudo to read `/etc/dnsmasq.conf` during DNS setup to avoid permission errors.
- **Test Host Config Independence** - Tests no longer depend on host global configuration, making CI more reliable.

## [1.3.3] - 2026-03-04

### Added

- **Single VERSION File** - Version is now managed in a single `VERSION` file at the repo root, read by Go (via Makefile ldflags), VitePress (via `fs.readFileSync`), and CI workflows. Eliminates version drift across binaries, docs, and nav.
- **MageOS 2.1.0 Support** - Added MageOS 2.1.0 as the default distribution for new projects (PHP 8.2/8.3/8.4).
- **Weekly Upstream Release Check** - GitHub Action runs every Monday to detect new Magento/MageOS releases and opens an issue when new versions are found.

### Changed

- **Simplified Quick Install** - Removed Redis and RabbitMQ from `--quick` mode to keep it minimal and avoid connection errors during setup. Users can add these services later via `.magebox.yaml`.
- **OPcache/JIT Disabled by Default** - OPcache and JIT are now disabled by default in PHP-FPM pool settings. Prevents segfaults during `setup:upgrade` on PHP 8.3 and eliminates stale cache issues during development. Users can re-enable via `php_ini` in `.magebox.yaml` for production-like testing.
- **Increased PHP-FPM Pool Sizes** - Doubled `pm.max_children` from 25 to 50 and proportionally increased start/spare servers (8/4/12) for both shared and isolated pools. Reduces 502 errors under concurrent Magento requests.
- **PHP Wrapper Disables OPcache CLI** - The PHP wrapper now passes `-d opcache.enable_cli=0` to all CLI commands, preventing JIT-related segfaults in `setup:install`, `setup:upgrade`, and other bin/magento commands.

### Fixed

- **Quick Install Database Name** - Fixed `setup:install` using raw project name (e.g. `product-feeds`) as `--db-name` while `ensureDatabase` created the sanitized name (`product_feeds`). Both interactive and quick install now use the sanitized name.
- **Quick Install PHP Wrapper** - Fixed `setup:install`, `sampledata:deploy`, `setup:upgrade`, `indexer:reindex`, and `cache:flush` being executed via the Composer wrapper instead of the PHP wrapper, causing "Command bin/magento is not defined" errors.
- **Embedded MageOS Default** - Fixed embedded `versions.yaml` fallback still using MageOS 2.0.0 as default instead of 2.1.0.
- **Composer Version Maps** - Synced hardcoded composer version maps with `versions.yaml`: added MageOS 2.1.0/1.3.x/1.2.0, Magento 2.4.8/2.4.8-p1/2.4.8-p2/2.4.8-p3/2.4.7-p4/2.4.6-p8. Fixed "unsupported MageOS version: 2.1.0" error.
- **CI Lint Errors** - Fixed all `golangci-lint` errors (unchecked `json.Decode` return values, `goimports` formatting).

## [1.3.0] - 2026-02-21

### Added

- **Project-Level Custom Nginx Snippets** - Add custom nginx config per project:
  - Create `{project}/.magebox/nginx/*.conf` files to add custom nginx directives
  - Snippets are automatically included inside the server block of the generated vhost
  - Useful for custom headers, rewrites, proxy rules, or additional location blocks
  - No need to modify the global nginx configuration

- **Project-Level Vhost Template Override** - Full control over the nginx vhost template per project:
  - Place a custom template at `{project}/.magebox/templates/nginx/vhost.conf.tmpl`
  - Overrides the default vhost structure for that project only
  - Override precedence: project → user global (`~/.magebox/yaml-local/`) → default embedded
  - Useful when custom snippets aren't enough and you need to change the entire vhost structure

### Fixed

- **Fedora Nginx Permissions** - Fixed `/var/lib/nginx` ownership not persisting across reboots:
  - Added `/var/lib/nginx` base directory to tmpfiles.d config
  - Added `/var/log/nginx` directory entry for proper log permissions
  - Renamed tmpfiles.d config to `nginx.conf` to properly override the system default
  - Prevents systemd from resetting ownership to `nginx:root` on boot

## [1.2.7] - 2026-02-04

### Fixed

- **Isolated PHP-FPM Template** - Fixed "Array are not allowed in the global section" error:
  - `php_admin_value[]` directives were incorrectly placed in the `[global]` section of isolated PHP-FPM configs
  - PHP-FPM only allows these directives in pool sections
  - Moved system settings (`opcache.enable_cli`, etc.) into the pool section where they work correctly
  - `php_admin_value` in pool sections can still override PHP_INI_SYSTEM settings, preserving full isolation

- **Isolated PHP-FPM Config Regeneration** - Fixed configs not updating on restart:
  - `StartAllIsolated()` and `Restart()` now regenerate the config from the embedded template before starting
  - Previously, config was only generated during initial `Enable()`, so template fixes required re-isolating projects
  - Ensures config always reflects the latest template and settings

- **Fedora SELinux Log Permissions** - Fixed nginx 502 errors caused by SELinux:
  - Added `httpd_log_t` context for `~/.magebox/logs` directory
  - Nginx was denied writing log files due to `user_home_t` context on the logs directory
  - Added both persistent `semanage fcontext` rule and immediate `chcon` fallback

- **Fedora PHP-FPM Management** - Removed systemd enable/start for PHP-FPM:
  - MageBox manages PHP-FPM directly to avoid SELinux `httpd_t` restrictions on user home directories
  - Prevents conflicts between systemd-managed and MageBox-managed PHP-FPM processes

- **Fedora SELinux PHP 8.5 Support** - Added PHP 8.5 to Remi run directory SELinux rules

- **Arch Linux Bootstrap** - Moved common directory setup to base installer

## [1.2.6] - 2026-01-24

### Fixed

- **Nginx Config Setup Resilience** - Made MageBox include insertion more robust:
  - Tries multiple include markers (conf.d, sites-enabled)
  - Falls back to finding http block closing brace
  - Provides helpful error message with manual instructions if all methods fail
  - Works with custom nginx configurations that lack standard markers

## [1.2.5] - 2026-01-24

### Fixed

- **Ubuntu Nginx Permissions** - Fixed `/var/lib/nginx` permission denied errors:
  - Added ownership fix for nginx lib directory (same as Fedora)
  - Added tmpfiles.d config for persistent permissions across reboots
  - Nginx now restarts after user change to apply configuration

- **Sudoers Temp File Whitelist** - Fixed nginx config update failing without password:
  - Changed temp file pattern from `nginx-conf-*` to `magebox-nginx-*`
  - Matches existing sudoers whitelist `/tmp/magebox-*`

- **Ubuntu PHP-FPM Bootstrap** - Fixed "no pool defined" error:
  - Creates placeholder pool before restarting PHP-FPM
  - PHP-FPM restart is now non-fatal during bootstrap
  - Real pools are created when `magebox start` runs

## [1.2.4] - 2026-01-24

### Fixed

- **MariaDB Port Mapping** - Fixed env.php port mapping to match docker-compose:
  - MariaDB 10.6 now correctly uses port 33106 (was incorrectly 33110)
  - MariaDB 11.4 now correctly uses port 33114 (was incorrectly 33111)
  - All port mappings now use explicit maps instead of string manipulation

- **Docker Compose Service Drift** - Fixed "last project wins" bug:
  - `magebox start` now aggregates services from ALL registered projects
  - Prevents services required by other projects from being dropped
  - Each project start regenerates compose with full service set

- **Admin Command MariaDB Support** - Fixed fallback database config:
  - Now uses sanitized database name (hyphens → underscores)
  - Added MariaDB port mapping support in admin password/list commands

- **Isolated PHP-FPM Stop** - Fixed isolated masters not stopping:
  - `magebox stop` now properly stops isolated PHP-FPM master processes
  - Correctly handles both isolated and shared pool configurations

- **Dry-Run Accuracy** - Fixed misleading `--dry-run` output:
  - No longer claims Docker containers would be stopped
  - Accurately reflects that shared services remain running

- **Exit Code on Errors** - Fixed start/stop returning success on failure:
  - Commands now return proper error codes for automation reliability

## [1.2.3] - 2026-01-21

### Fixed

- **PHP Wrapper Recursion Loop** - Fixed infinite loop when `/usr/local/bin/php` symlinks to MageBox wrapper:
  - Added recursion detection using `MAGEBOX_PHP_WRAPPER` environment variable
  - Removed problematic symlink recommendation from documentation
  - Wrapper now errors clearly instead of hanging forever

## [1.2.2] - 2026-01-20

### Fixed

- **Bootstrap Nginx Warning Fix** - Fixed false "Nginx is not installed" warning:
  - Error message was shown even after successful nginx installation during bootstrap
  - Removed premature error that was added before offering to install nginx

## [1.2.1] - 2026-01-20

### Fixed

- **Ubuntu PHP-FPM Installation Failure** - Fixed installation failure on Ubuntu with Sury PPA:
  - Handles www.conf socket conflict that caused exit code 78 (configuration error)
  - Auto-recovers by disabling conflicting www.conf and completing package configuration
  - Applies to all PHP versions installed via ondrej/php PPA

## [1.2.0] - 2026-01-13

### Added

- **IPv6 Support for DNS Resolution** - dnsmasq now responds to both IPv4 and IPv6 (AAAA) queries:
  - Fixes 30+ second DNS resolution delays on `.test` domains
  - Added `address=/test/::1` alongside existing `address=/test/127.0.0.1`
  - Applied to runtime dnsmasq configuration and all bootstrap installers (Fedora, Ubuntu, Arch)
  - Prevents IPv6 AAAA query timeouts that caused slow domain resolution

- **New `magebox php system` Commands** - Manage system-wide PHP INI settings (PHP_INI_SYSTEM):
  - `mbox php system list` - List current system PHP settings
  - `mbox php system set <key> <value>` - Set a system-wide PHP INI value
  - `mbox php system get <key>` - Get current value of a system PHP setting
  - `mbox php system unset <key>` - Remove a system PHP INI override
  - These settings apply to the PHP-FPM master process (opcache, JIT, preload, etc.)

### Changed

- **Improved PHP-FPM Pool Defaults** - Updated default pool settings for better Magento performance:
  - `pm.max_children`: 10 → 25 (handle more concurrent requests)
  - `pm.start_servers`: 2 → 4 (faster startup response)
  - `pm.min_spare_servers`: 1 → 2 (better availability)
  - `pm.max_spare_servers`: 3 → 6 (handle traffic spikes)
  - `pm.max_requests`: 500 → 1000 (reduce worker recycling overhead)

### Fixed

- **DNS Resolution Speed** - Resolves the issue where `.test` domains took 6+ seconds to resolve after periods of inactivity due to IPv6 AAAA query timeouts

### Technical Notes

- **PHP_INI_SYSTEM vs Pool Settings**: Pool-level `php_admin_value` directives only work for PHP_INI_PERDIR settings. True system-level settings (opcache.memory_consumption, opcache.jit, opcache.preload) require PHP-FPM master process configuration, which is now managed via `magebox php system` commands.

## [1.1.3] - 2026-01-10

### Added

- **Separated PHP_INI_SYSTEM Settings** - System-level PHP settings now managed separately from pool settings:
  - Pool configs use `php_admin_value` for per-request settings
  - System settings (opcache, preload) require master process configuration
  - Clearer separation between project-specific and system-wide PHP configuration

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
- **Homebrew tap** for easy installation: `brew tap qoliber/magebox && brew install magebox`
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

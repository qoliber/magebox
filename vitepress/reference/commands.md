# CLI Commands

Complete reference for all MageBox commands.

## Global Flags

These flags work with any command:

### `-v`, `-vv`, `-vvv`

Enable verbose logging for debugging and troubleshooting.

```bash
magebox -v start      # Basic - shows commands being run
magebox -vv start     # Detailed - shows command output
magebox -vvv start    # Debug - full debug info
```

**Verbosity levels:**
- `-v` (basic) - Shows commands being executed
- `-vv` (detailed) - Shows command output and results
- `-vvv` (debug) - Full debug information including platform detection

Output is color-coded: `[verbose]` (cyan), `[debug]` (yellow), `[trace]` (magenta)

## Project Commands

### `magebox init [name]`

Initialize a new MageBox project.

```bash
magebox init mystore
```

Creates a `.magebox.yaml` configuration file in the current directory.

**Arguments:**
- `name` - Project name (optional, defaults to directory name)

---

### `magebox start`

Start the project environment.

```bash
magebox start
magebox start --all    # Start all discovered projects
```

This command:
1. Generates PHP-FPM pool configuration
2. Generates Nginx vhost configuration
3. Starts required Docker services
4. Updates DNS (if hosts mode)
5. Generates SSL certificates (if needed)
6. Reloads Nginx

**Options:**
- `--all` - Start all discovered MageBox projects at once

---

### `magebox stop`

Stop the project environment.

```bash
magebox stop
magebox stop --all      # Stop all running projects
magebox stop --dry-run  # Preview what would happen
```

Stops PHP-FPM pool and removes Nginx configuration.

**Options:**
- `--all` - Stop all running MageBox projects at once
- `--dry-run` - Preview what would happen without making changes

---

### `magebox restart`

Restart the project environment.

```bash
magebox restart
magebox restart --all   # Restart all projects
```

Equivalent to `stop` followed by `start`.

**Options:**
- `--all` - Restart all MageBox projects at once

---

### `magebox status`

Show project status.

```bash
magebox status
```

Displays:
- PHP version and pool status
- Nginx vhost status
- Service connectivity
- Domain information

---

### `magebox new [directory]`

Create a new Magento/MageOS installation.

```bash
magebox new mystore
```

Interactive wizard that guides through:
- Distribution selection (Magento/MageOS)
- Version selection
- PHP version
- Composer authentication
- Database configuration
- Search engine selection
- Service configuration
- Sample data installation

## PHP Commands

### `magebox php [version]`

Show or switch PHP version.

```bash
# Show current version
magebox php

# Switch to PHP 8.3
magebox php 8.3
```

Switching creates/updates `.magebox.local.yaml` with the new version.

**Available versions:** 8.1, 8.2, 8.3, 8.4

---

### `magebox shell`

Open a shell with the correct PHP in PATH.

```bash
magebox shell
```

Opens a new shell where:
- PHP points to project's configured version
- Environment variables are set
- Working directory is project root

---

### `magebox php ini set <key> <value>`

Set a PHP INI value for the current project.

```bash
magebox php ini set memory_limit 512M
magebox php ini set max_execution_time 300
magebox php ini set display_errors On
```

Settings are stored in `.magebox.yaml` and applied to the project's PHP-FPM pool.

---

### `magebox php ini get <key>`

Get the current value of a PHP INI setting.

```bash
magebox php ini get memory_limit
```

Shows the effective value from pool configuration.

---

### `magebox php ini list`

List all PHP INI settings for the project.

```bash
magebox php ini list
```

Shows both default values and custom overrides.

---

### `magebox php ini unset <key>`

Remove a custom PHP INI override.

```bash
magebox php ini unset memory_limit
```

Reverts the setting to its default value.

---

### `magebox php opcache status`

Show OPcache status for the project.

```bash
magebox php opcache status
```

Displays current OPcache configuration (enabled/disabled).

---

### `magebox php opcache enable`

Enable OPcache for the project.

```bash
magebox php opcache enable
```

Sets `opcache.enable=1` in the project's PHP configuration.

---

### `magebox php opcache disable`

Disable OPcache for the project.

```bash
magebox php opcache disable
```

Sets `opcache.enable=0` - useful during development for immediate code changes.

---

### `magebox php opcache clear`

Clear OPcache by reloading PHP-FPM.

```bash
magebox php opcache clear
```

Forces PHP-FPM to reload, clearing the OPcache.

---

### `magebox php system`

View PHP system-level settings (PHP_INI_SYSTEM) and their activation status.

```bash
magebox php system
```

Shows which project owns the system settings, their values, and whether they're active.

::: info PHP_INI_SYSTEM Settings
Some PHP settings like `opcache.preload`, `opcache.jit`, and `opcache.memory_consumption` can only be set in php.ini (not per-pool). These settings apply to ALL projects using the same PHP version.
:::

---

### `magebox php system enable`

Enable PHP system settings by creating a symlink.

```bash
magebox php system enable
```

Creates a symlink from your MageBox system INI to the PHP scan directory. Requires sudo.

---

### `magebox php system disable`

Disable PHP system settings by removing the symlink.

```bash
magebox php system disable
```

---

### `magebox php system list`

List all PHP_INI_SYSTEM setting names.

```bash
magebox php system list
```

Shows all PHP settings that can only be set globally (opcache.preload, opcache.jit, etc.).

---

## PHP Isolation Commands

Isolated PHP-FPM masters allow you to configure PHP_INI_SYSTEM settings (like opcache.memory_consumption, opcache.jit, opcache.preload) independently for each project.

### `magebox php isolate`

Enable a dedicated PHP-FPM master process for the current project.

```bash
# Enable isolation with opcache disabled (default for development)
magebox php isolate

# Enable with custom opcache settings
magebox php isolate --opcache-memory=512 --jit=tracing

# Enable with preload script
magebox php isolate --preload=/path/to/preload.php
```

**Options:**
- `--opcache-memory`: OPcache memory consumption in MB (e.g., 512)
- `--jit`: OPcache JIT mode (off, tracing, function)
- `--preload`: Path to preload script

---

### `magebox php isolate status`

Show isolation status for the current project.

```bash
magebox php isolate status
```

Displays:
- Whether project is using isolated or shared PHP-FPM
- Socket path, PID file, and config path
- PHP_INI_SYSTEM settings configured
- Running status

---

### `magebox php isolate disable`

Disable isolation and return to shared PHP-FPM pool.

```bash
magebox php isolate disable
```

---

### `magebox php isolate list`

List all projects with isolated PHP-FPM masters.

```bash
magebox php isolate list
```

---

### `magebox php isolate restart`

Restart the isolated PHP-FPM master for the current project.

```bash
magebox php isolate restart
```

::: tip When to Use Isolation
Use `php isolate` when you need:
- **Different opcache memory** per project
- **JIT compilation** for specific projects only
- **Preload scripts** that are project-specific
- **Complete PHP setting isolation** from other projects

For simple settings like `memory_limit` or `max_execution_time`, use `php ini` instead - those work at the pool level without isolation.
:::

---

### Running Magento CLI

Use the MageBox PHP wrapper or project shell to run Magento commands:

```bash
# Using the shell (recommended)
magebox shell
php bin/magento cache:flush

# Or directly with the wrapper
~/.magebox/bin/php bin/magento cache:flush
```

The PHP wrapper automatically uses the correct PHP version for your project.

::: tip
See [CLI Wrappers](/guide/php-wrapper) for more details on using `php`, `composer`, and other CLI tools.
:::

## Mode Commands

### `magebox dev`

Switch to development mode optimized for debugging.

```bash
magebox dev
```

This command configures:
- **OPcache:** Disabled (code changes apply immediately)
- **Xdebug:** Enabled (step debugging available)
- **Blackfire:** Disabled (conflicts with Xdebug)

Settings are persisted in `.magebox.local.yaml`.

---

### `magebox prod`

Switch to production mode optimized for performance.

```bash
magebox prod
```

This command configures:
- **OPcache:** Enabled (faster PHP execution)
- **Xdebug:** Disabled (no debugging overhead)
- **Blackfire:** Disabled (enable manually when profiling)

Settings are persisted in `.magebox.local.yaml`.

::: tip
Use `magebox dev` during active development, and `magebox prod` when testing production-like performance.
:::

## Custom Commands

### `magebox run <name>`

Execute a custom command.

```bash
magebox run deploy
magebox run reindex
```

Runs commands defined in `.magebox.yaml`:

```yaml
commands:
  deploy: "php bin/magento deploy:mode:set production"
```

**Options:**
- `--list` - Show available commands

## Database Commands

### `magebox db shell`

Open MySQL/MariaDB shell.

```bash
magebox db shell
```

Connects to the project's database with correct credentials.

---

### `magebox db import [file]`

Import SQL dump with progress tracking.

```bash
magebox db import dump.sql
magebox db import dump.sql.gz
```

**Arguments:**
- `file` - SQL file to import

**Features:**
- Real-time progress bar showing percentage, speed, and ETA
- Supports both plain SQL and gzipped files
- Tracks compressed file size for accurate progress on `.sql.gz` files

**Example output:**
```
Importing dump.sql.gz into database 'mystore' (magebox-mysql-8.0)
  Importing: ████████████████████░░░░░░░░░░░░░░░░░░░░ 52.3% (156.2 MB/298.5 MB) 24.5 MB/s ETA: 6s
```

---

### `magebox db export [file]`

Export database.

```bash
magebox db export backup.sql
magebox db export backup.sql.gz
magebox db export - > backup.sql
```

**Arguments:**
- `file` - Output file (use `-` for stdout)

---

### `magebox db create`

Create the project database.

```bash
magebox db create
```

Creates the database defined in `.magebox.yaml` if it doesn't exist.

---

### `magebox db drop`

Drop the project database.

```bash
magebox db drop
```

::: danger
This permanently deletes all data. Requires confirmation.
:::

---

### `magebox db reset`

Drop and recreate the project database.

```bash
magebox db reset
```

Equivalent to `db drop` followed by `db create`. Requires confirmation.

---

### `magebox db snapshot create [name]`

Create a database snapshot for quick backup.

```bash
magebox db snapshot create              # Auto-named with timestamp
magebox db snapshot create mybackup     # Named snapshot
```

**Arguments:**
- `name` - Snapshot name (optional, defaults to timestamp)

Snapshots are compressed with gzip and stored in `~/.magebox/snapshots/{project}/`.

---

### `magebox db snapshot restore <name>`

Restore database from a snapshot.

```bash
magebox db snapshot restore mybackup
```

**Arguments:**
- `name` - Snapshot name to restore (required)

::: warning
This replaces the current database. The existing data will be lost.
:::

---

### `magebox db snapshot list`

List all snapshots for the current project.

```bash
magebox db snapshot list
```

Shows snapshot name, size, and creation date.

---

### `magebox db snapshot delete <name>`

Delete a snapshot.

```bash
magebox db snapshot delete mybackup
```

**Arguments:**
- `name` - Snapshot name to delete (required)

## Redis Commands

### `magebox redis shell`

Open Redis CLI.

```bash
magebox redis shell
```

---

### `magebox redis flush`

Clear all Redis data.

```bash
magebox redis flush
```

---

### `magebox redis info`

Show Redis statistics.

```bash
magebox redis info
```

## Queue Commands

Commands for managing RabbitMQ message queues.

### `magebox queue status`

View RabbitMQ queue status with message counts.

```bash
magebox queue status
```

Shows all queues with:
- Queue name
- Message count (ready/unacked)
- Consumer count

Uses the RabbitMQ Management API.

---

### `magebox queue flush`

Purge all messages from all queues.

```bash
magebox queue flush
```

::: danger
This permanently deletes all queued messages. Use with caution!
:::

---

### `magebox queue consumer [name]`

Run Magento queue consumers.

```bash
# Run a specific consumer
magebox queue consumer product_action_attribute.update

# Run all consumers via cron
magebox queue consumer --all
```

**Arguments:**
- `name` - Consumer name (optional if using `--all`)

**Options:**
- `--all` - Start all configured consumers via Magento cron

## Log Commands

### `magebox logs`

View Magento logs in split-screen using multitail.

```bash
magebox logs
```

Opens `system.log` and `exception.log` side-by-side:
- Left column: `var/log/system.log`
- Right column: `var/log/exception.log`

**Keyboard controls:**
- `q` - Quit
- `b` - Scroll back in history
- `↑/↓` - Scroll in current window

::: tip
Requires `multitail`. Run `magebox bootstrap` to install it.
:::

---

### `magebox report`

Watch for Magento error reports.

```bash
magebox report
```

Monitors `var/report/` directory and displays:
- Latest error report on startup
- New reports as they're created in real-time

Press `Ctrl+C` to stop watching.

## Varnish Commands

### `magebox varnish status`

Show Varnish cache statistics.

```bash
magebox varnish status
```

---

### `magebox varnish purge [url]`

Purge specific URL from cache.

```bash
magebox varnish purge /category/page.html
```

---

### `magebox varnish flush`

Clear all Varnish cache.

```bash
magebox varnish flush
```

---

### `magebox varnish enable`

Enable Varnish for the current project.

```bash
magebox varnish enable
```

This command:
1. Updates `.magebox.yaml` to enable Varnish
2. Regenerates Nginx vhost to proxy through Varnish
3. Starts the Varnish Docker container
4. Reloads Nginx

---

### `magebox varnish disable`

Disable Varnish for the current project.

```bash
magebox varnish disable
```

Restores direct Nginx → PHP-FPM routing.

## Docker Commands (macOS)

Commands for managing Docker providers on macOS. On Linux, the default Docker installation is used.

### `magebox docker`

Show Docker provider status.

```bash
magebox docker
```

Displays:
- Current active Docker provider
- Docker socket path
- Available providers (Docker Desktop, Colima, OrbStack, etc.)
- Running status for each provider

---

### `magebox docker use <provider>`

Switch to a different Docker provider.

```bash
magebox docker use colima
magebox docker use orbstack
magebox docker use desktop
```

**Arguments:**
- `provider` - Docker provider to use

**Available providers:**
- `desktop` - Docker Desktop
- `colima` - Colima
- `orbstack` - OrbStack
- `rancher` - Rancher Desktop
- `lima` - Lima

After switching, you'll need to set `DOCKER_HOST` in your shell profile.

::: tip
OrbStack and Colima are lightweight alternatives to Docker Desktop with better performance on Apple Silicon.
:::

## Global Commands

### `magebox bootstrap`

First-time environment setup.

```bash
magebox bootstrap
```

Performs:
- Dependency checking
- Global configuration creation
- SSL CA setup
- Nginx configuration
- Docker service startup
- DNS configuration

---

### `magebox global start`

Start global services.

```bash
magebox global start
```

Starts Nginx and Docker services.

---

### `magebox global stop`

Stop all MageBox services.

```bash
magebox global stop
```

Stops all Docker containers and Nginx.

---

### `magebox global status`

Show all projects and services.

```bash
magebox global status
```

---

### `magebox list`

List all discovered projects.

```bash
magebox list
```

Shows projects found from Nginx vhost configurations.

---

### `magebox uninstall`

Clean uninstall of MageBox components.

```bash
magebox uninstall                    # Interactive uninstall
magebox uninstall --force            # Skip confirmation
magebox uninstall --dry-run          # Preview what would happen
magebox uninstall --keep-vhosts      # Keep nginx configurations
```

This command:
1. Stops all running MageBox projects
2. Removes CLI wrappers (php, composer, blackfire) from `~/.magebox/bin/`
3. Removes nginx vhost configurations
4. Cleans up MageBox configuration

**Options:**
- `--force` - Skip confirmation prompt
- `--dry-run` - Preview what would be removed without making changes
- `--keep-vhosts` - Preserve nginx vhost configurations

::: warning
This does not uninstall MageBox itself (the binary), only the components it manages.
:::

## SSL Commands

### `magebox ssl trust`

Trust the local certificate authority.

```bash
magebox ssl trust
```

Installs mkcert CA in system trust store.

---

### `magebox ssl generate`

Regenerate SSL certificates.

```bash
magebox ssl generate
```

Generates certificates for all configured domains.

## DNS Commands

### `magebox dns setup`

Configure DNS resolution.

```bash
magebox dns setup
```

Sets up hosts file or dnsmasq based on configuration.

---

### `magebox dns status`

Show DNS configuration.

```bash
magebox dns status
```

## Domain Commands

### `magebox domain add <host>`

Add a domain to the project.

```bash
magebox domain add store.test
magebox domain add de.store.test --store-code=german
magebox domain add api.store.test --root=pub --ssl=false
```

**Options:**
- `--store-code` - Magento store code (sets `MAGE_RUN_CODE`)
- `--root` - Document root relative to project (default: `pub`)
- `--ssl` - Enable SSL for the domain (default: `true`)

Automatically generates SSL certificate, creates nginx vhost, and reloads nginx.

---

### `magebox domain remove <host>`

Remove a domain from the project.

```bash
magebox domain remove old.store.test
```

Removes the nginx vhost and updates configuration.

---

### `magebox domain list`

List all configured domains.

```bash
magebox domain list
```

Shows URL, root, store code, and SSL status for each domain.

## Configuration Commands

### `magebox config show`

Display global configuration.

```bash
magebox config show
```

---

### `magebox config init`

Initialize configuration with defaults.

```bash
magebox config init
```

Creates `~/.magebox/config.yaml`.

---

### `magebox config set <key> <value>`

Set configuration value.

```bash
magebox config set dns_mode dnsmasq
magebox config set default_php 8.3
magebox config set tld local
magebox config set portainer true
```

**Available keys:**
- `dns_mode` - DNS resolution mode (hosts/dnsmasq)
- `default_php` - Default PHP version
- `tld` - Top-level domain (default: test)
- `portainer` - Enable Portainer UI (true/false)
- `editor` - Preferred editor
- `auto_start` - Auto-start services (true/false)

## Library Commands

Commands for managing the MageBox configuration library.

### `magebox lib status`

Show library status.

```bash
magebox lib status
```

Displays:
- Current library version
- Git branch and commit
- Local modifications
- Available updates
- Custom path (if configured)

---

### `magebox lib update`

Update the configuration library.

```bash
magebox lib update
```

Pulls the latest configuration files from the magebox-lib repository.

---

### `magebox lib path`

Show library path.

```bash
magebox lib path
```

Displays the filesystem path to the configuration library.

---

### `magebox lib list`

List available installers.

```bash
magebox lib list
```

Shows all available platform installer configurations.

---

### `magebox lib templates`

List available templates.

```bash
magebox lib templates
```

Lists all configuration templates organized by category (nginx, php, varnish, etc.).

---

### `magebox lib show [platform]`

Show installer configuration details.

```bash
magebox lib show           # Auto-detect platform
magebox lib show fedora    # Show specific platform
```

Displays the installer configuration with variable expansion.

---

### `magebox lib set <path>`

Set a custom library path.

```bash
magebox lib set ~/my-magebox-configs
magebox lib set /path/to/custom/lib
```

Use your own templates and installer configurations. The path should contain:
- `templates/` - Template files organized by category
- `installers/` - Platform-specific YAML configuration files

---

### `magebox lib unset`

Remove custom library path.

```bash
magebox lib unset
```

Reverts to using the default `~/.magebox/yaml` directory.

---

### `magebox lib reset`

Reset library to upstream.

```bash
magebox lib reset
```

Discards all local changes and resets to the upstream version.

::: tip
See the [Configuration Library](/guide/configuration-library) guide for detailed information about customizing templates and installers.
:::

## Update Commands

### `magebox self-update`

Update MageBox to latest version.

```bash
magebox self-update
```

Downloads and installs the latest release from GitHub.

---

### `magebox self-update check`

Check for updates.

```bash
magebox self-update check
```

Shows available updates without installing.

## Admin Commands

### `magebox admin list`

List all Magento admin users.

```bash
magebox admin list
```

---

### `magebox admin create`

Create a new admin user interactively.

```bash
magebox admin create
```

Prompts for username, email, password, first name, and last name.

---

### `magebox admin password <email> [password]`

Reset admin user password.

```bash
magebox admin password admin@example.com newpassword
magebox admin password admin@example.com  # Interactive
```

---

### `magebox admin disable-2fa`

Disable Two-Factor Authentication for local development.

```bash
magebox admin disable-2fa
```

::: warning
Only use this for local development, never on production.
:::

## Xdebug Commands

### `magebox xdebug on`

Enable Xdebug for the project's PHP version.

```bash
magebox xdebug on
```

---

### `magebox xdebug off`

Disable Xdebug.

```bash
magebox xdebug off
```

---

### `magebox xdebug status`

Show Xdebug installation and configuration status.

```bash
magebox xdebug status
```

## Blackfire Commands

### `magebox blackfire on`

Enable Blackfire profiler.

```bash
magebox blackfire on
```

---

### `magebox blackfire off`

Disable Blackfire profiler.

```bash
magebox blackfire off
```

---

### `magebox blackfire status`

Show Blackfire status.

```bash
magebox blackfire status
```

---

### `magebox blackfire install`

Install Blackfire agent and PHP extension.

```bash
magebox blackfire install
```

---

### `magebox blackfire config`

Configure Blackfire credentials.

```bash
# Interactive mode
magebox blackfire config

# Non-interactive with flags
magebox blackfire config \
  --server-id=your-server-id \
  --server-token=your-server-token \
  --client-id=your-client-id \
  --client-token=your-client-token
```

**Options:**
- `--server-id` - Blackfire Server ID
- `--server-token` - Blackfire Server Token
- `--client-id` - Blackfire Client ID
- `--client-token` - Blackfire Client Token

Get credentials from your [Blackfire account](https://blackfire.io). When flags are omitted, prompts interactively.

## Tideways Commands

### `magebox tideways on`

Enable Tideways profiler.

```bash
magebox tideways on
```

---

### `magebox tideways off`

Disable Tideways profiler.

```bash
magebox tideways off
```

---

### `magebox tideways status`

Show Tideways status.

```bash
magebox tideways status
```

---

### `magebox tideways install`

Install Tideways daemon and PHP extension.

```bash
magebox tideways install
```

---

### `magebox tideways config`

Configure Tideways API key.

```bash
# Interactive mode
magebox tideways config

# Non-interactive with flag
magebox tideways config --api-key=your-api-key
```

**Options:**
- `--api-key` - Tideways API key

Get your API key from your [Tideways account](https://tideways.com). When flag is omitted, prompts interactively.

## Team Commands

### `magebox team list`

List all configured teams.

```bash
magebox team list
```

---

### `magebox team add <name>`

Add a new team configuration.

```bash
# Interactive mode
magebox team add myteam

# Non-interactive with flags
magebox team add myteam \
  --provider github \
  --org myorganization \
  --auth ssh

# Self-hosted GitLab
magebox team add myteam \
  --provider gitlab \
  --org mygroup \
  --url https://gitlab.mycompany.com \
  --auth ssh

# Self-hosted Bitbucket Server
magebox team add myteam \
  --provider bitbucket \
  --org MYPROJECT \
  --url https://bitbucket.mycompany.com \
  --auth ssh

# With asset storage
magebox team add myteam \
  --provider github \
  --org myorg \
  --auth https \
  --asset-provider sftp \
  --asset-host backup.example.com \
  --asset-port 22 \
  --asset-path /backups \
  --asset-username deploy
```

**Options:**
- `--provider` - Repository provider (github/gitlab/bitbucket)
- `--org` - Organization/namespace name
- `--url` - Custom URL for self-hosted instances (GitLab CE/EE, Bitbucket Server)
- `--auth` - Authentication method (https/ssh, default: https)
- `--asset-provider` - Asset storage provider (sftp/ftp)
- `--asset-host` - Asset storage hostname
- `--asset-port` - Asset storage port
- `--asset-path` - Base path on asset storage
- `--asset-username` - Asset storage username

When flags are omitted, an interactive wizard prompts for the values.

::: tip
Use `--auth=https` for public repositories (no SSH key needed). Use `--auth=ssh` for private repositories.
:::

---

### `magebox team remove <name>`

Remove a team configuration.

```bash
magebox team remove myteam
```

---

### `magebox team <name> show`

Show team configuration details.

```bash
magebox team myteam show
```

---

### `magebox team <name> repos`

Browse repositories in the team's namespace.

```bash
magebox team myteam repos
magebox team myteam repos --filter "magento*"
```

**Options:**
- `--filter` - Glob pattern to filter repositories

---

### `magebox team <name> project list`

List all projects in a team.

```bash
magebox team myteam project list
```

---

### `magebox team <name> project add <project>`

Add a project to a team.

```bash
magebox team myteam project add shop \
  --repo myorg/shop \
  --branch main \
  --db shop/latest.sql.gz \
  --media shop/media.tar.gz
```

**Options:**
- `--repo` - Repository path (org/repo)
- `--branch` - Default branch (default: main)
- `--db` - Path to database dump on asset storage
- `--media` - Path to media archive on asset storage

---

### `magebox team <name> project remove <project>`

Remove a project from a team.

```bash
magebox team myteam project remove shop
```

---

### `magebox clone <project>`

Clone a team project repository.

```bash
magebox clone myteam/myproject
magebox clone myproject              # If only one team
```

**Options:**
- `--branch` - Specific branch to checkout
- `--fetch` - Also fetch database and media after cloning
- `--to` - Custom destination directory
- `--dry-run` - Show what would happen

**What it does:**
1. Clones the repository from the configured provider
2. Creates `.magebox.yaml` if not present
3. Runs `composer install`

---

### `magebox fetch`

Download database and media from team asset storage for the current project.

```bash
cd myproject
magebox fetch                        # Download & import database
magebox fetch --media                # Also download & extract media
```

**Options:**
- `--media` - Also download and extract media files
- `--backup` - Backup current database before importing
- `--team` - Specify team explicitly (if project in multiple teams)
- `--dry-run` - Show what would happen

::: tip
Run from within a project directory. Reads project name from `.magebox.yaml` and searches team asset storage for matching files.
:::

---

### `magebox sync`

Sync database and media for existing project with progress tracking.

```bash
magebox sync
magebox sync --db
magebox sync --media
```

**Options:**
- `--db` - Only sync database
- `--media` - Only sync media
- `--backup` - Backup current database before import
- `--dry-run` - Show what would happen

**Features:**
- Progress bar for database import (see `db import`)
- Progress bar for media extraction showing percentage, speed, and ETA

::: tip
Run from within a project directory. Auto-detects team from git remote.
:::

## Test Commands

Commands for running tests and code quality checks.

### `magebox test setup`

Interactive wizard to install testing tools.

```bash
magebox test setup
```

Installs PHPUnit, PHPStan, PHPCS, and PHPMD based on your selections.

---

### `magebox test unit`

Run PHPUnit unit tests.

```bash
magebox test unit
magebox test unit --filter=ProductTest
magebox test unit --testsuite=unit
```

**Options:**
- `--filter` - Filter tests by name
- `--testsuite` - Run specific test suite

---

### `magebox test integration`

Run Magento integration tests with optional RAM-based MySQL.

```bash
magebox test integration
magebox test integration --tmpfs              # Use RAM-based MySQL (10-100x faster)
magebox test integration --tmpfs --tmpfs-size=2g
magebox test integration --tmpfs --keep-alive # Keep container for repeated runs
```

**Options:**
- `--tmpfs` - Use RAM-based MySQL container
- `--tmpfs-size` - RAM allocation (default: 1g)
- `--mysql-version` - MySQL version (default: 8.0)
- `--keep-alive` - Keep container running after tests

---

### `magebox test phpstan`

Run PHPStan static analysis.

```bash
magebox test phpstan
magebox test phpstan --level=5
```

**Options:**
- `--level` - Analysis level 0-9 (default from config or 1)

---

### `magebox test phpcs`

Run PHP_CodeSniffer.

```bash
magebox test phpcs
magebox test phpcs --standard=PSR12
```

**Options:**
- `--standard` - Coding standard (Magento2/PSR12)

---

### `magebox test phpmd`

Run PHP Mess Detector.

```bash
magebox test phpmd
magebox test phpmd --ruleset=cleancode,design
```

**Options:**
- `--ruleset` - Rulesets to apply (comma-separated)

---

### `magebox test all`

Run all tests except integration.

```bash
magebox test all
```

Runs unit tests, PHPStan, PHPCS, and PHPMD in sequence. Ideal for CI/CD pipelines.

---

### `magebox test status`

Show installed testing tools and their configuration.

```bash
magebox test status
```

::: tip
See the [Testing & Code Quality](/guide/testing-tools) guide for detailed configuration options.
:::

## Utility Commands

### `magebox check`

Check project health and dependencies.

```bash
magebox check
```

Verifies:
- PHP version and extensions
- Required services (MySQL, Redis, etc.)
- SSL certificates
- Nginx vhost configuration
- File permissions

---

### `magebox install`

Check and install dependencies.

```bash
magebox install
```

Verifies required software and provides installation guidance.

---

### `magebox --version`

Show MageBox version.

```bash
magebox --version
```

---

### `magebox --help`

Show help information.

```bash
magebox --help
magebox start --help
```

# CLI Commands

Complete reference for all MageBox commands.

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
```

This command:
1. Generates PHP-FPM pool configuration
2. Generates Nginx vhost configuration
3. Starts required Docker services
4. Updates DNS (if hosts mode)
5. Generates SSL certificates (if needed)
6. Reloads Nginx

---

### `magebox stop`

Stop the project environment.

```bash
magebox stop
```

Stops PHP-FPM pool and removes Nginx configuration.

---

### `magebox restart`

Restart the project environment.

```bash
magebox restart
```

Equivalent to `stop` followed by `start`.

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

### `magebox cli [args...]`

Run Magento CLI command.

```bash
magebox cli cache:flush
magebox cli indexer:reindex
magebox cli setup:upgrade
```

Executes `php bin/magento` with correct PHP version and environment.

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

Import SQL dump.

```bash
magebox db import dump.sql
magebox db import dump.sql.gz
cat dump.sql | magebox db import
```

**Arguments:**
- `file` - SQL file to import (optional, reads from stdin if omitted)

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
magebox blackfire config
```

Prompts for Server ID, Server Token, Client ID, and Client Token from your [Blackfire account](https://blackfire.io).

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
magebox tideways config
```

Prompts for API key from your [Tideways account](https://tideways.com).

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
magebox team add myteam
```

Interactive wizard that configures:
- Repository provider (github/gitlab/bitbucket)
- Organization name
- Authentication method (ssh/token)
- Asset storage (optional)

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

### `magebox fetch <project>`

Fetch a project (clone + database + media).

```bash
magebox fetch myteam/myproject
magebox fetch myproject              # If only one team
```

**Options:**
- `--branch` - Specific branch to checkout
- `--no-db` - Skip database download/import
- `--no-media` - Skip media download/extraction
- `--db-only` - Only download and import database
- `--dry-run` - Show what would happen
- `--to` - Custom destination directory

---

### `magebox sync`

Sync database and media for existing project.

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

::: tip
Run from within a project directory. Auto-detects team from git remote.
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

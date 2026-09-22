# PHP-FPM

MageBox uses native PHP-FPM for maximum performance, with per-project pools supporting different PHP versions simultaneously.

## Overview

Each MageBox project gets its own PHP-FPM pool with:

- Dedicated Unix socket
- Project-specific PHP version
- Custom environment variables
- PHP INI overrides
- Isolated process group

## Supported Versions

| Version | Status | Magento Compatibility |
|---------|--------|----------------------|
| PHP 8.1 | Supported | Magento 2.4.4 - 2.4.6 |
| PHP 8.2 | Supported | Magento 2.4.6 - 2.4.7 |
| PHP 8.3 | Supported | Magento 2.4.7+ |
| PHP 8.4 | Supported | Future versions |

## Pool Configuration

### Location

Pool configurations are organized by PHP version:

```
~/.magebox/php/pools/{version}/{project}.conf
```

For example:
```
~/.magebox/php/pools/
├── 8.1/
│   └── project-a.conf
├── 8.2/
│   ├── project-b.conf
│   └── project-c.conf
└── 8.3/
    └── project-d.conf
```

This allows multiple PHP versions to run simultaneously without conflicts.

### Generated Structure

```ini
[mystore]

user = yourusername
group = yourgroup

listen = /tmp/magebox/mystore-php8.2.sock
listen.owner = yourusername
listen.group = yourgroup
listen.mode = 0666

pm = dynamic
pm.max_children = 50
pm.start_servers = 8
pm.min_spare_servers = 4
pm.max_spare_servers = 12
pm.max_requests = 1000

; Error logging
php_admin_value[error_log] = ~/.magebox/logs/php-fpm/mystore.log
php_admin_flag[log_errors] = on

; Magento recommended settings
php_value[memory_limit] = 756M
php_value[max_execution_time] = 18000
php_value[max_input_time] = 600
php_value[max_input_vars] = 10000

; OPcache settings
php_admin_value[opcache.enable] = 1
php_admin_value[opcache.memory_consumption] = 512
php_admin_value[opcache.max_accelerated_files] = 130986

; Environment variables
env[MAGE_MODE] = developer
```

### Process Manager

The `pm.*` directives above are the defaults. Override them per project with a `pm` block in `.magebox.yaml` or `.magebox.local.yaml`:

```yaml
pm:
  mode: dynamic          # static | dynamic | ondemand
  max_children: 12
  start_servers: 4
  min_spare_servers: 2
  max_spare_servers: 6
  max_requests: 1000
```

Only the keys relevant to the chosen mode are written to the pool file — `ondemand` emits `pm.process_idle_timeout` instead of the spare-server directives, and `static` emits neither.

#### Running many projects on one machine

Each pool defaults to `pm.max_children = 50`, sized as if it were the only project on the host. With a handful of projects started at once, their peaks can collectively demand far more memory and CPU than the machine has — PHP-FPM starts workers well past the core count, and each active Magento worker can hold hundreds of megabytes.

Idle pools are cheap (a waiting worker holds very little), so the cost shows up during bursts rather than at rest. Two ways to bound it:

```yaml
# ~/.magebox/config.yaml — applies to every project
default_pm:
  mode: ondemand
  max_children: 12
```

`ondemand` starts no workers until a request arrives and reaps them after `process_idle_timeout`, so dormant projects cost nothing. Alternatively keep `dynamic` and simply lower `max_children` — throughput is bounded by CPU cores, so a value near your core count loses nothing while keeping bursts from swapping the machine.

A project's own `pm` block overrides the global default key by key, so the project you are actively working on can stay `dynamic` while the rest run `ondemand`.

::: warning
A malformed pool prevents the PHP-FPM master from starting, which affects every project on that PHP version. MageBox validates the values before writing the pool file and reports the problem instead.
:::

See [Configuration Options](/reference/config-options#pm) for the full list of keys.

## PHP Version Management

### Checking Current Version

```bash
# Show project's PHP version
magebox php

# Or check the config
cat .magebox.yaml | grep php
```

### Switching Versions

```bash
# Switch to PHP 8.3
magebox php 8.3
```

This creates/updates `.magebox.local.yaml` and restarts services.

### PHP Wrapper

MageBox installs a smart PHP wrapper at `~/.magebox/bin/php` that automatically uses the correct PHP version:

```bash
# Add to ~/.zshrc or ~/.bashrc
export PATH="$HOME/.magebox/bin:$PATH"

# Now php uses the project's version
cd /path/to/project
php -v  # Uses PHP version from .magebox.yaml
```

### Running Multiple Versions

Different projects can use different PHP versions simultaneously:

```bash
# Project A uses PHP 8.1
cd /path/to/project-a
php -v  # PHP 8.1

# Project B uses PHP 8.3
cd /path/to/project-b
php -v  # PHP 8.3
```

Each project has its own PHP-FPM pool with the correct version.

## PHP INI Configuration

### Project-Level Overrides

Add `php_ini` section to `.magebox.yaml`:

```yaml
php_ini:
  memory_limit: "2G"
  max_execution_time: "3600"
  display_errors: "On"
  error_reporting: "E_ALL"
```

### Local Overrides

For development-specific settings, use `.magebox.local.yaml`:

```yaml
php_ini:
  opcache.enable: "0"
  xdebug.mode: "debug"
  xdebug.start_with_request: "yes"
```

### Common Settings

#### Development

```yaml
php_ini:
  opcache.enable: "0"           # See changes immediately
  display_errors: "On"
  error_reporting: "E_ALL"
  xdebug.mode: "debug,coverage"
```

#### Performance Testing

```yaml
php_ini:
  opcache.enable: "1"
  opcache.validate_timestamps: "0"
  display_errors: "Off"
```

#### Large Imports

```yaml
php_ini:
  memory_limit: "4G"
  max_execution_time: "7200"
  post_max_size: "256M"
  upload_max_filesize: "256M"
```

## Environment Variables

### Setting Variables

Add `env` section to `.magebox.yaml`:

```yaml
env:
  MAGE_MODE: developer
  COMPOSER_MEMORY_LIMIT: "-1"
  XDEBUG_MODE: debug
```

### Common Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `MAGE_MODE` | Magento mode | `developer`, `production` |
| `MAGE_RUN_CODE` | Store code | `default`, `german` |
| `MAGE_RUN_TYPE` | Run type | `store`, `website` |
| `COMPOSER_MEMORY_LIMIT` | Composer memory | `-1` (unlimited) |

## Process Management

### Pool Settings

Default process manager settings:

| Setting | Default | Description |
|---------|---------|-------------|
| `pm` | dynamic | Process manager type |
| `pm.max_children` | 50 | Maximum worker processes |
| `pm.start_servers` | 5 | Initial workers |
| `pm.min_spare_servers` | 2 | Minimum idle workers |
| `pm.max_spare_servers` | 10 | Maximum idle workers |
| `pm.max_requests` | 500 | Requests before worker restart |

### Checking Status

```bash
# Check if PHP-FPM is running
pgrep -l php-fpm

# Check pool status (if status page enabled)
curl http://localhost/status
```

## Logging

### Log Location

```bash
# Project-specific PHP-FPM logs
~/.magebox/logs/php-fpm/mystore.log

# View logs
tail -f ~/.magebox/logs/php-fpm/mystore.log
```

### Enabling Debug Logging

```yaml
# .magebox.local.yaml
php_ini:
  display_errors: "On"
  log_errors: "On"
  error_reporting: "E_ALL"
```

## Troubleshooting

### Socket Not Found

```
connect() to unix:/tmp/magebox/mystore-php8.2.sock failed
```

**Solution:**

```bash
# Check if socket exists
ls -la /tmp/magebox/

# Restart the project
magebox restart
```

### Wrong PHP Version

```bash
# Check which PHP is being used
which php
php -v

# Ensure MageBox bin is in PATH
echo $PATH | grep magebox

# Add to PATH if missing
export PATH="$HOME/.magebox/bin:$PATH"
```

### Memory Limit Errors

```yaml
# .magebox.yaml
php_ini:
  memory_limit: "2G"
```

Then restart:

```bash
magebox restart
```

### OPcache Issues

If code changes aren't reflected:

```yaml
# .magebox.local.yaml
php_ini:
  opcache.enable: "0"
```

Or flush OPcache:

```bash
# Clear OPcache via Magento
php bin/magento cache:flush
```

### Permission Denied

On Linux, ensure PHP-FPM runs as your user:

```bash
# Check pool config (replace 8.2 with your PHP version)
cat ~/.magebox/php/pools/8.2/mystore.conf | grep user

# Should show your username
```

## Installing PHP Versions

### macOS (Homebrew)

```bash
brew install php@8.1 php@8.2 php@8.3
```

### Ubuntu/Debian

```bash
sudo add-apt-repository ppa:ondrej/php
sudo apt update
sudo apt install php8.2-fpm php8.2-cli php8.2-common \
    php8.2-mysql php8.2-xml php8.2-curl php8.2-mbstring \
    php8.2-zip php8.2-gd php8.2-intl php8.2-bcmath php8.2-soap
```

### Fedora

```bash
sudo dnf install https://rpms.remirepo.net/fedora/remi-release-$(rpm -E %fedora).rpm
sudo dnf module enable php:remi-8.2
sudo dnf install php php-fpm php-cli php-common php-mysqlnd \
    php-xml php-curl php-mbstring php-zip php-gd php-intl php-bcmath php-soap
```

## Performance Tips

### OPcache Tuning

For production-like performance:

```yaml
php_ini:
  opcache.enable: "1"
  opcache.memory_consumption: "512"
  opcache.max_accelerated_files: "130986"
  opcache.validate_timestamps: "0"  # Don't check file changes
  opcache.interned_strings_buffer: "20"
```

### Realpath Cache

Already optimized in generated pools:

```ini
php_admin_value[realpath_cache_size] = 10M
php_admin_value[realpath_cache_ttl] = 7200
```

### Process Manager Tuning

For high-traffic testing, adjust in your pool config:

```ini
pm.max_children = 100
pm.start_servers = 10
pm.min_spare_servers = 5
pm.max_spare_servers = 20
```

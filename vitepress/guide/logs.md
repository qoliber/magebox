# Logs & Error Reports

MageBox provides powerful tools for viewing Magento logs and error reports in real-time.

## Overview

Debugging Magento issues often requires watching multiple log files simultaneously. MageBox includes a command to make this easier:

- **`magebox logs`** - View system.log and exception.log side-by-side

## magebox logs

Opens Magento's main log files in a split-screen terminal view using [multitail](https://github.com/folkertvanheusden/multitail).

### Usage

```bash
cd /path/to/magento
magebox logs
```

### What It Shows

The screen is split into 2 columns:

```
┌─────────────────────────────┬─────────────────────────────┐
│         system.log          │       exception.log         │
├─────────────────────────────┼─────────────────────────────┤
│ [2025-12-13 10:15:23] main. │ [2025-12-13 10:14:02]       │
│ INFO: Cache flush requested │ report.CRITICAL: Error      │
│                             │ processing order #100001    │
│ [2025-12-13 10:15:24] main. │                             │
│ INFO: Cache flushed         │ #0 /var/www/html/vendor/... │
│                             │ #1 /var/www/html/app/...    │
│                             │                             │
└─────────────────────────────┴─────────────────────────────┘
```

### Keyboard Controls

| Key | Action |
|-----|--------|
| `q` | Quit multitail |
| `b` | Scroll back through history |
| `↑` / `↓` | Scroll in current window |
| `F1` | Help menu |

### Features

- **Live updates** - New log entries appear in real-time
- **Scrollback buffer** - Last 500 lines available for review
- **Side-by-side view** - See errors and system events together
- **Auto-create** - Creates log files if they don't exist

### Example Session

```bash
$ magebox logs
Watching: /var/www/mystore/var/log
Press 'q' to quit, 'b' to scroll back

# Now trigger an action in Magento...
# Watch the logs update in real-time!
```

### Requirements

Requires `multitail` to be installed. MageBox bootstrap installs it automatically, or install manually:

```bash
# macOS
brew install multitail

# Fedora/RHEL
sudo dnf install multitail

# Ubuntu/Debian
sudo apt install multitail
```

## Service-Specific Logs

MageBox provides dedicated log commands for each service, making it easy to debug issues without hunting for log file paths.

### Global Flags

All `magebox logs` subcommands support these flags:

| Flag | Description |
|------|-------------|
| `-f`, `--follow` | Follow log output (tail -f) |
| `-n`, `--lines` | Number of lines to show (default: 100) |

### `magebox logs php`

View PHP-FPM error logs for the current project.

```bash
magebox logs php        # View in multitail (or tail if unavailable)
magebox logs php -f     # Follow mode
magebox logs php -n 50  # Show last 50 lines
```

Log files are located in `~/.magebox/logs/php-fpm/` and are matched by project name:

```
~/.magebox/logs/php-fpm/
├── mystore-error.log
└── mystore-slow.log
```

### `magebox logs nginx`

View Nginx access and error logs for the current project's domains.

```bash
magebox logs nginx        # View in multitail split-screen
magebox logs nginx -f     # Follow mode
```

Log files are located in `~/.magebox/logs/nginx/` with per-domain files:

```
~/.magebox/logs/nginx/
├── mystore.test-access.log
├── mystore.test-error.log
├── api.mystore.test-access.log
└── api.mystore.test-error.log
```

When multiple domains are configured, multitail shows them in a split-screen layout.

#### Common Nginx Errors

| Error | Likely Cause |
|-------|-------------|
| `502 Bad Gateway` | PHP-FPM not running or socket missing |
| `404 Not Found` | Wrong document root (should be `pub`) |
| `connect() failed` | PHP-FPM socket not found |

### `magebox logs mysql`

Stream logs from the MySQL or MariaDB Docker container.

```bash
magebox logs mysql          # Stream container logs
magebox logs mysql -n 200   # Show last 200 lines
```

Uses `docker compose logs` under the hood. Automatically detects whether your project uses MySQL or MariaDB. Press `Ctrl+C` to stop.

### `magebox logs redis`

Stream logs from the Redis (or Valkey) Docker container.

```bash
magebox logs redis          # Stream container logs
magebox logs redis -n 200   # Show last 200 lines
```

Uses `docker compose logs` under the hood. Press `Ctrl+C` to stop.

::: tip
Redis or Valkey must be configured in your `.magebox.yaml` for this command to work.
:::

## Combining Commands

For comprehensive debugging, run multiple log commands in separate terminal tabs:

**Terminal 1:**
```bash
magebox logs
```

**Terminal 2:**
```bash
magebox logs php -f
```

This gives you:
- Real-time Magento log monitoring (system + exceptions)
- PHP-FPM error tracking

## Tips

### Filtering Logs

For more advanced log filtering, you can use grep with tail:

```bash
# Watch for specific errors
tail -f var/log/system.log | grep -i "error"

# Watch for specific module
tail -f var/log/system.log | grep "Vendor_Module"
```

### Log Locations

| Log File | Purpose |
|----------|---------|
| `var/log/system.log` | General system events, cache, indexing |
| `var/log/exception.log` | PHP exceptions and stack traces |
| `var/log/debug.log` | Debug-level messages (when enabled) |
| `var/report/*` | Unhandled exception reports with unique IDs |
| `~/.magebox/logs/nginx/{domain}-access.log` | Per-domain nginx access logs (`magebox logs nginx`) |
| `~/.magebox/logs/nginx/{domain}-error.log` | Per-domain nginx error logs (`magebox logs nginx`) |
| `~/.magebox/logs/php-fpm/{project}-error.log` | PHP-FPM error logs per project (`magebox logs php`) |

### Cleaning Up Reports

Old report files can accumulate. Clean them periodically:

```bash
# Remove reports older than 7 days
find var/report -type f -mtime +7 -delete

# Remove all reports
rm -rf var/report/*
```

## Troubleshooting

### "multitail is not installed"

Run `magebox bootstrap` or install manually (see Requirements above).

### "Log directory not found"

Make sure you're in the Magento project root directory (where `app/`, `var/`, `pub/` exist).

### Logs Not Updating

1. Check Magento logging is enabled in `app/etc/env.php`
2. Verify file permissions on `var/log/`
3. Check disk space isn't full

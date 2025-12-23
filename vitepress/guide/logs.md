# Logs & Error Reports

MageBox provides powerful tools for viewing Magento logs and error reports in real-time.

## Overview

Debugging Magento issues often requires watching multiple log files simultaneously. MageBox includes two commands to make this easier:

- **`magebox logs`** - View system.log and exception.log side-by-side
- **`magebox report`** - Watch for new error reports in var/report

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

## magebox report

Watches the `var/report` directory for Magento error reports and displays them automatically.

### Usage

```bash
cd /path/to/magento
magebox report
```

### What It Shows

When an unhandled exception occurs, Magento writes a report file to `var/report/`. This command:

1. Shows the **latest existing report** immediately
2. **Watches for new reports** and displays them as they're created
3. Clears the screen and formats the report for readability

### Example Output

```
═══════════════════════════════════════════════════════════
                    Magento Error Report
═══════════════════════════════════════════════════════════

File: 1234567890123
Time: 2025-12-13 10:15:23
------------------------------------------------------------

a]0]:4:"type";i:1;s:7:"message";s:45:"Call to undefined method
Magento\Catalog\Model\Product::getInvalidMethod()";s:5:"file";
s:89:"/var/www/mystore/app/code/Vendor/Module/Controller/Index.php";
s:4:"line";i:42;s:5:"trace";s:2048:"#0 /var/www/mystore/vendor/
magento/framework/App/Action/Action.php(107): Vendor\Module\
Controller\Index->execute()
#1 /var/www/mystore/vendor/magento/framework/Interception/...

------------------------------------------------------------

Watching for new reports... (Ctrl+C to stop)
```

### How It Works

1. **Initial Display**: Shows the most recent report file on startup
2. **File Watching**: Uses filesystem notifications (fsnotify) to detect new files
3. **Auto-Refresh**: New reports are displayed immediately when created
4. **Clean Display**: Clears screen and formats each report for easy reading

### Triggering Test Errors

To test the report watcher, you can trigger an error:

```bash
# In another terminal, cause an error:
curl "https://mystore.test/nonexistent/page/that/causes/error"

# Or in PHP:
php -r "throw new Exception('Test error');"
```

### Stopping the Watcher

Press `Ctrl+C` to stop watching for new reports.

## Nginx Logs

MageBox stores per-domain nginx logs for easy debugging:

### Location

```
~/.magebox/logs/nginx/
├── mystore.test-access.log
├── mystore.test-error.log
├── api.mystore.test-access.log
└── api.mystore.test-error.log
```

Each domain configured in your project gets its own access and error log.

### Viewing Nginx Logs

```bash
# Watch access log for a specific domain
tail -f ~/.magebox/logs/nginx/mystore.test-access.log

# Watch error log for a specific domain
tail -f ~/.magebox/logs/nginx/mystore.test-error.log

# Watch all nginx logs
tail -f ~/.magebox/logs/nginx/*.log
```

### Common Errors

| Error | Likely Cause |
|-------|-------------|
| `502 Bad Gateway` | PHP-FPM not running or socket missing |
| `404 Not Found` | Wrong document root (should be `pub`) |
| `connect() failed` | PHP-FPM socket not found |

## Combining Both Commands

For comprehensive debugging, run both commands in separate terminal tabs:

**Terminal 1:**
```bash
magebox logs
```

**Terminal 2:**
```bash
magebox report
```

This gives you:
- Real-time log monitoring (system + exceptions)
- Instant notification of crash reports
- Full context when debugging issues

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
| `~/.magebox/logs/nginx/{domain}-access.log` | Per-domain nginx access logs |
| `~/.magebox/logs/nginx/{domain}-error.log` | Per-domain nginx error logs |
| `~/.magebox/logs/php-fpm/{project}.log` | PHP-FPM error logs per project |

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

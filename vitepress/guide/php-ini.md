# PHP INI Settings

MageBox allows you to override PHP configuration settings per-project using the `php_ini` section in your `.magebox.yaml` or `.magebox.local.yaml` file.

## Overview

PHP INI settings are injected into the PHP-FPM pool configuration and take precedence over system-wide PHP settings. This allows different projects to have different PHP configurations.

## Basic Usage

Add `php_ini` settings to your `.magebox.yaml` file:

```yaml
php_ini:
  opcache.enable: "0"            # Disable OPcache
  display_errors: "On"           # Show errors
  error_reporting: "E_ALL"       # Report all errors
  max_execution_time: "3600"     # 1 hour timeout
  memory_limit: "2G"             # 2GB memory
```

## Common Use Cases

### Disable OPcache for Development

By default, OPcache is enabled for performance. Disable it during active development to see code changes immediately:

```yaml
php_ini:
  opcache.enable: "0"
```

### Enable Xdebug

Configure Xdebug for debugging and profiling:

```yaml
php_ini:
  xdebug.mode: "debug,coverage"
  xdebug.start_with_request: "yes"
  xdebug.client_host: "localhost"
  xdebug.client_port: "9003"
```

::: tip
Make sure Xdebug is installed on your system:
```bash
# macOS
brew install php@8.3-xdebug

# Ubuntu/Debian
sudo apt install php8.3-xdebug
```
:::

### Increase Limits for Large Imports

For heavy operations like database imports or product imports:

```yaml
php_ini:
  max_execution_time: "7200"     # 2 hours
  memory_limit: "4G"             # 4GB
  post_max_size: "256M"
  upload_max_filesize: "256M"
```

### Production-like Settings

Optimize for production testing:

```yaml
php_ini:
  opcache.enable: "1"
  opcache.validate_timestamps: "0"  # Don't check file changes
  display_errors: "Off"
  error_reporting: "E_ALL & ~E_DEPRECATED & ~E_STRICT"
```

### Error Display and Logging

Debug issues with verbose error output:

```yaml
php_ini:
  display_errors: "On"
  display_startup_errors: "On"
  error_reporting: "E_ALL"
  log_errors: "On"
  error_log: "/tmp/php_errors.log"
```

## Development vs Production

Use `.magebox.local.yaml` for development-specific overrides that shouldn't be committed:

**.magebox.yaml** (committed to git):
```yaml
name: mystore
php: "8.3"

services:
  mysql: "8.0"
  redis: true
```

**.magebox.local.yaml** (in .gitignore):
```yaml
php_ini:
  opcache.enable: "0"            # Development only
  display_errors: "On"
  xdebug.mode: "debug"
```

## Available Directives

You can override any PHP INI directive that can be set via `php_admin_value` in PHP-FPM:

### Performance

| Directive | Description | Example |
|-----------|-------------|---------|
| `opcache.enable` | Enable OPcache | `"1"` or `"0"` |
| `opcache.validate_timestamps` | Check file changes | `"1"` or `"0"` |
| `realpath_cache_size` | Realpath cache | `"4096k"` |
| `realpath_cache_ttl` | Cache TTL | `"600"` |

### Debugging

| Directive | Description | Example |
|-----------|-------------|---------|
| `display_errors` | Show errors | `"On"` or `"Off"` |
| `display_startup_errors` | Show startup errors | `"On"` or `"Off"` |
| `error_reporting` | Error level | `"E_ALL"` |
| `log_errors` | Log errors | `"On"` or `"Off"` |
| `error_log` | Log file path | `"/tmp/php.log"` |

### Xdebug

| Directive | Description | Example |
|-----------|-------------|---------|
| `xdebug.mode` | Xdebug mode | `"debug,coverage"` |
| `xdebug.start_with_request` | Auto-start | `"yes"` or `"trigger"` |
| `xdebug.client_host` | IDE host | `"localhost"` |
| `xdebug.client_port` | IDE port | `"9003"` |

### Resource Limits

| Directive | Description | Example |
|-----------|-------------|---------|
| `memory_limit` | Max memory | `"2G"` |
| `max_execution_time` | Max runtime (seconds) | `"3600"` |
| `max_input_time` | Max input parsing time | `"300"` |
| `max_input_vars` | Max input variables | `"10000"` |

### Upload/Post

| Directive | Description | Example |
|-----------|-------------|---------|
| `post_max_size` | Max POST size | `"256M"` |
| `upload_max_filesize` | Max upload size | `"256M"` |
| `max_file_uploads` | Max simultaneous uploads | `"20"` |

### Session

| Directive | Description | Example |
|-----------|-------------|---------|
| `session.save_handler` | Session handler | `"redis"` |
| `session.gc_maxlifetime` | Session lifetime | `"86400"` |

## Applying Changes

After modifying `php_ini` settings, restart your project:

```bash
magebox restart
```

This regenerates the PHP-FPM pool configuration and reloads PHP-FPM.

## Viewing Generated Configuration

The generated PHP-FPM pool configuration is located at:

```
~/.magebox/php/pools/{project-name}.conf
```

View it to see all applied settings:

```bash
cat ~/.magebox/php/pools/mystore.conf
```

## Merge Behavior

When using both `.magebox.yaml` and `.magebox.local.yaml`:

- Settings in `.magebox.local.yaml` override `.magebox.yaml`
- Settings are merged at the key level
- Only specified keys are overridden

**Example:**

`.magebox.yaml`:
```yaml
php_ini:
  memory_limit: "1G"
  display_errors: "Off"
```

`.magebox.local.yaml`:
```yaml
php_ini:
  display_errors: "On"
  xdebug.mode: "debug"
```

**Result:**
```yaml
php_ini:
  memory_limit: "1G"        # From .magebox.yaml
  display_errors: "On"      # Overridden by .magebox.local.yaml
  xdebug.mode: "debug"      # Added from .magebox.local.yaml
```

## Troubleshooting

### Setting Not Applied

1. Make sure the directive can be set via PHP-FPM (not all can)
2. Restart the project: `magebox restart`
3. Check the generated pool config

### Syntax Errors

PHP INI values must be strings in YAML:

```yaml
# Correct
php_ini:
  memory_limit: "2G"
  max_execution_time: "3600"

# Incorrect (will cause issues)
php_ini:
  memory_limit: 2G
  max_execution_time: 3600
```

### Xdebug Not Working

1. Verify Xdebug is installed: `php -m | grep xdebug`
2. Check Xdebug version matches PHP version
3. Verify IDE is listening on the configured port

# Xdebug

::: warning Work in Progress
This feature is currently being tested and may not work as expected on all platforms.
:::

MageBox provides built-in Xdebug management for PHP debugging.

## Quick Start

```bash
# Enable Xdebug
magebox xdebug on

# Disable Xdebug
magebox xdebug off

# Check status
magebox xdebug status
```

## Commands

### Enable Xdebug

```bash
magebox xdebug on
```

This will:
1. Enable Xdebug for the project's PHP version
2. Restart PHP-FPM to apply changes
3. Configure debug mode with default settings

Default configuration:
- Mode: `debug`
- Client Host: `127.0.0.1`
- Client Port: `9003`
- IDE Key: `PHPSTORM`

### Disable Xdebug

```bash
magebox xdebug off
```

Disabling Xdebug removes the performance overhead when you don't need debugging.

### Check Status

```bash
magebox xdebug status
```

Shows:
- Whether Xdebug is installed
- Whether Xdebug is enabled
- Current mode
- INI file path

## IDE Configuration

### PhpStorm

1. Go to **Settings → PHP → Debug**
2. Set Debug Port to `9003`
3. Enable "Can accept external connections"

To start debugging:
1. Set breakpoints in your code
2. Click "Start Listening for PHP Debug Connections"
3. Open your site in the browser with `?XDEBUG_TRIGGER=1`

### VS Code

Install the **PHP Debug** extension, then add to `.vscode/launch.json`:

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Listen for Xdebug",
      "type": "php",
      "request": "launch",
      "port": 9003,
      "pathMappings": {
        "/path/to/project": "${workspaceFolder}"
      }
    }
  ]
}
```

## Environment Variables

Configure Xdebug behavior in `.magebox.yaml`:

```yaml
env:
  XDEBUG_MODE: debug
  XDEBUG_CONFIG: "client_host=127.0.0.1 client_port=9003"
  XDEBUG_SESSION: PHPSTORM
```

### Available Modes

| Mode | Description |
|------|-------------|
| `off` | Disable Xdebug |
| `debug` | Step debugging |
| `develop` | Development helpers |
| `coverage` | Code coverage |
| `profile` | Profiling |
| `trace` | Function tracing |

Combine modes with commas:

```yaml
env:
  XDEBUG_MODE: debug,coverage
```

## Triggering Debug Sessions

### Browser Extension

Install browser extensions for easy triggering:
- [Xdebug Helper for Chrome](https://chrome.google.com/webstore/detail/xdebug-helper)
- [Xdebug Helper for Firefox](https://addons.mozilla.org/en-US/firefox/addon/xdebug-helper-for-firefox/)

### Query Parameter

Add `?XDEBUG_TRIGGER=1` to any URL:

```
https://mystore.test/?XDEBUG_TRIGGER=1
```

### Cookie

Set a cookie named `XDEBUG_TRIGGER` with any value.

### CLI Debugging

For CLI scripts, set the environment variable:

```bash
XDEBUG_TRIGGER=1 php bin/magento cache:flush
```

Or in your shell:

```bash
export XDEBUG_TRIGGER=1
magebox shell
php bin/magento indexer:reindex
```

## Profiling

Enable profiling mode:

```yaml
env:
  XDEBUG_MODE: profile
  XDEBUG_OUTPUT_DIR: /tmp/xdebug
```

Profile files are saved as `cachegrind.out.*` and can be analyzed with:
- [KCacheGrind](https://kcachegrind.github.io/) (Linux)
- [QCacheGrind](https://sourceforge.net/projects/qcachegrindwin/) (Windows/macOS)
- [Webgrind](https://github.com/jokkedk/webgrind) (Web-based)

## Performance Considerations

Xdebug adds overhead even when not actively debugging. For best performance:

1. Keep Xdebug disabled when not needed:
   ```bash
   magebox xdebug off
   ```

2. Use `trigger_value` to require explicit activation:
   ```yaml
   env:
     XDEBUG_MODE: debug
     XDEBUG_START_WITH_REQUEST: trigger
   ```

3. Disable in production (never install Xdebug on production servers)

## Troubleshooting

### Xdebug Not Connecting

1. Check firewall allows port 9003
2. Verify IDE is listening for connections
3. Check `XDEBUG_CONFIG` has correct `client_host`

```bash
# Test connection from CLI
php -r "xdebug_info();"
```

### Xdebug Not Installed

Install Xdebug via PECL:

```bash
# macOS (Homebrew)
pecl install xdebug

# Ubuntu/Debian
sudo apt install php8.2-xdebug  # Replace with your PHP version

# Fedora
sudo dnf install php-xdebug
```

### Wrong PHP Version

MageBox uses the project's configured PHP version. Check:

```bash
magebox status
```

Ensure Xdebug is installed for that specific PHP version.

## Local Overrides

Keep Xdebug settings personal in `.magebox.local.yaml`:

```yaml
env:
  XDEBUG_MODE: debug
  XDEBUG_CONFIG: "client_host=host.docker.internal"
```

This file should be in `.gitignore`.

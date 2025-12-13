# Tideways

[Tideways](https://tideways.com/) is an application performance monitoring (APM) tool for PHP. MageBox provides full integration for monitoring your Magento store's performance.

## Installation

Install the Tideways daemon and PHP extension:

```bash
magebox tideways install
```

This installs:
- **Tideways Daemon** - Collects and forwards monitoring data
- **PHP Extension** - Instruments PHP for monitoring (installed for all PHP versions)

### Platform-Specific Installation

| Platform | Daemon | PHP Extension |
|----------|--------|---------------|
| macOS | `brew install tideways-daemon` | `brew install tideways-php` |
| Ubuntu/Debian | apt repository | apt repository |
| Fedora/RHEL | dnf repository | dnf repository |
| Arch Linux | AUR | AUR |

## Configuration

Configure your Tideways API key:

```bash
magebox tideways config
```

You'll be prompted for your API key from your Tideways account.

### Credential Storage

The API key is stored securely in `~/.magebox/config.yaml`:

```yaml
profiling:
  tideways:
    api_key: "your-api-key"
```

### Environment Variables

You can also use an environment variable (takes precedence):

```bash
export TIDEWAYS_API_KEY="your-api-key"
```

## Usage

### Enable Monitoring

```bash
magebox tideways on
```

This:
1. Enables the Tideways PHP extension
2. Starts the Tideways daemon
3. Disables Xdebug (to avoid conflicts)
4. Restarts PHP-FPM

### Disable Monitoring

```bash
magebox tideways off
```

This:
1. Disables the Tideways PHP extension
2. Stops the Tideways daemon
3. Restarts PHP-FPM

### Check Status

```bash
magebox tideways status
```

Output:
```
Tideways Status
===============

Daemon:
  ✓ Installed
  ✓ Running

PHP Extension:
  ✓ PHP 8.1: Installed, Enabled
  ✓ PHP 8.2: Installed, Enabled
  ✓ PHP 8.3: Installed, Enabled

Configuration:
  ✓ API key configured
```

## Monitoring Your Store

Once Tideways is enabled:

1. Browse your Magento store normally
2. Data is automatically sent to Tideways
3. View results in the [Tideways Dashboard](https://app.tideways.io/)

### What Tideways Tracks

- **Request traces** - Full execution traces for slow requests
- **Database queries** - SQL query performance and N+1 detection
- **External calls** - HTTP requests, Redis, cache operations
- **Errors and exceptions** - Automatic error tracking
- **Custom spans** - Add your own instrumentation

## Xdebug Conflict

::: warning
Tideways and Xdebug cannot run simultaneously. MageBox automatically disables Xdebug when enabling Tideways.
:::

To switch back to Xdebug:

```bash
magebox tideways off
# Then enable Xdebug in your php_ini settings
```

## Blackfire vs Tideways

| Feature | Blackfire | Tideways |
|---------|-----------|----------|
| **Purpose** | On-demand profiling | Continuous monitoring |
| **Data collection** | Manual trigger | Automatic sampling |
| **Best for** | Deep performance analysis | Production monitoring |
| **Overhead** | None when disabled | Minimal (~1-2%) |

::: tip
Use **Blackfire** for detailed profiling during development. Use **Tideways** for continuous monitoring in staging/production-like environments.
:::

## Troubleshooting

### Daemon Not Running

```bash
# Check daemon status
systemctl status tideways-daemon  # Linux
brew services list | grep tideways  # macOS

# Restart daemon
sudo systemctl restart tideways-daemon  # Linux
brew services restart tideways-daemon  # macOS
```

### Extension Not Loading

Check PHP configuration:

```bash
php -m | grep tideways
php --ini | grep tideways
```

### No Data in Dashboard

1. Verify the API key is correct:
   ```bash
   magebox tideways status
   ```

2. Check daemon logs:
   ```bash
   # Linux
   sudo journalctl -u tideways-daemon

   # macOS
   cat /usr/local/var/log/tideways-daemon.log
   ```

3. Verify network connectivity to Tideways servers

## Resources

- [Tideways Documentation](https://tideways.com/profiler/docs)
- [Tideways for Magento](https://tideways.com/profiler/docs/integrations/magento)
- [Tideways Dashboard](https://app.tideways.io/)

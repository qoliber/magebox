# Blackfire

[Blackfire](https://blackfire.io/) is a performance profiler for PHP applications. MageBox provides full integration for profiling your Magento store.

## Installation

Install the Blackfire agent and PHP extension:

```bash
magebox blackfire install
```

This installs:
- **Blackfire Agent** - Collects and forwards profiling data
- **PHP Extension** - Instruments PHP for profiling (installed for all PHP versions)

### Platform-Specific Installation

| Platform | Agent | PHP Extension |
|----------|-------|---------------|
| macOS | `brew install blackfire` | `brew install blackfire-php82` |
| Ubuntu/Debian | apt repository | apt repository |
| Fedora/RHEL | dnf repository | dnf repository |
| Arch Linux | AUR | AUR |

## Configuration

Configure your Blackfire credentials:

```bash
magebox blackfire config
```

You'll be prompted for:
- **Server ID** - From your Blackfire account
- **Server Token** - From your Blackfire account
- **Client ID** - For CLI profiling (optional)
- **Client Token** - For CLI profiling (optional)

### Credential Storage

Credentials are stored securely in `~/.magebox/config.yaml`:

```yaml
profiling:
  blackfire:
    server_id: "your-server-id"
    server_token: "your-server-token"
    client_id: "your-client-id"
    client_token: "your-client-token"
```

### Environment Variables

You can also use environment variables (these take precedence):

```bash
export BLACKFIRE_SERVER_ID="your-server-id"
export BLACKFIRE_SERVER_TOKEN="your-server-token"
export BLACKFIRE_CLIENT_ID="your-client-id"
export BLACKFIRE_CLIENT_TOKEN="your-client-token"
```

## Usage

### Enable Profiling

```bash
magebox blackfire on
```

This:
1. Enables the Blackfire PHP extension
2. Starts the Blackfire agent
3. Disables Xdebug (to avoid conflicts)
4. Restarts PHP-FPM

### Disable Profiling

```bash
magebox blackfire off
```

This:
1. Disables the Blackfire PHP extension
2. Stops the Blackfire agent
3. Restarts PHP-FPM

### Check Status

```bash
magebox blackfire status
```

Output:
```
Blackfire Status
================

Agent:
  ✓ Installed
  ✓ Running

PHP Extension:
  ✓ PHP 8.1: Installed, Enabled
  ✓ PHP 8.2: Installed, Enabled
  ✓ PHP 8.3: Installed, Enabled

Configuration:
  ✓ Credentials configured
```

## Profiling Your Store

### Browser Extension

1. Install the [Blackfire browser extension](https://blackfire.io/docs/integrations/browsers)
2. Navigate to your Magento store
3. Click the Blackfire icon to start profiling
4. View results in the Blackfire dashboard

### CLI Profiling

Profile bin/magento commands:

```bash
blackfire run php bin/magento cache:flush
blackfire run php bin/magento indexer:reindex
```

### Comparison Profiling

Compare before/after performance:

```bash
# Profile baseline
blackfire run --reference php bin/magento cache:flush

# Make changes, then profile again
blackfire run --reference php bin/magento cache:flush
```

## Xdebug Conflict

::: warning
Blackfire and Xdebug cannot run simultaneously. MageBox automatically disables Xdebug when enabling Blackfire.
:::

To switch back to Xdebug:

```bash
magebox blackfire off
# Then enable Xdebug in your php_ini settings
```

## Troubleshooting

### Agent Not Running

```bash
# Check agent status
systemctl status blackfire-agent  # Linux
brew services list | grep blackfire  # macOS

# Restart agent
sudo systemctl restart blackfire-agent  # Linux
brew services restart blackfire  # macOS
```

### Extension Not Loading

Check PHP configuration:

```bash
php -m | grep blackfire
php --ini | grep blackfire
```

### Credentials Not Found

Verify credentials are set:

```bash
magebox blackfire status
```

Or check environment variables:

```bash
echo $BLACKFIRE_SERVER_ID
```

## Resources

- [Blackfire Documentation](https://blackfire.io/docs)
- [Blackfire for Magento](https://blackfire.io/docs/integrations/magento)
- [Blackfire Browser Extension](https://blackfire.io/docs/integrations/browsers)

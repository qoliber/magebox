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

Configure your Tideways credentials:

```bash
magebox tideways config
```

Tideways uses **three separate credentials**. The API key is per-project;
the access token and environment label are global:

- **API Key** — the per-project key the PHP extension embeds into every
  transmitted trace. Found on each project's **Installation** page in the
  Tideways dashboard:
  `https://app.tideways.io/o/<organization>/<project>/installation`.
  Because each key is tied to a single Tideways project, it **must** be set
  per Magento project, not globally. MageBox stores it in the project's
  `.magebox.local.yaml` under `php_ini.tideways.api_key`, and the project's
  PHP-FPM pool renders it as a `php_admin_value` so the extension picks it
  up on the next reload.
- **Access Token** — a personal token used by the `tideways` commandline
  tool (`tideways run`, `tideways event create`, `tideways tracepoint create`).
  MageBox imports it via `tideways import <token>`. Generated at
  [app.tideways.io/user/cli-import-settings](https://app.tideways.io/user/cli-import-settings).
  Stored globally in `~/.magebox/config.yaml`. Optional — only needed if
  you use the `tideways` CLI.
- **Environment Label** — the environment name your traces are tagged with
  in Tideways (defaults to `local_<username>`). Stored globally and written
  to a `tideways-daemon` systemd drop-in on Linux.

### Non-interactive configuration

```bash
# Global (access token + environment label)
magebox tideways config --access-token "your-cli-token" --environment "local_alice"

# Per-project API key (run from inside the project)
magebox tideways config --project-api-key "your-project-key"
```

When run interactively from inside a project that does not yet have an API
key set, `magebox tideways config` prompts for it after the environment
prompt and writes it to `.magebox.local.yaml`.

### Credential Storage

**Per-project API key** lives in `.magebox.local.yaml` at the project root:

```yaml
php_ini:
  tideways:
    api_key: "your-project-key"
```

`.magebox.local.yaml` is the personal override file and is typically
gitignored, which keeps the secret out of the shared repo. To share a key
with your team, move it to `.magebox.yaml` manually.

After changing the API key, run `magebox restart` so the FPM pool picks up
the new `php_admin_value`.

**Global access token and environment** live in `~/.magebox/config.yaml`:

```yaml
profiling:
  tideways:
    access_token: "your-cli-token"
    environment: "local_alice"
```

### Environment Variables

The global credentials can be overridden via environment variables:

```bash
export TIDEWAYS_CLI_TOKEN="your-cli-token"   # read by MageBox for the CLI import
export TIDEWAYS_ENVIRONMENT="local_alice"    # environment label for the daemon
```

::: warning Removed in 1.14.2
The `TIDEWAYS_API_KEY` environment variable is no longer read. Because
the API key is now per-project, set it via `--project-api-key` or in
`.magebox.local.yaml` instead.
:::

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

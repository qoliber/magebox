# MageBox Commands - Test Mode Compatibility

This document lists all MageBox commands and their compatibility with test mode (`MAGEBOX_TEST_MODE=1`).

## Legend

| Symbol | Meaning |
|--------|---------|
| ✅ Yes | Fully works in test mode |
| ⚠️ Partial | Partially works (some features skipped) |
| ❌ No | Requires Docker/services |
| 🔒 Root | Requires root/sudo access |

## Command Reference

### Core Commands

| Command | Subcommands | Test Mode | Notes |
|---------|-------------|-----------|-------|
| `magebox --version` | - | ✅ Yes | |
| `magebox --help` | - | ✅ Yes | |
| `magebox init` | - | ✅ Yes | Creates .magebox.yaml |
| `magebox check` | - | ✅ Yes | Validates config |
| `magebox status` | - | ✅ Yes | Shows "(test mode)" for Docker services |
| `magebox list` | - | ✅ Yes | Discovers from nginx vhosts |
| `magebox start` | `--all` | ⚠️ Partial | PHP-FPM/Nginx work, Docker skipped |
| `magebox stop` | `--all`, `--dry-run` | ⚠️ Partial | Nginx/PHP-FPM work, Docker skipped |
| `magebox restart` | `--all` | ⚠️ Partial | Same as start/stop |
| `magebox uninstall` | `--dry-run` | ✅ Yes | --dry-run works fully |

### Configuration

| Command | Subcommands | Test Mode | Notes |
|---------|-------------|-----------|-------|
| `magebox config init` | - | ✅ Yes | Creates global config |
| `magebox config show` | - | ✅ Yes | Reads config |
| `magebox config set` | - | ✅ Yes | Modifies config |

### Domain Management

| Command | Subcommands | Test Mode | Notes |
|---------|-------------|-----------|-------|
| `magebox domain list` | - | ✅ Yes | Reads config |
| `magebox domain add` | - | ✅ Yes | Modifies config, regenerates vhost |
| `magebox domain remove` | - | ✅ Yes | Modifies config |

### SSL Certificates

| Command | Subcommands | Test Mode | Notes |
|---------|-------------|-----------|-------|
| `magebox ssl generate` | - | ✅ Yes | Uses mkcert (no Docker needed) |
| `magebox ssl trust` | - | 🔒 Root | Trusts local CA (needs sudo) |

### DNS Configuration

| Command | Subcommands | Test Mode | Notes |
|---------|-------------|-----------|-------|
| `magebox dns setup` | - | 🔒 Root | Sets up dnsmasq (needs sudo) |
| `magebox dns status` | - | ✅ Yes | Shows DNS configuration |

### PHP Tools

| Command | Subcommands | Test Mode | Notes |
|---------|-------------|-----------|-------|
| `magebox php` | - | ✅ Yes | Switches PHP version in config |
| `magebox xdebug on` | - | ✅ Yes | Modifies PHP config |
| `magebox xdebug off` | - | ✅ Yes | Modifies PHP config |
| `magebox xdebug status` | - | ✅ Yes | Checks PHP config |
| `magebox blackfire on` | - | ✅ Yes | Modifies PHP config |
| `magebox blackfire off` | - | ✅ Yes | Modifies PHP config |
| `magebox blackfire status` | - | ✅ Yes | Checks status |
| `magebox blackfire config` | - | ✅ Yes | Sets credentials |
| `magebox blackfire install` | - | 🔒 Root | Installs system packages |

### Logs & Reports

| Command | Subcommands | Test Mode | Notes |
|---------|-------------|-----------|-------|
| `magebox logs` | - | ✅ Yes | Reads Magento log files |
| `magebox report` | - | ✅ Yes | Reads Magento report files |

### Database (Requires Docker)

| Command | Subcommands | Test Mode | Notes |
|---------|-------------|-----------|-------|
| `magebox db create` | - | ❌ No | Needs MySQL container |
| `magebox db drop` | - | ❌ No | Needs MySQL container |
| `magebox db export` | - | ❌ No | Needs MySQL container |
| `magebox db import` | - | ❌ No | Needs MySQL container |
| `magebox db reset` | - | ❌ No | Needs MySQL container |
| `magebox db shell` | - | ❌ No | Needs MySQL container |

### Redis (Requires Docker)

| Command | Subcommands | Test Mode | Notes |
|---------|-------------|-----------|-------|
| `magebox redis flush` | - | ❌ No | Needs Redis container |
| `magebox redis info` | - | ❌ No | Needs Redis container |
| `magebox redis shell` | - | ❌ No | Needs Redis container |

### Varnish (Requires Docker)

| Command | Subcommands | Test Mode | Notes |
|---------|-------------|-----------|-------|
| `magebox varnish enable` | - | ❌ No | Needs Varnish container |
| `magebox varnish disable` | - | ❌ No | Needs Varnish container |
| `magebox varnish flush` | - | ❌ No | Needs Varnish container |
| `magebox varnish purge` | - | ❌ No | Needs Varnish container |
| `magebox varnish status` | - | ❌ No | Needs Varnish container |

### Admin (Requires Database)

| Command | Subcommands | Test Mode | Notes |
|---------|-------------|-----------|-------|
| `magebox admin create` | - | ❌ No | Needs DB connection |
| `magebox admin list` | - | ❌ No | Needs DB connection |
| `magebox admin password` | - | ❌ No | Needs DB connection |
| `magebox admin disable-2fa` | - | ❌ No | Needs DB connection |

### Global Services

| Command | Subcommands | Test Mode | Notes |
|---------|-------------|-----------|-------|
| `magebox global start` | - | ❌ No | Starts Docker services |
| `magebox global stop` | - | ❌ No | Stops Docker services |
| `magebox global status` | - | ⚠️ Partial | Can check, Docker services skipped |

### Team Collaboration

| Command | Subcommands | Test Mode | Notes |
|---------|-------------|-----------|-------|
| `magebox team add` | - | ✅ Yes | Config only |
| `magebox team list` | - | ✅ Yes | Config only |
| `magebox team remove` | - | ✅ Yes | Config only |
| `magebox team <name> show` | - | ✅ Yes | Config only |
| `magebox team <name> repos` | - | ✅ Yes | API call to provider |

### Other Commands

| Command | Subcommands | Test Mode | Notes |
|---------|-------------|-----------|-------|
| `magebox completion` | bash/zsh/fish/powershell | ✅ Yes | Generates shell completion |
| `magebox self-update` | - | ✅ Yes | Downloads new binary |
| `magebox new` | - | ⚠️ Partial | Composer works, services need Docker |
| `magebox fetch` | - | ⚠️ Partial | Git clone works, DB/media need services |
| `magebox sync` | - | ❌ No | Needs running services |
| `magebox shell` | - | ✅ Yes | Opens shell in project dir |
| `magebox run` | - | ✅ Yes | Runs custom command |
| `magebox bootstrap` | - | 🔒 Root | Installs system packages |
| `magebox install` | - | 🔒 Root | Installs dependencies |

## Test Mode Behavior

When `MAGEBOX_TEST_MODE=1` is set:

1. **Docker operations are skipped:**
   - `docker-compose up/down` not called
   - Container health checks skipped
   - Database creation skipped
   - Redis flush skipped

2. **DNS operations are skipped:**
   - No modifications to `/etc/hosts`
   - dnsmasq configuration skipped

3. **Status shows test mode:**
   - Docker services show "(test mode)" suffix
   - Services reported as stopped

4. **What still works:**
   - PHP-FPM pool generation
   - Nginx vhost generation
   - SSL certificate generation
   - Configuration file management
   - Project discovery
   - All config-only commands

## Summary Statistics

| Category | Total | Works in Test Mode |
|----------|-------|-------------------|
| Core Commands | 10 | 7 fully, 3 partial |
| Config Commands | 3 | 3 fully |
| Domain Commands | 3 | 3 fully |
| SSL Commands | 2 | 1 fully, 1 needs root |
| DNS Commands | 2 | 1 fully, 1 needs root |
| PHP Tools | 10 | 9 fully, 1 needs root |
| Log Commands | 2 | 2 fully |
| Database Commands | 6 | 0 (needs Docker) |
| Redis Commands | 3 | 0 (needs Docker) |
| Varnish Commands | 5 | 0 (needs Docker) |
| Admin Commands | 4 | 0 (needs Docker) |
| Global Commands | 3 | 1 partial |
| Team Commands | 5 | 5 fully |
| Other Commands | 8 | 4 fully, 2 partial, 2 need root |

**Total: ~66 commands/subcommands**
- **~35 work fully** in test mode
- **~6 work partially** in test mode
- **~18 require Docker** (skipped)
- **~7 require root** access

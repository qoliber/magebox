# Migrating from Valet / Valet+

This guide helps you migrate your Magento projects from [Laravel Valet](https://laravel.com/docs/valet) or [Valet+](https://github.com/weprovide/valet-plus) to MageBox.

## Why Migrate from Valet?

### The Valet Situation

Laravel Valet is excellent for Laravel projects but has limitations for Magento:

- **Valet** - Designed for Laravel, limited Magento support
- **Valet+** - Magento-focused fork, but maintenance has slowed
- **Valet Linux** - Community port with varying compatibility

### Performance Comparison

| Aspect | Valet/Valet+ | MageBox |
|--------|--------------|---------|
| **PHP-FPM** | Native ✓ | Native ✓ |
| **Nginx** | Native ✓ | Native ✓ |
| **MySQL** | Native (manual) | Docker (managed) |
| **Redis** | Native (manual) | Docker (managed) |
| **Multi-PHP** | Manual switching | Per-project config |
| **Services** | Manual setup | Automatic |

### Key Benefits of MageBox

1. **Automatic Service Management** - MySQL, Redis, OpenSearch managed per-project
2. **Per-Project PHP Versions** - No global switching needed
3. **Magento-Optimized** - Built specifically for Magento workflows
4. **Consistent Environments** - Same config across team members
5. **Database Isolation** - Each project gets its own database automatically
6. **Active Development** - Regularly updated for latest Magento versions

## What's Similar?

MageBox shares Valet's philosophy:
- Native PHP-FPM and Nginx for speed
- Minimal resource usage
- Simple configuration
- `.test` domain by default

The difference: MageBox adds **managed Docker services** and **Magento-specific tooling**.

## Migration Steps

### Step 1: Export Your Databases

With Valet, you likely have MySQL installed via Homebrew:

```bash
# List your databases
mysql -u root -e "SHOW DATABASES;"

# Export each Magento database
mysqldump -u root your_database > your_database.sql
```

### Step 2: Note Your Configuration

Check your current setup:

```bash
# PHP version
php -v

# Linked sites
valet links

# Parked directories
valet paths
```

### Step 3: Stop Valet Services

```bash
valet stop
```

### Step 4: Install MageBox

```bash
# macOS
brew tap qoliber/magebox
brew install magebox

# Run bootstrap (installs dependencies, configures system)
magebox bootstrap
```

::: warning Note
MageBox bootstrap will set up its own Nginx configuration. Your Valet nginx configs will remain but won't conflict.
:::

### Step 5: Create MageBox Configuration

In your Magento project root, create `.magebox.yaml`:

```yaml
name: your-project
php: "8.2"

domains:
  - host: your-project.test
    root: pub

services:
  mysql: "8.0"
  redis: true
  opensearch: "2.19.4"
  mailpit: true
```

### Step 6: Start MageBox

```bash
cd /path/to/your/magento
magebox start
```

### Step 7: Import Your Database

```bash
magebox db import your_database.sql
```

### Step 8: Update Magento Config

MageBox auto-generates `env.php`, but verify:

```bash
bin/magento setup:db:status
bin/magento cache:flush
```

## Configuration Mapping

### Valet → MageBox

| Valet Concept | MageBox Equivalent |
|---------------|-------------------|
| `valet link mysite` | `domains[].host` in `.magebox.yaml` |
| `valet secure mysite` | SSL enabled by default |
| `valet php@8.2` | `php: "8.2"` in `.magebox.yaml` |
| `valet park` | Not needed - per-project config |
| `~/.config/valet/` | `~/.magebox/` |

### PHP Version Switching

**Valet** - Global switch affects all projects:
```bash
valet use php@8.2
```

**MageBox** - Per-project, automatic:
```yaml
# .magebox.yaml
php: "8.2"  # This project uses 8.2
```

```yaml
# Another project's .magebox.yaml
php: "8.3"  # This project uses 8.3
```

No manual switching needed!

## Service Differences

### MySQL

**Valet**: Manual Homebrew MySQL, shared across all projects

```bash
# Valet - one MySQL for everything
brew services start mysql
mysql -u root
```

**MageBox**: Docker MySQL, per-version, managed automatically

```bash
# MageBox - automatic, version-specific
magebox start  # MySQL starts automatically
magebox db shell  # Connect to project database
```

### Redis

**Valet**: Manual setup

```bash
# Valet
brew install redis
brew services start redis
```

**MageBox**: Automatic when configured

```yaml
# MageBox - just add to config
services:
  redis: true
```

### Elasticsearch/OpenSearch

**Valet/Valet+**: Complex manual setup

**MageBox**: One line

```yaml
services:
  opensearch: "2.19.4"
```

## Command Comparison

| Task | Valet | MageBox |
|------|-------|---------|
| Start services | `valet start` | `magebox start` |
| Stop services | `valet stop` | `magebox stop` |
| Link project | `valet link name` | Add to `.magebox.yaml` |
| Secure (SSL) | `valet secure name` | Automatic |
| Switch PHP | `valet use php@8.2` | Set in `.magebox.yaml` |
| View logs | `valet log` | `tail -f /var/log/nginx/*.log` |
| Trust certs | `valet trust` | `mkcert -install` |

## Handling Valet+ Features

If you're using Valet+ specific features:

### Elasticsearch Driver

Valet+:
```bash
valet elasticsearch on
```

MageBox:
```yaml
services:
  opensearch: "2.19.4"  # Or elasticsearch: "8.11"
```

### Xdebug

Valet+:
```bash
valet xdebug on
```

MageBox:
```bash
magebox xdebug on
```

### Mailhog/Mailpit

Valet+:
```bash
valet mailhog on
```

MageBox:
```yaml
services:
  mailpit: true  # Modern Mailhog replacement
```

## Running Both (Temporarily)

During migration, you can run both:

1. Valet uses ports 80/443 by default
2. MageBox uses ports 80/443 by default

To run both temporarily:

```bash
# Stop Valet's nginx
valet stop

# Start MageBox
magebox start

# When done testing, switch back
magebox stop
valet start
```

## Removing Valet

Once you're confident with MageBox:

```bash
# Uninstall Valet
valet uninstall

# Or just stop it
valet stop
brew services stop nginx
brew services stop php
brew services stop dnsmasq

# Remove Valet config
rm -rf ~/.config/valet
```

Keep your Homebrew MySQL if you need it for other purposes, or remove it:

```bash
brew services stop mysql
brew uninstall mysql
```

## Multi-Store Setup

**Valet**: Requires custom Nginx config per store

**MageBox**: Built-in support

```yaml
domains:
  - host: store1.test
    root: pub
    store_code: store1
  - host: store2.test
    root: pub
    store_code: store2
```

## Benefits Summary

| Feature | Valet/Valet+ | MageBox |
|---------|--------------|---------|
| Native PHP/Nginx | ✓ | ✓ |
| Automatic SSL | ✓ | ✓ |
| Per-project PHP | ✗ (global) | ✓ |
| Managed MySQL | ✗ | ✓ |
| Managed Redis | ✗ | ✓ |
| Managed OpenSearch | ✗ | ✓ |
| Managed RabbitMQ | ✗ | ✓ |
| Multi-store support | Manual | Built-in |
| Magento env.php | Manual | Auto-generated |
| Database per project | Manual | Automatic |
| Cross-platform | macOS only* | macOS + Linux |

*Valet Linux exists but is a separate project

## Need Help?

- [MageBox Documentation](/)
- [GitHub Issues](https://github.com/qoliber/magebox/issues)
- [Troubleshooting Guide](/guide/troubleshooting)

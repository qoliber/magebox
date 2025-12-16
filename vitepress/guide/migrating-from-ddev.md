# Migrating from DDEV

This guide helps you migrate your Magento projects from [DDEV](https://ddev.com/) to MageBox.

## Why Migrate from DDEV?

### Performance Comparison

| Aspect | DDEV | MageBox |
|--------|------|---------|
| **PHP Execution** | Docker container | Native PHP-FPM |
| **File Sync** | Mutagen (macOS) / bind mount | Native filesystem |
| **Cold Start** | 15-45 seconds | 2-5 seconds |
| **Memory Usage** | 1.5-3 GB | 500 MB - 1 GB |
| **Page Load** | 1.5-4 seconds | 0.5-1.5 seconds |

### Key Benefits of MageBox

1. **Native Performance** - No Docker filesystem overhead for PHP/Nginx
2. **Simpler Setup** - No Mutagen configuration or sync issues
3. **Instant Switching** - Jump between projects without container management
4. **Resource Efficient** - Only database services run in Docker
5. **Better Debugging** - Native Xdebug without Docker network complexity

## Architecture Differences

### DDEV Architecture
```
┌─────────────────────────────────────────┐
│           DDEV Docker Network           │
│  ┌─────────────────────────────────┐   │
│  │    ddev-webserver Container     │   │
│  │    (Nginx + PHP-FPM + Node)     │   │
│  └─────────────────────────────────┘   │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐   │
│  │   DB    │ │ ddev-   │ │ Mailhog │   │
│  │Container│ │ router  │ │Container│   │
│  └─────────┘ └─────────┘ └─────────┘   │
└─────────────────────────────────────────┘
        │ Mutagen Sync / Bind Mount
        ▼
   [Your Code]
```

### MageBox Architecture
```
┌─────────────────────────────────────────┐
│           Native (No Sync Needed)       │
│  ┌─────────────────────────────────┐   │
│  │  Nginx + PHP-FPM (per project)  │   │
│  └─────────────────────────────────┘   │
│              │ Direct Access            │
│              ▼                          │
│         [Your Code]                     │
└─────────────────────────────────────────┘
┌─────────────────────────────────────────┐
│         Docker (Services Only)          │
│  ┌───────┐ ┌───────┐ ┌──────────┐      │
│  │ MySQL │ │ Redis │ │ Mailpit  │      │
│  └───────┘ └───────┘ └──────────┘      │
└─────────────────────────────────────────┘
```

## Migration Steps

### Step 1: Export Your Database

In your DDEV project:

```bash
# Start DDEV if not running
ddev start

# Export database
ddev export-db --file=database-backup.sql.gz

# Or without compression
ddev export-db --gzip=false --file=database-backup.sql
```

### Step 2: Note Your Configuration

Check your `.ddev/config.yaml`:

```bash
cat .ddev/config.yaml
```

Key settings to note:
- `php_version`
- `database` (type and version)
- `webserver_type`
- `additional_hostnames`

### Step 3: Stop DDEV

```bash
ddev stop
```

### Step 4: Install MageBox

If you haven't already:

```bash
# macOS
brew tap qoliber/magebox
brew install magebox
magebox bootstrap

# Linux (Ubuntu)
curl -fsSL https://raw.githubusercontent.com/qoliber/magebox/main/install.sh | bash
magebox bootstrap
```

### Step 5: Create MageBox Configuration

Create `.magebox.yaml` in your project root:

```yaml
name: your-project-name
php: "8.2"  # Match your DDEV PHP version

domains:
  - host: your-project.test
    root: pub

services:
  mysql: "8.0"      # Map from DDEV database config
  redis: true
  opensearch: "2.19.4"
  mailpit: true
```

### Step 6: Start MageBox

```bash
magebox start
```

### Step 7: Import Your Database

```bash
# If gzipped
gunzip database-backup.sql.gz

# Import
magebox db import database-backup.sql
```

### Step 8: Verify Magento

```bash
# Check database connection
bin/magento setup:db:status

# Clear cache
bin/magento cache:flush

# Test the site
curl -I https://your-project.test
```

## Configuration Mapping

### DDEV → MageBox

| DDEV (`.ddev/config.yaml`) | MageBox (`.magebox.yaml`) |
|---------------------------|---------------------------|
| `name` | `name` |
| `php_version` | `php` |
| `additional_hostnames` | `domains[].host` |
| `database.type: mysql` | `services.mysql` |
| `database.type: mariadb` | `services.mariadb` |
| `database.version` | `services.mysql: "X.X"` |

### DDEV Database Versions → MageBox

| DDEV Database | MageBox Config |
|---------------|----------------|
| `mysql:8.0` | `mysql: "8.0"` |
| `mysql:5.7` | `mysql: "5.7"` |
| `mariadb:10.6` | `mariadb: "10.6"` |
| `mariadb:10.11` | `mariadb: "10.11"` |

### Hooks Migration

DDEV hooks in `.ddev/config.yaml`:

```yaml
# DDEV
hooks:
  post-start:
    - exec: "bin/magento cache:flush"
```

MageBox custom commands in `.magebox.yaml`:

```yaml
# MageBox
commands:
  flush:
    description: "Flush Magento cache"
    run: "bin/magento cache:flush"
```

Run with: `magebox run flush`

## Common Differences

### Database Access

| | DDEV | MageBox |
|--|------|---------|
| Host (from Magento) | `db` | `127.0.0.1` |
| Host (from host machine) | `127.0.0.1` | `127.0.0.1` |
| MySQL 8.0 Port | `3306` (internal) | `33080` |
| Root Password | `root` | `magebox` |

### CLI Commands

| Task | DDEV | MageBox |
|------|------|---------|
| Start project | `ddev start` | `magebox start` |
| Stop project | `ddev stop` | `magebox stop` |
| SSH into web | `ddev ssh` | N/A (native) |
| Run Magento CLI | `ddev magento ...` | `bin/magento ...` |
| Import DB | `ddev import-db` | `magebox db import` |
| Export DB | `ddev export-db` | `magebox db export` |
| MySQL shell | `ddev mysql` | `magebox db shell` |

### No SSH Needed

With DDEV, you often need `ddev ssh` to run commands. With MageBox, everything runs natively:

```bash
# DDEV
ddev ssh
cd /var/www/html
bin/magento cache:flush

# MageBox - just run directly
bin/magento cache:flush
```

### Composer

```bash
# DDEV
ddev composer install

# MageBox - uses the magebox composer wrapper
composer install  # Automatically uses correct PHP version
```

## Mutagen Issues? No More!

One of DDEV's pain points on macOS is Mutagen file synchronization. Common issues:

- Sync conflicts
- Slow initial sync
- Files not appearing
- Memory usage

**MageBox eliminates all of this** by using native filesystem access.

## Common Migration Issues

### SSL Certificates

DDEV and MageBox use different certificate authorities:

```bash
# Trust MageBox certificates
mkcert -install

# Restart browser to pick up new CA
```

### Redis Connection

DDEV uses container networking. MageBox uses localhost:

```php
// DDEV env.php
'host' => 'redis'

// MageBox env.php (auto-generated)
'host' => '127.0.0.1'
```

### Elasticsearch/OpenSearch

```php
// DDEV
'hostname' => 'elasticsearch'

// MageBox
'hostname' => '127.0.0.1'
```

## Removing DDEV

Once MageBox is working:

```bash
# Remove DDEV containers for this project
ddev delete --omit-snapshot

# Or keep the database snapshot
ddev delete

# Optionally uninstall DDEV globally
brew uninstall ddev  # macOS
```

## Performance Comparison

Real-world Magento 2 homepage load times:

| Environment | First Load | Cached Load |
|-------------|------------|-------------|
| DDEV (macOS, Mutagen) | 3.2s | 1.8s |
| DDEV (Linux) | 2.1s | 1.2s |
| **MageBox (macOS)** | **1.1s** | **0.6s** |
| **MageBox (Linux)** | **0.9s** | **0.5s** |

## Need Help?

- [MageBox Documentation](/)
- [GitHub Issues](https://github.com/qoliber/magebox/issues)
- [Troubleshooting Guide](/guide/troubleshooting)

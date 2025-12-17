# Migrating from DDEV

This guide helps you migrate your Magento projects from [DDEV](https://ddev.com/) to MageBox.

## Step 1: Export Your Database

```bash
# Start DDEV if not running
ddev start

# Export database
ddev export-db --file=database-backup.sql.gz
```

## Step 2: Note Your Configuration

Check your `.ddev/config.yaml` for `php_version`, `database`, and `additional_hostnames`.

## Step 3: Stop DDEV

```bash
ddev stop
```

## Step 4: Install MageBox

```bash
# macOS
brew install qoliber/magebox/magebox
magebox bootstrap

# Linux
curl -fsSL https://get.magebox.dev | bash
magebox bootstrap
```

## Step 5: Create MageBox Configuration

Create `.magebox.yaml` in your project root:

```yaml
name: your-project-name
php: "8.2"

domains:
  - host: your-project.test
    root: pub

services:
  mysql: "8.0"
  redis: true
  opensearch: "2.19"
  mailpit: true
```

## Step 6: Start and Import

```bash
magebox start

# Decompress if needed
gunzip database-backup.sql.gz

magebox db import database-backup.sql
```

## Step 7: Verify

```bash
bin/magento setup:db:status
bin/magento cache:flush
```

## Configuration Mapping

| DDEV | MageBox |
|------|---------|
| `name` | `name` |
| `php_version` | `php` |
| `additional_hostnames` | `domains[].host` |
| `database.type: mysql` | `services.mysql` |
| `database.type: mariadb` | `services.mariadb` |

## Command Mapping

| Task | DDEV | MageBox |
|------|------|---------|
| Start | `ddev start` | `magebox start` |
| Stop | `ddev stop` | `magebox stop` |
| Magento CLI | `ddev magento ...` | `bin/magento ...` |
| Import DB | `ddev import-db` | `magebox db import` |
| Export DB | `ddev export-db` | `magebox db export` |
| MySQL shell | `ddev mysql` | `magebox db shell` |
| Composer | `ddev composer` | `composer` |

## Database Differences

| | DDEV | MageBox |
|--|------|---------|
| Host (in Magento) | `db` | `127.0.0.1` |
| MySQL 8.0 Port | 3306 | 33080 |
| Root Password | `root` | `magebox` |

## Removing DDEV

Once verified:

```bash
ddev delete --omit-snapshot
```

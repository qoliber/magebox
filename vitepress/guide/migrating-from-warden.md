# Migrating from Warden

This guide helps you migrate your Magento projects from [Warden](https://docs.warden.dev/) to MageBox.

## Step 1: Export Your Database

```bash
# Start Warden if not running
warden env up

# Export database
warden db dump > database-backup.sql
```

## Step 2: Note Your Configuration

Check your `.warden-env.yml` for PHP version, database type, and services.

## Step 3: Stop Warden

```bash
warden env stop
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
magebox db import database-backup.sql
```

## Step 7: Verify

```bash
bin/magento setup:db:status
bin/magento cache:flush
```

## Configuration Mapping

| Warden | MageBox |
|--------|---------|
| `WARDEN_ENV_NAME` | `name` |
| `TRAEFIK_DOMAIN` | `domains[].host` |
| `PHP_VERSION` | `php` |
| `MYSQL_VERSION` | `services.mysql` |
| `MARIADB_VERSION` | `services.mariadb` |
| `REDIS_VERSION` | `services.redis: true` |
| `ELASTICSEARCH_VERSION` | `services.opensearch` |

## Port Differences

| Service | Warden | MageBox |
|---------|--------|---------|
| MySQL 8.0 | 3306 | 33080 |
| Redis | 6379 | 6379 |
| OpenSearch | 9200 | 9200 |

## Removing Warden

Once verified:

```bash
warden env down -v
```

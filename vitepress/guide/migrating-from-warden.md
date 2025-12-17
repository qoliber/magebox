# Migrating from Warden

This guide helps you migrate your Magento projects from [Warden](https://docs.warden.dev/) to MageBox.

## Architecture Overview

Warden runs all services in Docker containers. MageBox takes a hybrid approach:

```
MageBox Architecture
┌─────────────────────────────────────────┐
│           Native Services               │
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
│  │ MySQL │ │ Redis │ │OpenSearch│      │
│  └───────┘ └───────┘ └──────────┘      │
└─────────────────────────────────────────┘
```

## Migration Steps

### Step 1: Export Your Database

In your Warden project:

```bash
# Start Warden environment if not running
warden env up

# Export database
warden db dump > database-backup.sql
```

### Step 2: Note Your Configuration

Check your `.env` file for:
- Database name
- Redis configuration
- Elasticsearch/OpenSearch version
- Varnish settings (if used)

```bash
# View current Warden config
cat .warden-env.yml
```

### Step 3: Stop Warden

```bash
warden env stop
```

### Step 4: Install MageBox

If you haven't already:

```bash
# macOS
brew tap qoliber/magebox
brew install magebox
magebox bootstrap

# Linux (Fedora)
curl -fsSL https://raw.githubusercontent.com/qoliber/magebox/main/install.sh | bash
magebox bootstrap
```

### Step 5: Create MageBox Configuration

Create `.magebox.yaml` in your project root:

```yaml
name: your-project-name
php: "8.2"  # Match your Warden PHP version

domains:
  - host: your-project.test
    root: pub

services:
  mysql: "8.0"      # Or mariadb: "10.6"
  redis: true
  opensearch: "2.19.4"
  mailpit: true
  # varnish: true   # Uncomment if using Varnish
```

### Step 6: Start MageBox

```bash
magebox start
```

### Step 7: Import Your Database

```bash
magebox db import database-backup.sql
```

### Step 8: Update Magento Configuration

MageBox automatically generates `app/etc/env.php`, but verify the settings:

```bash
# Check database connection
bin/magento setup:db:status

# Clear cache
bin/magento cache:flush

# Reindex if needed
bin/magento indexer:reindex
```

## Configuration Mapping

### Warden to MageBox

| Warden (`.warden-env.yml`) | MageBox (`.magebox.yaml`) |
|---------------------------|---------------------------|
| `WARDEN_ENV_NAME` | `name` |
| `TRAEFIK_DOMAIN` | `domains[].host` |
| `TRAEFIK_SUBDOMAIN` | `domains[].host` |
| `PHP_VERSION` | `php` |
| `MYSQL_VERSION` | `services.mysql` |
| `MARIADB_VERSION` | `services.mariadb` |
| `REDIS_VERSION` | `services.redis: true` |
| `ELASTICSEARCH_VERSION` | `services.opensearch` |
| `RABBITMQ_VERSION` | `services.rabbitmq: true` |

### Environment Variables

Warden uses `.env` for Magento config. MageBox can use `env:` section:

```yaml
# .magebox.yaml
env:
  MAGE_MODE: developer
  MAGE_RUN_TYPE: store
```

## Common Differences

### Database Ports

MageBox uses different ports than Warden:

| Service | Warden Port | MageBox Port |
|---------|-------------|--------------|
| MySQL 8.0 | 3306 | 33080 |
| Redis | 6379 | 6379 |
| OpenSearch | 9200 | 9200 |

### SSL Certificates

MageBox uses `mkcert` for SSL. If you see certificate warnings:

```bash
# Regenerate certificates
mkcert -install
magebox stop
magebox start
```

### File Permissions

MageBox runs as your user:

```bash
# Fix permissions if needed
chmod -R 755 var/ generated/ pub/static/
```

## Removing Warden

Once you've verified MageBox is working:

```bash
# Remove Warden containers
warden env down -v

# Optionally remove Warden
brew uninstall warden  # macOS
```

## Need Help?

- [MageBox Documentation](/)
- [GitHub Issues](https://github.com/qoliber/magebox/issues)
- [Troubleshooting Guide](/guide/troubleshooting)

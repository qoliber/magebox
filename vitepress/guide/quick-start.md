# Quick Start

Get a Magento project running with MageBox in minutes.

## Prerequisites

Before starting, ensure you have:

1. MageBox installed (`magebox --version`)
2. Completed bootstrap (`magebox bootstrap`)
3. PHP wrapper in PATH (`which php` shows `~/.magebox/bin/php`)

## Fastest Way: Quick Install

For beginners or quick testing, use the `--quick` flag:

```bash
# Create new store with sensible defaults (no questions asked!)
magebox new mystore --quick

# Start the project
cd mystore && magebox start
```

Then run the Magento installer:

```bash
php bin/magento setup:install \
    --base-url=https://mystore.test \
    --backend-frontname=admin \
    --db-host=127.0.0.1:33080 \
    --db-name=mystore \
    --db-user=root \
    --db-password=magebox \
    --search-engine=opensearch \
    --opensearch-host=127.0.0.1 \
    --opensearch-port=9200 \
    --opensearch-index-prefix=magento2 \
    --session-save=redis \
    --session-save-redis-host=127.0.0.1 \
    --session-save-redis-port=6379 \
    --session-save-redis-db=2 \
    --cache-backend=redis \
    --cache-backend-redis-server=127.0.0.1 \
    --cache-backend-redis-port=6379 \
    --cache-backend-redis-db=0 \
    --page-cache=redis \
    --page-cache-redis-server=127.0.0.1 \
    --page-cache-redis-port=6379 \
    --page-cache-redis-db=1 \
    --amqp-host=127.0.0.1 \
    --amqp-port=5672 \
    --amqp-user=guest \
    --amqp-password=guest
```

**That's it!** Open https://mystore.test in your browser.

::: tip What does --quick install?
The `--quick` flag installs:
- **MageOS** (no Adobe authentication required)
- **PHP 8.3**
- **MySQL 8.0**
- **Redis** (cache + sessions)
- **OpenSearch 2.19** (with ICU and Phonetic plugins)
- **RabbitMQ**
- **Mailpit**
- **Sample data** included
:::

## Interactive Wizard

For full control over your setup:

```bash
magebox new mystore
```

The wizard guides you through:
1. **Distribution** - Magento Open Source or MageOS
2. **Version** - 2.4.7-p3, 2.4.6-p7, etc.
3. **PHP Version** - Shows compatible versions only
4. **Composer Auth** - Marketplace keys (Magento) or skip (MageOS)
5. **Database** - MySQL 8.0/8.4 or MariaDB 10.6/11.4
6. **Search Engine** - OpenSearch, Elasticsearch, or none
7. **Services** - Redis, RabbitMQ, Mailpit
8. **Sample Data** - Optional demo products
9. **Project Details** - Name and domain

## Existing Project

If you have an existing Magento project:

```bash
# Navigate to your project
cd /path/to/magento

# Initialize MageBox
magebox init

# Start the environment
magebox start
```

Your site is now available at `https://yourproject.test`

## Configuration

The `magebox init` or `magebox new` commands create a `.magebox.yaml` file:

```yaml
name: mystore

domains:
  - host: mystore.test
    root: pub
    ssl: true

php: "8.3"

services:
  mysql: "8.0"
  redis: true
  opensearch:
    version: "2.19"
    memory: "1g"
  rabbitmq: true
  mailpit: true
```

Edit this file to customize your environment.

## Common Commands

```bash
# Start project
magebox start

# Stop project
magebox stop

# Check status
magebox status

# Run Magento CLI (uses correct PHP automatically)
php bin/magento cache:flush

# Open shell with correct PHP
magebox shell

# View logs
magebox logs

# Import database
magebox db import dump.sql

# Export database
magebox db export backup.sql
```

## Accessing Services

After starting, your services are available at:

| Service | URL |
|---------|-----|
| Magento Store | https://mystore.test |
| Mailpit UI | http://localhost:8025 |
| RabbitMQ UI | http://localhost:15672 |

## Database Connection

The PHP wrapper ensures the correct PHP version is used automatically. Database connection details:

| Property | Value |
|----------|-------|
| Host | 127.0.0.1 |
| Port | 33080 (MySQL 8.0) |
| Database | mystore |
| Username | root |
| Password | magebox |

Use the database shell:

```bash
magebox db shell
```

## Multiple Projects

You can run multiple projects simultaneously:

```bash
# Project 1
cd /path/to/store1
magebox start

# Project 2 (different terminal)
cd /path/to/store2
magebox start

# List all projects
magebox list
```

Each project uses its own PHP version and services.

## Troubleshooting

### Site not loading?

1. Check if services are running:
   ```bash
   magebox status
   ```

2. Verify DNS resolution:
   ```bash
   ping mystore.test
   ```

3. Check Nginx configuration:
   ```bash
   nginx -t
   ```

### Database connection failed?

1. Verify Docker containers are running:
   ```bash
   docker ps
   ```

2. Check the correct port in your config:
   ```bash
   magebox status
   ```

### PHP errors?

1. Check PHP-FPM logs:
   ```bash
   magebox logs
   ```

2. Verify PHP version:
   ```bash
   php -v
   ```

## Next Steps

- [Project Configuration](/guide/project-config) - Detailed config options
- [PHP INI Settings](/guide/php-ini) - Customize PHP settings per project
- [Services](/guide/services-overview) - Configure MySQL, Redis, etc.
- [Custom Commands](/guide/custom-commands) - Define project shortcuts
- [PHP Version Wrapper](/guide/php-wrapper) - How automatic PHP switching works

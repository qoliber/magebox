# Migrating from Laravel Herd

This guide helps you migrate from [Laravel Herd](https://herd.laravel.com/) to MageBox.

## Step 1: Export Your Database

```bash
# Using MySQL CLI
mysqldump -u root your_database > database-backup.sql

# Or through Herd's database UI
```

## Step 2: Stop Herd

1. Open Herd application
2. Stop all services from the menu

::: tip
You can keep Herd installed and switch between tools as needed.
:::

## Step 3: Clean Up Shell Config

Edit `~/.zshrc` and remove or comment out Herd-related lines:

```bash
# Remove these lines:
# export HERD_PHP_83_INI_SCAN_DIR="..."
# export PATH="/Users/.../.config/herd-lite/bin:$PATH"
```

Then reload:

```bash
source ~/.zshrc
```

## Step 4: Install MageBox

```bash
brew install qoliber/magebox/magebox
magebox bootstrap
```

## Step 5: Create MageBox Configuration

Create `.magebox.yaml`:

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

## Key Differences

| | Herd | MageBox |
|--|------|---------|
| PHP version | Global | Per-project |
| MySQL | Herd DBngin | Docker container |
| Redis | Manual | Docker container |
| OpenSearch | Manual | Docker container |

## Database Connection

MageBox uses Docker for MySQL. Update connection or let MageBox regenerate:

```bash
magebox stop
magebox start
```

## Removing Herd

Once verified, you can uninstall Herd from Applications.

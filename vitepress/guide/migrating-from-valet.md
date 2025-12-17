# Migrating from Valet / Valet+

This guide helps you migrate from [Laravel Valet](https://laravel.com/docs/valet) or [Valet+](https://github.com/weprovide/valet-plus) to MageBox.

## Step 1: Export Your Database

```bash
# Using MySQL CLI
mysqldump -u root your_database > database-backup.sql

# Or with Valet+ (if available)
valet db export your_database
```

## Step 2: Stop Valet

```bash
valet stop
```

## Step 3: Unlink Project

```bash
cd /path/to/magento
valet unlink
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

| | Valet/Valet+ | MageBox |
|--|--------------|---------|
| MySQL | Manual install | Docker container |
| Redis | Manual install | Docker container |
| OpenSearch | Manual install | Docker container |
| PHP version | Global | Per-project |
| Services | System-wide | Per-project |

## Database Connection

Update your `app/etc/env.php`:

```php
'db' => [
    'connection' => [
        'default' => [
            'host' => '127.0.0.1:33080',
            'dbname' => 'your_database',
            'username' => 'root',
            'password' => 'magebox',
        ]
    ]
]
```

Or let MageBox regenerate it:

```bash
magebox stop
magebox start
```

## Removing Valet

Once verified:

```bash
valet uninstall
```

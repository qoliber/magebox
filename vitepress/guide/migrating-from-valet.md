# Migrating from Valet / Valet+

This guide helps you migrate from [Laravel Valet](https://laravel.com/docs/valet) or [Valet+](https://github.com/weprovide/valet-plus) to MageBox.

## Step 1: Move databases from Brew MySQL (optional)

If you still have databases on Homebrew MySQL from the Valet era, MageBox ships a migration helper:

```bash
bash ./scripts/valet/setup.sh
```

This installs `valet-to-magebox` to `~/bin/`, saves Brew and MageBox MySQL credentials, and can update `app/etc/env.php`, Laravel `.env`, and WordPress `wp-config.php` for all projects in your Valet parked paths.

To patch credentials, secure URLs, and add `.magebox.yaml` where missing:

```bash
valet-to-magebox --update-projects --dry-run
valet-to-magebox --update-projects
valet-to-magebox --start-projects
```

`--start-projects` runs `magebox start` in each parked project that has `.magebox.yaml`. You need this for HTTPS: without it, nginx has no per-site SSL vhost and Chrome may show `ERR_CERT_COMMON_NAME_INVALID` for every old Valet site except those you already started manually.

List and move databases:

```bash
valet-to-magebox --list
valet-to-magebox --move=your_database
```

See [scripts/valet/README.md](https://github.com/qoliber/magebox/blob/main/scripts/valet/README.md) for full usage.

Alternatively, export manually:

```bash
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
brew tap qoliber/magebox
brew install magebox
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
  redis: true          # or valkey: true
  opensearch:
    version: "2.19"
    memory: "2g"
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
| Redis/Valkey | Manual install | Docker container |
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

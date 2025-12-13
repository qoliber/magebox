# Migrating from Laravel Herd

This guide helps you switch from Laravel Herd to MageBox for your Magento development environment.

## Why Switch to MageBox?

- **Magento-optimized**: Built specifically for Magento 2 development
- **Multi-project support**: Run multiple Magento projects with different PHP versions simultaneously
- **Native performance**: PHP runs natively on your machine, not in containers
- **Flexible architecture**: Uses native Nginx and PHP-FPM with Docker for databases, search, and caching services
- **Automatic PHP version switching**: Smart PHP wrapper automatically uses the correct version per project

## Prerequisites

Before starting, make sure you have:

- Homebrew installed (macOS)
- Docker Desktop installed and running
- At least one PHP version installed via Homebrew

## Step 1: Stop Herd Services

Stop all Herd services to prevent conflicts:

1. Open Herd application
2. Stop all services from the menu

::: tip
You don't need to uninstall Herd completely. You can keep it installed and switch between Herd and MageBox as needed.
:::

## Step 2: Clean Up Herd Configuration

Herd adds several lines to your shell configuration. Edit `~/.zshrc` (or `~/.bashrc`) and **remove or comment out** all Herd-related lines:

```bash
# Open the file
nano ~/.zshrc
```

Look for and **remove** these sections:

```bash
# Remove these Herd PATH entries:
export PATH="/Users/YOUR_USERNAME/.config/herd-lite/bin:$PATH"
export PATH="/Users/YOUR_USERNAME/Library/Application Support/Herd/bin/":$PATH

# Remove this PHP configuration:
export PHP_INI_SCAN_DIR="/Users/YOUR_USERNAME/.config/herd-lite/bin:$PHP_INI_SCAN_DIR"

# Remove all HERD_PHP_* environment variables:
export HERD_PHP_84_INI_SCAN_DIR="..."
export HERD_PHP_83_INI_SCAN_DIR="..."
# ... and any other PHP versions

# Remove Herd shell script sourcing:
[[ -f "/Applications/Herd.app/Contents/Resources/config/shell/zshrc.zsh" ]] && builtin source "..."
```

After removing these lines, reload your shell:

```bash
source ~/.zshrc
```

## Step 3: Install PHP via Homebrew

Herd uses its own PHP builds. For MageBox, you need Homebrew PHP:

```bash
# Install PHP versions you need
brew install php@8.1 php@8.2 php@8.3

# Verify installation
brew list | grep php
```

## Step 4: Install Other Dependencies

```bash
# Install Nginx
brew install nginx

# Install mkcert for SSL certificates
brew install mkcert nss

# Ensure Docker Desktop is installed and running
```

## Step 5: Install MageBox

::: code-group

```bash [macOS Apple Silicon]
sudo curl -L https://github.com/qoliber/magebox/releases/latest/download/magebox-darwin-arm64 \
  -o /usr/local/bin/magebox && sudo chmod +x /usr/local/bin/magebox
```

```bash [macOS Intel]
sudo curl -L https://github.com/qoliber/magebox/releases/latest/download/magebox-darwin-amd64 \
  -o /usr/local/bin/magebox && sudo chmod +x /usr/local/bin/magebox
```

:::

Verify installation:

```bash
magebox --version
```

## Step 6: Bootstrap MageBox

Run the one-time setup:

```bash
magebox bootstrap
```

This command will:
1. Check dependencies
2. Create global configuration
3. Set up SSL certificates with mkcert
4. Configure port forwarding (80→8080, 443→8443)
5. Configure Nginx
6. Start Docker services (MySQL, Redis, Mailpit)
7. Set up DNS resolution
8. Install PHP wrapper script

## Step 7: Add MageBox PHP Wrapper to PATH

```bash
# Add to ~/.zshrc
echo 'export PATH="$HOME/.magebox/bin:$PATH"' >> ~/.zshrc

# Reload shell
source ~/.zshrc
```

::: warning Important
Make sure this line is **before** any other PHP-related PATH modifications.
:::

Verify the wrapper is working:

```bash
which php
# Should output: /Users/YOUR_USERNAME/.magebox/bin/php

php -v
# Should show PHP version without any Herd warnings
```

## Step 8: Migrate Your Projects

### Initialize Existing Project

```bash
cd /path/to/your/magento/project

# Initialize MageBox configuration
magebox init

# Start the project
magebox start
```

### Or Create New Project

```bash
# Quick start with MageOS (no Adobe credentials required)
magebox new mystore --quick

cd mystore
magebox start
```

## Step 9: Update Database Connection

Update your Magento `app/etc/env.php`:

```php
'db' => [
    'table_prefix' => '',
    'connection' => [
        'default' => [
            'host' => '127.0.0.1:33080',  // Changed from Herd's MySQL
            'dbname' => 'your_database',
            'username' => 'root',
            'password' => 'magebox',      // Changed from Herd's password
            'model' => 'mysql4',
            'engine' => 'innodb',
            'active' => '1',
        ]
    ]
],
```

## Step 10: Import Your Database (Optional)

If you want to import your existing database from Herd:

### Export from Herd's MySQL

```bash
# Find Herd's MySQL and export
mysqldump -uroot -p your_database > backup.sql
```

### Import to MageBox

```bash
magebox db import backup.sql
```

## Verification Checklist

After migration, verify everything works:

- [ ] `which php` shows `/Users/YOUR_USERNAME/.magebox/bin/php`
- [ ] `php -v` works without Herd warnings
- [ ] `magebox status` shows your project running
- [ ] Your Magento site loads at https://yoursite.test
- [ ] `php bin/magento cache:flush` works
- [ ] `composer install` uses correct PHP version automatically

## Common Issues

### PHP Still Shows Herd Warnings

**Problem**: `php -v` shows warnings about loading Herd extensions.

**Solution**:
```bash
# Make sure ALL Herd config is removed from ~/.zshrc
grep -i herd ~/.zshrc
# Should return nothing

# Open a completely new terminal window
```

### `which php` Points to Herd

**Problem**: `which php` returns Herd's PHP path.

**Solution**:
```bash
# Verify MageBox bin is in PATH
echo $PATH | grep magebox

# If not found, add it
echo 'export PATH="$HOME/.magebox/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

### Port Conflicts

**Problem**: Nginx fails to start because port 8080 or 8443 is in use.

**Solution**:
```bash
# Check what's using the ports
lsof -i :8080
lsof -i :8443

# Stop Herd services if still running
```

### SSL Certificate Errors

**Solution**:
```bash
magebox ssl trust
magebox ssl generate
magebox restart
```

## Switching Back to Herd

If you need to switch back temporarily:

1. Stop MageBox services:
   ```bash
   magebox stop
   magebox global stop
   ```

2. Re-add Herd configuration to `~/.zshrc`

3. Restart terminal

4. Start Herd services

You can keep both installed and switch by controlling which one is in your PATH first.

## Benefits After Migration

Once migrated, you'll enjoy:

- **Automatic PHP version switching** - The PHP wrapper detects your project's config
- **Multi-project support** - Run multiple Magento projects with different PHP versions
- **Native performance** - No Docker overhead for PHP and Nginx
- **Magento-optimized** - Nginx, Varnish, and PHP-FPM tuned for Magento
- **Flexible services** - Easy to switch MySQL versions, enable OpenSearch, RabbitMQ, etc.

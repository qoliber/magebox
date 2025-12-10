# Migrating from Laravel Herd to MageBox

This guide helps you switch from Laravel Herd to MageBox for your Magento development environment.

## Why Switch to MageBox?

- **Magento-optimized**: Built specifically for Magento 2 development
- **Multi-project support**: Run multiple Magento projects with different PHP versions simultaneously
- **Native performance**: PHP runs natively on your machine, not in containers
- **Flexible architecture**: Uses native Nginx, PHP-FPM, and Varnish with Docker only for databases
- **Automatic PHP version switching**: Smart PHP wrapper automatically uses the correct version per project

## Prerequisites

Before starting, make sure you have:

- Homebrew installed
- Docker Desktop installed and running
- At least one PHP version installed via Homebrew (see installation instructions below)

## Step 1: Stop Herd Services

First, stop all Herd services to prevent conflicts:

```bash
# Stop Herd services
# You can do this from the Herd application menu, or:
# Open Herd app → Stop all services
```

> **Note**: You don't need to uninstall Herd completely. You can keep it installed and switch between Herd and MageBox as needed.

## Step 2: Clean Up Herd Configuration from Shell

Herd adds several lines to your shell configuration file (`~/.zshrc` for zsh or `~/.bashrc` for bash). You need to remove these to prevent conflicts.

### For Zsh users (macOS default):

Edit your `~/.zshrc` file and **remove or comment out** all Herd-related lines:

```bash
# Open the file in your editor
nano ~/.zshrc
# or
code ~/.zshrc
```

Look for and **remove** these sections:

```bash
# Remove these Herd PATH entries:
export PATH="/Users/YOUR_USERNAME/.config/herd-lite/bin:$PATH"
export PATH="/Users/YOUR_USERNAME/Library/Application Support/Herd/bin/":$PATH

# Remove this PHP configuration:
export PHP_INI_SCAN_DIR="/Users/YOUR_USERNAME/.config/herd-lite/bin:$PHP_INI_SCAN_DIR"

# Remove all HERD_PHP_* environment variables:
export HERD_PHP_84_INI_SCAN_DIR="/Users/YOUR_USERNAME/Library/Application Support/Herd/config/php/84/"
export HERD_PHP_83_INI_SCAN_DIR="/Users/YOUR_USERNAME/Library/Application Support/Herd/config/php/83/"
export HERD_PHP_82_INI_SCAN_DIR="/Users/YOUR_USERNAME/Library/Application Support/Herd/config/php/82/"
export HERD_PHP_81_INI_SCAN_DIR="/Users/YOUR_USERNAME/Library/Application Support/Herd/config/php/81/"
# ... and any other PHP versions

# Remove Herd NVM configuration (if you don't use it):
export NVM_DIR="/Users/YOUR_USERNAME/Library/Application Support/Herd/config/nvm"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"

# Remove Herd shell script sourcing:
[[ -f "/Applications/Herd.app/Contents/Resources/config/shell/zshrc.zsh" ]] && builtin source "/Applications/Herd.app/Contents/Resources/config/shell/zshrc.zsh"
```

After removing these lines, save the file and reload your shell:

```bash
source ~/.zshrc
```

Or open a new terminal window.

## Step 3: Install PHP via Homebrew

Herd uses its own PHP builds. For MageBox, you need Homebrew PHP versions:

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

# Ensure Docker Desktop is installed
# Download from: https://www.docker.com/products/docker-desktop
```

## Step 5: Install MageBox

### macOS (Apple Silicon):

```bash
sudo curl -L https://github.com/qoliber/magebox/releases/latest/download/magebox-darwin-arm64 -o /usr/local/bin/magebox && sudo chmod +x /usr/local/bin/magebox
```

### macOS (Intel):

```bash
sudo curl -L https://github.com/qoliber/magebox/releases/latest/download/magebox-darwin-amd64 -o /usr/local/bin/magebox && sudo chmod +x /usr/local/bin/magebox
```

### Verify installation:

```bash
magebox --version
```

## Step 6: Bootstrap MageBox

This one-time setup will configure your environment:

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
8. **Install PHP wrapper script**

## Step 7: Add MageBox PHP Wrapper to PATH

After bootstrap completes, add MageBox's PHP wrapper to your shell configuration:

```bash
# Add to ~/.zshrc (or ~/.bashrc for bash)
echo 'export PATH="$HOME/.magebox/bin:$PATH"' >> ~/.zshrc

# Reload shell
source ~/.zshrc
```

**Important**: Make sure this line is **before** any other PHP-related PATH modifications.

Verify the PHP wrapper is working:

```bash
which php
# Should output: /Users/YOUR_USERNAME/.magebox/bin/php

php -v
# Should show your PHP version without any warnings
```

## Step 8: Migrate Your Existing Projects

### Option A: Initialize Existing Magento Project

If you already have a Magento project from Herd:

```bash
cd /path/to/your/magento/project

# Initialize MageBox configuration
magebox init

# This will ask you:
# - PHP version (8.1, 8.2, or 8.3)
# - Domain name (e.g., mystore.test)
# - MySQL version
# - Redis, OpenSearch, etc.

# Start the project
magebox start
```

### Option B: Create New Project

```bash
# Quick start with MageOS (no Adobe credentials required)
magebox new mystore --quick

cd mystore
magebox start
```

## Step 9: Update Database Connection

Update your Magento `app/etc/env.php` to use MageBox's MySQL:

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
            'initStatements' => 'SET NAMES utf8;',
            'active' => '1',
            'driver_options' => [
                1014 => false
            ]
        ]
    ]
],
```

## Step 10: Import Your Database (Optional)

If you want to import your existing database from Herd:

### Export from Herd's MySQL:

```bash
# Find Herd's MySQL socket or port
# Usually at: /Users/YOUR_USERNAME/Library/Application Support/Herd/config/mysql/mysql.sock

# Export database
mysqldump -uroot -p your_database > backup.sql
```

### Import to MageBox:

```bash
# Import to MageBox MySQL
magebox db import backup.sql
```

## Verification Checklist

After migration, verify everything works:

- [ ] `which php` shows `/Users/YOUR_USERNAME/.magebox/bin/php`
- [ ] `php -v` works without Herd warnings
- [ ] `magebox status` shows your project running
- [ ] Your Magento site loads at https://yoursite.test
- [ ] `magebox cli cache:flush` works
- [ ] `composer install` uses correct PHP version automatically

## Common Issues and Solutions

### Issue 1: PHP still shows Herd warnings

**Problem**: `php -v` shows warnings about loading Herd extensions.

**Solution**:
```bash
# 1. Make sure you removed ALL Herd configuration from ~/.zshrc
grep -i herd ~/.zshrc
# Should return nothing

# 2. Restart your terminal completely (don't just source ~/.zshrc)
# Close terminal and open a new one

# 3. Verify PATH order
echo $PATH | tr ':' '\n' | head -5
# First line should be: /Users/YOUR_USERNAME/.magebox/bin
```

### Issue 2: `which php` still points to Herd

**Problem**: `which php` returns `/Users/YOUR_USERNAME/Library/Application Support/Herd/bin/php`

**Solution**:
```bash
# Check if MageBox bin is in PATH
echo $PATH | grep magebox

# If not found, add it again
echo 'export PATH="$HOME/.magebox/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc

# Open a completely new terminal window
```

### Issue 3: Port conflicts

**Problem**: Nginx fails to start because port 8080 or 8443 is in use.

**Solution**:
```bash
# Check what's using the ports
lsof -i :8080
lsof -i :8443

# Stop Herd services if still running
# Or kill the process using the port
```

### Issue 4: SSL certificate errors

**Problem**: Browser shows SSL certificate errors.

**Solution**:
```bash
# Re-trust the MageBox CA
magebox ssl trust

# Regenerate certificates
magebox ssl generate

# Restart your project
magebox restart
```

### Issue 5: Database connection errors

**Problem**: Magento can't connect to the database.

**Solution**:
```bash
# Check if MySQL is running
magebox global status

# Start global services if stopped
magebox global start

# Verify database connection
magebox db shell
```

## Switching Back to Herd (If Needed)

If you need to switch back to Herd temporarily:

1. Stop MageBox services:
   ```bash
   magebox stop
   magebox global stop
   ```

2. Re-add Herd configuration to your `~/.zshrc` (you can keep the backup)

3. Restart your terminal

4. Start Herd services

You can keep both MageBox and Herd installed and switch between them as needed by controlling which one is in your PATH first.

## Benefits After Migration

Once you've successfully migrated, you'll enjoy:

✅ **Automatic PHP version switching** - The PHP wrapper detects your project's `.magebox.yaml` and uses the correct version
✅ **Multi-project support** - Run multiple Magento projects with different PHP versions simultaneously
✅ **Native performance** - No Docker overhead for PHP and Nginx
✅ **Magento-optimized** - Nginx, Varnish, and PHP-FPM configurations tuned for Magento
✅ **Flexible services** - Easy to switch between MySQL versions, enable OpenSearch, RabbitMQ, etc.
✅ **Simple commands** - `magebox cli` runs bin/magento commands with correct PHP version automatically

## Getting Help

If you encounter issues during migration:

1. Check the main [README.md](../README.md) for detailed documentation
2. Open an issue at: https://github.com/qoliber/magebox/issues
3. Make sure you followed each step carefully, especially removing all Herd configuration

## Additional Resources

- [MageBox Documentation](../README.md)
- [Configuration Reference](../README.md#configuration)
- [Troubleshooting Guide](../README.md#troubleshooting)
- [All Commands](../README.md#all-commands)

---

**Happy Magento development with MageBox!** 🎉

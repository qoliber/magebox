# PHP Version Wrapper

MageBox includes a smart PHP wrapper that automatically uses the correct PHP version based on your project configuration.

## Overview

After running `magebox bootstrap`, a PHP wrapper script is installed at `~/.magebox/bin/php`. This wrapper:

1. Walks up the directory tree looking for `.magebox.yaml` or `.magebox.local.yaml`
2. Extracts the PHP version from the config file
3. Executes the correct PHP binary (e.g., `/opt/homebrew/opt/php@8.3/bin/php`)
4. Falls back to system PHP if no config file is found

## Installation

The PHP wrapper is installed automatically during bootstrap:

```bash
magebox bootstrap
```

Then add MageBox bin to your PATH:

```bash
# Add to ~/.zshrc or ~/.bashrc
export PATH="$HOME/.magebox/bin:$PATH"

# Reload shell
source ~/.zshrc  # or source ~/.bashrc
```

::: warning Important
Make sure the MageBox bin directory is **first** in your PATH, before any other PHP installations.
:::

## Verification

Verify the wrapper is working:

```bash
# Check which php is being used
which php
# Should output: /Users/YOUR_USERNAME/.magebox/bin/php

# Test version detection
cd /path/to/your/project
php -v
# Shows PHP version from your .magebox.yaml
```

## How It Works

```
                    ┌─────────────────────┐
                    │   php command       │
                    └──────────┬──────────┘
                               │
                    ┌──────────▼──────────┐
                    │  ~/.magebox/bin/php │
                    │   (wrapper script)  │
                    └──────────┬──────────┘
                               │
              ┌────────────────┼────────────────┐
              │                │                │
              ▼                ▼                ▼
     ┌────────────────┐ ┌────────────┐ ┌──────────────┐
     │ Search for     │ │ Read PHP   │ │ Execute      │
     │ .magebox.yaml  │ │ version    │ │ correct      │
     │ in parent dirs │ │ from file  │ │ PHP binary   │
     └────────────────┘ └────────────┘ └──────────────┘
```

## Usage Examples

### Automatic Version Switching

```bash
# Project 1 uses PHP 8.2
cd /path/to/project1
cat .magebox.yaml
# php: "8.2"
php -v
# PHP 8.2.x ...

# Project 2 uses PHP 8.3
cd /path/to/project2
cat .magebox.yaml
# php: "8.3"
php -v
# PHP 8.3.x ...
```

### Works in Subdirectories

The wrapper searches up the directory tree:

```bash
cd /path/to/project/app/code/Vendor/Module
php -v
# Still uses PHP version from /path/to/project/.magebox.yaml
```

### Composer Uses Correct PHP

```bash
cd /path/to/project
composer install
# Composer automatically uses the correct PHP version

composer require some/package
# Also uses correct PHP version
```

### Magento Commands

```bash
cd /path/to/project
php bin/magento cache:flush
# Uses correct PHP version

bin/magento setup:upgrade
# Also uses correct PHP version
```

## Fallback Behavior

When no `.magebox.yaml` is found in the directory tree:

1. Wrapper checks for `.magebox.local.yaml`
2. If still not found, uses system default PHP
3. System PHP is determined by standard PATH resolution

## Multiple PHP Versions

The wrapper supports all PHP versions installed on your system:

**macOS (Homebrew):**
```bash
brew install php@8.1 php@8.2 php@8.3 php@8.4
```

**Linux (Ubuntu/Debian):**
```bash
sudo apt install php8.1-fpm php8.2-fpm php8.3-fpm
```

Each project can specify which version to use:

```yaml
# project-a/.magebox.yaml
php: "8.1"

# project-b/.magebox.yaml
php: "8.3"
```

## IDE Integration

### PHPStorm

Configure PHPStorm to use the MageBox PHP wrapper:

1. **Settings → PHP**
2. Click **...** next to CLI Interpreter
3. Add new interpreter: `~/.magebox/bin/php`
4. Or configure per-project PHP version

### VS Code

For PHP Intelephense or similar extensions:

```json
{
  "php.executablePath": "~/.magebox/bin/php"
}
```

## Troubleshooting

### `which php` Shows Wrong Path

```bash
# Check PATH order
echo $PATH | tr ':' '\n' | head -5

# MageBox bin should be first
# If not, check your ~/.zshrc or ~/.bashrc
```

### PHP Version Not Changing

1. Ensure `.magebox.yaml` exists in project root
2. Check file has correct `php:` key
3. Verify PHP version is installed

```bash
# Check config
cat .magebox.yaml | grep php

# Check installed versions
ls /opt/homebrew/opt/ | grep php  # macOS
ls /usr/bin/ | grep php           # Linux
```

### Wrapper Not Found

```bash
# Check wrapper exists
ls -la ~/.magebox/bin/php

# Re-run bootstrap if missing
magebox bootstrap
```

### Wrong PHP Used in Scripts

Some scripts may bypass the wrapper. Use full path:

```bash
~/.magebox/bin/php script.php
```

Or ensure PATH is set in the script's environment.

## Comparison with Manual Switching

| Feature | PHP Wrapper | Manual (`magebox php`) |
|---------|-------------|------------------------|
| Automatic | Yes | No |
| Per-directory | Yes | Project-wide |
| Works with Composer | Yes | Yes |
| Works with IDE | Yes | Requires config |
| Persistent | Yes | Until changed |

## Disabling the Wrapper

If you need to use system PHP directly:

```bash
# Temporarily bypass wrapper
/usr/bin/php -v

# Or remove from PATH (in current session)
export PATH=$(echo $PATH | sed 's|$HOME/.magebox/bin:||')
```

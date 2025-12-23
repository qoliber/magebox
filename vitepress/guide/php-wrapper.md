# CLI Wrappers

MageBox includes smart CLI wrappers that automatically use the correct PHP version based on your project configuration. After bootstrap, three wrappers are installed in `~/.magebox/bin/`:

- **php** - Automatically uses project's PHP version
- **composer** - Runs Composer with project's PHP version
- **blackfire** - Runs Blackfire profiler with project's PHP version

## Overview

When you run `php`, `composer`, or `blackfire` commands, the wrappers:

1. Walk up the directory tree looking for a project directory (containing `.magebox.yaml` or `.magebox.local.yaml`)
2. Check `.magebox.local.yaml` first for PHP version (local overrides have priority)
3. Fall back to `.magebox.yaml` if no local override exists
4. Execute the correct PHP binary for your platform
5. Fall back to system PHP if no config file is found

This means you can switch between projects with different PHP versions without manual intervention.

::: tip Local Overrides Take Priority
If both `.magebox.yaml` and `.magebox.local.yaml` exist, the PHP version from `.magebox.local.yaml` is used. This allows you to test different PHP versions locally without changing the shared project config.

```yaml
# .magebox.yaml (committed to git)
php: "8.2"

# .magebox.local.yaml (not committed, your local override)
php: "8.1"  # This version will be used
```
:::

## Installation

The wrappers are installed automatically during bootstrap:

```bash
magebox bootstrap
```

Bootstrap also adds `~/.magebox/bin` to your PATH automatically. After bootstrap, reload your shell:

```bash
source ~/.zshrc  # or source ~/.bashrc
```

::: warning Important
Make sure the MageBox bin directory is **first** in your PATH, before any other PHP installations.
:::

## Verification

Verify the wrappers are working:

```bash
# Check which php is being used
which php
# Should output: /Users/YOUR_USERNAME/.magebox/bin/php

# Check which composer is being used
which composer
# Should output: /Users/YOUR_USERNAME/.magebox/bin/composer

# Test version detection
cd /path/to/your/project
php -v
# Shows PHP version from your .magebox.yaml
```

## How It Works

```
┌─────────────────────────────────────────────────────────────────┐
│                        Command Execution                         │
└─────────────────────────────────────────────────────────────────┘
                               │
              ┌────────────────┼────────────────┐
              │                │                │
              ▼                ▼                ▼
     ┌─────────────┐   ┌─────────────┐   ┌─────────────┐
     │    php      │   │  composer   │   │  blackfire  │
     │   wrapper   │   │   wrapper   │   │   wrapper   │
     └──────┬──────┘   └──────┬──────┘   └──────┬──────┘
            │                 │                 │
            └────────────────┬┴─────────────────┘
                             │
                  ┌──────────▼──────────┐
                  │  Find project dir   │
                  │  with .magebox.yaml │
                  │  or .magebox.local  │
                  └──────────┬──────────┘
                             │
                  ┌──────────▼──────────┐
                  │  Check local first: │
                  │  .magebox.local.yaml│
                  │  → .magebox.yaml    │
                  └──────────┬──────────┘
                             │
                  ┌──────────▼──────────┐
                  │  Find PHP binary    │
                  │  for platform       │
                  └──────────┬──────────┘
                             │
            ┌────────────────┼────────────────┐
            │                │                │
            ▼                ▼                ▼
    ┌──────────────┐ ┌──────────────┐ ┌──────────────┐
    │   macOS      │ │   Ubuntu/    │ │   Fedora/    │
    │  Homebrew    │ │   Debian     │ │   RHEL       │
    │  Cellar path │ │   Ondrej PPA │ │   Remi       │
    └──────────────┘ └──────────────┘ └──────────────┘
```

## PHP Wrapper

The PHP wrapper at `~/.magebox/bin/php` is the foundation. It:

- Detects project PHP version from `.magebox.yaml`
- Sets `memory_limit=-1` for unlimited CLI memory (important for `setup:di:compile`)
- Finds the correct PHP binary for your platform

### Platform-Specific Binary Detection

| Platform | PHP 8.2 Path |
|----------|--------------|
| macOS (Apple Silicon) | `/opt/homebrew/Cellar/php@8.2/*/bin/php` |
| macOS (Intel) | `/usr/local/Cellar/php@8.2/*/bin/php` |
| Ubuntu/Debian | `/usr/bin/php8.2` |
| Fedora/RHEL (Remi) | `/usr/bin/php82` |

### Usage Examples

```bash
# Automatic version switching
cd /path/to/project1  # php: "8.2" in .magebox.yaml
php -v
# PHP 8.2.x ...

cd /path/to/project2  # php: "8.3" in .magebox.yaml
php -v
# PHP 8.3.x ...

# Works in subdirectories
cd /path/to/project/app/code/Vendor/Module
php -v
# Still uses PHP version from /path/to/project/.magebox.yaml

# Magento commands
php bin/magento cache:flush
php bin/magento setup:di:compile  # Uses unlimited memory
```

## Composer Wrapper

The Composer wrapper at `~/.magebox/bin/composer` leverages the PHP wrapper:

- Finds the real Composer binary (skipping the wrapper itself)
- Executes Composer using the PHP wrapper
- Ensures Composer runs with the correct PHP version

### How It Works

```bash
# When you run:
composer install

# The wrapper executes:
~/.magebox/bin/php /path/to/real/composer install
```

### Usage Examples

```bash
cd /path/to/project
composer install          # Uses project's PHP version
composer require vendor/package
composer update --dry-run

# All Composer commands automatically use correct PHP
composer diagnose
composer validate
```

::: tip
The `magebox composer` command was removed in v0.12.5. Just use `composer` directly - the wrapper handles everything automatically.
:::

## Blackfire Wrapper

The Blackfire wrapper at `~/.magebox/bin/blackfire` handles profiling commands:

- Intercepts `blackfire run php ...` commands
- Replaces `php` argument with the project's PHP binary
- Passes other commands through to real Blackfire CLI

### How It Works

```bash
# When you run:
blackfire run php bin/magento cache:flush

# The wrapper executes:
/usr/bin/blackfire run /opt/homebrew/Cellar/php@8.2/8.2.x/bin/php -d memory_limit=-1 bin/magento cache:flush
```

### Usage Examples

```bash
cd /path/to/project

# Profile Magento commands
blackfire run php bin/magento setup:di:compile
blackfire run php bin/magento indexer:reindex

# Profile custom scripts
blackfire run php script.php

# Non-profiling commands pass through
blackfire version
blackfire config
blackfire agent:start
```

### Blackfire with Ignore Exit Status

```bash
# Profile commands that may exit non-zero
blackfire --ignore-exit-status run php bin/magento some:command
```

## Multiple PHP Versions

The wrappers support all PHP versions installed on your system:

**macOS (Homebrew):**
```bash
brew install php@8.1 php@8.2 php@8.3 php@8.4
```

**Ubuntu/Debian (Ondrej PPA):**
```bash
sudo apt install php8.1-fpm php8.2-fpm php8.3-fpm
```

**Fedora (Remi):**
```bash
sudo dnf install php81-php-fpm php82-php-fpm php83-php-fpm
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

### PhpStorm Terminal Uses System PHP

PhpStorm's built-in terminal may ignore your shell PATH and use its own configured PHP. This happens because:

1. **PhpStorm injects its CLI interpreter into PATH** - Check Settings → Tools → Terminal → look for PHP-related options
2. **GUI apps don't inherit shell PATH** - PhpStorm reads environment at launch, not from shell config

**Solution 1: Disable PhpStorm's PHP PATH injection**

Go to **Settings → Tools → Terminal** and uncheck any option like "Add PHP interpreter to PATH" or similar.

**Solution 2: Create a symlink at `/usr/local/bin`**

```bash
# Create symlink that takes precedence
sudo ln -sf ~/.magebox/bin/php /usr/local/bin/php

# Add /usr/local/bin first in PATH (in ~/.zshenv)
export PATH="/usr/local/bin:$PATH"
```

**Solution 3: Configure PhpStorm to use MageBox wrapper**

1. Go to **Settings → PHP → CLI Interpreter**
2. Add new interpreter pointing to `~/.magebox/bin/php`
3. Set it as project default

**Verify in PhpStorm terminal:**
```bash
which php          # Should NOT be /usr/bin/php
php -v             # Should match project's PHP version
env | grep -i php  # Check if PhpStorm injected PHP paths
```

::: warning Restart Required
After changing PATH in `~/.zshenv`, you must **fully quit and restart PhpStorm** (not just the terminal tab) for changes to take effect.
:::

### PHP Version Not Changing

1. Ensure `.magebox.yaml` exists in project root
2. Check file has correct `php:` key
3. Verify PHP version is installed

```bash
# Check config
cat .magebox.yaml | grep php

# Check installed versions (macOS)
ls /opt/homebrew/opt/ | grep php

# Check installed versions (Linux)
ls /usr/bin/ | grep php
```

### Wrapper Not Found

```bash
# Check wrapper exists
ls -la ~/.magebox/bin/

# Re-run bootstrap if missing
magebox bootstrap
```

### Composer Using Wrong PHP

```bash
# Verify composer wrapper
which composer
# Should show: ~/.magebox/bin/composer

# Check Composer's PHP
composer diagnose | head -5
```

### Blackfire Not Using Project PHP

```bash
# Verify blackfire wrapper
which blackfire
# Should show: ~/.magebox/bin/blackfire

# Test with verbose output
cd /path/to/project
blackfire run php -v
```

## Fallback Behavior

When no `.magebox.yaml` is found in the directory tree:

1. Wrapper checks for `.magebox.local.yaml`
2. If still not found, uses system default PHP
3. System PHP is determined by standard PATH resolution (excluding the wrapper)

## Disabling the Wrappers

If you need to use system PHP/Composer/Blackfire directly:

```bash
# Temporarily bypass wrappers
/usr/bin/php -v
/usr/local/bin/composer --version
/usr/bin/blackfire version

# Or remove from PATH (in current session)
export PATH=$(echo $PATH | sed "s|$HOME/.magebox/bin:||")
```

## Wrapper Script Locations

| Wrapper | Path | Purpose |
|---------|------|---------|
| PHP | `~/.magebox/bin/php` | Automatic PHP version detection |
| Composer | `~/.magebox/bin/composer` | Runs Composer with correct PHP |
| Blackfire | `~/.magebox/bin/blackfire` | Profiles with correct PHP |

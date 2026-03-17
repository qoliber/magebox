# PHP Extensions

MageBox includes all required PHP extensions for Magento out of the box. For additional extensions, use `magebox ext` to install, remove, list, and search extensions across all your PHP versions.

## Pre-installed Extensions

The following extensions are installed automatically during `magebox bootstrap`:

`bcmath`, `cli`, `common`, `curl`, `fpm`, `gd`, `intl`, `mbstring`, `mysql`, `opcache`, `soap`, `sodium`, `xml`, `zip`, `imagick`

## Listing Extensions

```bash
magebox ext list
```

Shows all loaded extensions for the current project's PHP version (via `php -m`).

## Installing Extensions

### System packages

For well-known extensions, MageBox resolves the correct platform-specific package automatically:

```bash
magebox ext install redis
magebox ext install pgsql apcu memcached
```

Under the hood this runs:
- **Ubuntu/Debian:** `sudo apt install -y php8.3-redis`
- **Fedora/RHEL:** `sudo dnf install -y php83-php-pecl-redis`
- **Arch Linux:** `sudo pacman -S --noconfirm php-redis`
- **macOS:** `pecl install redis` (via the version-specific pecl binary)

### PHP version selection

When multiple PHP versions are installed, you'll be prompted:

```
Installed PHP versions:
  0) All installed (default)
  1) PHP 8.1
  2) PHP 8.2
  3) PHP 8.3
  4) PHP 8.4

Select PHP version [0]:
```

Press Enter to install for all versions, or pick a specific one.

### Custom extensions via PIE

For extensions not available as system packages, use [PIE](https://github.com/php/pie) (PHP Installer for Extensions) with the `vendor/package` format:

```bash
magebox ext install noisebynorthwest/php-spx
magebox ext install openswoole/openswoole
```

PIE is the official PECL replacement from the PHP Foundation. It compiles extensions from source using the correct PHP version's build tools.

MageBox handles everything automatically:
1. **Installs PIE** if not present (prompts for confirmation)
2. **Installs PHP dev tools** (`php-dev` / `php-devel`) needed for compilation
3. **Compiles and installs** the extension for the selected PHP version(s)
4. **Restarts PHP-FPM** to load the new extension

Browse available PIE extensions at [packagist.org/extensions](https://packagist.org/extensions).

### Installing PIE manually

You can also pre-install PIE without installing an extension:

```bash
magebox ext pie
```

## Removing Extensions

```bash
# System package
magebox ext remove redis

# PIE extension
magebox ext remove noisebynorthwest/php-spx
```

Prompts for PHP version selection, then removes the extension and restarts PHP-FPM.

## Searching Extensions

```bash
magebox ext search redis
magebox ext search mongo
```

Searches the system package manager for matching extensions and shows whether each is already installed:

```
  php8.4-redis - An alternative Redis client library for PHP [installed]
  php8.4-mongodb - MongoDB driver for PHP [not installed]
```

## Supported Extension Mappings

MageBox knows the correct package names for these extensions across all platforms:

| Extension | Ubuntu/Debian | Fedora/RHEL | Arch | macOS |
|-----------|--------------|-------------|------|-------|
| redis | php{v}-redis | php{vnd}-php-pecl-redis | php-redis | pecl |
| xdebug | php{v}-xdebug | php{vnd}-php-xdebug | xdebug | pecl |
| imagick | php{v}-imagick | php{vnd}-php-pecl-imagick-im7 | php-imagick | pecl |
| memcached | php{v}-memcached | php{vnd}-php-pecl-memcached | php-memcached | pecl |
| apcu | php{v}-apcu | php{vnd}-php-pecl-apcu | php-apcu | pecl |
| mongodb | php{v}-mongodb | php{vnd}-php-pecl-mongodb | pecl | pecl |
| pgsql | php{v}-pgsql | php{vnd}-php-pgsql | php-pgsql | pecl |
| amqp | php{v}-amqp | php{vnd}-php-pecl-amqp | pecl | pecl |
| grpc | php{v}-grpc | php{vnd}-php-pecl-grpc | pecl | pecl |
| tidy | php{v}-tidy | php{vnd}-php-tidy | php-tidy | pecl |
| ldap | php{v}-ldap | php{vnd}-php-ldap | pecl | pecl |

For unknown extension names, MageBox falls back to a naming convention guess (e.g., `php8.3-{name}` on Ubuntu). Use `magebox ext search` to find the exact package name if the guess fails.

For anything not available as a system package, use the `vendor/package` PIE format instead.

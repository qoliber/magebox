# Custom Commands

Define project-specific commands in your `.magebox.yaml` configuration.

## Basic Usage

Add commands to your `.magebox.yaml` file:

```yaml
commands:
  reindex: "php bin/magento indexer:reindex"
```

Run with:

```bash
magebox run reindex
```

## Command Formats

### Simple String

```yaml
commands:
  cache: "php bin/magento cache:flush"
```

### With Description

```yaml
commands:
  cache:
    description: "Flush all Magento caches"
    run: "php bin/magento cache:flush"
```

### Multi-line

```yaml
commands:
  deploy:
    description: "Full deployment process"
    run: |
      php bin/magento maintenance:enable
      php bin/magento setup:upgrade
      php bin/magento setup:di:compile
      php bin/magento setup:static-content:deploy -f
      php bin/magento maintenance:disable
```

## Example Commands

### Development

```yaml
commands:
  # Quick cache clear
  cc:
    description: "Clear cache"
    run: "php bin/magento cache:clean"

  # Full cache flush
  cf:
    description: "Flush all caches"
    run: |
      php bin/magento cache:flush
      redis-cli FLUSHALL

  # Reindex
  ri:
    description: "Reindex all"
    run: "php bin/magento indexer:reindex"

  # Development mode
  dev:
    description: "Switch to developer mode"
    run: "php bin/magento deploy:mode:set developer"
```

### Code Quality

```yaml
commands:
  # PHPStan
  stan:
    description: "Run PHPStan analysis"
    run: "php vendor/bin/phpstan analyse app/code --level=6"

  # PHP CodeSniffer
  cs:
    description: "Check coding standards"
    run: "php vendor/bin/phpcs app/code --standard=Magento2"

  # Fix coding standards
  cs-fix:
    description: "Fix coding standards"
    run: "php vendor/bin/phpcbf app/code --standard=Magento2"

  # Unit tests
  test:
    description: "Run unit tests"
    run: "php vendor/bin/phpunit -c dev/tests/unit/phpunit.xml.dist"
```

### Module Development

```yaml
commands:
  # Enable module
  mod-enable:
    description: "Enable and setup module"
    run: |
      php bin/magento module:enable Vendor_Module
      php bin/magento setup:upgrade
      php bin/magento cache:flush

  # Generate URN catalog
  urn:
    description: "Generate URN catalog for IDE"
    run: "php bin/magento dev:urn-catalog:generate .idea/misc.xml"

  # Generate DI
  di:
    description: "Compile DI"
    run: "php bin/magento setup:di:compile"
```

### Production Simulation

```yaml
commands:
  # Production mode
  prod:
    description: "Switch to production mode"
    run: |
      php bin/magento deploy:mode:set production
      php bin/magento setup:static-content:deploy -f
      php bin/magento cache:flush

  # Full build
  build:
    description: "Full production build"
    run: |
      composer install --no-dev
      php bin/magento setup:upgrade --keep-generated
      php bin/magento setup:di:compile
      php bin/magento setup:static-content:deploy -f -j 4
```

### Database Operations

```yaml
commands:
  # Create backup
  backup:
    description: "Create database backup"
    run: "magebox db export backups/$(date +%Y%m%d_%H%M%S).sql.gz"

  # Reset database
  db-reset:
    description: "Reset database (WARNING: destructive)"
    run: |
      mysql -h 127.0.0.1 -P 33080 -u root -proot -e "DROP DATABASE IF EXISTS mystore; CREATE DATABASE mystore;"
      php bin/magento setup:install ...
```

### Utility

```yaml
commands:
  # Admin user
  admin:
    description: "Create admin user"
    run: |
      php bin/magento admin:user:create \
        --admin-user=admin \
        --admin-password=admin123 \
        --admin-email=admin@example.com \
        --admin-firstname=Admin \
        --admin-lastname=User

  # Sample data
  sampledata:
    description: "Install sample data"
    run: |
      php bin/magento sampledata:deploy
      php bin/magento setup:upgrade
```

## Personal Commands

Add personal commands in `.magebox.local.yaml`:

```yaml
# .magebox.local.yaml
commands:
  # My test file
  my-test:
    description: "Run my specific tests"
    run: "php vendor/bin/phpunit tests/Unit/MyFeature"

  # Quick debug
  debug:
    description: "Enable Xdebug and clear cache"
    run: |
      export XDEBUG_MODE=debug
      php bin/magento cache:flush
```

## Listing Commands

View available commands:

```bash
magebox run --list
```

Or check your `.magebox.yaml` file.

## Command Environment

Commands run with:
- Correct PHP version from project config
- Environment variables from `env:` section
- Working directory set to project root

```yaml
env:
  COMPOSER_MEMORY_LIMIT: -1
  MAGE_MODE: developer

commands:
  install:
    run: "composer install"  # Uses env vars above
```

## Tips

1. **Use short names** for frequently used commands (`cc`, `ri`, `cf`)

2. **Group related commands** with prefixes (`test-unit`, `test-integration`)

3. **Add descriptions** for complex commands

4. **Use multi-line** for sequences that should run together

5. **Keep destructive commands** in `.magebox.local.yaml` to prevent accidents

6. **Document team commands** in `.magebox.yaml` for shared workflows

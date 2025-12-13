# Local Overrides

The `.magebox.local.yaml` file allows personal customization without affecting the shared `.magebox.yaml` configuration.

## Purpose

Use `.magebox.local.yaml` to:

- Override PHP version for your machine
- Add personal environment variables
- Enable/disable services locally
- Define personal shortcuts

## Creating Local Overrides

Create a `.magebox.local.yaml` file in your project root:

```yaml
# .magebox.local.yaml
php: "8.3"

env:
  XDEBUG_MODE: debug
```

## Configuration Merge

Local settings merge with and override project settings:

```
.magebox.yaml (committed)     .magebox.local.yaml (personal)     Result
─────────────────────    ─────────────────────────     ───────
php: "8.2"               php: "8.3"                    php: "8.3"

services:                services:                     services:
  mysql: "8.0"             redis: false                  mysql: "8.0"
  redis: true                                            redis: false

env:                     env:                          env:
  MAGE_MODE: developer     XDEBUG_MODE: debug            MAGE_MODE: developer
                                                         XDEBUG_MODE: debug
```

## Common Use Cases

### Different PHP Version

Your project uses PHP 8.2, but you want to test with 8.3:

```yaml
# .magebox.local
php: "8.3"
```

### Enable Xdebug

```yaml
# .magebox.local
env:
  XDEBUG_MODE: debug
  XDEBUG_CONFIG: "client_host=127.0.0.1 client_port=9003"
```

### Disable Unused Services

Save resources by disabling services you don't need:

```yaml
# .magebox.local
services:
  rabbitmq: false
  opensearch: false
```

### Additional Domains

Add a personal testing domain:

```yaml
# .magebox.local
domains:
  - host: mystore.test
    root: pub
  - host: dev.mystore.test
    root: pub
```

### Personal Commands

Add shortcuts for your workflow:

```yaml
# .magebox.local
commands:
  quick-test:
    description: "Run my test suite"
    run: "php vendor/bin/phpunit tests/Unit/MyTests"
```

## Git Configuration

Add `.magebox.local.yaml` to `.gitignore`:

```gitignore
# .gitignore
.magebox.local.yaml
```

This ensures personal settings aren't committed to the repository.

## Switching PHP Versions

The quickest way to change PHP version locally:

```bash
magebox php 8.3
```

This automatically updates `.magebox.local.yaml`:

```yaml
# .magebox.local.yaml (created/updated automatically)
php: "8.3"
```

## Validation

MageBox validates the merged configuration. If `.magebox.local.yaml` contains invalid values, you'll see an error:

```
Error: Invalid PHP version '7.4' in .magebox.local.yaml
```

## Example Team Setup

### Shared Configuration (`.magebox.yaml`)

```yaml
name: acme-store

domains:
  - host: acme.test
    root: pub

php: "8.2"

services:
  mysql: "8.0"
  redis: true
  opensearch: "2.12"
  rabbitmq: true
  mailpit: true

commands:
  deploy: "php bin/magento deploy:mode:set production"
  test: "php vendor/bin/phpunit"
```

### Developer A (`.magebox.local`)

```yaml
php: "8.3"  # Testing PHP 8.3 compatibility

env:
  XDEBUG_MODE: debug

commands:
  my-test: "php vendor/bin/phpunit tests/Unit/Feature/MyFeatureTest.php"
```

### Developer B (`.magebox.local`)

```yaml
services:
  rabbitmq: false  # Not working on queue features

env:
  COMPOSER_MEMORY_LIMIT: 4G
```

Both developers share the same base configuration but have personal adjustments for their workflow.

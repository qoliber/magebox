# Project Configuration

Each MageBox project is configured via a `.magebox.yaml` file in the project root.

## Creating Configuration

Initialize a new configuration:

```bash
magebox init mystore
```

This creates a `.magebox.yaml` file with sensible defaults.

::: tip Backward Compatibility
MageBox also supports the legacy `.magebox` filename for backward compatibility.
:::

## Configuration File

### Complete Example

```yaml
# Project name (used for database name, container names)
name: mystore

# Domain configuration
domains:
  - host: mystore.test
    root: pub
    ssl: true
  - host: api.mystore.test
    root: pub
    ssl: true

# PHP version (required)
php: "8.3"

# PHP INI overrides (optional)
php_ini:
  opcache.enable: "0"
  display_errors: "On"
  xdebug.mode: "debug"
  memory_limit: "2G"

# Services configuration
services:
  mysql: "8.0"
  redis: true
  opensearch:
    version: "2.19"
    memory: "2g"
  rabbitmq: true
  mailpit: true
  varnish: false

# Environment variables
env:
  MAGE_MODE: developer
  COMPOSER_MEMORY_LIMIT: -1

# Custom commands
commands:
  deploy:
    description: "Deploy to production mode"
    run: |
      php bin/magento deploy:mode:set production
      php bin/magento setup:di:compile
      php bin/magento setup:static-content:deploy -f

  reindex:
    description: "Reindex all indexes"
    run: "php bin/magento indexer:reindex"

  cache:
    description: "Flush all caches"
    run: "php bin/magento cache:flush"
```

## Configuration Options

### name

**Required** - Project identifier used for:
- Database name
- Docker container naming
- PHP-FPM pool name
- Nginx vhost file name

```yaml
name: mystore
```

### domains

Array of domain configurations:

```yaml
domains:
  - host: mystore.test      # Domain name
    root: pub               # Document root (relative to project)
    ssl: true               # Enable HTTPS (default: true)
```

#### Multiple Domains

```yaml
domains:
  - host: mystore.test
    root: pub
  - host: admin.mystore.test
    root: pub
  - host: api.mystore.test
    root: pub
```

#### Custom Document Root

For non-standard Magento setups:

```yaml
domains:
  - host: mystore.test
    root: public_html
```

### php

**Required** - PHP version for this project:

```yaml
php: "8.3"
```

Supported versions:
- `8.1`
- `8.2`
- `8.3`
- `8.4`

### php_ini

Override PHP configuration settings per-project:

```yaml
php_ini:
  opcache.enable: "0"            # Disable OPcache for development
  display_errors: "On"           # Show PHP errors
  error_reporting: "E_ALL"       # Report all errors
  max_execution_time: "3600"     # 1 hour timeout
  memory_limit: "2G"             # 2GB memory
  xdebug.mode: "debug,coverage"  # Enable Xdebug
```

See [PHP INI Settings](/guide/php-ini) for detailed information.

### services

Configure which services to enable:

#### MySQL

```yaml
services:
  mysql: "8.0"  # Version: "5.7", "8.0", "8.4"
```

#### MariaDB (instead of MySQL)

```yaml
services:
  mariadb: "10.6"  # Version: "10.4", "10.6", "11.4"
```

::: warning
Only configure `mysql` OR `mariadb`, not both.
:::

#### Redis

```yaml
services:
  redis: true  # Enable/disable
```

#### OpenSearch

```yaml
services:
  # Simple format (uses default 1GB memory)
  opensearch: "2.19"

  # Extended format with memory allocation
  opensearch:
    version: "2.19"
    memory: "2g"    # Recommended for production-like performance
```

MageBox automatically installs **ICU Analysis** and **Phonetic Analysis** plugins.

#### Elasticsearch (alternative to OpenSearch)

```yaml
services:
  elasticsearch:
    version: "8.11"
    memory: "2g"
```

#### RabbitMQ

```yaml
services:
  rabbitmq: true  # Enable/disable
```

#### Mailpit

```yaml
services:
  mailpit: true  # Enable/disable
```

#### Varnish

```yaml
services:
  varnish: true  # Enable/disable
```

### env

Environment variables passed to PHP-FPM:

```yaml
env:
  MAGE_MODE: developer
  COMPOSER_MEMORY_LIMIT: -1
  XDEBUG_MODE: debug
  XDEBUG_CONFIG: "client_host=host.docker.internal"
```

These are available in your PHP code via `$_ENV` and `getenv()`.

### commands

Define custom commands for your project:

#### Simple Command

```yaml
commands:
  reindex: "php bin/magento indexer:reindex"
```

Run with:
```bash
magebox run reindex
```

#### Command with Description

```yaml
commands:
  reindex:
    description: "Reindex all indexes"
    run: "php bin/magento indexer:reindex"
```

#### Multi-line Command

```yaml
commands:
  deploy:
    description: "Full production deployment"
    run: |
      php bin/magento maintenance:enable
      php bin/magento setup:upgrade
      php bin/magento setup:di:compile
      php bin/magento setup:static-content:deploy -f
      php bin/magento maintenance:disable
```

## Validation

MageBox validates your configuration on every command. Common errors:

### Missing Required Fields

```
Error: 'name' is required in .magebox.yaml
```

### Invalid PHP Version

```
Error: PHP version '7.4' is not supported. Use 8.1, 8.2, 8.3, or 8.4
```

### Invalid Service Version

```
Error: MySQL version '5.6' is not supported. Use 5.7, 8.0, or 8.4
```

## Best Practices

1. **Commit `.magebox.yaml`** to version control so team members share the same configuration

2. **Use `.magebox.local.yaml`** for personal overrides (add to `.gitignore`)

3. **Match production PHP version** to catch compatibility issues early

4. **Define common commands** to standardize team workflows

5. **Use descriptive project names** to easily identify projects in `magebox list`

## Database Credentials

Default database credentials:

| Property | Value |
|----------|-------|
| Host | 127.0.0.1 |
| Port | See [Service Ports](/reference/ports) |
| Username | root |
| Password | magebox |
| Database | `<project-name>` |

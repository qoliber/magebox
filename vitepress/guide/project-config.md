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

# Custom Docker containers (optional)
compose_file: docker-compose.yml

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

::: tip Auto-discovery
When `root` is omitted, MageBox automatically detects the document root by checking for common directories in your project: `pub`, `public`, `web`, `htdocs`, `httpdocs`. The first one found is used. If none exist, it defaults to `pub` (Magento) or `public` (Laravel).
:::

#### Multiple Domains

```yaml
domains:
  - host: mystore.test
  - host: admin.mystore.test
  - host: api.mystore.test
```

#### Custom Document Root

For non-standard setups, set `root` explicitly:

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

#### Redis / Valkey

```yaml
services:
  redis: true   # Enable/disable
  # or
  valkey: true  # Redis-compatible alternative
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

### compose_file

Path to a project-specific `docker-compose.yml` for custom Docker containers:

```yaml
compose_file: docker-compose.yml
```

When set, `magebox start` and `magebox stop` will prompt you to start/stop these containers alongside the standard MageBox services. The containers are automatically connected to the MageBox Docker network, so they can communicate with MySQL, Redis/Valkey, OpenSearch, and other MageBox services.

**Example:** A Magento project with a Python microservice for PDF generation:

```yaml
name: mystore
php: "8.3"
domains:
  - host: mystore.test
services:
  mysql: "8.0"
  redis: true
compose_file: docker-compose.yml
```

```yaml
# docker-compose.yml (in project root)
services:
  pdf-service:
    build: ./pdf-service
    ports:
      - "8001:8000"
    volumes:
      - ./storage:/data
    restart: unless-stopped
```

After `magebox start`, the `pdf-service` container can reach MySQL at `magebox-mysql-8.0:3306` on the shared network, and your PHP application can reach the PDF service at `localhost:8001`.

::: tip
The path is relative to the project root. Use absolute paths if the compose file is elsewhere.
:::

::: info Confirmation Prompt
MageBox always asks for confirmation before starting or stopping custom containers, so they won't be accidentally disrupted.
:::

---

### include_config

Split your `.magebox.yaml` across multiple files by listing them under `include_config`. Paths are relative to the file that declares them. This is useful for large projects where you want to organise configuration by concern (e.g. services, commands, testing).

```yaml
include_config:
  - ./.magebox/services.yaml
  - ./.magebox/commands.yaml
  - ./.magebox/testing.yaml
```

You can also point to a **directory** — every `.yaml`/`.yml` file inside it is loaded automatically, sorted by filename:

```yaml
include_config:
  - ./.magebox  # loads all .yaml/.yml files in this directory
```

#### How merging works

Included files are merged in the order they are listed. Fields set in the *current file* always win over values from included files. Map fields (`env`, `commands`, `php_ini`) and services are deep-merged, so entries from multiple files accumulate; later entries override earlier ones for the same key.

**Example: splitting a large config**

```yaml
# .magebox.yaml
name: example-store
domains:
  - host: example-store.test
php: "8.3"
services:
  mysql: "8.0"
include_config:
  - ./.magebox/init.yaml
  - ./.magebox/deploy.yaml
  - ./.magebox/review.yaml
```

```yaml
# .magebox/init.yaml
commands:
  setup:
    description: "Install Magento"
    run: |
      composer install
      php bin/magento setup:install
```

```yaml
# .magebox/deploy.yaml
commands:
  deploy:
    description: "Deploy to production mode"
    run: |
      php bin/magento deploy:mode:set production
      php bin/magento setup:di:compile
      php bin/magento setup:static-content:deploy -f
env:
  DEPLOY_ENV: production
```

::: tip
Included files follow the exact same format as `.magebox.yaml`. They can themselves contain `include_config` entries for deeper nesting.
:::

::: warning Circular includes
MageBox detects circular includes and returns an error if a file is included more than once in the same load chain.
:::

---

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

6. **Split large configs** with `include_config` — put commands, services, and testing config in separate files under a `.magebox/` directory for easier maintenance

## Database Credentials

Default database credentials:

| Property | Value |
|----------|-------|
| Host | 127.0.0.1 |
| Port | See [Service Ports](/reference/ports) |
| Username | root |
| Password | magebox |
| Database | `<project-name>` |

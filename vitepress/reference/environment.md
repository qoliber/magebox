# Environment Variables

Reference for environment variables in MageBox.

## Project Environment Variables

Define environment variables in your `.magebox.yaml` file:

```yaml
env:
  MAGE_MODE: developer
  COMPOSER_MEMORY_LIMIT: -1
  XDEBUG_MODE: debug
```

These are passed to PHP-FPM and available in your PHP code.

## Common Variables

### Magento Mode

```yaml
env:
  MAGE_MODE: developer  # or: production, default
```

### Composer

```yaml
env:
  COMPOSER_MEMORY_LIMIT: -1
  COMPOSER_HOME: /home/user/.composer
```

### Xdebug

```yaml
env:
  XDEBUG_MODE: debug,coverage
  XDEBUG_CONFIG: "client_host=127.0.0.1 client_port=9003"
  XDEBUG_SESSION: PHPSTORM
```

### PHP

```yaml
env:
  PHP_MEMORY_LIMIT: 2G
  PHP_MAX_EXECUTION_TIME: 300
```

## Accessing in PHP

### $_ENV

```php
$mode = $_ENV['MAGE_MODE'] ?? 'default';
```

### getenv()

```php
$mode = getenv('MAGE_MODE') ?: 'default';
```

### Magento Configuration

Magento automatically reads several environment variables:

| Variable | Purpose |
|----------|---------|
| `MAGE_MODE` | Application mode |
| `MAGE_RUN_CODE` | Store/website code |
| `MAGE_RUN_TYPE` | Run type (store/website) |

## MageBox System Variables

MageBox sets these variables automatically:

| Variable | Description |
|----------|-------------|
| `MAGEBOX_PROJECT` | Project name |
| `MAGEBOX_PHP_VERSION` | Configured PHP version |
| `MAGEBOX_ROOT` | Project root directory |

## Shell Environment

When using `magebox shell`, these variables are set:

```bash
magebox shell
echo $PHP_VERSION    # 8.2
echo $MAGEBOX_ROOT   # /path/to/project
```

## Global Configuration Override

Override global settings via environment:

| Variable | Config Key |
|----------|------------|
| `MAGEBOX_DNS_MODE` | dns_mode |
| `MAGEBOX_DEFAULT_PHP` | default_php |
| `MAGEBOX_TLD` | tld |

Example:

```bash
MAGEBOX_DEFAULT_PHP=8.4 magebox init mystore
```

## Service Connection Variables

Use these in your application to connect to services:

### Database

```yaml
env:
  DB_HOST: 127.0.0.1
  DB_PORT: "33080"
  DB_NAME: mystore
  DB_USER: root
  DB_PASSWORD: magebox
```

### Redis

```yaml
env:
  REDIS_HOST: 127.0.0.1
  REDIS_PORT: "6379"
```

### OpenSearch

```yaml
env:
  OPENSEARCH_HOST: 127.0.0.1
  OPENSEARCH_PORT: "9200"
```

### RabbitMQ

```yaml
env:
  RABBITMQ_HOST: 127.0.0.1
  RABBITMQ_PORT: "5672"
  RABBITMQ_USER: guest
  RABBITMQ_PASSWORD: guest
```

### Mailpit

```yaml
env:
  MAIL_HOST: 127.0.0.1
  MAIL_PORT: "1025"
```

## Example: Complete Configuration

```yaml
name: mystore

domains:
  - host: mystore.test
    root: pub

php: "8.2"

services:
  mysql: "8.0"
  redis: true
  opensearch: "2.19.4"
  rabbitmq: true
  mailpit: true

env:
  # Magento
  MAGE_MODE: developer

  # PHP
  COMPOSER_MEMORY_LIMIT: -1

  # Xdebug
  XDEBUG_MODE: debug
  XDEBUG_CONFIG: "client_host=127.0.0.1 client_port=9003"

  # Database
  DB_HOST: 127.0.0.1
  DB_PORT: "33080"
  DB_NAME: mystore
  DB_USER: root
  DB_PASSWORD: magebox

  # Redis
  REDIS_HOST: 127.0.0.1
  REDIS_PORT: "6379"

  # Search
  OPENSEARCH_HOST: 127.0.0.1
  OPENSEARCH_PORT: "9200"

  # Queue
  RABBITMQ_HOST: 127.0.0.1
  RABBITMQ_PORT: "5672"
  RABBITMQ_USER: guest
  RABBITMQ_PASSWORD: guest

  # Mail
  MAIL_HOST: 127.0.0.1
  MAIL_PORT: "1025"
```

## Local Overrides

Add personal environment variables in `.magebox.local.yaml`:

```yaml
env:
  XDEBUG_MODE: debug
  MY_DEBUG_FLAG: "1"
```

These merge with the project's environment variables.

## Precedence

Environment variable precedence (highest to lowest):

1. Shell environment (exported before running command)
2. `.magebox.local.yaml` env section
3. `.magebox.yaml` env section
4. MageBox defaults

Example:

```bash
# This takes highest precedence
MAGE_MODE=production magebox cli cache:flush
```

## Using with Magento Setup

During installation:

```bash
magebox cli setup:install \
  --db-host=${DB_HOST} \
  --db-name=${DB_NAME} \
  --db-user=${DB_USER} \
  --db-password=${DB_PASSWORD}
```

Or reference in `app/etc/env.php`:

```php
return [
    'db' => [
        'connection' => [
            'default' => [
                'host' => getenv('DB_HOST') ?: '127.0.0.1',
                'dbname' => getenv('DB_NAME') ?: 'magento',
                'username' => getenv('DB_USER') ?: 'root',
                'password' => getenv('DB_PASSWORD') ?: 'magebox',
                'port' => getenv('DB_PORT') ?: '3306'
            ]
        ]
    ]
];
```

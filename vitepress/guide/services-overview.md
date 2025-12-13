# Services Overview

MageBox runs supporting services in Docker containers while keeping PHP and Nginx native.

## Architecture

```
Native (Full Speed)          Docker (Isolated)
───────────────────          ─────────────────
PHP-FPM ◄─────────────────►  MySQL/MariaDB
Nginx   ◄─────────────────►  Redis
                             OpenSearch
                             RabbitMQ
                             Mailpit
                             Varnish
```

## Service Configuration

Enable services in your `.magebox.yaml` file:

```yaml
services:
  mysql: "8.0"
  redis: true
  opensearch: "2.12"
  rabbitmq: true
  mailpit: true
  varnish: false
```

## Available Services

| Service | Versions | Default Port(s) |
|---------|----------|-----------------|
| MySQL | 5.7, 8.0, 8.4 | 33057, 33080, 33084 |
| MariaDB | 10.4, 10.6, 11.4 | 33104, 33106, 33114 |
| Redis | latest | 6379 |
| OpenSearch | 2.x | 9200 |
| Elasticsearch | 7.x, 8.x | 9200 |
| RabbitMQ | latest | 5672, 15672 |
| Mailpit | latest | 1025, 8025 |
| Varnish | latest | 6081 |

## Service Management

### Start All Services

```bash
magebox global start
```

### Stop All Services

```bash
magebox global stop
```

### Check Status

```bash
magebox global status
```

### Project-Specific

When you run `magebox start` in a project, it ensures all configured services are running.

## Default Credentials

### MySQL/MariaDB

| Property | Value |
|----------|-------|
| Host | 127.0.0.1 |
| Port | See version table |
| Database | `<project-name>` |
| Username | root |
| Password | magebox |

### RabbitMQ

| Property | Value |
|----------|-------|
| Host | 127.0.0.1 |
| AMQP Port | 5672 |
| Management Port | 15672 |
| Username | guest |
| Password | guest |

### Mailpit

| Property | Value |
|----------|-------|
| SMTP Host | 127.0.0.1 |
| SMTP Port | 1025 |
| Web UI | http://localhost:8025 |

## Port Strategy

MageBox uses unique ports per database version to avoid conflicts:

```
MySQL 5.7   → 33057
MySQL 8.0   → 33080
MySQL 8.4   → 33084
MariaDB 10.4 → 33104
MariaDB 10.6 → 33106
MariaDB 11.4 → 33114
```

This allows running multiple database versions simultaneously for different projects.

## Magento Configuration

### Database (app/etc/env.php)

```php
'db' => [
    'connection' => [
        'default' => [
            'host' => '127.0.0.1',
            'dbname' => 'mystore',
            'username' => 'root',
            'password' => 'magebox',
            'port' => '33080',
            'model' => 'mysql4',
            'engine' => 'innodb',
            'active' => '1'
        ]
    ]
]
```

### Redis Session

```php
'session' => [
    'save' => 'redis',
    'redis' => [
        'host' => '127.0.0.1',
        'port' => '6379',
        'database' => '0'
    ]
]
```

### Redis Cache

```php
'cache' => [
    'frontend' => [
        'default' => [
            'backend' => 'Magento\\Framework\\Cache\\Backend\\Redis',
            'backend_options' => [
                'server' => '127.0.0.1',
                'port' => '6379',
                'database' => '1'
            ]
        ]
    ]
]
```

### OpenSearch

```php
'system' => [
    'default' => [
        'catalog' => [
            'search' => [
                'engine' => 'opensearch',
                'opensearch_server_hostname' => '127.0.0.1',
                'opensearch_server_port' => '9200'
            ]
        ]
    ]
]
```

### RabbitMQ

```php
'queue' => [
    'amqp' => [
        'host' => '127.0.0.1',
        'port' => '5672',
        'user' => 'guest',
        'password' => 'guest',
        'virtualhost' => '/'
    ]
]
```

### Email (Mailpit)

```php
'system' => [
    'default' => [
        'smtp' => [
            'transport' => 'smtp',
            'host' => '127.0.0.1',
            'port' => '1025'
        ]
    ]
]
```

## Persistent Data

Service data is stored in Docker volumes:

```bash
# List MageBox volumes
docker volume ls | grep magebox
```

Volumes are named:
- `magebox_mysql80_data`
- `magebox_redis_data`
- `magebox_opensearch_data`
- etc.

## Resource Management

Docker services use system resources. Monitor with:

```bash
docker stats
```

To reduce resource usage, disable unused services:

```yaml
# .magebox.yaml or .magebox.local.yaml
services:
  mysql: "8.0"
  redis: true
  opensearch: false   # Disabled
  rabbitmq: false     # Disabled
  mailpit: false      # Disabled
```

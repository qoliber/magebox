# Service Ports

Reference for all service ports used by MageBox.

## Port Overview

| Service | Port | Protocol | Purpose |
|---------|------|----------|---------|
| Nginx HTTP | 80 | HTTP | Web server |
| Nginx HTTPS | 443 | HTTPS | Web server (SSL) |
| MySQL 5.7 | 33057 | TCP | Database |
| MySQL 8.0 | 33080 | TCP | Database |
| MySQL 8.4 | 33084 | TCP | Database |
| MariaDB 10.4 | 33104 | TCP | Database |
| MariaDB 10.6 | 33106 | TCP | Database |
| MariaDB 11.4 | 33114 | TCP | Database |
| Redis | 6379 | TCP | Cache/Sessions |
| OpenSearch | 9200 | HTTP | Search |
| Elasticsearch | 9200 | HTTP | Search |
| RabbitMQ AMQP | 5672 | AMQP | Message queue |
| RabbitMQ Management | 15672 | HTTP | Management UI |
| Mailpit SMTP | 1025 | SMTP | Email capture |
| Mailpit Web | 8025 | HTTP | Email UI |
| Varnish | 6081 | HTTP | Cache |
| Portainer | 9000 | HTTP | Docker UI |

## Database Ports

### Port Naming Convention

Database ports follow a pattern: `33` + version digits

| Version | Calculation | Port |
|---------|-------------|------|
| MySQL 5.7 | 33 + 057 | 33057 |
| MySQL 8.0 | 33 + 080 | 33080 |
| MySQL 8.4 | 33 + 084 | 33084 |
| MariaDB 10.4 | 33 + 104 | 33104 |
| MariaDB 10.6 | 33 + 106 | 33106 |
| MariaDB 11.4 | 33 + 114 | 33114 |

### Why Different Ports?

Using unique ports per version allows:
- Running multiple database versions simultaneously
- Different projects with different MySQL/MariaDB versions
- No port conflicts

### Connection Strings

```bash
# MySQL 5.7
mysql -h 127.0.0.1 -P 33057 -u root -pmagebox

# MySQL 8.0
mysql -h 127.0.0.1 -P 33080 -u root -pmagebox

# MySQL 8.4
mysql -h 127.0.0.1 -P 33084 -u root -pmagebox

# MariaDB 10.4
mysql -h 127.0.0.1 -P 33104 -u root -pmagebox

# MariaDB 10.6
mysql -h 127.0.0.1 -P 33106 -u root -pmagebox

# MariaDB 11.4
mysql -h 127.0.0.1 -P 33114 -u root -pmagebox
```

## Web Server Ports

### Nginx

| Port | Protocol | Usage |
|------|----------|-------|
| 80 | HTTP | Redirects to HTTPS |
| 443 | HTTPS | Primary access |

### Varnish

| Port | Protocol | Usage |
|------|----------|-------|
| 6081 | HTTP | Cache layer |

Access your site:
- **Direct**: https://mystore.test (port 443)
- **Via Varnish**: http://mystore.test:6081

## Cache & Search Ports

### Redis

| Port | Protocol |
|------|----------|
| 6379 | TCP |

```bash
redis-cli -h 127.0.0.1 -p 6379
```

### OpenSearch / Elasticsearch

| Port | Protocol |
|------|----------|
| 9200 | HTTP |

```bash
curl http://127.0.0.1:9200
```

## Message Queue Ports

### RabbitMQ

| Port | Protocol | Purpose |
|------|----------|---------|
| 5672 | AMQP | Message queue |
| 15672 | HTTP | Management UI |

Access Management UI: http://localhost:15672

## Email Ports

### Mailpit

| Port | Protocol | Purpose |
|------|----------|---------|
| 1025 | SMTP | Email capture |
| 8025 | HTTP | Web interface |

Access Web UI: http://localhost:8025

## Magento Configuration

### Database (app/etc/env.php)

```php
'db' => [
    'connection' => [
        'default' => [
            'host' => '127.0.0.1',
            'port' => '33080',  // Adjust for your MySQL version
            'dbname' => 'mystore',
            'username' => 'root',
            'password' => 'magebox'
        ]
    ]
]
```

### Redis

```php
'session' => [
    'save' => 'redis',
    'redis' => [
        'host' => '127.0.0.1',
        'port' => '6379',
        'database' => '0'
    ]
],
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
        'password' => 'guest'
    ]
]
```

### Email (Mailpit)

```php
'system' => [
    'default' => [
        'smtp' => [
            'host' => '127.0.0.1',
            'port' => '1025'
        ]
    ]
]
```

## Checking Port Usage

### List Used Ports

```bash
# All MageBox-related ports
netstat -tlnp | grep -E "33(057|080|084|104|106|114)|6379|9200|5672|15672|1025|8025|6081"

# Or with lsof
lsof -i -P -n | grep LISTEN | grep -E "33|6379|9200|5672|15672|1025|8025|6081"
```

### Check Specific Port

```bash
# Check if port 33080 is in use
lsof -i :33080
```

## Port Conflicts

### Common Conflicts

| Port | Possible Conflict |
|------|-------------------|
| 80/443 | Apache, other web servers |
| 3306 | Local MySQL installation |
| 6379 | Local Redis installation |
| 9200 | Local Elasticsearch |

### Resolution

1. Stop conflicting services:

```bash
# Stop local MySQL
sudo systemctl stop mysql

# Stop local Apache
sudo systemctl stop apache2
```

2. Or configure MageBox to use different ports (coming in future versions)

## Docker Port Mapping

Services bind to localhost only:

```
127.0.0.1:33080 → container:3306
127.0.0.1:6379  → container:6379
127.0.0.1:9200  → container:9200
```

This means services are only accessible from your machine, not from the network.

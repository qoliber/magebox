# Redis

Redis provides fast in-memory caching and session storage for Magento.

## Configuration

Enable Redis in your `.magebox.yaml` file:

```yaml
services:
  redis: true
```

## Connection Details

| Property | Value |
|----------|-------|
| Host | 127.0.0.1 |
| Port | 6379 |
| Password | None (no authentication) |

## Redis Commands

### Redis Shell

Connect to Redis CLI:

```bash
magebox redis shell
```

### Flush All Data

Clear all Redis data:

```bash
magebox redis flush
```

### View Info

Show Redis statistics:

```bash
magebox redis info
```

## Magento Configuration

### Session Storage

Store sessions in Redis (`app/etc/env.php`):

```php
'session' => [
    'save' => 'redis',
    'redis' => [
        'host' => '127.0.0.1',
        'port' => '6379',
        'password' => '',
        'timeout' => '2.5',
        'persistent_identifier' => '',
        'database' => '0',
        'compression_threshold' => '2048',
        'compression_library' => 'gzip',
        'log_level' => '1',
        'max_concurrency' => '6',
        'break_after_frontend' => '5',
        'break_after_adminhtml' => '30',
        'first_lifetime' => '600',
        'bot_first_lifetime' => '60',
        'bot_lifetime' => '7200',
        'disable_locking' => '0',
        'min_lifetime' => '60',
        'max_lifetime' => '2592000'
    ]
]
```

### Default Cache

Use Redis for the default cache:

```php
'cache' => [
    'frontend' => [
        'default' => [
            'id_prefix' => 'mystore_',
            'backend' => 'Magento\\Framework\\Cache\\Backend\\Redis',
            'backend_options' => [
                'server' => '127.0.0.1',
                'port' => '6379',
                'database' => '1',
                'compress_data' => '1'
            ]
        ]
    ]
]
```

### Page Cache

Use Redis for full page cache:

```php
'cache' => [
    'frontend' => [
        'default' => [
            // ... default cache config
        ],
        'page_cache' => [
            'id_prefix' => 'mystore_',
            'backend' => 'Magento\\Framework\\Cache\\Backend\\Redis',
            'backend_options' => [
                'server' => '127.0.0.1',
                'port' => '6379',
                'database' => '2',
                'compress_data' => '0'
            ]
        ]
    ]
]
```

## Database Allocation

Recommended Redis database allocation for Magento:

| Database | Purpose |
|----------|---------|
| 0 | Sessions |
| 1 | Default cache |
| 2 | Page cache |

## Common Operations

### View All Keys

```bash
magebox redis shell
127.0.0.1:6379> KEYS *
```

### Check Memory Usage

```bash
magebox redis shell
127.0.0.1:6379> INFO memory
```

### Monitor Operations

Watch real-time Redis commands:

```bash
magebox redis shell
127.0.0.1:6379> MONITOR
```

Press `Ctrl+C` to stop.

### Clear Specific Database

```bash
magebox redis shell
127.0.0.1:6379> SELECT 1
127.0.0.1:6379[1]> FLUSHDB
```

### Clear All Databases

```bash
magebox redis shell
127.0.0.1:6379> FLUSHALL
```

## Troubleshooting

### Connection Refused

Check if Redis container is running:

```bash
docker ps | grep redis
```

Start services:

```bash
magebox global start
```

### Cache Not Working

Verify Redis is configured in Magento:

```bash
magebox cli config:show | grep redis
```

Check Redis connection from Magento:

```bash
magebox shell
php -r "echo (new Redis())->connect('127.0.0.1', 6379) ? 'Connected' : 'Failed';"
```

### Memory Full

Check memory usage:

```bash
magebox redis info | grep used_memory
```

If memory is full, consider:

1. Flush old data: `magebox redis flush`
2. Configure memory limits in Redis
3. Review cache TTL settings

### Sessions Lost

If sessions are being lost:

1. Check Redis is running
2. Verify session config in `env.php`
3. Check Redis database 0 has data:

```bash
magebox redis shell
127.0.0.1:6379> SELECT 0
127.0.0.1:6379[0]> KEYS sess:*
```

## Performance Tips

1. **Use separate databases** for sessions, cache, and FPC to allow independent flushing

2. **Enable compression** for default cache to reduce memory usage

3. **Disable compression** for page cache for faster response times

4. **Monitor memory** regularly to prevent cache eviction issues

5. **Use persistent connections** to reduce connection overhead

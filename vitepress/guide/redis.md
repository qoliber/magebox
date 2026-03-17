# Redis / Valkey

Redis and Valkey provide fast in-memory caching and session storage for Magento.

[Valkey](https://valkey.io/) is a Redis-compatible fork maintained by the Linux Foundation. It uses the same protocol and port, so the Magento configuration is identical.

## Configuration

Enable Redis or Valkey in your `.magebox.yaml` file:

```yaml
# Option 1: Redis (default)
services:
  redis: true
```

```yaml
# Option 2: Valkey
services:
  valkey: true
```

::: tip
Valkey is wire-compatible with Redis. The Magento configuration (env.php) is identical for both — only the Docker container differs.
:::

## Connection Details

| Property | Value |
|----------|-------|
| Host | 127.0.0.1 |
| Port | 6379 |
| Password | None (no authentication) |

## Cache Commands

### Cache Shell

Connect to Redis/Valkey CLI:

```bash
magebox redis shell
```

### Flush All Data

Clear all cache data:

```bash
magebox redis flush
```

### View Info

Show server statistics:

```bash
magebox redis info
```

::: info
The `magebox redis` commands automatically detect whether Redis or Valkey is configured and use the appropriate CLI tool.
:::

## Magento Configuration

### Session Storage

Store sessions in Redis/Valkey (`app/etc/env.php`):

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

Use Redis/Valkey for the default cache:

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

Use Redis/Valkey for full page cache:

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

::: tip
Magento's `env.php` uses `redis` as the backend name even when using Valkey, since Valkey is protocol-compatible.
:::

## Database Allocation

Recommended database allocation for Magento:

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

Watch real-time commands:

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

Check if the cache container is running:

```bash
docker ps | grep -E "redis|valkey"
```

Start services:

```bash
magebox global start
```

### Cache Not Working

Verify the cache service is configured in Magento:

```bash
magebox cli config:show | grep redis
```

### Memory Full

Check memory usage:

```bash
magebox redis info | grep used_memory
```

If memory is full, consider:

1. Flush old data: `magebox redis flush`
2. Configure memory limits
3. Review cache TTL settings

### Sessions Lost

If sessions are being lost:

1. Check the cache service is running
2. Verify session config in `env.php`
3. Check the session database has data:

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

# Redis / Valkey

MageBox runs Redis or Valkey in Docker for caching, sessions, and full-page cache storage.

[Valkey](https://valkey.io/) is a Redis-compatible fork maintained by the Linux Foundation. It uses the same protocol, port, and CLI commands, making it a drop-in replacement.

## Overview

Redis/Valkey is a high-performance in-memory data store used by Magento for:

- **Cache storage** - Configuration, layout, block HTML
- **Session storage** - Customer sessions
- **Full Page Cache (FPC)** - Complete page responses

## Configuration

### Enabling Redis

In `.magebox.yaml`:

```yaml
services:
  redis: true
```

### Enabling Valkey

In `.magebox.yaml`:

```yaml
services:
  valkey: true
```

::: tip
Valkey is wire-compatible with Redis. The Magento configuration (`env.php`) is identical for both — only the Docker container differs.
:::

### Connection Details

| Setting | Value |
|---------|-------|
| Host | `127.0.0.1` |
| Port | `6379` |
| Password | None (no authentication) |

## Magento Configuration

### Via Install Command

```bash
php bin/magento setup:install \
    --session-save=redis \
    --session-save-redis-host=127.0.0.1 \
    --session-save-redis-port=6379 \
    --session-save-redis-db=2 \
    --cache-backend=redis \
    --cache-backend-redis-server=127.0.0.1 \
    --cache-backend-redis-port=6379 \
    --cache-backend-redis-db=0 \
    --page-cache=redis \
    --page-cache-redis-server=127.0.0.1 \
    --page-cache-redis-port=6379 \
    --page-cache-redis-db=1 \
    # ... other options
```

::: info
Magento uses `redis` as the backend name in all commands and configuration, even when using Valkey.
:::

### Via env.php

```php
// app/etc/env.php
'session' => [
    'save' => 'redis',
    'redis' => [
        'host' => '127.0.0.1',
        'port' => '6379',
        'password' => '',
        'timeout' => '2.5',
        'persistent_identifier' => '',
        'database' => '2',
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
],
'cache' => [
    'frontend' => [
        'default' => [
            'backend' => 'Magento\\Framework\\Cache\\Backend\\Redis',
            'backend_options' => [
                'server' => '127.0.0.1',
                'port' => '6379',
                'database' => '0',
                'compress_data' => '1'
            ]
        ],
        'page_cache' => [
            'backend' => 'Magento\\Framework\\Cache\\Backend\\Redis',
            'backend_options' => [
                'server' => '127.0.0.1',
                'port' => '6379',
                'database' => '1',
                'compress_data' => '0'
            ]
        ]
    ]
],
```

### Database Separation

Best practice is to use separate databases:

| Database | Purpose |
|----------|---------|
| `0` | Default cache |
| `1` | Full Page Cache |
| `2` | Sessions |

## Cache Commands

The `magebox redis` commands work with both Redis and Valkey. They automatically detect which service is configured and use the appropriate CLI tool (`redis-cli` or `valkey-cli`).

### Open CLI Shell

```bash
magebox redis shell
```

This opens an interactive CLI connected to the container.

### Flush All Data

```bash
magebox redis flush
```

This clears all databases (cache, FPC, sessions).

### Show Server Info

```bash
magebox redis info
```

Displays server statistics, memory usage, and connection info.

### Direct CLI Access

```bash
# Connect directly (works for both Redis and Valkey)
redis-cli -h 127.0.0.1 -p 6379

# Run specific command
redis-cli -h 127.0.0.1 -p 6379 INFO memory
```

## Common Operations

### Clear Specific Database

```bash
# Clear default cache (db 0)
redis-cli -h 127.0.0.1 SELECT 0
redis-cli -h 127.0.0.1 FLUSHDB

# Clear FPC (db 1)
redis-cli -h 127.0.0.1 SELECT 1
redis-cli -h 127.0.0.1 FLUSHDB

# Clear sessions (db 2)
redis-cli -h 127.0.0.1 SELECT 2
redis-cli -h 127.0.0.1 FLUSHDB
```

### Monitor Commands

```bash
# Watch all commands in real-time
redis-cli -h 127.0.0.1 MONITOR
```

### Check Memory Usage

```bash
redis-cli -h 127.0.0.1 INFO memory
```

### Count Keys

```bash
# Count keys in default cache
redis-cli -h 127.0.0.1 -n 0 DBSIZE

# Count keys in FPC
redis-cli -h 127.0.0.1 -n 1 DBSIZE

# Count keys in sessions
redis-cli -h 127.0.0.1 -n 2 DBSIZE
```

## Docker Container

### Container Status

```bash
# Redis
docker ps | grep redis

# Valkey
docker ps | grep valkey
```

### Container Logs

```bash
# Redis
docker logs magebox-redis

# Valkey
docker logs magebox-valkey

# Follow logs
magebox logs redis
```

### Restart Container

```bash
# Redis
docker restart magebox-redis

# Valkey
docker restart magebox-valkey
```

## Performance Tuning

### Memory Limits

By default, the cache service uses available memory. For production-like testing, you can limit memory:

```bash
# Set max memory to 512MB
redis-cli -h 127.0.0.1 CONFIG SET maxmemory 512mb
redis-cli -h 127.0.0.1 CONFIG SET maxmemory-policy allkeys-lru
```

### Persistence

MageBox runs the cache service without persistence (data is lost on container restart). This is intentional for development speed.

For persistent data, consider backing up before container operations:

```bash
# Save current state
redis-cli -h 127.0.0.1 BGSAVE
```

## Troubleshooting

### Connection Refused

```
Redis connection refused
```

**Solutions:**

1. Check if the container is running:
   ```bash
   docker ps | grep -E "redis|valkey"
   ```

2. Start services:
   ```bash
   magebox global start
   ```

3. Check port availability:
   ```bash
   lsof -i :6379
   ```

### Out of Memory

```
OOM command not allowed when used memory > 'maxmemory'
```

**Solutions:**

1. Flush the cache:
   ```bash
   magebox redis flush
   ```

2. Clear Magento cache:
   ```bash
   php bin/magento cache:flush
   ```

3. Increase memory limit:
   ```bash
   redis-cli CONFIG SET maxmemory 1gb
   ```

### Slow Performance

Check if the cache is being used correctly:

```bash
# Check hit rate
redis-cli -h 127.0.0.1 INFO stats | grep keyspace

# Monitor slow queries
redis-cli -h 127.0.0.1 SLOWLOG GET 10
```

### Session Issues

If sessions are not persisting:

1. Verify configuration in `env.php`
2. Check session database:
   ```bash
   redis-cli -h 127.0.0.1 -n 2 KEYS "*"
   ```
3. Test connection:
   ```bash
   redis-cli -h 127.0.0.1 PING
   # Should return: PONG
   ```

## Best Practices

### Development

- Use `magebox redis flush` after major code changes
- Keep FPC disabled during active development
- Monitor memory usage if running multiple projects

### Testing

- Flush cache before performance testing
- Enable FPC to test production-like behavior
- Use `MONITOR` command to debug cache issues

### Cache Warming

After flushing cache:

```bash
# Reindex
php bin/magento indexer:reindex

# Warm cache by visiting key pages
curl -s https://mystore.test/ > /dev/null
curl -s https://mystore.test/catalog/category/view/id/3 > /dev/null
```

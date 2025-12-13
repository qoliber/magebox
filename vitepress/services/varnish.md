# Varnish

MageBox supports Varnish HTTP cache for production-like full page caching.

## Overview

Varnish is a powerful HTTP accelerator that:

- **Caches full pages** - Serves cached responses without hitting PHP
- **Reduces server load** - Dramatically improves performance
- **Edge Side Includes (ESI)** - Dynamic content in cached pages

## Configuration

### Enabling Varnish

In `.magebox.yaml`:

```yaml
services:
  varnish: true
```

Or with custom settings:

```yaml
services:
  varnish:
    version: "7.5"    # Varnish version (default: 7.5)
    memory: "512m"    # Cache memory (default: 256m)
```

### Enable/Disable Commands

```bash
# Enable Varnish for current project
magebox varnish enable

# Disable Varnish
magebox varnish disable
```

## Connection Details

| Setting | Value |
|---------|-------|
| Host | `127.0.0.1` |
| Varnish Port | `6081` |
| Admin Port | `6082` |
| Backend Port | `8080` (Nginx) |

## MageBox Commands

### Check Varnish Status

```bash
magebox varnish status
```

Shows if Varnish is running and backend health:

```
Varnish: running
Backend: 5/5 healthy
```

### Purge Specific URL

```bash
# Purge a specific path
magebox varnish purge /category/page.html

# Purge homepage
magebox varnish purge /
```

### Flush All Cache

```bash
magebox varnish flush
```

Clears the entire Varnish cache using `varnishadm ban`.

## How It Works

### Request Flow with Varnish

```
Browser → Varnish (:6081) → Nginx (:8080) → PHP-FPM
                ↓
        Cache Hit? → Return cached response
```

### Without Varnish (Default)

```
Browser → Nginx (:80/443) → PHP-FPM
```

### Docker Container

Varnish runs as a Docker container (`magebox-varnish`) with:

- **Image**: `varnish:7.5`
- **Ports**: 6081 (HTTP), 6082 (admin)
- **VCL Config**: `~/.magebox/varnish/default.vcl`
- **Network**: Uses host LAN IP to reach Nginx backend

## Magento Configuration

### Via CLI

```bash
# Enable Varnish in Magento
php bin/magento config:set system/full_page_cache/caching_application 2

# Configure Varnish backend
php bin/magento config:set system/full_page_cache/varnish/backend_host 127.0.0.1
php bin/magento config:set system/full_page_cache/varnish/backend_port 8080

# Clear cache
php bin/magento cache:flush
```

### Via env.php

```php
// app/etc/env.php
'system' => [
    'default' => [
        'full_page_cache' => [
            'caching_application' => '2',  // Varnish
            'varnish' => [
                'access_list' => '127.0.0.1',
                'backend_host' => '127.0.0.1',
                'backend_port' => '8080',
                'grace_period' => '300'
            ]
        ]
    ]
]
```

### Generate VCL

Magento can generate an optimized VCL configuration:

```bash
# Generate VCL for Varnish 7
php bin/magento varnish:vcl:generate --export-version=7 > varnish.vcl

# Generate with specific backend
php bin/magento varnish:vcl:generate \
    --backend-host=127.0.0.1 \
    --backend-port=8080 \
    --export-version=7 > varnish.vcl
```

## VCL Configuration

MageBox generates a default VCL at `~/.magebox/varnish/default.vcl`:

```vcl
vcl 4.1;

backend projectname {
    .host = "192.168.x.x";  # Your LAN IP
    .port = "8080";
    .probe = {
        .url = "/health_check.php";
        .timeout = 2s;
        .interval = 5s;
        .window = 5;
        .threshold = 3;
    }
}

sub vcl_recv {
    # Handle purge requests
    if (req.method == "PURGE") {
        return (purge);
    }

    # Don't cache admin
    if (req.url ~ "^/admin") {
        return (pass);
    }
}

sub vcl_backend_response {
    # Set cache TTL
    set beresp.ttl = 1h;
}
```

## Common Operations

### Check Cache Hit/Miss

Look for the `X-Magento-Cache-Debug` header:

```bash
curl -I https://mystore.test/

# Look for:
# X-Magento-Cache-Debug: HIT  (served from cache)
# X-Magento-Cache-Debug: MISS (not in cache)
```

### Enable Debug Headers

```bash
php bin/magento config:set dev/debug/debug_headers 1
```

### Monitor Cache Statistics

```bash
# Check backend health
docker exec magebox-varnish varnishadm backend.list

# View cache stats
docker exec magebox-varnish varnishstat -1 | grep cache

# Watch live stats
docker exec -it magebox-varnish varnishstat
```

### View Request Log

```bash
docker exec -it magebox-varnish varnishlog
```

## Docker Container Management

### Container Status

```bash
docker ps | grep varnish
```

### Container Logs

```bash
docker logs magebox-varnish

# Follow logs
docker logs -f magebox-varnish
```

### Restart Container

```bash
docker restart magebox-varnish
```

### Check Varnish Internals

```bash
# Ping daemon
docker exec magebox-varnish varnishadm ping

# Check status
docker exec magebox-varnish varnishadm status

# List backends
docker exec magebox-varnish varnishadm backend.list
```

## Troubleshooting

### 503 Backend Fetch Failed

Backend (Nginx) is not responding. Common causes:

**1. PHP-FPM not running:**
```bash
# Check if socket exists
ls -la /tmp/magebox/*.sock

# Restart project to reload PHP-FPM
magebox stop && magebox start
```

**2. Backend marked as sick:**
```bash
# Check backend health
docker exec magebox-varnish varnishadm backend.list

# Force backend healthy (temporary)
docker exec magebox-varnish varnishadm "backend.set_health boot1.projectname healthy"
```

**3. Network connectivity:**
```bash
# Test Nginx is reachable
curl -I http://127.0.0.1:8080/ -H "Host: mystore.test"
```

### Pages Not Caching

1. Check Varnish is enabled in Magento:
   ```bash
   php bin/magento config:show system/full_page_cache/caching_application
   # Should return 2 (Varnish)
   ```

2. Verify debug headers:
   ```bash
   curl -I https://mystore.test/ | grep -i cache
   ```

3. Check for cookies preventing caching

4. Verify VCL configuration

### Stale Content

```bash
# Flush Varnish cache
magebox varnish flush

# Clear Magento cache
php bin/magento cache:flush
```

### ESI Not Working

1. Ensure ESI is enabled in VCL
2. Check Magento block is ESI-enabled
3. Verify TTL settings

## Performance Testing

### With Varnish

```bash
# Benchmark cached page
ab -n 1000 -c 10 http://127.0.0.1:6081/

# Should see high requests/second on cache hits
```

### Without Varnish

```bash
# Bypass Varnish for comparison
ab -n 100 -c 10 http://127.0.0.1:8080/
```

## Best Practices

### Development

- **Keep Varnish disabled** during active development
- Use for performance testing only
- Clear cache frequently

### Testing FPC Behavior

1. Enable Varnish
2. Visit page (MISS)
3. Visit again (HIT)
4. Make change
5. Purge cache
6. Verify update

### Cache Warming

After deployment or cache flush:

```bash
# Warm cache by visiting key pages
curl -s https://mystore.test/ > /dev/null
curl -s https://mystore.test/catalog/category/view/id/3 > /dev/null
curl -s https://mystore.test/customer/account/login > /dev/null
```

## Disabling Varnish

### Via Command

```bash
magebox varnish disable
```

### Manually

1. Update `.magebox.yaml`:
   ```yaml
   services:
     varnish: false
   ```

2. Update Magento:
   ```bash
   php bin/magento config:set system/full_page_cache/caching_application 1
   ```

3. Clear cache:
   ```bash
   php bin/magento cache:flush
   ```

4. Restart project:
   ```bash
   magebox restart
   ```

## Varnish vs Redis FPC

| Feature | Varnish | Redis FPC |
|---------|---------|-----------|
| Performance | Fastest | Fast |
| Memory | Dedicated | Shared |
| ESI Support | Yes | No |
| HTTP Standards | Full | N/A |
| Complexity | Higher | Lower |

For development, Redis FPC is usually sufficient. Use Varnish for production-like testing.

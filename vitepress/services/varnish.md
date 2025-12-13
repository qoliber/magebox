# Varnish

MageBox supports Varnish HTTP cache for production-like full page caching.

## Overview

Varnish is a powerful HTTP accelerator that:

- **Caches full pages** - Serves cached responses without hitting PHP
- **Reduces server load** - Dramatically improves performance
- **Edge Side Includes (ESI)** - Dynamic content in cached pages

::: warning Work in Progress
Varnish support in MageBox is currently being refined. Basic functionality is available, but some features may require manual configuration.
:::

## Configuration

### Enabling Varnish

In `.magebox.yaml`:

```yaml
services:
  varnish: true
```

## Connection Details

| Setting | Value |
|---------|-------|
| Host | `127.0.0.1` |
| Port | `6081` |
| Backend Port | `80` or `8080` (Nginx) |

## Magento Configuration

### Generate VCL

Magento can generate an optimized VCL configuration:

1. Go to **Stores → Configuration → Advanced → System → Full Page Cache**
2. Set **Caching Application** to **Varnish Cache**
3. Click **Export VCL**

### Via CLI

```bash
# Generate VCL for Varnish 7
php bin/magento varnish:vcl:generate --export-version=7 > varnish.vcl

# Generate with specific backend
php bin/magento varnish:vcl:generate \
    --backend-host=127.0.0.1 \
    --backend-port=8080 \
    --export-version=7 > varnish.vcl
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

## MageBox Commands

### Check Varnish Status

```bash
magebox varnish status
```

Displays cache statistics and health information.

### Purge Specific URL

```bash
magebox varnish purge /category/page.html
```

### Flush All Cache

```bash
magebox varnish flush
```

Clears the entire Varnish cache.

## How It Works

### Request Flow with Varnish

```
Browser → Varnish (:6081) → Nginx (:80/8080) → PHP-FPM
                ↓
        Cache Hit? → Return cached response
```

### Without Varnish (Default)

```
Browser → Nginx (:80/443) → PHP-FPM
```

## VCL Configuration

### Basic VCL Structure

```vcl
vcl 4.1;

backend default {
    .host = "127.0.0.1";
    .port = "8080";
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

### Magento-Specific VCL

Magento's generated VCL includes:

- Health checks
- Grace mode (serve stale content)
- ESI support
- Admin exclusion
- Static file handling
- Cookie management

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

In Magento Admin:
1. Go to **Stores → Configuration → Advanced → Developer**
2. Set **Debug** → **Enable Debug Headers** → **Yes**

Or via CLI:

```bash
php bin/magento config:set dev/debug/debug_headers 1
```

### Monitor Cache

```bash
# Watch cache stats
varnishstat

# View request log
varnishlog
```

## Docker Container

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

## Troubleshooting

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

### 503 Backend Fetch Failed

Backend (Nginx) is not responding:

```bash
# Check Nginx is running
systemctl status nginx

# Check backend port
curl -I http://127.0.0.1:8080/health_check.php
```

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
ab -n 1000 -c 10 https://mystore.test/

# Should see high requests/second
```

### Without Varnish

```bash
# Bypass Varnish for comparison
ab -n 100 -c 10 http://127.0.0.1:8080/

# Compare with Varnish results
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

To return to direct Nginx:

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

## Varnish vs Redis FPC

| Feature | Varnish | Redis FPC |
|---------|---------|-----------|
| Performance | Fastest | Fast |
| Memory | Dedicated | Shared |
| ESI Support | Yes | No |
| HTTP Standards | Full | N/A |
| Complexity | Higher | Lower |

For development, Redis FPC is usually sufficient. Use Varnish for production-like testing.

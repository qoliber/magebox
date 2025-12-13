# Varnish

Varnish is a high-performance HTTP cache that significantly improves Magento storefront performance.

## Configuration

Enable Varnish in your `.magebox.yaml` file:

```yaml
services:
  varnish: true
```

## Connection Details

| Property | Value |
|----------|-------|
| Varnish Port | 6081 |
| Backend Port | 80 (Nginx) |

## Architecture

With Varnish enabled:

```
Browser → Varnish (6081) → Nginx (80) → PHP-FPM
              ↓
         Cache Hit?
              ↓
         Return cached response
```

## Magento Configuration

### Enable Full Page Cache

```bash
magebox cli config:set system/full_page_cache/caching_application 2
```

### Generate VCL

Export Magento's VCL configuration:

```bash
magebox cli varnish:vcl:generate --export-version=6 > varnish.vcl
```

### Access via Varnish

Your site is available at:
- **With Varnish**: http://mystore.test:6081
- **Without Varnish**: https://mystore.test

## Varnish Commands

### Check Status

```bash
magebox varnish status
```

### Purge URL

```bash
magebox varnish purge /category/page.html
```

### Flush All Cache

```bash
magebox varnish flush
```

## Cache Management

### Purge from Magento

Magento automatically purges Varnish when content changes:
- Product save
- Category save
- CMS page save
- Cache flush from admin

### Manual Purge

Using varnishadm:

```bash
docker exec magebox_varnish varnishadm "ban req.url ~ /"
```

### Purge by Tag

```bash
docker exec magebox_varnish varnishadm "ban obj.http.X-Magento-Tags ~ product"
```

## Cache Headers

Magento sends cache headers that Varnish uses:

| Header | Purpose |
|--------|---------|
| `X-Magento-Tags` | Cache tags for targeted purging |
| `Cache-Control` | TTL and cacheability |
| `X-Magento-Cache-Debug` | Debug information |

### Debug Headers

Enable debug headers:

```bash
magebox cli config:set dev/debug/template_hints_storefront 1
```

Check cache status in response headers:
- `X-Magento-Cache-Control: HIT` - Served from cache
- `X-Magento-Cache-Control: MISS` - Fetched from backend

## Development vs Production

### Development

In development, you may want to bypass Varnish for easier debugging:

```yaml
# .magebox.local.yaml
services:
  varnish: false
```

Or access Nginx directly on port 443.

### Production-like Testing

To test with Varnish:

1. Enable Varnish in config
2. Access site via port 6081
3. Check cache hit rates:

```bash
magebox varnish status
```

## Troubleshooting

### Pages Not Caching

1. Check cache application setting:

```bash
magebox cli config:show system/full_page_cache/caching_application
```

Should return `2` (Varnish).

2. Check response headers for `Cache-Control: no-cache`:

```bash
curl -I http://mystore.test:6081/
```

3. Ensure you're not logged in (logged-in users bypass cache)

### Stale Content

If old content appears after updates:

```bash
# Flush Varnish
magebox varnish flush

# Flush Magento FPC
magebox cli cache:flush full_page
```

### Backend Fetch Failed

If Varnish can't reach Nginx:

```bash
# Check Nginx is running
magebox status

# Check Varnish logs
docker logs magebox_varnish
```

### High MISS Rate

If cache hit rate is low:

1. Check TTL settings in Magento
2. Review VCL for correct configuration
3. Check if pages have `no-cache` headers

## Performance Tips

1. **Monitor hit rate** - Aim for >90% hit rate on cacheable pages

2. **Use cache tags** - Leverage Magento's tag-based purging for efficient invalidation

3. **Set appropriate TTLs** - Balance freshness vs performance

4. **Exclude dynamic content** - Use ESI or AJAX for personalized content

5. **Warm the cache** - After deployment, warm critical pages:

```bash
curl -s http://mystore.test:6081/ > /dev/null
curl -s http://mystore.test:6081/category.html > /dev/null
```

## When to Use Varnish

Enable Varnish when:
- Testing production-like caching behavior
- Measuring page load performance
- Debugging cache-related issues
- Load testing

Disable Varnish when:
- Developing features that need immediate feedback
- Debugging PHP code
- Working with logged-in user features
- Minimal resource usage is needed

## VCL Customization

For custom VCL rules, create a custom VCL file:

```vcl
# custom.vcl
vcl 4.0;

include "default.vcl";

sub vcl_recv {
    # Custom rules here
}
```

Mount it in your Docker configuration or use `magebox varnish vcl` commands.

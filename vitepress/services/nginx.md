# Nginx

MageBox uses native Nginx for maximum performance, automatically generating optimized configurations for Magento.

## Overview

Nginx runs directly on your machine (not in Docker) and handles:

- SSL termination with auto-generated certificates
- Static file serving with proper caching headers
- PHP request proxying to PHP-FPM via Unix sockets
- Multi-store routing with `MAGE_RUN_CODE`

## How It Works

### Port Configuration

| Platform | HTTP | HTTPS | Notes |
|----------|------|-------|-------|
| **Linux** | 80 | 443 | Direct binding |
| **macOS** | 8080 | 8443 | Port forwarding via pf (80→8080, 443→8443) |

On macOS, port forwarding is set up during `magebox bootstrap` so you can still access sites at standard ports (80/443).

### Vhost Generation

When you run `magebox start`, MageBox generates nginx vhost configurations:

```
~/.magebox/nginx/vhosts/
├── mystore-mystore.test.conf
├── mystore-api.mystore.test.conf
└── another-another.test.conf
```

Files are named `{project}-{domain}.conf` to support multiple domains per project.

### System Integration

MageBox integrates with your system nginx differently based on platform:

| Platform | Method | Location |
|----------|--------|----------|
| **macOS** | Symlink | `/opt/homebrew/etc/nginx/servers/magebox` |
| **Ubuntu/Debian** | Symlink | `/etc/nginx/sites-enabled/magebox` |
| **Fedora/Arch** | Include directive | Added to `/etc/nginx/nginx.conf` |

## Generated Configuration

Each vhost includes Magento-optimized settings:

```nginx
# Upstream definition (PHP-FPM socket)
upstream fastcgi_backend_mystore {
    server unix:/tmp/magebox/mystore-php8.2.sock;
}

# HTTP server (redirects to HTTPS)
server {
    listen 80;
    server_name mystore.test;

    location / {
        return 301 https://$host$request_uri;
    }
}

# HTTPS server
server {
    listen 443 ssl http2;
    server_name mystore.test;

    ssl_certificate ~/.magebox/certs/mystore.test/cert.pem;
    ssl_certificate_key ~/.magebox/certs/mystore.test/key.pem;

    set $MAGE_ROOT /path/to/project/pub;
    set $MAGE_RUN_CODE default;
    set $MAGE_RUN_TYPE store;

    root $MAGE_ROOT;

    # Static files with caching
    location /static/ {
        expires max;
        # ...
    }

    # Media files
    location /media/ {
        try_files $uri $uri/ /get.php$is_args$args;
        # ...
    }

    # PHP handling
    location ~ \.php$ {
        fastcgi_pass fastcgi_backend_mystore;
        # ...
    }
}
```

### Key Features

- **HTTP/2 enabled** for faster loading
- **Gzip compression** for text-based assets
- **Static file caching** with 1-year expiry
- **Security headers** (X-Frame-Options)
- **Magento error pages** configured

## Custom Configuration

### Adding Custom Rules

Create a custom configuration file alongside the generated one:

```bash
~/.magebox/nginx/vhosts/mystore-mystore.test.conf.custom
```

::: warning
Custom files are not automatically included. You'll need to add an include directive manually or modify the main nginx.conf.
:::

### Store Code Mapping

For complex multi-store setups, create a mapping file:

```nginx
# ~/.magebox/nginx/conf.d/store-mapping.conf
map $http_host $MAGE_RUN_CODE {
    default         default;
    de.mystore.test german;
    fr.mystore.test french;
    b2b.mystore.test b2b;
}
```

## Commands

### Testing Configuration

```bash
# Test nginx configuration syntax
sudo nginx -t

# View full nginx configuration
sudo nginx -T
```

### Reloading

```bash
# Reload after manual changes
sudo nginx -s reload

# Or use MageBox (regenerates configs)
magebox restart
```

### Viewing Logs

MageBox stores per-domain logs for easy debugging:

```bash
# Per-domain logs (recommended)
tail -f ~/.magebox/logs/nginx/mystore.test-access.log
tail -f ~/.magebox/logs/nginx/mystore.test-error.log

# System-wide nginx logs (fallback)
tail -f /var/log/nginx/access.log
tail -f /var/log/nginx/error.log

# macOS (Homebrew)
tail -f /opt/homebrew/var/log/nginx/error.log
```

::: tip Per-Domain Logging
Each domain configured in your project gets its own access and error log in `~/.magebox/logs/nginx/`. This makes it easy to debug issues for specific projects without searching through system-wide logs.
:::

## Troubleshooting

### 502 Bad Gateway

PHP-FPM is not running or socket doesn't exist:

```bash
# Check if socket exists
ls -la /tmp/magebox/*.sock

# Restart the project
magebox restart
```

### 404 Not Found

Check document root configuration:

```yaml
# .magebox.yaml
domains:
  - host: mystore.test
    root: pub  # Should be 'pub' for Magento
```

### SSL Certificate Error

```bash
# Regenerate certificates
magebox ssl generate

# Trust the CA
magebox ssl trust
```

### Configuration Not Loading

```bash
# Check if symlink/include exists
# macOS
ls -la /opt/homebrew/etc/nginx/servers/

# Ubuntu/Debian
ls -la /etc/nginx/sites-enabled/

# Fedora - check nginx.conf for include line
grep magebox /etc/nginx/nginx.conf
```

### Permission Denied on Linux

On Linux, nginx must run as your user to read SSL certificates from `~/.magebox/certs/`:

```bash
# Check nginx user
grep "^user" /etc/nginx/nginx.conf

# Should show your username, not www-data or nginx
# Fix if needed:
sudo sed -i "s/^user .*/user $USER;/" /etc/nginx/nginx.conf
sudo systemctl restart nginx
```

## Performance Tips

### Enable HTTP/2

HTTP/2 is enabled by default in MageBox-generated configs. Verify:

```bash
curl -I --http2 https://mystore.test/
```

### Static File Optimization

Generated configs already include:

- 1-year cache for versioned static files
- Gzip compression for text assets
- Direct file serving (bypasses PHP)

### Monitoring

```bash
# Check nginx status
systemctl status nginx  # Linux
brew services list | grep nginx  # macOS

# View active connections
curl http://localhost/nginx_status  # If status module enabled
```

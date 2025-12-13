# Architecture

Understanding how MageBox works under the hood.

## Overview

MageBox manages a hybrid environment where performance-critical components run natively while supporting services run in Docker containers.

### Default Flow (without Varnish)

```
┌───────────────────────────────────────────────────────────────────────┐
│                           Your Machine                                │
├───────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐                        │
│  │ Browser  │───▶│  Nginx   │───▶│ PHP-FPM  │                        │
│  │          │    │ (native) │    │ (native) │                        │
│  └──────────┘    └──────────┘    └────┬─────┘                        │
│                                       │                               │
│                       ┌───────────────┼─────────────────────┐        │
│                       │    Docker     │                     │        │
│                       │  ┌────────────▼──────────────────┐  │        │
│                       │  │  MySQL / MariaDB              │  │        │
│                       │  ├───────────────────────────────┤  │        │
│                       │  │  Redis                        │  │        │
│                       │  ├───────────────────────────────┤  │        │
│                       │  │  OpenSearch / Elasticsearch   │  │        │
│                       │  ├───────────────────────────────┤  │        │
│                       │  │  RabbitMQ                     │  │        │
│                       │  ├───────────────────────────────┤  │        │
│                       │  │  Mailpit                      │  │        │
│                       │  └───────────────────────────────┘  │        │
│                       └─────────────────────────────────────┘        │
│                                                                       │
└───────────────────────────────────────────────────────────────────────┘
```

### With Varnish (optional)

When Varnish is enabled, Nginx terminates SSL and proxies to Varnish. On cache miss, Varnish proxies back to Nginx backend:

```
┌────────────────────────────────────────────────────────────────────────────┐
│                              Your Machine                                  │
├────────────────────────────────────────────────────────────────────────────┤
│                                                                            │
│  ┌────────┐   ┌─────────┐   ┌─────────┐   ┌─────────┐   ┌─────────┐       │
│  │Browser │──▶│  Nginx  │──▶│ Varnish │──▶│  Nginx  │──▶│ PHP-FPM │       │
│  │        │   │  (SSL)  │   │ (Docker)│   │(backend)│   │ (native)│       │
│  └────────┘   └─────────┘   └─────────┘   └─────────┘   └────┬────┘       │
│                                                              │             │
│                                              ┌───────────────┼───────┐    │
│                                              │    Docker     │       │    │
│                                              │  ┌────────────▼────┐  │    │
│                                              │  │ MySQL / Redis   │  │    │
│                                              │  │ OpenSearch etc. │  │    │
│                                              │  └─────────────────┘  │    │
│                                              └───────────────────────┘    │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘
```

**Flow explanation:**
1. **Nginx (SSL)** - Terminates HTTPS, proxies HTTP to Varnish
2. **Varnish** - Returns cached response (HIT) or forwards to backend (MISS)
3. **Nginx (backend)** - Handles PHP routing, proxies to PHP-FPM
4. **PHP-FPM** - Executes Magento code, connects to Docker services

Varnish caches full page responses, serving them directly without hitting PHP - dramatically improving performance for anonymous users.

## Component Breakdown

### Nginx (Native)

Nginx runs directly on your machine, configured to:

- Listen on ports 80/443 (Linux) or 8080/8443 with port forwarding (macOS)
- Route requests to the correct project based on domain
- Proxy PHP requests to the project's PHP-FPM pool via Unix sockets
- Serve static files directly for optimal performance
- Handle SSL termination with auto-generated certificates

**Configuration location**: `~/.magebox/nginx/vhosts/`

Each project gets its own vhost configuration file(s), automatically generated from `.magebox.yaml`. Files are named `{project}-{domain}.conf`.

#### Nginx Include Mechanism

MageBox integrates with your system nginx differently based on platform:

| Platform | Include Method | Location |
|----------|---------------|----------|
| **macOS** | Symlink | `/opt/homebrew/etc/nginx/servers/magebox` → `~/.magebox/nginx/vhosts/` |
| **Ubuntu/Debian** | Symlink | `/etc/nginx/sites-enabled/magebox` → `~/.magebox/nginx/vhosts/` |
| **Fedora/Arch** | Direct include | Added to `/etc/nginx/nginx.conf` |

#### Generated Vhost Structure

Each vhost configuration includes:

```nginx
# Upstream definition (PHP-FPM socket)
upstream fastcgi_backend_mystore {
    server unix:/tmp/magebox/mystore-php8.2.sock;
}

# HTTP server (redirects to HTTPS or serves directly)
server {
    listen 80;  # or 8080 on macOS
    server_name mystore.test;
    # ...
}

# HTTPS server (if SSL enabled)
server {
    listen 443 ssl;  # or 8443 on macOS
    server_name mystore.test;

    ssl_certificate ~/.magebox/certs/mystore.test/cert.pem;
    ssl_certificate_key ~/.magebox/certs/mystore.test/key.pem;

    # Magento-specific configuration
    set $MAGE_ROOT /path/to/project/pub;
    set $MAGE_RUN_CODE default;
    set $MAGE_RUN_TYPE store;
    # ...
}
```

### PHP-FPM (Native)

Each project runs its own PHP-FPM pool with:

- Dedicated Unix socket file
- Project-specific PHP version
- Isolated process group
- Custom environment variables from `env:` section
- PHP INI overrides from `php_ini:` section

**Pool location**: `~/.magebox/php/pools/{project}.conf`

#### Generated Pool Structure

```ini
[mystore]

user = yourusername
group = yourgroup

listen = /tmp/magebox/mystore-php8.2.sock
listen.owner = yourusername
listen.group = yourgroup

pm = dynamic
pm.max_children = 50
pm.start_servers = 5
pm.min_spare_servers = 2
pm.max_spare_servers = 10

; Magento recommended settings
php_value[memory_limit] = 756M
php_value[max_execution_time] = 18000

; OPcache settings
php_admin_value[opcache.enable] = 1
php_admin_value[opcache.memory_consumption] = 512

; Custom environment variables
env[MAGE_MODE] = developer

; Custom PHP INI overrides
php_admin_value[xdebug.mode] = debug
```

This allows different projects to use different PHP versions simultaneously.

### Docker Services

Services run in Docker for isolation and easy versioning:

| Service | Purpose | Why Docker? |
|---------|---------|-------------|
| MySQL/MariaDB | Database | Easy version switching, data isolation |
| Redis | Session/Cache | Stateless, quick startup |
| OpenSearch | Catalog search | Complex setup, consistent versions |
| RabbitMQ | Message queue | Isolated from system |
| Mailpit | Email testing | No system mail config needed |

**Compose files**: `~/.magebox/docker/`

## File Structure

### MageBox Directory (`~/.magebox/`)

All MageBox configuration and generated files are stored in your home directory:

```
~/.magebox/
├── config.yaml              # Global configuration (dns_mode, default_php, etc.)
├── bin/
│   ├── php                  # Smart PHP wrapper script
│   └── composer             # Composer wrapper (uses correct PHP)
├── certs/                   # SSL certificates (mkcert)
│   ├── rootCA.pem           # Root Certificate Authority
│   ├── rootCA-key.pem       # Root CA private key
│   └── {domain}/            # Per-domain certificates
│       ├── cert.pem         # Domain certificate
│       └── key.pem          # Domain private key
├── nginx/
│   └── vhosts/              # Generated Nginx vhost configs
│       ├── mystore-mystore.test.conf
│       ├── mystore-api.mystore.test.conf
│       └── another-another.test.conf
├── php/
│   └── pools/               # Generated PHP-FPM pool configs
│       ├── mystore.conf
│       └── another.conf
├── docker/                  # Docker Compose configuration
│   ├── docker-compose.yml   # Service definitions
│   └── .env                 # Docker environment variables
├── logs/                    # MageBox log files
│   └── php-fpm/             # PHP-FPM error logs per project
│       └── mystore.log
└── data/                    # Persistent data (future use)
```

### Runtime Files

PHP-FPM sockets are stored in a temporary directory for fast communication:

```
/tmp/magebox/
├── mystore-php8.2.sock      # PHP-FPM Unix socket for mystore project
└── another-php8.3.sock      # PHP-FPM Unix socket for another project
```

::: info Socket Naming Convention
Socket files are named `{project}-php{version}.sock` to support multiple projects with different PHP versions running simultaneously.
:::

### Project Files

In your project directory, MageBox only creates configuration files:

```
/path/to/your/project/
├── .magebox.yaml            # Project configuration (committed to git)
├── .magebox.local.yaml      # Local overrides (add to .gitignore)
└── ... (your Magento files)
```

### System Files

MageBox integrates with system services:

| File | Platform | Purpose |
|------|----------|---------|
| `/etc/pf.anchors/com.magebox` | macOS | Port forwarding rules (80→8080, 443→8443) |
| `/Library/LaunchDaemons/com.magebox.portforward.plist` | macOS | Launch daemon for pf rules |
| `/etc/nginx/sites-enabled/magebox` | Ubuntu/Debian | Symlink to vhosts directory |
| `/etc/nginx/nginx.conf` | Fedora/Arch | Include directive added |
| `/etc/systemd/resolved.conf.d/magebox.conf` | Linux | DNS routing for `.test` TLD |
| `/etc/dnsmasq.d/magebox.conf` | Linux | dnsmasq configuration |
| `/etc/resolver/test` | macOS | DNS resolver for `.test` TLD |

## Request Flow

1. **DNS Resolution**: Browser resolves `mystore.test` to `127.0.0.1` (via /etc/hosts or dnsmasq)

2. **Port Forwarding** (macOS only): System pf forwards ports 80→8080 and 443→8443

3. **Nginx Receives Request**: Nginx matches the domain to the correct vhost configuration

4. **Static Files**: If requesting a static file (CSS, JS, images), Nginx serves it directly with caching headers

5. **PHP Processing**: For PHP requests, Nginx proxies to the project's PHP-FPM Unix socket

6. **PHP-FPM Execution**: The correct PHP version processes the request with:
   - Project-specific environment variables (`MAGE_MODE`, `MAGE_RUN_CODE`, etc.)
   - Custom PHP INI settings from `.magebox.yaml`
   - OPcache enabled by default for performance

7. **Service Connections**: PHP connects to Docker services via localhost ports (e.g., `127.0.0.1:33080` for MySQL)

## Configuration Lifecycle

### When Configs Are Generated

MageBox generates configuration files at specific times:

| Event | What Gets Generated |
|-------|---------------------|
| `magebox bootstrap` | Global config, SSL CA, system integrations |
| `magebox start` | Nginx vhost, PHP-FPM pool, SSL certificates, DNS entries |
| `magebox stop` | Configs are removed (cleanup) |
| `magebox ssl generate` | SSL certificates only |
| `magebox domain add` | Updates config, regenerates nginx vhost, SSL cert |

### Regeneration Flow

When you run `magebox start`:

```
.magebox.yaml + .magebox.local.yaml
        │
        ▼
┌───────────────────┐
│  Config Parser    │  Merges project + local config
└─────────┬─────────┘
          │
    ┌─────┴─────┬─────────────┐
    ▼           ▼             ▼
┌───────┐  ┌─────────┐  ┌──────────┐
│ Nginx │  │ PHP-FPM │  │   SSL    │
│ Vhost │  │  Pool   │  │  Certs   │
└───┬───┘  └────┬────┘  └────┬─────┘
    │           │            │
    ▼           ▼            ▼
~/.magebox/   ~/.magebox/   ~/.magebox/
nginx/vhosts/ php/pools/    certs/
```

### Viewing Generated Configs

You can inspect the generated configurations:

```bash
# View nginx vhost
cat ~/.magebox/nginx/vhosts/mystore-mystore.test.conf

# View PHP-FPM pool
cat ~/.magebox/php/pools/mystore.conf

# Test nginx configuration
sudo nginx -t
```

## Project Discovery

MageBox discovers projects by scanning Nginx vhost configurations:

```bash
magebox list
```

This allows you to see all configured projects and their status without maintaining a separate registry.

## Configuration Cascade

MageBox uses a cascading configuration system:

```
Global Config (~/.magebox/config.yaml)
         │
         ▼
Project Config (.magebox.yaml)
         │
         ▼
Local Overrides (.magebox.local.yaml)
```

Each level can override settings from the level above, allowing team-wide defaults with personal customization.

## Service Port Allocation

To avoid conflicts, each database version uses a unique port:

| Service | Port |
|---------|------|
| MySQL 5.7 | 33057 |
| MySQL 8.0 | 33080 |
| MySQL 8.4 | 33084 |
| MariaDB 10.4 | 33104 |
| MariaDB 10.6 | 33106 |
| MariaDB 11.4 | 33114 |
| Redis | 6379 |
| OpenSearch | 9200 |
| RabbitMQ | 5672 / 15672 |
| Mailpit | 1025 / 8025 |
| Varnish | 6081 |

This means you can run multiple database versions simultaneously for different projects.

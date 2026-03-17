# Laravel Support

MageBox supports Laravel projects alongside Magento/MageOS. Laravel projects get a dedicated nginx vhost template optimized for Laravel routing, with the same native PHP-FPM, SSL, and service integration you get with Magento.

::: tip Contributed by
Laravel support was contributed by [Peter Jaap Blaakmeer](https://github.com/peterjaap).
:::

## Quick Start

### 1. Initialize a Laravel project

```bash
cd /path/to/your/laravel-project
mbox init --type laravel
```

This creates a `.magebox.yaml` with `type: laravel` and sets the document root to `public/`.

### 2. Configure your project

Your `.magebox.yaml` will look like:

```yaml
type: laravel
domains:
  - name: myapp.test
php: "8.3"
mysql: "8.0"
```

### 3. Start services

```bash
mbox start
```

MageBox will:
- Generate a Laravel-optimized nginx vhost (with `try_files $uri $uri/ /index.php$is_args$args`)
- Set up SSL certificates for your `.test` domain
- Start PHP-FPM, MySQL, and any other configured services
- Your app is available at `https://myapp.test`

## Configuration

### Project Type

Set the project type in `.magebox.yaml`:

```yaml
type: laravel
```

This tells MageBox to use the Laravel nginx vhost template instead of the Magento one. The key differences:

| Feature | Magento | Laravel |
|---------|---------|---------|
| Document root | `pub/` | `public/` |
| Nginx routing | Magento-specific rewrites | `try_files` to `index.php` |
| Static assets | Magento static content pipeline | Standard file serving |

### Services

Laravel projects can use the same services as Magento:

```yaml
type: laravel
domains:
  - name: myapp.test
php: "8.3"
mysql: "8.0"
redis: true
```

Available services:
- **MySQL** / **MariaDB** - Database
- **Redis** / **Valkey** - Cache, sessions, queues
- **Mailpit** - Email testing
- **OpenSearch** - Full-text search (Laravel Scout)
- **RabbitMQ** - Queue driver

### Database Connection

Update your Laravel `.env` file to connect to MageBox services:

```env
DB_CONNECTION=mysql
DB_HOST=127.0.0.1
DB_PORT=33080
DB_DATABASE=myapp
DB_USERNAME=root
DB_PASSWORD=magebox
```

::: info Port Convention
MageBox uses version-specific ports: MySQL 8.0 = `33080`, MySQL 8.4 = `33084`. See [Service Ports](/reference/ports) for the full list.
:::

### Redis Connection

```env
REDIS_HOST=127.0.0.1
REDIS_PORT=6380
REDIS_PASSWORD=null
```

### Mail (Mailpit)

```env
MAIL_MAILER=smtp
MAIL_HOST=127.0.0.1
MAIL_PORT=1025
```

Access the Mailpit UI at `http://localhost:8025`.

## PHP Wrapper

The MageBox PHP wrapper works with Laravel CLI tools:

```bash
# Uses the correct PHP version from .magebox.yaml
php artisan migrate
php artisan serve   # Not needed - use mbox start instead
php artisan tinker
```

## Custom Nginx Config

Add project-specific nginx snippets in `.magebox/nginx/*.conf`:

```bash
mkdir -p .magebox/nginx
```

Example - add custom headers:

```nginx
# .magebox/nginx/headers.conf
add_header X-Frame-Options "SAMEORIGIN" always;
add_header X-Content-Type-Options "nosniff" always;
```

## Multiple Laravel Projects

Each project gets its own domain and configuration:

```bash
# Project A
cd ~/projects/api-backend
mbox init --type laravel
mbox start

# Project B
cd ~/projects/admin-panel
mbox init --type laravel
mbox start
```

All projects share the same MySQL, Redis/Valkey, and other Docker services but have separate nginx vhosts and PHP-FPM pools.

## Isolated PHP-FPM

For projects that need different PHP settings:

```yaml
type: laravel
php: "8.3"
isolated: true
php_ini:
  opcache.enable: 1
  memory_limit: 512M
```

## Existing Laravel Project

If you have an existing Laravel project:

```bash
cd /path/to/existing-laravel-app
mbox init --type laravel
mbox start

# Create the database
mbox db create myapp

# Run migrations
php artisan migrate
```

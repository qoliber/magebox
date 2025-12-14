# MageBox Roadmap

This document outlines planned features and improvements for MageBox.

## Current Version: 0.10.12

### Completed Features

- PHP version management (8.1, 8.2, 8.3, 8.4)
- Nginx vhost generation with SSL support
- Docker services (MySQL, Redis, Mailpit, OpenSearch, RabbitMQ)
- Composer wrapper with automatic PHP version switching
- PHP wrapper with automatic version switching
- Project initialization and lifecycle management
- Database import/export
- DNS configuration (dnsmasq and /etc/hosts)
- Health check command (`magebox check`)
- Xdebug toggle (`magebox xdebug on/off`)
- Blackfire profiler integration (`magebox blackfire on/off/install/config`)
- Tideways profiler integration (`magebox tideways on/off/install/config`)
- Global profiling credentials storage (`~/.magebox/config.yaml`)
- Database management (`magebox db create/drop/reset`)
- Automatic database creation on `magebox start`

## Planned Features

### Additional Services

#### Elasticsearch Support

- Support for Elasticsearch 7.x and 8.x as alternative to OpenSearch
- Version configuration in .magebox.yaml

#### Varnish Improvements

- Full Varnish integration for production-like caching
- VCL file generation for Magento
- Cache purge integration

### Developer Experience

#### Magento CLI Integration

- `magebox mage` - Run bin/magento commands with correct PHP version
- Command completion and suggestions

#### Log Management

- `magebox logs php` - Tail PHP-FPM logs
- `magebox logs nginx` - Tail Nginx access/error logs
- `magebox logs mysql` - Tail MySQL logs
- Consolidated log viewer

### Multi-Project Support

- Project switching without stopping services
- Shared service management
- Resource optimization

## Version 2.0 - User-Customizable Templates

### Overview

Move all configuration generation to external template files that users can customize.
Templates will use Go's `text/template` engine which supports conditionals, loops, and functions.

### Template Engine Features

Go's `text/template` supports:

```
{{if .HasRedis}}         - Conditionals
{{else if .HasFiles}}    - Else-if branches
{{else}}                 - Else branch
{{end}}                  - End block

{{range .Domains}}       - Loop over arrays/slices
  {{.Host}}              - Access fields
{{end}}

{{with .Services}}       - Scoped context
  {{.MySQL.Version}}
{{end}}

{{eq .A .B}}             - Equality comparison
{{ne .A .B}}             - Not equal
{{and .A .B}}            - Logical AND
{{or .A .B}}             - Logical OR
{{not .A}}               - Logical NOT

{{.Value | printf "%s"}} - Pipelines/filters
```

### Architecture

```
~/.magebox/
├── templates/                    # User-customizable templates (override defaults)
│   ├── env.php.tmpl             # Magento env.php
│   ├── vhost.conf.tmpl          # Nginx virtual host
│   ├── proxy.conf.tmpl          # Nginx reverse proxy
│   ├── pool.conf.tmpl           # PHP-FPM pool config
│   ├── default.vcl.tmpl         # Varnish VCL
│   ├── dnsmasq.conf.tmpl        # dnsmasq config
│   ├── composer.json.tmpl       # Composer project template
│   └── docker-compose.yml.tmpl  # Docker Compose (optional)
└── ...
```

**Behavior:**
1. Check if user has custom template in `~/.magebox/templates/`
2. Fall back to embedded default template if not found
3. Provide `magebox templates init` to copy defaults for customization
4. Provide `magebox templates reset` to restore defaults

### Templates to Create

#### High Priority (Complex String Builders → Templates)

| Template | Source | Lines | Description |
|----------|--------|-------|-------------|
| `env.php.tmpl` | `internal/project/env.go` | 240 | Magento app/etc/env.php with conditionals for Redis, Varnish, Mailpit, MySQL/MariaDB |
| `composer.json.tmpl` | `internal/templates/composer.go` | 134 | Composer project skeleton |

#### Medium Priority (Already Templates, Make Customizable)

| Template | Source | Description |
|----------|--------|-------------|
| `vhost.conf.tmpl` | `internal/nginx/templates/` | Nginx virtual host - already templated |
| `proxy.conf.tmpl` | `internal/nginx/templates/` | Nginx reverse proxy - already templated |
| `pool.conf.tmpl` | `internal/php/templates/` | PHP-FPM pool - already templated |
| `default.vcl.tmpl` | `internal/varnish/templates/` | Varnish VCL - already templated |

#### Low Priority (Simple Configs)

| Template | Source | Lines | Description |
|----------|--------|-------|-------------|
| `dnsmasq.conf.tmpl` | `internal/dns/dnsmasq.go` | 15 | Simple dnsmasq config |
| `hosts-entry.tmpl` | `internal/dns/hosts.go` | 10 | /etc/hosts entry format |

### Example: env.php.tmpl

```php
<?php
return [
    'backend' => [
        'frontName' => '{{.AdminPath}}'
    ],
    'crypt' => [
        'key' => '{{.CryptKey}}'
    ],
{{if .HasRedis}}
    'session' => [
        'save' => 'redis',
        'redis' => [
            'host' => '127.0.0.1',
            'port' => '6379',
            'database' => '{{.RedisSessionDB}}'
        ]
    ],
{{else}}
    'session' => [
        'save' => 'files'
    ],
{{end}}
    'db' => [
        'connection' => [
            'default' => [
                'host' => '{{.DatabaseHost}}:{{.DatabasePort}}',
                'dbname' => '{{.DatabaseName}}',
                'username' => '{{.DatabaseUser}}',
                'password' => '{{.DatabasePassword}}'
            ]
        ]
    ],
{{if .HasVarnish}}
    'http_cache_hosts' => [
        ['host' => '127.0.0.1', 'port' => '6081']
    ],
{{end}}
    'MAGE_MODE' => '{{.MageMode}}'
];
```

### Template Data Structures

```go
// EnvPHPData contains all variables available in env.php.tmpl
type EnvPHPData struct {
    // Project
    ProjectName   string
    MageMode      string
    AdminPath     string
    CryptKey      string
    CacheIDPrefix string

    // Database
    DatabaseHost     string
    DatabasePort     string
    DatabaseName     string
    DatabaseUser     string
    DatabasePassword string

    // Services (for conditionals)
    HasRedis    bool
    HasVarnish  bool
    HasMailpit  bool

    // Redis databases
    RedisSessionDB   string
    RedisCacheDB     string
    RedisPageCacheDB string

    // Mailpit
    MailpitHost string
    MailpitPort string
}
```

### CLI Commands

```bash
# Initialize custom templates directory with defaults
magebox templates init

# Reset a specific template to default
magebox templates reset env.php.tmpl

# Reset all templates to defaults
magebox templates reset --all

# List available templates and their status (default/custom)
magebox templates list

# Validate custom templates
magebox templates validate
```

### Migration Path

1. Create template loader that checks `~/.magebox/templates/` first
2. Convert `env.go` string builders to template file
3. Make existing templates (nginx, php-fpm, varnish) user-overridable
4. Add CLI commands for template management
5. Document template variables and customization

## Contributing

Want to contribute to MageBox? Check out our [Contributing Guide](CONTRIBUTING.md) for more information.

## Feature Requests

Have a feature request? Open an issue on GitHub with the `enhancement` label.

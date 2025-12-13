# Roadmap

This document outlines planned features and improvements for MageBox.

::: tip
For completed features and version history, see the [Changelog](/changelog).
:::

## Planned Features

### Database Management Improvements

- Automatic database creation on `magebox start`

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

```go
{{if .HasRedis}}         // Conditionals
{{else if .HasFiles}}    // Else-if branches
{{else}}                 // Else branch
{{end}}                  // End block

{{range .Domains}}       // Loop over arrays/slices
  {{.Host}}              // Access fields
{{end}}

{{with .Services}}       // Scoped context
  {{.MySQL.Version}}
{{end}}

{{eq .A .B}}             // Equality comparison
{{ne .A .B}}             // Not equal
{{and .A .B}}            // Logical AND
{{or .A .B}}             // Logical OR
{{not .A}}               // Logical NOT

{{.Value | printf "%s"}} // Pipelines/filters
```

### Architecture

```
~/.magebox/
├── templates/                    # User-customizable templates
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

### Planned Templates

#### High Priority

| Template | Description |
|----------|-------------|
| `env.php.tmpl` | Magento app/etc/env.php with conditionals for Redis, Varnish, Mailpit |
| `composer.json.tmpl` | Composer project skeleton |

#### Medium Priority

| Template | Description |
|----------|-------------|
| `vhost.conf.tmpl` | Nginx virtual host (already templated internally) |
| `proxy.conf.tmpl` | Nginx reverse proxy |
| `pool.conf.tmpl` | PHP-FPM pool configuration |
| `default.vcl.tmpl` | Varnish VCL |

#### Low Priority

| Template | Description |
|----------|-------------|
| `dnsmasq.conf.tmpl` | dnsmasq configuration |
| `hosts-entry.tmpl` | /etc/hosts entry format |

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

## Contributing

Want to contribute to MageBox? We welcome contributions! Open an issue or submit a pull request on [GitHub](https://github.com/qoliber/magebox).

## Feature Requests

Have a feature request? Open an issue on GitHub with the `enhancement` label.

# Roadmap

This document outlines planned features and improvements for MageBox.

::: tip
For completed features and version history, see the [Changelog](/changelog).
:::

## Recently Completed

The following features have been implemented:

| Feature | Version | Description |
|---------|---------|-------------|
| Varnish Full Integration | v0.10.5 | Automatic Nginx proxy, VCL generation, cache purge |
| Log Viewer | v0.10.10 | `magebox logs` with split-screen multitail |
| Error Reports | v0.10.10 | `magebox report` with filesystem watching |
| Elasticsearch Support | v0.9.0 | Elasticsearch 7.x and 8.x alongside OpenSearch |
| Blackfire Profiler | v0.10.0 | Full Blackfire integration |
| Tideways Profiler | v0.10.0 | Full Tideways integration |
| Database Management | v0.10.2 | `db create`, `db drop`, `db reset` commands |
| Multi-Domain Support | v0.7.1 | Multiple domains per project with store codes |

## Planned Features

### Service-Specific Log Tailing

Dedicated log commands for each service:

```bash
magebox logs php      # PHP-FPM logs
magebox logs nginx    # Nginx access/error logs
magebox logs mysql    # MySQL query logs
magebox logs redis    # Redis logs
```

### PHP INI Customization

Allow users to customize PHP INI settings via `.magebox.yaml`:

```yaml
# .magebox.yaml
php: "8.2"
php_ini:
  memory_limit: "2G"
  max_execution_time: 3600
  upload_max_filesize: "128M"
```

Settings would apply to both CLI wrapper and FPM pool.

### Database Snapshots

Quick database backup and restore:

```bash
magebox db snapshot create mybackup
magebox db snapshot restore mybackup
magebox db snapshot list
```

### Performance Profiling Dashboard

Web-based performance visualization:

- Query analysis
- Cache hit rates
- Request timing breakdown

### IDE Plugins

- PHPStorm plugin for MageBox integration
- VS Code extension

## Version 2.0 - User-Customizable Templates

### Overview

Move all configuration generation to external template files that users can customize.
Templates will use Go's `text/template` engine which supports conditionals, loops, and functions.

### Template Customization

Currently, templates are embedded in the binary. Version 2.0 will allow users to override them:

```
~/.magebox/
├── templates/                    # User-customizable templates
│   ├── env.php.tmpl             # Magento env.php
│   ├── vhost.conf.tmpl          # Nginx virtual host
│   ├── pool.conf.tmpl           # PHP-FPM pool config
│   ├── default.vcl.tmpl         # Varnish VCL
│   └── docker-compose.yml.tmpl  # Docker Compose
└── ...
```

**Behavior:**
1. Check if user has custom template in `~/.magebox/templates/`
2. Fall back to embedded default template if not found
3. Provide `magebox templates init` to copy defaults for customization
4. Provide `magebox templates reset` to restore defaults

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

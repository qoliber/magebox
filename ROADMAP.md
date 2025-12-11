# MageBox Roadmap

This document outlines planned features and improvements for MageBox.

## Current Version: 0.4.0

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

## Planned Features

### Performance Profiling (v0.5.0)

#### Blackfire Integration

Integrate Blackfire profiler for performance analysis:

- `magebox blackfire on` - Enable Blackfire profiling
- `magebox blackfire off` - Disable Blackfire profiling
- `magebox blackfire profile` - Run a profile session
- Automatic probe installation per PHP version
- Docker agent container for profile collection
- Configuration via environment variables or .magebox.yaml

#### Tideways Integration

Integrate Tideways for application performance monitoring:

- `magebox tideways on` - Enable Tideways monitoring
- `magebox tideways off` - Disable Tideways monitoring
- `magebox tideways status` - Show current status
- Automatic daemon configuration
- Environment variable configuration

### Database Management Improvements (v0.5.0)

- `magebox db create` - Create database from project config
- `magebox db drop` - Drop project database
- `magebox db reset` - Drop and recreate database
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

## Contributing

Want to contribute to MageBox? Check out our [Contributing Guide](CONTRIBUTING.md) for more information.

## Feature Requests

Have a feature request? Open an issue on GitHub with the `enhancement` label.

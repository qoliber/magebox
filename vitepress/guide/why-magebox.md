# Why MageBox?

MageBox takes a hybrid approach to Magento development: native PHP and Nginx for performance-critical operations, Docker for stateless services.

## The MageBox Approach

### Native Where It Matters

PHP-FPM and Nginx run directly on your machine:

```
Host Machine
    │
    └── Your Code ←── PHP-FPM (native)
         (direct access)
```

This means:
- **Zero file sync latency** - Code changes are instant
- **Lower memory footprint** - No PHP container overhead
- **Fast startup** - Projects start in seconds
- **Easy debugging** - Standard tools like Xdebug work natively

### Docker Where It Helps

Database and cache services run in containers:

- **MySQL/MariaDB** - Easy version management
- **Redis** - Isolated and consistent
- **OpenSearch** - No system pollution
- **RabbitMQ** - Standard ports, easy setup

## Key Benefits

### Instant Code Changes
Save a file, refresh the browser. No sync delays, no waiting.

### Easy PHP Switching
Switch PHP versions with one command. Each project can use a different version simultaneously.

```bash
magebox php 8.3  # Switch to PHP 8.3
```

### Low Resource Usage
Running multiple Magento projects uses less memory since PHP runs natively and shares resources efficiently.

### Simple Architecture
Fewer moving parts means fewer things that can break. Native services are straightforward to debug and maintain.

### Better IDE Integration
Xdebug, PHPStan, and other tools work without extra configuration for container networking.

## Feature Overview

| Feature | Description |
|---------|-------------|
| Native PHP/Nginx | Full native speed for PHP and web server |
| Auto SSL certificates | HTTPS with mkcert, works out of the box |
| Multi-domain support | Multiple domains per project with store codes |
| Database management | Import, export, snapshots |
| Redis integration | Session and cache storage |
| OpenSearch/Elasticsearch | Full-text search support |
| Varnish support | HTTP caching when needed |
| RabbitMQ support | Message queue for async operations |
| Email testing | Mailpit for local email capture |
| Project discovery | See all projects with `magebox list` |
| Custom commands | Define project-specific commands |
| Team collaboration | Share configs, repos, and assets |
| Self-updating | `magebox self-update` |

## When to Choose MageBox

MageBox is a great fit if you:

- Want fast iteration during development
- Work on multiple Magento projects
- Prefer native tools and simple debugging
- Use macOS, Linux, or Windows WSL2

## Background

MageBox grew out of an internal tool we maintained at [qoliber](https://qoliber.com) for years. After experiencing file sync frustrations on macOS (particularly with Mutagen), we decided to take a different approach: keep PHP and Nginx native, use Docker only for stateless services.

The result is a tool that's fast, simple, and gets out of your way so you can focus on building.

## Migration Guides

If you're coming from another tool, check out our migration guides:

- [From Warden](/guide/migrating-from-warden)
- [From DDEV](/guide/migrating-from-ddev)
- [From Valet/Valet+](/guide/migrating-from-valet)
- [From Herd](/guide/migrating-from-herd)

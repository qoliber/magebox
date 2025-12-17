# What is MageBox?

MageBox is a modern development environment for Magento 2 and MageOS that combines **native performance** with **Docker convenience**.

## The Hybrid Approach

MageBox runs PHP and Nginx natively on your machine while using Docker for supporting services:

```
Your Machine
├── Nginx (native)          ← Direct file access
├── PHP-FPM (native)        ← Native performance
└── Docker Containers
    ├── MySQL/MariaDB       ← Easy version management
    ├── Redis               ← Isolated cache
    ├── OpenSearch          ← Search engine
    ├── RabbitMQ            ← Message queue
    └── Mailpit             ← Email testing
```

**Native where it matters**: PHP and Nginx run directly on your machine. File changes are instant because there's no synchronization layer.

**Docker where it helps**: Database and cache services run in containers. They're easy to version, isolated from your system, and simple to manage.

## Key Benefits

### Instant Code Changes
Your changes are available immediately. Save a file, refresh the browser - it's that simple.

### Low Resource Usage
PHP runs natively and shares system resources efficiently.

### Fast Startup
Starting a project takes about 2 seconds.

### Easy PHP Switching
Switch PHP versions with one command.

```bash
# Switch to PHP 8.3
magebox php 8.3

# Check current version
magebox php
```

### Multiple Projects
Run multiple Magento projects simultaneously, each with its own PHP version and services.

## Who Should Use MageBox?

MageBox is ideal for:

- Developers who want **fast iteration** during development
- Teams working on **multiple Magento projects** with different requirements
- Developers on macOS, Linux, or Windows WSL2 who want a **native experience**
- Anyone who prefers **simple, straightforward** tooling

## Requirements

- macOS, Linux, or Windows WSL2
- Docker (for services)
- PHP 8.1+ (installed via Homebrew or apt)
- Nginx
- Composer

MageBox helps you install and configure all dependencies with the `magebox bootstrap` command.

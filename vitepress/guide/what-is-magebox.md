# What is MageBox?

MageBox is a modern, fast development environment for Magento 2 and MageOS that prioritizes **native performance** over containerization.

## The Problem with Docker-based Solutions

Traditional Docker-based development environments (Warden, DDEV, etc.) run everything in containers:

- **File sync overhead**: Your code lives on your machine but runs in containers, requiring synchronization
- **Resource intensive**: Multiple containers consume significant memory and CPU
- **Slow startup**: Waiting 30+ seconds to start your environment
- **PHP version changes**: Need to rebuild containers to switch PHP versions

## The MageBox Approach

MageBox takes a hybrid approach:

```
Your Machine
├── Nginx (native)          ← Full native speed
├── PHP-FPM (native)        ← No container overhead
└── Docker Containers
    ├── MySQL/MariaDB       ← Stateless, easy to manage
    ├── Redis               ← Quick container startup
    ├── OpenSearch          ← Isolated from system
    ├── RabbitMQ            ← Standard ports
    └── Mailpit             ← Email testing
```

**Native where it matters**: PHP and Nginx run directly on your machine. File changes are instant because there's no synchronization layer.

**Docker where it helps**: Database and cache services run in containers. They're stateless, easy to version, and don't pollute your system.

## Key Benefits

### Instant Code Changes
No file sync means your changes are available immediately. Save a file, refresh the browser - it's that simple.

### Low Resource Usage
Without PHP running in containers, you use significantly less memory. Great for machines running multiple projects.

### Fast Startup
Starting a project takes about 2 seconds. Stop waiting, start coding.

### Easy PHP Switching
Switch PHP versions with one command. No rebuilding, no waiting.

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

- Developers who want **maximum performance** during development
- Teams working on **multiple Magento projects** with different requirements
- Anyone frustrated with **slow Docker-based** development environments
- Developers on macOS, Linux, or Windows WSL2 who want a **native experience**

## Requirements

- macOS, Linux, or Windows WSL2
- Docker (for services)
- PHP 8.1+ (installed via Homebrew or apt)
- Nginx
- Composer

MageBox helps you install and configure all dependencies with the `magebox bootstrap` command.

# What is MageBox?

MageBox is a development environment for Magento 2 and MageOS. It runs PHP and Nginx natively on your machine while using Docker for supporting services like databases and caches.

## How It Works

```
Your Machine
├── Nginx (native)
├── PHP-FPM (native)
└── Docker
    ├── MySQL/MariaDB
    ├── Redis
    ├── OpenSearch
    ├── RabbitMQ
    └── Mailpit
```

PHP and Nginx run directly on your machine for direct file access. Database and cache services run in Docker for easy version management.

## Getting Started

```bash
# Install
brew install qoliber/magebox/magebox

# First-time setup
magebox bootstrap

# Create a new project
magebox new mystore

# Or use an existing project
cd /path/to/magento
magebox init
magebox start
```

## Requirements

- macOS, Linux, or Windows WSL2
- Docker
- PHP 8.1+
- Nginx

The `magebox bootstrap` command helps install and configure all dependencies.

## Background

MageBox started as an internal tool at [qoliber](https://qoliber.com). We wanted something simple: native PHP for direct file access, Docker for databases. After using it for years, we open-sourced it.

## Next Steps

- [Installation](/guide/installation) - Detailed installation options
- [Quick Start](/guide/quick-start) - Get a project running
- [Architecture](/guide/architecture) - Technical deep dive

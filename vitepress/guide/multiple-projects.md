# Multiple Projects

MageBox supports running multiple Magento projects simultaneously.

## How It Works

Each project has its own:
- PHP-FPM pool (with project-specific PHP version)
- Nginx vhost configuration
- Docker services (shared by port)
- SSL certificates

```
Project A (PHP 8.2)     Project B (PHP 8.3)     Project C (PHP 8.4)
     │                       │                       │
     └───────────────────────┴───────────────────────┘
                             │
                    Shared Docker Services
                    (MySQL, Redis, etc.)
```

## Setup

### Project A

```bash
cd /path/to/store-a
magebox init storea
magebox start
```

```yaml
# .magebox.yaml
name: storea
domains:
  - host: storea.test
php: "8.2"
services:
  mysql: "8.0"
```

### Project B

```bash
cd /path/to/store-b
magebox init storeb
magebox start
```

```yaml
# .magebox.yaml
name: storeb
domains:
  - host: storeb.test
php: "8.3"
services:
  mysql: "8.0"  # Same MySQL version, shares container
```

### Project C

```bash
cd /path/to/store-c
magebox init storec
magebox start
```

```yaml
# .magebox.yaml
name: storec
domains:
  - host: storec.test
php: "8.4"
services:
  mariadb: "10.6"  # Different DB, different container
```

## Listing Projects

View all MageBox projects:

```bash
magebox list
```

Output:

```
MageBox Projects
================

Name      Domain          PHP    Status
----      ------          ---    ------
storea    storea.test     8.2    running
storeb    storeb.test     8.3    running
storec    storec.test     8.4    stopped
```

## Global Status

Check all services and projects:

```bash
magebox global status
```

Output:

```
Global Services
===============
MySQL 8.0      running (port 33080)
MariaDB 10.6   running (port 33106)
Redis          running (port 6379)
Mailpit        running (port 8025)

Projects
========
storea    running    https://storea.test
storeb    running    https://storeb.test
storec    stopped    https://storec.test
```

## Managing Projects

### Start All

```bash
magebox global start
```

### Stop All

```bash
magebox global stop
```

### Start Specific Project

```bash
cd /path/to/storea
magebox start
```

### Stop Specific Project

```bash
cd /path/to/storea
magebox stop
```

## Database Isolation

Each project uses its own database:

| Project | Database Name |
|---------|---------------|
| storea | storea |
| storeb | storeb |
| storec | storec |

All databases on the same MySQL version share the same container but have separate databases.

### Connecting to Specific Database

```bash
cd /path/to/storea
magebox db shell
# Connected to 'storea' database
```

## PHP Version Isolation

Each project can use a different PHP version:

```
storea → PHP 8.2 → php82-fpm pool
storeb → PHP 8.3 → php83-fpm pool
storec → PHP 8.4 → php84-fpm pool
```

PHP-FPM pools are isolated, so different projects don't affect each other.

## Port Allocation

Services use consistent ports:

| Service | Port | Shared By |
|---------|------|-----------|
| MySQL 8.0 | 33080 | storea, storeb |
| MariaDB 10.6 | 33106 | storec |
| Redis | 6379 | All projects |
| Mailpit | 8025 | All projects |

## Resource Management

### Memory Usage

Running multiple projects increases resource usage:

- Each PHP-FPM pool: ~50-200MB per project
- MySQL container: ~500MB-1GB
- Redis container: ~50MB
- OpenSearch: ~500MB-1GB

### Reducing Resources

1. Stop unused projects:

```bash
cd /path/to/unused-project
magebox stop
```

2. Disable unused services per project:

```yaml
# .magebox.local.yaml
services:
  opensearch: false
  rabbitmq: false
```

3. Stop global services when not developing:

```bash
magebox global stop
```

## Switching Projects

### Terminal Workflow

```bash
# Work on Project A
cd /path/to/storea
magebox shell
# Make changes...

# Switch to Project B
cd /path/to/storeb
magebox shell
# Make changes...
```

### IDE Configuration

Configure your IDE to use the correct PHP version per project:

**PHPStorm:**
1. Settings → PHP
2. Set CLI Interpreter per project
3. Point to the correct PHP version

## Common Patterns

### Client Projects

```
~/projects/
├── client-a/
│   └── .magebox.yaml (PHP 8.2, MySQL 8.0)
├── client-b/
│   └── .magebox.yaml (PHP 8.3, MySQL 8.0)
└── client-c/
    └── .magebox.yaml (PHP 8.1, MariaDB 10.6)
```

### Version Testing

```
~/magento-versions/
├── magento-243/
│   └── .magebox.yaml (PHP 8.1, MySQL 8.0)
├── magento-246/
│   └── .magebox.yaml (PHP 8.2, MySQL 8.0)
└── magento-247/
    └── .magebox.yaml (PHP 8.3, MySQL 8.4)
```

### Module Development

```
~/modules/
├── my-module/
│   ├── magento-instance/
│   │   └── .magebox.yaml
│   └── src/  # Symlinked into Magento
```

## Troubleshooting

### Port Conflicts

If two projects try to use the same port:

```bash
magebox global status
# Check for conflicting services
```

### Domain Conflicts

Each project needs unique domains:

```yaml
# Project A
domains:
  - host: storea.test

# Project B - DON'T use storea.test
domains:
  - host: storeb.test
```

### PHP Version Unavailable

If a project needs a PHP version you don't have:

```bash
# macOS
brew install php@8.4

# Linux
sudo apt install php8.4-fpm
```

Then restart the project:

```bash
magebox stop
magebox start
```

# Quick Reference

Your go-to cheatsheet for MageBox commands. Print it, bookmark it, love it.

## Essential Commands

```bash
# First-time setup
magebox bootstrap          # Set up your system (one-time)

# Project lifecycle
magebox init               # Initialize project in current directory
magebox start              # Start all services for project
magebox stop               # Stop all services
magebox restart            # Restart services
magebox status             # Check what's running

# Start/stop all projects
magebox start --all        # Start all MageBox projects
magebox stop --all         # Stop all projects
```

## Daily Workflow

```bash
# Database
magebox db shell           # MySQL shell access
magebox db import dump.sql # Import database
magebox db export          # Export to backup.sql
magebox db snapshot create # Quick snapshot
magebox db snapshot restore <name>

# Domains & SSL
magebox domain add api.mystore.test
magebox domain list
magebox ssl generate       # Regenerate certificates

# Logs
magebox logs nginx         # Nginx logs
magebox logs php           # PHP-FPM logs
magebox logs mysql         # MySQL logs
```

## Debugging & Profiling

```bash
# Xdebug (step debugging)
magebox xdebug on          # Enable
magebox xdebug off         # Disable (faster performance)
magebox xdebug status

# Blackfire (profiling)
magebox blackfire on
magebox blackfire off

# Tideways (APM)
magebox tideways on
magebox tideways off

# Quick mode switching
magebox dev                # Dev mode: Xdebug ON, OPcache OFF
magebox prod               # Prod mode: Xdebug OFF, OPcache ON
```

## Testing & Code Quality

```bash
# Run tests
magebox test unit          # PHPUnit unit tests
magebox test integration   # Integration tests (with tmpfs DB)
magebox test phpstan       # Static analysis
magebox test phpcs         # Code standards
magebox test all           # Run everything

# Setup testing tools
magebox test setup         # Interactive installer
magebox test status        # Check installed tools
```

## Team Collaboration

```bash
# Setup team
magebox team add myteam    # Configure team (interactive)
magebox team list          # List configured teams

# Clone projects
magebox clone myproject              # Clone repo + composer install
magebox clone myproject --fetch      # Clone + DB + media

# Fetch assets (from project directory)
magebox fetch              # Download & import DB
magebox fetch --media      # Also download media

# Sync existing project
magebox sync               # Update DB and media
magebox sync --db          # Database only
magebox sync --media       # Media only
```

## PHP Management

```bash
# Switch PHP version
magebox php 8.3            # Use PHP 8.3 for current project

# PHP wrapper (auto-detects project version)
php bin/magento            # Uses project's PHP version
composer install           # Uses project's PHP version
```

## Service Management

```bash
# Docker services
magebox docker up          # Start Docker services
magebox docker down        # Stop Docker services
magebox docker status      # Check container status

# Redis
magebox redis flush        # Clear all Redis data
magebox redis cli          # Redis CLI access

# Varnish
magebox varnish on         # Enable Varnish caching
magebox varnish off        # Bypass Varnish
magebox varnish purge      # Clear Varnish cache

# Queue (RabbitMQ)
magebox queue status       # View queue status
magebox queue flush        # Purge all queues
magebox queue consumer     # Run queue consumers
```

## Configuration

```bash
# Global settings
magebox config list        # Show all settings
magebox config set dns_mode dnsmasq
magebox config get default_php

# DNS
magebox dns status         # Check DNS configuration
```

## Keyboard Shortcuts (CLI)

| Key | Action |
|-----|--------|
| `Tab` | Autocomplete commands |
| `↑/↓` | Navigate command history |
| `Ctrl+C` | Cancel current operation |
| `Ctrl+R` | Search command history |

## Default Credentials

| Service | Host | Port | User | Password |
|---------|------|------|------|----------|
| MySQL 8.0 | localhost | 33080 | root | magebox |
| MariaDB 10.6 | localhost | 33106 | root | magebox |
| Redis | localhost | 6379 | - | - |
| RabbitMQ | localhost | 15672 | guest | guest |
| Mailpit | localhost | 8025 | - | - |

## File Locations

```bash
~/.magebox/
├── config.yaml            # Global config
├── bin/                   # PHP/Composer wrappers
├── certs/                 # SSL certificates
├── nginx/vhosts/          # Nginx configs
├── php/pools/             # PHP-FPM pools
├── docker/                # Docker compose files
├── logs/                  # Service logs
└── snapshots/             # Database snapshots

# Project files
.magebox.yaml              # Project config (commit this)
.magebox.local.yaml        # Local overrides (gitignore this)
```

## Quick Fixes

```bash
# Port forwarding not working (macOS)
magebox bootstrap          # Re-run to fix

# Nginx config error
nginx -t                   # Test config
sudo nginx -s reload       # Reload

# PHP-FPM not responding
brew services restart php@8.3  # macOS
sudo systemctl restart php8.3-fpm  # Linux

# SSL certificate issues
magebox ssl trust          # Re-trust CA
magebox ssl generate       # Regenerate certs

# Clear everything and start fresh
magebox stop
magebox start
```

## Getting Help

```bash
magebox --help             # General help
magebox <command> --help   # Command-specific help
magebox --version          # Check version
magebox check              # System health check
magebox report             # Generate debug report
```

---

::: tip Pro Tip
Add `alias mbox="magebox"` to your shell config for even faster access!
:::

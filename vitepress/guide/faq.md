# Frequently Asked Questions

Quick answers to common questions about MageBox.

## General

### What is MageBox?

MageBox is a fast, native Magento 2 development environment. Unlike Docker-based solutions that containerize everything, MageBox runs PHP and Nginx natively on your machine for maximum performance, while using Docker only for supporting services (MySQL, Redis, OpenSearch).

### How is MageBox different from Warden, DDEV, or Valet?

| Feature | MageBox | Warden | DDEV | Valet |
|---------|---------|--------|------|-------|
| PHP/Nginx | Native | Docker | Docker | Native |
| Performance | Fastest | Good | Good | Fast |
| Setup time | ~5 min | ~15 min | ~15 min | ~10 min |
| Multi-project | Yes | Yes | Yes | Yes |
| Team sync | Built-in | No | No | No |
| Magento-specific | Yes | Yes | No | No |

MageBox gives you the speed of native execution with the convenience of Docker for databases.

### Is MageBox free?

Yes! MageBox is free and open source under the MIT license.

### Which platforms are supported?

- **macOS**: 12 (Monterey), 13 (Ventura), 14 (Sonoma), 15 (Sequoia)
- **Linux**: Ubuntu 20.04/22.04/24.04, Fedora 38-43, Debian 11/12, Arch Linux
- **Windows**: Via WSL2 (Ubuntu recommended)

### Which PHP versions are supported?

PHP 8.1, 8.2, 8.3, and 8.4. Each project can use a different PHP version simultaneously.

---

## Installation

### Do I need Docker installed?

Yes, Docker is required for database, Redis, OpenSearch, and other supporting services. MageBox supports Docker Desktop, OrbStack, Colima, and Rancher Desktop.

### Why does bootstrap need sudo?

Bootstrap needs elevated permissions (one time only) to:
- Set up port forwarding rules (macOS)
- Configure DNS resolution
- Trust SSL certificates
- Modify nginx configuration

After bootstrap, daily operations don't require sudo.

### Can I install without Homebrew on macOS?

Homebrew is the recommended method for macOS as it manages PHP versions and dependencies. Manual installation is possible but not recommended.

---

## Performance

### Why is MageBox faster than Docker-based solutions?

Docker adds virtualization overhead, especially on macOS where it runs in a VM. MageBox runs PHP and Nginx directly on your machine, eliminating:
- File synchronization delays
- Network overhead between containers
- Memory overhead of container runtimes

Typical improvement: **2-5x faster** page loads compared to full Docker setups.

### How do I maximize performance?

1. **Disable Xdebug** when not debugging: `magebox xdebug off`
2. **Use OPcache** (enabled by default)
3. **Use Redis** for sessions and cache
4. **Enable Varnish** for full-page caching

Quick mode switch:
```bash
magebox prod   # Production mode: fast
magebox dev    # Development mode: debugging enabled
```

### My site is still slow. What should I check?

1. Run `magebox status` to verify services are running
2. Check if Xdebug is enabled: `magebox xdebug status`
3. Verify Redis is being used (not file cache)
4. Profile with Blackfire: `magebox blackfire on`

---

## Domains & SSL

### Why do my domains end in .test?

The `.test` TLD is reserved for local development and will never conflict with real websites. MageBox uses dnsmasq to resolve `*.test` to localhost.

### Can I use a different TLD?

Yes, but `.test` is recommended. You can add any domain to `/etc/hosts` or configure dnsmasq for other TLDs.

### Why do I get SSL certificate warnings?

Your browser doesn't trust the local Certificate Authority yet. Run:
```bash
magebox ssl trust
```

For Firefox, you may need to import the CA manually (Firefox uses its own certificate store).

### How do I add multiple domains to one project?

Edit `.magebox.yaml`:
```yaml
domains:
  - name: mystore.test
    ssl: true
  - name: api.mystore.test
    ssl: true
  - name: admin.mystore.test
    ssl: true
```

Then restart: `magebox restart`

---

## Database

### What's the default MySQL password?

- **User**: `root`
- **Password**: `magebox`
- **Port**: `33080` (MySQL 8.0), `33106` (MariaDB 10.6), etc.

### Can I run multiple database versions?

Yes! Each version runs on a different port. Configure in `.magebox.yaml`:
```yaml
services:
  mysql:
    version: "8.0"    # Port 33080
    # version: "8.4"  # Port 33084
  # mariadb:
  #   version: "10.6" # Port 33106
```

### How do I import a large database?

```bash
# Direct import (fastest)
magebox db import dump.sql

# With progress for large files
pv dump.sql | magebox db shell

# From gzip
gunzip -c dump.sql.gz | magebox db shell
```

### How do I connect with a GUI client?

Use these settings in TablePlus, Sequel Pro, DBeaver, etc.:
- **Host**: `127.0.0.1`
- **Port**: `33080` (or your configured port)
- **User**: `root`
- **Password**: `magebox`

---

## PHP

### How does PHP version switching work?

MageBox uses a smart PHP wrapper at `~/.magebox/bin/php` that reads your project's `.magebox.yaml` and uses the correct PHP version automatically.

### Can different projects use different PHP versions?

Yes! Each project has its own PHP-FPM pool running its configured version. You can have Project A on PHP 8.2 and Project B on PHP 8.3 simultaneously.

### Where are PHP logs?

```bash
# View PHP-FPM logs
magebox logs php

# Or check directly
tail -f ~/.magebox/logs/php-fpm/*.log
```

---

## Team Features

### What is team sync?

Team sync allows you to share project configurations, databases, and media files with your team. When a colleague runs `magebox clone myproject`, they get:
- Git repository cloned
- `.magebox.yaml` created if not present
- `composer install` executed

Then running `magebox fetch` from the project directory downloads and imports the database.

### Do we need a server for team features?

For basic team sync (fetch/sync), you need:
- A Git provider (GitHub, GitLab, Bitbucket)
- SFTP/FTP server for database dumps and media

For advanced features (centralized user management, SSH key distribution), you can run MageBox Team Server.

### Is Team Server required?

No, Team Server is optional. Basic team collaboration works with just Git + SFTP. Team Server adds centralized access management for larger teams.

---

## macOS Specific

### What is port forwarding and why do I need it?

On macOS, only root can bind to ports below 1024 (like 80 and 443). Port forwarding lets Nginx run as your user on ports 8080/8443, while the system transparently forwards from 80/443.

### Port forwarding stops working after sleep/restart

This was fixed in v1.0.2! Run `magebox bootstrap` to upgrade your LaunchDaemon with the new sleep/wake recovery mechanism.

### I'm using Little Snitch/firewall software

Some firewalls reset pf rules. MageBox's LaunchDaemon automatically restores rules every 30 seconds. If issues persist, whitelist `/etc/pf.anchors/com.magebox`.

---

## Linux Specific

### Do I need to run nginx as root?

No, MageBox configures nginx to run as your user so it can access SSL certificates in your home directory.

### SELinux is blocking connections (Fedora)

Run `magebox bootstrap` to configure SELinux contexts, or manually:
```bash
sudo setsebool -P httpd_can_network_connect on
sudo setsebool -P httpd_read_user_content on
sudo chcon -R -t httpd_config_t ~/.magebox/nginx/
sudo chcon -R -t httpd_config_t ~/.magebox/certs/
```

---

## Troubleshooting

### "Connection refused" errors

1. Check services are running: `magebox status`
2. Verify Docker is running
3. Check the specific service: `magebox logs mysql`

### "502 Bad Gateway" from nginx

PHP-FPM isn't running or crashed:
```bash
magebox logs php         # Check for errors
magebox restart          # Restart services
```

### Changes not appearing

Clear caches:
```bash
php bin/magento cache:flush
magebox redis flush      # If using Redis
```

### DNS not resolving (.test domains)

```bash
magebox dns status       # Check configuration
ping mystore.test        # Test resolution

# If using hosts mode, verify entry exists:
cat /etc/hosts | grep mystore
```

### How do I report a bug?

1. Run `magebox report` to generate debug info
2. Open an issue at [github.com/qoliber/magebox](https://github.com/qoliber/magebox/issues)
3. Include the report output and steps to reproduce

---

## Updates

### How do I update MageBox?

```bash
# Via Homebrew (recommended)
brew upgrade qoliber/magebox/magebox

# Or self-update
magebox selfupdate
```

### Will updates break my projects?

MageBox maintains backward compatibility. Your `.magebox.yaml` files will continue to work. After major updates, run `magebox bootstrap` to ensure system configuration is current.

---

::: tip Question not answered?
Open a discussion on [GitHub](https://github.com/qoliber/magebox/discussions) or check the [Troubleshooting guide](/guide/troubleshooting).
:::

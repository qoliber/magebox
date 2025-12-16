# Bootstrap

The `magebox bootstrap` command performs first-time setup of your development environment.

## Platform-Specific Installers

MageBox v0.9.0 introduced a new **platform-specific installer architecture** that provides clean abstraction for different operating systems:

| Platform | Package Manager | PHP Repository |
|----------|----------------|----------------|
| macOS | Homebrew | `shivammathur/php` tap |
| Ubuntu/Debian | apt | Ondrej PPA |
| Fedora/RHEL | dnf | Remi Repository |
| Arch Linux | pacman | Official repos |

### Supported OS Versions

MageBox validates your OS version during bootstrap:

**macOS:**
- 12 (Monterey), 13 (Ventura), 14 (Sonoma), 15 (Sequoia)

**Linux:**
- Fedora: 38, 39, 40, 41, 42
- Ubuntu: 20.04, 22.04, 24.04 (LTS versions)
- Debian: 11 (Bullseye), 12 (Bookworm)
- Arch: Rolling release

::: tip Unsupported Versions
Other OS versions may still work but are not officially tested. Bootstrap will display a warning and continue.
:::

For details on extending MageBox for new distributions, see [Linux Installers](/guide/linux-installers).

## What Bootstrap Does

Running `magebox bootstrap` will:

1. **Check Dependencies**
   - Verify Docker is installed and running
   - Check for Nginx installation
   - Detect installed PHP versions
   - Verify mkcert is available

2. **Create Global Configuration**
   - Initialize `~/.magebox/config.yaml`
   - Set up directory structure
   - Configure default settings

3. **Setup SSL Certificate Authority**
   - Install mkcert root CA
   - Trust the CA in your system keychain
   - Enable HTTPS for all projects

4. **Port Forwarding (macOS)**
   - Set up pf (packet filter) rules
   - Forward port 80 → 8080
   - Forward port 443 → 8443
   - Install LaunchDaemon for persistence

5. **Configure Nginx**
   - Create MageBox vhosts directory
   - Include MageBox configs in Nginx
   - Configure Nginx to listen on 8080/8443 (macOS) or 80/443 (Linux)
   - **Linux**: Configure nginx to run as your user (required for SSL cert access)
   - **Fedora**: Configure SELinux contexts and booleans for nginx
   - Test and reload Nginx configuration

6. **Start Docker Services**
   - Pull required Docker images
   - Start database containers
   - Start Redis, Mailpit, etc.

7. **Configure DNS (dnsmasq by default)**
   - Install and configure dnsmasq for wildcard `*.test` DNS
   - **Linux**: Configure dnsmasq + systemd-resolved for `.test` wildcard DNS
   - **macOS**: Create `/etc/resolver/test` for wildcard resolution
   - Falls back to `/etc/hosts` mode if dnsmasq setup fails

8. **Install PHP Wrapper**
   - Create smart PHP wrapper at `~/.magebox/bin/php`
   - Automatic PHP version detection per project

9. **PHP-FPM Setup (Linux)**
   - Enable and start PHP-FPM services for installed versions
   - Uses default repository logging paths (no config modifications)

10. **Sudoers Config (Linux)**
    - Set up passwordless sudo for nginx/php-fpm control
    - Enables seamless service management without password prompts

## Running Bootstrap

```bash
magebox bootstrap
```

You may be prompted for your password (sudo) **once** to:
- Install port forwarding rules (macOS)
- Modify Nginx configuration
- Update /etc/hosts
- Trust SSL certificates

::: tip No Sudo After Bootstrap
After bootstrap completes, all daily commands run without sudo. This is achieved through port forwarding on macOS.
:::

## Bootstrap Output

```
MageBox Bootstrap
=================

Checking dependencies...
  ✓ Docker installed and running
  ✓ Nginx installed
  ✓ PHP 8.1, 8.2, 8.3, 8.4 detected
  ✓ mkcert installed

Creating configuration...
  ✓ Created ~/.magebox/config.yaml
  ✓ Created directory structure

Setting up SSL...
  ✓ Root CA installed
  ✓ CA trusted in system keychain

Setting up port forwarding (macOS)...
  ✓ Created pf rules at /etc/pf.anchors/com.magebox
  ✓ Installed LaunchDaemon
  ✓ Port forwarding active (80→8080, 443→8443)

Configuring Nginx...
  ✓ Created vhosts directory
  ✓ Added include to nginx.conf
  ✓ Nginx configuration valid
  ✓ Nginx reloaded

Starting Docker services...
  ✓ MySQL 8.0 started
  ✓ Redis started
  ✓ Mailpit started

Configuring DNS...
  Installing dnsmasq... done
  Configuring dnsmasq for *.test domains... done
  Set dns_mode: dnsmasq ✓

Installing PHP wrapper...
  ✓ Created ~/.magebox/bin/php

Bootstrap complete!

Add this to your ~/.zshrc or ~/.bashrc:
  export PATH="$HOME/.magebox/bin:$PATH"

Then run 'magebox init' in your project directory to get started.
```

## Port Forwarding (macOS)

### Why Port Forwarding?

On macOS, MageBox uses **packet filter (pf)** to enable services to run as your regular user:

- **Problem**: Web servers need ports 80/443, but only root can bind to ports below 1024
- **Solution**: Nginx runs on 8080/8443, pf transparently forwards from 80/443
- **Result**: Access sites at clean URLs like `https://mystore.test` without sudo

### How It Works

```
Browser → Port 443 → pf → Port 8443 → Nginx (your user)
```

### What Gets Installed

**1. PF Rules File** (`/etc/pf.anchors/com.magebox`):
```
rdr pass on lo0 inet proto tcp from any to any port 80 -> 127.0.0.1 port 8080
rdr pass on lo0 inet proto tcp from any to any port 443 -> 127.0.0.1 port 8443
```

**2. LaunchDaemon** (`/Library/LaunchDaemons/com.magebox.portforward.plist`):
- Loads pf rules automatically on boot
- Runs as root (required for pf)
- Enables forwarding transparently

### Verification

```bash
# Check if LaunchDaemon exists
ls -la /Library/LaunchDaemons/com.magebox.portforward.plist

# Check if pf rules exist
cat /etc/pf.anchors/com.magebox

# Verify it works
curl -I https://mystore.test
```

## Linux Configuration

On Linux, MageBox uses a different approach than macOS for privileged ports and service management.

### Nginx User Configuration

MageBox configures nginx to run as your user so it can access SSL certificates in `~/.magebox/certs`:

```bash
# Bootstrap automatically updates nginx.conf:
user YOUR_USERNAME;
```

### SELinux Configuration (Fedora/RHEL)

Fedora has SELinux enabled by default. Bootstrap automatically configures:

1. **Network connections** - Allow nginx to proxy to Docker containers:
   ```bash
   setsebool -P httpd_can_network_connect on
   ```

2. **Config file access** - Set correct SELinux context on MageBox directories:
   ```bash
   chcon -R -t httpd_config_t ~/.magebox/nginx/
   chcon -R -t httpd_config_t ~/.magebox/certs/
   ```

::: tip SELinux Troubleshooting
If you encounter HTTPS issues after bootstrap, see [Troubleshooting: SELinux Issues](/guide/troubleshooting#selinux-issues-fedora-rhel) for manual fixes.
:::

### PHP-FPM Logging

MageBox uses the default PHP-FPM logging paths provided by each distribution's repository:

| Distribution | Log Path |
|-------------|----------|
| Ubuntu/Debian | `/var/log/php8.X-fpm.log` |
| Fedora/RHEL | `/var/opt/remi/phpXX/log/php-fpm/` |
| Arch Linux | `/var/log/php-fpm.log` |

### Sudoers Configuration

Bootstrap configures passwordless sudo for specific commands:

```bash
# /etc/sudoers.d/magebox
YOUR_USERNAME ALL=(ALL) NOPASSWD: /usr/bin/systemctl reload nginx
YOUR_USERNAME ALL=(ALL) NOPASSWD: /usr/bin/systemctl reload php*-fpm
YOUR_USERNAME ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart nginx
YOUR_USERNAME ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart php*-fpm
```

This allows MageBox to manage services without password prompts during daily operations.

### DNS with systemd-resolved

On modern Linux distributions using systemd-resolved, bootstrap configures:

1. **dnsmasq** on port 5354 (to avoid conflict with systemd-resolved on port 53)
2. **systemd-resolved** to forward `.test` queries to dnsmasq

See [DNS Configuration](/guide/dns) for manual setup options

## DNS Mode Selection

::: tip New in v0.16.6
Dnsmasq is now the **default DNS mode**. Bootstrap automatically installs and configures dnsmasq on all platforms.
:::

### dnsmasq Mode (Default)

Uses dnsmasq for wildcard domain resolution:

- All `*.test` domains resolve automatically
- No hosts file modifications needed
- No sudo prompts during `magebox start/stop`
- Supports unlimited subdomains

```bash
# All *.test domains resolve to 127.0.0.1
curl https://anything.test  # Works automatically
curl https://api.anything.test  # Also works!
```

### hosts Mode (Fallback)

Uses `/etc/hosts` for domain resolution. MageBox automatically falls back to this mode if dnsmasq setup fails:

- Simple and reliable
- Requires adding each domain manually
- MageBox updates hosts file automatically

```
# /etc/hosts
127.0.0.1 mystore.test
127.0.0.1 another.test
```

To manually switch to hosts mode:

```bash
magebox config set dns_mode hosts
```

## PHP Wrapper Setup

After bootstrap, add the PHP wrapper to your PATH:

```bash
# Add to ~/.zshrc or ~/.bashrc
export PATH="$HOME/.magebox/bin:$PATH"

# Reload shell
source ~/.zshrc
```

Verify:

```bash
which php
# Should show: /Users/YOUR_USERNAME/.magebox/bin/php
```

See [PHP Version Wrapper](/guide/php-wrapper) for details.

## Re-running Bootstrap

You can safely re-run bootstrap to:

- Fix configuration issues
- Update after system changes
- Switch DNS modes
- Reinstall PHP wrapper

```bash
magebox bootstrap
```

Existing configuration will be preserved unless you explicitly reset it.

## After Bootstrap

Services running after bootstrap:

| Service | Address | Credentials |
|---------|---------|-------------|
| MySQL 8.0 | localhost:33080 | root / magebox |
| Redis | localhost:6379 | - |
| Mailpit | http://localhost:8025 | - |

## Troubleshooting

### Port Forwarding Not Working (macOS)

1. Verify LaunchDaemon is loaded:
   ```bash
   sudo launchctl list | grep magebox
   ```

2. Reload pf rules manually:
   ```bash
   sudo pfctl -ef /etc/pf.anchors/com.magebox
   ```

3. Check nginx is listening:
   ```bash
   lsof -nP -iTCP:8080 -sTCP:LISTEN
   lsof -nP -iTCP:8443 -sTCP:LISTEN
   ```

### Docker not running

```bash
# macOS
open -a Docker

# Linux
sudo systemctl start docker
```

### Nginx permission denied

```bash
chmod 755 ~/.magebox
chmod 755 ~/.magebox/nginx
chmod 755 ~/.magebox/nginx/vhosts
```

### mkcert CA not trusted

```bash
mkcert -install
```

### Port conflicts

If ports 80 or 443 are in use:

```bash
sudo lsof -i :80
sudo lsof -i :443
```

Stop conflicting services (Apache, other web servers).

### SELinux blocking nginx (Fedora)

If HTTPS doesn't work on Fedora after bootstrap:

```bash
# Re-run SELinux configuration
sudo setsebool -P httpd_can_network_connect on
sudo chcon -R -t httpd_config_t ~/.magebox/nginx/
sudo chcon -R -t httpd_config_t ~/.magebox/certs/
sudo systemctl restart nginx
```

See [Troubleshooting: SELinux Issues](/guide/troubleshooting#selinux-issues-fedora-rhel) for more details.

## Next Steps

After bootstrap completes:

1. Add PHP wrapper to PATH (see above)
2. Navigate to your Magento project
3. Run `magebox init`
4. Run `magebox start`

See [Quick Start](/guide/quick-start) for detailed instructions.

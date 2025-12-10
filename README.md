# MageBox

```
                            _
                           | |
 _ __ ___   __ _  __ _  ___| |__   _____  __
| '_ ` _ \ / _` |/ _` |/ _ \ '_ \ / _ \ \/ /
| | | | | | (_| | (_| |  __/ |_) | (_) >  <
|_| |_| |_|\__,_|\__, |\___|_.__/ \___/_/\_\
                  __/ |
                 |___/  0.2.0
```

A modern, fast development environment for Magento 2. Uses native PHP-FPM, Nginx, and Varnish for maximum performance, with Docker only for stateless services like MySQL, Redis, and OpenSearch.

## Why MageBox?

Unlike Docker-based solutions (Warden, DDEV), MageBox runs PHP and Nginx natively on your machine:

- **No file sync overhead** - Native filesystem access means instant file changes
- **Native performance** - PHP runs at full speed, not inside a container
- **Simple architecture** - Docker only for databases and search engines
- **Multi-project support** - Run multiple Magento projects simultaneously
- **Easy PHP switching** - Change PHP versions per project with one command
- **No sudo required** - After one-time setup, all commands run as your user (macOS uses port forwarding)

---

## Quick Start (For Beginners)

**Want to get started quickly? Just 4 commands:**

```bash
# 1. Install MageBox (Linux)
sudo curl -L https://github.com/qoliber/magebox/releases/latest/download/magebox-linux-amd64 -o /usr/local/bin/magebox && sudo chmod +x /usr/local/bin/magebox

# 2. Set up your environment (one-time)
magebox bootstrap

# 3. Create a new store with sample data (no questions asked!)
magebox new mystore --quick

# 4. Start your store
cd mystore && magebox start
```

Then run the Magento installer:
```bash
magebox cli setup:install \
    --base-url=https://mystore.test \
    --db-host=127.0.0.1:33080 \
    --db-name=mystore \
    --db-user=root \
    --db-password=magebox \
    --search-engine=opensearch \
    --opensearch-host=127.0.0.1 \
    --opensearch-port=9200 \
    --admin-firstname=Admin \
    --admin-lastname=User \
    --admin-email=admin@example.com \
    --admin-user=admin \
    --admin-password=Admin123!
```

**That's it!** Open https://mystore.test in your browser.

> **Note:** The `--quick` flag installs MageOS (no Adobe auth required) with sample data, PHP 8.3, MySQL 8.0, Redis, and OpenSearch - perfect for learning or testing.

---

## Step-by-Step Setup Guide (Detailed)

### Step 1: Install System Dependencies

#### macOS

```bash
# Install Homebrew if not installed
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install PHP versions you need
brew install php@8.1 php@8.2 php@8.3

# Install Nginx
brew install nginx

# Install mkcert for SSL certificates
brew install mkcert nss

# Install Docker Desktop
brew install --cask docker

# Install Composer
brew install composer
```

#### Linux (Ubuntu/Debian)

```bash
# Add PHP repository
sudo add-apt-repository ppa:ondrej/php
sudo apt update

# Install PHP 8.2 (example - install versions you need)
sudo apt install php8.2-fpm php8.2-cli php8.2-common php8.2-mysql php8.2-xml \
    php8.2-curl php8.2-mbstring php8.2-zip php8.2-gd php8.2-intl php8.2-bcmath php8.2-soap

# Install PHP 8.3
sudo apt install php8.3-fpm php8.3-cli php8.3-common php8.3-mysql php8.3-xml \
    php8.3-curl php8.3-mbstring php8.3-zip php8.3-gd php8.3-intl php8.3-bcmath php8.3-soap

# Install Nginx
sudo apt install nginx

# Install mkcert
sudo apt install mkcert libnss3-tools

# Install Docker
curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker $USER
# Log out and back in for group changes to take effect

# Install Composer
curl -sS https://getcomposer.org/installer | php
sudo mv composer.phar /usr/local/bin/composer
```

#### Linux (Fedora/RHEL)

```bash
# Install PHP (Remi repository)
sudo dnf install https://rpms.remirepo.net/fedora/remi-release-$(rpm -E %fedora).rpm
sudo dnf module enable php:remi-8.2
sudo dnf install php php-fpm php-cli php-common php-mysqlnd php-xml \
    php-curl php-mbstring php-zip php-gd php-intl php-bcmath php-soap

# Install Nginx
sudo dnf install nginx

# Install mkcert
sudo dnf install mkcert nss-tools

# Install Docker
sudo dnf install docker docker-compose
sudo systemctl enable --now docker
sudo usermod -aG docker $USER

# Install Composer
curl -sS https://getcomposer.org/installer | php
sudo mv composer.phar /usr/local/bin/composer
```

---

### Step 2: Install MageBox

#### Binary Download (Recommended)

**macOS (Apple Silicon):**
```bash
curl -L https://github.com/qoliber/magebox/releases/latest/download/magebox-darwin-arm64 -o /usr/local/bin/magebox
chmod +x /usr/local/bin/magebox
```

**macOS (Intel):**
```bash
curl -L https://github.com/qoliber/magebox/releases/latest/download/magebox-darwin-amd64 -o /usr/local/bin/magebox
chmod +x /usr/local/bin/magebox
```

**Linux (x86_64):**
```bash
sudo curl -L https://github.com/qoliber/magebox/releases/latest/download/magebox-linux-amd64 -o /usr/local/bin/magebox
sudo chmod +x /usr/local/bin/magebox
```

**Linux (ARM64):**
```bash
sudo curl -L https://github.com/qoliber/magebox/releases/latest/download/magebox-linux-arm64 -o /usr/local/bin/magebox
sudo chmod +x /usr/local/bin/magebox
```

#### Build from Source

```bash
git clone https://github.com/qoliber/magebox.git
cd magebox
go build -o magebox ./cmd/magebox
sudo mv magebox /usr/local/bin/
```

#### Verify Installation

```bash
magebox --version
```

---

### Step 3: Bootstrap MageBox (One-Time Setup)

Run the bootstrap command to set up your environment:

```bash
magebox bootstrap
```

This performs:
1. **Dependency Check** - Verifies Docker, Nginx, mkcert, PHP are installed
2. **Global Config** - Creates `~/.magebox/config.yaml`
3. **SSL Setup** - Installs mkcert CA (all `.test` domains get valid HTTPS)
4. **Port Forwarding** (macOS only) - Sets up transparent port forwarding (80→8080, 443→8443)
   - Allows Nginx to run as your user without sudo
   - Requires sudo password once during bootstrap
   - After setup, no sudo needed for daily operations
5. **Nginx Config** - Configures Nginx to include MageBox vhosts
6. **Docker Services** - Starts MySQL 8.0, Redis, Mailpit containers
7. **DNS Setup** - Configures DNS resolution for `.test` domains

After bootstrap, these services are running:

| Service | Address | Credentials |
|---------|---------|-------------|
| MySQL 8.0 | `localhost:33080` | root / magebox |
| Redis | `localhost:6379` | - |
| Mailpit | http://localhost:8025 | - |

---

### Step 4: Create or Initialize a Project

#### Option A: Quick Install (Recommended for Beginners)

```bash
magebox new mystore --quick
```

This installs MageOS with sensible defaults:
- MageOS 1.0.4 (no Adobe authentication required)
- PHP 8.3, MySQL 8.0, Redis, OpenSearch
- Sample data included
- Domain: mystore.test

#### Option B: Interactive Wizard

```bash
magebox new mystore
```

This launches an interactive wizard where you choose:

1. **Distribution** - Magento Open Source or MageOS
2. **Version** - 2.4.7-p3, 2.4.6-p7, etc.
3. **PHP Version** - Shows compatible versions only
4. **Composer Auth** - Marketplace keys (Magento) or skip (MageOS)
5. **Database** - MySQL 8.0/8.4 or MariaDB 10.6/11.4
6. **Search Engine** - OpenSearch, Elasticsearch, or none
7. **Services** - Redis, RabbitMQ, Mailpit
8. **Sample Data** - Optional demo products
9. **Project Details** - Name and domain

After completion:
```bash
cd mystore
magebox start
magebox cli setup:install \
    --base-url=https://mystore.test \
    --db-host=127.0.0.1:33080 \
    --db-name=mystore \
    --db-user=root \
    --db-password=magebox \
    --admin-firstname=Admin \
    --admin-lastname=User \
    --admin-email=admin@example.com \
    --admin-user=admin \
    --admin-password=admin123
```

#### Option B: Initialize Existing Project

```bash
cd /path/to/your/magento/project
magebox init
```

Answer the prompts to create a `.magebox` config file.

---

### Step 5: Start Your Project

```bash
magebox start
```

This will:
- Verify PHP version is installed
- Generate SSL certificates for your domain
- Create PHP-FPM pool configuration
- Create Nginx vhost configuration
- Ensure database containers are running
- Create the project database
- Configure DNS for your domain

---

### Step 6: Access Your Site

Open your browser and navigate to:

```
https://mystore.test
```

HTTPS works automatically with a valid certificate!

---

## Daily Usage

### Starting/Stopping Projects

```bash
magebox start             # Start project services
magebox stop              # Stop project services
magebox status            # Show project status
magebox restart           # Restart project services
```

### Running Magento Commands

```bash
magebox cli cache:clean              # Run bin/magento commands
magebox cli setup:upgrade
magebox cli indexer:reindex
```

### PHP Version Management

```bash
magebox php                          # Show current PHP version
magebox php 8.3                      # Switch to PHP 8.3
magebox shell                        # Open shell with correct PHP in PATH
```

### Database Operations

```bash
magebox db shell                     # Open MySQL shell
magebox db import dump.sql           # Import database
magebox db export                    # Export to {project}.sql
magebox db export backup.sql         # Export to specific file
```

### Redis Operations

```bash
magebox redis shell                  # Open Redis CLI
magebox redis flush                  # Flush all data
magebox redis info                   # Show server info
```

### Log Viewing

```bash
magebox logs                         # Show last 20 lines of all logs
magebox logs -f                      # Follow mode (live tail)
magebox logs -n 100                  # Show last 100 lines
magebox logs system.log              # Specific log file
magebox logs "exception*"            # Wildcard pattern
```

### Custom Commands

Define in `.magebox`:
```yaml
commands:
  deploy: "php bin/magento deploy:mode:set production"
  reindex:
    description: "Reindex all"
    run: "php bin/magento indexer:reindex"
```

Run with:
```bash
magebox run deploy
magebox run reindex
```

---

## Configuration Reference

### .magebox File

```yaml
name: mystore                    # Required: project name (used for DB name)

domains:                         # Required: at least one domain
  - host: mystore.test           # Required: domain name
    root: pub                    # Optional: document root (default: pub)
    ssl: true                    # Optional: enable SSL (default: true)
  - host: api.mystore.test       # Additional domains
    root: pub

php: "8.2"                       # Required: PHP version

php_ini:                         # Optional: PHP INI overrides
  opcache.enable: "0"            # Disable OPcache for development
  display_errors: "On"           # Show PHP errors
  xdebug.mode: "debug"           # Enable Xdebug debugging
  max_execution_time: "3600"     # Increase execution time
  memory_limit: "2G"             # Override memory limit

services:
  mysql: "8.0"                   # MySQL version
  # mariadb: "10.6"              # Or MariaDB (choose one)
  redis: true                    # Enable Redis
  opensearch: "2.12"             # OpenSearch version
  # elasticsearch: "8.11"        # Or Elasticsearch
  # rabbitmq: true               # Enable RabbitMQ
  # mailpit: true                # Enable Mailpit

env:                             # Optional: environment variables
  MAGE_MODE: developer

commands:                        # Optional: custom commands
  deploy: "php bin/magento deploy:mode:set production"
  reindex:
    description: "Reindex all Magento indexes"
    run: "php bin/magento indexer:reindex"
  setup:
    description: "Full project setup"
    run: |
      composer install
      php bin/magento setup:upgrade
      php bin/magento cache:flush
```

### .magebox.local File

Override settings locally (add to `.gitignore`):

```yaml
php: "8.3"                       # Use different PHP locally

php_ini:                         # Override PHP settings locally
  opcache.enable: "0"            # Disable OPcache for local dev
  xdebug.mode: "debug,coverage"  # Enable Xdebug
  display_errors: "On"

services:
  mysql:
    version: "8.0"
    port: 3307                   # Different port
```

### Global Configuration

Location: `~/.magebox/config.yaml`

```bash
magebox config show              # View current config
magebox config set dns_mode dnsmasq
magebox config set default_php 8.3
magebox config set tld test
magebox config set portainer true
```

---

## All Commands

| Command | Description |
|---------|-------------|
| `magebox` | Show help and logo |
| `magebox bootstrap` | One-time environment setup |
| `magebox new [dir]` | Create new project (interactive wizard) |
| `magebox new [dir] --quick` | Quick install MageOS with sample data |
| `magebox init` | Initialize existing project |
| `magebox start` | Start project services |
| `magebox stop` | Stop project services |
| `magebox status` | Show project status |
| `magebox restart` | Restart project services |
| `magebox list` | List all MageBox projects |
| `magebox php [version]` | Show/switch PHP version |
| `magebox shell` | Open shell with correct PHP |
| `magebox cli <command>` | Run bin/magento command |
| `magebox db shell` | Open database shell |
| `magebox db import <file>` | Import database |
| `magebox db export [file]` | Export database |
| `magebox redis shell` | Open Redis CLI |
| `magebox redis flush` | Flush Redis data |
| `magebox redis info` | Show Redis info |
| `magebox logs [-f] [-n N]` | View/tail logs |
| `magebox varnish status` | Show Varnish status |
| `magebox varnish purge <url>` | Purge URL from cache |
| `magebox varnish flush` | Flush all cache |
| `magebox run <name>` | Run custom command |
| `magebox global start` | Start global services |
| `magebox global stop` | Stop all services |
| `magebox global status` | Show all services |
| `magebox ssl trust` | Trust local CA |
| `magebox ssl generate` | Regenerate certificates |
| `magebox dns status` | Show DNS status |
| `magebox dns setup` | Setup dnsmasq |
| `magebox config show` | Show global config |
| `magebox config set <k> <v>` | Set config value |
| `magebox config init` | Initialize config |
| `magebox self-update` | Update MageBox |
| `magebox self-update check` | Check for updates |
| `magebox install` | Install dependencies |

---

## Service Ports

### Database Ports

| Service | Version | Port |
|---------|---------|------|
| MySQL | 5.7 | 33057 |
| MySQL | 8.0 | 33080 |
| MySQL | 8.4 | 33084 |
| MariaDB | 10.4 | 33104 |
| MariaDB | 10.6 | 33106 |
| MariaDB | 11.4 | 33114 |

### Other Services

| Service | Port |
|---------|------|
| Redis | 6379 |
| OpenSearch | 9200 |
| Elasticsearch | 9200 |
| RabbitMQ | 5672 (AMQP), 15672 (Web) |
| Mailpit SMTP | 1025 |
| Mailpit Web | 8025 |
| Portainer | 9000 |

---

## Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                          Your Machine                             │
├──────────────────────────────────────────────────────────────────┤
│                                                                   │
│  Browser                                                          │
│     │                                                             │
│     │ https://mystore.test (port 443)                            │
│     ▼                                                             │
│  ┌──────────────┐                                                 │
│  │ pf (macOS)   │  Port forwarding: 80→8080, 443→8443           │
│  └──────┬───────┘                                                 │
│         │ port 8443                                               │
│         ▼                                                         │
│  ┌─────────────┐    ┌─────────────┐    ┌───────────────────────┐  │
│  │   Nginx     │    │  PHP-FPM    │    │   Docker Containers   │  │
│  │  (native)   │    │  (native)   │    │                       │  │
│  │  as user    │    │  as user    │    │  ┌───────┐ ┌────────┐ │  │
│  │ 8080/8443   │───▶│ Unix Socket │    │  │ MySQL │ │  Redis │ │  │
│  │             │    │             │    │  └───────┘ └────────┘ │  │
│  └─────────────┘    └─────────────┘    │  ┌────────────┐       │  │
│         │                  │           │  │ OpenSearch │       │  │
│         │                  │           │  └────────────┘       │  │
│         │                  │           │  ┌─────────┐          │  │
│         ▼                  ▼           │  │ Mailpit │          │  │
│  ┌─────────────────────────────────┐   │  └─────────┘          │  │
│  │        Project Directory        │   └───────────────────────┘  │
│  │         /path/to/magento        │◀──────────────┘              │
│  └─────────────────────────────────┘                              │
│                                                                   │
└──────────────────────────────────────────────────────────────────┘
```

**Key Features:**
- **No sudo after setup** - Services run as your user (pf handles port forwarding)
- **Native performance** - PHP and Nginx run directly on your machine
- **Unix sockets** - Fast PHP-FPM communication via sockets instead of TCP
- **Docker for services** - MySQL, Redis, OpenSearch in isolated containers
- **Clean URLs** - Access sites at https://mystore.test (no :8443 suffix)

---

## Port Forwarding (macOS)

### Overview

On macOS, MageBox uses **packet filter (pf)** to enable services to run as your regular user without requiring sudo for daily operations. This is the same approach used by ddev and other professional development tools.

### How It Works

**The Problem:**
- Web servers need to listen on privileged ports 80 (HTTP) and 443 (HTTPS)
- Only root can bind to ports below 1024
- Running services as root is a security risk

**MageBox Solution:**
1. Nginx runs on **unprivileged ports 8080 and 8443** as your user
2. macOS `pf` (packet filter) transparently forwards:
   - Port 80 → 8080
   - Port 443 → 8443
3. You access sites at clean URLs like `https://mystore.test` (no port suffix)
4. Behind the scenes, traffic is forwarded to high ports

### What Gets Installed During Bootstrap

When you run `magebox bootstrap`, it creates (requires sudo password once):

#### 1. PF Rules File
**Location:** `/etc/pf.anchors/com.magebox`
```
# MageBox port forwarding rules
rdr pass on lo0 inet proto tcp from any to any port 80 -> 127.0.0.1 port 8080
rdr pass on lo0 inet proto tcp from any to any port 443 -> 127.0.0.1 port 8443
```

#### 2. LaunchDaemon
**Location:** `/Library/LaunchDaemons/com.magebox.portforward.plist`

This daemon:
- Loads the pf rules automatically on system boot
- Runs as root (required for pf)
- Enables the forwarding transparently

### Benefits

✅ **No sudo after setup** - All daily commands (`start`, `stop`, `restart`) run as your user
✅ **Secure** - Services don't run as root
✅ **Transparent** - Clean URLs without port numbers
✅ **Persistent** - Survives reboots
✅ **Standard approach** - Same method used by ddev, Lando, etc.

### Verification

Check if port forwarding is installed:

```bash
# Check if LaunchDaemon exists
ls -la /Library/LaunchDaemons/com.magebox.portforward.plist

# Check if pf rules exist
cat /etc/pf.anchors/com.magebox

# Test that it works
curl -I https://mystore.test  # Should connect without :8443
```

### Troubleshooting

**Port forwarding not working?**

1. **Verify LaunchDaemon is loaded:**
   ```bash
   sudo launchctl list | grep magebox
   ```
   Should show: `com.magebox.portforward`

2. **Reload pf rules manually:**
   ```bash
   sudo pfctl -ef /etc/pf.anchors/com.magebox
   ```

3. **Check nginx is listening on 8080/8443:**
   ```bash
   lsof -nP -iTCP:8080 -sTCP:LISTEN
   lsof -nP -iTCP:8443 -sTCP:LISTEN
   ```

4. **Re-run bootstrap:**
   ```bash
   magebox bootstrap
   ```

### Uninstalling Port Forwarding

If you need to remove the port forwarding setup:

```bash
# Unload LaunchDaemon
sudo launchctl unload /Library/LaunchDaemons/com.magebox.portforward.plist

# Remove files
sudo rm /Library/LaunchDaemons/com.magebox.portforward.plist
sudo rm /etc/pf.anchors/com.magebox
```

### Linux Alternative

On Linux, MageBox doesn't need port forwarding because:
- Services are managed by systemd (which can bind to privileged ports)
- Alternatively, you can use `setcap` to grant capabilities to the nginx binary

---

## File Locations

```
~/.magebox/
├── config.yaml            # Global configuration
├── certs/                 # SSL certificates
│   └── {domain}/
│       ├── cert.pem
│       └── key.pem
├── nginx/
│   └── vhosts/            # Generated Nginx configs
├── php/
│   └── pools/             # Generated PHP-FPM pools
├── docker/
│   └── docker-compose.yml # Docker services
└── run/                   # Runtime files (sockets, PIDs)
```

---

## PHP INI Configuration

### Overview

MageBox allows you to override PHP configuration settings per-project using the `php_ini` section in your `.magebox` or `.magebox.local` file. These settings are injected into the PHP-FPM pool configuration and take precedence over system-wide PHP settings.

### Usage

Add `php_ini` settings to your `.magebox` file:

```yaml
php_ini:
  opcache.enable: "0"            # Disable OPcache
  display_errors: "On"           # Show errors
  error_reporting: "E_ALL"       # Report all errors
  max_execution_time: "3600"     # 1 hour timeout
  memory_limit: "2G"             # 2GB memory
```

### Common Use Cases

#### 1. Disable OPcache for Development

By default, OPcache is enabled for performance. Disable it during active development to see code changes immediately:

```yaml
php_ini:
  opcache.enable: "0"
```

#### 2. Enable Xdebug

Configure Xdebug for debugging and profiling:

```yaml
php_ini:
  xdebug.mode: "debug,coverage"
  xdebug.start_with_request: "yes"
  xdebug.client_host: "localhost"
  xdebug.client_port: "9003"
```

#### 3. Increase Limits for Large Imports

For heavy operations like database imports or product imports:

```yaml
php_ini:
  max_execution_time: "7200"     # 2 hours
  memory_limit: "4G"             # 4GB
  post_max_size: "256M"
  upload_max_filesize: "256M"
```

#### 4. Production Settings

Optimize for production (use in `.magebox`, not `.magebox.local`):

```yaml
php_ini:
  opcache.enable: "1"
  opcache.validate_timestamps: "0"  # Don't check file changes
  display_errors: "Off"
  error_reporting: "E_ALL & ~E_DEPRECATED & ~E_STRICT"
```

### Development vs Production

Use `.magebox.local` for development-specific overrides:

**.magebox** (committed to git):
```yaml
php: "8.2"
# Production settings
```

**.magebox.local** (in .gitignore):
```yaml
php_ini:
  opcache.enable: "0"            # Development only
  display_errors: "On"
  xdebug.mode: "debug"
```

### Available Directives

You can override any PHP INI directive that can be set via `php_admin_value` in PHP-FPM. Common ones include:

- **Performance**: `opcache.*`, `realpath_cache_size`, `realpath_cache_ttl`
- **Debugging**: `display_errors`, `error_reporting`, `xdebug.*`
- **Limits**: `memory_limit`, `max_execution_time`, `max_input_time`, `max_input_vars`
- **Uploads**: `post_max_size`, `upload_max_filesize`
- **Session**: `session.save_handler`, `session.gc_maxlifetime`

### Applying Changes

After modifying `php_ini` settings, restart your project:

```bash
magebox restart
```

This regenerates the PHP-FPM pool configuration and reloads PHP-FPM.

### Viewing Generated Configuration

The generated PHP-FPM pool configuration is located at:

```bash
~/.magebox/php/pools/{project-name}.conf
```

You can view it to see all applied settings:

```bash
cat ~/.magebox/php/pools/mystore.conf
```

---

## FAQ

### Why do I need to enter my password during bootstrap?

During the **one-time** `magebox bootstrap` setup, you'll be prompted for your sudo password to:
- Install port forwarding rules (macOS only) - allows Nginx to run as your user
- Configure system-level services (Nginx, PHP-FPM symlinks)

**After bootstrap is complete, you never need sudo again.** All daily commands (`start`, `stop`, `restart`) run as your regular user.

### Do I need sudo to start/stop projects?

**No!** After running `magebox bootstrap` once, all project commands run without sudo:
```bash
magebox start   # No sudo needed
magebox stop    # No sudo needed
magebox restart # No sudo needed
```

This is possible because:
- Port forwarding (pf) runs as a system daemon
- Nginx listens on unprivileged ports (8080/8443)
- PHP-FPM pools run as your user

### How does MageBox avoid port conflicts with other tools?

**For web ports (80/443):**
- MageBox uses port forwarding (macOS) or systemd (Linux)
- Nginx actually listens on 8080/8443
- Port forwarding makes it accessible on 80/443
- If you're using other tools (MAMP, Valet, etc.), stop them first

**For database ports:**
- MySQL: 33080 (not standard 3306)
- Redis: 6379 (standard, may conflict)
- OpenSearch: 9200 (standard, may conflict)

### Can I run MageBox alongside ddev/Lando/Warden?

**Yes, but not simultaneously.** MageBox uses the same approach (port forwarding on macOS), so:

✅ **Safe:** Run MageBox and ddev on different days
✅ **Safe:** Stop ddev before starting MageBox
❌ **Conflict:** Running both at the same time will cause port conflicts

```bash
# If you have ddev running:
ddev poweroff

# Then start MageBox:
magebox start
```

### Why use native PHP instead of Docker?

**Performance!** Native PHP is significantly faster than containerized PHP:
- No file sync overhead (Docker on macOS syncs files slowly)
- Direct filesystem access
- No virtualization layer
- Faster composer installs
- Faster Magento compilation

Docker is still used for services that benefit from isolation (MySQL, Redis, OpenSearch).

### Where are project files stored?

MageBox **does NOT move** your project files. Your Magento installation stays exactly where it is:
- `/Volumes/qoliber/jbfurniturem2` ← Your project
- `~/.magebox/` ← MageBox configuration only

MageBox only creates:
- Nginx vhost configs pointing to your project
- PHP-FPM pool configs for your project
- SSL certificates for your domains

### What happens if I reboot my Mac?

Everything persists across reboots:
- ✅ Port forwarding LaunchDaemon loads automatically
- ✅ Docker containers start automatically (if configured)
- ⚠️ **You need to run `magebox start` again** for project-specific services

### Can I use multiple PHP versions simultaneously?

**Yes!** Each project can use a different PHP version:

```bash
# Project 1 uses PHP 8.1
cd /path/to/project1
cat .magebox
php: "8.1"

# Project 2 uses PHP 8.3
cd /path/to/project2
cat .magebox
php: "8.3"
```

MageBox creates separate PHP-FPM pools for each project with the correct version.

### How do I disable OPcache for development?

Add to your `.magebox` or `.magebox.local` file:

```yaml
php_ini:
  opcache.enable: "0"
```

Then restart:
```bash
magebox stop && magebox start
```

### Why does my site show a Magento error instead of my test file?

Magento's Nginx configuration routes all `.php` files through `index.php`. For direct PHP file access (like `info.php`), the file needs to exist and not be routed through Magento.

MageBox now allows all `.php` files to execute directly, so files like `info.php` in the `pub/` directory will work.

### Can I use xdebug?

**Yes!** Configure it in your `.magebox` file:

```yaml
php_ini:
  xdebug.mode: "debug"
  xdebug.start_with_request: "yes"
  xdebug.client_host: "localhost"
  xdebug.client_port: "9003"
```

Make sure xdebug is installed:
```bash
# macOS
brew install php@8.1-xdebug  # or php@8.2-xdebug, php@8.3-xdebug

# Linux
sudo apt install php8.1-xdebug
```

---

## Troubleshooting

### PHP version not found

```
[ERROR] PHP 8.3 not found

Install it with:
  macOS:  brew install php@8.3
  Ubuntu: sudo apt install php8.3-fpm php8.3-cli ...
```

### Configuration file not found

```
[ERROR] Configuration file not found: /path/.magebox

[INFO] Run magebox init to create one
```

### Permission denied on /etc/hosts

MageBox needs sudo to modify /etc/hosts. You'll be prompted for your password.

### Port already in use

```bash
lsof -i :80
lsof -i :443
lsof -i :33080
```

### Nginx configuration error

```bash
sudo nginx -t
```

### Docker not running

```bash
# macOS
open -a Docker

# Linux
sudo systemctl start docker
```

### SSL certificate issues

```bash
magebox ssl trust      # Re-trust CA
magebox ssl generate   # Regenerate certs
```

---

## Development

### Building from source

```bash
git clone https://github.com/qoliber/magebox.git
cd magebox
go mod tidy
go build -o magebox ./cmd/magebox
```

### Running tests

```bash
go test ./... -v
```

### Project structure

```
magebox/
├── cmd/magebox/           # CLI entry point (~2800 lines)
├── internal/
│   ├── cli/               # Colors, logging, UI helpers
│   ├── config/            # .magebox parsing, global config
│   ├── platform/          # OS detection, paths
│   ├── php/               # PHP detection, FPM pools
│   ├── nginx/             # Vhost generation
│   ├── ssl/               # mkcert integration
│   ├── docker/            # Docker Compose generation
│   ├── dns/               # /etc/hosts, dnsmasq
│   ├── varnish/           # Varnish VCL generation
│   ├── project/           # Lifecycle management
│   └── updater/           # Self-update functionality
└── .github/workflows/     # CI/CD pipelines
```

---

## License

MIT License - see LICENSE file.

## Contributing

Contributions are welcome! Please open an issue or submit a PR.

## Credits

Built by [Qoliber](https://qoliber.com) for the Magento community.

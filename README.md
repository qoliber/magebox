# MageBox

A modern, fast development environment for Magento 2. Uses native PHP-FPM, Nginx, and Varnish for maximum performance, with Docker only for stateless services like MySQL, Redis, and OpenSearch.

## Why MageBox?

Unlike Docker-based solutions (Warden, DDEV), MageBox runs PHP and Nginx natively on your machine:

- **No file sync overhead** - Native filesystem access means instant file changes
- **Native performance** - PHP runs at full speed, not inside a container
- **Simple architecture** - Docker only for databases and search engines
- **Multi-project support** - Run multiple Magento projects simultaneously
- **Easy PHP switching** - Change PHP versions per project with one command

## Requirements

### macOS

```bash
# Install Homebrew if not installed
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install PHP (install versions you need)
brew install php@8.1 php@8.2 php@8.3 php@8.4 php@8.5

# Install Nginx
brew install nginx

# Install mkcert for SSL certificates
brew install mkcert nss
mkcert -install

# Install Docker Desktop
brew install --cask docker

# Install Composer (required for `magebox new`)
brew install composer
```

### Linux (Ubuntu/Debian)

```bash
# Add PHP repository
sudo add-apt-repository ppa:ondrej/php
sudo apt update

# Install PHP (install versions you need)
sudo apt install php8.1-fpm php8.1-cli php8.1-common php8.1-mysql php8.1-xml \
    php8.1-curl php8.1-mbstring php8.1-zip php8.1-gd php8.1-intl php8.1-bcmath php8.1-soap

sudo apt install php8.2-fpm php8.2-cli php8.2-common php8.2-mysql php8.2-xml \
    php8.2-curl php8.2-mbstring php8.2-zip php8.2-gd php8.2-intl php8.2-bcmath php8.2-soap

sudo apt install php8.3-fpm php8.3-cli php8.3-common php8.3-mysql php8.3-xml \
    php8.3-curl php8.3-mbstring php8.3-zip php8.3-gd php8.3-intl php8.3-bcmath php8.3-soap

sudo apt install php8.4-fpm php8.4-cli php8.4-common php8.4-mysql php8.4-xml \
    php8.4-curl php8.4-mbstring php8.4-zip php8.4-gd php8.4-intl php8.4-bcmath php8.4-soap

# PHP 8.5 (when available)
# sudo apt install php8.5-fpm php8.5-cli php8.5-common php8.5-mysql php8.5-xml \
#     php8.5-curl php8.5-mbstring php8.5-zip php8.5-gd php8.5-intl php8.5-bcmath php8.5-soap

# Install Nginx
sudo apt install nginx

# Install mkcert
sudo apt install mkcert libnss3-tools
mkcert -install

# Install Docker
curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker $USER
# Log out and back in for group changes to take effect

# Install Composer (required for `magebox new`)
curl -sS https://getcomposer.org/installer | php
sudo mv composer.phar /usr/local/bin/composer
```

## Installation

### Binary Download (Recommended)

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

### From Source

```bash
# Clone the repository
git clone https://github.com/qoliber/magebox.git
cd magebox

# Build
go build -o magebox ./cmd/magebox

# Move to PATH
sudo mv magebox /usr/local/bin/

# Verify installation
magebox --version
```

### Update

```bash
# Self-update to latest version
magebox self-update

# Check for updates without installing
magebox self-update check
```

## Quick Start

### 1. Bootstrap MageBox (First-Time Setup)

After installing dependencies and MageBox binary, run the bootstrap command:

```bash
magebox bootstrap
```

This performs a one-time setup:
- ✓ Checks all required dependencies (Docker, Nginx, mkcert, PHP)
- ✓ Initializes global configuration (`~/.magebox/config.yaml`)
- ✓ Sets up mkcert CA for HTTPS support (all `.test` domains will have valid SSL)
- ✓ Configures Nginx to include MageBox vhosts
- ✓ Creates and starts Docker services (MySQL 8.0, Redis, Mailpit)
- ✓ Sets up DNS resolution

After bootstrap, the following services are available:
- **MySQL 8.0:** `localhost:33080` (root password: `magebox`)
- **Redis:** `localhost:6379`
- **Mailpit:** http://localhost:8025 (catch-all email testing)

### 2. Initialize a project

```bash
cd /path/to/your/magento/project
magebox init
```

This creates a `.magebox` file:

```yaml
name: mystore
domains:
  - host: mystore.test
php: "8.2"
services:
  mysql: "8.0"
  redis: true
```

### 3. Start the project

```bash
magebox start
```

This will:
- Check PHP 8.2 is installed (or show install instructions)
- Generate SSL certificates for mystore.test
- Create PHP-FPM pool configuration
- Create Nginx vhost configuration
- Ensure MySQL and Redis containers are running
- Create the database
- Add mystore.test to /etc/hosts (or use dnsmasq)

### 4. Access your site

Open https://mystore.test in your browser - HTTPS works out of the box!

## Configuration

### .magebox file

```yaml
name: mystore                    # Required: project name (used for DB name)

domains:                         # Required: at least one domain
  - host: mystore.test           # Required: domain name
    root: pub                    # Optional: document root (default: pub)
    ssl: true                    # Optional: enable SSL (default: true)
  - host: api.mystore.test
    root: pub

php: "8.2"                       # Required: PHP version

services:
  mysql: "8.0"                   # MySQL version (or mariadb: "10.6")
  redis: true                    # Enable Redis
  opensearch: "2.12"             # OpenSearch version
  # elasticsearch: "8.11"        # Alternative to OpenSearch
  # rabbitmq: true               # Enable RabbitMQ
  # mailpit: true                # Enable Mailpit for email testing

env:                             # Optional: environment variables
  MAGE_MODE: developer

commands:                        # Optional: custom commands (run with "magebox run <name>")
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

### .magebox.local file

Override settings locally without affecting the team:

```yaml
# Switch to PHP 8.3 locally
php: "8.3"

# Use different MySQL port to avoid conflicts
services:
  mysql:
    version: "8.0"
    port: 3307
```

Add `.magebox.local` to your `.gitignore`.

## Commands

### Create New Project

```bash
magebox new mystore       # Create new Magento/MageOS project in ./mystore
magebox new .             # Create in current directory
```

The `new` command launches an interactive wizard that guides you through:

1. **Distribution** - Choose between Magento Open Source or MageOS
2. **Version** - Select from available versions (2.4.7-p3, 2.4.6-p7, etc.)
3. **PHP Version** - Pick compatible PHP version (shows only compatible options)
4. **Composer Auth** - Enter marketplace keys (Magento) or skip (MageOS)
5. **Database** - MySQL 8.0/8.4 or MariaDB 10.6/11.4
6. **Search Engine** - OpenSearch, Elasticsearch, or none
7. **Services** - Redis, RabbitMQ, Mailpit
8. **Sample Data** - Optional demo products and content
9. **Project Details** - Name and domain

### Project Commands

```bash
magebox init              # Initialize existing project (creates .magebox)
magebox start             # Start project services
magebox stop              # Stop project services
magebox status            # Show project status
magebox restart           # Restart project services
```

### PHP Commands

```bash
magebox php               # Show current PHP version
magebox php 8.3           # Switch to PHP 8.3 (updates .magebox.local)
magebox shell             # Open shell with correct PHP in PATH
magebox cli cache:clean   # Run bin/magento command
```

### Custom Commands

```bash
magebox run <name>        # Run custom command from .magebox
magebox run deploy        # Example: run deploy command
magebox run setup         # Example: run setup command
```

### Database Commands

```bash
magebox db shell          # Open MySQL shell
magebox db import dump.sql # Import database
magebox db export          # Export database to file
```

### Redis Commands

```bash
magebox redis flush       # Flush all Redis data
magebox redis shell       # Open Redis CLI shell
magebox redis info        # Show Redis server info and stats
```

### Log Commands

```bash
magebox logs              # Tail all var/log/*.log files (last 20 lines)
magebox logs -f           # Follow mode (continuous, like tail -f)
magebox logs -n 50        # Show last 50 lines
magebox logs system.log   # Tail only system.log
magebox logs "error*"     # Tail files matching pattern
```

### Varnish Commands

```bash
magebox varnish status    # Show Varnish status and stats
magebox varnish purge /   # Purge a URL from cache
magebox varnish flush     # Flush all cached content
```

### Global Commands

```bash
magebox bootstrap         # First-time setup (run once after install)
magebox global start      # Start global services (Nginx, Docker containers)
magebox global stop       # Stop all MageBox services
magebox global status     # Show all projects and services
```

### SSL Commands

```bash
magebox ssl trust         # Trust local CA (run once)
magebox ssl generate      # Regenerate certificates
```

### DNS Commands

```bash
magebox dns status        # Show DNS configuration status
magebox dns setup         # Setup dnsmasq for wildcard *.test resolution
```

### Project List

```bash
magebox list              # List all discovered MageBox projects
```

### Self-Update

```bash
magebox self-update       # Update MageBox to latest version
magebox self-update check # Check for updates without installing
```

### Configuration Commands

```bash
magebox config show       # Show current global configuration
magebox config init       # Initialize ~/.magebox/config.yaml
magebox config set <key> <value>  # Set a configuration value
```

**Available config keys:**
- `dns_mode` - DNS resolution mode: "hosts" or "dnsmasq"
- `default_php` - Default PHP version for new projects (e.g., "8.2")
- `tld` - Top-level domain for local dev (default: "test")
- `portainer` - Enable Portainer Docker UI: "true" or "false"

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Your Machine                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │   Nginx     │  │  PHP-FPM    │  │    Docker Containers    │  │
│  │  (native)   │  │  (native)   │  │                         │  │
│  │             │  │             │  │  ┌─────┐ ┌──────────┐   │  │
│  │ vhost.conf  │──│ pool.conf   │  │  │MySQL│ │OpenSearch│   │  │
│  │ per project │  │ per project │  │  └─────┘ └──────────┘   │  │
│  │             │  │             │  │  ┌─────┐ ┌──────────┐   │  │
│  └─────────────┘  └─────────────┘  │  │Redis│ │ Mailpit  │   │  │
│         │                │         │  └─────┘ └──────────┘   │  │
│         └────────────────┘         └─────────────────────────┘  │
│                  │                              │               │
│                  ▼                              │               │
│         ┌─────────────┐                         │               │
│         │   Project   │◄────────────────────────┘               │
│         │  /var/www   │                                         │
│         └─────────────┘                                         │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## Service Ports

### Database Ports (to avoid conflicts)

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
| RabbitMQ | 5672, 15672 (management) |
| Mailpit SMTP | 1025 |
| Mailpit Web | 8025 |

## File Locations

### macOS

```
~/.magebox/
├── certs/                 # SSL certificates
├── nginx/vhosts/          # Generated Nginx configs
├── php/pools/             # Generated PHP-FPM pools
├── docker/                # Docker Compose files
└── run/                   # Runtime files (sockets, PIDs)
```

### Linux

```
~/.magebox/                # Same structure as macOS
```

## Troubleshooting

### PHP version not found

```
✗ PHP 8.3 not found

Install it with:
  macOS:  brew install php@8.3
  Ubuntu: sudo apt install php8.3-fpm php8.3-cli ...
```

### Permission denied on /etc/hosts

MageBox needs sudo to modify /etc/hosts. You'll be prompted for your password.

### Port already in use

Check if another service is using the port:

```bash
lsof -i :80
lsof -i :443
```

### Nginx configuration error

Test Nginx configuration:

```bash
sudo nginx -t
```

### Docker not running

Start Docker:

```bash
# macOS
open -a Docker

# Linux
sudo systemctl start docker
```

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
├── cmd/magebox/           # CLI entry point
├── internal/
│   ├── config/            # .magebox parsing
│   ├── platform/          # OS detection
│   ├── php/               # PHP detection & pools
│   ├── nginx/             # Vhost generation
│   ├── ssl/               # Certificate management
│   ├── docker/            # Docker Compose
│   ├── dns/               # /etc/hosts
│   ├── varnish/           # Varnish VCL
│   └── project/           # Lifecycle management
└── templates/             # Config templates
```

## License

MIT License - see LICENSE file.

## Contributing

Contributions are welcome! Please read our contributing guidelines before submitting PRs.

## Credits

Built by [Qoliber](https://qoliber.com) for the Magento community.

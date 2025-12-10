# MageBox

```
      ___________
     /          /|
    /  MAGE   /  |
   /   BOX   /   |
  /__________/   |
  |          |   /
  |  0.1.0   |  /
  |__________|_/
```

A modern, fast development environment for Magento 2. Uses native PHP-FPM, Nginx, and Varnish for maximum performance, with Docker only for stateless services like MySQL, Redis, and OpenSearch.

## Why MageBox?

Unlike Docker-based solutions (Warden, DDEV), MageBox runs PHP and Nginx natively on your machine:

- **No file sync overhead** - Native filesystem access means instant file changes
- **Native performance** - PHP runs at full speed, not inside a container
- **Simple architecture** - Docker only for databases and search engines
- **Multi-project support** - Run multiple Magento projects simultaneously
- **Easy PHP switching** - Change PHP versions per project with one command

---

## Step-by-Step Setup Guide

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
4. **Nginx Config** - Configures Nginx to include MageBox vhosts
5. **Docker Services** - Starts MySQL 8.0, Redis, Mailpit containers
6. **DNS Setup** - Configures DNS resolution for `.test` domains

After bootstrap, these services are running:

| Service | Address | Credentials |
|---------|---------|-------------|
| MySQL 8.0 | `localhost:33080` | root / magebox |
| Redis | `localhost:6379` | - |
| Mailpit | http://localhost:8025 | - |

---

### Step 4: Create or Initialize a Project

#### Option A: Create New Magento/MageOS Project

```bash
magebox new mystore
```

This launches an interactive wizard:

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
| `magebox new [dir]` | Create new Magento/MageOS project |
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
│  ┌─────────────┐    ┌─────────────┐    ┌───────────────────────┐  │
│  │   Nginx     │    │  PHP-FPM    │    │   Docker Containers   │  │
│  │  (native)   │    │  (native)   │    │                       │  │
│  │             │    │             │    │  ┌───────┐ ┌────────┐ │  │
│  │ Port 80/443 │───▶│ Unix Socket │    │  │ MySQL │ │  Redis │ │  │
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

# Installation

## Quick Install

The fastest way to install MageBox:

```bash
curl -fsSL https://get.magebox.dev | bash
```

This downloads the latest release and installs it to `/usr/local/bin/magebox`.

## Manual Installation

### Download Binary

Download the appropriate binary for your platform from [GitHub Releases](https://github.com/qoliber/magebox/releases):

::: code-group

```bash [macOS Apple Silicon]
curl -L https://github.com/qoliber/magebox/releases/latest/download/magebox-darwin-arm64 -o magebox
chmod +x magebox
sudo mv magebox /usr/local/bin/
```

```bash [macOS Intel]
curl -L https://github.com/qoliber/magebox/releases/latest/download/magebox-darwin-amd64 -o magebox
chmod +x magebox
sudo mv magebox /usr/local/bin/
```

```bash [Linux x64]
curl -L https://github.com/qoliber/magebox/releases/latest/download/magebox-linux-amd64 -o magebox
chmod +x magebox
sudo mv magebox /usr/local/bin/
```

```bash [Linux ARM64]
curl -L https://github.com/qoliber/magebox/releases/latest/download/magebox-linux-arm64 -o magebox
chmod +x magebox
sudo mv magebox /usr/local/bin/
```

:::

### Build from Source

Requires Go 1.21 or later:

```bash
git clone https://github.com/qoliber/magebox.git
cd magebox
go build -o magebox ./cmd/magebox
sudo mv magebox /usr/local/bin/
```

## Verify Installation

```bash
magebox --version
# MageBox v0.1.0
```

## System Requirements

### macOS

- macOS 12 (Monterey) or later
- Homebrew (for PHP and Nginx installation)
- Docker Desktop or Colima

### Linux

- Ubuntu 20.04+, Debian 11+, or Fedora 36+
- Docker Engine
- systemd (for service management)

### Windows WSL2

- Windows 10 (version 2004+) or Windows 11
- WSL2 with Ubuntu or Fedora distribution
- Docker Desktop with WSL2 backend enabled

## Dependencies

MageBox requires several dependencies to function. The `magebox bootstrap` command will check for and help install these:

| Dependency | Purpose | Installation |
|------------|---------|--------------|
| Docker | Run services | Docker Desktop / Engine |
| Nginx | Web server | Homebrew / apt |
| PHP 8.1+ | PHP runtime | Homebrew / Ondrej PPA |
| mkcert | SSL certificates | Homebrew / apt |
| Composer | PHP packages | Homebrew / apt |

### macOS Dependencies

```bash
# Install Homebrew if needed
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install dependencies
brew install nginx php@8.2 mkcert composer
brew install --cask docker
```

### Ubuntu/Debian Dependencies

```bash
# Add PHP repository
sudo add-apt-repository ppa:ondrej/php
sudo apt update

# Install dependencies
sudo apt install nginx php8.2-fpm php8.2-cli php8.2-common \
    php8.2-mysql php8.2-xml php8.2-curl php8.2-gd php8.2-mbstring \
    php8.2-zip php8.2-bcmath php8.2-intl php8.2-soap composer

# Install mkcert
sudo apt install libnss3-tools
curl -JLO "https://dl.filippo.io/mkcert/latest?for=linux/amd64"
chmod +x mkcert-v*-linux-amd64
sudo mv mkcert-v*-linux-amd64 /usr/local/bin/mkcert

# Install Docker
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
```

## Updating MageBox

MageBox includes self-update functionality:

```bash
# Check for updates
magebox self-update check

# Update to latest version
magebox self-update
```

## Next Steps

After installation, run the bootstrap command to set up your environment:

```bash
magebox bootstrap
```

See [Bootstrap](/guide/bootstrap) for details on what this command does.

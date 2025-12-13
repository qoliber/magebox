# Global Configuration

MageBox stores global settings in `~/.magebox/config.yaml`.

## Viewing Configuration

```bash
magebox config show
```

Output:
```yaml
dns_mode: hosts
default_php: "8.2"
tld: test
portainer: false
editor: code
auto_start: true
```

## Setting Values

```bash
magebox config set <key> <value>
```

Examples:
```bash
magebox config set dns_mode dnsmasq
magebox config set default_php 8.3
magebox config set portainer true
```

## Configuration Options

### dns_mode

DNS resolution method for `.test` domains.

| Value | Description |
|-------|-------------|
| `hosts` | Modify `/etc/hosts` for each domain (default) |
| `dnsmasq` | Use dnsmasq for wildcard `*.test` resolution |

```bash
magebox config set dns_mode dnsmasq
```

### default_php

Default PHP version for new projects.

```bash
magebox config set default_php 8.3
```

When you run `magebox init`, this version will be used unless specified otherwise.

### tld

Top-level domain for projects (default: `test`).

```bash
magebox config set tld local
```

::: warning
Changing TLD requires updating DNS configuration and regenerating SSL certificates.
:::

### portainer

Enable Portainer Docker management UI.

```bash
magebox config set portainer true
```

Access at `http://localhost:9000` after enabling.

### editor

Preferred text editor for opening configuration files.

```bash
magebox config set editor vim
magebox config set editor "code -w"
magebox config set editor nano
```

### auto_start

Automatically start global services when running project commands.

```bash
magebox config set auto_start true
```

## Initializing Configuration

Reset to defaults:

```bash
magebox config init
```

This creates a fresh `~/.magebox/config.yaml` with default values.

## Manual Editing

You can also edit the config file directly:

```bash
nano ~/.magebox/config.yaml
```

```yaml
dns_mode: hosts
default_php: "8.2"
tld: test
portainer: false
editor: code
auto_start: true
```

## Directory Structure

Global configuration creates this structure:

```
~/.magebox/
├── config.yaml          # This file
├── certs/               # SSL certificates
├── nginx/
│   └── vhosts/          # Nginx configurations
├── php/
│   └── pools/           # PHP-FPM pool configs
├── docker/
│   ├── docker-compose.yml
│   └── .env
└── run/                 # Runtime files (sockets, PIDs)
```

## Environment Variables

Some settings can be overridden via environment variables:

| Variable | Config Key |
|----------|------------|
| `MAGEBOX_DNS_MODE` | dns_mode |
| `MAGEBOX_DEFAULT_PHP` | default_php |
| `MAGEBOX_TLD` | tld |

Example:
```bash
MAGEBOX_DEFAULT_PHP=8.4 magebox init mystore
```

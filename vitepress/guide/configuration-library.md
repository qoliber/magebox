# Configuration Library

MageBox uses a separate configuration library (`magebox-lib`) to store platform-specific YAML configurations and templates. This allows updating configurations without rebuilding MageBox.

## Overview

The configuration library contains:
- **Installer configurations** - Platform-specific YAML files defining packages, services, and commands for Fedora, Ubuntu, Arch, and macOS
- **Templates** - Configuration templates for Nginx, PHP-FPM, Varnish, and other services

```
~/.magebox/yaml/
├── installers/
│   ├── fedora.yaml
│   ├── ubuntu.yaml
│   ├── arch.yaml
│   └── darwin.yaml
├── templates/
│   ├── nginx/
│   ├── php/
│   ├── varnish/
│   ├── dns/
│   └── ...
└── version.txt
```

## Library Commands

### Check Library Status

```bash
mbox lib status
```

Shows:
- Current library version
- Git branch and commit (if installed via git)
- Whether updates are available
- Custom path (if configured)

### Update Library

```bash
mbox lib update
```

Pulls the latest configuration files from the magebox-lib repository.

### Show Library Path

```bash
mbox lib path
```

Displays the filesystem path to the configuration library.

### List Available Installers

```bash
mbox lib list
```

Shows all available platform installer configurations.

### List Available Templates

```bash
mbox lib templates
```

Lists all available configuration templates organized by category (nginx, php, varnish, etc.).

### Show Installer Details

```bash
mbox lib show [platform]
```

Displays the installer configuration for a platform with variable expansion. If no platform is specified, auto-detects the current platform.

```bash
# Show current platform config
mbox lib show

# Show specific platform
mbox lib show fedora
```

### Reset Library

```bash
mbox lib reset
```

Discards all local changes and resets the library to the upstream version.

## Custom Library Path

You can use your own templates and installer configurations instead of the default library.

### Set Custom Path

```bash
mbox lib set ~/my-magebox-configs
mbox lib set /path/to/custom/lib
```

The custom path should contain:
- `templates/` - Template files organized by category
- `installers/` - Platform-specific YAML configuration files

### Remove Custom Path

```bash
mbox lib unset
```

Reverts to using the default `~/.magebox/yaml` directory.

## Directory Structure

### Installers

Each platform has its own YAML configuration:

```yaml
# ~/.magebox/yaml/installers/fedora.yaml
schema_version: "1.0"

meta:
  platform: linux
  distro: fedora
  display_name: "Fedora Linux"
  supported_versions: ["40", "41", "42"]

package_manager:
  name: dnf
  install: "sudo dnf install -y"
  update: "sudo dnf update -y"

php:
  version_format: "php${versionNoDot}"
  versions: ["8.1", "8.2", "8.3", "8.4"]
  packages:
    core:
      - "${phpPrefix}-php-fpm"
      - "${phpPrefix}-php-cli"
    extensions:
      - "${phpPrefix}-php-mysqlnd"
      - "${phpPrefix}-php-xml"
      # ... more extensions
  paths:
    binary: "/usr/bin/php${versionNoDot}"
    ini: "/etc/opt/remi/php${versionNoDot}/php.ini"
  services:
    fpm:
      name: "php${versionNoDot}-php-fpm"
      start: "sudo systemctl start ${serviceName}"
      reload: "sudo systemctl reload ${serviceName}"

nginx:
  packages: ["nginx"]
  paths:
    config: "/etc/nginx/nginx.conf"
  services:
    nginx:
      name: nginx
      reload: "sudo systemctl reload nginx"

selinux:
  enabled: true
  booleans:
    - "httpd_can_network_connect on"
    - "httpd_read_user_content on"

sudoers:
  enabled: true
  file: "/etc/sudoers.d/magebox"
  rules:
    - "${user} ALL=(ALL) NOPASSWD: /usr/bin/systemctl reload nginx"
```

### Templates

Templates are organized by service:

```
templates/
├── nginx/
│   ├── vhost.conf.tmpl          # Main vhost template
│   └── upstream.conf.tmpl       # PHP-FPM upstream
├── php/
│   ├── pool.conf.tmpl           # PHP-FPM pool config
│   └── xdebug.ini.tmpl          # Xdebug configuration
├── varnish/
│   └── vcl.vcl.tmpl             # Varnish VCL
├── dns/
│   ├── dnsmasq.conf.tmpl        # dnsmasq config
│   └── hosts.tmpl               # hosts file entry
├── ssl/
│   ├── mkcert.sh.tmpl           # Certificate generation
│   └── trust.sh.tmpl            # CA trust script
├── project/
│   └── env.php.tmpl             # Magento env.php
├── wrappers/
│   └── php-wrapper.sh.tmpl      # PHP CLI wrapper
└── xdebug/
    └── xdebug.ini.tmpl          # Xdebug configuration
```

## Variable Substitution

Templates and installer configurations support variable substitution:

| Variable | Example | Description |
|----------|---------|-------------|
| `${user}` | `jakub` | Current username |
| `${homeDir}` | `/home/jakub` | User home directory |
| `${mageboxDir}` | `/home/jakub/.magebox` | MageBox directory |
| `${phpVersion}` | `8.3` | PHP version being used |
| `${versionNoDot}` | `83` | PHP version without dots |
| `${phpPrefix}` | `php83` | PHP package prefix |
| `${tld}` | `test` | Top-level domain |
| `${osVersion}` | `42` | OS version number |

## Local Overrides

You can override specific templates or installers without modifying the main library:

```
~/.magebox/yaml-local/
├── installers/
│   └── fedora.yaml    # Custom Fedora overrides
└── templates/
    └── nginx/
        └── vhost.conf.tmpl    # Custom nginx template
```

Local overrides take precedence over the main library files.

### Override Priority

1. **Local overrides** (`~/.magebox/yaml-local/`) - Highest priority
2. **Custom library** (`mbox lib set <path>`) - If configured
3. **Default library** (`~/.magebox/yaml/`) - Standard installation
4. **Embedded fallbacks** - Built into MageBox binary as last resort

## Customizing Templates

### Example: Custom Nginx Template

Create a local override:

```bash
mkdir -p ~/.magebox/yaml-local/templates/nginx
cp ~/.magebox/yaml/templates/nginx/vhost.conf.tmpl ~/.magebox/yaml-local/templates/nginx/
```

Edit the local copy to add custom configuration:

```nginx
# ~/.magebox/yaml-local/templates/nginx/vhost.conf.tmpl
server {
    listen 80;
    server_name {{ .Domain }};

    # Custom header
    add_header X-Custom-Header "My Custom Config";

    # ... rest of template
}
```

Restart your project to apply:

```bash
mbox restart
```

### Example: Custom Installer Configuration

For a development setup with additional packages:

```yaml
# ~/.magebox/yaml-local/installers/fedora.yaml
php:
  packages:
    extensions:
      # Add custom extensions
      - "${phpPrefix}-php-xhprof"
      - "${phpPrefix}-php-pcov"
```

## Benefits

1. **Instant updates** - Fix bootstrap issues by updating the library, no MageBox rebuild needed
2. **Transparency** - See exactly what commands will run before bootstrap
3. **Customization** - Override any template or configuration locally
4. **Contribution** - Submit YAML changes to improve MageBox for everyone
5. **Version control** - Library has independent versioning from MageBox
6. **Offline support** - Cached library works without internet
7. **Fallback safety** - Embedded templates ensure MageBox works even if library is missing

## Troubleshooting

### Library Not Installed

If `mbox lib status` shows "Library is not installed":

```bash
mbox lib update
# or
mbox bootstrap
```

### Local Changes Warning

If you have local modifications:

```bash
# View changes
cd ~/.magebox/yaml && git status

# Discard and reset
mbox lib reset
```

### Custom Path Issues

If templates aren't loading from your custom path:

```bash
# Verify path is set
mbox lib path

# Check structure
ls -la $(mbox lib path)/templates/
ls -la $(mbox lib path)/installers/
```

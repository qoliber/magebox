# Linux Installers

MageBox uses a **declarative YAML-based configuration system** to handle the differences between Linux distributions. Platform-specific packages, commands, and paths are defined in external YAML files that can be updated independently of the MageBox binary.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│  magebox-lib (external repository)                          │
│  ├── installers/                                            │
│  │   ├── fedora.yaml                                        │
│  │   ├── ubuntu.yaml                                        │
│  │   ├── arch.yaml                                          │
│  │   └── darwin.yaml                                        │
│  └── templates/                                             │
│      ├── nginx/                                             │
│      ├── php/                                               │
│      └── ...                                                │
└─────────────────────────────────────────────────────────────┘
                          │
                          │ git clone / git pull
                          ▼
┌─────────────────────────────────────────────────────────────┐
│  ~/.magebox/yaml/ (local installation)                      │
│  ├── installers/                                            │
│  └── templates/                                             │
└─────────────────────────────────────────────────────────────┘
```

::: tip
See the [Configuration Library](/guide/configuration-library) guide for detailed information about managing and customizing the library.
:::

## File Locations

### YAML Configuration Files

The installer configurations are stored in the configuration library:

```
~/.magebox/yaml/installers/
├── fedora.yaml   # Fedora/RHEL/CentOS (dnf + Remi)
├── ubuntu.yaml   # Ubuntu/Debian (apt + Ondrej PPA)
├── arch.yaml     # Arch Linux (pacman)
└── darwin.yaml   # macOS (Homebrew)
```

### Go Implementation

The installer code that loads and processes YAML is located in:

```
internal/bootstrap/installer/
├── base.go       # Common functionality for all installers
├── types.go      # Interfaces, types, and supported versions
├── generic.go    # Generic installer using YAML config
├── darwin.go     # macOS-specific fallback
├── fedora.go     # Fedora-specific fallback
├── ubuntu.go     # Ubuntu-specific fallback
└── arch.go       # Arch-specific fallback
```

## The Installer Interface

Each platform installer implements the `Installer` interface defined in `types.go`:

```go
type Installer interface {
    // Platform returns the platform this installer handles
    Platform() platform.Type

    // Distro returns the Linux distribution (empty for Darwin)
    Distro() platform.LinuxDistro

    // ValidateOSVersion checks if the current OS version is supported
    ValidateOSVersion() (OSVersionInfo, error)

    // InstallPrerequisites installs system prerequisites (curl, git, etc.)
    InstallPrerequisites() error

    // InstallPHP installs a specific PHP version
    InstallPHP(version string) error

    // InstallNginx installs Nginx
    InstallNginx() error

    // InstallMkcert installs mkcert for SSL certificates
    InstallMkcert() error

    // InstallDocker installs Docker (instructions only)
    InstallDocker() string

    // InstallDnsmasq installs dnsmasq for DNS resolution
    InstallDnsmasq() error

    // ConfigurePHPFPM configures PHP-FPM for the platform
    ConfigurePHPFPM(versions []string) error

    // ConfigureNginx configures Nginx for MageBox
    ConfigureNginx() error

    // ConfigureSudoers sets up passwordless sudo for services
    ConfigureSudoers() error

    // SetupDNS configures DNS resolution for .test domains
    SetupDNS() error

    // PackageManager returns the package manager name
    PackageManager() string

    // InstallCommand returns the install command format
    InstallCommand(packages ...string) string
}
```

## Supported Versions

Supported OS versions are defined in `types.go`:

```go
var SupportedVersions = map[platform.Type]map[string][]string{
    platform.Darwin: {
        "macos": {"12", "13", "14", "15"}, // Monterey, Ventura, Sonoma, Sequoia
    },
    platform.Linux: {
        "fedora": {"38", "39", "40", "41", "42"},
        "ubuntu": {"20.04", "22.04", "24.04"},    // LTS versions
        "debian": {"11", "12"},                   // Bullseye, Bookworm
        "arch":   {"rolling"},
    },
}

var PHPVersions = []string{"8.1", "8.2", "8.3", "8.4", "8.5"}
```

## PHP Package Naming Conventions

Different distributions use different naming conventions for PHP packages. Understanding these is crucial for extending MageBox.

### Ubuntu/Debian (Ondrej PPA)

Uses version numbers **with dots** in the package name:

```bash
# Pattern: php{VERSION}-{EXTENSION}
php8.2-fpm
php8.2-cli
php8.2-mysql
php8.2-xml
php8.2-curl
php8.2-mbstring
php8.2-zip
php8.2-gd
php8.2-intl
php8.2-bcmath
php8.2-soap
php8.2-opcache
php8.2-sodium
```

Service name: `php8.2-fpm`

Config path: `/etc/php/8.2/fpm/php-fpm.conf`

### Fedora/RHEL (Remi Repository)

Uses version numbers **without dots** in the package name:

```bash
# Pattern: php{VERSION_NO_DOT}-php-{EXTENSION}
php82-php-fpm
php82-php-cli
php82-php-mysqlnd
php82-php-xml
php82-php-mbstring
php82-php-zip
php82-php-gd
php82-php-intl
php82-php-bcmath
php82-php-soap
php82-php-opcache
php82-php-sodium
```

System service name: `php82-php-fpm`

Config path: `/etc/opt/remi/php82/php-fpm.conf`

::: warning PHP-FPM Management on Fedora
MageBox uses **direct process management** for PHP-FPM on Fedora, NOT systemd. This is intentional due to SELinux context issues:

- When PHP-FPM runs via systemd, it gets the `httpd_t` SELinux context
- The `httpd_t` context cannot access files in user home directories (`user_home_t`)
- Even with `httpd_read_user_content=on`, symlinks and certain operations are blocked

**How MageBox manages PHP-FPM on Fedora:**
```
~/.magebox/php/php-fpm-8.3.conf     # Master config
~/.magebox/php/pools/8.3/*.conf     # Pool configs per project
~/.magebox/run/*.sock               # Unix sockets
~/.magebox/run/php-fpm-8.3.pid      # PID file
```

MageBox starts PHP-FPM directly as your user:
```bash
php-fpm -y ~/.magebox/php/php-fpm-8.3.conf
```

This runs PHP-FPM with `unconfined_t` SELinux context, giving full access to your project files.
:::

::: tip PHP-FPM Management on Debian/Ubuntu
On Debian-based systems (Ubuntu, Linux Mint, Pop!_OS, etc.), MageBox uses **systemd** for PHP-FPM management. AppArmor (used instead of SELinux) is more permissive and allows access to user home directories.

```bash
# MageBox uses systemd on Debian/Ubuntu:
sudo systemctl start php8.3-fpm
sudo systemctl reload php8.3-fpm
```

Pool configs are still stored in `~/.magebox/php/pools/{version}/` and included by the system PHP-FPM service.
:::

::: tip SELinux Configuration
Fedora has SELinux enabled by default. MageBox bootstrap automatically configures:
- `setsebool -P httpd_can_network_connect on` - allows nginx to proxy to Docker
- `setsebool -P httpd_read_user_content on` - allows nginx to read files from home directories
- `chcon -R -t httpd_config_t ~/.magebox/nginx/` - allows nginx to read vhost configs
- `chcon -R -t httpd_config_t ~/.magebox/certs/` - allows nginx to read SSL certs

See [Troubleshooting: SELinux](/guide/troubleshooting#selinux-issues-fedora-rhel) for manual fixes.
:::

### Arch Linux (Official Repos)

Uses a **single PHP version** in official repos:

```bash
# No version in package name (only one version available)
php
php-fpm
php-gd
php-intl
php-sodium
```

Service name: `php-fpm`

Config path: `/etc/php/php-fpm.conf`

::: warning Arch Linux Limitation
Arch Linux official repositories only provide one PHP version at a time. For multiple PHP versions, users need to use AUR packages or manual compilation.
:::

### macOS (Homebrew)

Uses the `php@{VERSION}` formula naming:

```bash
# Pattern: php@{VERSION}
php@8.1
php@8.2
php@8.3
php@8.4
```

Service: `brew services start php@8.2`

## YAML Installer Schema

Each platform installer is defined in a YAML file with the following structure:

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
  version_format: "php${versionNoDot}"  # php82, php83, etc.
  versions: ["8.1", "8.2", "8.3", "8.4"]
  packages:
    core:
      - "${phpPrefix}-php-fpm"
      - "${phpPrefix}-php-cli"
    extensions:
      - "${phpPrefix}-php-mysqlnd"
      - "${phpPrefix}-php-xml"
      - "${phpPrefix}-php-mbstring"
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
    tmp_dir: "/var/lib/nginx/tmp/"
  services:
    nginx:
      name: nginx
      reload: "sudo systemctl reload nginx"
  fixes:
    tmp_permissions:
      commands:
        - "sudo chown -R ${user}:${user} /var/lib/nginx/tmp/"

selinux:
  enabled: true
  booleans:
    - "httpd_can_network_connect on"
    - "httpd_read_user_content on"
  contexts:
    - path: "${mageboxDir}/run"
      type: "httpd_var_run_t"

sudoers:
  enabled: true
  file: "/etc/sudoers.d/magebox"
  rules:
    - "${user} ALL=(ALL) NOPASSWD: /usr/bin/systemctl reload nginx"
    - "${user} ALL=(ALL) NOPASSWD: /usr/bin/systemctl start php*-php-fpm"
```

### Available Variables

| Variable | Example | Description |
|----------|---------|-------------|
| `${user}` | `jakub` | Current user |
| `${homeDir}` | `/home/jakub` | User home directory |
| `${mageboxDir}` | `/home/jakub/.magebox` | MageBox directory |
| `${phpVersion}` | `8.2` | PHP version being installed |
| `${versionNoDot}` | `82` | PHP version without dots |
| `${phpPrefix}` | `php82` | Computed from version_format |
| `${tld}` | `test` | Top-level domain from config |
| `${osVersion}` | `42` | OS version number |

### View Current Configuration

Use `mbox lib show` to see the resolved configuration for your platform:

```bash
mbox lib show fedora
```

This displays package names, paths, and commands with all variables expanded.

## Adding Support for a New Distribution

To add support for a new Linux distribution, you can now simply create a YAML configuration file.

### Option 1: YAML Configuration Only

Create a new YAML file in the configuration library:

```yaml
# ~/.magebox/yaml/installers/opensuse.yaml
schema_version: "1.0"

meta:
  platform: linux
  distro: opensuse
  display_name: "openSUSE"
  supported_versions: ["15.5", "15.6", "tumbleweed"]

package_manager:
  name: zypper
  install: "sudo zypper install -y"
  update: "sudo zypper update -y"

php:
  version_format: "php${versionNoDot}"
  versions: ["8.2", "8.3"]
  packages:
    core:
      - "php${versionNoDot}"
      - "php${versionNoDot}-fpm"
    extensions:
      - "php${versionNoDot}-mysql"
      - "php${versionNoDot}-xml"
      # ... more packages
  paths:
    binary: "/usr/bin/php${versionNoDot}"
    ini: "/etc/php${versionNoDot}/php.ini"
  services:
    fpm:
      name: "php${versionNoDot}-fpm"

nginx:
  packages: ["nginx"]
  paths:
    config: "/etc/nginx/nginx.conf"

sudoers:
  enabled: true
  file: "/etc/sudoers.d/magebox"
  rules:
    - "${user} ALL=(ALL) NOPASSWD: /usr/bin/systemctl reload nginx"
```

### Option 2: Go Implementation (Advanced)

For distributions requiring special logic, create a Go installer file:

Create a new file in `internal/bootstrap/installer/`, e.g., `opensuse.go`:

```go
package installer

import (
    "fmt"
    "os"
    "strings"

    "github.com/qoliber/magebox/internal/platform"
)

type OpenSUSEInstaller struct {
    BaseInstaller
}

func NewOpenSUSEInstaller(p *platform.Platform) *OpenSUSEInstaller {
    return &OpenSUSEInstaller{
        BaseInstaller: BaseInstaller{Platform: p},
    }
}

func (o *OpenSUSEInstaller) Platform() platform.Type {
    return platform.Linux
}

func (o *OpenSUSEInstaller) Distro() platform.LinuxDistro {
    return platform.DistroOpenSUSE // Add to platform package
}

func (o *OpenSUSEInstaller) ValidateOSVersion() (OSVersionInfo, error) {
    // Read /etc/os-release and check version
    // ...
}

func (o *OpenSUSEInstaller) InstallPHP(version string) error {
    // OpenSUSE uses zypper and different package names
    // Research the correct package naming for your distro
    packages := []string{
        fmt.Sprintf("php%s", strings.ReplaceAll(version, ".", "")),
        fmt.Sprintf("php%s-fpm", strings.ReplaceAll(version, ".", "")),
        // ... more packages
    }
    args := append([]string{"zypper", "install", "-y"}, packages...)
    return o.RunSudo(args...)
}

// Implement all other interface methods...
```

### 2. Add to Supported Versions

Update `types.go`:

```go
var SupportedVersions = map[platform.Type]map[string][]string{
    // ...
    platform.Linux: {
        // ...existing distros...
        "opensuse": {"15.5", "15.6", "tumbleweed"},
    },
}
```

### 3. Add Distro Detection

Update `internal/platform/platform.go` to detect the new distribution:

```go
const (
    DistroFedora  LinuxDistro = "fedora"
    DistroDebian  LinuxDistro = "debian"
    DistroArch    LinuxDistro = "arch"
    DistroOpenSUSE LinuxDistro = "opensuse" // Add this
)
```

### 4. Register the Installer

Update the installer factory to create instances of your new installer when the distribution is detected.

## Base Installer Helpers

The `BaseInstaller` struct provides common functionality:

```go
type BaseInstaller struct {
    Platform *platform.Platform
}

// RunCommand executes a shell command
func (b *BaseInstaller) RunCommand(cmdStr string) error

// RunSudo executes a command with sudo
func (b *BaseInstaller) RunSudo(args ...string) error

// FileExists checks if a file exists
func (b *BaseInstaller) FileExists(path string) bool

// WriteFile writes content to a file using sudo
func (b *BaseInstaller) WriteFile(path, content string) error

// CommandExists checks if a command is available
func (b *BaseInstaller) CommandExists(name string) bool
```

## Service Configuration Patterns

### PHP-FPM Logging

All Linux installers configure centralized logging to `/var/log/magebox/`:

```go
func (f *FedoraInstaller) ConfigurePHPFPM(versions []string) error {
    // Create log directory
    if err := f.RunSudo("mkdir", "-p", "/var/log/magebox"); err != nil {
        return err
    }
    if err := f.RunSudo("chmod", "755", "/var/log/magebox"); err != nil {
        return err
    }

    for _, v := range versions {
        // Update error_log path in php-fpm.conf
        // Enable and restart service
    }
    return nil
}
```

### Nginx User Configuration

Linux installers configure nginx to run as the current user (for SSL cert access):

```go
func (f *FedoraInstaller) ConfigureNginx() error {
    currentUser := os.Getenv("USER")
    if currentUser != "" {
        nginxConf := "/etc/nginx/nginx.conf"
        return f.RunSudo("sed", "-i",
            fmt.Sprintf("s/^user .*/user %s;/", currentUser),
            nginxConf)
    }
    return nil
}
```

### Sudoers Configuration

Linux installers set up passwordless sudo for service control:

```go
func (f *FedoraInstaller) ConfigureSudoers() error {
    sudoersContent := fmt.Sprintf(`# MageBox - Allow %[1]s to control services
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl start nginx
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl stop nginx
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl reload nginx
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart nginx
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl start php*-fpm
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl stop php*-fpm
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl reload php*-fpm
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart php*-fpm
`, currentUser)

    return f.WriteFile("/etc/sudoers.d/magebox", sudoersContent)
}
```

## Testing Your Installer

When adding support for a new distribution:

1. **Test on a clean VM** - Use a fresh installation of the target OS
2. **Verify PHP installation** - Check that all PHP versions install correctly
3. **Test service management** - Ensure start/stop/restart work without password prompts
4. **Verify DNS resolution** - Test that `*.test` domains resolve correctly
5. **Test SSL certificates** - Ensure mkcert and certificate generation work
6. **Run the full bootstrap** - `magebox bootstrap` should complete without errors

## Contributing

If you add support for a new distribution, please:

1. Follow the existing code patterns
2. Add comprehensive error handling
3. Test on actual hardware/VMs (not just containers)
4. Submit a pull request with documentation updates
5. Add the distribution to the [Bootstrap](/guide/bootstrap) supported versions list

See the [GitHub repository](https://github.com/qoliber/magebox) for contribution guidelines.

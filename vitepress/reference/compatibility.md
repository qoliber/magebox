# Compatibility Matrix

Version compatibility information for MageBox, Magento, and related technologies.

## Magento / Adobe Commerce PHP Compatibility

| Magento Version | PHP 8.1 | PHP 8.2 | PHP 8.3 | PHP 8.4 | Recommended |
|-----------------|---------|---------|---------|---------|-------------|
| 2.4.7-p4        | :white_check_mark: | :white_check_mark: | :white_check_mark: | :x: | PHP 8.3 |
| 2.4.7           | :white_check_mark: | :white_check_mark: | :white_check_mark: | :x: | PHP 8.3 |
| 2.4.6-p8        | :white_check_mark: | :white_check_mark: | :x: | :x: | PHP 8.2 |
| 2.4.6           | :white_check_mark: | :white_check_mark: | :x: | :x: | PHP 8.2 |
| 2.4.5-p10       | :white_check_mark: | :x: | :x: | :x: | PHP 8.1 |
| 2.4.5           | :white_check_mark: | :x: | :x: | :x: | PHP 8.1 |
| 2.4.4-p11       | :white_check_mark: | :x: | :x: | :x: | PHP 8.1 |
| 2.4.4           | :white_check_mark: | :x: | :x: | :x: | PHP 8.1 |

::: tip
MageBox installs PHP 8.1, 8.2, 8.3, and 8.4 by default during bootstrap. Switch between versions per-project using `magebox php 8.x`.
:::

## MageOS PHP Compatibility

| MageOS Version | PHP 8.1 | PHP 8.2 | PHP 8.3 | PHP 8.4 | Recommended |
|----------------|---------|---------|---------|---------|-------------|
| 1.0.x          | :white_check_mark: | :white_check_mark: | :white_check_mark: | :x: | PHP 8.3 |

## Database Compatibility

### MySQL

| Magento Version | MySQL 5.7 | MySQL 8.0 | MySQL 8.4 | Recommended |
|-----------------|-----------|-----------|-----------|-------------|
| 2.4.7+          | :x: | :white_check_mark: | :white_check_mark: | MySQL 8.4 |
| 2.4.6           | :white_check_mark: | :white_check_mark: | :x: | MySQL 8.0 |
| 2.4.5           | :white_check_mark: | :white_check_mark: | :x: | MySQL 8.0 |
| 2.4.4           | :white_check_mark: | :white_check_mark: | :x: | MySQL 8.0 |

### MariaDB

| Magento Version | MariaDB 10.4 | MariaDB 10.6 | MariaDB 11.4 | Recommended |
|-----------------|--------------|--------------|--------------|-------------|
| 2.4.7+          | :x: | :white_check_mark: | :white_check_mark: | MariaDB 11.4 |
| 2.4.6           | :white_check_mark: | :white_check_mark: | :x: | MariaDB 10.6 |
| 2.4.5           | :white_check_mark: | :white_check_mark: | :x: | MariaDB 10.6 |
| 2.4.4           | :white_check_mark: | :white_check_mark: | :x: | MariaDB 10.6 |

::: info
MageBox uses Docker for database services. Configure your database in `.magebox.yaml`:

```yaml
services:
  mysql: "8.4"    # MySQL version
  # or
  mariadb: "11.4" # MariaDB version
```
:::

## Search Engine Compatibility

| Magento Version | OpenSearch 2.x | Elasticsearch 7.x | Elasticsearch 8.x |
|-----------------|----------------|-------------------|-------------------|
| 2.4.7+          | :white_check_mark: | :white_check_mark: | :white_check_mark: |
| 2.4.6           | :white_check_mark: | :white_check_mark: | :white_check_mark: |
| 2.4.5           | :white_check_mark: | :white_check_mark: | :x: |
| 2.4.4           | :white_check_mark: | :white_check_mark: | :x: |

::: tip
OpenSearch is recommended as it's fully open-source and actively maintained. Configure in `.magebox.yaml`:

```yaml
services:
  opensearch: "2.19"  # OpenSearch (recommended)
  # or
  elasticsearch: "8"  # Elasticsearch
```
:::

## Operating System Support

### macOS

| macOS Version | Intel | Apple Silicon | Status |
|---------------|-------|---------------|--------|
| 15 Sequoia    | :white_check_mark: | :white_check_mark: | Fully supported |
| 14 Sonoma     | :white_check_mark: | :white_check_mark: | Fully supported |
| 13 Ventura    | :white_check_mark: | :white_check_mark: | Fully supported |
| 12 Monterey   | :white_check_mark: | :white_check_mark: | Supported |

### Linux

| Distribution | Versions | Status |
|--------------|----------|--------|
| **Fedora** | 38, 39, 40, 41, 42, 43 | Fully supported |
| **Ubuntu** | 20.04 LTS, 22.04 LTS, 24.04 LTS | Fully supported |
| **Debian** | 11 (Bullseye), 12 (Bookworm) | Fully supported |
| **Arch Linux** | Rolling | Supported |
| **Rocky Linux** | 9 | Supported |
| **EndeavourOS** | Rolling | Supported (Arch-based) |
| **Pop!_OS** | 22.04 | Supported (Ubuntu-based) |

### Windows

| Platform | Status |
|----------|--------|
| WSL2 (Ubuntu) | Supported |
| WSL2 (Fedora) | Supported |
| Native Windows | Not supported |

## Docker Provider Support (macOS)

| Provider | Status | Notes |
|----------|--------|-------|
| Docker Desktop | :white_check_mark: | Default, GUI included |
| Colima | :white_check_mark: | Lightweight, CLI-only |
| OrbStack | :white_check_mark: | Fast, native-like performance |
| Rancher Desktop | :white_check_mark: | Kubernetes included |
| Lima | :white_check_mark: | Minimal, advanced users |

Switch providers with `magebox docker use <provider>`.

## PHP Extension Requirements

MageBox installs all required Magento PHP extensions during bootstrap:

| Extension | Required | Notes |
|-----------|----------|-------|
| bcmath | :white_check_mark: | Mathematical functions |
| ctype | :white_check_mark: | Character type checking |
| curl | :white_check_mark: | URL transfers |
| dom | :white_check_mark: | DOM manipulation |
| gd | :white_check_mark: | Image processing |
| hash | :white_check_mark: | Hashing |
| iconv | :white_check_mark: | Character encoding |
| intl | :white_check_mark: | Internationalization |
| json | :white_check_mark: | JSON handling |
| libxml | :white_check_mark: | XML processing |
| mbstring | :white_check_mark: | Multibyte strings |
| openssl | :white_check_mark: | SSL/TLS |
| pcre | :white_check_mark: | Regular expressions |
| pdo_mysql | :white_check_mark: | MySQL PDO driver |
| simplexml | :white_check_mark: | SimpleXML |
| soap | :white_check_mark: | SOAP protocol |
| sodium | :white_check_mark: | Cryptography |
| spl | :white_check_mark: | Standard PHP Library |
| tokenizer | :white_check_mark: | PHP tokenizer |
| xmlwriter | :white_check_mark: | XML writing |
| xsl | :white_check_mark: | XSL transformations |
| zip | :white_check_mark: | ZIP archives |
| imagick | :white_check_mark: | ImageMagick (v0.17.2+) |

### Optional Extensions

| Extension | Purpose |
|-----------|---------|
| xdebug | Step debugging (`magebox xdebug on`) |
| blackfire | Profiling (`magebox blackfire on`) |
| tideways | Profiling (`magebox tideways on`) |
| opcache | Performance (enabled in prod mode) |

## Service Ports

Default ports used by MageBox Docker services:

| Service | Port | Notes |
|---------|------|-------|
| MySQL 5.7 | 33057 | |
| MySQL 8.0 | 33080 | |
| MySQL 8.4 | 33084 | |
| MariaDB 10.4 | 33104 | |
| MariaDB 10.6 | 33106 | |
| MariaDB 11.4 | 33114 | |
| Redis | 6379 | Default Redis port |
| OpenSearch/Elasticsearch | 9200, 9300 | HTTP, Transport |
| RabbitMQ | 5672, 15672 | AMQP, Management UI |
| Mailpit | 1025, 8025 | SMTP, Web UI |
| Varnish | 8080 | HTTP cache |
| Portainer | 9000 | Container management UI |

## Version History

See the [Changelog](/changelog) for a complete version history.

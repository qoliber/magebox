# Configuration Options

Complete reference for all configuration options.

## Project Configuration (.magebox.yaml)

### name

**Required** | `string`

Project identifier used for database name, container naming, and configuration files.

```yaml
name: mystore
```

---

### domains

**Required** | `array`

List of domain configurations for the project.

```yaml
domains:
  - host: mystore.test
    root: pub
    ssl: true
  - host: de.mystore.test
    root: pub
    store_code: german
```

#### Domain Properties

| Property | Type | Default | Description |
|----------|------|---------|-------------|
| `host` | string | required | Domain name |
| `root` | string | `pub` | Document root relative to project |
| `ssl` | boolean | `true` | Enable HTTPS |
| `store_code` | string | `default` | Magento store code (sets `MAGE_RUN_CODE`) |

---

### php

**Required** | `string`

PHP version for this project.

```yaml
php: "8.2"
```

**Supported values:** `"8.1"`, `"8.2"`, `"8.3"`, `"8.4"`

---

### services

`object`

Docker services configuration.

```yaml
services:
  mysql: "8.0"
  redis: true
  opensearch: "2.19.4"
  rabbitmq: true
  mailpit: true
  varnish: false
```

#### Database Options

| Option | Values | Ports |
|--------|--------|-------|
| `mysql` | `"5.7"`, `"8.0"`, `"8.4"` | 33057, 33080, 33084 |
| `mariadb` | `"10.4"`, `"10.6"`, `"11.4"` | 33104, 33106, 33114 |

::: warning
Use either `mysql` OR `mariadb`, not both.
:::

#### Other Services

| Option | Type | Port(s) | Description |
|--------|------|---------|-------------|
| `redis` | boolean | 6379 | In-memory cache/session |
| `opensearch` | string/boolean | 9200 | Catalog search |
| `elasticsearch` | string/boolean | 9200 | Catalog search (alternative) |
| `rabbitmq` | boolean | 5672, 15672 | Message queue |
| `mailpit` | boolean | 1025, 8025 | Email testing |
| `varnish` | boolean | 6081 | HTTP cache |

---

### env

`object`

Environment variables passed to PHP-FPM.

```yaml
env:
  MAGE_MODE: developer
  COMPOSER_MEMORY_LIMIT: -1
  XDEBUG_MODE: debug
```

---

### commands

`object`

Custom commands for the project.

```yaml
commands:
  # Simple format
  reindex: "php bin/magento indexer:reindex"

  # Extended format
  deploy:
    description: "Deploy to production"
    run: |
      php bin/magento deploy:mode:set production
      php bin/magento cache:flush
```

#### Command Properties

| Property | Type | Description |
|----------|------|-------------|
| `description` | string | Help text for the command |
| `run` | string | Command(s) to execute |

---

## Global Configuration (~/.magebox/config.yaml)

### dns_mode

`string` | Default: `"dnsmasq"` (since v0.16.6)

DNS resolution method.

```yaml
dns_mode: dnsmasq
```

| Value | Description |
|-------|-------------|
| `dnsmasq` | Use dnsmasq for wildcard *.test resolution (default) |
| `hosts` | Modify /etc/hosts for each domain (fallback) |

---

### default_php

`string` | Default: `"8.2"`

Default PHP version for new projects.

```yaml
default_php: "8.3"
```

---

### tld

`string` | Default: `"test"`

Top-level domain for projects.

```yaml
tld: test
```

---

### portainer

`boolean` | Default: `false`

Enable Portainer Docker management UI.

```yaml
portainer: true
```

Access at http://localhost:9000 when enabled.

---

### editor

`string` | Default: platform default

Preferred text editor.

```yaml
editor: code
editor: vim
editor: "code -w"
```

---

### auto_start

`boolean` | Default: `true`

Automatically start global services when running project commands.

```yaml
auto_start: true
```

---

## Local Overrides (.magebox.local.yaml)

Override any project setting locally without affecting the shared configuration.

```yaml
# .magebox.local.yaml
php: "8.3"

services:
  rabbitmq: false

env:
  XDEBUG_MODE: debug

commands:
  my-test: "php vendor/bin/phpunit tests/MyTest"
```

### Merge Behavior

Local settings are merged with project settings:

- Scalar values (strings, numbers, booleans) are replaced
- Objects are deeply merged
- Arrays replace the original (not appended)

### Example Merge

**.magebox.yaml:**
```yaml
php: "8.2"
services:
  mysql: "8.0"
  redis: true
env:
  MAGE_MODE: developer
```

**.magebox.local.yaml:**
```yaml
php: "8.3"
services:
  redis: false
env:
  XDEBUG_MODE: debug
```

**Result:**
```yaml
php: "8.3"                    # Replaced
services:
  mysql: "8.0"                # Kept from original
  redis: false                # Replaced
env:
  MAGE_MODE: developer        # Kept from original
  XDEBUG_MODE: debug          # Added
```

---

## Environment Variables

Some settings can be overridden via environment variables:

| Variable | Config Key | Description |
|----------|------------|-------------|
| `MAGEBOX_DNS_MODE` | dns_mode | Override DNS mode |
| `MAGEBOX_DEFAULT_PHP` | default_php | Override default PHP |
| `MAGEBOX_TLD` | tld | Override TLD |

Example:

```bash
MAGEBOX_DEFAULT_PHP=8.4 magebox init mystore
```

---

## Complete Example

### .magebox.yaml

```yaml
name: acme-store

domains:
  - host: acme.test
    root: pub
    ssl: true
  - host: api.acme.test
    root: pub
    ssl: true

php: "8.2"

services:
  mysql: "8.0"
  redis: true
  opensearch: "2.19.4"
  rabbitmq: true
  mailpit: true
  varnish: false

env:
  MAGE_MODE: developer
  COMPOSER_MEMORY_LIMIT: -1

commands:
  cc:
    description: "Clear cache"
    run: "php bin/magento cache:clean"

  ri:
    description: "Reindex"
    run: "php bin/magento indexer:reindex"

  deploy:
    description: "Production deployment"
    run: |
      php bin/magento maintenance:enable
      php bin/magento setup:upgrade
      php bin/magento setup:di:compile
      php bin/magento setup:static-content:deploy -f
      php bin/magento maintenance:disable
```

### ~/.magebox/config.yaml

```yaml
dns_mode: dnsmasq
default_php: "8.2"
tld: test
portainer: false
editor: code
auto_start: true
```

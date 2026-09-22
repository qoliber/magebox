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

### type

`string` | Default: `"magento"`

Project type. Controls default document root detection and Magento-specific behaviour.

```yaml
type: magento   # default
type: laravel   # Laravel project
```

| Value | Document root default | Notes |
|-------|----------------------|-------|
| `magento` | `pub` | Enables Magento-specific commands |
| `laravel` | `public` | Skips Magento-specific setup steps |

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

### isolated

`boolean` | Default: `false`

Run this project's PHP-FPM in a dedicated master process, isolated from other projects using the same PHP version.

```yaml
isolated: true
```

Useful when you need per-project PHP system settings (`opcache.preload`, `opcache.jit`, etc.) that can only be set in `php.ini` and would otherwise affect all projects on the same PHP version.

See [`magebox php isolate`](/reference/commands#magebox-php-isolate) for more details.

---

### services

`object`

Docker services configuration.

```yaml
services:
  mysql: "8.0"
  redis: true          # or valkey: true
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
| `valkey` | boolean | 6379 | In-memory cache/session (Redis alternative) |
| `opensearch` | string/boolean | 9200 | Catalog search |
| `elasticsearch` | string/boolean | 9500 | Catalog search (alternative) |
| `rabbitmq` | boolean | 5672, 15672 | Message queue |
| `mailpit` | boolean | 1025, 8025 | Email testing |
| `varnish` | boolean | 6081 | HTTP cache |

---

### compose_file

`string`

Path to a project-specific Docker Compose file for custom containers.

```yaml
compose_file: docker-compose.yml
```

When configured, `magebox start` and `magebox stop` prompt to start/stop these containers. They are automatically connected to the MageBox Docker network so they can communicate with MySQL, Redis/Valkey, and other MageBox services.

The path is relative to the project root unless an absolute path is given.

---

### include_config

`array of strings`

Split your configuration across multiple files. Each entry is a path (relative to the declaring file) to another YAML config file **or** a directory — when a directory is given, all `.yaml`/`.yml` files inside are loaded in alphabetical order.

```yaml
include_config:
  - ./.magebox/services.yaml
  - ./.magebox/commands.yaml
  - ./.magebox           # auto-include all .yaml/.yml files in the directory
```

Included files are merged in declaration order. Fields set in the current file always take final precedence. Map fields (`env`, `commands`, `php_ini`) and services are deep-merged; for the same key, later entries override earlier ones.

Included files can themselves contain `include_config` entries. Circular includes are detected and rejected with an error.

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

### php_ini

`object`

Per-project PHP INI overrides. Values are injected into the PHP-FPM pool as `php_admin_value` directives, so they apply only to this project's pool and override any global `php.ini` settings.

```yaml
php_ini:
  memory_limit: 1G
  max_execution_time: 300
  tideways.api_key: abc123       # Per-project Tideways API key
  xdebug.mode: debug,develop
```

::: tip
Most settings can also be managed through `magebox php ini set <key> <value>`, which writes to this section automatically.
:::

---

### pm

`object`

PHP-FPM process manager tuning for this project's pool. Every key is optional; anything you leave out falls back to the global `default_pm` and then to the built-in default.

```yaml
pm:
  mode: ondemand           # static | dynamic | ondemand
  max_children: 12
  process_idle_timeout: 30s
```

| Option | Type | Default | Applies to | Description |
|--------|------|---------|------------|-------------|
| `mode` | string | `dynamic` | all | Process manager: `static`, `dynamic` or `ondemand` |
| `max_children` | int | `50` | all | Maximum number of worker processes |
| `max_requests` | int | `1000` | all | Requests a worker handles before respawning (`0` disables) |
| `start_servers` | int | `8` | `dynamic` | Workers created at startup |
| `min_spare_servers` | int | `4` | `dynamic` | Minimum idle workers |
| `max_spare_servers` | int | `12` | `dynamic` | Maximum idle workers |
| `process_idle_timeout` | string | `10s` | `ondemand` | Idle worker lifetime before termination |

#### Choosing a mode

* **`dynamic`** (default) keeps a pool of idle workers warm and scales up under load. Good for the project you are actively working on.
* **`ondemand`** starts no workers until a request arrives and reaps them after `process_idle_timeout`. Useful when a machine hosts many projects: the ones you are not using cost nothing.
* **`static`** keeps exactly `max_children` workers alive at all times. Predictable, but reserves the memory whether you use it or not.

#### Sizing max_children

The default of `50` assumes a pool has the machine to itself. When several projects run side by side, their peaks can collectively oversubscribe the host: PHP-FPM will happily start more workers than there are CPU cores, and each active Magento worker can hold hundreds of megabytes.

Since throughput is bounded by cores rather than workers, a value close to your core count is usually enough for local development. Requests beyond that queue on the socket instead of competing for memory.

::: warning
PHP-FPM refuses to start the entire master process when one pool is malformed, which would take down every project sharing that PHP version. MageBox therefore validates these values before writing the pool file — `min_spare_servers` must not exceed `max_spare_servers`, `max_spare_servers` must not exceed `max_children`, and `start_servers` must fall between the two.

If you only lower `max_children`, MageBox scales the untouched spare-server defaults down to fit rather than erroring.
:::

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

### testing

`object`

Testing tool configuration. Used by `magebox test` commands.

```yaml
testing:
  phpunit:
    config: dev/tests/unit/phpunit.xml
  phpstan:
    level: 5
    paths:
      - app/code/Vendor/Module
  phpcs:
    standard: Magento2
    paths:
      - app/code/Vendor/Module
  phpmd:
    ruleset: cleancode,design
    paths:
      - app/code/Vendor/Module
```

::: tip
Run `magebox test setup` to configure this section interactively.
:::

---

### sandbox

`object`

Bubblewrap sandbox configuration for AI coding agents (`magebox sandbox`). Controls which filesystem paths are accessible inside the sandbox.

```yaml
sandbox:
  tool_profiles:
    claude:
      extra_binds:
        - /path/to/extra/dir
    codex:
      extra_binds: []
```

#### Sandbox Properties

| Property | Type | Description |
|----------|------|-------------|
| `tool_profiles` | object | Per-tool overrides keyed by tool name (`claude`, `codex`) |
| `tool_profiles.<tool>.extra_binds` | array | Extra read-write bind mounts added to the sandbox |

See [`magebox sandbox`](/reference/commands#magebox-sandbox-tool----tool-args) for details on what is accessible by default.

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

### default_pm

`object`

Machine-wide PHP-FPM process manager baseline, applied to every project that does not set its own [`pm`](#pm) block. Accepts the same keys.

```yaml
default_pm:
  mode: ondemand
  max_children: 12
```

This is the place to size PHP-FPM for the host rather than per project. A project's own `pm` block overrides it key by key, so a single project can still run `dynamic` while the rest stay `ondemand`.

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
  redis: true          # or valkey: true
  opensearch: "2.19.4"
  rabbitmq: true
  mailpit: true
  varnish: false

compose_file: docker-compose.yml

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

default_pm:
  mode: ondemand
  max_children: 12
```

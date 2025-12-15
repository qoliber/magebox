# Testing & Code Quality

MageBox provides integrated support for running tests and static analysis tools on your Magento projects. All tools use the correct PHP version configured for your project.

## Overview

| Tool | Command | Description |
|------|---------|-------------|
| PHPUnit | `magebox test unit` | Run unit tests |
| PHPUnit | `magebox test integration` | Run Magento integration tests |
| PHPStan | `magebox test phpstan` | Static analysis |
| PHP_CodeSniffer | `magebox test phpcs` | Code style checking |
| PHP Mess Detector | `magebox test phpmd` | Detect code issues |

## Quick Start

```bash
# Install all testing tools
magebox test setup

# Run all tests (except integration)
magebox test all

# Check what's installed
magebox test status
```

---

## Command Reference

### `magebox test setup`

Install and configure testing tools via Composer.

**Usage:**
```bash
magebox test setup [type]
```

**Arguments:**

| Argument | Description |
|----------|-------------|
| (none) | Interactive wizard - select which tools to install |
| `unit` | Install PHPUnit only |
| `static` | Install static analysis tools (PHPStan, PHPCS, PHPMD) |
| `all` | Install all testing tools |

**Examples:**
```bash
# Interactive wizard
magebox test setup

# Install all tools at once
magebox test setup all

# Install only PHPUnit
magebox test setup unit

# Install only static analysis tools
magebox test setup static
```

**Composer Packages Installed:**

| Tool | Packages |
|------|----------|
| PHPUnit | `phpunit/phpunit` |
| PHPStan | `phpstan/phpstan`, `bitexpert/phpstan-magento` |
| PHPCS | `squizlabs/php_codesniffer`, `magento/magento-coding-standard` |
| PHPMD | `phpmd/phpmd` |

All packages are installed as dev dependencies (`composer require --dev`).

---

### `magebox test unit`

Run PHPUnit unit tests.

**Usage:**
```bash
magebox test unit [options]
```

**Options:**

| Option | Short | Description |
|--------|-------|-------------|
| `--filter` | `-f` | Filter tests by name (regex pattern) |
| `--testsuite` | `-t` | Run specific test suite |

**Examples:**
```bash
# Run all unit tests
magebox test unit

# Run tests matching a pattern
magebox test unit --filter=ProductTest
magebox test unit -f "Test::testMethod"

# Run specific test suite
magebox test unit --testsuite=Unit
magebox test unit -t Unit

# Combine options
magebox test unit --filter=Cart --testsuite=Unit
```

**How It Works:**

1. MageBox locates PHPUnit at `vendor/bin/phpunit`
2. Uses the PHP version from your `.magebox.yaml`
3. Looks for config file in order:
   - Path specified in `.magebox.yaml` (`testing.phpunit.config`)
   - `phpunit.xml`
   - `phpunit.xml.dist`
4. Runs with `--colors=always` for readable output

**Configuration in `.magebox.yaml`:**
```yaml
testing:
  phpunit:
    enabled: true
    config: "phpunit.xml"      # Custom config path
    testsuite: "Unit"          # Default test suite
```

---

### `magebox test integration`

Run Magento integration tests.

::: warning
Integration tests require a separate database and can take a long time to run. They may also modify data, so use a dedicated test database.
:::

::: tip Fast Mode with tmpfs
Use `--tmpfs` to run MySQL entirely in RAM. This can make integration tests **10-100x faster** by eliminating disk I/O!
:::

**Usage:**
```bash
magebox test integration [options]
```

**Options:**

| Option | Short | Description |
|--------|-------|-------------|
| `--filter` | `-f` | Filter tests by name |
| `--testsuite` | `-t` | Run specific test suite |
| `--tmpfs` | | Run MySQL in RAM for faster tests |
| `--tmpfs-size` | | RAM allocation for tmpfs MySQL (default: `1g`) |
| `--mysql-version` | | MySQL version for test container (default: `8.0`) |
| `--keep-alive` | | Keep test container running after tests finish |

**Examples:**
```bash
# Run all integration tests
magebox test integration

# Run specific test
magebox test integration --filter=CartTest

# Run specific test suite
magebox test integration --testsuite=Integration

# FAST: Run with MySQL in RAM (recommended!)
magebox test integration --tmpfs

# Use more RAM for larger test suites
magebox test integration --tmpfs --tmpfs-size=2g

# Keep container for repeated test runs
magebox test integration --tmpfs --keep-alive

# Use specific MySQL version
magebox test integration --tmpfs --mysql-version=8.4
```

**Tmpfs Mode (RAM-based MySQL):**

When using `--tmpfs`, MageBox creates a dedicated MySQL container:
- Container name: `mysql-{version}-test` (e.g., `mysql-8-0-test`)
- All data stored in RAM - extremely fast I/O
- Automatically cleaned up after tests (unless `--keep-alive`)
- Perfect for CI/CD pipelines

**Container Management:**
```bash
# Check if test container is running
docker ps | grep mysql-.*-test

# Manually stop and remove test container
docker stop mysql-8-0-test && docker rm mysql-8-0-test
```

**How It Works:**

1. Uses PHPUnit with Magento's integration test bootstrap
2. Looks for config at `dev/tests/integration/phpunit.xml` or `.xml.dist`
3. Requires Magento's integration test framework to be set up
4. With `--tmpfs`: Creates ephemeral MySQL container with RAM storage

**Configuration in `.magebox.yaml`:**
```yaml
testing:
  integration:
    enabled: true
    config: "dev/tests/integration/phpunit.xml"
    db_host: "127.0.0.1"
    db_port: 33080
    db_name: "magento_integration_tests"
    db_user: "root"
    db_pass: "root"
    # Enable tmpfs by default for this project
    tmpfs: true
    tmpfs_size: "2g"
    keep_alive: false
```

**Setting Up Integration Tests:**

1. Copy the sample config:
   ```bash
   cp dev/tests/integration/phpunit.xml.dist dev/tests/integration/phpunit.xml
   ```

2. Create test database (or use `--tmpfs` to auto-create):
   ```bash
   magebox db shell
   CREATE DATABASE magento_integration_tests;
   ```

3. Configure `dev/tests/integration/etc/install-config-mysql.php`:
   ```php
   return [
       'db-host' => '127.0.0.1:33080',
       'db-user' => 'root',
       'db-password' => 'magebox',
       'db-name' => 'magento_integration_tests',
       // ...
   ];
   ```

---

### `magebox test phpstan`

Run PHPStan static analysis to find bugs without running code.

**Usage:**
```bash
magebox test phpstan [paths...] [options]
```

**Options:**

| Option | Short | Description |
|--------|-------|-------------|
| `--level` | `-l` | Analysis level 0-9 (default: 1) |

**Arguments:**

| Argument | Description |
|----------|-------------|
| `paths...` | Paths to analyze (default: `app/code`) |

**Examples:**
```bash
# Analyze with defaults (level 1, app/code)
magebox test phpstan

# Analyze at higher level
magebox test phpstan --level=5
magebox test phpstan -l 5

# Analyze specific path
magebox test phpstan app/code/MyVendor/MyModule

# Analyze multiple paths
magebox test phpstan app/code/MyVendor app/code/AnotherVendor

# Combine options
magebox test phpstan app/code/MyModule --level=3
```

**Analysis Levels:**

| Level | What It Checks |
|-------|----------------|
| 0 | Basic checks, unknown classes, unknown functions |
| 1 | Unknown variables, unknown methods on `$this` (default) |
| 2 | Unknown methods on all expressions |
| 3 | Return types, phpDoc types |
| 4 | Dead code, always true/false conditions |
| 5 | Argument types |
| 6 | Missing type hints |
| 7 | Union types |
| 8 | Nullable types, strict null checks |
| 9 | Mixed types |

**How It Works:**

1. MageBox locates PHPStan at `vendor/bin/phpstan`
2. Checks for config file:
   - Path specified in `.magebox.yaml`
   - `phpstan.neon`
   - `phpstan.neon.dist`
3. Runs with `--no-progress --error-format=table`

**Configuration in `.magebox.yaml`:**
```yaml
testing:
  phpstan:
    enabled: true
    level: 1                   # Default level
    config: "phpstan.neon"     # Custom config path
    paths:
      - "app/code"
      - "app/design"
```

**Example `phpstan.neon`:**
```neon
parameters:
    level: 1
    paths:
        - app/code

    # Magento-specific settings
    treatPhpDocTypesAsCertain: false

    # Exclude generated code
    excludePaths:
        - */generated/*
        - */vendor/*

    # Ignore specific errors
    ignoreErrors:
        - '#Variable \$[a-zA-Z]+ might not be defined#'
        - '#Call to an undefined method#'
```

---

### `magebox test phpcs`

Run PHP_CodeSniffer to check coding standards.

**Usage:**
```bash
magebox test phpcs [paths...] [options]
```

**Options:**

| Option | Short | Description |
|--------|-------|-------------|
| `--standard` | `-s` | Coding standard to use (default: Magento2) |

**Arguments:**

| Argument | Description |
|----------|-------------|
| `paths...` | Paths to check (default: `app/code`) |

**Examples:**
```bash
# Check with Magento2 standard (default)
magebox test phpcs

# Use different standard
magebox test phpcs --standard=PSR12
magebox test phpcs -s PSR12

# Check specific path
magebox test phpcs app/code/MyVendor/MyModule

# Check multiple paths
magebox test phpcs app/code app/design

# Combine options
magebox test phpcs app/code/MyModule --standard=Magento2
```

**Available Standards:**

| Standard | Description |
|----------|-------------|
| `Magento2` | Official Magento 2 coding standard (default) |
| `PSR12` | PHP-FIG PSR-12 Extended Coding Style |
| `PSR1` | PHP-FIG PSR-1 Basic Coding Standard |
| `PSR2` | PHP-FIG PSR-2 Coding Style (deprecated, use PSR12) |
| `Squiz` | Squiz coding standard |
| `Zend` | Zend Framework coding standard |

**How It Works:**

1. MageBox locates PHPCS at `vendor/bin/phpcs`
2. Checks for config file:
   - Path specified in `.magebox.yaml`
   - `phpcs.xml`
   - `phpcs.xml.dist`
   - `.phpcs.xml`
3. Runs with `--colors -p -s` (colors, progress, show sniff codes)

**Configuration in `.magebox.yaml`:**
```yaml
testing:
  phpcs:
    enabled: true
    standard: "Magento2"       # Default standard
    config: "phpcs.xml"        # Custom config path
    paths:
      - "app/code"
```

**Example `phpcs.xml`:**
```xml
<?xml version="1.0"?>
<ruleset name="Project Coding Standard">
    <description>Project coding standard based on Magento2</description>

    <!-- What to scan -->
    <file>app/code</file>

    <!-- Exclude patterns -->
    <exclude-pattern>vendor/*</exclude-pattern>
    <exclude-pattern>generated/*</exclude-pattern>
    <exclude-pattern>*.min.js</exclude-pattern>

    <!-- Use Magento2 standard -->
    <rule ref="Magento2"/>

    <!-- Customize line length -->
    <rule ref="Generic.Files.LineLength">
        <properties>
            <property name="lineLimit" value="120"/>
            <property name="absoluteLineLimit" value="0"/>
        </properties>
    </rule>

    <!-- Allow long array syntax in specific files -->
    <rule ref="Generic.Arrays.DisallowShortArraySyntax">
        <exclude-pattern>*.phtml</exclude-pattern>
    </rule>
</ruleset>
```

**Fixing Issues Automatically:**

Many PHPCS errors can be auto-fixed using PHPCBF:

```bash
# Fix all issues
vendor/bin/phpcbf app/code/MyModule

# Fix with specific standard
vendor/bin/phpcbf --standard=Magento2 app/code/MyModule
```

---

### `magebox test phpmd`

Run PHP Mess Detector to find potential problems.

**Usage:**
```bash
magebox test phpmd [paths...] [options]
```

**Options:**

| Option | Short | Description |
|--------|-------|-------------|
| `--ruleset` | `-r` | Comma-separated list of rulesets |

**Arguments:**

| Argument | Description |
|----------|-------------|
| `paths...` | Paths to analyze (default: `app/code`) |

**Examples:**
```bash
# Analyze with default rulesets
magebox test phpmd

# Use specific rulesets
magebox test phpmd --ruleset=cleancode,design
magebox test phpmd -r cleancode,codesize

# Analyze specific path
magebox test phpmd app/code/MyVendor/MyModule

# Combine options
magebox test phpmd app/code/MyModule --ruleset=cleancode
```

**Available Rulesets:**

| Ruleset | What It Detects |
|---------|-----------------|
| `cleancode` | Static methods, else branches, coupling |
| `codesize` | Cyclomatic complexity, excessive methods, too many parameters |
| `controversial` | Superglobals, CamelCase naming |
| `design` | Depth of inheritance, coupling, number of children |
| `naming` | Short/long variable names, constructor naming |
| `unusedcode` | Unused variables, parameters, methods |

**Default Ruleset:** `cleancode,codesize,design`

**How It Works:**

1. MageBox locates PHPMD at `vendor/bin/phpmd`
2. Checks for config file:
   - Path specified in `.magebox.yaml`
   - `phpmd.xml`
   - `phpmd.xml.dist`
3. Runs with `text` output format and `--exclude vendor,generated`

**Configuration in `.magebox.yaml`:**
```yaml
testing:
  phpmd:
    enabled: true
    ruleset: "cleancode,codesize,design"
    config: "phpmd.xml"        # Custom config path
    paths:
      - "app/code"
```

**Example `phpmd.xml`:**
```xml
<?xml version="1.0"?>
<ruleset name="Project PHPMD Rules"
         xmlns="http://pmd.sf.net/ruleset/1.0.0"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://pmd.sf.net/ruleset/1.0.0 http://pmd.sf.net/ruleset_xml_schema.xsd">

    <description>Project PHPMD rules</description>

    <!-- Exclude patterns -->
    <exclude-pattern>vendor/*</exclude-pattern>
    <exclude-pattern>generated/*</exclude-pattern>

    <!-- Include rulesets -->
    <rule ref="rulesets/cleancode.xml"/>
    <rule ref="rulesets/codesize.xml"/>
    <rule ref="rulesets/design.xml"/>

    <!-- Customize method length -->
    <rule ref="rulesets/codesize.xml/ExcessiveMethodLength">
        <properties>
            <property name="minimum" value="150"/>
        </properties>
    </rule>

    <!-- Customize cyclomatic complexity -->
    <rule ref="rulesets/codesize.xml/CyclomaticComplexity">
        <properties>
            <property name="reportLevel" value="15"/>
        </properties>
    </rule>

    <!-- Allow static access for factories -->
    <rule ref="rulesets/cleancode.xml/StaticAccess">
        <properties>
            <property name="exceptions" value="Magento\Framework\App\ObjectManager"/>
        </properties>
    </rule>
</ruleset>
```

---

### `magebox test all`

Run all tests and static analysis tools **except integration tests**.

**Usage:**
```bash
magebox test all
```

**What It Runs:**
1. PHPUnit unit tests
2. PHPStan analysis
3. PHP_CodeSniffer
4. PHP Mess Detector

**Exit Code:**
- `0` - All checks passed
- `1` - One or more checks failed

**Example Output:**
```
─── 1/4 - Running PHPUnit Unit Tests ───

PHPUnit 10.5.2 by Sebastian Bergmann and contributors.

...............                                             15 / 15 (100%)

Time: 00:02.345, Memory: 24.00 MB

OK (15 tests, 42 assertions)

✓ Unit tests passed

─── 2/4 - Running PHPStan ───

 [OK] No errors

✓ PHPStan analysis passed

─── 3/4 - Running PHP_CodeSniffer ───

...

✓ Code style checks passed

─── 4/4 - Running PHP Mess Detector ───

✓ No mess detected

─── Summary ───

✓ All checks passed!
```

---

### `magebox test status`

Show the installation and configuration status of all testing tools.

**Usage:**
```bash
magebox test status
```

**Example Output:**
```
─── Testing Tools Status ───

  PHPUnit:              Installed (10.5.2)
    Config: phpunit.xml
  Integration Tests:    Installed (10.5.2)
    No config file found
  PHPStan:              Installed (1.10.50)
    Config: phpstan.neon
  PHP_CodeSniffer:      Installed (3.8.0)
    Config: phpcs.xml
  PHP Mess Detector:    Not installed

Run magebox test setup to install missing tools
```

---

## Full Configuration Reference

Complete `.magebox.yaml` testing configuration:

```yaml
name: myproject
php: "8.2"

# ... other config ...

testing:
  # PHPUnit unit tests
  phpunit:
    enabled: true
    config: "phpunit.xml"           # Config file path
    testsuite: "Unit"               # Default test suite

  # Magento integration tests
  integration:
    enabled: true
    config: "dev/tests/integration/phpunit.xml"
    db_host: "127.0.0.1"            # Test database host
    db_port: 33080                  # Test database port
    db_name: "magento_integration_tests"
    db_user: "root"
    db_pass: "magebox"

  # PHPStan static analysis
  phpstan:
    enabled: true
    level: 1                        # Analysis level (0-9)
    config: "phpstan.neon"          # Config file path
    paths:                          # Paths to analyze
      - "app/code"
      - "app/design"

  # PHP_CodeSniffer
  phpcs:
    enabled: true
    standard: "Magento2"            # Coding standard
    config: "phpcs.xml"             # Config file path
    paths:                          # Paths to check
      - "app/code"

  # PHP Mess Detector
  phpmd:
    enabled: true
    ruleset: "cleancode,codesize,design"
    config: "phpmd.xml"             # Config file path
    paths:                          # Paths to analyze
      - "app/code"
```

---

## CI/CD Integration

### GitHub Actions

```yaml
name: Code Quality

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4

      - name: Setup PHP
        uses: shivammathur/setup-php@v2
        with:
          php-version: '8.2'
          tools: composer

      - name: Install dependencies
        run: composer install --prefer-dist --no-progress

      - name: PHPUnit
        run: vendor/bin/phpunit --testsuite=Unit

      - name: PHPStan
        run: vendor/bin/phpstan analyse --level=1 app/code

      - name: PHPCS
        run: vendor/bin/phpcs --standard=Magento2 app/code

      - name: PHPMD
        run: vendor/bin/phpmd app/code text cleancode,codesize,design
```

### GitLab CI

```yaml
stages:
  - test

code-quality:
  stage: test
  image: php:8.2
  before_script:
    - composer install --prefer-dist --no-progress
  script:
    - vendor/bin/phpunit --testsuite=Unit
    - vendor/bin/phpstan analyse --level=1 app/code
    - vendor/bin/phpcs --standard=Magento2 app/code
    - vendor/bin/phpmd app/code text cleancode,codesize,design
```

### Pre-commit Hook

Create `.git/hooks/pre-commit`:

```bash
#!/bin/bash

echo "Running code quality checks..."

# Run all tests except integration
magebox test all

if [ $? -ne 0 ]; then
    echo "Code quality checks failed. Commit aborted."
    exit 1
fi

echo "All checks passed!"
```

Make it executable:
```bash
chmod +x .git/hooks/pre-commit
```

---

## Troubleshooting

### Tool not found

```
PHPStan is not installed. Run: magebox test setup
```

**Solution:** Run `magebox test setup` to install the missing tools.

### Wrong PHP version

If tests fail because of PHP version mismatch:

1. Check your `.magebox.yaml` has correct PHP version
2. Run `magebox test status` to verify tools are installed
3. Ensure MageBox PHP wrapper is in your PATH:
   ```bash
   export PATH="$HOME/.magebox/bin:$PATH"
   ```

### Config file not found

If MageBox can't find your config file:

1. Check the file exists in your project root
2. Verify the path in `.magebox.yaml` is correct
3. Run `magebox test status` to see detected config paths

### PHPStan memory issues

For large projects, you may need to increase memory:

```bash
# In phpstan.neon
parameters:
    memory_limit: 1G
```

Or run manually:
```bash
php -d memory_limit=2G vendor/bin/phpstan analyse
```

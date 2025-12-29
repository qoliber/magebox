# Common Workflows

Real-world scenarios and how to accomplish them with MageBox. Follow these step-by-step guides to get things done quickly.

## Starting a New Project from Scratch

**Scenario**: You're starting a fresh Magento 2 project.

```bash
# 1. Create project directory
mkdir ~/projects/mystore && cd ~/projects/mystore

# 2. Install Magento via Composer
composer create-project --repository-url=https://repo.magento.com/ \
  magento/project-community-edition .

# 3. Initialize MageBox
magebox init

# 4. Configure your project (edit .magebox.yaml)
#    - Set PHP version
#    - Choose database version
#    - Add your domain

# 5. Start services
magebox start

# 6. Install Magento
php bin/magento setup:install \
  --base-url=https://mystore.test \
  --db-host=127.0.0.1:33080 \
  --db-name=mystore \
  --db-user=root \
  --db-password=magebox \
  --admin-firstname=Admin \
  --admin-lastname=User \
  --admin-email=admin@example.com \
  --admin-user=admin \
  --admin-password=admin123 \
  --language=en_US \
  --currency=USD \
  --timezone=America/New_York \
  --use-rewrites=1 \
  --search-engine=opensearch \
  --opensearch-host=127.0.0.1 \
  --opensearch-port=9200

# 7. Open in browser
open https://mystore.test
```

**Time**: ~10 minutes

---

## Joining an Existing Team Project

**Scenario**: You're a new developer joining a team that uses MageBox.

```bash
# 1. One-time setup (if not done)
magebox bootstrap

# 2. Add the team configuration
magebox team add mycompany
# Follow the prompts:
#   - Provider: github/gitlab/bitbucket
#   - Organization: your-org-name
#   - Asset storage: SFTP details for DB/media

# 3. Clone the project
magebox clone storefront
# This creates ~/projects/storefront with code ready

# 4. Navigate and fetch database
cd ~/projects/storefront
magebox fetch              # Download & import database

# 5. Start services
magebox start

# 6. Open in browser
open https://storefront.test
```

**Time**: ~5 minutes (depending on DB/media size)

---

## Debugging a Bug with Xdebug

**Scenario**: You need to step through code to find a bug.

```bash
# 1. Enable Xdebug
magebox xdebug on

# 2. Configure your IDE
#    PhpStorm: Settings → PHP → Debug → Port: 9003
#    VS Code: Install PHP Debug extension

# 3. Set breakpoints in your IDE

# 4. Start listening in IDE
#    PhpStorm: Click "Start Listening for PHP Debug Connections"
#    VS Code: Run "Listen for Xdebug" configuration

# 5. Trigger debugging in browser
#    Add ?XDEBUG_TRIGGER=1 to URL
#    Or use browser extension (Xdebug Helper)

# 6. When done, disable for performance
magebox xdebug off
```

**Quick tip**: Use `magebox dev` to enable Xdebug + disable OPcache in one command.

---

## Profiling Performance Issues

**Scenario**: Your store is slow and you need to find bottlenecks.

### Option A: Blackfire (Detailed Profiling)

```bash
# 1. Configure Blackfire (one-time)
magebox blackfire config
# Enter your server/client credentials

# 2. Enable Blackfire
magebox blackfire on

# 3. Profile a page
#    - Install Blackfire browser extension
#    - Click "Profile" on any page
#    - View results at blackfire.io

# 4. Profile CLI commands
blackfire run php bin/magento cache:flush

# 5. Disable when done
magebox blackfire off
```

### Option B: Tideways (Continuous Monitoring)

```bash
# 1. Configure Tideways (one-time)
magebox tideways config
# Enter your API key

# 2. Enable Tideways
magebox tideways on

# 3. Browse your site normally
#    Tideways automatically collects data

# 4. View insights at app.tideways.io

# 5. Disable when done
magebox tideways off
```

---

## Running Tests Before Committing

**Scenario**: You want to ensure code quality before pushing.

```bash
# 1. Setup testing tools (one-time)
magebox test setup
# Select: PHPUnit, PHPStan, PHPCS, PHPMD

# 2. Run all checks
magebox test all

# Or run specific tests:
magebox test unit                    # Unit tests
magebox test phpstan --level=6       # Static analysis
magebox test phpcs                   # Code style
magebox test phpmd                   # Mess detection

# 3. For integration tests (uses fast tmpfs database)
magebox test integration --tmpfs
```

**Pro tip**: Add to git pre-commit hook for automatic checks.

---

## Working with Multiple Store Views

**Scenario**: You have multiple domains for different store views.

```bash
# 1. Add additional domains
magebox domain add de.mystore.test --store-code=de_store
magebox domain add fr.mystore.test --store-code=fr_store

# 2. Or edit .magebox.yaml directly:
```

```yaml
domains:
  - name: mystore.test
    ssl: true
  - name: de.mystore.test
    ssl: true
    store_code: de_store
    store_type: store
  - name: fr.mystore.test
    ssl: true
    store_code: fr_store
    store_type: store
```

```bash
# 3. Restart to apply
magebox restart

# 4. Configure in Magento Admin
#    Stores → Configuration → Web → Base URLs
```

---

## Syncing Latest Database from Staging

**Scenario**: You need fresh data from a team environment.

```bash
# 1. Create a backup of current local DB (optional)
magebox db snapshot create before-sync

# 2. Sync database from team storage
magebox sync --db

# 3. If something goes wrong, restore
magebox db snapshot restore before-sync
```

---

## Switching PHP Versions

**Scenario**: You need to test with a different PHP version.

```bash
# 1. Check available versions
magebox status

# 2. Switch PHP for current project
magebox php 8.2

# 3. Verify
php -v

# 4. To make permanent, edit .magebox.yaml:
```

```yaml
php: "8.2"
```

---

## Setting Up Varnish Caching

**Scenario**: You want to test with Varnish enabled.

```bash
# 1. Enable Varnish in config
```

```yaml
# .magebox.yaml
services:
  varnish:
    enabled: true
```

```bash
# 2. Restart services
magebox restart

# 3. Configure Magento
php bin/magento config:set system/full_page_cache/caching_application 2

# 4. Test caching
curl -I https://mystore.test
# Look for: X-Magento-Cache-Control, X-Varnish headers

# 5. Purge cache when needed
magebox varnish purge
```

---

## End of Day Cleanup

**Scenario**: You're done for the day and want to free resources.

```bash
# Option A: Stop current project only
magebox stop

# Option B: Stop all projects
magebox stop --all

# Option C: Keep services but disable debugging
magebox xdebug off
magebox blackfire off
```

---

## Troubleshooting: Fresh Start

**Scenario**: Something's broken and you want to start fresh.

```bash
# 1. Stop everything
magebox stop

# 2. Clear generated files
rm -rf generated/* var/cache/* var/page_cache/* var/view_preprocessed/*

# 3. Clear Redis
magebox redis flush

# 4. Restart services
magebox start

# 5. Rebuild
php bin/magento setup:upgrade
php bin/magento setup:di:compile
php bin/magento cache:flush
```

---

## CI/CD Integration

**Scenario**: You want to run MageBox tests in CI.

### GitHub Actions

```yaml
# .github/workflows/test.yml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install MageBox
        run: curl -fsSL https://get.magebox.dev | bash

      - name: Bootstrap
        run: magebox bootstrap
        env:
          MAGEBOX_TEST_MODE: 1

      - name: Run Tests
        run: |
          magebox test phpstan --level=6
          magebox test phpcs
          magebox test unit
```

### GitLab CI

```yaml
# .gitlab-ci.yml
test:
  image: ubuntu:24.04
  script:
    - curl -fsSL https://get.magebox.dev | bash
    - magebox bootstrap
    - magebox test all
  variables:
    MAGEBOX_TEST_MODE: 1
```

---

::: tip Have a workflow to suggest?
Open an issue on [GitHub](https://github.com/qoliber/magebox) with your common workflow!
:::

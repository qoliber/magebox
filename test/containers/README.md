# MageBox Integration Test Containers

This directory contains Dockerfiles and scripts for testing MageBox on different Linux distributions without requiring Docker-in-Docker.

## Composer Authentication (for Magento)

To test Magento installation via Composer, you need to provide authentication credentials for `repo.magento.com`:

1. Copy the example file:
   ```bash
   cp test.auth.json.example test.auth.json
   ```

2. Edit `test.auth.json` with your Magento Marketplace credentials:
   ```json
   {
       "http-basic": {
           "repo.magento.com": {
               "username": "YOUR_PUBLIC_KEY",
               "password": "YOUR_PRIVATE_KEY"
           }
       }
   }
   ```

3. Get your keys from [Magento Marketplace](https://marketplace.magento.com/customer/accessKeys/)

The `test.auth.json` file is in `.gitignore` and will not be committed.

## Blackfire Credentials (for profiler testing)

To test Blackfire profiler configuration, you can provide Blackfire credentials:

1. Copy the example file:
   ```bash
   cp test.blackfire.env.example test.blackfire.env
   ```

2. Edit `test.blackfire.env` with your Blackfire credentials:
   ```bash
   BLACKFIRE_SERVER_ID=your-server-id
   BLACKFIRE_SERVER_TOKEN=your-server-token
   BLACKFIRE_CLIENT_ID=your-client-id      # optional
   BLACKFIRE_CLIENT_TOKEN=your-client-token # optional
   ```

3. Get your credentials from [Blackfire.io Settings](https://blackfire.io/my/settings/credentials)

The `test.blackfire.env` file is in `.gitignore` and will not be committed.

## Test Mode

MageBox supports a test mode that skips Docker and DNS operations, allowing you to test most functionality without actual Docker containers running inside the test container.

Set `MAGEBOX_TEST_MODE=1` to enable test mode.

## Available Distributions

| Distribution | Dockerfile | PHP Versions |
|--------------|------------|--------------|
| Fedora 42 | `Dockerfile.fedora42` | 8.1, 8.2, 8.3, 8.4 (Remi) |
| Ubuntu 24.04 | `Dockerfile.ubuntu` | 8.1, 8.2, 8.3, 8.4 (ondrej/php) |
| Ubuntu 22.04 | `Dockerfile.ubuntu22` | 8.1, 8.2, 8.3, 8.4 (ondrej/php) |
| Arch Linux | `Dockerfile.archlinux` | Latest only (pacman) |

## Running Tests

### Run all tests
```bash
./run-tests.sh
```

### Run specific distributions
```bash
./run-tests.sh fedora42 ubuntu
```

### Build only (no tests)
```bash
./run-tests.sh --build-only
```

### Run tests only (containers must exist)
```bash
./run-tests.sh --run-only
```

### Clean up containers and images
```bash
./run-tests.sh --clean
```

## What Gets Tested

1. **Version command** - `magebox --version`
2. **Help command** - `magebox --help`
3. **Init command** - `magebox init`
4. **Check command** - `magebox check`
5. **Status command** - `magebox status`
6. **List command** - `magebox list`
7. **PHP detection** - Available PHP versions
8. **Composer** - Version and auth configuration

## Manual Testing

Build a specific container:
```bash
docker build -t magebox-test:fedora42 -f Dockerfile.fedora42 ../..
```

Run interactive shell:
```bash
docker run -it --rm -e MAGEBOX_TEST_MODE=1 magebox-test:fedora42 bash
```

## Test Mode Behavior

When `MAGEBOX_TEST_MODE=1`:
- Docker container management is skipped
- DNS configuration is skipped
- Service status shows "(test mode)" for Docker services
- All other functionality works normally

This allows testing:
- Configuration file parsing
- PHP-FPM pool generation
- Nginx vhost generation
- SSL certificate generation
- CLI wrapper functionality
- Project discovery

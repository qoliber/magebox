# Troubleshooting

Common issues and solutions when using MageBox.

## Verbose Mode for Debugging

When troubleshooting issues, use the verbose flags to get more information:

```bash
# Basic verbose - shows commands being run
magebox -v start

# Detailed verbose - shows command output
magebox -vv start

# Debug verbose - shows full debug info including platform detection
magebox -vvv start
```

**Debug output includes:**
- Platform and Linux distro detection
- Docker Compose command detection (V2 vs V1)
- PHP version detection
- Service startup details
- SSL certificate generation

Example debug output:
```
[trace] MageBox version: 1.0.0
[trace] Verbosity level: 3
[trace] Detecting platform...
[trace] Platform type: linux, arch: amd64
[trace] os-release: ID=fedora, ID_LIKE=, PRETTY_NAME=Fedora Linux 42
[trace] Detecting Docker Compose command...
[trace] Docker Compose V2 detected: docker compose
```

## Services Not Starting

### Docker Not Running

```
Error: Cannot connect to the Docker daemon
```

**Solution:**

```bash
# macOS
open -a Docker

# Linux
sudo systemctl start docker
```

### Port Already in Use

```
Error: port 33080 is already in use
```

**Solution:**

```bash
# Find what's using the port
lsof -i :33080

# Stop the conflicting service or use a different database version
```

### Permission Denied

```
Error: permission denied while trying to connect to Docker
```

**Solution (Linux):**

```bash
# Add user to docker group
sudo usermod -aG docker $USER

# Log out and back in, or run:
newgrp docker
```

## PHP Issues

### Wrong PHP Version

```
PHP Fatal error: Composer detected issues in your platform
```

**Solution:**

Check your project's PHP version:

```bash
magebox status
```

Update `.magebox.yaml` if needed:

```yaml
php: "8.2"
```

### PHP-FPM Not Running

```
502 Bad Gateway
```

**Solution:**

```bash
# Check PHP-FPM status
# macOS
brew services list | grep php

# Linux
systemctl status php8.2-fpm

# Restart PHP-FPM
magebox stop && magebox start
```

### PHP Extensions Missing

```
PHP Fatal error: Class 'SomeClass' not found
```

**Solution:**

Install required extensions:

```bash
# macOS
brew install php@8.2-intl php@8.2-soap

# Ubuntu/Debian
sudo apt install php8.2-intl php8.2-soap php8.2-gd

# Fedora
sudo dnf install php-intl php-soap php-gd
```

## Database Issues

### Cannot Connect to Database

```
SQLSTATE[HY000] [2002] Connection refused
```

**Solution:**

1. Check services are running:
   ```bash
   magebox status
   ```

2. Verify port in `app/etc/env.php`:
   ```php
   'host' => '127.0.0.1:33080',  // Check port matches your MySQL version
   ```

3. Check Docker container:
   ```bash
   docker ps | grep mysql
   ```

### Database Import Failed

```
ERROR 1045 (28000): Access denied for user
```

**Solution:**

Use correct credentials:

```bash
# Default MageBox credentials
mysql -h 127.0.0.1 -P 33080 -u root -pmagebox
```

### Database Too Large to Import

**Solution:**

```bash
# Increase timeout
magebox db import large-dump.sql --timeout=3600

# Or import directly with mysql
mysql -h 127.0.0.1 -P 33080 -u root -pmagebox dbname < dump.sql
```

## Nginx Issues

### 404 Not Found

**Solution:**

1. Check document root in `.magebox.yaml`:
   ```yaml
   domains:
     - host: mystore.test
       root: pub  # Should be 'pub' for Magento
   ```

2. Regenerate nginx config:
   ```bash
   magebox stop && magebox start
   ```

### SSL Certificate Error

```
NET::ERR_CERT_AUTHORITY_INVALID
```

**Solution:**

```bash
# Trust the local CA
magebox ssl trust

# Regenerate certificates
magebox ssl generate
```

### 502 Bad Gateway

**Solution:**

1. Check PHP-FPM is running
2. Check socket exists:
   ```bash
   ls -la /tmp/magebox/*.sock
   ```
3. Restart services:
   ```bash
   magebox stop && magebox start
   ```

## DNS Issues

### Domain Not Resolving

```
Could not resolve host: mystore.test
```

**Solution for hosts mode:**

```bash
# Check /etc/hosts
cat /etc/hosts | grep mystore

# If missing, restart MageBox
magebox stop && magebox start
```

**Solution for dnsmasq mode:**

```bash
# Check dnsmasq is running
systemctl status dnsmasq

# Test resolution
dig mystore.test @127.0.0.1
```

### Browser Using Wrong DNS

**Solution:**

Disable DNS over HTTPS in your browser:

- **Chrome**: Settings → Privacy → Security → Use secure DNS → Off
- **Firefox**: Settings → Network Settings → Enable DNS over HTTPS → Off

## Redis Issues

### Cannot Connect to Redis

```
Redis connection refused
```

**Solution:**

```bash
# Check Redis is running
docker ps | grep redis

# Test connection
redis-cli -h 127.0.0.1 -p 6379 ping
```

### Redis Out of Memory

**Solution:**

```bash
# Flush Redis
magebox redis flush

# Or connect and flush
redis-cli -h 127.0.0.1 FLUSHALL
```

## Search Issues (OpenSearch/Elasticsearch)

### Search Not Working

```
No alive nodes found in your cluster
```

**Solution:**

1. Check service is running:
   ```bash
   curl http://127.0.0.1:9200
   ```

2. Check Magento config:
   ```bash
   php bin/magento config:show catalog/search/engine
   ```

3. Reindex:
   ```bash
   php bin/magento indexer:reindex catalogsearch_fulltext
   ```

### Cluster Health Red

**Solution:**

```bash
# Check cluster health
curl http://127.0.0.1:9200/_cluster/health?pretty

# Delete and recreate indices
curl -X DELETE http://127.0.0.1:9200/magento2_*
php bin/magento indexer:reindex catalogsearch_fulltext
```

## Performance Issues

### Slow Page Loads

**Solutions:**

1. Disable Xdebug:
   ```bash
   magebox xdebug off
   ```

2. Enable caching:
   ```bash
   php bin/magento cache:enable
   ```

3. Check Redis is being used for cache/sessions

4. Increase PHP memory in `.magebox.yaml`:
   ```yaml
   php_ini:
     memory_limit: 2G
   ```

### High CPU Usage

**Solution:**

Check for runaway processes:

```bash
# Check PHP processes
ps aux | grep php

# Check Docker containers
docker stats
```

## MageBox CLI Issues

### Command Not Found

```
magebox: command not found
```

**Solution:**

```bash
# Check installation
which magebox

# Add to PATH (add to ~/.bashrc or ~/.zshrc)
export PATH="$HOME/.magebox/bin:$PATH"
```

### Self-Update Failed

```
Error: failed to download update
```

**Solution:**

```bash
# Manual update
curl -sSL https://github.com/qoliber/magebox/releases/latest/download/magebox-linux-amd64 -o ~/.magebox/bin/magebox
chmod +x ~/.magebox/bin/magebox
```

## Linux-Specific Issues

### HTTPS Not Working (Port 443 Not Listening)

If nginx is not binding to port 443:

1. **Check SSL cert permissions** - nginx must be able to read `~/.magebox/certs/`

2. **Verify nginx user** - Must run as your user on Linux:
   ```bash
   # Check nginx user
   grep "^user" /etc/nginx/nginx.conf
   # Should show: user yourusername;

   # Fix if needed
   sudo sed -i "s/^user .*/user $USER;/" /etc/nginx/nginx.conf
   sudo systemctl restart nginx
   ```

3. **Check if 443 is listening:**
   ```bash
   ss -tlnp | grep 443
   ```

### DNS Resolution Not Working

If `.test` domains don't resolve:

1. **Check dnsmasq is running:**
   ```bash
   systemctl status dnsmasq
   ```

2. **Check systemd-resolved config:**
   ```bash
   cat /etc/systemd/resolved.conf.d/magebox.conf
   # Should contain:
   # [Resolve]
   # DNS=127.0.0.1
   # Domains=~test
   ```

3. **Test DNS resolution:**
   ```bash
   resolvectl query mystore.test
   # Should return 127.0.0.1
   ```

4. **Re-run bootstrap if needed:**
   ```bash
   magebox bootstrap
   ```

### SELinux Issues (Fedora/RHEL)

Fedora has SELinux enabled by default. MageBox bootstrap automatically configures SELinux, but if you encounter issues:

#### HTTPS not working (port 443 not listening)

Nginx can't read MageBox configs or SSL certs:

```bash
# Allow nginx to read MageBox configs and certs
sudo chcon -R -t httpd_config_t ~/.magebox/nginx/
sudo chcon -R -t httpd_config_t ~/.magebox/certs/

# Restart nginx
sudo systemctl restart nginx
```

#### 502 Bad Gateway errors

Nginx can't proxy to Docker containers:

```bash
# Allow nginx to make network connections
sudo setsebool -P httpd_can_network_connect on
```

#### Nginx can't read files from home directory

Permission denied errors when accessing project files:

```bash
# Allow nginx to read user content
sudo setsebool -P httpd_read_user_content on
```

#### PHP-FPM fails to start

PHP-FPM can't write to log directories:

```bash
# Set correct context on PHP-FPM log directories
sudo chcon -R -t httpd_log_t /var/opt/remi/php*/log/

# Restart PHP-FPM services
sudo systemctl restart php81-php-fpm php82-php-fpm php83-php-fpm php84-php-fpm
```

#### Debugging SELinux issues

```bash
# Check SELinux status
getenforce

# View recent SELinux denials
sudo ausearch -m avc -ts recent | grep -E "nginx|php-fpm"

# Temporarily disable SELinux (for testing only)
sudo setenforce 0
```

## Getting Help

If you're still stuck:

1. Run with verbose mode:
   ```bash
   magebox -vvv start
   ```

2. Check the logs:
   ```bash
   magebox logs
   ```

3. Check service status:
   ```bash
   magebox status
   docker ps
   ```

4. Open an issue on [GitHub](https://github.com/qoliber/magebox/issues) with verbose output

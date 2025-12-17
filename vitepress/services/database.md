# Database (MySQL/MariaDB)

MageBox runs MySQL and MariaDB in Docker containers, with each version on a unique port to allow multiple versions simultaneously.

## Supported Versions

### MySQL

| Version | Port | Magento Compatibility |
|---------|------|----------------------|
| MySQL 5.7 | 33057 | Magento 2.4.0 - 2.4.3 |
| MySQL 8.0 | 33080 | Magento 2.4.4+ (recommended) |
| MySQL 8.4 | 33084 | Magento 2.4.7+ |

### MariaDB

| Version | Port | Magento Compatibility |
|---------|------|----------------------|
| MariaDB 10.4 | 33104 | Magento 2.4.0 - 2.4.5 |
| MariaDB 10.6 | 33106 | Magento 2.4.4+ |
| MariaDB 11.4 | 33114 | Magento 2.4.7+ |

## Configuration

### Selecting Database Version

In `.magebox.yaml`:

```yaml
services:
  mysql: "8.0"    # Use MySQL 8.0
```

Or for MariaDB:

```yaml
services:
  mariadb: "10.6"  # Use MariaDB 10.6
```

::: warning
Use either `mysql` OR `mariadb`, not both.
:::

## Connection Details

### Default Credentials

| Setting | Value |
|---------|-------|
| Host | `127.0.0.1` |
| Port | Depends on version (see table above) |
| Username | `root` |
| Password | `magebox` |
| Database | Project name from `.magebox.yaml` |

### Magento Configuration

In `app/etc/env.php`:

```php
'db' => [
    'table_prefix' => '',
    'connection' => [
        'default' => [
            'host' => '127.0.0.1:33080',  // Port for MySQL 8.0
            'dbname' => 'mystore',
            'username' => 'root',
            'password' => 'magebox',
            'model' => 'mysql4',
            'engine' => 'innodb',
            'initStatements' => 'SET NAMES utf8;',
            'active' => '1',
        ]
    ]
],
```

### Magento Install Command

```bash
php bin/magento setup:install \
    --db-host=127.0.0.1:33080 \
    --db-name=mystore \
    --db-user=root \
    --db-password=magebox \
    # ... other options
```

## Database Commands

### Open Database Shell

```bash
magebox db shell
```

This connects to the correct database with proper credentials.

### Import Database

```bash
# From SQL file
magebox db import dump.sql

# From gzipped file
magebox db import dump.sql.gz
```

Import includes a real-time progress bar showing:
- Percentage complete
- Bytes imported / total size
- Transfer speed (MB/s)
- Estimated time remaining (ETA)

```
Importing dump.sql.gz into database 'mystore' (magebox-mysql-8.0)
  Importing: ████████████████████░░░░░░░░░░░░░░░░░░░░ 52.3% (156.2 MB/298.5 MB) 24.5 MB/s ETA: 6s
```

### Export Database

```bash
# To default file ({project}.sql)
magebox db export

# To specific file
magebox db export backup.sql

# Gzipped
magebox db export backup.sql.gz

# To stdout
magebox db export -
```

### Direct MySQL Access

```bash
# Connect directly with mysql client
mysql -h 127.0.0.1 -P 33080 -u root -pmagebox mystore
```

## Docker Container

### Viewing Container Status

```bash
# List running containers
docker ps | grep mysql

# Check specific container
docker ps -f name=magebox-mysql
```

### Container Logs

```bash
# View MySQL logs
docker logs magebox-mysql-8.0

# Follow logs
docker logs -f magebox-mysql-8.0
```

### Data Persistence

Database data is stored in Docker volumes:

```bash
# List volumes
docker volume ls | grep magebox

# Inspect volume
docker volume inspect magebox-mysql-8.0-data
```

## Multiple Database Versions

You can run multiple database versions simultaneously for different projects:

**Project A** (`.magebox.yaml`):
```yaml
name: project-a
services:
  mysql: "8.0"  # Port 33080
```

**Project B** (`.magebox.yaml`):
```yaml
name: project-b
services:
  mysql: "5.7"  # Port 33057
```

Both containers run simultaneously on different ports.

## Performance Tuning

### Docker Resource Limits

For large databases, you may need to adjust Docker resources:

1. Open Docker Desktop settings
2. Resources → Advanced
3. Increase Memory (4GB+ recommended for Magento)
4. Increase CPU

### MySQL Configuration

For production-like performance, you can mount a custom MySQL config:

```bash
# Create custom config
cat > ~/.magebox/docker/mysql-custom.cnf << EOF
[mysqld]
innodb_buffer_pool_size = 1G
innodb_log_file_size = 256M
max_connections = 200
EOF
```

## Troubleshooting

### Connection Refused

```
SQLSTATE[HY000] [2002] Connection refused
```

**Solutions:**

1. Check Docker is running:
   ```bash
   docker ps
   ```

2. Verify correct port:
   ```bash
   # Check which database version you're using
   cat .magebox.yaml | grep mysql
   ```

3. Start services:
   ```bash
   magebox global start
   ```

### Access Denied

```
ERROR 1045 (28000): Access denied for user 'root'@'localhost'
```

**Solution:** Use correct credentials:

```bash
mysql -h 127.0.0.1 -P 33080 -u root -pmagebox
```

Note: Password is `magebox`, not `magento`.

### Database Too Large to Import

For large databases:

```bash
# Increase timeout
mysql -h 127.0.0.1 -P 33080 -u root -pmagebox \
    --max_allowed_packet=1G \
    mystore < dump.sql

# Or split the import
split -l 100000 dump.sql dump_part_
for f in dump_part_*; do
    mysql -h 127.0.0.1 -P 33080 -u root -pmagebox mystore < $f
done
```

### Container Won't Start

```bash
# Check container logs
docker logs magebox-mysql-8.0

# Remove and recreate
docker rm -f magebox-mysql-8.0
magebox global start
```

### Port Already in Use

```bash
# Find what's using the port
lsof -i :33080

# Stop conflicting service or use different database version
```

## Backup Best Practices

### Regular Backups

```bash
# Create timestamped backup
magebox db export backup-$(date +%Y%m%d-%H%M%S).sql.gz
```

### Before Major Changes

```bash
# Backup before upgrade
magebox db export pre-upgrade.sql.gz

# Run upgrade
php bin/magento setup:upgrade

# If something goes wrong
magebox db import pre-upgrade.sql.gz
```

### Automated Backups

Create a cron job or script:

```bash
#!/bin/bash
# ~/.magebox/scripts/backup.sh
cd /path/to/project
magebox db export ~/backups/mystore-$(date +%Y%m%d).sql.gz

# Keep only last 7 days
find ~/backups -name "mystore-*.sql.gz" -mtime +7 -delete
```

## Switching Database Versions

To switch from MySQL 8.0 to MySQL 5.7:

1. Export your database:
   ```bash
   magebox db export backup.sql
   ```

2. Update `.magebox.yaml`:
   ```yaml
   services:
     mysql: "5.7"
   ```

3. Restart services:
   ```bash
   magebox restart
   ```

4. Import database:
   ```bash
   magebox db import backup.sql
   ```

5. Update `app/etc/env.php` with new port (33057).

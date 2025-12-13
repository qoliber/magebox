# Database (MySQL/MariaDB)

MageBox supports multiple MySQL and MariaDB versions running simultaneously.

## Configuration

### MySQL

```yaml
services:
  mysql: "8.0"  # Options: "5.7", "8.0", "8.4"
```

### MariaDB

```yaml
services:
  mariadb: "10.6"  # Options: "10.4", "10.6", "11.4"
```

::: warning
Configure either `mysql` OR `mariadb`, not both. Using both will result in an error.
:::

## Versions and Ports

| Version | Port | Recommended For |
|---------|------|-----------------|
| MySQL 5.7 | 33057 | Legacy Magento 2.3.x |
| MySQL 8.0 | 33080 | Magento 2.4.x (recommended) |
| MySQL 8.4 | 33084 | Testing latest features |
| MariaDB 10.4 | 33104 | Magento 2.4.x alternative |
| MariaDB 10.6 | 33106 | Magento 2.4.6+ |
| MariaDB 11.4 | 33114 | Latest features |

## Credentials

| Property | Value |
|----------|-------|
| Host | 127.0.0.1 |
| Port | See version table |
| Database | `<project-name>` |
| Username | root |
| Password | magebox |

## Database Commands

### Database Shell

Connect to MySQL/MariaDB CLI:

```bash
magebox db shell
```

This automatically uses the correct port and credentials for your project.

### Import Database

```bash
# Import from file
magebox db import dump.sql

# Import gzipped dump
magebox db import dump.sql.gz

# Import from stdin
cat dump.sql | magebox db import
```

### Export Database

```bash
# Export to file
magebox db export backup.sql

# Export with compression
magebox db export backup.sql.gz

# Export to stdout
magebox db export - > backup.sql
```

## Magento Configuration

Update `app/etc/env.php`:

```php
'db' => [
    'table_prefix' => '',
    'connection' => [
        'default' => [
            'host' => '127.0.0.1',
            'dbname' => 'mystore',      // Your project name
            'username' => 'root',
            'password' => 'magebox',
            'port' => '33080',           // Port for your MySQL version
            'model' => 'mysql4',
            'engine' => 'innodb',
            'initStatements' => 'SET NAMES utf8;',
            'active' => '1',
            'driver_options' => [
                1014 => false
            ]
        ]
    ]
],
'resource' => [
    'default_setup' => [
        'connection' => 'default'
    ]
]
```

## Multiple Databases

Different projects can use different database versions:

**Project A (.magebox.yaml)**
```yaml
name: projecta
services:
  mysql: "8.0"  # Port 33080
```

**Project B (.magebox.yaml)**
```yaml
name: projectb
services:
  mariadb: "10.6"  # Port 33106
```

Both run simultaneously without conflicts.

## Database Management Tools

### Using Sequel Pro / TablePlus / DBeaver

Connect with these settings:

| Field | Value |
|-------|-------|
| Host | 127.0.0.1 |
| Port | 33080 (or your version's port) |
| User | root |
| Password | magebox |
| Database | Your project name |

### Using MySQL Workbench

Create a new connection:
1. Connection Method: Standard TCP/IP
2. Hostname: 127.0.0.1
3. Port: 33080 (or your version's port)
4. Username: root
5. Password: magebox

## Troubleshooting

### Connection Refused

Check if the container is running:

```bash
docker ps | grep mysql
```

Start services if needed:

```bash
magebox global start
```

### Access Denied

Verify credentials match your configuration:

```bash
mysql -h 127.0.0.1 -P 33080 -u root -pmagebox
```

### Database Doesn't Exist

Create the database:

```bash
magebox db shell
mysql> CREATE DATABASE mystore;
```

### Performance Issues

For large imports, increase MySQL memory:

```bash
# Connect as root
mysql -h 127.0.0.1 -P 33080 -u root -pmagebox

# Increase buffer pool
SET GLOBAL innodb_buffer_pool_size = 1073741824;  # 1GB
```

## Data Persistence

Database data is stored in Docker volumes:

```bash
# View volume
docker volume inspect magebox_mysql80_data
```

To completely reset a database:

```bash
# Stop services
magebox global stop

# Remove volume
docker volume rm magebox_mysql80_data

# Restart services
magebox global start
```

::: danger
Removing the volume deletes all databases for that MySQL version.
:::

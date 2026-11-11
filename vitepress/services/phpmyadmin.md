# phpMyAdmin

MageBox supports [phpMyAdmin](https://www.phpmyadmin.net/), a web-based administration tool for MySQL and MariaDB databases.

## Overview

phpMyAdmin provides a visual interface for:

- **Browsing databases**, tables, and records
- **Running SQL queries** directly in the browser
- **Importing and exporting** data (SQL, CSV, etc.)
- **Managing users**, permissions, and privileges
- **Viewing and editing** table structure and data

## Enabling phpMyAdmin

### Via Command

```bash
magebox phpmyadmin enable
```

### Via Global Config

```bash
magebox config set phpmyadmin true
```

Then start services:

```bash
magebox global start
```

### Via Project Config

Add to your `.magebox.yaml`:

```yaml
services:
  mysql: "8.0"
  phpmyadmin: true
```

To use a custom port:

```yaml
services:
  mysql: "8.0"
  phpmyadmin:
    port: 9090
```

## Connection Details

| Setting | Value |
|---------|-------|
| Web UI | `http://localhost:8036` |
| Username | `root` |
| Password | `magebox` |

## Getting Started

1. Enable phpMyAdmin:
   ```bash
   magebox phpmyadmin enable
   ```

2. In your project directory, run:
   ```bash
   magebox phpmyadmin open
   ```

3. phpMyAdmin opens the project's database, logged in as root

## Connecting to Multiple Databases

One phpMyAdmin serves every project. Each MySQL and MariaDB container that a project uses is listed as its own server, labelled with its version (e.g. "MariaDB 12.3"), and you can switch between them with the server selector. The default database version is server 1.

`magebox phpmyadmin open` in a project directory selects the right server and database for that project.

**Arbitrary server mode** is also enabled, so you can connect to any other MySQL or MariaDB instance from the login screen. Use the container name as the server host:

| Database | Server Host |
|----------|-------------|
| MySQL 5.7 | `magebox-mysql-5.7` |
| MySQL 8.0 | `magebox-mysql-8.0` |
| MySQL 8.4 | `magebox-mysql-8.4` |
| MariaDB 10.4 | `magebox-mariadb-10.4` |
| MariaDB 10.6 | `magebox-mariadb-10.6` |
| MariaDB 11.4 | `magebox-mariadb-11.4` |

## MageBox Commands

### Open in Browser

```bash
magebox phpmyadmin open
```

Opens the phpMyAdmin web UI in your default browser. In a project directory it opens that project's database on the matching server.

### Check Status

```bash
magebox phpmyadmin status
```

Shows whether phpMyAdmin is enabled and running.

### Disable

```bash
magebox phpmyadmin disable
```

Stops the container and removes it from docker-compose.

## Docker Container

phpMyAdmin runs as a Docker container (`magebox-phpmyadmin`) with:

- **Image**: `phpmyadmin:latest`
- **Port**: 8036 (Web UI)
- **Network**: magebox

## Troubleshooting

### Cannot Connect to Database

Ensure your database service is running:

```bash
magebox global status
```

If not, start services:

```bash
magebox global start
```

### Port 8036 Already in Use

If another service is using port 8036, stop it before enabling phpMyAdmin:

```bash
lsof -i :8036
```

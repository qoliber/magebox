# Team Collaboration

MageBox includes powerful team collaboration features that allow you to share project configurations, repository access, and asset storage across your team.

## Overview

The team feature enables:
- **Centralized repository configuration** - Define Git providers (GitHub/GitLab/Bitbucket) once
- **Asset storage** - SFTP/FTP access for database dumps and media files
- **Project definitions** - Pre-configured projects with repo, branch, DB, and media paths
- **One-command setup** - Fetch entire projects with `magebox fetch`

## Configuration Storage

Team configurations are stored in `~/.magebox/teams.yaml`:

```yaml
teams:
  myteam:
    repositories:
      provider: github
      organization: myorg
      auth: ssh
    assets:
      provider: sftp
      host: backup.example.com
      port: 22
      path: /backups
      username: deploy
    projects:
      myproject:
        repo: myorg/myproject
        branch: main
        db: myproject/latest.sql.gz
        media: myproject/media.tar.gz
```

## Quick Start

### 1. Add a Team

```bash
magebox team add myteam
```

This interactive wizard will ask for:
- Repository provider (github/gitlab/bitbucket)
- Organization/namespace name
- Authentication method (ssh/token)
- Asset storage details (optional)

### 2. Add a Project

```bash
magebox team myteam project add myproject --repo myorg/myproject --db backups/db.sql.gz
```

### 3. Fetch the Project

```bash
magebox fetch myteam/myproject
```

This will:
1. Clone the repository
2. Download the database dump
3. Import to MySQL
4. Download and extract media files

## Commands Reference

### Team Management

```bash
# List all teams
magebox team list

# Add a new team
magebox team add <name>

# Remove a team
magebox team remove <name>

# Show team details
magebox team <name> show

# Browse repositories in team namespace
magebox team <name> repos
magebox team <name> repos --filter "magento*"
```

### Project Management

```bash
# List projects in a team
magebox team <name> project list

# Add a project
magebox team <name> project add <project-name> \
  --repo org/repo \
  --branch main \
  --db path/to/db.sql.gz \
  --media path/to/media.tar.gz

# Remove a project
magebox team <name> project remove <project-name>
```

### Fetch & Sync

```bash
# Fetch a project (clone + db + media)
magebox fetch myteam/myproject
magebox fetch myproject              # If only one team configured

# Fetch options
magebox fetch myproject --branch dev # Specific branch
magebox fetch myproject --no-db      # Skip database
magebox fetch myproject --no-media   # Skip media
magebox fetch myproject --db-only    # Only database
magebox fetch myproject --dry-run    # Show what would happen
magebox fetch myproject --to /path   # Custom destination

# Sync existing project (run from project directory)
magebox sync                         # Sync both DB and media
magebox sync --db                    # Only sync database
magebox sync --media                 # Only sync media
magebox sync --backup                # Backup current DB first
magebox sync --dry-run               # Show what would happen
```

## Multiple Providers for Same Organization

If you have the same organization name on multiple providers (e.g., `qoliber` on both GitHub and GitLab), create separate teams:

```yaml
teams:
  qoliber-github:
    repositories:
      provider: github
      organization: qoliber
      auth: ssh
  qoliber-gitlab:
    repositories:
      provider: gitlab
      organization: qoliber
      auth: token
```

Then fetch with the explicit team name:
```bash
magebox fetch qoliber-github/myproject
magebox fetch qoliber-gitlab/myproject
```

## Authentication

### Repository Authentication

**SSH (recommended)**
```yaml
repositories:
  provider: github
  organization: myorg
  auth: ssh
```
Uses your SSH keys configured in `~/.ssh/`

**Token**
```yaml
repositories:
  provider: github
  organization: myorg
  auth: token
```
Set via environment variable:
```bash
export MAGEBOX_MYTEAM_TOKEN="ghp_xxxxxxxxxxxx"
# or generic
export MAGEBOX_GIT_TOKEN="ghp_xxxxxxxxxxxx"
```

### Asset Storage Authentication

Credentials are read from environment variables:

```bash
# SSH key path (for SFTP)
export MAGEBOX_MYTEAM_ASSET_KEY="~/.ssh/backup_key"

# Or password (for FTP/SFTP)
export MAGEBOX_MYTEAM_ASSET_PASS="secretpassword"
```

The team name is uppercased in the variable name (e.g., `myteam` -> `MAGEBOX_MYTEAM_*`).

## Asset Storage Setup

### SFTP (Recommended)

```yaml
assets:
  provider: sftp
  host: backup.example.com
  port: 22
  path: /backups
  username: deploy
```

Directory structure on server:
```
/backups/
  project1/
    latest.sql.gz
    media.tar.gz
  project2/
    latest.sql.gz
    media.tar.gz
```

### FTP

```yaml
assets:
  provider: ftp
  host: ftp.example.com
  port: 21
  path: /backups
  username: ftpuser
```

## Example Workflow

### Initial Setup (Team Lead)

```bash
# 1. Create team configuration
magebox team add acme

# 2. Add projects
magebox team acme project add shop \
  --repo acme/magento-shop \
  --branch main \
  --db shop/db-latest.sql.gz \
  --media shop/media-latest.tar.gz

magebox team acme project add b2b \
  --repo acme/magento-b2b \
  --branch develop \
  --db b2b/db.sql.gz

# 3. Share teams.yaml with team
# (or use a shared config management system)
```

### Developer Onboarding

```bash
# 1. Copy teams.yaml to ~/.magebox/teams.yaml

# 2. Set credentials
export MAGEBOX_ACME_TOKEN="ghp_xxxxx"
export MAGEBOX_ACME_ASSET_KEY="~/.ssh/backup_key"

# 3. Fetch project
magebox fetch acme/shop

# 4. Initialize and start
cd shop
magebox init
magebox start
```

### Daily Workflow

```bash
# Get latest database from backup server
cd /path/to/shop
magebox sync --db --backup

# After import, clear cache
bin/magento cache:flush
```

## Tips

1. **Use SSH keys** for both Git and SFTP - more secure and no password prompts
2. **Compress database dumps** with gzip (`.sql.gz`) - faster downloads
3. **Create media snapshots periodically** - don't need to sync every day
4. **Use `--dry-run`** to preview what will happen before fetching
5. **Backup before sync** - use `magebox sync --backup` to save current DB

## Troubleshooting

### "Team not found"
```bash
magebox team list  # Check configured teams
```

### "Failed to connect to asset storage"
- Check host/port are correct
- Verify credentials: `MAGEBOX_TEAMNAME_ASSET_KEY` or `MAGEBOX_TEAMNAME_ASSET_PASS`
- Test manually: `sftp user@host`

### "Repository not found"
- Verify repo path in project config
- Check token permissions for private repos
- Test manually: `git clone git@github.com:org/repo.git`

### "Permission denied during media extraction"
- Check pub/media directory permissions
- The media archive should contain files relative to pub/media/

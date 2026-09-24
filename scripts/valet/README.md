# Valet → MageBox migration scripts

Helper scripts to move databases from **Homebrew MySQL** (Valet-era local MySQL) to **MageBox MySQL** (Docker), and optionally point parked Valet projects at MageBox.

Brew MySQL runs on port **3307** by default so MageBox can keep using **3306** / **33080**.

## Quick start

```bash
bash ./scripts/valet/setup.sh
```

This will:

1. Ask for **Homebrew MySQL** credentials (user/password, port 3307)
2. Ask for **MageBox MySQL** credentials (host/port/user/password, default port 33080)
3. Save them to `~/.config/magebox/valet-to-magebox.env` (mode `600`)
4. Install `valet-to-magebox` to `~/bin/`
5. Optionally patch all Magento / Laravel / WordPress projects in your Valet paths

Uninstall after migration:

```bash
bash ./scripts/valet/setup.sh --uninstall
```

Patch project configs later (uses saved credentials):

```bash
valet-to-magebox --update-projects
valet-to-magebox --update-projects --dry-run
valet-to-magebox --start-projects
valet-to-magebox --update-projects --start-projects
```

## Files

| File | Purpose |
|------|---------|
| `setup.sh` | Install to `~/bin`, save credentials, uninstall |
| `valet-to-magebox.sh` | Databases + `--update-projects` (source in repo) |
| `lib/common.sh` | Credentials and Valet path discovery |
| `lib/patch-projects.sh` | Update `env.php`, `.env`, `wp-config.php` |
| `lib/start-projects.sh` | Run `magebox start` for projects with `.magebox.yaml` |
## Requirements

- Homebrew MySQL 8.0 (`/opt/homebrew/opt/mysql@8.0/bin`)
- MageBox MySQL running (default: `127.0.0.1:33080`, user `root`, password `magebox`)
- Bash
- `php` (to patch Magento `env.php` and WordPress `wp-config.php`)
- `python3` (recommended for Laravel `.env` updates)

## Credentials

Saved by `setup.sh` and loaded automatically by `valet-to-magebox`:

| Variable | Default | Description |
|----------|---------|-------------|
| `BREW_MYSQL_PORT` | `3307` | Brew MySQL port |
| `BREW_MYSQL_USER` | `root` | Brew MySQL user |
| `BREW_MYSQL_PASSWORD` | *(empty)* | Brew MySQL password |
| `MAGEBOX_MYSQL_HOST` | `127.0.0.1` | MageBox MySQL host |
| `MAGEBOX_MYSQL_PORT` | `33080` | MageBox MySQL port |
| `MAGEBOX_MYSQL_USER` | `root` | MageBox MySQL user |
| `MAGEBOX_MYSQL_PASSWORD` | `magebox` | MageBox MySQL password |

Override the file path with `VALET_TO_MAGEBOX_CONFIG`.

## Database tool

From anywhere (after install):

```bash
valet-to-magebox --help
valet-to-magebox --list
valet-to-magebox --move='old_shop_*'
```

From the repo without installing:

```bash
./scripts/valet/valet-to-magebox.sh --list
```

### List databases

```bash
valet-to-magebox --list
valet-to-magebox --list='acme_shop_2025*'
```

### Delete databases

```bash
valet-to-magebox --delete=acme_shop_20250320
valet-to-magebox --delete='acme_shop_2025*' --force
```

### Move databases to MageBox

```bash
valet-to-magebox --move=acme_shop_20250320
valet-to-magebox --move='acme_shop_2025*'
valet-to-magebox --move=acme_shop_20250320 --keep-brew
```

## Project credential patching

`valet-to-magebox --update-projects` scans:

- `~/.config/valet/Sites` (linked sites)
- Paths from `~/.config/valet/config.json` (parked directories, one level deep)
- Output of `valet paths` when available

| Stack | File updated | Backup (before first patch) |
|-------|----------------|-----------------------------|
| Magento 2 | `app/etc/env.php` | `app/etc/env.php.valet` |
| Laravel | `.env` | `.env.valet` |
| WordPress | `wp-config.php` | `wp-config.php.valet` |

**Database:** host, user, password → MageBox (port 33080). Database names are unchanged.

**`.magebox.yaml`** (Magento + Laravel only, skipped if already present):
- Generated from Valet site name, `composer.json` PHP version, and `~/.magebox/config.yaml` defaults
- Domain `https://<site>.test`, `root: pub` (Magento) or `public` (Laravel), `ssl: true`, MySQL/Redis/Mailpit services
- Magento search: OpenSearch `2.19` with `memory: 2g` by default, or `elasticsearch: "8.11"` when `env.php` uses an Elasticsearch engine (or global defaults prefer Elasticsearch)

**URLs (https):**
- Magento — `unsecure` → `http://…/`, `secure` → `https://…/`, enables `use_in_frontend` / `use_in_adminhtml` in `env.php` + database
- Laravel — `APP_URL` (from existing value or `https://<valet-name>.test`)
- WordPress — `WP_HOME` and `WP_SITEURL`

Site hostname comes from `.magebox.yaml` (`domains[].host`), else the Valet Sites link name, else the project folder name + `.test`.

The original file is copied to `*.valet` once; re-running `--update-projects` keeps that backup and only updates the live file. Restore with `cp app/etc/env.php.valet app/etc/env.php` (etc.). Review each project after patching.

## Typical workflow

1. `magebox global start` (MageBox MySQL on 33080)
2. `bash ./scripts/valet/setup.sh`
3. `valet-to-magebox --list` then `--move=…` for each database
4. `valet-to-magebox --update-projects` (or `--dry-run` first)
5. `valet-to-magebox --start-projects` (or `--update-projects --start-projects`) — registers nginx vhosts + SSL per site
6. `bash ./scripts/valet/setup.sh --uninstall` when Brew MySQL is no longer needed

## Safety notes

- System databases (`mysql`, `sys`, …) are never listed, deleted, or moved.
- Delete and move actions ask for confirmation unless `--force` is used.
- If Brew MySQL is not running, the tool can start it temporarily on port `3307`.

## Troubleshooting

**`ERR_CERT_COMMON_NAME_INVALID` / wrong certificate in Chrome**  
This usually affects **all old Valet projects** after `--update-projects`, except sites you already ran `magebox start` on.

`--update-projects` patches DB/URLs and can create `.magebox.yaml`, but **does not** register the site in MageBox nginx. Without a vhost under `~/.magebox/nginx/vhosts/`, HTTPS falls back to the **first** SSL `server` block nginx loaded (wrong hostname in the certificate).

Fix for every migrated project:

```bash
valet-to-magebox --start-projects
# or per project:
cd /path/to/project && magebox start
```

Then reload nginx if needed:

```bash
magebox global stop && magebox global start
# or: nginx -s reload
```

Ensure mkcert trust is installed: `mkcert -install`

**`~/bin` not in PATH**  
Add `export PATH="${HOME}/bin:${PATH}"` to `~/.zshrc`.

**MageBox connection failed**  
Run `magebox global start`.

**No projects found for `--update-projects`**  
Ensure Valet paths exist or run `valet link` / `valet park` as before.

**`command not found: =--delete=...`**  
Use `--delete=name`, not `=--delete=name`.

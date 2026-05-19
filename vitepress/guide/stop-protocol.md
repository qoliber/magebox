# STOP (Static Precompilation & OPcache Protocol)

STOP enables OPcache and configures `app/preload.php` as the OPcache preload
script for the current project. It also raises `opcache.memory_consumption`
and turns on the tracing JIT — settings tuned for a real Magento store rather
than the conservative PHP defaults.

Where `magebox xdebug on` makes the dev loop *more* debuggable at the cost of
performance, `magebox stop-protocol enable` does the opposite: it makes the
project behave closer to production so you can profile, measure, or work
against realistic timings locally.

## Quick Start

```bash
# Enable STOP for the current project
magebox stop-protocol enable

# Disable STOP
magebox stop-protocol disable

# Show current STOP status
magebox stop-protocol status
```

`magebox stop-protocol` on its own is equivalent to `magebox stop-protocol status`.

## What it changes

Enabling STOP writes the following keys to `.magebox.local.yaml` under
`php_ini` and restarts PHP-FPM:

| Directive                     | Value                              |
|-------------------------------|------------------------------------|
| `opcache.enable`              | `1`                                |
| `opcache.preload`             | `<project>/app/preload.php`        |
| `opcache.preload_user`        | The current OS user                |
| `opcache.memory_consumption`  | `512` (MB)                         |
| `opcache.jit`                 | `tracing`                          |
| `opcache.jit_buffer_size`     | `100M`                             |

Disabling STOP sets `opcache.enable` back to `0` and removes the other
entries from `.magebox.local.yaml`. Your project-level `.magebox.yaml`
is never modified.

::: tip Why `.magebox.local.yaml`?
STOP is a per-developer toggle — you might want preload on while profiling
a slow page, but off again the rest of the time. Writing to
`.magebox.local.yaml` keeps the change personal and out of git.
:::

## Why a full restart, not a reload?

`opcache.preload` is only evaluated when the PHP-FPM master process starts.
A reload (SIGUSR2) recycles workers but keeps the master, which leaves the
old preload state in place and produces
`OPcache can't be temporary enabled` warnings on every worker spawn.

`magebox stop-protocol enable` and `disable` therefore perform a full
restart of PHP-FPM (`systemctl restart` on Linux, stop-then-start on macOS).
Expect a brief drop in requests during the restart.

## The preload script

STOP assumes the preload script lives at `app/preload.php` relative to your
project root. The path is fixed — MageBox does not let you point at a
different file, because the value is intended to match the convention used
by the Magento community's preload tooling.

If `app/preload.php` doesn't exist when you enable STOP, MageBox warns you
and continues anyway. OPcache will simply skip preloading until you create
the file, so it's safe to enable STOP first and add the preload script
later.

A minimal preload script looks like:

```php
<?php
// app/preload.php
opcache_compile_file(__DIR__ . '/../vendor/autoload.php');

foreach (glob(__DIR__ . '/code/**/*.php') as $file) {
    opcache_compile_file($file);
}
```

For a real Magento store you'll usually want a list generated from a warm
run — see the linked tools below.

## Status output

```bash
$ magebox stop-protocol status

STOP (Static Precompilation & OPcache Protocol) Status

OPcache enabled: true
Preload script:  /home/alice/projects/store/app/preload.php
Preload user:    alice

Disable with: magebox stop-protocol disable
```

If the preload file is missing on disk, `status` adds a warning so you
notice the misconfiguration before wondering why preload didn't help.

## Typical workflow

1. **Enable STOP before measurement work**

   ```bash
   magebox stop-protocol enable
   ```

   PHP-FPM restarts with OPcache and JIT on. The dev loop slows down a bit
   because OPcache now caches bytecode aggressively.

2. **Run your profiling / load tests**

   Use Blackfire, Tideways, k6, or whatever you normally reach for. The
   numbers will be much closer to what you'd see in production.

3. **Disable STOP when you go back to feature work**

   ```bash
   magebox stop-protocol disable
   ```

   OPcache is turned off and code changes are picked up on every request
   again.

## Interaction with `php_ini`

STOP writes raw PHP INI keys to `.magebox.local.yaml`, so it composes with
the [PHP INI settings](/guide/php-ini) system normally:

- Anything STOP sets can be overridden by adding the same key to
  `.magebox.yaml` or `.magebox.local.yaml` manually.
- Anything STOP doesn't touch (e.g. `opcache.validate_timestamps`) keeps
  the value from your existing configuration.

If you want a customized OPcache profile, run `magebox stop-protocol
enable` first, then edit `.magebox.local.yaml` to taste.

## Troubleshooting

### `OPcache can't be temporary enabled` warnings

This means PHP-FPM was reloaded rather than restarted. Re-run the toggle
command — `magebox stop-protocol enable` / `disable` perform a full
restart and should clear the warning. If you toggled `opcache.preload`
manually in YAML, run `magebox restart` to force a fresh master process.

### Preload runs but the site is broken

A preload script that errors out can take down the whole pool. Check
PHP-FPM logs:

```bash
magebox logs php
```

If preloading is the culprit, disable STOP, fix the script, and re-enable.

### Why is `opcache.preload_user` set?

PHP refuses to preload as root. STOP fills `opcache.preload_user` with
the current OS user so it matches the user PHP-FPM runs as in MageBox's
native setup. If you've customized the FPM pool user, override
`opcache.preload_user` in `.magebox.local.yaml` after enabling STOP.

## See also

- [PHP INI Settings](/guide/php-ini) — underlying mechanism STOP writes to
- [Local Overrides](/guide/local-overrides) — how `.magebox.local.yaml` works
- [Xdebug](/guide/xdebug) — the opposite end of the dev-vs-production spectrum

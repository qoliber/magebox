# Hyvä Theme

MageBox provides built-in support for installing and activating the [Hyvä theme](https://www.hyva.io/), a modern, performance-focused Magento 2 frontend built with Alpine.js and Tailwind CSS.

## Quick Start

The fastest way to get a Hyvä-powered store:

```bash
magebox new mystore --quick --hyva
```

This creates a new MageOS project with the Hyvä theme installed and activated, using sensible defaults.

## Installation

### New Project

Add the `--hyva` flag to `magebox new`:

```bash
# Quick install with Hyvä
magebox new mystore --quick --hyva

# Interactive wizard with Hyvä
magebox new mystore --hyva
```

MageBox will:
1. Prompt for your Hyvä Composer credentials (if not already configured)
2. Add the Hyvä Private Packagist repository to the project
3. Install the `hyva-themes/magento2-default-theme` package
4. Run `setup:upgrade` to register the theme
5. Activate the Hyvä theme automatically

### Hyvä Composer Credentials

To install Hyvä, you need a valid Hyvä license. MageBox will prompt you for:

1. **Hyvä Composer Repository URL** - Found in your [Hyvä account](https://www.hyva.io/customer/account/login/)
2. **Hyvä Composer Token** - Your Private Packagist authentication token

These credentials are stored globally in your Composer auth config (`~/.config/composer/auth.json` or `~/.composer/auth.json`), so you only need to enter them once.

::: tip
If you've already configured Hyvä credentials for another project (e.g., via `composer config`), MageBox will detect and reuse them automatically.
:::

## How It Works

### Authentication

MageBox uses Composer's `http-basic` authentication with Hyvä's Private Packagist at `hyva-themes.repo.packagist.com`. The token is stored as:

```
http-basic.hyva-themes.repo.packagist.com: token <your-token>
```

### Theme Activation

After `setup:upgrade` registers the Hyvä theme in the database, MageBox activates it by setting:

```bash
php bin/magento config:set design/theme/theme_id 5
```

This sets the Hyvä Default Theme as the active storefront theme.

::: warning
The theme ID `5` assumes a fresh Magento installation where Hyvä is the first additional theme installed. If you have other themes installed, the ID may differ. In that case, check the correct ID with:
```bash
php bin/magento dev:query "SELECT theme_id, theme_path FROM theme WHERE theme_path LIKE '%hyva%'"
```
:::

## Troubleshooting

### Authentication Failed

If you see authentication errors:

1. Verify your Hyvä license is active at [hyva.io](https://www.hyva.io/)
2. Check your stored credentials:
   ```bash
   composer config --global --list | grep hyva
   ```
3. Clear and re-enter credentials:
   ```bash
   composer config --global --unset http-basic.hyva-themes.repo.packagist.com
   ```
   Then run `magebox new --hyva` again.

### Theme Not Visible

If the Hyvä theme doesn't appear on the storefront after installation:

1. Clear all caches:
   ```bash
   php bin/magento cache:flush
   ```
2. Verify the theme is registered:
   ```bash
   php bin/magento dev:query "SELECT * FROM theme"
   ```
3. Check the active theme ID:
   ```bash
   php bin/magento config:show design/theme/theme_id
   ```

## Resources

- [Hyvä Documentation](https://docs.hyva.io/)
- [Hyvä GitHub](https://github.com/hyva-themes)
- [Hyvä Compatibility Module Tracker](https://gitlab.hyva.io/hyva-themes/hyva-compat)

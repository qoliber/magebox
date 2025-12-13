# Admin Commands

MageBox provides commands to manage Magento admin users directly from the CLI.

## Commands Overview

| Command | Description |
|---------|-------------|
| `magebox admin list` | List all admin users |
| `magebox admin create` | Create a new admin user |
| `magebox admin password` | Reset admin password |
| `magebox admin disable-2fa` | Disable Two-Factor Authentication |

## List Admin Users

```bash
magebox admin list
```

Output:

```
Admin Users

USERNAME        EMAIL                          FIRST NAME      LAST NAME       ACTIVE
------------------------------------------------------------------------------------------
admin           admin@example.com              John            Doe             Yes
developer       dev@example.com                Jane            Smith           Yes
```

## Create Admin User

```bash
magebox admin create
```

Interactive prompts:

```
Create Admin User

Username: newadmin
Email: newadmin@example.com
Password: ********
First Name: New
Last Name: Admin

Creating admin user... done

Admin user created!
Username: newadmin
Email: newadmin@example.com
```

## Reset Admin Password

Reset password for an existing admin user:

```bash
# With password as argument
magebox admin password admin@example.com newpassword123

# Interactive (prompts for password)
magebox admin password admin@example.com
```

This command:
1. Unlocks the user (if locked due to failed attempts)
2. Resets the password directly in the database
3. Uses Magento's password hashing algorithm

## Disable Two-Factor Authentication

For local development, 2FA can be inconvenient:

```bash
magebox admin disable-2fa
```

This will:
1. Disable `Magento_TwoFactorAuth` module
2. Disable `Magento_AdminAdobeImsTwoFactorAuth` module
3. Set 2FA config to disabled
4. Clear cache

::: warning
Never disable 2FA on production environments. This command is intended for local development only.
:::

## Common Workflows

### Fresh Installation Setup

After installing Magento, create your admin user:

```bash
# Create admin during setup
php bin/magento setup:install \
  --admin-user=admin \
  --admin-password=admin123 \
  --admin-email=admin@example.com \
  --admin-firstname=Admin \
  --admin-lastname=User \
  ...

# Or create separately
magebox admin create
```

### Forgot Admin Password

```bash
# Reset password
magebox admin password admin@example.com mynewpassword

# If you don't know the email
magebox admin list
```

### Locked Out of Admin

If you're locked out due to too many failed attempts or 2FA issues:

```bash
# Reset password (also unlocks the account)
magebox admin password admin@example.com newpassword

# Disable 2FA if that's the issue
magebox admin disable-2fa
```

### Development Team Setup

Create separate admin accounts for team members:

```bash
magebox admin create
# Enter: developer1, dev1@company.com, password, First, Last

magebox admin create
# Enter: developer2, dev2@company.com, password, First, Last
```

## Database Direct Access

For advanced operations, access the database directly:

```bash
# Open database shell
magebox db shell

# Query admin users
SELECT user_id, username, email, is_active FROM admin_user;

# Unlock a user
UPDATE admin_user SET failures_num = 0, lock_expires = NULL WHERE email = 'admin@example.com';
```

## Troubleshooting

### "No project config found"

Run from your Magento project directory:

```bash
cd /path/to/magento
magebox admin list
```

### Database Connection Failed

Ensure services are running:

```bash
magebox start
magebox status
```

### Password Reset Not Working

Try the Magento CLI directly:

```bash
php bin/magento admin:user:unlock admin@example.com
php bin/magento admin:user:create --admin-user=newadmin ...
```

### 2FA Still Enabled After Disable

Clear all caches and regenerate:

```bash
php bin/magento cache:flush
php bin/magento setup:upgrade
php bin/magento setup:di:compile
```

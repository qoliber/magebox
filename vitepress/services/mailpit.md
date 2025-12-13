# Mailpit

MageBox runs Mailpit in Docker for email testing during development.

## Overview

Mailpit is a modern email testing tool that:

- **Captures all outgoing emails** - No emails are actually sent
- **Provides a web interface** - View and inspect emails
- **Supports SMTP** - Works with any application
- **Zero configuration** - Just works out of the box

## Why Use Mailpit?

During development, you want to:

- Test email templates without sending real emails
- Verify transactional emails (order confirmations, password resets)
- Debug email content and formatting
- Avoid spamming real email addresses

## Configuration

### Enabling Mailpit

In `.magebox.yaml`:

```yaml
services:
  mailpit: true
```

## Connection Details

| Service | Host | Port |
|---------|------|------|
| SMTP | `127.0.0.1` | `1025` |
| Web Interface | `127.0.0.1` | `8025` |

## Web Interface

Access the Mailpit web interface at:

```
http://localhost:8025
```

### Features

- **Inbox view** - See all captured emails
- **Email preview** - View HTML and plain text versions
- **Attachments** - Download email attachments
- **Source view** - Inspect raw email headers
- **Search** - Find specific emails
- **Delete** - Clear inbox

## Magento Configuration

### Automatic Configuration

When you run `magebox new` or generate env.php, MageBox automatically configures Mailpit SMTP:

```php
// Automatically added to app/etc/env.php
'system' => [
    'default' => [
        'smtp' => [
            'disable' => '0',
            'host' => '127.0.0.1',
            'port' => '1025'
        ]
    ]
]
```

This prevents accidental emails to real addresses during development - all emails are captured by Mailpit.

### Manual Configuration

If you need to configure manually in `app/etc/env.php`:

```php
// app/etc/env.php
'system' => [
    'default' => [
        'smtp' => [
            'disable' => '0',
            'host' => '127.0.0.1',
            'port' => '1025'
        ],
        'trans_email' => [
            'ident_general' => [
                'email' => 'general@mystore.test'
            ],
            'ident_sales' => [
                'email' => 'sales@mystore.test'
            ],
            'ident_support' => [
                'email' => 'support@mystore.test'
            ]
        ]
    ]
]
```

### Via Admin Panel

1. Go to **Stores → Configuration → Advanced → System → Mail Sending Settings**
2. Set **Disable Email Communications** to **No**
3. Save configuration

### SMTP Configuration

For Magento modules that support custom SMTP (like Mageplaza SMTP):

| Setting | Value |
|---------|-------|
| SMTP Server | `127.0.0.1` |
| SMTP Port | `1025` |
| Protocol | None (no encryption) |
| Authentication | None |

## Testing Emails

### Sending Test Email

From Magento Admin:

1. Go to **Marketing → Communications → Email Templates**
2. Click **Add New Template**
3. Load a default template
4. Click **Preview Template**

### Common Test Scenarios

#### Order Confirmation

1. Place a test order
2. Check Mailpit for order confirmation
3. Verify template content

#### Password Reset

1. Go to customer login
2. Click "Forgot Password"
3. Enter email address
4. Check Mailpit for reset email

#### Contact Form

1. Fill out contact form
2. Submit
3. Check Mailpit for contact notification

### CLI Email Testing

```bash
# Send test email via PHP
php -r "mail('test@example.com', 'Test Subject', 'Test body');"

# Check Mailpit
open http://localhost:8025
```

## Docker Container

### Container Status

```bash
docker ps | grep mailpit
```

### Container Logs

```bash
docker logs magebox-mailpit

# Follow logs
docker logs -f magebox-mailpit
```

### Restart Container

```bash
docker restart magebox-mailpit
```

## API Access

Mailpit provides a REST API for automation:

### List Messages

```bash
curl http://localhost:8025/api/v1/messages | jq
```

### Get Message

```bash
curl http://localhost:8025/api/v1/message/{id} | jq
```

### Delete All Messages

```bash
curl -X DELETE http://localhost:8025/api/v1/messages
```

### Search Messages

```bash
curl "http://localhost:8025/api/v1/search?query=order" | jq
```

## Troubleshooting

### Emails Not Appearing

1. Check Mailpit is running:
   ```bash
   docker ps | grep mailpit
   ```

2. Verify SMTP settings in Magento:
   ```bash
   php bin/magento config:show system/smtp/disable
   # Should return 0
   ```

3. Check PHP mail settings:
   ```bash
   php -i | grep sendmail
   ```

### Connection Refused

```
Connection refused on port 1025
```

**Solutions:**

1. Start services:
   ```bash
   magebox global start
   ```

2. Check port is available:
   ```bash
   lsof -i :1025
   ```

### Emails Going to Real Recipients

Mailpit captures ALL emails sent through its SMTP server. If emails are being delivered:

1. Check Magento isn't using an external SMTP service
2. Verify the SMTP configuration points to `127.0.0.1:1025`
3. Disable any SMTP extensions that override settings

### HTML Not Rendering

In Mailpit web interface, switch between:

- **HTML** - Rendered view
- **Plain** - Text version
- **Source** - Raw email

## Best Practices

### Development Workflow

1. Keep Mailpit running during development
2. Check emails after every email-triggering action
3. Verify both HTML and plain text versions
4. Test on different "devices" using Mailpit's preview options

### Email Template Development

1. Make template changes
2. Clear Magento cache:
   ```bash
   php bin/magento cache:flush
   ```
3. Trigger email
4. Check Mailpit
5. Repeat

### Clearing Inbox

Periodically clear the inbox to keep it manageable:

```bash
# Via API
curl -X DELETE http://localhost:8025/api/v1/messages

# Or use web interface
```

## Integration with Magento Email Testing

### Testing All Email Templates

Create a script to test all email templates:

```bash
#!/bin/bash
# Test various email scenarios

# Create customer
php bin/magento customer:create test@example.com password Test User

# Request password reset
# (Do via frontend)

# Place order
# (Do via frontend)

# Check Mailpit
open http://localhost:8025
```

### Automated Testing

For integration tests, use Mailpit API:

```php
// PHPUnit test example
$response = file_get_contents('http://localhost:8025/api/v1/messages');
$messages = json_decode($response, true);
$this->assertGreaterThan(0, count($messages['messages']));
```

## Comparison with Other Tools

| Feature | Mailpit | Mailhog | Mailtrap |
|---------|---------|---------|----------|
| Local | Yes | Yes | No (SaaS) |
| Free | Yes | Yes | Limited |
| Modern UI | Yes | Basic | Yes |
| API | Yes | Yes | Yes |
| Docker | Yes | Yes | N/A |

Mailpit is the modern, actively maintained successor to Mailhog.

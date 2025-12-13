# Mailpit

Mailpit is a local email testing tool that catches all outgoing emails from your Magento store.

## Configuration

Enable Mailpit in your `.magebox.yaml` file:

```yaml
services:
  mailpit: true
```

## Connection Details

| Property | Value |
|----------|-------|
| SMTP Host | 127.0.0.1 |
| SMTP Port | 1025 |
| Web UI | http://localhost:8025 |

## Web Interface

Access the Mailpit UI to view captured emails:

```
http://localhost:8025
```

Features:
- View all captured emails
- Search by subject, sender, recipient
- View HTML and plain text versions
- Download attachments
- View email source/headers

## Magento Configuration

### Via Admin Panel

1. Go to **Stores > Configuration > Advanced > System > Mail Sending Settings**
2. Set:
   - Transport: SMTP
   - Host: 127.0.0.1
   - Port: 1025

### Via env.php

```php
'system' => [
    'default' => [
        'smtp' => [
            'disable' => '0',
            'transport' => 'smtp',
            'host' => '127.0.0.1',
            'port' => '1025'
        ]
    ]
]
```

### Using a Module

For more control, use a SMTP module like `mageplaza/module-smtp`:

```bash
composer require mageplaza/module-smtp
magebox cli setup:upgrade
```

Configure in Admin:
- SMTP Server: 127.0.0.1
- Port: 1025
- Protocol: None
- Authentication: None

## Testing Emails

### Send Test Email

```bash
magebox cli dev:email:send test@example.com "Test Subject" "Test Body"
```

Or trigger emails through Magento:
1. Create a test order
2. Request password reset
3. Create customer account

### Check Captured Emails

Open http://localhost:8025 to see all emails.

## Features

### Search

Filter emails by:
- Sender
- Recipient
- Subject
- Content

### HTML Preview

View emails exactly as customers would see them, including:
- Responsive layouts
- Images
- Styling

### Source View

Inspect raw email headers and MIME structure for debugging.

### API Access

Mailpit provides a REST API:

```bash
# List messages
curl http://localhost:8025/api/v1/messages

# Get specific message
curl http://localhost:8025/api/v1/message/{id}

# Delete all messages
curl -X DELETE http://localhost:8025/api/v1/messages
```

## Troubleshooting

### Emails Not Appearing

1. Check Mailpit is running:

```bash
docker ps | grep mailpit
```

2. Verify SMTP configuration in Magento

3. Check Magento logs for mail errors:

```bash
magebox logs system.log | grep -i mail
```

### Connection Refused

If Magento can't connect to Mailpit:

```bash
# Test SMTP connection
telnet 127.0.0.1 1025
```

Start services if needed:

```bash
magebox global start
```

### Emails Going to Real Recipients

Ensure you're using Mailpit's SMTP server, not a real one. Double-check:
- Host is `127.0.0.1`
- Port is `1025`

## Development Workflow

1. **Always enable Mailpit** in development to prevent sending real emails

2. **Test email templates** by triggering various Magento emails

3. **Check responsive design** using Mailpit's HTML preview

4. **Verify transactional data** appears correctly in emails

5. **Clear inbox regularly** using the "Delete all" feature

## When to Use Mailpit

Enable Mailpit when:
- Developing locally (always recommended)
- Testing email templates
- Debugging email delivery issues
- Reviewing transactional emails

Disable only when:
- Testing with a real email service
- Minimal resource usage is needed

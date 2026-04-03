# RabbitMQ

MageBox runs RabbitMQ in Docker for asynchronous message processing in Magento.

## Overview

RabbitMQ is a message broker that enables:

- **Asynchronous operations** - Defer time-consuming tasks
- **Message queues** - Process bulk operations in background
- **Improved performance** - Don't block user requests

## Magento Use Cases

Magento uses RabbitMQ for:

- Bulk API operations
- Product import/export
- Stock updates
- Inventory synchronization
- B2B features (quotes, orders)
- Customer notifications

## Configuration

### Enabling RabbitMQ

In `.magebox.yaml`:

```yaml
services:
  rabbitmq: true
```

## Connection Details

| Setting | Value |
|---------|-------|
| Host | `127.0.0.1` |
| AMQP Port | `5672` |
| Management Port | `15672` |
| Username | `guest` |
| Password | `guest` |
| Virtual Host | `/` |

## Magento Configuration

### Via Install Command

```bash
php bin/magento setup:install \
    --amqp-host=127.0.0.1 \
    --amqp-port=5672 \
    --amqp-user=guest \
    --amqp-password=guest \
    # ... other options
```

### Via env.php

```php
// app/etc/env.php
'queue' => [
    'amqp' => [
        'host' => '127.0.0.1',
        'port' => '5672',
        'user' => 'guest',
        'password' => 'guest',
        'virtualhost' => '/'
    ]
],
```

### Enable Async Operations

To use RabbitMQ for specific operations, update `env.php`:

```php
'queue' => [
    'consumers_wait_for_messages' => 0,
    'amqp' => [
        'host' => '127.0.0.1',
        'port' => '5672',
        'user' => 'guest',
        'password' => 'guest',
        'virtualhost' => '/'
    ]
],
```

## Management Interface

### Web UI

Access the RabbitMQ management interface at:

```
http://localhost:15672
```

**Credentials:**
- Username: `guest`
- Password: `guest`

### Features

The management UI allows you to:

- View queues and messages
- Monitor connections
- Manage exchanges
- View message rates
- Purge queues

## Common Operations

### List Queues

```bash
# Via management API
curl -u guest:guest http://localhost:15672/api/queues | jq
```

### Purge a Queue

```bash
# Via management API
curl -X DELETE -u guest:guest \
    http://localhost:15672/api/queues/%2F/queue_name/contents
```

### Check Connection

```bash
curl -u guest:guest http://localhost:15672/api/overview | jq '.message_stats'
```

## Message Consumers

### Start Consumers

Magento requires running consumers to process messages:

```bash
# Start all consumers
php bin/magento queue:consumers:start

# Start specific consumer
php bin/magento queue:consumers:start async.operations.all

# Run consumer for specific number of messages
php bin/magento queue:consumers:start async.operations.all --max-messages=100
```

### List Consumers

```bash
php bin/magento queue:consumers:list
```

### Common Consumers

| Consumer | Purpose |
|----------|---------|
| `async.operations.all` | Process all async operations |
| `product_action_attribute.update` | Bulk product attribute updates |
| `product_action_attribute.website.update` | Website assignment updates |
| `exportProcessor` | Export operations |
| `inventory.reservations.cleanup` | Inventory cleanup |
| `inventory.reservations.update` | Inventory updates |

### Background Consumer Processing

For development, run consumers in background:

```bash
# Run in background
nohup php bin/magento queue:consumers:start async.operations.all &

# Or use supervisor for production-like setup
```

## Docker Container

### Container Status

```bash
docker ps | grep rabbitmq
```

### Container Logs

```bash
docker logs magebox-rabbitmq

# Follow logs
docker logs -f magebox-rabbitmq
```

### Restart Container

```bash
docker restart magebox-rabbitmq
```

## Troubleshooting

### Connection Refused

```
Failed to connect to RabbitMQ
```

**Solutions:**

1. Check container is running:
   ```bash
   docker ps | grep rabbitmq
   ```

2. Verify port is accessible:
   ```bash
   curl -u guest:guest http://localhost:15672/api/overview
   ```

3. Start services:
   ```bash
   magebox global start
   ```

### Messages Not Processing

1. Check if consumers are running:
   ```bash
   ps aux | grep queue:consumers
   ```

2. Start consumers:
   ```bash
   php bin/magento queue:consumers:start async.operations.all
   ```

3. Check queue status in management UI:
   ```
   http://localhost:15672/#/queues
   ```

### Queue Full / Backlog

If messages are accumulating:

```bash
# Check queue size
curl -u guest:guest http://localhost:15672/api/queues | jq '.[].messages'

# Run consumer with batch processing
php bin/magento queue:consumers:start async.operations.all --max-messages=1000
```

### Consumer Errors

Check Magento logs:

```bash
tail -f var/log/system.log | grep -i queue
tail -f var/log/exception.log
```

### Authentication Failed

Verify credentials in `env.php`:

```php
'queue' => [
    'amqp' => [
        'user' => 'guest',
        'password' => 'guest',
    ]
],
```

## Development vs Production

### Development

- Consumers can be run manually when needed
- Management UI is useful for debugging
- Messages can be purged freely

### Production-Like Testing

For testing async operations:

1. Start consumers in background:
   ```bash
   php bin/magento queue:consumers:start async.operations.all &
   ```

2. Monitor via management UI
3. Test bulk operations (import, mass actions)

## Disabling RabbitMQ

If you don't need async operations:

1. Remove from `.magebox.yaml`:
   ```yaml
   services:
     rabbitmq: false  # or remove the line
   ```

2. Update `env.php` to use MySQL for queues:
   ```php
   'queue' => [
       'consumers_wait_for_messages' => 0,
       // Remove 'amqp' section
   ],
   ```

3. Magento will fall back to database queues (slower but functional).

## Best Practices

### Consumer Management

- Run consumers with `--max-messages` to prevent memory leaks
- Restart consumers periodically
- Monitor queue sizes during bulk operations

### Queue Monitoring

```bash
# Simple monitoring script
watch -n 5 'curl -s -u guest:guest http://localhost:15672/api/queues | jq ".[].messages"'
```

### Testing Async Operations

1. Ensure consumers are running
2. Perform bulk operation (e.g., mass product update)
3. Check queue in management UI
4. Verify operation completes

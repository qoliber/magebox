# RabbitMQ

RabbitMQ provides message queue functionality for Magento's asynchronous operations.

## Configuration

Enable RabbitMQ in your `.magebox.yaml` file:

```yaml
services:
  rabbitmq: true
```

## Connection Details

| Property | Value |
|----------|-------|
| Host | 127.0.0.1 |
| AMQP Port | 5672 |
| Management UI Port | 15672 |
| Username | guest |
| Password | guest |
| Virtual Host | / |

## Management UI

Access the RabbitMQ management interface:

```
http://localhost:15672
```

Login with:
- Username: `guest`
- Password: `guest`

## Magento Configuration

Configure Magento to use RabbitMQ (`app/etc/env.php`):

```php
'queue' => [
    'amqp' => [
        'host' => '127.0.0.1',
        'port' => '5672',
        'user' => 'guest',
        'password' => 'guest',
        'virtualhost' => '/'
    ]
]
```

Or via CLI:

```bash
magebox cli setup:config:set \
    --amqp-host=127.0.0.1 \
    --amqp-port=5672 \
    --amqp-user=guest \
    --amqp-password=guest \
    --amqp-virtualhost=/
```

## Consumer Management

### List Consumers

```bash
magebox cli queue:consumers:list
```

### Start a Consumer

```bash
magebox cli queue:consumers:start <consumer_name>
```

Common consumers:
- `async.operations.all` - General async operations
- `product_action_attribute.update` - Bulk attribute updates
- `product_action_attribute.website.update` - Website assignments
- `codegeneratorProcessor` - Coupon code generation
- `exportProcessor` - Export operations
- `inventory.reservations.updateSalabilityStatus` - Inventory updates

### Start All Consumers

```bash
# Start in background
for consumer in $(magebox cli queue:consumers:list); do
    magebox cli queue:consumers:start $consumer &
done
```

### Run Consumer Once

Process messages and exit:

```bash
magebox cli queue:consumers:start async.operations.all --max-messages=100
```

## Common Operations

### View Queues

Via Management UI or CLI:

```bash
curl -u guest:guest http://localhost:15672/api/queues
```

### Purge a Queue

```bash
curl -u guest:guest -X DELETE \
    http://localhost:15672/api/queues/%2F/async.operations.all/contents
```

### Check Queue Depth

```bash
curl -u guest:guest \
    http://localhost:15672/api/queues/%2F/async.operations.all | jq '.messages'
```

## Use Cases

### Bulk Product Updates

When updating many products:

1. Ensure RabbitMQ is running
2. Perform bulk update in Admin
3. Start the consumer:

```bash
magebox cli queue:consumers:start product_action_attribute.update
```

### Async Email Sending

Configure async email in Magento, then:

```bash
magebox cli queue:consumers:start async.operations.all
```

### Inventory Updates

For MSI (Multi-Source Inventory):

```bash
magebox cli queue:consumers:start inventory.reservations.updateSalabilityStatus
```

## Troubleshooting

### Connection Refused

Check if RabbitMQ is running:

```bash
docker ps | grep rabbitmq
```

Start services:

```bash
magebox global start
```

### Messages Not Processing

1. Verify consumer is running:

```bash
ps aux | grep queue:consumers
```

2. Check queue has messages:

```bash
curl -u guest:guest http://localhost:15672/api/queues/%2F/async.operations.all
```

3. Check for errors in Magento logs:

```bash
magebox logs exception.log
```

### Consumer Crashes

If consumers keep dying:

1. Check memory limits in PHP
2. Look for exceptions in logs
3. Try processing fewer messages:

```bash
magebox cli queue:consumers:start async.operations.all --max-messages=10
```

### Queue Backlog

If queues are backing up:

1. Start more consumer instances
2. Check consumer performance
3. Consider running consumers in supervisor/systemd

## Development Tips

1. **Disable async in development** - For easier debugging, you can process operations synchronously by disabling async:

```bash
magebox cli config:set dev/grid/async_indexing 0
```

2. **Monitor queue depth** - Keep an eye on queue sizes during development

3. **Use max-messages** - Prevent runaway consumers:

```bash
magebox cli queue:consumers:start <name> --max-messages=1000
```

4. **Check consumer status** regularly when debugging async issues

## When to Use RabbitMQ

Enable RabbitMQ when using:
- Bulk product operations
- Async email sending
- MSI (Multi-Source Inventory)
- B2B features (shared catalogs, quotes)
- Custom async operations

For simple development without these features, you can disable RabbitMQ:

```yaml
services:
  rabbitmq: false
```

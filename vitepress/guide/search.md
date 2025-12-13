# OpenSearch / Elasticsearch

MageBox supports both OpenSearch and Elasticsearch for Magento catalog search, with automatic plugin installation.

## Configuration

### OpenSearch (Recommended)

```yaml
services:
  # Simple format (uses default 1GB memory)
  opensearch: "2.19"

  # Extended format with memory allocation
  opensearch:
    version: "2.19"
    memory: "2g"    # Recommended for production-like performance
```

### Elasticsearch

```yaml
services:
  # Simple format
  elasticsearch: "8.11"

  # Extended format
  elasticsearch:
    version: "8.11"
    memory: "2g"
```

::: tip
OpenSearch is recommended for new projects. It's fully compatible with Elasticsearch and is the default for Magento 2.4.6+.
:::

## Automatic Plugin Installation

MageBox automatically installs the required Magento plugins:

| Plugin | Description |
|--------|-------------|
| **analysis-icu** | International Components for Unicode (ICU) analyzer |
| **analysis-phonetic** | Phonetic token filter for soundex and metaphone algorithms |

These plugins are required by Magento for proper search functionality and are installed automatically when the container starts.

## Memory Configuration

By default, search engines are allocated **1GB of RAM**. For better performance:

```yaml
services:
  opensearch:
    version: "2.19"
    memory: "2g"    # 2GB for production-like performance
```

**Recommended settings:**
- Development: `1g` (default)
- Production-like testing: `2g`
- Large catalogs: `4g`

## Connection Details

| Property | Value |
|----------|-------|
| Host | 127.0.0.1 |
| Port | 9200 |
| Protocol | http |

## Magento Configuration

### OpenSearch

Via CLI:

```bash
php bin/magento config:set catalog/search/engine opensearch
php bin/magento config:set catalog/search/opensearch_server_hostname 127.0.0.1
php bin/magento config:set catalog/search/opensearch_server_port 9200
php bin/magento config:set catalog/search/opensearch_index_prefix magento2
```

Or in `app/etc/env.php`:

```php
'system' => [
    'default' => [
        'catalog' => [
            'search' => [
                'engine' => 'opensearch',
                'opensearch_server_hostname' => '127.0.0.1',
                'opensearch_server_port' => '9200',
                'opensearch_index_prefix' => 'magento2',
                'opensearch_enable_auth' => '0',
                'opensearch_server_timeout' => '15'
            ]
        ]
    ]
]
```

### Elasticsearch 7

```php
'system' => [
    'default' => [
        'catalog' => [
            'search' => [
                'engine' => 'elasticsearch7',
                'elasticsearch7_server_hostname' => '127.0.0.1',
                'elasticsearch7_server_port' => '9200',
                'elasticsearch7_index_prefix' => 'magento2',
                'elasticsearch7_enable_auth' => '0',
                'elasticsearch7_server_timeout' => '15'
            ]
        ]
    ]
]
```

### Elasticsearch 8

```php
'system' => [
    'default' => [
        'catalog' => [
            'search' => [
                'engine' => 'elasticsearch8',
                'elasticsearch8_server_hostname' => '127.0.0.1',
                'elasticsearch8_server_port' => '9200',
                'elasticsearch8_index_prefix' => 'magento2',
                'elasticsearch8_enable_auth' => '0',
                'elasticsearch8_server_timeout' => '15'
            ]
        ]
    ]
]
```

## Reindexing

After configuring search, reindex the catalog:

```bash
php bin/magento indexer:reindex catalogsearch_fulltext
```

## Supported Versions

### OpenSearch

- 2.19.x (latest, recommended)
- 2.x series

### Elasticsearch

- 8.11 (for Magento 2.4.6+)
- 7.17 (for Magento 2.4.4+)

## Common Operations

### Check Cluster Health

```bash
curl http://127.0.0.1:9200/_cluster/health?pretty
```

### List Indices

```bash
curl http://127.0.0.1:9200/_cat/indices?v
```

### View Index Mapping

```bash
curl http://127.0.0.1:9200/magento2_product_1/_mapping?pretty
```

### Delete All Indices

```bash
curl -X DELETE http://127.0.0.1:9200/magento2_*
```

Then reindex:

```bash
php bin/magento indexer:reindex catalogsearch_fulltext
```

### Check Installed Plugins

```bash
curl http://127.0.0.1:9200/_cat/plugins
```

Should show `analysis-icu` and `analysis-phonetic`.

## Troubleshooting

### Connection Refused

Check if the container is running:

```bash
docker ps | grep -E "opensearch|elastic"
```

Start services:

```bash
magebox global start
```

### Search Not Working

1. Verify the search engine is configured:

```bash
php bin/magento config:show catalog/search/engine
```

2. Check the service is accessible:

```bash
curl http://127.0.0.1:9200
```

3. Reindex:

```bash
php bin/magento indexer:reindex catalogsearch_fulltext
```

### Index Not Found

If you see "index not found" errors:

```bash
# Delete stale indices
curl -X DELETE http://127.0.0.1:9200/magento2_*

# Reindex
php bin/magento indexer:reindex catalogsearch_fulltext
```

### Memory Issues

OpenSearch/Elasticsearch requires significant memory. Check container resources:

```bash
docker stats
```

Increase memory allocation in your config:

```yaml
services:
  opensearch:
    version: "2.19"
    memory: "2g"    # Increase from default 1g
```

Then restart:

```bash
magebox restart
```

### Plugin Missing Error

If Magento reports missing ICU or Phonetic plugin:

1. Check plugins are installed:
```bash
curl http://127.0.0.1:9200/_cat/plugins
```

2. Restart the search container:
```bash
docker restart $(docker ps -qf "name=opensearch")
```

3. If still missing, recreate the container:
```bash
magebox stop
magebox start
```

### Slow Indexing

For faster indexing:

1. Increase PHP memory limit
2. Use batched indexing
3. Increase search engine memory
4. Disable real-time indexing during initial setup

```bash
# Set to manual mode for bulk operations
php bin/magento indexer:set-mode schedule catalogsearch_fulltext

# After bulk import, reindex manually
php bin/magento indexer:reindex catalogsearch_fulltext

# Restore real-time
php bin/magento indexer:set-mode realtime catalogsearch_fulltext
```

## Switching Search Engines

To switch from Elasticsearch to OpenSearch:

1. Update `.magebox.yaml`:

```yaml
services:
  # elasticsearch: "7.17"  # Remove
  opensearch:              # Add
    version: "2.19"
    memory: "2g"
```

2. Restart services:

```bash
magebox stop
magebox start
```

3. Update Magento config:

```bash
php bin/magento config:set catalog/search/engine opensearch
php bin/magento config:set catalog/search/opensearch_server_hostname 127.0.0.1
php bin/magento config:set catalog/search/opensearch_server_port 9200
```

4. Reindex:

```bash
php bin/magento indexer:reindex catalogsearch_fulltext
```

## Performance Tips

1. **Allocate sufficient memory** - At least 1GB for development, 2GB for production-like testing

2. **Use SSD storage** - Search indices benefit greatly from fast disk I/O

3. **Index only necessary attributes** - Review which attributes are searchable

4. **Use index prefixes** - Separate indices per environment to avoid conflicts

5. **Monitor cluster health** - Regularly check for yellow/red status

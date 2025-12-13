# OpenSearch / Elasticsearch

MageBox runs OpenSearch or Elasticsearch in Docker with Magento-required plugins pre-installed.

## Overview

Search engines power Magento's catalog search, providing:

- Product search with relevance ranking
- Layered navigation (filters)
- Search suggestions
- Category browsing optimization

## Supported Versions

### OpenSearch (Recommended)

| Version | Port | Magento Compatibility |
|---------|------|----------------------|
| OpenSearch 2.x | 9200 | Magento 2.4.6+ |
| OpenSearch 2.19 | 9200 | Latest (default) |

### Elasticsearch

| Version | Port | Magento Compatibility |
|---------|------|----------------------|
| Elasticsearch 7.17 | 9200 | Magento 2.4.4 - 2.4.5 |
| Elasticsearch 8.x | 9200 | Magento 2.4.6+ |

::: tip
OpenSearch is recommended for new projects. It's a community-driven fork of Elasticsearch with full Magento compatibility.
:::

## Configuration

### Basic Configuration

```yaml
# .magebox.yaml
services:
  opensearch: "2.19"
```

Or for Elasticsearch:

```yaml
services:
  elasticsearch: "8.11"
```

### With Memory Allocation

For better performance (recommended for large catalogs):

```yaml
services:
  opensearch:
    version: "2.19"
    memory: "2g"  # Default is 1g
```

## Pre-installed Plugins

MageBox automatically installs these Magento-required plugins:

| Plugin | Purpose |
|--------|---------|
| `analysis-icu` | International Components for Unicode (ICU) analyzer |
| `analysis-phonetic` | Phonetic token filter (soundex, metaphone) |

These plugins enable:
- Multi-language search support
- "Did you mean" suggestions
- Phonetic matching (finding similar-sounding words)

## Connection Details

| Setting | Value |
|---------|-------|
| Host | `127.0.0.1` |
| Port | `9200` |
| Protocol | HTTP |

## Magento Configuration

### Via Install Command

```bash
php bin/magento setup:install \
    --search-engine=opensearch \
    --opensearch-host=127.0.0.1 \
    --opensearch-port=9200 \
    --opensearch-index-prefix=magento2 \
    --opensearch-timeout=15 \
    # ... other options
```

For Elasticsearch:

```bash
php bin/magento setup:install \
    --search-engine=elasticsearch8 \
    --elasticsearch-host=127.0.0.1 \
    --elasticsearch-port=9200 \
    --elasticsearch-index-prefix=magento2 \
    --elasticsearch-timeout=15 \
    # ... other options
```

### Via Admin Panel

1. Go to **Stores → Configuration → Catalog → Catalog → Catalog Search**
2. Set **Search Engine** to OpenSearch or Elasticsearch
3. Configure connection details
4. Save and reindex

### Via env.php

```php
// app/etc/env.php
'system' => [
    'default' => [
        'catalog' => [
            'search' => [
                'engine' => 'opensearch',
                'opensearch_server_hostname' => '127.0.0.1',
                'opensearch_server_port' => '9200',
                'opensearch_index_prefix' => 'magento2',
                'opensearch_server_timeout' => '15'
            ]
        ]
    ]
]
```

## Common Operations

### Check Cluster Health

```bash
curl http://127.0.0.1:9200/_cluster/health?pretty
```

Response should show `"status": "green"` or `"yellow"`.

### View Indices

```bash
curl http://127.0.0.1:9200/_cat/indices?v
```

### Reindex Catalog

```bash
php bin/magento indexer:reindex catalogsearch_fulltext
```

### Delete and Recreate Indices

```bash
# Delete all Magento indices
curl -X DELETE http://127.0.0.1:9200/magento2_*

# Reindex
php bin/magento indexer:reindex catalogsearch_fulltext
```

### Check Installed Plugins

```bash
curl http://127.0.0.1:9200/_cat/plugins?v
```

Should show `analysis-icu` and `analysis-phonetic`.

## Docker Container

### Container Status

```bash
docker ps | grep opensearch
# or
docker ps | grep elasticsearch
```

### Container Logs

```bash
docker logs magebox-opensearch-2.19

# Follow logs
docker logs -f magebox-opensearch-2.19
```

### Restart Container

```bash
docker restart magebox-opensearch-2.19
```

## Memory Configuration

### Recommended Settings

| Catalog Size | Recommended Memory |
|--------------|-------------------|
| < 10,000 products | 1g (default) |
| 10,000 - 50,000 products | 2g |
| 50,000+ products | 4g |

### Setting Memory

```yaml
# .magebox.yaml
services:
  opensearch:
    version: "2.19"
    memory: "2g"
```

After changing, restart services:

```bash
magebox restart
```

### Checking Memory Usage

```bash
curl http://127.0.0.1:9200/_nodes/stats/jvm?pretty | grep heap
```

## Troubleshooting

### No Alive Nodes Found

```
No alive nodes found in your cluster
```

**Solutions:**

1. Check container is running:
   ```bash
   docker ps | grep opensearch
   ```

2. Check service is accessible:
   ```bash
   curl http://127.0.0.1:9200
   ```

3. Start services:
   ```bash
   magebox global start
   ```

### Cluster Health Red

```bash
curl http://127.0.0.1:9200/_cluster/health?pretty
```

If status is "red":

```bash
# Delete all indices and reindex
curl -X DELETE http://127.0.0.1:9200/magento2_*
php bin/magento indexer:reindex catalogsearch_fulltext
```

### Out of Memory

If container keeps restarting:

1. Increase memory allocation:
   ```yaml
   services:
     opensearch:
       version: "2.19"
       memory: "2g"
   ```

2. Restart services:
   ```bash
   magebox restart
   ```

### Index Not Found

```
index_not_found_exception
```

**Solution:** Reindex:

```bash
php bin/magento indexer:reindex catalogsearch_fulltext
```

### Search Not Returning Results

1. Check indexer status:
   ```bash
   php bin/magento indexer:status
   ```

2. Reindex if needed:
   ```bash
   php bin/magento indexer:reindex
   ```

3. Verify index exists:
   ```bash
   curl http://127.0.0.1:9200/_cat/indices?v
   ```

### Plugin Missing Error

If Magento reports missing ICU or phonetic plugin:

```bash
# Check plugins
curl http://127.0.0.1:9200/_cat/plugins?v

# If missing, restart container (MageBox installs them automatically)
docker restart magebox-opensearch-2.19
```

## Performance Tips

### Index Optimization

After large catalog imports:

```bash
# Force merge (reduces segment count)
curl -X POST "http://127.0.0.1:9200/magento2_*/_forcemerge?max_num_segments=1"
```

### Query Debugging

Enable search debugging in Magento:

```bash
php bin/magento config:set dev/debug/debug_logging 1
```

Then check `var/log/debug.log` for search queries.

### Monitoring

```bash
# Watch indexing in real-time
watch -n 1 'curl -s http://127.0.0.1:9200/_cat/indices?v | grep magento'

# Check pending tasks
curl http://127.0.0.1:9200/_cluster/pending_tasks?pretty
```

## Switching Search Engines

To switch from Elasticsearch to OpenSearch:

1. Export configuration (optional)
2. Update `.magebox.yaml`:
   ```yaml
   services:
     opensearch: "2.19"
     # elasticsearch: "8.11"  # Comment out or remove
   ```

3. Restart services:
   ```bash
   magebox restart
   ```

4. Update Magento configuration:
   ```bash
   php bin/magento config:set catalog/search/engine opensearch
   ```

5. Reindex:
   ```bash
   php bin/magento indexer:reindex catalogsearch_fulltext
   ```

# Elasticvue

MageBox supports [Elasticvue](https://elasticvue.com/), a browser-based UI for managing and browsing your OpenSearch or Elasticsearch data.

## Overview

Elasticvue provides a visual interface for:

- **Browsing indices** and viewing documents
- **Running queries** against your search engine
- **Inspecting mappings** and index settings
- **Monitoring cluster health**, node info, and shard allocation

## Enabling Elasticvue

### Via Command

```bash
magebox elasticvue enable
```

### Via Global Config

```bash
magebox config set elasticvue true
```

Then start services:

```bash
magebox global start
```

## Connection Details

| Setting | Value |
|---------|-------|
| Web UI | `http://localhost:8080` |

## Getting Started

1. Enable Elasticvue:
   ```bash
   magebox elasticvue enable
   ```

2. Open **http://localhost:8080** in your browser

3. Add your cluster:
   - Click **Add Cluster**
   - Enter URI: `http://localhost:9200`
   - Click **Connect**

You can now browse your Magento search indices (e.g., `magento2_product_1`), view documents, and run queries.

## MageBox Commands

### Check Status

```bash
magebox elasticvue status
```

Shows whether Elasticvue is enabled and running.

### Disable

```bash
magebox elasticvue disable
```

Stops the container and removes it from docker-compose.

## Docker Container

Elasticvue runs as a Docker container (`magebox-elasticvue`) with:

- **Image**: `cars10/elasticvue:latest`
- **Port**: 8080 (Web UI)
- **Network**: magebox

## Troubleshooting

### Cannot Connect to Cluster

Ensure OpenSearch or Elasticsearch is running:

```bash
curl http://127.0.0.1:9200
```

If not, start services:

```bash
magebox global start
```

### Port 8080 Already in Use

If another service is using port 8080, stop it before enabling Elasticvue:

```bash
lsof -i :8080
```

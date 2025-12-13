# Multi-Domain Setup

MageBox supports multiple domains for a single Magento installation.

## Use Cases

- **Multi-store**: Different storefronts (store.com, store.de, store.fr)
- **Subdomains**: admin.store.test, api.store.test
- **Multi-website**: Separate websites with unique domains

## Domain Commands

MageBox provides CLI commands to manage domains:

```bash
# Add a new domain
magebox domain add store.test

# Add with store code for multi-store
magebox domain add de.store.test --store-code=german

# Add with custom options
magebox domain add api.store.test --root=pub --ssl=true

# Remove a domain
magebox domain remove old.store.test

# List all domains
magebox domain list
```

When adding a domain, MageBox automatically:
- Updates `.magebox.yaml`
- Generates SSL certificate
- Creates nginx vhost with `MAGE_RUN_CODE` support
- Updates `/etc/hosts` (if using hosts mode)
- Reloads nginx

## Configuration

### Basic Multi-Domain

```yaml
name: mystore

domains:
  - host: mystore.test
    root: pub
    ssl: true
  - host: admin.mystore.test
    root: pub
    ssl: true
  - host: api.mystore.test
    root: pub
    ssl: true

php: "8.2"

services:
  mysql: "8.0"
  redis: true
```

### Different Document Roots

For non-standard setups:

```yaml
domains:
  - host: mystore.test
    root: pub
  - host: legacy.mystore.test
    root: public_html
```

## Magento Store Configuration

### Single Website, Multiple Stores

In Magento Admin:
1. **Stores > All Stores**
2. Create Store Views for each domain
3. Configure base URLs for each store view

### Multiple Websites

1. Create websites in Magento Admin
2. Create stores and store views
3. Configure each domain in `.magebox.yaml`
4. Set up store codes (see below)

## Store Code Mapping

### Using store_code in Configuration (Recommended)

MageBox supports `store_code` directly in domain configuration. This sets `MAGE_RUN_CODE` in nginx:

```yaml
name: mystore

domains:
  - host: mystore.test
    root: pub
    store_code: default
  - host: de.mystore.test
    root: pub
    store_code: german
  - host: fr.mystore.test
    root: pub
    store_code: french

php: "8.2"

services:
  mysql: "8.0"
```

Or add domains with store codes via CLI:

```bash
magebox domain add mystore.test
magebox domain add de.mystore.test --store-code=german
magebox domain add fr.mystore.test --store-code=french
```

### Using Environment Variables

For global store type configuration:

```yaml
env:
  MAGE_RUN_TYPE: store  # or "website" for multi-website
```

### Using Custom Nginx Configuration

For more complex routing, create `~/.magebox/nginx/vhosts/mystore.test.conf.custom`:

```nginx
map $http_host $MAGE_RUN_CODE {
    default default;
    de.mystore.test german;
    fr.mystore.test french;
}
```

## Example: Multi-Store Setup

### Magento Configuration

1. Create store views: `default`, `german`, `french`
2. Set base URLs for each:
   - default: https://mystore.test/
   - german: https://de.mystore.test/
   - french: https://fr.mystore.test/

### MageBox Configuration

```yaml
name: mystore

domains:
  - host: mystore.test
    root: pub
    store_code: default
  - host: de.mystore.test
    root: pub
    store_code: german
  - host: fr.mystore.test
    root: pub
    store_code: french

php: "8.2"

services:
  mysql: "8.0"
  redis: true
```

Or configure via CLI:

```bash
magebox init mystore
magebox domain add mystore.test
magebox domain add de.mystore.test --store-code=german
magebox domain add fr.mystore.test --store-code=french
```

### app/etc/env.php

```php
'system' => [
    'default' => [
        'web' => [
            'unsecure' => [
                'base_url' => 'https://mystore.test/'
            ],
            'secure' => [
                'base_url' => 'https://mystore.test/'
            ]
        ]
    ],
    'stores' => [
        'german' => [
            'web' => [
                'unsecure' => [
                    'base_url' => 'https://de.mystore.test/'
                ],
                'secure' => [
                    'base_url' => 'https://de.mystore.test/'
                ]
            ]
        ],
        'french' => [
            'web' => [
                'unsecure' => [
                    'base_url' => 'https://fr.mystore.test/'
                ],
                'secure' => [
                    'base_url' => 'https://fr.mystore.test/'
                ]
            ]
        ]
    ]
]
```

## Example: Multi-Website Setup

### Magento Configuration

1. Create websites: `base`, `b2b`
2. Create stores and store views for each
3. Set website codes

### MageBox Configuration

```yaml
name: multisite

domains:
  - host: store.test
    root: pub
  - host: b2b.test
    root: pub

php: "8.2"

services:
  mysql: "8.0"
```

### Store Code Routing

Add to `pub/index.php`:

```php
$params = $_SERVER;

$websites = [
    'store.test' => ['code' => 'base', 'type' => 'website'],
    'b2b.test' => ['code' => 'b2b', 'type' => 'website'],
];

$host = $_SERVER['HTTP_HOST'];
if (isset($websites[$host])) {
    $params[\Magento\Store\Model\StoreManager::PARAM_RUN_CODE] = $websites[$host]['code'];
    $params[\Magento\Store\Model\StoreManager::PARAM_RUN_TYPE] = $websites[$host]['type'];
}

$bootstrap = \Magento\Framework\App\Bootstrap::create(BP, $params);
```

## SSL Certificates

MageBox automatically generates SSL certificates for all configured domains:

```bash
# Regenerate certificates
magebox ssl generate
```

Certificates are stored in `~/.magebox/certs/`.

## DNS Resolution

### Hosts Mode

All domains are added to `/etc/hosts`:

```
127.0.0.1 mystore.test
127.0.0.1 de.mystore.test
127.0.0.1 fr.mystore.test
```

### Dnsmasq Mode

Wildcard resolution handles all subdomains automatically:

```
*.test → 127.0.0.1
```

## Troubleshooting

### Domain Not Resolving

Check DNS configuration:

```bash
ping mystore.test
```

If using hosts mode, verify entries:

```bash
cat /etc/hosts | grep mystore
```

### Wrong Store Loading

1. Clear Magento cache:

```bash
magebox cli cache:flush
```

2. Check store configuration in Admin
3. Verify base URLs match domain configuration

### SSL Certificate Issues

Regenerate certificates:

```bash
magebox ssl generate
```

Trust the CA:

```bash
magebox ssl trust
```

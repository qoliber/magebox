# SSL Certificates

MageBox provides automatic HTTPS for all your development domains using mkcert.

## How It Works

MageBox uses [mkcert](https://github.com/FiloSottile/mkcert) to:

1. Create a local Certificate Authority (CA)
2. Install the CA in your system trust store
3. Generate certificates for your project domains

## Initial Setup

SSL is configured during bootstrap:

```bash
magebox bootstrap
```

This:
- Installs the mkcert root CA
- Trusts the CA in your browser
- Creates the certificate directory

## Certificate Commands

### Trust CA

If you need to re-trust the certificate authority:

```bash
magebox ssl trust
```

### Generate Certificates

Regenerate certificates for all configured domains:

```bash
magebox ssl generate
```

This reads domains from your `.magebox.yaml` file and creates certificates.

## Configuration

### Enable SSL (Default)

SSL is enabled by default:

```yaml
domains:
  - host: mystore.test
    root: pub
    ssl: true  # Default
```

### Disable SSL

For HTTP-only access:

```yaml
domains:
  - host: mystore.test
    root: pub
    ssl: false
```

## Certificate Location

Certificates are stored in `~/.magebox/certs/` on all platforms (macOS and Linux):

```
~/.magebox/certs/
├── mystore.test/
│   ├── cert.pem        # Domain certificate
│   └── key.pem         # Domain private key
└── another.test/
    ├── cert.pem
    └── key.pem
```

::: info Linux Note
On Linux, nginx is configured to run as your user during bootstrap. This allows nginx to read certificates from your home directory without requiring root permissions.
:::

## Browser Trust

### Automatic Trust

The `magebox bootstrap` and `magebox ssl trust` commands automatically:

- **macOS**: Add CA to System Keychain
- **Linux**: Add CA to system CA store and NSS (Firefox)

### Manual Trust

If automatic trust fails:

#### macOS

```bash
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain ~/.magebox/certs/rootCA.pem
```

#### Linux (Ubuntu/Debian)

```bash
sudo cp ~/.magebox/certs/rootCA.pem /usr/local/share/ca-certificates/magebox-ca.crt
sudo update-ca-certificates
```

#### Firefox

Firefox uses its own certificate store:

1. Open Firefox Settings
2. Search for "Certificates"
3. Click "View Certificates"
4. Import `~/.magebox/certs/rootCA.pem`
5. Trust for websites

## Troubleshooting

### "Not Secure" Warning

If browsers show security warnings:

1. Re-trust the CA:

```bash
magebox ssl trust
```

2. Regenerate certificates:

```bash
magebox ssl generate
```

3. Restart browser (especially Firefox)

### Certificate Not Found

If Nginx can't find certificates:

```bash
# Check certificates exist
ls -la ~/.magebox/certs/

# Regenerate
magebox ssl generate
```

### mkcert Not Installed

Install mkcert:

```bash
# macOS
brew install mkcert

# Ubuntu/Debian
sudo apt install libnss3-tools
curl -JLO "https://dl.filippo.io/mkcert/latest?for=linux/amd64"
chmod +x mkcert-v*-linux-amd64
sudo mv mkcert-v*-linux-amd64 /usr/local/bin/mkcert
```

Then re-run bootstrap:

```bash
magebox bootstrap
```

### Firefox Still Shows Warning

Firefox requires special handling:

```bash
# Install to Firefox's NSS store
mkcert -install
```

Or manually import the CA in Firefox settings.

## Multiple Domains

When you have multiple domains:

```yaml
domains:
  - host: mystore.test
  - host: api.mystore.test
  - host: admin.mystore.test
```

MageBox generates a single certificate covering all domains (SAN certificate).

## API and CLI Access

For CLI tools that need HTTPS:

### curl

```bash
# Should work automatically after trusting CA
curl https://mystore.test/

# If not, specify CA
curl --cacert ~/.magebox/certs/rootCA.pem https://mystore.test/
```

### PHP

PHP uses the system CA store, which should include the MageBox CA after `ssl trust`.

If needed:

```php
$context = stream_context_create([
    'ssl' => [
        'cafile' => getenv('HOME') . '/.magebox/certs/rootCA.pem'
    ]
]);
```

## Security Notes

- The root CA private key is stored locally
- Never share your CA or private keys
- The CA is only trusted on your machine
- Certificates are for development only

## Comparison with Production

| Aspect | MageBox (Dev) | Production |
|--------|---------------|------------|
| Certificate Source | mkcert (local CA) | Let's Encrypt / Commercial |
| Trust | Manual/Automatic | Public CA |
| Renewal | Manual regeneration | Automatic |
| Validation | None | Domain / Organization |

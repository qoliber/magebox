# Nginx Vhost Template

This directory contains the Nginx vhost configuration template used by MageBox.

## Template File

- `vhost.conf.tmpl` - Nginx vhost configuration template for Magento 2

## Available Variables

The template has access to the following variables:

| Variable | Type | Description | Example |
|----------|------|-------------|---------|
| `ProjectName` | string | Name of the project | `mystore` |
| `Domain` | string | Domain name | `mystore.test` |
| `DocumentRoot` | string | Absolute path to document root | `/var/www/mystore/pub` |
| `PHPVersion` | string | PHP version | `8.2` |
| `PHPSocketPath` | string | Path to PHP-FPM socket | `/tmp/magebox/mystore-php8.2.sock` |
| `SSLEnabled` | bool | Whether SSL is enabled | `true` |
| `SSLCertFile` | string | Path to SSL certificate file (only if SSLEnabled=true) | `/path/to/cert.pem` |
| `SSLKeyFile` | string | Path to SSL key file (only if SSLEnabled=true) | `/path/to/key.pem` |
| `UseVarnish` | bool | Whether Varnish is enabled (currently unused) | `false` |
| `VarnishPort` | int | Varnish port number (currently unused) | `6081` |

## Template Syntax

The template uses Go's `text/template` syntax:

```go
// Conditional blocks
{{if .SSLEnabled}}
server {
    listen 8443 ssl http2;
    ssl_certificate {{.SSLCertFile}};
    ssl_certificate_key {{.SSLKeyFile}};
}
{{else}}
server {
    listen 8080;
}
{{end}}

// Variable substitution
upstream fastcgi_backend_{{.ProjectName}} {
    server unix:{{.PHPSocketPath}};
}
```

## Magento 2 Specific Configuration

The template includes Magento 2 optimized configuration:

- Static content serving with proper caching headers
- Media file handling
- PHP-FPM FastCGI configuration
- Gzip compression
- Security headers
- Error page handling

## Modifying the Template

1. Edit `vhost.conf.tmpl` with your changes
2. Rebuild MageBox: `go build -o magebox ./cmd/magebox`
3. The template is embedded in the binary at compile time

## Testing

Template validity is tested in `vhost_test.go`:

```bash
go test ./internal/nginx -run TestVhostTemplateValidity
```

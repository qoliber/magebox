# Varnish VCL Template

This directory contains the Varnish VCL configuration template used by MageBox.

## Template File

- `default.vcl.tmpl` - Varnish VCL configuration template optimized for Magento 2

## Available Variables

The template has access to the following variables:

### Root Level Variables

| Variable | Type | Description | Example |
|----------|------|-------------|---------|
| `Backends` | []BackendConfig | Array of backend configurations | See below |
| `DefaultBackend` | string | Name of the default backend to use | `mystore` |
| `GracePeriod` | string | Grace period for serving stale content | `300s` |
| `PurgeACL` | []string | Array of IP addresses/ranges allowed to purge | `["localhost", "127.0.0.1"]` |

### BackendConfig Structure

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `Name` | string | Backend name (sanitized project name) | `mystore` |
| `Host` | string | Backend host | `127.0.0.1` |
| `Port` | int | Backend port | `80` |
| `ProbeURL` | string | Health check URL | `/health_check.php` |
| `ProbeInterval` | string | Health check interval | `5s` |

## Template Syntax

The template uses Go's `text/template` syntax:

```go
// Iterating over backends
{{range .Backends}}
backend {{.Name}} {
    .host = "{{.Host}}";
    .port = "{{.Port}}";
    {{if .ProbeURL}}
    .probe = {
        .url = "{{.ProbeURL}}";
        .interval = {{.ProbeInterval}};
    }
    {{end}}
}
{{end}}

// Iterating over ACL entries
acl purge {
{{range .PurgeACL}}
    "{{.}}";
{{end}}
}

// Using variables in VCL subroutines
set req.backend_hint = {{.DefaultBackend}};
set beresp.grace = {{.GracePeriod}};
```

## Magento 2 Specific Configuration

The VCL template includes Magento 2 optimized configuration:

### Cache Management
- **PURGE** requests - Purge specific URLs
- **BAN** requests - Ban by Magento tags pattern
- `X-Magento-Tags-Pattern` header support
- `X-Magento-Cache-Control` header support
- `X-Magento-Vary` cookie handling

### URL Handling
- Marketing parameter stripping (utm_, gclid, fbclid, etc.)
- Bypass rules for admin, checkout, customer areas
- Static content caching
- Health check bypass

### Cookie Management
- Frontend session cookie detection
- Admin session cookie detection
- Store/currency variation support

### Response Handling
- Grace mode for serving stale content
- TTL configuration based on Magento headers
- Static content long-term caching
- Set-Cookie bypass

## Modifying the Template

1. Edit `default.vcl.tmpl` with your changes
2. Rebuild MageBox: `go build -o magebox ./cmd/magebox`
3. The template is embedded in the binary at compile time

## VCL Version

The template uses **VCL 4.1** syntax.

## Testing

Template validity is tested in `vcl_test.go`:

```bash
go test ./internal/varnish -run TestVCLTemplateValidity
```

## Debugging

Enable debug headers by setting `X-Magento-Debug` header:

- `X-Cache: HIT` or `X-Cache: MISS` - Cache status
- `X-Cache-Hits` - Number of hits for cached object

In production, these headers are automatically removed.

# PHP-FPM Pool Template

This directory contains the PHP-FPM pool configuration template used by MageBox.

## Template File

- `pool.conf.tmpl` - PHP-FPM pool configuration template

## Available Variables

The template has access to the following variables:

| Variable | Type | Description | Example |
|----------|------|-------------|---------|
| `ProjectName` | string | Name of the project | `mystore` |
| `PHPVersion` | string | PHP version | `8.2` |
| `SocketPath` | string | Path to PHP-FPM socket | `/tmp/magebox/mystore-php8.2.sock` |
| `LogPath` | string | Path to PHP-FPM error log | `~/.magebox/logs/php-fpm/mystore-error.log` |
| `User` | string | System user running PHP-FPM | `jakub` |
| `Group` | string | System group running PHP-FPM | `staff` |
| `MaxChildren` | int | Maximum number of child processes | `10` |
| `StartServers` | int | Number of child processes created on startup | `2` |
| `MinSpareServers` | int | Minimum number of idle server processes | `1` |
| `MaxSpareServers` | int | Maximum number of idle server processes | `3` |
| `MaxRequests` | int | Number of requests each child process should execute before respawning | `500` |
| `Env` | map[string]string | Environment variables to set | `{"MAGE_MODE": "developer"}` |
| `PHPINI` | map[string]string | PHP INI overrides | `{"opcache.enable": "0"}` |

## Template Syntax

The template uses Go's `text/template` syntax:

```go
// Simple variable substitution
[{{.ProjectName}}]
listen = {{.SocketPath}}

// Iterating over maps
{{range $key, $value := .PHPINI}}
php_admin_value[{{$key}}] = {{$value}}
{{end}}
```

## Modifying the Template

1. Edit `pool.conf.tmpl` with your changes
2. Rebuild MageBox: `go build -o magebox ./cmd/magebox`
3. The template is embedded in the binary at compile time

## Testing

Template validity is tested in `pool_test.go`:

```bash
go test ./internal/php -run TestPoolTemplateValidity
```

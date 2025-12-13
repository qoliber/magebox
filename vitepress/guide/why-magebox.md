# Why MageBox?

A detailed comparison of MageBox with other Magento development environments.

## Performance Comparison

| Metric | MageBox | Warden | DDEV | Valet+ |
|--------|---------|--------|------|--------|
| Page load (uncached) | ~200ms | ~800ms | ~700ms | ~250ms |
| File sync latency | 0ms | 50-200ms | 50-200ms | 0ms |
| Cold start time | 2s | 30-60s | 20-40s | N/A |
| Memory usage | ~500MB | ~2-4GB | ~2-3GB | ~400MB |
| Multi-PHP switching | Instant | Rebuild | Rebuild | Manual |

## File System Performance

The biggest difference between MageBox and Docker-based solutions is file system performance.

### Docker Volume Issues

Docker-based tools must synchronize files between your host machine and containers:

```
Host Machine               Container
    │                          │
    └── Your Code ──sync──→ Container Volume
                    (latency)
```

This sync layer adds latency to every file operation. Magento performs thousands of file operations per request, making this overhead significant.

### MageBox Native Approach

MageBox accesses files directly:

```
Host Machine
    │
    └── Your Code ←── PHP-FPM (native)
         (zero latency)
```

No sync. No overhead. Full native speed.

## Feature Comparison

| Feature | MageBox | Warden | DDEV |
|---------|---------|--------|------|
| Native PHP/Nginx | ✅ | ❌ | ❌ |
| Auto SSL certificates | ✅ | ✅ | ✅ |
| Multi-domain support | ✅ | ✅ | ✅ |
| Database import/export | ✅ | ✅ | ✅ |
| Redis integration | ✅ | ✅ | ✅ |
| OpenSearch support | ✅ | ✅ | ✅ |
| Varnish support | ✅ | ✅ | ❌ |
| RabbitMQ support | ✅ | ✅ | ✅ |
| Email testing | ✅ | ✅ | ✅ |
| Project discovery | ✅ | ❌ | ❌ |
| Custom commands | ✅ | ❌ | ❌ |
| Self-updating | ✅ | ❌ | ✅ |
| Single binary | ✅ | ❌ | ❌ |

## When to Choose MageBox

### Choose MageBox if:

- **Performance is critical**: You want the fastest possible development experience
- **Multiple projects**: You work on several Magento sites with different PHP requirements
- **Limited resources**: Your machine has limited RAM or CPU
- **Quick iteration**: You make frequent code changes and need instant feedback
- **Local development**: You're developing on macOS, Linux, or Windows WSL2

### Consider alternatives if:
- **Exact production parity**: You need containers identical to production
- **Team standardization**: Your team requires identical Docker environments
- **CI/CD integration**: You need the same environment in CI pipelines

## Migration from Other Tools

### From Warden

```bash
# Stop Warden
warden env stop

# Initialize MageBox
magebox init myproject

# Update .magebox.yaml with your services
# Start MageBox
magebox start
```

### From DDEV

```bash
# Stop DDEV
ddev stop

# Initialize MageBox
magebox init myproject

# Start MageBox
magebox start
```

### From Valet+

```bash
# Unlink from Valet
valet unlink

# Initialize MageBox
magebox init myproject

# Start MageBox
magebox start
```

## Real-World Benefits

### Developer Experience

- **Faster feedback loops**: Code changes appear instantly
- **Less context switching**: Spend time coding, not waiting
- **Simpler debugging**: Native PHP means standard debugging tools work perfectly
- **Better IDE integration**: Xdebug, PHPStan, and other tools work without container networking

### Resource Efficiency

Running 3 Magento projects with Docker-based tools might use 8-12GB of RAM. With MageBox, the same projects use 1-2GB because PHP runs natively and shares resources efficiently.

### Reliability

Native services are simpler. Fewer moving parts means fewer things that can break. No Docker networking issues, no volume permission problems, no sync conflicts.

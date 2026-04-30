# MageBox

MageBox is a Magento 2 development environment using native PHP-FPM/Nginx with Docker for stateful services only. Built in Go with Cobra CLI framework.

## Build & Test

```bash
make build          # Build binary for current platform
make lint           # Run golangci-lint (required before committing)
make test           # Run tests (go test -v ./...)
make fmt            # Format code (go fmt + goimports)
make build-all      # Cross-compile for darwin/linux amd64/arm64
```

Single package test: `go test ./internal/config/... -v`

## Pre-commit requirements

Always run `make lint` and `make test` before committing Go code. Both must pass.

## Documentation

Documentation lives in `vitepress/`. The main files to keep in sync with code:

- `vitepress/reference/commands.md` — one section per non-hidden CLI command
- `vitepress/reference/config-options.md` — every field in `internal/config/types.go`
- `vitepress/guide/bootstrap.md` — reflects actual bootstrap behaviour per platform

**Whenever you add or change a feature, proactively check whether the docs need updating before committing.** Specifically:

- New CLI command or subcommand → add a section to `commands.md`
- New config field in `types.go` → add it to `config-options.md`
- Changed behaviour (flags, defaults, internal mechanics) → update the relevant guide or reference page
- Removed or renamed command/flag → remove or correct the docs entry

Do not wait to be asked. Treat outdated docs as a bug.

## Project structure

- `cmd/magebox/` - CLI commands (one file per command, uses Cobra)
- `internal/` - Private packages (config, platform, php, nginx, ssl, docker, dns, varnish, project, etc.)
- `lib/templates/` - Embedded config/Docker templates
- `vitepress/` - Documentation site (magebox.dev)
- `docs/` - Internal design docs
- `VERSION` - Single-line semver, drives auto-tag CI workflow
- `CHANGELOG.md` - Keep a Changelog format

## Release process

1. Update `VERSION` file with new semver
2. Add release entry to `CHANGELOG.md` (Keep a Changelog format, date as YYYY-MM-DD)
3. Commit and push to main
4. CI auto-tags from VERSION file, then builds/releases binaries and updates Homebrew tap

## Code conventions

- Go 1.24, CGO_ENABLED=0
- Return errors, don't panic
- Table-driven tests preferred
- Linters: errcheck, gosimple, govet, ineffassign, staticcheck, unused, gofmt, goimports, misspell
- Commit messages: imperative mood, reference PR numbers with `(#XX)`

## Vision

### MageBox is a development environment, not a toolbox

MageBox does one thing well: provide a fast, native Magento 2 development environment. PHP and Nginx run directly on your machine for maximum performance, Docker only for stateful services. One `.magebox.yaml` configures everything.

The original motivation: native PHP + Nginx so the same environment works identically on macOS and Linux — no Mutagen sync, no duplicated Docker images, multiple domain support by default. The goal has always been a fast and stable local dev environment that reduces manual operations to a bare minimum.

### Principles

**1. Native where possible, Docker where it makes sense**

PHP-FPM and Nginx run natively — no virtualization overhead, no bind mount issues, direct file access. Docker is only used for services that benefit from isolation and version management: databases, caches, search engines.

**2. Set and forget**

MageBox manages complexity so the developer doesn't have to. PHP versions, SSL certificates, DNS, port forwarding — everything is configured automatically. The same applies to tooling MageBox integrates with: the correct version is automatically selected based on the project configuration.

**3. Don't duplicate what already exists**

MageBox doesn't build its own version of functionality that existing tools already handle well. If magerun2 already supports database operations, MageBox integrates with it rather than rebuilding it. Features in core must provide real added value over what's already available — such as a progress bar for imports, or automatic version detection.

**4. Core is for everyone, specific is for you**

Functionality in MageBox core must be valuable to the broad Magento community. Workflow-specific tooling — things that fit how your team works but aren't universal — belongs in custom commands (`.magebox.yaml`) or is shared via the team server. The custom commands system and `.magebox.local.yaml` override mechanism are deliberately designed to enable this without bloating core.

**5. Pragmatic, not speculative**

MageBox adds support for tools and technologies when there is actual adoption in the Magento ecosystem. Potentially interesting but not yet widely adopted technologies are not preemptively supported.

**6. One config, the whole team**

Project configuration lives in `.magebox.yaml` in the repository. Personal preferences in `.magebox.local.yaml`. Team-wide settings, repositories, and assets via the team server. This means a new developer can get started with `magebox clone` + `magebox start` — no manual setup required.

**7. Magento-first, framework-open**

MageBox is built for Magento 2 and MageOS, but the architecture is open to other PHP frameworks in the future. The Magento-specific experience is not sacrificed for generality.

### What belongs in core

- Environment management: PHP, Nginx, Docker services, SSL, DNS
- Project configuration and lifecycle: init, start, stop, status
- Integrations that provide clear added value on top of existing tools
- Team functionality: clone, fetch, sync, team server
- Debugging and profiling: Xdebug, Blackfire, Tideways

### What doesn't belong in core

- Wrappers around existing tools without clear added value
- Workflow-specific automation that isn't universal
- Features for technologies without broad adoption in the ecosystem

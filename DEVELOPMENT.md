# MageBox Development Guide

This guide explains how to set up your development environment and contribute to MageBox.

## Prerequisites

### Go Installation

MageBox requires Go 1.21 or later.

```bash
# macOS
brew install go

# Ubuntu/Debian
sudo apt install golang-go

# Or download from https://go.dev/dl/
wget https://go.dev/dl/go1.23.4.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.23.4.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```

Verify installation:

```bash
go version
# go version go1.23.4 linux/amd64
```

## Getting Started

### Clone the Repository

```bash
git clone https://github.com/qoliber/magebox.git
cd magebox
```

### Install Dependencies

```bash
go mod tidy
```

### Build

```bash
# Build for current platform
go build -o magebox ./cmd/magebox

# Build for all platforms
GOOS=darwin GOARCH=amd64 go build -o magebox-darwin-amd64 ./cmd/magebox
GOOS=darwin GOARCH=arm64 go build -o magebox-darwin-arm64 ./cmd/magebox
GOOS=linux GOARCH=amd64 go build -o magebox-linux-amd64 ./cmd/magebox
```

### Run Tests

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test ./... -v

# Run tests for a specific package
go test ./internal/config/... -v

# Run tests with coverage
go test ./... -cover

# Generate coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Run the CLI

```bash
# Run directly
go run ./cmd/magebox --help

# Or build and run
./magebox --help
```

## Project Structure

```
magebox/
├── cmd/
│   └── magebox/
│       └── main.go              # CLI entry point, command definitions
│
├── internal/                    # Private packages (not importable externally)
│   ├── config/
│   │   ├── types.go             # Config structs and validation
│   │   ├── loader.go            # .magebox file loading and merging
│   │   ├── types_test.go
│   │   └── loader_test.go
│   │
│   ├── platform/
│   │   ├── platform.go          # OS detection, paths, install commands
│   │   └── platform_test.go
│   │
│   ├── php/
│   │   ├── detector.go          # PHP version detection
│   │   ├── pool.go              # PHP-FPM pool generation
│   │   ├── detector_test.go
│   │   └── pool_test.go
│   │
│   ├── nginx/
│   │   ├── vhost.go             # Nginx vhost generation
│   │   └── vhost_test.go
│   │
│   ├── ssl/
│   │   ├── mkcert.go            # SSL certificate management
│   │   └── mkcert_test.go
│   │
│   ├── docker/
│   │   ├── compose.go           # Docker Compose generation
│   │   └── compose_test.go
│   │
│   ├── dns/
│   │   ├── hosts.go             # /etc/hosts management
│   │   └── hosts_test.go
│   │
│   ├── varnish/                 # (TODO)
│   │   ├── vcl.go
│   │   └── vcl_test.go
│   │
│   └── project/
│       ├── lifecycle.go         # Start/stop orchestration
│       └── lifecycle_test.go
│
├── templates/                   # (reserved for embedded templates)
│
├── go.mod
├── go.sum
├── README.md
├── DEVELOPMENT.md
└── LICENSE
```

## Package Responsibilities

### `config`

Handles `.magebox` and `.magebox.local` file parsing:

- `types.go`: Defines `Config`, `Domain`, `Services`, `ServiceConfig` structs
- `loader.go`: Loads and merges config files, validates configuration

```go
// Load config from a path
cfg, err := config.LoadFromPath("/path/to/project")

// Load from current directory
cfg, err := config.LoadFromCurrentDir()
```

### `platform`

OS-specific paths and commands:

```go
p, _ := platform.Detect()

// Get paths
p.NginxConfigDir()      // /etc/nginx or /opt/homebrew/etc/nginx
p.PHPFPMBinary("8.2")   // /usr/sbin/php-fpm8.2
p.MageBoxDir()          // ~/.magebox

// Get install commands
p.PHPInstallCommand("8.3")  // brew install php@8.3 or apt install ...
```

### `php`

PHP version detection and FPM pool management:

```go
detector := php.NewDetector(platform)
versions := detector.DetectInstalled()  // ["8.1", "8.2", "8.3"]

poolGen := php.NewPoolGenerator(platform)
poolGen.Generate("myproject", "8.2", envVars)
```

### `nginx`

Nginx vhost configuration:

```go
vhostGen := nginx.NewVhostGenerator(platform, sslManager)
vhostGen.Generate(config, "/path/to/project")

controller := nginx.NewController(platform)
controller.Reload()
```

### `ssl`

SSL certificate management via mkcert:

```go
sslMgr := ssl.NewManager(platform)
sslMgr.EnsureCAInstalled()
cert, _ := sslMgr.GenerateCert("mystore.test")
```

### `docker`

Docker Compose file generation:

```go
composeGen := docker.NewComposeGenerator(platform)
composeGen.GenerateGlobalServices(configs)

controller := docker.NewDockerController(composeFilePath)
controller.Up()
controller.CreateDatabase("mysql80", "mystore")
```

### `dns`

/etc/hosts management:

```go
hostsMgr := dns.NewHostsManager(platform)
hostsMgr.AddDomains([]string{"mystore.test", "api.mystore.test"})
hostsMgr.RemoveDomains([]string{"oldproject.test"})
```

### `project`

High-level orchestration:

```go
mgr := project.NewManager(platform)

// Start a project
result, err := mgr.Start("/path/to/project")

// Stop a project
err := mgr.Stop("/path/to/project")

// Get status
status, err := mgr.Status("/path/to/project")
```

## Adding a New Feature

### 1. Create the package

```bash
mkdir -p internal/newfeature
touch internal/newfeature/newfeature.go
touch internal/newfeature/newfeature_test.go
```

### 2. Implement with tests

Always write tests alongside your implementation:

```go
// internal/newfeature/newfeature.go
package newfeature

type FeatureManager struct {
    // ...
}

func NewFeatureManager() *FeatureManager {
    return &FeatureManager{}
}

func (m *FeatureManager) DoSomething() error {
    // implementation
}
```

```go
// internal/newfeature/newfeature_test.go
package newfeature

import "testing"

func TestFeatureManager_DoSomething(t *testing.T) {
    m := NewFeatureManager()
    err := m.DoSomething()
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
}
```

### 3. Wire up to CLI

Add command in `cmd/magebox/main.go`:

```go
var newCmd = &cobra.Command{
    Use:   "newfeature",
    Short: "Description",
    RunE:  runNewFeature,
}

func runNewFeature(cmd *cobra.Command, args []string) error {
    p, _ := platform.Detect()
    mgr := newfeature.NewFeatureManager()
    return mgr.DoSomething()
}

func init() {
    rootCmd.AddCommand(newCmd)
}
```

## Code Style

### Formatting

```bash
# Format all code
go fmt ./...

# Or use goimports for import management
goimports -w .
```

### Linting

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
golangci-lint run
```

### Conventions

- Use meaningful variable names
- Keep functions small and focused
- Return errors, don't panic
- Write table-driven tests
- Use `t.TempDir()` for test files
- Document exported functions and types

## Testing Guidelines

### Table-Driven Tests

```go
func TestSomething(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {
            name:     "valid input",
            input:    "hello",
            expected: "HELLO",
        },
        {
            name:    "empty input",
            input:   "",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Transform(tt.input)
            if tt.wantErr {
                if err == nil {
                    t.Error("expected error")
                }
                return
            }
            if got != tt.expected {
                t.Errorf("got %v, want %v", got, tt.expected)
            }
        })
    }
}
```

### Test Helpers

```go
func setupTest(t *testing.T) (*Manager, string) {
    tmpDir := t.TempDir()
    p := &platform.Platform{
        Type:    platform.Linux,
        HomeDir: tmpDir,
    }
    return NewManager(p), tmpDir
}
```

## Debugging

### Enable verbose output

```go
// Add to your code temporarily
fmt.Printf("DEBUG: value = %+v\n", someValue)
```

### Use delve debugger

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug
dlv debug ./cmd/magebox -- start
```

## Release Process

1. Update version in `cmd/magebox/main.go`
2. Run all tests: `go test ./...`
3. Build binaries for all platforms
4. Create git tag: `git tag v1.0.0`
5. Push tag: `git push origin v1.0.0`

## Getting Help

- Open an issue on GitHub
- Check existing issues for similar problems
- Include Go version, OS, and full error output in bug reports

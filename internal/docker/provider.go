package docker

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Provider represents a Docker provider (Docker Desktop, Colima, OrbStack, etc.)
type Provider struct {
	Name       string
	SocketPath string
	IsRunning  bool
	IsActive   bool
}

// ProviderManager manages Docker providers
type ProviderManager struct {
	homeDir string
}

// NewProviderManager creates a new provider manager
func NewProviderManager() *ProviderManager {
	homeDir, _ := os.UserHomeDir()
	return &ProviderManager{homeDir: homeDir}
}

// GetProviders returns all detected Docker providers
func (m *ProviderManager) GetProviders() []Provider {
	if runtime.GOOS != "darwin" {
		// On Linux, just return the default docker
		return []Provider{
			{
				Name:       "docker",
				SocketPath: "/var/run/docker.sock",
				IsRunning:  m.isSocketResponding("/var/run/docker.sock"),
				IsActive:   true,
			},
		}
	}

	// macOS providers
	providers := []Provider{}
	activeSocket := m.GetActiveSocket()

	// Docker Desktop
	desktopSocket := "/var/run/docker.sock"
	if m.isDockerDesktopRunning() {
		providers = append(providers, Provider{
			Name:       "desktop",
			SocketPath: desktopSocket,
			IsRunning:  true,
			IsActive:   activeSocket == desktopSocket,
		})
	} else if m.socketExists(desktopSocket) {
		providers = append(providers, Provider{
			Name:       "desktop",
			SocketPath: desktopSocket,
			IsRunning:  false,
			IsActive:   false,
		})
	}

	// Colima
	colimaSocket := filepath.Join(m.homeDir, ".colima", "default", "docker.sock")
	if m.socketExists(colimaSocket) {
		providers = append(providers, Provider{
			Name:       "colima",
			SocketPath: colimaSocket,
			IsRunning:  m.isSocketResponding(colimaSocket),
			IsActive:   activeSocket == colimaSocket,
		})
	}

	// OrbStack
	orbstackSocket := filepath.Join(m.homeDir, ".orbstack", "run", "docker.sock")
	if m.socketExists(orbstackSocket) {
		providers = append(providers, Provider{
			Name:       "orbstack",
			SocketPath: orbstackSocket,
			IsRunning:  m.isSocketResponding(orbstackSocket),
			IsActive:   activeSocket == orbstackSocket,
		})
	}

	// Rancher Desktop
	rancherSocket := filepath.Join(m.homeDir, ".rd", "docker.sock")
	if m.socketExists(rancherSocket) {
		providers = append(providers, Provider{
			Name:       "rancher",
			SocketPath: rancherSocket,
			IsRunning:  m.isSocketResponding(rancherSocket),
			IsActive:   activeSocket == rancherSocket,
		})
	}

	// Lima (generic)
	limaSocket := filepath.Join(m.homeDir, ".lima", "default", "sock", "docker.sock")
	if m.socketExists(limaSocket) {
		providers = append(providers, Provider{
			Name:       "lima",
			SocketPath: limaSocket,
			IsRunning:  m.isSocketResponding(limaSocket),
			IsActive:   activeSocket == limaSocket,
		})
	}

	return providers
}

// GetActiveSocket returns the currently active Docker socket
func (m *ProviderManager) GetActiveSocket() string {
	// Check DOCKER_HOST environment variable first
	dockerHost := os.Getenv("DOCKER_HOST")
	if dockerHost != "" {
		// Parse unix:// prefix
		if strings.HasPrefix(dockerHost, "unix://") {
			return strings.TrimPrefix(dockerHost, "unix://")
		}
		return dockerHost
	}

	// Check Docker context
	cmd := exec.Command("docker", "context", "inspect", "--format", "{{.Endpoints.docker.Host}}")
	output, err := cmd.Output()
	if err == nil {
		host := strings.TrimSpace(string(output))
		if strings.HasPrefix(host, "unix://") {
			return strings.TrimPrefix(host, "unix://")
		}
		if host != "" {
			return host
		}
	}

	// Default socket
	return "/var/run/docker.sock"
}

// GetActiveProvider returns the currently active provider
func (m *ProviderManager) GetActiveProvider() *Provider {
	providers := m.GetProviders()
	for _, p := range providers {
		if p.IsActive {
			return &p
		}
	}

	// If no active provider found, check if any is running
	for _, p := range providers {
		if p.IsRunning {
			return &p
		}
	}

	return nil
}

// GetRunningProviders returns providers that are currently running
func (m *ProviderManager) GetRunningProviders() []Provider {
	var running []Provider
	for _, p := range m.GetProviders() {
		if p.IsRunning {
			running = append(running, p)
		}
	}
	return running
}

// GetProviderByName returns a provider by name
func (m *ProviderManager) GetProviderByName(name string) *Provider {
	for _, p := range m.GetProviders() {
		if p.Name == name {
			return &p
		}
	}
	return nil
}

// GetSocketForProvider returns the socket path for a provider name
func (m *ProviderManager) GetSocketForProvider(name string) string {
	switch name {
	case "desktop":
		return "/var/run/docker.sock"
	case "colima":
		return filepath.Join(m.homeDir, ".colima", "default", "docker.sock")
	case "orbstack":
		return filepath.Join(m.homeDir, ".orbstack", "run", "docker.sock")
	case "rancher":
		return filepath.Join(m.homeDir, ".rd", "docker.sock")
	case "lima":
		return filepath.Join(m.homeDir, ".lima", "default", "sock", "docker.sock")
	default:
		return "/var/run/docker.sock"
	}
}

// socketExists checks if a socket file exists
func (m *ProviderManager) socketExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	// Check if it's a socket
	return info.Mode()&os.ModeSocket != 0
}

// isSocketResponding checks if a Docker socket is responding
func (m *ProviderManager) isSocketResponding(socketPath string) bool {
	cmd := exec.Command("docker", "--host", fmt.Sprintf("unix://%s", socketPath), "info")
	err := cmd.Run()
	return err == nil
}

// isDockerDesktopRunning checks if Docker Desktop is running
func (m *ProviderManager) isDockerDesktopRunning() bool {
	if runtime.GOOS != "darwin" {
		return false
	}

	// Check if Docker Desktop app is running
	cmd := exec.Command("pgrep", "-f", "Docker Desktop")
	err := cmd.Run()
	if err == nil {
		return true
	}

	// Also check if com.docker.backend is running
	cmd = exec.Command("pgrep", "-f", "com.docker.backend")
	return cmd.Run() == nil
}

// FormatDockerHost formats a socket path as DOCKER_HOST value
func FormatDockerHost(socketPath string) string {
	if strings.HasPrefix(socketPath, "unix://") {
		return socketPath
	}
	return fmt.Sprintf("unix://%s", socketPath)
}

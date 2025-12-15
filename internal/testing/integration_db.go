package testing

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/qoliber/magebox/internal/config"
)

// IntegrationDBManager manages MySQL containers for integration testing
type IntegrationDBManager struct {
	manager *Manager
	config  *config.IntegrationTestConfig
}

// NewIntegrationDBManager creates a new integration database manager
func NewIntegrationDBManager(m *Manager, cfg *config.IntegrationTestConfig) *IntegrationDBManager {
	return &IntegrationDBManager{
		manager: m,
		config:  cfg,
	}
}

// GetContainerName returns the container name based on MySQL version
// Format: mysql-{version}-test (e.g., mysql-8.0-test)
func (m *IntegrationDBManager) GetContainerName(version string) string {
	// Sanitize version for container name
	v := strings.ReplaceAll(version, ".", "-")
	return fmt.Sprintf("mysql-%s-test", v)
}

// GetDefaultVersion returns the default MySQL version
func (m *IntegrationDBManager) GetDefaultVersion() string {
	return "8.0"
}

// GetTmpfsSize returns the tmpfs size from config or default
func (m *IntegrationDBManager) GetTmpfsSize() string {
	if m.config != nil && m.config.TmpfsSize != "" {
		return m.config.TmpfsSize
	}
	return "1g"
}

// GetDBPort returns the database port from config or default
func (m *IntegrationDBManager) GetDBPort() int {
	if m.config != nil && m.config.DBPort > 0 {
		return m.config.DBPort
	}
	return 33080 // Default integration test port
}

// GetDBName returns the database name from config or default
func (m *IntegrationDBManager) GetDBName() string {
	if m.config != nil && m.config.DBName != "" {
		return m.config.DBName
	}
	return "magento_integration_tests"
}

// GetDBUser returns the database user from config or default
func (m *IntegrationDBManager) GetDBUser() string {
	if m.config != nil && m.config.DBUser != "" {
		return m.config.DBUser
	}
	return "root"
}

// GetDBPass returns the database password from config or default
func (m *IntegrationDBManager) GetDBPass() string {
	if m.config != nil && m.config.DBPass != "" {
		return m.config.DBPass
	}
	return "root"
}

// IsContainerRunning checks if the test container is already running
func (m *IntegrationDBManager) IsContainerRunning(containerName string) bool {
	cmd := exec.Command("docker", "ps", "-q", "-f", fmt.Sprintf("name=^%s$", containerName))
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) != ""
}

// ContainerExists checks if the container exists (running or stopped)
func (m *IntegrationDBManager) ContainerExists(containerName string) bool {
	cmd := exec.Command("docker", "ps", "-aq", "-f", fmt.Sprintf("name=^%s$", containerName))
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) != ""
}

// StartContainer starts the MySQL test container with tmpfs if enabled
func (m *IntegrationDBManager) StartContainer(version string, useTmpfs bool) error {
	containerName := m.GetContainerName(version)

	// Check if container is already running
	if m.IsContainerRunning(containerName) {
		fmt.Printf("  ✓ Container %s is already running\n", containerName)
		return nil
	}

	// Check if container exists but is stopped
	if m.ContainerExists(containerName) {
		fmt.Printf("  → Starting existing container %s...\n", containerName)
		cmd := exec.Command("docker", "start", containerName)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to start container: %w", err)
		}
		return m.waitForMySQL(containerName)
	}

	// Create new container
	fmt.Printf("  → Creating MySQL %s test container", version)
	if useTmpfs {
		fmt.Printf(" (tmpfs: %s)", m.GetTmpfsSize())
	}
	fmt.Println("...")

	args := []string{
		"run", "-d",
		"--name", containerName,
		"-p", fmt.Sprintf("%d:3306", m.GetDBPort()),
		"-e", fmt.Sprintf("MYSQL_ROOT_PASSWORD=%s", m.GetDBPass()),
		"-e", fmt.Sprintf("MYSQL_DATABASE=%s", m.GetDBName()),
	}

	// Add tmpfs mount if enabled
	if useTmpfs {
		tmpfsSize := m.GetTmpfsSize()
		args = append(args, "--tmpfs", fmt.Sprintf("/var/lib/mysql:rw,size=%s", tmpfsSize))
	}

	// Add MySQL image
	args = append(args, fmt.Sprintf("mysql:%s", version))

	// Add MySQL server configuration for faster tests
	args = append(args,
		"--innodb-flush-log-at-trx-commit=0",
		"--innodb-flush-method=nosync",
		"--innodb-doublewrite=0",
		"--innodb-log-file-size=256M",
		"--innodb-buffer-pool-size=512M",
		"--max-connections=200",
		"--skip-log-bin",
	)

	cmd := exec.Command("docker", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create container: %w\n%s", err, string(output))
	}

	return m.waitForMySQL(containerName)
}

// waitForMySQL waits for MySQL to be ready to accept connections
func (m *IntegrationDBManager) waitForMySQL(containerName string) error {
	fmt.Print("  → Waiting for MySQL to be ready")

	maxAttempts := 30
	for i := 0; i < maxAttempts; i++ {
		cmd := exec.Command("docker", "exec", containerName,
			"mysqladmin", "ping", "-h", "localhost",
			"-u", m.GetDBUser(),
			fmt.Sprintf("-p%s", m.GetDBPass()),
			"--silent",
		)
		if err := cmd.Run(); err == nil {
			fmt.Println(" ✓")
			return nil
		}
		fmt.Print(".")
		time.Sleep(time.Second)
	}

	fmt.Println(" ✗")
	return fmt.Errorf("MySQL did not become ready in time")
}

// StopContainer stops and optionally removes the test container
func (m *IntegrationDBManager) StopContainer(version string, remove bool) error {
	containerName := m.GetContainerName(version)

	if !m.ContainerExists(containerName) {
		return nil
	}

	if m.IsContainerRunning(containerName) {
		fmt.Printf("  → Stopping container %s...\n", containerName)
		cmd := exec.Command("docker", "stop", containerName)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to stop container: %w", err)
		}
	}

	if remove {
		fmt.Printf("  → Removing container %s...\n", containerName)
		cmd := exec.Command("docker", "rm", containerName)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to remove container: %w", err)
		}
	}

	return nil
}

// GetConnectionInfo returns the connection information for the test database
func (m *IntegrationDBManager) GetConnectionInfo(version string) map[string]string {
	return map[string]string{
		"host":      "127.0.0.1",
		"port":      fmt.Sprintf("%d", m.GetDBPort()),
		"database":  m.GetDBName(),
		"user":      m.GetDBUser(),
		"password":  m.GetDBPass(),
		"container": m.GetContainerName(version),
	}
}

// Status returns the status of the test database container
func (m *IntegrationDBManager) Status(version string) string {
	containerName := m.GetContainerName(version)

	if !m.ContainerExists(containerName) {
		return "not created"
	}

	if m.IsContainerRunning(containerName) {
		return "running"
	}

	return "stopped"
}

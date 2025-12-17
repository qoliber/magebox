// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package remote

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Environment represents a remote server environment
type Environment struct {
	Name       string `yaml:"name"`
	User       string `yaml:"user"`
	Host       string `yaml:"host"`
	Port       int    `yaml:"port,omitempty"`
	SSHKeyPath string `yaml:"ssh_key,omitempty"`
	SSHCommand string `yaml:"ssh_command,omitempty"` // Custom SSH command for tunnels
}

// DefaultPort is the default SSH port
const DefaultPort = 22

// Validate checks if the environment configuration is valid
func (e *Environment) Validate() error {
	if e.Name == "" {
		return fmt.Errorf("environment name is required")
	}
	if e.User == "" {
		return fmt.Errorf("user is required")
	}
	if e.Host == "" {
		return fmt.Errorf("host is required")
	}
	if e.Port < 0 || e.Port > 65535 {
		return fmt.Errorf("port must be between 0 and 65535")
	}
	if e.SSHKeyPath != "" {
		if _, err := os.Stat(e.SSHKeyPath); os.IsNotExist(err) {
			return fmt.Errorf("SSH key file not found: %s", e.SSHKeyPath)
		}
	}
	return nil
}

// GetPort returns the port, defaulting to 22 if not set
func (e *Environment) GetPort() int {
	if e.Port == 0 {
		return DefaultPort
	}
	return e.Port
}

// BuildSSHCommand builds the SSH command for this environment
func (e *Environment) BuildSSHCommand(additionalArgs ...string) *exec.Cmd {
	// If custom SSH command is specified, use it
	if e.SSHCommand != "" {
		return exec.Command("sh", "-c", e.SSHCommand)
	}

	// Build standard SSH command
	args := []string{}

	// Add SSH key if specified
	if e.SSHKeyPath != "" {
		args = append(args, "-i", e.SSHKeyPath)
	}

	// Add port if non-standard
	if e.GetPort() != DefaultPort {
		args = append(args, "-p", strconv.Itoa(e.GetPort()))
	}

	// Add any additional args
	args = append(args, additionalArgs...)

	// Add user@host
	args = append(args, fmt.Sprintf("%s@%s", e.User, e.Host))

	return exec.Command("ssh", args...)
}

// GetConnectionString returns a human-readable connection string
func (e *Environment) GetConnectionString() string {
	var parts []string
	parts = append(parts, fmt.Sprintf("%s@%s", e.User, e.Host))

	if e.GetPort() != DefaultPort {
		parts[0] = fmt.Sprintf("%s:%d", parts[0], e.GetPort())
	}

	if e.SSHKeyPath != "" {
		parts = append(parts, fmt.Sprintf("(key: %s)", e.SSHKeyPath))
	}

	if e.SSHCommand != "" {
		parts = append(parts, "(custom command)")
	}

	return strings.Join(parts, " ")
}

// Manager handles environment operations
type Manager struct {
	environments []Environment
}

// NewManager creates a new environment manager
func NewManager(envs []Environment) *Manager {
	return &Manager{
		environments: envs,
	}
}

// List returns all environments
func (m *Manager) List() []Environment {
	return m.environments
}

// Get returns an environment by name
func (m *Manager) Get(name string) (*Environment, error) {
	for i := range m.environments {
		if m.environments[i].Name == name {
			return &m.environments[i], nil
		}
	}
	return nil, fmt.Errorf("environment '%s' not found", name)
}

// Add adds a new environment
func (m *Manager) Add(env Environment) error {
	// Check for duplicate
	for _, e := range m.environments {
		if e.Name == env.Name {
			return fmt.Errorf("environment '%s' already exists", env.Name)
		}
	}

	// Validate
	if err := env.Validate(); err != nil {
		return err
	}

	m.environments = append(m.environments, env)
	return nil
}

// Remove removes an environment by name
func (m *Manager) Remove(name string) error {
	for i, e := range m.environments {
		if e.Name == name {
			m.environments = append(m.environments[:i], m.environments[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("environment '%s' not found", name)
}

// Update updates an existing environment
func (m *Manager) Update(env Environment) error {
	for i, e := range m.environments {
		if e.Name == env.Name {
			if err := env.Validate(); err != nil {
				return err
			}
			m.environments[i] = env
			return nil
		}
	}
	return fmt.Errorf("environment '%s' not found", env.Name)
}

// GetEnvironments returns the current list of environments
func (m *Manager) GetEnvironments() []Environment {
	return m.environments
}

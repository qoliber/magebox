// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package lib

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Manager handles the magebox-lib repository operations
type Manager struct {
	paths   *Paths
	repoURL string
}

// NewManager creates a new lib manager
func NewManager(paths *Paths) *Manager {
	return &Manager{
		paths:   paths,
		repoURL: DefaultRepoURL,
	}
}

// NewManagerWithURL creates a new lib manager with a custom repo URL
func NewManagerWithURL(paths *Paths, repoURL string) *Manager {
	return &Manager{
		paths:   paths,
		repoURL: repoURL,
	}
}

// Status represents the current state of the lib
type Status struct {
	Installed       bool
	IsGitRepo       bool
	Version         string
	Branch          string
	CommitHash      string
	HasLocalChanges bool
	BehindRemote    int
	AheadOfRemote   int
	Error           error
}

// EnsureInstalled makes sure the lib is installed, cloning if necessary
func (m *Manager) EnsureInstalled() error {
	if m.paths.Exists() {
		return nil
	}
	return m.Clone()
}

// Clone clones the lib repository
func (m *Manager) Clone() error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(m.paths.LibDir), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Clone the repository
	cmd := exec.Command("git", "clone", "--depth", "1", m.repoURL, m.paths.LibDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone lib repository: %w", err)
	}

	return nil
}

// Update pulls the latest changes from the remote
func (m *Manager) Update() error {
	if !m.paths.Exists() {
		return m.Clone()
	}

	if !m.paths.IsGitRepo() {
		return fmt.Errorf("lib directory exists but is not a git repository")
	}

	// Fetch latest changes
	fetchCmd := exec.Command("git", "-C", m.paths.LibDir, "fetch", "origin")
	fetchCmd.Stdout = os.Stdout
	fetchCmd.Stderr = os.Stderr
	if err := fetchCmd.Run(); err != nil {
		return fmt.Errorf("failed to fetch updates: %w", err)
	}

	// Reset to origin/main (or origin/master)
	branch := m.getCurrentBranch()
	if branch == "" {
		branch = "main"
	}

	resetCmd := exec.Command("git", "-C", m.paths.LibDir, "reset", "--hard", "origin/"+branch)
	resetCmd.Stdout = os.Stdout
	resetCmd.Stderr = os.Stderr
	if err := resetCmd.Run(); err != nil {
		return fmt.Errorf("failed to update lib: %w", err)
	}

	return nil
}

// Reset discards all local changes and resets to upstream
func (m *Manager) Reset() error {
	if !m.paths.Exists() {
		return fmt.Errorf("lib is not installed")
	}

	if !m.paths.IsGitRepo() {
		// Not a git repo, remove and re-clone
		if err := os.RemoveAll(m.paths.LibDir); err != nil {
			return fmt.Errorf("failed to remove lib directory: %w", err)
		}
		return m.Clone()
	}

	// Fetch and hard reset
	fetchCmd := exec.Command("git", "-C", m.paths.LibDir, "fetch", "origin")
	if err := fetchCmd.Run(); err != nil {
		return fmt.Errorf("failed to fetch: %w", err)
	}

	branch := m.getCurrentBranch()
	if branch == "" {
		branch = "main"
	}

	resetCmd := exec.Command("git", "-C", m.paths.LibDir, "reset", "--hard", "origin/"+branch)
	resetCmd.Stdout = os.Stdout
	resetCmd.Stderr = os.Stderr
	if err := resetCmd.Run(); err != nil {
		return fmt.Errorf("failed to reset lib: %w", err)
	}

	// Clean untracked files
	cleanCmd := exec.Command("git", "-C", m.paths.LibDir, "clean", "-fd")
	_ = cleanCmd.Run()

	return nil
}

// GetStatus returns the current status of the lib
func (m *Manager) GetStatus() *Status {
	status := &Status{}

	if !m.paths.Exists() {
		return status
	}
	status.Installed = true

	// Get version from version.txt (works for both git and non-git installs)
	if data, err := os.ReadFile(m.paths.VersionFile); err == nil {
		status.Version = strings.TrimSpace(string(data))
	}

	if !m.paths.IsGitRepo() {
		return status
	}
	status.IsGitRepo = true

	// Get current branch
	status.Branch = m.getCurrentBranch()

	// Get commit hash
	hashCmd := exec.Command("git", "-C", m.paths.LibDir, "rev-parse", "--short", "HEAD")
	if output, err := hashCmd.Output(); err == nil {
		status.CommitHash = strings.TrimSpace(string(output))
	}

	// Check for local changes
	statusCmd := exec.Command("git", "-C", m.paths.LibDir, "status", "--porcelain")
	if output, err := statusCmd.Output(); err == nil {
		status.HasLocalChanges = len(strings.TrimSpace(string(output))) > 0
	}

	// Check ahead/behind (requires fetch first)
	_ = exec.Command("git", "-C", m.paths.LibDir, "fetch", "origin").Run()

	revListCmd := exec.Command("git", "-C", m.paths.LibDir, "rev-list", "--left-right", "--count", "HEAD...origin/"+status.Branch)
	if output, err := revListCmd.Output(); err == nil {
		parts := strings.Fields(strings.TrimSpace(string(output)))
		if len(parts) == 2 {
			_, _ = fmt.Sscanf(parts[0], "%d", &status.AheadOfRemote)
			_, _ = fmt.Sscanf(parts[1], "%d", &status.BehindRemote)
		}
	}

	return status
}

// GetVersion returns the current lib version
func (m *Manager) GetVersion() string {
	if data, err := os.ReadFile(m.paths.VersionFile); err == nil {
		return strings.TrimSpace(string(data))
	}
	return "unknown"
}

// GetPath returns the lib directory path
func (m *Manager) GetPath() string {
	return m.paths.LibDir
}

// ListInstallers returns a list of available installer names
func (m *Manager) ListInstallers() ([]string, error) {
	if !m.paths.Exists() {
		return nil, fmt.Errorf("lib is not installed")
	}

	entries, err := os.ReadDir(m.paths.InstallersDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read installers directory: %w", err)
	}

	var installers []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".yaml") {
			installers = append(installers, strings.TrimSuffix(name, ".yaml"))
		}
	}

	return installers, nil
}

func (m *Manager) getCurrentBranch() string {
	cmd := exec.Command("git", "-C", m.paths.LibDir, "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

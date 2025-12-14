// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package team

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ConfigPath returns the path to the teams config file
func ConfigPath(homeDir string) string {
	return filepath.Join(homeDir, ".magebox", "teams.yaml")
}

// LoadConfig loads the teams configuration from disk
func LoadConfig(homeDir string) (*TeamsConfig, error) {
	configPath := ConfigPath(homeDir)

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty config if file doesn't exist
			return &TeamsConfig{
				Teams: make(map[string]*Team),
			}, nil
		}
		return nil, fmt.Errorf("failed to read teams config: %w", err)
	}

	var config TeamsConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse teams config: %w", err)
	}

	if config.Teams == nil {
		config.Teams = make(map[string]*Team)
	}

	// Set team names from map keys
	for name, t := range config.Teams {
		t.Name = name
		if t.Projects == nil {
			t.Projects = make(map[string]*Project)
		}
	}

	return &config, nil
}

// SaveConfig saves the teams configuration to disk
func SaveConfig(homeDir string, config *TeamsConfig) error {
	configPath := ConfigPath(homeDir)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal teams config: %w", err)
	}

	// Add header comment
	header := "# MageBox Teams Configuration\n# Manage with: magebox team <command>\n\n"
	data = append([]byte(header), data...)

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write teams config: %w", err)
	}

	return nil
}

// GetTeam returns a team by name
func (c *TeamsConfig) GetTeam(name string) (*Team, error) {
	team, ok := c.Teams[name]
	if !ok {
		return nil, fmt.Errorf("team '%s' not found", name)
	}
	return team, nil
}

// AddTeam adds a new team to the config
func (c *TeamsConfig) AddTeam(team *Team) error {
	if team.Name == "" {
		return fmt.Errorf("team name is required")
	}
	if _, exists := c.Teams[team.Name]; exists {
		return fmt.Errorf("team '%s' already exists", team.Name)
	}
	if team.Projects == nil {
		team.Projects = make(map[string]*Project)
	}
	c.Teams[team.Name] = team
	return nil
}

// RemoveTeam removes a team from the config
func (c *TeamsConfig) RemoveTeam(name string) error {
	if _, exists := c.Teams[name]; !exists {
		return fmt.Errorf("team '%s' not found", name)
	}
	delete(c.Teams, name)
	return nil
}

// GetProject returns a project by team and project name
func (c *TeamsConfig) GetProject(teamName, projectName string) (*Team, *Project, error) {
	team, err := c.GetTeam(teamName)
	if err != nil {
		return nil, nil, err
	}

	project, ok := team.Projects[projectName]
	if !ok {
		return nil, nil, fmt.Errorf("project '%s' not found in team '%s'", projectName, teamName)
	}

	return team, project, nil
}

// FindProject searches for a project across all teams
// If only one team is configured, it searches in that team
// Otherwise, projectName should be in format "team/project"
func (c *TeamsConfig) FindProject(projectName string) (*Team, *Project, error) {
	// Check if format is "team/project"
	if strings.Contains(projectName, "/") {
		parts := strings.SplitN(projectName, "/", 2)
		return c.GetProject(parts[0], parts[1])
	}

	// If only one team, search in that team
	if len(c.Teams) == 1 {
		for _, team := range c.Teams {
			if project, ok := team.Projects[projectName]; ok {
				return team, project, nil
			}
		}
		// Get the only team name for error message
		var teamName string
		for name := range c.Teams {
			teamName = name
		}
		return nil, nil, fmt.Errorf("project '%s' not found in team '%s'", projectName, teamName)
	}

	// Multiple teams - need explicit team name
	if len(c.Teams) == 0 {
		return nil, nil, fmt.Errorf("no teams configured - run 'magebox team add <name>' first")
	}

	return nil, nil, fmt.Errorf("multiple teams configured - use format 'team/project'")
}

// GetToken returns the git token for a team from environment
func (t *Team) GetToken() string {
	// Try team-specific token first
	teamKey := strings.ToUpper(strings.ReplaceAll(t.Name, "-", "_"))
	if token := os.Getenv(fmt.Sprintf("MAGEBOX_%s_TOKEN", teamKey)); token != "" {
		return token
	}
	// Fall back to generic token
	return os.Getenv("MAGEBOX_GIT_TOKEN")
}

// GetAssetPassword returns the asset storage password from environment
func (t *Team) GetAssetPassword() string {
	teamKey := strings.ToUpper(strings.ReplaceAll(t.Name, "-", "_"))
	if pass := os.Getenv(fmt.Sprintf("MAGEBOX_%s_ASSET_PASS", teamKey)); pass != "" {
		return pass
	}
	return os.Getenv("MAGEBOX_ASSET_PASS")
}

// GetAssetKeyPath returns the asset storage SSH key path from environment
func (t *Team) GetAssetKeyPath() string {
	teamKey := strings.ToUpper(strings.ReplaceAll(t.Name, "-", "_"))
	if key := os.Getenv(fmt.Sprintf("MAGEBOX_%s_ASSET_KEY", teamKey)); key != "" {
		return key
	}
	if key := os.Getenv("MAGEBOX_ASSET_KEY"); key != "" {
		return key
	}
	// Default to ~/.ssh/id_rsa
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ssh", "id_rsa")
}

// GetCloneURL returns the clone URL for a project based on auth method
func (t *Team) GetCloneURL(project *Project) string {
	switch t.Repositories.Provider {
	case ProviderGitHub:
		if t.Repositories.Auth == AuthSSH {
			return fmt.Sprintf("git@github.com:%s.git", project.Repo)
		}
		return fmt.Sprintf("https://github.com/%s.git", project.Repo)
	case ProviderGitLab:
		if t.Repositories.Auth == AuthSSH {
			return fmt.Sprintf("git@gitlab.com:%s.git", project.Repo)
		}
		return fmt.Sprintf("https://gitlab.com/%s.git", project.Repo)
	case ProviderBitbucket:
		if t.Repositories.Auth == AuthSSH {
			return fmt.Sprintf("git@bitbucket.org:%s.git", project.Repo)
		}
		return fmt.Sprintf("https://bitbucket.org/%s.git", project.Repo)
	}
	return ""
}

// GetDefaultPort returns the default port for the asset provider
func (a *AssetConfig) GetDefaultPort() int {
	if a.Port > 0 {
		return a.Port
	}
	switch a.Provider {
	case AssetSFTP:
		return 22
	case AssetFTP:
		return 21
	}
	return 22
}

// Validate validates the team configuration
func (t *Team) Validate() error {
	if t.Name == "" {
		return fmt.Errorf("team name is required")
	}
	if t.Repositories.Provider == "" {
		return fmt.Errorf("repository provider is required")
	}
	if t.Repositories.Organization == "" {
		return fmt.Errorf("repository organization is required")
	}
	return nil
}

// ValidateProject validates a project configuration
func (p *Project) Validate() error {
	if p.Repo == "" {
		return fmt.Errorf("project repo is required")
	}
	return nil
}

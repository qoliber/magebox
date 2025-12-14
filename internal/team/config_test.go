// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package team

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()

	config, err := LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if config.Teams == nil {
		t.Error("Teams map should not be nil")
	}

	if len(config.Teams) != 0 {
		t.Errorf("Expected 0 teams, got %d", len(config.Teams))
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a config
	config := &TeamsConfig{
		Teams: map[string]*Team{
			"testteam": {
				Name: "testteam",
				Repositories: RepositoryConfig{
					Provider:     ProviderGitHub,
					Organization: "testorg",
					Auth:         AuthSSH,
				},
				Assets: AssetConfig{
					Provider: AssetSFTP,
					Host:     "backup.example.com",
					Port:     22,
					Path:     "/backups",
					Username: "deploy",
				},
				Projects: map[string]*Project{
					"myproject": {
						Repo:   "testorg/myproject",
						Branch: "main",
						DB:     "myproject/db.sql.gz",
						Media:  "myproject/media.tar.gz",
					},
				},
			},
		},
	}

	// Save config
	if err := SaveConfig(tmpDir, config); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Verify file exists
	configPath := ConfigPath(tmpDir)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Load config
	loaded, err := LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify team
	team, ok := loaded.Teams["testteam"]
	if !ok {
		t.Fatal("Team 'testteam' not found")
	}

	if team.Repositories.Provider != ProviderGitHub {
		t.Errorf("Expected provider github, got %s", team.Repositories.Provider)
	}

	if team.Repositories.Organization != "testorg" {
		t.Errorf("Expected organization testorg, got %s", team.Repositories.Organization)
	}

	if team.Assets.Host != "backup.example.com" {
		t.Errorf("Expected host backup.example.com, got %s", team.Assets.Host)
	}

	// Verify project
	project, ok := team.Projects["myproject"]
	if !ok {
		t.Fatal("Project 'myproject' not found")
	}

	if project.Repo != "testorg/myproject" {
		t.Errorf("Expected repo testorg/myproject, got %s", project.Repo)
	}
}

func TestConfigPath(t *testing.T) {
	path := ConfigPath("/home/user")
	expected := filepath.Join("/home/user", ".magebox", "teams.yaml")
	if path != expected {
		t.Errorf("Expected %s, got %s", expected, path)
	}
}

func TestAddTeam(t *testing.T) {
	config := &TeamsConfig{
		Teams: make(map[string]*Team),
	}

	team := &Team{
		Name: "newteam",
		Repositories: RepositoryConfig{
			Provider:     ProviderGitLab,
			Organization: "myorg",
			Auth:         AuthToken,
		},
	}

	if err := config.AddTeam(team); err != nil {
		t.Fatalf("AddTeam failed: %v", err)
	}

	if _, ok := config.Teams["newteam"]; !ok {
		t.Error("Team was not added")
	}

	// Adding same team again should fail
	if err := config.AddTeam(team); err == nil {
		t.Error("Expected error when adding duplicate team")
	}
}

func TestRemoveTeam(t *testing.T) {
	config := &TeamsConfig{
		Teams: map[string]*Team{
			"existingteam": {
				Name: "existingteam",
			},
		},
	}

	if err := config.RemoveTeam("existingteam"); err != nil {
		t.Fatalf("RemoveTeam failed: %v", err)
	}

	if _, ok := config.Teams["existingteam"]; ok {
		t.Error("Team was not removed")
	}

	// Removing non-existent team should fail
	if err := config.RemoveTeam("nonexistent"); err == nil {
		t.Error("Expected error when removing non-existent team")
	}
}

func TestGetTeam(t *testing.T) {
	config := &TeamsConfig{
		Teams: map[string]*Team{
			"myteam": {
				Name: "myteam",
			},
		},
	}

	team, err := config.GetTeam("myteam")
	if err != nil {
		t.Fatalf("GetTeam failed: %v", err)
	}

	if team.Name != "myteam" {
		t.Errorf("Expected team name myteam, got %s", team.Name)
	}

	// Getting non-existent team should fail
	_, err = config.GetTeam("nonexistent")
	if err == nil {
		t.Error("Expected error when getting non-existent team")
	}
}

func TestGetProject(t *testing.T) {
	config := &TeamsConfig{
		Teams: map[string]*Team{
			"myteam": {
				Name: "myteam",
				Projects: map[string]*Project{
					"myproject": {
						Repo: "org/myproject",
					},
				},
			},
		},
	}

	team, project, err := config.GetProject("myteam", "myproject")
	if err != nil {
		t.Fatalf("GetProject failed: %v", err)
	}

	if team.Name != "myteam" {
		t.Errorf("Expected team name myteam, got %s", team.Name)
	}

	if project.Repo != "org/myproject" {
		t.Errorf("Expected repo org/myproject, got %s", project.Repo)
	}

	// Getting non-existent project should fail
	_, _, err = config.GetProject("myteam", "nonexistent")
	if err == nil {
		t.Error("Expected error when getting non-existent project")
	}
}

func TestFindProject(t *testing.T) {
	tests := []struct {
		name        string
		config      *TeamsConfig
		projectRef  string
		expectError bool
	}{
		{
			name: "single team, simple project name",
			config: &TeamsConfig{
				Teams: map[string]*Team{
					"onlyteam": {
						Name: "onlyteam",
						Projects: map[string]*Project{
							"myproject": {Repo: "org/myproject"},
						},
					},
				},
			},
			projectRef:  "myproject",
			expectError: false,
		},
		{
			name: "explicit team/project format",
			config: &TeamsConfig{
				Teams: map[string]*Team{
					"team1": {
						Name: "team1",
						Projects: map[string]*Project{
							"project1": {Repo: "org1/project1"},
						},
					},
					"team2": {
						Name: "team2",
						Projects: map[string]*Project{
							"project2": {Repo: "org2/project2"},
						},
					},
				},
			},
			projectRef:  "team1/project1",
			expectError: false,
		},
		{
			name: "multiple teams, simple name requires explicit team",
			config: &TeamsConfig{
				Teams: map[string]*Team{
					"team1": {Name: "team1", Projects: map[string]*Project{}},
					"team2": {Name: "team2", Projects: map[string]*Project{}},
				},
			},
			projectRef:  "myproject",
			expectError: true,
		},
		{
			name: "no teams configured",
			config: &TeamsConfig{
				Teams: map[string]*Team{},
			},
			projectRef:  "myproject",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := tt.config.FindProject(tt.projectRef)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestGetCloneURL(t *testing.T) {
	tests := []struct {
		name     string
		team     *Team
		project  *Project
		expected string
	}{
		{
			name: "GitHub SSH",
			team: &Team{
				Repositories: RepositoryConfig{
					Provider: ProviderGitHub,
					Auth:     AuthSSH,
				},
			},
			project:  &Project{Repo: "org/repo"},
			expected: "git@github.com:org/repo.git",
		},
		{
			name: "GitHub HTTPS",
			team: &Team{
				Repositories: RepositoryConfig{
					Provider: ProviderGitHub,
					Auth:     AuthToken,
				},
			},
			project:  &Project{Repo: "org/repo"},
			expected: "https://github.com/org/repo.git",
		},
		{
			name: "GitLab SSH",
			team: &Team{
				Repositories: RepositoryConfig{
					Provider: ProviderGitLab,
					Auth:     AuthSSH,
				},
			},
			project:  &Project{Repo: "org/repo"},
			expected: "git@gitlab.com:org/repo.git",
		},
		{
			name: "Bitbucket HTTPS",
			team: &Team{
				Repositories: RepositoryConfig{
					Provider: ProviderBitbucket,
					Auth:     AuthToken,
				},
			},
			project:  &Project{Repo: "org/repo"},
			expected: "https://bitbucket.org/org/repo.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := tt.team.GetCloneURL(tt.project)
			if url != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, url)
			}
		})
	}
}

func TestGetDefaultPort(t *testing.T) {
	tests := []struct {
		name     string
		config   AssetConfig
		expected int
	}{
		{
			name:     "SFTP default",
			config:   AssetConfig{Provider: AssetSFTP},
			expected: 22,
		},
		{
			name:     "FTP default",
			config:   AssetConfig{Provider: AssetFTP},
			expected: 21,
		},
		{
			name:     "Custom port",
			config:   AssetConfig{Provider: AssetSFTP, Port: 2222},
			expected: 2222,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port := tt.config.GetDefaultPort()
			if port != tt.expected {
				t.Errorf("Expected port %d, got %d", tt.expected, port)
			}
		})
	}
}

func TestTeamValidate(t *testing.T) {
	tests := []struct {
		name        string
		team        *Team
		expectError bool
	}{
		{
			name: "valid team",
			team: &Team{
				Name: "myteam",
				Repositories: RepositoryConfig{
					Provider:     ProviderGitHub,
					Organization: "myorg",
				},
			},
			expectError: false,
		},
		{
			name: "missing name",
			team: &Team{
				Repositories: RepositoryConfig{
					Provider:     ProviderGitHub,
					Organization: "myorg",
				},
			},
			expectError: true,
		},
		{
			name: "missing provider",
			team: &Team{
				Name: "myteam",
				Repositories: RepositoryConfig{
					Organization: "myorg",
				},
			},
			expectError: true,
		},
		{
			name: "missing organization",
			team: &Team{
				Name: "myteam",
				Repositories: RepositoryConfig{
					Provider: ProviderGitHub,
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.team.Validate()
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestProjectValidate(t *testing.T) {
	tests := []struct {
		name        string
		project     *Project
		expectError bool
	}{
		{
			name:        "valid project",
			project:     &Project{Repo: "org/repo"},
			expectError: false,
		},
		{
			name:        "missing repo",
			project:     &Project{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.project.Validate()
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

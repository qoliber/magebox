// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package team

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMatchFilter(t *testing.T) {
	tests := []struct {
		name     string
		repoName string
		filter   string
		expected bool
	}{
		{
			name:     "empty filter matches all",
			repoName: "myrepo",
			filter:   "",
			expected: true,
		},
		{
			name:     "exact match",
			repoName: "myrepo",
			filter:   "myrepo",
			expected: true,
		},
		{
			name:     "exact match case insensitive",
			repoName: "MyRepo",
			filter:   "myrepo",
			expected: true,
		},
		{
			name:     "prefix wildcard",
			repoName: "magento2-module",
			filter:   "*module",
			expected: true,
		},
		{
			name:     "suffix wildcard",
			repoName: "magento2-module",
			filter:   "magento*",
			expected: true,
		},
		{
			name:     "contains wildcard",
			repoName: "my-magento-module",
			filter:   "*magento*",
			expected: true,
		},
		{
			name:     "no match",
			repoName: "other-project",
			filter:   "magento*",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchFilter(tt.repoName, tt.filter)
			if result != tt.expected {
				t.Errorf("matchFilter(%q, %q) = %v, want %v",
					tt.repoName, tt.filter, result, tt.expected)
			}
		})
	}
}

func TestGetProjectPath(t *testing.T) {
	cwd, _ := os.Getwd()

	tests := []struct {
		name        string
		destPath    string
		projectName string
		expected    string
	}{
		{
			name:        "custom path",
			destPath:    "/custom/path",
			projectName: "myproject",
			expected:    "/custom/path",
		},
		{
			name:        "default path",
			destPath:    "",
			projectName: "myproject",
			expected:    filepath.Join(cwd, "myproject"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetProjectPath(tt.destPath, tt.projectName)
			if result != tt.expected {
				t.Errorf("GetProjectPath(%q, %q) = %q, want %q",
					tt.destPath, tt.projectName, result, tt.expected)
			}
		})
	}
}

func TestNewRepositoryClient(t *testing.T) {
	team := &Team{
		Name: "testteam",
		Repositories: RepositoryConfig{
			Provider:     ProviderGitHub,
			Organization: "testorg",
			Auth:         AuthSSH,
		},
	}

	client := NewRepositoryClient(team)
	if client == nil {
		t.Fatal("NewRepositoryClient returned nil")
	}
	if client.team != team {
		t.Error("RepositoryClient team reference mismatch")
	}
}

// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package team

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// RepositoryClient handles git repository operations
type RepositoryClient struct {
	team *Team
}

// NewRepositoryClient creates a new repository client for a team
func NewRepositoryClient(team *Team) *RepositoryClient {
	return &RepositoryClient{team: team}
}

// ListRepositories lists all repositories in the organization/namespace
func (r *RepositoryClient) ListRepositories(filter string) ([]Repository, error) {
	switch r.team.Repositories.Provider {
	case ProviderGitHub:
		return r.listGitHubRepos(filter)
	case ProviderGitLab:
		return r.listGitLabRepos(filter)
	case ProviderBitbucket:
		return r.listBitbucketRepos(filter)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", r.team.Repositories.Provider)
	}
}

// Clone clones a repository to the specified path
func (r *RepositoryClient) Clone(project *Project, destPath string, progress func(string)) error {
	cloneURL := r.team.GetCloneURL(project)
	branch := project.Branch
	if branch == "" {
		// Detect default branch from remote
		branch = r.detectDefaultBranch(cloneURL)
		if branch == "" {
			branch = "main" // Final fallback
		}
	}

	if progress != nil {
		progress(fmt.Sprintf("Cloning %s (branch: %s)...", project.Repo, branch))
	}

	// Check if destination exists
	if _, err := os.Stat(destPath); err == nil {
		return fmt.Errorf("destination path already exists: %s", destPath)
	}

	// Build clone command
	args := []string{"clone", "--progress", "--branch", branch, cloneURL, destPath}

	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	return nil
}

// DetectDefaultBranch detects the default branch from a remote repository (exported)
func DetectDefaultBranch(cloneURL string) string {
	return detectDefaultBranchFromURL(cloneURL)
}

// detectDefaultBranch detects the default branch from a remote repository
func (r *RepositoryClient) detectDefaultBranch(cloneURL string) string {
	return detectDefaultBranchFromURL(cloneURL)
}

// detectDefaultBranchFromURL is the implementation
func detectDefaultBranchFromURL(cloneURL string) string {
	// Use git ls-remote --symref to detect the default branch
	cmd := exec.Command("git", "ls-remote", "--symref", cloneURL, "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	// Parse output: "ref: refs/heads/master\tHEAD"
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "ref:") {
			// Line format: "ref: refs/heads/master\tHEAD"
			parts := strings.Fields(line)
			if len(parts) >= 2 && strings.HasPrefix(parts[1], "refs/heads/") {
				return strings.TrimPrefix(parts[1], "refs/heads/")
			}
		}
	}

	return ""
}

// Pull pulls latest changes in a repository
func (r *RepositoryClient) Pull(repoPath string, progress func(string)) error {
	if progress != nil {
		progress("Pulling latest changes...")
	}

	cmd := exec.Command("git", "pull", "--progress")
	cmd.Dir = repoPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to pull: %w", err)
	}

	return nil
}

// listGitHubRepos lists repositories from GitHub
func (r *RepositoryClient) listGitHubRepos(filter string) ([]Repository, error) {
	org := r.team.Repositories.Organization
	token := r.team.GetToken()

	url := fmt.Sprintf("https://api.github.com/orgs/%s/repos?per_page=100", org)

	// Try user repos if org fails
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repositories: %w", err)
	}
	defer resp.Body.Close()

	// If org endpoint fails, try user endpoint
	if resp.StatusCode == 404 {
		url = fmt.Sprintf("https://api.github.com/users/%s/repos?per_page=100", org)
		req, _ = http.NewRequest("GET", url, nil)
		req.Header.Set("Accept", "application/vnd.github.v3+json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, err = client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch repositories: %w", err)
		}
		defer resp.Body.Close()
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %s - %s", resp.Status, string(body))
	}

	var ghRepos []struct {
		Name          string `json:"name"`
		FullName      string `json:"full_name"`
		Description   string `json:"description"`
		CloneURL      string `json:"clone_url"`
		SSHURL        string `json:"ssh_url"`
		Private       bool   `json:"private"`
		DefaultBranch string `json:"default_branch"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ghRepos); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub response: %w", err)
	}

	var repos []Repository
	for _, gr := range ghRepos {
		// Apply filter
		if filter != "" && !matchFilter(gr.Name, filter) {
			continue
		}
		repos = append(repos, Repository{
			Name:          gr.Name,
			FullName:      gr.FullName,
			Description:   gr.Description,
			CloneURL:      gr.CloneURL,
			SSHURL:        gr.SSHURL,
			Private:       gr.Private,
			DefaultBranch: gr.DefaultBranch,
		})
	}

	return repos, nil
}

// listGitLabRepos lists repositories from GitLab (cloud or self-hosted)
func (r *RepositoryClient) listGitLabRepos(filter string) ([]Repository, error) {
	org := r.team.Repositories.Organization
	token := r.team.GetToken()

	// Use custom URL for self-hosted GitLab, or default to gitlab.com
	baseURL := "https://gitlab.com"
	if r.team.Repositories.URL != "" {
		baseURL = strings.TrimSuffix(r.team.Repositories.URL, "/")
	}

	// GitLab uses groups or users
	url := fmt.Sprintf("%s/api/v4/groups/%s/projects?per_page=100", baseURL, org)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if token != "" {
		req.Header.Set("PRIVATE-TOKEN", token)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repositories: %w", err)
	}
	defer resp.Body.Close()

	// If group endpoint fails, try user endpoint
	if resp.StatusCode == 404 {
		url = fmt.Sprintf("%s/api/v4/users/%s/projects?per_page=100", baseURL, org)
		req, _ = http.NewRequest("GET", url, nil)
		if token != "" {
			req.Header.Set("PRIVATE-TOKEN", token)
		}
		resp, err = client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch repositories: %w", err)
		}
		defer resp.Body.Close()
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitLab API error: %s - %s", resp.Status, string(body))
	}

	var glRepos []struct {
		Name              string `json:"name"`
		PathWithNamespace string `json:"path_with_namespace"`
		Description       string `json:"description"`
		HTTPURLToRepo     string `json:"http_url_to_repo"`
		SSHURLToRepo      string `json:"ssh_url_to_repo"`
		Visibility        string `json:"visibility"`
		DefaultBranch     string `json:"default_branch"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&glRepos); err != nil {
		return nil, fmt.Errorf("failed to parse GitLab response: %w", err)
	}

	var repos []Repository
	for _, gr := range glRepos {
		if filter != "" && !matchFilter(gr.Name, filter) {
			continue
		}
		repos = append(repos, Repository{
			Name:          gr.Name,
			FullName:      gr.PathWithNamespace,
			Description:   gr.Description,
			CloneURL:      gr.HTTPURLToRepo,
			SSHURL:        gr.SSHURLToRepo,
			Private:       gr.Visibility == "private",
			DefaultBranch: gr.DefaultBranch,
		})
	}

	return repos, nil
}

// listBitbucketRepos lists repositories from Bitbucket (cloud or self-hosted)
func (r *RepositoryClient) listBitbucketRepos(filter string) ([]Repository, error) {
	org := r.team.Repositories.Organization
	token := r.team.GetToken()

	// Use custom URL for self-hosted Bitbucket Server, or default to Bitbucket Cloud
	var url string
	var isServer bool
	if r.team.Repositories.URL != "" {
		// Self-hosted Bitbucket Server uses different API
		baseURL := strings.TrimSuffix(r.team.Repositories.URL, "/")
		url = fmt.Sprintf("%s/rest/api/1.0/projects/%s/repos?limit=100", baseURL, org)
		isServer = true
	} else {
		// Bitbucket Cloud
		url = fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%s?pagelen=100", org)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repositories: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return nil, fmt.Errorf("authentication required - set MAGEBOX_%s_TOKEN environment variable",
			strings.ToUpper(strings.ReplaceAll(r.team.Name, "-", "_")))
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("bitbucket API error: %s - %s", resp.Status, string(body))
	}

	// Handle self-hosted Bitbucket Server response
	if isServer {
		return r.parseBitbucketServerResponse(resp.Body, filter)
	}

	var bbResponse struct {
		Values []struct {
			Name        string `json:"name"`
			FullName    string `json:"full_name"`
			Description string `json:"description"`
			IsPrivate   bool   `json:"is_private"`
			MainBranch  struct {
				Name string `json:"name"`
			} `json:"mainbranch"`
			Links struct {
				Clone []struct {
					Name string `json:"name"`
					Href string `json:"href"`
				} `json:"clone"`
			} `json:"links"`
		} `json:"values"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&bbResponse); err != nil {
		return nil, fmt.Errorf("failed to parse Bitbucket response: %w", err)
	}

	var repos []Repository
	for _, br := range bbResponse.Values {
		if filter != "" && !matchFilter(br.Name, filter) {
			continue
		}

		var cloneURL, sshURL string
		for _, link := range br.Links.Clone {
			if link.Name == "https" {
				cloneURL = link.Href
			} else if link.Name == "ssh" {
				sshURL = link.Href
			}
		}

		repos = append(repos, Repository{
			Name:          br.Name,
			FullName:      br.FullName,
			Description:   br.Description,
			CloneURL:      cloneURL,
			SSHURL:        sshURL,
			Private:       br.IsPrivate,
			DefaultBranch: br.MainBranch.Name,
		})
	}

	return repos, nil
}

// parseBitbucketServerResponse parses the response from self-hosted Bitbucket Server
func (r *RepositoryClient) parseBitbucketServerResponse(body io.Reader, filter string) ([]Repository, error) {
	var serverResponse struct {
		Values []struct {
			Slug        string `json:"slug"`
			Name        string `json:"name"`
			Description string `json:"description"`
			Public      bool   `json:"public"`
			Links       struct {
				Clone []struct {
					Name string `json:"name"`
					Href string `json:"href"`
				} `json:"clone"`
			} `json:"links"`
		} `json:"values"`
	}

	if err := json.NewDecoder(body).Decode(&serverResponse); err != nil {
		return nil, fmt.Errorf("failed to parse Bitbucket Server response: %w", err)
	}

	var repos []Repository
	org := r.team.Repositories.Organization
	for _, br := range serverResponse.Values {
		if filter != "" && !matchFilter(br.Name, filter) {
			continue
		}

		var cloneURL, sshURL string
		for _, link := range br.Links.Clone {
			if link.Name == "http" || link.Name == "https" {
				cloneURL = link.Href
			} else if link.Name == "ssh" {
				sshURL = link.Href
			}
		}

		repos = append(repos, Repository{
			Name:          br.Slug,
			FullName:      fmt.Sprintf("%s/%s", org, br.Slug),
			Description:   br.Description,
			CloneURL:      cloneURL,
			SSHURL:        sshURL,
			Private:       !br.Public,
			DefaultBranch: "master", // Bitbucket Server doesn't return default branch in list
		})
	}

	return repos, nil
}

// matchFilter checks if a name matches a glob-like filter
func matchFilter(name, filter string) bool {
	// Simple glob matching with * wildcard
	if filter == "" {
		return true
	}

	// Convert glob to simple matching
	filter = strings.ToLower(filter)
	name = strings.ToLower(name)

	if strings.HasPrefix(filter, "*") && strings.HasSuffix(filter, "*") {
		// *pattern* - contains
		pattern := strings.Trim(filter, "*")
		return strings.Contains(name, pattern)
	} else if strings.HasPrefix(filter, "*") {
		// *pattern - ends with
		pattern := strings.TrimPrefix(filter, "*")
		return strings.HasSuffix(name, pattern)
	} else if strings.HasSuffix(filter, "*") {
		// pattern* - starts with
		pattern := strings.TrimSuffix(filter, "*")
		return strings.HasPrefix(name, pattern)
	}

	// Exact match
	return name == filter
}

// GetProjectPath returns the local path for a project
func GetProjectPath(destPath, projectName string) string {
	if destPath != "" {
		return destPath
	}
	// Default to current directory + project name
	cwd, _ := os.Getwd()
	return filepath.Join(cwd, projectName)
}

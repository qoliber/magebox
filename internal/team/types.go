// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package team

// RepositoryProvider represents a git hosting provider
type RepositoryProvider string

const (
	ProviderGitHub    RepositoryProvider = "github"
	ProviderGitLab    RepositoryProvider = "gitlab"
	ProviderBitbucket RepositoryProvider = "bitbucket"
)

// AuthMethod represents authentication method for repositories
type AuthMethod string

const (
	AuthSSH   AuthMethod = "ssh"
	AuthHTTPS AuthMethod = "https"
	AuthToken AuthMethod = "token"
)

// AssetProvider represents asset storage provider
type AssetProvider string

const (
	AssetSFTP AssetProvider = "sftp"
	AssetFTP  AssetProvider = "ftp"
)

// Team represents a team configuration
type Team struct {
	Name         string              `yaml:"-"`
	Repositories RepositoryConfig    `yaml:"repositories"`
	Assets       AssetConfig         `yaml:"assets"`
	Projects     map[string]*Project `yaml:"projects,omitempty"`
}

// RepositoryConfig holds git repository provider configuration
type RepositoryConfig struct {
	Provider     RepositoryProvider `yaml:"provider"`
	Organization string             `yaml:"organization"`
	Auth         AuthMethod         `yaml:"auth"`
	URL          string             `yaml:"url,omitempty"` // Custom URL for self-hosted instances (e.g., https://gitlab.mycompany.com)
	// Token is read from environment variable MAGEBOX_{TEAM}_TOKEN or MAGEBOX_GIT_TOKEN
}

// AssetConfig holds asset storage configuration
type AssetConfig struct {
	Provider AssetProvider `yaml:"provider"`
	Host     string        `yaml:"host"`
	Port     int           `yaml:"port,omitempty"`
	Path     string        `yaml:"path"`
	Username string        `yaml:"username"`
	// Password/key read from MAGEBOX_{TEAM}_ASSET_PASS or MAGEBOX_{TEAM}_ASSET_KEY
}

// Project represents a project within a team
type Project struct {
	Repo      string   `yaml:"repo"`
	Branch    string   `yaml:"branch,omitempty"`
	PHP       string   `yaml:"php,omitempty"`
	DB        string   `yaml:"db,omitempty"`
	Media     string   `yaml:"media,omitempty"`
	PostFetch []string `yaml:"post_fetch,omitempty"`
}

// TeamsConfig holds all team configurations
type TeamsConfig struct {
	Teams map[string]*Team `yaml:"teams"`
}

// Repository represents a remote repository
type Repository struct {
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	Description   string `json:"description"`
	CloneURL      string `json:"clone_url"`
	SSHURL        string `json:"ssh_url"`
	Private       bool   `json:"private"`
	DefaultBranch string `json:"default_branch"`
}

// DownloadProgress tracks download progress
type DownloadProgress struct {
	Filename   string
	TotalBytes int64
	Downloaded int64
	Speed      float64 // bytes per second
	Percentage float64
	ETA        string
}

// FetchOptions holds options for the fetch command
type FetchOptions struct {
	Branch    string
	NoDB      bool
	NoMedia   bool
	DBOnly    bool
	MediaOnly bool
	DryRun    bool
	DestPath  string
}

// SyncOptions holds options for the sync command
type SyncOptions struct {
	DBOnly      bool
	MediaOnly   bool
	BackupFirst bool
	DryRun      bool
}

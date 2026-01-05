// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package lib

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	// DefaultRepoURL is the default repository URL for magebox-lib
	DefaultRepoURL = "https://qoliber/magebox-lib.git"

	// LibDirName is the name of the lib directory inside .magebox
	LibDirName = "yaml"

	// LocalOverrideDirName is the name of the local override directory
	LocalOverrideDirName = "yaml-local"

	// InstallersDirName is the name of the installers subdirectory
	InstallersDirName = "installers"

	// TemplatesDirName is the name of the templates subdirectory
	TemplatesDirName = "templates"

	// VersionFileName is the name of the version file
	VersionFileName = "version.txt"
)

// Paths holds all the relevant paths for the lib system
type Paths struct {
	// MageBoxDir is the root .magebox directory (e.g., ~/.magebox)
	MageBoxDir string

	// LibDir is the main lib directory (e.g., ~/.magebox/yaml)
	LibDir string

	// LocalDir is the local override directory (e.g., ~/.magebox/yaml-local)
	LocalDir string

	// InstallersDir is the installers directory (e.g., ~/.magebox/yaml/installers)
	InstallersDir string

	// LocalInstallersDir is the local installers override directory
	LocalInstallersDir string

	// TemplatesDir is the templates directory (e.g., ~/.magebox/yaml/templates)
	TemplatesDir string

	// LocalTemplatesDir is the local templates override directory
	LocalTemplatesDir string

	// VersionFile is the path to version.txt
	VersionFile string
}

// NewPaths creates a new Paths instance for the given home directory
func NewPaths(homeDir string) *Paths {
	mageboxDir := filepath.Join(homeDir, ".magebox")
	libDir := filepath.Join(mageboxDir, LibDirName)
	localDir := filepath.Join(mageboxDir, LocalOverrideDirName)

	return &Paths{
		MageBoxDir:         mageboxDir,
		LibDir:             libDir,
		LocalDir:           localDir,
		InstallersDir:      filepath.Join(libDir, InstallersDirName),
		LocalInstallersDir: filepath.Join(localDir, InstallersDirName),
		TemplatesDir:       filepath.Join(libDir, TemplatesDirName),
		LocalTemplatesDir:  filepath.Join(localDir, TemplatesDirName),
		VersionFile:        filepath.Join(libDir, VersionFileName),
	}
}

// DefaultPaths returns paths using the current user's home directory
// If a custom lib_path is configured in global config, it uses that instead
func DefaultPaths() (*Paths, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	// Check for custom lib path in global config
	customPath := getCustomLibPath(homeDir)
	if customPath != "" {
		return NewPathsWithCustomLib(homeDir, customPath), nil
	}

	return NewPaths(homeDir), nil
}

// NewPathsWithCustomLib creates a Paths instance using a custom lib directory
func NewPathsWithCustomLib(homeDir, customLibPath string) *Paths {
	mageboxDir := filepath.Join(homeDir, ".magebox")
	localDir := filepath.Join(mageboxDir, LocalOverrideDirName)

	return &Paths{
		MageBoxDir:         mageboxDir,
		LibDir:             customLibPath,
		LocalDir:           localDir,
		InstallersDir:      filepath.Join(customLibPath, InstallersDirName),
		LocalInstallersDir: filepath.Join(localDir, InstallersDirName),
		TemplatesDir:       filepath.Join(customLibPath, TemplatesDirName),
		LocalTemplatesDir:  filepath.Join(localDir, TemplatesDirName),
		VersionFile:        filepath.Join(customLibPath, VersionFileName),
	}
}

// getCustomLibPath reads the lib_path from global config if set
func getCustomLibPath(homeDir string) string {
	configPath := filepath.Join(homeDir, ".magebox", "config.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return ""
	}

	// Simple struct to extract just the lib_path field
	var cfg struct {
		LibPath string `yaml:"lib_path"`
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return ""
	}

	return cfg.LibPath
}

// InstallerPath returns the path to a specific installer YAML file
// It checks local override first, then falls back to main lib
func (p *Paths) InstallerPath(name string) string {
	// Check local override first
	localPath := filepath.Join(p.LocalInstallersDir, name+".yaml")
	if fileExists(localPath) {
		return localPath
	}

	// Fall back to main lib
	return filepath.Join(p.InstallersDir, name+".yaml")
}

// InstallerPaths returns both the main and local override paths for an installer
func (p *Paths) InstallerPaths(name string) (mainPath, localPath string) {
	return filepath.Join(p.InstallersDir, name+".yaml"),
		filepath.Join(p.LocalInstallersDir, name+".yaml")
}

// TemplatePath returns the path to a specific template file
// It checks local override first, then falls back to main lib
// category is the template subdirectory (e.g., "nginx", "php", "varnish")
// name is the template filename (e.g., "vhost.conf.tmpl")
func (p *Paths) TemplatePath(category, name string) string {
	// Check local override first
	localPath := filepath.Join(p.LocalTemplatesDir, category, name)
	if fileExists(localPath) {
		return localPath
	}

	// Fall back to main lib
	return filepath.Join(p.TemplatesDir, category, name)
}

// TemplateExists returns true if a template exists (in local or main lib)
func (p *Paths) TemplateExists(category, name string) bool {
	return fileExists(p.TemplatePath(category, name))
}

// Exists returns true if the lib directory exists
func (p *Paths) Exists() bool {
	return fileExists(p.LibDir)
}

// IsGitRepo returns true if the lib directory is a git repository
func (p *Paths) IsGitRepo() bool {
	return fileExists(filepath.Join(p.LibDir, ".git"))
}

// EnsureLocalDir creates the local override directory if it doesn't exist
func (p *Paths) EnsureLocalDir() error {
	return os.MkdirAll(p.LocalInstallersDir, 0755)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

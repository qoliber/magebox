package project

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/qoliber/magebox/internal/config"
	"github.com/qoliber/magebox/internal/platform"
)

// ProjectInfo contains information about a discovered project
type ProjectInfo struct {
	Name       string
	Path       string
	Domains    []string
	PHPVersion string
	ConfigFile string
	HasConfig  bool
}

// ProjectDiscovery discovers MageBox projects
type ProjectDiscovery struct {
	platform *platform.Platform
}

// NewProjectDiscovery creates a new project discovery instance
func NewProjectDiscovery(p *platform.Platform) *ProjectDiscovery {
	return &ProjectDiscovery{platform: p}
}

// DiscoverProjects finds all MageBox projects by scanning nginx vhosts
func (d *ProjectDiscovery) DiscoverProjects() ([]ProjectInfo, error) {
	vhostsDir := filepath.Join(d.platform.MageBoxDir(), "nginx", "vhosts")

	// Check if vhosts directory exists
	if _, err := os.Stat(vhostsDir); os.IsNotExist(err) {
		return []ProjectInfo{}, nil
	}

	// Find all .conf files
	files, err := filepath.Glob(filepath.Join(vhostsDir, "*.conf"))
	if err != nil {
		return nil, err
	}

	projects := make([]ProjectInfo, 0)
	seen := make(map[string]bool)

	for _, file := range files {
		info, err := d.parseVhostFile(file)
		if err != nil {
			continue
		}

		// Deduplicate by path
		if seen[info.Path] {
			// Merge domains
			for i := range projects {
				if projects[i].Path == info.Path {
					projects[i].Domains = append(projects[i].Domains, info.Domains...)
					break
				}
			}
			continue
		}

		seen[info.Path] = true
		projects = append(projects, *info)
	}

	return projects, nil
}

// parseVhostFile parses a nginx vhost file to extract project info
func (d *ProjectDiscovery) parseVhostFile(path string) (*ProjectInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info := &ProjectInfo{
		Domains: make([]string, 0),
	}

	// Regex patterns
	serverNameRegex := regexp.MustCompile(`server_name\s+([^;]+);`)
	rootRegex := regexp.MustCompile(`root\s+([^;]+);`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Extract server_name
		if matches := serverNameRegex.FindStringSubmatch(line); len(matches) > 1 {
			domains := strings.Fields(matches[1])
			info.Domains = append(info.Domains, domains...)
		}

		// Extract root path
		if matches := rootRegex.FindStringSubmatch(line); len(matches) > 1 {
			rootPath := strings.TrimSpace(matches[1])
			// Root is typically /path/to/project/pub, so go up one level
			info.Path = filepath.Dir(rootPath)
		}
	}

	if info.Path == "" {
		return nil, err
	}

	// Try to load project config
	configPath := filepath.Join(info.Path, ".magebox")
	if _, err := os.Stat(configPath); err == nil {
		info.HasConfig = true
		info.ConfigFile = configPath

		// Load config to get name and PHP version
		cfg, err := config.LoadFromPath(info.Path)
		if err == nil {
			info.Name = cfg.Name
			info.PHPVersion = cfg.PHP
		}
	} else {
		// Try to derive name from path
		info.Name = filepath.Base(info.Path)
	}

	return info, nil
}

// FindProjectByDomain finds a project by its domain
func (d *ProjectDiscovery) FindProjectByDomain(domain string) (*ProjectInfo, error) {
	projects, err := d.DiscoverProjects()
	if err != nil {
		return nil, err
	}

	for _, p := range projects {
		for _, d := range p.Domains {
			if d == domain {
				return &p, nil
			}
		}
	}

	return nil, nil
}

// FindProjectByPath finds a project by its path
func (d *ProjectDiscovery) FindProjectByPath(path string) (*ProjectInfo, error) {
	projects, err := d.DiscoverProjects()
	if err != nil {
		return nil, err
	}

	// Normalize path
	absPath, _ := filepath.Abs(path)

	for _, p := range projects {
		projectAbs, _ := filepath.Abs(p.Path)
		if projectAbs == absPath {
			return &p, nil
		}
	}

	return nil, nil
}

// CountProjects returns the number of registered projects
func (d *ProjectDiscovery) CountProjects() int {
	projects, err := d.DiscoverProjects()
	if err != nil {
		return 0
	}
	return len(projects)
}

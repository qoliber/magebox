package magerunwrapper

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"qoliber/magebox/internal/lib"
	"qoliber/magebox/internal/platform"
)

//go:embed templates/magerun2.sh
var magerun2ScriptEmbed string

func init() {
	lib.RegisterFallbackTemplate(lib.TemplateWrapper, "magerun2.sh", magerun2ScriptEmbed)
}

const WrapperName = "magerun2"

// Manager handles n98-magerun2 phar lifecycle and wrapper script installation.
type Manager struct {
	platform   *platform.Platform
	httpClient *http.Client
}

// NewManager creates a new Manager.
func NewManager(p *platform.Platform) *Manager {
	return &Manager{
		platform: p,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Resolve returns the path to the correct n98-magerun2 phar for the project in
// projectDir. The GitHub API is called at most once — on the very first run
// when no suitable phar is cached yet. Subsequent calls are fully offline.
func (m *Manager) Resolve(projectDir string) (string, error) {
	magentoVersion := m.detectMagentoVersion(projectDir)
	pharDir := filepath.Join(m.platform.MageBoxDir(), "magerun")

	if needsLegacy(magentoVersion) {
		return m.ensurePhar(legacyVersion)
	}

	// Use any cached modern phar — no API call needed.
	if cached := findCachedModernVersion(pharDir); cached != "" {
		return filepath.Join(pharDir, fmt.Sprintf("n98-magerun2-%s.phar", cached)), nil
	}

	// First run: fetch latest version from GitHub and download it.
	version, err := fetchLatestVersion(m.httpClient)
	if err != nil {
		return "", fmt.Errorf("could not determine latest n98-magerun2 version: %w", err)
	}

	return m.ensurePhar(version)
}

// findCachedModernVersion returns the version string of the newest non-legacy
// phar found in pharDir, or "" if none exist.
func findCachedModernVersion(pharDir string) string {
	entries, err := os.ReadDir(pharDir)
	if err != nil {
		return ""
	}
	var best string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, "n98-magerun2-") || !strings.HasSuffix(name, ".phar") {
			continue
		}
		version := strings.TrimSuffix(strings.TrimPrefix(name, "n98-magerun2-"), ".phar")
		if version == legacyVersion {
			continue
		}
		if best == "" || compareVersions(version, best) > 0 {
			best = version
		}
	}
	return best
}

// compareVersions returns >0 if a is newer than b, 0 if equal, <0 if older.
func compareVersions(a, b string) int {
	ap := strings.SplitN(a, ".", 3)
	bp := strings.SplitN(b, ".", 3)
	for i := range 3 {
		x := versionPart(ap, i)
		y := versionPart(bp, i)
		if x != y {
			return x - y
		}
	}
	return 0
}

func versionPart(parts []string, i int) int {
	if i >= len(parts) {
		return 0
	}
	n, _ := strconv.Atoi(parts[i])
	return n
}

// detectMagentoVersion parses composer.lock in projectDir and returns the
// installed Magento package version (e.g. "2.4.8"), or "" if not found.
func (m *Manager) detectMagentoVersion(projectDir string) string {
	if projectDir == "" {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(projectDir, "composer.lock"))
	if err != nil {
		return ""
	}

	var lock struct {
		Packages []struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(data, &lock); err != nil {
		return ""
	}

	for _, pkg := range lock.Packages {
		for _, name := range magentoPackageNames {
			if pkg.Name == name {
				return pkg.Version
			}
		}
	}
	return ""
}

// ensurePhar returns the path to the cached phar for version, downloading it
// from GitHub if not already present.
func (m *Manager) ensurePhar(version string) (string, error) {
	pharDir := filepath.Join(m.platform.MageBoxDir(), "magerun")
	if err := os.MkdirAll(pharDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create magerun directory: %w", err)
	}

	dest := filepath.Join(pharDir, fmt.Sprintf("n98-magerun2-%s.phar", version))
	if _, err := os.Stat(dest); err == nil {
		return dest, nil
	}

	fmt.Fprintf(os.Stderr, "Downloading n98-magerun2 v%s...\n", version)
	url := fmt.Sprintf(pharDownloadBase, version)
	if err := downloadFile(m.httpClient, url, dest); err != nil {
		return "", fmt.Errorf("failed to download n98-magerun2 v%s: %w", version, err)
	}
	if err := verifyPharChecksum(m.httpClient, dest, version); err != nil {
		_ = os.Remove(dest)
		return "", err
	}
	if err := os.Chmod(dest, 0755); err != nil {
		return "", err
	}

	return dest, nil
}

// CachedVersion returns the version of the best cached phar
// ("7.5.0", "9.4.0", etc.), or "" if nothing is cached yet.
func (m *Manager) CachedVersion() string {
	pharDir := filepath.Join(m.platform.MageBoxDir(), "magerun")
	if modern := findCachedModernVersion(pharDir); modern != "" {
		return modern
	}
	legacyPhar := filepath.Join(pharDir, fmt.Sprintf("n98-magerun2-%s.phar", legacyVersion))
	if _, err := os.Stat(legacyPhar); err == nil {
		return legacyVersion
	}
	return ""
}

// Install writes the magerun2 wrapper script to ~/.magebox/bin/magerun2.
func (m *Manager) Install() error {
	binDir := filepath.Join(m.platform.MageBoxDir(), "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	content, err := lib.GetTemplate(lib.TemplateWrapper, "magerun2.sh")
	if err != nil {
		content = magerun2ScriptEmbed
	}

	dest := filepath.Join(binDir, WrapperName)
	if err := os.WriteFile(dest, []byte(content), 0755); err != nil {
		return fmt.Errorf("failed to write magerun2 wrapper: %w", err)
	}
	return nil
}

// IsInstalled reports whether the wrapper script is installed and executable.
func (m *Manager) IsInstalled() bool {
	dest := filepath.Join(m.platform.MageBoxDir(), "bin", WrapperName)
	info, err := os.Stat(dest)
	if err != nil {
		return false
	}
	return info.Mode().Perm()&0111 != 0
}

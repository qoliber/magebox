package magerunwrapper

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	githubReleasesURL = "https://api.github.com/repos/netz98/n98-magerun2/releases/latest"
	pharDownloadBase  = "https://github.com/netz98/n98-magerun2/releases/download/%s/n98-magerun2.phar"
	pharSHA256Base    = "https://github.com/netz98/n98-magerun2/releases/download/%s/n98-magerun2.phar.sha256"
)

type githubRelease struct {
	TagName string `json:"tag_name"`
}

func fetchLatestVersion(client *http.Client) (string, error) {
	req, err := http.NewRequest("GET", githubReleasesURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "MageBox")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("GitHub API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", fmt.Errorf("failed to parse GitHub response: %w", err)
	}

	return strings.TrimPrefix(rel.TagName, "v"), nil
}

// verifyPharChecksum downloads the published SHA256 checksum and verifies the
// phar at pharPath. Returns nil if the checksum file is unavailable (graceful
// degradation) or if the hash matches.
func verifyPharChecksum(client *http.Client, pharPath, version string) error {
	url := fmt.Sprintf(pharSHA256Base, version)
	resp, err := client.Get(url)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		return nil // checksum file not published — skip verification
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	// Format is "hexhash  filename" (shasum standard)
	fields := strings.Fields(string(body))
	if len(fields) == 0 {
		return nil
	}
	expected := strings.ToLower(fields[0])

	data, err := os.ReadFile(pharPath)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	actual := hex.EncodeToString(sum[:])

	if actual != expected {
		return fmt.Errorf("SHA256 mismatch for n98-magerun2 v%s: expected %s, got %s", version, expected, actual)
	}
	return nil
}

func downloadFile(client *http.Client, url, dest string) error {
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned HTTP %d", resp.StatusCode)
	}

	tmp, err := os.CreateTemp(filepath.Dir(dest), ".n98-magerun2-*.phar")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmpName, dest)
}

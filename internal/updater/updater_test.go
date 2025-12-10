package updater

import (
	"runtime"
	"strings"
	"testing"
)

func TestNewUpdater(t *testing.T) {
	u := NewUpdater("0.1.0")

	if u == nil {
		t.Fatal("NewUpdater should not return nil")
	}
	if u.currentVersion != "0.1.0" {
		t.Errorf("currentVersion = %v, want 0.1.0", u.currentVersion)
	}
	if u.httpClient == nil {
		t.Error("httpClient should not be nil")
	}
}

func TestUpdater_isNewerVersion(t *testing.T) {
	tests := []struct {
		current  string
		latest   string
		expected bool
	}{
		{"0.1.0", "0.2.0", true},
		{"0.1.0", "0.1.1", true},
		{"0.1.0", "1.0.0", true},
		{"0.2.0", "0.1.0", false},
		{"0.1.0", "0.1.0", false},
		{"v0.1.0", "v0.2.0", true},
		{"0.1.0", "v0.2.0", true},
		{"v0.1.0", "0.2.0", true},
		{"1.0.0", "0.9.9", false},
		{"0.10.0", "0.9.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.current+"_vs_"+tt.latest, func(t *testing.T) {
			u := NewUpdater(tt.current)
			if got := u.isNewerVersion(tt.latest); got != tt.expected {
				t.Errorf("isNewerVersion(%s) = %v, want %v", tt.latest, got, tt.expected)
			}
		})
	}
}

func TestUpdater_getAssetName(t *testing.T) {
	u := NewUpdater("0.1.0")
	assetName := u.getAssetName()

	// Should contain OS and arch
	if !strings.Contains(assetName, runtime.GOOS) {
		t.Errorf("Asset name %q should contain OS %q", assetName, runtime.GOOS)
	}
	if !strings.Contains(assetName, runtime.GOARCH) {
		t.Errorf("Asset name %q should contain arch %q", assetName, runtime.GOARCH)
	}
	if !strings.HasPrefix(assetName, "magebox-") {
		t.Errorf("Asset name %q should start with 'magebox-'", assetName)
	}
}

func TestGetPlatformInfo(t *testing.T) {
	info := GetPlatformInfo()

	if !strings.Contains(info, runtime.GOOS) {
		t.Errorf("Platform info %q should contain OS", info)
	}
	if !strings.Contains(info, runtime.GOARCH) {
		t.Errorf("Platform info %q should contain arch", info)
	}
}

func TestUpdateResult(t *testing.T) {
	result := UpdateResult{
		CurrentVersion:  "0.1.0",
		LatestVersion:   "0.2.0",
		UpdateAvailable: true,
		ReleaseNotes:    "Bug fixes",
		DownloadURL:     "https://example.com/download",
		AssetName:       "magebox-linux-amd64",
	}

	if result.CurrentVersion != "0.1.0" {
		t.Error("CurrentVersion mismatch")
	}
	if result.LatestVersion != "0.2.0" {
		t.Error("LatestVersion mismatch")
	}
	if !result.UpdateAvailable {
		t.Error("UpdateAvailable should be true")
	}
}

func TestGitHubRelease(t *testing.T) {
	release := GitHubRelease{
		TagName:    "v0.2.0",
		Name:       "Version 0.2.0",
		Body:       "Release notes here",
		Draft:      false,
		Prerelease: false,
		Assets: []ReleaseAsset{
			{
				Name:               "magebox-linux-amd64",
				BrowserDownloadURL: "https://github.com/qoliber/magebox/releases/download/v0.2.0/magebox-linux-amd64",
				Size:               10000000,
			},
		},
	}

	if release.TagName != "v0.2.0" {
		t.Error("TagName mismatch")
	}
	if len(release.Assets) != 1 {
		t.Error("Should have 1 asset")
	}
	if release.Assets[0].Name != "magebox-linux-amd64" {
		t.Error("Asset name mismatch")
	}
}

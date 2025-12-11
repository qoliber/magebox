package php

import (
	"strings"
	"testing"

	"github.com/qoliber/magebox/internal/platform"
)

func TestNewDetector(t *testing.T) {
	p := &platform.Platform{Type: platform.Linux}
	d := NewDetector(p)

	if d == nil {
		t.Fatal("NewDetector should not return nil")
	}
	if d.platform != p {
		t.Error("NewDetector should store the platform")
	}
}

func TestDetector_DetectAll(t *testing.T) {
	p := &platform.Platform{Type: platform.Linux}
	d := NewDetector(p)

	versions := d.DetectAll()

	if len(versions) != len(SupportedVersions) {
		t.Errorf("DetectAll() returned %d versions, want %d", len(versions), len(SupportedVersions))
	}

	// Check that all supported versions are represented
	versionMap := make(map[string]bool)
	for _, v := range versions {
		versionMap[v.Version] = true
	}

	for _, sv := range SupportedVersions {
		if !versionMap[sv] {
			t.Errorf("Version %s not found in DetectAll() result", sv)
		}
	}
}

func TestDetector_Detect(t *testing.T) {
	tests := []struct {
		name         string
		platformType platform.Type
		version      string
		wantBinary   string
		wantFPM      string
	}{
		{
			name:         "linux php 8.2",
			platformType: platform.Linux,
			version:      "8.2",
			wantBinary:   "/usr/bin/php8.2",
			wantFPM:      "/usr/sbin/php-fpm8.2",
		},
		{
			name:         "darwin arm64 php 8.2",
			platformType: platform.Darwin,
			version:      "8.2",
			wantBinary:   "/opt/homebrew/opt/php@8.2/bin/php",
			wantFPM:      "/opt/homebrew/opt/php@8.2/sbin/php-fpm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &platform.Platform{
				Type:           tt.platformType,
				IsAppleSilicon: tt.platformType == platform.Darwin,
			}
			d := NewDetector(p)

			v := d.Detect(tt.version)

			if v.Version != tt.version {
				t.Errorf("Version = %v, want %v", v.Version, tt.version)
			}
			if v.PHPBinary != tt.wantBinary {
				t.Errorf("PHPBinary = %v, want %v", v.PHPBinary, tt.wantBinary)
			}
			if v.FPMBinary != tt.wantFPM {
				t.Errorf("FPMBinary = %v, want %v", v.FPMBinary, tt.wantFPM)
			}
		})
	}
}

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"8.2", "8.2"},
		{"82", "8.2"},
		{"8.3", "8.3"},
		{"83", "8.3"},
		{"php8.2", "8.2"},
		{"PHP8.2", "8.2"},
		{"8.1", "8.1"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := normalizeVersion(tt.input); got != tt.expected {
				t.Errorf("normalizeVersion(%v) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestRequiredExtensions(t *testing.T) {
	exts := RequiredExtensions()

	if len(exts) == 0 {
		t.Error("RequiredExtensions() should not return empty slice")
	}

	// Check for some critical extensions
	required := []string{"pdo_mysql", "curl", "mbstring", "gd", "intl", "soap"}
	extMap := make(map[string]bool)
	for _, e := range exts {
		extMap[e] = true
	}

	for _, r := range required {
		if !extMap[r] {
			t.Errorf("Extension %s should be in required list", r)
		}
	}
}

func TestFormatNotInstalledMessage(t *testing.T) {
	tests := []struct {
		name         string
		version      string
		platformType platform.Type
		contains     []string
	}{
		{
			name:         "linux message",
			version:      "8.2",
			platformType: platform.Linux,
			contains:     []string{"PHP 8.2 not found", "ppa:ondrej/php", "php8.2-fpm"},
		},
		{
			name:         "darwin message",
			version:      "8.3",
			platformType: platform.Darwin,
			contains:     []string{"PHP 8.3 not found", "brew install php@8.3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &platform.Platform{Type: tt.platformType}
			msg := FormatNotInstalledMessage(tt.version, p)

			for _, s := range tt.contains {
				if !strings.Contains(msg, s) {
					t.Errorf("Message should contain %q, got:\n%s", s, msg)
				}
			}
		})
	}
}

func TestDetector_GetRecommendation(t *testing.T) {
	p := &platform.Platform{Type: platform.Linux}
	d := NewDetector(p)

	rec := d.GetRecommendation("8.2")

	if rec.Version != "8.2" {
		t.Errorf("Version = %v, want 8.2", rec.Version)
	}

	if rec.InstallCommand == "" {
		t.Error("InstallCommand should not be empty")
	}

	if !strings.Contains(rec.InstallCommand, "8.2") {
		t.Errorf("InstallCommand should contain version, got: %s", rec.InstallCommand)
	}
}

func TestSupportedVersions(t *testing.T) {
	if len(SupportedVersions) == 0 {
		t.Error("SupportedVersions should not be empty")
	}

	// All versions should be in X.Y format
	for _, v := range SupportedVersions {
		if !strings.Contains(v, ".") {
			t.Errorf("Version %s should be in X.Y format", v)
		}
	}

	// Should contain common versions
	expectedVersions := []string{"8.1", "8.2", "8.3"}
	for _, ev := range expectedVersions {
		found := false
		for _, v := range SupportedVersions {
			if v == ev {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("SupportedVersions should contain %s", ev)
		}
	}
}

func TestDetector_DetectInstalled(t *testing.T) {
	p := &platform.Platform{Type: platform.Linux}
	d := NewDetector(p)

	installed := d.DetectInstalled()

	// All returned versions should have Installed = true
	for _, v := range installed {
		if !v.Installed {
			t.Errorf("DetectInstalled returned version %s with Installed=false", v.Version)
		}
	}
}

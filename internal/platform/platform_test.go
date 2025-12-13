package platform

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDetect(t *testing.T) {
	p, err := Detect()
	if err != nil {
		t.Fatalf("Detect() failed: %v", err)
	}

	if p.HomeDir == "" {
		t.Error("HomeDir should not be empty")
	}

	if p.Arch == "" {
		t.Error("Arch should not be empty")
	}

	// Type should match the current OS
	switch runtime.GOOS {
	case "darwin":
		if p.Type != Darwin {
			t.Errorf("Type = %v, want darwin", p.Type)
		}
	case "linux":
		if p.Type != Linux {
			t.Errorf("Type = %v, want linux", p.Type)
		}
	}
}

func TestPlatform_IsSupported(t *testing.T) {
	tests := []struct {
		name     string
		platform Platform
		expected bool
	}{
		{
			name:     "darwin is supported",
			platform: Platform{Type: Darwin},
			expected: true,
		},
		{
			name:     "linux is supported",
			platform: Platform{Type: Linux},
			expected: true,
		},
		{
			name:     "unknown is not supported",
			platform: Platform{Type: Unknown},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.platform.IsSupported(); got != tt.expected {
				t.Errorf("IsSupported() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPlatform_MageBoxDir(t *testing.T) {
	p := &Platform{HomeDir: "/home/testuser"}
	expected := "/home/testuser/.magebox"
	if got := p.MageBoxDir(); got != expected {
		t.Errorf("MageBoxDir() = %v, want %v", got, expected)
	}
}

func TestPlatform_NginxConfigDir(t *testing.T) {
	tests := []struct {
		name     string
		platform Platform
		expected string
	}{
		{
			name:     "darwin arm64",
			platform: Platform{Type: Darwin, IsAppleSilicon: true},
			expected: "/opt/homebrew/etc/nginx",
		},
		{
			name:     "darwin amd64",
			platform: Platform{Type: Darwin, IsAppleSilicon: false},
			expected: "/usr/local/etc/nginx",
		},
		{
			name:     "linux",
			platform: Platform{Type: Linux},
			expected: "/etc/nginx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.platform.NginxConfigDir(); got != tt.expected {
				t.Errorf("NginxConfigDir() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPlatform_PHPFPMConfigDir(t *testing.T) {
	tests := []struct {
		name     string
		platform Platform
		version  string
		expected string
	}{
		{
			name:     "darwin arm64 php 8.2",
			platform: Platform{Type: Darwin, IsAppleSilicon: true},
			version:  "8.2",
			expected: "/opt/homebrew/etc/php/8.2/php-fpm.d",
		},
		{
			name:     "darwin amd64 php 8.2",
			platform: Platform{Type: Darwin, IsAppleSilicon: false},
			version:  "8.2",
			expected: "/usr/local/etc/php/8.2/php-fpm.d",
		},
		{
			name:     "linux php 8.2",
			platform: Platform{Type: Linux},
			version:  "8.2",
			expected: "/etc/php/8.2/fpm/pool.d",
		},
		{
			name:     "linux php 8.3",
			platform: Platform{Type: Linux},
			version:  "8.3",
			expected: "/etc/php/8.3/fpm/pool.d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.platform.PHPFPMConfigDir(tt.version); got != tt.expected {
				t.Errorf("PHPFPMConfigDir() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPlatform_PHPFPMBinary(t *testing.T) {
	tests := []struct {
		name           string
		platform       Platform
		version        string
		expectedPrefix string
		expectedSuffix string
	}{
		{
			name:           "darwin arm64 php 8.2",
			platform:       Platform{Type: Darwin, IsAppleSilicon: true},
			version:        "8.2",
			expectedPrefix: "/opt/homebrew/",
			expectedSuffix: "sbin/php-fpm",
		},
		{
			name:           "darwin amd64 php 8.2",
			platform:       Platform{Type: Darwin, IsAppleSilicon: false},
			version:        "8.2",
			expectedPrefix: "/usr/local/",
			expectedSuffix: "sbin/php-fpm",
		},
		{
			name:           "linux php 8.2",
			platform:       Platform{Type: Linux},
			version:        "8.2",
			expectedPrefix: "/usr/sbin/",
			expectedSuffix: "php-fpm8.2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.platform.PHPFPMBinary(tt.version)
			// Check prefix and suffix to handle both Cellar and opt symlink paths
			if !strings.HasPrefix(got, tt.expectedPrefix) || !strings.HasSuffix(got, tt.expectedSuffix) {
				t.Errorf("PHPFPMBinary() = %v, want prefix %v and suffix %v", got, tt.expectedPrefix, tt.expectedSuffix)
			}
		})
	}
}

func TestPlatform_PHPBinary(t *testing.T) {
	tests := []struct {
		name           string
		platform       Platform
		version        string
		expectedPrefix string
		expectedSuffix string
	}{
		{
			name:           "darwin arm64 php 8.2",
			platform:       Platform{Type: Darwin, IsAppleSilicon: true},
			version:        "8.2",
			expectedPrefix: "/opt/homebrew/",
			expectedSuffix: "bin/php",
		},
		{
			name:           "linux php 8.2",
			platform:       Platform{Type: Linux},
			version:        "8.2",
			expectedPrefix: "/usr/bin/",
			expectedSuffix: "php8.2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.platform.PHPBinary(tt.version)
			// Check prefix and suffix to handle both Cellar and opt symlink paths
			if !strings.HasPrefix(got, tt.expectedPrefix) || !strings.HasSuffix(got, tt.expectedSuffix) {
				t.Errorf("PHPBinary() = %v, want prefix %v and suffix %v", got, tt.expectedPrefix, tt.expectedSuffix)
			}
		})
	}
}

func TestPlatform_PHPInstallCommand(t *testing.T) {
	tests := []struct {
		name     string
		platform Platform
		version  string
		contains string
	}{
		{
			name:     "darwin",
			platform: Platform{Type: Darwin},
			version:  "8.2",
			contains: "brew install php@8.2",
		},
		{
			name:     "linux",
			platform: Platform{Type: Linux},
			version:  "8.2",
			contains: "ppa:ondrej/php",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.platform.PHPInstallCommand(tt.version)
			if got == "" {
				t.Error("PHPInstallCommand() should not be empty")
			}
			if !contains(got, tt.contains) {
				t.Errorf("PHPInstallCommand() = %v, should contain %v", got, tt.contains)
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
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := normalizeVersion(tt.input); got != tt.expected {
				t.Errorf("normalizeVersion(%v) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestCommandExists(t *testing.T) {
	// "ls" should exist on all Unix systems
	if !CommandExists("ls") {
		t.Error("ls command should exist")
	}

	// "nonexistentcommand12345" should not exist
	if CommandExists("nonexistentcommand12345") {
		t.Error("nonexistentcommand12345 should not exist")
	}
}

func TestBinaryExists(t *testing.T) {
	// Create a temp file to test
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "testbinary")
	if err := os.WriteFile(tmpFile, []byte("test"), 0755); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	if !BinaryExists(tmpFile) {
		t.Error("BinaryExists() should return true for existing file")
	}

	if BinaryExists(filepath.Join(tmpDir, "nonexistent")) {
		t.Error("BinaryExists() should return false for non-existing file")
	}
}

func TestPlatform_HostsFilePath(t *testing.T) {
	p := &Platform{}
	expected := "/etc/hosts"
	if got := p.HostsFilePath(); got != expected {
		t.Errorf("HostsFilePath() = %v, want %v", got, expected)
	}
}

func TestPlatform_VarnishBinary(t *testing.T) {
	tests := []struct {
		name     string
		platform Platform
		expected string
	}{
		{
			name:     "darwin arm64",
			platform: Platform{Type: Darwin, IsAppleSilicon: true},
			expected: "/opt/homebrew/sbin/varnishd",
		},
		{
			name:     "darwin amd64",
			platform: Platform{Type: Darwin, IsAppleSilicon: false},
			expected: "/usr/local/sbin/varnishd",
		},
		{
			name:     "linux",
			platform: Platform{Type: Linux},
			expected: "/usr/sbin/varnishd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.platform.VarnishBinary(); got != tt.expected {
				t.Errorf("VarnishBinary() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

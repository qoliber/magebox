package ssl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/qoliber/magebox/internal/platform"
)

func TestNewManager(t *testing.T) {
	// Test Linux - now uses ~/.magebox/certs (same as macOS)
	// nginx runs as current user, so it can access home dir certs
	p := &platform.Platform{
		Type:    platform.Linux,
		HomeDir: "/home/testuser",
	}
	m := NewManager(p)

	if m == nil {
		t.Fatal("NewManager should not return nil")
	}

	expectedCertsDir := "/home/testuser/.magebox/certs"
	if m.certsDir != expectedCertsDir {
		t.Errorf("certsDir = %v, want %v", m.certsDir, expectedCertsDir)
	}

	// Test macOS - uses ~/.magebox/certs
	pMac := &platform.Platform{
		Type:    platform.Darwin,
		HomeDir: "/Users/testuser",
	}
	mMac := NewManager(pMac)
	expectedMacCertsDir := "/Users/testuser/.magebox/certs"
	if mMac.certsDir != expectedMacCertsDir {
		t.Errorf("certsDir (macOS) = %v, want %v", mMac.certsDir, expectedMacCertsDir)
	}
}

func TestManager_CertsDir(t *testing.T) {
	// Test Linux - now uses ~/.magebox/certs (same as macOS)
	p := &platform.Platform{
		Type:    platform.Linux,
		HomeDir: "/home/testuser",
	}
	m := NewManager(p)

	expected := "/home/testuser/.magebox/certs"
	if got := m.CertsDir(); got != expected {
		t.Errorf("CertsDir() = %v, want %v", got, expected)
	}

	// Test macOS
	pMac := &platform.Platform{
		Type:    platform.Darwin,
		HomeDir: "/Users/testuser",
	}
	mMac := NewManager(pMac)
	expectedMac := "/Users/testuser/.magebox/certs"
	if got := mMac.CertsDir(); got != expectedMac {
		t.Errorf("CertsDir() (macOS) = %v, want %v", got, expectedMac)
	}
}

func TestManager_GetCertPaths(t *testing.T) {
	// Use macOS platform for testing (uses home dir, not /etc)
	p := &platform.Platform{
		Type:    platform.Darwin,
		HomeDir: "/Users/testuser",
	}
	m := NewManager(p)

	paths := m.GetCertPaths("mystore.test")

	expectedCert := "/Users/testuser/.magebox/certs/mystore.test/cert.pem"
	expectedKey := "/Users/testuser/.magebox/certs/mystore.test/key.pem"

	if paths.CertFile != expectedCert {
		t.Errorf("CertFile = %v, want %v", paths.CertFile, expectedCert)
	}
	if paths.KeyFile != expectedKey {
		t.Errorf("KeyFile = %v, want %v", paths.KeyFile, expectedKey)
	}
	if paths.Domain != "mystore.test" {
		t.Errorf("Domain = %v, want mystore.test", paths.Domain)
	}
}

func TestManager_CertExists(t *testing.T) {
	// Create temp directory structure - use Darwin to use home dir based certs
	tmpDir := t.TempDir()
	p := &platform.Platform{
		Type:    platform.Darwin,
		HomeDir: tmpDir,
	}
	m := NewManager(p)

	// Initially should not exist
	if m.CertExists("mystore.test") {
		t.Error("CertExists should return false for non-existent cert")
	}

	// Create cert files
	domainDir := filepath.Join(m.certsDir, "mystore.test")
	if err := os.MkdirAll(domainDir, 0755); err != nil {
		t.Fatalf("failed to create domain dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(domainDir, "cert.pem"), []byte("cert"), 0644); err != nil {
		t.Fatalf("failed to create cert file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(domainDir, "key.pem"), []byte("key"), 0644); err != nil {
		t.Fatalf("failed to create key file: %v", err)
	}

	// Now should exist
	if !m.CertExists("mystore.test") {
		t.Error("CertExists should return true for existing cert")
	}
}

func TestManager_RemoveCert(t *testing.T) {
	tmpDir := t.TempDir()
	p := &platform.Platform{
		Type:    platform.Darwin,
		HomeDir: tmpDir,
	}
	m := NewManager(p)

	// Create cert files
	domainDir := filepath.Join(m.certsDir, "mystore.test")
	if err := os.MkdirAll(domainDir, 0755); err != nil {
		t.Fatalf("failed to create domain dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(domainDir, "cert.pem"), []byte("cert"), 0644); err != nil {
		t.Fatalf("failed to create cert file: %v", err)
	}

	// Remove cert
	if err := m.RemoveCert("mystore.test"); err != nil {
		t.Errorf("RemoveCert failed: %v", err)
	}

	// Should no longer exist
	if m.CertExists("mystore.test") {
		t.Error("CertExists should return false after RemoveCert")
	}
}

func TestManager_ListCerts(t *testing.T) {
	tmpDir := t.TempDir()
	p := &platform.Platform{
		Type:    platform.Darwin,
		HomeDir: tmpDir,
	}
	m := NewManager(p)

	// Initially empty
	certs, err := m.ListCerts()
	if err != nil {
		t.Fatalf("ListCerts failed: %v", err)
	}
	if len(certs) != 0 {
		t.Errorf("ListCerts should return empty slice, got %v", certs)
	}

	// Create some cert directories
	domains := []string{"mystore.test", "api.mystore.test", "other.test"}
	for _, domain := range domains {
		domainDir := filepath.Join(m.certsDir, domain)
		if err := os.MkdirAll(domainDir, 0755); err != nil {
			t.Fatalf("failed to create domain dir: %v", err)
		}
	}

	// Should list all domains
	certs, err = m.ListCerts()
	if err != nil {
		t.Fatalf("ListCerts failed: %v", err)
	}
	if len(certs) != 3 {
		t.Errorf("ListCerts should return 3 certs, got %d", len(certs))
	}
}

func TestExtractBaseDomain(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"mystore.test", "mystore.test"},
		{"api.mystore.test", "mystore.test"},
		{"admin.api.mystore.test", "mystore.test"},
		{"test", "test"},
		{"localhost", "localhost"},
		{"sub.domain.example.com", "example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := ExtractBaseDomain(tt.input); got != tt.expected {
				t.Errorf("ExtractBaseDomain(%v) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestGroupDomainsByBase(t *testing.T) {
	domains := []string{
		"mystore.test",
		"api.mystore.test",
		"admin.mystore.test",
		"other.test",
		"api.other.test",
	}

	groups := GroupDomainsByBase(domains)

	if len(groups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(groups))
	}

	if len(groups["mystore.test"]) != 3 {
		t.Errorf("Expected 3 domains in mystore.test group, got %d", len(groups["mystore.test"]))
	}

	if len(groups["other.test"]) != 2 {
		t.Errorf("Expected 2 domains in other.test group, got %d", len(groups["other.test"]))
	}
}

func TestMkcertNotInstalledError(t *testing.T) {
	tests := []struct {
		name         string
		platformType platform.Type
		linuxDistro  platform.LinuxDistro
		contains     string
	}{
		{
			name:         "linux debian error message",
			platformType: platform.Linux,
			linuxDistro:  platform.DistroDebian,
			contains:     "apt install",
		},
		{
			name:         "linux fedora error message",
			platformType: platform.Linux,
			linuxDistro:  platform.DistroFedora,
			contains:     "dnf install",
		},
		{
			name:         "darwin error message",
			platformType: platform.Darwin,
			contains:     "brew install mkcert",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &platform.Platform{Type: tt.platformType, LinuxDistro: tt.linuxDistro}
			err := &MkcertNotInstalledError{Platform: p}

			msg := err.Error()
			if msg == "" {
				t.Error("Error message should not be empty")
			}

			if !containsString(msg, tt.contains) {
				t.Errorf("Error message should contain %q, got:\n%s", tt.contains, msg)
			}
		})
	}
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

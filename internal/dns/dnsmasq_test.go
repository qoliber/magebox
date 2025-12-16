package dns

import (
	"strings"
	"testing"

	"github.com/qoliber/magebox/internal/platform"
)

func TestNewDnsmasqManager(t *testing.T) {
	p := &platform.Platform{Type: platform.Linux}
	m := NewDnsmasqManager(p)

	if m == nil {
		t.Error("NewDnsmasqManager should not return nil")
	}
}

func TestDnsmasqManager_getConfigDir(t *testing.T) {
	tests := []struct {
		name         string
		platformType platform.Type
		expected     string
	}{
		{
			name:         "Linux",
			platformType: platform.Linux,
			expected:     "/etc/dnsmasq.d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &platform.Platform{Type: tt.platformType}
			m := NewDnsmasqManager(p)

			if got := m.getConfigDir(); got != tt.expected {
				t.Errorf("getConfigDir() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDnsmasqManager_getConfigPath(t *testing.T) {
	p := &platform.Platform{Type: platform.Linux}
	m := NewDnsmasqManager(p)

	path := m.getConfigPath()

	if !strings.HasSuffix(path, "magebox.conf") {
		t.Errorf("getConfigPath() = %v, should end with magebox.conf", path)
	}
	if !strings.Contains(path, "dnsmasq") {
		t.Errorf("getConfigPath() = %v, should contain dnsmasq", path)
	}
}

func TestDnsmasqManager_generateConfig(t *testing.T) {
	tests := []struct {
		name          string
		platformType  platform.Type
		expectedLines []string
	}{
		{
			name:         "Linux",
			platformType: platform.Linux,
			expectedLines: []string{
				"address=/test/127.0.0.1",
				"listen-address=127.0.0.2", // Linux uses 127.0.0.2 to avoid systemd-resolved conflict
				"bind-interfaces",
				"MageBox",
			},
		},
		{
			name:         "macOS",
			platformType: platform.Darwin,
			expectedLines: []string{
				"address=/test/127.0.0.1",
				"listen-address=127.0.0.1", // macOS uses 127.0.0.1
				"bind-interfaces",
				"MageBox",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &platform.Platform{Type: tt.platformType}
			m := NewDnsmasqManager(p)

			config := m.generateConfig()

			for _, line := range tt.expectedLines {
				if !strings.Contains(config, line) {
					t.Errorf("Config should contain %q", line)
				}
			}
		})
	}
}

func TestDnsmasqManager_InstallCommand(t *testing.T) {
	tests := []struct {
		name         string
		platformType platform.Type
		expected     string
	}{
		{
			name:         "Linux",
			platformType: platform.Linux,
			expected:     "sudo apt install -y dnsmasq",
		},
		{
			name:         "macOS",
			platformType: platform.Darwin,
			expected:     "brew install dnsmasq",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &platform.Platform{Type: tt.platformType}
			m := NewDnsmasqManager(p)

			if got := m.InstallCommand(); got != tt.expected {
				t.Errorf("InstallCommand() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDnsmasqStatus(t *testing.T) {
	status := DnsmasqStatus{
		Installed:  true,
		Configured: true,
		Running:    true,
		TestDomain: "test.test",
		Resolving:  true,
	}

	if !status.Installed {
		t.Error("Installed should be true")
	}
	if !status.Configured {
		t.Error("Configured should be true")
	}
	if !status.Running {
		t.Error("Running should be true")
	}
	if status.TestDomain != "test.test" {
		t.Errorf("TestDomain = %v, want test.test", status.TestDomain)
	}
}

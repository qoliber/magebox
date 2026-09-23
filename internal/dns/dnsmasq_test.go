package dns

import (
	"strings"
	"testing"

	"qoliber/magebox/internal/platform"
)

func TestNewDnsmasqManager(t *testing.T) {
	p := &platform.Platform{Type: platform.Linux}
	if m := NewDnsmasqManager(p); m == nil {
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
	// Use temp dir as HOME so global config defaults apply (TLD=test)
	t.Setenv("HOME", t.TempDir())

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

// Ubuntu's dnsmasq-base package ships the binary without a service unit.
// MageBox starts dnsmasq through systemd, so the binary alone is not enough:
// without this check bootstrap reports dnsmasq as installed and then fails to
// start it with a bare exit code.
func TestSystemdUnitListed(t *testing.T) {
	tests := []struct {
		name   string
		output string
		unit   string
		want   bool
	}{
		{
			name:   "unit present",
			output: "UNIT FILE         STATE    PRESET\ndnsmasq.service   enabled  enabled\n\n1 unit files listed.",
			unit:   "dnsmasq.service",
			want:   true,
		},
		{
			name:   "no unit files",
			output: "UNIT FILE   STATE   PRESET\n\n0 unit files listed.",
			unit:   "dnsmasq.service",
			want:   false,
		},
		{
			name:   "different unit listed",
			output: "UNIT FILE        STATE    PRESET\nnginx.service    enabled  enabled\n\n1 unit files listed.",
			unit:   "dnsmasq.service",
			want:   false,
		},
		{
			name:   "empty output",
			output: "",
			unit:   "dnsmasq.service",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := systemdUnitListed(tt.output, tt.unit); got != tt.want {
				t.Errorf("systemdUnitListed(%q) = %v, want %v", tt.unit, got, tt.want)
			}
		})
	}
}

// When dnsmasq cannot be started, MageBox falls back to /etc/hosts. Leaving the
// systemd-resolved drop-in in place then points every .test lookup at a
// resolver that does not exist, so names fail outright instead of falling
// through to the hosts file.
func TestNeedsResolvedCleanup(t *testing.T) {
	tests := []struct {
		name        string
		dnsmasqOK   bool
		dropInFound bool
		want        bool
	}{
		{name: "dnsmasq works, drop-in belongs there", dnsmasqOK: true, dropInFound: true, want: false},
		{name: "fell back with a stale drop-in", dnsmasqOK: false, dropInFound: true, want: true},
		{name: "fell back with nothing to clean", dnsmasqOK: false, dropInFound: false, want: false},
		{name: "dnsmasq works, no drop-in yet", dnsmasqOK: true, dropInFound: false, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NeedsResolvedCleanup(tt.dnsmasqOK, tt.dropInFound); got != tt.want {
				t.Errorf("NeedsResolvedCleanup(%v, %v) = %v, want %v", tt.dnsmasqOK, tt.dropInFound, got, tt.want)
			}
		})
	}
}

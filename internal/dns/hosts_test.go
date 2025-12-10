package dns

import (
	"strings"
	"testing"

	"github.com/qoliber/magebox/internal/platform"
)

func TestNewHostsManager(t *testing.T) {
	p := &platform.Platform{Type: platform.Linux}
	m := NewHostsManager(p)

	if m == nil {
		t.Error("NewHostsManager should not return nil")
	}
	if m.hostsFile != "/etc/hosts" {
		t.Errorf("hostsFile = %v, want /etc/hosts", m.hostsFile)
	}
}

func TestHostsManager_extractMageBoxDomains(t *testing.T) {
	p := &platform.Platform{Type: platform.Linux}
	m := NewHostsManager(p)

	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name: "with magebox section",
			content: `127.0.0.1 localhost
# >>> MageBox managed hosts - do not edit manually >>>
127.0.0.1 mystore.test
127.0.0.1 api.mystore.test
# <<< MageBox managed hosts <<<
`,
			expected: []string{"mystore.test", "api.mystore.test"},
		},
		{
			name: "without magebox section",
			content: `127.0.0.1 localhost
127.0.0.1 someother.test
`,
			expected: []string{},
		},
		{
			name:     "empty file",
			content:  "",
			expected: []string{},
		},
		{
			name: "multiple domains on one line",
			content: `# >>> MageBox managed hosts - do not edit manually >>>
127.0.0.1 mystore.test api.mystore.test admin.mystore.test
# <<< MageBox managed hosts <<<
`,
			expected: []string{"mystore.test", "api.mystore.test", "admin.mystore.test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			domains := m.extractMageBoxDomains(tt.content)

			if len(domains) != len(tt.expected) {
				t.Errorf("got %d domains, want %d", len(domains), len(tt.expected))
				return
			}

			for i, d := range domains {
				if d != tt.expected[i] {
					t.Errorf("domain[%d] = %v, want %v", i, d, tt.expected[i])
				}
			}
		})
	}
}

func TestHostsManager_removeMageBoxSection(t *testing.T) {
	p := &platform.Platform{Type: platform.Linux}
	m := NewHostsManager(p)

	content := `127.0.0.1 localhost
::1 localhost

# >>> MageBox managed hosts - do not edit manually >>>
127.0.0.1 mystore.test
127.0.0.1 api.mystore.test
# <<< MageBox managed hosts <<<

# Some other comment
127.0.0.1 other.test
`

	result := m.removeMageBoxSection(content)

	// Should not contain MageBox markers
	if strings.Contains(result, MageBoxStartMarker) {
		t.Error("Result should not contain start marker")
	}
	if strings.Contains(result, MageBoxEndMarker) {
		t.Error("Result should not contain end marker")
	}

	// Should not contain MageBox domains
	if strings.Contains(result, "mystore.test") {
		t.Error("Result should not contain mystore.test")
	}

	// Should still contain other entries
	if !strings.Contains(result, "localhost") {
		t.Error("Result should contain localhost")
	}
	if !strings.Contains(result, "other.test") {
		t.Error("Result should contain other.test")
	}
}

func TestGenerateMageBoxSection(t *testing.T) {
	tests := []struct {
		name     string
		domains  []string
		contains []string
	}{
		{
			name:     "empty domains",
			domains:  []string{},
			contains: []string{},
		},
		{
			name:    "single domain",
			domains: []string{"mystore.test"},
			contains: []string{
				MageBoxStartMarker,
				"127.0.0.1 mystore.test",
				MageBoxEndMarker,
			},
		},
		{
			name:    "multiple domains",
			domains: []string{"mystore.test", "api.mystore.test"},
			contains: []string{
				MageBoxStartMarker,
				"127.0.0.1 mystore.test",
				"127.0.0.1 api.mystore.test",
				MageBoxEndMarker,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateMageBoxSection(tt.domains)

			if len(tt.domains) == 0 && result != "" {
				t.Error("Empty domains should return empty string")
				return
			}

			for _, c := range tt.contains {
				if !strings.Contains(result, c) {
					t.Errorf("Result should contain %q", c)
				}
			}
		})
	}
}

func TestParseHostsLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected *HostEntry
	}{
		{
			name:     "valid entry",
			line:     "127.0.0.1 mystore.test",
			expected: &HostEntry{IP: "127.0.0.1", Domain: "mystore.test"},
		},
		{
			name:     "with extra whitespace",
			line:     "  127.0.0.1   mystore.test  ",
			expected: &HostEntry{IP: "127.0.0.1", Domain: "mystore.test"},
		},
		{
			name:     "comment line",
			line:     "# This is a comment",
			expected: nil,
		},
		{
			name:     "empty line",
			line:     "",
			expected: nil,
		},
		{
			name:     "whitespace only",
			line:     "   ",
			expected: nil,
		},
		{
			name:     "ipv6 entry",
			line:     "::1 localhost",
			expected: &HostEntry{IP: "::1", Domain: "localhost"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseHostsLine(tt.line)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil, got %v", result)
				}
				return
			}

			if result == nil {
				t.Error("Expected non-nil result")
				return
			}

			if result.IP != tt.expected.IP {
				t.Errorf("IP = %v, want %v", result.IP, tt.expected.IP)
			}
			if result.Domain != tt.expected.Domain {
				t.Errorf("Domain = %v, want %v", result.Domain, tt.expected.Domain)
			}
		})
	}
}

func TestFormatHostsLine(t *testing.T) {
	entry := HostEntry{IP: "127.0.0.1", Domain: "mystore.test"}
	expected := "127.0.0.1 mystore.test"

	if got := FormatHostsLine(entry); got != expected {
		t.Errorf("FormatHostsLine() = %v, want %v", got, expected)
	}
}

func TestMageBoxMarkers(t *testing.T) {
	// Ensure markers are consistent and valid
	if !strings.HasPrefix(MageBoxStartMarker, "#") {
		t.Error("Start marker should be a comment")
	}
	if !strings.HasPrefix(MageBoxEndMarker, "#") {
		t.Error("End marker should be a comment")
	}
	if MageBoxStartMarker == MageBoxEndMarker {
		t.Error("Start and end markers should be different")
	}
}

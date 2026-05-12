package magerunwrapper

import "testing"

func TestParseMagentoMinor(t *testing.T) {
	tests := []struct {
		version string
		minor   int
		ok      bool
	}{
		{"2.4.0", 0, true},
		{"2.4.4", 4, true},
		{"2.4.5", 5, true},
		{"2.4.8", 8, true},
		{"2.4.5-p1", 5, true},
		{"2.4.8-p3", 8, true},
		{"v2.4.7", 7, true},
		// non-2.4.x inputs
		{"2.3.7", 0, false},
		{"3.0.0", 0, false},
		{"103.0.8", 0, false}, // magento/framework version
		{"", 0, false},
		{"invalid", 0, false},
	}

	for _, tt := range tests {
		got, ok := parseMagentoMinor(tt.version)
		if ok != tt.ok || got != tt.minor {
			t.Errorf("parseMagentoMinor(%q) = (%d, %v), want (%d, %v)", tt.version, got, ok, tt.minor, tt.ok)
		}
	}
}

func TestNeedsLegacy(t *testing.T) {
	tests := []struct {
		version string
		legacy  bool
	}{
		{"2.4.0", true},
		{"2.4.3", true},
		{"2.4.4", true},
		{"2.4.5", false},
		{"2.4.8", false},
		{"2.4.4-p1", true},
		{"2.4.5-p2", false},
		// unknown/empty — fall back to latest
		{"", false},
		{"invalid", false},
		{"2.3.7", false},
	}

	for _, tt := range tests {
		got := needsLegacy(tt.version)
		if got != tt.legacy {
			t.Errorf("needsLegacy(%q) = %v, want %v", tt.version, got, tt.legacy)
		}
	}
}

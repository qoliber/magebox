package installer

import "testing"

// Ondrej's PHP PPA publishes nothing for Ubuntu 26.04, so bootstrap must pin
// the newest suite it does publish. Without this, PHP 8.1 through 8.4 simply do
// not exist in apt and only Ubuntu's own PHP can be installed.
func TestPickPPASuite(t *testing.T) {
	published := func(suites ...string) func(string) bool {
		set := make(map[string]bool, len(suites))
		for _, s := range suites {
			set[s] = true
		}
		return func(suite string) bool { return set[suite] }
	}

	tests := []struct {
		name         string
		current      string
		published    func(string) bool
		wantSuite    string
		wantFallback bool
	}{
		{
			name:      "current release is published",
			current:   "noble",
			published: published("noble", "jammy"),
			wantSuite: "noble",
		},
		{
			name:         "falls back to the newest published release",
			current:      "resolute",
			published:    published("noble", "jammy"),
			wantSuite:    "noble",
			wantFallback: true,
		},
		{
			name:         "skips unpublished releases in between",
			current:      "questing",
			published:    published("jammy"),
			wantSuite:    "jammy",
			wantFallback: true,
		},
		{
			name:         "unknown newer codename starts at the top of the list",
			current:      "vivacious",
			published:    published("noble"),
			wantSuite:    "noble",
			wantFallback: true,
		},
		{
			name:      "nothing published keeps the current release",
			current:   "resolute",
			published: published(),
			wantSuite: "resolute",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suite, fallback := pickPPASuite(tt.current, tt.published)
			if suite != tt.wantSuite || fallback != tt.wantFallback {
				t.Errorf("pickPPASuite(%q) = (%q, %v), want (%q, %v)", tt.current, suite, fallback, tt.wantSuite, tt.wantFallback)
			}
		})
	}
}

func TestUbuntuCodenamesAreOrderedNewestFirst(t *testing.T) {
	if len(ubuntuCodenames) < 5 {
		t.Fatalf("expected a meaningful list of codenames, got %d", len(ubuntuCodenames))
	}
	if ubuntuCodenames[len(ubuntuCodenames)-1] != "focal" {
		t.Errorf("oldest entry = %q, want focal (20.04, the oldest supported release)", ubuntuCodenames[len(ubuntuCodenames)-1])
	}
	seen := make(map[string]bool)
	for _, c := range ubuntuCodenames {
		if seen[c] {
			t.Errorf("duplicate codename %q", c)
		}
		seen[c] = true
	}
}

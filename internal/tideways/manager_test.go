// Copyright (c) qoliber

package tideways

import (
	"strings"
	"testing"
)

// TestStripIniDirective covers the migration path used by
// CleanLegacyExtensionDirectives. Older MageBox versions briefly wrote
// tideways.api_key and tideways.environment to the PHP extension ini; we now
// evict both on every `magebox tideways config` run.
func TestStripIniDirective(t *testing.T) {
	tests := []struct {
		name      string
		existing  string
		directive string
		want      string
	}{
		{
			name:      "strips uncommented stale api_key line",
			existing:  "extension=tideways.so\ntideways.api_key=legacy\n",
			directive: "tideways.api_key",
			want:      "extension=tideways.so\n",
		},
		{
			name:      "strips commented stale environment line",
			existing:  "extension=tideways.so\n;tideways.environment=local_x\n",
			directive: "tideways.environment",
			want:      "extension=tideways.so\n",
		},
		{
			name:      "strips directive with leading whitespace and comment marker",
			existing:  "extension=tideways.so\n  ; tideways.api_key=legacy\n",
			directive: "tideways.api_key",
			want:      "extension=tideways.so\n",
		},
		{
			name:      "no-op when directive absent",
			existing:  "extension=tideways.so\n",
			directive: "tideways.environment",
			want:      "extension=tideways.so\n",
		},
		{
			name:      "handles empty file",
			existing:  "",
			directive: "tideways.api_key",
			want:      "",
		},
		{
			name:      "strips multiple occurrences",
			existing:  "tideways.api_key=a\nextension=tideways.so\ntideways.api_key=b\n",
			directive: "tideways.api_key",
			want:      "extension=tideways.so\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripIniDirective(tt.existing, tt.directive)
			if got != tt.want {
				t.Errorf("stripIniDirective(%q, %q)\n got: %q\nwant: %q", tt.existing, tt.directive, got, tt.want)
			}
		})
	}
}

// TestStripIniDirective_ComposedCleanup exercises the realistic case where
// both stale Tideways directives are present and CleanLegacyExtensionDirectives
// runs stripIniDirective twice in sequence.
func TestStripIniDirective_ComposedCleanup(t *testing.T) {
	existing := "; Tideways PHP extension\nextension=tideways.so\ntideways.api_key=legacy\ntideways.environment=local_old\n"
	got := stripIniDirective(existing, "tideways.api_key")
	got = stripIniDirective(got, "tideways.environment")

	want := "; Tideways PHP extension\nextension=tideways.so\n"
	if got != want {
		t.Errorf("composed cleanup:\n got: %q\nwant: %q", got, want)
	}
}

// TestRenderDaemonEnvironmentDropIn verifies the systemd drop-in body shape.
// This is the file contents that get written to
// /etc/systemd/system/tideways-daemon.service.d/magebox-environment.conf
// on Linux so the daemon inherits TIDEWAYS_ENVIRONMENT.
func TestRenderDaemonEnvironmentDropIn(t *testing.T) {
	got := renderDaemonEnvironmentDropIn("local_peterjaap")

	// The [Service] section and the exact Environment= line are load-bearing
	// — systemd parses them literally.
	if !strings.Contains(got, "[Service]") {
		t.Errorf("drop-in missing [Service] section:\n%s", got)
	}
	if !strings.Contains(got, `Environment="TIDEWAYS_ENVIRONMENT=local_peterjaap"`) {
		t.Errorf("drop-in missing expected Environment= line:\n%s", got)
	}
	// Managed-by header so it's obvious who owns this file.
	if !strings.Contains(got, "Managed by MageBox") {
		t.Errorf("drop-in missing managed-by marker:\n%s", got)
	}
}

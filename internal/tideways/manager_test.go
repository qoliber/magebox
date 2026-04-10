// Copyright (c) qoliber

package tideways

import (
	"strings"
	"testing"
)

func TestRewriteIniDirective(t *testing.T) {
	tests := []struct {
		name      string
		existing  string
		directive string
		value     string
		want      string
	}{
		{
			name:      "appends api_key to file without directive",
			existing:  "; Tideways PHP extension\nextension=tideways.so\n",
			directive: "tideways.api_key",
			value:     "abc123",
			want:      "; Tideways PHP extension\nextension=tideways.so\ntideways.api_key=abc123\n",
		},
		{
			name:      "replaces existing uncommented api_key directive",
			existing:  "extension=tideways.so\ntideways.api_key=old\n",
			directive: "tideways.api_key",
			value:     "new",
			want:      "extension=tideways.so\ntideways.api_key=new\n",
		},
		{
			name:      "replaces existing commented api_key directive",
			existing:  "extension=tideways.so\n;tideways.api_key=old\n",
			directive: "tideways.api_key",
			value:     "new",
			want:      "extension=tideways.so\ntideways.api_key=new\n",
		},
		{
			name:      "replaces api_key directive with whitespace and comment marker",
			existing:  "extension=tideways.so\n  ; tideways.api_key=old\n",
			directive: "tideways.api_key",
			value:     "new",
			want:      "extension=tideways.so\ntideways.api_key=new\n",
		},
		{
			name:      "ensures trailing newline when input has none",
			existing:  "extension=tideways.so",
			directive: "tideways.api_key",
			value:     "k",
			want:      "extension=tideways.so\ntideways.api_key=k\n",
		},
		{
			name:      "handles empty file",
			existing:  "",
			directive: "tideways.api_key",
			value:     "k",
			want:      "tideways.api_key=k\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rewriteIniDirective(tt.existing, tt.directive, tt.value)
			if got != tt.want {
				t.Errorf("rewriteIniDirective(%q, %q, %q)\n got: %q\nwant: %q", tt.existing, tt.directive, tt.value, got, tt.want)
			}
		})
	}
}

// TestStripIniDirective verifies we can evict a stale directive left behind
// by an earlier MageBox version. Specifically, v1.14.x briefly wrote
// `tideways.environment` to the PHP extension ini before we learned that is
// a daemon-level setting, not an extension-level one.
func TestStripIniDirective(t *testing.T) {
	tests := []struct {
		name      string
		existing  string
		directive string
		want      string
	}{
		{
			name:      "strips uncommented stale environment line",
			existing:  "extension=tideways.so\ntideways.api_key=abc\ntideways.environment=local_x\n",
			directive: "tideways.environment",
			want:      "extension=tideways.so\ntideways.api_key=abc\n",
		},
		{
			name:      "strips commented stale environment line",
			existing:  "extension=tideways.so\n;tideways.environment=local_x\ntideways.api_key=abc\n",
			directive: "tideways.environment",
			want:      "extension=tideways.so\ntideways.api_key=abc\n",
		},
		{
			name:      "no-op when directive absent",
			existing:  "extension=tideways.so\ntideways.api_key=abc\n",
			directive: "tideways.environment",
			want:      "extension=tideways.so\ntideways.api_key=abc\n",
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

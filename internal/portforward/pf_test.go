// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package portforward

import (
	"fmt"
	"testing"
)

func TestNextAction(t *testing.T) {
	tests := []struct {
		name      string
		installed bool
		outdated  bool
		active    bool
		want      daemonAction
	}{
		{
			name:      "not installed requires bootstrap",
			installed: false,
			want:      actionBootstrapRequired,
		},
		{
			name:      "outdated daemon is upgraded even when active",
			installed: true,
			outdated:  true,
			active:    true,
			want:      actionUpgrade,
		},
		{
			name:      "outdated and inactive daemon is upgraded",
			installed: true,
			outdated:  true,
			active:    false,
			want:      actionUpgrade,
		},
		{
			name:      "current but inactive daemon is kickstarted",
			installed: true,
			outdated:  false,
			active:    false,
			want:      actionKickstart,
		},
		{
			name:      "current and active daemon needs nothing",
			installed: true,
			outdated:  false,
			active:    true,
			want:      actionNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nextAction(tt.installed, tt.outdated, tt.active)
			if got != tt.want {
				t.Errorf("nextAction(%v, %v, %v) = %v, want %v", tt.installed, tt.outdated, tt.active, got, tt.want)
			}
		})
	}
}

func TestPlistOutdated(t *testing.T) {
	current := fmt.Sprintf("<!-- MageBox-Version-%s -->\n<plist></plist>", launchDaemonVersion)

	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "current version marker is up to date",
			content: current,
			want:    false,
		},
		{
			name:    "old version marker is outdated",
			content: "<!-- MageBox-Version-4 -->\n<plist></plist>",
			want:    true,
		},
		{
			name:    "missing marker is outdated",
			content: "<plist></plist>",
			want:    true,
		},
		{
			name:    "empty content is outdated",
			content: "",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := plistOutdated([]byte(tt.content)); got != tt.want {
				t.Errorf("plistOutdated(%q) = %v, want %v", tt.content, got, tt.want)
			}
		})
	}
}

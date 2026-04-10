// Copyright (c) qoliber

package tideways

import "testing"

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
		{
			name:      "appends environment to file without directive",
			existing:  "extension=tideways.so\ntideways.api_key=abc\n",
			directive: "tideways.environment",
			value:     "local_peterjaap",
			want:      "extension=tideways.so\ntideways.api_key=abc\ntideways.environment=local_peterjaap\n",
		},
		{
			name:      "replaces existing environment directive",
			existing:  "extension=tideways.so\ntideways.environment=production\n",
			directive: "tideways.environment",
			value:     "local_peterjaap",
			want:      "extension=tideways.so\ntideways.environment=local_peterjaap\n",
		},
		{
			name:      "replaces commented environment directive",
			existing:  "extension=tideways.so\n; tideways.environment=staging\n",
			directive: "tideways.environment",
			value:     "local_peterjaap",
			want:      "extension=tideways.so\ntideways.environment=local_peterjaap\n",
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

// TestRewriteIniDirective_Composition exercises the realistic case of
// applying both tideways.api_key and tideways.environment in sequence, which
// is what WriteExtensionConfig does.
func TestRewriteIniDirective_Composition(t *testing.T) {
	existing := "; Tideways PHP extension\nextension=tideways.so\n"

	withKey := rewriteIniDirective(existing, "tideways.api_key", "abc123")
	withBoth := rewriteIniDirective(withKey, "tideways.environment", "local_peterjaap")

	want := "; Tideways PHP extension\nextension=tideways.so\ntideways.api_key=abc123\ntideways.environment=local_peterjaap\n"
	if withBoth != want {
		t.Errorf("composition result:\n got: %q\nwant: %q", withBoth, want)
	}

	// Re-applying with new values should replace both in place, not append.
	replaced := rewriteIniDirective(withBoth, "tideways.api_key", "xyz789")
	replaced = rewriteIniDirective(replaced, "tideways.environment", "local_other")

	want2 := "; Tideways PHP extension\nextension=tideways.so\ntideways.api_key=xyz789\ntideways.environment=local_other\n"
	if replaced != want2 {
		t.Errorf("replace-in-place result:\n got: %q\nwant: %q", replaced, want2)
	}
}

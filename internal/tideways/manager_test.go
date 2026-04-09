// Copyright (c) qoliber

package tideways

import "testing"

func TestRewriteAPIKeyIni(t *testing.T) {
	tests := []struct {
		name     string
		existing string
		apiKey   string
		want     string
	}{
		{
			name:     "appends to file without directive",
			existing: "; Tideways PHP extension\nextension=tideways.so\n",
			apiKey:   "abc123",
			want:     "; Tideways PHP extension\nextension=tideways.so\ntideways.api_key=abc123\n",
		},
		{
			name:     "replaces existing uncommented directive",
			existing: "extension=tideways.so\ntideways.api_key=old\n",
			apiKey:   "new",
			want:     "extension=tideways.so\ntideways.api_key=new\n",
		},
		{
			name:     "replaces existing commented directive",
			existing: "extension=tideways.so\n;tideways.api_key=old\n",
			apiKey:   "new",
			want:     "extension=tideways.so\ntideways.api_key=new\n",
		},
		{
			name:     "replaces directive with whitespace and comment marker",
			existing: "extension=tideways.so\n  ; tideways.api_key=old\n",
			apiKey:   "new",
			want:     "extension=tideways.so\ntideways.api_key=new\n",
		},
		{
			name:     "ensures trailing newline when input has none",
			existing: "extension=tideways.so",
			apiKey:   "k",
			want:     "extension=tideways.so\ntideways.api_key=k\n",
		},
		{
			name:     "handles empty file",
			existing: "",
			apiKey:   "k",
			want:     "tideways.api_key=k\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rewriteAPIKeyIni(tt.existing, tt.apiKey)
			if got != tt.want {
				t.Errorf("rewriteAPIKeyIni(%q, %q)\n got: %q\nwant: %q", tt.existing, tt.apiKey, got, tt.want)
			}
		})
	}
}

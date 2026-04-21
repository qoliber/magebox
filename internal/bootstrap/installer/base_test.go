// Copyright (c) qoliber

package installer

import (
	"reflect"
	"testing"
)

func TestFilterPackages(t *testing.T) {
	tests := []struct {
		name        string
		pkgs        []string
		available   map[string]bool
		wantKept    []string
		wantDropped []string
	}{
		{
			name:        "all available",
			pkgs:        []string{"php8.4-cli", "php8.4-opcache", "php8.4-fpm"},
			available:   map[string]bool{"php8.4-cli": true, "php8.4-opcache": true, "php8.4-fpm": true},
			wantKept:    []string{"php8.4-cli", "php8.4-opcache", "php8.4-fpm"},
			wantDropped: nil,
		},
		{
			name:        "opcache merged into core on 8.5",
			pkgs:        []string{"php8.5-fpm", "php8.5-cli", "php8.5-opcache", "php8.5-zip"},
			available:   map[string]bool{"php8.5-fpm": true, "php8.5-cli": true, "php8.5-zip": true},
			wantKept:    []string{"php8.5-fpm", "php8.5-cli", "php8.5-zip"},
			wantDropped: []string{"php8.5-opcache"},
		},
		{
			name:        "all missing",
			pkgs:        []string{"php9.0-cli"},
			available:   map[string]bool{},
			wantKept:    nil,
			wantDropped: []string{"php9.0-cli"},
		},
		{
			name:        "preserves order",
			pkgs:        []string{"a", "b", "c", "d"},
			available:   map[string]bool{"a": true, "c": true},
			wantKept:    []string{"a", "c"},
			wantDropped: []string{"b", "d"},
		},
		{
			name:        "empty input",
			pkgs:        nil,
			available:   map[string]bool{},
			wantKept:    nil,
			wantDropped: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pred := func(p string) bool { return tt.available[p] }
			kept, dropped := filterPackages(tt.pkgs, pred)
			if !reflect.DeepEqual(kept, tt.wantKept) {
				t.Errorf("kept = %v, want %v", kept, tt.wantKept)
			}
			if !reflect.DeepEqual(dropped, tt.wantDropped) {
				t.Errorf("dropped = %v, want %v", dropped, tt.wantDropped)
			}
		})
	}
}

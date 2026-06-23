package main

import "testing"

func TestDecideServiceUI(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
		running bool
		want    serviceUIDecision
	}{
		{"disabled and stopped", false, false, decisionNotEnabled},
		{"disabled but somehow running", false, true, decisionNotEnabled},
		{"enabled but stopped", true, false, decisionStart},
		{"enabled and running", true, true, decisionProceed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := decideServiceUI(tt.enabled, tt.running); got != tt.want {
				t.Errorf("decideServiceUI(%v, %v) = %v, want %v", tt.enabled, tt.running, got, tt.want)
			}
		})
	}
}

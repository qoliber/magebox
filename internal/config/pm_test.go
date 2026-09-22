package config

import "testing"

func intPtr(i int) *int { return &i }

func TestResolvePM_DefaultsWhenUnset(t *testing.T) {
	got, err := ResolvePM(nil, nil)
	if err != nil {
		t.Fatalf("ResolvePM(nil, nil) returned error: %v", err)
	}

	want := DefaultResolvedPM()
	if got != want {
		t.Errorf("ResolvePM(nil, nil) = %+v, want %+v", got, want)
	}
}

func TestResolvePM_ProjectOverridesGlobal(t *testing.T) {
	global := &PMConfig{Mode: PMModeOnDemand, MaxChildren: intPtr(20)}
	project := &PMConfig{MaxChildren: intPtr(8)}

	got, err := ResolvePM(global, project)
	if err != nil {
		t.Fatalf("ResolvePM returned error: %v", err)
	}

	// max_children comes from the project, mode falls through from global.
	if got.MaxChildren != 8 {
		t.Errorf("MaxChildren = %d, want 8", got.MaxChildren)
	}
	if got.Mode != PMModeOnDemand {
		t.Errorf("Mode = %q, want %q", got.Mode, PMModeOnDemand)
	}
}

func TestResolvePM_GlobalAppliesWhenProjectUnset(t *testing.T) {
	global := &PMConfig{Mode: PMModeStatic, MaxChildren: intPtr(4), MaxRequests: intPtr(200)}

	got, err := ResolvePM(global, nil)
	if err != nil {
		t.Fatalf("ResolvePM returned error: %v", err)
	}

	if got.Mode != PMModeStatic {
		t.Errorf("Mode = %q, want %q", got.Mode, PMModeStatic)
	}
	if got.MaxChildren != 4 {
		t.Errorf("MaxChildren = %d, want 4", got.MaxChildren)
	}
	if got.MaxRequests != 200 {
		t.Errorf("MaxRequests = %d, want 200", got.MaxRequests)
	}
}

// Lowering only max_children must not produce a config PHP-FPM rejects: the
// untouched spare-server defaults (4/12, start 8) would exceed it.
func TestResolvePM_ScalesDynamicDefaultsToFitMaxChildren(t *testing.T) {
	got, err := ResolvePM(nil, &PMConfig{MaxChildren: intPtr(6)})
	if err != nil {
		t.Fatalf("ResolvePM returned error: %v", err)
	}

	if got.MaxSpareServers > got.MaxChildren {
		t.Errorf("MaxSpareServers = %d, must not exceed MaxChildren = %d", got.MaxSpareServers, got.MaxChildren)
	}
	if got.StartServers > got.MaxSpareServers || got.StartServers < got.MinSpareServers {
		t.Errorf("StartServers = %d, want within [%d, %d]", got.StartServers, got.MinSpareServers, got.MaxSpareServers)
	}
	if err := got.Validate(); err != nil {
		t.Errorf("scaled result should be valid, got: %v", err)
	}
}

// Explicit values are never silently rewritten; a contradictory combination is
// reported instead.
func TestResolvePM_ExplicitValuesAreNotScaled(t *testing.T) {
	_, err := ResolvePM(nil, &PMConfig{
		MaxChildren:     intPtr(4),
		MaxSpareServers: intPtr(10),
	})
	if err == nil {
		t.Fatal("expected an error when max_spare_servers exceeds max_children, got nil")
	}
}

func TestResolvePM_OnDemandSkipsDynamicConstraints(t *testing.T) {
	got, err := ResolvePM(nil, &PMConfig{
		Mode:               PMModeOnDemand,
		MaxChildren:        intPtr(5),
		ProcessIdleTimeout: "30s",
	})
	if err != nil {
		t.Fatalf("ResolvePM returned error: %v", err)
	}

	if !got.IsOnDemand() {
		t.Errorf("IsOnDemand() = false, want true for mode %q", got.Mode)
	}
	if got.ProcessIdleTimeout != "30s" {
		t.Errorf("ProcessIdleTimeout = %q, want %q", got.ProcessIdleTimeout, "30s")
	}
}

func TestResolvePM_NormalisesMode(t *testing.T) {
	got, err := ResolvePM(nil, &PMConfig{Mode: "  OnDemand  "})
	if err != nil {
		t.Fatalf("ResolvePM returned error: %v", err)
	}
	if got.Mode != PMModeOnDemand {
		t.Errorf("Mode = %q, want %q", got.Mode, PMModeOnDemand)
	}
}

func TestResolvePM_Invalid(t *testing.T) {
	tests := []struct {
		name string
		pm   *PMConfig
	}{
		{"unknown mode", &PMConfig{Mode: "adaptive"}},
		{"zero max_children", &PMConfig{MaxChildren: intPtr(0)}},
		{"negative max_children", &PMConfig{MaxChildren: intPtr(-1)}},
		{"negative max_requests", &PMConfig{MaxRequests: intPtr(-5)}},
		{
			"min spare above max spare",
			&PMConfig{MinSpareServers: intPtr(9), MaxSpareServers: intPtr(3), StartServers: intPtr(3)},
		},
		{
			"start_servers outside spare range",
			&PMConfig{StartServers: intPtr(20), MinSpareServers: intPtr(2), MaxSpareServers: intPtr(6)},
		},
		{
			"bad idle timeout",
			&PMConfig{Mode: PMModeOnDemand, ProcessIdleTimeout: "soon"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ResolvePM(nil, tt.pm); err == nil {
				t.Errorf("ResolvePM(%+v) = nil error, want an error", tt.pm)
			}
		})
	}
}

func TestResolvePM_ValidIdleTimeoutFormats(t *testing.T) {
	for _, v := range []string{"10", "10s", "5m", "1h", "1d"} {
		t.Run(v, func(t *testing.T) {
			if _, err := ResolvePM(nil, &PMConfig{Mode: PMModeOnDemand, ProcessIdleTimeout: v}); err != nil {
				t.Errorf("ProcessIdleTimeout %q should be valid, got: %v", v, err)
			}
		})
	}
}

func TestMergePM_OverrideKeepsUnsetBaseFields(t *testing.T) {
	base := &PMConfig{Mode: PMModeDynamic, MaxChildren: intPtr(30), MaxRequests: intPtr(500)}
	override := &PMConfig{MaxChildren: intPtr(10)}

	merged := mergePM(base, override)

	if merged.Mode != PMModeDynamic {
		t.Errorf("Mode = %q, want %q", merged.Mode, PMModeDynamic)
	}
	if *merged.MaxChildren != 10 {
		t.Errorf("MaxChildren = %d, want 10", *merged.MaxChildren)
	}
	if *merged.MaxRequests != 500 {
		t.Errorf("MaxRequests = %d, want 500", *merged.MaxRequests)
	}
}

func TestMergePM_NilHandling(t *testing.T) {
	if merged := mergePM(nil, nil); merged != nil {
		t.Errorf("mergePM(nil, nil) = %+v, want nil", merged)
	}

	base := &PMConfig{MaxChildren: intPtr(7)}
	if merged := mergePM(base, nil); merged == nil || *merged.MaxChildren != 7 {
		t.Errorf("mergePM(base, nil) should preserve base, got %+v", merged)
	}
	if merged := mergePM(nil, base); merged == nil || *merged.MaxChildren != 7 {
		t.Errorf("mergePM(nil, override) should use override, got %+v", merged)
	}
}

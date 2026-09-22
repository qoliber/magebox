package config

import (
	"fmt"
	"regexp"
	"strings"
)

// Process manager modes supported by PHP-FPM.
const (
	PMModeStatic   = "static"
	PMModeDynamic  = "dynamic"
	PMModeOnDemand = "ondemand"
)

// Built-in process manager defaults. These match the values MageBox used
// before `pm` became configurable, so an absent `pm` block keeps the
// previous behavior.
const (
	DefaultPMMode               = PMModeDynamic
	DefaultPMMaxChildren        = 50
	DefaultPMStartServers       = 8
	DefaultPMMinSpareServers    = 4
	DefaultPMMaxSpareServers    = 12
	DefaultPMMaxRequests        = 1000
	DefaultPMProcessIdleTimeout = "10s"
)

// pmDurationPattern matches PHP-FPM duration values: a positive integer with
// an optional s/m/h/d suffix (a bare number means seconds).
var pmDurationPattern = regexp.MustCompile(`^[0-9]+[smhd]?$`)

// PMConfig configures the PHP-FPM process manager for a project's pool.
//
// Numeric fields are pointers so that "not set" is distinguishable from an
// explicit 0, which matters when layering project config over global
// defaults. Use Resolve to collapse the layers into concrete values.
type PMConfig struct {
	// Mode is the PHP-FPM process manager: "static", "dynamic" or "ondemand".
	Mode string `yaml:"mode,omitempty"`

	// MaxChildren is the maximum number of worker processes. Applies to all modes.
	MaxChildren *int `yaml:"max_children,omitempty"`

	// StartServers is the number of workers created on startup (dynamic only).
	StartServers *int `yaml:"start_servers,omitempty"`

	// MinSpareServers is the minimum number of idle workers (dynamic only).
	MinSpareServers *int `yaml:"min_spare_servers,omitempty"`

	// MaxSpareServers is the maximum number of idle workers (dynamic only).
	MaxSpareServers *int `yaml:"max_spare_servers,omitempty"`

	// MaxRequests is the number of requests a worker handles before respawning.
	// 0 disables respawning. Applies to all modes.
	MaxRequests *int `yaml:"max_requests,omitempty"`

	// ProcessIdleTimeout is how long an idle worker lives before being killed
	// (ondemand only). Accepts PHP-FPM duration syntax, e.g. "10s", "1m".
	ProcessIdleTimeout string `yaml:"process_idle_timeout,omitempty"`
}

// ResolvedPM holds concrete process manager values ready to render into a pool
// config. Every field is populated; there are no "unset" values.
type ResolvedPM struct {
	Mode               string
	MaxChildren        int
	StartServers       int
	MinSpareServers    int
	MaxSpareServers    int
	MaxRequests        int
	ProcessIdleTimeout string
}

// DefaultResolvedPM returns the built-in process manager settings, used when a
// project and the global config both leave `pm` unset.
func DefaultResolvedPM() ResolvedPM {
	return ResolvedPM{
		Mode:               DefaultPMMode,
		MaxChildren:        DefaultPMMaxChildren,
		StartServers:       DefaultPMStartServers,
		MinSpareServers:    DefaultPMMinSpareServers,
		MaxSpareServers:    DefaultPMMaxSpareServers,
		MaxRequests:        DefaultPMMaxRequests,
		ProcessIdleTimeout: DefaultPMProcessIdleTimeout,
	}
}

// IsDynamic reports whether the resolved mode uses the dynamic process manager.
func (r ResolvedPM) IsDynamic() bool { return r.Mode == PMModeDynamic }

// IsOnDemand reports whether the resolved mode uses the ondemand process manager.
func (r ResolvedPM) IsOnDemand() bool { return r.Mode == PMModeOnDemand }

// mergePM layers an override PMConfig on top of a base, field by field. Fields
// left unset in the override keep the base value. Either argument may be nil.
func mergePM(base, override *PMConfig) *PMConfig {
	if base == nil && override == nil {
		return nil
	}

	result := PMConfig{}
	if base != nil {
		result = *base
	}
	if override == nil {
		return &result
	}

	if override.Mode != "" {
		result.Mode = override.Mode
	}
	if override.MaxChildren != nil {
		result.MaxChildren = override.MaxChildren
	}
	if override.StartServers != nil {
		result.StartServers = override.StartServers
	}
	if override.MinSpareServers != nil {
		result.MinSpareServers = override.MinSpareServers
	}
	if override.MaxSpareServers != nil {
		result.MaxSpareServers = override.MaxSpareServers
	}
	if override.MaxRequests != nil {
		result.MaxRequests = override.MaxRequests
	}
	if override.ProcessIdleTimeout != "" {
		result.ProcessIdleTimeout = override.ProcessIdleTimeout
	}

	return &result
}

// ResolvePM collapses the global default and project-level process manager
// settings into concrete values, filling gaps with the built-in defaults.
// Precedence is project > global > built-in. Both arguments may be nil.
//
// The result is validated; an invalid configuration is returned as an error
// rather than written to a pool file, because PHP-FPM refuses to start the
// entire master process when a single pool is malformed.
func ResolvePM(global, project *PMConfig) (ResolvedPM, error) {
	merged := mergePM(global, project)
	resolved := DefaultResolvedPM()

	if merged == nil {
		return resolved, nil
	}

	if merged.Mode != "" {
		resolved.Mode = strings.ToLower(strings.TrimSpace(merged.Mode))
	}
	if merged.MaxChildren != nil {
		resolved.MaxChildren = *merged.MaxChildren
	}
	if merged.StartServers != nil {
		resolved.StartServers = *merged.StartServers
	}
	if merged.MinSpareServers != nil {
		resolved.MinSpareServers = *merged.MinSpareServers
	}
	if merged.MaxSpareServers != nil {
		resolved.MaxSpareServers = *merged.MaxSpareServers
	}
	if merged.MaxRequests != nil {
		resolved.MaxRequests = *merged.MaxRequests
	}
	if merged.ProcessIdleTimeout != "" {
		resolved.ProcessIdleTimeout = strings.TrimSpace(merged.ProcessIdleTimeout)
	}

	// When only max_children is lowered, the dynamic spare-server defaults can
	// exceed it and PHP-FPM would refuse to start. Scale the untouched spare
	// values down to fit rather than making the user restate all four.
	if resolved.IsDynamic() && resolved.MaxChildren > 0 {
		if merged.MaxSpareServers == nil && resolved.MaxSpareServers > resolved.MaxChildren {
			resolved.MaxSpareServers = resolved.MaxChildren
		}
		if merged.MinSpareServers == nil && resolved.MinSpareServers > resolved.MaxSpareServers {
			resolved.MinSpareServers = resolved.MaxSpareServers
		}
		if merged.StartServers == nil {
			if resolved.StartServers > resolved.MaxSpareServers {
				resolved.StartServers = resolved.MaxSpareServers
			}
			if resolved.StartServers < resolved.MinSpareServers {
				resolved.StartServers = resolved.MinSpareServers
			}
		}
	}

	if err := resolved.Validate(); err != nil {
		return ResolvedPM{}, err
	}

	return resolved, nil
}

// Validate checks the resolved settings against the constraints PHP-FPM
// enforces at startup. It mirrors those rules so misconfiguration surfaces as
// a MageBox error instead of a failed FPM master that takes down every project
// sharing the same PHP version.
func (r ResolvedPM) Validate() error {
	switch r.Mode {
	case PMModeStatic, PMModeDynamic, PMModeOnDemand:
	default:
		return fmt.Errorf("pm.mode %q is invalid: must be %q, %q or %q",
			r.Mode, PMModeStatic, PMModeDynamic, PMModeOnDemand)
	}

	if r.MaxChildren < 1 {
		return fmt.Errorf("pm.max_children must be at least 1, got %d", r.MaxChildren)
	}

	if r.MaxRequests < 0 {
		return fmt.Errorf("pm.max_requests cannot be negative, got %d", r.MaxRequests)
	}

	if r.IsOnDemand() && !pmDurationPattern.MatchString(r.ProcessIdleTimeout) {
		return fmt.Errorf("pm.process_idle_timeout %q is invalid: expected a number with an optional s/m/h/d suffix, e.g. \"10s\"",
			r.ProcessIdleTimeout)
	}

	if !r.IsDynamic() {
		return nil
	}

	if r.StartServers < 1 {
		return fmt.Errorf("pm.start_servers must be at least 1 when pm.mode is %q, got %d", PMModeDynamic, r.StartServers)
	}
	if r.MinSpareServers < 1 {
		return fmt.Errorf("pm.min_spare_servers must be at least 1 when pm.mode is %q, got %d", PMModeDynamic, r.MinSpareServers)
	}
	if r.MaxSpareServers < 1 {
		return fmt.Errorf("pm.max_spare_servers must be at least 1 when pm.mode is %q, got %d", PMModeDynamic, r.MaxSpareServers)
	}
	if r.MinSpareServers > r.MaxSpareServers {
		return fmt.Errorf("pm.min_spare_servers (%d) cannot exceed pm.max_spare_servers (%d)", r.MinSpareServers, r.MaxSpareServers)
	}
	if r.MaxSpareServers > r.MaxChildren {
		return fmt.Errorf("pm.max_spare_servers (%d) cannot exceed pm.max_children (%d)", r.MaxSpareServers, r.MaxChildren)
	}
	if r.StartServers < r.MinSpareServers || r.StartServers > r.MaxSpareServers {
		return fmt.Errorf("pm.start_servers (%d) must be between pm.min_spare_servers (%d) and pm.max_spare_servers (%d)",
			r.StartServers, r.MinSpareServers, r.MaxSpareServers)
	}

	return nil
}

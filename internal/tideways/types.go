// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package tideways

// Credentials contains Tideways API credentials
type Credentials struct {
	APIKey string `yaml:"api_key"`
}

// Status represents the current Tideways status
type Status struct {
	DaemonInstalled    bool
	DaemonRunning      bool
	ExtensionInstalled map[string]bool // PHP version -> installed
	ExtensionEnabled   map[string]bool // PHP version -> enabled
	Configured         bool
}

// IsFullyConfigured returns true if Tideways is fully set up
func (s *Status) IsFullyConfigured() bool {
	return s.DaemonInstalled && s.DaemonRunning && s.Configured
}

// HasAnyExtension returns true if any PHP version has the extension installed
func (s *Status) HasAnyExtension() bool {
	for _, installed := range s.ExtensionInstalled {
		if installed {
			return true
		}
	}
	return false
}

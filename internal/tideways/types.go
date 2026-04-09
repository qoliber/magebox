// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package tideways

// Credentials contains Tideways credentials.
//
// APIKey is the project-level "API Key" used by the Tideways PHP extension
// (written to php.ini as tideways.api_key).
//
// AccessToken is the personal CLI token used by the `tideways` commandline
// tool (imported via `tideways import <token>`). It is a separate credential
// from APIKey.
type Credentials struct {
	APIKey      string `yaml:"api_key"`
	AccessToken string `yaml:"access_token"`
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

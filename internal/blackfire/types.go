// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package blackfire

// Credentials contains Blackfire API credentials
type Credentials struct {
	ServerID    string `yaml:"server_id"`
	ServerToken string `yaml:"server_token"`
	ClientID    string `yaml:"client_id"`
	ClientToken string `yaml:"client_token"`
}

// Status represents the current Blackfire status
type Status struct {
	AgentInstalled     bool
	AgentRunning       bool
	ExtensionInstalled map[string]bool // PHP version -> installed
	ExtensionEnabled   map[string]bool // PHP version -> enabled
	Configured         bool
}

// IsFullyConfigured returns true if Blackfire is fully set up
func (s *Status) IsFullyConfigured() bool {
	return s.AgentInstalled && s.AgentRunning && s.Configured
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

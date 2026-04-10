// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package tideways

// Credentials contains the globally-scoped Tideways credentials.
//
// AccessToken is the personal CLI token used by the `tideways` commandline
// tool (imported via `tideways import <token>`). It is scoped to a user's
// Tideways account and reused across projects.
//
// Environment is the free-text label the local tideways-daemon stamps onto
// every trace it forwards. It is a daemon-level setting so it is machine-
// wide and lives in the global config.
//
// The Tideways API key is *not* part of this struct. It is per Tideways
// project and lives in each project's .magebox.yaml under
// php_ini.tideways.api_key, which MageBox renders into that project's FPM
// pool config as a php_admin_value.
type Credentials struct {
	AccessToken string `yaml:"access_token"`
	Environment string `yaml:"environment"`
}

// Status represents the current Tideways status
type Status struct {
	DaemonInstalled    bool
	DaemonRunning      bool
	ExtensionInstalled map[string]bool // PHP version -> installed
	ExtensionEnabled   map[string]bool // PHP version -> enabled
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

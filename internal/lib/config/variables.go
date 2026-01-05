// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package config

import (
	"os"
	"os/user"
	"regexp"
	"runtime"
	"strings"
)

// Variables holds all substitution variables
type Variables struct {
	values map[string]string
}

// NewVariables creates a new Variables instance with default values
func NewVariables() *Variables {
	v := &Variables{
		values: make(map[string]string),
	}
	v.initDefaults()
	return v
}

// initDefaults sets up default variable values
func (v *Variables) initDefaults() {
	// User info
	currentUser, _ := user.Current()
	if currentUser != nil {
		v.values["user"] = currentUser.Username
		v.values["homeDir"] = currentUser.HomeDir
		v.values["mageboxDir"] = currentUser.HomeDir + "/.magebox"
	}

	// Platform
	v.values["platform"] = runtime.GOOS
	v.values["arch"] = runtime.GOARCH

	// OS Version (will be set by caller based on detection)
	v.values["osVersion"] = ""

	// Default TLD
	v.values["tld"] = "test"
}

// Set sets a variable value
func (v *Variables) Set(name, value string) {
	v.values[name] = value
}

// Get returns a variable value
func (v *Variables) Get(name string) string {
	return v.values[name]
}

// SetPHPVersion sets PHP version and derived variables
func (v *Variables) SetPHPVersion(version string) {
	v.values["phpVersion"] = version
	v.values["versionNoDot"] = strings.ReplaceAll(version, ".", "")
	// phpPrefix is set based on version_format from config
}

// SetPHPPrefix sets the PHP prefix (computed from version_format)
func (v *Variables) SetPHPPrefix(prefix string) {
	v.values["phpPrefix"] = prefix
}

// SetServiceName sets the current service name (for service commands)
func (v *Variables) SetServiceName(name string) {
	v.values["serviceName"] = name
}

// SetOSVersion sets the OS version
func (v *Variables) SetOSVersion(version string) {
	v.values["osVersion"] = version
}

// SetTLD sets the top-level domain
func (v *Variables) SetTLD(tld string) {
	v.values["tld"] = tld
}

// SetBrewPrefix sets the Homebrew prefix (macOS)
func (v *Variables) SetBrewPrefix(prefix string) {
	v.values["brewPrefix"] = prefix
}

// SetPeclBin sets the PECL binary path (macOS)
func (v *Variables) SetPeclBin(path string) {
	v.values["peclBin"] = path
}

// SetSELinuxContext sets SELinux context variables
func (v *Variables) SetSELinuxContext(path, typ, pattern string) {
	v.values["path"] = path
	v.values["type"] = typ
	v.values["pattern"] = pattern
}

// SetPackages sets packages variable (for install commands)
func (v *Variables) SetPackages(packages string) {
	v.values["packages"] = packages
}

// Expand replaces all ${variable} patterns in the string
func (v *Variables) Expand(s string) string {
	// Match ${varname} pattern
	re := regexp.MustCompile(`\$\{([a-zA-Z_][a-zA-Z0-9_]*)\}`)

	return re.ReplaceAllStringFunc(s, func(match string) string {
		// Extract variable name from ${name}
		name := match[2 : len(match)-1]
		if val, ok := v.values[name]; ok {
			return val
		}
		// Try environment variable as fallback
		if val := os.Getenv(name); val != "" {
			return val
		}
		// Return original if not found
		return match
	})
}

// ExpandSlice expands variables in a slice of strings
func (v *Variables) ExpandSlice(slice []string) []string {
	result := make([]string, len(slice))
	for i, s := range slice {
		result[i] = v.Expand(s)
	}
	return result
}

// ExpandMap expands variables in a map of strings
func (v *Variables) ExpandMap(m map[string]string) map[string]string {
	result := make(map[string]string)
	for k, val := range m {
		result[k] = v.Expand(val)
	}
	return result
}

// Clone returns a copy of the Variables
func (v *Variables) Clone() *Variables {
	clone := &Variables{
		values: make(map[string]string),
	}
	for k, val := range v.values {
		clone.values[k] = val
	}
	return clone
}

// All returns all variable values
func (v *Variables) All() map[string]string {
	result := make(map[string]string)
	for k, val := range v.values {
		result[k] = val
	}
	return result
}

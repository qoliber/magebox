package sandbox

import "qoliber/magebox/internal/config"

// ToolProfile holds the resolved sandbox profile for a tool
type ToolProfile struct {
	Command     string
	Args        []string
	ConfigDirs  []string
	ConfigFiles []string
}

// defaultProfiles returns the built-in tool profiles
func defaultProfiles() map[string]ToolProfile {
	return map[string]ToolProfile{
		"claude": {
			Command:     "claude",
			Args:        []string{"--dangerously-skip-permissions"},
			ConfigDirs:  []string{"~/.claude", "~/.cache"},
			ConfigFiles: []string{"~/.claude.json"},
		},
		"codex": {
			Command:    "codex",
			Args:       []string{},
			ConfigDirs: []string{"~/.codex", "~/.cache"},
		},
	}
}

// ResolveProfile returns a merged tool profile for the given tool name,
// combining built-in defaults with any config overrides.
func ResolveProfile(name string, cfg *config.SandboxConfig) ToolProfile {
	defaults := defaultProfiles()
	profile, ok := defaults[name]
	if !ok {
		// Unknown tool: use the name as command with no special config
		profile = ToolProfile{
			Command: name,
		}
	}

	if cfg == nil {
		return profile
	}

	cfgProfiles := cfg.ToolProfiles
	if cfgProfiles == nil {
		return profile
	}

	override, ok := cfgProfiles[name]
	if !ok {
		return profile
	}

	if override.Command != "" {
		profile.Command = override.Command
	}
	if len(override.Args) > 0 {
		profile.Args = override.Args
	}
	if len(override.ConfigDirs) > 0 {
		profile.ConfigDirs = override.ConfigDirs
	}
	if len(override.ConfigFiles) > 0 {
		profile.ConfigFiles = override.ConfigFiles
	}

	return profile
}

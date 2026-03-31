package sandbox

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"qoliber/magebox/internal/config"
)

func TestBuildArgs_DefaultClaude(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()

	// Create paths that the builder checks for
	mustCreate(t, filepath.Join(home, ".claude"))
	mustCreate(t, filepath.Join(home, ".cache"))
	mustCreateFile(t, filepath.Join(home, ".claude.json"))
	mustCreateFile(t, filepath.Join(home, ".bashrc"))
	mustCreateFile(t, filepath.Join(home, ".gitconfig"))

	mgr := NewManager(home, project)
	profile := ResolveProfile("claude", nil)
	opts := Options{Profile: profile}

	args := mgr.BuildArgs("claude", nil, opts)
	joined := strings.Join(args, " ")

	// Should have basic sandbox setup
	assertContains(t, joined, "--tmpfs /tmp")
	assertContains(t, joined, "--dev /dev")
	assertContains(t, joined, "--proc /proc")
	assertContains(t, joined, "--hostname magebox-sandbox --unshare-uts")

	// Should bind project dir read-write
	assertContains(t, joined, "--bind "+project+" "+project)

	// Should bind ~/.claude read-write
	assertContains(t, joined, "--bind "+filepath.Join(home, ".claude")+" "+filepath.Join(home, ".claude"))

	// Should bind ~/.cache read-write
	assertContains(t, joined, "--bind "+filepath.Join(home, ".cache")+" "+filepath.Join(home, ".cache"))

	// Should bind ~/.claude.json read-write
	assertContains(t, joined, "--bind "+filepath.Join(home, ".claude.json")+" "+filepath.Join(home, ".claude.json"))

	// Should bind user configs read-only
	assertContains(t, joined, "--ro-bind "+filepath.Join(home, ".bashrc")+" "+filepath.Join(home, ".bashrc"))
	assertContains(t, joined, "--ro-bind "+filepath.Join(home, ".gitconfig")+" "+filepath.Join(home, ".gitconfig"))

	// Should end with claude --dangerously-skip-permissions
	assertContains(t, joined, "claude --dangerously-skip-permissions")
}

func TestBuildArgs_SkipsNonExistentPaths(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()

	// Don't create any user config files
	mgr := NewManager(home, project)
	profile := ToolProfile{
		Command:    "claude",
		ConfigDirs: []string{"~/.nonexistent-dir"},
	}
	opts := Options{Profile: profile}

	args := mgr.BuildArgs("claude", nil, opts)
	joined := strings.Join(args, " ")

	// Should NOT contain nonexistent paths
	if strings.Contains(joined, ".nonexistent-dir") {
		t.Error("should skip non-existent config dirs")
	}
	if strings.Contains(joined, ".bashrc") {
		t.Error("should skip non-existent user config files")
	}
}

func TestBuildArgs_CustomToolArgs(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()

	mgr := NewManager(home, project)
	profile := ToolProfile{
		Command: "claude",
		Args:    []string{"--dangerously-skip-permissions"},
	}
	opts := Options{Profile: profile}

	// When explicit args are provided, they should replace defaults
	args := mgr.BuildArgs("claude", []string{"--resume"}, opts)
	joined := strings.Join(args, " ")

	assertContains(t, joined, "claude --resume")
	if strings.Contains(joined, "--dangerously-skip-permissions") {
		t.Error("explicit tool args should override default args")
	}
}

func TestBuildArgs_ExtraBinds(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()

	// Create the extra bind path
	sshDir := filepath.Join(home, ".ssh")
	mustCreate(t, sshDir)
	composerDir := filepath.Join(home, ".composer")
	mustCreate(t, composerDir)

	mgr := NewManager(home, project)
	profile := ToolProfile{Command: "claude"}
	opts := Options{
		Profile:      profile,
		ExtraROBinds: []string{"~/.ssh"},
		ExtraBinds:   []string{"~/.composer"},
	}

	args := mgr.BuildArgs("claude", nil, opts)
	joined := strings.Join(args, " ")

	assertContains(t, joined, "--ro-bind "+sshDir+" "+sshDir)
	assertContains(t, joined, "--bind "+composerDir+" "+composerDir)
}

func TestResolveProfile_Defaults(t *testing.T) {
	profile := ResolveProfile("claude", nil)

	if profile.Command != "claude" {
		t.Errorf("expected command 'claude', got %q", profile.Command)
	}
	if len(profile.Args) != 1 || profile.Args[0] != "--dangerously-skip-permissions" {
		t.Errorf("unexpected default args: %v", profile.Args)
	}
}

func TestResolveProfile_UnknownTool(t *testing.T) {
	profile := ResolveProfile("my-custom-tool", nil)

	if profile.Command != "my-custom-tool" {
		t.Errorf("expected command 'my-custom-tool', got %q", profile.Command)
	}
	if len(profile.Args) != 0 {
		t.Errorf("unknown tool should have no default args, got %v", profile.Args)
	}
}

func TestResolveProfile_ConfigOverride(t *testing.T) {
	cfg := &config.SandboxConfig{
		ToolProfiles: map[string]config.SandboxToolProfile{
			"claude": {
				Args: []string{"--model", "opus"},
			},
		},
	}

	profile := ResolveProfile("claude", cfg)

	if profile.Command != "claude" {
		t.Errorf("command should remain 'claude', got %q", profile.Command)
	}
	if len(profile.Args) != 2 || profile.Args[0] != "--model" {
		t.Errorf("args should be overridden, got %v", profile.Args)
	}
	// ConfigDirs should still have defaults since not overridden
	if len(profile.ConfigDirs) != 2 {
		t.Errorf("config dirs should retain defaults, got %v", profile.ConfigDirs)
	}
}

func TestMergeSandboxConfigs(t *testing.T) {
	global := &config.SandboxConfig{
		DefaultTool:  "claude",
		ExtraROBinds: []string{"~/.ssh"},
	}
	project := &config.SandboxConfig{
		DefaultTool: "codex",
		ExtraBinds:  []string{"~/.composer"},
	}

	merged := MergeSandboxConfigs(global, project)

	if merged.DefaultTool != "codex" {
		t.Errorf("project default_tool should override global, got %q", merged.DefaultTool)
	}
	if len(merged.ExtraROBinds) != 1 || merged.ExtraROBinds[0] != "~/.ssh" {
		t.Errorf("global ro binds should carry over, got %v", merged.ExtraROBinds)
	}
	if len(merged.ExtraBinds) != 1 || merged.ExtraBinds[0] != "~/.composer" {
		t.Errorf("project binds should carry over, got %v", merged.ExtraBinds)
	}
}

func TestMergeSandboxConfigs_BothNil(t *testing.T) {
	merged := MergeSandboxConfigs(nil, nil)
	if merged == nil {
		t.Error("merged config should not be nil")
	}
}

func TestExpandHome(t *testing.T) {
	tests := []struct {
		input    string
		home     string
		expected string
	}{
		{"~/.claude", "/home/user", "/home/user/.claude"},
		{"~", "/home/user", "/home/user"},
		{"/etc/hosts", "/home/user", "/etc/hosts"},
		{"relative/path", "/home/user", "relative/path"},
	}

	for _, tt := range tests {
		result := expandHome(tt.input, tt.home)
		if result != tt.expected {
			t.Errorf("expandHome(%q, %q) = %q, want %q", tt.input, tt.home, result, tt.expected)
		}
	}
}

// Helpers

func mustCreate(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatalf("failed to create dir %s: %v", path, err)
	}
}

func mustCreateFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte{}, 0644); err != nil {
		t.Fatalf("failed to create file %s: %v", path, err)
	}
}

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("expected to contain %q, got:\n%s", needle, haystack)
	}
}

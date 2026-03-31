package sandbox

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"qoliber/magebox/internal/config"
	"qoliber/magebox/internal/verbose"
)

const bwrapBinary = "bwrap"

// systemROBinds are system paths mounted read-only in the sandbox.
// Paths that don't exist on the host are silently skipped.
var systemROBinds = []string{
	"/bin",
	"/lib",
	"/lib32",
	"/lib64",
	"/usr/bin",
	"/usr/lib",
	"/usr/local/bin",
	"/usr/local/lib",
	"/etc/alternatives",
	"/etc/resolv.conf",
	"/etc/ssl/certs",
	"/etc/ld.so.cache",
	"/etc/ld.so.conf",
	"/etc/ld.so.conf.d",
	"/etc/localtime",
	"/etc/profile.d",
	"/etc/bash_completion.d",
	"/etc/nsswitch.conf",
	"/etc/hosts",
	"/etc/ssl/openssl.cnf",
	"/usr/share/terminfo",
	"/usr/share/ca-certificates",
	"/usr/share/zoneinfo",
}

// userROBinds are user home paths mounted read-only (relative to ~).
var userROBinds = []string{
	".bashrc",
	".zshrc",
	".profile",
	".gitconfig",
	".local",
}

// Manager handles sandbox execution
type Manager struct {
	homeDir    string
	projectDir string
}

// NewManager creates a new sandbox manager
func NewManager(homeDir, projectDir string) *Manager {
	return &Manager{homeDir: homeDir, projectDir: projectDir}
}

// CheckAvailable verifies bwrap is installed and that user namespaces are permitted
func (m *Manager) CheckAvailable() error {
	bwrapPath, err := exec.LookPath(bwrapBinary)
	if err != nil {
		return fmt.Errorf("bubblewrap (bwrap) is not installed\n\n" +
			"Install it with:\n" +
			"  Debian/Ubuntu:  sudo apt install bubblewrap\n" +
			"  Fedora/RHEL:    sudo dnf install bubblewrap\n" +
			"  Arch:           sudo pacman -S bubblewrap")
	}

	// Quick smoke test: try a minimal bwrap invocation to catch permission issues early
	test := exec.Command(bwrapPath, "--ro-bind", "/", "/", "true")
	if output, err := test.CombinedOutput(); err != nil {
		out := strings.TrimSpace(string(output))
		if strings.Contains(out, "Permission denied") && isAppArmorRestrictingUserns() {
			return fmt.Errorf("bubblewrap cannot create user namespaces (blocked by AppArmor)\n\n"+
				"Fix by creating an AppArmor profile for bwrap:\n\n"+
				"  sudo tee /etc/apparmor.d/bwrap <<'EOF'\n"+
				"  abi <abi/4.0>,\n"+
				"  include <tunables/global>\n\n"+
				"  profile bwrap %s flags=(unconfined) {\n"+
				"    userns,\n"+
				"    include if exists <local/bwrap>\n"+
				"  }\n"+
				"  EOF\n\n"+
				"  sudo apparmor_parser -r /etc/apparmor.d/bwrap", bwrapPath)
		}
		return fmt.Errorf("bubblewrap test failed: %s\n%s", err, out)
	}

	return nil
}

// isAppArmorRestrictingUserns checks if AppArmor is blocking unprivileged user namespaces
func isAppArmorRestrictingUserns() bool {
	data, err := os.ReadFile("/proc/sys/kernel/apparmor_restrict_unprivileged_userns")
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(data)) == "1"
}

// Options holds the resolved options for a sandbox run
type Options struct {
	Profile      ToolProfile
	ExtraROBinds []string
	ExtraBinds   []string
}

// BuildArgs constructs the full bwrap argument list.
// The returned slice does NOT include the bwrap binary itself.
func (m *Manager) BuildArgs(tool string, toolArgs []string, opts Options) []string {
	var args []string

	// Filesystem setup
	args = append(args, "--tmpfs", "/tmp")
	args = append(args, "--dev", "/dev")
	args = append(args, "--proc", "/proc")

	// Hostname isolation
	args = append(args, "--hostname", "magebox-sandbox", "--unshare-uts")

	// System read-only binds
	for _, p := range systemROBinds {
		if pathExists(p) {
			args = append(args, "--ro-bind", p, p)
		}
	}

	// User read-only binds
	for _, rel := range userROBinds {
		p := filepath.Join(m.homeDir, rel)
		if pathExists(p) {
			args = append(args, "--ro-bind", p, p)
		}
	}

	// Tool config dirs (read-write)
	for _, dir := range opts.Profile.ConfigDirs {
		expanded := expandHome(dir, m.homeDir)
		if pathExists(expanded) {
			args = append(args, "--bind", expanded, expanded)
		}
	}

	// Tool config files (read-write)
	for _, file := range opts.Profile.ConfigFiles {
		expanded := expandHome(file, m.homeDir)
		if pathExists(expanded) {
			args = append(args, "--bind", expanded, expanded)
		}
	}

	// Extra read-only binds from config
	for _, bind := range opts.ExtraROBinds {
		expanded := expandHome(bind, m.homeDir)
		if pathExists(expanded) {
			args = append(args, "--ro-bind", expanded, expanded)
		}
	}

	// Extra read-write binds from config
	for _, bind := range opts.ExtraBinds {
		expanded := expandHome(bind, m.homeDir)
		if pathExists(expanded) {
			args = append(args, "--bind", expanded, expanded)
		}
	}

	// Project directory (read-write)
	args = append(args, "--bind", m.projectDir, m.projectDir)

	// Separator between bwrap flags and the tool command
	args = append(args, "--")

	// Tool command and arguments
	toolBin := opts.Profile.Command
	if toolBin == "" {
		toolBin = tool
	}
	args = append(args, toolBin)

	// Default tool args (unless overridden by explicit toolArgs)
	if len(toolArgs) == 0 {
		args = append(args, opts.Profile.Args...)
	} else {
		args = append(args, toolArgs...)
	}

	return args
}

// FormatCommand returns the full bwrap command as a string for dry-run display.
// Groups flag arguments together on the same line for readability.
func (m *Manager) FormatCommand(tool string, toolArgs []string, opts Options) string {
	args := m.BuildArgs(tool, toolArgs, opts)

	var lines []string
	lines = append(lines, bwrapBinary)

	i := 0
	for i < len(args) {
		arg := args[i]

		// "--" separator marks the end of bwrap flags
		if arg == "--" {
			i++
			// Everything after "--" is the tool command + args
			var cmdParts []string
			for i < len(args) {
				val := args[i]
				if strings.Contains(val, " ") {
					val = fmt.Sprintf("%q", val)
				}
				cmdParts = append(cmdParts, val)
				i++
			}
			lines = append(lines, "    "+strings.Join(cmdParts, " "))
			break
		}

		if strings.HasPrefix(arg, "--") {
			// Collect the flag and its arguments on one line
			line := "    " + arg
			i++
			for i < len(args) && !strings.HasPrefix(args[i], "--") {
				val := args[i]
				if strings.Contains(val, " ") {
					val = fmt.Sprintf("%q", val)
				}
				line += " " + val
				i++
			}
			lines = append(lines, line)
		} else {
			i++
		}
	}

	return strings.Join(lines, " \\\n")
}

// Run executes the tool inside the bubblewrap sandbox
func (m *Manager) Run(tool string, toolArgs []string, opts Options) error {
	bwrapPath, err := exec.LookPath(bwrapBinary)
	if err != nil {
		return fmt.Errorf("bwrap not found: %w", err)
	}

	args := m.BuildArgs(tool, toolArgs, opts)
	verbose.Debug("Running: %s %s", bwrapPath, strings.Join(args, " "))

	cmd := exec.Command(bwrapPath, args...)
	cmd.Dir = m.projectDir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	return cmd.Run()
}

// MergeSandboxConfigs merges global and project sandbox configs.
// Project config values take precedence.
func MergeSandboxConfigs(global, project *config.SandboxConfig) *config.SandboxConfig {
	if global == nil && project == nil {
		return &config.SandboxConfig{}
	}
	if global == nil {
		return project
	}
	if project == nil {
		return global
	}

	merged := &config.SandboxConfig{
		DefaultTool:  global.DefaultTool,
		ExtraROBinds: append([]string{}, global.ExtraROBinds...),
		ExtraBinds:   append([]string{}, global.ExtraBinds...),
		ToolProfiles: make(map[string]config.SandboxToolProfile),
	}

	// Project default tool overrides global
	if project.DefaultTool != "" {
		merged.DefaultTool = project.DefaultTool
	}

	// Accumulate extra binds from both
	merged.ExtraROBinds = append(merged.ExtraROBinds, project.ExtraROBinds...)
	merged.ExtraBinds = append(merged.ExtraBinds, project.ExtraBinds...)

	// Merge tool profiles
	for k, v := range global.ToolProfiles {
		merged.ToolProfiles[k] = v
	}
	for k, v := range project.ToolProfiles {
		merged.ToolProfiles[k] = v
	}

	return merged
}

// pathExists returns true if the path exists on the filesystem
func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// expandHome expands a leading ~ to the given home directory
func expandHome(path, homeDir string) string {
	if path == "~" {
		return homeDir
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(homeDir, path[2:])
	}
	return path
}

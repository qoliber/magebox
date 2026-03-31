package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"qoliber/magebox/internal/cli"
	"qoliber/magebox/internal/config"
	"qoliber/magebox/internal/platform"
	"qoliber/magebox/internal/sandbox"
)

var sandboxDryRun bool

var sandboxCmd = &cobra.Command{
	Use:   "sandbox [tool] [-- tool-args...]",
	Short: "Run an AI coding agent in a sandboxed environment",
	Long: `Launches an AI coding agent inside a bubblewrap (bwrap) sandbox.

The sandbox restricts filesystem access to:
  - Read-only: system binaries, libraries, and certificates
  - Read-write: current project directory and tool config dirs

This prevents AI agents from accessing or modifying files outside the project.

Requires bubblewrap to be installed (Linux only).

Examples:
  magebox sandbox                             # Launch claude (default) in sandbox
  magebox sandbox claude                      # Same as above, explicit
  magebox sandbox codex                       # Launch codex in sandbox
  magebox sandbox claude --resume             # Pass extra args to claude
  magebox sandbox --dry-run                   # Print the bwrap command without running it`,
	RunE:               runSandbox,
	Args:               cobra.ArbitraryArgs,
	Aliases:            []string{"sb"},
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
}

func init() {
	sandboxCmd.Flags().BoolVar(&sandboxDryRun, "dry-run", false, "Print the bwrap command without executing it")
	rootCmd.AddCommand(sandboxCmd)
}

func runSandbox(cmd *cobra.Command, args []string) error {
	// Platform check - Linux only
	p, err := getPlatform()
	if err != nil {
		return err
	}
	if p.Type != platform.Linux {
		return fmt.Errorf("the sandbox command requires Linux (bubblewrap is a Linux-specific technology)")
	}

	cwd, err := getCwd()
	if err != nil {
		return err
	}

	mgr := sandbox.NewManager(p.HomeDir, cwd)

	// Check bwrap is installed and working (skip for dry-run)
	if !sandboxDryRun {
		if err := mgr.CheckAvailable(); err != nil {
			return err
		}
	}

	// Load config (global + project, both optional)
	homeDir, _ := os.UserHomeDir()
	globalCfg, _ := config.LoadGlobalConfig(homeDir)
	projectCfg, _ := config.LoadFromPath(cwd)

	var globalSandbox, projectSandbox *config.SandboxConfig
	if globalCfg != nil {
		globalSandbox = globalCfg.Sandbox
	}
	if projectCfg != nil {
		projectSandbox = projectCfg.Sandbox
	}
	sandboxCfg := sandbox.MergeSandboxConfigs(globalSandbox, projectSandbox)

	// Determine tool name and args
	toolName := "claude"
	if sandboxCfg.DefaultTool != "" {
		toolName = sandboxCfg.DefaultTool
	}
	var toolArgs []string

	if len(args) > 0 {
		if strings.HasPrefix(args[0], "-") {
			// First arg is a flag — pass all args to the default tool
			toolArgs = args
		} else {
			toolName = args[0]
			toolArgs = args[1:]
		}
	}

	// Resolve tool profile
	profile := sandbox.ResolveProfile(toolName, sandboxCfg)

	opts := sandbox.Options{
		Profile:      profile,
		ExtraROBinds: sandboxCfg.ExtraROBinds,
		ExtraBinds:   sandboxCfg.ExtraBinds,
	}

	if sandboxDryRun {
		cli.PrintInfo("Dry run — bwrap command:")
		fmt.Println()
		fmt.Println(mgr.FormatCommand(toolName, toolArgs, opts))
		return nil
	}

	cli.PrintInfo("Launching %s in bubblewrap sandbox...", cli.Highlight(toolName))
	return mgr.Run(toolName, toolArgs, opts)
}

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"qoliber/magebox/internal/cli"
	"qoliber/magebox/internal/config"
	"qoliber/magebox/internal/updater"
	"qoliber/magebox/internal/verbose"
)

var version = "dev"

// verbosity is the count of -v flags
var verbosity int

// versionChecker runs an async update check in the background
var versionChecker *updater.VersionChecker

func main() {
	// If the first non-flag argument is not a known command,
	// check if it's a custom command from .magebox and delegate to "run".
	if len(os.Args) > 1 {
		firstArg := os.Args[1]
		// Skip flags
		if firstArg != "" && firstArg[0] != '-' {
			cmd, _, _ := rootCmd.Find(os.Args[1:])
			if cmd == rootCmd {
				// Not a known subcommand — check if it's a custom project command
				delegated := false
				if cwd, err := os.Getwd(); err == nil {
					if cfg, err := config.LoadFromPath(cwd); err == nil {
						if _, ok := cfg.Commands[firstArg]; ok {
							// Insert "run" before the unknown command
							newArgs := make([]string, 0, len(os.Args)+1)
							newArgs = append(newArgs, os.Args[0], "run")
							newArgs = append(newArgs, os.Args[1:]...)
							os.Args = newArgs
							delegated = true
						}
					}
				}

				// Fall back to magerun2 if it's available in PATH and it
				// actually knows about the requested command. Otherwise
				// let Cobra show its normal "unknown command" error.
				if !delegated {
					if binary, ok := findMagerun(); ok {
						if magerunHasCommand(binary, firstArg) {
							os.Exit(runMagerun(binary, os.Args[1:]))
						}
					}
				}
			}
		}
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "magebox",
	Short: "MageBox - Modern Magento Development Environment",
	Long: `MageBox is a modern, fast development environment for Magento.

It uses native PHP-FPM and Nginx for maximum performance,
with Docker for services like MySQL, Redis, OpenSearch, and Varnish.`,
	Version: version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Set verbosity level based on -v count
		verbose.SetLevel(verbose.Level(verbosity))

		if verbose.IsEnabled(verbose.LevelDebug) {
			verbose.Debug("MageBox version: %s", version)
			verbose.Debug("Verbosity level: %d", verbosity)
			verbose.Env()
		}

		// Start async version check (skip for self-update and dev builds)
		if cmd.Name() != "self-update" && version != "dev" {
			if homeDir, err := os.UserHomeDir(); err == nil {
				versionChecker = updater.NewVersionChecker(version, homeDir)
				versionChecker.Start()
			}
		}
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if versionChecker != nil {
			if msg := versionChecker.Result(); msg != "" {
				fmt.Println()
				cli.PrintInfo("%s", msg)
			}
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Show logo when running without subcommand
		cli.PrintLogoSmall(version)
		fmt.Println()
		_ = cmd.Help()
	},
}

func init() {
	// Global verbose flag - can be repeated: -v, -vv, -vvv
	rootCmd.PersistentFlags().CountVarP(&verbosity, "verbose", "v", "Increase verbosity (-v, -vv, -vvv)")
}

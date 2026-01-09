package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"qoliber/magebox/internal/cli"
	"qoliber/magebox/internal/verbose"
)

var version = "1.1.3"

// verbosity is the count of -v flags
var verbosity int

func main() {
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

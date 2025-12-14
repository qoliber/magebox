package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/qoliber/magebox/internal/cli"
)

var version = "0.12.8"

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
	Run: func(cmd *cobra.Command, args []string) {
		// Show logo when running without subcommand
		cli.PrintLogoSmall(version)
		fmt.Println()
		_ = cmd.Help()
	},
}

func init() {
	// Root commands are registered in each command file's init()
	// This keeps root.go clean and each command self-contained
}

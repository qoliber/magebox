package main

import (
	"github.com/spf13/cobra"

	"qoliber/magebox/internal/portforward"
)

var portforwardDaemonCmd = &cobra.Command{
	Use:    "_portforward",
	Short:  "Run port forwarding daemon (internal — called by LaunchDaemon)",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return portforward.RunProxy(portforward.DefaultPairs())
	},
}

func init() {
	rootCmd.AddCommand(portforwardDaemonCmd)
}

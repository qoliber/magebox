package main

import (
	"github.com/spf13/cobra"

	"github.com/qoliber/magebox/internal/cli"
	"github.com/qoliber/magebox/internal/project"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop project services",
	Long:  "Stops all services for the current project",
	RunE:  runStop,
}

func init() {
	rootCmd.AddCommand(stopCmd)
}

func runStop(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	p, err := getPlatform()
	if err != nil {
		return err
	}

	cli.PrintInfo("Stopping MageBox services...")

	mgr := project.NewManager(p)
	if err := mgr.Stop(cwd); err != nil {
		cli.PrintError("%v", err)
		return nil
	}

	cli.PrintSuccess("Project stopped successfully!")
	return nil
}

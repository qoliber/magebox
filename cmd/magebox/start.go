package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/qoliber/magebox/internal/cli"
	"github.com/qoliber/magebox/internal/project"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start project services",
	Long:  "Starts all services defined in .magebox for the current project",
	RunE:  runStart,
}

func init() {
	rootCmd.AddCommand(startCmd)
}

func runStart(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	p, err := getPlatform()
	if err != nil {
		return err
	}

	cli.PrintTitle("Starting MageBox Services")
	fmt.Println()

	mgr := project.NewManager(p)

	// Validate first
	cfg, warnings, err := mgr.ValidateConfig(cwd)
	if err != nil {
		cli.PrintError("%v", err)
		return nil
	}

	// Show warnings
	for _, w := range warnings {
		cli.PrintWarning("%s", w)
	}

	// Start services
	result, err := mgr.Start(cwd)
	if err != nil {
		cli.PrintError("%v", err)
		return nil
	}

	// Show results
	fmt.Printf("Project: %s\n", cli.Highlight(cfg.Name))
	fmt.Printf("PHP:     %s\n", cli.Highlight(result.PHPVersion))
	fmt.Println()

	fmt.Println(cli.Header("Domains"))
	for _, d := range result.Domains {
		fmt.Printf("  %s\n", cli.URL("https://"+d))
	}
	fmt.Println()

	fmt.Println(cli.Header("Services"))
	for _, s := range result.Services {
		fmt.Printf("  %s %s\n", cli.Success(""), s)
	}
	fmt.Println()

	// Show warnings from start
	for _, w := range result.Warnings {
		cli.PrintWarning("%s", w)
	}

	// Show errors
	for _, e := range result.Errors {
		cli.PrintError("%v", e)
	}

	if len(result.Errors) == 0 {
		cli.PrintSuccess("Project started successfully!")
	}

	return nil
}

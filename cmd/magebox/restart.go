package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/qoliber/magebox/internal/cli"
	"github.com/qoliber/magebox/internal/project"
)

var restartAllProjects bool

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart project services",
	Long:  "Stops and starts all services for the current project, or all projects with --all",
	RunE:  runRestart,
}

func init() {
	restartCmd.Flags().BoolVarP(&restartAllProjects, "all", "a", false, "Restart all MageBox projects")
	rootCmd.AddCommand(restartCmd)
}

func runRestart(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	mgr := project.NewManager(p)

	if restartAllProjects {
		return restartAll(p, mgr)
	}

	// Restart current project
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	cli.PrintTitle("Restarting MageBox Services")
	fmt.Println()

	// Stop
	fmt.Print("Stopping services... ")
	if err := mgr.Stop(cwd); err != nil {
		fmt.Println(cli.Warning("warning"))
		cli.PrintWarning("  %v", err)
	} else {
		fmt.Println(cli.Success("done"))
	}

	// Start
	fmt.Print("Starting services... ")
	result, err := mgr.Start(cwd)
	if err != nil {
		fmt.Println(cli.Error("failed"))
		cli.PrintError("  %v", err)
		return nil
	}
	fmt.Println(cli.Success("done"))
	fmt.Println()

	// Show domains
	fmt.Println(cli.Header("Domains"))
	for _, d := range result.Domains {
		fmt.Printf("  %s\n", cli.URL("https://"+d))
	}
	fmt.Println()

	// Show any warnings/errors
	for _, w := range result.Warnings {
		cli.PrintWarning("%s", w)
	}
	for _, e := range result.Errors {
		cli.PrintError("%v", e)
	}

	if len(result.Errors) == 0 {
		cli.PrintSuccess("Project restarted successfully!")
	}

	return nil
}

func restartAll(p interface{ MageBoxDir() string }, mgr *project.Manager) error {
	cli.PrintTitle("Restarting All MageBox Projects")
	fmt.Println()

	plat, err := getPlatform()
	if err != nil {
		return err
	}

	discovery := project.NewProjectDiscovery(plat)
	projects, err := discovery.DiscoverProjects()
	if err != nil {
		cli.PrintError("Failed to discover projects: %v", err)
		return nil
	}

	if len(projects) == 0 {
		cli.PrintInfo("No projects found")
		return nil
	}

	restarted := 0
	failed := 0

	for _, proj := range projects {
		if !proj.HasConfig {
			continue
		}

		fmt.Printf("Restarting %s... ", cli.Highlight(proj.Name))

		// Stop (ignore errors)
		_ = mgr.Stop(proj.Path)

		// Start
		_, _, err := mgr.ValidateConfig(proj.Path)
		if err != nil {
			fmt.Println(cli.Error("failed"))
			cli.PrintError("  %v", err)
			failed++
			continue
		}

		result, err := mgr.Start(proj.Path)
		if err != nil {
			fmt.Println(cli.Error("failed"))
			cli.PrintError("  %v", err)
			failed++
		} else if len(result.Errors) > 0 {
			fmt.Println(cli.Warning("partial"))
			restarted++
		} else {
			fmt.Println(cli.Success("done"))
			restarted++
		}
	}

	fmt.Println()
	if failed > 0 {
		cli.PrintWarning("Restarted %d project(s), %d failed", restarted, failed)
	} else {
		cli.PrintSuccess("Restarted %d project(s)", restarted)
	}

	return nil
}

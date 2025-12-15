package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/qoliber/magebox/internal/cli"
	"github.com/qoliber/magebox/internal/project"
)

var startAllProjects bool

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start project services",
	Long:  "Starts all services defined in .magebox for the current project, or all projects with --all",
	RunE:  runStart,
}

func init() {
	startCmd.Flags().BoolVarP(&startAllProjects, "all", "a", false, "Start all MageBox projects")
	rootCmd.AddCommand(startCmd)
}

func runStart(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	mgr := project.NewManager(p)

	if startAllProjects {
		return startAll(p, mgr)
	}

	// Start current project
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	return startProject(mgr, cwd, true)
}

func startProject(mgr *project.Manager, projectPath string, verbose bool) error {
	if verbose {
		cli.PrintTitle("Starting MageBox Services")
		fmt.Println()
	}

	// Validate first
	cfg, warnings, err := mgr.ValidateConfig(projectPath)
	if err != nil {
		cli.PrintError("%v", err)
		return nil
	}

	// Show warnings
	for _, w := range warnings {
		cli.PrintWarning("%s", w)
	}

	// Start services
	result, err := mgr.Start(projectPath)
	if err != nil {
		cli.PrintError("%v", err)
		return nil
	}

	if verbose {
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
	}

	return nil
}

func startAll(p interface{ MageBoxDir() string }, mgr *project.Manager) error {
	cli.PrintTitle("Starting All MageBox Projects")
	fmt.Println()

	// Need platform type for discovery
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

	started := 0
	failed := 0

	for _, proj := range projects {
		if !proj.HasConfig {
			continue // Skip projects without .magebox.yaml
		}

		fmt.Printf("Starting %s... ", cli.Highlight(proj.Name))

		// Validate and start
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
			for _, e := range result.Errors {
				cli.PrintError("  %v", e)
			}
			started++ // Count as started even with warnings
		} else {
			fmt.Println(cli.Success("done"))
			started++
		}
	}

	fmt.Println()
	if failed > 0 {
		cli.PrintWarning("Started %d project(s), %d failed", started, failed)
	} else {
		cli.PrintSuccess("Started %d project(s)", started)
	}

	return nil
}

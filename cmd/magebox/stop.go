package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"qoliber/magebox/internal/cli"
	"qoliber/magebox/internal/config"
	"qoliber/magebox/internal/project"
)

var (
	stopAllProjects bool
	stopDryRun      bool
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop project services",
	Long:  "Stops all services for the current project, or all projects with --all",
	RunE:  runStop,
}

func init() {
	stopCmd.Flags().BoolVarP(&stopAllProjects, "all", "a", false, "Stop all MageBox projects")
	stopCmd.Flags().BoolVar(&stopDryRun, "dry-run", false, "Show what would be stopped without stopping")
	rootCmd.AddCommand(stopCmd)
}

func runStop(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	mgr := project.NewManager(p)

	if stopAllProjects {
		return stopAll(p, mgr)
	}

	// Stop current project
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	if stopDryRun {
		return stopDryRunSingle(cwd)
	}

	cli.PrintInfo("Stopping MageBox services...")

	if err := mgr.Stop(cwd); err != nil {
		cli.PrintError("%v", err)
		return nil
	}

	cli.PrintSuccess("Project stopped successfully!")
	return nil
}

func stopDryRunSingle(cwd string) error {
	cfg, err := config.LoadFromPath(cwd)
	if err != nil {
		cli.PrintError("No .magebox.yaml found in current directory")
		return nil
	}

	cli.PrintTitle("Dry Run: Would stop")
	fmt.Println()
	fmt.Printf("Project: %s\n", cli.Highlight(cfg.Name))
	fmt.Printf("Path:    %s\n", cli.Path(cwd))
	fmt.Println()
	fmt.Println("Services that would be stopped:")
	fmt.Printf("  %s PHP-FPM %s\n", cli.Bullet(""), cfg.PHP)
	fmt.Printf("  %s Nginx vhosts\n", cli.Bullet(""))
	if cfg.Services.HasMySQL() {
		fmt.Printf("  %s MySQL container\n", cli.Bullet(""))
	}
	if cfg.Services.HasMariaDB() {
		fmt.Printf("  %s MariaDB container\n", cli.Bullet(""))
	}
	if cfg.Services.HasRedis() {
		fmt.Printf("  %s Redis container\n", cli.Bullet(""))
	}
	if cfg.Services.HasOpenSearch() {
		fmt.Printf("  %s OpenSearch container\n", cli.Bullet(""))
	}
	if cfg.Services.HasElasticsearch() {
		fmt.Printf("  %s Elasticsearch container\n", cli.Bullet(""))
	}
	if cfg.Services.HasRabbitMQ() {
		fmt.Printf("  %s RabbitMQ container\n", cli.Bullet(""))
	}
	if cfg.Services.HasMailpit() {
		fmt.Printf("  %s Mailpit container\n", cli.Bullet(""))
	}
	if cfg.Services.HasVarnish() {
		fmt.Printf("  %s Varnish container\n", cli.Bullet(""))
	}
	fmt.Println()
	cli.PrintInfo("Run without --dry-run to actually stop")
	return nil
}

func stopAll(p interface{ MageBoxDir() string }, mgr *project.Manager) error {
	cli.PrintTitle("Stopping All MageBox Projects")
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

	stopped := 0
	failed := 0

	for _, proj := range projects {
		if !proj.HasConfig {
			continue // Skip projects without .magebox.yaml
		}

		fmt.Printf("Stopping %s... ", cli.Highlight(proj.Name))

		if err := mgr.Stop(proj.Path); err != nil {
			fmt.Println(cli.Error("failed"))
			cli.PrintError("  %v", err)
			failed++
		} else {
			fmt.Println(cli.Success("done"))
			stopped++
		}
	}

	fmt.Println()
	if failed > 0 {
		cli.PrintWarning("Stopped %d project(s), %d failed", stopped, failed)
	} else {
		cli.PrintSuccess("Stopped %d project(s)", stopped)
	}

	return nil
}

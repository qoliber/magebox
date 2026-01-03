package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"qoliber/magebox/internal/cli"
	"qoliber/magebox/internal/project"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show project status",
	Long:  "Shows the status of all services for the current project",
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	p, err := getPlatform()
	if err != nil {
		return err
	}

	mgr := project.NewManager(p)
	status, err := mgr.Status(cwd)
	if err != nil {
		cli.PrintError("%v", err)
		return nil
	}

	cli.PrintTitle("Project Status")
	fmt.Println()
	fmt.Printf("Project: %s\n", cli.Highlight(status.Name))
	fmt.Printf("Path:    %s\n", cli.Path(status.Path))
	fmt.Printf("PHP:     %s\n", cli.Highlight(status.PHPVersion))

	fmt.Println(cli.Header("Domains"))
	for _, d := range status.Domains {
		fmt.Printf("  %s\n", cli.URL(d))
	}

	fmt.Println(cli.Header("Services"))
	for _, svc := range status.Services {
		fmt.Printf("  %-20s %s\n", svc.Name, cli.Status(svc.IsRunning))
	}

	fmt.Println(cli.Header("Config Files"))
	fmt.Printf("  Project:  %s\n", cli.Path(status.ConfigPaths.ProjectConfig))
	if _, err := os.Stat(status.ConfigPaths.PHPFPMPool); err == nil {
		fmt.Printf("  PHP-FPM:  %s\n", cli.Path(status.ConfigPaths.PHPFPMPool))
	}
	for _, vhost := range status.ConfigPaths.NginxVhosts {
		fmt.Printf("  Nginx:    %s\n", cli.Path(vhost))
	}

	return nil
}

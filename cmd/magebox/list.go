package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"qoliber/magebox/internal/cli"
	"qoliber/magebox/internal/project"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all MageBox projects",
	Long:  "Lists all discovered MageBox projects from nginx vhosts",
	RunE:  runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	cli.PrintTitle("MageBox Projects")
	fmt.Println()

	discovery := project.NewProjectDiscovery(p)
	projects, err := discovery.DiscoverProjects()
	if err != nil {
		cli.PrintError("Failed to discover projects: %v", err)
		return nil
	}

	if len(projects) == 0 {
		cli.PrintInfo("No projects found")
		fmt.Println()
		fmt.Println(cli.Bullet("Run " + cli.Command("magebox init") + " in a Magento project directory"))
		fmt.Println(cli.Bullet("Then run " + cli.Command("magebox start") + " to start services"))
		return nil
	}

	for i, proj := range projects {
		// Project header
		if proj.HasConfig {
			fmt.Printf("%s %s\n", cli.Success(""), cli.Highlight(proj.Name))
		} else {
			fmt.Printf("%s %s %s\n", cli.Warning(""), proj.Name, cli.Subtitle("(no .magebox file)"))
		}

		// Details
		fmt.Printf("    Path: %s\n", cli.Path(proj.Path))
		if proj.PHPVersion != "" {
			fmt.Printf("    PHP:  %s\n", proj.PHPVersion)
		}
		if len(proj.Domains) > 0 {
			fmt.Printf("    URLs: ")
			for j, domain := range proj.Domains {
				if j > 0 {
					fmt.Print(", ")
				}
				fmt.Print(cli.URL("https://" + domain))
			}
			fmt.Println()
		}

		if i < len(projects)-1 {
			fmt.Println()
		}
	}

	fmt.Println()
	fmt.Printf("Total: %d project(s)\n", len(projects))

	return nil
}

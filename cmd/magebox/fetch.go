// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/qoliber/magebox/internal/cli"
	"github.com/qoliber/magebox/internal/team"
)

var fetchCmd = &cobra.Command{
	Use:   "fetch <project>",
	Short: "Fetch a team project (clone repo, download DB & media)",
	Long: `Fetches a project from a configured team.

This command:
1. Clones the repository from the configured provider
2. Downloads the database dump from asset storage
3. Imports the database to MySQL
4. Downloads and extracts media files

Project can be specified as:
  - "project" (if only one team configured)
  - "team/project" (explicit team)

Examples:
  magebox fetch myproject              # Fetch from default team
  magebox fetch qoliber/myproject      # Fetch from specific team
  magebox fetch myproject --branch dev # Fetch specific branch
  magebox fetch myproject --no-db      # Skip database
  magebox fetch myproject --dry-run    # Show what would happen`,
	Args: cobra.ExactArgs(1),
	RunE: runFetch,
}

var (
	fetchBranch    string
	fetchNoDB      bool
	fetchNoMedia   bool
	fetchDBOnly    bool
	fetchMediaOnly bool
	fetchDryRun    bool
	fetchDestPath  string
)

func init() {
	fetchCmd.Flags().StringVar(&fetchBranch, "branch", "", "Branch to clone")
	fetchCmd.Flags().BoolVar(&fetchNoDB, "no-db", false, "Skip database download")
	fetchCmd.Flags().BoolVar(&fetchNoMedia, "no-media", false, "Skip media download")
	fetchCmd.Flags().BoolVar(&fetchDBOnly, "db-only", false, "Only download database")
	fetchCmd.Flags().BoolVar(&fetchMediaOnly, "media-only", false, "Only download media")
	fetchCmd.Flags().BoolVar(&fetchDryRun, "dry-run", false, "Show what would happen without making changes")
	fetchCmd.Flags().StringVar(&fetchDestPath, "to", "", "Destination path (default: current dir + project name)")

	rootCmd.AddCommand(fetchCmd)
}

func runFetch(cmd *cobra.Command, args []string) error {
	projectRef := args[0]

	p, err := getPlatform()
	if err != nil {
		return err
	}

	// Load team config
	config, err := team.LoadConfig(p.HomeDir)
	if err != nil {
		return err
	}

	// Find project
	t, project, err := config.FindProject(projectRef)
	if err != nil {
		return err
	}

	// Determine destination path
	projectName := filepath.Base(project.Repo)
	destPath := fetchDestPath
	if destPath == "" {
		cwd, _ := os.Getwd()
		destPath = filepath.Join(cwd, projectName)
	}

	// Print header
	cli.PrintTitle("Fetching Project")
	fmt.Println()
	fmt.Printf("  Team:        %s\n", cli.Highlight(t.Name))
	fmt.Printf("  Project:     %s\n", cli.Highlight(projectName))
	fmt.Printf("  Repository:  %s\n", project.Repo)
	fmt.Printf("  Branch:      %s\n", getBranchForFetch(project))
	fmt.Printf("  Destination: %s\n", destPath)
	fmt.Println()

	// Check what will be fetched
	fetchItems := []string{}
	if !fetchNoDB && !fetchMediaOnly && project.DB != "" {
		fetchItems = append(fetchItems, "database")
	}
	if !fetchNoMedia && !fetchDBOnly && project.Media != "" {
		fetchItems = append(fetchItems, "media")
	}
	if len(fetchItems) > 0 {
		fmt.Printf("  Assets:      %s\n", strings.Join(fetchItems, ", "))
		if t.Assets.Host != "" {
			fmt.Printf("  From:        %s@%s\n", t.Assets.Username, t.Assets.Host)
		}
		fmt.Println()
	}

	if fetchDryRun {
		fmt.Println(cli.Warning("DRY RUN - No changes will be made"))
		fmt.Println()
	}

	// Create fetch options
	options := team.FetchOptions{
		Branch:    fetchBranch,
		NoDB:      fetchNoDB,
		NoMedia:   fetchNoMedia,
		DBOnly:    fetchDBOnly,
		MediaOnly: fetchMediaOnly,
		DryRun:    fetchDryRun,
		DestPath:  destPath,
	}

	// Create fetcher and execute
	fetcher := team.NewFetcher(t, project, options)
	fetcher.SetProgressCallback(func(msg string) {
		// Handle carriage return for progress updates
		if strings.HasPrefix(msg, "\r") {
			fmt.Print(msg)
		} else {
			fmt.Println(msg)
		}
	})

	if err := fetcher.Execute(); err != nil {
		return err
	}

	fmt.Println()
	cli.PrintSuccess("Project fetched successfully!")
	fmt.Println()

	// Show next steps
	cli.PrintInfo("Next steps:")
	fmt.Printf("  cd %s\n", destPath)
	fmt.Println("  magebox init        # Initialize MageBox configuration")
	fmt.Println("  magebox start       # Start services")

	return nil
}

func getBranchForFetch(project *team.Project) string {
	if fetchBranch != "" {
		return fetchBranch
	}
	if project.Branch != "" {
		return project.Branch
	}
	return "main"
}

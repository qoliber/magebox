// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/qoliber/magebox/internal/cli"
	"github.com/qoliber/magebox/internal/config"
	"github.com/qoliber/magebox/internal/team"
)

var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Download database and media from team asset storage",
	Long: `Downloads database and media files from the team's asset storage for the current project.

This command reads the project name from .magebox.yaml and searches for it in
the configured team asset storage. If found, it downloads and imports the database.

Examples:
  magebox fetch              # Download & import database
  magebox fetch --media      # Also download & extract media
  magebox fetch --dry-run    # Show what would happen
  magebox fetch --backup     # Backup current DB before importing`,
	RunE: runFetch,
}

var (
	fetchMedia   bool
	fetchDryRun  bool
	fetchBackup  bool
	fetchTeam    string
)

func init() {
	fetchCmd.Flags().BoolVar(&fetchMedia, "media", false, "Also download and extract media")
	fetchCmd.Flags().BoolVar(&fetchDryRun, "dry-run", false, "Show what would happen without making changes")
	fetchCmd.Flags().BoolVar(&fetchBackup, "backup", false, "Backup current database before importing")
	fetchCmd.Flags().StringVar(&fetchTeam, "team", "", "Specify team (if project exists in multiple teams)")

	rootCmd.AddCommand(fetchCmd)
}

func runFetch(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	p, err := getPlatform()
	if err != nil {
		return err
	}

	// Load project config from current directory
	projectConfig, err := config.LoadFromPath(cwd)
	if err != nil {
		return fmt.Errorf("not in a MageBox project directory: %w\n\nRun this command from a project with .magebox.yaml", err)
	}

	projectName := projectConfig.Name
	if projectName == "" {
		return fmt.Errorf("project name not set in .magebox.yaml")
	}

	// Load team config
	teamConfig, err := team.LoadConfig(p.HomeDir)
	if err != nil {
		return fmt.Errorf("no team configuration found: %w\n\nRun 'magebox team add' to configure a team", err)
	}

	// Find team with this project in asset storage
	t, dbPath, mediaPath, err := findProjectInAssetStorage(teamConfig, projectName, fetchTeam)
	if err != nil {
		return err
	}

	// Print header
	cli.PrintTitle("Fetching Assets")
	fmt.Println()
	fmt.Printf("  Project:  %s\n", cli.Highlight(projectName))
	fmt.Printf("  Team:     %s\n", cli.Highlight(t.Name))
	fmt.Printf("  From:     %s@%s\n", t.Assets.Username, t.Assets.Host)
	fmt.Printf("  Database: %s\n", dbPath)
	if fetchMedia {
		fmt.Printf("  Media:    %s\n", mediaPath)
	}
	fmt.Println()

	if fetchDryRun {
		fmt.Println(cli.Warning("DRY RUN - No changes will be made"))
		fmt.Println()
		return dryRunFetch(t, dbPath, mediaPath, fetchMedia)
	}

	// Create fetcher and execute
	options := team.FetchOptions{
		DryRun:    fetchDryRun,
		NoMedia:   !fetchMedia,
		DestPath:  cwd,
	}

	fetcher := team.NewAssetFetcher(t, projectName, dbPath, mediaPath, options)
	fetcher.SetProgressCallback(createProgressCallback())

	// Backup current database if requested
	if fetchBackup {
		if err := backupDatabase(cwd); err != nil {
			cli.PrintWarning("Failed to backup database: %v", err)
		}
	}

	if err := fetcher.Execute(); err != nil {
		return err
	}

	fmt.Println()
	cli.PrintSuccess("Assets fetched successfully!")

	return nil
}

// findProjectInAssetStorage searches team asset storage for the project
func findProjectInAssetStorage(cfg *team.TeamsConfig, projectName, preferredTeam string) (*team.Team, string, string, error) {
	// Build default paths
	dbPath := fmt.Sprintf("%s/%s.sql.gz", projectName, projectName)
	mediaPath := fmt.Sprintf("%s/%s.tar.gz", projectName, projectName)

	// If team specified, use it directly
	if preferredTeam != "" {
		t, ok := cfg.Teams[preferredTeam]
		if !ok {
			return nil, "", "", fmt.Errorf("team '%s' not found in configuration", preferredTeam)
		}
		return t, dbPath, mediaPath, nil
	}

	// Check if project is configured in any team
	var matchedTeams []*team.Team
	for _, t := range cfg.Teams {
		// First check if project is explicitly configured
		if _, ok := t.Projects[projectName]; ok {
			proj := t.Projects[projectName]
			if proj.DB != "" {
				dbPath = proj.DB
			}
			if proj.Media != "" {
				mediaPath = proj.Media
			}
			return t, dbPath, mediaPath, nil
		}
		// Otherwise, add team with asset storage configured to candidates
		if t.Assets.Host != "" {
			matchedTeams = append(matchedTeams, t)
		}
	}

	// If only one team has asset storage, use it
	if len(matchedTeams) == 1 {
		return matchedTeams[0], dbPath, mediaPath, nil
	}

	// Multiple teams - need to check which one has the file
	if len(matchedTeams) > 1 {
		// Try to find the project in asset storage
		for _, t := range matchedTeams {
			assetClient := team.NewAssetClient(t, nil)
			if err := assetClient.Connect(); err != nil {
				continue
			}
			if assetClient.FileExists(dbPath) {
				assetClient.Close()
				return t, dbPath, mediaPath, nil
			}
			assetClient.Close()
		}

		// Project not found in any team's asset storage
		teamNames := make([]string, 0, len(matchedTeams))
		for _, t := range matchedTeams {
			teamNames = append(teamNames, t.Name)
		}
		return nil, "", "", fmt.Errorf("project '%s' not found in asset storage\n\nChecked teams: %v\nExpected path: %s\n\nUse --team to specify which team to use, or add the project to team config:\n  magebox team %s project add %s",
			projectName, teamNames, dbPath, matchedTeams[0].Name, projectName)
	}

	return nil, "", "", fmt.Errorf("no team with asset storage configured\n\nRun 'magebox team add' to configure a team with asset storage")
}

// dryRunFetch shows what would happen
func dryRunFetch(t *team.Team, dbPath, mediaPath string, includeMedia bool) error {
	// Connect to verify files exist
	assetClient := team.NewAssetClient(t, nil)
	if err := assetClient.Connect(); err != nil {
		return fmt.Errorf("failed to connect to asset storage: %w", err)
	}
	defer assetClient.Close()

	// Check database
	if assetClient.FileExists(dbPath) {
		size, _ := assetClient.GetFileSize(dbPath)
		fmt.Printf("Would download: %s (%s)\n", dbPath, team.FormatBytes(size))
		fmt.Println("Would import to MySQL via 'magebox db import'")
	} else {
		fmt.Printf("%s Database not found: %s\n", cli.Warning("!"), dbPath)
	}

	// Check media
	if includeMedia {
		if assetClient.FileExists(mediaPath) {
			size, _ := assetClient.GetFileSize(mediaPath)
			fmt.Printf("\nWould download: %s (%s)\n", mediaPath, team.FormatBytes(size))
			fmt.Println("Would extract to pub/media")
		} else {
			fmt.Printf("\n%s Media not found: %s\n", cli.Warning("!"), mediaPath)
		}
	}

	return nil
}

// createProgressCallback creates a progress callback for fetch operations
func createProgressCallback() func(string) {
	return func(msg string) {
		if len(msg) > 0 && msg[0] == '\r' {
			fmt.Print(msg)
		} else {
			fmt.Println(msg)
		}
	}
}

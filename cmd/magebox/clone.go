// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/qoliber/magebox/internal/cli"
	"github.com/qoliber/magebox/internal/team"
)

var cloneCmd = &cobra.Command{
	Use:   "clone <project>",
	Short: "Clone a team project repository",
	Long: `Clones a project repository from a configured team.

This command:
1. Clones the repository from the configured provider
2. Creates .magebox.yaml if not present
3. Runs composer install

Project can be specified as:
  - "project" (if only one team configured)
  - "team/project" (explicit team)

Examples:
  magebox clone myproject              # Clone from default team
  magebox clone qoliber/myproject      # Clone from specific team
  magebox clone myproject --branch dev # Clone specific branch
  magebox clone myproject --fetch      # Clone and fetch DB/media
  magebox clone myproject --to ~/code  # Clone to specific directory`,
	Args: cobra.ExactArgs(1),
	RunE: runClone,
}

var (
	cloneBranch   string
	cloneFetch    bool
	cloneDestPath string
	cloneDryRun   bool
)

func init() {
	cloneCmd.Flags().StringVar(&cloneBranch, "branch", "", "Branch to clone")
	cloneCmd.Flags().BoolVar(&cloneFetch, "fetch", false, "Also fetch database and media after cloning")
	cloneCmd.Flags().StringVar(&cloneDestPath, "to", "", "Destination path (default: current dir + project name)")
	cloneCmd.Flags().BoolVar(&cloneDryRun, "dry-run", false, "Show what would happen without making changes")

	rootCmd.AddCommand(cloneCmd)
}

func runClone(cmd *cobra.Command, args []string) error {
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
	destPath := cloneDestPath
	if destPath == "" {
		cwd, _ := os.Getwd()
		destPath = filepath.Join(cwd, projectName)
	}

	// Determine branch
	branch := getBranchForClone(t, project)

	// Print header
	cli.PrintTitle("Cloning Project")
	fmt.Println()
	fmt.Printf("  Team:        %s\n", cli.Highlight(t.Name))
	fmt.Printf("  Project:     %s\n", cli.Highlight(projectName))
	fmt.Printf("  Repository:  %s\n", project.Repo)
	fmt.Printf("  Branch:      %s\n", branch)
	fmt.Printf("  Destination: %s\n", destPath)
	fmt.Println()

	if cloneDryRun {
		fmt.Println(cli.Warning("DRY RUN - No changes will be made"))
		fmt.Println()
		return dryRunClone(t, project, destPath, branch)
	}

	// Create clone options
	options := team.CloneOptions{
		Branch:   cloneBranch,
		DestPath: destPath,
		DryRun:   cloneDryRun,
	}

	// Create cloner and execute
	cloner := team.NewCloner(t, project, options)
	cloner.SetProgressCallback(createProgressCallback())

	if err := cloner.Execute(); err != nil {
		return err
	}

	fmt.Println()
	cli.PrintSuccess("Project cloned successfully!")
	fmt.Println()

	// If --fetch flag, run fetch from the cloned directory
	if cloneFetch {
		fmt.Println()
		cli.PrintInfo("Fetching assets...")

		// Change to project directory for fetch
		originalDir, _ := os.Getwd()
		if err := os.Chdir(destPath); err != nil {
			return fmt.Errorf("failed to change to project directory: %w", err)
		}
		defer os.Chdir(originalDir)

		// Build paths
		dbPath := fmt.Sprintf("%s/%s.sql.gz", projectName, projectName)
		mediaPath := fmt.Sprintf("%s/%s.tar.gz", projectName, projectName)

		// Use project-specific paths if configured
		if project.DB != "" {
			dbPath = project.DB
		}
		if project.Media != "" {
			mediaPath = project.Media
		}

		fetchOptions := team.FetchOptions{
			NoMedia:  false, // Fetch both DB and media when using --fetch
			DestPath: destPath,
		}

		fetcher := team.NewAssetFetcher(t, projectName, dbPath, mediaPath, fetchOptions)
		fetcher.SetProgressCallback(createProgressCallback())

		if err := fetcher.Execute(); err != nil {
			cli.PrintWarning("Failed to fetch assets: %v", err)
			fmt.Println("You can run 'magebox fetch' later to download assets")
		}
	}

	// Show next steps
	cli.PrintInfo("Next steps:")
	fmt.Printf("  cd %s\n", destPath)
	if !cloneFetch {
		fmt.Println("  magebox fetch       # Download database")
	}
	fmt.Println("  magebox start       # Start services")

	return nil
}

func getBranchForClone(t *team.Team, project *team.Project) string {
	if cloneBranch != "" {
		return cloneBranch
	}
	if project.Branch != "" {
		return project.Branch
	}
	// Detect default branch from remote
	if branch := team.DetectDefaultBranch(t.GetCloneURL(project)); branch != "" {
		return branch
	}
	return "main"
}

func dryRunClone(t *team.Team, project *team.Project, destPath, branch string) error {
	fmt.Printf("Would clone: %s\n", t.GetCloneURL(project))
	fmt.Printf("  Branch: %s\n", branch)
	fmt.Printf("  Destination: %s\n", destPath)
	fmt.Println("  Would create .magebox.yaml if not present")
	fmt.Println("  Would run: composer install")

	if cloneFetch {
		projectName := filepath.Base(project.Repo)
		dbPath := fmt.Sprintf("%s/%s.sql.gz", projectName, projectName)
		mediaPath := fmt.Sprintf("%s/%s.tar.gz", projectName, projectName)
		if project.DB != "" {
			dbPath = project.DB
		}
		if project.Media != "" {
			mediaPath = project.Media
		}

		fmt.Println()
		fmt.Println("Would fetch assets:")
		fmt.Printf("  Database: %s\n", dbPath)
		fmt.Printf("  Media: %s\n", mediaPath)
		fmt.Printf("  From: %s@%s:%s\n", t.Assets.Username, t.Assets.Host, t.Assets.Path)
	}

	return nil
}

// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/qoliber/magebox/internal/cli"
	"github.com/qoliber/magebox/internal/team"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync database and media from team asset storage",
	Long: `Syncs the latest database dump and media files from the team's asset storage.

This command must be run from within a project directory that was fetched
using 'magebox fetch'. It reads the team/project configuration to determine
what to sync.

Examples:
  magebox sync              # Sync both DB and media
  magebox sync --db         # Only sync database
  magebox sync --media      # Only sync media
  magebox sync --backup     # Backup current DB before syncing
  magebox sync --dry-run    # Show what would happen`,
	RunE: runSync,
}

var (
	syncDBOnly    bool
	syncMediaOnly bool
	syncBackup    bool
	syncDryRun    bool
)

func init() {
	syncCmd.Flags().BoolVar(&syncDBOnly, "db", false, "Only sync database")
	syncCmd.Flags().BoolVar(&syncMediaOnly, "media", false, "Only sync media")
	syncCmd.Flags().BoolVar(&syncBackup, "backup", false, "Backup current database before syncing")
	syncCmd.Flags().BoolVar(&syncDryRun, "dry-run", false, "Show what would happen without making changes")

	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	p, err := getPlatform()
	if err != nil {
		return err
	}

	// Try to find team/project from git remote or .magebox.yaml
	t, project, err := findProjectForSync(cwd, p.HomeDir)
	if err != nil {
		return err
	}

	cli.PrintTitle("Syncing Project")
	fmt.Println()
	fmt.Printf("  Team:    %s\n", cli.Highlight(t.Name))
	fmt.Printf("  Project: %s\n", cli.Highlight(filepath.Base(project.Repo)))
	fmt.Println()

	if syncDryRun {
		fmt.Println(cli.Warning("DRY RUN - No changes will be made"))
		fmt.Println()
	}

	// Backup current database if requested
	if syncBackup && !syncMediaOnly && !syncDryRun {
		if err := backupDatabase(cwd); err != nil {
			cli.PrintWarning("Failed to backup database: %v", err)
		}
	}

	// Connect to asset storage
	assetClient := team.NewAssetClient(t, func(prog team.DownloadProgress) {
		fmt.Printf("\r  %s: %.1f%% (%s/%s) %s ETA: %s",
			prog.Filename, prog.Percentage,
			team.FormatBytes(prog.Downloaded), team.FormatBytes(prog.TotalBytes),
			team.FormatSpeed(prog.Speed), prog.ETA)
	})

	if !syncDryRun {
		if err := assetClient.Connect(); err != nil {
			return fmt.Errorf("failed to connect to asset storage: %w", err)
		}
		defer assetClient.Close()
	}

	// Sync database
	if !syncMediaOnly && project.DB != "" {
		if syncDryRun {
			size, _ := assetClient.GetFileSize(project.DB)
			fmt.Printf("Would download database: %s (%s)\n", project.DB, team.FormatBytes(size))
			fmt.Println("Would import to MySQL")
		} else {
			fmt.Println("Downloading database...")

			tmpDir := os.TempDir()
			localPath := filepath.Join(tmpDir, "magebox-sync-db-"+filepath.Base(project.DB))

			if err := assetClient.Download(project.DB, localPath); err != nil {
				return fmt.Errorf("failed to download database: %w", err)
			}
			defer os.Remove(localPath)

			fmt.Println()
			fmt.Println("Importing database...")

			importCmd := exec.Command("magebox", "db", "import", localPath)
			importCmd.Dir = cwd
			importCmd.Stdout = os.Stdout
			importCmd.Stderr = os.Stderr

			if err := importCmd.Run(); err != nil {
				return fmt.Errorf("failed to import database: %w", err)
			}

			fmt.Println(cli.Success("Database synced!"))
		}
	}

	// Sync media
	if !syncDBOnly && project.Media != "" {
		if syncDryRun {
			size, _ := assetClient.GetFileSize(project.Media)
			fmt.Printf("Would download media: %s (%s)\n", project.Media, team.FormatBytes(size))
			fmt.Println("Would extract to pub/media")
		} else {
			fmt.Println()
			fmt.Println("Downloading media...")

			tmpDir := os.TempDir()
			localPath := filepath.Join(tmpDir, "magebox-sync-media-"+filepath.Base(project.Media))

			if err := assetClient.Download(project.Media, localPath); err != nil {
				return fmt.Errorf("failed to download media: %w", err)
			}
			defer os.Remove(localPath)

			fmt.Println()
			fmt.Println("Extracting media...")

			mediaDir := filepath.Join(cwd, "pub", "media")
			if err := os.MkdirAll(mediaDir, 0755); err != nil {
				return fmt.Errorf("failed to create media directory: %w", err)
			}

			// Use tar command for extraction (faster than Go implementation for large files)
			tarCmd := exec.Command("tar", "-xzf", localPath, "-C", mediaDir)
			tarCmd.Stdout = os.Stdout
			tarCmd.Stderr = os.Stderr

			if err := tarCmd.Run(); err != nil {
				return fmt.Errorf("failed to extract media: %w", err)
			}

			fmt.Println(cli.Success("Media synced!"))
		}
	}

	fmt.Println()
	cli.PrintSuccess("Sync completed!")

	return nil
}

// findProjectForSync finds the team/project for the current directory
func findProjectForSync(cwd, homeDir string) (*team.Team, *team.Project, error) {
	// Load team config
	config, err := team.LoadConfig(homeDir)
	if err != nil {
		return nil, nil, err
	}

	// Try to get git remote origin
	remoteURL := getGitRemoteOrigin(cwd)
	if remoteURL == "" {
		return nil, nil, fmt.Errorf("not a git repository or no remote origin set")
	}

	// Extract repo path from remote URL
	repoPath := extractRepoPath(remoteURL)
	if repoPath == "" {
		return nil, nil, fmt.Errorf("cannot parse git remote: %s", remoteURL)
	}

	// Search for project in all teams
	for _, t := range config.Teams {
		for _, proj := range t.Projects {
			if strings.EqualFold(proj.Repo, repoPath) {
				return t, proj, nil
			}
		}
	}

	return nil, nil, fmt.Errorf("project '%s' not found in any team configuration", repoPath)
}

// getGitRemoteOrigin returns the origin remote URL
func getGitRemoteOrigin(dir string) string {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// extractRepoPath extracts org/repo from git URL
func extractRepoPath(url string) string {
	// Handle SSH URLs: git@github.com:org/repo.git
	if strings.HasPrefix(url, "git@") {
		parts := strings.SplitN(url, ":", 2)
		if len(parts) == 2 {
			return strings.TrimSuffix(parts[1], ".git")
		}
	}

	// Handle HTTPS URLs: https://github.com/org/repo.git
	if strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://") {
		// Remove protocol
		url = strings.TrimPrefix(url, "https://")
		url = strings.TrimPrefix(url, "http://")
		// Remove host
		parts := strings.SplitN(url, "/", 2)
		if len(parts) == 2 {
			return strings.TrimSuffix(parts[1], ".git")
		}
	}

	return ""
}

// backupDatabase creates a backup of the current database
func backupDatabase(projectDir string) error {
	timestamp := time.Now().Format("20060102-150405")
	backupFile := filepath.Join(projectDir, fmt.Sprintf("backup-%s.sql.gz", timestamp))

	fmt.Printf("Backing up database to %s...\n", filepath.Base(backupFile))

	cmd := exec.Command("magebox", "db", "export", backupFile)
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return err
	}

	fmt.Printf("Backup created: %s\n", cli.Success(filepath.Base(backupFile)))
	return nil
}

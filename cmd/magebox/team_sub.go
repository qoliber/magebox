// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/qoliber/magebox/internal/cli"
	"github.com/qoliber/magebox/internal/team"
)

// Team-specific subcommands are registered dynamically
// Usage: magebox team <teamname> <subcommand>

var teamReposCmd = &cobra.Command{
	Use:   "repos",
	Short: "List repositories in the team namespace",
	RunE:  runTeamRepos,
}

var teamReposFilter string

var teamProjectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage team projects",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var teamProjectAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a project to the team",
	Args:  cobra.ExactArgs(1),
	RunE:  runTeamProjectAdd,
}

var teamProjectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List projects in the team",
	RunE:  runTeamProjectList,
}

var teamProjectRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a project from the team",
	Args:  cobra.ExactArgs(1),
	RunE:  runTeamProjectRemove,
}

var (
	projectRepo      string
	projectBranch    string
	projectPHP       string
	projectDB        string
	projectMedia     string
	projectPostFetch []string
)

func init() {
	// Register team-specific subcommands
	// These are accessed via: magebox team <teamname> <subcommand>

	teamReposCmd.Flags().StringVar(&teamReposFilter, "filter", "", "Filter repositories by pattern (e.g., 'mage*')")

	teamProjectAddCmd.Flags().StringVar(&projectRepo, "repo", "", "Repository path (e.g., org/repo)")
	teamProjectAddCmd.Flags().StringVar(&projectBranch, "branch", "", "Default branch")
	teamProjectAddCmd.Flags().StringVar(&projectPHP, "php", "", "PHP version")
	teamProjectAddCmd.Flags().StringVar(&projectDB, "db", "", "Database dump path on asset storage")
	teamProjectAddCmd.Flags().StringVar(&projectMedia, "media", "", "Media archive path on asset storage")
	teamProjectAddCmd.Flags().StringSliceVar(&projectPostFetch, "post-fetch", nil, "Post-fetch commands")

	teamProjectCmd.AddCommand(teamProjectAddCmd)
	teamProjectCmd.AddCommand(teamProjectListCmd)
	teamProjectCmd.AddCommand(teamProjectRemoveCmd)

	// Create dynamic team command that accepts team name as first arg
	teamDynamicCmd := &cobra.Command{
		Use:                "<teamname>",
		Short:              "Team-specific commands",
		DisableFlagParsing: true,
		RunE:               runTeamDynamic,
	}

	teamCmd.AddCommand(teamDynamicCmd)
}

func runTeamDynamic(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return cmd.Help()
	}

	teamName := args[0]
	subArgs := args[1:]

	p, err := getPlatform()
	if err != nil {
		return err
	}

	config, err := team.LoadConfig(p.HomeDir)
	if err != nil {
		return err
	}

	// Check if team exists
	t, err := config.GetTeam(teamName)
	if err != nil {
		return err
	}

	// Store team in context for subcommands
	currentTeam = t
	currentTeamConfig = config

	if len(subArgs) == 0 {
		// Show team by default
		return runTeamShowForTeam(t)
	}

	// Parse subcommand
	switch subArgs[0] {
	case "show":
		return runTeamShowForTeam(t)
	case "repos":
		filter := ""
		if len(subArgs) > 1 {
			for i, arg := range subArgs[1:] {
				if arg == "--filter" && i+2 < len(subArgs) {
					filter = subArgs[i+2]
				}
			}
		}
		return runTeamReposForTeam(t, filter)
	case "project":
		if len(subArgs) < 2 {
			fmt.Println("Usage: magebox team <teamname> project <add|list|remove> [args]")
			return nil
		}
		return runTeamProjectSubcommand(t, config, p.HomeDir, subArgs[1:])
	default:
		return fmt.Errorf("unknown subcommand: %s", subArgs[0])
	}
}

var currentTeam *team.Team
var currentTeamConfig *team.TeamsConfig

func runTeamShowForTeam(t *team.Team) error {
	cli.PrintTitle("Team: %s", t.Name)
	fmt.Println()

	fmt.Println("Repository Configuration")
	fmt.Println("------------------------")
	fmt.Printf("  Provider:     %s\n", t.Repositories.Provider)
	fmt.Printf("  Organization: %s\n", t.Repositories.Organization)
	fmt.Printf("  Auth:         %s\n", t.Repositories.Auth)
	fmt.Println()

	if t.Assets.Provider != "" {
		fmt.Println("Asset Storage")
		fmt.Println("-------------")
		fmt.Printf("  Provider: %s\n", t.Assets.Provider)
		fmt.Printf("  Host:     %s\n", t.Assets.Host)
		fmt.Printf("  Port:     %d\n", t.Assets.GetDefaultPort())
		fmt.Printf("  Path:     %s\n", t.Assets.Path)
		fmt.Printf("  Username: %s\n", t.Assets.Username)
		fmt.Println()
	}

	if len(t.Projects) > 0 {
		fmt.Println("Projects")
		fmt.Println("--------")
		for name, proj := range t.Projects {
			fmt.Printf("  %s\n", cli.Highlight(name))
			fmt.Printf("    Repo:   %s\n", proj.Repo)
			if proj.Branch != "" {
				fmt.Printf("    Branch: %s\n", proj.Branch)
			}
			if proj.PHP != "" {
				fmt.Printf("    PHP:    %s\n", proj.PHP)
			}
			if proj.DB != "" {
				fmt.Printf("    DB:     %s\n", proj.DB)
			}
			if proj.Media != "" {
				fmt.Printf("    Media:  %s\n", proj.Media)
			}
		}
		fmt.Println()
	}

	return nil
}

func runTeamRepos(cmd *cobra.Command, args []string) error {
	if currentTeam == nil {
		return fmt.Errorf("team not set")
	}
	return runTeamReposForTeam(currentTeam, teamReposFilter)
}

func runTeamReposForTeam(t *team.Team, filter string) error {
	cli.PrintTitle("Repositories: %s", t.Repositories.Organization)
	fmt.Println()

	client := team.NewRepositoryClient(t)
	repos, err := client.ListRepositories(filter)
	if err != nil {
		return err
	}

	if len(repos) == 0 {
		cli.PrintInfo("No repositories found")
		if filter != "" {
			cli.PrintInfo("Filter: %s", filter)
		}
		return nil
	}

	for _, repo := range repos {
		visibility := cli.Success("public")
		if repo.Private {
			visibility = cli.Warning("private")
		}
		fmt.Printf("  %s [%s]\n", cli.Highlight(repo.FullName), visibility)
		if repo.Description != "" {
			fmt.Printf("    %s\n", repo.Description)
		}
		fmt.Printf("    Branch: %s\n", repo.DefaultBranch)
	}

	fmt.Println()
	fmt.Printf("Total: %d repositories\n", len(repos))

	return nil
}

func runTeamProjectSubcommand(t *team.Team, config *team.TeamsConfig, homeDir string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("project subcommand required")
	}

	switch args[0] {
	case "add":
		if len(args) < 2 {
			return fmt.Errorf("project name required")
		}
		return runTeamProjectAddForTeam(t, config, homeDir, args[1], args[2:])
	case "list":
		return runTeamProjectListForTeam(t)
	case "remove":
		if len(args) < 2 {
			return fmt.Errorf("project name required")
		}
		return runTeamProjectRemoveForTeam(t, config, homeDir, args[1])
	default:
		return fmt.Errorf("unknown project subcommand: %s", args[0])
	}
}

func runTeamProjectAdd(cmd *cobra.Command, args []string) error {
	if currentTeam == nil || currentTeamConfig == nil {
		return fmt.Errorf("team not set")
	}
	p, _ := getPlatform()
	return runTeamProjectAddForTeam(currentTeam, currentTeamConfig, p.HomeDir, args[0], nil)
}

func runTeamProjectAddForTeam(t *team.Team, config *team.TeamsConfig, homeDir, projectName string, extraArgs []string) error {
	// Check if project already exists
	if _, exists := t.Projects[projectName]; exists {
		return fmt.Errorf("project '%s' already exists in team '%s'", projectName, t.Name)
	}

	// Parse extra args for flags
	repo := projectRepo
	branch := projectBranch
	php := projectPHP
	db := projectDB
	media := projectMedia
	postFetch := projectPostFetch

	// Parse inline flags from extraArgs
	for i := 0; i < len(extraArgs); i++ {
		switch extraArgs[i] {
		case "--repo":
			if i+1 < len(extraArgs) {
				repo = extraArgs[i+1]
				i++
			}
		case "--branch":
			if i+1 < len(extraArgs) {
				branch = extraArgs[i+1]
				i++
			}
		case "--php":
			if i+1 < len(extraArgs) {
				php = extraArgs[i+1]
				i++
			}
		case "--db":
			if i+1 < len(extraArgs) {
				db = extraArgs[i+1]
				i++
			}
		case "--media":
			if i+1 < len(extraArgs) {
				media = extraArgs[i+1]
				i++
			}
		}
	}

	// If repo not provided, prompt for it
	if repo == "" {
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("Repository (e.g., %s/%s): ", t.Repositories.Organization, projectName)
		repo, _ = reader.ReadString('\n')
		repo = strings.TrimSpace(repo)
		if repo == "" {
			repo = fmt.Sprintf("%s/%s", t.Repositories.Organization, projectName)
		}
	}

	project := &team.Project{
		Repo:      repo,
		Branch:    branch,
		PHP:       php,
		DB:        db,
		Media:     media,
		PostFetch: postFetch,
	}

	if err := project.Validate(); err != nil {
		return err
	}

	t.Projects[projectName] = project

	if err := team.SaveConfig(homeDir, config); err != nil {
		return err
	}

	cli.PrintSuccess("Project '%s' added to team '%s'", projectName, t.Name)

	// Show fetch command
	fmt.Println()
	cli.PrintInfo("Fetch with: magebox fetch %s/%s", t.Name, projectName)

	return nil
}

func runTeamProjectList(cmd *cobra.Command, args []string) error {
	if currentTeam == nil {
		return fmt.Errorf("team not set")
	}
	return runTeamProjectListForTeam(currentTeam)
}

func runTeamProjectListForTeam(t *team.Team) error {
	if len(t.Projects) == 0 {
		cli.PrintInfo("No projects in team '%s'", t.Name)
		cli.PrintInfo("Add a project with: magebox team %s project add <name>", t.Name)
		return nil
	}

	cli.PrintTitle("Projects in %s", t.Name)
	fmt.Println()

	for name, proj := range t.Projects {
		fmt.Printf("  %s\n", cli.Highlight(name))
		fmt.Printf("    Repo:   %s\n", proj.Repo)
		if proj.Branch != "" {
			fmt.Printf("    Branch: %s\n", proj.Branch)
		}
		if proj.DB != "" {
			fmt.Printf("    DB:     %s\n", proj.DB)
		}
		if proj.Media != "" {
			fmt.Printf("    Media:  %s\n", proj.Media)
		}
		fmt.Println()
	}

	return nil
}

func runTeamProjectRemove(cmd *cobra.Command, args []string) error {
	if currentTeam == nil || currentTeamConfig == nil {
		return fmt.Errorf("team not set")
	}
	p, _ := getPlatform()
	return runTeamProjectRemoveForTeam(currentTeam, currentTeamConfig, p.HomeDir, args[0])
}

func runTeamProjectRemoveForTeam(t *team.Team, config *team.TeamsConfig, homeDir, projectName string) error {
	if _, exists := t.Projects[projectName]; !exists {
		return fmt.Errorf("project '%s' not found in team '%s'", projectName, t.Name)
	}

	delete(t.Projects, projectName)

	if err := team.SaveConfig(homeDir, config); err != nil {
		return err
	}

	cli.PrintSuccess("Project '%s' removed from team '%s'", projectName, t.Name)
	return nil
}

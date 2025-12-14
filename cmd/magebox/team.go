// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/qoliber/magebox/internal/cli"
	"github.com/qoliber/magebox/internal/team"
)

var teamCmd = &cobra.Command{
	Use:   "team",
	Short: "Manage team configurations",
	Long: `Manage team configurations for collaborative project fetching.

Teams store repository provider settings (GitHub, GitLab, Bitbucket),
asset storage settings (SFTP/FTP), and project configurations.

Examples:
  magebox team add myteam           # Add a new team interactively
  magebox team list                 # List all configured teams
  magebox team myteam show          # Show team configuration
  magebox team myteam repos         # List repositories in namespace
  magebox team remove myteam        # Remove a team`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var teamAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new team configuration",
	Long:  `Adds a new team configuration interactively.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runTeamAdd,
}

var teamListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured teams",
	RunE:  runTeamList,
}

var teamRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a team configuration",
	Args:  cobra.ExactArgs(1),
	RunE:  runTeamRemove,
}

func init() {
	teamCmd.AddCommand(teamAddCmd)
	teamCmd.AddCommand(teamListCmd)
	teamCmd.AddCommand(teamRemoveCmd)
	rootCmd.AddCommand(teamCmd)
}

func runTeamAdd(cmd *cobra.Command, args []string) error {
	teamName := args[0]

	p, err := getPlatform()
	if err != nil {
		return err
	}

	// Load existing config
	config, err := team.LoadConfig(p.HomeDir)
	if err != nil {
		return err
	}

	// Check if team already exists
	if _, exists := config.Teams[teamName]; exists {
		return fmt.Errorf("team '%s' already exists", teamName)
	}

	cli.PrintTitle("Add Team: %s", teamName)
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	// Repository configuration
	fmt.Println("Repository Configuration")
	fmt.Println("------------------------")

	// Provider selection
	fmt.Print("Provider [github/gitlab/bitbucket]: ")
	providerStr, _ := reader.ReadString('\n')
	providerStr = strings.TrimSpace(strings.ToLower(providerStr))
	if providerStr == "" {
		providerStr = "github"
	}
	provider := team.RepositoryProvider(providerStr)
	if provider != team.ProviderGitHub && provider != team.ProviderGitLab && provider != team.ProviderBitbucket {
		return fmt.Errorf("invalid provider: %s", providerStr)
	}

	// Organization
	fmt.Print("Organization/Namespace: ")
	org, _ := reader.ReadString('\n')
	org = strings.TrimSpace(org)
	if org == "" {
		return fmt.Errorf("organization is required")
	}

	// Auth method
	fmt.Print("Auth method [ssh/token]: ")
	authStr, _ := reader.ReadString('\n')
	authStr = strings.TrimSpace(strings.ToLower(authStr))
	if authStr == "" {
		authStr = "ssh"
	}
	auth := team.AuthMethod(authStr)
	if auth != team.AuthSSH && auth != team.AuthToken {
		return fmt.Errorf("invalid auth method: %s", authStr)
	}

	fmt.Println()

	// Asset configuration
	fmt.Println("Asset Storage Configuration")
	fmt.Println("---------------------------")

	fmt.Print("Provider [sftp/ftp] (leave empty to skip): ")
	assetProviderStr, _ := reader.ReadString('\n')
	assetProviderStr = strings.TrimSpace(strings.ToLower(assetProviderStr))

	var assetConfig team.AssetConfig
	if assetProviderStr != "" {
		assetProvider := team.AssetProvider(assetProviderStr)
		if assetProvider != team.AssetSFTP && assetProvider != team.AssetFTP {
			return fmt.Errorf("invalid asset provider: %s", assetProviderStr)
		}
		assetConfig.Provider = assetProvider

		fmt.Print("Host: ")
		host, _ := reader.ReadString('\n')
		assetConfig.Host = strings.TrimSpace(host)
		if assetConfig.Host == "" {
			return fmt.Errorf("asset host is required")
		}

		fmt.Print("Port (default 22 for SFTP, 21 for FTP): ")
		portStr, _ := reader.ReadString('\n')
		portStr = strings.TrimSpace(portStr)
		if portStr != "" {
			port, err := strconv.Atoi(portStr)
			if err != nil {
				return fmt.Errorf("invalid port: %s", portStr)
			}
			assetConfig.Port = port
		}

		fmt.Print("Path (remote base path): ")
		path, _ := reader.ReadString('\n')
		assetConfig.Path = strings.TrimSpace(path)

		fmt.Print("Username: ")
		username, _ := reader.ReadString('\n')
		assetConfig.Username = strings.TrimSpace(username)
		if assetConfig.Username == "" {
			return fmt.Errorf("asset username is required")
		}
	}

	// Create team
	newTeam := &team.Team{
		Name: teamName,
		Repositories: team.RepositoryConfig{
			Provider:     provider,
			Organization: org,
			Auth:         auth,
		},
		Assets:   assetConfig,
		Projects: make(map[string]*team.Project),
	}

	if err := config.AddTeam(newTeam); err != nil {
		return err
	}

	if err := team.SaveConfig(p.HomeDir, config); err != nil {
		return err
	}

	fmt.Println()
	cli.PrintSuccess("Team '%s' added successfully!", teamName)
	fmt.Println()

	// Show environment variable hints
	teamKey := strings.ToUpper(strings.ReplaceAll(teamName, "-", "_"))
	if auth == team.AuthToken {
		cli.PrintInfo("Set git token: export MAGEBOX_%s_TOKEN=your-token", teamKey)
	}
	if assetConfig.Provider != "" {
		cli.PrintInfo("Set asset credentials:")
		cli.PrintInfo("  SSH key: export MAGEBOX_%s_ASSET_KEY=/path/to/key", teamKey)
		cli.PrintInfo("  Or password: export MAGEBOX_%s_ASSET_PASS=your-password", teamKey)
	}

	fmt.Println()
	cli.PrintInfo("Add projects with: magebox team %s project add <name>", teamName)

	return nil
}

func runTeamList(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	config, err := team.LoadConfig(p.HomeDir)
	if err != nil {
		return err
	}

	if len(config.Teams) == 0 {
		cli.PrintInfo("No teams configured")
		cli.PrintInfo("Add a team with: magebox team add <name>")
		return nil
	}

	cli.PrintTitle("Configured Teams")
	fmt.Println()

	for name, t := range config.Teams {
		fmt.Printf("  %s\n", cli.Highlight(name))
		fmt.Printf("    Repository: %s (%s/%s)\n", t.Repositories.Provider, t.Repositories.Organization, t.Repositories.Auth)
		if t.Assets.Provider != "" {
			fmt.Printf("    Assets:     %s://%s@%s:%d%s\n",
				t.Assets.Provider, t.Assets.Username, t.Assets.Host,
				t.Assets.GetDefaultPort(), t.Assets.Path)
		}
		fmt.Printf("    Projects:   %d\n", len(t.Projects))
		fmt.Println()
	}

	return nil
}

func runTeamRemove(cmd *cobra.Command, args []string) error {
	teamName := args[0]

	p, err := getPlatform()
	if err != nil {
		return err
	}

	config, err := team.LoadConfig(p.HomeDir)
	if err != nil {
		return err
	}

	if err := config.RemoveTeam(teamName); err != nil {
		return err
	}

	if err := team.SaveConfig(p.HomeDir, config); err != nil {
		return err
	}

	cli.PrintSuccess("Team '%s' removed", teamName)
	return nil
}

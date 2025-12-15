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
	DisableFlagParsing: true,
	RunE:               runTeamCmd,
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

// Flags for team add command
var (
	teamAddProvider      string
	teamAddOrg           string
	teamAddAuth          string
	teamAddAssetProvider string
	teamAddAssetHost     string
	teamAddAssetPort     int
	teamAddAssetPath     string
	teamAddAssetUsername string
)

func init() {
	// Add flags for non-interactive team creation
	teamAddCmd.Flags().StringVar(&teamAddProvider, "provider", "", "Repository provider (github, gitlab, bitbucket)")
	teamAddCmd.Flags().StringVar(&teamAddOrg, "org", "", "Organization/namespace")
	teamAddCmd.Flags().StringVar(&teamAddAuth, "auth", "https", "Auth method (ssh, https, token)")
	teamAddCmd.Flags().StringVar(&teamAddAssetProvider, "asset-provider", "", "Asset storage provider (sftp, ftp)")
	teamAddCmd.Flags().StringVar(&teamAddAssetHost, "asset-host", "", "Asset storage host")
	teamAddCmd.Flags().IntVar(&teamAddAssetPort, "asset-port", 0, "Asset storage port")
	teamAddCmd.Flags().StringVar(&teamAddAssetPath, "asset-path", "", "Asset storage base path")
	teamAddCmd.Flags().StringVar(&teamAddAssetUsername, "asset-username", "", "Asset storage username")

	teamCmd.AddCommand(teamAddCmd)
	teamCmd.AddCommand(teamListCmd)
	teamCmd.AddCommand(teamRemoveCmd)
	rootCmd.AddCommand(teamCmd)
}

// runTeamCmd handles the team command and routes to subcommands or team-specific actions
func runTeamCmd(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return cmd.Help()
	}

	// Check for known subcommands first
	switch args[0] {
	case "add":
		// Parse flags and run add command
		if err := teamAddCmd.ParseFlags(args[1:]); err != nil {
			return err
		}
		remainingArgs := teamAddCmd.Flags().Args()
		if len(remainingArgs) != 1 {
			return fmt.Errorf("team add requires exactly one argument: team name")
		}
		return runTeamAdd(teamAddCmd, remainingArgs)
	case "list":
		return runTeamList(teamListCmd, args[1:])
	case "remove":
		if len(args) < 2 {
			return fmt.Errorf("team remove requires a team name")
		}
		return runTeamRemove(teamRemoveCmd, args[1:])
	case "-h", "--help", "help":
		return cmd.Help()
	}

	// If not a known subcommand, treat first arg as team name
	return runTeamDynamic(cmd, args)
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

	// Check if we have enough flags for non-interactive mode
	nonInteractive := teamAddProvider != "" && teamAddOrg != ""

	var providerStr, org, authStr string
	var assetProviderStr, assetHost, assetPath, assetUsername string
	var assetPort int

	if nonInteractive {
		// Use flag values
		providerStr = strings.ToLower(teamAddProvider)
		org = teamAddOrg
		authStr = strings.ToLower(teamAddAuth)
		if authStr == "" {
			authStr = "ssh"
		}
		assetProviderStr = strings.ToLower(teamAddAssetProvider)
		assetHost = teamAddAssetHost
		assetPort = teamAddAssetPort
		assetPath = teamAddAssetPath
		assetUsername = teamAddAssetUsername
	} else {
		// Interactive mode
		cli.PrintTitle("Add Team: %s", teamName)
		fmt.Println()

		reader := bufio.NewReader(os.Stdin)

		// Repository configuration
		fmt.Println("Repository Configuration")
		fmt.Println("------------------------")

		// Provider selection
		fmt.Print("Provider [github/gitlab/bitbucket]: ")
		providerStr, _ = reader.ReadString('\n')
		providerStr = strings.TrimSpace(strings.ToLower(providerStr))
		if providerStr == "" {
			providerStr = "github"
		}

		// Organization
		fmt.Print("Organization/Namespace: ")
		org, _ = reader.ReadString('\n')
		org = strings.TrimSpace(org)
		if org == "" {
			return fmt.Errorf("organization is required")
		}

		// Auth method
		fmt.Print("Auth method [ssh/token]: ")
		authStr, _ = reader.ReadString('\n')
		authStr = strings.TrimSpace(strings.ToLower(authStr))
		if authStr == "" {
			authStr = "ssh"
		}

		fmt.Println()

		// Asset configuration
		fmt.Println("Asset Storage Configuration")
		fmt.Println("---------------------------")

		fmt.Print("Provider [sftp/ftp] (leave empty to skip): ")
		assetProviderStr, _ = reader.ReadString('\n')
		assetProviderStr = strings.TrimSpace(strings.ToLower(assetProviderStr))

		if assetProviderStr != "" {
			fmt.Print("Host: ")
			assetHost, _ = reader.ReadString('\n')
			assetHost = strings.TrimSpace(assetHost)

			fmt.Print("Port (default 22 for SFTP, 21 for FTP): ")
			portStr, _ := reader.ReadString('\n')
			portStr = strings.TrimSpace(portStr)
			if portStr != "" {
				assetPort, _ = strconv.Atoi(portStr)
			}

			fmt.Print("Path (remote base path): ")
			assetPath, _ = reader.ReadString('\n')
			assetPath = strings.TrimSpace(assetPath)

			fmt.Print("Username: ")
			assetUsername, _ = reader.ReadString('\n')
			assetUsername = strings.TrimSpace(assetUsername)
		}
	}

	// Validate provider
	provider := team.RepositoryProvider(providerStr)
	if provider != team.ProviderGitHub && provider != team.ProviderGitLab && provider != team.ProviderBitbucket {
		return fmt.Errorf("invalid provider: %s (use github, gitlab, or bitbucket)", providerStr)
	}

	// Validate auth
	auth := team.AuthMethod(authStr)
	if auth != team.AuthSSH && auth != team.AuthHTTPS && auth != team.AuthToken {
		return fmt.Errorf("invalid auth method: %s (use ssh, https, or token)", authStr)
	}

	// Build asset config
	var assetConfig team.AssetConfig
	if assetProviderStr != "" {
		assetProvider := team.AssetProvider(assetProviderStr)
		if assetProvider != team.AssetSFTP && assetProvider != team.AssetFTP {
			return fmt.Errorf("invalid asset provider: %s (use sftp or ftp)", assetProviderStr)
		}
		if assetHost == "" {
			return fmt.Errorf("asset host is required when asset provider is set")
		}
		if assetUsername == "" {
			return fmt.Errorf("asset username is required when asset provider is set")
		}
		assetConfig = team.AssetConfig{
			Provider: assetProvider,
			Host:     assetHost,
			Port:     assetPort,
			Path:     assetPath,
			Username: assetUsername,
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

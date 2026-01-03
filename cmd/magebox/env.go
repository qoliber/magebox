// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"qoliber/magebox/internal/cli"
	"qoliber/magebox/internal/config"
	"qoliber/magebox/internal/remote"
	"qoliber/magebox/internal/teamserver"
	"github.com/spf13/cobra"
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage remote environments",
	Long: `Manage remote server environments for SSH access.

Environments are stored globally and can be used to quickly SSH into remote servers.
Supports custom SSH keys and SSH tunnel configurations.`,
	RunE: runEnvList,
}

var envAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new remote environment",
	Long: `Add a new remote environment configuration.

Examples:
  magebox env add staging --user deploy --host staging.example.com
  magebox env add production --user deploy --host prod.example.com --port 2222
  magebox env add cloud --user magento --host 10.0.0.1 --key ~/.ssh/cloud_key
  magebox env add tunnel --ssh-command "ssh -J jump@bastion.example.com deploy@internal.example.com"`,
	Args: cobra.ExactArgs(1),
	RunE: runEnvAdd,
}

var envRemoveCmd = &cobra.Command{
	Use:     "remove <name>",
	Aliases: []string{"rm", "delete"},
	Short:   "Remove a remote environment",
	Args:    cobra.ExactArgs(1),
	RunE:    runEnvRemove,
}

var envSSHCmd = &cobra.Command{
	Use:   "ssh <name>",
	Short: "SSH into a remote environment",
	Long: `SSH into a configured remote environment.

Example:
  magebox env ssh staging
  magebox env ssh production`,
	Args: cobra.ExactArgs(1),
	RunE: runEnvSSH,
}

var envShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show details of a remote environment",
	Args:  cobra.ExactArgs(1),
	RunE:  runEnvShow,
}

var envSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync environments from team server",
	Long: `Sync the list of accessible environments from the team server.

This updates the local cache of environments you have access to.
Run this after your access has been updated on the server.

Example:
  magebox env sync`,
	RunE: runEnvSync,
}

// Flags for env add
var (
	envUser       string
	envHost       string
	envPort       int
	envSSHKey     string
	envSSHCommand string
)

func init() {
	// Add subcommands
	envCmd.AddCommand(envAddCmd)
	envCmd.AddCommand(envRemoveCmd)
	envCmd.AddCommand(envSSHCmd)
	envCmd.AddCommand(envShowCmd)
	envCmd.AddCommand(envSyncCmd)

	// Add flags to env add
	envAddCmd.Flags().StringVarP(&envUser, "user", "u", "", "SSH username (required)")
	envAddCmd.Flags().StringVarP(&envHost, "host", "H", "", "SSH host/IP (required unless using --ssh-command)")
	envAddCmd.Flags().IntVarP(&envPort, "port", "p", 22, "SSH port")
	envAddCmd.Flags().StringVarP(&envSSHKey, "key", "k", "", "Path to SSH private key")
	envAddCmd.Flags().StringVar(&envSSHCommand, "ssh-command", "", "Custom SSH command (for tunnels/jump hosts)")

	// Add to root command
	rootCmd.AddCommand(envCmd)
}

func runEnvList(_ *cobra.Command, _ []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	globalCfg, err := config.LoadGlobalConfig(homeDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	envs := globalCfg.Environments
	if len(envs) == 0 {
		fmt.Println("No remote environments configured.")
		fmt.Println()
		fmt.Println("Add an environment with:")
		fmt.Println("  magebox env add <name> --user <user> --host <host>")
		return nil
	}

	fmt.Println("Remote Environments:")
	fmt.Println()

	// Calculate column widths
	maxName := 4  // "NAME"
	maxConn := 10 // "CONNECTION"
	for _, env := range envs {
		if len(env.Name) > maxName {
			maxName = len(env.Name)
		}
		conn := env.GetConnectionString()
		if len(conn) > maxConn {
			maxConn = len(conn)
		}
	}

	// Print header
	fmt.Printf("  %-*s  %s\n", maxName, "NAME", "CONNECTION")
	fmt.Printf("  %s  %s\n", strings.Repeat("-", maxName), strings.Repeat("-", maxConn))

	// Print environments
	for _, env := range envs {
		fmt.Printf("  %-*s  %s\n", maxName, env.Name, env.GetConnectionString())
	}

	fmt.Println()
	fmt.Println("SSH into an environment with: magebox env ssh <name>")

	return nil
}

func runEnvAdd(_ *cobra.Command, args []string) error {
	name := args[0]

	// Validate inputs
	if envSSHCommand == "" {
		// Standard SSH connection - require user and host
		if envUser == "" {
			return fmt.Errorf("--user is required")
		}
		if envHost == "" {
			return fmt.Errorf("--host is required")
		}
	}

	// Expand SSH key path if provided
	sshKeyPath := envSSHKey
	if sshKeyPath != "" {
		if strings.HasPrefix(sshKeyPath, "~") {
			homeDir, _ := os.UserHomeDir()
			sshKeyPath = strings.Replace(sshKeyPath, "~", homeDir, 1)
		}
	}

	env := remote.Environment{
		Name:       name,
		User:       envUser,
		Host:       envHost,
		Port:       envPort,
		SSHKeyPath: sshKeyPath,
		SSHCommand: envSSHCommand,
	}

	// For custom SSH commands, validation is different
	if envSSHCommand != "" {
		// Only name and command are required
		env.User = envUser // May be empty
		env.Host = envHost // May be empty
	} else {
		// Standard validation
		if err := env.Validate(); err != nil {
			return fmt.Errorf("invalid environment: %w", err)
		}
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	globalCfg, err := config.LoadGlobalConfig(homeDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check for duplicate
	for _, e := range globalCfg.Environments {
		if e.Name == name {
			return fmt.Errorf("environment '%s' already exists", name)
		}
	}

	globalCfg.Environments = append(globalCfg.Environments, env)

	if err := config.SaveGlobalConfig(homeDir, globalCfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("[OK] Added environment '%s'\n", name)
	if envSSHCommand != "" {
		fmt.Printf("     Command: %s\n", envSSHCommand)
	} else {
		fmt.Printf("     Connection: %s\n", env.GetConnectionString())
	}
	fmt.Println()
	fmt.Printf("SSH into it with: magebox env ssh %s\n", name)

	return nil
}

func runEnvRemove(_ *cobra.Command, args []string) error {
	name := args[0]

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	globalCfg, err := config.LoadGlobalConfig(homeDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := globalCfg.RemoveEnvironment(name); err != nil {
		return err
	}

	if err := config.SaveGlobalConfig(homeDir, globalCfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("[OK] Removed environment '%s'\n", name)
	return nil
}

func runEnvSSH(_ *cobra.Command, args []string) error {
	name := args[0]

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	globalCfg, err := config.LoadGlobalConfig(homeDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	env, err := globalCfg.GetEnvironment(name)
	if err != nil {
		return err
	}

	fmt.Printf("Connecting to %s...\n", name)

	cmd := env.BuildSSHCommand()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func runEnvShow(_ *cobra.Command, args []string) error {
	name := args[0]

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	globalCfg, err := config.LoadGlobalConfig(homeDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	env, err := globalCfg.GetEnvironment(name)
	if err != nil {
		return err
	}

	fmt.Printf("Environment: %s\n", env.Name)
	fmt.Println()

	if env.SSHCommand != "" {
		fmt.Printf("  SSH Command: %s\n", env.SSHCommand)
	} else {
		fmt.Printf("  User: %s\n", env.User)
		fmt.Printf("  Host: %s\n", env.Host)
		fmt.Printf("  Port: %d\n", env.GetPort())
		if env.SSHKeyPath != "" {
			fmt.Printf("  SSH Key: %s\n", env.SSHKeyPath)
		}
	}

	fmt.Println()
	fmt.Println("SSH Command:")
	cmd := env.BuildSSHCommand()
	fmt.Printf("  %s %s\n", cmd.Path, strings.Join(cmd.Args[1:], " "))

	return nil
}

func runEnvSync(_ *cobra.Command, _ []string) error {
	// Load client config
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(homeDir, ".magebox", "teamserver", "client.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("not connected to a team server. Use 'magebox server join' first")
		}
		return err
	}

	var clientCfg clientConfig
	if err := json.Unmarshal(data, &clientCfg); err != nil {
		return err
	}

	cli.PrintInfo("Syncing environments from %s...", clientCfg.ServerURL)

	// Fetch environments from server
	req, err := http.NewRequest("GET", clientCfg.ServerURL+"/api/environments", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+clientCfg.SessionToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("session expired. Rejoin with: magebox server join")
	}

	if resp.StatusCode != http.StatusOK {
		var errResp teamserver.ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("failed to fetch environments: %s", errResp.Error)
	}

	var envs []struct {
		Name       string `json:"name"`
		Project    string `json:"project"`
		Host       string `json:"host"`
		Port       int    `json:"port"`
		DeployUser string `json:"deploy_user"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&envs); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Update client config with new environments
	clientCfg.Environments = make([]clientEnvironmentConfig, len(envs))
	for i, env := range envs {
		clientCfg.Environments[i] = clientEnvironmentConfig{
			Name:       env.Name,
			Project:    env.Project,
			Host:       env.Host,
			Port:       env.Port,
			DeployUser: env.DeployUser,
		}
	}

	// Save updated config
	updatedData, err := json.MarshalIndent(clientCfg, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(configPath, updatedData, 0600); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	cli.PrintSuccess("Synced %d environment(s)", len(envs))

	if len(envs) > 0 {
		fmt.Println()
		cli.PrintInfo("Available environments:")
		for _, env := range envs {
			fmt.Printf("  - %s (%s@%s)\n", env.Name, env.DeployUser, env.Host)
		}
		fmt.Println()
		cli.PrintInfo("Connect with: magebox ssh <environment-name>")
	}

	return nil
}

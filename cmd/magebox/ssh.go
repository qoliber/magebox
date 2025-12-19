/**
 * Created by Qoliber
 *
 * @category    Qoliber
 * @package     MageBox
 * @author      Jakub Winkler <jwinkler@qoliber.com>
 */

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/qoliber/magebox/internal/cli"
	"github.com/spf13/cobra"
)

var sshCmd = &cobra.Command{
	Use:   "ssh <environment>",
	Short: "SSH into a team server environment",
	Long: `SSH into an environment from your team server.

This command uses the SSH key generated when you joined the team server.
Environment names are in the format: project/environment (e.g., myproject/staging)

Examples:
  magebox ssh myproject/staging
  magebox ssh myproject/production`,
	Args: cobra.ExactArgs(1),
	RunE: runSSH,
}

func init() {
	rootCmd.AddCommand(sshCmd)
}

func runSSH(cmd *cobra.Command, args []string) error {
	envName := args[0]

	// Load client config
	config, err := loadTeamServerConfig()
	if err != nil {
		return err
	}

	// Find the environment
	var targetEnv *clientEnvironmentConfig
	for i, env := range config.Environments {
		if env.Name == envName {
			targetEnv = &config.Environments[i]
			break
		}
		// Also try matching just the environment name without project prefix
		parts := strings.Split(env.Name, "/")
		if len(parts) == 2 && parts[1] == envName {
			targetEnv = &config.Environments[i]
			break
		}
	}

	if targetEnv == nil {
		// List available environments
		cli.PrintError("Environment '%s' not found", envName)
		if len(config.Environments) > 0 {
			fmt.Println()
			cli.PrintInfo("Available environments:")
			for _, env := range config.Environments {
				fmt.Printf("  - %s (%s@%s)\n", env.Name, env.DeployUser, env.Host)
			}
		}
		fmt.Println()
		cli.PrintInfo("To sync environments from server: magebox env sync")
		return fmt.Errorf("environment not found")
	}

	// Check SSH key exists
	if config.KeyPath == "" {
		return fmt.Errorf("no SSH key configured. Rejoin the team server with: magebox server join")
	}
	if _, err := os.Stat(config.KeyPath); os.IsNotExist(err) {
		return fmt.Errorf("SSH key not found at %s. Rejoin the team server with: magebox server join", config.KeyPath)
	}

	// Build SSH command
	port := targetEnv.Port
	if port == 0 {
		port = 22
	}

	sshArgs := []string{
		"-i", config.KeyPath,
		"-p", strconv.Itoa(port),
		"-o", "StrictHostKeyChecking=accept-new",
		fmt.Sprintf("%s@%s", targetEnv.DeployUser, targetEnv.Host),
	}

	cli.PrintInfo("Connecting to %s (%s@%s:%d)...", targetEnv.Name, targetEnv.DeployUser, targetEnv.Host, port)

	sshPath, err := exec.LookPath("ssh")
	if err != nil {
		return fmt.Errorf("ssh not found in PATH: %w", err)
	}

	sshExec := exec.Command(sshPath, sshArgs...)
	sshExec.Stdin = os.Stdin
	sshExec.Stdout = os.Stdout
	sshExec.Stderr = os.Stderr

	return sshExec.Run()
}

// loadTeamServerConfig loads the team server client config
func loadTeamServerConfig() (*clientConfig, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(homeDir, ".magebox", "teamserver", "client.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("not connected to a team server. Use 'magebox server join' first")
		}
		return nil, err
	}

	var config clientConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

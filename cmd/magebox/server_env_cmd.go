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
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"qoliber/magebox/internal/cli"
	"qoliber/magebox/internal/teamserver"
)

var (
	serverEnvHost       string
	serverEnvPort       int
	serverEnvDeployUser string
	serverEnvDeployKey  string
	serverEnvProject    string
)

var serverEnvCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage team server environments",
	Long: `Manage remote environments on the MageBox team server.

Environments define remote servers that team members can access.
Each environment belongs to a project. Users granted access to a project
can access all environments within that project.

Examples:
  magebox server env list
  magebox server env add production --project myproject --host prod.example.com --deploy-user deploy --deploy-key ~/.ssh/deploy_prod
  magebox server env remove myproject/staging`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var serverEnvAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new environment",
	Long: `Add a new environment to a project on the team server.

The deploy key is the SSH private key used to connect to the server.
The environment must belong to an existing project.

Examples:
  magebox server env add production --project myproject --host prod.example.com --deploy-user deploy --deploy-key ~/.ssh/deploy_key
  magebox server env add staging --project myproject --host staging.example.com --deploy-user deploy --deploy-key ~/.ssh/deploy_key`,
	Args: cobra.ExactArgs(1),
	RunE: runServerEnvAdd,
}

var serverEnvRemoveCmd = &cobra.Command{
	Use:   "remove <project/name>",
	Short: "Remove an environment",
	Long: `Remove an environment from the team server.

This removes the environment configuration. Team members will no longer
have access once the keys are synced.

Examples:
  magebox server env remove myproject/staging`,
	Args: cobra.ExactArgs(1),
	RunE: runServerEnvRemove,
}

var serverEnvListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all environments",
	Long:  `List all environments configured on the team server.`,
	RunE:  runServerEnvList,
}

var serverEnvShowCmd = &cobra.Command{
	Use:   "show <project/name>",
	Short: "Show environment details",
	Long: `Show detailed information about a specific environment.

Examples:
  magebox server env show myproject/production`,
	Args: cobra.ExactArgs(1),
	RunE: runServerEnvShow,
}

var serverEnvSyncCmd = &cobra.Command{
	Use:   "sync [name]",
	Short: "Sync SSH keys to environments",
	Long: `Synchronize SSH public keys to remote environments.

This deploys the public keys of authorized users to the
authorized_keys file on each environment.

Examples:
  magebox server env sync              # Sync all environments
  magebox server env sync production   # Sync specific environment`,
	RunE: runServerEnvSync,
}

func init() {
	// Environment add flags
	serverEnvAddCmd.Flags().StringVar(&serverEnvProject, "project", "", "Project this environment belongs to (required)")
	serverEnvAddCmd.Flags().StringVar(&serverEnvHost, "host", "", "Environment hostname (required)")
	serverEnvAddCmd.Flags().IntVar(&serverEnvPort, "port", 22, "SSH port")
	serverEnvAddCmd.Flags().StringVar(&serverEnvDeployUser, "deploy-user", "deploy", "Deploy username")
	serverEnvAddCmd.Flags().StringVar(&serverEnvDeployKey, "deploy-key", "", "Path to deploy SSH private key (required)")
	_ = serverEnvAddCmd.MarkFlagRequired("project")
	_ = serverEnvAddCmd.MarkFlagRequired("host")
	_ = serverEnvAddCmd.MarkFlagRequired("deploy-key")

	serverEnvCmd.AddCommand(serverEnvAddCmd)
	serverEnvCmd.AddCommand(serverEnvRemoveCmd)
	serverEnvCmd.AddCommand(serverEnvListCmd)
	serverEnvCmd.AddCommand(serverEnvShowCmd)
	serverEnvCmd.AddCommand(serverEnvSyncCmd)

	serverCmd.AddCommand(serverEnvCmd)
}

func runServerEnvAdd(cmd *cobra.Command, args []string) error {
	envName := args[0]

	adminToken, err := getAdminToken()
	if err != nil {
		return err
	}

	// Read deploy key
	keyData, err := os.ReadFile(serverEnvDeployKey)
	if err != nil {
		return fmt.Errorf("failed to read deploy key: %w", err)
	}
	keyContent := string(keyData)

	// Validate it looks like a private key
	if !strings.Contains(keyContent, "PRIVATE KEY") {
		return fmt.Errorf("deploy-key should be a private key file (not .pub)")
	}

	reqBody := map[string]interface{}{
		"name":        envName,
		"project":     serverEnvProject,
		"host":        serverEnvHost,
		"port":        serverEnvPort,
		"deploy_user": serverEnvDeployUser,
		"deploy_key":  keyContent,
	}

	resp, err := apiRequest("POST", "/api/admin/environments", reqBody, adminToken)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp teamserver.ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("failed to create environment: %s", errResp.Error)
	}

	var result struct {
		Name       string `json:"name"`
		Project    string `json:"project"`
		Host       string `json:"host"`
		Port       int    `json:"port"`
		DeployUser string `json:"deploy_user"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	cli.PrintSuccess("Environment '%s/%s' created!", result.Project, envName)
	fmt.Println()
	cli.PrintInfo("Project:     %s", result.Project)
	cli.PrintInfo("Host:        %s:%d", result.Host, result.Port)
	cli.PrintInfo("Deploy User: %s", result.DeployUser)
	fmt.Println()
	cli.PrintInfo("Run 'magebox server env sync %s/%s' to deploy SSH keys", result.Project, envName)

	return nil
}

func runServerEnvRemove(cmd *cobra.Command, args []string) error {
	envPath := args[0]

	// Parse project/name format
	parts := strings.SplitN(envPath, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("environment must be specified as project/name (e.g., myproject/staging)")
	}

	adminToken, err := getAdminToken()
	if err != nil {
		return err
	}

	resp, err := apiRequest("DELETE", "/api/admin/environments/"+envPath, nil, adminToken)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp teamserver.ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("failed to remove environment: %s", errResp.Error)
	}

	cli.PrintSuccess("Environment '%s' removed", envPath)
	cli.PrintWarning("Note: User SSH keys on this server will need to be cleaned up manually")

	return nil
}

func runServerEnvList(cmd *cobra.Command, args []string) error {
	// Try admin token first, fall back to user session
	var token string
	var endpoint string

	adminToken, err := getAdminToken()
	if err == nil {
		token = adminToken
		endpoint = "/api/admin/environments"
	} else {
		// Try as regular user
		config, err := loadClientConfig()
		if err != nil {
			return fmt.Errorf("not authenticated. Use 'magebox server join' or set MAGEBOX_ADMIN_TOKEN")
		}
		token = config.SessionToken
		endpoint = "/api/environments"
	}

	resp, err := apiRequest("GET", endpoint, nil, token)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp teamserver.ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("failed to list environments: %s", errResp.Error)
	}

	var envs []struct {
		Name       string    `json:"name"`
		Project    string    `json:"project"`
		Host       string    `json:"host"`
		Port       int       `json:"port"`
		DeployUser string    `json:"deploy_user"`
		CreatedAt  time.Time `json:"created_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&envs); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if len(envs) == 0 {
		cli.PrintInfo("No environments configured")
		cli.PrintInfo("Add an environment with: magebox server env add <name> --project <project> --host <host> --deploy-key <key>")
		return nil
	}

	cli.PrintTitle("Environments")
	fmt.Println()

	// Group environments by project
	projectEnvs := make(map[string][]struct {
		Name       string    `json:"name"`
		Project    string    `json:"project"`
		Host       string    `json:"host"`
		Port       int       `json:"port"`
		DeployUser string    `json:"deploy_user"`
		CreatedAt  time.Time `json:"created_at"`
	})

	for _, env := range envs {
		projectEnvs[env.Project] = append(projectEnvs[env.Project], env)
	}

	for project, envList := range projectEnvs {
		fmt.Printf("  %s\n", cli.Highlight(project))
		for _, env := range envList {
			fmt.Printf("    └─ %s\n", env.Name)
			fmt.Printf("         Host: %s:%d  User: %s\n", env.Host, env.Port, env.DeployUser)
		}
		fmt.Println()
	}

	fmt.Printf("Total: %d environments in %d projects\n", len(envs), len(projectEnvs))

	return nil
}

func runServerEnvShow(cmd *cobra.Command, args []string) error {
	envPath := args[0]

	// Parse project/name format
	parts := strings.SplitN(envPath, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("environment must be specified as project/name (e.g., myproject/staging)")
	}
	projectName := parts[0]

	adminToken, err := getAdminToken()
	if err != nil {
		return err
	}

	resp, err := apiRequest("GET", "/api/admin/environments/"+envPath, nil, adminToken)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp teamserver.ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("failed to get environment: %s", errResp.Error)
	}

	var env struct {
		ID         int64     `json:"id"`
		Name       string    `json:"name"`
		Project    string    `json:"project"`
		Host       string    `json:"host"`
		Port       int       `json:"port"`
		DeployUser string    `json:"deploy_user"`
		CreatedAt  time.Time `json:"created_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	cli.PrintTitle("Environment: %s/%s", env.Project, env.Name)
	fmt.Println()
	fmt.Printf("  Project:     %s\n", env.Project)
	fmt.Printf("  Host:        %s\n", env.Host)
	fmt.Printf("  Port:        %d\n", env.Port)
	fmt.Printf("  Deploy User: %s\n", env.DeployUser)
	fmt.Printf("  Created:     %s\n", env.CreatedAt.Format("2006-01-02 15:04"))

	// Show which users have access to this project
	fmt.Println()
	cli.PrintInfo("Users with access (via project '%s'):", projectName)

	usersResp, err := apiRequest("GET", "/api/admin/users", nil, adminToken)
	if err == nil && usersResp.StatusCode == http.StatusOK {
		var users []struct {
			Name     string   `json:"name"`
			Role     string   `json:"role"`
			Projects []string `json:"projects"`
		}
		json.NewDecoder(usersResp.Body).Decode(&users)
		usersResp.Body.Close()

		accessCount := 0
		for _, u := range users {
			for _, p := range u.Projects {
				if p == projectName {
					fmt.Printf("    - %s (%s)\n", u.Name, u.Role)
					accessCount++
					break
				}
			}
		}

		if accessCount == 0 {
			fmt.Printf("    (no users)\n")
		}
	}

	return nil
}

func runServerEnvSync(cmd *cobra.Command, args []string) error {
	adminToken, err := getAdminToken()
	if err != nil {
		return err
	}

	var envName string
	if len(args) > 0 {
		envName = args[0]
	}

	cli.PrintInfo("Starting key synchronization...")
	fmt.Println()

	resp, err := apiRequest("POST", "/api/admin/sync", map[string]string{"environment": envName}, adminToken)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp teamserver.ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("sync failed: %s", errResp.Error)
	}

	var result teamserver.SuccessResponse
	json.NewDecoder(resp.Body).Decode(&result)

	cli.PrintSuccess("Key synchronization completed!")
	if result.Message != "" {
		cli.PrintInfo("%s", result.Message)
	}

	return nil
}

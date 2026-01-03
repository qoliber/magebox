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
	"time"

	"github.com/spf13/cobra"

	"qoliber/magebox/internal/cli"
	"qoliber/magebox/internal/teamserver"
)

var (
	serverProjectDescription string
)

var serverProjectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage team server projects",
	Long: `Manage projects on the MageBox team server.

Projects are containers for environments. Users are granted access
to projects, which gives them access to all environments within that project.

Examples:
  magebox server project list
  magebox server project add myproject --description "My Project"
  magebox server project remove myproject`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var serverProjectAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new project",
	Long: `Add a new project to the team server.

Projects are containers for environments. After creating a project,
you can add environments to it and grant users access.

Examples:
  magebox server project add myproject
  magebox server project add myproject --description "Production servers"`,
	Args: cobra.ExactArgs(1),
	RunE: runServerProjectAdd,
}

var serverProjectRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a project",
	Long: `Remove a project from the team server.

WARNING: This will also remove all environments within the project
and revoke user access to those environments.

Examples:
  magebox server project remove myproject`,
	Args: cobra.ExactArgs(1),
	RunE: runServerProjectRemove,
}

var serverProjectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all projects",
	Long:  `List all projects configured on the team server.`,
	RunE:  runServerProjectList,
}

var serverProjectShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show project details",
	Long:  `Show detailed information about a specific project including its environments.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runServerProjectShow,
}

func init() {
	// Project add flags
	serverProjectAddCmd.Flags().StringVar(&serverProjectDescription, "description", "", "Project description")

	serverProjectCmd.AddCommand(serverProjectAddCmd)
	serverProjectCmd.AddCommand(serverProjectRemoveCmd)
	serverProjectCmd.AddCommand(serverProjectListCmd)
	serverProjectCmd.AddCommand(serverProjectShowCmd)

	serverCmd.AddCommand(serverProjectCmd)
}

func runServerProjectAdd(cmd *cobra.Command, args []string) error {
	projectName := args[0]

	adminToken, err := getAdminToken()
	if err != nil {
		return err
	}

	reqBody := map[string]interface{}{
		"name":        projectName,
		"description": serverProjectDescription,
	}

	resp, err := apiRequest("POST", "/api/admin/projects", reqBody, adminToken)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp teamserver.ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("failed to create project: %s", errResp.Error)
	}

	var result struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	cli.PrintSuccess("Project '%s' created!", projectName)
	if result.Description != "" {
		cli.PrintInfo("Description: %s", result.Description)
	}
	fmt.Println()
	cli.PrintInfo("Next steps:")
	cli.PrintInfo("  1. Add environments: magebox server env add staging --project %s --host staging.example.com --deploy-key ~/.ssh/deploy_key", projectName)
	cli.PrintInfo("  2. Grant user access: magebox server user grant <username> --project %s", projectName)

	return nil
}

func runServerProjectRemove(cmd *cobra.Command, args []string) error {
	projectName := args[0]

	adminToken, err := getAdminToken()
	if err != nil {
		return err
	}

	resp, err := apiRequest("DELETE", "/api/admin/projects/"+projectName, nil, adminToken)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp teamserver.ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("failed to remove project: %s", errResp.Error)
	}

	cli.PrintSuccess("Project '%s' removed", projectName)
	cli.PrintWarning("Note: All environments in this project have been removed")

	return nil
}

func runServerProjectList(cmd *cobra.Command, args []string) error {
	adminToken, err := getAdminToken()
	if err != nil {
		return err
	}

	resp, err := apiRequest("GET", "/api/admin/projects", nil, adminToken)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp teamserver.ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("failed to list projects: %s", errResp.Error)
	}

	var projects []struct {
		Name        string    `json:"name"`
		Description string    `json:"description"`
		CreatedAt   time.Time `json:"created_at"`
		CreatedBy   string    `json:"created_by"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if len(projects) == 0 {
		cli.PrintInfo("No projects configured")
		cli.PrintInfo("Add a project with: magebox server project add <name>")
		return nil
	}

	cli.PrintTitle("Projects")
	fmt.Println()

	for _, project := range projects {
		fmt.Printf("  %s\n", cli.Highlight(project.Name))
		if project.Description != "" {
			fmt.Printf("      Description: %s\n", project.Description)
		}
		fmt.Printf("      Created:     %s", project.CreatedAt.Format("2006-01-02 15:04"))
		if project.CreatedBy != "" {
			fmt.Printf(" by %s", project.CreatedBy)
		}
		fmt.Println()
		fmt.Println()
	}

	fmt.Printf("Total: %d projects\n", len(projects))

	return nil
}

func runServerProjectShow(cmd *cobra.Command, args []string) error {
	projectName := args[0]

	adminToken, err := getAdminToken()
	if err != nil {
		return err
	}

	// Get project details
	resp, err := apiRequest("GET", "/api/admin/projects/"+projectName, nil, adminToken)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp teamserver.ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("failed to get project: %s", errResp.Error)
	}

	var project struct {
		ID          int64     `json:"id"`
		Name        string    `json:"name"`
		Description string    `json:"description"`
		CreatedAt   time.Time `json:"created_at"`
		CreatedBy   string    `json:"created_by"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&project); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	cli.PrintTitle("Project: %s", project.Name)
	fmt.Println()
	if project.Description != "" {
		fmt.Printf("  Description: %s\n", project.Description)
	}
	fmt.Printf("  Created:     %s", project.CreatedAt.Format("2006-01-02 15:04"))
	if project.CreatedBy != "" {
		fmt.Printf(" by %s", project.CreatedBy)
	}
	fmt.Println()

	// Get environments in this project
	envResp, err := apiRequest("GET", "/api/admin/environments", nil, adminToken)
	if err == nil && envResp.StatusCode == http.StatusOK {
		var envs []struct {
			Name    string `json:"name"`
			Project string `json:"project"`
			Host    string `json:"host"`
			Port    int    `json:"port"`
		}
		json.NewDecoder(envResp.Body).Decode(&envs)
		envResp.Body.Close()

		// Filter environments for this project
		var projectEnvs []struct {
			Name    string `json:"name"`
			Project string `json:"project"`
			Host    string `json:"host"`
			Port    int    `json:"port"`
		}
		for _, env := range envs {
			if env.Project == projectName {
				projectEnvs = append(projectEnvs, env)
			}
		}

		fmt.Println()
		cli.PrintInfo("Environments (%d):", len(projectEnvs))
		if len(projectEnvs) == 0 {
			fmt.Printf("    (no environments)\n")
		} else {
			for _, env := range projectEnvs {
				fmt.Printf("    - %s (%s:%d)\n", env.Name, env.Host, env.Port)
			}
		}
	}

	// Get users with access to this project
	usersResp, err := apiRequest("GET", "/api/admin/users", nil, adminToken)
	if err == nil && usersResp.StatusCode == http.StatusOK {
		var users []struct {
			Name     string   `json:"name"`
			Role     string   `json:"role"`
			Projects []string `json:"projects"`
		}
		json.NewDecoder(usersResp.Body).Decode(&users)
		usersResp.Body.Close()

		var accessUsers []string
		for _, u := range users {
			for _, p := range u.Projects {
				if p == projectName {
					accessUsers = append(accessUsers, fmt.Sprintf("%s (%s)", u.Name, u.Role))
					break
				}
			}
		}

		fmt.Println()
		cli.PrintInfo("Users with access (%d):", len(accessUsers))
		if len(accessUsers) == 0 {
			fmt.Printf("    (no users)\n")
		} else {
			for _, u := range accessUsers {
				fmt.Printf("    - %s\n", u)
			}
		}
	}

	return nil
}

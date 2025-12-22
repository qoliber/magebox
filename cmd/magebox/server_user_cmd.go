/**
 * Created by Qoliber
 *
 * @category    Qoliber
 * @package     MageBox
 * @author      Jakub Winkler <jwinkler@qoliber.com>
 */

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/qoliber/magebox/internal/cli"
	"github.com/qoliber/magebox/internal/teamserver"
)

// validateServerURL validates that the URL is safe to connect to
// Returns the normalized URL or an error if the URL is invalid or unsafe
func validateServerURL(serverURLArg string) (string, error) {
	// Normalize URL
	if !strings.HasPrefix(serverURLArg, "http://") && !strings.HasPrefix(serverURLArg, "https://") {
		serverURLArg = "https://" + serverURLArg
	}
	serverURLArg = strings.TrimSuffix(serverURLArg, "/")

	// Parse and validate URL structure
	parsedURL, err := url.Parse(serverURLArg)
	if err != nil {
		return "", fmt.Errorf("invalid URL format: %w", err)
	}

	// Ensure scheme is http or https
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", fmt.Errorf("URL scheme must be http or https, got: %s", parsedURL.Scheme)
	}

	// Warn about http (but allow it for development)
	if parsedURL.Scheme == "http" {
		cli.PrintWarning("Using unencrypted HTTP connection - not recommended for production")
	}

	// Validate hostname is not empty
	host := parsedURL.Hostname()
	if host == "" {
		return "", fmt.Errorf("URL must include a hostname")
	}

	// Check for localhost/private IPs (warn but allow for development)
	if isPrivateHost(host) {
		cli.PrintWarning("Connecting to a private/local address - ensure this is intended")
	}

	return serverURLArg, nil
}

// isPrivateHost checks if a host is a private/local address
func isPrivateHost(host string) bool {
	// Check for localhost
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}

	// Try to parse as IP
	ip := net.ParseIP(host)
	if ip == nil {
		// Not an IP, could be a hostname - we can't check without DNS resolution
		// which could leak information, so we allow it
		return false
	}

	// Check for private IP ranges
	privateBlocks := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"fc00::/7",  // IPv6 private
		"fe80::/10", // IPv6 link-local
		"::1/128",   // IPv6 loopback
	}

	for _, block := range privateBlocks {
		_, network, err := net.ParseCIDR(block)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}

	return false
}

var (
	userEmail      string
	userRole       string
	userExpiryDays int
	inviteToken    string
	userProject    string
)

var serverUserCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage team server users",
	Long: `Manage users on the MageBox team server.

These commands require admin authentication to the team server.

Examples:
  magebox server user list
  magebox server user add developer --email dev@example.com --role dev
  magebox server user remove developer
  magebox server user renew developer --expires 90`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var serverUserAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new user (creates invite)",
	Long: `Create a new user invitation on the team server.

The invite token returned should be sent to the user so they can join.

Examples:
  magebox server user add alice --email alice@example.com --role dev
  magebox server user add bob --email bob@example.com --role readonly --expires 30`,
	Args: cobra.ExactArgs(1),
	RunE: runServerUserAdd,
}

var serverUserRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a user",
	Long: `Remove a user from the team server.

This revokes their access immediately and their SSH keys will be
removed from all environments on the next sync.

Examples:
  magebox server user remove alice`,
	Args: cobra.ExactArgs(1),
	RunE: runServerUserRemove,
}

var serverUserListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all users",
	Long:  `List all users registered on the team server.`,
	RunE:  runServerUserList,
}

var serverUserShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show user details",
	Long:  `Show detailed information about a specific user.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runServerUserShow,
}

var serverUserRenewCmd = &cobra.Command{
	Use:   "renew <name>",
	Short: "Renew user access",
	Long: `Extend a user's access expiration.

Examples:
  magebox server user renew alice --expires 90`,
	Args: cobra.ExactArgs(1),
	RunE: runServerUserRenew,
}

var serverUserGrantCmd = &cobra.Command{
	Use:   "grant <name>",
	Short: "Grant project access to a user",
	Long: `Grant a user access to a project.

Users can access all environments within a project they have access to.

Examples:
  magebox server user grant alice --project myproject`,
	Args: cobra.ExactArgs(1),
	RunE: runServerUserGrant,
}

var serverUserRevokeCmd = &cobra.Command{
	Use:   "revoke <name>",
	Short: "Revoke project access from a user",
	Long: `Revoke a user's access to a project.

The user will no longer be able to access environments in this project.

Examples:
  magebox server user revoke alice --project myproject`,
	Args: cobra.ExactArgs(1),
	RunE: runServerUserRevoke,
}

var serverJoinCmd = &cobra.Command{
	Use:   "join <server-url>",
	Short: "Join a team server",
	Long: `Join a MageBox team server using an invite token.

The server generates a unique SSH key pair for you. The private key is
downloaded and saved locally. You can then use 'magebox ssh <env>' to
connect to environments you have access to.

Examples:
  magebox server join https://teamserver.example.com --token <invite-token>`,
	Args: cobra.ExactArgs(1),
	RunE: runServerJoin,
}

var serverWhoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show current user info",
	Long:  `Show information about the currently authenticated user.`,
	RunE:  runServerWhoami,
}

func init() {
	// User add flags
	serverUserAddCmd.Flags().StringVar(&userEmail, "email", "", "User email address (required)")
	serverUserAddCmd.Flags().StringVar(&userRole, "role", "dev", "User role: admin, dev, readonly")
	serverUserAddCmd.Flags().IntVar(&userExpiryDays, "expires", 0, "Access expiry in days (0 = use default)")
	_ = serverUserAddCmd.MarkFlagRequired("email")

	// User renew flags
	serverUserRenewCmd.Flags().IntVar(&userExpiryDays, "expires", 90, "New expiry in days from now")

	// User grant/revoke flags
	serverUserGrantCmd.Flags().StringVar(&userProject, "project", "", "Project to grant access to (required)")
	_ = serverUserGrantCmd.MarkFlagRequired("project")
	serverUserRevokeCmd.Flags().StringVar(&userProject, "project", "", "Project to revoke access from (required)")
	_ = serverUserRevokeCmd.MarkFlagRequired("project")

	// Join flags
	serverJoinCmd.Flags().StringVar(&inviteToken, "token", "", "Invite token (required)")
	_ = serverJoinCmd.MarkFlagRequired("token")

	serverUserCmd.AddCommand(serverUserAddCmd)
	serverUserCmd.AddCommand(serverUserRemoveCmd)
	serverUserCmd.AddCommand(serverUserListCmd)
	serverUserCmd.AddCommand(serverUserShowCmd)
	serverUserCmd.AddCommand(serverUserRenewCmd)
	serverUserCmd.AddCommand(serverUserGrantCmd)
	serverUserCmd.AddCommand(serverUserRevokeCmd)

	serverCmd.AddCommand(serverUserCmd)
	serverCmd.AddCommand(serverJoinCmd)
	serverCmd.AddCommand(serverWhoamiCmd)
}

// Client configuration for connecting to team server
type clientConfig struct {
	ServerURL    string                    `json:"server_url"`
	SessionToken string                    `json:"session_token"`
	UserName     string                    `json:"user_name"`
	Role         string                    `json:"role"`
	JoinedAt     string                    `json:"joined_at"`
	KeyPath      string                    `json:"key_path"`     // Path to private SSH key (used by older config)
	KeyFile      string                    `json:"key_file"`     // Path to private SSH key (for cert command)
	CAEnabled    bool                      `json:"ca_enabled"`   // Whether SSH CA is enabled
	Environments []clientEnvironmentConfig `json:"environments"` // Accessible environments
}

// clientEnvironmentConfig stores environment info for SSH connections
type clientEnvironmentConfig struct {
	Name       string `json:"name"` // Full name: project/env
	Project    string `json:"project"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	DeployUser string `json:"deploy_user"`
}

func getClientConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".magebox", "teamserver", "client.json"), nil
}

func loadClientConfig() (*clientConfig, error) {
	path, err := getClientConfigPath()
	if err != nil {
		return nil, err
	}

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

func saveClientConfig(config *clientConfig) error {
	path, err := getClientConfigPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

func getAdminToken() (string, error) {
	// Check environment variable first
	if token := os.Getenv("MAGEBOX_ADMIN_TOKEN"); token != "" {
		return token, nil
	}

	// Check server config file
	dataDir, err := getServerDataDir()
	if err != nil {
		return "", err
	}

	configPath := filepath.Join(dataDir, "server.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("admin token not found. Set MAGEBOX_ADMIN_TOKEN or run 'magebox server init'")
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return "", err
	}

	// Note: We can't retrieve the original token from the hash
	return "", fmt.Errorf("admin token required. Set MAGEBOX_ADMIN_TOKEN environment variable")
}

func getServerBaseURL() (string, error) {
	// Check if connected as client
	clientCfg, err := loadClientConfig()
	if err == nil && clientCfg.ServerURL != "" {
		return clientCfg.ServerURL, nil
	}

	// Default to local server
	return "http://localhost:7443", nil
}

func apiRequest(method, endpoint string, body interface{}, token string) (*http.Response, error) {
	baseURL, err := getServerBaseURL()
	if err != nil {
		return nil, err
	}

	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequest(method, baseURL+endpoint, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	return client.Do(req)
}

func runServerUserAdd(cmd *cobra.Command, args []string) error {
	userName := args[0]

	adminToken, err := getAdminToken()
	if err != nil {
		return err
	}

	// Validate role
	role := teamserver.Role(userRole)
	if !role.IsValid() {
		return fmt.Errorf("invalid role '%s'. Valid roles: admin, dev, readonly", userRole)
	}

	reqBody := map[string]interface{}{
		"name":  userName,
		"email": userEmail,
		"role":  userRole,
	}
	if userExpiryDays > 0 {
		reqBody["expiry_days"] = userExpiryDays
	}

	resp, err := apiRequest("POST", "/api/admin/users", reqBody, adminToken)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp teamserver.ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("failed to create user: %s", errResp.Error)
	}

	var result struct {
		User struct {
			Name  string `json:"name"`
			Email string `json:"email"`
			Role  string `json:"role"`
		} `json:"user"`
		InviteToken string `json:"invite_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	cli.PrintSuccess("User invitation created!")
	fmt.Println()
	cli.PrintInfo("User:  %s", result.User.Name)
	cli.PrintInfo("Email: %s", result.User.Email)
	cli.PrintInfo("Role:  %s", result.User.Role)
	fmt.Println()
	cli.PrintTitle("Invite Token")
	fmt.Println()
	fmt.Printf("  %s\n", result.InviteToken)
	fmt.Println()
	cli.PrintWarning("Send this token to the user. It can only be used once!")
	fmt.Println()
	cli.PrintInfo("User should run:")
	fmt.Printf("  magebox server join <server-url> --token %s\n", result.InviteToken)

	return nil
}

func runServerUserRemove(cmd *cobra.Command, args []string) error {
	userName := args[0]

	adminToken, err := getAdminToken()
	if err != nil {
		return err
	}

	resp, err := apiRequest("DELETE", "/api/admin/users/"+userName, nil, adminToken)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp teamserver.ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("failed to remove user: %s", errResp.Error)
	}

	cli.PrintSuccess("User '%s' removed", userName)
	cli.PrintInfo("Their access has been revoked. Run 'magebox server sync' to update environment keys.")

	return nil
}

func runServerUserList(cmd *cobra.Command, args []string) error {
	adminToken, err := getAdminToken()
	if err != nil {
		return err
	}

	resp, err := apiRequest("GET", "/api/admin/users", nil, adminToken)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp teamserver.ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("failed to list users: %s", errResp.Error)
	}

	var users []struct {
		Name         string     `json:"name"`
		Email        string     `json:"email"`
		Role         string     `json:"role"`
		MFAEnabled   bool       `json:"mfa_enabled"`
		ExpiresAt    *time.Time `json:"expires_at"`
		CreatedAt    time.Time  `json:"created_at"`
		LastAccessAt *time.Time `json:"last_access_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if len(users) == 0 {
		cli.PrintInfo("No users registered")
		cli.PrintInfo("Create a user with: magebox server user add <name> --email <email>")
		return nil
	}

	cli.PrintTitle("Team Server Users")
	fmt.Println()

	for _, u := range users {
		// Status indicator
		status := cli.Success("●")
		if u.ExpiresAt != nil && time.Now().After(*u.ExpiresAt) {
			status = cli.Error("●")
		}

		fmt.Printf("  %s %s <%s>\n", status, cli.Highlight(u.Name), u.Email)
		fmt.Printf("      Role: %s", u.Role)
		if u.MFAEnabled {
			fmt.Printf(" [MFA]")
		}
		fmt.Println()

		if u.ExpiresAt != nil {
			if time.Now().After(*u.ExpiresAt) {
				fmt.Printf("      Expires: %s (EXPIRED)\n", cli.Error(u.ExpiresAt.Format("2006-01-02")))
			} else {
				daysLeft := int(time.Until(*u.ExpiresAt).Hours() / 24)
				fmt.Printf("      Expires: %s (%d days left)\n", u.ExpiresAt.Format("2006-01-02"), daysLeft)
			}
		}

		if u.LastAccessAt != nil {
			fmt.Printf("      Last access: %s\n", u.LastAccessAt.Format("2006-01-02 15:04"))
		}
		fmt.Println()
	}

	fmt.Printf("Total: %d users\n", len(users))

	return nil
}

func runServerUserShow(cmd *cobra.Command, args []string) error {
	userName := args[0]

	adminToken, err := getAdminToken()
	if err != nil {
		return err
	}

	resp, err := apiRequest("GET", "/api/admin/users/"+userName, nil, adminToken)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp teamserver.ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("failed to get user: %s", errResp.Error)
	}

	var user struct {
		ID           int64      `json:"id"`
		Name         string     `json:"name"`
		Email        string     `json:"email"`
		Role         string     `json:"role"`
		PublicKey    string     `json:"public_key"`
		MFAEnabled   bool       `json:"mfa_enabled"`
		ExpiresAt    *time.Time `json:"expires_at"`
		CreatedAt    time.Time  `json:"created_at"`
		CreatedBy    string     `json:"created_by"`
		LastAccessAt *time.Time `json:"last_access_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	cli.PrintTitle("User: %s", user.Name)
	fmt.Println()

	fmt.Printf("  Email:      %s\n", user.Email)
	fmt.Printf("  Role:       %s\n", user.Role)
	fmt.Printf("  MFA:        %v\n", user.MFAEnabled)
	fmt.Printf("  Created:    %s by %s\n", user.CreatedAt.Format("2006-01-02 15:04"), user.CreatedBy)

	if user.ExpiresAt != nil {
		if time.Now().After(*user.ExpiresAt) {
			fmt.Printf("  Expires:    %s (EXPIRED)\n", cli.Error(user.ExpiresAt.Format("2006-01-02")))
		} else {
			daysLeft := int(time.Until(*user.ExpiresAt).Hours() / 24)
			fmt.Printf("  Expires:    %s (%d days left)\n", user.ExpiresAt.Format("2006-01-02"), daysLeft)
		}
	} else {
		fmt.Printf("  Expires:    Never\n")
	}

	if user.LastAccessAt != nil {
		fmt.Printf("  Last access: %s\n", user.LastAccessAt.Format("2006-01-02 15:04"))
	} else {
		fmt.Printf("  Last access: Never\n")
	}

	if user.PublicKey != "" {
		fmt.Println()
		fmt.Println("  Public Key:")
		// Truncate for display
		keyDisplay := user.PublicKey
		if len(keyDisplay) > 80 {
			keyDisplay = keyDisplay[:77] + "..."
		}
		fmt.Printf("    %s\n", keyDisplay)
	}

	return nil
}

func runServerUserRenew(cmd *cobra.Command, args []string) error {
	userName := args[0]

	adminToken, err := getAdminToken()
	if err != nil {
		return err
	}

	reqBody := map[string]interface{}{
		"expiry_days": userExpiryDays,
	}

	resp, err := apiRequest("PUT", "/api/admin/users/"+userName, reqBody, adminToken)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp teamserver.ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("failed to renew user: %s", errResp.Error)
	}

	newExpiry := time.Now().AddDate(0, 0, userExpiryDays)
	cli.PrintSuccess("User '%s' access renewed until %s", userName, newExpiry.Format("2006-01-02"))

	return nil
}

func runServerJoin(cmd *cobra.Command, args []string) error {
	// Validate and normalize the server URL
	serverURLArg, err := validateServerURL(args[0])
	if err != nil {
		return fmt.Errorf("invalid server URL: %w", err)
	}

	cli.PrintInfo("Joining team server: %s", serverURLArg)
	fmt.Println()

	// Make join request (server will generate SSH key pair)
	reqBody := map[string]string{
		"invite_token": inviteToken,
	}

	jsonData, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", serverURLArg+"/api/join", bytes.NewReader(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp teamserver.ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("join failed: %s", errResp.Error)
	}

	var result struct {
		SessionToken string     `json:"session_token"`
		PrivateKey   string     `json:"private_key"`
		Certificate  string     `json:"certificate,omitempty"`
		ValidUntil   *time.Time `json:"valid_until,omitempty"`
		Principals   []string   `json:"principals,omitempty"`
		CAEnabled    bool       `json:"ca_enabled"`
		CAPublicKey  string     `json:"ca_public_key,omitempty"`
		ServerHost   string     `json:"server_host"`
		User         struct {
			Name string `json:"name"`
			Role string `json:"role"`
		} `json:"user"`
		Environments []struct {
			Name       string `json:"name"`
			Project    string `json:"project"`
			Host       string `json:"host"`
			Port       int    `json:"port"`
			DeployUser string `json:"deploy_user"`
		} `json:"environments"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Save SSH private key
	homeDir, _ := os.UserHomeDir()
	keysDir := filepath.Join(homeDir, ".magebox", "keys")
	if err := os.MkdirAll(keysDir, 0700); err != nil {
		return fmt.Errorf("failed to create keys directory: %w", err)
	}

	// Create safe filename from server host
	safeHost := strings.ReplaceAll(result.ServerHost, ":", "_")
	safeHost = strings.ReplaceAll(safeHost, "/", "_")
	keyFileName := fmt.Sprintf("%s_%s.key", safeHost, result.User.Name)
	keyPath := filepath.Join(keysDir, keyFileName)

	if err := os.WriteFile(keyPath, []byte(result.PrivateKey), 0600); err != nil {
		return fmt.Errorf("failed to save SSH private key: %w", err)
	}

	// Save certificate if provided
	var certPath string
	if result.Certificate != "" {
		certPath = keyPath + "-cert.pub"
		if err := os.WriteFile(certPath, []byte(result.Certificate+"\n"), 0644); err != nil {
			cli.PrintWarning("Failed to save SSH certificate: %v", err)
		}
	}

	// Convert environments to client config format
	envConfigs := make([]clientEnvironmentConfig, len(result.Environments))
	for i, env := range result.Environments {
		envConfigs[i] = clientEnvironmentConfig{
			Name:       env.Name,
			Project:    env.Project,
			Host:       env.Host,
			Port:       env.Port,
			DeployUser: env.DeployUser,
		}
	}

	// Save client config
	config := &clientConfig{
		ServerURL:    serverURLArg,
		SessionToken: result.SessionToken,
		UserName:     result.User.Name,
		Role:         result.User.Role,
		JoinedAt:     time.Now().Format(time.RFC3339),
		KeyPath:      keyPath,
		KeyFile:      keyPath,
		CAEnabled:    result.CAEnabled,
		Environments: envConfigs,
	}

	if err := saveClientConfig(config); err != nil {
		cli.PrintWarning("Failed to save session: %v", err)
	}

	cli.PrintSuccess("Successfully joined team server!")
	fmt.Println()
	cli.PrintInfo("User: %s", result.User.Name)
	cli.PrintInfo("Role: %s", result.User.Role)
	cli.PrintInfo("SSH Key: %s", keyPath)

	// Show certificate info if CA is enabled
	if result.CAEnabled && certPath != "" {
		cli.PrintInfo("Certificate: %s", certPath)
		if result.ValidUntil != nil {
			cli.PrintInfo("Valid Until: %s", result.ValidUntil.Format("2006-01-02 15:04:05"))
		}
		if len(result.Principals) > 0 {
			cli.PrintInfo("Principals: %v", result.Principals)
		}
		fmt.Println()
		cli.PrintInfo("Certificate expires automatically. Renew with: magebox cert renew")
	}

	if len(result.Environments) > 0 {
		fmt.Println()
		cli.PrintInfo("Accessible environments:")
		for _, env := range result.Environments {
			fmt.Printf("  - %s (%s@%s)\n", env.Name, env.DeployUser, env.Host)
		}
		fmt.Println()
		cli.PrintInfo("Connect with: magebox ssh <environment-name>")
	}

	fmt.Println()
	cli.PrintInfo("Your session has been saved. Use 'magebox server whoami' to check status.")

	return nil
}

func runServerWhoami(cmd *cobra.Command, args []string) error {
	config, err := loadClientConfig()
	if err != nil {
		return err
	}

	// Verify session is still valid
	resp, err := apiRequest("GET", "/api/me", nil, config.SessionToken)
	if err != nil {
		return fmt.Errorf("failed to verify session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("session expired or invalid. Use 'magebox server join' to reconnect")
	}

	if resp.StatusCode != http.StatusOK {
		var errResp teamserver.ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("failed to get user info: %s", errResp.Error)
	}

	var user struct {
		Name       string     `json:"name"`
		Email      string     `json:"email"`
		Role       string     `json:"role"`
		MFAEnabled bool       `json:"mfa_enabled"`
		ExpiresAt  *time.Time `json:"expires_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	cli.PrintTitle("Current User")
	fmt.Println()
	fmt.Printf("  Server:  %s\n", config.ServerURL)
	fmt.Printf("  User:    %s\n", user.Name)
	fmt.Printf("  Email:   %s\n", user.Email)
	fmt.Printf("  Role:    %s\n", user.Role)

	if user.ExpiresAt != nil {
		daysLeft := int(time.Until(*user.ExpiresAt).Hours() / 24)
		if daysLeft < 0 {
			fmt.Printf("  Status:  %s\n", cli.Error("EXPIRED"))
		} else if daysLeft < 7 {
			fmt.Printf("  Status:  %s (%d days left)\n", cli.Warning("Expiring soon"), daysLeft)
		} else {
			fmt.Printf("  Status:  %s (%d days left)\n", cli.Success("Active"), daysLeft)
		}
	} else {
		fmt.Printf("  Status:  %s\n", cli.Success("Active (no expiry)"))
	}

	// Get accessible environments
	envResp, err := apiRequest("GET", "/api/environments", nil, config.SessionToken)
	if err == nil && envResp.StatusCode == http.StatusOK {
		var envs []struct {
			Name string `json:"name"`
			Host string `json:"host"`
		}
		json.NewDecoder(envResp.Body).Decode(&envs)
		envResp.Body.Close()

		if len(envs) > 0 {
			fmt.Println()
			cli.PrintInfo("Accessible environments:")
			for _, env := range envs {
				fmt.Printf("    - %s (%s)\n", env.Name, env.Host)
			}
		}
	}

	return nil
}

func runServerUserGrant(cmd *cobra.Command, args []string) error {
	userName := args[0]

	adminToken, err := getAdminToken()
	if err != nil {
		return err
	}

	reqBody := map[string]interface{}{
		"project": userProject,
	}

	resp, err := apiRequest("POST", "/api/admin/users/"+userName+"/access", reqBody, adminToken)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp teamserver.ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("failed to grant access: %s", errResp.Error)
	}

	cli.PrintSuccess("Granted '%s' access to project '%s'", userName, userProject)
	cli.PrintInfo("Run 'magebox server env sync' to deploy SSH keys to environments")

	return nil
}

func runServerUserRevoke(cmd *cobra.Command, args []string) error {
	userName := args[0]

	adminToken, err := getAdminToken()
	if err != nil {
		return err
	}

	reqBody := map[string]interface{}{
		"project": userProject,
	}

	resp, err := apiRequest("DELETE", "/api/admin/users/"+userName+"/access", reqBody, adminToken)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp teamserver.ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("failed to revoke access: %s", errResp.Error)
	}

	cli.PrintSuccess("Revoked '%s' access from project '%s'", userName, userProject)
	cli.PrintInfo("Run 'magebox server env sync' to update SSH keys on environments")

	return nil
}

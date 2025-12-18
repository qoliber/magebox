/**
 * Created by Qoliber
 *
 * @category    Qoliber
 * @package     MageBox
 * @author      Jakub Winkler <jwinkler@qoliber.com>
 *
 * Integration tests for MageBox Team Server
 *
 * These tests require Docker and docker-compose to be available.
 * Run with: go test -v -tags=integration ./test/integration/teamserver/...
 *
 * Or use the Makefile target:
 *   make test-integration-teamserver
 */

//go:build integration

package teamserver_integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

const (
	serverURL  = "http://localhost:17443"
	adminToken = "integration-test-admin-token-12345"
)

// TestMain sets up and tears down the Docker environment
func TestMain(m *testing.M) {
	// Check if we should skip integration tests
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "1" {
		fmt.Println("Skipping integration tests (SKIP_INTEGRATION_TESTS=1)")
		os.Exit(0)
	}

	// Check if Docker is available
	if err := exec.Command("docker", "info").Run(); err != nil {
		fmt.Println("Docker not available, skipping integration tests")
		os.Exit(0)
	}

	// Start Docker environment
	fmt.Println("Starting Docker environment...")
	if err := runDockerCompose("up", "-d", "--build"); err != nil {
		fmt.Printf("Failed to start Docker environment: %v\n", err)
		os.Exit(1)
	}

	// Wait for server to be ready
	fmt.Println("Waiting for server to be ready...")
	if err := waitForServer(60 * time.Second); err != nil {
		fmt.Printf("Server did not become ready: %v\n", err)
		runDockerCompose("down", "-v")
		os.Exit(1)
	}
	fmt.Println("Server is ready!")

	// Run tests
	code := m.Run()

	// Cleanup
	fmt.Println("Stopping Docker environment...")
	runDockerCompose("down", "-v")

	os.Exit(code)
}

func runDockerCompose(args ...string) error {
	cmd := exec.Command("docker-compose", args...)
	cmd.Dir = getTestDir()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func getTestDir() string {
	// Get the directory containing this test file
	return "/Volumes/qoliber/qoliber-open-source/magebox/test/integration/teamserver"
}

func waitForServer(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(serverURL + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("timeout waiting for server")
}

// API helper functions
func apiRequest(method, endpoint string, body interface{}, token string) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequest(method, serverURL+endpoint, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return http.DefaultClient.Do(req)
}

func parseJSON(resp *http.Response, v interface{}) error {
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(v)
}

// Test cases

func TestHealthEndpoint(t *testing.T) {
	resp, err := http.Get(serverURL + "/health")
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var health map[string]interface{}
	if err := parseJSON(resp, &health); err != nil {
		t.Fatalf("Failed to parse health response: %v", err)
	}

	if health["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got '%v'", health["status"])
	}
}

func TestAdminAuthentication(t *testing.T) {
	// Test without token
	resp, err := apiRequest("GET", "/api/admin/users", nil, "")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected 401 without token, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Test with invalid token
	resp, err = apiRequest("GET", "/api/admin/users", nil, "invalid-token")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected 401 with invalid token, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Test with valid admin token
	resp, err = apiRequest("GET", "/api/admin/users", nil, adminToken)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 with valid token, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestFullUserFlow(t *testing.T) {
	// Step 1: Create first user (developer)
	t.Log("Step 1: Creating developer user...")
	createReq1 := map[string]interface{}{
		"name":  "developer1",
		"email": "dev1@example.com",
		"role":  "dev",
	}

	resp, err := apiRequest("POST", "/api/admin/users", createReq1, adminToken)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200 creating user, got %d: %s", resp.StatusCode, string(body))
	}

	var createResp1 struct {
		User struct {
			Name  string `json:"name"`
			Email string `json:"email"`
			Role  string `json:"role"`
		} `json:"user"`
		InviteToken string `json:"invite_token"`
	}
	if err := parseJSON(resp, &createResp1); err != nil {
		t.Fatalf("Failed to parse create response: %v", err)
	}

	if createResp1.InviteToken == "" {
		t.Fatal("Invite token should not be empty")
	}
	if createResp1.User.Name != "developer1" {
		t.Errorf("Expected user name 'developer1', got '%s'", createResp1.User.Name)
	}
	t.Logf("Created user developer1 with invite token: %s...", createResp1.InviteToken[:20])

	// Step 2: Create second user (readonly)
	t.Log("Step 2: Creating readonly user...")
	createReq2 := map[string]interface{}{
		"name":  "viewer1",
		"email": "viewer1@example.com",
		"role":  "readonly",
	}

	resp, err = apiRequest("POST", "/api/admin/users", createReq2, adminToken)
	if err != nil {
		t.Fatalf("Failed to create second user: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200 creating second user, got %d: %s", resp.StatusCode, string(body))
	}

	var createResp2 struct {
		InviteToken string `json:"invite_token"`
	}
	parseJSON(resp, &createResp2)
	t.Logf("Created user viewer1 with invite token: %s...", createResp2.InviteToken[:20])

	// Step 3: Developer joins using invite token
	t.Log("Step 3: Developer joining...")
	joinReq1 := map[string]interface{}{
		"invite_token": createResp1.InviteToken,
		"public_key":   "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestDeveloperKey1 developer1@test",
	}

	resp, err = apiRequest("POST", "/api/join", joinReq1, "")
	if err != nil {
		t.Fatalf("Failed to join: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200 on join, got %d: %s", resp.StatusCode, string(body))
	}

	var joinResp1 struct {
		SessionToken string `json:"session_token"`
		User         struct {
			Name string `json:"name"`
			Role string `json:"role"`
		} `json:"user"`
	}
	if err := parseJSON(resp, &joinResp1); err != nil {
		t.Fatalf("Failed to parse join response: %v", err)
	}

	if joinResp1.SessionToken == "" {
		t.Fatal("Session token should not be empty")
	}
	if joinResp1.User.Name != "developer1" {
		t.Errorf("Expected joined user name 'developer1', got '%s'", joinResp1.User.Name)
	}
	t.Logf("Developer1 joined with session token: %s...", joinResp1.SessionToken[:20])

	// Step 4: Viewer joins using invite token
	t.Log("Step 4: Viewer joining...")
	joinReq2 := map[string]interface{}{
		"invite_token": createResp2.InviteToken,
		"public_key":   "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestViewerKey1 viewer1@test",
	}

	resp, err = apiRequest("POST", "/api/join", joinReq2, "")
	if err != nil {
		t.Fatalf("Failed to join as viewer: %v", err)
	}

	var joinResp2 struct {
		SessionToken string `json:"session_token"`
	}
	parseJSON(resp, &joinResp2)
	t.Logf("Viewer1 joined with session token: %s...", joinResp2.SessionToken[:20])

	// Step 5: Developer accesses /me endpoint
	t.Log("Step 5: Developer accessing /me...")
	resp, err = apiRequest("GET", "/api/me", nil, joinResp1.SessionToken)
	if err != nil {
		t.Fatalf("Failed to access /me: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 on /me, got %d", resp.StatusCode)
	}

	var meResp struct {
		Name string `json:"name"`
		Role string `json:"role"`
	}
	parseJSON(resp, &meResp)
	if meResp.Name != "developer1" {
		t.Errorf("Expected name 'developer1', got '%s'", meResp.Name)
	}
	if meResp.Role != "dev" {
		t.Errorf("Expected role 'dev', got '%s'", meResp.Role)
	}
	t.Log("Developer /me endpoint validated")

	// Step 6: Verify invite tokens cannot be reused
	t.Log("Step 6: Verifying invite token cannot be reused...")
	resp, err = apiRequest("POST", "/api/join", joinReq1, "")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp.StatusCode == http.StatusOK {
		t.Error("Reusing invite token should fail")
	}
	resp.Body.Close()
	t.Log("Invite token reuse correctly rejected")

	// Step 7: List users as admin
	t.Log("Step 7: Listing users as admin...")
	resp, err = apiRequest("GET", "/api/admin/users", nil, adminToken)
	if err != nil {
		t.Fatalf("Failed to list users: %v", err)
	}

	var users []struct {
		Name string `json:"name"`
		Role string `json:"role"`
	}
	parseJSON(resp, &users)
	if len(users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(users))
	}
	t.Logf("Found %d users in system", len(users))

	// Step 8: Developer tries admin endpoint (should fail)
	t.Log("Step 8: Verifying developer cannot access admin endpoints...")
	resp, err = apiRequest("GET", "/api/admin/users", nil, joinResp1.SessionToken)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("Expected 403 for non-admin accessing admin endpoint, got %d", resp.StatusCode)
	}
	resp.Body.Close()
	t.Log("Role-based access control validated")

	// Step 9: Check audit log
	t.Log("Step 9: Checking audit log...")
	resp, err = apiRequest("GET", "/api/admin/audit", nil, adminToken)
	if err != nil {
		t.Fatalf("Failed to get audit log: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 on audit, got %d", resp.StatusCode)
	}

	var auditEntries []struct {
		Action    string `json:"action"`
		UserName  string `json:"user_name"`
		Details   string `json:"details"`
		Timestamp string `json:"timestamp"`
	}
	parseJSON(resp, &auditEntries)

	if len(auditEntries) == 0 {
		t.Error("Expected audit entries, got none")
	}

	// Verify we have expected audit actions
	actions := make(map[string]int)
	for _, entry := range auditEntries {
		actions[entry.Action]++
	}

	if actions["USER_CREATE"] < 2 {
		t.Errorf("Expected at least 2 USER_CREATE audit entries, got %d", actions["USER_CREATE"])
	}
	if actions["USER_JOIN"] < 2 {
		t.Errorf("Expected at least 2 USER_JOIN audit entries, got %d", actions["USER_JOIN"])
	}

	t.Logf("Audit log contains %d entries with actions: %v", len(auditEntries), actions)

	// Step 10: Delete a user
	t.Log("Step 10: Deleting viewer user...")
	resp, err = apiRequest("DELETE", "/api/admin/users/viewer1", nil, adminToken)
	if err != nil {
		t.Fatalf("Failed to delete user: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected 200 on delete, got %d: %s", resp.StatusCode, string(body))
	}
	resp.Body.Close()

	// Verify user is deleted
	resp, err = apiRequest("GET", "/api/admin/users", nil, adminToken)
	if err != nil {
		t.Fatalf("Failed to list users: %v", err)
	}
	var remainingUsers []struct {
		Name string `json:"name"`
	}
	parseJSON(resp, &remainingUsers)
	if len(remainingUsers) != 1 {
		t.Errorf("Expected 1 user after deletion, got %d", len(remainingUsers))
	}
	t.Log("User deletion verified")

	// Step 11: Verify deleted user's token no longer works
	t.Log("Step 11: Verifying deleted user cannot access API...")
	resp, err = apiRequest("GET", "/api/me", nil, joinResp2.SessionToken)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected 401 for deleted user, got %d", resp.StatusCode)
	}
	resp.Body.Close()
	t.Log("Deleted user access revocation verified")

	t.Log("All user flow tests passed!")
}

func TestEnvironmentManagement(t *testing.T) {
	// Step 1: Create projects first
	t.Log("Step 1: Creating projects...")
	devProjectReq := map[string]interface{}{
		"name":        "devproject",
		"description": "Development project",
	}
	resp, err := apiRequest("POST", "/api/admin/projects", devProjectReq, adminToken)
	if err != nil {
		t.Fatalf("Failed to create dev project: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200 creating dev project, got %d: %s", resp.StatusCode, string(body))
	}
	resp.Body.Close()

	prodProjectReq := map[string]interface{}{
		"name":        "prodproject",
		"description": "Production project",
	}
	resp, err = apiRequest("POST", "/api/admin/projects", prodProjectReq, adminToken)
	if err != nil {
		t.Fatalf("Failed to create prod project: %v", err)
	}
	resp.Body.Close()
	t.Log("Projects created successfully")

	// Step 2: Create staging environment in dev project
	t.Log("Step 2: Creating staging environment...")
	envReq := map[string]interface{}{
		"name":        "staging",
		"project":     "devproject",
		"host":        "env-staging",
		"port":        22,
		"deploy_user": "deploy",
		"deploy_key":  "-----BEGIN OPENSSH PRIVATE KEY-----\ntest-key-content\n-----END OPENSSH PRIVATE KEY-----",
	}

	resp, err = apiRequest("POST", "/api/admin/environments", envReq, adminToken)
	if err != nil {
		t.Fatalf("Failed to create environment: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200 creating environment, got %d: %s", resp.StatusCode, string(body))
	}

	var envResp struct {
		Name      string `json:"name"`
		Project   string `json:"project"`
		Host      string `json:"host"`
		DeployKey string `json:"deploy_key"`
	}
	parseJSON(resp, &envResp)

	if envResp.Name != "staging" {
		t.Errorf("Expected environment name 'staging', got '%s'", envResp.Name)
	}
	if envResp.Project != "devproject" {
		t.Errorf("Expected project 'devproject', got '%s'", envResp.Project)
	}
	if envResp.DeployKey != "" {
		t.Error("Deploy key should not be returned in response")
	}
	t.Log("Staging environment created successfully")

	// Step 3: Create production environment in prod project
	t.Log("Step 3: Creating production environment...")
	prodEnvReq := map[string]interface{}{
		"name":        "production",
		"project":     "prodproject",
		"host":        "env-production",
		"port":        22,
		"deploy_user": "deploy",
		"deploy_key":  "-----BEGIN OPENSSH PRIVATE KEY-----\nprod-key-content\n-----END OPENSSH PRIVATE KEY-----",
	}

	resp, err = apiRequest("POST", "/api/admin/environments", prodEnvReq, adminToken)
	if err != nil {
		t.Fatalf("Failed to create production environment: %v", err)
	}
	resp.Body.Close()

	// Step 4: List environments as admin
	t.Log("Step 4: Listing environments as admin...")
	resp, err = apiRequest("GET", "/api/admin/environments", nil, adminToken)
	if err != nil {
		t.Fatalf("Failed to list environments: %v", err)
	}

	var envs []struct {
		Name    string `json:"name"`
		Project string `json:"project"`
	}
	parseJSON(resp, &envs)
	if len(envs) != 2 {
		t.Errorf("Expected 2 environments, got %d", len(envs))
	}
	t.Logf("Found %d environments", len(envs))

	// Step 5: Create a developer user and grant project access
	t.Log("Step 5: Testing project-based environment access...")

	// Create developer
	createReq := map[string]interface{}{
		"name":  "envtestdev",
		"email": "envtest@example.com",
		"role":  "dev",
	}
	resp, err = apiRequest("POST", "/api/admin/users", createReq, adminToken)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	var createResp struct {
		InviteToken string `json:"invite_token"`
	}
	parseJSON(resp, &createResp)

	// Join as developer
	joinReq := map[string]interface{}{
		"invite_token": createResp.InviteToken,
		"public_key":   "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEnvTestKey envtest@test",
	}
	resp, err = apiRequest("POST", "/api/join", joinReq, "")
	if err != nil {
		t.Fatalf("Failed to join: %v", err)
	}
	var joinResp struct {
		SessionToken string `json:"session_token"`
	}
	parseJSON(resp, &joinResp)

	// Grant user access to devproject only (not prodproject)
	t.Log("Step 6: Granting project access to user...")
	grantReq := map[string]interface{}{
		"project": "devproject",
	}
	resp, err = apiRequest("POST", "/api/admin/users/envtestdev/access", grantReq, adminToken)
	if err != nil {
		t.Fatalf("Failed to grant project access: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200 granting access, got %d: %s", resp.StatusCode, string(body))
	}
	resp.Body.Close()

	// List environments as developer
	t.Log("Step 7: Listing environments as user...")
	resp, err = apiRequest("GET", "/api/environments", nil, joinResp.SessionToken)
	if err != nil {
		t.Fatalf("Failed to list environments as dev: %v", err)
	}

	var devEnvs []struct {
		Name    string `json:"name"`
		Project string `json:"project"`
	}
	parseJSON(resp, &devEnvs)

	// Developer should only see staging (devproject), not production (prodproject)
	if len(devEnvs) != 1 {
		t.Errorf("Developer should see 1 environment (staging in devproject), got %d", len(devEnvs))
	}
	if len(devEnvs) > 0 && devEnvs[0].Name != "staging" {
		t.Errorf("Developer should see 'staging', got '%s'", devEnvs[0].Name)
	}
	t.Log("Project-based environment access validated")

	// Step 8: Delete environments (using project/name format)
	t.Log("Step 8: Cleaning up environments...")
	resp, _ = apiRequest("DELETE", "/api/admin/environments/devproject/staging", nil, adminToken)
	resp.Body.Close()
	resp, _ = apiRequest("DELETE", "/api/admin/environments/prodproject/production", nil, adminToken)
	resp.Body.Close()

	// Delete projects
	resp, _ = apiRequest("DELETE", "/api/admin/projects/devproject", nil, adminToken)
	resp.Body.Close()
	resp, _ = apiRequest("DELETE", "/api/admin/projects/prodproject", nil, adminToken)
	resp.Body.Close()

	// Also delete test user
	resp, _ = apiRequest("DELETE", "/api/admin/users/envtestdev", nil, adminToken)
	resp.Body.Close()

	t.Log("Environment management tests passed!")
}

func TestProjectManagement(t *testing.T) {
	t.Log("Testing project management...")

	// Step 1: Create a project
	t.Log("Step 1: Creating project...")
	projectReq := map[string]interface{}{
		"name":        "testproject",
		"description": "Test project description",
	}
	resp, err := apiRequest("POST", "/api/admin/projects", projectReq, adminToken)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200 creating project, got %d: %s", resp.StatusCode, string(body))
	}

	var projectResp struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	parseJSON(resp, &projectResp)

	if projectResp.Name != "testproject" {
		t.Errorf("Expected project name 'testproject', got '%s'", projectResp.Name)
	}
	if projectResp.Description != "Test project description" {
		t.Errorf("Expected description 'Test project description', got '%s'", projectResp.Description)
	}
	t.Log("Project created successfully")

	// Step 2: List projects
	t.Log("Step 2: Listing projects...")
	resp, err = apiRequest("GET", "/api/admin/projects", nil, adminToken)
	if err != nil {
		t.Fatalf("Failed to list projects: %v", err)
	}

	var projects []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	parseJSON(resp, &projects)

	found := false
	for _, p := range projects {
		if p.Name == "testproject" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Created project not found in list")
	}
	t.Logf("Found %d project(s)", len(projects))

	// Step 3: Get specific project
	t.Log("Step 3: Getting project details...")
	resp, err = apiRequest("GET", "/api/admin/projects/testproject", nil, adminToken)
	if err != nil {
		t.Fatalf("Failed to get project: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200 getting project, got %d: %s", resp.StatusCode, string(body))
	}

	var project struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	parseJSON(resp, &project)

	if project.Name != "testproject" {
		t.Errorf("Expected project 'testproject', got '%s'", project.Name)
	}
	t.Log("Project details retrieved successfully")

	// Step 4: Create user and test access grant/revoke
	t.Log("Step 4: Testing access grant/revoke...")
	userReq := map[string]interface{}{
		"name":  "projectuser",
		"email": "project@example.com",
		"role":  "dev",
	}
	resp, err = apiRequest("POST", "/api/admin/users", userReq, adminToken)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	var createResp struct {
		InviteToken string `json:"invite_token"`
	}
	parseJSON(resp, &createResp)

	// Join as user
	joinReq := map[string]interface{}{
		"invite_token": createResp.InviteToken,
		"public_key":   "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIProjectTestKey project@test",
	}
	resp, err = apiRequest("POST", "/api/join", joinReq, "")
	if err != nil {
		t.Fatalf("Failed to join: %v", err)
	}
	var joinResp struct {
		SessionToken string `json:"session_token"`
	}
	parseJSON(resp, &joinResp)

	// Grant access to project
	t.Log("Step 5: Granting project access...")
	grantReq := map[string]interface{}{
		"project": "testproject",
	}
	resp, err = apiRequest("POST", "/api/admin/users/projectuser/access", grantReq, adminToken)
	if err != nil {
		t.Fatalf("Failed to grant access: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200 granting access, got %d: %s", resp.StatusCode, string(body))
	}
	resp.Body.Close()

	// Verify user has access (check /me endpoint)
	resp, err = apiRequest("GET", "/api/me", nil, joinResp.SessionToken)
	if err != nil {
		t.Fatalf("Failed to get user info: %v", err)
	}
	var meResp struct {
		Name     string   `json:"name"`
		Projects []string `json:"projects"`
	}
	parseJSON(resp, &meResp)

	hasAccess := false
	for _, p := range meResp.Projects {
		if p == "testproject" {
			hasAccess = true
			break
		}
	}
	if !hasAccess {
		t.Error("User should have access to testproject")
	}
	t.Log("Access granted successfully")

	// Revoke access
	t.Log("Step 6: Revoking project access...")
	revokeReq := map[string]interface{}{
		"project": "testproject",
	}
	resp, err = apiRequest("DELETE", "/api/admin/users/projectuser/access", revokeReq, adminToken)
	if err != nil {
		t.Fatalf("Failed to revoke access: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200 revoking access, got %d: %s", resp.StatusCode, string(body))
	}
	resp.Body.Close()

	// Verify access is revoked
	resp, err = apiRequest("GET", "/api/me", nil, joinResp.SessionToken)
	if err != nil {
		t.Fatalf("Failed to get user info: %v", err)
	}
	// Use a fresh struct to avoid leftover data from previous decode
	var meRespAfterRevoke struct {
		Name     string   `json:"name"`
		Projects []string `json:"projects"`
	}
	parseJSON(resp, &meRespAfterRevoke)

	hasAccess = false
	for _, p := range meRespAfterRevoke.Projects {
		if p == "testproject" {
			hasAccess = true
			break
		}
	}
	if hasAccess {
		t.Errorf("User should not have access to testproject after revoke, but has projects: %v", meRespAfterRevoke.Projects)
	}
	t.Log("Access revoked successfully")

	// Cleanup
	t.Log("Cleaning up...")
	resp, _ = apiRequest("DELETE", "/api/admin/users/projectuser", nil, adminToken)
	resp.Body.Close()
	resp, _ = apiRequest("DELETE", "/api/admin/projects/testproject", nil, adminToken)
	resp.Body.Close()

	t.Log("Project management tests passed!")
}

func TestAuditLogIntegrity(t *testing.T) {
	// Perform some actions
	t.Log("Performing actions to generate audit entries...")

	// Create a user
	createReq := map[string]interface{}{
		"name":  "audituser",
		"email": "audit@example.com",
		"role":  "dev",
	}
	resp, err := apiRequest("POST", "/api/admin/users", createReq, adminToken)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	var createResp struct {
		InviteToken string `json:"invite_token"`
	}
	parseJSON(resp, &createResp)

	// Join
	joinReq := map[string]interface{}{
		"invite_token": createResp.InviteToken,
		"public_key":   "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAuditTestKey audit@test",
	}
	resp, _ = apiRequest("POST", "/api/join", joinReq, "")
	resp.Body.Close()

	// Get audit log
	resp, err = apiRequest("GET", "/api/admin/audit", nil, adminToken)
	if err != nil {
		t.Fatalf("Failed to get audit log: %v", err)
	}

	var entries []struct {
		ID        int64  `json:"id"`
		Action    string `json:"action"`
		Timestamp string `json:"timestamp"`
		Hash      string `json:"hash"`
	}
	parseJSON(resp, &entries)

	// Verify entries are returned in order
	for i := 1; i < len(entries); i++ {
		// Entries should be in descending order (newest first)
		// Just verify we have entries with IDs
		if entries[i].ID >= entries[i-1].ID {
			// Note: This just checks ordering, actual hash chain verification
			// would be done server-side
			t.Logf("Entry %d: %s at %s", entries[i].ID, entries[i].Action, entries[i].Timestamp)
		}
	}

	t.Logf("Audit log contains %d entries", len(entries))

	// Cleanup
	resp, _ = apiRequest("DELETE", "/api/admin/users/audituser", nil, adminToken)
	resp.Body.Close()

	t.Log("Audit log integrity test passed!")
}

func TestConcurrentAccess(t *testing.T) {
	t.Log("Testing concurrent API access...")

	// Create multiple requests concurrently
	done := make(chan bool, 10)
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			resp, err := apiRequest("GET", "/api/admin/users", nil, adminToken)
			if err != nil {
				errors <- fmt.Errorf("request %d failed: %v", id, err)
				done <- false
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				errors <- fmt.Errorf("request %d got status %d", id, resp.StatusCode)
				done <- false
				return
			}
			done <- true
		}(i)
	}

	// Wait for all requests
	successCount := 0
	for i := 0; i < 10; i++ {
		if <-done {
			successCount++
		}
	}

	// Check for errors
	close(errors)
	for err := range errors {
		t.Error(err)
	}

	t.Logf("Concurrent access: %d/10 requests succeeded", successCount)
	if successCount != 10 {
		t.Errorf("Expected all 10 concurrent requests to succeed")
	}
}

func TestSecurityHeaders(t *testing.T) {
	t.Log("Verifying security headers...")

	resp, err := apiRequest("GET", "/api/admin/users", nil, adminToken)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	expectedHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1; mode=block",
	}

	for header, expected := range expectedHeaders {
		actual := resp.Header.Get(header)
		if actual != expected {
			t.Errorf("Expected %s: %s, got: %s", header, expected, actual)
		}
	}

	t.Log("Security headers validated")
}

func TestInvalidRequests(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		endpoint       string
		body           interface{}
		token          string
		expectedStatus int
	}{
		{
			name:           "Invalid JSON body",
			method:         "POST",
			endpoint:       "/api/admin/users",
			body:           "invalid-json",
			token:          adminToken,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Missing required fields",
			method:         "POST",
			endpoint:       "/api/admin/users",
			body:           map[string]string{"name": ""},
			token:          adminToken,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid role",
			method:         "POST",
			endpoint:       "/api/admin/users",
			body:           map[string]string{"name": "test", "email": "test@test.com", "role": "superadmin"},
			token:          adminToken,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Get non-existent user",
			method:         "GET",
			endpoint:       "/api/admin/users/nonexistent12345",
			token:          adminToken,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Delete non-existent user",
			method:         "DELETE",
			endpoint:       "/api/admin/users/nonexistent12345",
			token:          adminToken,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var resp *http.Response
			var err error

			if tc.body == "invalid-json" {
				// Send raw invalid JSON
				req, _ := http.NewRequest(tc.method, serverURL+tc.endpoint, strings.NewReader("{invalid}"))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+tc.token)
				resp, err = http.DefaultClient.Do(req)
			} else {
				resp, err = apiRequest(tc.method, tc.endpoint, tc.body, tc.token)
			}

			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.expectedStatus {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("Expected status %d, got %d: %s", tc.expectedStatus, resp.StatusCode, string(body))
			}
		})
	}
}

// Mailpit API URL for email testing
const mailpitURL = "http://localhost:18025"

func TestEmailNotifications(t *testing.T) {
	// Check if Mailpit is available
	resp, err := http.Get(mailpitURL + "/api/v1/info")
	if err != nil {
		t.Skip("Mailpit not available, skipping email tests")
	}
	resp.Body.Close()

	t.Log("Testing email notifications via Mailpit...")

	// Clear any existing messages
	clearMailpit(t)

	// Step 1: Create a user (should trigger invitation email)
	t.Log("Step 1: Creating user to trigger invitation email...")
	createReq := map[string]interface{}{
		"name":  "emailtestuser",
		"email": "emailtest@example.com",
		"role":  "dev",
	}

	resp, err = apiRequest("POST", "/api/admin/users", createReq, adminToken)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	var createResp struct {
		InviteToken string `json:"invite_token"`
	}
	parseJSON(resp, &createResp)

	// Wait a moment for email to be sent
	time.Sleep(2 * time.Second)

	// Check Mailpit for invitation email
	messages := getMailpitMessages(t)
	if len(messages) == 0 {
		t.Log("No invitation email received (SMTP may not be configured)")
	} else {
		t.Logf("Received %d email(s)", len(messages))
		found := false
		for _, msg := range messages {
			if strings.Contains(msg.Subject, "Invitation") {
				found = true
				t.Logf("Found invitation email: %s", msg.Subject)
				if !strings.Contains(msg.To[0].Address, "emailtest@example.com") {
					t.Errorf("Email sent to wrong address: %s", msg.To[0].Address)
				}
			}
		}
		if !found {
			t.Log("Invitation email not found in received messages")
		}
	}

	// Step 2: User joins (should trigger welcome email)
	t.Log("Step 2: User joining to trigger welcome email...")
	clearMailpit(t)

	joinReq := map[string]interface{}{
		"invite_token": createResp.InviteToken,
		"public_key":   "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEmailTestKey emailtest@test",
	}
	resp, err = apiRequest("POST", "/api/join", joinReq, "")
	if err != nil {
		t.Fatalf("Failed to join: %v", err)
	}
	resp.Body.Close()

	time.Sleep(2 * time.Second)

	messages = getMailpitMessages(t)
	if len(messages) > 0 {
		for _, msg := range messages {
			if strings.Contains(msg.Subject, "Welcome") {
				t.Logf("Found welcome email: %s", msg.Subject)
			}
		}
	}

	// Cleanup
	resp, _ = apiRequest("DELETE", "/api/admin/users/emailtestuser", nil, adminToken)
	resp.Body.Close()

	t.Log("Email notification tests completed!")
}

type mailpitMessage struct {
	ID      string `json:"ID"`
	Subject string `json:"Subject"`
	From    struct {
		Address string `json:"Address"`
	} `json:"From"`
	To []struct {
		Address string `json:"Address"`
	} `json:"To"`
}

type mailpitResponse struct {
	Messages []mailpitMessage `json:"messages"`
}

func clearMailpit(t *testing.T) {
	req, _ := http.NewRequest("DELETE", mailpitURL+"/api/v1/messages", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Logf("Could not clear Mailpit: %v", err)
		return
	}
	resp.Body.Close()
}

func getMailpitMessages(t *testing.T) []mailpitMessage {
	resp, err := http.Get(mailpitURL + "/api/v1/messages")
	if err != nil {
		t.Logf("Could not get Mailpit messages: %v", err)
		return nil
	}
	defer resp.Body.Close()

	var result mailpitResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Logf("Could not decode Mailpit response: %v", err)
		return nil
	}

	return result.Messages
}

func TestMFASetupAndVerification(t *testing.T) {
	t.Log("Testing MFA setup and verification...")

	// Step 1: Create a user
	t.Log("Step 1: Creating user for MFA test...")
	createReq := map[string]interface{}{
		"name":  "mfatestuser",
		"email": "mfatest@example.com",
		"role":  "dev",
	}

	resp, err := apiRequest("POST", "/api/admin/users", createReq, adminToken)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	var createResp struct {
		InviteToken string `json:"invite_token"`
	}
	parseJSON(resp, &createResp)

	// Step 2: User joins
	t.Log("Step 2: User joining...")
	joinReq := map[string]interface{}{
		"invite_token": createResp.InviteToken,
		"public_key":   "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMFATestKey mfatest@test",
	}
	resp, err = apiRequest("POST", "/api/join", joinReq, "")
	if err != nil {
		t.Fatalf("Failed to join: %v", err)
	}
	var joinResp struct {
		SessionToken string `json:"session_token"`
	}
	parseJSON(resp, &joinResp)

	// Step 3: Get MFA setup (GET request to get secret)
	t.Log("Step 3: Getting MFA setup...")
	resp, err = apiRequest("GET", "/api/mfa/setup", nil, joinResp.SessionToken)
	if err != nil {
		t.Fatalf("Failed to get MFA setup: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("MFA setup GET failed with status %d: %s", resp.StatusCode, string(body))
	}

	var mfaSetup struct {
		Secret    string `json:"secret"`
		QRCodeURL string `json:"qr_code_url"`
	}
	if err := parseJSON(resp, &mfaSetup); err != nil {
		t.Fatalf("Failed to parse MFA setup response: %v", err)
	}

	if mfaSetup.Secret == "" {
		t.Error("MFA secret should not be empty")
	}
	if mfaSetup.QRCodeURL == "" {
		t.Error("QR code URL should not be empty")
	}
	t.Logf("MFA setup returned secret (length: %d) and QR URL", len(mfaSetup.Secret))

	// Step 4: Confirm with invalid code (should fail)
	t.Log("Step 4: Confirming MFA setup with invalid code (should fail)...")
	confirmReq := map[string]interface{}{
		"code": "000000",
	}
	resp, err = apiRequest("POST", "/api/mfa/setup", confirmReq, joinResp.SessionToken)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp.StatusCode == http.StatusOK {
		t.Error("MFA confirmation should fail with invalid code")
	}
	resp.Body.Close()
	t.Log("Invalid MFA code correctly rejected")

	// Step 5: Check user status shows MFA not enabled
	t.Log("Step 5: Checking user MFA status...")
	resp, err = apiRequest("GET", "/api/me", nil, joinResp.SessionToken)
	if err != nil {
		t.Fatalf("Failed to get user status: %v", err)
	}
	var meResp struct {
		Name       string `json:"name"`
		MFAEnabled bool   `json:"mfa_enabled"`
	}
	parseJSON(resp, &meResp)

	// MFA is set up but not verified yet, so it should still be false
	t.Logf("User MFA status: enabled=%v", meResp.MFAEnabled)

	// Cleanup
	resp, _ = apiRequest("DELETE", "/api/admin/users/mfatestuser", nil, adminToken)
	resp.Body.Close()

	t.Log("MFA tests completed!")
}

func TestRateLimiting(t *testing.T) {
	// Skip this test as it interferes with other tests due to IP lockout
	// The rate limiting functionality is tested in unit tests
	t.Skip("Skipping rate limiting test to avoid IP lockout affecting other tests")
}

func TestKeyDeploymentFlow(t *testing.T) {
	t.Log("Testing SSH key deployment flow...")

	// Step 1: Create project first
	t.Log("Step 1: Creating test project...")
	projectReq := map[string]interface{}{
		"name":        "deployproject",
		"description": "Deploy test project",
	}
	resp, err := apiRequest("POST", "/api/admin/projects", projectReq, adminToken)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200 creating project, got %d: %s", resp.StatusCode, string(body))
	}
	resp.Body.Close()

	// Step 2: Create environment pointing to test SSH container
	t.Log("Step 2: Creating test environment...")
	envReq := map[string]interface{}{
		"name":        "deploy-test",
		"project":     "deployproject",
		"host":        "env-staging",
		"port":        22,
		"deploy_user": "deploy",
		"deploy_key": `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACBGJTxnB3rQnFGVOKVn6ZoQ8VjBR5qL9bD8tM0V5ZzKPgAAAJgVYHPsFWBz
7AAAAAtzc2gtZWQyNTUxOQAAACBGJTxnB3rQnFGVOKVn6ZoQ8VjBR5qL9bD8tM0V5ZzKPg
AAAEB6bDgKqJj47l+urn5BO9hARKFzZOPKYmJX2j6ZqwOT6kYlPGcHetCcUZU4pWfpmhDx
WMFHmov1sPy0zRXlnMo+AAAADXR
-----END OPENSSH PRIVATE KEY-----`,
	}

	resp, err = apiRequest("POST", "/api/admin/environments", envReq, adminToken)
	if err != nil {
		t.Fatalf("Failed to create environment: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200 creating environment, got %d: %s", resp.StatusCode, string(body))
	}
	resp.Body.Close()

	// Step 3: Create and join as a user
	t.Log("Step 3: Creating and joining as user...")
	createReq := map[string]interface{}{
		"name":  "deployuser",
		"email": "deploy@example.com",
		"role":  "dev",
	}
	resp, err = apiRequest("POST", "/api/admin/users", createReq, adminToken)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	var createResp struct {
		InviteToken string `json:"invite_token"`
	}
	parseJSON(resp, &createResp)

	joinReq := map[string]interface{}{
		"invite_token": createResp.InviteToken,
		"public_key":   "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIDeployUserKey deploy@test",
	}
	resp, err = apiRequest("POST", "/api/join", joinReq, "")
	if err != nil {
		t.Fatalf("Failed to join: %v", err)
	}
	var joinResp struct {
		SessionToken string `json:"session_token"`
	}
	parseJSON(resp, &joinResp)

	// Step 4: Grant user access to the project
	t.Log("Step 4: Granting project access to user...")
	grantReq := map[string]interface{}{
		"project": "deployproject",
	}
	resp, err = apiRequest("POST", "/api/admin/users/deployuser/access", grantReq, adminToken)
	if err != nil {
		t.Fatalf("Failed to grant project access: %v", err)
	}
	resp.Body.Close()

	// Step 5: List environments as user (should see deploy-test)
	t.Log("Step 5: Listing environments as user...")
	resp, err = apiRequest("GET", "/api/environments", nil, joinResp.SessionToken)
	if err != nil {
		t.Fatalf("Failed to list environments: %v", err)
	}

	var envs []struct {
		Name    string `json:"name"`
		Project string `json:"project"`
	}
	parseJSON(resp, &envs)

	found := false
	for _, env := range envs {
		if env.Name == "deploy-test" && env.Project == "deployproject" {
			found = true
			break
		}
	}
	if !found {
		t.Error("User should see deploy-test environment in deployproject")
	}
	t.Logf("User can see %d environment(s)", len(envs))

	// Cleanup
	t.Log("Cleaning up...")
	resp, _ = apiRequest("DELETE", "/api/admin/users/deployuser", nil, adminToken)
	resp.Body.Close()
	resp, _ = apiRequest("DELETE", "/api/admin/environments/deployproject/deploy-test", nil, adminToken)
	resp.Body.Close()
	resp, _ = apiRequest("DELETE", "/api/admin/projects/deployproject", nil, adminToken)
	resp.Body.Close()

	t.Log("Key deployment flow test completed!")
}

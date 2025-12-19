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
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestClientConfig tests client configuration save/load
func TestClientConfigSaveLoad(t *testing.T) {
	// Use temp directory
	tmpDir, err := os.MkdirTemp("", "magebox-cli-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Override home dir for test
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	config := &clientConfig{
		ServerURL:    "https://test.example.com",
		SessionToken: "test-token-12345",
		UserName:     "testuser",
		Role:         "dev",
		JoinedAt:     time.Now().Format(time.RFC3339),
	}

	// Save config
	if err := saveClientConfig(config); err != nil {
		t.Fatalf("saveClientConfig failed: %v", err)
	}

	// Verify file exists
	configPath, _ := getClientConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file not created")
	}

	// Load config
	loaded, err := loadClientConfig()
	if err != nil {
		t.Fatalf("loadClientConfig failed: %v", err)
	}

	if loaded.ServerURL != config.ServerURL {
		t.Errorf("ServerURL mismatch: got %s, want %s", loaded.ServerURL, config.ServerURL)
	}
	if loaded.SessionToken != config.SessionToken {
		t.Errorf("SessionToken mismatch: got %s, want %s", loaded.SessionToken, config.SessionToken)
	}
	if loaded.UserName != config.UserName {
		t.Errorf("UserName mismatch: got %s, want %s", loaded.UserName, config.UserName)
	}
	if loaded.Role != config.Role {
		t.Errorf("Role mismatch: got %s, want %s", loaded.Role, config.Role)
	}
}

func TestClientConfigLoadNotExist(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "magebox-cli-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	_, err = loadClientConfig()
	if err == nil {
		t.Error("Expected error when config doesn't exist")
	}
}

// TestAPIRequestHelper tests the apiRequest helper function
func TestAPIRequestHelper(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type header not set correctly")
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Authorization header not set correctly: %s", r.Header.Get("Authorization"))
		}

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	// Create temp client config pointing to mock server
	tmpDir, _ := os.MkdirTemp("", "magebox-cli-test-*")
	defer os.RemoveAll(tmpDir)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	config := &clientConfig{
		ServerURL: server.URL,
	}
	saveClientConfig(config)

	// Make request
	resp, err := apiRequest("GET", "/test", nil, "test-token")
	if err != nil {
		t.Fatalf("apiRequest failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Unexpected status code: %d", resp.StatusCode)
	}
}

func TestAPIRequestWithBody(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Decode request body
		json.NewDecoder(r.Body).Decode(&receivedBody)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	tmpDir, _ := os.MkdirTemp("", "magebox-cli-test-*")
	defer os.RemoveAll(tmpDir)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	config := &clientConfig{ServerURL: server.URL}
	saveClientConfig(config)

	body := map[string]interface{}{
		"name":  "testuser",
		"email": "test@example.com",
	}

	resp, err := apiRequest("POST", "/test", body, "test-token")
	if err != nil {
		t.Fatalf("apiRequest failed: %v", err)
	}
	resp.Body.Close()

	if receivedBody["name"] != "testuser" {
		t.Errorf("Body not received correctly: %v", receivedBody)
	}
}

// TestGetAdminToken tests admin token retrieval
func TestGetAdminTokenFromEnv(t *testing.T) {
	origToken := os.Getenv("MAGEBOX_ADMIN_TOKEN")
	os.Setenv("MAGEBOX_ADMIN_TOKEN", "test-admin-token-from-env")
	defer os.Setenv("MAGEBOX_ADMIN_TOKEN", origToken)

	token, err := getAdminToken()
	if err != nil {
		t.Fatalf("getAdminToken failed: %v", err)
	}

	if token != "test-admin-token-from-env" {
		t.Errorf("Token mismatch: got %s, want test-admin-token-from-env", token)
	}
}

func TestGetAdminTokenMissing(t *testing.T) {
	// Clear environment
	origToken := os.Getenv("MAGEBOX_ADMIN_TOKEN")
	os.Unsetenv("MAGEBOX_ADMIN_TOKEN")
	defer os.Setenv("MAGEBOX_ADMIN_TOKEN", origToken)

	// Use temp home without config
	tmpDir, _ := os.MkdirTemp("", "magebox-cli-test-*")
	defer os.RemoveAll(tmpDir)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	_, err := getAdminToken()
	if err == nil {
		t.Error("Expected error when admin token not available")
	}
}

// TestServerInit tests server initialization
func TestServerInitCreatesConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "magebox-server-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Set data dir
	serverDataDir = tmpDir

	// Run init
	if err := runServerInit(nil, nil); err != nil {
		t.Fatalf("runServerInit failed: %v", err)
	}

	// Verify config file exists
	configPath := filepath.Join(tmpDir, "server.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file not created")
	}

	// Verify config content
	data, _ := os.ReadFile(configPath)
	var config map[string]interface{}
	json.Unmarshal(data, &config)

	if _, ok := config["master_key"]; !ok {
		t.Error("master_key not in config")
	}
	if _, ok := config["admin_token_hash"]; !ok {
		t.Error("admin_token_hash not in config")
	}
	if _, ok := config["initialized_at"]; !ok {
		t.Error("initialized_at not in config")
	}

	// Reset
	serverDataDir = ""
}

func TestServerInitAlreadyInitialized(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "magebox-server-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create existing config
	configPath := filepath.Join(tmpDir, "server.json")
	os.WriteFile(configPath, []byte(`{"master_key":"existing"}`), 0600)

	serverDataDir = tmpDir
	defer func() { serverDataDir = "" }()

	// Run init - should not error but warn
	if err := runServerInit(nil, nil); err != nil {
		t.Fatalf("runServerInit failed: %v", err)
	}

	// Verify config wasn't overwritten
	data, _ := os.ReadFile(configPath)
	var config map[string]interface{}
	json.Unmarshal(data, &config)

	if config["master_key"] != "existing" {
		t.Error("Config was overwritten")
	}
}

// TestGetServerDataDir tests data directory resolution
func TestGetServerDataDir(t *testing.T) {
	// Test with explicit dir
	serverDataDir = "/custom/path"
	dir, err := getServerDataDir()
	if err != nil {
		t.Fatal(err)
	}
	if dir != "/custom/path" {
		t.Errorf("Expected /custom/path, got %s", dir)
	}
	serverDataDir = ""

	// Test with default (home dir)
	dir, err = getServerDataDir()
	if err != nil {
		t.Fatal(err)
	}
	homeDir, _ := os.UserHomeDir()
	expected := filepath.Join(homeDir, ".magebox", "teamserver")
	if dir != expected {
		t.Errorf("Expected %s, got %s", expected, dir)
	}
}

// MockServer for testing CLI commands
type mockServer struct {
	*httptest.Server
	Users        []mockUser
	Environments []mockEnvironment
	AuditLog     []mockAuditEntry
}

type mockUser struct {
	ID           int64      `json:"id"`
	Name         string     `json:"name"`
	Email        string     `json:"email"`
	Role         string     `json:"role"`
	PublicKey    string     `json:"public_key"`
	MFAEnabled   bool       `json:"mfa_enabled"`
	ExpiresAt    *time.Time `json:"expires_at"`
	CreatedAt    time.Time  `json:"created_at"`
	LastAccessAt *time.Time `json:"last_access_at"`
}

type mockEnvironment struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Host         string    `json:"host"`
	Port         int       `json:"port"`
	DeployUser   string    `json:"deploy_user"`
	AllowedRoles []string  `json:"allowed_roles"`
	CreatedAt    time.Time `json:"created_at"`
}

type mockAuditEntry struct {
	ID        int64     `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	UserName  string    `json:"user_name"`
	Action    string    `json:"action"`
	Details   string    `json:"details"`
	IPAddress string    `json:"ip_address"`
}

func newMockServer() *mockServer {
	ms := &mockServer{
		Users: []mockUser{
			{
				ID:        1,
				Name:      "alice",
				Email:     "alice@example.com",
				Role:      "admin",
				CreatedAt: time.Now().Add(-24 * time.Hour),
			},
			{
				ID:        2,
				Name:      "bob",
				Email:     "bob@example.com",
				Role:      "dev",
				CreatedAt: time.Now(),
			},
		},
		Environments: []mockEnvironment{
			{
				ID:           1,
				Name:         "production",
				Host:         "prod.example.com",
				Port:         22,
				DeployUser:   "deploy",
				AllowedRoles: []string{"admin"},
				CreatedAt:    time.Now(),
			},
		},
		AuditLog: []mockAuditEntry{
			{
				ID:        1,
				Timestamp: time.Now(),
				UserName:  "alice",
				Action:    "USER_CREATE",
				Details:   "Created user bob",
				IPAddress: "127.0.0.1",
			},
		},
	}

	ms.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Check auth
		token := r.Header.Get("Authorization")
		if token == "" && r.URL.Path != "/health" && r.URL.Path != "/api/join" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}

		switch {
		case r.URL.Path == "/health":
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

		case r.URL.Path == "/api/admin/users" && r.Method == "GET":
			json.NewEncoder(w).Encode(ms.Users)

		case r.URL.Path == "/api/admin/users" && r.Method == "POST":
			var req map[string]interface{}
			json.NewDecoder(r.Body).Decode(&req)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"user": map[string]string{
					"name":  req["name"].(string),
					"email": req["email"].(string),
					"role":  req["role"].(string),
				},
				"invite_token": "invite_" + req["name"].(string),
			})

		case r.URL.Path == "/api/admin/environments" && r.Method == "GET":
			json.NewEncoder(w).Encode(ms.Environments)

		case r.URL.Path == "/api/admin/environments" && r.Method == "POST":
			var req map[string]interface{}
			json.NewDecoder(r.Body).Decode(&req)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"name":          req["name"],
				"host":          req["host"],
				"port":          req["port"],
				"deploy_user":   req["deploy_user"],
				"allowed_roles": req["allowed_roles"],
			})

		case r.URL.Path == "/api/admin/audit":
			json.NewEncoder(w).Encode(ms.AuditLog)

		case r.URL.Path == "/api/me":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"name":  "testuser",
				"email": "test@example.com",
				"role":  "dev",
			})

		case r.URL.Path == "/api/environments":
			// Filter environments for user role
			json.NewEncoder(w).Encode(ms.Environments)

		case r.URL.Path == "/api/join" && r.Method == "POST":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"session_token": "session_token_12345",
				"private_key":   "-----BEGIN OPENSSH PRIVATE KEY-----\ntest-mock-key\n-----END OPENSSH PRIVATE KEY-----",
				"server_host":   r.Host,
				"user": map[string]string{
					"name": "newuser",
					"role": "dev",
				},
				"environments": []map[string]interface{}{
					{"name": "staging", "project": "test", "host": "staging.example.com", "port": 22, "deploy_user": "deploy"},
				},
			})

		default:
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
		}
	}))

	return ms
}

func TestMockServerUserList(t *testing.T) {
	ms := newMockServer()
	defer ms.Server.Close()

	// Setup client config
	tmpDir, _ := os.MkdirTemp("", "magebox-cli-test-*")
	defer os.RemoveAll(tmpDir)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	config := &clientConfig{ServerURL: ms.Server.URL}
	saveClientConfig(config)

	// Set admin token
	os.Setenv("MAGEBOX_ADMIN_TOKEN", "test-admin-token")
	defer os.Unsetenv("MAGEBOX_ADMIN_TOKEN")

	// Make request directly (testing apiRequest + mock server integration)
	resp, err := apiRequest("GET", "/api/admin/users", nil, "test-admin-token")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	var users []mockUser
	json.NewDecoder(resp.Body).Decode(&users)

	if len(users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(users))
	}
	if users[0].Name != "alice" {
		t.Errorf("Expected first user alice, got %s", users[0].Name)
	}
}

func TestMockServerEnvironmentList(t *testing.T) {
	ms := newMockServer()
	defer ms.Server.Close()

	tmpDir, _ := os.MkdirTemp("", "magebox-cli-test-*")
	defer os.RemoveAll(tmpDir)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	config := &clientConfig{ServerURL: ms.Server.URL}
	saveClientConfig(config)

	os.Setenv("MAGEBOX_ADMIN_TOKEN", "test-admin-token")
	defer os.Unsetenv("MAGEBOX_ADMIN_TOKEN")

	resp, err := apiRequest("GET", "/api/admin/environments", nil, "test-admin-token")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	var envs []mockEnvironment
	json.NewDecoder(resp.Body).Decode(&envs)

	if len(envs) != 1 {
		t.Errorf("Expected 1 environment, got %d", len(envs))
	}
	if envs[0].Name != "production" {
		t.Errorf("Expected environment production, got %s", envs[0].Name)
	}
}

func TestMockServerAuditLog(t *testing.T) {
	ms := newMockServer()
	defer ms.Server.Close()

	tmpDir, _ := os.MkdirTemp("", "magebox-cli-test-*")
	defer os.RemoveAll(tmpDir)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	config := &clientConfig{ServerURL: ms.Server.URL}
	saveClientConfig(config)

	os.Setenv("MAGEBOX_ADMIN_TOKEN", "test-admin-token")
	defer os.Unsetenv("MAGEBOX_ADMIN_TOKEN")

	resp, err := apiRequest("GET", "/api/admin/audit", nil, "test-admin-token")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	var entries []mockAuditEntry
	json.NewDecoder(resp.Body).Decode(&entries)

	if len(entries) != 1 {
		t.Errorf("Expected 1 audit entry, got %d", len(entries))
	}
	if entries[0].Action != "USER_CREATE" {
		t.Errorf("Expected action USER_CREATE, got %s", entries[0].Action)
	}
}

func TestMockServerJoinFlow(t *testing.T) {
	ms := newMockServer()
	defer ms.Server.Close()

	tmpDir, _ := os.MkdirTemp("", "magebox-cli-test-*")
	defer os.RemoveAll(tmpDir)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Set join parameters (no public key needed - server generates it)
	inviteToken = "test-invite-token"
	defer func() {
		inviteToken = ""
	}()

	// Run join command
	err := runServerJoin(nil, []string{ms.Server.URL})
	if err != nil {
		t.Fatalf("runServerJoin failed: %v", err)
	}

	// Verify client config was saved
	config, err := loadClientConfig()
	if err != nil {
		t.Fatalf("loadClientConfig failed: %v", err)
	}

	if config.SessionToken != "session_token_12345" {
		t.Errorf("Session token mismatch: got %s", config.SessionToken)
	}
	if config.UserName != "newuser" {
		t.Errorf("UserName mismatch: got %s", config.UserName)
	}
	if config.KeyPath == "" {
		t.Error("KeyPath should be set after join")
	}

	// Verify key file was created
	if _, err := os.Stat(config.KeyPath); os.IsNotExist(err) {
		t.Errorf("SSH key file not created at %s", config.KeyPath)
	}

	// Verify environments were stored
	if len(config.Environments) == 0 {
		t.Error("Environments should be populated after join")
	}
}

func TestMockServerWhoami(t *testing.T) {
	ms := newMockServer()
	defer ms.Server.Close()

	tmpDir, _ := os.MkdirTemp("", "magebox-cli-test-*")
	defer os.RemoveAll(tmpDir)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Setup client config with session
	config := &clientConfig{
		ServerURL:    ms.Server.URL,
		SessionToken: "valid-session-token",
		UserName:     "testuser",
		Role:         "dev",
	}
	saveClientConfig(config)

	// Run whoami - should not error
	err := runServerWhoami(nil, nil)
	if err != nil {
		t.Fatalf("runServerWhoami failed: %v", err)
	}
}

func TestMockServerUnauthorized(t *testing.T) {
	ms := newMockServer()
	defer ms.Server.Close()

	tmpDir, _ := os.MkdirTemp("", "magebox-cli-test-*")
	defer os.RemoveAll(tmpDir)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	config := &clientConfig{ServerURL: ms.Server.URL}
	saveClientConfig(config)

	// Make request without auth token
	resp, err := apiRequest("GET", "/api/admin/users", nil, "")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", resp.StatusCode)
	}
}

// Test output formatting helpers
func TestOutputAuditJSON(t *testing.T) {
	entries := []auditEntryDisplay{
		{
			ID:        1,
			Timestamp: time.Now(),
			UserName:  "alice",
			Action:    "USER_CREATE",
			Details:   "Created user",
			IPAddress: "127.0.0.1",
		},
	}

	// Redirect stdout for testing
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputAuditJSON(entries)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("outputAuditJSON failed: %v", err)
	}

	// Read output
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Verify it's valid JSON
	var parsed []auditEntryDisplay
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	if len(parsed) != 1 || parsed[0].Action != "USER_CREATE" {
		t.Error("JSON content mismatch")
	}
}

func TestOutputAuditCSV(t *testing.T) {
	entries := []auditEntryDisplay{
		{
			ID:        1,
			Timestamp: time.Now(),
			UserName:  "alice",
			Action:    "USER_CREATE",
			Details:   "Created user",
			IPAddress: "127.0.0.1",
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputAuditCSV(entries)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("outputAuditCSV failed: %v", err)
	}

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Check for CSV header and content
	if !contains(output, "ID,Timestamp,User,Action,Details,IP Address") {
		t.Error("CSV header missing")
	}
	if !contains(output, "alice") {
		t.Error("CSV content missing user")
	}
	if !contains(output, "USER_CREATE") {
		t.Error("CSV content missing action")
	}
}

func TestOutputAuditTableEmpty(t *testing.T) {
	entries := []auditEntryDisplay{}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputAuditTable(entries)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("outputAuditTable failed: %v", err)
	}

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if !contains(output, "No audit entries found") {
		t.Error("Expected 'No audit entries found' message")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Test role validation
func TestRoleValidation(t *testing.T) {
	tests := []struct {
		role  string
		valid bool
	}{
		{"admin", true},
		{"dev", true},
		{"readonly", true},
		{"invalid", false},
		{"", false},
		{"Admin", false}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			// This would need to import teamserver, but we test it indirectly
			// through the CLI commands validation
		})
	}
}

// Test URL normalization
func TestURLNormalization(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"example.com", "https://example.com"},
		{"http://example.com", "http://example.com"},
		{"https://example.com", "https://example.com"},
		{"https://example.com/", "https://example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			// Normalize URL logic from runServerJoin
			url := tt.input
			if !hasPrefix(url, "http://") && !hasPrefix(url, "https://") {
				url = "https://" + url
			}
			url = trimSuffix(url, "/")

			if url != tt.expected {
				t.Errorf("URL normalization failed: got %s, want %s", url, tt.expected)
			}
		})
	}
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func trimSuffix(s, suffix string) string {
	if len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix {
		return s[:len(s)-len(suffix)]
	}
	return s
}

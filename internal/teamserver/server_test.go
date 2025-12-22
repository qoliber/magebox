/**
 * Created by Qoliber
 *
 * @category    Qoliber
 * @package     MageBox
 * @author      Jakub Winkler <jwinkler@qoliber.com>
 */

package teamserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func setupTestServer(t *testing.T) (*Server, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "magebox-server-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	masterKey, err := GenerateMasterKey()
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to generate master key: %v", err)
	}

	config := &ServerConfig{
		Port:    7443,
		Host:    "127.0.0.1",
		DataDir: tmpDir,
		Security: SecurityConfig{
			RateLimitEnabled:  false,
			InviteExpiry:      "48h",
			SessionExpiry:     "720h",
			DefaultAccessDays: 90,
		},
	}

	server, err := NewServer(config, masterKey)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create server: %v", err)
	}

	cleanup := func() {
		server.storage.Close()
		os.RemoveAll(tmpDir)
	}

	return server, cleanup
}

func setupTestServerWithAdmin(t *testing.T) (*Server, string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "magebox-server-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	masterKey, err := GenerateMasterKey()
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to generate master key: %v", err)
	}

	// Generate admin token
	adminToken, err := GenerateToken(32)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to generate admin token: %v", err)
	}

	adminTokenHash, err := HashToken(adminToken)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to hash admin token: %v", err)
	}

	config := &ServerConfig{
		Port:           7443,
		Host:           "127.0.0.1",
		DataDir:        tmpDir,
		AdminTokenHash: adminTokenHash,
		Security: SecurityConfig{
			RateLimitEnabled:  false,
			InviteExpiry:      "48h",
			SessionExpiry:     "720h",
			DefaultAccessDays: 90,
		},
	}

	server, err := NewServer(config, masterKey)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create server: %v", err)
	}

	cleanup := func() {
		server.storage.Close()
		os.RemoveAll(tmpDir)
	}

	return server, adminToken, cleanup
}

func TestNewServer(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	if server == nil {
		t.Error("Server should not be nil")
	}
}

func TestHealthEndpoint(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	server.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Error("Expected status to be healthy")
	}
}

func TestJoinEndpointWithoutToken(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	body := `{"invite_token": "invalid"}`
	req := httptest.NewRequest(http.MethodPost, "/api/join", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	server.mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestJoinEndpointMethodNotAllowed(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/join", nil)
	w := httptest.NewRecorder()

	server.mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestMeEndpointUnauthorized(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	w := httptest.NewRecorder()

	server.mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestAdminUsersEndpointUnauthorized(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
	w := httptest.NewRecorder()

	server.mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestAdminUsersWithValidToken(t *testing.T) {
	server, adminToken, cleanup := setupTestServerWithAdmin(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()

	server.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var users []User
	if err := json.NewDecoder(w.Body).Decode(&users); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should return empty list initially
	if len(users) != 0 {
		t.Errorf("Expected 0 users, got %d", len(users))
	}
}

func TestCreateUserInvite(t *testing.T) {
	server, adminToken, cleanup := setupTestServerWithAdmin(t)
	defer cleanup()

	body := `{"name": "testuser", "email": "test@example.com", "role": "dev"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/users", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var response CreateUserResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.InviteToken == "" {
		t.Error("Invite token should not be empty")
	}
	if response.User.Name != "testuser" {
		t.Errorf("Expected user name testuser, got %s", response.User.Name)
	}
}

func TestFullJoinFlow(t *testing.T) {
	server, adminToken, cleanup := setupTestServerWithAdmin(t)
	defer cleanup()

	// Step 1: Create invite
	createBody := `{"name": "newdev", "email": "newdev@example.com", "role": "dev"}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/admin/users", bytes.NewBufferString(createBody))
	createReq.Header.Set("Authorization", "Bearer "+adminToken)
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()

	server.mux.ServeHTTP(createW, createReq)

	if createW.Code != http.StatusOK {
		t.Fatalf("Failed to create invite: %s", createW.Body.String())
	}

	var createResponse CreateUserResponse
	json.NewDecoder(createW.Body).Decode(&createResponse)

	// Step 2: Join with invite (server generates SSH key)
	joinBody := map[string]string{
		"invite_token": createResponse.InviteToken,
	}
	joinBodyJSON, _ := json.Marshal(joinBody)

	joinReq := httptest.NewRequest(http.MethodPost, "/api/join", bytes.NewReader(joinBodyJSON))
	joinReq.Header.Set("Content-Type", "application/json")
	joinReq.Host = "teamserver.example.com"
	joinW := httptest.NewRecorder()

	server.mux.ServeHTTP(joinW, joinReq)

	if joinW.Code != http.StatusOK {
		t.Fatalf("Failed to join: %s", joinW.Body.String())
	}

	var joinResponse JoinResponse
	json.NewDecoder(joinW.Body).Decode(&joinResponse)

	if joinResponse.SessionToken == "" {
		t.Error("Session token should not be empty")
	}
	if joinResponse.User.Name != "newdev" {
		t.Errorf("Expected user name newdev, got %s", joinResponse.User.Name)
	}

	// Step 3: Verify private key is returned
	if joinResponse.PrivateKey == "" {
		t.Error("Private key should be returned on join")
	}
	if !strings.HasPrefix(joinResponse.PrivateKey, "-----BEGIN OPENSSH PRIVATE KEY-----") {
		t.Errorf("Private key should be in OpenSSH format, got: %s...", joinResponse.PrivateKey[:50])
	}

	// Step 4: Use session token to access /me
	meReq := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+joinResponse.SessionToken)
	meW := httptest.NewRecorder()

	server.mux.ServeHTTP(meW, meReq)

	if meW.Code != http.StatusOK {
		t.Errorf("Expected status 200 for /me, got %d", meW.Code)
	}

	var user User
	json.NewDecoder(meW.Body).Decode(&user)

	if user.Name != "newdev" {
		t.Errorf("Expected user name newdev, got %s", user.Name)
	}
	if user.PublicKey == "" {
		t.Error("User should have a public key stored")
	}
	if !strings.HasPrefix(user.PublicKey, "ssh-ed25519 ") {
		t.Errorf("Public key should be Ed25519, got: %s", user.PublicKey)
	}
}

func TestEnvironmentCRUD(t *testing.T) {
	server, adminToken, cleanup := setupTestServerWithAdmin(t)
	defer cleanup()

	// First create a project (required for environments)
	projectBody := `{"name": "testproject", "description": "Test project"}`
	projectReq := httptest.NewRequest(http.MethodPost, "/api/admin/projects", bytes.NewBufferString(projectBody))
	projectReq.Header.Set("Authorization", "Bearer "+adminToken)
	projectReq.Header.Set("Content-Type", "application/json")
	projectW := httptest.NewRecorder()

	server.mux.ServeHTTP(projectW, projectReq)

	if projectW.Code != http.StatusOK {
		t.Fatalf("Failed to create project: %s", projectW.Body.String())
	}

	// Create environment
	createBody := `{
		"name": "production",
		"project": "testproject",
		"host": "prod.example.com",
		"port": 22,
		"deploy_user": "deploy",
		"deploy_key": "-----BEGIN OPENSSH PRIVATE KEY-----\ntest\n-----END OPENSSH PRIVATE KEY-----"
	}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/admin/environments", bytes.NewBufferString(createBody))
	createReq.Header.Set("Authorization", "Bearer "+adminToken)
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()

	server.mux.ServeHTTP(createW, createReq)

	if createW.Code != http.StatusOK {
		t.Fatalf("Failed to create environment: %s", createW.Body.String())
	}

	var env Environment
	json.NewDecoder(createW.Body).Decode(&env)

	if env.Name != "production" {
		t.Errorf("Expected env name production, got %s", env.Name)
	}
	if env.Project != "testproject" {
		t.Errorf("Expected env project testproject, got %s", env.Project)
	}
	if env.DeployKey != "" {
		t.Error("Deploy key should not be returned in response")
	}

	// List environments
	listReq := httptest.NewRequest(http.MethodGet, "/api/admin/environments", nil)
	listReq.Header.Set("Authorization", "Bearer "+adminToken)
	listW := httptest.NewRecorder()

	server.mux.ServeHTTP(listW, listReq)

	if listW.Code != http.StatusOK {
		t.Fatalf("Failed to list environments: %s", listW.Body.String())
	}

	var envs []Environment
	json.NewDecoder(listW.Body).Decode(&envs)

	if len(envs) != 1 {
		t.Errorf("Expected 1 environment, got %d", len(envs))
	}

	// Delete environment (using project/name format)
	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/admin/environments/testproject/production", nil)
	deleteReq.Header.Set("Authorization", "Bearer "+adminToken)
	deleteW := httptest.NewRecorder()

	server.mux.ServeHTTP(deleteW, deleteReq)

	if deleteW.Code != http.StatusOK {
		t.Errorf("Expected status 200 for delete, got %d: %s", deleteW.Code, deleteW.Body.String())
	}

	// Verify deletion
	listReq2 := httptest.NewRequest(http.MethodGet, "/api/admin/environments", nil)
	listReq2.Header.Set("Authorization", "Bearer "+adminToken)
	listW2 := httptest.NewRecorder()

	server.mux.ServeHTTP(listW2, listReq2)

	var envsAfter []Environment
	json.NewDecoder(listW2.Body).Decode(&envsAfter)

	if len(envsAfter) != 0 {
		t.Errorf("Expected 0 environments after delete, got %d", len(envsAfter))
	}
}

func TestAuditLog(t *testing.T) {
	server, adminToken, cleanup := setupTestServerWithAdmin(t)
	defer cleanup()

	// Create a user to generate audit entries
	createBody := `{"name": "audituser", "email": "audit@example.com", "role": "dev"}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/admin/users", bytes.NewBufferString(createBody))
	createReq.Header.Set("Authorization", "Bearer "+adminToken)
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()

	server.mux.ServeHTTP(createW, createReq)

	// Get audit log
	auditReq := httptest.NewRequest(http.MethodGet, "/api/admin/audit", nil)
	auditReq.Header.Set("Authorization", "Bearer "+adminToken)
	auditW := httptest.NewRecorder()

	server.mux.ServeHTTP(auditW, auditReq)

	if auditW.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", auditW.Code)
	}

	var entries []AuditEntry
	json.NewDecoder(auditW.Body).Decode(&entries)

	if len(entries) == 0 {
		t.Error("Expected at least one audit entry")
	}

	// Verify we have a USER_CREATE entry
	found := false
	for _, e := range entries {
		if e.Action == AuditUserCreate {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected to find USER_CREATE audit entry")
	}
}

func TestRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(5, time.Minute)

	// Should allow first 5 requests
	for i := 0; i < 5; i++ {
		if !limiter.Allow("192.168.1.1") {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 6th request should be denied
	if limiter.Allow("192.168.1.1") {
		t.Error("6th request should be denied")
	}

	// Different IP should be allowed
	if !limiter.Allow("192.168.1.2") {
		t.Error("Request from different IP should be allowed")
	}
}

func TestGetClientIP(t *testing.T) {
	// Test without trusted proxies - should always use RemoteAddr
	t.Run("NoTrustedProxies", func(t *testing.T) {
		server := &Server{config: DefaultServerConfig()}
		tests := []struct {
			name       string
			headers    map[string]string
			remoteAddr string
			expected   string
		}{
			{
				name:       "X-Forwarded-For ignored without trusted proxy",
				headers:    map[string]string{"X-Forwarded-For": "10.0.0.1, 10.0.0.2"},
				remoteAddr: "127.0.0.1:8080",
				expected:   "127.0.0.1", // Uses RemoteAddr, not X-Forwarded-For
			},
			{
				name:       "X-Real-IP ignored without trusted proxy",
				headers:    map[string]string{"X-Real-IP": "10.0.0.1"},
				remoteAddr: "127.0.0.1:8080",
				expected:   "127.0.0.1", // Uses RemoteAddr, not X-Real-IP
			},
			{
				name:       "RemoteAddr used directly",
				headers:    map[string]string{},
				remoteAddr: "192.168.1.1:12345",
				expected:   "192.168.1.1",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.RemoteAddr = tt.remoteAddr
				for k, v := range tt.headers {
					req.Header.Set(k, v)
				}

				ip := server.getClientIP(req)
				if ip != tt.expected {
					t.Errorf("Expected IP %s, got %s", tt.expected, ip)
				}
			})
		}
	})

	// Test with trusted proxies
	t.Run("WithTrustedProxies", func(t *testing.T) {
		config := DefaultServerConfig()
		config.Security.TrustedProxies = []string{"127.0.0.1", "10.0.0.0/8"}
		server := &Server{config: config}

		tests := []struct {
			name       string
			headers    map[string]string
			remoteAddr string
			expected   string
		}{
			{
				name:       "X-Forwarded-For from trusted proxy",
				headers:    map[string]string{"X-Forwarded-For": "203.0.113.1, 10.0.0.2"},
				remoteAddr: "127.0.0.1:8080",
				expected:   "203.0.113.1", // Rightmost non-proxy IP
			},
			{
				name:       "X-Real-IP from trusted proxy",
				headers:    map[string]string{"X-Real-IP": "203.0.113.1"},
				remoteAddr: "127.0.0.1:8080",
				expected:   "203.0.113.1",
			},
			{
				name:       "Untrusted proxy - ignore headers",
				headers:    map[string]string{"X-Forwarded-For": "203.0.113.1"},
				remoteAddr: "192.168.1.1:12345",
				expected:   "192.168.1.1", // Untrusted, use RemoteAddr
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.RemoteAddr = tt.remoteAddr
				for k, v := range tt.headers {
					req.Header.Set(k, v)
				}

				ip := server.getClientIP(req)
				if ip != tt.expected {
					t.Errorf("Expected IP %s, got %s", tt.expected, ip)
				}
			})
		}
	})
}

func TestMatchCIDR(t *testing.T) {
	tests := []struct {
		ip     string
		cidr   string
		expect bool
	}{
		{"192.168.1.1", "192.168.1.0/24", true},
		{"192.168.2.1", "192.168.1.0/24", false},
		{"10.0.0.1", "10.0.0.0/8", true},
		{"192.168.1.1", "192.168.1.1", true},
		{"192.168.1.2", "192.168.1.1", false},
	}

	for _, tt := range tests {
		result := matchCIDR(tt.ip, tt.cidr)
		if result != tt.expect {
			t.Errorf("matchCIDR(%s, %s) = %v, expected %v", tt.ip, tt.cidr, result, tt.expect)
		}
	}
}

func TestServerStartStop(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "magebox-server-startstop-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	masterKey, _ := GenerateMasterKey()

	config := &ServerConfig{
		Port:    0, // Use random port
		Host:    "127.0.0.1",
		DataDir: tmpDir,
		TLS: TLSConfig{
			Enabled: false,
		},
		Security: SecurityConfig{
			RateLimitEnabled: false,
		},
	}

	server, err := NewServer(config, masterKey)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Start server in background
	go func() {
		server.Start()
	}()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	// Stop server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Stop(ctx); err != nil {
		t.Errorf("Failed to stop server: %v", err)
	}
}

func TestIPAllowlist(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "magebox-allowlist-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	masterKey, _ := GenerateMasterKey()
	adminToken, _ := GenerateToken(32)
	adminTokenHash, _ := HashToken(adminToken)

	config := &ServerConfig{
		Port:           7443,
		Host:           "127.0.0.1",
		DataDir:        tmpDir,
		AdminTokenHash: adminTokenHash,
		Security: SecurityConfig{
			RateLimitEnabled: false,
			AllowedIPs:       []string{"10.0.0.1"}, // Only allow this IP
		},
	}

	server, err := NewServer(config, masterKey)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer server.storage.Close()

	// Request from non-allowed IP should be forbidden
	req := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()

	server.mux.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}
}

func TestStorageAccess(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	storage := server.GetStorage()
	if storage == nil {
		t.Error("GetStorage should return storage instance")
	}

	crypto := server.GetCrypto()
	if crypto == nil {
		t.Error("GetCrypto should return crypto instance")
	}
}

// Test expired user access
func TestExpiredUserAccess(t *testing.T) {
	server, adminToken, cleanup := setupTestServerWithAdmin(t)
	defer cleanup()

	// Create a user directly with an expired date
	crypto := server.GetCrypto()
	_ = crypto // Need for potential decryption

	pastTime := time.Now().Add(-24 * time.Hour)
	sessionToken, _ := GenerateSessionToken()
	tokenHash, _ := HashToken(sessionToken)

	expiredUser := &User{
		Name:      "expireduser",
		Email:     "expired@example.com",
		Role:      RoleDev,
		TokenHash: tokenHash,
		ExpiresAt: &pastTime,
		CreatedBy: "test",
	}

	if err := server.storage.CreateUser(expiredUser); err != nil {
		t.Fatalf("Failed to create expired user: %v", err)
	}

	// Try to access with expired user token
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Authorization", "Bearer "+sessionToken)
	w := httptest.NewRecorder()

	server.mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for expired user, got %d", w.Code)
	}

	// Admin should still work
	adminReq := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
	adminReq.Header.Set("Authorization", "Bearer "+adminToken)
	adminW := httptest.NewRecorder()

	server.mux.ServeHTTP(adminW, adminReq)

	if adminW.Code != http.StatusOK {
		t.Errorf("Expected status 200 for admin, got %d", adminW.Code)
	}
}

func TestDataDirCreation(t *testing.T) {
	tmpBase, err := os.MkdirTemp("", "magebox-datadir-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpBase)

	masterKey, _ := GenerateMasterKey()

	// Use a nested path that doesn't exist
	dataDir := filepath.Join(tmpBase, "nested", "data", "dir")

	config := &ServerConfig{
		Port:    7443,
		Host:    "127.0.0.1",
		DataDir: dataDir,
		Security: SecurityConfig{
			RateLimitEnabled: false,
		},
	}

	server, err := NewServer(config, masterKey)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer server.storage.Close()

	// Verify directory was created
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		t.Error("Data directory should have been created")
	}
}

func TestLoginAttemptTracker(t *testing.T) {
	lat := NewLoginAttemptTracker(3)

	t.Run("initial state", func(t *testing.T) {
		if lat.IsLocked("10.0.0.1") {
			t.Error("New IP should not be locked")
		}
		if lat.GetFailureCount("10.0.0.1") != 0 {
			t.Error("New IP should have 0 failures")
		}
	})

	t.Run("record failures", func(t *testing.T) {
		ip := "10.0.0.2"

		// First failure
		locked := lat.RecordFailure(ip)
		if locked {
			t.Error("First failure should not lock")
		}
		if lat.GetFailureCount(ip) != 1 {
			t.Errorf("Expected 1 failure, got %d", lat.GetFailureCount(ip))
		}

		// Second failure
		locked = lat.RecordFailure(ip)
		if locked {
			t.Error("Second failure should not lock")
		}

		// Third failure - should lock
		locked = lat.RecordFailure(ip)
		if !locked {
			t.Error("Third failure should lock")
		}
		if !lat.IsLocked(ip) {
			t.Error("IP should be locked after 3 failures")
		}
	})

	t.Run("clear attempts", func(t *testing.T) {
		ip := "10.0.0.3"

		lat.RecordFailure(ip)
		lat.RecordFailure(ip)
		if lat.GetFailureCount(ip) != 2 {
			t.Errorf("Expected 2 failures, got %d", lat.GetFailureCount(ip))
		}

		lat.ClearAttempts(ip)
		if lat.GetFailureCount(ip) != 0 {
			t.Error("Failures should be cleared")
		}
		if lat.IsLocked(ip) {
			t.Error("IP should not be locked after clearing")
		}
	})

	t.Run("different IPs independent", func(t *testing.T) {
		ip1 := "10.0.0.4"
		ip2 := "10.0.0.5"

		lat.RecordFailure(ip1)
		lat.RecordFailure(ip1)
		lat.RecordFailure(ip1)

		if !lat.IsLocked(ip1) {
			t.Error("IP1 should be locked")
		}
		if lat.IsLocked(ip2) {
			t.Error("IP2 should not be locked")
		}
	})
}

func TestRequireAdminMFA(t *testing.T) {
	config := DefaultServerConfig()

	server := &Server{
		config: config,
	}

	tests := []struct {
		name       string
		user       *User
		mfaSetting string
		wantErr    bool
	}{
		{
			name:       "nil user",
			user:       nil,
			mfaSetting: "required",
			wantErr:    false,
		},
		{
			name:       "non-admin user with required MFA",
			user:       &User{Name: "dev", Role: RoleDev},
			mfaSetting: "required",
			wantErr:    false,
		},
		{
			name:       "admin with MFA enabled and required",
			user:       &User{Name: "admin", Role: RoleAdmin, MFAEnabled: true},
			mfaSetting: "required",
			wantErr:    false,
		},
		{
			name:       "admin without MFA and required",
			user:       &User{Name: "admin", Role: RoleAdmin, MFAEnabled: false},
			mfaSetting: "required",
			wantErr:    true,
		},
		{
			name:       "admin without MFA but optional",
			user:       &User{Name: "admin", Role: RoleAdmin, MFAEnabled: false},
			mfaSetting: "optional",
			wantErr:    false,
		},
		{
			name:       "admin without MFA but disabled",
			user:       &User{Name: "admin", Role: RoleAdmin, MFAEnabled: false},
			mfaSetting: "disabled",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server.config.Security.AdminMFA = tt.mfaSetting
			err := server.requireAdminMFA(tt.user)
			if (err != nil) != tt.wantErr {
				t.Errorf("requireAdminMFA() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoginLockoutIntegration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "magebox-lockout-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	masterKey, _ := GenerateMasterKey()

	config := &ServerConfig{
		Port:    7443,
		Host:    "127.0.0.1",
		DataDir: tmpDir,
		Security: SecurityConfig{
			RateLimitEnabled: false,
			LoginAttempts:    3, // Lock after 3 failed attempts
		},
	}

	server, err := NewServer(config, masterKey)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer server.storage.Close()

	// Make 3 failed login attempts
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		req.RemoteAddr = "192.168.1.100:12345"
		w := httptest.NewRecorder()
		server.mux.ServeHTTP(w, req)
	}

	// 4th attempt should be blocked due to lockout
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()
	server.mux.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 (locked), got %d", w.Code)
	}

	// Different IP should not be locked
	req2 := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req2.Header.Set("Authorization", "Bearer invalid-token")
	req2.RemoteAddr = "192.168.1.101:12345"
	w2 := httptest.NewRecorder()
	server.mux.ServeHTTP(w2, req2)

	if w2.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 (not locked), got %d", w2.Code)
	}
}

func TestMFARequirementForAdmin(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "magebox-mfa-req-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	masterKey, _ := GenerateMasterKey()

	// Create admin token for a user without MFA
	sessionToken, _ := GenerateSessionToken()
	tokenHash, _ := HashToken(sessionToken)

	config := &ServerConfig{
		Port:    7443,
		Host:    "127.0.0.1",
		DataDir: tmpDir,
		Security: SecurityConfig{
			RateLimitEnabled: false,
			AdminMFA:         "required", // MFA required for admin
		},
	}

	server, err := NewServer(config, masterKey)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer server.storage.Close()

	// Create admin user without MFA
	adminUser := &User{
		Name:       "testadmin",
		Email:      "testadmin@example.com",
		Role:       RoleAdmin,
		TokenHash:  tokenHash,
		MFAEnabled: false,
		CreatedBy:  "test",
	}

	if err := server.storage.CreateUser(adminUser); err != nil {
		t.Fatalf("Failed to create admin user: %v", err)
	}

	// Try to access admin endpoint - should be forbidden due to MFA requirement
	req := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
	req.Header.Set("Authorization", "Bearer "+sessionToken)
	w := httptest.NewRecorder()
	server.mux.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 (MFA required), got %d. Body: %s", w.Code, w.Body.String())
	}

	// Enable MFA for the user
	adminUser.MFAEnabled = true
	if err := server.storage.UpdateUser(adminUser); err != nil {
		t.Fatalf("Failed to update admin user: %v", err)
	}

	// Now admin endpoint should be accessible
	req2 := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
	req2.Header.Set("Authorization", "Bearer "+sessionToken)
	w2 := httptest.NewRecorder()
	server.mux.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Expected status 200 after MFA enabled, got %d. Body: %s", w2.Code, w2.Body.String())
	}
}

// SSH Key Generation Integration Tests

func TestJoinFlowWithSSHKeyGeneration(t *testing.T) {
	server, adminToken, cleanup := setupTestServerWithAdmin(t)
	defer cleanup()

	// Create invite
	createBody := `{"name": "sshuser", "email": "sshuser@example.com", "role": "dev"}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/admin/users", bytes.NewBufferString(createBody))
	createReq.Header.Set("Authorization", "Bearer "+adminToken)
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()

	server.mux.ServeHTTP(createW, createReq)

	if createW.Code != http.StatusOK {
		t.Fatalf("Failed to create invite: %s", createW.Body.String())
	}

	var createResponse CreateUserResponse
	json.NewDecoder(createW.Body).Decode(&createResponse)

	// Join with invite (no public_key - server generates)
	joinBody := `{"invite_token": "` + createResponse.InviteToken + `"}`
	joinReq := httptest.NewRequest(http.MethodPost, "/api/join", bytes.NewBufferString(joinBody))
	joinReq.Header.Set("Content-Type", "application/json")
	joinReq.Host = "test.magebox.io"
	joinW := httptest.NewRecorder()

	server.mux.ServeHTTP(joinW, joinReq)

	if joinW.Code != http.StatusOK {
		t.Fatalf("Failed to join: %s", joinW.Body.String())
	}

	var joinResponse JoinResponse
	json.NewDecoder(joinW.Body).Decode(&joinResponse)

	// Verify private key is valid Ed25519
	if !strings.HasPrefix(joinResponse.PrivateKey, "-----BEGIN OPENSSH PRIVATE KEY-----") {
		t.Error("Private key should be in OpenSSH format")
	}
	if !strings.HasSuffix(strings.TrimSpace(joinResponse.PrivateKey), "-----END OPENSSH PRIVATE KEY-----") {
		t.Error("Private key should end with OpenSSH footer")
	}

	// Verify ServerHost is returned
	if joinResponse.ServerHost == "" {
		t.Error("ServerHost should be returned")
	}

	// Verify we can parse the public key that was stored
	user, err := server.storage.GetUser("sshuser")
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	if user.PublicKey == "" {
		t.Error("User should have public key stored")
	}

	keyType, fingerprint, err := ParseSSHPublicKey(user.PublicKey)
	if err != nil {
		t.Fatalf("Failed to parse stored public key: %v", err)
	}

	if keyType != "ssh-ed25519" {
		t.Errorf("Expected key type ssh-ed25519, got %s", keyType)
	}

	if !strings.HasPrefix(fingerprint, "SHA256:") {
		t.Errorf("Fingerprint should start with SHA256:, got %s", fingerprint)
	}
}

func TestEnvironmentAccessForUser(t *testing.T) {
	server, adminToken, cleanup := setupTestServerWithAdmin(t)
	defer cleanup()

	// Create a project
	projectBody := `{"name": "myproject", "description": "Test project"}`
	projectReq := httptest.NewRequest(http.MethodPost, "/api/admin/projects", bytes.NewBufferString(projectBody))
	projectReq.Header.Set("Authorization", "Bearer "+adminToken)
	projectReq.Header.Set("Content-Type", "application/json")
	projectW := httptest.NewRecorder()

	server.mux.ServeHTTP(projectW, projectReq)
	if projectW.Code != http.StatusOK {
		t.Fatalf("Failed to create project: %s", projectW.Body.String())
	}

	// Create an environment
	envBody := `{
		"name": "staging",
		"project": "myproject",
		"host": "staging.example.com",
		"port": 22,
		"deploy_user": "deploy",
		"deploy_key": "-----BEGIN OPENSSH PRIVATE KEY-----\ntest\n-----END OPENSSH PRIVATE KEY-----"
	}`
	envReq := httptest.NewRequest(http.MethodPost, "/api/admin/environments", bytes.NewBufferString(envBody))
	envReq.Header.Set("Authorization", "Bearer "+adminToken)
	envReq.Header.Set("Content-Type", "application/json")
	envW := httptest.NewRecorder()

	server.mux.ServeHTTP(envW, envReq)
	if envW.Code != http.StatusOK {
		t.Fatalf("Failed to create environment: %s", envW.Body.String())
	}

	// Create a user invite
	userBody := `{"name": "envuser", "email": "envuser@example.com", "role": "dev"}`
	userReq := httptest.NewRequest(http.MethodPost, "/api/admin/users", bytes.NewBufferString(userBody))
	userReq.Header.Set("Authorization", "Bearer "+adminToken)
	userReq.Header.Set("Content-Type", "application/json")
	userW := httptest.NewRecorder()

	server.mux.ServeHTTP(userW, userReq)
	if userW.Code != http.StatusOK {
		t.Fatalf("Failed to create user invite: %s", userW.Body.String())
	}

	var userResponse CreateUserResponse
	json.NewDecoder(userW.Body).Decode(&userResponse)

	// User joins first
	joinBody := `{"invite_token": "` + userResponse.InviteToken + `"}`
	joinReq := httptest.NewRequest(http.MethodPost, "/api/join", bytes.NewBufferString(joinBody))
	joinReq.Header.Set("Content-Type", "application/json")
	joinReq.Host = "teamserver.example.com"
	joinW := httptest.NewRecorder()

	server.mux.ServeHTTP(joinW, joinReq)
	if joinW.Code != http.StatusOK {
		t.Fatalf("Failed to join: %s", joinW.Body.String())
	}

	var joinResponse JoinResponse
	json.NewDecoder(joinW.Body).Decode(&joinResponse)

	// Grant access to project (after user has joined)
	accessBody := `{"project": "myproject", "environments": ["staging"]}`
	accessReq := httptest.NewRequest(http.MethodPost, "/api/admin/users/envuser/access", bytes.NewBufferString(accessBody))
	accessReq.Header.Set("Authorization", "Bearer "+adminToken)
	accessReq.Header.Set("Content-Type", "application/json")
	accessW := httptest.NewRecorder()

	server.mux.ServeHTTP(accessW, accessReq)
	if accessW.Code != http.StatusOK {
		t.Fatalf("Failed to grant access: %s", accessW.Body.String())
	}

	// Test /api/environments endpoint
	envListReq := httptest.NewRequest(http.MethodGet, "/api/environments", nil)
	envListReq.Header.Set("Authorization", "Bearer "+joinResponse.SessionToken)
	envListW := httptest.NewRecorder()

	server.mux.ServeHTTP(envListW, envListReq)

	if envListW.Code != http.StatusOK {
		t.Fatalf("Failed to list environments: %s (status: %d)", envListW.Body.String(), envListW.Code)
	}

	var envList []EnvironmentForUser
	json.NewDecoder(envListW.Body).Decode(&envList)

	if len(envList) == 0 {
		t.Error("Environment list should not be empty")
	}

	// Verify environment details (should not contain deploy key)
	found := false
	for _, env := range envList {
		if env.Name == "myproject/staging" || (env.Project == "myproject" && env.Name == "staging") {
			found = true
			if env.Host != "staging.example.com" {
				t.Errorf("Expected host staging.example.com, got %s", env.Host)
			}
			if env.DeployUser != "deploy" {
				t.Errorf("Expected deploy user deploy, got %s", env.DeployUser)
			}
			if env.Port != 22 {
				t.Errorf("Expected port 22, got %d", env.Port)
			}
		}
	}

	if !found {
		t.Errorf("Expected to find staging environment, got: %+v", envList)
	}
}

func TestJoinFlowKeyUniqueness(t *testing.T) {
	server, adminToken, cleanup := setupTestServerWithAdmin(t)
	defer cleanup()

	// Create two users
	users := []string{"user1", "user2"}
	var privateKeys []string

	for _, userName := range users {
		// Create invite
		createBody := `{"name": "` + userName + `", "email": "` + userName + `@example.com", "role": "dev"}`
		createReq := httptest.NewRequest(http.MethodPost, "/api/admin/users", bytes.NewBufferString(createBody))
		createReq.Header.Set("Authorization", "Bearer "+adminToken)
		createReq.Header.Set("Content-Type", "application/json")
		createW := httptest.NewRecorder()

		server.mux.ServeHTTP(createW, createReq)
		if createW.Code != http.StatusOK {
			t.Fatalf("Failed to create invite for %s: %s", userName, createW.Body.String())
		}

		var createResponse CreateUserResponse
		json.NewDecoder(createW.Body).Decode(&createResponse)

		// Join
		joinBody := `{"invite_token": "` + createResponse.InviteToken + `"}`
		joinReq := httptest.NewRequest(http.MethodPost, "/api/join", bytes.NewBufferString(joinBody))
		joinReq.Header.Set("Content-Type", "application/json")
		joinReq.Host = "teamserver.example.com"
		joinW := httptest.NewRecorder()

		server.mux.ServeHTTP(joinW, joinReq)
		if joinW.Code != http.StatusOK {
			t.Fatalf("Failed to join for %s: %s", userName, joinW.Body.String())
		}

		var joinResponse JoinResponse
		json.NewDecoder(joinW.Body).Decode(&joinResponse)

		privateKeys = append(privateKeys, joinResponse.PrivateKey)
	}

	// Verify keys are unique
	if privateKeys[0] == privateKeys[1] {
		t.Error("Each user should have a unique private key")
	}

	// Verify both users have different public keys stored
	user1, _ := server.storage.GetUser("user1")
	user2, _ := server.storage.GetUser("user2")

	if user1.PublicKey == user2.PublicKey {
		t.Error("Each user should have a unique public key")
	}
}

func TestJoinInviteCanOnlyBeUsedOnce(t *testing.T) {
	server, adminToken, cleanup := setupTestServerWithAdmin(t)
	defer cleanup()

	// Create invite
	createBody := `{"name": "onceuser", "email": "once@example.com", "role": "dev"}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/admin/users", bytes.NewBufferString(createBody))
	createReq.Header.Set("Authorization", "Bearer "+adminToken)
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()

	server.mux.ServeHTTP(createW, createReq)
	if createW.Code != http.StatusOK {
		t.Fatalf("Failed to create invite: %s", createW.Body.String())
	}

	var createResponse CreateUserResponse
	json.NewDecoder(createW.Body).Decode(&createResponse)

	// First join should succeed
	joinBody := `{"invite_token": "` + createResponse.InviteToken + `"}`
	joinReq := httptest.NewRequest(http.MethodPost, "/api/join", bytes.NewBufferString(joinBody))
	joinReq.Header.Set("Content-Type", "application/json")
	joinReq.Host = "teamserver.example.com"
	joinW := httptest.NewRecorder()

	server.mux.ServeHTTP(joinW, joinReq)
	if joinW.Code != http.StatusOK {
		t.Fatalf("First join should succeed: %s", joinW.Body.String())
	}

	// Second join with same invite should fail
	joinReq2 := httptest.NewRequest(http.MethodPost, "/api/join", bytes.NewBufferString(joinBody))
	joinReq2.Header.Set("Content-Type", "application/json")
	joinReq2.Host = "teamserver.example.com"
	joinW2 := httptest.NewRecorder()

	server.mux.ServeHTTP(joinW2, joinReq2)
	if joinW2.Code == http.StatusOK {
		t.Error("Second join with same invite should fail")
	}
}

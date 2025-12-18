/**
 * Created by Qoliber
 *
 * @category    Qoliber
 * @package     MageBox
 * @author      Jakub Winkler <jwinkler@qoliber.com>
 */

package teamserver

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupTestStorage(t *testing.T) (*Storage, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "magebox-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	key, err := GenerateMasterKey()
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to generate master key: %v", err)
	}

	crypto, err := NewCrypto(key)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create crypto: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	storage, err := NewStorage(dbPath, crypto)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create storage: %v", err)
	}

	cleanup := func() {
		storage.Close()
		os.RemoveAll(tmpDir)
	}

	return storage, cleanup
}

func TestNewStorage(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	if storage == nil {
		t.Error("Storage should not be nil")
	}
}

func TestNewStorageCreatesDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "magebox-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	key, _ := GenerateMasterKey()
	crypto, _ := NewCrypto(key)

	// Use a nested path that doesn't exist
	dbPath := filepath.Join(tmpDir, "nested", "path", "test.db")
	storage, err := NewStorage(dbPath, crypto)
	if err != nil {
		t.Fatalf("Failed to create storage with nested path: %v", err)
	}
	defer storage.Close()

	// Check directory was created
	if _, err := os.Stat(filepath.Dir(dbPath)); os.IsNotExist(err) {
		t.Error("Directory should have been created")
	}
}

// User tests

func TestCreateAndGetUser(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	expires := time.Now().Add(24 * time.Hour)
	user := &User{
		Name:      "testuser",
		Email:     "test@example.com",
		Role:      RoleDev,
		PublicKey: "ssh-rsa AAAA...",
		TokenHash: "hash123",
		ExpiresAt: &expires,
		CreatedBy: "admin",
	}

	err := storage.CreateUser(user)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	if user.ID == 0 {
		t.Error("User ID should be set after creation")
	}

	// Retrieve user
	retrieved, err := storage.GetUser("testuser")
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}

	if retrieved.Name != user.Name {
		t.Errorf("Name mismatch: got %s, want %s", retrieved.Name, user.Name)
	}
	if retrieved.Email != user.Email {
		t.Errorf("Email mismatch: got %s, want %s", retrieved.Email, user.Email)
	}
	if retrieved.Role != user.Role {
		t.Errorf("Role mismatch: got %s, want %s", retrieved.Role, user.Role)
	}
	if retrieved.PublicKey != user.PublicKey {
		t.Errorf("PublicKey mismatch: got %s, want %s", retrieved.PublicKey, user.PublicKey)
	}
	if retrieved.CreatedBy != user.CreatedBy {
		t.Errorf("CreatedBy mismatch: got %s, want %s", retrieved.CreatedBy, user.CreatedBy)
	}
}

func TestCreateUserWithMFASecret(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	user := &User{
		Name:       "mfauser",
		Email:      "mfa@example.com",
		Role:       RoleAdmin,
		MFASecret:  "JBSWY3DPEHPK3PXP",
		MFAEnabled: true,
		CreatedBy:  "admin",
	}

	err := storage.CreateUser(user)
	if err != nil {
		t.Fatalf("CreateUser with MFA failed: %v", err)
	}

	retrieved, err := storage.GetUser("mfauser")
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}

	// MFA secret should be decrypted and match
	if retrieved.MFASecret != user.MFASecret {
		t.Errorf("MFASecret mismatch: got %s, want %s", retrieved.MFASecret, user.MFASecret)
	}
	if !retrieved.MFAEnabled {
		t.Error("MFAEnabled should be true")
	}
}

func TestGetUserNotFound(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	_, err := storage.GetUser("nonexistent")
	if err == nil {
		t.Error("GetUser should fail for non-existent user")
	}
}

func TestGetUserByTokenHash(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	tokenHash := "uniquetokenhash123"
	user := &User{
		Name:      "tokenuser",
		Email:     "token@example.com",
		Role:      RoleDev,
		TokenHash: tokenHash,
		CreatedBy: "admin",
	}

	err := storage.CreateUser(user)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	retrieved, err := storage.GetUserByTokenHash(tokenHash)
	if err != nil {
		t.Fatalf("GetUserByTokenHash failed: %v", err)
	}

	if retrieved.Name != user.Name {
		t.Errorf("Name mismatch: got %s, want %s", retrieved.Name, user.Name)
	}
}

func TestListUsers(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	users := []User{
		{Name: "alice", Email: "alice@example.com", Role: RoleAdmin, CreatedBy: "system"},
		{Name: "bob", Email: "bob@example.com", Role: RoleDev, CreatedBy: "alice"},
		{Name: "charlie", Email: "charlie@example.com", Role: RoleReadonly, CreatedBy: "alice"},
	}

	for i := range users {
		if err := storage.CreateUser(&users[i]); err != nil {
			t.Fatalf("CreateUser failed: %v", err)
		}
	}

	list, err := storage.ListUsers()
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}

	if len(list) != 3 {
		t.Errorf("Expected 3 users, got %d", len(list))
	}

	// Should be sorted by name
	if list[0].Name != "alice" || list[1].Name != "bob" || list[2].Name != "charlie" {
		t.Error("Users should be sorted by name")
	}
}

func TestUpdateUser(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	user := &User{
		Name:      "updateuser",
		Email:     "original@example.com",
		Role:      RoleDev,
		CreatedBy: "admin",
	}

	err := storage.CreateUser(user)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Update user
	user.Email = "updated@example.com"
	user.Role = RoleAdmin
	user.MFASecret = "NEWSECRET"
	user.MFAEnabled = true

	err = storage.UpdateUser(user)
	if err != nil {
		t.Fatalf("UpdateUser failed: %v", err)
	}

	retrieved, err := storage.GetUser("updateuser")
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}

	if retrieved.Email != "updated@example.com" {
		t.Errorf("Email not updated: got %s", retrieved.Email)
	}
	if retrieved.Role != RoleAdmin {
		t.Errorf("Role not updated: got %s", retrieved.Role)
	}
	if retrieved.MFASecret != "NEWSECRET" {
		t.Error("MFASecret not updated")
	}
}

func TestDeleteUser(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	user := &User{
		Name:      "deleteuser",
		Email:     "delete@example.com",
		Role:      RoleDev,
		CreatedBy: "admin",
	}

	err := storage.CreateUser(user)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	err = storage.DeleteUser("deleteuser")
	if err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}

	_, err = storage.GetUser("deleteuser")
	if err == nil {
		t.Error("User should be deleted")
	}
}

func TestDeleteUserNotFound(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	err := storage.DeleteUser("nonexistent")
	if err == nil {
		t.Error("DeleteUser should fail for non-existent user")
	}
}

func TestUpdateUserLastAccess(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	user := &User{
		Name:      "accessuser",
		Email:     "access@example.com",
		Role:      RoleDev,
		CreatedBy: "admin",
	}

	err := storage.CreateUser(user)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	err = storage.UpdateUserLastAccess("accessuser")
	if err != nil {
		t.Fatalf("UpdateUserLastAccess failed: %v", err)
	}

	retrieved, err := storage.GetUser("accessuser")
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}

	if retrieved.LastAccessAt == nil {
		t.Error("LastAccessAt should be set")
	}
}

// Environment tests

func TestCreateAndGetEnvironment(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// First create a project (environments must belong to a project)
	project := &Project{
		Name:        "testproject",
		Description: "Test project",
	}
	if err := storage.CreateProject(project); err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	env := &Environment{
		Name:       "production",
		Project:    "testproject",
		Host:       "prod.example.com",
		Port:       22,
		DeployUser: "deploy",
		DeployKey:  "-----BEGIN OPENSSH PRIVATE KEY-----\ntest\n-----END OPENSSH PRIVATE KEY-----",
	}

	err := storage.CreateEnvironment(env)
	if err != nil {
		t.Fatalf("CreateEnvironment failed: %v", err)
	}

	if env.ID == 0 {
		t.Error("Environment ID should be set after creation")
	}

	retrieved, err := storage.GetEnvironment("testproject", "production")
	if err != nil {
		t.Fatalf("GetEnvironment failed: %v", err)
	}

	if retrieved.Name != env.Name {
		t.Errorf("Name mismatch: got %s, want %s", retrieved.Name, env.Name)
	}
	if retrieved.Project != env.Project {
		t.Errorf("Project mismatch: got %s, want %s", retrieved.Project, env.Project)
	}
	if retrieved.Host != env.Host {
		t.Errorf("Host mismatch: got %s, want %s", retrieved.Host, env.Host)
	}
	if retrieved.Port != env.Port {
		t.Errorf("Port mismatch: got %d, want %d", retrieved.Port, env.Port)
	}
	if retrieved.DeployUser != env.DeployUser {
		t.Errorf("DeployUser mismatch: got %s, want %s", retrieved.DeployUser, env.DeployUser)
	}
	// Deploy key should be decrypted
	if retrieved.DeployKey != env.DeployKey {
		t.Error("DeployKey should be decrypted and match original")
	}
}

func TestGetEnvironmentNotFound(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	_, err := storage.GetEnvironment("nonexistent", "env")
	if err == nil {
		t.Error("GetEnvironment should fail for non-existent environment")
	}
}

func TestListEnvironments(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create projects first
	projects := []string{"project1", "project2"}
	for _, p := range projects {
		if err := storage.CreateProject(&Project{Name: p}); err != nil {
			t.Fatalf("CreateProject failed: %v", err)
		}
	}

	envs := []Environment{
		{Name: "production", Project: "project1", Host: "prod.example.com", DeployUser: "deploy", DeployKey: "key1"},
		{Name: "staging", Project: "project1", Host: "staging.example.com", DeployUser: "deploy", DeployKey: "key2"},
		{Name: "development", Project: "project2", Host: "dev.example.com", DeployUser: "deploy", DeployKey: "key3"},
	}

	for i := range envs {
		if err := storage.CreateEnvironment(&envs[i]); err != nil {
			t.Fatalf("CreateEnvironment failed: %v", err)
		}
	}

	list, err := storage.ListEnvironments()
	if err != nil {
		t.Fatalf("ListEnvironments failed: %v", err)
	}

	if len(list) != 3 {
		t.Errorf("Expected 3 environments, got %d", len(list))
	}

	// Deploy keys should NOT be included in list
	for _, env := range list {
		if env.DeployKey != "" {
			t.Error("DeployKey should not be included in list")
		}
	}
}

func TestListEnvironmentsForUser(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create projects
	projects := []string{"projectA", "projectB"}
	for _, p := range projects {
		if err := storage.CreateProject(&Project{Name: p}); err != nil {
			t.Fatalf("CreateProject failed: %v", err)
		}
	}

	// Create environments in different projects
	envs := []Environment{
		{Name: "production", Project: "projectA", Host: "prod.example.com", DeployUser: "deploy", DeployKey: "key1"},
		{Name: "staging", Project: "projectA", Host: "staging.example.com", DeployUser: "deploy", DeployKey: "key2"},
		{Name: "development", Project: "projectB", Host: "dev.example.com", DeployUser: "deploy", DeployKey: "key3"},
	}

	for i := range envs {
		if err := storage.CreateEnvironment(&envs[i]); err != nil {
			t.Fatalf("CreateEnvironment failed: %v", err)
		}
	}

	// Create users
	user1 := &User{Name: "alice", Email: "alice@example.com", Role: RoleDev, TokenHash: "hash1"}
	user2 := &User{Name: "bob", Email: "bob@example.com", Role: RoleDev, TokenHash: "hash2"}

	if err := storage.CreateUser(user1); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if err := storage.CreateUser(user2); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Grant project access
	if err := storage.GrantProjectAccess("alice", "projectA", "admin"); err != nil {
		t.Fatalf("GrantProjectAccess failed: %v", err)
	}
	if err := storage.GrantProjectAccess("alice", "projectB", "admin"); err != nil {
		t.Fatalf("GrantProjectAccess failed: %v", err)
	}
	if err := storage.GrantProjectAccess("bob", "projectA", "admin"); err != nil {
		t.Fatalf("GrantProjectAccess failed: %v", err)
	}

	// Alice should see all 3 environments (access to both projects)
	aliceEnvs, err := storage.ListEnvironmentsForUser("alice")
	if err != nil {
		t.Fatalf("ListEnvironmentsForUser failed: %v", err)
	}
	if len(aliceEnvs) != 3 {
		t.Errorf("Alice should see 3 environments, got %d", len(aliceEnvs))
	}

	// Bob should see only 2 environments (access to projectA only)
	bobEnvs, err := storage.ListEnvironmentsForUser("bob")
	if err != nil {
		t.Fatalf("ListEnvironmentsForUser failed: %v", err)
	}
	if len(bobEnvs) != 2 {
		t.Errorf("Bob should see 2 environments, got %d", len(bobEnvs))
	}

	// Unknown user should see 0 environments
	noEnvs, err := storage.ListEnvironmentsForUser("nobody")
	if err != nil {
		t.Fatalf("ListEnvironmentsForUser failed: %v", err)
	}
	if len(noEnvs) != 0 {
		t.Errorf("Unknown user should see 0 environments, got %d", len(noEnvs))
	}
}

func TestDeleteEnvironment(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create project first
	if err := storage.CreateProject(&Project{Name: "testproject"}); err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	env := &Environment{
		Name:       "todelete",
		Project:    "testproject",
		Host:       "delete.example.com",
		DeployUser: "deploy",
		DeployKey:  "key",
	}

	if err := storage.CreateEnvironment(env); err != nil {
		t.Fatalf("CreateEnvironment failed: %v", err)
	}

	err := storage.DeleteEnvironment("testproject", "todelete")
	if err != nil {
		t.Fatalf("DeleteEnvironment failed: %v", err)
	}

	_, err = storage.GetEnvironment("testproject", "todelete")
	if err == nil {
		t.Error("Environment should be deleted")
	}
}

// Invite tests

func TestCreateAndGetInvite(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	invite := &Invite{
		TokenHash: "invitetokenhash123",
		UserName:  "newuser",
		Email:     "new@example.com",
		Role:      RoleDev,
		Projects:  []string{"projectA", "projectB"},
		ExpiresAt: time.Now().Add(48 * time.Hour),
	}

	err := storage.CreateInvite(invite)
	if err != nil {
		t.Fatalf("CreateInvite failed: %v", err)
	}

	if invite.ID == 0 {
		t.Error("Invite ID should be set after creation")
	}

	retrieved, err := storage.GetInviteByTokenHash("invitetokenhash123")
	if err != nil {
		t.Fatalf("GetInviteByTokenHash failed: %v", err)
	}

	if retrieved.UserName != invite.UserName {
		t.Errorf("UserName mismatch: got %s, want %s", retrieved.UserName, invite.UserName)
	}
	if retrieved.Email != invite.Email {
		t.Errorf("Email mismatch: got %s, want %s", retrieved.Email, invite.Email)
	}
	if retrieved.Role != invite.Role {
		t.Errorf("Role mismatch: got %s, want %s", retrieved.Role, invite.Role)
	}
	if len(retrieved.Projects) != 2 {
		t.Errorf("Expected 2 projects, got %d", len(retrieved.Projects))
	}
}

func TestGetInviteNotFound(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	_, err := storage.GetInviteByTokenHash("nonexistent")
	if err == nil {
		t.Error("GetInviteByTokenHash should fail for non-existent invite")
	}
}

func TestMarkInviteUsed(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	invite := &Invite{
		TokenHash: "usedinvite",
		UserName:  "user",
		Email:     "user@example.com",
		Role:      RoleDev,
		ExpiresAt: time.Now().Add(48 * time.Hour),
	}

	if err := storage.CreateInvite(invite); err != nil {
		t.Fatalf("CreateInvite failed: %v", err)
	}

	err := storage.MarkInviteUsed(invite.ID)
	if err != nil {
		t.Fatalf("MarkInviteUsed failed: %v", err)
	}

	retrieved, err := storage.GetInviteByTokenHash("usedinvite")
	if err != nil {
		t.Fatalf("GetInviteByTokenHash failed: %v", err)
	}

	if retrieved.UsedAt == nil {
		t.Error("UsedAt should be set")
	}
	if !retrieved.IsUsed() {
		t.Error("IsUsed() should return true")
	}
}

func TestDeleteExpiredInvites(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create expired invite
	expiredInvite := &Invite{
		TokenHash: "expiredinvite",
		UserName:  "expired",
		Email:     "expired@example.com",
		Role:      RoleDev,
		ExpiresAt: time.Now().Add(-24 * time.Hour), // Already expired
	}

	// Create valid invite
	validInvite := &Invite{
		TokenHash: "validinvite",
		UserName:  "valid",
		Email:     "valid@example.com",
		Role:      RoleDev,
		ExpiresAt: time.Now().Add(48 * time.Hour),
	}

	if err := storage.CreateInvite(expiredInvite); err != nil {
		t.Fatalf("CreateInvite failed: %v", err)
	}
	if err := storage.CreateInvite(validInvite); err != nil {
		t.Fatalf("CreateInvite failed: %v", err)
	}

	deleted, err := storage.DeleteExpiredInvites()
	if err != nil {
		t.Fatalf("DeleteExpiredInvites failed: %v", err)
	}

	if deleted != 1 {
		t.Errorf("Expected 1 deleted invite, got %d", deleted)
	}

	// Expired invite should be gone
	_, err = storage.GetInviteByTokenHash("expiredinvite")
	if err == nil {
		t.Error("Expired invite should be deleted")
	}

	// Valid invite should remain
	_, err = storage.GetInviteByTokenHash("validinvite")
	if err != nil {
		t.Error("Valid invite should still exist")
	}
}

// Audit tests

func TestCreateAuditEntry(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	entry := &AuditEntry{
		UserName:  "admin",
		Action:    AuditUserCreate,
		Details:   "Created user: testuser",
		IPAddress: "192.168.1.1",
	}

	err := storage.CreateAuditEntry(entry)
	if err != nil {
		t.Fatalf("CreateAuditEntry failed: %v", err)
	}

	if entry.ID == 0 {
		t.Error("Entry ID should be set after creation")
	}
	if entry.Hash == "" {
		t.Error("Entry hash should be computed")
	}
	if entry.PrevHash != "" {
		t.Error("First entry should have empty PrevHash")
	}
}

func TestAuditHashChain(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create multiple entries
	entries := []AuditEntry{
		{UserName: "admin", Action: AuditUserCreate, Details: "Created user: user1", IPAddress: "127.0.0.1"},
		{UserName: "admin", Action: AuditUserCreate, Details: "Created user: user2", IPAddress: "127.0.0.1"},
		{UserName: "admin", Action: AuditEnvCreate, Details: "Created env: prod", IPAddress: "127.0.0.1"},
	}

	for i := range entries {
		if err := storage.CreateAuditEntry(&entries[i]); err != nil {
			t.Fatalf("CreateAuditEntry failed: %v", err)
		}
	}

	// Verify chain
	valid, idx, err := storage.VerifyAuditLog()
	if err != nil {
		t.Fatalf("VerifyAuditLog failed: %v", err)
	}
	if !valid {
		t.Errorf("Audit chain should be valid, failed at index %d", idx)
	}
}

func TestListAuditEntries(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create entries
	entries := []AuditEntry{
		{UserName: "admin", Action: AuditUserCreate, Details: "Entry 1", IPAddress: "127.0.0.1"},
		{UserName: "dev", Action: AuditEnvAccess, Details: "Entry 2", IPAddress: "192.168.1.1"},
		{UserName: "admin", Action: AuditUserRemove, Details: "Entry 3", IPAddress: "127.0.0.1"},
	}

	for i := range entries {
		if err := storage.CreateAuditEntry(&entries[i]); err != nil {
			t.Fatalf("CreateAuditEntry failed: %v", err)
		}
	}

	// List all
	all, err := storage.ListAuditEntries(nil, nil, "", "", 0)
	if err != nil {
		t.Fatalf("ListAuditEntries failed: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(all))
	}

	// Filter by user
	adminEntries, err := storage.ListAuditEntries(nil, nil, "admin", "", 0)
	if err != nil {
		t.Fatalf("ListAuditEntries failed: %v", err)
	}
	if len(adminEntries) != 2 {
		t.Errorf("Expected 2 admin entries, got %d", len(adminEntries))
	}

	// Filter by action
	userCreateEntries, err := storage.ListAuditEntries(nil, nil, "", AuditUserCreate, 0)
	if err != nil {
		t.Fatalf("ListAuditEntries failed: %v", err)
	}
	if len(userCreateEntries) != 1 {
		t.Errorf("Expected 1 USER_CREATE entry, got %d", len(userCreateEntries))
	}

	// Limit
	limited, err := storage.ListAuditEntries(nil, nil, "", "", 2)
	if err != nil {
		t.Fatalf("ListAuditEntries failed: %v", err)
	}
	if len(limited) != 2 {
		t.Errorf("Expected 2 entries with limit, got %d", len(limited))
	}
}

func TestListAuditEntriesByDateRange(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create entry
	entry := &AuditEntry{
		UserName:  "admin",
		Action:    AuditUserCreate,
		Details:   "Test entry",
		IPAddress: "127.0.0.1",
	}
	if err := storage.CreateAuditEntry(entry); err != nil {
		t.Fatalf("CreateAuditEntry failed: %v", err)
	}

	// Query with date range that includes entry
	from := time.Now().Add(-1 * time.Hour)
	to := time.Now().Add(1 * time.Hour)

	results, err := storage.ListAuditEntries(&from, &to, "", "", 0)
	if err != nil {
		t.Fatalf("ListAuditEntries failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 entry in date range, got %d", len(results))
	}

	// Query with date range that excludes entry
	futureFrom := time.Now().Add(1 * time.Hour)
	futureTo := time.Now().Add(2 * time.Hour)

	noResults, err := storage.ListAuditEntries(&futureFrom, &futureTo, "", "", 0)
	if err != nil {
		t.Fatalf("ListAuditEntries failed: %v", err)
	}
	if len(noResults) != 0 {
		t.Errorf("Expected 0 entries in future date range, got %d", len(noResults))
	}
}

// Config tests

func TestSetAndGetConfig(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	err := storage.SetConfig("test_key", "test_value")
	if err != nil {
		t.Fatalf("SetConfig failed: %v", err)
	}

	value, err := storage.GetConfig("test_key")
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}

	if value != "test_value" {
		t.Errorf("Config value mismatch: got %s, want test_value", value)
	}
}

func TestGetConfigNotFound(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	value, err := storage.GetConfig("nonexistent")
	if err != nil {
		t.Fatalf("GetConfig should not error for missing key: %v", err)
	}
	if value != "" {
		t.Errorf("Missing config should return empty string, got %s", value)
	}
}

func TestSetConfigOverwrite(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	err := storage.SetConfig("overwrite_key", "original")
	if err != nil {
		t.Fatalf("SetConfig failed: %v", err)
	}

	err = storage.SetConfig("overwrite_key", "updated")
	if err != nil {
		t.Fatalf("SetConfig update failed: %v", err)
	}

	value, err := storage.GetConfig("overwrite_key")
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}

	if value != "updated" {
		t.Errorf("Config should be updated: got %s, want updated", value)
	}
}

// Transaction test

func TestTransaction(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Successful transaction
	err := storage.Transaction(func(tx *sql.Tx) error {
		_, err := tx.Exec("INSERT INTO config (key, value) VALUES (?, ?)", "tx_key", "tx_value")
		return err
	})
	if err != nil {
		t.Fatalf("Transaction failed: %v", err)
	}

	value, _ := storage.GetConfig("tx_key")
	if value != "tx_value" {
		t.Error("Transaction should have committed")
	}
}

func TestTransactionRollback(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Failed transaction - should rollback
	err := storage.Transaction(func(tx *sql.Tx) error {
		_, err := tx.Exec("INSERT INTO config (key, value) VALUES (?, ?)", "rollback_key", "value")
		if err != nil {
			return err
		}
		// Force error
		return &testError{msg: "intentional error"}
	})

	if err == nil {
		t.Error("Transaction should have failed")
	}

	value, _ := storage.GetConfig("rollback_key")
	if value != "" {
		t.Error("Transaction should have rolled back")
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

// Edge cases

func TestCreateDuplicateUser(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	user := &User{
		Name:      "duplicate",
		Email:     "dup@example.com",
		Role:      RoleDev,
		CreatedBy: "admin",
	}

	if err := storage.CreateUser(user); err != nil {
		t.Fatalf("First CreateUser failed: %v", err)
	}

	user2 := &User{
		Name:      "duplicate",
		Email:     "dup2@example.com",
		Role:      RoleDev,
		CreatedBy: "admin",
	}

	err := storage.CreateUser(user2)
	if err == nil {
		t.Error("Creating duplicate user should fail")
	}
}

func TestCreateDuplicateEnvironment(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create project first
	if err := storage.CreateProject(&Project{Name: "testproject"}); err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	env := &Environment{
		Name:       "duplicate",
		Project:    "testproject",
		Host:       "dup.example.com",
		DeployUser: "deploy",
		DeployKey:  "key",
	}

	if err := storage.CreateEnvironment(env); err != nil {
		t.Fatalf("First CreateEnvironment failed: %v", err)
	}

	env2 := &Environment{
		Name:       "duplicate",
		Project:    "testproject",
		Host:       "dup2.example.com",
		DeployUser: "deploy",
		DeployKey:  "key2",
	}

	err := storage.CreateEnvironment(env2)
	if err == nil {
		t.Error("Creating duplicate environment in same project should fail")
	}

	// Creating same name in different project should work
	if err := storage.CreateProject(&Project{Name: "otherproject"}); err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	env3 := &Environment{
		Name:       "duplicate",
		Project:    "otherproject",
		Host:       "dup3.example.com",
		DeployUser: "deploy",
		DeployKey:  "key3",
	}

	if err := storage.CreateEnvironment(env3); err != nil {
		t.Errorf("Creating environment with same name in different project should succeed: %v", err)
	}
}

// Project tests

func TestCreateAndGetProject(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	project := &Project{
		Name:        "testproject",
		Description: "Test project description",
		CreatedBy:   "admin",
	}

	err := storage.CreateProject(project)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	if project.ID == 0 {
		t.Error("Project ID should be set after creation")
	}

	retrieved, err := storage.GetProject("testproject")
	if err != nil {
		t.Fatalf("GetProject failed: %v", err)
	}

	if retrieved.Name != project.Name {
		t.Errorf("Name mismatch: got %s, want %s", retrieved.Name, project.Name)
	}
	if retrieved.Description != project.Description {
		t.Errorf("Description mismatch: got %s, want %s", retrieved.Description, project.Description)
	}
	if retrieved.CreatedBy != project.CreatedBy {
		t.Errorf("CreatedBy mismatch: got %s, want %s", retrieved.CreatedBy, project.CreatedBy)
	}
}

func TestListProjects(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	projects := []string{"alpha", "beta", "gamma"}
	for _, name := range projects {
		if err := storage.CreateProject(&Project{Name: name}); err != nil {
			t.Fatalf("CreateProject failed: %v", err)
		}
	}

	list, err := storage.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects failed: %v", err)
	}

	if len(list) != 3 {
		t.Errorf("Expected 3 projects, got %d", len(list))
	}
}

func TestDeleteProject(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	if err := storage.CreateProject(&Project{Name: "todelete"}); err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	err := storage.DeleteProject("todelete")
	if err != nil {
		t.Fatalf("DeleteProject failed: %v", err)
	}

	_, err = storage.GetProject("todelete")
	if err == nil {
		t.Error("Project should be deleted")
	}
}

// User-Project access tests

func TestGrantAndRevokeProjectAccess(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create project and user
	if err := storage.CreateProject(&Project{Name: "project1"}); err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	user := &User{Name: "alice", Email: "alice@example.com", Role: RoleDev, TokenHash: "hash1"}
	if err := storage.CreateUser(user); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Initially user should have no project access
	projects, err := storage.GetUserProjects("alice")
	if err != nil {
		t.Fatalf("GetUserProjects failed: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("User should have no project access initially, got %d", len(projects))
	}

	// Grant access
	if err := storage.GrantProjectAccess("alice", "project1", "admin"); err != nil {
		t.Fatalf("GrantProjectAccess failed: %v", err)
	}

	// User should now have access
	projects, err = storage.GetUserProjects("alice")
	if err != nil {
		t.Fatalf("GetUserProjects failed: %v", err)
	}
	if len(projects) != 1 {
		t.Errorf("User should have 1 project access, got %d", len(projects))
	}
	if projects[0] != "project1" {
		t.Errorf("Expected project1, got %s", projects[0])
	}

	// Get user and check projects field
	retrieved, err := storage.GetUser("alice")
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	if len(retrieved.Projects) != 1 || retrieved.Projects[0] != "project1" {
		t.Errorf("User.Projects mismatch: got %v", retrieved.Projects)
	}

	// Revoke access
	if err := storage.RevokeProjectAccess("alice", "project1"); err != nil {
		t.Fatalf("RevokeProjectAccess failed: %v", err)
	}

	// User should have no access again
	projects, err = storage.GetUserProjects("alice")
	if err != nil {
		t.Fatalf("GetUserProjects failed: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("User should have no project access after revoke, got %d", len(projects))
	}
}

func TestGetProjectUsers(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create project
	if err := storage.CreateProject(&Project{Name: "sharedproject"}); err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create users
	users := []string{"user1", "user2", "user3"}
	for _, name := range users {
		user := &User{Name: name, Email: name + "@example.com", Role: RoleDev, TokenHash: "hash_" + name}
		if err := storage.CreateUser(user); err != nil {
			t.Fatalf("CreateUser failed: %v", err)
		}
	}

	// Grant access to two users
	if err := storage.GrantProjectAccess("user1", "sharedproject", "admin"); err != nil {
		t.Fatalf("GrantProjectAccess failed: %v", err)
	}
	if err := storage.GrantProjectAccess("user2", "sharedproject", "admin"); err != nil {
		t.Fatalf("GrantProjectAccess failed: %v", err)
	}

	// Check project users
	projectUsers, err := storage.GetProjectUsers("sharedproject")
	if err != nil {
		t.Fatalf("GetProjectUsers failed: %v", err)
	}
	if len(projectUsers) != 2 {
		t.Errorf("Expected 2 users, got %d", len(projectUsers))
	}
}

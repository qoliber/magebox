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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Storage handles all database operations
type Storage struct {
	db     *sql.DB
	crypto *Crypto
}

// NewStorage creates a new storage instance
func NewStorage(dbPath string, crypto *Crypto) (*Storage, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=ON")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	s := &Storage{db: db, crypto: crypto}

	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return s, nil
}

// Close closes the database connection
func (s *Storage) Close() error {
	return s.db.Close()
}

// migrate creates or updates database schema
func (s *Storage) migrate() error {
	schema := `
	-- Projects table
	CREATE TABLE IF NOT EXISTS projects (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		description TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		created_by TEXT
	);

	-- Users table
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		email TEXT NOT NULL,
		role TEXT NOT NULL,
		public_key TEXT,
		token_hash TEXT,
		mfa_secret TEXT,
		mfa_enabled INTEGER DEFAULT 0,
		expires_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		created_by TEXT,
		last_access_at DATETIME
	);

	-- User-Project access mapping
	CREATE TABLE IF NOT EXISTS user_projects (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_name TEXT NOT NULL,
		project_name TEXT NOT NULL,
		granted_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		granted_by TEXT,
		UNIQUE(user_name, project_name)
	);

	-- Environments table (belongs to a project)
	CREATE TABLE IF NOT EXISTS environments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		project TEXT NOT NULL,
		host TEXT NOT NULL,
		port INTEGER DEFAULT 22,
		deploy_user TEXT NOT NULL,
		deploy_key TEXT NOT NULL,
		host_key TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(name, project)
	);

	-- Invites table
	CREATE TABLE IF NOT EXISTS invites (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		token_hash TEXT UNIQUE NOT NULL,
		user_name TEXT NOT NULL,
		email TEXT NOT NULL,
		role TEXT NOT NULL,
		projects TEXT,
		expires_at DATETIME NOT NULL,
		used_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Audit log table
	CREATE TABLE IF NOT EXISTS audit_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		user_name TEXT,
		action TEXT NOT NULL,
		details TEXT,
		ip_address TEXT,
		prev_hash TEXT,
		hash TEXT NOT NULL
	);

	-- Config table
	CREATE TABLE IF NOT EXISTS config (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);

	-- Indexes
	CREATE INDEX IF NOT EXISTS idx_projects_name ON projects(name);
	CREATE INDEX IF NOT EXISTS idx_users_name ON users(name);
	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	CREATE INDEX IF NOT EXISTS idx_users_token_hash ON users(token_hash);
	CREATE INDEX IF NOT EXISTS idx_user_projects_user ON user_projects(user_name);
	CREATE INDEX IF NOT EXISTS idx_user_projects_project ON user_projects(project_name);
	CREATE INDEX IF NOT EXISTS idx_environments_name ON environments(name);
	CREATE INDEX IF NOT EXISTS idx_environments_project ON environments(project);
	CREATE INDEX IF NOT EXISTS idx_invites_token_hash ON invites(token_hash);
	CREATE INDEX IF NOT EXISTS idx_audit_log_timestamp ON audit_log(timestamp);
	CREATE INDEX IF NOT EXISTS idx_audit_log_user ON audit_log(user_name);
	CREATE INDEX IF NOT EXISTS idx_audit_log_action ON audit_log(action);
	`

	_, err := s.db.Exec(schema)
	return err
}

// Project operations

// CreateProject creates a new project
func (s *Storage) CreateProject(project *Project) error {
	result, err := s.db.Exec(`
		INSERT INTO projects (name, description, created_by)
		VALUES (?, ?, ?)`,
		project.Name, project.Description, project.CreatedBy)
	if err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	id, _ := result.LastInsertId()
	project.ID = id
	project.CreatedAt = time.Now()
	return nil
}

// GetProject retrieves a project by name
func (s *Storage) GetProject(name string) (*Project, error) {
	project := &Project{}
	var description sql.NullString
	var createdBy sql.NullString

	err := s.db.QueryRow(`
		SELECT id, name, description, created_at, created_by
		FROM projects WHERE name = ?`, name).Scan(
		&project.ID, &project.Name, &description, &project.CreatedAt, &createdBy)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("project not found: %s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	if description.Valid {
		project.Description = description.String
	}
	if createdBy.Valid {
		project.CreatedBy = createdBy.String
	}

	return project, nil
}

// ListProjects returns all projects
func (s *Storage) ListProjects() ([]Project, error) {
	rows, err := s.db.Query(`
		SELECT id, name, description, created_at, created_by
		FROM projects ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var project Project
		var description sql.NullString
		var createdBy sql.NullString

		if err := rows.Scan(&project.ID, &project.Name, &description, &project.CreatedAt, &createdBy); err != nil {
			return nil, fmt.Errorf("failed to scan project: %w", err)
		}

		if description.Valid {
			project.Description = description.String
		}
		if createdBy.Valid {
			project.CreatedBy = createdBy.String
		}

		projects = append(projects, project)
	}

	return projects, nil
}

// DeleteProject deletes a project and all its environments
func (s *Storage) DeleteProject(name string) error {
	// Delete environments first
	_, err := s.db.Exec("DELETE FROM environments WHERE project = ?", name)
	if err != nil {
		return fmt.Errorf("failed to delete project environments: %w", err)
	}

	// Remove user access to this project
	_, err = s.db.Exec("DELETE FROM user_projects WHERE project_name = ?", name)
	if err != nil {
		return fmt.Errorf("failed to remove user access: %w", err)
	}

	// Delete project
	result, err := s.db.Exec("DELETE FROM projects WHERE name = ?", name)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("project not found: %s", name)
	}
	return nil
}

// User-Project access operations

// GrantProjectAccess grants a user access to a project
func (s *Storage) GrantProjectAccess(userName, projectName, grantedBy string) error {
	_, err := s.db.Exec(`
		INSERT OR IGNORE INTO user_projects (user_name, project_name, granted_by)
		VALUES (?, ?, ?)`,
		userName, projectName, grantedBy)
	if err != nil {
		return fmt.Errorf("failed to grant project access: %w", err)
	}
	return nil
}

// RevokeProjectAccess revokes a user's access to a project
func (s *Storage) RevokeProjectAccess(userName, projectName string) error {
	result, err := s.db.Exec(`
		DELETE FROM user_projects WHERE user_name = ? AND project_name = ?`,
		userName, projectName)
	if err != nil {
		return fmt.Errorf("failed to revoke project access: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("user %s does not have access to project %s", userName, projectName)
	}
	return nil
}

// GetUserProjects returns all projects a user has access to
func (s *Storage) GetUserProjects(userName string) ([]string, error) {
	rows, err := s.db.Query(`
		SELECT project_name FROM user_projects WHERE user_name = ? ORDER BY project_name`,
		userName)
	if err != nil {
		return nil, fmt.Errorf("failed to get user projects: %w", err)
	}
	defer rows.Close()

	var projects []string
	for rows.Next() {
		var projectName string
		if err := rows.Scan(&projectName); err != nil {
			return nil, fmt.Errorf("failed to scan project: %w", err)
		}
		projects = append(projects, projectName)
	}

	return projects, nil
}

// GetProjectUsers returns all users with access to a project
func (s *Storage) GetProjectUsers(projectName string) ([]string, error) {
	rows, err := s.db.Query(`
		SELECT user_name FROM user_projects WHERE project_name = ? ORDER BY user_name`,
		projectName)
	if err != nil {
		return nil, fmt.Errorf("failed to get project users: %w", err)
	}
	defer rows.Close()

	var users []string
	for rows.Next() {
		var userName string
		if err := rows.Scan(&userName); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, userName)
	}

	return users, nil
}

// User operations

// CreateUser creates a new user
func (s *Storage) CreateUser(user *User) error {
	// Encrypt MFA secret if present
	mfaSecret := ""
	if user.MFASecret != "" {
		encrypted, err := s.crypto.EncryptString(user.MFASecret)
		if err != nil {
			return fmt.Errorf("failed to encrypt MFA secret: %w", err)
		}
		mfaSecret = encrypted
	}

	result, err := s.db.Exec(`
		INSERT INTO users (name, email, role, public_key, token_hash, mfa_secret, mfa_enabled, expires_at, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		user.Name, user.Email, user.Role, user.PublicKey, user.TokenHash,
		mfaSecret, user.MFAEnabled, user.ExpiresAt, user.CreatedBy)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	id, _ := result.LastInsertId()
	user.ID = id
	user.CreatedAt = time.Now()
	return nil
}

// GetUser retrieves a user by name
func (s *Storage) GetUser(name string) (*User, error) {
	user := &User{}
	var mfaSecret sql.NullString
	var expiresAt sql.NullTime
	var lastAccessAt sql.NullTime

	err := s.db.QueryRow(`
		SELECT id, name, email, role, public_key, token_hash, mfa_secret, mfa_enabled,
		       expires_at, created_at, created_by, last_access_at
		FROM users WHERE name = ?`, name).Scan(
		&user.ID, &user.Name, &user.Email, &user.Role, &user.PublicKey, &user.TokenHash,
		&mfaSecret, &user.MFAEnabled, &expiresAt, &user.CreatedAt, &user.CreatedBy, &lastAccessAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found: %s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Decrypt MFA secret if present
	if mfaSecret.Valid && mfaSecret.String != "" {
		decrypted, err := s.crypto.DecryptString(mfaSecret.String)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt MFA secret: %w", err)
		}
		user.MFASecret = decrypted
	}

	if expiresAt.Valid {
		user.ExpiresAt = &expiresAt.Time
	}
	if lastAccessAt.Valid {
		user.LastAccessAt = &lastAccessAt.Time
	}

	// Load user's projects
	projects, err := s.GetUserProjects(name)
	if err != nil {
		return nil, fmt.Errorf("failed to load user projects: %w", err)
	}
	user.Projects = projects

	return user, nil
}

// GetUserByTokenHash retrieves a user by their session token hash
func (s *Storage) GetUserByTokenHash(tokenHash string) (*User, error) {
	user := &User{}
	var mfaSecret sql.NullString
	var expiresAt sql.NullTime
	var lastAccessAt sql.NullTime

	err := s.db.QueryRow(`
		SELECT id, name, email, role, public_key, token_hash, mfa_secret, mfa_enabled,
		       expires_at, created_at, created_by, last_access_at
		FROM users WHERE token_hash = ?`, tokenHash).Scan(
		&user.ID, &user.Name, &user.Email, &user.Role, &user.PublicKey, &user.TokenHash,
		&mfaSecret, &user.MFAEnabled, &expiresAt, &user.CreatedAt, &user.CreatedBy, &lastAccessAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if mfaSecret.Valid && mfaSecret.String != "" {
		decrypted, err := s.crypto.DecryptString(mfaSecret.String)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt MFA secret: %w", err)
		}
		user.MFASecret = decrypted
	}

	if expiresAt.Valid {
		user.ExpiresAt = &expiresAt.Time
	}
	if lastAccessAt.Valid {
		user.LastAccessAt = &lastAccessAt.Time
	}

	// Load user's projects
	projects, err := s.GetUserProjects(user.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to load user projects: %w", err)
	}
	user.Projects = projects

	return user, nil
}

// ListUsers returns all users
func (s *Storage) ListUsers() ([]User, error) {
	rows, err := s.db.Query(`
		SELECT id, name, email, role, public_key, token_hash, mfa_enabled, expires_at, created_at, created_by, last_access_at
		FROM users ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		var expiresAt sql.NullTime
		var lastAccessAt sql.NullTime
		var tokenHash sql.NullString

		if err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.Role, &user.PublicKey,
			&tokenHash, &user.MFAEnabled, &expiresAt, &user.CreatedAt, &user.CreatedBy, &lastAccessAt); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		if tokenHash.Valid {
			user.TokenHash = tokenHash.String
		}
		if expiresAt.Valid {
			user.ExpiresAt = &expiresAt.Time
		}
		if lastAccessAt.Valid {
			user.LastAccessAt = &lastAccessAt.Time
		}

		// Load user's projects
		projects, _ := s.GetUserProjects(user.Name)
		user.Projects = projects

		users = append(users, user)
	}

	return users, nil
}

// UpdateUser updates a user
func (s *Storage) UpdateUser(user *User) error {
	mfaSecret := ""
	if user.MFASecret != "" {
		encrypted, err := s.crypto.EncryptString(user.MFASecret)
		if err != nil {
			return fmt.Errorf("failed to encrypt MFA secret: %w", err)
		}
		mfaSecret = encrypted
	}

	_, err := s.db.Exec(`
		UPDATE users SET email = ?, role = ?, public_key = ?, token_hash = ?,
		       mfa_secret = ?, mfa_enabled = ?, expires_at = ?, last_access_at = ?
		WHERE name = ?`,
		user.Email, user.Role, user.PublicKey, user.TokenHash,
		mfaSecret, user.MFAEnabled, user.ExpiresAt, user.LastAccessAt, user.Name)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

// DeleteUser deletes a user
func (s *Storage) DeleteUser(name string) error {
	result, err := s.db.Exec("DELETE FROM users WHERE name = ?", name)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("user not found: %s", name)
	}
	return nil
}

// UpdateUserLastAccess updates the last access time
func (s *Storage) UpdateUserLastAccess(name string) error {
	_, err := s.db.Exec("UPDATE users SET last_access_at = ? WHERE name = ?", time.Now(), name)
	return err
}

// Environment operations

// CreateEnvironment creates a new environment
func (s *Storage) CreateEnvironment(env *Environment) error {
	// Encrypt deploy key
	encryptedKey, err := s.crypto.EncryptString(env.DeployKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt deploy key: %w", err)
	}

	result, err := s.db.Exec(`
		INSERT INTO environments (name, project, host, port, deploy_user, deploy_key, host_key)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		env.Name, env.Project, env.Host, env.Port, env.DeployUser, encryptedKey, env.HostKey)
	if err != nil {
		return fmt.Errorf("failed to create environment: %w", err)
	}

	id, _ := result.LastInsertId()
	env.ID = id
	env.CreatedAt = time.Now()
	return nil
}

// GetEnvironment retrieves an environment by project/name
func (s *Storage) GetEnvironment(project, name string) (*Environment, error) {
	env := &Environment{}
	var encryptedKey string
	var hostKey sql.NullString

	err := s.db.QueryRow(`
		SELECT id, name, project, host, port, deploy_user, deploy_key, host_key, created_at
		FROM environments WHERE project = ? AND name = ?`, project, name).Scan(
		&env.ID, &env.Name, &env.Project, &env.Host, &env.Port, &env.DeployUser, &encryptedKey, &hostKey, &env.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("environment not found: %s/%s", project, name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get environment: %w", err)
	}

	// Decrypt deploy key
	decrypted, err := s.crypto.DecryptString(encryptedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt deploy key: %w", err)
	}
	env.DeployKey = decrypted
	env.HostKey = hostKey.String

	return env, nil
}

// UpdateEnvironmentHostKey updates the SSH host key fingerprint for an environment
func (s *Storage) UpdateEnvironmentHostKey(project, name, hostKey string) error {
	result, err := s.db.Exec(`
		UPDATE environments SET host_key = ? WHERE project = ? AND name = ?`,
		hostKey, project, name)
	if err != nil {
		return fmt.Errorf("failed to update host key: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("environment not found: %s/%s", project, name)
	}

	return nil
}

// ListEnvironments returns all environments (without deploy keys for security)
func (s *Storage) ListEnvironments() ([]Environment, error) {
	rows, err := s.db.Query(`
		SELECT id, name, project, host, port, deploy_user, created_at
		FROM environments ORDER BY project, name`)
	if err != nil {
		return nil, fmt.Errorf("failed to list environments: %w", err)
	}
	defer rows.Close()

	var envs []Environment
	for rows.Next() {
		var env Environment

		if err := rows.Scan(&env.ID, &env.Name, &env.Project, &env.Host, &env.Port, &env.DeployUser, &env.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan environment: %w", err)
		}

		envs = append(envs, env)
	}

	return envs, nil
}

// ListEnvironmentsByProject returns environments for a specific project
func (s *Storage) ListEnvironmentsByProject(projectName string) ([]Environment, error) {
	rows, err := s.db.Query(`
		SELECT id, name, project, host, port, deploy_user, created_at
		FROM environments WHERE project = ? ORDER BY name`, projectName)
	if err != nil {
		return nil, fmt.Errorf("failed to list environments: %w", err)
	}
	defer rows.Close()

	var envs []Environment
	for rows.Next() {
		var env Environment

		if err := rows.Scan(&env.ID, &env.Name, &env.Project, &env.Host, &env.Port, &env.DeployUser, &env.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan environment: %w", err)
		}

		envs = append(envs, env)
	}

	return envs, nil
}

// ListEnvironmentsForUser returns environments accessible by a user
func (s *Storage) ListEnvironmentsForUser(userName string) ([]Environment, error) {
	// Get user's projects
	projects, err := s.GetUserProjects(userName)
	if err != nil {
		return nil, err
	}

	if len(projects) == 0 {
		return []Environment{}, nil
	}

	// Build query with placeholders
	placeholders := make([]string, len(projects))
	args := make([]interface{}, len(projects))
	for i, p := range projects {
		placeholders[i] = "?"
		args[i] = p
	}

	query := fmt.Sprintf(`
		SELECT id, name, project, host, port, deploy_user, created_at
		FROM environments WHERE project IN (%s) ORDER BY project, name`,
		strings.Join(placeholders, ","))

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list environments: %w", err)
	}
	defer rows.Close()

	var envs []Environment
	for rows.Next() {
		var env Environment

		if err := rows.Scan(&env.ID, &env.Name, &env.Project, &env.Host, &env.Port, &env.DeployUser, &env.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan environment: %w", err)
		}

		envs = append(envs, env)
	}

	return envs, nil
}

// DeleteEnvironment deletes an environment
func (s *Storage) DeleteEnvironment(project, name string) error {
	result, err := s.db.Exec("DELETE FROM environments WHERE project = ? AND name = ?", project, name)
	if err != nil {
		return fmt.Errorf("failed to delete environment: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("environment not found: %s/%s", project, name)
	}
	return nil
}

// Invite operations

// CreateInvite creates a new invite
func (s *Storage) CreateInvite(invite *Invite) error {
	projects := strings.Join(invite.Projects, ",")

	result, err := s.db.Exec(`
		INSERT INTO invites (token_hash, user_name, email, role, projects, expires_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		invite.TokenHash, invite.UserName, invite.Email, invite.Role, projects, invite.ExpiresAt)
	if err != nil {
		return fmt.Errorf("failed to create invite: %w", err)
	}

	id, _ := result.LastInsertId()
	invite.ID = id
	invite.CreatedAt = time.Now()
	return nil
}

// GetInviteByTokenHash retrieves an invite by token hash
func (s *Storage) GetInviteByTokenHash(tokenHash string) (*Invite, error) {
	invite := &Invite{}
	var projectsStr string
	var usedAt sql.NullTime

	err := s.db.QueryRow(`
		SELECT id, token_hash, user_name, email, role, projects, expires_at, used_at, created_at
		FROM invites WHERE token_hash = ?`, tokenHash).Scan(
		&invite.ID, &invite.TokenHash, &invite.UserName, &invite.Email, &invite.Role,
		&projectsStr, &invite.ExpiresAt, &usedAt, &invite.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invite not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get invite: %w", err)
	}

	if projectsStr != "" {
		invite.Projects = strings.Split(projectsStr, ",")
	}
	if usedAt.Valid {
		invite.UsedAt = &usedAt.Time
	}

	return invite, nil
}

// MarkInviteUsed marks an invite as used
func (s *Storage) MarkInviteUsed(id int64) error {
	_, err := s.db.Exec("UPDATE invites SET used_at = ? WHERE id = ?", time.Now(), id)
	return err
}

// DeleteExpiredInvites removes expired invites
func (s *Storage) DeleteExpiredInvites() (int64, error) {
	result, err := s.db.Exec("DELETE FROM invites WHERE expires_at < ? AND used_at IS NULL", time.Now())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// Audit operations

// CreateAuditEntry creates a new audit log entry with hash chain
func (s *Storage) CreateAuditEntry(entry *AuditEntry) error {
	// Get the last entry's hash for chaining
	var prevHash string
	err := s.db.QueryRow("SELECT hash FROM audit_log ORDER BY id DESC LIMIT 1").Scan(&prevHash)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to get last audit hash: %w", err)
	}

	entry.PrevHash = prevHash
	entry.Timestamp = time.Now().UTC()

	// Insert with placeholder hash first to get the ID
	result, err := s.db.Exec(`
		INSERT INTO audit_log (timestamp, user_name, action, details, ip_address, prev_hash, hash)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		entry.Timestamp, entry.UserName, entry.Action, entry.Details, entry.IPAddress,
		entry.PrevHash, "placeholder")
	if err != nil {
		return fmt.Errorf("failed to create audit entry: %w", err)
	}

	id, _ := result.LastInsertId()
	entry.ID = id

	// Compute hash with the actual ID
	entry.Hash = ComputeAuditHash(entry, prevHash)

	// Update with the correct hash
	_, err = s.db.Exec("UPDATE audit_log SET hash = ? WHERE id = ?", entry.Hash, entry.ID)
	if err != nil {
		return fmt.Errorf("failed to update audit hash: %w", err)
	}

	return nil
}

// ListAuditEntries returns audit entries with optional filters
func (s *Storage) ListAuditEntries(from, to *time.Time, userName string, action AuditAction, limit int) ([]AuditEntry, error) {
	query := "SELECT id, timestamp, user_name, action, details, ip_address, prev_hash, hash FROM audit_log WHERE 1=1"
	var args []interface{}

	if from != nil {
		query += " AND timestamp >= ?"
		args = append(args, from.UTC().Format("2006-01-02 15:04:05"))
	}
	if to != nil {
		query += " AND timestamp <= ?"
		args = append(args, to.UTC().Format("2006-01-02 15:04:05"))
	}
	if userName != "" {
		query += " AND user_name = ?"
		args = append(args, userName)
	}
	if action != "" {
		query += " AND action = ?"
		args = append(args, action)
	}

	query += " ORDER BY timestamp DESC"

	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list audit entries: %w", err)
	}
	defer rows.Close()

	var entries []AuditEntry
	for rows.Next() {
		var entry AuditEntry
		var userName sql.NullString
		var details sql.NullString
		var ipAddress sql.NullString

		if err := rows.Scan(&entry.ID, &entry.Timestamp, &userName, &entry.Action,
			&details, &ipAddress, &entry.PrevHash, &entry.Hash); err != nil {
			return nil, fmt.Errorf("failed to scan audit entry: %w", err)
		}

		if userName.Valid {
			entry.UserName = userName.String
		}
		if details.Valid {
			entry.Details = details.String
		}
		if ipAddress.Valid {
			entry.IPAddress = ipAddress.String
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// VerifyAuditLog verifies the integrity of the audit log
func (s *Storage) VerifyAuditLog() (bool, int, error) {
	entries, err := s.ListAuditEntries(nil, nil, "", "", 0)
	if err != nil {
		return false, -1, err
	}

	// Reverse to check in chronological order
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	valid, idx := VerifyAuditChain(entries)
	return valid, idx, nil
}

// DeleteOldAuditEntries removes entries older than retention period
func (s *Storage) DeleteOldAuditEntries(retentionDays int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	result, err := s.db.Exec("DELETE FROM audit_log WHERE timestamp < ?", cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// Config operations

// GetConfig retrieves a config value
func (s *Storage) GetConfig(key string) (string, error) {
	var value string
	err := s.db.QueryRow("SELECT value FROM config WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// SetConfig sets a config value
func (s *Storage) SetConfig(key, value string) error {
	_, err := s.db.Exec(`
		INSERT INTO config (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

// Transaction helper
func (s *Storage) Transaction(fn func(tx *sql.Tx) error) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

// CA key storage constants
const (
	configCAPrivateKey = "ca_private_key"
	configCAPublicKey  = "ca_public_key"
)

// SaveCAKeys stores the CA key pair (private key is encrypted)
func (s *Storage) SaveCAKeys(privateKeyPEM, publicKeySSH string) error {
	// Encrypt the private key before storing
	encryptedPrivateKey, err := s.crypto.EncryptString(privateKeyPEM)
	if err != nil {
		return fmt.Errorf("failed to encrypt CA private key: %w", err)
	}

	// Store both keys
	if err := s.SetConfig(configCAPrivateKey, encryptedPrivateKey); err != nil {
		return fmt.Errorf("failed to store CA private key: %w", err)
	}

	if err := s.SetConfig(configCAPublicKey, publicKeySSH); err != nil {
		return fmt.Errorf("failed to store CA public key: %w", err)
	}

	return nil
}

// GetCAPrivateKey retrieves and decrypts the CA private key
func (s *Storage) GetCAPrivateKey() (string, error) {
	encryptedKey, err := s.GetConfig(configCAPrivateKey)
	if err != nil {
		return "", fmt.Errorf("failed to get CA private key: %w", err)
	}
	if encryptedKey == "" {
		return "", fmt.Errorf("CA private key not found")
	}

	// Decrypt the private key
	privateKey, err := s.crypto.DecryptString(encryptedKey)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt CA private key: %w", err)
	}

	return privateKey, nil
}

// GetCAPublicKey retrieves the CA public key
func (s *Storage) GetCAPublicKey() (string, error) {
	publicKey, err := s.GetConfig(configCAPublicKey)
	if err != nil {
		return "", fmt.Errorf("failed to get CA public key: %w", err)
	}
	if publicKey == "" {
		return "", fmt.Errorf("CA public key not found")
	}

	return publicKey, nil
}

// HasCAKeys checks if CA keys exist
func (s *Storage) HasCAKeys() bool {
	privateKey, _ := s.GetConfig(configCAPrivateKey)
	publicKey, _ := s.GetConfig(configCAPublicKey)
	return privateKey != "" && publicKey != ""
}

/**
 * Created by Qoliber
 *
 * @category    Qoliber
 * @package     MageBox
 * @author      Jakub Winkler <jwinkler@qoliber.com>
 */

package teamserver

import (
	"time"
)

// Role defines user access levels (for server management permissions)
type Role string

const (
	RoleAdmin    Role = "admin"    // Can manage users, projects, environments
	RoleDev      Role = "dev"      // Standard developer access
	RoleReadonly Role = "readonly" // Read-only access
)

// ValidRoles returns all valid roles
func ValidRoles() []Role {
	return []Role{RoleAdmin, RoleDev, RoleReadonly}
}

// IsValid checks if role is valid
func (r Role) IsValid() bool {
	for _, valid := range ValidRoles() {
		if r == valid {
			return true
		}
	}
	return false
}

// CanManageUsers checks if role can manage users
func (r Role) CanManageUsers() bool {
	return r == RoleAdmin
}

// CanManageProjects checks if role can manage projects
func (r Role) CanManageProjects() bool {
	return r == RoleAdmin
}

// CanManageEnvironments checks if role can manage environments
func (r Role) CanManageEnvironments() bool {
	return r == RoleAdmin
}

// Project represents a project that contains environments
type Project struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	CreatedBy   string    `json:"created_by,omitempty"`
}

// User represents a team member
type User struct {
	ID           int64      `json:"id"`
	Name         string     `json:"name"`
	Email        string     `json:"email"`
	Role         Role       `json:"role"`
	Projects     []string   `json:"projects,omitempty"` // Projects user has access to
	PublicKey    string     `json:"public_key,omitempty"`
	TokenHash    string     `json:"-"` // Never expose
	MFASecret    string     `json:"-"` // Never expose
	MFAEnabled   bool       `json:"mfa_enabled"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	CreatedBy    string     `json:"created_by"`
	LastAccessAt *time.Time `json:"last_access_at,omitempty"`
}

// HasProjectAccess checks if user has access to a project
func (u *User) HasProjectAccess(projectName string) bool {
	for _, p := range u.Projects {
		if p == projectName {
			return true
		}
	}
	return false
}

// IsExpired checks if user access has expired
func (u *User) IsExpired() bool {
	if u.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*u.ExpiresAt)
}

// Environment represents a remote server environment (belongs to a project)
type Environment struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`    // Environment name (e.g., "staging", "production")
	Project    string    `json:"project"` // Project this environment belongs to
	Host       string    `json:"host"`    // SSH hostname
	Port       int       `json:"port"`    // SSH port
	DeployUser string    `json:"deploy_user"`
	DeployKey  string    `json:"-"` // Never expose - encrypted private key
	HostKey    string    `json:"-"` // SSH host key fingerprint for verification (SHA256:...)
	CreatedAt  time.Time `json:"created_at"`
}

// GetPort returns port with default fallback
func (e *Environment) GetPort() int {
	if e.Port == 0 {
		return 22
	}
	return e.Port
}

// FullName returns project/environment format
func (e *Environment) FullName() string {
	return e.Project + "/" + e.Name
}

// Invite represents a pending user invitation
type Invite struct {
	ID        int64      `json:"id"`
	TokenHash string     `json:"-"` // Never expose
	UserName  string     `json:"user_name"`
	Email     string     `json:"email"`
	Role      Role       `json:"role"`
	Projects  []string   `json:"projects,omitempty"` // Projects to grant access to
	ExpiresAt time.Time  `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// IsExpired checks if invite has expired
func (i *Invite) IsExpired() bool {
	return time.Now().After(i.ExpiresAt)
}

// IsUsed checks if invite has been used
func (i *Invite) IsUsed() bool {
	return i.UsedAt != nil
}

// AuditAction defines the type of auditable action
type AuditAction string

const (
	// User actions
	AuditUserCreate AuditAction = "USER_CREATE"
	AuditUserRemove AuditAction = "USER_REMOVE"
	AuditUserJoin   AuditAction = "USER_JOIN"
	AuditUserUpdate AuditAction = "USER_UPDATE"
	AuditUserRenew  AuditAction = "USER_RENEW"

	// Environment actions
	AuditEnvCreate AuditAction = "ENV_CREATE"
	AuditEnvRemove AuditAction = "ENV_REMOVE"
	AuditEnvAccess AuditAction = "ENV_ACCESS"

	// Key actions
	AuditKeyDeployed AuditAction = "KEY_DEPLOYED"
	AuditKeyRemoved  AuditAction = "KEY_REMOVED"
	AuditKeyRotated  AuditAction = "KEY_ROTATED"
	AuditKeySync     AuditAction = "KEY_SYNC"

	// Auth actions
	AuditAuthSuccess AuditAction = "AUTH_SUCCESS"
	AuditAuthFailed  AuditAction = "AUTH_FAILED"
	AuditMFASetup    AuditAction = "MFA_SETUP"
	AuditMFAVerify   AuditAction = "MFA_VERIFY"

	// Admin actions
	AuditAdminAction  AuditAction = "ADMIN_ACTION"
	AuditConfigChange AuditAction = "CONFIG_CHANGE"
)

// AuditEntry represents a single audit log entry
type AuditEntry struct {
	ID        int64       `json:"id"`
	Timestamp time.Time   `json:"timestamp"`
	UserName  string      `json:"user_name,omitempty"`
	Action    AuditAction `json:"action"`
	Details   string      `json:"details,omitempty"`
	IPAddress string      `json:"ip_address,omitempty"`
	PrevHash  string      `json:"-"` // Hash chain - previous entry hash
	Hash      string      `json:"-"` // Hash chain - this entry hash
}

// Session represents an authenticated session
type Session struct {
	UserID    int64     `json:"user_id"`
	UserName  string    `json:"user_name"`
	Role      Role      `json:"role"`
	TokenHash string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	IPAddress string    `json:"ip_address"`
}

// IsExpired checks if session has expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port           int    `yaml:"port"`
	Host           string `yaml:"host"`
	AdminTokenHash string `yaml:"admin_token_hash"`
	DataDir        string `yaml:"data_dir"`

	TLS TLSConfig `yaml:"tls"`

	Security SecurityConfig `yaml:"security"`

	CA CAConfig `yaml:"ca"`

	Notifications NotificationConfig `yaml:"notifications"`

	Audit AuditConfig `yaml:"audit"`
}

// TLSConfig holds TLS settings
type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
	AutoTLS  bool   `yaml:"auto_tls"`
	Domain   string `yaml:"domain"`
}

// SecurityConfig holds security settings
type SecurityConfig struct {
	AdminMFA           string   `yaml:"admin_mfa"` // required, optional, disabled
	InviteExpiry       string   `yaml:"invite_expiry"`
	SessionExpiry      string   `yaml:"session_expiry"`
	RateLimitEnabled   bool     `yaml:"rate_limit_enabled"`
	RateLimitPerMinute int      `yaml:"rate_limit_per_minute"`
	LoginAttempts      int      `yaml:"login_attempts"`
	AllowedIPs         []string `yaml:"allowed_ips"`
	DefaultAccessDays  int      `yaml:"default_access_days"`
	TrustedProxies     []string `yaml:"trusted_proxies"` // IPs/CIDRs of trusted reverse proxies (enables X-Forwarded-For)
}

// CAConfig holds SSH Certificate Authority settings
type CAConfig struct {
	Enabled           bool     `yaml:"enabled"`            // Enable SSH CA (default: true)
	CertValidity      string   `yaml:"cert_validity"`      // Certificate validity duration (default: 24h)
	DefaultPrincipals []string `yaml:"default_principals"` // Default principals for certificates (default: ["deploy"])
}

// NotificationConfig holds notification settings
type NotificationConfig struct {
	SMTP    SMTPConfig    `yaml:"smtp"`
	Webhook WebhookConfig `yaml:"webhook"`
}

// SMTPConfig holds email settings
type SMTPConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	From     string `yaml:"from"`
}

// WebhookConfig holds webhook settings
type WebhookConfig struct {
	Enabled bool   `yaml:"enabled"`
	URL     string `yaml:"url"`
}

// AuditConfig holds audit settings
type AuditConfig struct {
	RetentionDays int `yaml:"retention_days"`
}

// DefaultServerConfig returns config with sensible defaults
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Port:    7443,
		Host:    "0.0.0.0",
		DataDir: "/var/lib/magebox/teamserver",
		TLS: TLSConfig{
			Enabled: true,
		},
		Security: SecurityConfig{
			AdminMFA:           "optional",
			InviteExpiry:       "48h",
			SessionExpiry:      "720h", // 30 days
			RateLimitEnabled:   false,
			RateLimitPerMinute: 0,
			LoginAttempts:      5,
			DefaultAccessDays:  90,
		},
		CA: CAConfig{
			Enabled:           true,
			CertValidity:      "24h",
			DefaultPrincipals: []string{"deploy"},
		},
		Audit: AuditConfig{
			RetentionDays: 365,
		},
	}
}

// API Request/Response types

// CreateUserRequest represents user creation request
type CreateUserRequest struct {
	Name       string   `json:"name"`
	Email      string   `json:"email"`
	Role       Role     `json:"role"`
	Projects   []string `json:"projects,omitempty"` // Projects to grant access to
	ExpiryDays int      `json:"expiry_days,omitempty"`
}

// CreateUserResponse represents user creation response
type CreateUserResponse struct {
	User        *User  `json:"user"`
	InviteToken string `json:"invite_token"`
}

// JoinRequest represents user join request
type JoinRequest struct {
	InviteToken string `json:"invite_token"`
}

// JoinResponse represents user join response
type JoinResponse struct {
	SessionToken string               `json:"session_token"`
	PrivateKey   string               `json:"private_key"`           // PEM-encoded private key for user to save
	Certificate  string               `json:"certificate,omitempty"` // SSH certificate (if CA enabled)
	ValidUntil   *time.Time           `json:"valid_until,omitempty"` // Certificate expiry time
	Principals   []string             `json:"principals,omitempty"`  // Allowed principals (usernames)
	User         *User                `json:"user"`
	Environments []EnvironmentForUser `json:"environments"`
	ServerHost   string               `json:"server_host"`             // Team server hostname for key storage
	CAEnabled    bool                 `json:"ca_enabled"`              // Whether SSH CA is enabled
	CAPublicKey  string               `json:"ca_public_key,omitempty"` // CA public key for reference
}

// EnvironmentForUser represents environment info returned to users (for SSH connection)
type EnvironmentForUser struct {
	Name       string `json:"name"` // Full name: project/env
	Project    string `json:"project"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	DeployUser string `json:"deploy_user"` // Username to SSH as
}

// CreateEnvironmentRequest represents environment creation request
type CreateEnvironmentRequest struct {
	Name       string `json:"name"`
	Project    string `json:"project"` // Project this environment belongs to
	Host       string `json:"host"`
	Port       int    `json:"port,omitempty"`
	DeployUser string `json:"deploy_user"`
	DeployKey  string `json:"deploy_key"`
}

// CreateProjectRequest represents project creation request
type CreateProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// GrantAccessRequest represents a request to grant project access to a user
type GrantAccessRequest struct {
	Project string `json:"project"`
}

// RevokeAccessRequest represents a request to revoke project access from a user
type RevokeAccessRequest struct {
	Project string `json:"project"`
}

// ErrorResponse represents an API error
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// SuccessResponse represents a simple success response
type SuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// CertRenewResponse represents certificate renewal response
type CertRenewResponse struct {
	Certificate string    `json:"certificate"` // New SSH certificate
	ValidUntil  time.Time `json:"valid_until"` // Certificate expiry time
	Principals  []string  `json:"principals"`  // Allowed principals
	Serial      uint64    `json:"serial"`      // Certificate serial number
}

// CertInfoResponse represents certificate info response
type CertInfoResponse struct {
	HasCertificate bool       `json:"has_certificate"`
	Certificate    string     `json:"certificate,omitempty"`
	ValidAfter     *time.Time `json:"valid_after,omitempty"`
	ValidUntil     *time.Time `json:"valid_until,omitempty"`
	Principals     []string   `json:"principals,omitempty"`
	KeyID          string     `json:"key_id,omitempty"`
	IsExpired      bool       `json:"is_expired"`
	ExpiresIn      string     `json:"expires_in,omitempty"` // Human-readable time until expiry
}

// CAInfoResponse represents CA info response (for admin)
type CAInfoResponse struct {
	Enabled      bool     `json:"enabled"`
	PublicKey    string   `json:"public_key"`    // CA public key (for deployment)
	CertValidity string   `json:"cert_validity"` // Default certificate validity
	Principals   []string `json:"principals"`    // Default principals
	Fingerprint  string   `json:"fingerprint"`   // CA key fingerprint
}

// Audit actions for certificate operations
const (
	AuditCertIssue AuditAction = "CERT_ISSUE"
	AuditCertRenew AuditAction = "CERT_RENEW"
	AuditCertDeny  AuditAction = "CERT_DENY"
)

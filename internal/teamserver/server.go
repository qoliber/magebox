/**
 * Created by Qoliber
 *
 * @category    Qoliber
 * @package     MageBox
 * @author      Jakub Winkler <jwinkler@qoliber.com>
 */

package teamserver

import (
	"context"
	"crypto/ed25519"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Server represents the team server
type Server struct {
	config       *ServerConfig
	storage      *Storage
	crypto       *Crypto
	deployer     *Deployer
	mfa          *MFAManager
	notifier     *Notifier
	httpServer   *http.Server
	mux          *http.ServeMux
	rateLimiter  *RateLimiter
	loginTracker *LoginAttemptTracker
	logger       *log.Logger
	masterKey    []byte
	serverURL    string
	caPrivateKey ed25519.PrivateKey // CA private key for signing certificates
}

// RateLimiter implements a simple token bucket rate limiter
type RateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

// LoginAttemptTracker tracks failed login attempts
type LoginAttemptTracker struct {
	mu           sync.Mutex
	attempts     map[string][]time.Time // IP -> timestamps of failed attempts
	maxAttempts  int
	lockDuration time.Duration
}

// NewLoginAttemptTracker creates a new login attempt tracker
func NewLoginAttemptTracker(maxAttempts int) *LoginAttemptTracker {
	return &LoginAttemptTracker{
		attempts:     make(map[string][]time.Time),
		maxAttempts:  maxAttempts,
		lockDuration: 15 * time.Minute,
	}
}

// RecordFailure records a failed login attempt and returns whether IP is locked
func (lat *LoginAttemptTracker) RecordFailure(ip string) bool {
	lat.mu.Lock()
	defer lat.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-lat.lockDuration)

	// Filter old attempts
	var recent []time.Time
	for _, t := range lat.attempts[ip] {
		if t.After(windowStart) {
			recent = append(recent, t)
		}
	}

	recent = append(recent, now)
	lat.attempts[ip] = recent

	return len(recent) >= lat.maxAttempts
}

// IsLocked checks if an IP is locked out
func (lat *LoginAttemptTracker) IsLocked(ip string) bool {
	lat.mu.Lock()
	defer lat.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-lat.lockDuration)

	var recent int
	for _, t := range lat.attempts[ip] {
		if t.After(windowStart) {
			recent++
		}
	}

	return recent >= lat.maxAttempts
}

// ClearAttempts clears failed attempts for an IP (on successful login)
func (lat *LoginAttemptTracker) ClearAttempts(ip string) {
	lat.mu.Lock()
	defer lat.mu.Unlock()
	delete(lat.attempts, ip)
}

// GetFailureCount returns the number of recent failures for an IP
func (lat *LoginAttemptTracker) GetFailureCount(ip string) int {
	lat.mu.Lock()
	defer lat.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-lat.lockDuration)

	var count int
	for _, t := range lat.attempts[ip] {
		if t.After(windowStart) {
			count++
		}
	}
	return count
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

// Allow checks if a request from the given IP should be allowed
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	// Filter old requests
	var recent []time.Time
	for _, t := range rl.requests[ip] {
		if t.After(windowStart) {
			recent = append(recent, t)
		}
	}

	if len(recent) >= rl.limit {
		return false
	}

	rl.requests[ip] = append(recent, now)
	return true
}

// NewServer creates a new team server instance
func NewServer(config *ServerConfig, masterKey []byte) (*Server, error) {
	crypto, err := NewCrypto(masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create crypto: %w", err)
	}

	dbPath := filepath.Join(config.DataDir, "teamserver.db")
	storage, err := NewStorage(dbPath, crypto)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}

	// Build server URL for notifications
	protocol := "http"
	if config.TLS.Enabled {
		protocol = "https"
	}
	serverURL := fmt.Sprintf("%s://%s:%d", protocol, config.Host, config.Port)
	if config.TLS.Domain != "" {
		serverURL = fmt.Sprintf("%s://%s", protocol, config.TLS.Domain)
	}

	s := &Server{
		config:    config,
		storage:   storage,
		crypto:    crypto,
		deployer:  NewDeployer(),
		mfa:       NewMFAManager("MageBox"),
		notifier:  NewNotifier(config.Notifications.SMTP),
		mux:       http.NewServeMux(),
		masterKey: masterKey,
		serverURL: serverURL,
		logger:    log.New(os.Stdout, "[teamserver] ", log.LstdFlags),
	}

	// Load CA private key if CA is enabled
	if config.CA.Enabled {
		caPrivateKeyPEM, err := storage.GetCAPrivateKey()
		if err != nil {
			s.logger.Printf("Warning: CA enabled but failed to load CA private key: %v", err)
		} else {
			caPrivateKey, err := ParseCAPrivateKey(caPrivateKeyPEM)
			if err != nil {
				s.logger.Printf("Warning: Failed to parse CA private key: %v", err)
			} else {
				s.caPrivateKey = caPrivateKey
				s.logger.Println("SSH CA loaded successfully")
			}
		}
	}

	if config.Security.RateLimitEnabled {
		s.rateLimiter = NewRateLimiter(config.Security.RateLimitPerMinute, time.Minute)
	}

	// Initialize login attempt tracker
	maxAttempts := config.Security.LoginAttempts
	if maxAttempts == 0 {
		maxAttempts = 5
	}
	s.loginTracker = NewLoginAttemptTracker(maxAttempts)

	s.setupRoutes()

	return s, nil
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Health check
	s.mux.HandleFunc("/health", s.handleHealth)

	// Public endpoints
	s.mux.HandleFunc("/api/join", s.withMiddleware(s.handleJoin, false))

	// User endpoints (require authentication)
	s.mux.HandleFunc("/api/me", s.withMiddleware(s.handleMe, true))
	s.mux.HandleFunc("/api/environments", s.withMiddleware(s.handleUserEnvironments, true))
	s.mux.HandleFunc("/api/mfa/setup", s.withMiddleware(s.handleMFASetup, true))
	s.mux.HandleFunc("/api/mfa/verify", s.withMiddleware(s.handleMFAVerify, true))
	s.mux.HandleFunc("/api/cert/renew", s.withMiddleware(s.handleCertRenew, true))
	s.mux.HandleFunc("/api/cert/info", s.withMiddleware(s.handleCertInfo, true))

	// Admin endpoints (require admin authentication)
	s.mux.HandleFunc("/api/admin/users", s.withMiddleware(s.handleAdminUsers, true))
	s.mux.HandleFunc("/api/admin/users/", s.withMiddleware(s.handleAdminUserOrAccess, true))
	s.mux.HandleFunc("/api/admin/projects", s.withMiddleware(s.handleAdminProjects, true))
	s.mux.HandleFunc("/api/admin/projects/", s.withMiddleware(s.handleAdminProject, true))
	s.mux.HandleFunc("/api/admin/environments", s.withMiddleware(s.handleAdminEnvironments, true))
	s.mux.HandleFunc("/api/admin/environments/", s.withMiddleware(s.handleAdminEnvironment, true))
	s.mux.HandleFunc("/api/admin/audit", s.withMiddleware(s.handleAdminAudit, true))
	s.mux.HandleFunc("/api/admin/sync", s.withMiddleware(s.handleAdminSync, true))
	s.mux.HandleFunc("/api/admin/ca", s.withMiddleware(s.handleAdminCA, true))
}

// withMiddleware wraps a handler with common middleware
func (s *Server) withMiddleware(handler http.HandlerFunc, requireAuth bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set comprehensive security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Content-Type", "application/json")

		// HSTS - enforce HTTPS connections (1 year, include subdomains)
		if s.config.TLS.Enabled {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		// Content Security Policy - restrict resource loading (API server, very restrictive)
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")

		// Referrer Policy - don't leak referrer info
		w.Header().Set("Referrer-Policy", "no-referrer")

		// Permissions Policy - disable browser features not needed for API
		w.Header().Set("Permissions-Policy", "accelerometer=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=()")

		// Cache control for API responses
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, private")

		ip := s.getClientIP(r)

		// Rate limiting
		if s.rateLimiter != nil {
			if !s.rateLimiter.Allow(ip) {
				s.writeError(w, http.StatusTooManyRequests, "RATE_LIMITED", "Too many requests")
				return
			}
		}

		// Check if IP is locked due to too many failed login attempts
		if s.loginTracker != nil && s.loginTracker.IsLocked(ip) {
			s.writeError(w, http.StatusForbidden, "IP_LOCKED", "Too many failed login attempts. Please try again later.")
			return
		}

		// IP allowlist check
		if len(s.config.Security.AllowedIPs) > 0 {
			allowed := false
			for _, allowedIP := range s.config.Security.AllowedIPs {
				if ip == allowedIP || matchCIDR(ip, allowedIP) {
					allowed = true
					break
				}
			}
			if !allowed {
				s.writeError(w, http.StatusForbidden, "IP_NOT_ALLOWED", "IP address not in allowlist")
				return
			}
		}

		// Authentication
		if requireAuth {
			user, err := s.authenticateRequest(r)
			if err != nil {
				// Record failed login attempt
				if s.loginTracker != nil {
					locked := s.loginTracker.RecordFailure(ip)
					failCount := s.loginTracker.GetFailureCount(ip)
					s.logAudit(AuditAuthFailed, "", fmt.Sprintf("Authentication failed (attempt %d)", failCount), ip)

					// Alert on threshold breaches
					if failCount == 3 {
						s.logger.Printf("WARNING: Multiple failed login attempts from %s (3 failures)", ip)
					}
					if locked {
						s.logger.Printf("ALERT: IP %s locked out due to %d failed login attempts", ip, failCount)
						s.logAudit(AuditAuthFailed, "", fmt.Sprintf("IP locked out after %d failed attempts", failCount), ip)

						// Send security alert to admins (async)
						go s.sendSecurityAlertToAdmins("IP Lockout", ip, fmt.Sprintf("IP address %s has been locked out after %d failed login attempts", ip, failCount))
					}
				}
				s.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
				return
			}

			// Clear failed attempts on successful auth
			if s.loginTracker != nil {
				s.loginTracker.ClearAttempts(ip)
			}

			// Store user in context
			ctx := context.WithValue(r.Context(), contextKeyUser, user)
			r = r.WithContext(ctx)

			s.logAudit(AuditAuthSuccess, user.Name, "Successful authentication", ip)
		}

		handler(w, r)
	}
}

type contextKey string

const contextKeyUser contextKey = "user"

// requireAdminMFA checks if MFA is required for admin operations
func (s *Server) requireAdminMFA(user *User) error {
	if user == nil || user.Role != RoleAdmin {
		return nil // Only check for admin users
	}

	switch s.config.Security.AdminMFA {
	case "required":
		if !user.MFAEnabled {
			return fmt.Errorf("MFA is required for admin operations, please enable MFA first")
		}
	case "disabled":
		return nil
	default: // "optional" or empty
		return nil
	}

	return nil
}

// authenticateRequest validates the authorization header
func (s *Server) authenticateRequest(r *http.Request) (*User, error) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return nil, fmt.Errorf("missing authorization header")
	}

	if !strings.HasPrefix(auth, "Bearer ") {
		return nil, fmt.Errorf("invalid authorization format")
	}

	token := strings.TrimPrefix(auth, "Bearer ")

	// Check if it's the admin token
	if s.config.AdminTokenHash != "" && VerifyToken(token, s.config.AdminTokenHash) {
		return &User{
			Name: "admin",
			Role: RoleAdmin,
		}, nil
	}

	// Find user by verifying their token hash
	users, err := s.storage.ListUsers()
	if err != nil {
		return nil, fmt.Errorf("failed to lookup user")
	}

	for _, u := range users {
		if u.TokenHash != "" && VerifyToken(token, u.TokenHash) {
			// Check expiration
			if u.IsExpired() {
				return nil, fmt.Errorf("user access has expired")
			}

			// Update last access time
			_ = s.storage.UpdateUserLastAccess(u.Name)

			return &u, nil
		}
	}

	return nil, fmt.Errorf("invalid token")
}

// getCurrentUser extracts the authenticated user from context
func getCurrentUser(r *http.Request) *User {
	user, ok := r.Context().Value(contextKeyUser).(*User)
	if !ok {
		return nil
	}
	return user
}

// Health check handler
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "healthy",
		"version": "1.0.0",
	})
}

// handleJoin handles user join requests
func (s *Server) handleJoin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only POST is allowed")
		return
	}

	var req JoinRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if req.InviteToken == "" {
		s.writeError(w, http.StatusBadRequest, "MISSING_FIELDS", "invite_token is required")
		return
	}

	// Find and validate invite
	invites, err := s.findValidInvite(req.InviteToken)
	if err != nil {
		s.logAudit(AuditAuthFailed, "", "Invalid invite token", s.getClientIP(r))
		s.writeError(w, http.StatusUnauthorized, "INVALID_INVITE", err.Error())
		return
	}

	// Generate SSH key pair for the user
	keyComment := fmt.Sprintf("magebox-%s@%s", invites.UserName, r.Host)
	keyPair, err := GenerateSSHKeyPair(keyComment)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "KEY_GEN_ERROR", "Failed to generate SSH key pair")
		return
	}

	// Create the user
	sessionToken, err := GenerateSessionToken()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to generate session token")
		return
	}

	tokenHash, err := HashToken(sessionToken)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "HASH_ERROR", "Failed to hash token")
		return
	}

	// Calculate expiry
	var expiresAt *time.Time
	if s.config.Security.DefaultAccessDays > 0 {
		exp := time.Now().AddDate(0, 0, s.config.Security.DefaultAccessDays)
		expiresAt = &exp
	}

	user := &User{
		Name:      invites.UserName,
		Email:     invites.Email,
		Role:      invites.Role,
		PublicKey: keyPair.PublicKey,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		CreatedBy: "invite",
	}

	if err := s.storage.CreateUser(user); err != nil {
		s.writeError(w, http.StatusInternalServerError, "CREATE_ERROR", "Failed to create user")
		return
	}

	// Mark invite as used
	_ = s.storage.MarkInviteUsed(invites.ID)

	// Get accessible environments (based on user's projects)
	envs, _ := s.storage.ListEnvironmentsForUser(user.Name)

	// Convert to user-friendly format
	envsForUser := make([]EnvironmentForUser, len(envs))
	for i, e := range envs {
		envsForUser[i] = EnvironmentForUser{
			Name:       e.FullName(),
			Project:    e.Project,
			Host:       e.Host,
			Port:       e.GetPort(),
			DeployUser: e.DeployUser,
		}
	}

	s.logAudit(AuditUserJoin, user.Name, fmt.Sprintf("User joined: %s (SSH key generated)", user.Email), s.getClientIP(r))

	// Send welcome email (async, non-blocking)
	go func() {
		envNames := make([]string, len(envs))
		for i, e := range envs {
			envNames[i] = e.FullName()
		}
		if err := s.notifier.SendUserJoined(user.Email, user.Name, string(user.Role), envNames); err != nil {
			s.logger.Printf("Failed to send welcome email to %s: %v", user.Email, err)
		}
	}()

	// Deploy key to accessible environments (async, non-blocking)
	go s.deployUserKey(user)

	// Get server host for key storage naming
	serverHost := r.Host
	if serverHost == "" {
		serverHost = "teamserver"
	}

	// Prepare response
	response := JoinResponse{
		SessionToken: sessionToken,
		PrivateKey:   keyPair.PrivateKey,
		User:         user,
		Environments: envsForUser,
		ServerHost:   serverHost,
		CAEnabled:    s.config.CA.Enabled && s.caPrivateKey != nil,
	}

	// Sign certificate if CA is enabled
	if s.config.CA.Enabled && s.caPrivateKey != nil {
		certValidity := s.getCertValiditySeconds()
		principals := s.config.CA.DefaultPrincipals
		if len(principals) == 0 {
			principals = []string{"deploy"}
		}

		cert, err := SignSSHCertificate(s.caPrivateKey, keyPair.PublicKey, user.Email, principals, certValidity)
		if err != nil {
			s.logger.Printf("Warning: Failed to sign certificate for %s: %v", user.Name, err)
		} else {
			response.Certificate = cert.Certificate
			validUntil := time.Unix(int64(cert.ValidBefore), 0)
			response.ValidUntil = &validUntil
			response.Principals = principals
			s.logAudit(AuditCertIssue, user.Name, fmt.Sprintf("Certificate issued, valid until %s", validUntil.Format(time.RFC3339)), s.getClientIP(r))
		}

		// Include CA public key for reference
		caPublicKey, _ := s.storage.GetCAPublicKey()
		response.CAPublicKey = caPublicKey
	}

	_ = json.NewEncoder(w).Encode(response)
}

// deployUserKey deploys a user's public key to all accessible environments
func (s *Server) deployUserKey(user *User) {
	if user.PublicKey == "" {
		return
	}

	envs, err := s.storage.ListEnvironmentsForUser(user.Name)
	if err != nil {
		s.logger.Printf("Failed to list environments for key deployment: %v", err)
		return
	}

	for i := range envs {
		env := &envs[i]
		deployKey, err := s.crypto.Decrypt(env.DeployKey)
		if err != nil {
			s.logger.Printf("Failed to decrypt deploy key for %s/%s: %v", env.Project, env.Name, err)
			continue
		}

		userKey := UserKey{
			UserName:  user.Name,
			PublicKey: user.PublicKey,
		}

		if err := s.deployer.AddKey(env, string(deployKey), userKey); err != nil {
			s.logger.Printf("Failed to deploy key for %s to %s/%s: %v", user.Name, env.Project, env.Name, err)
			s.logAudit(AuditKeyDeployed, user.Name, fmt.Sprintf("Failed to deploy key to %s/%s: %v", env.Project, env.Name, err), "")
		} else {
			s.logger.Printf("Deployed key for %s to %s/%s", user.Name, env.Project, env.Name)
			s.logAudit(AuditKeyDeployed, user.Name, fmt.Sprintf("Deployed key to %s/%s", env.Project, env.Name), "")
		}
	}
}

// findValidInvite finds and validates an invite token
func (s *Server) findValidInvite(token string) (*Invite, error) {
	// Hash the token to compare
	// We need to iterate through invites and verify
	// This is secure but not optimal for large numbers of invites
	users, err := s.storage.ListUsers()
	if err != nil {
		return nil, fmt.Errorf("failed to check existing users")
	}

	// Check all invites (in production, you might want to index by partial hash)
	rows, err := s.storage.db.Query(`
		SELECT id, token_hash, user_name, email, role, projects, expires_at, used_at, created_at
		FROM invites WHERE used_at IS NULL`)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup invites")
	}
	defer rows.Close()

	for rows.Next() {
		var invite Invite
		var projectsStr string
		var usedAt *time.Time

		if err := rows.Scan(&invite.ID, &invite.TokenHash, &invite.UserName, &invite.Email,
			&invite.Role, &projectsStr, &invite.ExpiresAt, &usedAt, &invite.CreatedAt); err != nil {
			continue
		}

		if VerifyToken(token, invite.TokenHash) {
			if invite.IsExpired() {
				return nil, fmt.Errorf("invite has expired")
			}

			// Check if username already exists
			for _, u := range users {
				if u.Name == invite.UserName {
					return nil, fmt.Errorf("username already taken")
				}
			}

			if projectsStr != "" {
				invite.Projects = strings.Split(projectsStr, ",")
			}

			return &invite, nil
		}
	}

	return nil, fmt.Errorf("invalid or expired invite token")
}

// handleMe returns current user info
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only GET is allowed")
		return
	}

	user := getCurrentUser(r)
	if user == nil {
		s.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not found")
		return
	}

	_ = json.NewEncoder(w).Encode(user)
}

// handleUserEnvironments returns environments accessible by the current user
func (s *Server) handleUserEnvironments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only GET is allowed")
		return
	}

	user := getCurrentUser(r)
	if user == nil {
		s.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not found")
		return
	}

	envs, err := s.storage.ListEnvironmentsForUser(user.Name)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "LIST_ERROR", "Failed to list environments")
		return
	}

	// Convert to user-friendly format (hide sensitive deploy key info)
	envsForUser := make([]EnvironmentForUser, len(envs))
	for i, e := range envs {
		envsForUser[i] = EnvironmentForUser{
			Name:       e.FullName(),
			Project:    e.Project,
			Host:       e.Host,
			Port:       e.GetPort(),
			DeployUser: e.DeployUser,
		}
	}

	_ = json.NewEncoder(w).Encode(envsForUser)
}

// MFASetupRequest is the request to initiate MFA setup
type MFASetupRequest struct {
	// Empty - just initiates setup
}

// MFASetupConfirmRequest confirms MFA setup with a code
type MFASetupConfirmRequest struct {
	Code string `json:"code"`
}

// MFAVerifyRequest is the request to verify an MFA code
type MFAVerifyRequest struct {
	Code string `json:"code"`
}

// handleMFASetup handles MFA setup
func (s *Server) handleMFASetup(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil {
		s.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Check if MFA is already enabled
		if user.MFAEnabled {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"mfa_enabled": true,
				"message":     "MFA is already enabled",
			})
			return
		}

		// Generate setup data
		setup, err := s.mfa.GenerateSetup(user.Email)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "MFA_ERROR", "Failed to generate MFA setup")
			return
		}

		// Store the secret temporarily (encrypted) - will be confirmed with POST
		encryptedSecret, err := s.crypto.Encrypt([]byte(setup.Secret))
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "ENCRYPT_ERROR", "Failed to encrypt secret")
			return
		}

		// Save pending secret to user
		user.MFASecret = string(encryptedSecret)
		if err := s.storage.UpdateUser(user); err != nil {
			s.writeError(w, http.StatusInternalServerError, "UPDATE_ERROR", "Failed to save MFA setup")
			return
		}

		_ = json.NewEncoder(w).Encode(setup)

	case http.MethodPost:
		// Confirm MFA setup with a code
		var req MFASetupConfirmRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
			return
		}

		if req.Code == "" {
			s.writeError(w, http.StatusBadRequest, "MISSING_CODE", "MFA code is required")
			return
		}

		// Get the pending secret
		if user.MFASecret == "" {
			s.writeError(w, http.StatusBadRequest, "NO_SETUP", "MFA setup not initiated. Use GET first.")
			return
		}

		// Decrypt the secret
		decryptedSecret, err := s.crypto.Decrypt(user.MFASecret)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "DECRYPT_ERROR", "Failed to decrypt secret")
			return
		}

		// Validate the code with replay protection
		if !s.mfa.ValidateCodeForUser(user.Name, string(decryptedSecret), req.Code) {
			s.logAudit(AuditMFASetup, user.Name, "MFA setup failed - invalid code", s.getClientIP(r))
			s.writeError(w, http.StatusUnauthorized, "INVALID_CODE", "Invalid MFA code")
			return
		}

		// Enable MFA
		user.MFAEnabled = true
		if err := s.storage.UpdateUser(user); err != nil {
			s.writeError(w, http.StatusInternalServerError, "UPDATE_ERROR", "Failed to enable MFA")
			return
		}

		// Generate recovery codes
		recoveryCodes, err := s.mfa.GenerateRecoveryCodes(10)
		if err != nil {
			s.logger.Printf("Failed to generate recovery codes: %v", err)
		}

		s.logAudit(AuditMFASetup, user.Name, "MFA enabled successfully", s.getClientIP(r))

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success":        true,
			"message":        "MFA enabled successfully",
			"recovery_codes": recoveryCodes,
		})

	case http.MethodDelete:
		// Disable MFA
		if !user.MFAEnabled {
			s.writeError(w, http.StatusBadRequest, "NOT_ENABLED", "MFA is not enabled")
			return
		}

		user.MFAEnabled = false
		user.MFASecret = ""
		if err := s.storage.UpdateUser(user); err != nil {
			s.writeError(w, http.StatusInternalServerError, "UPDATE_ERROR", "Failed to disable MFA")
			return
		}

		s.logAudit(AuditMFASetup, user.Name, "MFA disabled", s.getClientIP(r))

		_ = json.NewEncoder(w).Encode(SuccessResponse{
			Success: true,
			Message: "MFA disabled",
		})

	default:
		s.writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only GET, POST, and DELETE are allowed")
	}
}

// handleMFAVerify handles MFA verification
func (s *Server) handleMFAVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only POST is allowed")
		return
	}

	user := getCurrentUser(r)
	if user == nil {
		s.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	if !user.MFAEnabled {
		s.writeError(w, http.StatusBadRequest, "MFA_NOT_ENABLED", "MFA is not enabled for this user")
		return
	}

	var req MFAVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if req.Code == "" {
		s.writeError(w, http.StatusBadRequest, "MISSING_CODE", "MFA code is required")
		return
	}

	// Decrypt the secret
	decryptedSecret, err := s.crypto.Decrypt(user.MFASecret)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "DECRYPT_ERROR", "Failed to decrypt secret")
		return
	}

	// Validate the code with replay protection
	if !s.mfa.ValidateCodeForUser(user.Name, string(decryptedSecret), req.Code) {
		s.logAudit(AuditMFAVerify, user.Name, "MFA verification failed", s.getClientIP(r))
		s.writeError(w, http.StatusUnauthorized, "INVALID_CODE", "Invalid MFA code")
		return
	}

	s.logAudit(AuditMFAVerify, user.Name, "MFA verification successful", s.getClientIP(r))

	_ = json.NewEncoder(w).Encode(SuccessResponse{
		Success: true,
		Message: "MFA verification successful",
	})
}

// Admin handlers

// handleAdminUsers handles user listing and creation
func (s *Server) handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil || !user.Role.CanManageUsers() {
		s.writeError(w, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	// Check MFA requirement for admin operations
	if err := s.requireAdminMFA(user); err != nil {
		s.writeError(w, http.StatusForbidden, "MFA_REQUIRED", err.Error())
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.listUsers(w, r)
	case http.MethodPost:
		s.createUser(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only GET and POST are allowed")
	}
}

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.storage.ListUsers()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "LIST_ERROR", "Failed to list users")
		return
	}

	_ = json.NewEncoder(w).Encode(users)
}

func (s *Server) createUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if req.Name == "" || req.Email == "" || !req.Role.IsValid() {
		s.writeError(w, http.StatusBadRequest, "MISSING_FIELDS", "name, email, and valid role are required")
		return
	}

	// Generate invite token
	inviteToken, err := GenerateInviteToken()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to generate invite token")
		return
	}

	tokenHash, err := HashToken(inviteToken)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "HASH_ERROR", "Failed to hash token")
		return
	}

	// Parse invite expiry
	expiryDuration, _ := time.ParseDuration(s.config.Security.InviteExpiry)
	if expiryDuration == 0 {
		expiryDuration = 48 * time.Hour
	}

	invite := &Invite{
		TokenHash: tokenHash,
		UserName:  req.Name,
		Email:     req.Email,
		Role:      req.Role,
		Projects:  req.Projects,
		ExpiresAt: time.Now().Add(expiryDuration),
	}

	if err := s.storage.CreateInvite(invite); err != nil {
		s.writeError(w, http.StatusInternalServerError, "CREATE_ERROR", "Failed to create invite")
		return
	}

	admin := getCurrentUser(r)
	s.logAudit(AuditUserCreate, admin.Name, fmt.Sprintf("Created invite for: %s (%s)", req.Name, req.Email), s.getClientIP(r))

	// Send invitation email (async, non-blocking)
	go func() {
		if err := s.notifier.SendUserInvited(req.Email, req.Name, string(req.Role), s.serverURL, inviteToken, invite.ExpiresAt); err != nil {
			s.logger.Printf("Failed to send invitation email to %s: %v", req.Email, err)
		}
	}()

	_ = json.NewEncoder(w).Encode(CreateUserResponse{
		User: &User{
			Name:  req.Name,
			Email: req.Email,
			Role:  req.Role,
		},
		InviteToken: inviteToken,
	})
}

// handleAdminUserOrAccess routes to user or access operations based on path
func (s *Server) handleAdminUserOrAccess(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil || !user.Role.CanManageUsers() {
		s.writeError(w, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	// Check MFA requirement for admin operations
	if err := s.requireAdminMFA(user); err != nil {
		s.writeError(w, http.StatusForbidden, "MFA_REQUIRED", err.Error())
		return
	}

	// Extract path after /api/admin/users/
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/users/")
	if path == "" {
		s.writeError(w, http.StatusBadRequest, "MISSING_NAME", "Username is required")
		return
	}

	// Check if this is an access operation (path ends with /access)
	if strings.HasSuffix(path, "/access") {
		userName := strings.TrimSuffix(path, "/access")
		s.handleUserAccess(w, r, userName)
		return
	}

	// Regular user operation
	s.handleAdminUser(w, r, path)
}

// handleAdminUser handles individual user operations
func (s *Server) handleAdminUser(w http.ResponseWriter, r *http.Request, name string) {
	switch r.Method {
	case http.MethodGet:
		s.getUser(w, r, name)
	case http.MethodPut:
		s.updateUser(w, r, name)
	case http.MethodDelete:
		s.deleteUser(w, r, name)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only GET, PUT, and DELETE are allowed")
	}
}

// handleUserAccess handles granting/revoking project access
func (s *Server) handleUserAccess(w http.ResponseWriter, r *http.Request, userName string) {
	// Verify user exists
	_, err := s.storage.GetUser(userName)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "NOT_FOUND", "User not found")
		return
	}

	switch r.Method {
	case http.MethodPost:
		s.grantUserAccess(w, r, userName)
	case http.MethodDelete:
		s.revokeUserAccess(w, r, userName)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only POST and DELETE are allowed")
	}
}

func (s *Server) grantUserAccess(w http.ResponseWriter, r *http.Request, userName string) {
	var req GrantAccessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if req.Project == "" {
		s.writeError(w, http.StatusBadRequest, "MISSING_FIELDS", "project is required")
		return
	}

	// Verify project exists
	_, err := s.storage.GetProject(req.Project)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "INVALID_PROJECT", "Project does not exist")
		return
	}

	admin := getCurrentUser(r)
	if err := s.storage.GrantProjectAccess(userName, req.Project, admin.Name); err != nil {
		s.writeError(w, http.StatusInternalServerError, "GRANT_ERROR", "Failed to grant access")
		return
	}

	s.logAudit(AuditAdminAction, admin.Name, fmt.Sprintf("Granted project access: %s -> %s", userName, req.Project), s.getClientIP(r))

	// Get updated user with projects
	user, _ := s.storage.GetUser(userName)

	_ = json.NewEncoder(w).Encode(SuccessResponse{
		Success: true,
		Message: fmt.Sprintf("Granted %s access to project %s", userName, req.Project),
	})

	// Trigger key sync for the user's new project access (async)
	if user != nil && user.PublicKey != "" {
		go s.deployUserKey(user)
	}
}

func (s *Server) revokeUserAccess(w http.ResponseWriter, r *http.Request, userName string) {
	var req RevokeAccessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if req.Project == "" {
		s.writeError(w, http.StatusBadRequest, "MISSING_FIELDS", "project is required")
		return
	}

	admin := getCurrentUser(r)
	if err := s.storage.RevokeProjectAccess(userName, req.Project); err != nil {
		s.writeError(w, http.StatusInternalServerError, "REVOKE_ERROR", "Failed to revoke access")
		return
	}

	s.logAudit(AuditAdminAction, admin.Name, fmt.Sprintf("Revoked project access: %s -> %s", userName, req.Project), s.getClientIP(r))

	_ = json.NewEncoder(w).Encode(SuccessResponse{
		Success: true,
		Message: fmt.Sprintf("Revoked %s access to project %s", userName, req.Project),
	})

	// TODO: Trigger key removal from the project's environments (async)
}

func (s *Server) getUser(w http.ResponseWriter, r *http.Request, name string) {
	user, err := s.storage.GetUser(name)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "NOT_FOUND", "User not found")
		return
	}

	_ = json.NewEncoder(w).Encode(user)
}

func (s *Server) updateUser(w http.ResponseWriter, r *http.Request, name string) {
	user, err := s.storage.GetUser(name)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "NOT_FOUND", "User not found")
		return
	}

	var updates struct {
		Email      string `json:"email,omitempty"`
		Role       Role   `json:"role,omitempty"`
		ExpiryDays int    `json:"expiry_days,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		s.writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if updates.Email != "" {
		user.Email = updates.Email
	}
	if updates.Role != "" && updates.Role.IsValid() {
		user.Role = updates.Role
	}
	if updates.ExpiryDays > 0 {
		exp := time.Now().AddDate(0, 0, updates.ExpiryDays)
		user.ExpiresAt = &exp
	}

	if err := s.storage.UpdateUser(user); err != nil {
		s.writeError(w, http.StatusInternalServerError, "UPDATE_ERROR", "Failed to update user")
		return
	}

	admin := getCurrentUser(r)
	s.logAudit(AuditUserUpdate, admin.Name, fmt.Sprintf("Updated user: %s", name), s.getClientIP(r))

	_ = json.NewEncoder(w).Encode(user)
}

func (s *Server) deleteUser(w http.ResponseWriter, r *http.Request, name string) {
	// Get user first to know their role for key removal
	user, err := s.storage.GetUser(name)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "NOT_FOUND", "User not found")
		return
	}

	// Remove keys from all environments (async)
	go s.removeUserKeys(user)

	// Delete from database
	if err := s.storage.DeleteUser(name); err != nil {
		s.writeError(w, http.StatusInternalServerError, "DELETE_ERROR", "Failed to delete user")
		return
	}

	admin := getCurrentUser(r)
	s.logAudit(AuditUserRemove, admin.Name, fmt.Sprintf("Removed user: %s", name), s.getClientIP(r))

	// Send access revoked email (async, non-blocking)
	go func() {
		if err := s.notifier.SendUserRemoved(user.Email, user.Name); err != nil {
			s.logger.Printf("Failed to send access revoked email to %s: %v", user.Email, err)
		}
	}()

	_ = json.NewEncoder(w).Encode(SuccessResponse{
		Success: true,
		Message: fmt.Sprintf("User %s removed", name),
	})
}

// removeUserKeys removes a user's public key from all environments
func (s *Server) removeUserKeys(user *User) {
	// Get all environments (we need to check all since roles may have changed)
	envs, err := s.storage.ListEnvironments()
	if err != nil {
		s.logger.Printf("Failed to list environments for key removal: %v", err)
		return
	}

	for i := range envs {
		env := &envs[i]
		deployKey, err := s.crypto.Decrypt(env.DeployKey)
		if err != nil {
			s.logger.Printf("Failed to decrypt deploy key for %s: %v", env.Name, err)
			continue
		}

		if err := s.deployer.RemoveKey(env, string(deployKey), user.Name); err != nil {
			s.logger.Printf("Failed to remove key for %s from %s: %v", user.Name, env.Name, err)
		} else {
			s.logger.Printf("Removed key for %s from %s", user.Name, env.Name)
			s.logAudit(AuditKeyRemoved, user.Name, fmt.Sprintf("Removed key from %s", env.Name), "")
		}
	}
}

// handleAdminProjects handles project listing and creation
func (s *Server) handleAdminProjects(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil || !user.Role.CanManageProjects() {
		s.writeError(w, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	// Check MFA requirement for admin operations
	if err := s.requireAdminMFA(user); err != nil {
		s.writeError(w, http.StatusForbidden, "MFA_REQUIRED", err.Error())
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.listProjects(w, r)
	case http.MethodPost:
		s.createProject(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only GET and POST are allowed")
	}
}

func (s *Server) listProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := s.storage.ListProjects()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "LIST_ERROR", "Failed to list projects")
		return
	}

	_ = json.NewEncoder(w).Encode(projects)
}

func (s *Server) createProject(w http.ResponseWriter, r *http.Request) {
	var req CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if req.Name == "" {
		s.writeError(w, http.StatusBadRequest, "MISSING_FIELDS", "name is required")
		return
	}

	admin := getCurrentUser(r)
	project := &Project{
		Name:        req.Name,
		Description: req.Description,
		CreatedBy:   admin.Name,
	}

	if err := s.storage.CreateProject(project); err != nil {
		s.writeError(w, http.StatusInternalServerError, "CREATE_ERROR", "Failed to create project")
		return
	}

	s.logAudit(AuditAdminAction, admin.Name, fmt.Sprintf("Created project: %s", req.Name), s.getClientIP(r))

	_ = json.NewEncoder(w).Encode(project)
}

// handleAdminProject handles individual project operations
func (s *Server) handleAdminProject(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil || !user.Role.CanManageProjects() {
		s.writeError(w, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	// Check MFA requirement for admin operations
	if err := s.requireAdminMFA(user); err != nil {
		s.writeError(w, http.StatusForbidden, "MFA_REQUIRED", err.Error())
		return
	}

	name := strings.TrimPrefix(r.URL.Path, "/api/admin/projects/")
	if name == "" {
		s.writeError(w, http.StatusBadRequest, "MISSING_NAME", "Project name is required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.getProject(w, r, name)
	case http.MethodDelete:
		s.deleteProject(w, r, name)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only GET and DELETE are allowed")
	}
}

func (s *Server) getProject(w http.ResponseWriter, r *http.Request, name string) {
	project, err := s.storage.GetProject(name)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Project not found")
		return
	}

	_ = json.NewEncoder(w).Encode(project)
}

func (s *Server) deleteProject(w http.ResponseWriter, r *http.Request, name string) {
	if err := s.storage.DeleteProject(name); err != nil {
		s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Project not found")
		return
	}

	admin := getCurrentUser(r)
	s.logAudit(AuditAdminAction, admin.Name, fmt.Sprintf("Deleted project: %s", name), s.getClientIP(r))

	_ = json.NewEncoder(w).Encode(SuccessResponse{
		Success: true,
		Message: fmt.Sprintf("Project %s removed", name),
	})
}

// handleAdminEnvironments handles environment listing and creation
func (s *Server) handleAdminEnvironments(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil || !user.Role.CanManageEnvironments() {
		s.writeError(w, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	// Check MFA requirement for admin operations
	if err := s.requireAdminMFA(user); err != nil {
		s.writeError(w, http.StatusForbidden, "MFA_REQUIRED", err.Error())
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.listEnvironments(w, r)
	case http.MethodPost:
		s.createEnvironment(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only GET and POST are allowed")
	}
}

func (s *Server) listEnvironments(w http.ResponseWriter, r *http.Request) {
	envs, err := s.storage.ListEnvironments()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "LIST_ERROR", "Failed to list environments")
		return
	}

	_ = json.NewEncoder(w).Encode(envs)
}

func (s *Server) createEnvironment(w http.ResponseWriter, r *http.Request) {
	var req CreateEnvironmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if req.Name == "" || req.Project == "" || req.Host == "" || req.DeployUser == "" || req.DeployKey == "" {
		s.writeError(w, http.StatusBadRequest, "MISSING_FIELDS", "name, project, host, deploy_user, and deploy_key are required")
		return
	}

	// Verify project exists
	_, err := s.storage.GetProject(req.Project)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "INVALID_PROJECT", "Project does not exist")
		return
	}

	port := req.Port
	if port == 0 {
		port = 22
	}

	env := &Environment{
		Name:       req.Name,
		Project:    req.Project,
		Host:       req.Host,
		Port:       port,
		DeployUser: req.DeployUser,
		DeployKey:  req.DeployKey,
	}

	if err := s.storage.CreateEnvironment(env); err != nil {
		s.writeError(w, http.StatusInternalServerError, "CREATE_ERROR", "Failed to create environment")
		return
	}

	admin := getCurrentUser(r)
	s.logAudit(AuditEnvCreate, admin.Name, fmt.Sprintf("Created environment: %s/%s (%s)", req.Project, req.Name, req.Host), s.getClientIP(r))

	// Don't return deploy key in response
	env.DeployKey = ""
	_ = json.NewEncoder(w).Encode(env)
}

// handleAdminEnvironment handles individual environment operations
// URL format: /api/admin/environments/{project}/{name}
func (s *Server) handleAdminEnvironment(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil || !user.Role.CanManageEnvironments() {
		s.writeError(w, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	// Check MFA requirement for admin operations
	if err := s.requireAdminMFA(user); err != nil {
		s.writeError(w, http.StatusForbidden, "MFA_REQUIRED", err.Error())
		return
	}

	// Parse project/name from path
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/environments/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		s.writeError(w, http.StatusBadRequest, "INVALID_PATH", "Path must be /api/admin/environments/{project}/{name}")
		return
	}
	project, name := parts[0], parts[1]

	switch r.Method {
	case http.MethodGet:
		s.getEnvironment(w, r, project, name)
	case http.MethodDelete:
		s.deleteEnvironment(w, r, project, name)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only GET and DELETE are allowed")
	}
}

func (s *Server) getEnvironment(w http.ResponseWriter, r *http.Request, project, name string) {
	env, err := s.storage.GetEnvironment(project, name)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Environment not found")
		return
	}

	// Don't expose deploy key
	env.DeployKey = ""
	_ = json.NewEncoder(w).Encode(env)
}

func (s *Server) deleteEnvironment(w http.ResponseWriter, r *http.Request, project, name string) {
	if err := s.storage.DeleteEnvironment(project, name); err != nil {
		s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Environment not found")
		return
	}

	admin := getCurrentUser(r)
	s.logAudit(AuditEnvRemove, admin.Name, fmt.Sprintf("Removed environment: %s/%s", project, name), s.getClientIP(r))

	_ = json.NewEncoder(w).Encode(SuccessResponse{
		Success: true,
		Message: fmt.Sprintf("Environment %s/%s removed", project, name),
	})
}

// handleAdminAudit returns audit log entries
func (s *Server) handleAdminAudit(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil || user.Role != RoleAdmin {
		s.writeError(w, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	// Check MFA requirement for admin operations
	if err := s.requireAdminMFA(user); err != nil {
		s.writeError(w, http.StatusForbidden, "MFA_REQUIRED", err.Error())
		return
	}

	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only GET is allowed")
		return
	}

	// Parse query parameters
	var from, to *time.Time
	userName := r.URL.Query().Get("user")
	action := AuditAction(r.URL.Query().Get("action"))
	limit := 100 // Default limit

	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			from = &t
		}
	}
	if toStr := r.URL.Query().Get("to"); toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			to = &t
		}
	}

	entries, err := s.storage.ListAuditEntries(from, to, userName, action, limit)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "LIST_ERROR", "Failed to list audit entries")
		return
	}

	_ = json.NewEncoder(w).Encode(entries)
}

// SyncRequest is the request body for key sync
type SyncRequest struct {
	Environment string `json:"environment,omitempty"`
}

// SyncResponse is the response for key sync
type SyncResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Results []SyncEnvResult `json:"results,omitempty"`
}

// SyncEnvResult is the result for a single environment sync
type SyncEnvResult struct {
	Environment string `json:"environment"`
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	KeysAdded   int    `json:"keys_added"`
	KeysRemoved int    `json:"keys_removed"`
	Error       string `json:"error,omitempty"`
}

// handleAdminSync triggers key synchronization
func (s *Server) handleAdminSync(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil || user.Role != RoleAdmin {
		s.writeError(w, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	// Check MFA requirement for admin operations
	if err := s.requireAdminMFA(user); err != nil {
		s.writeError(w, http.StatusForbidden, "MFA_REQUIRED", err.Error())
		return
	}

	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only POST is allowed")
		return
	}

	var req SyncRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // Optional body

	results, err := s.syncKeys(req.Environment)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "SYNC_ERROR", err.Error())
		return
	}

	// Count successes
	successCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		}
	}

	s.logAudit(AuditKeySync, user.Name, fmt.Sprintf("Synced keys to %d/%d environments", successCount, len(results)), s.getClientIP(r))

	_ = json.NewEncoder(w).Encode(SyncResponse{
		Success: successCount == len(results),
		Message: fmt.Sprintf("Synced %d/%d environments successfully", successCount, len(results)),
		Results: results,
	})
}

// syncKeys synchronizes SSH keys to environments
// envPath can be empty (sync all), "project" (sync all in project), or "project/name" (sync specific)
func (s *Server) syncKeys(envPath string) ([]SyncEnvResult, error) {
	var envs []Environment
	var err error

	if envPath != "" {
		parts := strings.SplitN(envPath, "/", 2)
		if len(parts) == 2 {
			// Sync specific environment: project/name
			env, err := s.storage.GetEnvironment(parts[0], parts[1])
			if err != nil {
				return nil, fmt.Errorf("environment not found: %s", envPath)
			}
			envs = []Environment{*env}
		} else {
			// Sync all environments in a project
			envs, err = s.storage.ListEnvironmentsByProject(parts[0])
			if err != nil {
				return nil, fmt.Errorf("failed to list environments for project: %w", err)
			}
		}
	} else {
		// Sync all environments
		envs, err = s.storage.ListEnvironments()
		if err != nil {
			return nil, fmt.Errorf("failed to list environments: %w", err)
		}
	}

	if len(envs) == 0 {
		return []SyncEnvResult{}, nil
	}

	// Get all users
	users, err := s.storage.ListUsers()
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	var results []SyncEnvResult

	for i := range envs {
		env := &envs[i]
		result := SyncEnvResult{
			Environment: env.FullName(),
		}

		// Get deploy key (decrypt it)
		deployKey, err := s.crypto.Decrypt(env.DeployKey)
		if err != nil {
			result.Error = fmt.Sprintf("failed to decrypt deploy key: %v", err)
			results = append(results, result)
			continue
		}

		// Build list of authorized keys for this environment
		var authorizedKeys []UserKey
		for _, u := range users {
			// Check if user has access to this environment's project
			if u.PublicKey != "" && u.HasProjectAccess(env.Project) {
				// Check if user hasn't expired
				if u.ExpiresAt == nil || time.Now().Before(*u.ExpiresAt) {
					authorizedKeys = append(authorizedKeys, UserKey{
						UserName:  u.Name,
						PublicKey: u.PublicKey,
					})
				}
			}
		}

		// Deploy keys
		deployResult, err := s.deployer.SyncEnvironment(env, string(deployKey), authorizedKeys)
		if err != nil {
			result.Error = err.Error()
			s.logger.Printf("Failed to sync %s: %v", env.FullName(), err)
		} else {
			result.Success = true
			result.Message = deployResult.Message
			result.KeysAdded = deployResult.KeysAdded
			result.KeysRemoved = deployResult.KeysRemoved
			s.logger.Printf("Synced %s: %s", env.FullName(), deployResult.Message)
		}

		results = append(results, result)
	}

	return results, nil
}

// logAudit creates an audit log entry
func (s *Server) logAudit(action AuditAction, userName, details, ip string) {
	entry := &AuditEntry{
		UserName:  userName,
		Action:    action,
		Details:   details,
		IPAddress: ip,
	}

	if err := s.storage.CreateAuditEntry(entry); err != nil {
		s.logger.Printf("Failed to create audit entry: %v", err)
	}
}

// sendSecurityAlertToAdmins sends a security alert email to all admins
func (s *Server) sendSecurityAlertToAdmins(alertType, ip, details string) {
	if !s.notifier.IsEnabled() {
		return
	}

	users, err := s.storage.ListUsers()
	if err != nil {
		s.logger.Printf("Failed to list users for security alert: %v", err)
		return
	}

	adminEmails := GetAdminEmails(users)
	for _, email := range adminEmails {
		if err := s.notifier.SendSecurityAlert(email, alertType, ip, details); err != nil {
			s.logger.Printf("Failed to send security alert to %s: %v", email, err)
		}
	}
}

// writeError writes a JSON error response
func (s *Server) writeError(w http.ResponseWriter, status int, code, message string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Error: message,
		Code:  code,
	})
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if s.config.TLS.Enabled {
		// Configure TLS
		tlsConfig := &tls.Config{
			MinVersion:               tls.VersionTLS12,
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			},
		}
		s.httpServer.TLSConfig = tlsConfig

		s.logger.Printf("Starting HTTPS server on %s", addr)
		return s.httpServer.ListenAndServeTLS(s.config.TLS.CertFile, s.config.TLS.KeyFile)
	}

	s.logger.Printf("Starting HTTP server on %s (TLS disabled)", addr)
	return s.httpServer.ListenAndServe()
}

// Stop gracefully stops the server
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Println("Shutting down server...")

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return err
	}

	if err := s.storage.Close(); err != nil {
		return err
	}

	s.logger.Println("Server stopped")
	return nil
}

// GetStorage returns the storage instance (for CLI commands)
func (s *Server) GetStorage() *Storage {
	return s.storage
}

// GetCrypto returns the crypto instance
func (s *Server) GetCrypto() *Crypto {
	return s.crypto
}

// getCertValiditySeconds returns the certificate validity duration in seconds
func (s *Server) getCertValiditySeconds() int64 {
	validity := s.config.CA.CertValidity
	if validity == "" {
		validity = "24h"
	}

	duration, err := time.ParseDuration(validity)
	if err != nil {
		// Default to 24 hours
		return 24 * 60 * 60
	}

	return int64(duration.Seconds())
}

// handleCertRenew handles certificate renewal requests
func (s *Server) handleCertRenew(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only POST is allowed")
		return
	}

	user := getCurrentUser(r)
	if user == nil {
		s.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	// Check if CA is enabled
	if !s.config.CA.Enabled || s.caPrivateKey == nil {
		s.writeError(w, http.StatusNotImplemented, "CA_DISABLED", "SSH CA is not enabled on this server")
		return
	}

	// Check if user still has access to any projects
	if len(user.Projects) == 0 {
		s.logAudit(AuditCertDeny, user.Name, "Certificate renewal denied: no project access", s.getClientIP(r))
		s.writeError(w, http.StatusForbidden, "NO_ACCESS", "No project access. Cannot renew certificate.")
		return
	}

	// Check if user is expired
	if user.IsExpired() {
		s.logAudit(AuditCertDeny, user.Name, "Certificate renewal denied: user expired", s.getClientIP(r))
		s.writeError(w, http.StatusForbidden, "USER_EXPIRED", "User access has expired. Cannot renew certificate.")
		return
	}

	// Check if user has a public key
	if user.PublicKey == "" {
		s.writeError(w, http.StatusBadRequest, "NO_KEY", "User has no public key. Re-join to generate new key pair.")
		return
	}

	// Sign new certificate
	certValidity := s.getCertValiditySeconds()
	principals := s.config.CA.DefaultPrincipals
	if len(principals) == 0 {
		principals = []string{"deploy"}
	}

	cert, err := SignSSHCertificate(s.caPrivateKey, user.PublicKey, user.Email, principals, certValidity)
	if err != nil {
		s.logger.Printf("Failed to sign certificate for %s: %v", user.Name, err)
		s.writeError(w, http.StatusInternalServerError, "SIGN_ERROR", "Failed to sign certificate")
		return
	}

	validUntil := time.Unix(int64(cert.ValidBefore), 0)
	s.logAudit(AuditCertRenew, user.Name, fmt.Sprintf("Certificate renewed, valid until %s", validUntil.Format(time.RFC3339)), s.getClientIP(r))

	_ = json.NewEncoder(w).Encode(CertRenewResponse{
		Certificate: cert.Certificate,
		ValidUntil:  validUntil,
		Principals:  principals,
		Serial:      cert.Serial,
	})
}

// handleCertInfo returns certificate information for the current user
func (s *Server) handleCertInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only GET is allowed")
		return
	}

	user := getCurrentUser(r)
	if user == nil {
		s.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	// Check if CA is enabled
	if !s.config.CA.Enabled || s.caPrivateKey == nil {
		_ = json.NewEncoder(w).Encode(CertInfoResponse{
			HasCertificate: false,
			IsExpired:      true,
		})
		return
	}

	// Return CA info and whether user can get a certificate
	canGetCert := user.PublicKey != "" && len(user.Projects) > 0 && !user.IsExpired()

	_ = json.NewEncoder(w).Encode(CertInfoResponse{
		HasCertificate: canGetCert,
		IsExpired:      !canGetCert,
		Principals:     s.config.CA.DefaultPrincipals,
		KeyID:          user.Email,
	})
}

// handleAdminCA returns CA information for admins
func (s *Server) handleAdminCA(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil || user.Role != RoleAdmin {
		s.writeError(w, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only GET is allowed")
		return
	}

	if !s.config.CA.Enabled {
		_ = json.NewEncoder(w).Encode(CAInfoResponse{
			Enabled: false,
		})
		return
	}

	caPublicKey, err := s.storage.GetCAPublicKey()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "CA_ERROR", "Failed to get CA public key")
		return
	}

	// Get fingerprint
	_, fingerprint, _ := ParseSSHPublicKey(caPublicKey)

	_ = json.NewEncoder(w).Encode(CAInfoResponse{
		Enabled:      true,
		PublicKey:    caPublicKey,
		CertValidity: s.config.CA.CertValidity,
		Principals:   s.config.CA.DefaultPrincipals,
		Fingerprint:  fingerprint,
	})
}

// Helper functions

// getClientIP extracts the real client IP address from the request.
// If trusted proxies are configured and the request comes from a trusted proxy,
// it will parse X-Forwarded-For or X-Real-IP headers.
// Otherwise, it returns the direct connection IP (RemoteAddr).
func (s *Server) getClientIP(r *http.Request) string {
	// Get the direct connection IP
	directIP, _, _ := net.SplitHostPort(r.RemoteAddr)
	if directIP == "" {
		directIP = r.RemoteAddr
	}

	// Only trust proxy headers if the direct connection is from a trusted proxy
	trustedProxies := s.config.Security.TrustedProxies
	if len(trustedProxies) == 0 {
		// No trusted proxies configured - always use direct IP
		return directIP
	}

	// Check if the direct connection is from a trusted proxy
	isTrustedProxy := false
	for _, proxy := range trustedProxies {
		if matchCIDR(directIP, proxy) {
			isTrustedProxy = true
			break
		}
	}

	if !isTrustedProxy {
		// Connection is not from a trusted proxy - ignore proxy headers
		return directIP
	}

	// Connection is from a trusted proxy - parse X-Forwarded-For
	// Take the rightmost IP that is NOT a trusted proxy (the actual client)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		// Walk backwards through the IPs to find the first non-proxy IP
		for i := len(ips) - 1; i >= 0; i-- {
			ip := strings.TrimSpace(ips[i])
			if ip == "" {
				continue
			}
			// Check if this IP is also a trusted proxy
			isProxy := false
			for _, proxy := range trustedProxies {
				if matchCIDR(ip, proxy) {
					isProxy = true
					break
				}
			}
			if !isProxy {
				return ip
			}
		}
	}

	// Check X-Real-IP as fallback
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to direct IP
	return directIP
}

func matchCIDR(ip, cidr string) bool {
	if !strings.Contains(cidr, "/") {
		return ip == cidr
	}

	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	return network.Contains(parsedIP)
}

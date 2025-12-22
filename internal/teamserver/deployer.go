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
	"encoding/base64"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// validUsernameRegex matches valid usernames (alphanumeric, underscore, hyphen, dot)
var validUsernameRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// validSSHKeyTypes lists valid SSH key type prefixes
var validSSHKeyTypes = []string{
	"ssh-rsa",
	"ssh-ed25519",
	"ecdsa-sha2-nistp256",
	"ecdsa-sha2-nistp384",
	"ecdsa-sha2-nistp521",
	"sk-ssh-ed25519@openssh.com",
	"sk-ecdsa-sha2-nistp256@openssh.com",
}

// Deployer handles SSH key deployment to remote environments
type Deployer struct {
	timeout time.Duration
}

// NewDeployer creates a new SSH key deployer
func NewDeployer() *Deployer {
	return &Deployer{
		timeout: 30 * time.Second,
	}
}

// DeployResult contains the result of a deployment operation
type DeployResult struct {
	Environment string
	Success     bool
	Message     string
	KeysAdded   int
	KeysRemoved int
	Error       error
}

// HostKeyMismatchError is returned when SSH host key doesn't match stored fingerprint
type HostKeyMismatchError struct {
	Expected   string
	Actual     string
	RemoteAddr string
}

func (e *HostKeyMismatchError) Error() string {
	return fmt.Sprintf("SSH host key mismatch for %s: expected %s, got %s", e.RemoteAddr, e.Expected, e.Actual)
}

// createHostKeyCallback creates a host key callback for SSH connections.
// If the environment has a stored host key fingerprint, it verifies against it.
// If no fingerprint is stored (first connection), it accepts any key but the
// fingerprint should be captured and stored by the caller.
func (d *Deployer) createHostKeyCallback(env *Environment) (ssh.HostKeyCallback, error) {
	if env.HostKey == "" {
		// No host key stored yet - this is a first connection
		// Accept any key (TOFU - Trust On First Use)
		// The caller should capture the key for future verification
		return ssh.InsecureIgnoreHostKey(), nil
	}

	// Verify against stored fingerprint
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		fingerprint := ssh.FingerprintSHA256(key)
		if fingerprint != env.HostKey {
			return &HostKeyMismatchError{
				Expected:   env.HostKey,
				Actual:     fingerprint,
				RemoteAddr: remote.String(),
			}
		}
		return nil
	}, nil
}

// GetHostKeyFingerprint connects to a host and returns its SSH host key fingerprint.
// This is used for initial host key capture (TOFU - Trust On First Use).
func (d *Deployer) GetHostKeyFingerprint(host string, port int) (string, error) {
	addr := fmt.Sprintf("%s:%d", host, port)

	var fingerprint string
	captureCallback := func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		fingerprint = ssh.FingerprintSHA256(key)
		// Return an error to close connection - we just want the key
		return fmt.Errorf("host key captured")
	}

	config := &ssh.ClientConfig{
		User:            "probe",
		HostKeyCallback: captureCallback,
		Timeout:         d.timeout,
	}

	// This will fail, but we capture the host key
	ssh.Dial("tcp", addr, config)

	if fingerprint == "" {
		return "", fmt.Errorf("failed to capture host key from %s", addr)
	}

	return fingerprint, nil
}

// SyncEnvironment synchronizes authorized_keys for an environment
// It ensures only the specified public keys are present
func (d *Deployer) SyncEnvironment(env *Environment, deployKey string, authorizedKeys []UserKey) (*DeployResult, error) {
	result := &DeployResult{
		Environment: env.Name,
	}

	// Parse the deploy private key
	signer, err := ssh.ParsePrivateKey([]byte(deployKey))
	if err != nil {
		result.Error = fmt.Errorf("failed to parse deploy key: %w", err)
		return result, result.Error
	}

	// Create host key callback - verify against stored fingerprint if available
	hostKeyCallback, err := d.createHostKeyCallback(env)
	if err != nil {
		result.Error = fmt.Errorf("failed to setup host key verification: %w", err)
		return result, result.Error
	}

	// Connect to the remote server
	config := &ssh.ClientConfig{
		User: env.DeployUser,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: hostKeyCallback,
		Timeout:         d.timeout,
	}

	addr := fmt.Sprintf("%s:%d", env.Host, env.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		result.Error = fmt.Errorf("failed to connect to %s: %w", addr, err)
		return result, result.Error
	}
	defer client.Close()

	// Get current authorized_keys
	currentKeys, err := d.readAuthorizedKeys(client)
	if err != nil {
		result.Error = fmt.Errorf("failed to read authorized_keys: %w", err)
		return result, result.Error
	}

	// Build new authorized_keys content
	newContent, added, removed := d.buildAuthorizedKeys(currentKeys, authorizedKeys)

	// Write new authorized_keys
	if err := d.writeAuthorizedKeys(client, newContent); err != nil {
		result.Error = fmt.Errorf("failed to write authorized_keys: %w", err)
		return result, result.Error
	}

	result.Success = true
	result.KeysAdded = added
	result.KeysRemoved = removed
	result.Message = fmt.Sprintf("Synced %d keys (+%d, -%d)", len(authorizedKeys), added, removed)

	return result, nil
}

// UserKey represents a user's SSH public key for deployment
type UserKey struct {
	UserName  string
	PublicKey string
}

// AddKey adds a single user's public key to an environment
func (d *Deployer) AddKey(env *Environment, deployKey string, userKey UserKey) error {
	signer, err := ssh.ParsePrivateKey([]byte(deployKey))
	if err != nil {
		return fmt.Errorf("failed to parse deploy key: %w", err)
	}

	config := &ssh.ClientConfig{
		User: env.DeployUser,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         d.timeout,
	}

	addr := fmt.Sprintf("%s:%d", env.Host, env.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", addr, err)
	}
	defer client.Close()

	// Read current keys
	currentKeys, err := d.readAuthorizedKeys(client)
	if err != nil {
		return fmt.Errorf("failed to read authorized_keys: %w", err)
	}

	// Check if key already exists
	keyLine := d.formatKeyLine(userKey)
	for _, line := range currentKeys {
		if d.keysMatch(line, keyLine) {
			// Key already exists
			return nil
		}
	}

	// Append the new key
	currentKeys = append(currentKeys, keyLine)
	content := strings.Join(currentKeys, "\n")
	if len(currentKeys) > 0 {
		content += "\n"
	}

	return d.writeAuthorizedKeys(client, content)
}

// RemoveKey removes a user's public key from an environment
func (d *Deployer) RemoveKey(env *Environment, deployKey string, userName string) error {
	signer, err := ssh.ParsePrivateKey([]byte(deployKey))
	if err != nil {
		return fmt.Errorf("failed to parse deploy key: %w", err)
	}

	config := &ssh.ClientConfig{
		User: env.DeployUser,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         d.timeout,
	}

	addr := fmt.Sprintf("%s:%d", env.Host, env.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", addr, err)
	}
	defer client.Close()

	// Read current keys
	currentKeys, err := d.readAuthorizedKeys(client)
	if err != nil {
		return fmt.Errorf("failed to read authorized_keys: %w", err)
	}

	// Filter out keys belonging to this user
	marker := fmt.Sprintf("magebox:%s", userName)
	var newKeys []string
	for _, line := range currentKeys {
		if !strings.Contains(line, marker) {
			newKeys = append(newKeys, line)
		}
	}

	// Write updated keys
	content := strings.Join(newKeys, "\n")
	if len(newKeys) > 0 {
		content += "\n"
	}

	return d.writeAuthorizedKeys(client, content)
}

// readAuthorizedKeys reads the authorized_keys file from the remote server
func (d *Deployer) readAuthorizedKeys(client *ssh.Client) ([]string, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	var stdout bytes.Buffer
	session.Stdout = &stdout

	// Read authorized_keys, create if doesn't exist
	cmd := "mkdir -p ~/.ssh && chmod 700 ~/.ssh && touch ~/.ssh/authorized_keys && cat ~/.ssh/authorized_keys"
	if err := session.Run(cmd); err != nil {
		return nil, err
	}

	content := stdout.String()
	if content == "" {
		return []string{}, nil
	}

	lines := strings.Split(strings.TrimSpace(content), "\n")
	var keys []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			keys = append(keys, line)
		}
	}

	return keys, nil
}

// writeAuthorizedKeys writes the authorized_keys file to the remote server
func (d *Deployer) writeAuthorizedKeys(client *ssh.Client, content string) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// Encode content as base64 to prevent any shell injection
	// This is safer than heredoc as base64 output cannot contain shell metacharacters
	encoded := base64.StdEncoding.EncodeToString([]byte(content))

	// Use echo with base64 decode - safe because base64 output is alphanumeric+/+=
	cmd := fmt.Sprintf("echo '%s' | base64 -d > ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys", encoded)

	var stderr bytes.Buffer
	session.Stderr = &stderr

	if err := session.Run(cmd); err != nil {
		return fmt.Errorf("%w: %s", err, stderr.String())
	}

	return nil
}

// validateSSHPublicKey validates that a string is a valid SSH public key
func validateSSHPublicKey(key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("empty SSH key")
	}

	parts := strings.Fields(key)
	if len(parts) < 2 {
		return fmt.Errorf("invalid SSH key format: must have at least type and data")
	}

	// Validate key type
	keyType := parts[0]
	validType := false
	for _, valid := range validSSHKeyTypes {
		if keyType == valid {
			validType = true
			break
		}
	}
	if !validType {
		return fmt.Errorf("unsupported SSH key type: %s", keyType)
	}

	// Validate base64 data
	keyData := parts[1]
	if _, err := base64.StdEncoding.DecodeString(keyData); err != nil {
		return fmt.Errorf("invalid SSH key data: not valid base64")
	}

	return nil
}

// sanitizeUsername ensures username contains only safe characters
func sanitizeUsername(username string) string {
	if !validUsernameRegex.MatchString(username) {
		// Replace invalid characters with underscore
		var result strings.Builder
		for _, r := range username {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
				(r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
				result.WriteRune(r)
			} else {
				result.WriteRune('_')
			}
		}
		return result.String()
	}
	return username
}

// formatKeyLine formats a user key as an authorized_keys line with MageBox marker
func (d *Deployer) formatKeyLine(key UserKey) string {
	// Add MageBox marker to identify keys we manage
	pubKey := strings.TrimSpace(key.PublicKey)

	// Validate the SSH key format
	if err := validateSSHPublicKey(pubKey); err != nil {
		// Return empty for invalid keys - they will be filtered out
		return ""
	}

	// Sanitize the username to prevent injection
	safeUsername := sanitizeUsername(key.UserName)

	// Check if key already has a comment
	parts := strings.Fields(pubKey)
	if len(parts) >= 2 {
		// Key format: type base64 [comment]
		// We'll append our marker to the comment
		if len(parts) == 2 {
			// No comment, add one
			return fmt.Sprintf("%s %s magebox:%s", parts[0], parts[1], safeUsername)
		}
		// Has comment, check if it's already a magebox marker
		if strings.HasPrefix(parts[2], "magebox:") {
			return pubKey // Already has our marker
		}
		// Replace comment with our marker (preserving original in marker)
		return fmt.Sprintf("%s %s magebox:%s", parts[0], parts[1], safeUsername)
	}

	return pubKey
}

// buildAuthorizedKeys builds the new authorized_keys content
// Returns: content, keys added, keys removed
func (d *Deployer) buildAuthorizedKeys(currentKeys []string, newKeys []UserKey) (string, int, int) {
	// Separate managed (magebox) keys from unmanaged keys
	var unmanagedKeys []string
	managedKeyMap := make(map[string]bool)

	for _, line := range currentKeys {
		if strings.Contains(line, "magebox:") {
			// Extract the key part for comparison
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				keyID := parts[0] + " " + parts[1]
				managedKeyMap[keyID] = true
			}
		} else {
			// Preserve unmanaged keys
			unmanagedKeys = append(unmanagedKeys, line)
		}
	}

	// Build new managed keys
	var newManagedKeys []string
	newKeyMap := make(map[string]bool)
	added := 0

	for _, key := range newKeys {
		line := d.formatKeyLine(key)
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			keyID := parts[0] + " " + parts[1]
			if !managedKeyMap[keyID] {
				added++
			}
			newKeyMap[keyID] = true
		}
		newManagedKeys = append(newManagedKeys, line)
	}

	// Count removed keys
	removed := 0
	for keyID := range managedKeyMap {
		if !newKeyMap[keyID] {
			removed++
		}
	}

	// Combine unmanaged + new managed keys
	var allKeys []string
	allKeys = append(allKeys, unmanagedKeys...)
	allKeys = append(allKeys, newManagedKeys...)

	content := ""
	if len(allKeys) > 0 {
		content = strings.Join(allKeys, "\n") + "\n"
	}

	return content, added, removed
}

// keysMatch checks if two key lines represent the same key
func (d *Deployer) keysMatch(line1, line2 string) bool {
	parts1 := strings.Fields(line1)
	parts2 := strings.Fields(line2)

	if len(parts1) < 2 || len(parts2) < 2 {
		return false
	}

	// Compare type and base64 key data
	return parts1[0] == parts2[0] && parts1[1] == parts2[1]
}

// TestConnection tests SSH connectivity to an environment
func (d *Deployer) TestConnection(env *Environment, deployKey string) error {
	signer, err := ssh.ParsePrivateKey([]byte(deployKey))
	if err != nil {
		return fmt.Errorf("failed to parse deploy key: %w", err)
	}

	config := &ssh.ClientConfig{
		User: env.DeployUser,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         d.timeout,
	}

	addr := fmt.Sprintf("%s:%d", env.Host, env.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", addr, err)
	}
	defer client.Close()

	// Run a simple command to verify
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	if err := session.Run("echo 'MageBox connection test'"); err != nil {
		return fmt.Errorf("failed to run test command: %w", err)
	}

	return nil
}

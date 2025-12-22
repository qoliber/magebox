/**
 * Created by Qoliber
 *
 * @category    Qoliber
 * @package     MageBox
 * @author      Jakub Winkler <jwinkler@qoliber.com>
 */

package teamserver

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/ssh"
)

// Argon2 parameters (OWASP recommended)
const (
	argon2Time    = 3
	argon2Memory  = 64 * 1024 // 64MB
	argon2Threads = 4
	argon2KeyLen  = 32
	saltLength    = 16
)

// Crypto handles all cryptographic operations
type Crypto struct {
	masterKey []byte
}

// NewCrypto creates a new Crypto instance with the given master key
func NewCrypto(masterKey []byte) (*Crypto, error) {
	if len(masterKey) != 32 {
		return nil, fmt.Errorf("master key must be 32 bytes (AES-256)")
	}
	return &Crypto{masterKey: masterKey}, nil
}

// GenerateMasterKey generates a new random master key
func GenerateMasterKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate master key: %w", err)
	}
	return key, nil
}

// GenerateToken generates a cryptographically secure random token
func GenerateToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// GenerateInviteToken generates a user-friendly invite token
func GenerateInviteToken() (string, error) {
	// Generate 24 random bytes = 32 char base64 string
	return GenerateToken(24)
}

// GenerateSessionToken generates a session token
func GenerateSessionToken() (string, error) {
	// Generate 32 random bytes = 43 char base64 string
	return GenerateToken(32)
}

// HashToken hashes a token using Argon2id
func HashToken(token string) (string, error) {
	salt := make([]byte, saltLength)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	hash := argon2.IDKey([]byte(token), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)

	// Format: $argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argon2Memory, argon2Time, argon2Threads, b64Salt, b64Hash), nil
}

// VerifyToken verifies a token against its hash
func VerifyToken(token, encodedHash string) bool {
	// Parse the encoded hash
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false
	}

	if parts[1] != "argon2id" {
		return false
	}

	var memory, time uint32
	var threads uint8
	_, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads)
	if err != nil {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}

	// Compute hash with same parameters
	computedHash := argon2.IDKey([]byte(token), salt, time, memory, threads, uint32(len(expectedHash)))

	// Constant-time comparison
	return subtle.ConstantTimeCompare(computedHash, expectedHash) == 1
}

// Encrypt encrypts data using AES-256-GCM
func (c *Crypto) Encrypt(plaintext []byte) (string, error) {
	block, err := aes.NewCipher(c.masterKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts data using AES-256-GCM
func (c *Crypto) Decrypt(encoded string) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	block, err := aes.NewCipher(c.masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// EncryptString encrypts a string
func (c *Crypto) EncryptString(plaintext string) (string, error) {
	return c.Encrypt([]byte(plaintext))
}

// DecryptString decrypts to a string
func (c *Crypto) DecryptString(encoded string) (string, error) {
	plaintext, err := c.Decrypt(encoded)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// HashForChain creates a SHA-256 hash for audit log chain
func HashForChain(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// ComputeAuditHash computes hash for an audit entry (for tamper detection)
func ComputeAuditHash(entry *AuditEntry, prevHash string) string {
	data := fmt.Sprintf("%d|%s|%s|%s|%s|%s|%s",
		entry.ID,
		entry.Timestamp.UTC().Format("2006-01-02T15:04:05Z"),
		entry.UserName,
		entry.Action,
		entry.Details,
		entry.IPAddress,
		prevHash,
	)
	return HashForChain(data)
}

// VerifyAuditChain verifies the integrity of audit log entries
func VerifyAuditChain(entries []AuditEntry) (bool, int) {
	prevHash := ""
	for i, entry := range entries {
		expectedHash := ComputeAuditHash(&entry, prevHash)
		if entry.Hash != expectedHash {
			return false, i
		}
		prevHash = entry.Hash
	}
	return true, -1
}

// MasterKeyToHex converts master key to hex string for storage
func MasterKeyToHex(key []byte) string {
	return hex.EncodeToString(key)
}

// MasterKeyFromHex converts hex string to master key
func MasterKeyFromHex(hexKey string) ([]byte, error) {
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("invalid master key hex: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("master key must be 32 bytes")
	}
	return key, nil
}

// SSHKeyPair represents a generated SSH key pair
type SSHKeyPair struct {
	PrivateKey    string // PEM-encoded private key
	PublicKey     string // OpenSSH format public key (for authorized_keys)
	PrivateKeyPEM []byte // Raw PEM bytes for saving to file
}

// GenerateSSHKeyPair generates a new Ed25519 SSH key pair
func GenerateSSHKeyPair(comment string) (*SSHKeyPair, error) {
	// Generate Ed25519 key pair
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate Ed25519 key: %w", err)
	}

	// Convert private key to OpenSSH format
	privKeyPEM, err := ssh.MarshalPrivateKey(privKey, comment)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key: %w", err)
	}

	// Convert public key to OpenSSH authorized_keys format
	sshPubKey, err := ssh.NewPublicKey(pubKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH public key: %w", err)
	}

	authorizedKey := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPubKey)))
	if comment != "" {
		authorizedKey = authorizedKey + " " + comment
	}

	return &SSHKeyPair{
		PrivateKey:    string(pem.EncodeToMemory(privKeyPEM)),
		PublicKey:     authorizedKey,
		PrivateKeyPEM: pem.EncodeToMemory(privKeyPEM),
	}, nil
}

// ParseSSHPublicKey parses an OpenSSH public key and returns the key type and fingerprint
func ParseSSHPublicKey(publicKey string) (keyType string, fingerprint string, err error) {
	pubKey, comment, _, _, err := ssh.ParseAuthorizedKey([]byte(publicKey))
	if err != nil {
		return "", "", fmt.Errorf("failed to parse public key: %w", err)
	}

	keyType = pubKey.Type()
	fingerprint = ssh.FingerprintSHA256(pubKey)

	// Include comment in fingerprint display if present
	if comment != "" {
		fingerprint = fingerprint + " (" + comment + ")"
	}

	return keyType, fingerprint, nil
}

// CAKeyPair represents an SSH Certificate Authority key pair
type CAKeyPair struct {
	PrivateKey    ed25519.PrivateKey // CA private key for signing
	PublicKey     ed25519.PublicKey  // CA public key
	PrivateKeyPEM string             // PEM-encoded private key (for storage)
	PublicKeySSH  string             // OpenSSH format public key (for deployment to servers)
}

// GenerateCAKeyPair generates a new Ed25519 CA key pair
func GenerateCAKeyPair() (*CAKeyPair, error) {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate CA key pair: %w", err)
	}

	// Convert private key to PEM format
	privKeyPEM, err := ssh.MarshalPrivateKey(privKey, "magebox-ca")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal CA private key: %w", err)
	}

	// Convert public key to OpenSSH format
	sshPubKey, err := ssh.NewPublicKey(pubKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH public key: %w", err)
	}

	publicKeySSH := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPubKey))) + " magebox-ca"

	return &CAKeyPair{
		PrivateKey:    privKey,
		PublicKey:     pubKey,
		PrivateKeyPEM: string(pem.EncodeToMemory(privKeyPEM)),
		PublicKeySSH:  publicKeySSH,
	}, nil
}

// ParseCAPrivateKey parses a PEM-encoded CA private key
func ParseCAPrivateKey(pemData string) (ed25519.PrivateKey, error) {
	privKey, err := ssh.ParseRawPrivateKey([]byte(pemData))
	if err != nil {
		return nil, fmt.Errorf("failed to parse CA private key: %w", err)
	}

	ed25519Key, ok := privKey.(*ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("CA private key is not Ed25519")
	}

	return *ed25519Key, nil
}

// SSHCertificate represents a signed SSH certificate
type SSHCertificate struct {
	Certificate    string   // OpenSSH certificate format (user saves as *-cert.pub)
	CertificateRaw []byte   // Raw certificate bytes
	Serial         uint64   // Certificate serial number
	ValidAfter     uint64   // Unix timestamp - valid from
	ValidBefore    uint64   // Unix timestamp - valid until
	KeyID          string   // Certificate key ID (usually user email)
	Principals     []string // Allowed usernames (e.g., "deploy")
}

// SignSSHCertificate signs a user's public key with the CA, creating a certificate
func SignSSHCertificate(caPrivateKey ed25519.PrivateKey, userPublicKey string, keyID string, principals []string, validityDuration int64) (*SSHCertificate, error) {
	// Parse the user's public key
	pubKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(userPublicKey))
	if err != nil {
		return nil, fmt.Errorf("failed to parse user public key: %w", err)
	}

	// Create CA signer
	caSigner, err := ssh.NewSignerFromKey(caPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create CA signer: %w", err)
	}

	// Generate certificate serial number
	serialBytes := make([]byte, 8)
	if _, err := io.ReadFull(rand.Reader, serialBytes); err != nil {
		return nil, fmt.Errorf("failed to generate serial: %w", err)
	}
	serial := uint64(serialBytes[0])<<56 | uint64(serialBytes[1])<<48 | uint64(serialBytes[2])<<40 |
		uint64(serialBytes[3])<<32 | uint64(serialBytes[4])<<24 | uint64(serialBytes[5])<<16 |
		uint64(serialBytes[6])<<8 | uint64(serialBytes[7])

	// Calculate validity times
	now := uint64(time.Now().Unix())
	validAfter := now - 60 // 1 minute grace period for clock skew
	validBefore := now + uint64(validityDuration)

	// Create the certificate
	cert := &ssh.Certificate{
		Key:             pubKey,
		Serial:          serial,
		CertType:        ssh.UserCert,
		KeyId:           keyID,
		ValidPrincipals: principals,
		ValidAfter:      validAfter,
		ValidBefore:     validBefore,
		Permissions: ssh.Permissions{
			Extensions: map[string]string{
				"permit-pty":              "",
				"permit-user-rc":          "",
				"permit-agent-forwarding": "",
				"permit-port-forwarding":  "",
			},
		},
	}

	// Sign the certificate
	if err := cert.SignCert(rand.Reader, caSigner); err != nil {
		return nil, fmt.Errorf("failed to sign certificate: %w", err)
	}

	// Marshal to OpenSSH format
	certBytes := ssh.MarshalAuthorizedKey(cert)
	certString := strings.TrimSpace(string(certBytes))

	return &SSHCertificate{
		Certificate:    certString,
		CertificateRaw: certBytes,
		Serial:         serial,
		ValidAfter:     validAfter,
		ValidBefore:    validBefore,
		KeyID:          keyID,
		Principals:     principals,
	}, nil
}

// SSHKeyPairWithCert represents an SSH key pair with a signed certificate
type SSHKeyPairWithCert struct {
	PrivateKey    string          // PEM-encoded private key
	PublicKey     string          // OpenSSH format public key
	PrivateKeyPEM []byte          // Raw PEM bytes
	Certificate   *SSHCertificate // Signed certificate
}

// GenerateSSHKeyPairWithCert generates a new SSH key pair and signs it with the CA
func GenerateSSHKeyPairWithCert(caPrivateKey ed25519.PrivateKey, keyID string, principals []string, validityDuration int64, comment string) (*SSHKeyPairWithCert, error) {
	// Generate the key pair
	keyPair, err := GenerateSSHKeyPair(comment)
	if err != nil {
		return nil, err
	}

	// Sign the public key
	cert, err := SignSSHCertificate(caPrivateKey, keyPair.PublicKey, keyID, principals, validityDuration)
	if err != nil {
		return nil, err
	}

	return &SSHKeyPairWithCert{
		PrivateKey:    keyPair.PrivateKey,
		PublicKey:     keyPair.PublicKey,
		PrivateKeyPEM: keyPair.PrivateKeyPEM,
		Certificate:   cert,
	}, nil
}

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
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/argon2"
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

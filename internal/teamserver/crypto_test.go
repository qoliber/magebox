/**
 * Created by Qoliber
 *
 * @category    Qoliber
 * @package     MageBox
 * @author      Jakub Winkler <jwinkler@qoliber.com>
 */

package teamserver

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateMasterKey(t *testing.T) {
	key, err := GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey failed: %v", err)
	}

	if len(key) != 32 {
		t.Errorf("Expected key length 32, got %d", len(key))
	}

	// Generate another key and ensure they're different
	key2, err := GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey failed: %v", err)
	}

	if string(key) == string(key2) {
		t.Error("Generated keys should be unique")
	}
}

func TestGenerateToken(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"short", 16},
		{"medium", 24},
		{"long", 32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GenerateToken(tt.length)
			if err != nil {
				t.Fatalf("GenerateToken failed: %v", err)
			}

			if token == "" {
				t.Error("Token should not be empty")
			}

			// Generate another and ensure uniqueness
			token2, _ := GenerateToken(tt.length)
			if token == token2 {
				t.Error("Tokens should be unique")
			}
		})
	}
}

func TestGenerateInviteToken(t *testing.T) {
	token, err := GenerateInviteToken()
	if err != nil {
		t.Fatalf("GenerateInviteToken failed: %v", err)
	}

	if token == "" {
		t.Error("Invite token should not be empty")
	}

	// Should be URL-safe base64
	if strings.ContainsAny(token, "+/") {
		t.Error("Invite token should be URL-safe (no + or /)")
	}
}

func TestHashAndVerifyToken(t *testing.T) {
	token := "test-token-12345"

	hash, err := HashToken(token)
	if err != nil {
		t.Fatalf("HashToken failed: %v", err)
	}

	// Hash should start with $argon2id$
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Errorf("Hash should start with $argon2id$, got: %s", hash[:20])
	}

	// Verify correct token
	if !VerifyToken(token, hash) {
		t.Error("VerifyToken should return true for correct token")
	}

	// Verify incorrect token
	if VerifyToken("wrong-token", hash) {
		t.Error("VerifyToken should return false for incorrect token")
	}

	// Verify empty token
	if VerifyToken("", hash) {
		t.Error("VerifyToken should return false for empty token")
	}
}

func TestHashTokenUniqueness(t *testing.T) {
	token := "same-token"

	hash1, _ := HashToken(token)
	hash2, _ := HashToken(token)

	// Same token should produce different hashes (different salts)
	if hash1 == hash2 {
		t.Error("Same token should produce different hashes due to random salt")
	}

	// But both should verify correctly
	if !VerifyToken(token, hash1) {
		t.Error("First hash should verify")
	}
	if !VerifyToken(token, hash2) {
		t.Error("Second hash should verify")
	}
}

func TestVerifyTokenInvalidFormats(t *testing.T) {
	tests := []struct {
		name string
		hash string
	}{
		{"empty", ""},
		{"no prefix", "invalid-hash"},
		{"wrong algorithm", "$bcrypt$..."},
		{"incomplete", "$argon2id$v=19"},
		{"missing parts", "$argon2id$v=19$m=65536"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if VerifyToken("any-token", tt.hash) {
				t.Errorf("VerifyToken should return false for invalid hash format: %s", tt.name)
			}
		})
	}
}

func TestCryptoEncryptDecrypt(t *testing.T) {
	key, _ := GenerateMasterKey()
	crypto, err := NewCrypto(key)
	if err != nil {
		t.Fatalf("NewCrypto failed: %v", err)
	}

	testCases := []struct {
		name      string
		plaintext string
	}{
		{"short", "hello"},
		{"medium", "this is a medium length string for testing"},
		{"long", strings.Repeat("long text ", 100)},
		{"unicode", "Hello, "},
		{"special chars", "!@#$%^&*()_+-=[]{}|;':\",./<>?"},
		{"empty", ""},
		{"ssh key", "-----BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaC1rZXktdjEAAAAABG5vbmUA..."},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			encrypted, err := crypto.EncryptString(tc.plaintext)
			if err != nil {
				t.Fatalf("Encrypt failed: %v", err)
			}

			// Encrypted should be different from plaintext
			if encrypted == tc.plaintext && tc.plaintext != "" {
				t.Error("Encrypted should differ from plaintext")
			}

			// Decrypt
			decrypted, err := crypto.DecryptString(encrypted)
			if err != nil {
				t.Fatalf("Decrypt failed: %v", err)
			}

			if decrypted != tc.plaintext {
				t.Errorf("Decrypted doesn't match original. Got: %s, Want: %s", decrypted, tc.plaintext)
			}
		})
	}
}

func TestCryptoEncryptUniqueness(t *testing.T) {
	key, _ := GenerateMasterKey()
	crypto, _ := NewCrypto(key)

	plaintext := "same plaintext"

	enc1, _ := crypto.EncryptString(plaintext)
	enc2, _ := crypto.EncryptString(plaintext)

	// Same plaintext should produce different ciphertexts (different nonces)
	if enc1 == enc2 {
		t.Error("Same plaintext should produce different ciphertexts")
	}

	// Both should decrypt correctly
	dec1, _ := crypto.DecryptString(enc1)
	dec2, _ := crypto.DecryptString(enc2)

	if dec1 != plaintext || dec2 != plaintext {
		t.Error("Both ciphertexts should decrypt to original")
	}
}

func TestCryptoWrongKey(t *testing.T) {
	key1, _ := GenerateMasterKey()
	key2, _ := GenerateMasterKey()

	crypto1, _ := NewCrypto(key1)
	crypto2, _ := NewCrypto(key2)

	plaintext := "secret data"
	encrypted, _ := crypto1.EncryptString(plaintext)

	// Try to decrypt with wrong key
	_, err := crypto2.DecryptString(encrypted)
	if err == nil {
		t.Error("Decrypt with wrong key should fail")
	}
}

func TestNewCryptoInvalidKey(t *testing.T) {
	tests := []struct {
		name string
		key  []byte
	}{
		{"too short", []byte("short")},
		{"too long", make([]byte, 64)},
		{"empty", []byte{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewCrypto(tt.key)
			if err == nil {
				t.Error("NewCrypto should fail with invalid key length")
			}
		})
	}
}

func TestHashForChain(t *testing.T) {
	data := "test data for hashing"

	hash1 := HashForChain(data)
	hash2 := HashForChain(data)

	// Same data should produce same hash
	if hash1 != hash2 {
		t.Error("Same data should produce same hash")
	}

	// Hash should be hex string of length 64 (SHA-256)
	if len(hash1) != 64 {
		t.Errorf("Hash length should be 64, got %d", len(hash1))
	}

	// Different data should produce different hash
	hash3 := HashForChain("different data")
	if hash1 == hash3 {
		t.Error("Different data should produce different hash")
	}
}

func TestComputeAuditHash(t *testing.T) {
	entry := &AuditEntry{
		ID:        1,
		Timestamp: time.Date(2025, 12, 17, 10, 30, 0, 0, time.UTC),
		UserName:  "admin",
		Action:    AuditUserCreate,
		Details:   "Created user: testuser",
		IPAddress: "192.168.1.1",
	}

	hash1 := ComputeAuditHash(entry, "")
	hash2 := ComputeAuditHash(entry, "")

	// Same entry and prevHash should produce same hash
	if hash1 != hash2 {
		t.Error("Same entry should produce same hash")
	}

	// Different prevHash should produce different hash
	hash3 := ComputeAuditHash(entry, "different-prev-hash")
	if hash1 == hash3 {
		t.Error("Different prevHash should produce different hash")
	}
}

func TestVerifyAuditChain(t *testing.T) {
	// Create valid chain
	entries := make([]AuditEntry, 3)
	prevHash := ""

	for i := 0; i < 3; i++ {
		entries[i] = AuditEntry{
			ID:        int64(i + 1),
			Timestamp: time.Now().UTC(),
			UserName:  "admin",
			Action:    AuditUserCreate,
			Details:   "Entry " + string(rune('A'+i)),
			IPAddress: "127.0.0.1",
			PrevHash:  prevHash,
		}
		entries[i].Hash = ComputeAuditHash(&entries[i], prevHash)
		prevHash = entries[i].Hash
	}

	// Valid chain should verify
	valid, idx := VerifyAuditChain(entries)
	if !valid {
		t.Errorf("Valid chain should verify, failed at index %d", idx)
	}

	// Tamper with middle entry
	entries[1].Details = "TAMPERED"

	valid, idx = VerifyAuditChain(entries)
	if valid {
		t.Error("Tampered chain should not verify")
	}
	if idx != 1 {
		t.Errorf("Should fail at tampered index 1, got %d", idx)
	}
}

func TestMasterKeyHexConversion(t *testing.T) {
	key, _ := GenerateMasterKey()

	hex := MasterKeyToHex(key)
	if len(hex) != 64 { // 32 bytes = 64 hex chars
		t.Errorf("Hex should be 64 chars, got %d", len(hex))
	}

	recovered, err := MasterKeyFromHex(hex)
	if err != nil {
		t.Fatalf("MasterKeyFromHex failed: %v", err)
	}

	if string(key) != string(recovered) {
		t.Error("Recovered key should match original")
	}
}

func TestMasterKeyFromHexInvalid(t *testing.T) {
	tests := []struct {
		name string
		hex  string
	}{
		{"invalid hex", "not-hex-string"},
		{"too short", "abcd1234"},
		{"too long", strings.Repeat("ab", 64)},
		{"odd length", "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := MasterKeyFromHex(tt.hex)
			if err == nil {
				t.Error("MasterKeyFromHex should fail for invalid input")
			}
		})
	}
}

// Benchmark tests
func BenchmarkHashToken(b *testing.B) {
	token := "benchmark-token-12345"
	for i := 0; i < b.N; i++ {
		HashToken(token)
	}
}

func BenchmarkVerifyToken(b *testing.B) {
	token := "benchmark-token-12345"
	hash, _ := HashToken(token)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		VerifyToken(token, hash)
	}
}

func BenchmarkEncrypt(b *testing.B) {
	key, _ := GenerateMasterKey()
	crypto, _ := NewCrypto(key)
	plaintext := strings.Repeat("benchmark data ", 10)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		crypto.EncryptString(plaintext)
	}
}

func BenchmarkDecrypt(b *testing.B) {
	key, _ := GenerateMasterKey()
	crypto, _ := NewCrypto(key)
	encrypted, _ := crypto.EncryptString(strings.Repeat("benchmark data ", 10))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		crypto.DecryptString(encrypted)
	}
}

// SSH Key Generation Tests

func TestGenerateSSHKeyPair(t *testing.T) {
	comment := "test@example.com"
	keyPair, err := GenerateSSHKeyPair(comment)
	if err != nil {
		t.Fatalf("GenerateSSHKeyPair failed: %v", err)
	}

	// Check private key format
	if !strings.HasPrefix(keyPair.PrivateKey, "-----BEGIN OPENSSH PRIVATE KEY-----") {
		t.Error("Private key should start with OpenSSH header")
	}
	if !strings.HasSuffix(strings.TrimSpace(keyPair.PrivateKey), "-----END OPENSSH PRIVATE KEY-----") {
		t.Error("Private key should end with OpenSSH footer")
	}

	// Check public key format
	if !strings.HasPrefix(keyPair.PublicKey, "ssh-ed25519 ") {
		t.Errorf("Public key should start with 'ssh-ed25519 ', got: %s", keyPair.PublicKey[:min(20, len(keyPair.PublicKey))])
	}

	// Check comment is included
	if !strings.HasSuffix(keyPair.PublicKey, comment) {
		t.Errorf("Public key should end with comment '%s', got: %s", comment, keyPair.PublicKey)
	}

	// Check PrivateKeyPEM is set
	if len(keyPair.PrivateKeyPEM) == 0 {
		t.Error("PrivateKeyPEM should not be empty")
	}
}

func TestGenerateSSHKeyPairUniqueness(t *testing.T) {
	keyPair1, _ := GenerateSSHKeyPair("test1")
	keyPair2, _ := GenerateSSHKeyPair("test2")

	// Private keys should be different
	if keyPair1.PrivateKey == keyPair2.PrivateKey {
		t.Error("Generated private keys should be unique")
	}

	// Public keys should be different
	// Compare just the key part, not the comment
	parts1 := strings.Fields(keyPair1.PublicKey)
	parts2 := strings.Fields(keyPair2.PublicKey)
	if len(parts1) >= 2 && len(parts2) >= 2 && parts1[1] == parts2[1] {
		t.Error("Generated public keys should be unique")
	}
}

func TestGenerateSSHKeyPairEmptyComment(t *testing.T) {
	keyPair, err := GenerateSSHKeyPair("")
	if err != nil {
		t.Fatalf("GenerateSSHKeyPair with empty comment failed: %v", err)
	}

	// Should still generate valid keys
	if !strings.HasPrefix(keyPair.PublicKey, "ssh-ed25519 ") {
		t.Error("Public key should be valid even with empty comment")
	}
}

func TestParseSSHPublicKey(t *testing.T) {
	// Generate a key pair first
	keyPair, err := GenerateSSHKeyPair("test@example.com")
	if err != nil {
		t.Fatalf("GenerateSSHKeyPair failed: %v", err)
	}

	// Parse the public key
	keyType, fingerprint, err := ParseSSHPublicKey(keyPair.PublicKey)
	if err != nil {
		t.Fatalf("ParseSSHPublicKey failed: %v", err)
	}

	// Check key type
	if keyType != "ssh-ed25519" {
		t.Errorf("Expected key type 'ssh-ed25519', got '%s'", keyType)
	}

	// Check fingerprint format (should start with SHA256:)
	if !strings.HasPrefix(fingerprint, "SHA256:") {
		t.Errorf("Fingerprint should start with 'SHA256:', got: %s", fingerprint)
	}
}

func TestParseSSHPublicKeyInvalid(t *testing.T) {
	tests := []struct {
		name      string
		publicKey string
	}{
		{"empty", ""},
		{"invalid format", "not a valid key"},
		{"incomplete", "ssh-ed25519"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParseSSHPublicKey(tt.publicKey)
			if err == nil {
				t.Error("ParseSSHPublicKey should fail for invalid key")
			}
		})
	}
}

func BenchmarkGenerateSSHKeyPair(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenerateSSHKeyPair("benchmark@test.com")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

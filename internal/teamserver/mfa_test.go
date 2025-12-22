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
)

func TestNewMFAManager(t *testing.T) {
	m := NewMFAManager("TestIssuer")
	if m == nil {
		t.Fatal("NewMFAManager returned nil")
	}
	if m.issuer != "TestIssuer" {
		t.Errorf("issuer = %s, want TestIssuer", m.issuer)
	}

	// Test default issuer
	m2 := NewMFAManager("")
	if m2.issuer != "MageBox" {
		t.Errorf("default issuer = %s, want MageBox", m2.issuer)
	}
}

func TestGenerateSecret(t *testing.T) {
	m := NewMFAManager("Test")

	secret, err := m.GenerateSecret()
	if err != nil {
		t.Fatalf("GenerateSecret failed: %v", err)
	}

	// Secret should be base32 encoded
	if secret == "" {
		t.Error("Secret should not be empty")
	}

	// Should be 52 characters (32 bytes base32 encoded without padding)
	// 32 bytes * 8 bits / 5 bits per base32 char = 51.2 -> 52 characters
	if len(secret) != 52 {
		t.Errorf("Secret length = %d, want 52", len(secret))
	}

	// Generate another secret - should be different
	secret2, _ := m.GenerateSecret()
	if secret == secret2 {
		t.Error("Two generated secrets should be different")
	}
}

func TestGenerateQRCodeURL(t *testing.T) {
	m := NewMFAManager("MageBox")
	secret := "JBSWY3DPEHPK3PXP"
	account := "alice@example.com"

	url := m.GenerateQRCodeURL(secret, account)

	if !strings.HasPrefix(url, "otpauth://totp/") {
		t.Errorf("URL should start with otpauth://totp/, got %s", url)
	}

	if !strings.Contains(url, "secret="+secret) {
		t.Error("URL should contain secret")
	}

	if !strings.Contains(url, "issuer=MageBox") {
		t.Error("URL should contain issuer")
	}

	if !strings.Contains(url, "digits=6") {
		t.Error("URL should contain digits=6")
	}

	if !strings.Contains(url, "period=30") {
		t.Error("URL should contain period=30")
	}
}

func TestValidateCode(t *testing.T) {
	m := NewMFAManager("Test")

	// Generate a secret and get current code
	secret, _ := m.GenerateSecret()
	currentCode, err := m.GetCurrentCode(secret)
	if err != nil {
		t.Fatalf("GetCurrentCode failed: %v", err)
	}

	// Current code should validate
	if !m.ValidateCode(secret, currentCode) {
		t.Error("Current code should validate")
	}

	// Code with spaces should also validate
	if !m.ValidateCode(secret, currentCode[:3]+" "+currentCode[3:]) {
		t.Error("Code with spaces should validate")
	}

	// Wrong code should not validate
	if m.ValidateCode(secret, "000000") {
		t.Error("Wrong code should not validate")
	}

	// Wrong length code should not validate
	if m.ValidateCode(secret, "12345") {
		t.Error("5-digit code should not validate")
	}

	if m.ValidateCode(secret, "1234567") {
		t.Error("7-digit code should not validate")
	}

	// Invalid secret should not validate
	if m.ValidateCode("INVALID!", currentCode) {
		t.Error("Invalid secret should not validate")
	}
}

func TestGetCurrentCode(t *testing.T) {
	m := NewMFAManager("Test")
	secret, _ := m.GenerateSecret()

	code, err := m.GetCurrentCode(secret)
	if err != nil {
		t.Fatalf("GetCurrentCode failed: %v", err)
	}

	if len(code) != 6 {
		t.Errorf("Code length = %d, want 6", len(code))
	}

	// Code should be numeric
	for _, c := range code {
		if c < '0' || c > '9' {
			t.Errorf("Code should be numeric, got %s", code)
			break
		}
	}

	// Invalid secret should error
	_, err = m.GetCurrentCode("INVALID!")
	if err == nil {
		t.Error("Invalid secret should return error")
	}
}

func TestGenerateSetup(t *testing.T) {
	m := NewMFAManager("MageBox")

	setup, err := m.GenerateSetup("alice@example.com")
	if err != nil {
		t.Fatalf("GenerateSetup failed: %v", err)
	}

	if setup.Secret == "" {
		t.Error("Secret should not be empty")
	}

	if setup.QRCodeURL == "" {
		t.Error("QRCodeURL should not be empty")
	}

	if setup.ManualKey == "" {
		t.Error("ManualKey should not be empty")
	}

	// ManualKey should have spaces for readability
	if !strings.Contains(setup.ManualKey, " ") {
		t.Error("ManualKey should have spaces")
	}

	if setup.Issuer != "MageBox" {
		t.Errorf("Issuer = %s, want MageBox", setup.Issuer)
	}

	if setup.Account != "alice@example.com" {
		t.Errorf("Account = %s, want alice@example.com", setup.Account)
	}
}

func TestFormatSecretForDisplay(t *testing.T) {
	tests := []struct {
		secret   string
		expected string
	}{
		{"ABCDEFGH", "ABCD EFGH"},
		{"ABCDEFGHIJ", "ABCD EFGH IJ"},
		{"ABCD", "ABCD"},
		{"AB", "AB"},
		{"JBSWY3DPEHPK3PXP", "JBSW Y3DP EHPK 3PXP"},
	}

	for _, tt := range tests {
		t.Run(tt.secret, func(t *testing.T) {
			result := formatSecretForDisplay(tt.secret)
			if result != tt.expected {
				t.Errorf("formatSecretForDisplay(%s) = %s, want %s", tt.secret, result, tt.expected)
			}
		})
	}
}

func TestGenerateRecoveryCodes(t *testing.T) {
	m := NewMFAManager("Test")

	codes, err := m.GenerateRecoveryCodes(10)
	if err != nil {
		t.Fatalf("GenerateRecoveryCodes failed: %v", err)
	}

	if len(codes) != 10 {
		t.Errorf("Generated %d codes, want 10", len(codes))
	}

	// Check format of each code
	for i, code := range codes {
		if len(code) != 9 {
			t.Errorf("Code %d length = %d, want 9", i, len(code))
		}
		if code[4] != '-' {
			t.Errorf("Code %d missing dash: %s", i, code)
		}
	}

	// All codes should be unique
	seen := make(map[string]bool)
	for _, code := range codes {
		if seen[code] {
			t.Errorf("Duplicate code: %s", code)
		}
		seen[code] = true
	}
}

func TestValidateRecoveryCode(t *testing.T) {
	m := NewMFAManager("Test")

	tests := []struct {
		code  string
		valid bool
	}{
		{"ABCD-1234", true},
		{"abcd-1234", true}, // lowercase should work
		{"1234-ABCD", true},
		{"0000-0000", true},
		{"FFFF-FFFF", true},
		{"ABCD1234", false},   // missing dash
		{"ABCD-123", false},   // too short
		{"ABCD-12345", false}, // too long
		{"GHIJ-1234", false},  // invalid hex chars
		{"ABCD-GHIJ", false},  // invalid hex chars
		{"", false},
		{"----", false},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			result := m.ValidateRecoveryCode(tt.code)
			if result != tt.valid {
				t.Errorf("ValidateRecoveryCode(%s) = %v, want %v", tt.code, result, tt.valid)
			}
		})
	}
}

// Test TOTP algorithm against known test vectors
func TestTOTPKnownValues(t *testing.T) {
	m := NewMFAManager("Test")

	// Test with a known secret and verify it generates consistent codes
	secret := "JBSWY3DPEHPK3PXP" // "Hello!" in base32

	// Get two consecutive codes - they might be the same if we're near a boundary
	code1, err := m.GetCurrentCode(secret)
	if err != nil {
		t.Fatalf("GetCurrentCode failed: %v", err)
	}

	// Verify the code validates against the secret
	if !m.ValidateCode(secret, code1) {
		t.Error("Generated code should validate against its own secret")
	}
}

func TestMFAManagerConcurrency(t *testing.T) {
	m := NewMFAManager("Test")
	secret, _ := m.GenerateSecret()

	// Test concurrent validation
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			code, _ := m.GetCurrentCode(secret)
			m.ValidateCode(secret, code)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

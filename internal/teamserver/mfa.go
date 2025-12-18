/**
 * Created by Qoliber
 *
 * @category    Qoliber
 * @package     MageBox
 * @author      Jakub Winkler <jwinkler@qoliber.com>
 */

package teamserver

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	// TOTPDigits is the number of digits in a TOTP code
	TOTPDigits = 6
	// TOTPPeriod is the time step in seconds (30 seconds is standard)
	TOTPPeriod = 30
	// TOTPSecretLength is the length of the secret in bytes
	TOTPSecretLength = 20
	// TOTPWindow allows for clock skew (1 period before/after)
	TOTPWindow = 1
)

// MFAManager handles multi-factor authentication operations
type MFAManager struct {
	issuer string
}

// NewMFAManager creates a new MFA manager
func NewMFAManager(issuer string) *MFAManager {
	if issuer == "" {
		issuer = "MageBox"
	}
	return &MFAManager{
		issuer: issuer,
	}
}

// GenerateSecret generates a new TOTP secret
func (m *MFAManager) GenerateSecret() (string, error) {
	secret := make([]byte, TOTPSecretLength)
	if _, err := rand.Read(secret); err != nil {
		return "", fmt.Errorf("failed to generate secret: %w", err)
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret), nil
}

// GenerateQRCodeURL generates a URL for QR code generation
// This URL can be used with any QR code generator to create a scannable code
func (m *MFAManager) GenerateQRCodeURL(secret, accountName string) string {
	// otpauth://totp/ISSUER:ACCOUNT?secret=SECRET&issuer=ISSUER&algorithm=SHA1&digits=6&period=30
	return fmt.Sprintf(
		"otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=%d&period=%d",
		url.PathEscape(m.issuer),
		url.PathEscape(accountName),
		secret,
		url.QueryEscape(m.issuer),
		TOTPDigits,
		TOTPPeriod,
	)
}

// ValidateCode validates a TOTP code against a secret
func (m *MFAManager) ValidateCode(secret, code string) bool {
	// Remove any spaces from the code
	code = strings.ReplaceAll(code, " ", "")

	if len(code) != TOTPDigits {
		return false
	}

	// Decode the secret
	secretBytes, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return false
	}

	// Get current time counter
	now := time.Now().Unix()
	counter := now / TOTPPeriod

	// Check current and adjacent time windows for clock skew tolerance
	for i := -TOTPWindow; i <= TOTPWindow; i++ {
		expectedCode := m.generateCode(secretBytes, counter+int64(i))
		if expectedCode == code {
			return true
		}
	}

	return false
}

// generateCode generates a TOTP code for a given counter value
func (m *MFAManager) generateCode(secret []byte, counter int64) string {
	// Convert counter to big-endian bytes
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(counter))

	// Generate HMAC-SHA1
	mac := hmac.New(sha1.New, secret)
	mac.Write(buf)
	sum := mac.Sum(nil)

	// Dynamic truncation
	offset := sum[len(sum)-1] & 0x0f
	code := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff

	// Generate 6-digit code
	code = code % 1000000

	return fmt.Sprintf("%06d", code)
}

// GetCurrentCode returns the current TOTP code (for testing/debugging)
func (m *MFAManager) GetCurrentCode(secret string) (string, error) {
	secretBytes, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return "", fmt.Errorf("invalid secret: %w", err)
	}

	counter := time.Now().Unix() / TOTPPeriod
	return m.generateCode(secretBytes, counter), nil
}

// MFASetupResponse contains the data needed to set up MFA
type MFASetupResponse struct {
	Secret    string `json:"secret"`
	QRCodeURL string `json:"qr_code_url"`
	ManualKey string `json:"manual_key"`
	Issuer    string `json:"issuer"`
	Account   string `json:"account"`
}

// GenerateSetup generates MFA setup data for a user
func (m *MFAManager) GenerateSetup(accountName string) (*MFASetupResponse, error) {
	secret, err := m.GenerateSecret()
	if err != nil {
		return nil, err
	}

	return &MFASetupResponse{
		Secret:    secret,
		QRCodeURL: m.GenerateQRCodeURL(secret, accountName),
		ManualKey: formatSecretForDisplay(secret),
		Issuer:    m.issuer,
		Account:   accountName,
	}, nil
}

// formatSecretForDisplay formats a secret with spaces for easier manual entry
func formatSecretForDisplay(secret string) string {
	var parts []string
	for i := 0; i < len(secret); i += 4 {
		end := i + 4
		if end > len(secret) {
			end = len(secret)
		}
		parts = append(parts, secret[i:end])
	}
	return strings.Join(parts, " ")
}

// RecoveryCode represents a single-use recovery code
type RecoveryCode struct {
	Code   string
	UsedAt *time.Time
}

// GenerateRecoveryCodes generates a set of single-use recovery codes
func (m *MFAManager) GenerateRecoveryCodes(count int) ([]string, error) {
	codes := make([]string, count)
	for i := 0; i < count; i++ {
		code, err := m.generateRecoveryCode()
		if err != nil {
			return nil, err
		}
		codes[i] = code
	}
	return codes, nil
}

// generateRecoveryCode generates a single recovery code
func (m *MFAManager) generateRecoveryCode() (string, error) {
	bytes := make([]byte, 5)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	// Format as XXXX-XXXX (8 hex characters with dash)
	return fmt.Sprintf("%04X-%04X",
		binary.BigEndian.Uint16(bytes[0:2]),
		binary.BigEndian.Uint16(bytes[2:4])), nil
}

// ValidateRecoveryCode validates a recovery code format
func (m *MFAManager) ValidateRecoveryCode(code string) bool {
	// Format: XXXX-XXXX
	code = strings.ToUpper(strings.TrimSpace(code))
	if len(code) != 9 {
		return false
	}
	if code[4] != '-' {
		return false
	}
	// Check all characters are hex
	for i, c := range code {
		if i == 4 {
			continue
		}
		if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

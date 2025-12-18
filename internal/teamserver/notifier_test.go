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

func TestNewNotifier(t *testing.T) {
	config := SMTPConfig{
		Enabled: true,
		Host:    "smtp.example.com",
		Port:    587,
		From:    "noreply@example.com",
	}

	n := NewNotifier(config)
	if n == nil {
		t.Fatal("NewNotifier returned nil")
	}

	if !n.enabled {
		t.Error("Notifier should be enabled")
	}

	if n.from != "noreply@example.com" {
		t.Errorf("from = %s, want noreply@example.com", n.from)
	}
}

func TestNewNotifierDisabled(t *testing.T) {
	config := SMTPConfig{
		Enabled: false,
	}

	n := NewNotifier(config)
	if n.IsEnabled() {
		t.Error("Notifier should not be enabled")
	}
}

func TestNewNotifierDefaultFrom(t *testing.T) {
	config := SMTPConfig{
		Enabled: true,
		Host:    "smtp.example.com",
		Port:    587,
	}

	n := NewNotifier(config)
	if n.from != "noreply@magebox.local" {
		t.Errorf("default from = %s, want noreply@magebox.local", n.from)
	}
}

func TestIsEnabled(t *testing.T) {
	tests := []struct {
		name   string
		config SMTPConfig
		want   bool
	}{
		{
			name: "enabled with host",
			config: SMTPConfig{
				Enabled: true,
				Host:    "smtp.example.com",
			},
			want: true,
		},
		{
			name: "disabled",
			config: SMTPConfig{
				Enabled: false,
				Host:    "smtp.example.com",
			},
			want: false,
		},
		{
			name: "enabled without host",
			config: SMTPConfig{
				Enabled: true,
				Host:    "",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := NewNotifier(tt.config)
			if n.IsEnabled() != tt.want {
				t.Errorf("IsEnabled() = %v, want %v", n.IsEnabled(), tt.want)
			}
		})
	}
}

func TestTemplatesExist(t *testing.T) {
	config := SMTPConfig{Enabled: true, Host: "smtp.example.com"}
	n := NewNotifier(config)

	expectedTemplates := []string{
		"user_invited",
		"user_joined",
		"user_removed",
		"security_alert",
		"access_expiry",
		"admin_notification",
	}

	for _, name := range expectedTemplates {
		if _, ok := n.templates[name]; !ok {
			t.Errorf("Template %s not found", name)
		}
	}
}

func TestSendMethodsReturnNilWhenDisabled(t *testing.T) {
	config := SMTPConfig{Enabled: false}
	n := NewNotifier(config)

	// All send methods should return nil when disabled
	if err := n.SendUserInvited("test@example.com", "test", "dev", "http://localhost", "token123", time.Now()); err != nil {
		t.Errorf("SendUserInvited should return nil when disabled, got %v", err)
	}

	if err := n.SendUserJoined("test@example.com", "test", "dev", nil); err != nil {
		t.Errorf("SendUserJoined should return nil when disabled, got %v", err)
	}

	if err := n.SendUserRemoved("test@example.com", "test"); err != nil {
		t.Errorf("SendUserRemoved should return nil when disabled, got %v", err)
	}

	if err := n.SendSecurityAlert("test@example.com", "Test Alert", "127.0.0.1", "Test details"); err != nil {
		t.Errorf("SendSecurityAlert should return nil when disabled, got %v", err)
	}

	if err := n.SendAccessExpiryWarning("test@example.com", "test", time.Now().AddDate(0, 0, 7), 7); err != nil {
		t.Errorf("SendAccessExpiryWarning should return nil when disabled, got %v", err)
	}

	if err := n.SendAdminNotification("test@example.com", "Test Event", "test", "Test details"); err != nil {
		t.Errorf("SendAdminNotification should return nil when disabled, got %v", err)
	}
}

func TestGetAdminEmails(t *testing.T) {
	users := []User{
		{Name: "admin1", Email: "admin1@example.com", Role: RoleAdmin},
		{Name: "admin2", Email: "admin2@example.com", Role: RoleAdmin},
		{Name: "dev1", Email: "dev1@example.com", Role: RoleDev},
		{Name: "readonly1", Email: "readonly1@example.com", Role: RoleReadonly},
		{Name: "admin_no_email", Role: RoleAdmin}, // No email
	}

	emails := GetAdminEmails(users)

	if len(emails) != 2 {
		t.Errorf("Expected 2 admin emails, got %d", len(emails))
	}

	// Check that both admin emails are present
	found1, found2 := false, false
	for _, e := range emails {
		if e == "admin1@example.com" {
			found1 = true
		}
		if e == "admin2@example.com" {
			found2 = true
		}
	}

	if !found1 || !found2 {
		t.Error("Not all admin emails were returned")
	}
}

func TestSanitizeEmail(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Test@Example.COM", "test@example.com"},
		{"  test@example.com  ", "test@example.com"},
		{"USER@DOMAIN.COM", "user@domain.com"},
	}

	for _, tt := range tests {
		result := SanitizeEmail(tt.input)
		if result != tt.expected {
			t.Errorf("SanitizeEmail(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestNotificationEventString(t *testing.T) {
	events := map[NotificationEvent]string{
		NotifyUserInvited:   "USER_INVITED",
		NotifyUserJoined:    "USER_JOINED",
		NotifyUserRemoved:   "USER_REMOVED",
		NotifySecurityAlert: "SECURITY_ALERT",
		NotifyAccessExpiry:  "ACCESS_EXPIRY",
		NotifyKeyDeployed:   "KEY_DEPLOYED",
		NotifyKeyRemoved:    "KEY_REMOVED",
	}

	for event, expected := range events {
		if event.String() != expected {
			t.Errorf("%v.String() = %s, want %s", event, event.String(), expected)
		}
	}
}

func TestTemplateRendering(t *testing.T) {
	config := SMTPConfig{Enabled: true, Host: "smtp.example.com", Port: 587, From: "noreply@test.com"}
	n := NewNotifier(config)

	t.Run("user_invited template", func(t *testing.T) {
		tmpl := n.templates["user_invited"]
		if tmpl == nil {
			t.Fatal("Template not found")
		}

		data := map[string]interface{}{
			"UserName":    "testuser",
			"Role":        "dev",
			"ServerURL":   "https://magebox.example.com",
			"InviteToken": "abc123",
			"ExpiresAt":   "January 15, 2025 at 12:00 UTC",
		}

		var buf strings.Builder
		if err := tmpl.Execute(&buf, data); err != nil {
			t.Fatalf("Failed to execute template: %v", err)
		}

		html := buf.String()
		if !strings.Contains(html, "testuser") {
			t.Error("Template should contain username")
		}
		if !strings.Contains(html, "dev") {
			t.Error("Template should contain role")
		}
		if !strings.Contains(html, "abc123") {
			t.Error("Template should contain invite token")
		}
	})

	t.Run("security_alert template", func(t *testing.T) {
		tmpl := n.templates["security_alert"]
		if tmpl == nil {
			t.Fatal("Template not found")
		}

		data := map[string]interface{}{
			"AlertType": "IP Lockout",
			"Timestamp": "January 15, 2025 at 12:00:00 UTC",
			"IPAddress": "192.168.1.100",
			"Details":   "IP has been locked out after 5 failed attempts",
		}

		var buf strings.Builder
		if err := tmpl.Execute(&buf, data); err != nil {
			t.Fatalf("Failed to execute template: %v", err)
		}

		html := buf.String()
		if !strings.Contains(html, "IP Lockout") {
			t.Error("Template should contain alert type")
		}
		if !strings.Contains(html, "192.168.1.100") {
			t.Error("Template should contain IP address")
		}
	})
}

// TestConnectionFailure tests that connection failures are handled properly
func TestConnectionFailure(t *testing.T) {
	config := SMTPConfig{
		Enabled: true,
		Host:    "nonexistent.invalid.host",
		Port:    587,
		From:    "test@example.com",
	}

	n := NewNotifier(config)

	err := n.TestConnection()
	if err == nil {
		t.Error("TestConnection should fail for invalid host")
	}
}

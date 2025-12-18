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

func TestNewDeployer(t *testing.T) {
	d := NewDeployer()
	if d == nil {
		t.Fatal("NewDeployer returned nil")
	}
	if d.timeout == 0 {
		t.Error("Deployer timeout should be set")
	}
}

func TestFormatKeyLine(t *testing.T) {
	d := NewDeployer()

	tests := []struct {
		name     string
		userKey  UserKey
		expected string
	}{
		{
			name: "key without comment",
			userKey: UserKey{
				UserName:  "alice",
				PublicKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITest",
			},
			expected: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITest magebox:alice",
		},
		{
			name: "key with existing comment",
			userKey: UserKey{
				UserName:  "bob",
				PublicKey: "ssh-rsa AAAAB3... bob@example.com",
			},
			expected: "ssh-rsa AAAAB3... magebox:bob",
		},
		{
			name: "key with magebox marker already",
			userKey: UserKey{
				UserName:  "charlie",
				PublicKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITest magebox:charlie",
			},
			expected: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITest magebox:charlie",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := d.formatKeyLine(tt.userKey)
			if result != tt.expected {
				t.Errorf("formatKeyLine() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestKeysMatch(t *testing.T) {
	d := NewDeployer()

	tests := []struct {
		name   string
		line1  string
		line2  string
		expect bool
	}{
		{
			name:   "identical keys",
			line1:  "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITest",
			line2:  "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITest",
			expect: true,
		},
		{
			name:   "same key different comments",
			line1:  "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITest user1@host",
			line2:  "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITest magebox:user1",
			expect: true,
		},
		{
			name:   "different keys",
			line1:  "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITest1",
			line2:  "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITest2",
			expect: false,
		},
		{
			name:   "different types",
			line1:  "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITest",
			line2:  "ssh-rsa AAAAC3NzaC1lZDI1NTE5AAAAITest",
			expect: false,
		},
		{
			name:   "empty lines",
			line1:  "",
			line2:  "",
			expect: false,
		},
		{
			name:   "malformed key",
			line1:  "ssh-ed25519",
			line2:  "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITest",
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := d.keysMatch(tt.line1, tt.line2)
			if result != tt.expect {
				t.Errorf("keysMatch(%q, %q) = %v, want %v", tt.line1, tt.line2, result, tt.expect)
			}
		})
	}
}

func TestBuildAuthorizedKeys(t *testing.T) {
	d := NewDeployer()

	tests := []struct {
		name          string
		currentKeys   []string
		newKeys       []UserKey
		expectAdded   int
		expectRemoved int
		expectKeys    []string
	}{
		{
			name:        "empty to new keys",
			currentKeys: []string{},
			newKeys: []UserKey{
				{UserName: "alice", PublicKey: "ssh-ed25519 AAAA1"},
				{UserName: "bob", PublicKey: "ssh-ed25519 AAAA2"},
			},
			expectAdded:   2,
			expectRemoved: 0,
			expectKeys:    []string{"ssh-ed25519 AAAA1 magebox:alice", "ssh-ed25519 AAAA2 magebox:bob"},
		},
		{
			name: "preserve unmanaged keys",
			currentKeys: []string{
				"ssh-rsa UNMANAGED user@external",
			},
			newKeys: []UserKey{
				{UserName: "alice", PublicKey: "ssh-ed25519 AAAA1"},
			},
			expectAdded:   1,
			expectRemoved: 0,
			expectKeys:    []string{"ssh-rsa UNMANAGED user@external", "ssh-ed25519 AAAA1 magebox:alice"},
		},
		{
			name: "remove old managed key",
			currentKeys: []string{
				"ssh-ed25519 AAAA1 magebox:alice",
				"ssh-ed25519 AAAA2 magebox:bob",
			},
			newKeys: []UserKey{
				{UserName: "alice", PublicKey: "ssh-ed25519 AAAA1"},
			},
			expectAdded:   0,
			expectRemoved: 1,
			expectKeys:    []string{"ssh-ed25519 AAAA1 magebox:alice"},
		},
		{
			name: "mixed managed and unmanaged",
			currentKeys: []string{
				"ssh-rsa UNMANAGED external@host",
				"ssh-ed25519 OLD magebox:olduser",
			},
			newKeys: []UserKey{
				{UserName: "newuser", PublicKey: "ssh-ed25519 NEW"},
			},
			expectAdded:   1,
			expectRemoved: 1,
			expectKeys:    []string{"ssh-rsa UNMANAGED external@host", "ssh-ed25519 NEW magebox:newuser"},
		},
		{
			name: "no changes",
			currentKeys: []string{
				"ssh-ed25519 AAAA1 magebox:alice",
			},
			newKeys: []UserKey{
				{UserName: "alice", PublicKey: "ssh-ed25519 AAAA1"},
			},
			expectAdded:   0,
			expectRemoved: 0,
			expectKeys:    []string{"ssh-ed25519 AAAA1 magebox:alice"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, added, removed := d.buildAuthorizedKeys(tt.currentKeys, tt.newKeys)

			if added != tt.expectAdded {
				t.Errorf("added = %d, want %d", added, tt.expectAdded)
			}
			if removed != tt.expectRemoved {
				t.Errorf("removed = %d, want %d", removed, tt.expectRemoved)
			}

			// Check content contains expected keys
			for _, key := range tt.expectKeys {
				if !strings.Contains(content, key) {
					t.Errorf("content missing expected key: %s", key)
				}
			}
		})
	}
}

func TestDeployResult(t *testing.T) {
	result := &DeployResult{
		Environment: "production",
		Success:     true,
		Message:     "Synced 5 keys",
		KeysAdded:   3,
		KeysRemoved: 2,
	}

	if result.Environment != "production" {
		t.Errorf("Environment = %s, want production", result.Environment)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
	if result.KeysAdded != 3 {
		t.Errorf("KeysAdded = %d, want 3", result.KeysAdded)
	}
	if result.KeysRemoved != 2 {
		t.Errorf("KeysRemoved = %d, want 2", result.KeysRemoved)
	}
}

func TestUserKey(t *testing.T) {
	key := UserKey{
		UserName:  "developer",
		PublicKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExample developer@example.com",
	}

	if key.UserName != "developer" {
		t.Errorf("UserName = %s, want developer", key.UserName)
	}
	if !strings.HasPrefix(key.PublicKey, "ssh-ed25519") {
		t.Error("PublicKey should start with ssh-ed25519")
	}
}

func TestUserHasProjectAccess(t *testing.T) {
	user := &User{
		Name:     "alice",
		Projects: []string{"project-a", "project-b"},
	}

	tests := []struct {
		project string
		expect  bool
	}{
		{"project-a", true},
		{"project-b", true},
		{"project-c", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.project, func(t *testing.T) {
			result := user.HasProjectAccess(tt.project)
			if result != tt.expect {
				t.Errorf("HasProjectAccess(%s) = %v, want %v", tt.project, result, tt.expect)
			}
		})
	}
}

func TestEnvironmentFullName(t *testing.T) {
	env := &Environment{
		Name:    "production",
		Project: "myproject",
	}

	expected := "myproject/production"
	if env.FullName() != expected {
		t.Errorf("FullName() = %s, want %s", env.FullName(), expected)
	}
}

// Integration tests require actual SSH connectivity
// These are run via Docker integration tests in test/integration/teamserver/

func TestBuildAuthorizedKeysContent(t *testing.T) {
	d := NewDeployer()

	// Test that content ends with newline when non-empty
	content, _, _ := d.buildAuthorizedKeys([]string{}, []UserKey{
		{UserName: "alice", PublicKey: "ssh-ed25519 AAAA1"},
	})

	if content != "" && !strings.HasSuffix(content, "\n") {
		t.Error("Non-empty content should end with newline")
	}

	// Test empty content
	emptyContent, _, _ := d.buildAuthorizedKeys([]string{}, []UserKey{})
	if emptyContent != "" {
		t.Errorf("Empty keys should produce empty content, got %q", emptyContent)
	}
}

func TestFormatKeyLineEdgeCases(t *testing.T) {
	d := NewDeployer()

	// Test with whitespace
	key := UserKey{
		UserName:  "alice",
		PublicKey: "  ssh-ed25519 AAAA1  ",
	}
	result := d.formatKeyLine(key)
	if strings.HasPrefix(result, " ") {
		t.Error("Result should not have leading whitespace")
	}

	// Test with only key type (malformed)
	malformed := UserKey{
		UserName:  "bob",
		PublicKey: "ssh-ed25519",
	}
	malformedResult := d.formatKeyLine(malformed)
	// Should return original if can't parse
	if malformedResult == "" {
		t.Error("Should return something for malformed key")
	}
}

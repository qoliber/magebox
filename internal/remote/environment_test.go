// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package remote

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnvironment_Validate(t *testing.T) {
	tests := []struct {
		name    string
		env     Environment
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid environment",
			env: Environment{
				Name: "staging",
				User: "deploy",
				Host: "staging.example.com",
				Port: 22,
			},
			wantErr: false,
		},
		{
			name: "valid environment with custom port",
			env: Environment{
				Name: "production",
				User: "magento",
				Host: "prod.example.com",
				Port: 2222,
			},
			wantErr: false,
		},
		{
			name: "missing name",
			env: Environment{
				User: "deploy",
				Host: "staging.example.com",
			},
			wantErr: true,
			errMsg:  "environment name is required",
		},
		{
			name: "missing user",
			env: Environment{
				Name: "staging",
				Host: "staging.example.com",
			},
			wantErr: true,
			errMsg:  "user is required",
		},
		{
			name: "missing host",
			env: Environment{
				Name: "staging",
				User: "deploy",
			},
			wantErr: true,
			errMsg:  "host is required",
		},
		{
			name: "invalid port negative",
			env: Environment{
				Name: "staging",
				User: "deploy",
				Host: "staging.example.com",
				Port: -1,
			},
			wantErr: true,
			errMsg:  "port must be between 0 and 65535",
		},
		{
			name: "invalid port too high",
			env: Environment{
				Name: "staging",
				User: "deploy",
				Host: "staging.example.com",
				Port: 70000,
			},
			wantErr: true,
			errMsg:  "port must be between 0 and 65535",
		},
		{
			name: "non-existent SSH key",
			env: Environment{
				Name:       "staging",
				User:       "deploy",
				Host:       "staging.example.com",
				SSHKeyPath: "/nonexistent/path/to/key",
			},
			wantErr: true,
			errMsg:  "SSH key file not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.env.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if err.Error() != tt.errMsg && !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want error containing %q", err, tt.errMsg)
				}
			}
		})
	}
}

func TestEnvironment_ValidateWithRealSSHKey(t *testing.T) {
	// Create a temporary SSH key file for testing
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test_key")
	if err := os.WriteFile(keyPath, []byte("fake-key-content"), 0600); err != nil {
		t.Fatalf("failed to create test key file: %v", err)
	}

	env := Environment{
		Name:       "staging",
		User:       "deploy",
		Host:       "staging.example.com",
		SSHKeyPath: keyPath,
	}

	if err := env.Validate(); err != nil {
		t.Errorf("Validate() with real SSH key file failed: %v", err)
	}
}

func TestEnvironment_GetPort(t *testing.T) {
	tests := []struct {
		name     string
		port     int
		wantPort int
	}{
		{"default port when zero", 0, DefaultPort},
		{"custom port", 2222, 2222},
		{"standard SSH port", 22, 22},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := Environment{Port: tt.port}
			if got := env.GetPort(); got != tt.wantPort {
				t.Errorf("GetPort() = %v, want %v", got, tt.wantPort)
			}
		})
	}
}

func TestEnvironment_GetConnectionString(t *testing.T) {
	tests := []struct {
		name string
		env  Environment
		want string
	}{
		{
			name: "basic connection",
			env: Environment{
				User: "deploy",
				Host: "staging.example.com",
			},
			want: "deploy@staging.example.com",
		},
		{
			name: "with custom port",
			env: Environment{
				User: "deploy",
				Host: "staging.example.com",
				Port: 2222,
			},
			want: "deploy@staging.example.com:2222",
		},
		{
			name: "with SSH key",
			env: Environment{
				User:       "deploy",
				Host:       "staging.example.com",
				SSHKeyPath: "/path/to/key",
			},
			want: "deploy@staging.example.com (key: /path/to/key)",
		},
		{
			name: "with custom SSH command",
			env: Environment{
				User:       "deploy",
				Host:       "staging.example.com",
				SSHCommand: "ssh -J jump@bastion deploy@internal",
			},
			want: "deploy@staging.example.com (custom command)",
		},
		{
			name: "with port and key",
			env: Environment{
				User:       "deploy",
				Host:       "staging.example.com",
				Port:       2222,
				SSHKeyPath: "/path/to/key",
			},
			want: "deploy@staging.example.com:2222 (key: /path/to/key)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.env.GetConnectionString(); got != tt.want {
				t.Errorf("GetConnectionString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEnvironment_BuildSSHCommand(t *testing.T) {
	tests := []struct {
		name     string
		env      Environment
		wantPath string
		wantArgs []string
	}{
		{
			name: "basic SSH command",
			env: Environment{
				User: "deploy",
				Host: "staging.example.com",
			},
			wantPath: "ssh",
			wantArgs: []string{"ssh", "deploy@staging.example.com"},
		},
		{
			name: "with custom port",
			env: Environment{
				User: "deploy",
				Host: "staging.example.com",
				Port: 2222,
			},
			wantPath: "ssh",
			wantArgs: []string{"ssh", "-p", "2222", "deploy@staging.example.com"},
		},
		{
			name: "with SSH key",
			env: Environment{
				User:       "deploy",
				Host:       "staging.example.com",
				SSHKeyPath: "/path/to/key",
			},
			wantPath: "ssh",
			wantArgs: []string{"ssh", "-i", "/path/to/key", "deploy@staging.example.com"},
		},
		{
			name: "with port and key",
			env: Environment{
				User:       "deploy",
				Host:       "staging.example.com",
				Port:       2222,
				SSHKeyPath: "/path/to/key",
			},
			wantPath: "ssh",
			wantArgs: []string{"ssh", "-i", "/path/to/key", "-p", "2222", "deploy@staging.example.com"},
		},
		{
			name: "custom SSH command",
			env: Environment{
				SSHCommand: "ssh -J jump@bastion deploy@internal",
			},
			wantPath: "sh",
			wantArgs: []string{"sh", "-c", "ssh -J jump@bastion deploy@internal"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.env.BuildSSHCommand()

			// Check command path contains expected binary
			if !contains(cmd.Path, tt.wantPath) {
				t.Errorf("BuildSSHCommand() path = %q, want to contain %q", cmd.Path, tt.wantPath)
			}

			// Check args
			if len(cmd.Args) != len(tt.wantArgs) {
				t.Errorf("BuildSSHCommand() args length = %d, want %d", len(cmd.Args), len(tt.wantArgs))
				t.Errorf("Got args: %v", cmd.Args)
				t.Errorf("Want args: %v", tt.wantArgs)
				return
			}

			for i, arg := range tt.wantArgs {
				if cmd.Args[i] != arg {
					t.Errorf("BuildSSHCommand() args[%d] = %q, want %q", i, cmd.Args[i], arg)
				}
			}
		})
	}
}

func TestManager_AddAndGet(t *testing.T) {
	mgr := NewManager(nil)

	env := Environment{
		Name: "staging",
		User: "deploy",
		Host: "staging.example.com",
		Port: 22,
	}

	// Add environment
	if err := mgr.Add(env); err != nil {
		t.Fatalf("Add() failed: %v", err)
	}

	// Get environment
	got, err := mgr.Get("staging")
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	if got.Name != env.Name || got.User != env.User || got.Host != env.Host {
		t.Errorf("Get() returned wrong environment: got %+v, want %+v", got, env)
	}
}

func TestManager_AddDuplicate(t *testing.T) {
	mgr := NewManager(nil)

	env := Environment{
		Name: "staging",
		User: "deploy",
		Host: "staging.example.com",
	}

	// Add first time
	if err := mgr.Add(env); err != nil {
		t.Fatalf("Add() first time failed: %v", err)
	}

	// Add duplicate
	err := mgr.Add(env)
	if err == nil {
		t.Error("Add() duplicate should fail")
	}
	if !contains(err.Error(), "already exists") {
		t.Errorf("Add() duplicate error = %v, want error containing 'already exists'", err)
	}
}

func TestManager_Remove(t *testing.T) {
	envs := []Environment{
		{Name: "staging", User: "deploy", Host: "staging.example.com"},
		{Name: "production", User: "deploy", Host: "prod.example.com"},
	}
	mgr := NewManager(envs)

	// Remove staging
	if err := mgr.Remove("staging"); err != nil {
		t.Fatalf("Remove() failed: %v", err)
	}

	// Verify staging is gone
	_, err := mgr.Get("staging")
	if err == nil {
		t.Error("Get() should fail after Remove()")
	}

	// Verify production still exists
	_, err = mgr.Get("production")
	if err != nil {
		t.Errorf("Get() production should still exist: %v", err)
	}

	// Verify list has one item
	list := mgr.List()
	if len(list) != 1 {
		t.Errorf("List() length = %d, want 1", len(list))
	}
}

func TestManager_RemoveNotFound(t *testing.T) {
	mgr := NewManager(nil)

	err := mgr.Remove("nonexistent")
	if err == nil {
		t.Error("Remove() nonexistent should fail")
	}
	if !contains(err.Error(), "not found") {
		t.Errorf("Remove() error = %v, want error containing 'not found'", err)
	}
}

func TestManager_Update(t *testing.T) {
	envs := []Environment{
		{Name: "staging", User: "deploy", Host: "old.example.com"},
	}
	mgr := NewManager(envs)

	// Update environment
	updated := Environment{
		Name: "staging",
		User: "newuser",
		Host: "new.example.com",
		Port: 2222,
	}

	if err := mgr.Update(updated); err != nil {
		t.Fatalf("Update() failed: %v", err)
	}

	// Verify update
	got, err := mgr.Get("staging")
	if err != nil {
		t.Fatalf("Get() after update failed: %v", err)
	}

	if got.User != "newuser" || got.Host != "new.example.com" || got.Port != 2222 {
		t.Errorf("Update() didn't apply: got %+v, want %+v", got, updated)
	}
}

func TestManager_UpdateNotFound(t *testing.T) {
	mgr := NewManager(nil)

	env := Environment{
		Name: "nonexistent",
		User: "deploy",
		Host: "example.com",
	}

	err := mgr.Update(env)
	if err == nil {
		t.Error("Update() nonexistent should fail")
	}
	if !contains(err.Error(), "not found") {
		t.Errorf("Update() error = %v, want error containing 'not found'", err)
	}
}

func TestManager_List(t *testing.T) {
	envs := []Environment{
		{Name: "staging", User: "deploy", Host: "staging.example.com"},
		{Name: "production", User: "deploy", Host: "prod.example.com"},
		{Name: "dev", User: "developer", Host: "dev.example.com"},
	}
	mgr := NewManager(envs)

	list := mgr.List()
	if len(list) != 3 {
		t.Errorf("List() length = %d, want 3", len(list))
	}
}

func TestManager_GetEnvironments(t *testing.T) {
	envs := []Environment{
		{Name: "staging", User: "deploy", Host: "staging.example.com"},
	}
	mgr := NewManager(envs)

	got := mgr.GetEnvironments()
	if len(got) != 1 {
		t.Errorf("GetEnvironments() length = %d, want 1", len(got))
	}
	if got[0].Name != "staging" {
		t.Errorf("GetEnvironments()[0].Name = %q, want %q", got[0].Name, "staging")
	}
}

func TestManager_GetNotFound(t *testing.T) {
	mgr := NewManager(nil)

	_, err := mgr.Get("nonexistent")
	if err == nil {
		t.Error("Get() nonexistent should fail")
	}
	if !contains(err.Error(), "not found") {
		t.Errorf("Get() error = %v, want error containing 'not found'", err)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

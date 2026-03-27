package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDomain_GetRoot(t *testing.T) {
	tests := []struct {
		name     string
		domain   Domain
		expected string
	}{
		{
			name:     "default root when empty",
			domain:   Domain{Host: "test.test"},
			expected: "pub",
		},
		{
			name:     "custom root",
			domain:   Domain{Host: "test.test", Root: "public"},
			expected: "public",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.domain.GetRoot(); got != tt.expected {
				t.Errorf("GetRoot() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDomain_IsSSLEnabled(t *testing.T) {
	boolPtr := func(b bool) *bool { return &b }

	tests := []struct {
		name     string
		domain   Domain
		expected bool
	}{
		{
			name:     "default SSL enabled when nil",
			domain:   Domain{Host: "test.test"},
			expected: true,
		},
		{
			name:     "SSL explicitly enabled",
			domain:   Domain{Host: "test.test", SSL: boolPtr(true)},
			expected: true,
		},
		{
			name:     "SSL explicitly disabled",
			domain:   Domain{Host: "test.test", SSL: boolPtr(false)},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.domain.IsSSLEnabled(); got != tt.expected {
				t.Errorf("IsSSLEnabled() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
		errorField  string
	}{
		{
			name: "valid config",
			config: Config{
				Name:    "mystore",
				Domains: []Domain{{Host: "mystore.test"}},
				PHP:     "8.2",
			},
			expectError: false,
		},
		{
			name: "missing name",
			config: Config{
				Domains: []Domain{{Host: "mystore.test"}},
				PHP:     "8.2",
			},
			expectError: true,
			errorField:  "name",
		},
		{
			name: "missing domains",
			config: Config{
				Name: "mystore",
				PHP:  "8.2",
			},
			expectError: true,
			errorField:  "domains",
		},
		{
			name: "empty domains",
			config: Config{
				Name:    "mystore",
				Domains: []Domain{},
				PHP:     "8.2",
			},
			expectError: true,
			errorField:  "domains",
		},
		{
			name: "domain without host",
			config: Config{
				Name:    "mystore",
				Domains: []Domain{{Root: "pub"}},
				PHP:     "8.2",
			},
			expectError: true,
			errorField:  "domains",
		},
		{
			name: "missing php",
			config: Config{
				Name:    "mystore",
				Domains: []Domain{{Host: "mystore.test"}},
			},
			expectError: true,
			errorField:  "php",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				if ve, ok := err.(*ValidationError); ok {
					if ve.Field != tt.errorField {
						t.Errorf("expected error field %s, got %s", tt.errorField, ve.Field)
					}
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestServiceConfig_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name            string
		yaml            string
		expectedEnabled bool
		expectedVersion string
		expectedPort    int
	}{
		{
			name:            "version string",
			yaml:            `"8.0"`,
			expectedEnabled: true,
			expectedVersion: "8.0",
		},
		{
			name:            "boolean true",
			yaml:            `true`,
			expectedEnabled: true,
			expectedVersion: "",
		},
		{
			name:            "boolean false",
			yaml:            `false`,
			expectedEnabled: false,
			expectedVersion: "",
		},
		{
			name: "object with version",
			yaml: `
version: "8.0"
port: 3307`,
			expectedEnabled: true,
			expectedVersion: "8.0",
			expectedPort:    3307,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sc ServiceConfig
			err := yaml.Unmarshal([]byte(tt.yaml), &sc)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if sc.Enabled != tt.expectedEnabled {
				t.Errorf("Enabled = %v, want %v", sc.Enabled, tt.expectedEnabled)
			}
			if sc.Version != tt.expectedVersion {
				t.Errorf("Version = %v, want %v", sc.Version, tt.expectedVersion)
			}
			if sc.Port != tt.expectedPort {
				t.Errorf("Port = %v, want %v", sc.Port, tt.expectedPort)
			}
		})
	}
}

func TestServices_HasMethods(t *testing.T) {
	enabled := &ServiceConfig{Enabled: true, Version: "8.0"}
	disabled := &ServiceConfig{Enabled: false}

	tests := []struct {
		name     string
		services Services
		method   func(*Services) bool
		expected bool
	}{
		{
			name:     "HasMySQL with enabled MySQL",
			services: Services{MySQL: enabled},
			method:   (*Services).HasMySQL,
			expected: true,
		},
		{
			name:     "HasMySQL with disabled MySQL",
			services: Services{MySQL: disabled},
			method:   (*Services).HasMySQL,
			expected: false,
		},
		{
			name:     "HasMySQL with nil",
			services: Services{},
			method:   (*Services).HasMySQL,
			expected: false,
		},
		{
			name:     "HasRedis with enabled Redis",
			services: Services{Redis: enabled},
			method:   (*Services).HasRedis,
			expected: true,
		},
		{
			name:     "HasOpenSearch with enabled OpenSearch",
			services: Services{OpenSearch: enabled},
			method:   (*Services).HasOpenSearch,
			expected: true,
		},
		{
			name:     "HasValkey with enabled Valkey",
			services: Services{Valkey: enabled},
			method:   (*Services).HasValkey,
			expected: true,
		},
		{
			name:     "HasValkey with nil",
			services: Services{},
			method:   (*Services).HasValkey,
			expected: false,
		},
		{
			name:     "HasCacheService with Redis",
			services: Services{Redis: enabled},
			method:   (*Services).HasCacheService,
			expected: true,
		},
		{
			name:     "HasCacheService with Valkey",
			services: Services{Valkey: enabled},
			method:   (*Services).HasCacheService,
			expected: true,
		},
		{
			name:     "HasCacheService with neither",
			services: Services{},
			method:   (*Services).HasCacheService,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.method(&tt.services); got != tt.expected {
				t.Errorf("got %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPullConfig_GetMagerun(t *testing.T) {
	tests := []struct {
		name string
		cfg  *PullConfig
		want string
	}{
		{"nil config", nil, "magerun2"},
		{"empty magerun", &PullConfig{}, "magerun2"},
		{"custom magerun", &PullConfig{Magerun: "n98-magerun2"}, "n98-magerun2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.GetMagerun(); got != tt.want {
				t.Errorf("GetMagerun() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPullConfig_GetStrip(t *testing.T) {
	tests := []struct {
		name string
		cfg  *PullConfig
		want string
	}{
		{"nil config", nil, "@stripped"},
		{"empty strip", &PullConfig{}, "@stripped"},
		{"custom strip", &PullConfig{Strip: "@stripped @trade @search"}, "@stripped @trade @search"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.GetStrip(); got != tt.want {
				t.Errorf("GetStrip() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPullConfig_GetRootPath(t *testing.T) {
	tests := []struct {
		name string
		cfg  *PullConfig
		want string
	}{
		{"nil config", nil, ""},
		{"empty root_path", &PullConfig{}, ""},
		{"custom root_path", &PullConfig{RootPath: "/data/web/current/"}, "/data/web/current/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.GetRootPath(); got != tt.want {
				t.Errorf("GetRootPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPullConfig_YAMLParsing(t *testing.T) {
	yamlData := `
pull:
  default: staging
  strip: "@stripped @trade @search"
  exclude:
    - custom_log
    - temp_import
  magerun: magerun2
  root_path: /data/web/project/current/
`
	var cfg struct {
		Pull *PullConfig `yaml:"pull"`
	}
	if err := yaml.Unmarshal([]byte(yamlData), &cfg); err != nil {
		t.Fatalf("failed to parse YAML: %v", err)
	}

	if cfg.Pull == nil {
		t.Fatal("Pull config should not be nil")
	}
	if cfg.Pull.Default != "staging" {
		t.Errorf("Default = %q, want %q", cfg.Pull.Default, "staging")
	}
	if cfg.Pull.GetStrip() != "@stripped @trade @search" {
		t.Errorf("Strip = %q, want %q", cfg.Pull.GetStrip(), "@stripped @trade @search")
	}
	if len(cfg.Pull.Exclude) != 2 {
		t.Errorf("Exclude length = %d, want 2", len(cfg.Pull.Exclude))
	}
	if cfg.Pull.GetMagerun() != "magerun2" {
		t.Errorf("Magerun = %q, want %q", cfg.Pull.GetMagerun(), "magerun2")
	}
	if cfg.Pull.GetRootPath() != "/data/web/project/current/" {
		t.Errorf("RootPath = %q, want %q", cfg.Pull.GetRootPath(), "/data/web/project/current/")
	}
}

func TestPullConfig_BuildStripArgument(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *PullConfig
		wantBase string
	}{
		{
			name:     "nil config uses default strip",
			cfg:      nil,
			wantBase: "@stripped",
		},
		{
			name:     "custom strip groups",
			cfg:      &PullConfig{Strip: "@stripped @trade @search"},
			wantBase: "@stripped @trade @search",
		},
		{
			name: "strip with exclude tables appended",
			cfg: &PullConfig{
				Strip:   "@stripped @trade",
				Exclude: []string{"custom_log", "temp_data"},
			},
			wantBase: "@stripped @trade",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.GetStrip()
			if got != tt.wantBase {
				t.Errorf("GetStrip() = %q, want %q", got, tt.wantBase)
			}

			// Simulate the exclude append logic from db_pull.go
			if tt.cfg != nil && len(tt.cfg.Exclude) > 0 {
				combined := got + " " + tt.cfg.Exclude[0]
				for _, tbl := range tt.cfg.Exclude[1:] {
					combined += " " + tbl
				}
				if combined != "@stripped @trade custom_log temp_data" {
					t.Errorf("combined strip = %q, want %q", combined, "@stripped @trade custom_log temp_data")
				}
			}
		})
	}
}

func TestServices_GetDatabaseService(t *testing.T) {
	mysql := &ServiceConfig{Enabled: true, Version: "8.0"}
	mariadb := &ServiceConfig{Enabled: true, Version: "10.6"}

	tests := []struct {
		name            string
		services        Services
		expectedNil     bool
		expectedVersion string
	}{
		{
			name:            "MySQL configured",
			services:        Services{MySQL: mysql},
			expectedNil:     false,
			expectedVersion: "8.0",
		},
		{
			name:            "MariaDB configured",
			services:        Services{MariaDB: mariadb},
			expectedNil:     false,
			expectedVersion: "10.6",
		},
		{
			name:            "MySQL takes precedence over MariaDB",
			services:        Services{MySQL: mysql, MariaDB: mariadb},
			expectedNil:     false,
			expectedVersion: "8.0",
		},
		{
			name:        "no database configured",
			services:    Services{},
			expectedNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.services.GetDatabaseService()
			if tt.expectedNil {
				if got != nil {
					t.Errorf("expected nil, got %v", got)
				}
				return
			}
			if got == nil {
				t.Error("expected non-nil, got nil")
				return
			}
			if got.Version != tt.expectedVersion {
				t.Errorf("Version = %v, want %v", got.Version, tt.expectedVersion)
			}
		})
	}
}

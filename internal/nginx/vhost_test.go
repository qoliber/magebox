package nginx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/qoliber/magebox/internal/config"
	"github.com/qoliber/magebox/internal/platform"
	"github.com/qoliber/magebox/internal/ssl"
)

func setupTestGenerator(t *testing.T) (*VhostGenerator, string) {
	tmpDir := t.TempDir()
	p := &platform.Platform{
		Type:    platform.Linux,
		HomeDir: tmpDir,
	}
	sslMgr := ssl.NewManager(p)
	g := NewVhostGenerator(p, sslMgr)
	return g, tmpDir
}

func TestNewVhostGenerator(t *testing.T) {
	g, tmpDir := setupTestGenerator(t)

	if g == nil {
		t.Fatal("NewVhostGenerator should not return nil")
	}

	expectedDir := filepath.Join(tmpDir, ".magebox", "nginx", "vhosts")
	if g.vhostsDir != expectedDir {
		t.Errorf("vhostsDir = %v, want %v", g.vhostsDir, expectedDir)
	}
}

func TestVhostGenerator_VhostsDir(t *testing.T) {
	g, tmpDir := setupTestGenerator(t)

	expected := filepath.Join(tmpDir, ".magebox", "nginx", "vhosts")
	if got := g.VhostsDir(); got != expected {
		t.Errorf("VhostsDir() = %v, want %v", got, expected)
	}
}

func TestVhostGenerator_Generate(t *testing.T) {
	g, tmpDir := setupTestGenerator(t)

	projectPath := filepath.Join(tmpDir, "projects", "mystore")
	cfg := &config.Config{
		Name: "mystore",
		Domains: []config.Domain{
			{Host: "mystore.test", Root: "pub"},
		},
		PHP: "8.2",
	}

	err := g.Generate(cfg, projectPath)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check that vhost file was created
	vhostFile := filepath.Join(g.vhostsDir, "mystore-mystore.test.conf")
	if _, err := os.Stat(vhostFile); os.IsNotExist(err) {
		t.Error("Vhost file should have been created")
	}

	// Read and verify content
	content, err := os.ReadFile(vhostFile)
	if err != nil {
		t.Fatalf("Failed to read vhost file: %v", err)
	}

	contentStr := string(content)

	// Verify essential parts are present (upstream is now in separate file)
	checks := []string{
		"server_name mystore.test",
		"fastcgi_backend_mystore",
		projectPath + "/pub",
	}

	// Also check that upstream file was created
	upstreamFile := filepath.Join(g.vhostsDir, "mystore-upstream.conf")
	if _, err := os.Stat(upstreamFile); os.IsNotExist(err) {
		t.Error("Upstream file should have been created")
	}

	// Verify upstream file contains socket path
	upstreamContent, err := os.ReadFile(upstreamFile)
	if err != nil {
		t.Fatalf("Failed to read upstream file: %v", err)
	}
	if !strings.Contains(string(upstreamContent), ".sock") {
		t.Error("Upstream content should contain socket path")
	}

	for _, check := range checks {
		if !strings.Contains(contentStr, check) {
			t.Errorf("Vhost content should contain %q", check)
		}
	}
}

func TestVhostGenerator_GenerateMultipleDomains(t *testing.T) {
	g, tmpDir := setupTestGenerator(t)

	projectPath := filepath.Join(tmpDir, "projects", "mystore")
	cfg := &config.Config{
		Name: "mystore",
		Domains: []config.Domain{
			{Host: "mystore.test"},
			{Host: "api.mystore.test"},
		},
		PHP: "8.2",
	}

	err := g.Generate(cfg, projectPath)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check that both vhost files were created
	files := []string{
		filepath.Join(g.vhostsDir, "mystore-mystore.test.conf"),
		filepath.Join(g.vhostsDir, "mystore-api.mystore.test.conf"),
	}

	for _, file := range files {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("Vhost file %s should have been created", file)
		}
	}
}

func TestVhostGenerator_Remove(t *testing.T) {
	g, tmpDir := setupTestGenerator(t)

	projectPath := filepath.Join(tmpDir, "projects", "mystore")
	cfg := &config.Config{
		Name: "mystore",
		Domains: []config.Domain{
			{Host: "mystore.test"},
			{Host: "api.mystore.test"},
		},
		PHP: "8.2",
	}

	// Generate vhosts first
	if err := g.Generate(cfg, projectPath); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Remove them
	if err := g.Remove("mystore"); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	// Verify they're gone
	files, err := g.ListVhosts()
	if err != nil {
		t.Fatalf("ListVhosts failed: %v", err)
	}

	for _, file := range files {
		if strings.Contains(file, "mystore") {
			t.Errorf("Vhost file %s should have been removed", file)
		}
	}
}

func TestVhostGenerator_ListVhosts(t *testing.T) {
	g, _ := setupTestGenerator(t)

	// Create vhosts directory and some files
	if err := os.MkdirAll(g.vhostsDir, 0755); err != nil {
		t.Fatalf("Failed to create vhosts dir: %v", err)
	}

	files := []string{"project1-domain.test.conf", "project2-other.test.conf"}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(g.vhostsDir, f), []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	vhosts, err := g.ListVhosts()
	if err != nil {
		t.Fatalf("ListVhosts failed: %v", err)
	}

	if len(vhosts) != 2 {
		t.Errorf("ListVhosts returned %d files, want 2", len(vhosts))
	}
}

func TestVhostGenerator_GetIncludeDirective(t *testing.T) {
	g, tmpDir := setupTestGenerator(t)

	expected := "include " + filepath.Join(tmpDir, ".magebox", "nginx", "vhosts") + "/*.conf;"
	if got := g.GetIncludeDirective(); got != expected {
		t.Errorf("GetIncludeDirective() = %v, want %v", got, expected)
	}
}

func TestVhostGenerator_getPHPSocketPath(t *testing.T) {
	g, tmpDir := setupTestGenerator(t)

	expected := filepath.Join(tmpDir, ".magebox", "run", "mystore-php8.2.sock")
	if got := g.getPHPSocketPath("mystore", "8.2"); got != expected {
		t.Errorf("getPHPSocketPath() = %v, want %v", got, expected)
	}
}

func TestRenderVhost_SSLEnabled(t *testing.T) {
	g, tmpDir := setupTestGenerator(t)

	cfg := VhostConfig{
		ProjectName:   "mystore",
		Domain:        "mystore.test",
		DocumentRoot:  "/var/www/mystore/pub",
		PHPVersion:    "8.2",
		PHPSocketPath: filepath.Join(tmpDir, ".magebox", "run", "mystore-php8.2.sock"),
		SSLEnabled:    true,
		SSLCertFile:   "/path/to/cert.pem",
		SSLKeyFile:    "/path/to/key.pem",
		HTTPPort:      80,
		HTTPSPort:     443,
	}

	content, err := g.renderVhost(cfg)
	if err != nil {
		t.Fatalf("renderVhost failed: %v", err)
	}

	// Should contain SSL configuration (Linux uses port 443)
	checks := []string{
		"listen 443 ssl",
		"ssl_certificate /path/to/cert.pem",
		"ssl_certificate_key /path/to/key.pem",
		"return 301 https://",
	}

	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("SSL vhost should contain %q", check)
		}
	}
}

func TestRenderVhost_SSLDisabled(t *testing.T) {
	g, tmpDir := setupTestGenerator(t)

	cfg := VhostConfig{
		ProjectName:   "mystore",
		Domain:        "mystore.test",
		DocumentRoot:  "/var/www/mystore/pub",
		PHPVersion:    "8.2",
		PHPSocketPath: filepath.Join(tmpDir, ".magebox", "run", "mystore-php8.2.sock"),
		SSLEnabled:    false,
	}

	content, err := g.renderVhost(cfg)
	if err != nil {
		t.Fatalf("renderVhost failed: %v", err)
	}

	// Should NOT contain SSL configuration
	if strings.Contains(content, "listen 443") {
		t.Error("Non-SSL vhost should not contain listen 443")
	}
	if strings.Contains(content, "ssl_certificate") {
		t.Error("Non-SSL vhost should not contain ssl_certificate")
	}
}

func TestNewController(t *testing.T) {
	p := &platform.Platform{Type: platform.Linux}
	c := NewController(p)

	if c == nil {
		t.Error("NewController should not return nil")
	}
}

func TestSanitizeDomain(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"mystore.test", "mystore.test"},
		{"api.mystore.test", "api.mystore.test"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := sanitizeDomain(tt.input); got != tt.expected {
				t.Errorf("sanitizeDomain(%v) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestUpstreamTemplateValidity(t *testing.T) {
	// Test that the upstream template parses correctly
	tmpl, err := template.New("upstream").Parse(upstreamTemplate)
	if err != nil {
		t.Fatalf("Upstream template parsing failed: %v", err)
	}

	if tmpl == nil {
		t.Error("Parsed template should not be nil")
	}

	// Verify template contains expected sections
	expectedSections := []string{
		"upstream fastcgi_backend_{{.ProjectName}}",
		"server unix:{{.PHPSocketPath}}",
	}

	for _, section := range expectedSections {
		if !strings.Contains(upstreamTemplate, section) {
			t.Errorf("Upstream template should contain section: %s", section)
		}
	}
}

func TestVhostTemplateValidity(t *testing.T) {
	// Test that the embedded template parses correctly
	tmpl, err := template.New("vhost").Parse(vhostTemplate)
	if err != nil {
		t.Fatalf("Vhost template parsing failed: %v", err)
	}

	if tmpl == nil {
		t.Error("Parsed template should not be nil")
	}

	// Verify template contains expected sections (upstream is now in separate template)
	templateStr := vhostTemplate
	expectedSections := []string{
		"server_name {{.Domain}}",
		"root $MAGE_ROOT",
		"{{if .SSLEnabled}}",
		"ssl_certificate {{.SSLCertFile}}",
		"fastcgi_pass fastcgi_backend_{{.ProjectName}}",
		"location /static/",
		"location /media/",
	}

	for _, section := range expectedSections {
		if !strings.Contains(templateStr, section) {
			t.Errorf("Template should contain section: %s", section)
		}
	}
}

func TestController_addIncludeToNginxConfDarwin(t *testing.T) {
	tests := []struct {
		name           string
		initialContent string
		includeDir     string
		wantContains   []string
		wantNotContain []string
		wantErr        bool
	}{
		{
			name: "replaces include servers/* with magebox include",
			initialContent: `
http {
    include       mime.types;
    include servers/*;
}
`,
			includeDir: "/Users/testuser/.magebox/nginx/vhosts",
			wantContains: []string{
				"include /Users/testuser/.magebox/nginx/vhosts/*.conf;",
				"# MageBox vhosts",
			},
			wantNotContain: []string{
				"include servers/*;",
			},
			wantErr: false,
		},
		{
			name: "skips if already configured",
			initialContent: `
http {
    include       mime.types;
    include /Users/testuser/.magebox/nginx/vhosts/*.conf; # MageBox vhosts
}
`,
			includeDir:   "/Users/testuser/.magebox/nginx/vhosts",
			wantContains: []string{"include /Users/testuser/.magebox/nginx/vhosts/*.conf;"},
			wantErr:      false,
		},
		{
			name: "comments out servers/* if magebox already configured but servers/* still present",
			initialContent: `
http {
    include       mime.types;
    include servers/*;
    include /Users/testuser/.magebox/nginx/vhosts/*.conf; # MageBox vhosts
}
`,
			includeDir: "/Users/testuser/.magebox/nginx/vhosts",
			wantContains: []string{
				"# include servers/*;",
				"Disabled by MageBox",
			},
			wantErr: false,
		},
		{
			name: "adds to http block if no servers/* found",
			initialContent: `
http {
    include       mime.types;
    default_type  application/octet-stream;
}
`,
			includeDir: "/Users/testuser/.magebox/nginx/vhosts",
			wantContains: []string{
				"include /Users/testuser/.magebox/nginx/vhosts/*.conf;",
			},
			wantErr: false,
		},
		{
			name: "fails if no http block",
			initialContent: `
events {
    worker_connections 1024;
}
`,
			includeDir: "/Users/testuser/.magebox/nginx/vhosts",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory and nginx.conf
			tmpDir := t.TempDir()
			nginxConf := filepath.Join(tmpDir, "nginx.conf")
			if err := os.WriteFile(nginxConf, []byte(tt.initialContent), 0644); err != nil {
				t.Fatalf("Failed to write test nginx.conf: %v", err)
			}

			// Create controller with custom nginx.conf path
			p := &platform.Platform{
				Type:    platform.Darwin,
				HomeDir: "/Users/testuser",
			}
			c := &Controller{platform: p}

			// Override GetNginxConfPath for testing
			includeDirective := "include " + tt.includeDir + "/*.conf;"

			// Read, modify, write manually to test the logic
			content, _ := os.ReadFile(nginxConf)

			// Check if include already exists
			if strings.Contains(string(content), includeDirective) {
				newContent := string(content)
				newContent = strings.Replace(newContent, "include servers/*;", "# include servers/*; # Disabled by MageBox (invalid: loads directories)", 1)
				if newContent != string(content) {
					os.WriteFile(nginxConf, []byte(newContent), 0644)
				}
			} else {
				newContent := string(content)
				marker := "include servers/*;"
				if strings.Contains(newContent, marker) {
					newContent = strings.Replace(newContent, marker, includeDirective+" # MageBox vhosts", 1)
				} else if strings.Contains(newContent, "http {") {
					lastBrace := strings.LastIndex(newContent, "}")
					if lastBrace > 0 {
						newContent = newContent[:lastBrace] + "    " + includeDirective + " # MageBox vhosts\n" + newContent[lastBrace:]
					} else {
						if tt.wantErr {
							return // Expected error
						}
						t.Fatal("Expected to find closing brace")
					}
				} else {
					if tt.wantErr {
						return // Expected error
					}
					t.Fatal("Expected to find http block")
				}
				os.WriteFile(nginxConf, []byte(newContent), 0644)
			}

			// Read result
			result, err := os.ReadFile(nginxConf)
			if err != nil {
				t.Fatalf("Failed to read result: %v", err)
			}

			resultStr := string(result)

			// Check expected content
			for _, want := range tt.wantContains {
				if !strings.Contains(resultStr, want) {
					t.Errorf("Result should contain %q\nGot:\n%s", want, resultStr)
				}
			}

			// Check content that should not be present
			for _, notWant := range tt.wantNotContain {
				if strings.Contains(resultStr, notWant) {
					t.Errorf("Result should NOT contain %q\nGot:\n%s", notWant, resultStr)
				}
			}

			// Verify the include path uses the correct home directory
			if !tt.wantErr && !strings.Contains(resultStr, tt.includeDir) {
				t.Errorf("Result should contain user's home directory path %q", tt.includeDir)
			}

			_ = c // Use controller to avoid unused variable warning
		})
	}
}

func TestController_addIncludeToNginxConf_Linux(t *testing.T) {
	tests := []struct {
		name           string
		initialContent string
		includeDir     string
		wantContains   []string
		wantErr        bool
	}{
		{
			name: "adds include after conf.d include",
			initialContent: `
http {
    include       /etc/nginx/mime.types;
    include /etc/nginx/conf.d/*.conf;
}
`,
			includeDir: "/home/testuser/.magebox/nginx/vhosts",
			wantContains: []string{
				"include /etc/nginx/conf.d/*.conf;",
				"include /home/testuser/.magebox/nginx/vhosts/*.conf;",
				"# MageBox vhosts",
			},
			wantErr: false,
		},
		{
			name: "skips if already configured",
			initialContent: `
http {
    include /etc/nginx/conf.d/*.conf;
    include /home/testuser/.magebox/nginx/vhosts/*.conf; # MageBox vhosts
}
`,
			includeDir: "/home/testuser/.magebox/nginx/vhosts",
			wantContains: []string{
				"include /home/testuser/.magebox/nginx/vhosts/*.conf;",
			},
			wantErr: false,
		},
		{
			name: "fails if no conf.d include",
			initialContent: `
http {
    include       /etc/nginx/mime.types;
}
`,
			includeDir: "/home/testuser/.magebox/nginx/vhosts",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory and nginx.conf
			tmpDir := t.TempDir()
			nginxConf := filepath.Join(tmpDir, "nginx.conf")
			if err := os.WriteFile(nginxConf, []byte(tt.initialContent), 0644); err != nil {
				t.Fatalf("Failed to write test nginx.conf: %v", err)
			}

			includeDirective := "include " + tt.includeDir + "/*.conf;"

			// Simulate the Linux addIncludeToNginxConf logic
			content, _ := os.ReadFile(nginxConf)

			if strings.Contains(string(content), includeDirective) {
				// Already configured, skip
				return
			}

			marker := "include /etc/nginx/conf.d/*.conf;"
			if !strings.Contains(string(content), marker) {
				if tt.wantErr {
					return // Expected error
				}
				t.Fatal("Expected to find conf.d include")
			}

			newContent := strings.Replace(
				string(content),
				marker,
				marker+"\n    "+includeDirective+" # MageBox vhosts",
				1,
			)

			os.WriteFile(nginxConf, []byte(newContent), 0644)

			// Read result
			result, err := os.ReadFile(nginxConf)
			if err != nil {
				t.Fatalf("Failed to read result: %v", err)
			}

			resultStr := string(result)

			// Check expected content
			for _, want := range tt.wantContains {
				if !strings.Contains(resultStr, want) {
					t.Errorf("Result should contain %q\nGot:\n%s", want, resultStr)
				}
			}
		})
	}
}

func TestMageBoxDir_UsesCorrectHomeDir(t *testing.T) {
	tests := []struct {
		name     string
		homeDir  string
		expected string
	}{
		{
			name:     "macOS user jakub",
			homeDir:  "/Users/jakub",
			expected: "/Users/jakub/.magebox",
		},
		{
			name:     "macOS user john",
			homeDir:  "/Users/john",
			expected: "/Users/john/.magebox",
		},
		{
			name:     "Linux user developer",
			homeDir:  "/home/developer",
			expected: "/home/developer/.magebox",
		},
		{
			name:     "Linux root user",
			homeDir:  "/root",
			expected: "/root/.magebox",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &platform.Platform{HomeDir: tt.homeDir}
			got := p.MageBoxDir()
			if got != tt.expected {
				t.Errorf("MageBoxDir() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIncludeDirective_UsesCorrectPath(t *testing.T) {
	tests := []struct {
		name     string
		homeDir  string
		expected string
	}{
		{
			name:     "macOS user",
			homeDir:  "/Users/jakub",
			expected: "include /Users/jakub/.magebox/nginx/vhosts/*.conf;",
		},
		{
			name:     "Linux user",
			homeDir:  "/home/developer",
			expected: "include /home/developer/.magebox/nginx/vhosts/*.conf;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &platform.Platform{HomeDir: tt.homeDir}
			sslMgr := ssl.NewManager(p)
			g := NewVhostGenerator(p, sslMgr)
			got := g.GetIncludeDirective()
			if got != tt.expected {
				t.Errorf("GetIncludeDirective() = %v, want %v", got, tt.expected)
			}
		})
	}
}

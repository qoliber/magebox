package ssl

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"qoliber/magebox/internal/lib"
	"qoliber/magebox/internal/platform"
)

//go:embed templates/not-installed-error.tmpl
var notInstalledErrorTemplateEmbed string

func init() {
	// Register embedded template as fallback
	lib.RegisterFallbackTemplate(lib.TemplateSSL, "not-installed-error.tmpl", notInstalledErrorTemplateEmbed)
}

// NotInstalledErrorData contains data for the not-installed error template
type NotInstalledErrorData struct {
	InstallCommand string
}

// Manager handles SSL certificate management using mkcert
type Manager struct {
	platform    *platform.Platform
	certsDir    string
	caInstalled bool
}

// CertPaths contains paths to certificate and key files
type CertPaths struct {
	CertFile string
	KeyFile  string
	Domain   string
}

// NewManager creates a new SSL manager
func NewManager(p *platform.Platform) *Manager {
	// Use ~/.magebox/certs on all platforms
	// nginx runs as current user (set in bootstrap), so it can access home dir
	certsDir := filepath.Join(p.MageBoxDir(), "certs")
	return &Manager{
		platform: p,
		certsDir: certsDir,
	}
}

// EnsureCAInstalled ensures the local CA is installed and trusted
func (m *Manager) EnsureCAInstalled() error {
	if !m.IsMkcertInstalled() {
		return &MkcertNotInstalledError{Platform: m.platform}
	}

	// Check if CA is already installed by looking for CAROOT
	caRoot, err := m.getCARoot()
	if err != nil {
		return fmt.Errorf("failed to get CA root: %w", err)
	}

	// Check if CA files exist
	if _, err := os.Stat(filepath.Join(caRoot, "rootCA.pem")); os.IsNotExist(err) {
		// Install the CA
		cmd := exec.Command("mkcert", "-install")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install CA: %w", err)
		}
	}

	m.caInstalled = true
	return nil
}

// IsCAInstalled checks if the mkcert CA is already installed
func (m *Manager) IsCAInstalled() bool {
	if !m.IsMkcertInstalled() {
		return false
	}

	caRoot, err := m.getCARoot()
	if err != nil {
		return false
	}

	// Check if CA files exist
	if _, err := os.Stat(filepath.Join(caRoot, "rootCA.pem")); os.IsNotExist(err) {
		return false
	}

	return true
}

// InstallCA installs the mkcert CA (runs mkcert -install)
func (m *Manager) InstallCA() error {
	if !m.IsMkcertInstalled() {
		return &MkcertNotInstalledError{Platform: m.platform}
	}

	cmd := exec.Command("mkcert", "-install")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to install CA: %w\nOutput: %s", err, output)
	}

	m.caInstalled = true
	return nil
}

// IsMkcertInstalled checks if mkcert is installed
func (m *Manager) IsMkcertInstalled() bool {
	return platform.CommandExists("mkcert")
}

// getCARoot returns the mkcert CA root directory
func (m *Manager) getCARoot() (string, error) {
	cmd := exec.Command("mkcert", "-CAROOT")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// GenerateCert generates a certificate for the given domain
func (m *Manager) GenerateCert(domain string) (*CertPaths, error) {
	if !m.IsMkcertInstalled() {
		return nil, &MkcertNotInstalledError{Platform: m.platform}
	}

	// Ensure certs directory exists
	domainDir := filepath.Join(m.certsDir, domain)
	if err := os.MkdirAll(domainDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create certs directory: %w", err)
	}

	certFile := filepath.Join(domainDir, "cert.pem")
	keyFile := filepath.Join(domainDir, "key.pem")

	// Check if cert already exists
	if m.CertExists(domain) {
		return &CertPaths{
			CertFile: certFile,
			KeyFile:  keyFile,
			Domain:   domain,
		}, nil
	}

	// Generate cert using mkcert
	cmd := exec.Command("mkcert",
		"-cert-file", certFile,
		"-key-file", keyFile,
		domain,
		"*."+domain,
	)
	cmd.Dir = domainDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to generate certificate: %w\nOutput: %s", err, output)
	}

	return &CertPaths{
		CertFile: certFile,
		KeyFile:  keyFile,
		Domain:   domain,
	}, nil
}

// GenerateCerts generates certificates for multiple domains
func (m *Manager) GenerateCerts(domains []string) ([]*CertPaths, error) {
	certs := make([]*CertPaths, 0, len(domains))

	for _, domain := range domains {
		cert, err := m.GenerateCert(domain)
		if err != nil {
			return nil, fmt.Errorf("failed to generate cert for %s: %w", domain, err)
		}
		certs = append(certs, cert)
	}

	return certs, nil
}

// CertExists checks if a certificate exists for the given domain
func (m *Manager) CertExists(domain string) bool {
	domainDir := filepath.Join(m.certsDir, domain)
	certFile := filepath.Join(domainDir, "cert.pem")
	keyFile := filepath.Join(domainDir, "key.pem")

	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		return false
	}
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		return false
	}

	return true
}

// GetCertPaths returns the certificate paths for a domain
func (m *Manager) GetCertPaths(domain string) *CertPaths {
	domainDir := filepath.Join(m.certsDir, domain)
	return &CertPaths{
		CertFile: filepath.Join(domainDir, "cert.pem"),
		KeyFile:  filepath.Join(domainDir, "key.pem"),
		Domain:   domain,
	}
}

// RemoveCert removes the certificate for a domain
func (m *Manager) RemoveCert(domain string) error {
	domainDir := filepath.Join(m.certsDir, domain)
	return os.RemoveAll(domainDir)
}

// ListCerts returns all generated certificates
func (m *Manager) ListCerts() ([]string, error) {
	entries, err := os.ReadDir(m.certsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	domains := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			domains = append(domains, entry.Name())
		}
	}

	return domains, nil
}

// CertsDir returns the certificates directory path
func (m *Manager) CertsDir() string {
	return m.certsDir
}

// MkcertNotInstalledError indicates mkcert is not installed
type MkcertNotInstalledError struct {
	Platform *platform.Platform
}

func (e *MkcertNotInstalledError) Error() string {
	data := NotInstalledErrorData{
		InstallCommand: e.Platform.MkcertInstallCommand(),
	}

	// Load template from lib (with embedded fallback)
	tmplContent, err := lib.GetTemplate(lib.TemplateSSL, "not-installed-error.tmpl")
	if err != nil {
		// Fallback to simple message
		return fmt.Sprintf("mkcert is not installed\n\nInstall it with:\n  %s\n\nThen run: magebox ssl:trust\n", e.Platform.MkcertInstallCommand())
	}

	tmpl, err := template.New("not-installed-error").Parse(tmplContent)
	if err != nil {
		// Fallback to simple message
		return fmt.Sprintf("mkcert is not installed\n\nInstall it with:\n  %s\n\nThen run: magebox ssl:trust\n", e.Platform.MkcertInstallCommand())
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Sprintf("mkcert is not installed\n\nInstall it with:\n  %s\n\nThen run: magebox ssl:trust\n", e.Platform.MkcertInstallCommand())
	}

	return buf.String()
}

// ExtractBaseDomain extracts the base domain from a hostname
// e.g., "api.mystore.test" -> "mystore.test"
func ExtractBaseDomain(hostname string) string {
	parts := strings.Split(hostname, ".")
	if len(parts) <= 2 {
		return hostname
	}
	// Return last two parts
	return strings.Join(parts[len(parts)-2:], ".")
}

// GroupDomainsByBase groups domains by their base domain
// This helps generate fewer certificates with wildcards
func GroupDomainsByBase(domains []string) map[string][]string {
	groups := make(map[string][]string)

	for _, domain := range domains {
		base := ExtractBaseDomain(domain)
		groups[base] = append(groups[base], domain)
	}

	return groups
}

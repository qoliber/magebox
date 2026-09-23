package nginx

import (
	"os"
	"path/filepath"
	"strings"
)

// SSLCertificatePaths returns the certificate and key files a vhost references.
func SSLCertificatePaths(content string) []string {
	var paths []string
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) < 2 {
			continue
		}
		if fields[0] != "ssl_certificate" && fields[0] != "ssl_certificate_key" {
			continue
		}
		paths = append(paths, strings.TrimSuffix(fields[1], ";"))
	}
	return paths
}

// DomainFromCertPath recovers the domain from a certificate path, which
// MageBox stores as <certs dir>/<domain>/cert.pem.
func DomainFromCertPath(path string) string {
	if path == "" {
		return ""
	}
	switch filepath.Base(path) {
	case "cert.pem", "key.pem":
	default:
		// Managed by something other than MageBox: not ours to regenerate.
		return ""
	}
	dir := filepath.Base(filepath.Dir(path))
	if dir == "." || dir == string(filepath.Separator) {
		return ""
	}
	return dir
}

// MissingCertificates reports the certificate files referenced by the vhosts in
// dir that do not exist, keyed by the vhost that references them.
//
// nginx refuses to start when a single certificate is missing, so one stale
// vhost takes every project on the machine offline.
func MissingCertificates(vhostsDir string) (map[string][]string, error) {
	entries, err := os.ReadDir(vhostsDir)
	if err != nil {
		return nil, err
	}

	missing := make(map[string][]string)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".conf") {
			continue
		}
		path := filepath.Join(vhostsDir, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, cert := range SSLCertificatePaths(string(content)) {
			if _, err := os.Stat(cert); err != nil {
				missing[path] = append(missing[path], cert)
			}
		}
	}
	return missing, nil
}

package ssl

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"qoliber/magebox/internal/platform"
)

// nssListsCASerial reports whether a certutil listing trusts the CA with this
// serial.
//
// mkcert names its authority "mkcert development CA <serial in decimal>", so
// the nickname identifies exactly which CA a browser trusts. A machine restored
// from another install often trusts a stale one, which no file check catches.
func nssListsCASerial(certutilOutput, serialDecimal string) bool {
	if serialDecimal == "" {
		return false
	}
	for _, line := range strings.Split(certutilOutput, "\n") {
		if strings.Contains(line, serialDecimal) {
			return true
		}
	}
	return false
}

// caSerialDecimal returns the serial of the local CA, as mkcert writes it into
// the trust store nickname.
func (m *Manager) caSerialDecimal() string {
	caRoot, err := m.getCARoot()
	if err != nil {
		return ""
	}
	content, err := os.ReadFile(filepath.Join(caRoot, "rootCA.pem"))
	if err != nil {
		return ""
	}
	block, _ := pem.Decode(content)
	if block == nil {
		return ""
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return ""
	}
	return cert.SerialNumber.String()
}

// nssDatabases are the browser trust stores mkcert writes to.
func nssDatabases() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{filepath.Join(home, ".pki", "nssdb")}
}

// IsCATrustedByBrowsers reports whether the browser trust store holds the
// current CA. It answers true when it cannot tell, so a missing certutil or an
// absent store never triggers a pointless reinstall.
func (m *Manager) IsCATrustedByBrowsers() bool {
	if !platform.CommandExists("certutil") {
		return true
	}
	serial := m.caSerialDecimal()
	if serial == "" {
		return true
	}

	for _, db := range nssDatabases() {
		if _, err := os.Stat(db); err != nil {
			continue
		}
		output, err := exec.Command("certutil", "-L", "-d", "sql:"+db).Output()
		if err != nil {
			continue
		}
		if !nssListsCASerial(string(output), serial) {
			return false
		}
	}
	return true
}

package ssl

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeTestCert generates a self-signed certificate carrying the given DNS SANs
// and returns the path to the written cert.pem. VerifyHostname (used by
// certCovers) only inspects the SAN list, so the cert need not be CA-signed.
func writeTestCert(t *testing.T, dir string, dnsNames []string) string {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: dnsNames[0]},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		DNSNames:     dnsNames,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}

	certFile := filepath.Join(dir, "cert.pem")
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	if err := os.WriteFile(certFile, pemBytes, 0644); err != nil {
		t.Fatal(err)
	}
	return certFile
}

func TestCertCovers(t *testing.T) {
	// Mirrors a worktree cert: base wildcard plus the exact nested host that the
	// wildcard cannot reach on its own.
	certFile := writeTestCert(t, t.TempDir(), []string{
		"b2b-case.localhost",
		"*.b2b-case.localhost",
		"shop.nl.b2b-case.localhost",
	})

	tests := []struct {
		name  string
		hosts []string
		want  bool
	}{
		{"empty is trivially covered", nil, true},
		{"exact nested host (added SAN)", []string{"shop.nl.b2b-case.localhost"}, true},
		{"wildcard single label", []string{"admin.b2b-case.localhost"}, true},
		{"base domain", []string{"b2b-case.localhost"}, true},
		{"two-label host not covered by wildcard", []string{"a.b.b2b-case.localhost"}, false},
		{"unrelated host", []string{"other.localhost"}, false},
		{"any uncovered host fails the set", []string{"shop.nl.b2b-case.localhost", "a.b.b2b-case.localhost"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := certCovers(certFile, tt.hosts); got != tt.want {
				t.Errorf("certCovers(%v) = %v, want %v", tt.hosts, got, tt.want)
			}
		})
	}
}

func TestCertCoversMissingFile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist.pem")
	if certCovers(missing, []string{"x.localhost"}) {
		t.Error("expected a missing cert file to report not covered")
	}
}

func TestDedupeStrings(t *testing.T) {
	got := dedupeStrings([]string{"a", "b", "a", "c", "b"})
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("dedupeStrings = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("dedupeStrings = %v, want %v", got, want)
		}
	}
}

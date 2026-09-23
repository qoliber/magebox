package nginx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSSLCertificatePaths(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name: "certificate and key",
			content: `server {
    listen 443 ssl;
    ssl_certificate /home/jakub/.magebox/certs/shop.test/cert.pem;
    ssl_certificate_key /home/jakub/.magebox/certs/shop.test/key.pem;
}`,
			want: []string{"/home/jakub/.magebox/certs/shop.test/cert.pem", "/home/jakub/.magebox/certs/shop.test/key.pem"},
		},
		{
			name:    "commented lines are ignored",
			content: "    # ssl_certificate /old/cert.pem;\n    ssl_certificate /new/cert.pem;",
			want:    []string{"/new/cert.pem"},
		},
		{
			name:    "directives that merely start the same are ignored",
			content: "    ssl_certificate_by_lua_file /etc/nginx/hook.lua;",
			want:    nil,
		},
		{
			name:    "no ssl at all",
			content: "server {\n    listen 80;\n}",
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SSLCertificatePaths(tt.content)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d paths %v, want %d %v", len(got), got, len(tt.want), tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("path %d = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// Only MageBox's own layout can be regenerated; a certificate managed
// elsewhere must be left alone and reported instead.
func TestDomainFromCertPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{path: "/home/jakub/.magebox/certs/shop.test/cert.pem", want: "shop.test"},
		{path: "/home/jakub/.magebox/certs/shop.test/key.pem", want: "shop.test"},
		{path: "/etc/ssl/other.pem", want: ""},
		{path: "/home/jakub/.magebox/certs/shop.test/fullchain.pem", want: ""},
		{path: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := DomainFromCertPath(tt.path); got != tt.want {
				t.Errorf("DomainFromCertPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

// One vhost pointing at a certificate that is not there stops nginx from
// starting at all, which takes every other project down with it.
func TestMissingCertificates(t *testing.T) {
	dir := t.TempDir()
	certDir := filepath.Join(dir, "certs", "present.test")
	if err := os.MkdirAll(certDir, 0755); err != nil {
		t.Fatal(err)
	}
	existing := filepath.Join(certDir, "cert.pem")
	if err := os.WriteFile(existing, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	vhosts := filepath.Join(dir, "vhosts")
	if err := os.MkdirAll(vhosts, 0755); err != nil {
		t.Fatal(err)
	}
	gone := filepath.Join(dir, "certs", "gone.test", "cert.pem")
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(vhosts, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("ok.conf", "ssl_certificate "+existing+";")
	write("broken.conf", "ssl_certificate "+gone+";")
	write("plain.conf", "listen 80;")
	write("notes.txt", "ssl_certificate /ignored/cert.pem;")

	missing, err := MissingCertificates(vhosts)
	if err != nil {
		t.Fatalf("MissingCertificates failed: %v", err)
	}
	if len(missing) != 1 {
		t.Fatalf("expected one broken vhost, got %d: %v", len(missing), missing)
	}
	paths, ok := missing[filepath.Join(vhosts, "broken.conf")]
	if !ok {
		t.Fatalf("broken.conf not reported: %v", missing)
	}
	if len(paths) != 1 || paths[0] != gone {
		t.Errorf("reported %v, want [%s]", paths, gone)
	}
}

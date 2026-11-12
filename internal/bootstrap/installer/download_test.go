package installer

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// The sury keyring is installed with `sudo dpkg -i`, so the file must live
// somewhere no other local user can create, replace or symlink beforehand.
func TestDownloadToPrivateFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/missing" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte("package-bytes"))
	}))
	defer srv.Close()

	t.Run("writes into a fresh owner-only directory", func(t *testing.T) {
		path, cleanup, err := downloadToPrivateFile(srv.URL+"/keyring.deb", "keyring.deb")
		if err != nil {
			t.Fatalf("download failed: %v", err)
		}
		dir := filepath.Dir(path)

		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("stat dir: %v", err)
		}
		if perm := info.Mode().Perm(); perm != 0o700 {
			t.Errorf("directory permissions = %o, want 700", perm)
		}
		if filepath.Base(path) != "keyring.deb" {
			t.Errorf("file name = %q, want keyring.deb", filepath.Base(path))
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read file: %v", err)
		}
		if string(got) != "package-bytes" {
			t.Errorf("content = %q, want package-bytes", got)
		}

		cleanup()
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Errorf("cleanup left %s behind", dir)
		}
	})

	t.Run("two downloads never share a path", func(t *testing.T) {
		a, cleanA, err := downloadToPrivateFile(srv.URL+"/keyring.deb", "keyring.deb")
		if err != nil {
			t.Fatal(err)
		}
		defer cleanA()
		b, cleanB, err := downloadToPrivateFile(srv.URL+"/keyring.deb", "keyring.deb")
		if err != nil {
			t.Fatal(err)
		}
		defer cleanB()
		if a == b {
			t.Errorf("both downloads used %s", a)
		}
	})

	t.Run("HTTP error fails and leaves nothing behind", func(t *testing.T) {
		before, _ := filepath.Glob(filepath.Join(os.TempDir(), "magebox-download-*"))
		_, _, err := downloadToPrivateFile(srv.URL+"/missing", "keyring.deb")
		if err == nil {
			t.Fatal("expected an error for a 404")
		}
		after, _ := filepath.Glob(filepath.Join(os.TempDir(), "magebox-download-*"))
		if len(after) != len(before) {
			t.Errorf("failed download left a temp directory behind")
		}
	})
}

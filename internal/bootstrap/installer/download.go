package installer

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// downloadToPrivateFile fetches url into a new directory only the current user
// can enter, and returns the file's path plus a cleanup that removes the
// directory.
//
// A fixed path such as /tmp/name.deb lets any other local user create the file
// first, or swap it between the download and a later `sudo dpkg -i`, and have
// their own package installed as root. A fresh 0700 directory from MkdirTemp
// rules both out.
func downloadToPrivateFile(url, name string) (string, func(), error) {
	dir, err := os.MkdirTemp("", "magebox-download-*")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create a temporary directory: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(dir) }

	path := filepath.Join(dir, name)
	if err := fetchTo(url, path); err != nil {
		cleanup()
		return "", nil, err
	}
	return path, cleanup, nil
}

func fetchTo(url, path string) error {
	client := &http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download %s: %s", url, resp.Status)
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", path, err)
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = f.Close()
		return fmt.Errorf("failed to download %s: %w", url, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}

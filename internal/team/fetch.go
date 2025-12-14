// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package team

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Fetcher handles the complete fetch workflow
type Fetcher struct {
	team       *Team
	project    *Project
	options    FetchOptions
	repoClient *RepositoryClient
	progress   func(string)
}

// NewFetcher creates a new fetcher for a team project
func NewFetcher(team *Team, project *Project, options FetchOptions) *Fetcher {
	return &Fetcher{
		team:       team,
		project:    project,
		options:    options,
		repoClient: NewRepositoryClient(team),
	}
}

// SetProgressCallback sets the progress callback function
func (f *Fetcher) SetProgressCallback(fn func(string)) {
	f.progress = fn
}

// Execute runs the fetch workflow
func (f *Fetcher) Execute() error {
	projectName := filepath.Base(f.project.Repo)
	destPath := GetProjectPath(f.options.DestPath, projectName)

	if f.options.DryRun {
		return f.dryRun(destPath)
	}

	// Step 1: Clone repository (unless DB/media only)
	if !f.options.DBOnly && !f.options.MediaOnly {
		if err := f.cloneRepository(destPath); err != nil {
			return err
		}
	}

	// Step 2: Download and import database
	if !f.options.NoDB && !f.options.MediaOnly && f.project.DB != "" {
		if err := f.downloadAndImportDB(destPath); err != nil {
			return err
		}
	}

	// Step 3: Download and extract media
	if !f.options.NoMedia && !f.options.DBOnly && f.project.Media != "" {
		if err := f.downloadAndExtractMedia(destPath); err != nil {
			return err
		}
	}

	// Step 4: Run post-fetch commands
	if len(f.project.PostFetch) > 0 && !f.options.DBOnly && !f.options.MediaOnly {
		if err := f.runPostFetchCommands(destPath); err != nil {
			return err
		}
	}

	return nil
}

// dryRun shows what would happen without actually doing it
func (f *Fetcher) dryRun(destPath string) error {
	f.report("DRY RUN - No changes will be made")
	f.report("")

	if !f.options.DBOnly && !f.options.MediaOnly {
		f.report("Would clone: %s", f.team.GetCloneURL(f.project))
		f.report("  Branch: %s", f.getBranch())
		f.report("  Destination: %s", destPath)
	}

	if !f.options.NoDB && !f.options.MediaOnly && f.project.DB != "" {
		f.report("")
		f.report("Would download database: %s", f.project.DB)
		f.report("  From: %s@%s:%s", f.team.Assets.Username, f.team.Assets.Host, f.team.Assets.Path)
		f.report("  Would import to MySQL")
	}

	if !f.options.NoMedia && !f.options.DBOnly && f.project.Media != "" {
		f.report("")
		f.report("Would download media: %s", f.project.Media)
		f.report("  Extract to: %s/pub/media", destPath)
	}

	if len(f.project.PostFetch) > 0 && !f.options.DBOnly && !f.options.MediaOnly {
		f.report("")
		f.report("Would run post-fetch commands:")
		for _, cmd := range f.project.PostFetch {
			f.report("  - %s", cmd)
		}
	}

	return nil
}

// cloneRepository clones the git repository
func (f *Fetcher) cloneRepository(destPath string) error {
	f.report("Cloning repository...")

	project := f.project
	if f.options.Branch != "" {
		// Create a copy with overridden branch
		projectCopy := *f.project
		projectCopy.Branch = f.options.Branch
		project = &projectCopy
	}

	return f.repoClient.Clone(project, destPath, f.progress)
}

// downloadAndImportDB downloads the database and imports it
func (f *Fetcher) downloadAndImportDB(destPath string) error {
	f.report("")
	f.report("Downloading database...")

	// Connect to asset storage
	assetClient := NewAssetClient(f.team, func(p DownloadProgress) {
		f.report("\r  %s: %.1f%% (%s/%s) %s ETA: %s",
			p.Filename, p.Percentage,
			FormatBytes(p.Downloaded), FormatBytes(p.TotalBytes),
			FormatSpeed(p.Speed), p.ETA)
	})

	if err := assetClient.Connect(); err != nil {
		return fmt.Errorf("failed to connect to asset storage: %w", err)
	}
	defer assetClient.Close()

	// Download to temp file
	tmpDir := os.TempDir()
	localPath := filepath.Join(tmpDir, "magebox-db-"+filepath.Base(f.project.DB))

	if err := assetClient.Download(f.project.DB, localPath); err != nil {
		return fmt.Errorf("failed to download database: %w", err)
	}
	defer os.Remove(localPath)

	f.report("")
	f.report("Database downloaded successfully")

	// Import database using magebox db import
	f.report("Importing database...")

	cmd := exec.Command("magebox", "db", "import", localPath)
	cmd.Dir = destPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to import database: %w", err)
	}

	f.report("Database imported successfully")
	return nil
}

// downloadAndExtractMedia downloads and extracts media files
func (f *Fetcher) downloadAndExtractMedia(destPath string) error {
	f.report("")
	f.report("Downloading media...")

	// Connect to asset storage
	assetClient := NewAssetClient(f.team, func(p DownloadProgress) {
		f.report("\r  %s: %.1f%% (%s/%s) %s ETA: %s",
			p.Filename, p.Percentage,
			FormatBytes(p.Downloaded), FormatBytes(p.TotalBytes),
			FormatSpeed(p.Speed), p.ETA)
	})

	if err := assetClient.Connect(); err != nil {
		return fmt.Errorf("failed to connect to asset storage: %w", err)
	}
	defer assetClient.Close()

	// Download to temp file
	tmpDir := os.TempDir()
	localPath := filepath.Join(tmpDir, "magebox-media-"+filepath.Base(f.project.Media))

	if err := assetClient.Download(f.project.Media, localPath); err != nil {
		return fmt.Errorf("failed to download media: %w", err)
	}
	defer os.Remove(localPath)

	f.report("")
	f.report("Media downloaded successfully")

	// Extract media
	f.report("Extracting media...")

	mediaDir := filepath.Join(destPath, "pub", "media")
	if err := os.MkdirAll(mediaDir, 0755); err != nil {
		return fmt.Errorf("failed to create media directory: %w", err)
	}

	if err := extractTarGz(localPath, mediaDir); err != nil {
		return fmt.Errorf("failed to extract media: %w", err)
	}

	f.report("Media extracted successfully")
	return nil
}

// runPostFetchCommands runs configured post-fetch commands
func (f *Fetcher) runPostFetchCommands(destPath string) error {
	f.report("")
	f.report("Running post-fetch commands...")

	for _, cmdStr := range f.project.PostFetch {
		f.report("  Running: %s", cmdStr)

		parts := strings.Fields(cmdStr)
		if len(parts) == 0 {
			continue
		}

		cmd := exec.Command(parts[0], parts[1:]...)
		cmd.Dir = destPath
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("post-fetch command failed '%s': %w", cmdStr, err)
		}
	}

	return nil
}

// getBranch returns the branch to use
func (f *Fetcher) getBranch() string {
	if f.options.Branch != "" {
		return f.options.Branch
	}
	if f.project.Branch != "" {
		return f.project.Branch
	}
	return "main"
}

// report outputs progress message
func (f *Fetcher) report(format string, args ...interface{}) {
	if f.progress != nil {
		f.progress(fmt.Sprintf(format, args...))
	} else {
		fmt.Printf(format+"\n", args...)
	}
}

// extractTarGz extracts a .tar.gz file to a directory
func extractTarGz(archivePath, destDir string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar read error: %w", err)
		}

		// Security: prevent path traversal
		targetPath := filepath.Join(destDir, header.Name)
		if !strings.HasPrefix(targetPath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path in archive: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return err
			}
			outFile, err := os.Create(targetPath)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
			if err := os.Chmod(targetPath, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if err := os.Symlink(header.Linkname, targetPath); err != nil {
				// Ignore symlink errors on some systems
				continue
			}
		}
	}

	return nil
}

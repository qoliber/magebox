// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package team

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractTarGz(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Create a test tar.gz file
	archivePath := filepath.Join(tmpDir, "test.tar.gz")
	destDir := filepath.Join(tmpDir, "extracted")

	// Create the archive with test files
	if err := createTestTarGz(archivePath); err != nil {
		t.Fatalf("Failed to create test archive: %v", err)
	}

	// Create destination directory
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("Failed to create dest dir: %v", err)
	}

	// Extract the archive
	if err := extractTarGz(archivePath, destDir); err != nil {
		t.Fatalf("extractTarGz failed: %v", err)
	}

	// Verify extracted files
	testFile := filepath.Join(destDir, "testfile.txt")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("Expected testfile.txt to be extracted")
	}

	// Verify file contents
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read extracted file: %v", err)
	}
	if string(content) != "test content" {
		t.Errorf("Expected 'test content', got '%s'", string(content))
	}

	// Verify directory was created
	testDir := filepath.Join(destDir, "testdir")
	info, err := os.Stat(testDir)
	if os.IsNotExist(err) {
		t.Error("Expected testdir to be extracted")
	} else if !info.IsDir() {
		t.Error("Expected testdir to be a directory")
	}
}

func TestExtractTarGz_PathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "malicious.tar.gz")
	destDir := filepath.Join(tmpDir, "extracted")

	// Create a malicious archive with path traversal
	if err := createMaliciousTarGz(archivePath); err != nil {
		t.Fatalf("Failed to create malicious archive: %v", err)
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("Failed to create dest dir: %v", err)
	}

	// Extraction should fail due to path traversal protection
	err := extractTarGz(archivePath, destDir)
	if err == nil {
		t.Error("Expected error for path traversal, got nil")
	}
	if !strings.Contains(err.Error(), "invalid file path") {
		t.Errorf("Expected 'invalid file path' error, got: %v", err)
	}
}

func TestClonerDryRun(t *testing.T) {
	team := &Team{
		Name: "testteam",
		Repositories: RepositoryConfig{
			Provider: ProviderGitHub,
			Auth:     AuthSSH,
		},
	}
	project := &Project{
		Repo:   "testorg/testproject",
		Branch: "main",
	}

	options := CloneOptions{
		DestPath: "/test/path",
		DryRun:   true,
	}

	cloner := NewCloner(team, project, options)

	var output []string
	cloner.SetProgressCallback(func(msg string) {
		output = append(output, msg)
	})

	if err := cloner.Execute(); err != nil {
		t.Fatalf("Cloner.Execute() failed: %v", err)
	}

	// Verify dry run output
	outputStr := strings.Join(output, "\n")
	if !strings.Contains(outputStr, "DRY RUN") {
		t.Error("Expected DRY RUN in output")
	}
	if !strings.Contains(outputStr, "Would clone") {
		t.Error("Expected 'Would clone' in output")
	}
}

func TestAssetFetcherDryRun(t *testing.T) {
	team := &Team{
		Name: "testteam",
		Assets: AssetConfig{
			Provider: AssetSFTP,
			Host:     "backup.example.com",
			Path:     "/backups",
			Username: "deploy",
		},
	}

	options := FetchOptions{
		DryRun:   true,
		NoMedia:  false,
		DestPath: "/test/path",
	}

	fetcher := NewAssetFetcher(team, "myproject", "myproject/db.sql.gz", "myproject/media.tar.gz", options)

	var output []string
	fetcher.SetProgressCallback(func(msg string) {
		output = append(output, msg)
	})

	if err := fetcher.Execute(); err != nil {
		t.Fatalf("AssetFetcher.Execute() failed: %v", err)
	}

	// Verify dry run output
	outputStr := strings.Join(output, "\n")
	if !strings.Contains(outputStr, "DRY RUN") {
		t.Error("Expected DRY RUN in output")
	}
	if !strings.Contains(outputStr, "Would download database") {
		t.Error("Expected 'Would download database' in output")
	}
	if !strings.Contains(outputStr, "Would download media") {
		t.Error("Expected 'Would download media' in output")
	}
}

func TestAssetFetcherDryRun_NoMedia(t *testing.T) {
	team := &Team{
		Name: "testteam",
		Assets: AssetConfig{
			Provider: AssetSFTP,
			Host:     "backup.example.com",
			Path:     "/backups",
			Username: "deploy",
		},
	}

	options := FetchOptions{
		DryRun:   true,
		NoMedia:  true,
		DestPath: "/test/path",
	}

	fetcher := NewAssetFetcher(team, "myproject", "myproject/db.sql.gz", "myproject/media.tar.gz", options)

	var output []string
	fetcher.SetProgressCallback(func(msg string) {
		output = append(output, msg)
	})

	if err := fetcher.Execute(); err != nil {
		t.Fatalf("AssetFetcher.Execute() failed: %v", err)
	}

	// Verify dry run output - should NOT include media
	outputStr := strings.Join(output, "\n")
	if !strings.Contains(outputStr, "Would download database") {
		t.Error("Expected 'Would download database' in output")
	}
	if strings.Contains(outputStr, "Would download media") {
		t.Error("Did not expect 'Would download media' in output when NoMedia=true")
	}
}

func TestClonerGetBranch(t *testing.T) {
	tests := []struct {
		name          string
		optionsBranch string
		projectBranch string
		expected      string
	}{
		{
			name:          "options branch takes precedence",
			optionsBranch: "feature",
			projectBranch: "main",
			expected:      "feature",
		},
		{
			name:          "project branch when no options branch",
			optionsBranch: "",
			projectBranch: "develop",
			expected:      "develop",
		},
		{
			name:          "fallback to main",
			optionsBranch: "",
			projectBranch: "",
			expected:      "main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			team := &Team{
				Repositories: RepositoryConfig{
					Provider: ProviderGitHub,
					Auth:     AuthSSH,
				},
			}
			project := &Project{
				Repo:   "org/repo",
				Branch: tt.projectBranch,
			}
			options := CloneOptions{
				Branch: tt.optionsBranch,
			}

			cloner := NewCloner(team, project, options)
			result := cloner.getBranch()

			if result != tt.expected {
				t.Errorf("getBranch() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestFetchOptions(t *testing.T) {
	// Test that FetchOptions struct is properly initialized
	options := FetchOptions{
		Branch:    "feature",
		NoDB:      true,
		NoMedia:   false,
		DBOnly:    false,
		MediaOnly: false,
		DryRun:    true,
		DestPath:  "/custom/path",
	}

	if options.Branch != "feature" {
		t.Errorf("Expected Branch 'feature', got '%s'", options.Branch)
	}
	if !options.NoDB {
		t.Error("Expected NoDB to be true")
	}
	if options.NoMedia {
		t.Error("Expected NoMedia to be false")
	}
	if !options.DryRun {
		t.Error("Expected DryRun to be true")
	}
}

func TestCloneOptions(t *testing.T) {
	// Test that CloneOptions struct is properly initialized
	options := CloneOptions{
		Branch:   "develop",
		DestPath: "/projects/myproject",
		DryRun:   false,
	}

	if options.Branch != "develop" {
		t.Errorf("Expected Branch 'develop', got '%s'", options.Branch)
	}
	if options.DestPath != "/projects/myproject" {
		t.Errorf("Expected DestPath '/projects/myproject', got '%s'", options.DestPath)
	}
	if options.DryRun {
		t.Error("Expected DryRun to be false")
	}
}

// Helper function to create a test tar.gz file
func createTestTarGz(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// Add a directory
	if err := tarWriter.WriteHeader(&tar.Header{
		Name:     "testdir/",
		Mode:     0755,
		Typeflag: tar.TypeDir,
	}); err != nil {
		return err
	}

	// Add a file
	content := []byte("test content")
	if err := tarWriter.WriteHeader(&tar.Header{
		Name: "testfile.txt",
		Mode: 0644,
		Size: int64(len(content)),
	}); err != nil {
		return err
	}
	if _, err := tarWriter.Write(content); err != nil {
		return err
	}

	return nil
}

// Helper function to create a malicious tar.gz with path traversal
func createMaliciousTarGz(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// Add a file with path traversal
	content := []byte("malicious content")
	if err := tarWriter.WriteHeader(&tar.Header{
		Name: "../../../etc/passwd",
		Mode: 0644,
		Size: int64(len(content)),
	}); err != nil {
		return err
	}
	if _, err := tarWriter.Write(content); err != nil {
		return err
	}

	return nil
}

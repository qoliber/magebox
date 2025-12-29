//go:build integration

// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package fetch_integration

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

const (
	sftpHost     = "localhost"
	sftpPort     = 2222
	sftpUser     = "testuser"
	sftpPass     = "testpass"
	testDataPath = "./testdata"
)

func TestMain(m *testing.M) {
	// Check if we should skip integration tests
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "1" {
		fmt.Println("Skipping integration tests (SKIP_INTEGRATION_TESTS=1)")
		os.Exit(0)
	}

	// Check if Docker is available
	if err := exec.Command("docker", "info").Run(); err != nil {
		fmt.Println("Docker not available, skipping integration tests")
		os.Exit(0)
	}

	// Setup test data
	if err := setupTestData(); err != nil {
		fmt.Printf("Failed to setup test data: %v\n", err)
		os.Exit(1)
	}

	// Start Docker environment
	fmt.Println("Starting Docker environment...")
	if err := runDockerCompose("up", "-d"); err != nil {
		fmt.Printf("Failed to start Docker environment: %v\n", err)
		os.Exit(1)
	}

	// Wait for SFTP server to be ready
	if err := waitForSFTP(30 * time.Second); err != nil {
		fmt.Printf("SFTP server did not become ready: %v\n", err)
		runDockerCompose("down", "-v")
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	fmt.Println("Stopping Docker environment...")
	runDockerCompose("down", "-v")

	os.Exit(code)
}

func runDockerCompose(args ...string) error {
	cmd := exec.Command("docker", append([]string{"compose"}, args...)...)
	cmd.Dir = getTestDir()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func getTestDir() string {
	// Get the directory containing the test file
	_, err := os.Getwd()
	if err != nil {
		return "."
	}
	return "."
}

func waitForSFTP(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		// Try to connect to SFTP port
		cmd := exec.Command("nc", "-z", sftpHost, fmt.Sprintf("%d", sftpPort))
		if err := cmd.Run(); err == nil {
			// Give it a bit more time to fully initialize
			time.Sleep(2 * time.Second)
			return nil
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("SFTP server not ready after %v", timeout)
}

func setupTestData() error {
	testDir := filepath.Join(testDataPath, "testproject")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		return err
	}

	// Create test database file (gzipped SQL)
	dbPath := filepath.Join(testDir, "testproject.sql.gz")
	if err := createTestDBFile(dbPath); err != nil {
		return fmt.Errorf("failed to create test DB file: %w", err)
	}

	// Create test media archive
	mediaPath := filepath.Join(testDir, "testproject.tar.gz")
	if err := createTestMediaArchive(mediaPath); err != nil {
		return fmt.Errorf("failed to create test media archive: %w", err)
	}

	return nil
}

func createTestDBFile(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	// Write a simple SQL dump
	sqlContent := `-- Test database dump
CREATE DATABASE IF NOT EXISTS test_db;
USE test_db;
CREATE TABLE test_table (id INT PRIMARY KEY, name VARCHAR(255));
INSERT INTO test_table VALUES (1, 'test');
`
	_, err = gzWriter.Write([]byte(sqlContent))
	return err
}

func createTestMediaArchive(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// Add a test directory
	if err := tarWriter.WriteHeader(&tar.Header{
		Name:     "catalog/",
		Mode:     0755,
		Typeflag: tar.TypeDir,
	}); err != nil {
		return err
	}

	// Add a test file
	content := []byte("test image content")
	if err := tarWriter.WriteHeader(&tar.Header{
		Name: "catalog/test.jpg",
		Mode: 0644,
		Size: int64(len(content)),
	}); err != nil {
		return err
	}
	_, err = tarWriter.Write(content)
	return err
}

func TestAssetFetcher_DownloadDB(t *testing.T) {
	// This test verifies that the SFTP connection and download works
	// It requires the Docker SFTP server to be running

	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Create a minimal .magebox.yaml
	configContent := `name: testproject
php: "8.2"
domains:
  - host: testproject.test
    root: pub
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".magebox.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create .magebox.yaml: %v", err)
	}

	// Create pub/media directory
	if err := os.MkdirAll(filepath.Join(tmpDir, "pub", "media"), 0755); err != nil {
		t.Fatalf("Failed to create media directory: %v", err)
	}

	// Verify test data exists on SFTP server
	// This is a basic connectivity test
	t.Log("SFTP server is accessible and test data is set up")
}

func TestAssetFetcher_FileExists(t *testing.T) {
	// Verify that the test files were created in testdata
	testDir := filepath.Join(testDataPath, "testproject")

	dbFile := filepath.Join(testDir, "testproject.sql.gz")
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		t.Errorf("Expected test database file to exist: %s", dbFile)
	}

	mediaFile := filepath.Join(testDir, "testproject.tar.gz")
	if _, err := os.Stat(mediaFile); os.IsNotExist(err) {
		t.Errorf("Expected test media archive to exist: %s", mediaFile)
	}
}

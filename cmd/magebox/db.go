package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/qoliber/magebox/internal/cli"
	"github.com/qoliber/magebox/internal/config"
	"github.com/qoliber/magebox/internal/docker"
	"github.com/qoliber/magebox/internal/progress"
)

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Database operations",
	Long:  "Database management commands",
}

var dbImportCmd = &cobra.Command{
	Use:   "import [file]",
	Short: "Import database",
	Long:  "Imports a SQL file into the project database",
	Args:  cobra.ExactArgs(1),
	RunE:  runDbImport,
}

var dbExportCmd = &cobra.Command{
	Use:   "export [file]",
	Short: "Export database",
	Long:  "Exports the project database to a SQL file",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runDbExport,
}

var dbShellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Open database shell",
	Long:  "Opens a MySQL shell connected to the project database",
	RunE:  runDbShell,
}

var dbCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create database",
	Long:  "Creates the project database if it doesn't exist",
	RunE:  runDbCreate,
}

var dbDropCmd = &cobra.Command{
	Use:   "drop",
	Short: "Drop database",
	Long:  "Drops the project database (DESTRUCTIVE - use with caution)",
	RunE:  runDbDrop,
}

var dbResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset database",
	Long:  "Drops and recreates the project database (DESTRUCTIVE - use with caution)",
	RunE:  runDbReset,
}

var dbSnapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Database snapshots",
	Long:  "Create, restore, and manage database snapshots for quick backup/restore",
}

var dbSnapshotCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a snapshot",
	Long:  "Creates a compressed snapshot of the current database",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runDbSnapshotCreate,
}

var dbSnapshotRestoreCmd = &cobra.Command{
	Use:   "restore [name]",
	Short: "Restore a snapshot",
	Long:  "Restores the database from a snapshot",
	Args:  cobra.ExactArgs(1),
	RunE:  runDbSnapshotRestore,
}

var dbSnapshotListCmd = &cobra.Command{
	Use:   "list",
	Short: "List snapshots",
	Long:  "Lists all available snapshots for this project",
	RunE:  runDbSnapshotList,
}

var dbSnapshotDeleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Delete a snapshot",
	Long:  "Deletes a snapshot",
	Args:  cobra.ExactArgs(1),
	RunE:  runDbSnapshotDelete,
}

func init() {
	dbCmd.AddCommand(dbImportCmd)
	dbCmd.AddCommand(dbExportCmd)
	dbCmd.AddCommand(dbShellCmd)
	dbCmd.AddCommand(dbCreateCmd)
	dbCmd.AddCommand(dbDropCmd)
	dbCmd.AddCommand(dbResetCmd)
	dbCmd.AddCommand(dbSnapshotCmd)

	// Snapshot subcommands
	dbSnapshotCmd.AddCommand(dbSnapshotCreateCmd)
	dbSnapshotCmd.AddCommand(dbSnapshotRestoreCmd)
	dbSnapshotCmd.AddCommand(dbSnapshotListCmd)
	dbSnapshotCmd.AddCommand(dbSnapshotDeleteCmd)

	rootCmd.AddCommand(dbCmd)
}

// dbInfo holds database connection information
type dbInfo struct {
	ContainerName string // e.g., "magebox-mysql-8.0"
	Version       string // e.g., "8.0"
	Type          string // "mysql" or "mariadb"
	Port          int    // e.g., 33080
}

// getDbInfo extracts database connection info from project config
func getDbInfo(cfg *config.Config) (*dbInfo, error) {
	if cfg.Services.MySQL != nil && cfg.Services.MySQL.Enabled {
		version := cfg.Services.MySQL.Version
		port := getDbPort("mysql", version)
		return &dbInfo{
			ContainerName: fmt.Sprintf("magebox-mysql-%s", version),
			Version:       version,
			Type:          "mysql",
			Port:          port,
		}, nil
	}
	if cfg.Services.MariaDB != nil && cfg.Services.MariaDB.Enabled {
		version := cfg.Services.MariaDB.Version
		port := getDbPort("mariadb", version)
		return &dbInfo{
			ContainerName: fmt.Sprintf("magebox-mariadb-%s", version),
			Version:       version,
			Type:          "mariadb",
			Port:          port,
		}, nil
	}
	return nil, fmt.Errorf("no database service configured in %s", config.ConfigFileName)
}

// getDbPort returns the host port for a database version
func getDbPort(dbType, version string) int {
	if dbType == "mysql" {
		ports := map[string]int{
			"5.7": 33057,
			"8.0": 33080,
			"8.4": 33084,
		}
		if port, ok := ports[version]; ok {
			return port
		}
		return 33080
	}
	// MariaDB
	ports := map[string]int{
		"10.4":  33104,
		"10.5":  33105,
		"10.6":  33106,
		"10.11": 33111,
		"11.0":  33110,
		"11.4":  33114,
	}
	if port, ok := ports[version]; ok {
		return port
	}
	return 33106
}

func runDbImport(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	cfg, ok := loadProjectConfig(cwd)
	if !ok {
		return nil
	}

	db, err := getDbInfo(cfg)
	if err != nil {
		cli.PrintError("%v", err)
		return nil
	}

	sqlFile := args[0]
	fmt.Printf("Importing %s into database '%s' (%s)\n", filepath.Base(sqlFile), cfg.Name, db.ContainerName)

	// Create database if it doesn't exist
	createCmd := exec.Command("docker", "exec", db.ContainerName,
		"mysql", "-uroot", "-p"+docker.DefaultDBRootPassword, "-e",
		fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", cfg.Name))
	createCmd.Stderr = os.Stderr
	if err := createCmd.Run(); err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	// Get file info for progress tracking
	fileInfo, err := os.Stat(sqlFile)
	if err != nil {
		return fmt.Errorf("failed to stat SQL file: %w", err)
	}
	fileSize := fileInfo.Size()

	// Open file
	file, err := os.Open(sqlFile)
	if err != nil {
		return fmt.Errorf("failed to open SQL file: %w", err)
	}
	defer file.Close()

	// Create progress bar
	bar := progress.NewBar("Importing:")

	// Use docker exec directly with container name
	importCmd := exec.Command("docker", "exec", "-i", db.ContainerName,
		"mysql", "-uroot", "-p"+docker.DefaultDBRootPassword, cfg.Name)

	// Handle gzip compressed files
	if strings.HasSuffix(sqlFile, ".gz") {
		// For gzip, track compressed bytes read
		progressReader := progress.NewReader(file, fileSize, bar.Update)

		gzReader, err := gzip.NewReader(progressReader)
		if err != nil {
			return fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzReader.Close()

		importCmd.Stdin = gzReader
		importCmd.Stderr = io.Discard // Suppress mysql warnings

		if err := importCmd.Run(); err != nil {
			bar.Finish()
			return fmt.Errorf("import failed: %w", err)
		}
	} else {
		// For plain SQL, track bytes directly
		progressReader := progress.NewReader(file, fileSize, bar.Update)

		importCmd.Stdin = progressReader
		importCmd.Stderr = io.Discard // Suppress mysql warnings

		if err := importCmd.Run(); err != nil {
			bar.Finish()
			return fmt.Errorf("import failed: %w", err)
		}
	}

	bar.Finish()
	cli.PrintSuccess("Import completed successfully!")
	return nil
}

func runDbExport(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	cfg, ok := loadProjectConfig(cwd)
	if !ok {
		return nil
	}

	db, err := getDbInfo(cfg)
	if err != nil {
		cli.PrintError("%v", err)
		return nil
	}

	// Determine output file
	var outputFile string
	if len(args) > 0 {
		outputFile = args[0]
	} else {
		outputFile = fmt.Sprintf("%s.sql", cfg.Name)
	}

	fmt.Printf("Exporting database '%s' to %s (%s)...\n", cfg.Name, outputFile, db.ContainerName)

	// Use docker exec directly with container name
	// --no-tablespaces: Skip TABLESPACE statements (avoids permission issues on import)
	exportCmd := exec.Command("docker", "exec", db.ContainerName,
		"mysqldump", "-uroot", "-p"+docker.DefaultDBRootPassword, "--no-tablespaces", cfg.Name)

	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	exportCmd.Stdout = file
	exportCmd.Stderr = os.Stderr

	if err := exportCmd.Run(); err != nil {
		return fmt.Errorf("export failed: %w", err)
	}

	cli.PrintSuccess("Export completed: %s", outputFile)
	return nil
}

func runDbShell(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	cfg, ok := loadProjectConfig(cwd)
	if !ok {
		return nil
	}

	db, err := getDbInfo(cfg)
	if err != nil {
		cli.PrintError("%v", err)
		return nil
	}

	fmt.Printf("Connecting to database '%s' (%s)...\n", cfg.Name, db.ContainerName)

	// Use docker exec directly with container name
	shellCmd := exec.Command("docker", "exec", "-it", db.ContainerName,
		"mysql", "-uroot", "-p"+docker.DefaultDBRootPassword, cfg.Name)
	shellCmd.Stdin = os.Stdin
	shellCmd.Stdout = os.Stdout
	shellCmd.Stderr = os.Stderr

	return shellCmd.Run()
}

func runDbCreate(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	cfg, ok := loadProjectConfig(cwd)
	if !ok {
		return nil
	}

	db, err := getDbInfo(cfg)
	if err != nil {
		cli.PrintError("%v", err)
		return nil
	}

	cli.PrintTitle("Creating Database")
	fmt.Printf("Database: %s\n", cli.Highlight(cfg.Name))
	fmt.Printf("Container: %s\n", cli.Highlight(db.ContainerName))
	fmt.Println()

	// Check if database already exists
	checkCmd := exec.Command("docker", "exec", db.ContainerName,
		"mysql", "-uroot", "-p"+docker.DefaultDBRootPassword, "-e",
		fmt.Sprintf("SELECT SCHEMA_NAME FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME = '%s'", cfg.Name))
	output, err := checkCmd.Output()
	if err == nil && strings.Contains(string(output), cfg.Name) {
		cli.PrintInfo("Database '%s' already exists", cfg.Name)
		return nil
	}

	// Create database
	fmt.Print("Creating database... ")
	createCmd := exec.Command("docker", "exec", db.ContainerName,
		"mysql", "-uroot", "-p"+docker.DefaultDBRootPassword, "-e",
		fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", cfg.Name))
	createCmd.Stderr = os.Stderr

	if err := createCmd.Run(); err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to create database: %w", err)
	}
	fmt.Println(cli.Success("done"))

	fmt.Println()
	cli.PrintSuccess("Database '%s' created!", cfg.Name)
	return nil
}

func runDbDrop(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	cfg, ok := loadProjectConfig(cwd)
	if !ok {
		return nil
	}

	db, err := getDbInfo(cfg)
	if err != nil {
		cli.PrintError("%v", err)
		return nil
	}

	cli.PrintTitle("Drop Database")
	fmt.Printf("Database: %s\n", cli.Highlight(cfg.Name))
	fmt.Printf("Container: %s\n", cli.Highlight(db.ContainerName))
	fmt.Println()

	cli.PrintWarning("This will permanently delete the database '%s'!", cfg.Name)
	fmt.Print("Are you sure? [y/N]: ")

	var confirm string
	_, _ = fmt.Scanln(&confirm)
	if confirm != "y" && confirm != "Y" {
		cli.PrintInfo("Aborted")
		return nil
	}

	fmt.Println()
	fmt.Print("Dropping database... ")
	dropCmd := exec.Command("docker", "exec", db.ContainerName,
		"mysql", "-uroot", "-p"+docker.DefaultDBRootPassword, "-e",
		fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", cfg.Name))
	dropCmd.Stderr = os.Stderr

	if err := dropCmd.Run(); err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to drop database: %w", err)
	}
	fmt.Println(cli.Success("done"))

	fmt.Println()
	cli.PrintSuccess("Database '%s' dropped!", cfg.Name)
	return nil
}

func runDbReset(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	cfg, ok := loadProjectConfig(cwd)
	if !ok {
		return nil
	}

	db, err := getDbInfo(cfg)
	if err != nil {
		cli.PrintError("%v", err)
		return nil
	}

	cli.PrintTitle("Reset Database")
	fmt.Printf("Database: %s\n", cli.Highlight(cfg.Name))
	fmt.Printf("Container: %s\n", cli.Highlight(db.ContainerName))
	fmt.Println()

	cli.PrintWarning("This will permanently delete ALL DATA in database '%s'!", cfg.Name)
	fmt.Print("Are you sure? [y/N]: ")

	var confirm string
	_, _ = fmt.Scanln(&confirm)
	if confirm != "y" && confirm != "Y" {
		cli.PrintInfo("Aborted")
		return nil
	}

	fmt.Println()

	// Drop database
	fmt.Print("Dropping database... ")
	dropCmd := exec.Command("docker", "exec", db.ContainerName,
		"mysql", "-uroot", "-p"+docker.DefaultDBRootPassword, "-e",
		fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", cfg.Name))
	dropCmd.Stderr = os.Stderr

	if err := dropCmd.Run(); err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to drop database: %w", err)
	}
	fmt.Println(cli.Success("done"))

	// Create database
	fmt.Print("Creating database... ")
	createCmd := exec.Command("docker", "exec", db.ContainerName,
		"mysql", "-uroot", "-p"+docker.DefaultDBRootPassword, "-e",
		fmt.Sprintf("CREATE DATABASE `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", cfg.Name))
	createCmd.Stderr = os.Stderr

	if err := createCmd.Run(); err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to create database: %w", err)
	}
	fmt.Println(cli.Success("done"))

	fmt.Println()
	cli.PrintSuccess("Database '%s' reset!", cfg.Name)
	return nil
}

// getSnapshotDir returns the directory for storing snapshots
func getSnapshotDir(projectName string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".magebox", "snapshots", projectName)
}

// getSnapshotPath returns the full path for a snapshot file
func getSnapshotPath(projectName, snapshotName string) string {
	return filepath.Join(getSnapshotDir(projectName), snapshotName+".sql.gz")
}

func runDbSnapshotCreate(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	cfg, ok := loadProjectConfig(cwd)
	if !ok {
		return nil
	}

	db, err := getDbInfo(cfg)
	if err != nil {
		cli.PrintError("%v", err)
		return nil
	}

	// Determine snapshot name
	var snapshotName string
	if len(args) > 0 {
		snapshotName = args[0]
	} else {
		// Generate name with timestamp
		snapshotName = time.Now().Format("2006-01-02_15-04-05")
	}

	// Ensure snapshot directory exists
	snapshotDir := getSnapshotDir(cfg.Name)
	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		return fmt.Errorf("failed to create snapshot directory: %w", err)
	}

	snapshotPath := getSnapshotPath(cfg.Name, snapshotName)

	// Check if snapshot already exists
	if _, err := os.Stat(snapshotPath); err == nil {
		cli.PrintError("Snapshot '%s' already exists", snapshotName)
		cli.PrintInfo("Use a different name or delete the existing snapshot first")
		return nil
	}

	cli.PrintTitle("Creating Snapshot")
	fmt.Printf("Database:  %s\n", cli.Highlight(cfg.Name))
	fmt.Printf("Snapshot:  %s\n", cli.Highlight(snapshotName))
	fmt.Printf("Container: %s\n", cli.Highlight(db.ContainerName))
	fmt.Println()

	fmt.Print("Dumping database... ")

	// Create gzipped dump
	dumpCmd := exec.Command("docker", "exec", db.ContainerName,
		"mysqldump", "-uroot", "-p"+docker.DefaultDBRootPassword,
		"--no-tablespaces", "--single-transaction", cfg.Name)

	// Create output file with gzip compression
	outFile, err := os.Create(snapshotPath)
	if err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to create snapshot file: %w", err)
	}
	defer outFile.Close()

	gzWriter := gzip.NewWriter(outFile)
	defer gzWriter.Close()

	dumpCmd.Stdout = gzWriter
	dumpCmd.Stderr = os.Stderr

	if err := dumpCmd.Run(); err != nil {
		fmt.Println(cli.Error("failed"))
		os.Remove(snapshotPath)
		return fmt.Errorf("dump failed: %w", err)
	}

	// Close gzip writer to flush data
	gzWriter.Close()
	outFile.Close()

	// Get file size
	info, _ := os.Stat(snapshotPath)
	fmt.Println(cli.Success("done"))

	fmt.Println()
	cli.PrintSuccess("Snapshot '%s' created (%s)", snapshotName, formatFileSize(info.Size()))
	cli.PrintInfo("Restore with: magebox db snapshot restore %s", snapshotName)
	return nil
}

func runDbSnapshotRestore(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	cfg, ok := loadProjectConfig(cwd)
	if !ok {
		return nil
	}

	db, err := getDbInfo(cfg)
	if err != nil {
		cli.PrintError("%v", err)
		return nil
	}

	snapshotName := args[0]
	snapshotPath := getSnapshotPath(cfg.Name, snapshotName)

	// Check if snapshot exists
	info, err := os.Stat(snapshotPath)
	if os.IsNotExist(err) {
		cli.PrintError("Snapshot '%s' not found", snapshotName)
		cli.PrintInfo("Use 'magebox db snapshot list' to see available snapshots")
		return nil
	}

	cli.PrintTitle("Restore Snapshot")
	fmt.Printf("Database:  %s\n", cli.Highlight(cfg.Name))
	fmt.Printf("Snapshot:  %s\n", cli.Highlight(snapshotName))
	fmt.Printf("Size:      %s\n", formatFileSize(info.Size()))
	fmt.Printf("Container: %s\n", cli.Highlight(db.ContainerName))
	fmt.Println()

	cli.PrintWarning("This will replace ALL data in database '%s'!", cfg.Name)
	fmt.Print("Are you sure? [y/N]: ")

	var confirm string
	_, _ = fmt.Scanln(&confirm)
	if confirm != "y" && confirm != "Y" {
		cli.PrintInfo("Aborted")
		return nil
	}

	fmt.Println()

	// Drop and recreate database
	fmt.Print("Resetting database... ")
	resetCmd := exec.Command("docker", "exec", db.ContainerName,
		"mysql", "-uroot", "-p"+docker.DefaultDBRootPassword, "-e",
		fmt.Sprintf("DROP DATABASE IF EXISTS `%s`; CREATE DATABASE `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", cfg.Name, cfg.Name))
	resetCmd.Stderr = os.Stderr
	if err := resetCmd.Run(); err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to reset database: %w", err)
	}
	fmt.Println(cli.Success("done"))

	// Restore from snapshot
	fmt.Print("Restoring snapshot... ")

	// Open gzipped snapshot
	inFile, err := os.Open(snapshotPath)
	if err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to open snapshot: %w", err)
	}
	defer inFile.Close()

	gzReader, err := gzip.NewReader(inFile)
	if err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to decompress snapshot: %w", err)
	}
	defer gzReader.Close()

	// Import into database
	importCmd := exec.Command("docker", "exec", "-i", db.ContainerName,
		"mysql", "-uroot", "-p"+docker.DefaultDBRootPassword, cfg.Name)
	importCmd.Stdin = gzReader
	importCmd.Stderr = os.Stderr

	if err := importCmd.Run(); err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("restore failed: %w", err)
	}
	fmt.Println(cli.Success("done"))

	fmt.Println()
	cli.PrintSuccess("Snapshot '%s' restored!", snapshotName)
	return nil
}

func runDbSnapshotList(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	cfg, ok := loadProjectConfig(cwd)
	if !ok {
		return nil
	}

	snapshotDir := getSnapshotDir(cfg.Name)

	// Check if directory exists
	if _, err := os.Stat(snapshotDir); os.IsNotExist(err) {
		cli.PrintInfo("No snapshots found for project '%s'", cfg.Name)
		cli.PrintInfo("Create one with: magebox db snapshot create [name]")
		return nil
	}

	// List snapshot files
	entries, err := os.ReadDir(snapshotDir)
	if err != nil {
		return fmt.Errorf("failed to read snapshot directory: %w", err)
	}

	// Filter and collect snapshot info
	type snapshotInfo struct {
		Name    string
		Size    int64
		ModTime time.Time
	}
	var snapshots []snapshotInfo

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql.gz") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".sql.gz")
		snapshots = append(snapshots, snapshotInfo{
			Name:    name,
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
	}

	if len(snapshots) == 0 {
		cli.PrintInfo("No snapshots found for project '%s'", cfg.Name)
		cli.PrintInfo("Create one with: magebox db snapshot create [name]")
		return nil
	}

	// Sort by modification time (newest first)
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].ModTime.After(snapshots[j].ModTime)
	})

	cli.PrintTitle("Database Snapshots")
	fmt.Printf("Project: %s\n", cli.Highlight(cfg.Name))
	fmt.Println()

	fmt.Printf("%-30s  %-10s  %s\n", "NAME", "SIZE", "CREATED")
	fmt.Println(strings.Repeat("-", 60))

	for _, s := range snapshots {
		fmt.Printf("%-30s  %-10s  %s\n",
			s.Name,
			formatFileSize(s.Size),
			s.ModTime.Format("2006-01-02 15:04:05"))
	}

	fmt.Println()
	cli.PrintInfo("Restore with: magebox db snapshot restore <name>")
	return nil
}

func runDbSnapshotDelete(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	cfg, ok := loadProjectConfig(cwd)
	if !ok {
		return nil
	}

	snapshotName := args[0]
	snapshotPath := getSnapshotPath(cfg.Name, snapshotName)

	// Check if snapshot exists
	info, err := os.Stat(snapshotPath)
	if os.IsNotExist(err) {
		cli.PrintError("Snapshot '%s' not found", snapshotName)
		return nil
	}

	cli.PrintTitle("Delete Snapshot")
	fmt.Printf("Snapshot: %s\n", cli.Highlight(snapshotName))
	fmt.Printf("Size:     %s\n", formatFileSize(info.Size()))
	fmt.Println()

	fmt.Print("Are you sure you want to delete this snapshot? [y/N]: ")

	var confirm string
	_, _ = fmt.Scanln(&confirm)
	if confirm != "y" && confirm != "Y" {
		cli.PrintInfo("Aborted")
		return nil
	}

	if err := os.Remove(snapshotPath); err != nil {
		return fmt.Errorf("failed to delete snapshot: %w", err)
	}

	fmt.Println()
	cli.PrintSuccess("Snapshot '%s' deleted", snapshotName)
	return nil
}

// formatFileSize formats bytes as human-readable size
func formatFileSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

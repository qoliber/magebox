package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/qoliber/magebox/internal/cli"
	"github.com/qoliber/magebox/internal/config"
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

func init() {
	dbCmd.AddCommand(dbImportCmd)
	dbCmd.AddCommand(dbExportCmd)
	dbCmd.AddCommand(dbShellCmd)
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
	fmt.Printf("Importing %s into database '%s' (%s)...\n", sqlFile, cfg.Name, db.ContainerName)

	// Use docker exec directly with container name
	importCmd := exec.Command("docker", "exec", "-i", db.ContainerName,
		"mysql", "-uroot", "-pmagebox", cfg.Name)

	file, err := os.Open(sqlFile)
	if err != nil {
		return fmt.Errorf("failed to open SQL file: %w", err)
	}
	defer file.Close()

	importCmd.Stdin = file
	importCmd.Stdout = os.Stdout
	importCmd.Stderr = os.Stderr

	if err := importCmd.Run(); err != nil {
		return fmt.Errorf("import failed: %w", err)
	}

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
		"mysqldump", "-uroot", "-pmagebox", "--no-tablespaces", cfg.Name)

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
		"mysql", "-uroot", "-pmagebox", cfg.Name)
	shellCmd.Stdin = os.Stdin
	shellCmd.Stdout = os.Stdout
	shellCmd.Stderr = os.Stderr

	return shellCmd.Run()
}

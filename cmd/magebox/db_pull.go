package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"qoliber/magebox/internal/cli"
	"qoliber/magebox/internal/config"
	"qoliber/magebox/internal/docker"
	"qoliber/magebox/internal/progress"
	"qoliber/magebox/internal/remote"
)

var dbPullCmd = &cobra.Command{
	Use:   "pull [environment]",
	Short: "Pull database from remote environment",
	Long: `Pull a database from a remote environment using magerun2 db:dump over SSH.

The dump uses magerun's --strip option to exclude sensitive customer data.
Strip groups and extra exclude tables can be configured in .magebox.yaml.

Environments are configured via 'magebox env add'.

Configuration in .magebox.yaml:
  pull:
    default: staging              # Default environment
    strip: "@stripped @trade"     # Magerun strip groups (default: @stripped)
    exclude:                      # Extra tables to exclude
      - custom_log_table
    magerun: magerun2             # Magerun binary (default: magerun2)
    root_path: /data/web/current/ # Remote project root

Examples:
  magebox db pull                 # Pull from default environment
  magebox db pull staging         # Pull from staging
  magebox db pull staging --no-import    # Only dump, don't import locally
  magebox db pull staging --no-compress  # Skip compression (faster on fast networks)
  magebox db pull staging -y             # Skip confirmation prompt`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDbPull,
}

var (
	dbPullNoImport   bool
	dbPullNoStrip    bool
	dbPullNoCompress bool
	dbPullBackup     bool
	dbPullYes        bool
)

func init() {
	dbPullCmd.Flags().BoolVar(&dbPullNoImport, "no-import", false, "Only download the dump, don't import locally")
	dbPullCmd.Flags().BoolVar(&dbPullNoStrip, "no-strip", false, "Dump without stripping sensitive data")
	dbPullCmd.Flags().BoolVar(&dbPullNoCompress, "no-compress", false, "Don't compress the dump (faster on fast networks)")
	dbPullCmd.Flags().BoolVar(&dbPullBackup, "backup", false, "Create a local snapshot before importing")
	dbPullCmd.Flags().BoolVarP(&dbPullYes, "yes", "y", false, "Skip confirmation prompt")

	dbCmd.AddCommand(dbPullCmd)
}

// pullSettings holds the resolved configuration for a db pull operation
type pullSettings struct {
	cfg        *config.Config
	env        *remote.Environment
	envName    string
	rootPath   string
	magerun    string
	strip      string
	remoteCmd  string
	compressed bool
}

// resolvePullSettings loads and validates all configuration needed for a pull
func resolvePullSettings(args []string) (*pullSettings, error) {
	cwd, err := getCwd()
	if err != nil {
		return nil, err
	}

	cfg, ok := loadProjectConfig(cwd)
	if !ok {
		return nil, nil
	}

	// Determine target environment
	envName := ""
	if len(args) > 0 {
		envName = args[0]
	} else if cfg.Pull != nil && cfg.Pull.Default != "" {
		envName = cfg.Pull.Default
	}

	if envName == "" {
		cli.PrintError("No environment specified")
		fmt.Println()
		cli.PrintInfo("Usage: magebox db pull <environment>")
		cli.PrintInfo("Or set a default in .magebox.yaml:")
		fmt.Println("  pull:")
		fmt.Println("    default: staging")
		return nil, nil
	}

	// Load global config to get environment SSH details
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	globalCfg, err := config.LoadGlobalConfig(homeDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load global config: %w", err)
	}

	env, err := globalCfg.GetEnvironment(envName)
	if err != nil {
		cli.PrintError("Environment '%s' not found", envName)
		fmt.Println()
		cli.PrintInfo("Add it with: magebox env add %s --user <user> --host <host>", envName)
		if len(globalCfg.Environments) > 0 {
			fmt.Println()
			cli.PrintInfo("Available environments:")
			for _, e := range globalCfg.Environments {
				fmt.Printf("  - %s (%s)\n", e.Name, e.GetConnectionString())
			}
		}
		return nil, nil
	}

	// Determine root path: environment > pull config > error
	rootPath := env.RootPath
	if rootPath == "" && cfg.Pull != nil {
		rootPath = cfg.Pull.GetRootPath()
	}
	if rootPath == "" {
		cli.PrintError("No root_path configured for environment '%s'", envName)
		fmt.Println()
		cli.PrintInfo("Set it on the environment:")
		fmt.Printf("  In ~/.magebox/config.yaml under the '%s' environment: root_path: /data/web/current/\n", envName)
		fmt.Println()
		cli.PrintInfo("Or set a default in .magebox.yaml:")
		fmt.Println("  pull:")
		fmt.Println("    root_path: /data/web/project/current/")
		return nil, nil
	}

	// Build magerun strip argument
	pullCfg := cfg.Pull
	magerun := pullCfg.GetMagerun()
	strip := pullCfg.GetStrip()
	if pullCfg != nil && len(pullCfg.Exclude) > 0 {
		strip = strip + " " + strings.Join(pullCfg.Exclude, " ")
	}

	// Build remote command
	compressed := !dbPullNoCompress
	dumpCmd := fmt.Sprintf("cd %s && %s db:dump", rootPath, magerun)
	if !dbPullNoStrip {
		dumpCmd += fmt.Sprintf(" --strip='%s'", strip)
	}
	dumpCmd += " --stdout"

	remoteCmd := dumpCmd
	if compressed {
		remoteCmd += " | gzip"
	}

	return &pullSettings{
		cfg:        cfg,
		env:        env,
		envName:    envName,
		rootPath:   rootPath,
		magerun:    magerun,
		strip:      strip,
		remoteCmd:  remoteCmd,
		compressed: compressed,
	}, nil
}

// printPullOverview displays the pull configuration summary
func printPullOverview(s *pullSettings) {
	cli.PrintTitle("Database Pull")
	fmt.Println()
	fmt.Printf("  Environment: %s\n", cli.Highlight(s.envName))
	fmt.Printf("  Connection:  %s\n", s.env.GetConnectionString())
	fmt.Printf("  Root path:   %s\n", cli.Highlight(s.rootPath))
	fmt.Printf("  Magerun:     %s\n", s.magerun)
	if !dbPullNoStrip {
		fmt.Printf("  Strip:       %s\n", s.strip)
	} else {
		fmt.Printf("  Strip:       %s\n", cli.Warning("disabled"))
	}

	if !dbPullNoImport {
		db, err := getDbInfo(s.cfg)
		if err == nil {
			fmt.Printf("  Target DB:   %s (%s)\n", cli.Highlight(s.cfg.DatabaseName()), db.ContainerName)
		}
	}
	fmt.Println()
}

// confirmPull asks for user confirmation before overwriting the database
func confirmPull(envName string) bool {
	if dbPullYes {
		return true
	}
	if strings.Contains(strings.ToLower(envName), "prod") {
		cli.PrintWarning("You are pulling from a PRODUCTION environment!")
	}
	cli.PrintWarning("This will overwrite the local database!")
	fmt.Print("Continue? [y/N]: ")
	var confirm string
	_, _ = fmt.Scanln(&confirm)
	return confirm == "y" || confirm == "Y"
}

func runDbPull(cmd *cobra.Command, args []string) error {
	s, err := resolvePullSettings(args)
	if err != nil {
		return err
	}
	if s == nil {
		return nil
	}

	printPullOverview(s)

	// Confirmation before overwriting local database
	if !dbPullNoImport {
		if !confirmPull(s.envName) {
			cli.PrintInfo("Aborted")
			return nil
		}
		fmt.Println()
	}

	// Backup current database if requested
	if dbPullBackup && !dbPullNoImport {
		if err := createPrePullBackup(s.cfg); err != nil {
			cli.PrintWarning("Backup failed: %v", err)
			if !dbPullYes {
				fmt.Print("Continue without backup? [y/N]: ")
				var confirm string
				_, _ = fmt.Scanln(&confirm)
				if confirm != "y" && confirm != "Y" {
					cli.PrintInfo("Aborted")
					return nil
				}
			}
		}
		fmt.Println()
	}

	startTime := time.Now()

	if dbPullNoImport {
		err = pullToFile(s)
	} else {
		err = pullAndImport(s)
	}

	if err == nil {
		elapsed := time.Since(startTime)
		cli.PrintInfo("Completed in %s", formatElapsed(elapsed))
	}

	return err
}

// pullToFile downloads the dump to a local file
func pullToFile(s *pullSettings) error {
	ext := ".sql.gz"
	if !s.compressed {
		ext = ".sql"
	}
	outputFile := fmt.Sprintf("%s-pull%s", s.cfg.Name, ext)

	fmt.Printf("Pulling database to %s...\n", cli.Highlight(outputFile))

	outFile, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	sshCmd := s.env.BuildRemoteCommand(s.remoteCmd)
	sshStdout, err := sshCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	sshCmd.Stderr = newWarningFilter(os.Stderr)

	if err := sshCmd.Start(); err != nil {
		return fmt.Errorf("failed to start SSH: %w", err)
	}

	bar := progress.NewBar("Downloading:")
	progressReader := progress.NewReader(sshStdout, 0, bar.Update)

	if _, err := io.Copy(outFile, progressReader); err != nil {
		_ = sshCmd.Process.Kill()
		bar.Finish()
		os.Remove(outputFile)
		return fmt.Errorf("pull failed: %w", err)
	}

	if err := sshCmd.Wait(); err != nil {
		bar.Finish()
		os.Remove(outputFile)
		return fmt.Errorf("pull failed: %w", err)
	}

	bar.Finish()
	info, _ := os.Stat(outputFile)
	cli.PrintSuccess("Database dumped to %s (%s)", outputFile, formatFileSize(info.Size()))
	cli.PrintInfo("Import with: magebox db import %s", outputFile)
	return nil
}

// pullAndImport streams the remote dump directly into the local database.
// With compression: SSH → [Go byte counter] → gunzip → kernel pipe → mysql
// Without compression: SSH → [Go byte counter] → mysql
func pullAndImport(s *pullSettings) error {
	db, err := getDbInfo(s.cfg)
	if err != nil {
		return err
	}

	dbName := s.cfg.DatabaseName()

	// Ensure database exists
	createCmd := exec.Command("docker", "exec", db.ContainerName,
		"mysql", "-uroot", "-p"+docker.DefaultDBRootPassword, "-e",
		fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", dbName))
	createCmd.Stderr = io.Discard
	if err := createCmd.Run(); err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	fmt.Printf("Pulling and importing into '%s'...\n", cli.Highlight(dbName))
	fmt.Println()

	// SSH command
	sshCmd := s.env.BuildRemoteCommand(s.remoteCmd)
	sshStdout, err := sshCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create SSH stdout pipe: %w", err)
	}
	sshCmd.Stderr = newWarningFilter(os.Stderr)

	// Progress counter on SSH output
	bar := progress.NewBar("Downloading:")
	progressReader := progress.NewReader(sshStdout, 0, bar.Update)

	// mysql import process — capture stderr for error reporting
	var mysqlStderr bytes.Buffer
	mysqlCmd := exec.Command("docker", "exec", "-i", db.ContainerName,
		"mysql", "-uroot", "-p"+docker.DefaultDBRootPassword, dbName)
	mysqlCmd.Stderr = &mysqlStderr

	var gunzipCmd *exec.Cmd
	if s.compressed {
		gunzipCmd = exec.Command("gunzip")
		gunzipCmd.Stdin = progressReader

		gunzipOut, err := gunzipCmd.StdoutPipe()
		if err != nil {
			return fmt.Errorf("failed to create gunzip pipe: %w", err)
		}
		mysqlCmd.Stdin = gunzipOut
	} else {
		mysqlCmd.Stdin = progressReader
	}

	// Start all processes
	if err := sshCmd.Start(); err != nil {
		return fmt.Errorf("failed to start SSH: %w", err)
	}
	if s.compressed {
		if err := gunzipCmd.Start(); err != nil {
			_ = sshCmd.Process.Kill()
			return fmt.Errorf("failed to start gunzip: %w", err)
		}
	}
	if err := mysqlCmd.Start(); err != nil {
		_ = sshCmd.Process.Kill()
		if s.compressed {
			_ = gunzipCmd.Process.Kill()
		}
		return fmt.Errorf("failed to start mysql import: %w", err)
	}

	// Wait for processes in reverse order
	mysqlErr := mysqlCmd.Wait()
	var gunzipErr error
	if s.compressed {
		gunzipErr = gunzipCmd.Wait()
	}
	sshErr := sshCmd.Wait()

	bar.Finish()

	if sshErr != nil {
		return fmt.Errorf("SSH failed: %w", sshErr)
	}
	if gunzipErr != nil {
		return fmt.Errorf("gunzip failed: %w", gunzipErr)
	}
	if mysqlErr != nil {
		errMsg := strings.TrimSpace(mysqlStderr.String())
		errMsg = filterWarnings(errMsg)
		if errMsg != "" {
			return fmt.Errorf("mysql import failed: %s", errMsg)
		}
		return fmt.Errorf("mysql import failed: %w", mysqlErr)
	}

	cli.PrintSuccess("Database pulled and imported from '%s'!", s.env.Name)
	return nil
}

// createPrePullBackup creates a database snapshot before pulling
func createPrePullBackup(cfg *config.Config) error {
	db, err := getDbInfo(cfg)
	if err != nil {
		return err
	}

	dbName := cfg.DatabaseName()
	snapshotName := "pre-pull"
	snapshotDir := getSnapshotDir(cfg.Name)

	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		return fmt.Errorf("failed to create snapshot directory: %w", err)
	}

	snapshotPath := getSnapshotPath(cfg.Name, snapshotName)

	// Remove existing pre-pull snapshot
	os.Remove(snapshotPath)

	fmt.Print("Creating backup snapshot... ")

	dumpCmd := exec.Command("docker", "exec", db.ContainerName,
		"mysqldump", "-uroot", "-p"+docker.DefaultDBRootPassword,
		"--no-tablespaces", "--single-transaction", dbName)

	outFile, err := os.Create(snapshotPath)
	if err != nil {
		fmt.Println(cli.Error("failed"))
		return err
	}
	defer outFile.Close()

	gzWriter := gzip.NewWriter(outFile)
	defer gzWriter.Close()

	dumpCmd.Stdout = gzWriter
	dumpCmd.Stderr = io.Discard

	if err := dumpCmd.Run(); err != nil {
		fmt.Println(cli.Error("failed"))
		os.Remove(snapshotPath)
		return err
	}

	gzWriter.Close()
	outFile.Close()

	fmt.Println(cli.Success("done"))
	return nil
}

// warningFilter filters out common noisy warnings from stderr.
// Buffers partial lines so warnings split across Write calls are still caught.
type warningFilter struct {
	inner io.Writer
	buf   []byte
}

func newWarningFilter(w io.Writer) *warningFilter {
	return &warningFilter{inner: w}
}

func (w *warningFilter) Write(p []byte) (n int, err error) {
	w.buf = append(w.buf, p...)

	for {
		idx := bytes.IndexByte(w.buf, '\n')
		if idx < 0 {
			break
		}
		line := string(w.buf[:idx+1])
		w.buf = w.buf[idx+1:]

		if isWarningLine(line) {
			continue
		}
		if _, err := w.inner.Write([]byte(line)); err != nil {
			return len(p), err
		}
	}

	return len(p), nil
}

// Flush writes any remaining buffered data
func (w *warningFilter) Flush() error {
	if len(w.buf) > 0 && !isWarningLine(string(w.buf)) {
		_, err := w.inner.Write(w.buf)
		w.buf = nil
		return err
	}
	w.buf = nil
	return nil
}

func isWarningLine(line string) bool {
	return strings.Contains(line, "Using a password on the command line") ||
		strings.Contains(line, "mysqldump: [Warning]")
}

// filterWarnings removes common mysql warning lines from error output
func filterWarnings(s string) string {
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.Contains(trimmed, "Using a password on the command line") {
			continue
		}
		lines = append(lines, trimmed)
	}
	return strings.Join(lines, "\n")
}

// formatElapsed formats a duration as a human-readable string
func formatElapsed(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm%ds", m, s)
}

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"qoliber/magebox/internal/cli"
	"qoliber/magebox/internal/config"
	"qoliber/magebox/internal/docker"
	"qoliber/magebox/internal/nginx"
	"qoliber/magebox/internal/platform"
	"qoliber/magebox/internal/ssl"
)

var exposeCmd = &cobra.Command{
	Use:   "expose [domain]",
	Short: "Expose project via Cloudflare Tunnel",
	Long: `Creates a public Cloudflare Tunnel URL pointing to your local project.

Uses cloudflared quick tunnels (no account required) to generate a
temporary *.trycloudflare.com URL. The tunnel domain is added to
.magebox.yaml so nginx serves it alongside your local domains.

Automatically updates Magento base URLs (across all scopes) to the
tunnel URL and reverts them when the tunnel is stopped.

Requires cloudflared to be installed (brew install cloudflared).`,
	Args: cobra.MaximumNArgs(1),
	RunE: runExpose,
}

var exposeStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the tunnel",
	Long:  "Stops the running Cloudflare Tunnel and reverts Magento base URLs",
	RunE:  runExposeStop,
}

var exposeStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show tunnel status",
	Long:  "Shows whether a tunnel is running for the current project",
	RunE:  runExposeStatus,
}

func init() {
	exposeCmd.AddCommand(exposeStopCmd)
	exposeCmd.AddCommand(exposeStatusCmd)
	rootCmd.AddCommand(exposeCmd)
}

// baseURLConfigPaths lists the Magento config paths for base URLs
var baseURLConfigPaths = []string{
	"web/unsecure/base_url",
	"web/secure/base_url",
	"web/unsecure/base_media_url",
	"web/secure/base_media_url",
	"web/unsecure/base_static_url",
	"web/secure/base_static_url",
}

// savedConfigRow represents a single core_config_data row
type savedConfigRow struct {
	Scope   string `json:"scope"`
	ScopeID string `json:"scope_id"`
	Path    string `json:"path"`
	Value   string `json:"value"`
}

func runExpose(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	p, err := getPlatform()
	if err != nil {
		return err
	}

	cfg, ok := loadProjectConfig(cwd)
	if !ok {
		return nil
	}

	// Check cloudflared is installed
	cloudflaredPath, err := exec.LookPath("cloudflared")
	if err != nil {
		cli.PrintError("cloudflared is not installed")
		cli.PrintInfo("Install it with: brew install cloudflared")
		return nil
	}

	// Determine which domain to expose
	var domain string
	if len(args) > 0 {
		domain = args[0]
		found := false
		for _, d := range cfg.Domains {
			if d.Host == domain {
				found = true
				break
			}
		}
		if !found {
			cli.PrintError("Domain '%s' is not configured for this project", domain)
			cli.PrintInfo("Available domains:")
			for _, d := range cfg.Domains {
				fmt.Printf("  %s\n", d.Host)
			}
			return nil
		}
	} else {
		if len(cfg.Domains) == 0 {
			cli.PrintError("No domains configured for this project")
			return nil
		}
		domain = cfg.Domains[0].Host
	}

	// Check if a tunnel is already running for this project
	pidFile := getTunnelPidFile(p, cfg.Name)
	if pid, err := readPidFile(pidFile); err == nil {
		if processRunning(pid) {
			cli.PrintWarning("A tunnel is already running for this project (PID %d)", pid)
			cli.PrintInfo("Stop it first with: magebox expose stop")
			return nil
		}
		os.Remove(pidFile)
	}

	// Get database info for direct SQL operations
	db, err := getDbInfo(cfg)
	if err != nil {
		cli.PrintWarning("Could not determine database info: %v", err)
	}

	// Determine local URL — cloudflared connects to nginx HTTP
	httpPort := 80
	if p.Type == platform.Darwin {
		httpPort = 8080
	}
	localURL := fmt.Sprintf("http://localhost:%d", httpPort)

	cli.PrintTitle("Exposing Project")
	fmt.Printf("Project: %s\n", cli.Highlight(cfg.Name))
	fmt.Printf("Domain:  %s\n", cli.Highlight(domain))
	fmt.Printf("Backend: %s\n", cli.Highlight(localURL))
	fmt.Println()

	// Save current base URLs from all scopes via direct SQL
	urlsFile := getTunnelBaseURLsFile(p, cfg.Name)
	if db != nil {
		saveBaseURLsFromDB(db, cfg.DatabaseName(), urlsFile, domain)
	}

	// Remove any locked base URLs from env.php (they override core_config_data)
	envBackupFile := getTunnelEnvBackupFile(p, cfg.Name)
	removeLockedBaseURLsFromEnv(cwd, p.PHPBinary(cfg.PHP), envBackupFile)

	// Start cloudflared tunnel — the tunnel hostname passes through as Host
	tunnelCmd := exec.Command(cloudflaredPath, "tunnel", "--url", localURL)
	tunnelCmd.Env = os.Environ()

	stderr, err := tunnelCmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := tunnelCmd.Start(); err != nil {
		return fmt.Errorf("failed to start cloudflared: %w", err)
	}

	if err := writePidFile(pidFile, tunnelCmd.Process.Pid); err != nil {
		cli.PrintWarning("Could not save tunnel PID: %v", err)
	}

	urlFile := getTunnelURLFile(p, cfg.Name)
	phpBin := p.PHPBinary(cfg.PHP)

	urlPattern := regexp.MustCompile(`https://[a-zA-Z0-9-]+\.trycloudflare\.com`)
	scanner := bufio.NewScanner(stderr)
	tunnelURL := ""

	fmt.Print("Starting tunnel... ")

	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			if match := urlPattern.FindString(line); match != "" && tunnelURL == "" {
				tunnelURL = match
				fmt.Println(cli.Success("done"))
				fmt.Println()

				_ = os.WriteFile(urlFile, []byte(tunnelURL), 0644)

				tunnelHost := extractHostname(tunnelURL)

				// Add tunnel domain to .magebox.yaml and regenerate nginx vhosts
				addTunnelDomain(p, cwd, cfg, tunnelHost, domain)

				// Update base URLs across all scopes via direct SQL
				if db != nil {
					updateBaseURLsInDB(db, cfg.DatabaseName(), tunnelURL+"/")
				}

				// Flush Magento cache
				flushMagentoCache(phpBin, cwd)

				fmt.Println()
				fmt.Printf("Public URL: %s\n", cli.URL(tunnelURL))
				fmt.Println()
				cli.PrintInfo("Press Ctrl+C to stop the tunnel and revert URLs")
				fmt.Println()
			}
		}
	}()

	// Handle Ctrl+C gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println()

		// Revert base URLs from saved data
		if db != nil {
			revertBaseURLsFromDB(db, cfg.DatabaseName(), urlsFile)
		}
		restoreLockedBaseURLsToEnv(cwd, envBackupFile)
		flushMagentoCache(phpBin, cwd)

		// Remove tunnel domain from .magebox.yaml and regenerate nginx vhosts
		removeTunnelDomain(p, cwd, cfg)

		fmt.Print("Stopping tunnel... ")
		_ = tunnelCmd.Process.Signal(syscall.SIGTERM)
	}()

	_ = tunnelCmd.Wait()

	// Clean up run files
	os.Remove(pidFile)
	os.Remove(urlFile)
	os.Remove(urlsFile)
	os.Remove(envBackupFile)

	if tunnelURL == "" {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("cloudflared exited without providing a tunnel URL")
	}

	fmt.Println(cli.Success("stopped"))
	return nil
}

func runExposeStop(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	p, err := getPlatform()
	if err != nil {
		return err
	}

	cfg, ok := loadProjectConfig(cwd)
	if !ok {
		return nil
	}

	pidFile := getTunnelPidFile(p, cfg.Name)
	urlsFile := getTunnelBaseURLsFile(p, cfg.Name)
	envBackupFile := getTunnelEnvBackupFile(p, cfg.Name)
	phpBin := p.PHPBinary(cfg.PHP)

	// Revert URLs and remove tunnel domain
	db, _ := getDbInfo(cfg)
	if db != nil {
		revertBaseURLsFromDB(db, cfg.DatabaseName(), urlsFile)
	}
	restoreLockedBaseURLsToEnv(cwd, envBackupFile)
	flushMagentoCache(phpBin, cwd)
	removeTunnelDomain(p, cwd, cfg)

	pid, err := readPidFile(pidFile)
	if err != nil || !processRunning(pid) {
		cli.PrintInfo("No tunnel process is running for this project")
		os.Remove(pidFile)
		os.Remove(getTunnelURLFile(p, cfg.Name))
		os.Remove(urlsFile)
		os.Remove(envBackupFile)
		return nil
	}

	fmt.Print("Stopping tunnel... ")
	process, err := os.FindProcess(pid)
	if err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to find process: %w", err)
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to stop tunnel: %w", err)
	}

	os.Remove(pidFile)
	os.Remove(getTunnelURLFile(p, cfg.Name))
	os.Remove(urlsFile)
	os.Remove(envBackupFile)
	fmt.Println(cli.Success("done"))

	return nil
}

func runExposeStatus(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	p, err := getPlatform()
	if err != nil {
		return err
	}

	cfg, ok := loadProjectConfig(cwd)
	if !ok {
		return nil
	}

	cli.PrintTitle("Tunnel Status")
	fmt.Printf("Project: %s\n", cli.Highlight(cfg.Name))
	fmt.Println()

	pidFile := getTunnelPidFile(p, cfg.Name)
	pid, err := readPidFile(pidFile)
	if err != nil || !processRunning(pid) {
		fmt.Println("Status: " + cli.Warning("not running"))
		fmt.Println()
		cli.PrintInfo("Start a tunnel with: magebox expose")
		return nil
	}

	fmt.Println("Status: " + cli.Success("running"))
	fmt.Printf("PID:    %d\n", pid)

	urlFile := getTunnelURLFile(p, cfg.Name)
	if urlBytes, err := os.ReadFile(urlFile); err == nil {
		tunnelURL := strings.TrimSpace(string(urlBytes))
		if tunnelURL != "" {
			fmt.Printf("URL:    %s\n", cli.URL(tunnelURL))
		}
	}

	return nil
}

// saveBaseURLsFromDB reads all base URL entries from core_config_data and saves them.
// If stale tunnel URLs from a previous run are detected, they are replaced with
// URLs derived from the local domain before saving.
func saveBaseURLsFromDB(db *dbInfo, dbName, urlsFile, localDomain string) {
	fmt.Print("Saving current base URLs... ")

	// Build WHERE clause for all URL paths
	pathConditions := make([]string, len(baseURLConfigPaths))
	for i, p := range baseURLConfigPaths {
		pathConditions[i] = fmt.Sprintf("'%s'", p)
	}
	query := fmt.Sprintf(
		"SELECT scope, scope_id, path, value FROM core_config_data WHERE path IN (%s)",
		strings.Join(pathConditions, ","),
	)

	cmd := exec.Command("docker", "exec", db.ContainerName,
		"mysql", "-uroot", "-p"+docker.DefaultDBRootPassword,
		"-N", "-B", dbName, "-e", query)
	out, err := cmd.Output()
	if err != nil {
		fmt.Println(cli.Warning("skipped"))
		return
	}

	localBaseURL := fmt.Sprintf("https://%s/", localDomain)

	var rows []savedConfigRow
	hasStale := false
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		fields := strings.SplitN(line, "\t", 4)
		if len(fields) == 4 {
			value := fields[3]

			// Detect stale tunnel URLs from a previous run that wasn't cleanly reverted
			if strings.Contains(value, ".trycloudflare.com") {
				hasStale = true
				// Derive the correct local URL from the path type
				switch {
				case strings.Contains(fields[2], "media"):
					value = localBaseURL + "media/"
				case strings.Contains(fields[2], "static"):
					value = localBaseURL + "static/"
				default:
					value = localBaseURL
				}
			}

			rows = append(rows, savedConfigRow{
				Scope:   fields[0],
				ScopeID: fields[1],
				Path:    fields[2],
				Value:   value,
			})
		}
	}

	// If stale tunnel URLs were found, also fix them in the database now
	if hasStale {
		cli.PrintWarning("Found stale tunnel URLs from a previous run, fixing...")
		revertBaseURLsFromSaved(db, dbName, rows)
		fmt.Print("Saving current base URLs... ")
	}

	data, err := json.Marshal(rows)
	if err != nil {
		fmt.Println(cli.Warning("skipped"))
		return
	}

	dir := filepath.Dir(urlsFile)
	_ = os.MkdirAll(dir, 0755)
	if err := os.WriteFile(urlsFile, data, 0644); err != nil {
		fmt.Println(cli.Warning("skipped"))
		return
	}

	fmt.Printf("%s (%d entries)\n", cli.Success("done"), len(rows))
}

// revertBaseURLsFromSaved restores base URLs from a slice of saved rows
func revertBaseURLsFromSaved(db *dbInfo, dbName string, rows []savedConfigRow) {
	var statements []string
	for _, row := range rows {
		statements = append(statements,
			fmt.Sprintf("UPDATE core_config_data SET value='%s' WHERE scope='%s' AND scope_id=%s AND path='%s'",
				row.Value, row.Scope, row.ScopeID, row.Path),
		)
	}

	query := strings.Join(statements, "; ")
	cmd := exec.Command("docker", "exec", db.ContainerName,
		"mysql", "-uroot", "-p"+docker.DefaultDBRootPassword,
		dbName, "-e", query)
	_ = cmd.Run()
}

// updateBaseURLsInDB replaces all base URL values in core_config_data with the tunnel URL
func updateBaseURLsInDB(db *dbInfo, dbName, tunnelBaseURL string) {
	fmt.Print("Updating Magento base URLs (all scopes)... ")

	// Build a single UPDATE that replaces the domain in all base URL entries.
	// For media URLs: value ends with /media/, for static: /static/, for base: just /
	// We set them all to the tunnel URL with the appropriate suffix.
	var statements []string
	for _, path := range baseURLConfigPaths {
		var newValue string
		switch {
		case strings.Contains(path, "media"):
			newValue = tunnelBaseURL + "media/"
		case strings.Contains(path, "static"):
			newValue = tunnelBaseURL + "static/"
		default:
			newValue = tunnelBaseURL
		}
		statements = append(statements,
			fmt.Sprintf("UPDATE core_config_data SET value='%s' WHERE path='%s'", newValue, path),
		)
	}

	query := strings.Join(statements, "; ")
	cmd := exec.Command("docker", "exec", db.ContainerName,
		"mysql", "-uroot", "-p"+docker.DefaultDBRootPassword,
		dbName, "-e", query)
	if err := cmd.Run(); err != nil {
		fmt.Println(cli.Error("failed"))
		return
	}
	fmt.Println(cli.Success("done"))
}

// revertBaseURLsFromDB restores all base URL entries from the saved JSON file
func revertBaseURLsFromDB(db *dbInfo, dbName, urlsFile string) {
	data, err := os.ReadFile(urlsFile)
	if err != nil {
		return
	}

	var rows []savedConfigRow
	if err := json.Unmarshal(data, &rows); err != nil {
		return
	}

	fmt.Print("Reverting Magento base URLs (all scopes)... ")
	revertBaseURLsFromSaved(db, dbName, rows)
	fmt.Println(cli.Success("done"))
}

// removeLockedBaseURLsFromEnv removes any locked web/unsecure/base_url and
// web/secure/base_url entries from env.php. These entries override
// core_config_data and would prevent our SQL-based URL changes from taking effect.
// A backup of the original env.php section is saved for restoration.
func removeLockedBaseURLsFromEnv(projectPath, phpBin, backupFile string) {
	envFile := filepath.Join(projectPath, "app", "etc", "env.php")
	content, err := os.ReadFile(envFile)
	if err != nil {
		return
	}

	contentStr := string(content)

	// Check if there are any locked base URLs in the system > default > web section
	if !strings.Contains(contentStr, "'web'") || !strings.Contains(contentStr, "'base_url'") {
		return
	}

	// Use PHP to safely remove the web section from the system default config
	// This is safer than regex on PHP array syntax
	phpCode := `<?php
$env = include '` + envFile + `';
$backup = [];
if (isset($env['system']['default']['web'])) {
    $backup = $env['system']['default']['web'];
    unset($env['system']['default']['web']);
    if (empty($env['system']['default'])) {
        unset($env['system']['default']);
    }
    if (empty($env['system'])) {
        unset($env['system']);
    }
    $content = "<?php\nreturn " . var_export($env, true) . ";\n";
    // Fix short array syntax
    $content = preg_replace('/array \(/', '[', $content);
    $content = preg_replace('/\)$/', ']', $content);
    $content = preg_replace('/\),/', '],', $content);
    file_put_contents('` + envFile + `', $content);
    echo json_encode($backup);
}
`
	fmt.Print("Removing locked base URLs from env.php... ")
	cmd := exec.Command(phpBin, "-r", phpCode)
	cmd.Dir = projectPath
	out, err := cmd.Output()
	if err != nil || len(out) == 0 {
		fmt.Println(cli.Warning("skipped"))
		return
	}

	// Save the backup
	dir := filepath.Dir(backupFile)
	_ = os.MkdirAll(dir, 0755)
	_ = os.WriteFile(backupFile, out, 0644)
	fmt.Println(cli.Success("done"))
}

// restoreLockedBaseURLsToEnv restores the web section to env.php from backup
func restoreLockedBaseURLsToEnv(projectPath, backupFile string) {
	backupData, err := os.ReadFile(backupFile)
	if err != nil || len(backupData) == 0 {
		return
	}

	envFile := filepath.Join(projectPath, "app", "etc", "env.php")

	// Use a simple PHP script to restore the web config
	// We find the PHP binary by looking at the env.php itself
	phpBin := "php" // fallback

	phpCode := `<?php
$env = include '` + envFile + `';
$web = json_decode('` + strings.ReplaceAll(string(backupData), "'", "\\'") + `', true);
if ($web) {
    if (!isset($env['system'])) $env['system'] = [];
    if (!isset($env['system']['default'])) $env['system']['default'] = [];
    $env['system']['default']['web'] = $web;
    $content = "<?php\nreturn " . var_export($env, true) . ";\n";
    $content = preg_replace('/array \(/', '[', $content);
    $content = preg_replace('/\)$/', ']', $content);
    $content = preg_replace('/\),/', '],', $content);
    file_put_contents('` + envFile + `', $content);
    echo "ok";
}
`
	fmt.Print("Restoring locked base URLs to env.php... ")
	cmd := exec.Command(phpBin, "-r", phpCode)
	cmd.Dir = projectPath
	if out, err := cmd.Output(); err == nil && strings.TrimSpace(string(out)) == "ok" {
		fmt.Println(cli.Success("done"))
	} else {
		fmt.Println(cli.Warning("skipped"))
	}

	os.Remove(backupFile)
}

// getTunnelEnvBackupFile returns the path to the env.php web config backup
func getTunnelEnvBackupFile(p *platform.Platform, projectName string) string {
	return filepath.Join(p.MageBoxDir(), "run", fmt.Sprintf("tunnel-%s.envweb.json", projectName))
}

// addTunnelDomain adds the tunnel hostname as a domain to .magebox.yaml,
// regenerates nginx vhosts, and reloads nginx. The tunnel domain uses the
// same document root as the source domain with SSL disabled (Cloudflare
// terminates SSL).
func addTunnelDomain(p *platform.Platform, cwd string, cfg *config.Config, tunnelHost, sourceDomain string) {
	fmt.Print("Adding tunnel domain to config... ")

	// Find the source domain's root
	var root string
	for _, d := range cfg.Domains {
		if d.Host == sourceDomain {
			root = d.Root
			break
		}
	}

	sslDisabled := false
	cfg.Domains = append(cfg.Domains, config.Domain{
		Host: tunnelHost,
		Root: root,
		SSL:  &sslDisabled,
	})

	if err := config.SaveToPath(cfg, cwd); err != nil {
		fmt.Println(cli.Error("failed: " + err.Error()))
		return
	}
	fmt.Println(cli.Success("done"))

	// Regenerate nginx vhosts and reload
	regenNginxVhosts(p, cfg, cwd)
}

// removeTunnelDomain removes any *.trycloudflare.com domain from .magebox.yaml,
// removes its vhost file, regenerates remaining vhosts, and reloads nginx.
func removeTunnelDomain(p *platform.Platform, cwd string, cfg *config.Config) {
	// Re-read config in case it changed
	freshCfg, err := config.LoadFromPath(cwd)
	if err != nil {
		return
	}

	var removedHosts []string
	newDomains := make([]config.Domain, 0, len(freshCfg.Domains))
	for _, d := range freshCfg.Domains {
		if strings.HasSuffix(d.Host, ".trycloudflare.com") {
			removedHosts = append(removedHosts, d.Host)
		} else {
			newDomains = append(newDomains, d)
		}
	}

	if len(removedHosts) == 0 {
		return
	}

	fmt.Print("Removing tunnel domain from config... ")

	freshCfg.Domains = newDomains
	if err := config.SaveToPath(freshCfg, cwd); err != nil {
		fmt.Println(cli.Error("failed: " + err.Error()))
		return
	}
	fmt.Println(cli.Success("done"))

	// Remove tunnel vhost files
	sslMgr := ssl.NewManager(p)
	vhostGen := nginx.NewVhostGenerator(p, sslMgr)
	for _, host := range removedHosts {
		vhostFile := filepath.Join(vhostGen.VhostsDir(), fmt.Sprintf("%s-%s.conf", freshCfg.Name, host))
		os.Remove(vhostFile)
	}

	// Regenerate remaining vhosts and reload nginx
	regenNginxVhosts(p, freshCfg, cwd)
}

// regenNginxVhosts regenerates nginx vhosts for the config and reloads nginx
func regenNginxVhosts(p *platform.Platform, cfg *config.Config, cwd string) {
	sslMgr := ssl.NewManager(p)
	vhostGen := nginx.NewVhostGenerator(p, sslMgr)

	fmt.Print("Regenerating nginx vhosts... ")
	if err := vhostGen.Generate(cfg, cwd); err != nil {
		fmt.Println(cli.Error("failed: " + err.Error()))
		return
	}
	fmt.Println(cli.Success("done"))

	ngxCtrl := nginx.NewController(p)

	// Ensure hash bucket size is large enough for long hostnames (e.g. tunnel domains)
	_ = ngxCtrl.EnsureHashBucketSize()

	fmt.Print("Reloading nginx... ")
	if err := ngxCtrl.Reload(); err != nil {
		fmt.Println(cli.Error("failed: " + err.Error()))
	} else {
		fmt.Println(cli.Success("done"))
	}
}

// flushMagentoCache runs bin/magento cache:clean and cache:flush
func flushMagentoCache(phpBin, cwd string) {
	fmt.Print("Flushing Magento cache... ")

	cleanCmd := exec.Command(phpBin, "bin/magento", "cache:clean")
	cleanCmd.Dir = cwd
	_ = cleanCmd.Run()

	flushCmd := exec.Command(phpBin, "bin/magento", "cache:flush")
	flushCmd.Dir = cwd
	if err := flushCmd.Run(); err != nil {
		fmt.Println(cli.Warning("skipped"))
	} else {
		fmt.Println(cli.Success("done"))
	}
}

// extractHostname extracts the hostname from a URL string
func extractHostname(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		h := strings.TrimPrefix(rawURL, "https://")
		h = strings.TrimPrefix(h, "http://")
		return strings.Split(h, "/")[0]
	}
	return u.Hostname()
}

// getTunnelPidFile returns the path to the tunnel PID file for a project
func getTunnelPidFile(p *platform.Platform, projectName string) string {
	return filepath.Join(p.MageBoxDir(), "run", fmt.Sprintf("tunnel-%s.pid", projectName))
}

// getTunnelURLFile returns the path to the tunnel URL file for a project
func getTunnelURLFile(p *platform.Platform, projectName string) string {
	return filepath.Join(p.MageBoxDir(), "run", fmt.Sprintf("tunnel-%s.url", projectName))
}

// getTunnelBaseURLsFile returns the path to the saved base URLs file
func getTunnelBaseURLsFile(p *platform.Platform, projectName string) string {
	return filepath.Join(p.MageBoxDir(), "run", fmt.Sprintf("tunnel-%s.baseurls.json", projectName))
}

// readPidFile reads a PID from a file
func readPidFile(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

// writePidFile writes a PID to a file
func writePidFile(path string, pid int) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strconv.Itoa(pid)), 0644)
}

// processRunning checks if a process with the given PID is still running
func processRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

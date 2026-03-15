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

Automatically updates Magento base URLs (across all scopes and config
files) to the tunnel URL and reverts them when the tunnel is stopped.

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

// savedExposeState holds all state needed to revert an expose session
type savedExposeState struct {
	DBRows         []savedConfigRow  `json:"db_rows"`
	EnvLocked      map[string]string `json:"env_locked"`       // path -> value for env.php locked entries
	ConfigPHPPaths []string          `json:"config_php_paths"` // paths that were in config.php (need --lock-env to override)
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

	phpBin := p.PHPBinary(cfg.PHP)

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

	// Save current state (DB rows + env.php/config.php locked values)
	stateFile := getTunnelStateFile(p, cfg.Name)
	saveExposeState(db, cfg.DatabaseName(), phpBin, cwd, stateFile, domain)

	// Start cloudflared tunnel
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

				// Update base URLs everywhere: DB (all scopes) + env.php (--lock-env)
				setAllBaseURLs(db, cfg.DatabaseName(), phpBin, cwd, tunnelURL+"/")

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

		revertExposeState(db, cfg.DatabaseName(), phpBin, cwd, stateFile)
		removeTunnelDomain(p, cwd, cfg)

		fmt.Print("Stopping tunnel... ")
		_ = tunnelCmd.Process.Signal(syscall.SIGTERM)
	}()

	_ = tunnelCmd.Wait()

	// Clean up run files
	os.Remove(pidFile)
	os.Remove(urlFile)
	os.Remove(stateFile)

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
	stateFile := getTunnelStateFile(p, cfg.Name)
	phpBin := p.PHPBinary(cfg.PHP)

	db, _ := getDbInfo(cfg)
	revertExposeState(db, cfg.DatabaseName(), phpBin, cwd, stateFile)
	removeTunnelDomain(p, cwd, cfg)

	pid, err := readPidFile(pidFile)
	if err != nil || !processRunning(pid) {
		cli.PrintInfo("No tunnel process is running for this project")
		os.Remove(pidFile)
		os.Remove(getTunnelURLFile(p, cfg.Name))
		os.Remove(stateFile)
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
	os.Remove(stateFile)
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

// saveExposeState captures the current base URL state from both the database
// and config files for later restoration
func saveExposeState(db *dbInfo, dbName, phpBin, cwd, stateFile, localDomain string) {
	fmt.Print("Saving current base URLs... ")

	state := savedExposeState{
		EnvLocked: make(map[string]string),
	}

	// 1. Read all base URL rows from core_config_data
	if db != nil {
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
		if out, err := cmd.Output(); err == nil {
			localBaseURL := fmt.Sprintf("https://%s/", localDomain)
			for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
				if line == "" {
					continue
				}
				fields := strings.SplitN(line, "\t", 4)
				if len(fields) == 4 {
					value := fields[3]
					// Fix stale tunnel URLs
					if strings.Contains(value, ".trycloudflare.com") {
						switch {
						case strings.Contains(fields[2], "media"):
							value = localBaseURL + "media/"
						case strings.Contains(fields[2], "static"):
							value = localBaseURL + "static/"
						default:
							value = localBaseURL
						}
					}
					state.DBRows = append(state.DBRows, savedConfigRow{
						Scope: fields[0], ScopeID: fields[1],
						Path: fields[2], Value: value,
					})
				}
			}
		}
	}

	// 2. Read locked values from env.php and config.php via bin/magento config:show
	// Values that differ between config:show (which reads all layers) and the DB
	// are locked in config files
	for _, path := range []string{"web/unsecure/base_url", "web/secure/base_url"} {
		cmd := exec.Command(phpBin, "bin/magento", "config:show", path)
		cmd.Dir = cwd
		if out, err := cmd.Output(); err == nil {
			val := strings.TrimSpace(string(out))
			if val != "" && !strings.Contains(val, ".trycloudflare.com") {
				state.EnvLocked[path] = val
			}
		}
	}

	// Save state
	data, err := json.Marshal(state)
	if err != nil {
		fmt.Println(cli.Warning("skipped"))
		return
	}

	dir := filepath.Dir(stateFile)
	_ = os.MkdirAll(dir, 0755)
	if err := os.WriteFile(stateFile, data, 0644); err != nil {
		fmt.Println(cli.Warning("skipped"))
		return
	}

	fmt.Printf("%s (%d DB entries, %d locked)\n",
		cli.Success("done"), len(state.DBRows), len(state.EnvLocked))
}

// setAllBaseURLs updates base URLs in both core_config_data and env.php
func setAllBaseURLs(db *dbInfo, dbName, phpBin, cwd, tunnelBaseURL string) {
	// 1. Update all rows in core_config_data via SQL
	if db != nil {
		fmt.Print("Updating base URLs in database... ")
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
		} else {
			fmt.Println(cli.Success("done"))
		}
	}

	// 2. Override in env.php via --lock-env (takes priority over config.php and DB)
	fmt.Print("Locking base URLs in env.php... ")
	failed := false
	for _, path := range []string{"web/unsecure/base_url", "web/secure/base_url"} {
		cmd := exec.Command(phpBin, "bin/magento", "config:set", "--lock-env", path, tunnelBaseURL)
		cmd.Dir = cwd
		if err := cmd.Run(); err != nil {
			failed = true
		}
	}
	if failed {
		fmt.Println(cli.Error("failed"))
	} else {
		fmt.Println(cli.Success("done"))
	}

	// 3. Flush cache
	flushMagentoCache(phpBin, cwd)
}

// revertExposeState restores the base URLs from saved state
func revertExposeState(db *dbInfo, dbName, phpBin, cwd, stateFile string) {
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return
	}

	var state savedExposeState
	if err := json.Unmarshal(data, &state); err != nil {
		return
	}

	// 1. Restore DB rows
	if db != nil && len(state.DBRows) > 0 {
		fmt.Print("Reverting base URLs in database... ")
		var statements []string
		for _, row := range state.DBRows {
			statements = append(statements,
				fmt.Sprintf("UPDATE core_config_data SET value='%s' WHERE scope='%s' AND scope_id=%s AND path='%s'",
					row.Value, row.Scope, row.ScopeID, row.Path),
			)
		}
		query := strings.Join(statements, "; ")
		cmd := exec.Command("docker", "exec", db.ContainerName,
			"mysql", "-uroot", "-p"+docker.DefaultDBRootPassword,
			dbName, "-e", query)
		if err := cmd.Run(); err != nil {
			fmt.Println(cli.Error("failed"))
		} else {
			fmt.Println(cli.Success("done"))
		}
	}

	// 2. Restore env.php locked values, or delete them if they weren't originally there
	fmt.Print("Reverting env.php base URLs... ")
	failed := false
	for _, path := range []string{"web/unsecure/base_url", "web/secure/base_url"} {
		if origVal, wasLocked := state.EnvLocked[path]; wasLocked {
			// Restore original locked value
			cmd := exec.Command(phpBin, "bin/magento", "config:set", "--lock-env", path, origVal)
			cmd.Dir = cwd
			if err := cmd.Run(); err != nil {
				failed = true
			}
		} else {
			// Remove the lock we added (delete from env.php)
			cmd := exec.Command(phpBin, "bin/magento", "config:delete", "--lock-env", path)
			cmd.Dir = cwd
			if err := cmd.Run(); err != nil {
				failed = true
			}
		}
	}
	if failed {
		fmt.Println(cli.Warning("partial"))
	} else {
		fmt.Println(cli.Success("done"))
	}

	// 3. Flush cache
	flushMagentoCache(phpBin, cwd)
}

// addTunnelDomain adds the tunnel hostname as a domain to .magebox.yaml,
// regenerates nginx vhosts, and reloads nginx.
func addTunnelDomain(p *platform.Platform, cwd string, cfg *config.Config, tunnelHost, sourceDomain string) {
	fmt.Print("Adding tunnel domain to config... ")

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

	regenNginxVhosts(p, cfg, cwd)
}

// removeTunnelDomain removes any *.trycloudflare.com domain from .magebox.yaml
func removeTunnelDomain(p *platform.Platform, cwd string, cfg *config.Config) {
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

	sslMgr := ssl.NewManager(p)
	vhostGen := nginx.NewVhostGenerator(p, sslMgr)
	for _, host := range removedHosts {
		vhostFile := filepath.Join(vhostGen.VhostsDir(), fmt.Sprintf("%s-%s.conf", freshCfg.Name, host))
		os.Remove(vhostFile)
	}

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

func getTunnelPidFile(p *platform.Platform, projectName string) string {
	return filepath.Join(p.MageBoxDir(), "run", fmt.Sprintf("tunnel-%s.pid", projectName))
}

func getTunnelURLFile(p *platform.Platform, projectName string) string {
	return filepath.Join(p.MageBoxDir(), "run", fmt.Sprintf("tunnel-%s.url", projectName))
}

func getTunnelStateFile(p *platform.Platform, projectName string) string {
	return filepath.Join(p.MageBoxDir(), "run", fmt.Sprintf("tunnel-%s.state.json", projectName))
}

func readPidFile(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

func writePidFile(path string, pid int) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strconv.Itoa(pid)), 0644)
}

func processRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
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
	"qoliber/magebox/internal/platform"
)

var exposeCmd = &cobra.Command{
	Use:   "expose [domain]",
	Short: "Expose project via Cloudflare Tunnel",
	Long: `Creates a public Cloudflare Tunnel URL pointing to your local project.

Uses cloudflared quick tunnels (no account required) to generate a
temporary *.trycloudflare.com URL. Traffic is routed to your local
nginx with the correct Host header so the right project is served.

Automatically updates Magento base URLs to the tunnel URL and reverts
them when the tunnel is stopped.

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

// savedBaseURLs stores original Magento base URLs for restoration
type savedBaseURLs struct {
	UnsecureBaseURL string `json:"unsecure_base_url"`
	SecureBaseURL   string `json:"secure_base_url"`
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

	// Determine local URL — cloudflared connects to nginx HTTPS
	httpsPort := 443
	if p.Type == platform.Darwin {
		httpsPort = 8443
	}
	localURL := fmt.Sprintf("https://localhost:%d", httpsPort)

	cli.PrintTitle("Exposing Project")
	fmt.Printf("Project: %s\n", cli.Highlight(cfg.Name))
	fmt.Printf("Domain:  %s\n", cli.Highlight(domain))
	fmt.Printf("Backend: %s\n", cli.Highlight(localURL))
	fmt.Println()

	// Save current Magento base URLs before starting the tunnel
	phpBin := p.PHPBinary(cfg.PHP)
	origURLs := readMagentoBaseURLs(phpBin, cwd)
	urlsFile := getTunnelBaseURLsFile(p, cfg.Name)
	if err := saveBaseURLs(urlsFile, origURLs); err != nil {
		cli.PrintWarning("Could not save original base URLs: %v", err)
	}

	// Start cloudflared tunnel
	tunnelArgs := []string{
		"tunnel",
		"--url", localURL,
		"--http-host-header", domain,
		"--no-tls-verify",
	}

	tunnelCmd := exec.Command(cloudflaredPath, tunnelArgs...)
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
				fmt.Printf("Public URL: %s\n", cli.URL(tunnelURL))
				fmt.Println()

				_ = os.WriteFile(urlFile, []byte(tunnelURL), 0644)

				// Update Magento base URLs to the tunnel URL
				setMagentoBaseURLs(phpBin, cwd, tunnelURL+"/")

				cli.PrintInfo("Magento base URLs updated to tunnel URL")
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

		// Revert Magento base URLs before stopping
		revertMagentoBaseURLs(phpBin, cwd, urlsFile)

		fmt.Print("Stopping tunnel... ")
		_ = tunnelCmd.Process.Signal(syscall.SIGTERM)
	}()

	_ = tunnelCmd.Wait()

	// Clean up run files
	os.Remove(pidFile)
	os.Remove(urlFile)
	os.Remove(urlsFile)

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
	pid, err := readPidFile(pidFile)
	if err != nil {
		cli.PrintInfo("No tunnel is running for this project")
		return nil
	}

	if !processRunning(pid) {
		cli.PrintInfo("No tunnel is running for this project")
		// Still revert URLs in case they weren't restored
		phpBin := p.PHPBinary(cfg.PHP)
		urlsFile := getTunnelBaseURLsFile(p, cfg.Name)
		revertMagentoBaseURLs(phpBin, cwd, urlsFile)
		os.Remove(pidFile)
		os.Remove(getTunnelURLFile(p, cfg.Name))
		os.Remove(urlsFile)
		return nil
	}

	// Revert Magento base URLs
	phpBin := p.PHPBinary(cfg.PHP)
	urlsFile := getTunnelBaseURLsFile(p, cfg.Name)
	revertMagentoBaseURLs(phpBin, cwd, urlsFile)

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
		url := strings.TrimSpace(string(urlBytes))
		if url != "" {
			fmt.Printf("URL:    %s\n", cli.URL(url))
		}
	}

	return nil
}

// readMagentoBaseURLs reads the current Magento base URLs via bin/magento
func readMagentoBaseURLs(phpBin, cwd string) savedBaseURLs {
	urls := savedBaseURLs{}

	unsecureCmd := exec.Command(phpBin, "bin/magento", "config:show", "web/unsecure/base_url")
	unsecureCmd.Dir = cwd
	if out, err := unsecureCmd.Output(); err == nil {
		urls.UnsecureBaseURL = strings.TrimSpace(string(out))
	}

	secureCmd := exec.Command(phpBin, "bin/magento", "config:show", "web/secure/base_url")
	secureCmd.Dir = cwd
	if out, err := secureCmd.Output(); err == nil {
		urls.SecureBaseURL = strings.TrimSpace(string(out))
	}

	return urls
}

// setMagentoBaseURLs updates Magento base URLs and flushes cache
func setMagentoBaseURLs(phpBin, cwd, baseURL string) {
	fmt.Print("Updating Magento base URLs... ")

	unsecureCmd := exec.Command(phpBin, "bin/magento", "config:set", "web/unsecure/base_url", baseURL)
	unsecureCmd.Dir = cwd
	if err := unsecureCmd.Run(); err != nil {
		fmt.Println(cli.Error("failed"))
		cli.PrintWarning("Could not set unsecure base URL: %v", err)
		return
	}

	secureCmd := exec.Command(phpBin, "bin/magento", "config:set", "web/secure/base_url", baseURL)
	secureCmd.Dir = cwd
	if err := secureCmd.Run(); err != nil {
		fmt.Println(cli.Error("failed"))
		cli.PrintWarning("Could not set secure base URL: %v", err)
		return
	}

	fmt.Println(cli.Success("done"))

	flushMagentoCache(phpBin, cwd)
}

// revertMagentoBaseURLs restores Magento base URLs from the saved file
func revertMagentoBaseURLs(phpBin, cwd, urlsFile string) {
	urls, err := loadBaseURLs(urlsFile)
	if err != nil {
		return
	}

	fmt.Print("Reverting Magento base URLs... ")

	if urls.UnsecureBaseURL != "" {
		cmd := exec.Command(phpBin, "bin/magento", "config:set", "web/unsecure/base_url", urls.UnsecureBaseURL)
		cmd.Dir = cwd
		_ = cmd.Run()
	}

	if urls.SecureBaseURL != "" {
		cmd := exec.Command(phpBin, "bin/magento", "config:set", "web/secure/base_url", urls.SecureBaseURL)
		cmd.Dir = cwd
		_ = cmd.Run()
	}

	fmt.Println(cli.Success("done"))

	flushMagentoCache(phpBin, cwd)
}

// flushMagentoCache runs bin/magento cache:flush
func flushMagentoCache(phpBin, cwd string) {
	fmt.Print("Flushing Magento cache... ")
	cacheCmd := exec.Command(phpBin, "bin/magento", "cache:flush")
	cacheCmd.Dir = cwd
	if err := cacheCmd.Run(); err != nil {
		fmt.Println(cli.Warning("skipped"))
	} else {
		fmt.Println(cli.Success("done"))
	}
}

// saveBaseURLs persists the original base URLs to a JSON file
func saveBaseURLs(path string, urls savedBaseURLs) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.Marshal(urls)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// loadBaseURLs reads saved base URLs from a JSON file
func loadBaseURLs(path string) (*savedBaseURLs, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var urls savedBaseURLs
	if err := json.Unmarshal(data, &urls); err != nil {
		return nil, err
	}
	return &urls, nil
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

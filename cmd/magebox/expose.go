package main

import (
	"bufio"
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

Requires cloudflared to be installed (brew install cloudflared).`,
	Args: cobra.MaximumNArgs(1),
	RunE: runExpose,
}

var exposeStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the tunnel",
	Long:  "Stops the running Cloudflare Tunnel for the current project",
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
		// Validate that the specified domain belongs to this project
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
		// Stale pid file, clean up
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

	// Start cloudflared tunnel
	tunnelArgs := []string{
		"tunnel",
		"--url", localURL,
		"--http-host-header", domain,
		"--no-tls-verify",
	}

	tunnelCmd := exec.Command(cloudflaredPath, tunnelArgs...)
	tunnelCmd.Env = os.Environ()

	// Capture stderr to extract the tunnel URL
	stderr, err := tunnelCmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the tunnel process
	if err := tunnelCmd.Start(); err != nil {
		return fmt.Errorf("failed to start cloudflared: %w", err)
	}

	// Save PID for later stop
	if err := writePidFile(pidFile, tunnelCmd.Process.Pid); err != nil {
		cli.PrintWarning("Could not save tunnel PID: %v", err)
	}

	// Also save the URL file path for status
	urlFile := getTunnelURLFile(p, cfg.Name)

	// Parse stderr for the tunnel URL
	urlPattern := regexp.MustCompile(`https://[a-zA-Z0-9-]+\.trycloudflare\.com`)
	scanner := bufio.NewScanner(stderr)
	tunnelURL := ""

	fmt.Print("Starting tunnel... ")

	// Read lines until we find the URL or process exits
	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			if match := urlPattern.FindString(line); match != "" && tunnelURL == "" {
				tunnelURL = match
				fmt.Println(cli.Success("done"))
				fmt.Println()
				fmt.Printf("Public URL: %s\n", cli.URL(tunnelURL))
				fmt.Println()
				cli.PrintInfo("Traffic to the public URL is forwarded to %s", domain)
				cli.PrintInfo("Press Ctrl+C to stop the tunnel")
				fmt.Println()

				// Save URL for status command
				_ = os.WriteFile(urlFile, []byte(tunnelURL), 0644)
			}
		}
	}()

	// Handle Ctrl+C gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println()
		fmt.Print("Stopping tunnel... ")
		_ = tunnelCmd.Process.Signal(syscall.SIGTERM)
	}()

	// Wait for the process to finish
	_ = tunnelCmd.Wait()

	// Clean up
	os.Remove(pidFile)
	os.Remove(urlFile)

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
		os.Remove(pidFile)
		os.Remove(getTunnelURLFile(p, cfg.Name))
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

	// Try to read the URL file
	urlFile := getTunnelURLFile(p, cfg.Name)
	if urlBytes, err := os.ReadFile(urlFile); err == nil {
		url := strings.TrimSpace(string(urlBytes))
		if url != "" {
			fmt.Printf("URL:    %s\n", cli.URL(url))
		}
	}

	return nil
}

// getTunnelPidFile returns the path to the tunnel PID file for a project
func getTunnelPidFile(p *platform.Platform, projectName string) string {
	return filepath.Join(p.MageBoxDir(), "run", fmt.Sprintf("tunnel-%s.pid", projectName))
}

// getTunnelURLFile returns the path to the tunnel URL file for a project
func getTunnelURLFile(p *platform.Platform, projectName string) string {
	return filepath.Join(p.MageBoxDir(), "run", fmt.Sprintf("tunnel-%s.url", projectName))
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

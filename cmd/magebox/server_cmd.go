/**
 * Created by Qoliber
 *
 * @category    Qoliber
 * @package     MageBox
 * @author      Jakub Winkler <jwinkler@qoliber.com>
 */

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/qoliber/magebox/internal/cli"
	"github.com/qoliber/magebox/internal/teamserver"
)

var (
	serverPort       int
	serverHost       string
	serverDataDir    string
	serverTLSCert    string
	serverTLSKey     string
	serverAdminToken string
	serverMasterKey  string
	serverBackground bool
	serverRateLimit  int

	// SMTP configuration
	serverSMTPHost     string
	serverSMTPPort     int
	serverSMTPUser     string
	serverSMTPPassword string
	serverSMTPFrom     string
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Manage MageBox team server",
	Long: `Manage the MageBox team server for centralized team access management.

The team server provides secure SSH key distribution, role-based access control,
and audit logging for team environments.

Examples:
  magebox server start --admin-token "secret"  # Start server with admin token
  magebox server status                        # Check server status
  magebox server stop                          # Stop the server`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var serverStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the team server",
	Long: `Start the MageBox team server.

The server requires a master key for encryption. On first run, a new key is generated
and stored in the data directory. The admin token is used for initial authentication.

Examples:
  magebox server start --admin-token "secret"
  magebox server start --port 7443 --admin-token "secret" --tls-cert cert.pem --tls-key key.pem`,
	RunE: runServerStart,
}

var serverStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the team server",
	Long:  `Stop the running MageBox team server gracefully.`,
	RunE:  runServerStop,
}

var serverStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check team server status",
	Long:  `Check if the MageBox team server is running and display its status.`,
	RunE:  runServerStatus,
}

var serverInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize team server configuration",
	Long: `Initialize the team server configuration and generate encryption keys.

This creates the data directory, generates a master encryption key, and
creates an admin token for initial access.

Examples:
  magebox server init
  magebox server init --data-dir /var/lib/magebox/teamserver`,
	RunE: runServerInit,
}

func init() {
	// Server start flags
	serverStartCmd.Flags().IntVar(&serverPort, "port", 7443, "Server port")
	serverStartCmd.Flags().StringVar(&serverHost, "host", "0.0.0.0", "Server host")
	serverStartCmd.Flags().StringVar(&serverDataDir, "data-dir", "", "Data directory (default: ~/.magebox/teamserver)")
	serverStartCmd.Flags().StringVar(&serverTLSCert, "tls-cert", "", "TLS certificate file")
	serverStartCmd.Flags().StringVar(&serverTLSKey, "tls-key", "", "TLS key file")
	serverStartCmd.Flags().StringVar(&serverAdminToken, "admin-token", "", "Admin authentication token")
	serverStartCmd.Flags().StringVar(&serverMasterKey, "master-key", "", "Master encryption key (hex)")
	serverStartCmd.Flags().BoolVar(&serverBackground, "background", false, "Run in background")
	serverStartCmd.Flags().IntVar(&serverRateLimit, "rate-limit", -1, "Rate limit per minute (0 to disable, -1 for default)")

	// SMTP configuration flags
	serverStartCmd.Flags().StringVar(&serverSMTPHost, "smtp-host", "", "SMTP server host for email notifications")
	serverStartCmd.Flags().IntVar(&serverSMTPPort, "smtp-port", 587, "SMTP server port")
	serverStartCmd.Flags().StringVar(&serverSMTPUser, "smtp-user", "", "SMTP username")
	serverStartCmd.Flags().StringVar(&serverSMTPPassword, "smtp-password", "", "SMTP password")
	serverStartCmd.Flags().StringVar(&serverSMTPFrom, "smtp-from", "", "SMTP from address")

	// Server init flags
	serverInitCmd.Flags().StringVar(&serverDataDir, "data-dir", "", "Data directory (default: ~/.magebox/teamserver)")
	serverInitCmd.Flags().StringVar(&serverAdminToken, "admin-token", "", "Admin token (optional, for testing)")
	serverInitCmd.Flags().StringVar(&serverMasterKey, "master-key", "", "Master key in hex (optional, for testing)")

	serverCmd.AddCommand(serverStartCmd)
	serverCmd.AddCommand(serverStopCmd)
	serverCmd.AddCommand(serverStatusCmd)
	serverCmd.AddCommand(serverInitCmd)
	rootCmd.AddCommand(serverCmd)
}

func getServerDataDir() (string, error) {
	if serverDataDir != "" {
		return serverDataDir, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	return filepath.Join(homeDir, ".magebox", "teamserver"), nil
}

func runServerInit(cmd *cobra.Command, args []string) error {
	dataDir, err := getServerDataDir()
	if err != nil {
		return err
	}

	cli.PrintTitle("Initializing Team Server")
	fmt.Println()

	// Create data directory
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}
	cli.PrintSuccess("Data directory: %s", dataDir)

	// Check if already initialized
	configPath := filepath.Join(dataDir, "server.json")
	if _, err := os.Stat(configPath); err == nil {
		cli.PrintWarning("Server already initialized. Use --data-dir to specify a different location.")
		return nil
	}

	// Generate or use provided master key
	var masterKey []byte
	var masterKeyHex string
	if serverMasterKey != "" {
		// Use provided master key (for testing)
		var err error
		masterKey, err = teamserver.MasterKeyFromHex(serverMasterKey)
		if err != nil {
			return fmt.Errorf("invalid master key: %w", err)
		}
		masterKeyHex = serverMasterKey
		cli.PrintSuccess("Using provided master key")
	} else {
		var err error
		masterKey, err = teamserver.GenerateMasterKey()
		if err != nil {
			return fmt.Errorf("failed to generate master key: %w", err)
		}
		masterKeyHex = teamserver.MasterKeyToHex(masterKey)
		cli.PrintSuccess("Master key generated")
	}

	// Generate or use provided admin token
	var adminToken string
	if serverAdminToken != "" {
		// Use provided admin token (for testing)
		adminToken = serverAdminToken
		cli.PrintSuccess("Using provided admin token")
	} else {
		var err error
		adminToken, err = teamserver.GenerateToken(32)
		if err != nil {
			return fmt.Errorf("failed to generate admin token: %w", err)
		}
		cli.PrintSuccess("Admin token generated")
	}

	adminTokenHash, err := teamserver.HashToken(adminToken)
	if err != nil {
		return fmt.Errorf("failed to hash admin token: %w", err)
	}

	// Generate SSH CA key pair
	caKeyPair, err := teamserver.GenerateCAKeyPair()
	if err != nil {
		return fmt.Errorf("failed to generate CA key pair: %w", err)
	}
	cli.PrintSuccess("SSH CA key pair generated")

	// Initialize database and store CA keys
	crypto, err := teamserver.NewCrypto(masterKey)
	if err != nil {
		return fmt.Errorf("failed to create crypto: %w", err)
	}

	dbPath := filepath.Join(dataDir, "teamserver.db")
	storage, err := teamserver.NewStorage(dbPath, crypto)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer storage.Close()

	// Store CA keys (encrypted)
	if err := storage.SaveCAKeys(caKeyPair.PrivateKeyPEM, caKeyPair.PublicKeySSH); err != nil {
		return fmt.Errorf("failed to store CA keys: %w", err)
	}
	cli.PrintSuccess("CA keys stored (encrypted)")

	// Save configuration
	config := map[string]interface{}{
		"master_key":       masterKeyHex,
		"admin_token_hash": adminTokenHash,
		"port":             7443,
		"host":             "0.0.0.0",
		"tls_enabled":      false,
		"ca_enabled":       true,
		"initialized_at":   time.Now().UTC().Format(time.RFC3339),
	}

	configData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, configData, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	cli.PrintSuccess("Configuration saved")

	fmt.Println()
	cli.PrintTitle("Server Credentials")
	fmt.Println()
	cli.PrintInfo("Master Key (save securely!):")
	fmt.Printf("  %s\n", masterKeyHex)
	fmt.Println()
	cli.PrintInfo("Admin Token (use for authentication):")
	fmt.Printf("  %s\n", adminToken)
	fmt.Println()
	cli.PrintTitle("SSH Certificate Authority")
	fmt.Println()
	cli.PrintInfo("CA Public Key (deploy to servers):")
	fmt.Printf("  %s\n", caKeyPair.PublicKeySSH)
	fmt.Println()
	cli.PrintInfo("Add this to /etc/ssh/sshd_config on target servers:")
	fmt.Printf("  TrustedUserCAKeys /etc/ssh/magebox-ca.pub\n")
	fmt.Println()
	cli.PrintWarning("Save these credentials! The admin token cannot be recovered.")
	fmt.Println()
	cli.PrintInfo("Start the server with:")
	fmt.Printf("  magebox server start\n")

	return nil
}

func runServerStart(cmd *cobra.Command, args []string) error {
	dataDir, err := getServerDataDir()
	if err != nil {
		return err
	}

	// Load configuration
	configPath := filepath.Join(dataDir, "server.json")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("server not initialized. Run 'magebox server init' first")
		}
		return fmt.Errorf("failed to read config: %w", err)
	}

	var savedConfig map[string]interface{}
	if err := json.Unmarshal(configData, &savedConfig); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Get master key
	var masterKey []byte
	if serverMasterKey != "" {
		masterKey, err = teamserver.MasterKeyFromHex(serverMasterKey)
		if err != nil {
			return fmt.Errorf("invalid master key: %w", err)
		}
	} else if mkHex, ok := savedConfig["master_key"].(string); ok {
		masterKey, err = teamserver.MasterKeyFromHex(mkHex)
		if err != nil {
			return fmt.Errorf("invalid saved master key: %w", err)
		}
	} else {
		return fmt.Errorf("master key required. Use --master-key or run 'magebox server init'")
	}

	// Get admin token hash
	var adminTokenHash string
	if serverAdminToken != "" {
		adminTokenHash, err = teamserver.HashToken(serverAdminToken)
		if err != nil {
			return fmt.Errorf("failed to hash admin token: %w", err)
		}
	} else if hash, ok := savedConfig["admin_token_hash"].(string); ok {
		adminTokenHash = hash
	}

	// Build server config
	config := teamserver.DefaultServerConfig()
	config.DataDir = dataDir
	config.Port = serverPort
	config.Host = serverHost
	config.AdminTokenHash = adminTokenHash

	if port, ok := savedConfig["port"].(float64); ok && serverPort == 7443 {
		config.Port = int(port)
	}
	if host, ok := savedConfig["host"].(string); ok && serverHost == "0.0.0.0" {
		config.Host = host
	}

	// TLS configuration
	if serverTLSCert != "" && serverTLSKey != "" {
		config.TLS.Enabled = true
		config.TLS.CertFile = serverTLSCert
		config.TLS.KeyFile = serverTLSKey
	} else if tlsEnabled, ok := savedConfig["tls_enabled"].(bool); ok && tlsEnabled {
		config.TLS.Enabled = true
		if cert, ok := savedConfig["tls_cert"].(string); ok {
			config.TLS.CertFile = cert
		}
		if key, ok := savedConfig["tls_key"].(string); ok {
			config.TLS.KeyFile = key
		}
	} else {
		config.TLS.Enabled = false
	}

	// Rate limit configuration
	if serverRateLimit == 0 {
		config.Security.RateLimitEnabled = false
		config.Security.RateLimitPerMinute = 0
	} else if serverRateLimit > 0 {
		config.Security.RateLimitEnabled = true
		config.Security.RateLimitPerMinute = serverRateLimit
	}

	// SMTP configuration (flags take precedence over env vars)
	smtpHost := serverSMTPHost
	if smtpHost == "" {
		smtpHost = os.Getenv("MAGEBOX_SMTP_HOST")
	}
	smtpPort := serverSMTPPort
	if smtpPort == 587 && os.Getenv("MAGEBOX_SMTP_PORT") != "" {
		fmt.Sscanf(os.Getenv("MAGEBOX_SMTP_PORT"), "%d", &smtpPort)
	}
	smtpUser := serverSMTPUser
	if smtpUser == "" {
		smtpUser = os.Getenv("MAGEBOX_SMTP_USER")
	}
	smtpPassword := serverSMTPPassword
	if smtpPassword == "" {
		smtpPassword = os.Getenv("MAGEBOX_SMTP_PASSWORD")
	}
	smtpFrom := serverSMTPFrom
	if smtpFrom == "" {
		smtpFrom = os.Getenv("MAGEBOX_SMTP_FROM")
	}

	if smtpHost != "" {
		config.Notifications.SMTP.Enabled = true
		config.Notifications.SMTP.Host = smtpHost
		config.Notifications.SMTP.Port = smtpPort
		config.Notifications.SMTP.User = smtpUser
		config.Notifications.SMTP.Password = smtpPassword
		config.Notifications.SMTP.From = smtpFrom
	}

	// Create and start server
	server, err := teamserver.NewServer(config, masterKey)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Save PID file
	pidFile := filepath.Join(dataDir, "server.pid")
	if err := os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0644); err != nil {
		cli.PrintWarning("Failed to write PID file: %v", err)
	}

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nShutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		server.Stop(ctx)
		os.Remove(pidFile)
		os.Exit(0)
	}()

	// Print startup message
	protocol := "http"
	if config.TLS.Enabled {
		protocol = "https"
	}

	cli.PrintTitle("MageBox Team Server")
	fmt.Println()
	cli.PrintSuccess("Server starting on %s://%s:%d", protocol, config.Host, config.Port)
	cli.PrintInfo("Data directory: %s", dataDir)
	cli.PrintInfo("Press Ctrl+C to stop")
	fmt.Println()

	// Start server (blocking)
	if err := server.Start(); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

func runServerStop(cmd *cobra.Command, args []string) error {
	dataDir, err := getServerDataDir()
	if err != nil {
		return err
	}

	pidFile := filepath.Join(dataDir, "server.pid")
	pidData, err := os.ReadFile(pidFile)
	if err != nil {
		if os.IsNotExist(err) {
			cli.PrintInfo("Server is not running (no PID file found)")
			return nil
		}
		return fmt.Errorf("failed to read PID file: %w", err)
	}

	var pid int
	if _, err := fmt.Sscanf(string(pidData), "%d", &pid); err != nil {
		return fmt.Errorf("invalid PID file: %w", err)
	}

	// Send SIGTERM to the process
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		// Process might already be dead
		os.Remove(pidFile)
		cli.PrintInfo("Server process not found (may have already stopped)")
		return nil
	}

	cli.PrintSuccess("Sent stop signal to server (PID: %d)", pid)

	// Wait a bit and check if it stopped
	time.Sleep(2 * time.Second)
	if err := process.Signal(syscall.Signal(0)); err != nil {
		os.Remove(pidFile)
		cli.PrintSuccess("Server stopped successfully")
	} else {
		cli.PrintWarning("Server may still be shutting down")
	}

	return nil
}

func runServerStatus(cmd *cobra.Command, args []string) error {
	dataDir, err := getServerDataDir()
	if err != nil {
		return err
	}

	cli.PrintTitle("Team Server Status")
	fmt.Println()

	// Check configuration
	configPath := filepath.Join(dataDir, "server.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cli.PrintInfo("Server not initialized")
		cli.PrintInfo("Run 'magebox server init' to initialize")
		return nil
	}

	configData, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(configData, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	cli.PrintSuccess("Server initialized")
	cli.PrintInfo("Data directory: %s", dataDir)

	if initAt, ok := config["initialized_at"].(string); ok {
		cli.PrintInfo("Initialized at: %s", initAt)
	}

	// Check if running
	pidFile := filepath.Join(dataDir, "server.pid")
	pidData, err := os.ReadFile(pidFile)
	if err != nil {
		if os.IsNotExist(err) {
			cli.PrintInfo("Status: Stopped")
			return nil
		}
		return fmt.Errorf("failed to read PID file: %w", err)
	}

	var pid int
	if _, err := fmt.Sscanf(strings.TrimSpace(string(pidData)), "%d", &pid); err != nil {
		cli.PrintWarning("Invalid PID file")
		return nil
	}

	// Check if process is running
	process, err := os.FindProcess(pid)
	if err != nil {
		cli.PrintInfo("Status: Stopped (stale PID file)")
		os.Remove(pidFile)
		return nil
	}

	if err := process.Signal(syscall.Signal(0)); err != nil {
		cli.PrintInfo("Status: Stopped (stale PID file)")
		os.Remove(pidFile)
		return nil
	}

	cli.PrintSuccess("Status: Running (PID: %d)", pid)

	// Show config details
	if port, ok := config["port"].(float64); ok {
		cli.PrintInfo("Port: %d", int(port))
	}
	if host, ok := config["host"].(string); ok {
		cli.PrintInfo("Host: %s", host)
	}
	if tlsEnabled, ok := config["tls_enabled"].(bool); ok {
		if tlsEnabled {
			cli.PrintInfo("TLS: Enabled")
		} else {
			cli.PrintInfo("TLS: Disabled")
		}
	}

	// Check database
	dbPath := filepath.Join(dataDir, "teamserver.db")
	if info, err := os.Stat(dbPath); err == nil {
		cli.PrintInfo("Database size: %.2f KB", float64(info.Size())/1024)
	}

	return nil
}

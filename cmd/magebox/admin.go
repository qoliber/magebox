// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"qoliber/magebox/internal/cli"
	"qoliber/magebox/internal/config"
)

var adminCmd = &cobra.Command{
	Use:   "admin",
	Short: "Magento admin management",
	Long:  "Manage Magento admin users and settings",
}

var adminPasswordCmd = &cobra.Command{
	Use:   "password <email> [new-password]",
	Short: "Reset admin password",
	Long: `Reset the password for a Magento admin user.

If no password is provided, you will be prompted to enter one.

Example:
  magebox admin password admin@example.com newpassword123`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runAdminPassword,
}

var adminDisable2FACmd = &cobra.Command{
	Use:   "disable-2fa",
	Short: "Disable 2FA for all admin users",
	Long: `Disables Two-Factor Authentication for all admin users.

This is useful for local development where 2FA can be inconvenient.
NOT recommended for production environments.`,
	RunE: runAdminDisable2FA,
}

var adminListCmd = &cobra.Command{
	Use:   "list",
	Short: "List admin users",
	Long:  "Lists all Magento admin users",
	RunE:  runAdminList,
}

var adminCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new admin user",
	Long: `Creates a new Magento admin user interactively.

You will be prompted for:
  - Username
  - Email
  - Password
  - First name
  - Last name`,
	RunE: runAdminCreate,
}

func init() {
	adminCmd.AddCommand(adminPasswordCmd)
	adminCmd.AddCommand(adminDisable2FACmd)
	adminCmd.AddCommand(adminListCmd)
	adminCmd.AddCommand(adminCreateCmd)
	rootCmd.AddCommand(adminCmd)
}

func runAdminPassword(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	p, err := getPlatform()
	if err != nil {
		return err
	}

	// Load project config
	cfg, err := config.LoadFromPath(cwd)
	if err != nil {
		cli.PrintError("No project config found - run 'magebox init' first")
		return nil
	}

	email := args[0]
	var password string

	if len(args) > 1 {
		password = args[1]
	} else {
		// Prompt for password
		fmt.Print("Enter new password: ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		password = strings.TrimSpace(input)

		if password == "" {
			cli.PrintError("Password cannot be empty")
			return nil
		}
	}

	cli.PrintTitle("Resetting Admin Password")
	fmt.Printf("Email: %s\n", cli.Highlight(email))
	fmt.Println()

	// Run bin/magento command
	phpBin := p.PHPBinary(cfg.PHP)
	magentoCmd := exec.Command(phpBin, "bin/magento", "admin:user:unlock", email)
	magentoCmd.Dir = cwd
	magentoCmd.Stdout = os.Stdout
	magentoCmd.Stderr = os.Stderr

	// First unlock the user (in case they're locked)
	_ = magentoCmd.Run()

	// Now reset the password using SQL (more reliable than CLI)
	fmt.Print("Resetting password... ")

	// Get database config from env.php
	dbHost, dbName, dbUser, dbPass := getMagentoDatabaseConfig(cwd)
	if dbHost == "" {
		// Fall back to standard MageBox config
		dbHost = "127.0.0.1"
		dbName = cfg.Name
		dbUser = "root"
		dbPass = "magebox"

		// Determine port from config
		if cfg.Services.HasMySQL() {
			dbHost = fmt.Sprintf("127.0.0.1:%d", getMySQLPortForVersion(cfg.Services.MySQL.Version))
		}
	}

	// Generate password hash using PHP (Magento's password hashing)
	hashCmd := exec.Command(phpBin, "-r", fmt.Sprintf(`
		$password = '%s';
		$salt = bin2hex(random_bytes(32));
		$hash = hash('sha256', $salt . $password) . ':' . $salt . ':1';
		echo $hash;
	`, strings.ReplaceAll(password, "'", "\\'")))
	hashCmd.Dir = cwd

	hashOutput, err := hashCmd.Output()
	if err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to generate password hash: %w", err)
	}
	passwordHash := strings.TrimSpace(string(hashOutput))

	// Update password in database
	mysqlCmd := exec.Command("mysql",
		"-h", strings.Split(dbHost, ":")[0],
		"-u", dbUser,
		fmt.Sprintf("-p%s", dbPass),
		dbName,
		"-e", fmt.Sprintf("UPDATE admin_user SET password = '%s' WHERE email = '%s'", passwordHash, email),
	)

	if strings.Contains(dbHost, ":") {
		port := strings.Split(dbHost, ":")[1]
		mysqlCmd = exec.Command("mysql",
			"-h", strings.Split(dbHost, ":")[0],
			"-P", port,
			"-u", dbUser,
			fmt.Sprintf("-p%s", dbPass),
			dbName,
			"-e", fmt.Sprintf("UPDATE admin_user SET password = '%s' WHERE email = '%s'", passwordHash, email),
		)
	}

	if err := mysqlCmd.Run(); err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to update password: %w", err)
	}

	fmt.Println(cli.Success("done"))
	fmt.Println()
	cli.PrintSuccess("Password reset successfully!")
	fmt.Printf("You can now log in with email %s\n", cli.Highlight(email))

	return nil
}

func runAdminDisable2FA(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	p, err := getPlatform()
	if err != nil {
		return err
	}

	// Load project config
	cfg, err := config.LoadFromPath(cwd)
	if err != nil {
		cli.PrintError("No project config found - run 'magebox init' first")
		return nil
	}

	cli.PrintTitle("Disabling 2FA")
	fmt.Println()

	phpBin := p.PHPBinary(cfg.PHP)

	// Method 1: Try disabling 2FA modules
	fmt.Print("Disabling 2FA modules... ")

	modules := []string{
		"Magento_TwoFactorAuth",
		"Magento_AdminAdobeImsTwoFactorAuth",
	}

	for _, module := range modules {
		disableCmd := exec.Command(phpBin, "bin/magento", "module:disable", module)
		disableCmd.Dir = cwd
		_ = disableCmd.Run() // Ignore errors (module may not exist)
	}
	fmt.Println(cli.Success("done"))

	// Method 2: Set config to bypass 2FA
	fmt.Print("Setting 2FA config... ")
	configCmd := exec.Command(phpBin, "bin/magento", "config:set", "twofactorauth/general/enable", "0")
	configCmd.Dir = cwd
	_ = configCmd.Run()
	fmt.Println(cli.Success("done"))

	// Clear cache
	fmt.Print("Clearing cache... ")
	cacheCmd := exec.Command(phpBin, "bin/magento", "cache:clean")
	cacheCmd.Dir = cwd
	if err := cacheCmd.Run(); err != nil {
		fmt.Println(cli.Warning("skipped"))
	} else {
		fmt.Println(cli.Success("done"))
	}

	fmt.Println()
	cli.PrintSuccess("2FA disabled!")
	cli.PrintWarning("Remember to re-enable 2FA before deploying to production")

	return nil
}

func runAdminList(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	p, err := getPlatform()
	if err != nil {
		return err
	}

	// Load project config
	cfg, err := config.LoadFromPath(cwd)
	if err != nil {
		cli.PrintError("No project config found - run 'magebox init' first")
		return nil
	}

	cli.PrintTitle("Admin Users")
	fmt.Println()

	// Get database config
	dbHost, dbName, dbUser, dbPass := getMagentoDatabaseConfig(cwd)
	if dbHost == "" {
		dbHost = "127.0.0.1"
		dbName = cfg.Name
		dbUser = "root"
		dbPass = "magebox"

		if cfg.Services.HasMySQL() {
			dbHost = fmt.Sprintf("127.0.0.1:%d", getMySQLPortForVersion(cfg.Services.MySQL.Version))
		}
	}

	// Query admin users
	query := "SELECT username, email, firstname, lastname, is_active FROM admin_user"

	var mysqlCmd *exec.Cmd
	if strings.Contains(dbHost, ":") {
		host := strings.Split(dbHost, ":")[0]
		port := strings.Split(dbHost, ":")[1]
		mysqlCmd = exec.Command("mysql",
			"-h", host, "-P", port,
			"-u", dbUser, fmt.Sprintf("-p%s", dbPass),
			"-N", "-e", query, dbName,
		)
	} else {
		mysqlCmd = exec.Command("mysql",
			"-h", dbHost,
			"-u", dbUser, fmt.Sprintf("-p%s", dbPass),
			"-N", "-e", query, dbName,
		)
	}

	output, err := mysqlCmd.Output()
	if err != nil {
		// Fall back to using PHP
		phpBin := p.PHPBinary(cfg.PHP)
		listCmd := exec.Command(phpBin, "bin/magento", "admin:user:list", "--format=csv")
		listCmd.Dir = cwd
		output, err = listCmd.Output()
		if err != nil {
			return fmt.Errorf("failed to list admin users: %w", err)
		}
	}

	if len(output) == 0 {
		fmt.Println("No admin users found")
		return nil
	}

	// Parse and display
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	fmt.Printf("%-15s %-30s %-15s %-15s %s\n", "USERNAME", "EMAIL", "FIRST NAME", "LAST NAME", "ACTIVE")
	fmt.Println(strings.Repeat("-", 90))

	for _, line := range lines {
		fields := strings.Split(line, "\t")
		if len(fields) >= 5 {
			active := "No"
			if fields[4] == "1" {
				active = "Yes"
			}
			fmt.Printf("%-15s %-30s %-15s %-15s %s\n",
				truncate(fields[0], 15),
				truncate(fields[1], 30),
				truncate(fields[2], 15),
				truncate(fields[3], 15),
				active,
			)
		}
	}

	return nil
}

func runAdminCreate(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	p, err := getPlatform()
	if err != nil {
		return err
	}

	// Load project config
	cfg, err := config.LoadFromPath(cwd)
	if err != nil {
		cli.PrintError("No project config found - run 'magebox init' first")
		return nil
	}

	cli.PrintTitle("Create Admin User")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Username: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	fmt.Print("Email: ")
	email, _ := reader.ReadString('\n')
	email = strings.TrimSpace(email)

	fmt.Print("Password: ")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)

	fmt.Print("First Name: ")
	firstName, _ := reader.ReadString('\n')
	firstName = strings.TrimSpace(firstName)

	fmt.Print("Last Name: ")
	lastName, _ := reader.ReadString('\n')
	lastName = strings.TrimSpace(lastName)

	fmt.Println()
	fmt.Print("Creating admin user... ")

	phpBin := p.PHPBinary(cfg.PHP)
	createCmd := exec.Command(phpBin, "bin/magento", "admin:user:create",
		"--admin-user", username,
		"--admin-password", password,
		"--admin-email", email,
		"--admin-firstname", firstName,
		"--admin-lastname", lastName,
	)
	createCmd.Dir = cwd

	output, err := createCmd.CombinedOutput()
	if err != nil {
		fmt.Println(cli.Error("failed"))
		fmt.Printf("Error: %s\n", string(output))
		return nil
	}

	fmt.Println(cli.Success("done"))
	fmt.Println()
	cli.PrintSuccess("Admin user created!")
	fmt.Printf("Username: %s\n", cli.Highlight(username))
	fmt.Printf("Email: %s\n", cli.Highlight(email))

	return nil
}

// getMagentoDatabaseConfig reads database config from app/etc/env.php
func getMagentoDatabaseConfig(projectPath string) (host, dbname, user, pass string) {
	envPath := filepath.Join(projectPath, "app", "etc", "env.php")

	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		return "", "", "", ""
	}

	// Read and parse env.php using PHP
	cmd := exec.Command("php", "-r", `
		$env = include($argv[1]);
		if (isset($env['db']['connection']['default'])) {
			$db = $env['db']['connection']['default'];
			echo ($db['host'] ?? '') . "\n";
			echo ($db['dbname'] ?? '') . "\n";
			echo ($db['username'] ?? '') . "\n";
			echo ($db['password'] ?? '') . "\n";
		}
	`, envPath)

	output, err := cmd.Output()
	if err != nil {
		return "", "", "", ""
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) >= 4 {
		return strings.TrimSpace(lines[0]),
			strings.TrimSpace(lines[1]),
			strings.TrimSpace(lines[2]),
			strings.TrimSpace(lines[3])
	}

	return "", "", "", ""
}

// getMySQLPortForVersion returns the MySQL port for a given version
func getMySQLPortForVersion(version string) int {
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

// truncate truncates a string to max length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

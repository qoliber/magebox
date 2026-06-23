package main

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"qoliber/magebox/internal/cli"
	"qoliber/magebox/internal/config"
	"qoliber/magebox/internal/docker"
)

const (
	phpmyadminDefaultPort = "8036"
	phpmyadminContainer   = "magebox-phpmyadmin"
	phpmyadminService     = "phpmyadmin"
)

// getPhpMyAdminURL returns the phpMyAdmin URL, reading the actual port from the running container.
// Falls back to the default port if the container is not running.
func getPhpMyAdminURL() string {
	portCmd := exec.Command("docker", "port", phpmyadminContainer, "80")
	output, err := portCmd.Output()
	if err == nil {
		// Output format: "0.0.0.0:8036" or "[::]:8036"
		parts := strings.TrimSpace(string(output))
		// Handle multiple lines (IPv4 + IPv6)
		for _, line := range strings.Split(parts, "\n") {
			line = strings.TrimSpace(line)
			if idx := strings.LastIndex(line, ":"); idx != -1 {
				port := line[idx+1:]
				return fmt.Sprintf("http://localhost:%s", port)
			}
		}
	}
	return fmt.Sprintf("http://localhost:%s", phpmyadminDefaultPort)
}

var phpmyadminCmd = &cobra.Command{
	Use:   "phpmyadmin",
	Short: "phpMyAdmin database UI",
	Long:  "Manage phpMyAdmin web UI for MySQL/MariaDB",
}

var phpmyadminEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable phpMyAdmin",
	Long:  "Enables phpMyAdmin web UI for browsing MySQL/MariaDB databases",
	RunE:  runPhpMyAdminEnable,
}

var phpmyadminDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable phpMyAdmin",
	Long:  "Disables phpMyAdmin web UI",
	RunE:  runPhpMyAdminDisable,
}

var phpmyadminStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show phpMyAdmin status",
	Long:  "Shows phpMyAdmin status and connection information",
	RunE:  runPhpMyAdminStatus,
}

var phpmyadminOpenCmd = &cobra.Command{
	Use:   "open",
	Short: "Open phpMyAdmin in browser",
	Long:  "Opens the phpMyAdmin web UI in the default browser, starting it if needed",
	RunE:  runPhpMyAdminOpen,
}

func init() {
	phpmyadminCmd.AddCommand(phpmyadminEnableCmd)
	phpmyadminCmd.AddCommand(phpmyadminDisableCmd)
	phpmyadminCmd.AddCommand(phpmyadminStatusCmd)
	phpmyadminCmd.AddCommand(phpmyadminOpenCmd)
	rootCmd.AddCommand(phpmyadminCmd)
}

func runPhpMyAdminEnable(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	globalCfg, err := config.LoadGlobalConfig(p.HomeDir)
	if err != nil {
		return fmt.Errorf("failed to load global config: %w", err)
	}

	switch decideServiceUI(globalCfg.PhpMyAdmin, isContainerRunning(phpmyadminContainer)) {
	case decisionProceed:
		cli.PrintInfo("phpMyAdmin is already enabled and running")
		fmt.Println()
		fmt.Printf("  Web UI: %s\n", cli.Highlight(getPhpMyAdminURL()))
		return nil

	case decisionStart:
		cli.PrintTitle("Starting phpMyAdmin")
		fmt.Println()
		fmt.Print("phpMyAdmin is enabled but stopped, starting... ")
		if err := ensureGlobalServiceRunning(p, phpmyadminService); err != nil {
			fmt.Println(cli.Error("failed"))
			cli.PrintError("%v", err)
			return nil
		}
		fmt.Println(cli.Success("done"))
		fmt.Println()
		cli.PrintSuccess("phpMyAdmin started!")
		fmt.Println()
		fmt.Printf("  Web UI: %s\n", cli.Highlight(getPhpMyAdminURL()))
		return nil

	default: // decisionNotEnabled
		cli.PrintTitle("Enabling phpMyAdmin")
		fmt.Println()

		globalCfg.PhpMyAdmin = true
		if err := config.SaveGlobalConfig(p.HomeDir, globalCfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Print("Starting phpMyAdmin container... ")
		if err := ensureGlobalServiceRunning(p, phpmyadminService); err != nil {
			fmt.Println(cli.Error("failed"))
			cli.PrintError("%v", err)
			return nil
		}
		fmt.Println(cli.Success("done"))

		fmt.Println()
		cli.PrintSuccess("phpMyAdmin enabled!")
		fmt.Println()
		fmt.Printf("  Web UI: %s\n", cli.Highlight(getPhpMyAdminURL()))
		fmt.Println()
		cli.PrintInfo("Use arbitrary server mode to connect to any MySQL/MariaDB instance")
		cli.PrintInfo("Database containers are accessible by their container name (e.g. magebox-mysql-8.0)")
		return nil
	}
}

func runPhpMyAdminDisable(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	globalCfg, err := config.LoadGlobalConfig(p.HomeDir)
	if err != nil {
		return fmt.Errorf("failed to load global config: %w", err)
	}

	if !globalCfg.PhpMyAdmin {
		cli.PrintInfo("phpMyAdmin is not enabled")
		return nil
	}

	cli.PrintTitle("Disabling phpMyAdmin")
	fmt.Println()

	// Stop container first
	fmt.Print("Stopping phpMyAdmin container... ")
	composeGen := docker.NewComposeGenerator(p)
	dockerCtrl := docker.NewDockerController(composeGen.ComposeFilePath())
	if err := dockerCtrl.StopService(phpmyadminService); err != nil {
		fmt.Println(cli.Warning("not running"))
	} else {
		fmt.Println(cli.Success("done"))
	}

	globalCfg.PhpMyAdmin = false
	if err := config.SaveGlobalConfig(p.HomeDir, globalCfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Regenerate docker-compose without phpMyAdmin
	if err := composeGen.GenerateGlobalServices(discoverAllConfigs(p)); err != nil {
		return fmt.Errorf("failed to update docker-compose: %w", err)
	}

	fmt.Println()
	cli.PrintSuccess("phpMyAdmin disabled!")

	return nil
}

func runPhpMyAdminStatus(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	globalCfg, err := config.LoadGlobalConfig(p.HomeDir)
	if err != nil {
		return fmt.Errorf("failed to load global config: %w", err)
	}

	cli.PrintTitle("phpMyAdmin Status")
	fmt.Println()

	if !globalCfg.PhpMyAdmin {
		fmt.Println("Enabled: " + cli.Warning("no"))
		fmt.Println()
		cli.PrintInfo("Enable with: magebox phpmyadmin enable")
		return nil
	}

	fmt.Println("Enabled: " + cli.Success("yes"))

	if isContainerRunning(phpmyadminContainer) {
		fmt.Println("Status:  " + cli.Success("running"))
		fmt.Printf("Web UI:  %s\n", cli.Highlight(getPhpMyAdminURL()))
	} else {
		fmt.Println("Status:  " + cli.Warning("stopped"))
		fmt.Println()
		cli.PrintInfo("Start with: magebox phpmyadmin open")
	}

	return nil
}

func runPhpMyAdminOpen(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	globalCfg, err := config.LoadGlobalConfig(p.HomeDir)
	if err != nil {
		return fmt.Errorf("failed to load global config: %w", err)
	}

	switch decideServiceUI(globalCfg.PhpMyAdmin, isContainerRunning(phpmyadminContainer)) {
	case decisionNotEnabled:
		cli.PrintError("phpMyAdmin is not enabled")
		fmt.Println()
		cli.PrintInfo("Enable with: magebox phpmyadmin enable")
		return nil

	case decisionStart:
		fmt.Print("phpMyAdmin is not running, starting... ")
		if err := ensureGlobalServiceRunning(p, phpmyadminService); err != nil {
			fmt.Println(cli.Error("failed"))
			cli.PrintError("%v", err)
			return nil
		}
		fmt.Println(cli.Success("done"))
	}

	url := getPhpMyAdminURL()
	cli.PrintInfo("Opening %s", cli.URL(url))
	return openInBrowser(url)
}

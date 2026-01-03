package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"qoliber/magebox/internal/cli"
	"qoliber/magebox/internal/config"
	"qoliber/magebox/internal/dns"
	"qoliber/magebox/internal/platform"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "MageBox configuration",
	Long:  "View and modify MageBox global configuration",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long:  "Displays the current MageBox global configuration",
	RunE:  runConfigShow,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Sets a global configuration value.

Available keys:
  dns_mode     - DNS resolution mode: "hosts" or "dnsmasq"
  default_php  - Default PHP version for new projects (e.g., "8.2")
  tld          - Top-level domain for local dev (default: "test")
  portainer    - Enable Portainer Docker UI: "true" or "false"`,
	Args: cobra.ExactArgs(2),
	RunE: runConfigSet,
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize global configuration",
	Long:  "Creates the global configuration file with defaults",
	RunE:  runConfigInit,
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configInitCmd)
	rootCmd.AddCommand(configCmd)
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	cfg, err := config.LoadGlobalConfig(homeDir)
	if err != nil {
		cli.PrintError("Failed to load config: %v", err)
		return nil
	}

	cli.PrintTitle("MageBox Global Configuration")
	fmt.Println()

	configPath := config.GlobalConfigPath(homeDir)
	if config.GlobalConfigExists(homeDir) {
		fmt.Printf("Config file: %s\n", cli.Path(configPath))
	} else {
		fmt.Printf("Config file: %s (using defaults)\n", cli.Subtitle("not created"))
	}
	fmt.Println()

	fmt.Printf("  %-14s %s\n", "dns_mode:", cli.Highlight(cfg.DNSMode))
	fmt.Printf("  %-14s %s\n", "default_php:", cli.Highlight(cfg.DefaultPHP))
	fmt.Printf("  %-14s %s\n", "tld:", cli.Highlight(cfg.TLD))
	fmt.Printf("  %-14s %s\n", "portainer:", cli.Highlight(fmt.Sprintf("%v", cfg.Portainer)))
	fmt.Printf("  %-14s %s\n", "auto_start:", cli.Highlight(fmt.Sprintf("%v", cfg.AutoStart)))

	fmt.Println(cli.Header("Default Services"))
	if cfg.DefaultServices.MySQL != "" {
		fmt.Printf("  %-14s %s\n", "mysql:", cli.Highlight(cfg.DefaultServices.MySQL))
	}
	if cfg.DefaultServices.MariaDB != "" {
		fmt.Printf("  %-14s %s\n", "mariadb:", cli.Highlight(cfg.DefaultServices.MariaDB))
	}
	fmt.Printf("  %-14s %s\n", "redis:", cli.Highlight(fmt.Sprintf("%v", cfg.DefaultServices.Redis)))
	if cfg.DefaultServices.OpenSearch != "" {
		fmt.Printf("  %-14s %s\n", "opensearch:", cli.Highlight(cfg.DefaultServices.OpenSearch))
	}
	fmt.Printf("  %-14s %s\n", "rabbitmq:", cli.Highlight(fmt.Sprintf("%v", cfg.DefaultServices.RabbitMQ)))
	fmt.Printf("  %-14s %s\n", "mailpit:", cli.Highlight(fmt.Sprintf("%v", cfg.DefaultServices.Mailpit)))

	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	cfg, err := config.LoadGlobalConfig(homeDir)
	if err != nil {
		cli.PrintError("Failed to load config: %v", err)
		return nil
	}

	oldTLD := cfg.GetTLD()

	switch key {
	case "dns_mode":
		if value != "hosts" && value != "dnsmasq" {
			cli.PrintError("Invalid value for dns_mode. Use 'hosts' or 'dnsmasq'")
			return nil
		}
		cfg.DNSMode = value
	case "default_php":
		cfg.DefaultPHP = value
	case "tld":
		cfg.TLD = value
	case "portainer":
		cfg.Portainer = (value == "true" || value == "1" || value == "yes")
	case "auto_start":
		cfg.AutoStart = (value == "true" || value == "1" || value == "yes")
	default:
		cli.PrintError("Unknown configuration key: %s", key)
		fmt.Println()
		cli.PrintInfo("Available keys: dns_mode, default_php, tld, portainer, auto_start")
		return nil
	}

	if err := config.SaveGlobalConfig(homeDir, cfg); err != nil {
		cli.PrintError("Failed to save config: %v", err)
		return nil
	}

	cli.PrintSuccess("Configuration updated: %s = %s", key, value)

	// If TLD changed and dnsmasq is configured, reconfigure DNS
	if key == "tld" && value != oldTLD {
		p, err := platform.Detect()
		if err != nil {
			cli.PrintWarning("Could not detect platform for DNS reconfiguration: %v", err)
			return nil
		}

		dnsMgr := dns.NewDnsmasqManager(p)
		if dnsMgr.IsConfigured() {
			cli.PrintInfo("Reconfiguring DNS for new TLD: %s", value)

			// Remove old macOS resolver if exists
			if p.Type == platform.Darwin {
				_ = dnsMgr.Remove() // This removes the old resolver file
			}

			// Reconfigure dnsmasq
			if err := dnsMgr.Configure(); err != nil {
				cli.PrintWarning("Failed to reconfigure DNS: %v", err)
				cli.PrintInfo("Run %s to reconfigure manually", cli.Command("magebox dns setup"))
				return nil
			}

			// Restart dnsmasq
			if err := dnsMgr.Restart(); err != nil {
				cli.PrintWarning("Failed to restart dnsmasq: %v", err)
				return nil
			}

			cli.PrintSuccess("DNS reconfigured for *.%s domains", value)
		}
	}

	return nil
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := config.GlobalConfigPath(homeDir)

	if config.GlobalConfigExists(homeDir) {
		cli.PrintWarning("Configuration already exists at %s", configPath)
		return nil
	}

	if err := config.InitGlobalConfig(homeDir); err != nil {
		cli.PrintError("Failed to initialize config: %v", err)
		return nil
	}

	cli.PrintSuccess("Created global configuration at %s", configPath)
	fmt.Println()
	cli.PrintInfo("Edit the file or use %s", cli.Command("magebox config set <key> <value>"))
	return nil
}

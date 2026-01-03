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

var dnsCmd = &cobra.Command{
	Use:   "dns",
	Short: "DNS configuration",
	Long:  "Manage DNS resolution for local domains",
}

var dnsSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Setup dnsmasq for wildcard DNS",
	Long: `Sets up dnsmasq to resolve local development domains to localhost.

This eliminates the need to add each domain to /etc/hosts manually.
Requires dnsmasq to be installed first.
The TLD used is configured via 'magebox config set tld <value>' (default: test).`,
	RunE: runDnsSetup,
}

var dnsStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show DNS configuration status",
	Long:  "Shows current DNS resolution status and configuration",
	RunE:  runDnsStatus,
}

func init() {
	dnsCmd.AddCommand(dnsSetupCmd)
	dnsCmd.AddCommand(dnsStatusCmd)
	rootCmd.AddCommand(dnsCmd)
}

func runDnsSetup(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	// Load global config for TLD
	homeDir, _ := os.UserHomeDir()
	globalCfg, _ := config.LoadGlobalConfig(homeDir)
	tld := globalCfg.GetTLD()

	dnsMgr := dns.NewDnsmasqManager(p)

	// Check if dnsmasq is installed
	if !dnsMgr.IsInstalled() {
		cli.PrintError("dnsmasq is not installed")
		fmt.Println()
		fmt.Printf("Install it with: %s\n", cli.Command(dnsMgr.InstallCommand()))
		return nil
	}

	cli.PrintInfo("Setting up dnsmasq for *.%s domain resolution...", tld)

	// Configure dnsmasq
	if err := dnsMgr.Configure(); err != nil {
		cli.PrintError("Failed to configure dnsmasq: %v", err)
		return nil
	}

	// Start/restart dnsmasq
	if dnsMgr.IsRunning() {
		if err := dnsMgr.Restart(); err != nil {
			cli.PrintError("Failed to restart dnsmasq: %v", err)
			return nil
		}
	} else {
		if err := dnsMgr.Start(); err != nil {
			cli.PrintError("Failed to start dnsmasq: %v", err)
			return nil
		}
	}

	// Update global config
	globalCfg.DNSMode = "dnsmasq"
	_ = config.SaveGlobalConfig(homeDir, globalCfg)

	cli.PrintSuccess("dnsmasq configured successfully!")
	fmt.Println()
	cli.PrintInfo("All *.%s domains now resolve to 127.0.0.1", tld)
	fmt.Println(cli.Bullet("No need to edit /etc/hosts for new projects"))

	// Show test command with correct DNS server address
	dnsServer := "127.0.0.1"
	if p.Type == platform.Linux {
		dnsServer = "127.0.0.2"
	}
	fmt.Println(cli.Bullet("Test with: " + cli.Command(fmt.Sprintf("dig test.%s @%s", tld, dnsServer))))

	return nil
}

func runDnsStatus(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	cli.PrintTitle("DNS Configuration Status")
	fmt.Println()

	// Check global config
	homeDir, _ := os.UserHomeDir()
	globalCfg, _ := config.LoadGlobalConfig(homeDir)

	fmt.Printf("DNS Mode:      %s\n", cli.Highlight(globalCfg.DNSMode))
	fmt.Printf("TLD:           %s\n", cli.Highlight(globalCfg.GetTLD()))

	// Check dnsmasq status
	dnsMgr := dns.NewDnsmasqManager(p)
	status := dnsMgr.GetStatus()

	fmt.Println(cli.Header("dnsmasq"))
	fmt.Printf("  %-14s %s\n", "Installed:", cli.StatusInstalled(status.Installed))
	fmt.Printf("  %-14s %s\n", "Configured:", cli.StatusInstalled(status.Configured))
	fmt.Printf("  %-14s %s\n", "Running:", cli.Status(status.Running))

	if status.Running {
		fmt.Printf("  %-14s %s\n", "Resolution:", cli.Status(status.Resolving))
		if !status.Resolving {
			cli.PrintWarning("DNS resolution test failed. Check dnsmasq configuration.")
		}
	}

	if !status.Installed {
		fmt.Println()
		cli.PrintInfo("To enable wildcard DNS:")
		fmt.Println(cli.Bullet("Install: " + cli.Command(dnsMgr.InstallCommand())))
		fmt.Println(cli.Bullet("Setup: " + cli.Command("magebox dns setup")))
	}

	return nil
}

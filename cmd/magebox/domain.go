package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/qoliber/magebox/internal/cli"
	"github.com/qoliber/magebox/internal/config"
	"github.com/qoliber/magebox/internal/dns"
	"github.com/qoliber/magebox/internal/nginx"
	"github.com/qoliber/magebox/internal/ssl"
)

var domainCmd = &cobra.Command{
	Use:   "domain",
	Short: "Domain management",
	Long:  "Manage project domains (add, remove, list)",
}

var domainAddCmd = &cobra.Command{
	Use:   "add <host>",
	Short: "Add a domain to the project",
	Long: `Adds a new domain to the current project's .magebox configuration.

Example:
  magebox domain add store.test
  magebox domain add store.test --store-code=german
  magebox domain add store.test --root=pub --ssl=false`,
	Args: cobra.ExactArgs(1),
	RunE: runDomainAdd,
}

var domainRemoveCmd = &cobra.Command{
	Use:   "remove <host>",
	Short: "Remove a domain from the project",
	Long: `Removes a domain from the current project's .magebox configuration.

Example:
  magebox domain remove store.test`,
	Args: cobra.ExactArgs(1),
	RunE: runDomainRemove,
}

var domainListCmd = &cobra.Command{
	Use:   "list",
	Short: "List project domains",
	Long:  "Lists all domains configured for the current project",
	RunE:  runDomainList,
}

var (
	domainStoreCode string
	domainRoot      string
	domainSSL       bool
)

func init() {
	domainAddCmd.Flags().StringVar(&domainStoreCode, "store-code", "", "Magento store code (default: \"default\")")
	domainAddCmd.Flags().StringVar(&domainRoot, "root", "pub", "Document root relative to project")
	domainAddCmd.Flags().BoolVar(&domainSSL, "ssl", true, "Enable SSL for the domain")

	domainCmd.AddCommand(domainAddCmd)
	domainCmd.AddCommand(domainRemoveCmd)
	domainCmd.AddCommand(domainListCmd)
	rootCmd.AddCommand(domainCmd)
}

func runDomainAdd(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	cfg, err := config.LoadFromPath(cwd)
	if err != nil {
		cli.PrintError("Failed to load config: %v", err)
		return nil
	}

	host := args[0]

	// Check if domain already exists
	for _, d := range cfg.Domains {
		if d.Host == host {
			cli.PrintError("Domain %s already exists in the configuration", host)
			return nil
		}
	}

	// Create new domain
	newDomain := config.Domain{
		Host:        host,
		Root:        domainRoot,
		MageRunCode: domainStoreCode,
	}

	// Only set SSL if explicitly changed from default (true)
	if !domainSSL {
		sslFlag := false
		newDomain.SSL = &sslFlag
	}

	cfg.Domains = append(cfg.Domains, newDomain)

	// Save config
	if err := config.SaveToPath(cfg, cwd); err != nil {
		cli.PrintError("Failed to save config: %v", err)
		return nil
	}

	cli.PrintSuccess("Added domain: %s", host)

	// Generate SSL certificate and nginx vhost
	p, err := getPlatform()
	if err != nil {
		return err
	}

	sslManager := ssl.NewManager(p)

	// Generate SSL certificate for the new domain
	if newDomain.IsSSLEnabled() {
		fmt.Println("Generating SSL certificate...")
		if _, err := sslManager.GenerateCert(host); err != nil {
			cli.PrintWarning("SSL certificate generation failed: %v", err)
		}
	}

	// Regenerate vhosts
	fmt.Println("Regenerating nginx vhosts...")
	vhostGen := nginx.NewVhostGenerator(p, sslManager)
	if err := vhostGen.Generate(cfg, cwd); err != nil {
		cli.PrintWarning("Failed to regenerate vhosts: %v", err)
	}

	// Update DNS (hosts file)
	homeDir, _ := os.UserHomeDir()
	globalCfg, _ := config.LoadGlobalConfig(homeDir)
	if globalCfg.UseHosts() {
		fmt.Println("Updating /etc/hosts...")
		hostsManager := dns.NewHostsManager(p)
		domains := make([]string, len(cfg.Domains))
		for i, d := range cfg.Domains {
			domains[i] = d.Host
		}
		if err := hostsManager.AddDomains(domains); err != nil {
			cli.PrintWarning("Failed to update hosts: %v", err)
		}
	}

	// Reload nginx
	fmt.Println("Reloading nginx...")
	ngxController := nginx.NewController(p)
	if err := ngxController.Test(); err != nil {
		cli.PrintError("Nginx config test failed: %v", err)
		return nil
	}
	if err := ngxController.Reload(); err != nil {
		cli.PrintWarning("Failed to reload nginx: %v", err)
	}

	fmt.Println()
	cli.PrintInfo("Domain %s configured with store code: %s", host, newDomain.GetStoreCode())

	return nil
}

func runDomainRemove(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	cfg, err := config.LoadFromPath(cwd)
	if err != nil {
		cli.PrintError("Failed to load config: %v", err)
		return nil
	}

	host := args[0]

	// Find and remove domain
	found := false
	newDomains := make([]config.Domain, 0, len(cfg.Domains))
	for _, d := range cfg.Domains {
		if d.Host == host {
			found = true
		} else {
			newDomains = append(newDomains, d)
		}
	}

	if !found {
		cli.PrintError("Domain %s not found in the configuration", host)
		return nil
	}

	if len(newDomains) == 0 {
		cli.PrintError("Cannot remove the last domain. At least one domain is required.")
		return nil
	}

	cfg.Domains = newDomains

	// Save config
	if err := config.SaveToPath(cfg, cwd); err != nil {
		cli.PrintError("Failed to save config: %v", err)
		return nil
	}

	cli.PrintSuccess("Removed domain: %s", host)

	// Regenerate vhosts (will remove the old vhost file)
	p, err := getPlatform()
	if err != nil {
		return err
	}

	sslManager := ssl.NewManager(p)
	vhostGen := nginx.NewVhostGenerator(p, sslManager)

	// Remove the specific vhost file
	fmt.Println("Removing nginx vhost...")
	vhostsDir := vhostGen.VhostsDir()
	vhostFile := filepath.Join(vhostsDir, fmt.Sprintf("%s-%s.conf", cfg.Name, host))
	if err := os.Remove(vhostFile); err != nil && !os.IsNotExist(err) {
		cli.PrintWarning("Failed to remove vhost file: %v", err)
	}

	// Remove from hosts file
	homeDir, _ := os.UserHomeDir()
	globalCfg, _ := config.LoadGlobalConfig(homeDir)
	if globalCfg.UseHosts() {
		fmt.Println("Removing from /etc/hosts...")
		hostsManager := dns.NewHostsManager(p)
		if err := hostsManager.RemoveDomains([]string{host}); err != nil {
			cli.PrintWarning("Failed to remove %s from hosts: %v", host, err)
		}
	}

	// Reload nginx
	fmt.Println("Reloading nginx...")
	ngxController := nginx.NewController(p)
	if err := ngxController.Test(); err != nil {
		cli.PrintError("Nginx config test failed: %v", err)
		return nil
	}
	if err := ngxController.Reload(); err != nil {
		cli.PrintWarning("Failed to reload nginx: %v", err)
	}

	return nil
}

func runDomainList(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	cfg, err := config.LoadFromPath(cwd)
	if err != nil {
		cli.PrintError("Failed to load config: %v", err)
		return nil
	}

	cli.PrintTitle("Project Domains: %s", cfg.Name)
	fmt.Println()

	for _, d := range cfg.Domains {
		sslStatus := "SSL"
		if !d.IsSSLEnabled() {
			sslStatus = "HTTP"
		}

		protocol := "https"
		if !d.IsSSLEnabled() {
			protocol = "http"
		}

		fmt.Printf("  %s\n", cli.Highlight(d.Host))
		fmt.Printf("    URL:        %s://%s\n", protocol, d.Host)
		fmt.Printf("    Root:       %s\n", d.GetRoot())
		fmt.Printf("    Store Code: %s\n", d.GetStoreCode())
		fmt.Printf("    SSL:        %s\n", sslStatus)
		fmt.Println()
	}

	return nil
}

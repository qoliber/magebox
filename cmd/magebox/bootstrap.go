/**
 * Created by Qoliber
 *
 * @category    Qoliber
 * @package     Qoliber_MageBox
 * @author      Jakub Winkler <jwinkler@qoliber.com>
 */

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/qoliber/magebox/internal/cli"
	"github.com/qoliber/magebox/internal/config"
	"github.com/qoliber/magebox/internal/dns"
	"github.com/qoliber/magebox/internal/docker"
	"github.com/qoliber/magebox/internal/nginx"
	"github.com/qoliber/magebox/internal/php"
	"github.com/qoliber/magebox/internal/platform"
	"github.com/qoliber/magebox/internal/portforward"
	"github.com/qoliber/magebox/internal/ssl"
)

var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Initialize MageBox environment",
	Long: `Sets up the MageBox development environment for first-time use.

This command performs the following steps:
  1. Checks all required dependencies (Docker, Nginx, mkcert, PHP)
  2. Initializes global configuration (~/.magebox/config.yaml)
  3. Sets up mkcert CA for HTTPS support
  4. Configures port forwarding (macOS: 80→8080, 443→8443)
  5. Configures Nginx to include MageBox vhosts
  6. Creates and starts Docker services (MySQL, Redis, Mailpit)
  7. Sets up DNS resolution (dnsmasq or /etc/hosts)

Run this once after installing MageBox to prepare your system.`,
	RunE: runBootstrap,
}

func runBootstrap(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	cli.PrintLogoSmall(version)
	fmt.Println()
	cli.PrintTitle("MageBox Bootstrap")
	fmt.Println()
	fmt.Println("Setting up MageBox development environment...")
	fmt.Println()

	// Track any errors but continue setup
	var errors []string

	// Step 1: Check dependencies
	fmt.Println(cli.Header("Step 1: Checking Dependencies"))

	// Check Docker
	dockerInstalled := platform.CommandExists("docker")
	fmt.Printf("  %-15s %s\n", "Docker:", cli.StatusInstalled(dockerInstalled))
	if !dockerInstalled {
		errors = append(errors, "Docker is not installed. Install: "+p.DockerInstallCommand())
	} else {
		// Check if Docker daemon is running
		dockerCmd := exec.Command("docker", "info")
		if dockerCmd.Run() != nil {
			cli.PrintWarning("Docker is installed but not running. Please start Docker.")
			errors = append(errors, "Docker daemon is not running")
		}
	}

	// Check Nginx
	nginxInstalled := p.IsNginxInstalled()
	fmt.Printf("  %-15s %s\n", "Nginx:", cli.StatusInstalled(nginxInstalled))
	if !nginxInstalled {
		errors = append(errors, "Nginx is not installed. Install: "+p.NginxInstallCommand())
	}

	// Check mkcert
	mkcertInstalled := platform.CommandExists("mkcert")
	fmt.Printf("  %-15s %s\n", "mkcert:", cli.StatusInstalled(mkcertInstalled))
	if !mkcertInstalled {
		errors = append(errors, "mkcert is not installed. Install: "+p.MkcertInstallCommand())
	}

	// Check PHP versions
	detector := php.NewDetector(p)
	phpInstalled := false
	for _, v := range php.SupportedVersions {
		version := detector.Detect(v)
		if version.Installed {
			phpInstalled = true
			break
		}
	}
	fmt.Printf("  %-15s %s\n", "PHP:", cli.StatusInstalled(phpInstalled))
	if !phpInstalled {
		errors = append(errors, "No PHP version installed. Install: "+p.PHPInstallCommand("8.2"))
	}

	fmt.Println()

	// If critical dependencies are missing, show errors and exit
	if !dockerInstalled || !nginxInstalled || !mkcertInstalled {
		cli.PrintError("Missing critical dependencies!")
		fmt.Println()
		for _, e := range errors {
			fmt.Printf("  %s %s\n", cli.Error("•"), e)
		}
		fmt.Println()
		cli.PrintInfo("Install missing dependencies and run " + cli.Command("magebox bootstrap") + " again")
		return nil
	}

	// Step 2: Initialize global config
	fmt.Println(cli.Header("Step 2: Global Configuration"))

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	globalCfg, err := config.LoadGlobalConfig(homeDir)
	if err != nil {
		// Config doesn't exist, create it
		if err := config.InitGlobalConfig(homeDir); err != nil {
			cli.PrintError("Failed to create config: %v", err)
			return err
		}
		globalCfg, _ = config.LoadGlobalConfig(homeDir)
		fmt.Printf("  Created %s\n", cli.Highlight(config.GlobalConfigPath(homeDir)))
	} else {
		fmt.Printf("  Config exists: %s\n", cli.Highlight(config.GlobalConfigPath(homeDir)))
	}
	fmt.Println()

	// Step 3: Setup mkcert CA
	fmt.Println(cli.Header("Step 3: SSL Certificate Authority"))

	sslMgr := ssl.NewManager(p)
	if sslMgr.IsCAInstalled() {
		fmt.Println("  Local CA already installed " + cli.Success("✓"))
	} else {
		fmt.Print("  Installing local CA... ")
		if err := sslMgr.InstallCA(); err != nil {
			fmt.Println(cli.Error("failed"))
			cli.PrintWarning("SSL setup failed: %v", err)
		} else {
			fmt.Println(cli.Success("done"))
		}
	}
	fmt.Println()

	// Step 4: Setup Port Forwarding (macOS only)
	if p.Type == platform.Darwin {
		fmt.Println(cli.Header("Step 4: Port Forwarding Setup"))
		fmt.Println("  Setting up transparent port forwarding (80→8080, 443→8443)")
		fmt.Println("  This allows Nginx to run as your user without sudo")
		fmt.Println()

		pfMgr := portforward.NewManager()
		if pfMgr.IsInstalled() {
			fmt.Println("  Port forwarding already configured " + cli.Success("✓"))
		} else {
			fmt.Print("  Installing pf rules and LaunchDaemon... ")
			if err := pfMgr.Setup(); err != nil {
				fmt.Println(cli.Error("failed"))
				cli.PrintWarning("Port forwarding setup failed: %v", err)
				cli.PrintWarning("You may need to manually configure port forwarding")
				errors = append(errors, "Port forwarding setup failed")
			} else {
				fmt.Println(cli.Success("done"))
			}
		}
		fmt.Println()
	}

	// Step 5: Setup Nginx configuration
	fmt.Println(cli.Header("Step 5: Nginx Configuration"))

	nginxCtrl := nginx.NewController(p)
	fmt.Printf("  Nginx config: %s\n", cli.Highlight(nginxCtrl.GetNginxConfPath()))

	// Create vhosts directory
	vhostsDir := filepath.Join(p.MageBoxDir(), "nginx", "vhosts")
	if err := os.MkdirAll(vhostsDir, 0755); err != nil {
		cli.PrintWarning("Failed to create vhosts dir: %v", err)
	} else {
		fmt.Printf("  Vhosts dir: %s\n", cli.Highlight(vhostsDir))
	}

	// Setup nginx.conf include
	fmt.Print("  Adding MageBox include... ")
	if err := nginxCtrl.SetupNginxConfig(); err != nil {
		fmt.Println(cli.Error("failed"))
		cli.PrintWarning("Nginx config update failed: %v", err)
		cli.PrintInfo("You may need to manually add to nginx.conf:")
		fmt.Printf("    include %s/*.conf;\n", vhostsDir)
	} else {
		fmt.Println(cli.Success("done"))
	}

	// Test and reload nginx
	fmt.Print("  Testing nginx config... ")
	if err := nginxCtrl.Test(); err != nil {
		fmt.Println(cli.Error("failed"))
	} else {
		fmt.Println(cli.Success("ok"))
		fmt.Print("  Starting nginx... ")
		if err := nginxCtrl.Start(); err != nil {
			// Try reload if already running
			if err := nginxCtrl.Reload(); err != nil {
				fmt.Println(cli.Error("failed"))
			} else {
				fmt.Println(cli.Success("reloaded"))
			}
		} else {
			fmt.Println(cli.Success("started"))
		}
	}
	fmt.Println()

	// Step 6: Setup Docker services
	fmt.Println(cli.Header("Step 6: Docker Services"))

	composeGen := docker.NewComposeGenerator(p)
	composeFile := composeGen.ComposeFilePath()

	fmt.Print("  Generating docker-compose.yml... ")
	if err := composeGen.GenerateDefaultServices(globalCfg); err != nil {
		fmt.Println(cli.Error("failed"))
		cli.PrintWarning("Docker compose generation failed: %v", err)
	} else {
		fmt.Println(cli.Success("done"))
		fmt.Printf("    %s\n", cli.Highlight(composeFile))
	}

	// Start Docker services
	fmt.Print("  Starting containers... ")
	dockerCtrl := docker.NewDockerController(composeFile)
	if err := dockerCtrl.Up(); err != nil {
		fmt.Println(cli.Error("failed"))
		cli.PrintWarning("Docker services failed to start: %v", err)
	} else {
		fmt.Println(cli.Success("done"))
	}

	// List running services
	if services, err := dockerCtrl.GetRunningServices(); err == nil && len(services) > 0 {
		fmt.Println("  Running services:")
		for _, svc := range services {
			fmt.Printf("    %s %s\n", cli.Success("✓"), svc)
		}
	}
	fmt.Println()

	// Step 7: DNS setup (informational)
	fmt.Println(cli.Header("Step 7: DNS Configuration"))

	if globalCfg.UseDnsmasq() {
		dnsManager := dns.NewDnsmasqManager(p)
		if dnsManager.IsConfigured() {
			fmt.Println("  dnsmasq configured for *.test " + cli.Success("✓"))
		} else {
			fmt.Println("  dnsmasq not yet configured")
			cli.PrintInfo("Run " + cli.Command("magebox dns setup") + " to configure wildcard DNS")
		}
	} else {
		fmt.Println("  Using /etc/hosts mode")
		cli.PrintInfo("Domains will be added to /etc/hosts when you run " + cli.Command("magebox start"))
	}
	fmt.Println()

	// Summary
	cli.PrintTitle("Bootstrap Complete!")
	fmt.Println()

	if len(errors) > 0 {
		cli.PrintWarning("Completed with warnings:")
		for _, e := range errors {
			fmt.Printf("  %s %s\n", cli.Warning("!"), e)
		}
		fmt.Println()
	}

	cli.PrintSuccess("MageBox is ready!")
	fmt.Println()
	fmt.Println("Services available:")
	fmt.Printf("  MySQL 8.0:    %s (root password: magebox)\n", cli.URL("localhost:33080"))
	fmt.Printf("  Redis:        %s\n", cli.URL("localhost:6379"))
	fmt.Printf("  Mailpit:      %s\n", cli.URL("http://localhost:8025"))
	if globalCfg.Portainer {
		fmt.Printf("  Portainer:    %s\n", cli.URL("http://localhost:9000"))
	}
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println(cli.Bullet("cd into your Magento project directory"))
	fmt.Println(cli.Bullet("Run " + cli.Command("magebox init") + " to create .magebox config"))
	fmt.Println(cli.Bullet("Run " + cli.Command("magebox start") + " to start your project"))
	fmt.Println()

	return nil
}

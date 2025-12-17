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

	"github.com/qoliber/magebox/internal/bootstrap"
	"github.com/qoliber/magebox/internal/bootstrap/installer"
	"github.com/qoliber/magebox/internal/cli"
	"github.com/qoliber/magebox/internal/config"
	"github.com/qoliber/magebox/internal/dns"
	"github.com/qoliber/magebox/internal/docker"
	"github.com/qoliber/magebox/internal/nginx"
	"github.com/qoliber/magebox/internal/php"
	"github.com/qoliber/magebox/internal/phpwrapper"
	"github.com/qoliber/magebox/internal/platform"
	"github.com/qoliber/magebox/internal/portforward"
	"github.com/qoliber/magebox/internal/ssl"
	"github.com/qoliber/magebox/internal/verbose"
)

var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Initialize MageBox environment",
	Long: `Sets up the MageBox development environment for first-time use.

This command performs the following steps:
  1. Validates OS version and checks dependencies (Docker, Nginx, mkcert)
  2. Installs PHP versions 8.1-8.4 via platform package manager
  3. Initializes global configuration (~/.magebox/config.yaml)
  4. Sets up mkcert CA for HTTPS support
  5. Configures port forwarding (macOS: 80→8080, 443→8443)
  6. Configures Nginx to include MageBox vhosts
  7. Creates and starts Docker services (MySQL, Redis, Mailpit)
  8. Sets up DNS resolution (dnsmasq or /etc/hosts)
  9. Installs CLI wrappers (PHP, Composer, Blackfire) for automatic version switching
  10. Configures sudoers for passwordless service control (Linux)

Supported platforms:
  - macOS 12-15 (Monterey, Ventura, Sonoma, Sequoia)
  - Fedora 38-43
  - Ubuntu 20.04, 22.04, 24.04
  - Arch Linux (rolling release)

Run this once after installing MageBox to prepare your system.`,
	RunE: runBootstrap,
}

func init() {
	rootCmd.AddCommand(bootstrapCmd)
}

func runBootstrap(cmd *cobra.Command, args []string) error {
	verbose.Section("Bootstrap Starting")

	p, err := getPlatform()
	if err != nil {
		verbose.Debug("Failed to detect platform: %v", err)
		return err
	}

	cli.PrintLogoSmall(version)
	fmt.Println()
	cli.PrintTitle("MageBox Bootstrap")
	fmt.Println()

	// Step 0: Platform Detection - show warning for untested distros
	fmt.Println(cli.Header("Step 0: Platform Detection"))

	if p.Type == platform.Linux {
		fmt.Printf("  OS: %s\n", cli.Highlight(p.DistroName))
		fmt.Printf("  Family: %s\n", p.LinuxDistro)

		if p.LinuxDistro == platform.DistroUnknown {
			fmt.Println()
			cli.PrintError("Unsupported Linux distribution")
			fmt.Println()
			fmt.Println("  MageBox requires a distribution based on:")
			fmt.Println("    - Debian/Ubuntu (apt)")
			fmt.Println("    - Fedora/RHEL/CentOS (dnf)")
			fmt.Println("    - Arch Linux (pacman)")
			fmt.Println()
			cli.PrintInfo("If your distro is based on one of these, please report this at:")
			fmt.Println("  https://github.com/qoliber/magebox/issues")
			return fmt.Errorf("unsupported distribution")
		}

		if !p.DistroTested {
			fmt.Println("  Status: " + cli.Warning("Not officially tested"))
			fmt.Println()
			cli.PrintWarning("MageBox has not been tested on %s", p.DistroName)
			fmt.Println("  It should work since it's based on " + string(p.LinuxDistro) + ", but you may encounter issues.")
			fmt.Println("  Please report any problems at: https://github.com/qoliber/magebox/issues")
			fmt.Println()

			// Ask user if they want to continue
			fmt.Print("Continue with bootstrap? [Y/n]: ")
			reader := bufio.NewReader(os.Stdin)
			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(strings.ToLower(answer))

			if answer != "" && answer != "y" && answer != "yes" {
				fmt.Println()
				cli.PrintInfo("Bootstrap canceled")
				return nil
			}
		} else {
			fmt.Println("  Status: " + cli.Success("Supported"))
		}
	}
	fmt.Println()

	// Create bootstrapper for this platform
	bootstrapper, err := bootstrap.NewBootstrapper(p)
	if err != nil {
		return fmt.Errorf("unsupported platform: %w", err)
	}

	// Validate OS version for additional info
	osInfo, err := bootstrapper.ValidateOS()
	if err == nil && osInfo.Version != "" {
		fmt.Printf("  Version: %s", osInfo.Version)
		if osInfo.Codename != "" {
			fmt.Printf(" (%s)", osInfo.Codename)
		}
		fmt.Println()
		fmt.Println()
	}

	fmt.Println("Setting up MageBox development environment...")
	fmt.Println()

	// Track any errors but continue setup
	var errors []string

	// Step 1: Check dependencies
	fmt.Println(cli.Header("Step 1: Checking Dependencies"))

	// Check Docker
	verbose.Debug("Checking Docker installation...")
	dockerInstalled := platform.CommandExists("docker")
	fmt.Printf("  %-15s %s\n", "Docker:", cli.StatusInstalled(dockerInstalled))
	if !dockerInstalled {
		verbose.Debug("Docker not found in PATH")
		errors = append(errors, "Docker is not installed. Install: "+bootstrapper.DockerInstallInstructions())
	} else {
		verbose.Debug("Docker binary found, checking daemon...")
		// Check if Docker daemon is running
		if !bootstrapper.CheckDockerRunning() {
			verbose.Debug("Docker daemon is not running")
			cli.PrintWarning("Docker is installed but not running. Please start Docker.")
			errors = append(errors, "Docker daemon is not running")
		} else {
			verbose.Debug("Docker daemon is running")
		}
	}

	// Check Nginx
	verbose.Debug("Checking Nginx installation...")
	nginxInstalled := p.IsNginxInstalled()
	verbose.Debug("Nginx installed: %v (binary: %s)", nginxInstalled, p.NginxBinary())
	fmt.Printf("  %-15s %s\n", "Nginx:", cli.StatusInstalled(nginxInstalled))
	if !nginxInstalled {
		errors = append(errors, "Nginx is not installed. Install: "+p.NginxInstallCommand())
	}

	// Check mkcert
	verbose.Debug("Checking mkcert installation...")
	mkcertInstalled := platform.CommandExists("mkcert")
	verbose.Debug("mkcert installed: %v", mkcertInstalled)
	fmt.Printf("  %-15s %s\n", "mkcert:", cli.StatusInstalled(mkcertInstalled))
	if !mkcertInstalled {
		errors = append(errors, "mkcert is not installed. Install: "+p.MkcertInstallCommand())
	}

	// Check PHP versions
	verbose.Debug("Detecting installed PHP versions...")
	detector := php.NewDetector(p)
	installedPHPVersions := detector.DetectInstalled()
	for _, ver := range installedPHPVersions {
		verbose.Debug("Found PHP %s at %s", ver.Version, ver.PHPBinary)
	}
	fmt.Printf("  %-15s %s (%d versions)\n", "PHP:", cli.StatusInstalled(len(installedPHPVersions) > 0), len(installedPHPVersions))

	fmt.Println()

	// If critical dependencies are missing, offer to install them
	if !dockerInstalled || !nginxInstalled || !mkcertInstalled {
		cli.PrintWarning("Missing dependencies detected!")
		fmt.Println()

		// Show what's missing
		var missingDeps []string
		if !nginxInstalled {
			missingDeps = append(missingDeps, "Nginx")
		}
		if !mkcertInstalled {
			missingDeps = append(missingDeps, "mkcert")
		}

		fmt.Println("  Missing: " + cli.Highlight(strings.Join(missingDeps, ", ")))
		fmt.Println()

		// Ask user if they want to install
		fmt.Print("Would you like MageBox to install these dependencies? [Y/n]: ")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))

		if answer == "" || answer == "y" || answer == "yes" {
			fmt.Println()
			inst := bootstrapper.GetInstaller()

			if !nginxInstalled {
				fmt.Println("  Installing Nginx...")
				if err := inst.InstallNginx(); err != nil {
					cli.PrintError("Failed to install Nginx: %v", err)
					fmt.Println()
					cli.PrintInfo("Please install manually and run %s again", cli.Command("magebox bootstrap"))
					return nil
				}
				fmt.Printf("  Nginx installed %s\n", cli.Success("✓"))
				fmt.Println()
			}

			if !mkcertInstalled {
				fmt.Println("  Installing mkcert...")
				if err := inst.InstallMkcert(); err != nil {
					cli.PrintError("Failed to install mkcert: %v", err)
					fmt.Println()
					cli.PrintInfo("Please install manually and run %s again", cli.Command("magebox bootstrap"))
					return nil
				}
				fmt.Printf("  mkcert installed %s\n", cli.Success("✓"))
				fmt.Println()
			}

			// Re-check after installation
			nginxInstalled = p.IsNginxInstalled()
			mkcertInstalled = platform.CommandExists("mkcert")
		} else {
			fmt.Println()
			cli.PrintInfo("Install missing dependencies and run %s again", cli.Command("magebox bootstrap"))
			return nil
		}

		// Final check - if still missing, exit
		if !nginxInstalled || !mkcertInstalled {
			cli.PrintError("Some dependencies failed to install")
			return nil
		}
	}

	// Docker must be installed manually (too complex for auto-install)
	if !dockerInstalled {
		cli.PrintError("Docker is required but not installed")
		fmt.Println()
		fmt.Printf("  Install Docker: %s\n", cli.Command(bootstrapper.DockerInstallInstructions()))
		fmt.Println()
		cli.PrintInfo("After installing Docker, run %s again", cli.Command("magebox bootstrap"))
		return nil
	}

	// Setup sudoers early (Linux only) - needed before PHP-FPM can restart
	if p.Type == platform.Linux {
		sudoersFile := "/etc/sudoers.d/magebox"
		if _, err := os.Stat(sudoersFile); os.IsNotExist(err) {
			fmt.Print("  Setting up passwordless nginx/php-fpm control... ")
			if err := bootstrapper.ConfigureSudoers(); err != nil {
				fmt.Println(cli.Error("failed"))
				cli.PrintWarning("Failed to setup sudoers: %v", err)
				errors = append(errors, fmt.Sprintf("Sudoers setup: %v", err))
			} else {
				fmt.Println(cli.Success("done"))
			}
			fmt.Println()
		}
	}

	// Step 2: Install PHP versions
	fmt.Println(cli.Header("Step 2: PHP Installation"))
	fmt.Println()

	// All PHP versions we want for Magento compatibility (8.5 is bleeding edge/dev)
	allPHPVersions := installer.PHPVersions

	// Check which versions are installed vs missing
	installedMap := make(map[string]bool)
	for _, ver := range installedPHPVersions {
		installedMap[ver.Version] = true
	}

	var missingVersions []string
	for _, v := range allPHPVersions {
		if !installedMap[v] {
			missingVersions = append(missingVersions, v)
		}
	}

	// Show installed versions
	if len(installedPHPVersions) > 0 {
		fmt.Println("  Installed PHP versions:")
		for _, ver := range installedPHPVersions {
			fmt.Printf("    %s PHP %s\n", cli.Success("✓"), ver.Version)
		}
		fmt.Println()
	}

	// Offer to install missing versions
	if len(missingVersions) > 0 {
		fmt.Println("  Missing PHP versions: " + cli.Highlight(strings.Join(missingVersions, ", ")))
		fmt.Println()

		fmt.Print("Would you like to install missing PHP versions? [Y/n]: ")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))

		if answer == "" || answer == "y" || answer == "yes" {
			fmt.Println()
			inst := bootstrapper.GetInstaller()
			for _, phpVer := range missingVersions {
				verbose.Debug("Installing PHP %s...", phpVer)
				fmt.Printf("  Installing PHP %s...\n", phpVer)
				if err := inst.InstallPHP(phpVer); err != nil {
					verbose.Debug("PHP %s installation error: %v", phpVer, err)
					cli.PrintWarning("Failed to install PHP %s: %v", phpVer, err)
				} else {
					verbose.Debug("PHP %s installed successfully", phpVer)
					fmt.Printf("  PHP %s installed %s\n", phpVer, cli.Success("✓"))
				}
				fmt.Println()
			}
			// Refresh detected versions
			installedPHPVersions = detector.DetectInstalled()
		}
	} else {
		fmt.Println("  All recommended PHP versions are installed " + cli.Success("✓"))
	}

	// Configure PHP-FPM for installed versions
	if p.Type == platform.Linux && len(installedPHPVersions) > 0 {
		fmt.Println()
		fmt.Print("  Setting up PHP-FPM services... ")

		versions := make([]string, len(installedPHPVersions))
		for i, ver := range installedPHPVersions {
			versions[i] = ver.Version
		}

		if err := bootstrapper.ConfigurePHPFPM(versions); err != nil {
			fmt.Println(cli.Warning("done with warnings"))
			errors = append(errors, fmt.Sprintf("PHP-FPM setup: %v", err))
		} else {
			fmt.Println(cli.Success("done"))
		}
	}

	// Install Xdebug for all installed PHP versions
	if len(installedPHPVersions) > 0 {
		fmt.Println()
		fmt.Print("  Installing Xdebug for all PHP versions... ")
		xdebugErrors := 0
		for _, ver := range installedPHPVersions {
			if err := bootstrapper.InstallXdebug(ver.Version); err != nil {
				xdebugErrors++
			}
		}
		if xdebugErrors > 0 {
			fmt.Println(cli.Warning("done with warnings"))
		} else {
			fmt.Println(cli.Success("done"))
		}
	}

	// Configure PHP INI settings for Magento (memory_limit, max_execution_time)
	if len(installedPHPVersions) > 0 {
		fmt.Println()
		fmt.Print("  Configuring PHP INI for Magento... ")
		versions := make([]string, len(installedPHPVersions))
		for i, ver := range installedPHPVersions {
			versions[i] = ver.Version
		}
		if err := bootstrapper.GetInstaller().ConfigurePHPINI(versions); err != nil {
			fmt.Println(cli.Warning("done with warnings"))
			cli.PrintWarning("PHP INI config: %v", err)
		} else {
			fmt.Println(cli.Success("done"))
		}
	}

	// Install Blackfire profiler
	if len(installedPHPVersions) > 0 {
		fmt.Println()
		fmt.Print("  Installing Blackfire profiler... ")
		versions := make([]string, len(installedPHPVersions))
		for i, ver := range installedPHPVersions {
			versions[i] = ver.Version
		}
		if err := bootstrapper.GetInstaller().InstallBlackfire(versions); err != nil {
			fmt.Println(cli.Warning("skipped"))
			cli.PrintWarning("Blackfire: %v", err)
		} else {
			fmt.Println(cli.Success("done"))
		}
	}

	// Install Tideways profiler
	if len(installedPHPVersions) > 0 {
		fmt.Println()
		fmt.Print("  Installing Tideways profiler... ")
		versions := make([]string, len(installedPHPVersions))
		for i, ver := range installedPHPVersions {
			versions[i] = ver.Version
		}
		if err := bootstrapper.GetInstaller().InstallTideways(versions); err != nil {
			fmt.Println(cli.Warning("skipped"))
			cli.PrintWarning("Tideways: %v", err)
		} else {
			fmt.Println(cli.Success("done"))
		}
	}
	fmt.Println()

	// Step 3: Initialize global config
	fmt.Println(cli.Header("Step 3: Global Configuration"))

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

	// Step 4: Setup mkcert CA
	fmt.Println(cli.Header("Step 4: SSL Certificate Authority"))

	verbose.Debug("Setting up mkcert CA...")
	sslMgr := ssl.NewManager(p)
	if sslMgr.IsCAInstalled() {
		verbose.Debug("Local CA already installed")
		fmt.Println("  Local CA already installed " + cli.Success("✓"))
	} else {
		fmt.Print("  Installing local CA... ")
		if err := sslMgr.InstallCA(); err != nil {
			verbose.Debug("SSL CA installation error: %v", err)
			fmt.Println(cli.Error("failed"))
			cli.PrintWarning("SSL setup failed: %v", err)
		} else {
			verbose.Debug("Local CA installed successfully")
			fmt.Println(cli.Success("done"))
		}
	}
	fmt.Println()

	// Step 5: Setup Port Forwarding (macOS only)
	if p.Type == platform.Darwin {
		fmt.Println(cli.Header("Step 5: Port Forwarding Setup"))
		fmt.Println("  Setting up transparent port forwarding (80→8080, 443→8443)")
		fmt.Println("  This allows Nginx to run as your user without sudo")
		fmt.Println()

		pfMgr := portforward.NewManager()
		// Always call Setup() - it will verify rules are active and reload if needed
		fmt.Print("  Configuring pf rules and LaunchDaemon... ")
		if err := pfMgr.Setup(); err != nil {
			fmt.Println(cli.Error("failed"))
			cli.PrintWarning("Port forwarding setup failed: %v", err)
			cli.PrintWarning("You may need to manually run: sudo pfctl -ef /etc/pf.conf")
			errors = append(errors, "Port forwarding setup failed")
		} else {
			fmt.Println(cli.Success("done"))
		}
		fmt.Println()
	}

	// Step 6: Setup Nginx configuration
	fmt.Println(cli.Header("Step 6: Nginx Configuration"))

	nginxCtrl := nginx.NewController(p)
	fmt.Printf("  Nginx config: %s\n", cli.Highlight(nginxCtrl.GetNginxConfPath()))

	// Configure nginx for this platform
	verbose.Debug("Configuring nginx for platform %s...", p.Type)
	if err := bootstrapper.ConfigureNginx(); err != nil {
		verbose.Debug("Nginx configuration error: %v", err)
		cli.PrintWarning("Nginx configuration failed: %v", err)
	}

	// Create vhosts directory
	vhostsDir := filepath.Join(p.MageBoxDir(), "nginx", "vhosts")
	if err := os.MkdirAll(vhostsDir, 0755); err != nil {
		cli.PrintWarning("Failed to create vhosts dir: %v", err)
	} else {
		fmt.Printf("  Vhosts dir: %s\n", cli.Highlight(vhostsDir))
	}

	// Generate Mailpit vhost BEFORE nginx test (so vhosts dir isn't empty)
	tld := globalCfg.GetTLD()
	mailpitDomain := fmt.Sprintf("mailpit.magebox.%s", tld)
	fmt.Print("  Creating Mailpit vhost... ")
	vhostGen := nginx.NewVhostGenerator(p, sslMgr)
	mailpitCfg := nginx.ProxyConfig{
		Name:       "mailpit",
		Domain:     mailpitDomain,
		ProxyHost:  "127.0.0.1",
		ProxyPort:  8025,
		SSLEnabled: true,
	}
	if err := vhostGen.GenerateProxyVhost(mailpitCfg); err != nil {
		fmt.Println(cli.Error("failed"))
		cli.PrintWarning("Mailpit vhost generation failed: %v", err)
	} else {
		fmt.Println(cli.Success("done"))
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

	// Configure SELinux for nginx (Fedora only)
	if p.Type == platform.Linux && p.LinuxDistro == platform.DistroFedora {
		fmt.Print("  Configuring SELinux for nginx... ")
		if err := bootstrapper.ConfigureSELinux(); err != nil {
			fmt.Println(cli.Warning("skipped"))
		} else {
			fmt.Println(cli.Success("done"))
		}
	}

	// Test and reload nginx
	fmt.Print("  Testing nginx config... ")
	if err := nginxCtrl.Test(); err != nil {
		fmt.Println(cli.Error("failed"))
	} else {
		fmt.Println(cli.Success("ok"))
		fmt.Print("  Restarting nginx... ")
		// Always restart (not reload) to pick up new listen ports
		if err := nginxCtrl.Restart(); err != nil {
			fmt.Println(cli.Error("failed"))
		} else {
			fmt.Println(cli.Success("done"))
		}
	}

	// Enable nginx to start on boot (Linux only)
	if p.Type == platform.Linux {
		fmt.Print("  Enabling nginx on boot... ")
		cmd := exec.Command("sudo", "systemctl", "enable", "nginx")
		if err := cmd.Run(); err != nil {
			fmt.Println(cli.Error("failed"))
		} else {
			fmt.Println(cli.Success("done"))
		}
	}
	fmt.Println()

	// Step 7: Setup Docker services
	fmt.Println(cli.Header("Step 7: Docker Services"))

	verbose.Debug("Setting up Docker services...")
	composeGen := docker.NewComposeGenerator(p)
	composeFile := composeGen.ComposeFilePath()
	verbose.Debug("Compose file path: %s", composeFile)

	fmt.Print("  Generating docker-compose.yml... ")
	if err := composeGen.GenerateDefaultServices(globalCfg); err != nil {
		verbose.Debug("Docker compose generation error: %v", err)
		fmt.Println(cli.Error("failed"))
		cli.PrintWarning("Docker compose generation failed: %v", err)
	} else {
		verbose.Debug("Docker compose file generated successfully")
		fmt.Println(cli.Success("done"))
		fmt.Printf("    %s\n", cli.Highlight(composeFile))
	}

	// Start Docker services
	verbose.Debug("Starting Docker containers...")
	fmt.Print("  Starting containers... ")
	dockerCtrl := docker.NewDockerController(composeFile)
	if err := dockerCtrl.Up(); err != nil {
		verbose.Debug("Docker up error: %v", err)
		fmt.Println(cli.Error("failed"))
		cli.PrintWarning("Docker services failed to start: %v", err)
	} else {
		verbose.Debug("Docker containers started successfully")
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

	// Step 8: DNS setup
	fmt.Println(cli.Header("Step 8: DNS Configuration"))

	dnsManager := dns.NewDnsmasqManager(p)

	// Auto-configure dnsmasq for wildcard DNS on all platforms
	dnsmasqConfigured := false

	if dnsManager.IsConfigured() && dnsManager.IsRunning() {
		fmt.Printf("  dnsmasq configured for *.%s %s\n", tld, cli.Success("✓"))
		dnsmasqConfigured = true
	} else {
		// Check if dnsmasq is installed
		if !dnsManager.IsInstalled() {
			fmt.Print("  Installing dnsmasq... ")
			if err := bootstrapper.GetInstaller().InstallDnsmasq(); err != nil {
				fmt.Println(cli.Error("failed"))
				cli.PrintWarning("dnsmasq installation failed: %v", err)
			} else {
				fmt.Println(cli.Success("done"))
			}
		}

		// Configure dnsmasq if now installed
		if dnsManager.IsInstalled() {
			fmt.Printf("  Configuring dnsmasq for *.%s domains... ", tld)
			if err := dnsManager.Configure(); err != nil {
				fmt.Println(cli.Error("failed"))
				cli.PrintWarning("dnsmasq config failed: %v", err)
			} else {
				fmt.Println(cli.Success("done"))

				// Start dnsmasq service
				fmt.Print("  Starting dnsmasq service... ")
				if err := dnsManager.Start(); err != nil {
					fmt.Println(cli.Error("failed"))
					cli.PrintWarning("dnsmasq start failed: %v", err)
				} else {
					fmt.Println(cli.Success("done"))
					dnsmasqConfigured = true
				}
			}
		}
	}

	// Set dns_mode to dnsmasq if successfully configured
	if dnsmasqConfigured && globalCfg.DNSMode != "dnsmasq" {
		globalCfg.DNSMode = "dnsmasq"
		if err := config.SaveGlobalConfig(homeDir, globalCfg); err != nil {
			cli.PrintWarning("Failed to save dns_mode config: %v", err)
		} else {
			fmt.Println("  Set dns_mode: dnsmasq " + cli.Success("✓"))
		}
	} else if !dnsmasqConfigured {
		// Fall back to hosts mode if dnsmasq setup failed
		fmt.Println("  Falling back to /etc/hosts mode")
		globalCfg.DNSMode = "hosts"
		_ = config.SaveGlobalConfig(homeDir, globalCfg)
		cli.PrintInfo("Domains will be added to /etc/hosts when you run %s", cli.Command("magebox start"))
	}

	// Test DNS resolution if dnsmasq was configured
	if dnsmasqConfigured {
		testDomain := fmt.Sprintf("test.%s", tld)
		fmt.Printf("  Testing DNS resolution for %s... ", testDomain)
		if dnsManager.TestResolution(testDomain) {
			fmt.Println(cli.Success("✓ resolves to 127.0.0.1"))
		} else {
			fmt.Println(cli.Warning("not resolving yet"))
			cli.PrintInfo("DNS may need a moment to propagate. Test with: dig +short %s @127.0.0.1", testDomain)
		}
	}
	fmt.Println()

	// Step 9: Install PHP, Composer, and Blackfire wrappers
	fmt.Println(cli.Header("Step 9: CLI Wrappers (PHP, Composer, Blackfire)"))

	wrapperMgr := phpwrapper.NewManager(p)

	// PHP wrapper
	if wrapperMgr.IsInstalled() {
		fmt.Println("  PHP wrapper already installed " + cli.Success("✓"))
	} else {
		fmt.Print("  Installing PHP wrapper script... ")
		if err := wrapperMgr.Install(); err != nil {
			fmt.Println(cli.Error("failed"))
			cli.PrintWarning("PHP wrapper installation failed: %v", err)
		} else {
			fmt.Println(cli.Success("done"))
		}
	}

	// Composer wrapper
	if wrapperMgr.IsComposerInstalled() {
		fmt.Println("  Composer wrapper already installed " + cli.Success("✓"))
	} else {
		fmt.Print("  Installing Composer wrapper script... ")
		if err := wrapperMgr.InstallComposer(); err != nil {
			fmt.Println(cli.Error("failed"))
			cli.PrintWarning("Composer wrapper installation failed: %v", err)
		} else {
			fmt.Println(cli.Success("done"))
		}
	}

	// Blackfire wrapper (uses project PHP for 'blackfire run' commands)
	if wrapperMgr.IsBlackfireInstalled() {
		fmt.Println("  Blackfire wrapper already installed " + cli.Success("✓"))
	} else {
		fmt.Print("  Installing Blackfire wrapper script... ")
		if err := wrapperMgr.InstallBlackfire(); err != nil {
			fmt.Println(cli.Error("failed"))
			cli.PrintWarning("Blackfire wrapper installation failed: %v", err)
		} else {
			fmt.Println(cli.Success("done"))
		}
	}

	// Configure shell PATH (Linux and macOS)
	fmt.Print("  Adding ~/.magebox/bin to shell PATH... ")
	if err := bootstrapper.GetInstaller().ConfigureShellPath(); err != nil {
		fmt.Println(cli.Warning("skipped"))
		cli.PrintInfo("Add manually: export PATH=\"$HOME/.magebox/bin:$PATH\"")
	} else {
		fmt.Println(cli.Success("done"))
	}

	// Multitail for log viewing
	if platform.CommandExists("multitail") {
		fmt.Println("  multitail already installed " + cli.Success("✓"))
	} else {
		fmt.Print("  Installing multitail for log viewing... ")
		if err := bootstrapper.GetInstaller().InstallMultitail(); err != nil {
			fmt.Println(cli.Error("failed"))
			cli.PrintWarning("multitail installation failed: %v", err)
		} else {
			fmt.Println(cli.Success("done"))
		}
	}
	fmt.Println()

	// Step 10: Setup sudoers for passwordless nginx/php-fpm control (Linux only)
	if p.Type == platform.Linux {
		fmt.Println(cli.Header("Step 10: Sudoers Configuration"))

		sudoersFile := "/etc/sudoers.d/magebox"
		if _, err := os.Stat(sudoersFile); err == nil {
			fmt.Println("  Sudoers already configured " + cli.Success("✓"))
		} else {
			fmt.Print("  Setting up passwordless nginx/php-fpm control... ")
			if err := bootstrapper.ConfigureSudoers(); err != nil {
				fmt.Println(cli.Error("failed"))
				cli.PrintWarning("Failed to setup sudoers: %v", err)
			} else {
				fmt.Println(cli.Success("done"))
			}
		}
		fmt.Println()
	}

	// Docker provider detection (macOS only)
	if p.Type == platform.Darwin {
		providerMgr := docker.NewProviderManager()
		runningProviders := providerMgr.GetRunningProviders()

		if len(runningProviders) > 1 {
			fmt.Println(cli.Header("Docker Providers"))
			cli.PrintWarning("Multiple Docker providers detected:")
			for _, prov := range runningProviders {
				marker := "  "
				if prov.IsActive {
					marker = "● "
				} else {
					marker = "○ "
				}
				fmt.Printf("  %s%s (%s)\n", marker, prov.Name, prov.SocketPath)
			}
			fmt.Println()
			activeProvider := providerMgr.GetActiveProvider()
			if activeProvider != nil {
				fmt.Printf("  Active: %s\n", cli.Highlight(activeProvider.Name))
			}
			fmt.Println()
			cli.PrintInfo("Run %s to view or switch providers", cli.Command("magebox docker"))
			fmt.Println()
		}
	}

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
	fmt.Printf("  Mailpit:      %s\n", cli.URL(fmt.Sprintf("https://mailpit.magebox.%s", tld)))
	if globalCfg.Portainer {
		fmt.Printf("  Portainer:    %s\n", cli.URL("http://localhost:9000"))
	}
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println(cli.Bullet("Reload your shell to activate the PHP wrapper:"))
	fmt.Println()
	shellName := filepath.Base(os.Getenv("SHELL"))
	if shellName == "" {
		shellName = "bash"
	}
	fmt.Printf("    source ~/.%src\n", shellName)
	fmt.Println()
	fmt.Println(cli.Bullet("cd into your Magento project directory"))
	fmt.Println(cli.Bullet("Run " + cli.Command("magebox init") + " to create .magebox.yaml config"))
	fmt.Println(cli.Bullet("Run " + cli.Command("magebox start") + " to start your project"))
	fmt.Println()

	return nil
}

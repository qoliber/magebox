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
)

var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Initialize MageBox environment",
	Long: `Sets up the MageBox development environment for first-time use.

This command performs the following steps:
  1. Checks all required dependencies (Docker, Nginx, mkcert)
  2. Installs PHP versions 8.1-8.4 via Homebrew (macOS)
  3. Initializes global configuration (~/.magebox/config.yaml)
  4. Sets up mkcert CA for HTTPS support
  5. Configures port forwarding (macOS: 80→8080, 443→8443)
  6. Configures Nginx to include MageBox vhosts
  7. Creates and starts Docker services (MySQL, Redis, Mailpit)
  8. Sets up DNS resolution (dnsmasq or /etc/hosts)
  9. Installs PHP and Composer wrappers for automatic version switching

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
	installedPHPVersions := detector.DetectInstalled()
	fmt.Printf("  %-15s %s (%d versions)\n", "PHP:", cli.StatusInstalled(len(installedPHPVersions) > 0), len(installedPHPVersions))

	fmt.Println()

	// If critical dependencies are missing, offer to install them
	if !dockerInstalled || !nginxInstalled || !mkcertInstalled {
		cli.PrintWarning("Missing dependencies detected!")
		fmt.Println()

		// Show what's missing
		var missingDeps []string
		var installCmds []string

		if !nginxInstalled {
			missingDeps = append(missingDeps, "Nginx")
			installCmds = append(installCmds, p.NginxInstallCommand())
		}
		if !mkcertInstalled {
			missingDeps = append(missingDeps, "mkcert")
			installCmds = append(installCmds, p.MkcertInstallCommand())
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
			for i, cmd := range installCmds {
				fmt.Printf("  Installing %s...\n", missingDeps[i])
				fmt.Printf("  Running: %s\n", cli.Command(cmd))

				// Parse and execute the command
				if err := runInstallCommand(cmd); err != nil {
					cli.PrintError("Failed to install %s: %v", missingDeps[i], err)
					fmt.Println()
					cli.PrintInfo("Please install manually and run " + cli.Command("magebox bootstrap") + " again")
					return nil
				}
				fmt.Printf("  %s installed %s\n", missingDeps[i], cli.Success("✓"))
				fmt.Println()
			}

			// Re-check after installation
			nginxInstalled = p.IsNginxInstalled()
			mkcertInstalled = platform.CommandExists("mkcert")
		} else {
			fmt.Println()
			cli.PrintInfo("Install missing dependencies and run " + cli.Command("magebox bootstrap") + " again")
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
		fmt.Printf("  Install Docker: %s\n", cli.Command(p.DockerInstallCommand()))
		fmt.Println()
		cli.PrintInfo("After installing Docker, run " + cli.Command("magebox bootstrap") + " again")
		return nil
	}

	// Step 2: Install PHP versions
	fmt.Println(cli.Header("Step 2: PHP Installation"))
	fmt.Println()

	// All PHP versions we want for Magento compatibility (8.5 is bleeding edge/dev)
	allPHPVersions := []string{"8.1", "8.2", "8.3", "8.4", "8.5"}

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
			fmt.Printf("    %s PHP %s\n", cli.Success(""), ver.Version)
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
			for _, phpVer := range missingVersions {
				fmt.Printf("  Installing PHP %s...\n", phpVer)
				installCmd := p.PHPInstallCommand(phpVer)
				fmt.Printf("  Running: %s\n", cli.Command(installCmd))

				if err := runInstallCommand(installCmd); err != nil {
					cli.PrintWarning("Failed to install PHP %s: %v", phpVer, err)
				} else {
					fmt.Printf("  PHP %s installed %s\n", phpVer, cli.Success(""))
				}
				fmt.Println()
			}
			// Refresh detected versions
			installedPHPVersions = detector.DetectInstalled()
		}
	} else {
		fmt.Println("  All recommended PHP versions are installed " + cli.Success(""))
	}

	// On Linux (Fedora/Remi), configure PHP-FPM logs to /var/log/magebox
	if p.Type == platform.Linux && p.LinuxDistro == platform.DistroFedora {
		fmt.Println()
		fmt.Print("  Setting up PHP-FPM services... ")

		// Create /var/log/magebox for PHP-FPM logs
		_ = exec.Command("sudo", "mkdir", "-p", "/var/log/magebox").Run()
		_ = exec.Command("sudo", "chmod", "755", "/var/log/magebox").Run()

		// Configure each PHP version to use /var/log/magebox
		phpVersions := []string{"81", "82", "83", "84", "85"}
		for _, v := range phpVersions {
			fpmConf := fmt.Sprintf("/etc/opt/remi/php%s/php-fpm.conf", v)
			logFile := fmt.Sprintf("/var/log/magebox/php%s-fpm.log", v)

			// Update error_log path in php-fpm.conf
			_ = exec.Command("sudo", "sed", "-i", fmt.Sprintf("s|^error_log = .*|error_log = %s|", logFile), fpmConf).Run()
		}

		// Enable and start PHP-FPM services for installed versions
		for _, ver := range installedPHPVersions {
			remiVer := strings.ReplaceAll(ver.Version, ".", "")
			serviceName := fmt.Sprintf("php%s-php-fpm", remiVer)
			_ = exec.Command("sudo", "systemctl", "enable", serviceName).Run()
			_ = exec.Command("sudo", "systemctl", "restart", serviceName).Run()
		}
		fmt.Println(cli.Success("done"))
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

	// Step 5: Setup Port Forwarding (macOS only)
	if p.Type == platform.Darwin {
		fmt.Println(cli.Header("Step 5: Port Forwarding Setup"))
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

	// Step 6: Setup Nginx configuration
	fmt.Println(cli.Header("Step 6: Nginx Configuration"))

	nginxCtrl := nginx.NewController(p)
	fmt.Printf("  Nginx config: %s\n", cli.Highlight(nginxCtrl.GetNginxConfPath()))

	// On Linux, configure nginx to run as current user (so it can access ~/.magebox/certs)
	if p.Type == platform.Linux {
		currentUser := os.Getenv("USER")
		if currentUser == "" {
			currentUser = os.Getenv("LOGNAME")
		}
		if currentUser != "" {
			fmt.Print("  Configuring nginx to run as " + cli.Highlight(currentUser) + "... ")
			nginxConf := nginxCtrl.GetNginxConfPath()
			cmd := exec.Command("sudo", "sed", "-i", fmt.Sprintf("s/^user .*/user %s;/", currentUser), nginxConf)
			if err := cmd.Run(); err != nil {
				fmt.Println(cli.Error("failed"))
			} else {
				fmt.Println(cli.Success("done"))
			}
		}
	}

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

	// Generate Mailpit vhost (mailpit.magebox.test)
	fmt.Print("  Setting up Mailpit vhost... ")
	vhostGen := nginx.NewVhostGenerator(p, sslMgr)
	mailpitCfg := nginx.ProxyConfig{
		Name:       "mailpit",
		Domain:     "mailpit.magebox.test",
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
	fmt.Println()

	// Step 8: DNS setup - auto-configure dnsmasq on Linux
	fmt.Println(cli.Header("Step 8: DNS Configuration"))

	dnsManager := dns.NewDnsmasqManager(p)

	if p.Type == platform.Linux {
		// On Linux, auto-configure dnsmasq for *.test wildcard DNS
		if dnsManager.IsConfigured() && dnsManager.IsRunning() {
			fmt.Println("  dnsmasq configured and running " + cli.Success(""))
		} else {
			if dnsManager.IsInstalled() {
				fmt.Print("  Configuring dnsmasq for *.test domains... ")
				if err := dnsManager.Configure(); err != nil {
					fmt.Println(cli.Error("failed"))
					cli.PrintWarning("dnsmasq config failed: %v", err)
				} else {
					fmt.Println(cli.Success("done"))
				}

				fmt.Print("  Starting dnsmasq... ")
				if err := dnsManager.Start(); err != nil {
					// Try restart if already running
					if err := dnsManager.Restart(); err != nil {
						fmt.Println(cli.Error("failed"))
					} else {
						fmt.Println(cli.Success("restarted"))
					}
				} else {
					fmt.Println(cli.Success("started"))
				}

				fmt.Print("  Enabling dnsmasq on boot... ")
				if err := dnsManager.Enable(); err != nil {
					fmt.Println(cli.Error("failed"))
				} else {
					fmt.Println(cli.Success("done"))
				}
			} else {
				fmt.Println("  dnsmasq not installed")
				cli.PrintInfo("Install with: " + cli.Command(dnsManager.InstallCommand()))
			}
		}
	} else if globalCfg.UseDnsmasq() {
		// macOS - check if dnsmasq configured
		if dnsManager.IsConfigured() {
			fmt.Println("  dnsmasq configured for *.test " + cli.Success(""))
		} else {
			fmt.Println("  dnsmasq not yet configured")
			cli.PrintInfo("Run " + cli.Command("magebox dns setup") + " to configure wildcard DNS")
		}
	} else {
		fmt.Println("  Using /etc/hosts mode")
		cli.PrintInfo("Domains will be added to /etc/hosts when you run " + cli.Command("magebox start"))
	}
	fmt.Println()

	// Step 9: Install PHP and Composer wrappers
	fmt.Println(cli.Header("Step 9: PHP & Composer Wrappers"))

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
	fmt.Println()

	// Step 10: Setup sudoers for passwordless nginx/php-fpm control (Linux only)
	if p.Type == platform.Linux {
		fmt.Println(cli.Header("Step 10: Sudoers Configuration"))

		sudoersFile := "/etc/sudoers.d/magebox"
		if _, err := os.Stat(sudoersFile); err == nil {
			fmt.Println("  Sudoers already configured " + cli.Success(""))
		} else {
			fmt.Print("  Setting up passwordless nginx/php-fpm control... ")

			// Get current user
			currentUser := os.Getenv("USER")
			if currentUser == "" {
				currentUser = os.Getenv("LOGNAME")
			}

			// Create sudoers content for nginx and php-fpm control
			sudoersContent := fmt.Sprintf(`# MageBox - Allow %[1]s to control nginx and php-fpm without password
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl start nginx
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl stop nginx
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl reload nginx
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart nginx
%[1]s ALL=(ALL) NOPASSWD: /usr/sbin/nginx -s reload
%[1]s ALL=(ALL) NOPASSWD: /usr/sbin/nginx -t
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/nginx -s reload
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/nginx -t
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/nginx
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl start php*-php-fpm
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl stop php*-php-fpm
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl reload php*-php-fpm
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart php*-php-fpm
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl start php*-fpm
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl stop php*-fpm
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl reload php*-fpm
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart php*-fpm
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/cp /tmp/magebox-* /etc/nginx/nginx.conf
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/mkdir -p /etc/nginx/*
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/rm /etc/nginx/*
%[1]s ALL=(ALL) NOPASSWD: /usr/bin/ln -s *
`, currentUser)

			// Write to temp file
			tmpFile, err := os.CreateTemp("", "magebox-sudoers-*")
			if err != nil {
				fmt.Println(cli.Error("failed"))
				cli.PrintWarning("Failed to create temp file: %v", err)
			} else {
				tmpPath := tmpFile.Name()
				_, _ = tmpFile.WriteString(sudoersContent)
				_ = tmpFile.Close()

				// Set correct permissions (sudoers files must be 0440)
				_ = os.Chmod(tmpPath, 0440)

				// Copy to sudoers.d with sudo
				cmd := exec.Command("sudo", "cp", tmpPath, sudoersFile)
				cmd.Stdin = os.Stdin
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				if err := cmd.Run(); err != nil {
					fmt.Println(cli.Error("failed"))
					cli.PrintWarning("Failed to setup sudoers: %v", err)
				} else {
					// Set correct ownership
					_ = exec.Command("sudo", "chmod", "0440", sudoersFile).Run()
					fmt.Println(cli.Success("done"))
				}
				_ = os.Remove(tmpPath)
			}
		}
		fmt.Println()
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
	fmt.Printf("  Mailpit:      %s\n", cli.URL("https://mailpit.magebox.test"))
	if globalCfg.Portainer {
		fmt.Printf("  Portainer:    %s\n", cli.URL("http://localhost:9000"))
	}
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println(cli.Bullet("Add MageBox bin to your PATH (recommended):"))
	fmt.Println()
	fmt.Println(wrapperMgr.GetInstructions())
	fmt.Println()
	fmt.Println(cli.Bullet("cd into your Magento project directory"))
	fmt.Println(cli.Bullet("Run " + cli.Command("magebox init") + " to create .magebox.yaml config"))
	fmt.Println(cli.Bullet("Run " + cli.Command("magebox start") + " to start your project"))
	fmt.Println()

	return nil
}

// runInstallCommand runs an install command with proper shell handling
func runInstallCommand(cmdStr string) error {
	// Use shell to handle complex commands (pipes, &&, etc.)
	cmd := exec.Command("sh", "-c", cmdStr)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

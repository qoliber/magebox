package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/qoliber/magebox/internal/cli"
	"github.com/qoliber/magebox/internal/dns"
	"github.com/qoliber/magebox/internal/docker"
	"github.com/qoliber/magebox/internal/phpwrapper"
	"github.com/qoliber/magebox/internal/platform"
	"github.com/qoliber/magebox/internal/portforward"
	"github.com/qoliber/magebox/internal/project"
)

var (
	uninstallForce      bool
	uninstallKeepVhosts bool
	uninstallDryRun     bool
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall MageBox components",
	Long: `Stops all projects and removes MageBox components:
  - Stops all running MageBox projects
  - Stops and removes MageBox Docker containers
  - Removes CLI wrappers (php, composer, blackfire)
  - Removes nginx vhosts (unless --keep-vhosts)
  - Removes port forwarding rules (macOS)
  - Stops and disables dnsmasq

Note: This does not uninstall system packages (PHP, nginx, etc.)`,
	RunE: runUninstall,
}

func init() {
	uninstallCmd.Flags().BoolVarP(&uninstallForce, "force", "f", false, "Skip confirmation prompt")
	uninstallCmd.Flags().BoolVar(&uninstallKeepVhosts, "keep-vhosts", false, "Keep nginx vhost configurations")
	uninstallCmd.Flags().BoolVar(&uninstallDryRun, "dry-run", false, "Show what would be removed without removing")
	rootCmd.AddCommand(uninstallCmd)
}

func runUninstall(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	if uninstallDryRun {
		return uninstallDryRunFunc(p)
	}

	cli.PrintTitle("MageBox Uninstall")
	fmt.Println()

	// Show what will be removed
	fmt.Println("This will:")
	fmt.Println(cli.Bullet("Stop all MageBox projects"))
	fmt.Println(cli.Bullet("Stop and remove MageBox Docker containers"))
	fmt.Println(cli.Bullet("Remove CLI wrappers (php, composer, blackfire) from ~/.magebox/bin/"))
	if !uninstallKeepVhosts {
		fmt.Println(cli.Bullet("Remove nginx vhost configurations from ~/.magebox/nginx/"))
	}
	if p.Type == platform.Darwin {
		fmt.Println(cli.Bullet("Remove port forwarding rules (pf)"))
	}
	fmt.Println(cli.Bullet("Stop and disable dnsmasq"))
	fmt.Println()

	fmt.Println(cli.Subtitle("This will NOT remove:"))
	fmt.Println(cli.Bullet("System packages (PHP, nginx, dnsmasq binaries, etc.)"))
	fmt.Println(cli.Bullet("Docker images"))
	fmt.Println(cli.Bullet("Project files or .magebox.yaml configs"))
	fmt.Println(cli.Bullet("SSL certificates"))
	fmt.Println()

	// Confirm unless --force
	if !uninstallForce {
		fmt.Print("Continue? [y/N] ")
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			cli.PrintInfo("Aborted")
			return nil
		}
		fmt.Println()
	}

	// Step 1: Stop all projects
	fmt.Println(cli.Header("Step 1: Stopping all projects"))

	discovery := project.NewProjectDiscovery(p)
	projects, err := discovery.DiscoverProjects()
	if err != nil {
		cli.PrintWarning("Failed to discover projects: %v", err)
	} else {
		mgr := project.NewManager(p)
		stopped := 0
		for _, proj := range projects {
			if !proj.HasConfig {
				continue
			}
			fmt.Printf("  Stopping %s... ", proj.Name)
			if err := mgr.Stop(proj.Path); err != nil {
				fmt.Println(cli.Warning("skipped"))
			} else {
				fmt.Println(cli.Success("done"))
				stopped++
			}
		}
		if stopped == 0 {
			fmt.Println("  No projects to stop")
		}
	}
	fmt.Println()

	// Step 2: Stop and remove Docker containers
	fmt.Println(cli.Header("Step 2: Stopping Docker containers"))

	composeFile := filepath.Join(p.MageBoxDir(), "docker", "docker-compose.yml")
	if _, err := os.Stat(composeFile); err == nil {
		dockerCtrl := docker.NewDockerController(composeFile)
		fmt.Print("  Stopping MageBox containers... ")
		if err := dockerCtrl.Down(); err != nil {
			fmt.Println(cli.Warning("failed"))
			cli.PrintWarning("    %v", err)
		} else {
			fmt.Println(cli.Success("done"))
		}
	} else {
		fmt.Println("  No docker-compose.yml found")
	}
	fmt.Println()

	// Step 3: Remove CLI wrappers
	fmt.Println(cli.Header("Step 3: Removing CLI wrappers"))

	wrapperMgr := phpwrapper.NewManager(p)

	// PHP wrapper
	if wrapperMgr.IsInstalled() {
		fmt.Print("  Removing php wrapper... ")
		if err := wrapperMgr.Uninstall(); err != nil {
			fmt.Println(cli.Error("failed"))
			cli.PrintWarning("    %v", err)
		} else {
			fmt.Println(cli.Success("done"))
		}
	} else {
		fmt.Println("  PHP wrapper not installed")
	}

	// Composer wrapper
	if wrapperMgr.IsComposerInstalled() {
		fmt.Print("  Removing composer wrapper... ")
		if err := wrapperMgr.UninstallComposer(); err != nil {
			fmt.Println(cli.Error("failed"))
			cli.PrintWarning("    %v", err)
		} else {
			fmt.Println(cli.Success("done"))
		}
	} else {
		fmt.Println("  Composer wrapper not installed")
	}

	// Blackfire wrapper
	if wrapperMgr.IsBlackfireInstalled() {
		fmt.Print("  Removing blackfire wrapper... ")
		if err := wrapperMgr.UninstallBlackfire(); err != nil {
			fmt.Println(cli.Error("failed"))
			cli.PrintWarning("    %v", err)
		} else {
			fmt.Println(cli.Success("done"))
		}
	} else {
		fmt.Println("  Blackfire wrapper not installed")
	}
	fmt.Println()

	// Step 4: Remove nginx vhosts (unless --keep-vhosts)
	if !uninstallKeepVhosts {
		fmt.Println(cli.Header("Step 4: Removing nginx vhosts"))

		vhostsDir := p.MageBoxDir() + "/nginx/vhosts"
		if _, err := os.Stat(vhostsDir); err == nil {
			entries, err := os.ReadDir(vhostsDir)
			if err != nil {
				cli.PrintWarning("Failed to read vhosts directory: %v", err)
			} else {
				removed := 0
				for _, entry := range entries {
					if strings.HasSuffix(entry.Name(), ".conf") {
						path := vhostsDir + "/" + entry.Name()
						if err := os.Remove(path); err != nil {
							cli.PrintWarning("  Failed to remove %s: %v", entry.Name(), err)
						} else {
							removed++
						}
					}
				}
				if removed > 0 {
					fmt.Printf("  Removed %d vhost file(s)\n", removed)
				} else {
					fmt.Println("  No vhost files to remove")
				}
			}
		} else {
			fmt.Println("  No vhosts directory found")
		}
		fmt.Println()
	}

	// Step 5: Remove port forwarding (macOS only)
	if p.Type == platform.Darwin {
		fmt.Println(cli.Header("Step 5: Removing port forwarding"))

		pfMgr := portforward.NewManager()
		if pfMgr.IsInstalled() {
			fmt.Print("  Removing pf rules and LaunchDaemon... ")
			if err := pfMgr.Remove(); err != nil {
				fmt.Println(cli.Warning("failed"))
				cli.PrintWarning("    %v", err)
			} else {
				fmt.Println(cli.Success("done"))
			}
		} else {
			fmt.Println("  Port forwarding not configured")
		}
		fmt.Println()
	}

	// Step 6: Stop and disable dnsmasq
	fmt.Println(cli.Header("Step 6: Stopping dnsmasq"))

	dnsManager := dns.NewDnsmasqManager(p)
	if dnsManager.IsInstalled() {
		if dnsManager.IsRunning() {
			fmt.Print("  Stopping dnsmasq service... ")
			if err := dnsManager.Stop(); err != nil {
				fmt.Println(cli.Warning("failed"))
				cli.PrintWarning("    %v", err)
			} else {
				fmt.Println(cli.Success("done"))
			}
		} else {
			fmt.Println("  dnsmasq not running")
		}

		// Remove MageBox dnsmasq config
		if dnsManager.IsConfigured() {
			fmt.Print("  Removing dnsmasq configuration... ")
			if err := dnsManager.Remove(); err != nil {
				fmt.Println(cli.Warning("failed"))
				cli.PrintWarning("    %v", err)
			} else {
				fmt.Println(cli.Success("done"))
			}
		}
	} else {
		fmt.Println("  dnsmasq not installed")
	}
	fmt.Println()

	cli.PrintSuccess("MageBox components removed")
	fmt.Println()
	fmt.Println("To completely remove MageBox:")
	fmt.Println(cli.Bullet("Remove the magebox binary from your PATH"))
	fmt.Println(cli.Bullet("Remove ~/.magebox directory: " + cli.Command("rm -rf ~/.magebox")))
	fmt.Println(cli.Bullet("Remove PATH entry from ~/.zshrc or ~/.bashrc"))

	return nil
}

func uninstallDryRunFunc(p interface {
	MageBoxDir() string
}) error {
	cli.PrintTitle("Dry Run: Would uninstall")
	fmt.Println()

	// Check projects
	plat, _ := getPlatform()
	discovery := project.NewProjectDiscovery(plat)
	projects, _ := discovery.DiscoverProjects()

	fmt.Println(cli.Header("Projects that would be stopped"))
	if len(projects) == 0 {
		fmt.Println("  No projects found")
	} else {
		for _, proj := range projects {
			if proj.HasConfig {
				fmt.Printf("  %s %s (%s)\n", cli.Bullet(""), proj.Name, cli.Path(proj.Path))
			}
		}
	}
	fmt.Println()

	// Check wrappers
	fmt.Println(cli.Header("CLI wrappers that would be removed"))
	wrapperMgr := phpwrapper.NewManager(plat)

	if wrapperMgr.IsInstalled() {
		fmt.Printf("  %s %s\n", cli.Bullet(""), wrapperMgr.GetWrapperPath())
	}
	if wrapperMgr.IsComposerInstalled() {
		fmt.Printf("  %s %s\n", cli.Bullet(""), wrapperMgr.GetComposerWrapperPath())
	}
	if wrapperMgr.IsBlackfireInstalled() {
		fmt.Printf("  %s %s\n", cli.Bullet(""), wrapperMgr.GetBlackfireWrapperPath())
	}
	if !wrapperMgr.IsInstalled() && !wrapperMgr.IsComposerInstalled() && !wrapperMgr.IsBlackfireInstalled() {
		fmt.Println("  No wrappers installed")
	}
	fmt.Println()

	// Check vhosts
	if !uninstallKeepVhosts {
		fmt.Println(cli.Header("Vhost files that would be removed"))
		vhostsDir := p.MageBoxDir() + "/nginx/vhosts"
		if entries, err := os.ReadDir(vhostsDir); err == nil {
			count := 0
			for _, entry := range entries {
				if strings.HasSuffix(entry.Name(), ".conf") {
					fmt.Printf("  %s %s\n", cli.Bullet(""), entry.Name())
					count++
				}
			}
			if count == 0 {
				fmt.Println("  No vhost files found")
			}
		} else {
			fmt.Println("  No vhosts directory found")
		}
		fmt.Println()
	}

	cli.PrintInfo("Run without --dry-run to actually uninstall")
	return nil
}

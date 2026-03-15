package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"qoliber/magebox/internal/cli"
	"qoliber/magebox/internal/config"
	"qoliber/magebox/internal/docker"
	"qoliber/magebox/internal/nginx"
	"qoliber/magebox/internal/ssl"
	"qoliber/magebox/internal/varnish"
)

var varnishCmd = &cobra.Command{
	Use:   "varnish",
	Short: "Varnish cache management",
	Long:  "Manage Varnish full-page cache",
}

var varnishPurgeCmd = &cobra.Command{
	Use:   "purge [url]",
	Short: "Purge a URL from cache",
	Long:  "Purges a specific URL from Varnish cache",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runVarnishPurge,
}

var varnishFlushCmd = &cobra.Command{
	Use:   "flush",
	Short: "Flush all cache",
	Long:  "Flushes all content from Varnish cache",
	RunE:  runVarnishFlush,
}

var varnishStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Varnish status",
	Long:  "Shows Varnish cache statistics and status",
	RunE:  runVarnishStatus,
}

var varnishEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable Varnish for project",
	Long:  "Enables Varnish full-page cache for the current project",
	RunE:  runVarnishEnable,
}

var varnishDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable Varnish for project",
	Long:  "Disables Varnish full-page cache for the current project",
	RunE:  runVarnishDisable,
}

var varnishVCLImportCmd = &cobra.Command{
	Use:   "vcl-import <file>",
	Short: "Import custom VCL file",
	Long:  "Imports a custom Varnish VCL file, replacing the auto-generated one",
	Args:  cobra.ExactArgs(1),
	RunE:  runVarnishVCLImport,
}

var varnishVCLResetCmd = &cobra.Command{
	Use:   "vcl-reset",
	Short: "Reset VCL to default",
	Long:  "Regenerates the default auto-generated VCL, removing any custom VCL",
	RunE:  runVarnishVCLReset,
}

func init() {
	varnishCmd.AddCommand(varnishPurgeCmd)
	varnishCmd.AddCommand(varnishFlushCmd)
	varnishCmd.AddCommand(varnishStatusCmd)
	varnishCmd.AddCommand(varnishEnableCmd)
	varnishCmd.AddCommand(varnishDisableCmd)
	varnishCmd.AddCommand(varnishVCLImportCmd)
	varnishCmd.AddCommand(varnishVCLResetCmd)
	rootCmd.AddCommand(varnishCmd)
}

func runVarnishPurge(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	p, err := getPlatform()
	if err != nil {
		return err
	}

	cfg, ok := loadProjectConfig(cwd)
	if !ok {
		return nil
	}

	vclGen := varnish.NewVCLGenerator(p)
	ctrl := varnish.NewController(p, vclGen.VCLFilePath())

	if !ctrl.IsRunning() {
		fmt.Println("Varnish is not running")
		return nil
	}

	// Determine URL to purge
	url := "/"
	if len(args) > 0 {
		url = args[0]
	}

	// Purge for each domain
	for _, domain := range cfg.Domains {
		fmt.Printf("Purging %s%s... ", domain.Host, url)
		if err := ctrl.Purge(domain.Host, url); err != nil {
			fmt.Printf("failed: %v\n", err)
		} else {
			fmt.Println("done")
		}
	}

	return nil
}

func runVarnishFlush(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	vclGen := varnish.NewVCLGenerator(p)
	ctrl := varnish.NewController(p, vclGen.VCLFilePath())

	if !ctrl.IsRunning() {
		fmt.Println("Varnish is not running")
		return nil
	}

	fmt.Print("Flushing all Varnish cache... ")
	if err := ctrl.FlushAll(); err != nil {
		return fmt.Errorf("failed: %w", err)
	}
	fmt.Println("done")

	return nil
}

func runVarnishStatus(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	vclGen := varnish.NewVCLGenerator(p)
	ctrl := varnish.NewController(p, vclGen.VCLFilePath())

	fmt.Println("Varnish Status")
	fmt.Println("==============")

	if ctrl.IsRunning() {
		fmt.Println("Status: " + cli.Success("running"))

		// Get backend health
		backendCmd := exec.Command("docker", "exec", "magebox-varnish", "varnishadm", "backend.list")
		backendOutput, err := backendCmd.Output()
		if err == nil {
			lines := strings.Split(string(backendOutput), "\n")
			fmt.Println()
			fmt.Println("Backends:")
			for _, line := range lines {
				if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "Backend name") {
					fmt.Printf("  %s\n", strings.TrimSpace(line))
				}
			}
		}

		// Get cache stats
		statsCmd := exec.Command("docker", "exec", "magebox-varnish", "varnishstat", "-1")
		output, err := statsCmd.Output()
		if err == nil {
			lines := strings.Split(string(output), "\n")
			fmt.Println()
			fmt.Println("Cache Statistics:")
			for _, line := range lines {
				if strings.Contains(line, "MAIN.cache_hit ") ||
					strings.Contains(line, "MAIN.cache_miss ") ||
					strings.Contains(line, "MAIN.client_req ") {
					fmt.Printf("  %s\n", strings.TrimSpace(line))
				}
			}
		}
	} else {
		fmt.Println("Status: " + cli.Warning("stopped"))
	}

	return nil
}

func runVarnishEnable(cmd *cobra.Command, args []string) error {
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

	cli.PrintTitle("Enabling Varnish")
	fmt.Printf("Project: %s\n", cli.Highlight(cfg.Name))
	fmt.Println()

	// Check if already enabled
	if cfg.Services.HasVarnish() {
		cli.PrintInfo("Varnish is already enabled for this project")
		return nil
	}

	// Update config to enable Varnish
	cfg.Services.Varnish = &config.ServiceConfig{
		Enabled: true,
		Version: "7.5",
	}

	// Save config
	if err := config.SaveToPath(cfg, cwd); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Generate VCL
	fmt.Print("Generating VCL configuration... ")
	vclGen := varnish.NewVCLGenerator(p)
	if err := vclGen.Generate([]*config.Config{cfg}); err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to generate VCL: %w", err)
	}
	fmt.Println(cli.Success("done"))

	// Regenerate docker-compose and start Varnish
	fmt.Print("Starting Varnish container... ")
	composeGen := docker.NewComposeGenerator(p)
	if err := composeGen.GenerateGlobalServices([]*config.Config{cfg}); err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to generate docker-compose: %w", err)
	}

	dockerCtrl := docker.NewDockerController(composeGen.ComposeFilePath())
	if err := dockerCtrl.Up(); err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to start Varnish: %w", err)
	}
	fmt.Println(cli.Success("done"))

	// Regenerate vhost configuration to proxy to Varnish
	fmt.Print("Regenerating Nginx vhosts... ")
	sslMgr := ssl.NewManager(p)
	vhostGen := nginx.NewVhostGenerator(p, sslMgr)
	if err := vhostGen.Generate(cfg, cwd); err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to regenerate vhosts: %w", err)
	}
	fmt.Println(cli.Success("done"))

	// Reload Nginx to apply changes
	fmt.Print("Reloading Nginx... ")
	nginxCtrl := nginx.NewController(p)
	if err := nginxCtrl.Reload(); err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to reload nginx: %w", err)
	}
	fmt.Println(cli.Success("done"))

	fmt.Println()
	cli.PrintSuccess("Varnish enabled!")
	fmt.Println()
	cli.PrintInfo("Configure Magento to use Varnish:")
	fmt.Println("  bin/magento config:set system/full_page_cache/caching_application 2")

	return nil
}

func runVarnishDisable(cmd *cobra.Command, args []string) error {
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

	cli.PrintTitle("Disabling Varnish")
	fmt.Printf("Project: %s\n", cli.Highlight(cfg.Name))
	fmt.Println()

	// Check if already disabled
	if !cfg.Services.HasVarnish() {
		cli.PrintInfo("Varnish is not enabled for this project")
		return nil
	}

	// Update config to disable Varnish
	cfg.Services.Varnish = nil

	// Save config
	if err := config.SaveToPath(cfg, cwd); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Stop Varnish container
	fmt.Print("Stopping Varnish container... ")
	composeGen := docker.NewComposeGenerator(p)
	dockerCtrl := docker.NewDockerController(composeGen.ComposeFilePath())
	if err := dockerCtrl.StopService("varnish"); err != nil {
		fmt.Println(cli.Warning("not running"))
	} else {
		fmt.Println(cli.Success("done"))
	}

	// Regenerate docker-compose without Varnish
	if err := composeGen.GenerateGlobalServices([]*config.Config{cfg}); err != nil {
		return fmt.Errorf("failed to update docker-compose: %w", err)
	}

	// Regenerate vhost configuration to remove Varnish proxy
	fmt.Print("Regenerating Nginx vhosts... ")
	sslMgr := ssl.NewManager(p)
	vhostGen := nginx.NewVhostGenerator(p, sslMgr)
	if err := vhostGen.Generate(cfg, cwd); err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to regenerate vhosts: %w", err)
	}
	fmt.Println(cli.Success("done"))

	// Reload Nginx to apply changes
	fmt.Print("Reloading Nginx... ")
	nginxCtrl := nginx.NewController(p)
	if err := nginxCtrl.Reload(); err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to reload nginx: %w", err)
	}
	fmt.Println(cli.Success("done"))

	fmt.Println()
	cli.PrintSuccess("Varnish disabled!")
	cli.PrintInfo("Configure Magento to use built-in cache:")
	fmt.Println("  bin/magento config:set system/full_page_cache/caching_application 1")

	return nil
}

func runVarnishVCLImport(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	vclFile := args[0]

	// Validate the file exists and is readable
	info, err := os.Stat(vclFile)
	if err != nil {
		return fmt.Errorf("cannot access VCL file: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory, not a file", vclFile)
	}

	cli.PrintTitle("Importing Custom VCL")
	fmt.Printf("Source: %s\n", cli.Path(vclFile))
	fmt.Println()

	// Read the source file
	src, err := os.Open(vclFile)
	if err != nil {
		return fmt.Errorf("failed to open VCL file: %w", err)
	}
	defer src.Close()

	// Write to the MageBox VCL location
	vclGen := varnish.NewVCLGenerator(p)
	vclDir := vclGen.VCLDir()
	if err := os.MkdirAll(vclDir, 0755); err != nil {
		return fmt.Errorf("failed to create VCL directory: %w", err)
	}

	destPath := vclGen.VCLFilePath()

	// Back up the existing VCL if it exists
	if _, err := os.Stat(destPath); err == nil {
		backupPath := filepath.Join(vclDir, "default.vcl.bak")
		fmt.Print("Backing up current VCL... ")
		if err := copyFile(destPath, backupPath); err != nil {
			fmt.Println(cli.Warning("failed: " + err.Error()))
		} else {
			fmt.Println(cli.Success("done"))
		}
	}

	// Copy the new VCL
	fmt.Print("Importing VCL... ")
	dest, err := os.Create(destPath)
	if err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to create destination VCL: %w", err)
	}
	defer dest.Close()

	if _, err := io.Copy(dest, src); err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to copy VCL: %w", err)
	}
	fmt.Println(cli.Success("done"))

	// Reload Varnish if running
	ctrl := varnish.NewController(p, destPath)
	if ctrl.IsRunning() {
		fmt.Print("Reloading Varnish... ")
		if err := ctrl.Reload(); err != nil {
			fmt.Println(cli.Error("failed: " + err.Error()))
			cli.PrintWarning("Varnish may need to be restarted manually")
		} else {
			fmt.Println(cli.Success("done"))
		}
	}

	fmt.Println()
	cli.PrintSuccess("Custom VCL imported!")
	cli.PrintInfo("Reset to auto-generated VCL with: magebox varnish vcl-reset")

	return nil
}

func runVarnishVCLReset(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	p, err := getPlatform()
	if err != nil {
		return err
	}

	cli.PrintTitle("Resetting VCL to Default")
	fmt.Println()

	// Load project config(s) to regenerate VCL
	var configs []*config.Config
	cfg, ok := loadProjectConfig(cwd)
	if ok {
		configs = append(configs, cfg)
	}

	// Regenerate VCL
	fmt.Print("Generating default VCL... ")
	vclGen := varnish.NewVCLGenerator(p)
	if err := vclGen.Generate(configs); err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to generate VCL: %w", err)
	}
	fmt.Println(cli.Success("done"))

	// Reload Varnish if running
	ctrl := varnish.NewController(p, vclGen.VCLFilePath())
	if ctrl.IsRunning() {
		fmt.Print("Reloading Varnish... ")
		if err := ctrl.Reload(); err != nil {
			fmt.Println(cli.Error("failed: " + err.Error()))
		} else {
			fmt.Println(cli.Success("done"))
		}
	}

	fmt.Println()
	cli.PrintSuccess("VCL reset to default!")

	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

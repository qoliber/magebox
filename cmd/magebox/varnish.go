package main

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/qoliber/magebox/internal/cli"
	"github.com/qoliber/magebox/internal/config"
	"github.com/qoliber/magebox/internal/docker"
	"github.com/qoliber/magebox/internal/varnish"
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

func init() {
	varnishCmd.AddCommand(varnishPurgeCmd)
	varnishCmd.AddCommand(varnishFlushCmd)
	varnishCmd.AddCommand(varnishStatusCmd)
	varnishCmd.AddCommand(varnishEnableCmd)
	varnishCmd.AddCommand(varnishDisableCmd)
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

	fmt.Println()
	cli.PrintSuccess("Varnish enabled!")
	fmt.Println()
	fmt.Println("Configuration:")
	fmt.Printf("  HTTP Port:  %s (for testing)\n", cli.Highlight("6081"))
	fmt.Printf("  Admin Port: %s\n", cli.Highlight("6082"))
	fmt.Println()
	cli.PrintInfo("Test Varnish is working:")
	fmt.Printf("  curl -I http://127.0.0.1:6081/ -H \"Host: %s\"\n", cfg.Domains[0].Host)
	fmt.Println()
	cli.PrintInfo("Configure Magento to use Varnish:")
	fmt.Println("  bin/magento config:set system/full_page_cache/caching_application 2")
	fmt.Println("  bin/magento config:set system/full_page_cache/varnish/backend_host 127.0.0.1")
	fmt.Println("  bin/magento config:set system/full_page_cache/varnish/backend_port 8080")

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

	fmt.Println()
	cli.PrintSuccess("Varnish disabled!")
	cli.PrintInfo("Configure Magento to use built-in cache:")
	fmt.Println("  bin/magento config:set system/full_page_cache/caching_application 1")

	return nil
}

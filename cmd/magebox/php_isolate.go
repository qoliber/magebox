package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"qoliber/magebox/internal/cli"
	"qoliber/magebox/internal/config"
	"qoliber/magebox/internal/php"
	"qoliber/magebox/internal/project"
)

var phpIsolateCmd = &cobra.Command{
	Use:   "isolate",
	Short: "Enable isolated PHP-FPM master for this project",
	Long: `Enable a dedicated PHP-FPM master process for this project.

Isolated masters allow you to configure PHP_INI_SYSTEM settings
(like opcache.memory_consumption, opcache.jit, opcache.preload)
independently for each project.

Without isolation, these settings are shared across all projects
using the same PHP version.`,
	RunE: runPhpIsolateEnable,
}

var phpIsolateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show isolation status for this project",
	RunE:  runPhpIsolateStatus,
}

var phpIsolateDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable isolation and return to shared PHP-FPM pool",
	RunE:  runPhpIsolateDisable,
}

var phpIsolateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all isolated projects",
	RunE:  runPhpIsolateList,
}

var phpIsolateRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart isolated PHP-FPM master for this project",
	RunE:  runPhpIsolateRestart,
}

// Flags
var (
	isolateOpcacheMemory string
	isolateOpcacheJIT    string
	isolatePreload       string
)

func init() {
	phpCmd.AddCommand(phpIsolateCmd)
	phpIsolateCmd.AddCommand(phpIsolateStatusCmd)
	phpIsolateCmd.AddCommand(phpIsolateDisableCmd)
	phpIsolateCmd.AddCommand(phpIsolateListCmd)
	phpIsolateCmd.AddCommand(phpIsolateRestartCmd)

	// Flags for isolate command
	phpIsolateCmd.Flags().StringVar(&isolateOpcacheMemory, "opcache-memory", "", "OPcache memory consumption in MB (e.g., 512)")
	phpIsolateCmd.Flags().StringVar(&isolateOpcacheJIT, "jit", "", "OPcache JIT mode (off, tracing, function)")
	phpIsolateCmd.Flags().StringVar(&isolatePreload, "preload", "", "Path to preload script")
}

func runPhpIsolateEnable(cmd *cobra.Command, args []string) error {
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

	controller := php.NewIsolatedFPMController(p)

	// Build settings from flags and existing config
	settings := make(map[string]string)

	// Get existing php_ini settings from config
	if cfg.PHPINI != nil {
		for k, v := range cfg.PHPINI {
			settings[k] = v
		}
	}

	// Override with flags
	if isolateOpcacheMemory != "" {
		settings["opcache.memory_consumption"] = isolateOpcacheMemory
	}
	if isolateOpcacheJIT != "" {
		settings["opcache.jit"] = isolateOpcacheJIT
		if isolateOpcacheJIT != "off" && isolateOpcacheJIT != "0" {
			settings["opcache.jit_buffer_size"] = "128M"
		}
	}
	if isolatePreload != "" {
		settings["opcache.preload"] = isolatePreload
		settings["opcache.preload_user"] = os.Getenv("USER")
	}

	// Default: disable opcache if no settings specified (for development)
	if len(settings) == 0 {
		settings["opcache.enable"] = "0"
	}

	fmt.Printf("Enabling isolation for %s (PHP %s)...\n", cli.Highlight(cfg.Name), cfg.PHP)
	fmt.Println()

	// Enable isolation
	isolatedProject, err := controller.Enable(cfg.Name, cwd, cfg.PHP, settings)
	if err != nil {
		cli.PrintError("Failed to enable isolation: %v", err)
		return nil
	}

	// Save isolation flag to local config
	if err := saveIsolationConfig(cwd, true); err != nil {
		cli.PrintWarning("Failed to save isolation config: %v", err)
	}

	// Restart the project to regenerate nginx with new socket
	fmt.Print("Regenerating nginx config with isolated socket... ")
	mgr := project.NewManager(p)
	if _, err := mgr.Start(cwd); err != nil {
		fmt.Println(cli.Error("failed"))
		cli.PrintWarning("Run 'mbox restart' to apply changes")
	} else {
		fmt.Println(cli.Success("done"))
	}

	fmt.Println()
	cli.PrintSuccess("Isolation enabled!")
	fmt.Println()
	fmt.Println(cli.Header("Isolated Master Details"))
	fmt.Printf("  Socket: %s\n", isolatedProject.SocketPath)
	fmt.Printf("  PID:    %s\n", isolatedProject.PIDPath)
	fmt.Printf("  Config: %s\n", isolatedProject.ConfigPath)
	fmt.Println()

	if len(settings) > 0 {
		fmt.Println(cli.Header("PHP_INI_SYSTEM Settings"))
		for k, v := range settings {
			fmt.Printf("  %s = %s\n", k, v)
		}
	}

	return nil
}

func runPhpIsolateStatus(cmd *cobra.Command, args []string) error {
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

	controller := php.NewIsolatedFPMController(p)

	if !controller.IsIsolated(cfg.Name) {
		fmt.Printf("Project %s is using %s PHP-FPM pool\n", cli.Highlight(cfg.Name), cli.Warning("shared"))
		fmt.Println()
		fmt.Println("Run 'mbox php isolate' to enable a dedicated PHP-FPM master.")
		return nil
	}

	status, err := controller.GetStatus(cfg.Name)
	if err != nil {
		cli.PrintError("Failed to get status: %v", err)
		return nil
	}

	fmt.Printf("Project %s is using %s PHP-FPM master\n", cli.Highlight(cfg.Name), cli.Success("isolated"))
	fmt.Println()

	fmt.Println(cli.Header("Master Details"))
	fmt.Printf("  PHP Version: %s\n", status["php_version"])
	fmt.Printf("  Socket:      %s\n", status["socket_path"])
	fmt.Printf("  PID File:    %s\n", status["pid_path"])
	fmt.Printf("  Config:      %s\n", status["config_path"])

	if running, ok := status["running"].(bool); ok && running {
		if pid, ok := status["pid"].(int); ok {
			fmt.Printf("  Status:      %s (PID: %d)\n", cli.Success("Running"), pid)
		} else {
			fmt.Printf("  Status:      %s\n", cli.Success("Running"))
		}
	} else {
		fmt.Printf("  Status:      %s\n", cli.Error("Stopped"))
	}
	fmt.Println()

	if settings, ok := status["settings"].(map[string]string); ok && len(settings) > 0 {
		fmt.Println(cli.Header("PHP_INI_SYSTEM Settings"))
		for k, v := range settings {
			fmt.Printf("  %s = %s\n", k, v)
		}
	}

	return nil
}

func runPhpIsolateDisable(cmd *cobra.Command, args []string) error {
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

	controller := php.NewIsolatedFPMController(p)

	if !controller.IsIsolated(cfg.Name) {
		fmt.Printf("Project %s is not isolated\n", cli.Highlight(cfg.Name))
		return nil
	}

	fmt.Printf("Disabling isolation for %s...\n", cli.Highlight(cfg.Name))

	if err := controller.Disable(cfg.Name); err != nil {
		cli.PrintError("Failed to disable isolation: %v", err)
		return nil
	}

	// Remove isolation flag from local config
	if err := saveIsolationConfig(cwd, false); err != nil {
		cli.PrintWarning("Failed to save config: %v", err)
	}

	// Restart project to regenerate nginx with shared socket
	fmt.Print("Restarting with shared PHP-FPM pool... ")
	mgr := project.NewManager(p)
	if _, err := mgr.Start(cwd); err != nil {
		fmt.Println(cli.Error("failed"))
		cli.PrintWarning("Run 'mbox restart' to apply changes")
	} else {
		fmt.Println(cli.Success("done"))
	}

	fmt.Println()
	cli.PrintSuccess("Isolation disabled. Project now uses shared PHP-FPM pool.")
	return nil
}

func runPhpIsolateList(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	controller := php.NewIsolatedFPMController(p)
	projects, err := controller.GetRegistry().List()
	if err != nil {
		cli.PrintError("Failed to list isolated projects: %v", err)
		return nil
	}

	if len(projects) == 0 {
		fmt.Println("No isolated projects found.")
		fmt.Println()
		fmt.Println("Run 'mbox php isolate' in a project directory to enable isolation.")
		return nil
	}

	fmt.Println(cli.Header("Isolated Projects"))
	fmt.Println()

	for _, proj := range projects {
		running := controller.IsRunning(proj.ProjectName)
		status := cli.Error("Stopped")
		if running {
			status = cli.Success("Running")
		}

		fmt.Printf("%s (PHP %s) - %s\n", cli.Highlight(proj.ProjectName), proj.PHPVersion, status)
		fmt.Printf("  Path:   %s\n", proj.ProjectPath)
		fmt.Printf("  Socket: %s\n", proj.SocketPath)

		if len(proj.Settings) > 0 {
			settingsList := make([]string, 0, len(proj.Settings))
			for k, v := range proj.Settings {
				settingsList = append(settingsList, fmt.Sprintf("%s=%s", k, v))
			}
			fmt.Printf("  Settings: %s\n", strings.Join(settingsList, ", "))
		}
		fmt.Println()
	}

	return nil
}

func runPhpIsolateRestart(cmd *cobra.Command, args []string) error {
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

	controller := php.NewIsolatedFPMController(p)

	if !controller.IsIsolated(cfg.Name) {
		fmt.Printf("Project %s is not isolated\n", cli.Highlight(cfg.Name))
		return nil
	}

	fmt.Printf("Restarting isolated PHP-FPM for %s... ", cli.Highlight(cfg.Name))

	if err := controller.Restart(cfg.Name); err != nil {
		fmt.Println(cli.Error("failed"))
		cli.PrintError("Failed to restart: %v", err)
		return nil
	}

	fmt.Println(cli.Success("done"))
	return nil
}

// saveIsolationConfig saves the isolation flag to .magebox.local.yaml
func saveIsolationConfig(cwd string, isolated bool) error {
	localPath := filepath.Join(cwd, config.LocalConfigFileName)

	// Load existing local config content
	existingContent := ""
	if data, err := os.ReadFile(localPath); err == nil {
		existingContent = string(data)
	}

	// Remove existing isolated line if present
	lines := strings.Split(existingContent, "\n")
	newLines := make([]string, 0, len(lines))
	for _, line := range lines {
		if !strings.HasPrefix(strings.TrimSpace(line), "isolated:") {
			newLines = append(newLines, line)
		}
	}

	// Add new isolated setting if true
	if isolated {
		newLines = append(newLines, "isolated: true")
	}

	// Clean up empty lines at end
	for len(newLines) > 0 && strings.TrimSpace(newLines[len(newLines)-1]) == "" {
		newLines = newLines[:len(newLines)-1]
	}

	content := strings.Join(newLines, "\n")
	if content != "" {
		content += "\n"
	}

	if content == "" {
		// Remove file if empty
		_ = os.Remove(localPath)
		return nil
	}

	return os.WriteFile(localPath, []byte(content), 0644)
}

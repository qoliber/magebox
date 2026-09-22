package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/spf13/cobra"

	"qoliber/magebox/internal/cli"
	"qoliber/magebox/internal/config"
	"qoliber/magebox/internal/php"
	"qoliber/magebox/internal/platform"
	"qoliber/magebox/internal/project"
)

// stopPreloadRelPath is the project-relative path to the OPcache preload script
// used by the STOP (Static Precompilation & OPcache Protocol) commands.
const stopPreloadRelPath = "app/preload.php"

// stopOpcacheMemory is the opcache.memory_consumption value (in megabytes) that
// STOP raises the default to. The PHP default of 128M is too small for a real
// Magento store once preload and request-time caching are both in play.
const stopOpcacheMemory = "512"

// stopOpcacheJIT enables the tracing JIT (the friendliest of PHP's JIT modes
// for general-purpose workloads). Magebox defaults this to "off" for dev to
// avoid JIT segfaults; STOP opts the project back in.
const stopOpcacheJIT = "tracing"

// stopOpcacheJITBufferSize is the JIT buffer size. JIT is disabled when this
// is 0, so any non-zero value is required alongside opcache.jit.
const stopOpcacheJITBufferSize = "100M"

var stopProtocolCmd = &cobra.Command{
	Use:   "stop-protocol",
	Short: "Manage STOP (Static Precompilation & OPcache Protocol)",
	Long: `Enable, disable, or check status of STOP (Static Precompilation & OPcache Protocol).

STOP enables OPcache for the current project and configures
app/preload.php as the OPcache preload script.`,
	RunE: runStopProtocolStatus,
}

var stopProtocolEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable STOP (Static Precompilation & OPcache Protocol)",
	Long: `Enables OPcache for the current project and configures
app/preload.php as the OPcache preload script.`,
	RunE: runStopProtocolEnable,
}

var stopProtocolDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable STOP (Static Precompilation & OPcache Protocol)",
	Long:  "Disables OPcache for the current project and clears the OPcache preload script setting.",
	RunE:  runStopProtocolDisable,
}

var stopProtocolStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show STOP (Static Precompilation & OPcache Protocol) status",
	Long:  "Shows whether OPcache is enabled for the project and which preload script is configured.",
	RunE:  runStopProtocolStatus,
}

func init() {
	stopProtocolCmd.AddCommand(stopProtocolEnableCmd)
	stopProtocolCmd.AddCommand(stopProtocolDisableCmd)
	stopProtocolCmd.AddCommand(stopProtocolStatusCmd)
	rootCmd.AddCommand(stopProtocolCmd)
}

func runStopProtocolEnable(cmd *cobra.Command, args []string) error {
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

	preloadPath := filepath.Join(cwd, stopPreloadRelPath)

	preloadUser, err := currentUsername()
	if err != nil {
		return fmt.Errorf("failed to determine current user for opcache.preload_user: %w", err)
	}

	localCfg, err := config.LoadLocalConfig(cwd)
	if err != nil {
		localCfg = &config.LocalConfig{PHPINI: make(map[string]string)}
	}
	if localCfg.PHPINI == nil {
		localCfg.PHPINI = make(map[string]string)
	}
	localCfg.PHPINI["opcache.enable"] = "1"
	localCfg.PHPINI["opcache.preload"] = preloadPath
	localCfg.PHPINI["opcache.preload_user"] = preloadUser
	localCfg.PHPINI["opcache.memory_consumption"] = stopOpcacheMemory
	localCfg.PHPINI["opcache.jit"] = stopOpcacheJIT
	localCfg.PHPINI["opcache.jit_buffer_size"] = stopOpcacheJITBufferSize

	if err := config.SaveLocalConfig(cwd, localCfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	cli.PrintSuccess("STOP (Static Precompilation & OPcache Protocol) enabled")
	fmt.Printf("  opcache.enable             = %s\n", cli.Highlight("1"))
	fmt.Printf("  opcache.preload            = %s\n", cli.Highlight(preloadPath))
	fmt.Printf("  opcache.preload_user       = %s\n", cli.Highlight(preloadUser))
	fmt.Printf("  opcache.memory_consumption = %s\n", cli.Highlight(stopOpcacheMemory))
	fmt.Printf("  opcache.jit                = %s\n", cli.Highlight(stopOpcacheJIT))
	fmt.Printf("  opcache.jit_buffer_size    = %s\n", cli.Highlight(stopOpcacheJITBufferSize))

	if _, err := os.Stat(preloadPath); os.IsNotExist(err) {
		cli.PrintWarning("Preload script does not exist: %s", preloadPath)
		cli.PrintInfo("OPcache will skip preloading until the file is created")
	}

	return applyStopReload(p, cwd, cfg.PHP)
}

func runStopProtocolDisable(cmd *cobra.Command, args []string) error {
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

	localCfg, err := config.LoadLocalConfig(cwd)
	if err != nil {
		localCfg = &config.LocalConfig{PHPINI: make(map[string]string)}
	}
	if localCfg.PHPINI == nil {
		localCfg.PHPINI = make(map[string]string)
	}
	localCfg.PHPINI["opcache.enable"] = "0"
	delete(localCfg.PHPINI, "opcache.preload")
	delete(localCfg.PHPINI, "opcache.preload_user")
	delete(localCfg.PHPINI, "opcache.memory_consumption")
	delete(localCfg.PHPINI, "opcache.jit")
	delete(localCfg.PHPINI, "opcache.jit_buffer_size")

	if err := config.SaveLocalConfig(cwd, localCfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	cli.PrintSuccess("STOP (Static Precompilation & OPcache Protocol) disabled")

	return applyStopReload(p, cwd, cfg.PHP)
}

func runStopProtocolStatus(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	cfg, ok := loadProjectConfig(cwd)
	if !ok {
		return nil
	}

	merged := php.GetMergedPHPINI(cfg.PHPINI)
	enabled := merged["opcache.enable"] == "1"
	preload := merged["opcache.preload"]
	preloadUser := merged["opcache.preload_user"]

	cli.PrintTitle("STOP (Static Precompilation & OPcache Protocol) Status")
	fmt.Println()
	fmt.Printf("OPcache enabled: %s\n", formatBool(enabled))
	if preload != "" {
		fmt.Printf("Preload script:  %s\n", cli.Highlight(preload))
		if _, err := os.Stat(preload); os.IsNotExist(err) {
			cli.PrintWarning("Preload script does not exist on disk")
		}
	} else {
		fmt.Printf("Preload script:  %s\n", cli.Warning("(not set)"))
	}
	if preloadUser != "" {
		fmt.Printf("Preload user:    %s\n", cli.Highlight(preloadUser))
	}

	fmt.Println()
	if !enabled || preload == "" {
		cli.PrintInfo("Enable with: %s", cli.Command("magebox stop-protocol enable"))
	} else {
		cli.PrintInfo("Disable with: %s", cli.Command("magebox stop-protocol disable"))
	}
	return nil
}

func currentUsername() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	if u.Username == "" {
		return "", fmt.Errorf("current user has empty username")
	}
	return u.Username, nil
}

func applyStopReload(p *platform.Platform, cwd, phpVersion string) error {
	mgr := project.NewManager(p)
	if err := mgr.RegenerateConfigs(cwd); err != nil {
		cli.PrintWarning("Failed to regenerate configs: %v", err)
	}

	// A full restart is required (not a reload): opcache.preload and the opcache
	// extension are only evaluated when the FPM master starts. SIGUSR2 / systemctl
	// reload would leave the existing master running with the old preload state and
	// produce "OPcache can't be temporary enabled" warnings on every worker spawn.
	fmt.Println("Restarting PHP-FPM to apply changes...")
	fpmController := php.NewFPMController(p, phpVersion)
	if err := fpmController.Restart(); err != nil {
		cli.PrintWarning("Failed to restart PHP-FPM: %v", err)
	} else {
		cli.PrintSuccess("PHP-FPM restarted")
	}
	return nil
}

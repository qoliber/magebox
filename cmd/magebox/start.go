package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"qoliber/magebox/internal/cli"
	"qoliber/magebox/internal/docker"
	"qoliber/magebox/internal/portforward"
	"qoliber/magebox/internal/project"
)

var startAllProjects bool

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start project services",
	Long:  "Starts all services defined in .magebox for the current project, or all projects with --all",
	RunE:  runStart,
}

func init() {
	startCmd.Flags().BoolVarP(&startAllProjects, "all", "a", false, "Start all MageBox projects")
	rootCmd.AddCommand(startCmd)
}

func runStart(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	mgr := project.NewManager(p)

	if startAllProjects {
		return startAll(p, mgr)
	}

	// Start current project
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	return startProject(mgr, cwd, true)
}

func ensurePortForwarding() {
	if runtime.GOOS != "darwin" {
		return
	}

	pfMgr := portforward.NewManager()
	if !pfMgr.IsInstalled() {
		return // Not set up yet, bootstrap needed
	}

	wasActive, err := pfMgr.EnsureRulesActive()
	if err != nil {
		cli.PrintWarning("Port forwarding: %v", err)
	} else if !wasActive {
		cli.PrintInfo("Port forwarding rules restored (80→8080, 443→8443)")
	}
}

func startProject(mgr *project.Manager, projectPath string, verbose bool) error {
	if verbose {
		cli.PrintTitle("Starting MageBox Services")
		fmt.Println()

		// On macOS, verify pf port forwarding is active (survives reboot/sleep)
		ensurePortForwarding()
	}

	// Validate first
	cfg, warnings, err := mgr.ValidateConfig(projectPath)
	if err != nil {
		cli.PrintError("%v", err)
		return err
	}

	// Show warnings
	for _, w := range warnings {
		cli.PrintWarning("%s", w)
	}

	// Start services
	result, err := mgr.Start(projectPath)
	if err != nil {
		cli.PrintError("%v", err)
		return err
	}

	if verbose {
		// Show results
		fmt.Printf("Project: %s\n", cli.Highlight(cfg.Name))
		fmt.Printf("PHP:     %s\n", cli.Highlight(result.PHPVersion))
		fmt.Println()

		fmt.Println(cli.Header("Domains"))
		for _, d := range result.Domains {
			fmt.Printf("  %s\n", cli.URL("https://"+d))
		}
		fmt.Println()

		fmt.Println(cli.Header("Services"))
		for _, s := range result.Services {
			fmt.Printf("  %s %s\n", cli.Success(""), s)
		}
		fmt.Println()

		// Show warnings from start
		for _, w := range result.Warnings {
			cli.PrintWarning("%s", w)
		}

		// Show errors
		for _, e := range result.Errors {
			cli.PrintError("%v", e)
		}

		if len(result.Errors) == 0 {
			cli.PrintSuccess("Project started successfully!")
		}

		// Show system INI instructions if there are system settings
		if result.SystemINIInfo != "" {
			fmt.Println(result.SystemINIInfo)
		}
	}

	// Handle project-specific compose file
	if cfg.ComposeFile != "" {
		composeFile := cfg.ComposeFile
		if !filepath.IsAbs(composeFile) {
			composeFile = filepath.Join(projectPath, composeFile)
		}
		if _, err := os.Stat(composeFile); err == nil {
			if err := promptComposeUp(composeFile); err != nil {
				cli.PrintWarning("Custom containers: %v", err)
			}
		} else {
			cli.PrintWarning("Compose file not found: %s", composeFile)
		}
	}

	return nil
}

// promptComposeUp asks the user whether to start project-specific Docker containers
func promptComposeUp(composeFile string) error {
	services, err := docker.ProjectComposeServices(composeFile)
	if err != nil {
		return fmt.Errorf("failed to read compose file: %w", err)
	}
	if len(services) == 0 {
		return nil
	}

	fmt.Println()
	cli.PrintInfo("Project has custom Docker containers (%s):", cli.Path(composeFile))
	for _, svc := range services {
		fmt.Printf("  %s %s\n", cli.Bullet(""), svc)
	}
	fmt.Print("Start these containers? [Y/n] ")

	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))

	if answer != "" && answer != "y" && answer != "yes" {
		cli.PrintInfo("Skipped custom containers")
		return nil
	}

	fmt.Println()
	if err := docker.ProjectComposeUp(composeFile); err != nil {
		return fmt.Errorf("failed to start containers: %w", err)
	}
	cli.PrintSuccess("Custom containers started and connected to MageBox network")
	return nil
}

func startAll(p interface{ MageBoxDir() string }, mgr *project.Manager) error {
	cli.PrintTitle("Starting All MageBox Projects")
	fmt.Println()

	// Need platform type for discovery
	plat, err := getPlatform()
	if err != nil {
		return err
	}

	discovery := project.NewProjectDiscovery(plat)
	projects, err := discovery.DiscoverProjects()
	if err != nil {
		cli.PrintError("Failed to discover projects: %v", err)
		return nil
	}

	if len(projects) == 0 {
		cli.PrintInfo("No projects found")
		return nil
	}

	started := 0
	failed := 0

	for _, proj := range projects {
		if !proj.HasConfig {
			continue // Skip projects without .magebox.yaml
		}

		fmt.Printf("Starting %s... ", cli.Highlight(proj.Name))

		// Validate and start
		_, _, err := mgr.ValidateConfig(proj.Path)
		if err != nil {
			fmt.Println(cli.Error("failed"))
			cli.PrintError("  %v", err)
			failed++
			continue
		}

		result, err := mgr.Start(proj.Path)
		if err != nil {
			fmt.Println(cli.Error("failed"))
			cli.PrintError("  %v", err)
			failed++
		} else if len(result.Errors) > 0 {
			fmt.Println(cli.Warning("partial"))
			for _, e := range result.Errors {
				cli.PrintError("  %v", e)
			}
			started++ // Count as started even with warnings
		} else {
			fmt.Println(cli.Success("done"))
			started++
		}
	}

	fmt.Println()
	if failed > 0 {
		cli.PrintWarning("Started %d project(s), %d failed", started, failed)
	} else {
		cli.PrintSuccess("Started %d project(s)", started)
	}

	return nil
}

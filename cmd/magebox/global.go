package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/qoliber/magebox/internal/cli"
	"github.com/qoliber/magebox/internal/docker"
	"github.com/qoliber/magebox/internal/nginx"
	"github.com/qoliber/magebox/internal/php"
	"github.com/qoliber/magebox/internal/platform"
)

var globalCmd = &cobra.Command{
	Use:   "global",
	Short: "Global service management",
	Long:  "Manage global MageBox services",
}

var globalStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start global services",
	Long:  "Starts Nginx, Varnish, and Docker services",
	RunE:  runGlobalStart,
}

var globalStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop global services",
	Long:  "Stops all global MageBox services",
	RunE:  runGlobalStop,
}

var globalStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show global status",
	Long:  "Shows status of all global services and registered projects",
	RunE:  runGlobalStatus,
}

func init() {
	globalCmd.AddCommand(globalStartCmd)
	globalCmd.AddCommand(globalStopCmd)
	globalCmd.AddCommand(globalStatusCmd)
	rootCmd.AddCommand(globalCmd)
}

func runGlobalStart(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	cli.PrintTitle("Starting Global Services")
	fmt.Println()

	// Check Docker is running
	dockerCmd := exec.Command("docker", "info")
	if dockerCmd.Run() != nil {
		cli.PrintError("Docker is not running. Please start Docker first.")
		return nil
	}

	// Start Nginx
	fmt.Print("  Nginx... ")
	nginxCtrl := nginx.NewController(p)
	if nginxCtrl.IsRunning() {
		fmt.Println(cli.Success("already running"))
	} else if err := nginxCtrl.Start(); err != nil {
		fmt.Println(cli.Error("failed: " + err.Error()))
	} else {
		fmt.Println(cli.Success("started"))
	}

	// Start Docker services
	composeGen := docker.NewComposeGenerator(p)
	composeFile := composeGen.ComposeFilePath()

	// Check if compose file exists
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		cli.PrintWarning("Docker services not configured. Run " + cli.Command("magebox bootstrap") + " first.")
	} else {
		fmt.Print("  Docker services... ")
		dockerCtrl := docker.NewDockerController(composeFile)
		if err := dockerCtrl.Up(); err != nil {
			fmt.Println(cli.Error("failed: " + err.Error()))
		} else {
			fmt.Println(cli.Success("started"))
		}

		// List running services
		if services, err := dockerCtrl.GetRunningServices(); err == nil && len(services) > 0 {
			for _, svc := range services {
				fmt.Printf("    %s %s\n", cli.Success("✓"), svc)
			}
		}
	}

	fmt.Println()
	cli.PrintSuccess("Global services started!")
	fmt.Println()
	cli.PrintInfo("Run " + cli.Command("magebox start") + " in your project directory to start project services.")

	return nil
}

func runGlobalStop(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	fmt.Println("Stopping global services...")

	// Stop Nginx
	fmt.Print("  Nginx... ")
	nginxCtrl := nginx.NewController(p)
	if err := nginxCtrl.Stop(); err != nil {
		fmt.Printf("failed: %v\n", err)
	} else {
		fmt.Println("stopped")
	}

	// Stop Docker services
	composeGen := docker.NewComposeGenerator(p)
	composeFile := composeGen.ComposeFilePath()
	if _, err := os.Stat(composeFile); err == nil {
		fmt.Print("  Docker services... ")
		dockerCtrl := docker.NewDockerController(composeFile)
		if err := dockerCtrl.Down(); err != nil {
			fmt.Printf("failed: %v\n", err)
		} else {
			fmt.Println("stopped")
		}
	}

	fmt.Println("\nGlobal services stopped!")
	return nil
}

func runGlobalStatus(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	cli.PrintTitle("Global Services Status")
	fmt.Println()

	// Check Nginx
	nginxCtrl := nginx.NewController(p)
	fmt.Printf("  %-20s %s\n", "Nginx", cli.Status(nginxCtrl.IsRunning()))

	// Check Docker
	dockerInstalled := platform.CommandExists("docker")
	dockerRunning := false
	if dockerInstalled {
		dockerCmd := exec.Command("docker", "info")
		dockerRunning = dockerCmd.Run() == nil
	}
	if !dockerInstalled {
		fmt.Printf("  %-20s %s\n", "Docker", cli.StatusInstalled(false))
	} else {
		fmt.Printf("  %-20s %s\n", "Docker", cli.Status(dockerRunning))
	}

	// Check mkcert
	fmt.Printf("  %-20s %s\n", "mkcert", cli.StatusInstalled(platform.CommandExists("mkcert")))

	// List PHP versions
	fmt.Println(cli.Header("PHP Versions"))
	detector := php.NewDetector(p)
	for _, v := range php.SupportedVersions {
		version := detector.Detect(v)
		var status string
		if !version.Installed {
			status = cli.StatusInstalled(false)
		} else if version.FPMRunning {
			status = cli.Status(true)
		} else {
			status = cli.StatusInstalled(true)
		}
		fmt.Printf("  %-20s %s\n", "PHP "+v, status)
	}

	return nil
}

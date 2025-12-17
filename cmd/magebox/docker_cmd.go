package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/qoliber/magebox/internal/cli"
	"github.com/qoliber/magebox/internal/config"
	"github.com/qoliber/magebox/internal/docker"
)

var dockerCmd = &cobra.Command{
	Use:   "docker",
	Short: "Manage Docker providers",
	Long: `View and manage Docker providers (Docker Desktop, Colima, OrbStack, etc.)

Without arguments, shows the current Docker provider status.
Use 'magebox docker use <provider>' to switch providers.`,
	RunE: runDockerStatus,
}

var dockerUseCmd = &cobra.Command{
	Use:   "use <provider>",
	Short: "Switch Docker provider",
	Long: `Switch to a different Docker provider.

Available providers:
  desktop   - Docker Desktop
  colima    - Colima
  orbstack  - OrbStack
  rancher   - Rancher Desktop
  lima      - Lima

Example:
  magebox docker use colima`,
	Args: cobra.ExactArgs(1),
	RunE: runDockerUse,
}

func init() {
	dockerCmd.AddCommand(dockerUseCmd)
	rootCmd.AddCommand(dockerCmd)
}

func runDockerStatus(cmd *cobra.Command, args []string) error {
	if runtime.GOOS != "darwin" {
		fmt.Println("Docker provider management is only available on macOS.")
		fmt.Println("On Linux, the default Docker installation is used.")
		return nil
	}

	providerMgr := docker.NewProviderManager()
	providers := providerMgr.GetProviders()
	activeProvider := providerMgr.GetActiveProvider()

	cli.PrintTitle("Docker Provider")
	fmt.Println()

	if activeProvider != nil {
		fmt.Printf("  Current:  %s\n", cli.Highlight(activeProvider.Name))
		fmt.Printf("  Socket:   %s\n", cli.Path(activeProvider.SocketPath))
	} else {
		fmt.Println("  Current:  " + cli.Warning("none detected"))
	}
	fmt.Println()

	// Check DOCKER_HOST
	dockerHost := os.Getenv("DOCKER_HOST")
	if dockerHost != "" {
		fmt.Printf("  DOCKER_HOST: %s\n", cli.Path(dockerHost))
		fmt.Println()
	}

	// List available providers
	fmt.Println(cli.Subtitle("Available Providers"))
	fmt.Println()

	if len(providers) == 0 {
		fmt.Println("  No Docker providers detected.")
		fmt.Println()
		fmt.Println("  Install one of:")
		fmt.Println("    - Docker Desktop: https://docker.com/products/docker-desktop")
		fmt.Println("    - Colima:         brew install colima docker")
		fmt.Println("    - OrbStack:       https://orbstack.dev")
		return nil
	}

	for _, p := range providers {
		marker := "○"
		if p.IsActive {
			marker = "●"
		}

		status := cli.Error("stopped")
		if p.IsRunning {
			status = cli.Success("running")
		}

		fmt.Printf("  %s %-12s %s   %s\n", marker, p.Name, status, cli.Path(p.SocketPath))
	}
	fmt.Println()

	// Show instructions if multiple running providers
	runningProviders := providerMgr.GetRunningProviders()
	if len(runningProviders) > 1 {
		cli.PrintWarning("Multiple Docker providers are running!")
		fmt.Println("  This may cause confusion about which Docker daemon is being used.")
		fmt.Println()
	}

	fmt.Println("To switch provider:")
	fmt.Println("  " + cli.Command("magebox docker use <provider>"))
	fmt.Println()

	return nil
}

func runDockerUse(cmd *cobra.Command, args []string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("docker provider management is only available on macOS")
	}

	providerName := args[0]
	validProviders := []string{"desktop", "colima", "orbstack", "rancher", "lima"}

	// Validate provider name
	valid := false
	for _, p := range validProviders {
		if p == providerName {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("unknown provider: %s\nValid providers: desktop, colima, orbstack, rancher, lima", providerName)
	}

	providerMgr := docker.NewProviderManager()
	provider := providerMgr.GetProviderByName(providerName)

	// Check if provider exists/is installed
	if provider == nil {
		socketPath := providerMgr.GetSocketForProvider(providerName)
		fmt.Printf("Provider '%s' not detected.\n", providerName)
		fmt.Printf("Expected socket: %s\n", socketPath)
		fmt.Println()

		switch providerName {
		case "colima":
			fmt.Println("Install Colima:")
			fmt.Println("  brew install colima docker docker-compose")
			fmt.Println("  colima start")
		case "orbstack":
			fmt.Println("Install OrbStack:")
			fmt.Println("  brew install orbstack")
			fmt.Println("  # Or download from https://orbstack.dev")
		case "desktop":
			fmt.Println("Install Docker Desktop:")
			fmt.Println("  brew install --cask docker")
			fmt.Println("  # Or download from https://docker.com/products/docker-desktop")
		case "rancher":
			fmt.Println("Install Rancher Desktop:")
			fmt.Println("  brew install --cask rancher")
		}
		return nil
	}

	// Check if provider is running
	if !provider.IsRunning {
		fmt.Printf("Provider '%s' is installed but not running.\n", providerName)
		fmt.Println()

		switch providerName {
		case "colima":
			fmt.Println("Start Colima:")
			fmt.Println("  colima start")
		case "orbstack":
			fmt.Println("Start OrbStack:")
			fmt.Println("  open -a OrbStack")
		case "desktop":
			fmt.Println("Start Docker Desktop:")
			fmt.Println("  open -a Docker")
		case "rancher":
			fmt.Println("Start Rancher Desktop:")
			fmt.Println("  open -a \"Rancher Desktop\"")
		}
		return nil
	}

	// Update global config
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	globalCfg, err := config.LoadGlobalConfig(homeDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	globalCfg.DockerProvider = providerName
	if err := config.SaveGlobalConfig(homeDir, globalCfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	cli.PrintSuccess("Docker provider set to: %s", providerName)
	fmt.Println()

	// Show instructions for setting DOCKER_HOST
	socketPath := provider.SocketPath
	dockerHost := docker.FormatDockerHost(socketPath)

	fmt.Println("To use this provider, add to your shell profile:")
	fmt.Println()

	shellName := filepath.Base(os.Getenv("SHELL"))
	shellRC := "~/.bashrc"
	if shellName == "zsh" {
		shellRC = "~/.zshrc"
	}

	fmt.Printf("  echo 'export DOCKER_HOST=\"%s\"' >> %s\n", dockerHost, shellRC)
	fmt.Println()
	fmt.Println("Then reload your shell:")
	fmt.Printf("  source %s\n", shellRC)
	fmt.Println()

	// Or for immediate use
	fmt.Println("Or for immediate use in this terminal:")
	fmt.Printf("  export DOCKER_HOST=\"%s\"\n", dockerHost)
	fmt.Println()

	return nil
}

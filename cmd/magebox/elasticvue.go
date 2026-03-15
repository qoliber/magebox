package main

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"qoliber/magebox/internal/cli"
	"qoliber/magebox/internal/config"
	"qoliber/magebox/internal/docker"
	"qoliber/magebox/internal/platform"
	"qoliber/magebox/internal/project"
)

var elasticvueCmd = &cobra.Command{
	Use:   "elasticvue",
	Short: "Elasticvue search engine UI",
	Long:  "Manage Elasticvue web UI for OpenSearch/Elasticsearch",
}

var elasticvueEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable Elasticvue",
	Long:  "Enables Elasticvue web UI for browsing OpenSearch/Elasticsearch data",
	RunE:  runElasticvueEnable,
}

var elasticvueDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable Elasticvue",
	Long:  "Disables Elasticvue web UI",
	RunE:  runElasticvueDisable,
}

var elasticvueStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Elasticvue status",
	Long:  "Shows Elasticvue status and connection information",
	RunE:  runElasticvueStatus,
}

func init() {
	elasticvueCmd.AddCommand(elasticvueEnableCmd)
	elasticvueCmd.AddCommand(elasticvueDisableCmd)
	elasticvueCmd.AddCommand(elasticvueStatusCmd)
	rootCmd.AddCommand(elasticvueCmd)
}

func runElasticvueEnable(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	globalCfg, err := config.LoadGlobalConfig(p.HomeDir)
	if err != nil {
		return fmt.Errorf("failed to load global config: %w", err)
	}

	if globalCfg.Elasticvue {
		cli.PrintInfo("Elasticvue is already enabled")
		fmt.Println()
		fmt.Printf("  Web UI: %s\n", cli.Highlight("http://localhost:8080"))
		return nil
	}

	cli.PrintTitle("Enabling Elasticvue")
	fmt.Println()

	globalCfg.Elasticvue = true
	if err := config.SaveGlobalConfig(p.HomeDir, globalCfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Regenerate docker-compose and start
	fmt.Print("Starting Elasticvue container... ")
	composeGen := docker.NewComposeGenerator(p)

	// Discover all project configs to regenerate docker-compose
	configs := discoverAllConfigs(p)
	if err := composeGen.GenerateGlobalServices(configs); err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to generate docker-compose: %w", err)
	}

	dockerCtrl := docker.NewDockerController(composeGen.ComposeFilePath())
	if err := dockerCtrl.StartService("elasticvue"); err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to start Elasticvue: %w", err)
	}
	fmt.Println(cli.Success("done"))

	fmt.Println()
	cli.PrintSuccess("Elasticvue enabled!")
	fmt.Println()
	fmt.Printf("  Web UI: %s\n", cli.Highlight("http://localhost:8080"))
	fmt.Println()
	cli.PrintInfo("Add your cluster in the Elasticvue UI using http://localhost:9200")

	return nil
}

func runElasticvueDisable(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	globalCfg, err := config.LoadGlobalConfig(p.HomeDir)
	if err != nil {
		return fmt.Errorf("failed to load global config: %w", err)
	}

	if !globalCfg.Elasticvue {
		cli.PrintInfo("Elasticvue is not enabled")
		return nil
	}

	cli.PrintTitle("Disabling Elasticvue")
	fmt.Println()

	// Stop container first
	fmt.Print("Stopping Elasticvue container... ")
	composeGen := docker.NewComposeGenerator(p)
	dockerCtrl := docker.NewDockerController(composeGen.ComposeFilePath())
	if err := dockerCtrl.StopService("elasticvue"); err != nil {
		fmt.Println(cli.Warning("not running"))
	} else {
		fmt.Println(cli.Success("done"))
	}

	globalCfg.Elasticvue = false
	if err := config.SaveGlobalConfig(p.HomeDir, globalCfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Regenerate docker-compose without Elasticvue
	configs := discoverAllConfigs(p)
	if err := composeGen.GenerateGlobalServices(configs); err != nil {
		return fmt.Errorf("failed to update docker-compose: %w", err)
	}

	fmt.Println()
	cli.PrintSuccess("Elasticvue disabled!")

	return nil
}

func runElasticvueStatus(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	globalCfg, err := config.LoadGlobalConfig(p.HomeDir)
	if err != nil {
		return fmt.Errorf("failed to load global config: %w", err)
	}

	cli.PrintTitle("Elasticvue Status")
	fmt.Println()

	if !globalCfg.Elasticvue {
		fmt.Println("Enabled: " + cli.Warning("no"))
		fmt.Println()
		cli.PrintInfo("Enable with: magebox elasticvue enable")
		return nil
	}

	fmt.Println("Enabled: " + cli.Success("yes"))

	// Check if container is running
	checkCmd := exec.Command("docker", "ps", "--filter", "name=magebox-elasticvue", "--filter", "status=running", "-q")
	output, err := checkCmd.Output()
	if err == nil && len(strings.TrimSpace(string(output))) > 0 {
		fmt.Println("Status:  " + cli.Success("running"))
		fmt.Printf("Web UI:  %s\n", cli.Highlight("http://localhost:8080"))
	} else {
		fmt.Println("Status:  " + cli.Warning("stopped"))
		fmt.Println()
		cli.PrintInfo("Start global services with: magebox global start")
	}

	return nil
}

// discoverAllConfigs loads configs from all registered MageBox projects
func discoverAllConfigs(p *platform.Platform) []*config.Config {
	var configs []*config.Config

	discovery := project.NewProjectDiscovery(p)
	projects, err := discovery.DiscoverProjects()
	if err != nil {
		return configs
	}

	for _, proj := range projects {
		if !proj.HasConfig {
			continue
		}
		projCfg, err := config.LoadFromPath(proj.Path)
		if err != nil {
			continue
		}
		configs = append(configs, projCfg)
	}

	return configs
}

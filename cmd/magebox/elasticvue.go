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

const (
	elasticvueDefaultPort = "8090"
	elasticvueContainer   = "magebox-elasticvue"
	elasticvueService     = "elasticvue"
)

// getElasticvueURL returns the Elasticvue URL, reading the actual port from the running container.
// Falls back to the default port if the container is not running.
func getElasticvueURL() string {
	portCmd := exec.Command("docker", "port", elasticvueContainer, "8080")
	output, err := portCmd.Output()
	if err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
			line = strings.TrimSpace(line)
			if idx := strings.LastIndex(line, ":"); idx != -1 {
				port := line[idx+1:]
				return fmt.Sprintf("http://localhost:%s", port)
			}
		}
	}
	return fmt.Sprintf("http://localhost:%s", elasticvueDefaultPort)
}

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

var elasticvueOpenCmd = &cobra.Command{
	Use:   "open",
	Short: "Open Elasticvue in browser",
	Long:  "Opens the Elasticvue web UI in the default browser, starting it if needed",
	RunE:  runElasticvueOpen,
}

func init() {
	elasticvueCmd.AddCommand(elasticvueEnableCmd)
	elasticvueCmd.AddCommand(elasticvueDisableCmd)
	elasticvueCmd.AddCommand(elasticvueStatusCmd)
	elasticvueCmd.AddCommand(elasticvueOpenCmd)
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

	switch decideServiceUI(globalCfg.Elasticvue, isContainerRunning(elasticvueContainer)) {
	case decisionProceed:
		cli.PrintInfo("Elasticvue is already enabled and running")
		fmt.Println()
		fmt.Printf("  Web UI: %s\n", cli.Highlight(getElasticvueURL()))
		return nil

	case decisionStart:
		cli.PrintTitle("Starting Elasticvue")
		fmt.Println()
		fmt.Print("Elasticvue is enabled but stopped, starting... ")
		if err := ensureGlobalServiceRunning(p, elasticvueService); err != nil {
			fmt.Println(cli.Error("failed"))
			cli.PrintError("%v", err)
			return nil
		}
		fmt.Println(cli.Success("done"))
		fmt.Println()
		cli.PrintSuccess("Elasticvue started!")
		fmt.Println()
		fmt.Printf("  Web UI: %s\n", cli.Highlight(getElasticvueURL()))
		return nil

	default: // decisionNotEnabled
		cli.PrintTitle("Enabling Elasticvue")
		fmt.Println()

		globalCfg.Elasticvue = true
		if err := config.SaveGlobalConfig(p.HomeDir, globalCfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Print("Starting Elasticvue container... ")
		if err := ensureGlobalServiceRunning(p, elasticvueService); err != nil {
			fmt.Println(cli.Error("failed"))
			cli.PrintError("%v", err)
			return nil
		}
		fmt.Println(cli.Success("done"))

		fmt.Println()
		cli.PrintSuccess("Elasticvue enabled!")
		fmt.Println()
		fmt.Printf("  Web UI: %s\n", cli.Highlight(getElasticvueURL()))
		fmt.Println()
		cli.PrintInfo("Add your cluster in the Elasticvue UI using http://localhost:<port>")
		cli.PrintInfo("Check 'magebox status' or the ports reference for your search engine's port")
		return nil
	}
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
	if err := dockerCtrl.StopService(elasticvueService); err != nil {
		fmt.Println(cli.Warning("not running"))
	} else {
		fmt.Println(cli.Success("done"))
	}

	globalCfg.Elasticvue = false
	if err := config.SaveGlobalConfig(p.HomeDir, globalCfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Regenerate docker-compose without Elasticvue
	if err := composeGen.GenerateGlobalServices(discoverAllConfigs(p)); err != nil {
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

	if isContainerRunning(elasticvueContainer) {
		fmt.Println("Status:  " + cli.Success("running"))
		fmt.Printf("Web UI:  %s\n", cli.Highlight(getElasticvueURL()))
	} else {
		fmt.Println("Status:  " + cli.Warning("stopped"))
		fmt.Println()
		cli.PrintInfo("Start with: magebox elasticvue open")
	}

	return nil
}

func runElasticvueOpen(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	globalCfg, err := config.LoadGlobalConfig(p.HomeDir)
	if err != nil {
		return fmt.Errorf("failed to load global config: %w", err)
	}

	switch decideServiceUI(globalCfg.Elasticvue, isContainerRunning(elasticvueContainer)) {
	case decisionNotEnabled:
		cli.PrintError("Elasticvue is not enabled")
		fmt.Println()
		cli.PrintInfo("Enable with: magebox elasticvue enable")
		return nil

	case decisionStart:
		fmt.Print("Elasticvue is not running, starting... ")
		if err := ensureGlobalServiceRunning(p, elasticvueService); err != nil {
			fmt.Println(cli.Error("failed"))
			cli.PrintError("%v", err)
			return nil
		}
		fmt.Println(cli.Success("done"))
	}

	url := getElasticvueURL()
	cli.PrintInfo("Opening %s", cli.URL(url))
	return openInBrowser(url)
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

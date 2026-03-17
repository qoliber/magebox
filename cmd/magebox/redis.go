package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"qoliber/magebox/internal/cli"
	"qoliber/magebox/internal/config"
	"qoliber/magebox/internal/docker"
)

var redisCmd = &cobra.Command{
	Use:   "redis",
	Short: "Redis/Valkey operations",
	Long:  "Redis/Valkey cache management commands",
}

var redisFlushCmd = &cobra.Command{
	Use:   "flush",
	Short: "Flush cache",
	Long:  "Flushes all data from the Redis/Valkey cache",
	RunE:  runRedisFlush,
}

var redisShellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Open cache shell",
	Long:  "Opens a Redis/Valkey CLI shell",
	RunE:  runRedisShell,
}

var redisInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show cache info",
	Long:  "Shows Redis/Valkey server information and statistics",
	RunE:  runRedisInfo,
}

func init() {
	redisCmd.AddCommand(redisFlushCmd)
	redisCmd.AddCommand(redisShellCmd)
	redisCmd.AddCommand(redisInfoCmd)
	rootCmd.AddCommand(redisCmd)
}

// getCacheServiceInfo returns the compose service name, CLI binary, and display name
// based on which cache service is configured for the project.
func getCacheServiceInfo(cfg *config.Config) (serviceName, cliBinary, displayName string, ok bool) {
	if cfg.Services.HasValkey() {
		return "valkey", "valkey-cli", "Valkey", true
	}
	if cfg.Services.HasRedis() {
		return "redis", "redis-cli", "Redis", true
	}
	return "", "", "", false
}

func runRedisFlush(cmd *cobra.Command, args []string) error {
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

	serviceName, cliBinary, displayName, hasCache := getCacheServiceInfo(cfg)
	if !hasCache {
		cli.PrintError("Neither Redis nor Valkey is configured in %s", config.ConfigFileName)
		return nil
	}

	composeGen := docker.NewComposeGenerator(p)
	composeFile := composeGen.ComposeFilePath()

	cli.PrintInfo("Flushing %s cache...", displayName)

	flushCmd := docker.BuildComposeCmd(composeFile, "exec", "-T", serviceName, cliBinary, "FLUSHALL")
	output, err := flushCmd.CombinedOutput()
	if err != nil {
		cli.PrintError("Failed to flush %s: %v", displayName, err)
		return nil
	}

	if strings.TrimSpace(string(output)) == "OK" {
		cli.PrintSuccess("%s cache flushed successfully", displayName)
	} else {
		fmt.Println(string(output))
	}

	return nil
}

func runRedisShell(cmd *cobra.Command, args []string) error {
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

	serviceName, cliBinary, displayName, hasCache := getCacheServiceInfo(cfg)
	if !hasCache {
		cli.PrintError("Neither Redis nor Valkey is configured in %s", config.ConfigFileName)
		return nil
	}

	composeGen := docker.NewComposeGenerator(p)
	composeFile := composeGen.ComposeFilePath()

	cli.PrintInfo("Connecting to %s...", displayName)
	fmt.Println()

	shellCmd := docker.BuildComposeCmd(composeFile, "exec", serviceName, cliBinary)
	shellCmd.Stdin = os.Stdin
	shellCmd.Stdout = os.Stdout
	shellCmd.Stderr = os.Stderr

	return shellCmd.Run()
}

func runRedisInfo(cmd *cobra.Command, args []string) error {
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

	serviceName, cliBinary, displayName, hasCache := getCacheServiceInfo(cfg)
	if !hasCache {
		cli.PrintError("Neither Redis nor Valkey is configured in %s", config.ConfigFileName)
		return nil
	}

	composeGen := docker.NewComposeGenerator(p)
	composeFile := composeGen.ComposeFilePath()

	cli.PrintTitle("%s Information", displayName)
	fmt.Println()

	infoCmd := docker.BuildComposeCmd(composeFile, "exec", "-T", serviceName, cliBinary, "INFO")
	output, err := infoCmd.Output()
	if err != nil {
		cli.PrintError("Failed to get %s info: %v", displayName, err)
		return nil
	}

	// Parse and display key info
	lines := strings.Split(string(output), "\n")
	sections := []string{"Server", "Memory", "Stats", "Keyspace"}
	currentSection := ""

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Section header
		if strings.HasPrefix(line, "#") {
			sectionName := strings.TrimPrefix(line, "# ")
			for _, s := range sections {
				if sectionName == s {
					currentSection = sectionName
					fmt.Println(cli.Header(sectionName))
					break
				}
			}
			continue
		}

		// Only show lines from selected sections
		if currentSection == "" {
			continue
		}

		// Parse key:value
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := parts[0]
			value := parts[1]

			// Highlight important values
			switch key {
			case "redis_version", "valkey_version", "used_memory_human", "connected_clients",
				"total_connections_received", "total_commands_processed",
				"keyspace_hits", "keyspace_misses":
				fmt.Printf("  %s: %s\n", key, cli.Highlight(value))
			default:
				if currentSection == "Keyspace" {
					fmt.Printf("  %s: %s\n", key, cli.Highlight(value))
				}
			}
		}
	}

	return nil
}

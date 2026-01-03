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
	Short: "Redis operations",
	Long:  "Redis cache management commands",
}

var redisFlushCmd = &cobra.Command{
	Use:   "flush",
	Short: "Flush Redis cache",
	Long:  "Flushes all data from the Redis cache",
	RunE:  runRedisFlush,
}

var redisShellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Open Redis shell",
	Long:  "Opens a Redis CLI shell",
	RunE:  runRedisShell,
}

var redisInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show Redis info",
	Long:  "Shows Redis server information and statistics",
	RunE:  runRedisInfo,
}

func init() {
	redisCmd.AddCommand(redisFlushCmd)
	redisCmd.AddCommand(redisShellCmd)
	redisCmd.AddCommand(redisInfoCmd)
	rootCmd.AddCommand(redisCmd)
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

	if !cfg.Services.HasRedis() {
		cli.PrintError("Redis is not configured in %s", config.ConfigFileName)
		return nil
	}

	composeGen := docker.NewComposeGenerator(p)
	composeFile := composeGen.ComposeFilePath()

	cli.PrintInfo("Flushing Redis cache...")

	// Run redis-cli FLUSHALL
	flushCmd := docker.BuildComposeCmd(composeFile, "exec", "-T", "redis", "redis-cli", "FLUSHALL")
	output, err := flushCmd.CombinedOutput()
	if err != nil {
		cli.PrintError("Failed to flush Redis: %v", err)
		return nil
	}

	if strings.TrimSpace(string(output)) == "OK" {
		cli.PrintSuccess("Redis cache flushed successfully")
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

	if !cfg.Services.HasRedis() {
		cli.PrintError("Redis is not configured in %s", config.ConfigFileName)
		return nil
	}

	composeGen := docker.NewComposeGenerator(p)
	composeFile := composeGen.ComposeFilePath()

	cli.PrintInfo("Connecting to Redis...")
	fmt.Println()

	// Open interactive redis-cli
	shellCmd := docker.BuildComposeCmd(composeFile, "exec", "redis", "redis-cli")
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

	if !cfg.Services.HasRedis() {
		cli.PrintError("Redis is not configured in %s", config.ConfigFileName)
		return nil
	}

	composeGen := docker.NewComposeGenerator(p)
	composeFile := composeGen.ComposeFilePath()

	cli.PrintTitle("Redis Information")
	fmt.Println()

	// Get Redis info
	infoCmd := docker.BuildComposeCmd(composeFile, "exec", "-T", "redis", "redis-cli", "INFO")
	output, err := infoCmd.Output()
	if err != nil {
		cli.PrintError("Failed to get Redis info: %v", err)
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
			case "redis_version", "used_memory_human", "connected_clients",
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

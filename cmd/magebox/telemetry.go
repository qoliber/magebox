package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"qoliber/magebox/internal/cli"
	"qoliber/magebox/internal/config"
	"qoliber/magebox/internal/telemetry"
)

var telemetryCmd = &cobra.Command{
	Use:   "telemetry",
	Short: "Manage anonymous usage telemetry",
	Long: `Manage MageBox's opt-in anonymous usage telemetry.

Telemetry is disabled by default. When enabled, MageBox records the command
name, exit code, duration, version, OS and architecture of each invocation,
and sends the batch to the public ingestion server at telemetry.magebox.dev.

No arguments, flag values, paths, hostnames or project names are collected.
See https://telemetry.magebox.dev for the full schema and public dashboard.`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var telemetryEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable anonymous usage telemetry",
	RunE:  runTelemetryEnable,
}

var telemetryDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable anonymous usage telemetry",
	RunE:  runTelemetryDisable,
}

var telemetryStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show telemetry status",
	RunE:  runTelemetryStatus,
}

var telemetryShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show the last events recorded locally",
	Long: `Show the last events MageBox has recorded locally. This is the exact data
that would be (or has been) sent to the ingestion server.

Use this to verify what telemetry contains before you enable it.`,
	RunE: runTelemetryShow,
}

var telemetryResetIDCmd = &cobra.Command{
	Use:   "reset-id",
	Short: "Generate a new anonymous installation ID",
	RunE:  runTelemetryResetID,
}

var telemetryPurgeCmd = &cobra.Command{
	Use:   "purge",
	Short: "Remove all local telemetry state (id, spool, log)",
	RunE:  runTelemetryPurge,
}

func init() {
	telemetryCmd.AddCommand(
		telemetryEnableCmd,
		telemetryDisableCmd,
		telemetryStatusCmd,
		telemetryShowCmd,
		telemetryResetIDCmd,
		telemetryPurgeCmd,
	)
	rootCmd.AddCommand(telemetryCmd)
}

func runTelemetryEnable(cmd *cobra.Command, args []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	cfg, err := config.LoadGlobalConfig(homeDir)
	if err != nil {
		cli.PrintError("Failed to load config: %v", err)
		return nil
	}
	if cfg.Telemetry == nil {
		cfg.Telemetry = &config.TelemetryConfig{}
	}
	cfg.Telemetry.Enabled = true
	cfg.Telemetry.Prompted = true
	if err := config.SaveGlobalConfig(homeDir, cfg); err != nil {
		cli.PrintError("Failed to save config: %v", err)
		return nil
	}
	cli.PrintSuccess("Telemetry enabled")
	cli.PrintInfo("Review events locally with %s", cli.Command("magebox telemetry show"))
	cli.PrintInfo("Public dashboard: https://telemetry.magebox.dev")
	return nil
}

func runTelemetryDisable(cmd *cobra.Command, args []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	cfg, err := config.LoadGlobalConfig(homeDir)
	if err != nil {
		cli.PrintError("Failed to load config: %v", err)
		return nil
	}
	if cfg.Telemetry == nil {
		cfg.Telemetry = &config.TelemetryConfig{}
	}
	cfg.Telemetry.Enabled = false
	cfg.Telemetry.Prompted = true
	if err := config.SaveGlobalConfig(homeDir, cfg); err != nil {
		cli.PrintError("Failed to save config: %v", err)
		return nil
	}
	cli.PrintSuccess("Telemetry disabled")
	cli.PrintInfo("Purge local telemetry state with %s", cli.Command("magebox telemetry purge"))
	return nil
}

func runTelemetryStatus(cmd *cobra.Command, args []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	cfg, err := config.LoadGlobalConfig(homeDir)
	if err != nil {
		cli.PrintError("Failed to load config: %v", err)
		return nil
	}

	cli.PrintTitle("MageBox Telemetry")
	fmt.Println()

	enabled := cfg.Telemetry != nil && cfg.Telemetry.Enabled
	state := "disabled"
	if enabled {
		state = "enabled"
	}
	fmt.Printf("  %-12s %s\n", "status:", cli.Highlight(state))

	endpoint := telemetry.DefaultEndpoint
	if cfg.Telemetry != nil && cfg.Telemetry.Endpoint != "" {
		endpoint = cfg.Telemetry.Endpoint
	}
	fmt.Printf("  %-12s %s\n", "endpoint:", cli.Highlight(endpoint))

	anonID := telemetry.ReadAnonID(homeDir)
	if anonID == "" {
		anonID = "(not yet generated)"
	}
	fmt.Printf("  %-12s %s\n", "anon_id:", cli.Highlight(anonID))

	fmt.Println()
	cli.PrintInfo("Review recent events: %s", cli.Command("magebox telemetry show"))
	return nil
}

func runTelemetryShow(cmd *cobra.Command, args []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	events, err := telemetry.ReadLog(homeDir)
	if err != nil {
		cli.PrintError("Failed to read telemetry log: %v", err)
		return nil
	}

	cli.PrintTitle("Local telemetry log")
	fmt.Println()
	fmt.Println("This is the exact payload MageBox has recorded locally. Every field")
	fmt.Println("shown here is everything that would be (or has been) sent.")
	fmt.Println()

	if len(events) == 0 {
		cli.PrintInfo("No events recorded yet.")
		return nil
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	for _, ev := range events {
		if err := enc.Encode(ev); err != nil {
			return err
		}
	}
	fmt.Println()
	cli.PrintInfo("%d event(s) in local log", len(events))
	return nil
}

func runTelemetryResetID(cmd *cobra.Command, args []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	id, err := telemetry.ResetAnonID(homeDir)
	if err != nil {
		cli.PrintError("Failed to reset anonymous ID: %v", err)
		return nil
	}
	cli.PrintSuccess("New anonymous ID: %s", id)
	return nil
}

func runTelemetryPurge(cmd *cobra.Command, args []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	if err := telemetry.Purge(homeDir); err != nil {
		cli.PrintError("Failed to purge telemetry state: %v", err)
		return nil
	}
	cli.PrintSuccess("Local telemetry state removed")
	return nil
}

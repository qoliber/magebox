// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"qoliber/magebox/internal/cli"
	"qoliber/magebox/internal/portforward"
)

var doctorAutoHeal bool

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose and repair MageBox environment issues",
	Long: `Diagnose and repair common issues with the MageBox environment.

Currently checks:
  - macOS port forwarding (pf rules, LaunchDaemon, rules active)

With --heal, attempts to automatically repair any issues found.

This is especially useful after a macOS reboot or update when pf rules
may have been cleared or /etc/pf.conf reset.`,
	RunE: runDoctor,
}

func init() {
	doctorCmd.Flags().BoolVar(&doctorAutoHeal, "heal", false, "Automatically repair issues found")
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) error {
	cli.PrintTitle("MageBox Doctor")
	fmt.Println()

	if runtime.GOOS != "darwin" {
		cli.PrintInfo("No macOS-specific checks to perform on %s", runtime.GOOS)
		cli.PrintSuccess("All good!")
		return nil
	}

	return checkPortForwarding()
}

func checkPortForwarding() error {
	fmt.Println(cli.Header("Port Forwarding (macOS pf)"))
	fmt.Println()

	pfMgr := portforward.NewManager()
	report := pfMgr.Diagnose()

	printCheck("PF rules file exists", report.PfRulesFileExists)
	printCheck("PF rules file is valid", report.PfRulesFileValid)
	printCheck("LaunchDaemon plist exists", report.LaunchDaemonExists)
	if report.LaunchDaemonVersion != "" {
		fmt.Printf("  %s LaunchDaemon version: %s\n", cli.Bullet(""), cli.Highlight(report.LaunchDaemonVersion))
	}
	printCheck("LaunchDaemon is loaded", report.LaunchDaemonLoaded)
	printCheck("pf is enabled", report.PfEnabled)
	printCheck("MageBox pf rules are active", report.RulesActive)

	fmt.Println()

	if len(report.Issues) == 0 {
		cli.PrintSuccess("Port forwarding is healthy!")
		return nil
	}

	cli.PrintWarning("Found %d issue(s):", len(report.Issues))
	for _, issue := range report.Issues {
		fmt.Printf("  • %s\n", issue)
	}
	fmt.Println()

	if !doctorAutoHeal {
		cli.PrintInfo("Run with --heal to attempt automatic repair:")
		fmt.Println("  magebox doctor --heal")
		return nil
	}

	cli.PrintInfo("Attempting repair...")
	fmt.Println()

	actions, err := pfMgr.Heal()
	for _, action := range actions {
		fmt.Printf("  %s %s\n", cli.Success("✓"), action)
	}

	if err != nil {
		fmt.Println()
		cli.PrintError("Heal failed: %v", err)
		return err
	}

	fmt.Println()

	// Re-check after healing
	report = pfMgr.Diagnose()
	if len(report.Issues) == 0 {
		cli.PrintSuccess("All issues repaired!")
	} else {
		cli.PrintWarning("Some issues remain:")
		for _, issue := range report.Issues {
			fmt.Printf("  • %s\n", issue)
		}
	}

	return nil
}

func printCheck(name string, ok bool) {
	if ok {
		fmt.Printf("  %s %s\n", cli.Success("✓"), name)
	} else {
		fmt.Printf("  %s %s\n", cli.Error("✗"), name)
	}
}

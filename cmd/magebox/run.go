package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/qoliber/magebox/internal/config"
	"github.com/qoliber/magebox/internal/php"
)

var runCmd = &cobra.Command{
	Use:   "run <command>",
	Short: "Run custom project command",
	Long: `Runs a custom command defined in .magebox file.

Example .magebox commands:

  commands:
    deploy: "php bin/magento deploy:mode:set production"
    reindex:
      description: "Reindex all Magento indexes"
      run: "php bin/magento indexer:reindex"
    setup:
      description: "Run full setup"
      run: "php bin/magento setup:upgrade && php bin/magento cache:flush"

Then run with: magebox run deploy`,
	Args:              cobra.MinimumNArgs(1),
	RunE:              runCustomCommand,
	ValidArgsFunction: completeCustomCommands,
}

func init() {
	rootCmd.AddCommand(runCmd)
}

func runCustomCommand(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	p, err := getPlatform()
	if err != nil {
		return err
	}

	cfg, cfgOk := loadProjectConfig(cwd)
	if !cfgOk {
		return nil
	}

	cmdName := args[0]

	// Check if command exists
	command, ok := cfg.Commands[cmdName]
	if !ok {
		// List available commands
		fmt.Printf("Command '%s' not found in .magebox\n\n", cmdName)
		if len(cfg.Commands) > 0 {
			fmt.Println("Available commands:")
			for name, cmd := range cfg.Commands {
				if cmd.Description != "" {
					fmt.Printf("  %-15s %s\n", name, cmd.Description)
				} else {
					fmt.Printf("  %s\n", name)
				}
			}
		} else {
			fmt.Printf("No commands defined in %s\n", config.ConfigFileName)
			fmt.Println()
			fmt.Printf("Add commands to your %s file:\n", config.ConfigFileName)
			fmt.Println()
			fmt.Println("  commands:")
			fmt.Println("    deploy: \"php bin/magento deploy:mode:set production\"")
			fmt.Println("    reindex:")
			fmt.Println("      description: \"Reindex all Magento indexes\"")
			fmt.Println("      run: \"php bin/magento indexer:reindex\"")
		}
		return nil
	}

	// Get PHP path
	detector := php.NewDetector(p)
	version := detector.Detect(cfg.PHP)
	if !version.Installed {
		return fmt.Errorf("PHP %s is not installed", cfg.PHP)
	}

	// Build command to run
	cmdToRun := command.Run

	// Append any additional arguments passed after the command name
	if len(args) > 1 {
		cmdToRun = cmdToRun + " " + strings.Join(args[1:], " ")
	}

	// Set up environment with correct PHP in PATH
	phpDir := filepath.Dir(version.PHPBinary)
	currentPath := os.Getenv("PATH")
	newPath := phpDir + string(os.PathListSeparator) + currentPath

	fmt.Printf("Running: %s\n\n", cmdToRun)

	// Execute command via shell
	shellCmd := exec.Command("bash", "-c", cmdToRun)
	shellCmd.Dir = cwd
	shellCmd.Stdin = os.Stdin
	shellCmd.Stdout = os.Stdout
	shellCmd.Stderr = os.Stderr
	shellCmd.Env = append(os.Environ(), "PATH="+newPath)

	// Add project env vars
	for key, value := range cfg.Env {
		shellCmd.Env = append(shellCmd.Env, key+"="+value)
	}

	return shellCmd.Run()
}

func completeCustomCommands(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	cwd, err := getCwd()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	cfg, err := config.LoadFromPath(cwd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []string
	for name, command := range cfg.Commands {
		if strings.HasPrefix(name, toComplete) {
			if command.Description != "" {
				completions = append(completions, name+"\t"+command.Description)
			} else {
				completions = append(completions, name)
			}
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

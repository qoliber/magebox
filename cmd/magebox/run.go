package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"qoliber/magebox/internal/config"
	"qoliber/magebox/internal/php"
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
	RunE:              runCustomCommand,
	ValidArgsFunction: completeCustomCommands,
}

var runListFlag bool

func init() {
	runCmd.Flags().BoolVarP(&runListFlag, "list", "l", false, "List available commands without interactive selection")
	rootCmd.AddCommand(runCmd)
}

func selectCommand(cfg *config.Config) (string, error) {
	if len(cfg.Commands) == 0 {
		fmt.Printf("No commands defined in %s\n", config.ConfigFileName)
		fmt.Println()
		fmt.Printf("Add commands to your %s file:\n", config.ConfigFileName)
		fmt.Println()
		fmt.Println("  commands:")
		fmt.Println("    deploy: \"php bin/magento deploy:mode:set production\"")
		fmt.Println("    reindex:")
		fmt.Println("      description: \"Reindex all Magento indexes\"")
		fmt.Println("      run: \"php bin/magento indexer:reindex\"")
		return "", fmt.Errorf("no commands available")
	}

	// Sort command names for stable ordering
	names := make([]string, 0, len(cfg.Commands))
	for name := range cfg.Commands {
		names = append(names, name)
	}
	sort.Strings(names)

	// Build select options
	options := make([]huh.Option[string], 0, len(names))
	for _, name := range names {
		label := name
		if cfg.Commands[name].Description != "" {
			label = name + " — " + cfg.Commands[name].Description
		}
		options = append(options, huh.NewOption(label, name))
	}

	var selected string
	err := huh.NewSelect[string]().
		Title("Select a command to run").
		Options(options...).
		Value(&selected).
		Run()
	if err != nil {
		return "", err
	}

	return selected, nil
}

func listAvailableCommands(cfg *config.Config) {
	if len(cfg.Commands) == 0 {
		fmt.Printf("No commands defined in %s\n", config.ConfigFileName)
		return
	}

	names := make([]string, 0, len(cfg.Commands))
	for name := range cfg.Commands {
		names = append(names, name)
	}
	sort.Strings(names)

	fmt.Println("Available commands:")
	for _, name := range names {
		if cfg.Commands[name].Description != "" {
			fmt.Printf("  %-15s %s\n", name, cfg.Commands[name].Description)
		} else {
			fmt.Printf("  %s\n", name)
		}
	}
}

func runCustomCommand(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	cfg, cfgOk := loadProjectConfig(cwd)
	if !cfgOk {
		return nil
	}

	if runListFlag {
		listAvailableCommands(cfg)
		return nil
	}

	// No arguments: show interactive selector
	if len(args) == 0 {
		selected, err := selectCommand(cfg)
		if err != nil {
			return err
		}
		args = []string{selected}
	}

	p, err := getPlatform()
	if err != nil {
		return err
	}

	cmdName := args[0]

	// Check if command exists
	command, ok := cfg.Commands[cmdName]
	if !ok {
		fmt.Printf("Command '%s' not found in .magebox\n\n", cmdName)
		selected, selectErr := selectCommand(cfg)
		if selectErr != nil {
			return selectErr
		}
		args = []string{selected}
		cmdName = selected
		command = cfg.Commands[cmdName]
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

	// Set up environment with correct PHP in PATH.
	// Prepend ~/.magebox/bin (where the php/composer/blackfire wrappers live)
	// so `php` resolves to the project-aware wrapper instead of a system PHP.
	// On Linux, filepath.Dir(PHPBinary) is /usr/bin, which would shadow the
	// wrapper and silently run the wrong PHP version.
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to determine home directory: %w", err)
	}
	wrapperDir := filepath.Join(homeDir, ".magebox", "bin")
	currentPath := os.Getenv("PATH")
	newPath := wrapperDir + string(os.PathListSeparator) + currentPath

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

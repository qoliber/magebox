package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/qoliber/magebox/internal/cli"
	"github.com/qoliber/magebox/internal/config"
	"github.com/qoliber/magebox/internal/project"
)

var initCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Initialize a new MageBox project",
	Long:  "Creates a .magebox configuration file in the current directory",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	// Determine project name
	var projectName string
	if len(args) > 0 {
		projectName = args[0]
	} else {
		// Use directory name
		projectName = filepath.Base(cwd)
		// Prompt for confirmation
		fmt.Printf("Project name [%s]: ", cli.Highlight(projectName))
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			projectName = input
		}
	}

	// Check if .magebox.yaml already exists
	configPath := filepath.Join(cwd, config.ConfigFileName)
	if _, err := os.Stat(configPath); err == nil {
		cli.PrintError("%s file already exists", config.ConfigFileName)
		return nil
	}

	p, err := getPlatform()
	if err != nil {
		return err
	}

	mgr := project.NewManager(p)
	if err := mgr.Init(cwd, projectName); err != nil {
		return err
	}

	// Get configured TLD
	homeDir, _ := os.UserHomeDir()
	globalCfg, _ := config.LoadGlobalConfig(homeDir)
	tld := globalCfg.GetTLD()

	cli.PrintSuccess("Created %s for project '%s'", config.ConfigFileName, projectName)
	fmt.Println()
	fmt.Printf("Domain: %s\n", cli.URL(projectName+"."+tld))
	fmt.Println()
	cli.PrintInfo("Next steps:")
	fmt.Println(cli.Bullet("Edit " + config.ConfigFileName + " to customize your configuration"))
	fmt.Println(cli.Bullet("Run " + cli.Command("magebox start") + " to start your project"))

	return nil
}

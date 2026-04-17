package main

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"

	"qoliber/magebox/internal/cli"
	"qoliber/magebox/internal/project"
)

var openCmd = &cobra.Command{
	Use:   "open",
	Short: "Open project in browser",
	Long:  "Starts the project if not already running, then opens the first domain from .magebox.yaml in the default browser",
	RunE:  runOpen,
}

func init() {
	rootCmd.AddCommand(openCmd)
}

func runOpen(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	cfg, ok := loadProjectConfig(cwd)
	if !ok {
		return nil
	}

	if len(cfg.Domains) == 0 {
		cli.PrintError("No domains configured in .magebox.yaml")
		return nil
	}

	p, err := getPlatform()
	if err != nil {
		return err
	}

	mgr := project.NewManager(p)
	if !projectFullyRunning(mgr, cwd) {
		if err := startProject(mgr, cwd, true); err != nil {
			return err
		}
		fmt.Println()
	}

	domain := cfg.Domains[0]
	scheme := "https"
	if !domain.IsSSLEnabled() {
		scheme = "http"
	}
	url := fmt.Sprintf("%s://%s", scheme, domain.Host)

	cli.PrintInfo("Opening %s", cli.URL(url))

	var openCmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		openCmd = exec.Command("open", url)
	default:
		openCmd = exec.Command("xdg-open", url)
	}

	return openCmd.Start()
}

// projectFullyRunning reports whether all essential services for the project
// are running. Optional debug tooling (xdebug, blackfire) is ignored: those
// being disabled is a normal state and should not trigger a start.
func projectFullyRunning(mgr *project.Manager, projectPath string) bool {
	status, err := mgr.Status(projectPath)
	if err != nil {
		return false
	}
	for name, svc := range status.Services {
		if name == "xdebug" || name == "blackfire" {
			continue
		}
		if !svc.IsRunning {
			return false
		}
	}
	return true
}

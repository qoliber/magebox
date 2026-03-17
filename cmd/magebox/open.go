package main

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"

	"qoliber/magebox/internal/cli"
)

var openCmd = &cobra.Command{
	Use:   "open",
	Short: "Open project in browser",
	Long:  "Opens the first domain from .magebox.yaml in the default browser",
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

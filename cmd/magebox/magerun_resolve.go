package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"qoliber/magebox/internal/magerunwrapper"
)

var magerunResolveProjectDir string

var magerunResolveCmd = &cobra.Command{
	Use:    "magerun-resolve",
	Hidden: true,
	Short:  "Resolve and ensure the correct n98-magerun2 phar is available",
	RunE:   runMagerunResolve,
}

func init() {
	magerunResolveCmd.Flags().StringVar(&magerunResolveProjectDir, "project-dir", "", "Project directory containing composer.lock")
	rootCmd.AddCommand(magerunResolveCmd)
}

func runMagerunResolve(_ *cobra.Command, _ []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	mgr := magerunwrapper.NewManager(p)
	pharPath, err := mgr.Resolve(magerunResolveProjectDir)
	if err != nil {
		return err
	}

	fmt.Println(pharPath)
	return nil
}

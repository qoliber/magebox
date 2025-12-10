/**
 * Created by Qoliber
 *
 * @category    Qoliber
 * @package     Qoliber_MageBox
 * @author      Jakub Winkler <jwinkler@qoliber.com>
 */

package main

import (
	"fmt"
	"os"

	"github.com/qoliber/magebox/internal/cli"
	"github.com/qoliber/magebox/internal/config"
	"github.com/qoliber/magebox/internal/platform"
)

// getPlatform returns the current platform
func getPlatform() (*platform.Platform, error) {
	return platform.Detect()
}

// getCwd returns the current working directory
func getCwd() (string, error) {
	return os.Getwd()
}

// loadProjectConfig loads the project config and handles errors nicely
// Returns (config, shouldContinue) - if shouldContinue is false, the command should return nil
func loadProjectConfig(cwd string) (*config.Config, bool) {
	cfg, err := config.LoadFromPath(cwd)
	if err != nil {
		// Check if it's a "not found" error - print nicely and don't return error
		if _, ok := err.(*config.ConfigNotFoundError); ok {
			cli.PrintError("Configuration file not found: %s/.magebox.yaml", cwd)
			fmt.Println()
			cli.PrintInfo("Run " + cli.Command("magebox init") + " to create one")
			return nil, false
		}
		// Other errors - print and return
		cli.PrintError("%v", err)
		return nil, false
	}
	return cfg, true
}

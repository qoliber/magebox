package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"qoliber/magebox/internal/docker"
	"qoliber/magebox/internal/platform"
)

// serviceUIDecision describes the action an enable/open command should take,
// based on whether the web-UI service is enabled and whether its container runs.
type serviceUIDecision int

const (
	// decisionNotEnabled: the service is not enabled in the global config.
	decisionNotEnabled serviceUIDecision = iota
	// decisionStart: the service is enabled but its container is not running.
	decisionStart
	// decisionProceed: the service is enabled and already running.
	decisionProceed
)

// decideServiceUI maps (enabled, running) to the action a web-UI command takes.
// Always-on services (e.g. Mailpit) pass enabled=true.
func decideServiceUI(enabled, running bool) serviceUIDecision {
	switch {
	case !enabled:
		return decisionNotEnabled
	case running:
		return decisionProceed
	default:
		return decisionStart
	}
}

// isContainerRunning reports whether a Docker container with the given name runs.
func isContainerRunning(name string) bool {
	cmd := exec.Command("docker", "ps", "--filter", "name="+name, "--filter", "status=running", "-q")
	output, err := cmd.Output()
	return err == nil && len(strings.TrimSpace(string(output))) > 0
}

// dockerRunning reports whether the Docker daemon is reachable.
func dockerRunning() bool {
	return exec.Command("docker", "info").Run() == nil
}

// openInBrowser opens the given URL in the default browser.
func openInBrowser(url string) error {
	var openCmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		openCmd = exec.Command("open", url)
	default:
		openCmd = exec.Command("xdg-open", url)
	}
	return openCmd.Start()
}

// ensureGlobalServiceRunning starts a single global web-UI service container.
//
// It regenerates the global compose file so the service is defined with the
// current configuration (also covering a missing compose file), then starts only
// that service. The web-UI services have no depends_on, so no databases are
// dragged in and no full stack is brought up. The Docker daemon is never started
// automatically: if it is not running, a clear, actionable error is returned.
func ensureGlobalServiceRunning(p *platform.Platform, service string) error {
	if !dockerRunning() {
		return fmt.Errorf("Docker is not running. Please start Docker first")
	}

	composeGen := docker.NewComposeGenerator(p)
	if err := composeGen.GenerateGlobalServices(discoverAllConfigs(p)); err != nil {
		return fmt.Errorf("failed to generate docker-compose: %w", err)
	}

	dockerCtrl := docker.NewDockerController(composeGen.ComposeFilePath())
	if err := dockerCtrl.StartService(service); err != nil {
		return fmt.Errorf("failed to start %s: %w", service, err)
	}
	return nil
}

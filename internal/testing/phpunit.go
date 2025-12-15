package testing

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/qoliber/magebox/internal/config"
)

// PHPUnitRunner handles PHPUnit test execution
type PHPUnitRunner struct {
	manager *Manager
	config  *config.PHPUnitTestConfig
}

// NewPHPUnitRunner creates a new PHPUnit runner
func NewPHPUnitRunner(m *Manager, cfg *config.PHPUnitTestConfig) *PHPUnitRunner {
	return &PHPUnitRunner{
		manager: m,
		config:  cfg,
	}
}

// RunUnit runs PHPUnit unit tests
func (r *PHPUnitRunner) RunUnit(filter string, testsuite string) error {
	if !r.manager.isComposerPackageInstalled("phpunit/phpunit") {
		return fmt.Errorf("PHPUnit is not installed. Run: magebox test setup")
	}

	args := r.buildArgs(filter, testsuite, false)
	return r.manager.StreamCommand("PHPUnit", args...)
}

// RunIntegration runs Magento integration tests
func (r *PHPUnitRunner) RunIntegration(filter string, testsuite string) error {
	if !r.manager.isComposerPackageInstalled("phpunit/phpunit") {
		return fmt.Errorf("PHPUnit is not installed. Run: magebox test setup")
	}

	args := r.buildArgs(filter, testsuite, true)
	return r.manager.StreamCommand("Integration Tests", args...)
}

// buildArgs builds the PHPUnit command arguments
func (r *PHPUnitRunner) buildArgs(filter string, testsuite string, integration bool) []string {
	phpBin := r.manager.GetPHPBinary()
	phpunitBin := filepath.Join(r.manager.GetVendorBinPath(), "phpunit")

	args := []string{phpBin, phpunitBin}

	// Determine config file
	configFile := r.getConfigFile(integration)
	if configFile != "" {
		args = append(args, "-c", configFile)
	}

	// Add testsuite if specified
	if testsuite != "" {
		args = append(args, "--testsuite", testsuite)
	} else if r.config != nil && r.config.TestSuite != "" && !integration {
		args = append(args, "--testsuite", r.config.TestSuite)
	}

	// Add filter if specified
	if filter != "" {
		args = append(args, "--filter", filter)
	}

	// Add colors for better output
	args = append(args, "--colors=always")

	return args
}

// getConfigFile returns the PHPUnit config file path
func (r *PHPUnitRunner) getConfigFile(integration bool) string {
	if integration {
		// Check for integration test config
		paths := []string{
			filepath.Join(r.manager.projectPath, "dev", "tests", "integration", "phpunit.xml"),
			filepath.Join(r.manager.projectPath, "dev", "tests", "integration", "phpunit.xml.dist"),
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
		return ""
	}

	// Check for config from yaml
	if r.config != nil && r.config.Config != "" {
		configPath := filepath.Join(r.manager.projectPath, r.config.Config)
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}
	}

	// Check for default config files
	paths := []string{
		filepath.Join(r.manager.projectPath, "phpunit.xml"),
		filepath.Join(r.manager.projectPath, "phpunit.xml.dist"),
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return ""
}

// GetAvailableTestSuites returns the available test suites from phpunit.xml
func (r *PHPUnitRunner) GetAvailableTestSuites() []string {
	// This is a simplified version - in a real implementation,
	// we would parse the phpunit.xml to extract test suite names
	return []string{"Unit", "Integration"}
}

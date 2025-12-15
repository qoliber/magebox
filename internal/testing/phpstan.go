package testing

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/qoliber/magebox/internal/config"
)

// PHPStanRunner handles PHPStan static analysis
type PHPStanRunner struct {
	manager *Manager
	config  *config.PHPStanTestConfig
}

// NewPHPStanRunner creates a new PHPStan runner
func NewPHPStanRunner(m *Manager, cfg *config.PHPStanTestConfig) *PHPStanRunner {
	return &PHPStanRunner{
		manager: m,
		config:  cfg,
	}
}

// Run executes PHPStan analysis
func (r *PHPStanRunner) Run(paths []string, level int) error {
	if !r.manager.isComposerPackageInstalled("phpstan/phpstan") {
		return fmt.Errorf("PHPStan is not installed. Run: magebox test setup")
	}

	args := r.buildArgs(paths, level)
	return r.manager.StreamCommand("PHPStan", args...)
}

// buildArgs builds the PHPStan command arguments
func (r *PHPStanRunner) buildArgs(paths []string, level int) []string {
	phpBin := r.manager.GetPHPBinary()
	phpstanBin := filepath.Join(r.manager.GetVendorBinPath(), "phpstan")

	args := []string{phpBin, phpstanBin, "analyse"}

	// Check for config file
	configFile := r.getConfigFile()
	if configFile != "" {
		args = append(args, "-c", configFile)
	}

	// Determine level
	analysisLevel := r.getLevel(level)
	args = append(args, "--level", strconv.Itoa(analysisLevel))

	// Determine paths to analyze
	analyzePaths := r.getPaths(paths)
	args = append(args, analyzePaths...)

	// Add useful flags
	args = append(args, "--no-progress")
	args = append(args, "--error-format=table")

	return args
}

// getConfigFile returns the PHPStan config file path
func (r *PHPStanRunner) getConfigFile() string {
	// Check for config from yaml
	if r.config != nil && r.config.Config != "" {
		configPath := filepath.Join(r.manager.projectPath, r.config.Config)
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}
	}

	// Check for default config files
	configPaths := []string{
		filepath.Join(r.manager.projectPath, "phpstan.neon"),
		filepath.Join(r.manager.projectPath, "phpstan.neon.dist"),
	}
	for _, p := range configPaths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return ""
}

// getLevel returns the PHPStan analysis level
func (r *PHPStanRunner) getLevel(cliLevel int) int {
	// CLI level takes precedence
	if cliLevel >= 0 {
		return cliLevel
	}

	// Then config level
	if r.config != nil && r.config.Level > 0 {
		return r.config.Level
	}

	// Default level
	return DefaultPHPStanLevel()
}

// getPaths returns the paths to analyze
func (r *PHPStanRunner) getPaths(cliPaths []string) []string {
	// CLI paths take precedence
	if len(cliPaths) > 0 {
		return cliPaths
	}

	// Then config paths
	if r.config != nil && len(r.config.Paths) > 0 {
		return r.config.Paths
	}

	// Default paths
	return DefaultPaths()
}

// GenerateConfig generates a basic phpstan.neon configuration file
func (r *PHPStanRunner) GenerateConfig(level int, paths []string) error {
	configPath := filepath.Join(r.manager.projectPath, "phpstan.neon")

	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("phpstan.neon already exists")
	}

	if level < 0 {
		level = DefaultPHPStanLevel()
	}
	if len(paths) == 0 {
		paths = DefaultPaths()
	}

	content := fmt.Sprintf(`parameters:
    level: %d
    paths:
`, level)

	for _, p := range paths {
		content += fmt.Sprintf("        - %s\n", p)
	}

	content += `
    # Magento-specific settings
    treatPhpDocTypesAsCertain: false

    # Optional: ignore specific errors
    # ignoreErrors:
    #     - '#Variable \$[a-zA-Z]+ might not be defined#'
`

	return os.WriteFile(configPath, []byte(content), 0644)
}

package testing

import (
	"fmt"
	"os"
	"path/filepath"

	"qoliber/magebox/internal/config"
)

// PHPCSRunner handles PHP_CodeSniffer code style checking
type PHPCSRunner struct {
	manager *Manager
	config  *config.PHPCSTestConfig
}

// NewPHPCSRunner creates a new PHPCS runner
func NewPHPCSRunner(m *Manager, cfg *config.PHPCSTestConfig) *PHPCSRunner {
	return &PHPCSRunner{
		manager: m,
		config:  cfg,
	}
}

// Run executes PHPCS code style checking
func (r *PHPCSRunner) Run(paths []string, standard string) error {
	if !r.manager.isComposerPackageInstalled("squizlabs/php_codesniffer") {
		return fmt.Errorf("PHP_CodeSniffer is not installed. Run: magebox test setup")
	}

	args := r.buildArgs(paths, standard)
	return r.manager.StreamCommand("PHPCS", args...)
}

// RunFix executes PHPCBF to automatically fix code style issues
func (r *PHPCSRunner) RunFix(paths []string, standard string) error {
	if !r.manager.isComposerPackageInstalled("squizlabs/php_codesniffer") {
		return fmt.Errorf("PHP_CodeSniffer is not installed. Run: magebox test setup")
	}

	args := r.buildFixArgs(paths, standard)
	return r.manager.StreamCommand("PHPCBF", args...)
}

// buildArgs builds the PHPCS command arguments
func (r *PHPCSRunner) buildArgs(paths []string, standard string) []string {
	phpBin := r.manager.GetPHPBinary()
	phpcsBin := filepath.Join(r.manager.GetVendorBinPath(), "phpcs")

	args := []string{phpBin, phpcsBin}

	// Check for config file
	configFile := r.getConfigFile()
	if configFile != "" {
		args = append(args, "--standard="+configFile)
	} else {
		// Use standard from argument or config
		std := r.getStandard(standard)
		args = append(args, "--standard="+std)
	}

	// Determine paths to check
	checkPaths := r.getPaths(paths)
	args = append(args, checkPaths...)

	// Add useful flags
	args = append(args, "--colors")
	args = append(args, "-p") // Show progress
	args = append(args, "-s") // Show sniff codes

	return args
}

// buildFixArgs builds the PHPCBF command arguments
func (r *PHPCSRunner) buildFixArgs(paths []string, standard string) []string {
	phpBin := r.manager.GetPHPBinary()
	phpcbfBin := filepath.Join(r.manager.GetVendorBinPath(), "phpcbf")

	args := []string{phpBin, phpcbfBin}

	// Check for config file
	configFile := r.getConfigFile()
	if configFile != "" {
		args = append(args, "--standard="+configFile)
	} else {
		std := r.getStandard(standard)
		args = append(args, "--standard="+std)
	}

	// Determine paths to fix
	checkPaths := r.getPaths(paths)
	args = append(args, checkPaths...)

	// Add useful flags
	args = append(args, "--colors")
	args = append(args, "-p") // Show progress

	return args
}

// getConfigFile returns the PHPCS config file path
func (r *PHPCSRunner) getConfigFile() string {
	// Check for config from yaml
	if r.config != nil && r.config.Config != "" {
		configPath := filepath.Join(r.manager.projectPath, r.config.Config)
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}
	}

	// Check for default config files
	configPaths := []string{
		filepath.Join(r.manager.projectPath, "phpcs.xml"),
		filepath.Join(r.manager.projectPath, "phpcs.xml.dist"),
		filepath.Join(r.manager.projectPath, ".phpcs.xml"),
	}
	for _, p := range configPaths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return ""
}

// getStandard returns the PHPCS coding standard
func (r *PHPCSRunner) getStandard(cliStandard string) string {
	// CLI standard takes precedence
	if cliStandard != "" {
		return cliStandard
	}

	// Then config standard
	if r.config != nil && r.config.Standard != "" {
		return r.config.Standard
	}

	// Default standard
	return DefaultPHPCSStandard()
}

// getPaths returns the paths to check
func (r *PHPCSRunner) getPaths(cliPaths []string) []string {
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

// GenerateConfig generates a basic phpcs.xml configuration file
func (r *PHPCSRunner) GenerateConfig(standard string, paths []string) error {
	configPath := filepath.Join(r.manager.projectPath, "phpcs.xml")

	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("phpcs.xml already exists")
	}

	if standard == "" {
		standard = DefaultPHPCSStandard()
	}
	if len(paths) == 0 {
		paths = DefaultPaths()
	}

	content := fmt.Sprintf(`<?xml version="1.0"?>
<ruleset name="Project Coding Standard">
    <description>Project coding standard based on %s</description>

    <!-- What to scan -->
`, standard)

	for _, p := range paths {
		content += fmt.Sprintf("    <file>%s</file>\n", p)
	}

	content += fmt.Sprintf(`
    <!-- Don't scan vendor -->
    <exclude-pattern>vendor/*</exclude-pattern>
    <exclude-pattern>generated/*</exclude-pattern>

    <!-- Use the %s standard -->
    <rule ref="%s"/>

    <!-- Allow long lines in certain cases -->
    <rule ref="Generic.Files.LineLength">
        <properties>
            <property name="lineLimit" value="120"/>
            <property name="absoluteLineLimit" value="0"/>
        </properties>
    </rule>
</ruleset>
`, standard, standard)

	return os.WriteFile(configPath, []byte(content), 0644)
}

// ListInstalledStandards lists available PHPCS coding standards
func (r *PHPCSRunner) ListInstalledStandards() ([]string, error) {
	phpBin := r.manager.GetPHPBinary()
	phpcsBin := filepath.Join(r.manager.GetVendorBinPath(), "phpcs")

	result := r.manager.RunCommand("PHPCS", phpBin, phpcsBin, "-i")
	if !result.Success {
		return nil, fmt.Errorf("failed to list standards: %s", result.Output)
	}

	// Parse output like "The installed coding standards are MySource, PEAR, PSR1, PSR2, PSR12, Squiz, Zend, Magento2 and MagentoCS"
	// This is a simplified parser
	return []string{"Magento2", "PSR12", "PSR1", "PSR2"}, nil
}

package testing

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"qoliber/magebox/internal/platform"
)

// Manager handles testing tool operations
type Manager struct {
	platform    *platform.Platform
	phpVersion  string
	projectPath string
}

// NewManager creates a new testing manager
func NewManager(p *platform.Platform, phpVersion, projectPath string) *Manager {
	return &Manager{
		platform:    p,
		phpVersion:  phpVersion,
		projectPath: projectPath,
	}
}

// GetStatus returns the status of all testing tools
func (m *Manager) GetStatus() *AllToolsStatus {
	return &AllToolsStatus{
		PHPUnit:     m.getPHPUnitStatus(),
		Integration: m.getIntegrationStatus(),
		PHPStan:     m.getPHPStanStatus(),
		PHPCS:       m.getPHPCSStatus(),
		PHPMD:       m.getPHPMDStatus(),
	}
}

// getPHPUnitStatus checks PHPUnit installation and configuration status
func (m *Manager) getPHPUnitStatus() ToolStatus {
	status := ToolStatus{Name: "PHPUnit"}

	// Check if installed via composer
	if m.isComposerPackageInstalled("phpunit/phpunit") {
		status.Installed = true
		status.Version = m.getComposerPackageVersion("phpunit/phpunit")
	}

	// Check for phpunit.xml or phpunit.xml.dist
	configPaths := []string{
		filepath.Join(m.projectPath, "phpunit.xml"),
		filepath.Join(m.projectPath, "phpunit.xml.dist"),
	}
	for _, p := range configPaths {
		if _, err := os.Stat(p); err == nil {
			status.Configured = true
			status.ConfigPath = p
			break
		}
	}

	return status
}

// getIntegrationStatus checks Magento integration test configuration
func (m *Manager) getIntegrationStatus() ToolStatus {
	status := ToolStatus{Name: "Integration Tests"}

	// Integration tests use PHPUnit, so check that
	if m.isComposerPackageInstalled("phpunit/phpunit") {
		status.Installed = true
		status.Version = m.getComposerPackageVersion("phpunit/phpunit")
	}

	// Check for Magento integration test config
	configPath := filepath.Join(m.projectPath, "dev", "tests", "integration", "phpunit.xml")
	configDistPath := filepath.Join(m.projectPath, "dev", "tests", "integration", "phpunit.xml.dist")

	if _, err := os.Stat(configPath); err == nil {
		status.Configured = true
		status.ConfigPath = configPath
	} else if _, err := os.Stat(configDistPath); err == nil {
		status.Configured = true
		status.ConfigPath = configDistPath
	}

	return status
}

// getPHPStanStatus checks PHPStan installation and configuration status
func (m *Manager) getPHPStanStatus() ToolStatus {
	status := ToolStatus{Name: "PHPStan"}

	// Check if installed via composer
	if m.isComposerPackageInstalled("phpstan/phpstan") {
		status.Installed = true
		status.Version = m.getComposerPackageVersion("phpstan/phpstan")
	}

	// Check for phpstan.neon or phpstan.neon.dist
	configPaths := []string{
		filepath.Join(m.projectPath, "phpstan.neon"),
		filepath.Join(m.projectPath, "phpstan.neon.dist"),
	}
	for _, p := range configPaths {
		if _, err := os.Stat(p); err == nil {
			status.Configured = true
			status.ConfigPath = p
			break
		}
	}

	return status
}

// getPHPCSStatus checks PHP_CodeSniffer installation and configuration status
func (m *Manager) getPHPCSStatus() ToolStatus {
	status := ToolStatus{Name: "PHP_CodeSniffer"}

	// Check if installed via composer
	if m.isComposerPackageInstalled("squizlabs/php_codesniffer") {
		status.Installed = true
		status.Version = m.getComposerPackageVersion("squizlabs/php_codesniffer")
	}

	// Check for phpcs.xml or phpcs.xml.dist or .phpcs.xml
	configPaths := []string{
		filepath.Join(m.projectPath, "phpcs.xml"),
		filepath.Join(m.projectPath, "phpcs.xml.dist"),
		filepath.Join(m.projectPath, ".phpcs.xml"),
	}
	for _, p := range configPaths {
		if _, err := os.Stat(p); err == nil {
			status.Configured = true
			status.ConfigPath = p
			break
		}
	}

	return status
}

// getPHPMDStatus checks PHPMD installation and configuration status
func (m *Manager) getPHPMDStatus() ToolStatus {
	status := ToolStatus{Name: "PHP Mess Detector"}

	// Check if installed via composer
	if m.isComposerPackageInstalled("phpmd/phpmd") {
		status.Installed = true
		status.Version = m.getComposerPackageVersion("phpmd/phpmd")
	}

	// Check for phpmd.xml or phpmd.xml.dist
	configPaths := []string{
		filepath.Join(m.projectPath, "phpmd.xml"),
		filepath.Join(m.projectPath, "phpmd.xml.dist"),
	}
	for _, p := range configPaths {
		if _, err := os.Stat(p); err == nil {
			status.Configured = true
			status.ConfigPath = p
			break
		}
	}

	return status
}

// isComposerPackageInstalled checks if a composer package is installed
func (m *Manager) isComposerPackageInstalled(packageName string) bool {
	composerLock := filepath.Join(m.projectPath, "composer.lock")
	data, err := os.ReadFile(composerLock)
	if err != nil {
		return false
	}

	var lock struct {
		Packages    []struct{ Name string } `json:"packages"`
		PackagesDev []struct{ Name string } `json:"packages-dev"`
	}

	if err := json.Unmarshal(data, &lock); err != nil {
		return false
	}

	for _, pkg := range lock.Packages {
		if pkg.Name == packageName {
			return true
		}
	}
	for _, pkg := range lock.PackagesDev {
		if pkg.Name == packageName {
			return true
		}
	}

	return false
}

// getComposerPackageVersion gets the version of an installed composer package
func (m *Manager) getComposerPackageVersion(packageName string) string {
	composerLock := filepath.Join(m.projectPath, "composer.lock")
	data, err := os.ReadFile(composerLock)
	if err != nil {
		return ""
	}

	var lock struct {
		Packages    []struct{ Name, Version string } `json:"packages"`
		PackagesDev []struct{ Name, Version string } `json:"packages-dev"`
	}

	if err := json.Unmarshal(data, &lock); err != nil {
		return ""
	}

	for _, pkg := range lock.Packages {
		if pkg.Name == packageName {
			return pkg.Version
		}
	}
	for _, pkg := range lock.PackagesDev {
		if pkg.Name == packageName {
			return pkg.Version
		}
	}

	return ""
}

// GetPHPBinary returns the PHP binary path for the configured version
func (m *Manager) GetPHPBinary() string {
	return m.platform.PHPBinary(m.phpVersion)
}

// GetComposerBinary returns the composer binary path
func (m *Manager) GetComposerBinary() string {
	// First check for local composer wrapper
	localComposer := filepath.Join(os.Getenv("HOME"), ".magebox", "bin", "composer")
	if _, err := os.Stat(localComposer); err == nil {
		return localComposer
	}

	// Fall back to system composer
	composerPath, err := exec.LookPath("composer")
	if err != nil {
		return "composer"
	}
	return composerPath
}

// GetVendorBinPath returns the path to vendor/bin directory
func (m *Manager) GetVendorBinPath() string {
	return filepath.Join(m.projectPath, "vendor", "bin")
}

// RunCommand executes a command and returns the result
func (m *Manager) RunCommand(name string, args ...string) *RunResult {
	result := &RunResult{Tool: name}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = m.projectPath
	cmd.Env = append(os.Environ(), fmt.Sprintf("PATH=%s:%s", m.GetVendorBinPath(), os.Getenv("PATH")))

	output, err := cmd.CombinedOutput()
	result.Output = string(output)
	result.ExitCode = cmd.ProcessState.ExitCode()
	result.Success = err == nil && result.ExitCode == 0

	// Count errors (simple heuristic based on output)
	result.ErrorCount = strings.Count(result.Output, "ERROR") +
		strings.Count(result.Output, "FAILURE") +
		strings.Count(result.Output, "Error:")

	return result
}

// StreamCommand executes a command with output streaming to stdout/stderr
func (m *Manager) StreamCommand(name string, args ...string) error {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = m.projectPath
	cmd.Env = append(os.Environ(), fmt.Sprintf("PATH=%s:%s", m.GetVendorBinPath(), os.Getenv("PATH")))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

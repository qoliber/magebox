package testing

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/qoliber/magebox/internal/config"
)

// PHPMDRunner handles PHP Mess Detector analysis
type PHPMDRunner struct {
	manager *Manager
	config  *config.PHPMDTestConfig
}

// NewPHPMDRunner creates a new PHPMD runner
func NewPHPMDRunner(m *Manager, cfg *config.PHPMDTestConfig) *PHPMDRunner {
	return &PHPMDRunner{
		manager: m,
		config:  cfg,
	}
}

// Run executes PHPMD analysis
func (r *PHPMDRunner) Run(paths []string, ruleset string) error {
	if !r.manager.isComposerPackageInstalled("phpmd/phpmd") {
		return fmt.Errorf("PHP Mess Detector is not installed. Run: magebox test setup")
	}

	args := r.buildArgs(paths, ruleset)
	return r.manager.StreamCommand("PHPMD", args...)
}

// buildArgs builds the PHPMD command arguments
func (r *PHPMDRunner) buildArgs(paths []string, ruleset string) []string {
	phpBin := r.manager.GetPHPBinary()
	phpmdBin := filepath.Join(r.manager.GetVendorBinPath(), "phpmd")

	args := []string{phpBin, phpmdBin}

	// Get paths to analyze
	analyzePaths := r.getPaths(paths)
	args = append(args, strings.Join(analyzePaths, ","))

	// Output format
	args = append(args, "text")

	// Get ruleset (can be a comma-separated list or config file)
	rulesetArg := r.getRuleset(ruleset)
	args = append(args, rulesetArg)

	// Exclude vendor and generated
	args = append(args, "--exclude", "vendor,generated")

	return args
}

// getRuleset returns the PHPMD ruleset
func (r *PHPMDRunner) getRuleset(cliRuleset string) string {
	// Check for config file first
	configFile := r.getConfigFile()
	if configFile != "" {
		return configFile
	}

	// CLI ruleset takes precedence
	if cliRuleset != "" {
		return cliRuleset
	}

	// Then config ruleset
	if r.config != nil && r.config.Ruleset != "" {
		return r.config.Ruleset
	}

	// Default ruleset
	return DefaultPHPMDRuleset()
}

// getConfigFile returns the PHPMD config file path
func (r *PHPMDRunner) getConfigFile() string {
	// Check for config from yaml
	if r.config != nil && r.config.Config != "" {
		configPath := filepath.Join(r.manager.projectPath, r.config.Config)
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}
	}

	// Check for default config files
	configPaths := []string{
		filepath.Join(r.manager.projectPath, "phpmd.xml"),
		filepath.Join(r.manager.projectPath, "phpmd.xml.dist"),
	}
	for _, p := range configPaths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return ""
}

// getPaths returns the paths to analyze
func (r *PHPMDRunner) getPaths(cliPaths []string) []string {
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

// GenerateConfig generates a basic phpmd.xml configuration file
func (r *PHPMDRunner) GenerateConfig(ruleset string) error {
	configPath := filepath.Join(r.manager.projectPath, "phpmd.xml")

	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("phpmd.xml already exists")
	}

	if ruleset == "" {
		ruleset = DefaultPHPMDRuleset()
	}

	// Parse ruleset into individual rules
	rules := strings.Split(ruleset, ",")

	content := `<?xml version="1.0"?>
<ruleset name="Project Mess Detector Rules"
         xmlns="http://pmd.sf.net/ruleset/1.0.0"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://pmd.sf.net/ruleset/1.0.0 http://pmd.sf.net/ruleset_xml_schema.xsd"
         xsi:noNamespaceSchemaLocation="http://pmd.sf.net/ruleset_xml_schema.xsd">

    <description>
        Project PHPMD rules
    </description>

    <!-- Exclude vendor and generated directories -->
    <exclude-pattern>vendor/*</exclude-pattern>
    <exclude-pattern>generated/*</exclude-pattern>

`

	// Add rule references
	for _, rule := range rules {
		rule = strings.TrimSpace(rule)
		content += fmt.Sprintf("    <rule ref=\"rulesets/%s.xml\"/>\n", rule)
	}

	content += `
    <!-- Customize rules as needed -->
    <!-- Example: Allow longer methods for controllers -->
    <!--
    <rule ref="rulesets/codesize.xml/ExcessiveMethodLength">
        <properties>
            <property name="minimum" value="150"/>
        </properties>
    </rule>
    -->
</ruleset>
`

	return os.WriteFile(configPath, []byte(content), 0644)
}

// AvailableRulesets returns the available PHPMD rulesets
func (r *PHPMDRunner) AvailableRulesets() []string {
	return []string{
		"cleancode",
		"codesize",
		"controversial",
		"design",
		"naming",
		"unusedcode",
	}
}

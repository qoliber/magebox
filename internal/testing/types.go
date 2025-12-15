package testing

// ToolStatus represents the status of a testing tool
type ToolStatus struct {
	Name       string
	Installed  bool
	Configured bool
	ConfigPath string
	Version    string
}

// AllToolsStatus represents the status of all testing tools
type AllToolsStatus struct {
	PHPUnit     ToolStatus
	Integration ToolStatus
	PHPStan     ToolStatus
	PHPCS       ToolStatus
	PHPMD       ToolStatus
}

// ComposerPackages defines the composer packages for each tool
var ComposerPackages = map[string][]string{
	"phpunit": {"phpunit/phpunit"},
	"phpstan": {"phpstan/phpstan", "bitexpert/phpstan-magento"},
	"phpcs":   {"squizlabs/php_codesniffer", "magento/magento-coding-standard"},
	"phpmd":   {"phpmd/phpmd"},
}

// DefaultPaths returns the default paths to analyze
func DefaultPaths() []string {
	return []string{"app/code"}
}

// DefaultPHPStanLevel returns the default PHPStan level
func DefaultPHPStanLevel() int {
	return 1
}

// DefaultPHPCSStandard returns the default PHPCS coding standard
func DefaultPHPCSStandard() string {
	return "Magento2"
}

// DefaultPHPMDRuleset returns the default PHPMD ruleset
func DefaultPHPMDRuleset() string {
	return "cleancode,codesize,design"
}

// RunResult represents the result of running a test tool
type RunResult struct {
	Tool       string
	Success    bool
	Output     string
	ExitCode   int
	Duration   float64 // seconds
	ErrorCount int
}

// SetupOptions represents options for the setup wizard
type SetupOptions struct {
	InstallPHPUnit bool
	InstallPHPStan bool
	InstallPHPCS   bool
	InstallPHPMD   bool
}

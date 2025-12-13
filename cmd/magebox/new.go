package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/qoliber/magebox/internal/cli"
	"github.com/qoliber/magebox/internal/config"
	"github.com/qoliber/magebox/internal/nginx"
	"github.com/qoliber/magebox/internal/php"
	"github.com/qoliber/magebox/internal/platform"
	"github.com/qoliber/magebox/internal/project"
	"github.com/qoliber/magebox/internal/templates"
)

var newCmd = &cobra.Command{
	Use:   "new [directory]",
	Short: "Create a new Magento/MageOS project",
	Long: `Creates a new Magento or MageOS project with interactive setup wizard.

This command will guide you through:
  1. Selecting Magento or MageOS distribution
  2. Choosing the version to install
  3. Configuring Composer authentication
  4. Selecting PHP version
  5. Choosing services (MySQL, Redis, OpenSearch, etc.)
  6. Optional sample data installation
  7. Database setup

Quick Mode (--quick):
  Skip all questions and install MageOS with sensible defaults:
  - MageOS 1.0.4 (latest stable, no auth required)
  - PHP 8.3
  - MySQL 8.0, Redis, OpenSearch
  - Sample data included
  - Domain: {directory}.test

Example:
  magebox new mystore              # Interactive wizard
  magebox new mystore --quick      # Quick install with defaults + sample data
  magebox new . --quick            # Quick install in current directory`,
	Args: cobra.ExactArgs(1),
	RunE: runNew,
}

// New command flags
var (
	newQuick      bool
	newWithSample bool
)

func init() {
	newCmd.Flags().BoolVarP(&newQuick, "quick", "q", false, "Quick install with defaults (MageOS + sample data)")
	newCmd.Flags().BoolVar(&newWithSample, "with-sample", false, "Include sample data (used with --quick)")
	rootCmd.AddCommand(newCmd)
}

// MagentoVersion represents a Magento/MageOS version
type MagentoVersion struct {
	Name        string
	Version     string
	Package     string
	PHPVersions []string
	Default     bool
}

// Distribution types
const (
	DistMagento = "magento"
	DistMageOS  = "mageos"
)

// Service readiness configuration
const (
	// OpenSearchReadinessMaxRetries is the number of retries when waiting for OpenSearch
	OpenSearchReadinessMaxRetries = 30
	// OpenSearchReadinessRetryInterval is the time between readiness checks
	OpenSearchReadinessRetryInterval = 2 * time.Second
	// OpenSearchDefaultPort is the default port for OpenSearch
	OpenSearchDefaultPort = 9200
)

// Default service versions for quick install
const (
	DefaultPHPVersion        = "8.3"
	DefaultMySQLVersion      = "8.0"
	DefaultOpenSearchVersion = "2.19.4"
)

// Default database credentials
const (
	DefaultDBUser     = "root"
	DefaultDBPassword = "magebox"
)

// Default admin credentials for quick install
const (
	DefaultAdminUser     = "admin"
	DefaultAdminPassword = "admin123"
	DefaultAdminEmail    = "admin@example.com"
)

// Redis default ports and database numbers
const (
	RedisDefaultPort     = 6379
	RedisSessionDB       = 2
	RedisCacheDB         = 0
	RedisFullPageCacheDB = 1
)

// RabbitMQ defaults
const (
	RabbitMQDefaultPort = 5672
	RabbitMQDefaultUser = "guest"
	RabbitMQDefaultPass = "guest"
)

// Available Magento versions
var magentoVersions = []MagentoVersion{
	{Name: "Magento 2.4.7-p3 (Latest)", Version: "2.4.7-p3", Package: "magento/project-community-edition", PHPVersions: []string{"8.3", "8.2"}, Default: true},
	{Name: "Magento 2.4.7-p2", Version: "2.4.7-p2", Package: "magento/project-community-edition", PHPVersions: []string{"8.3", "8.2"}},
	{Name: "Magento 2.4.7-p1", Version: "2.4.7-p1", Package: "magento/project-community-edition", PHPVersions: []string{"8.3", "8.2"}},
	{Name: "Magento 2.4.7", Version: "2.4.7", Package: "magento/project-community-edition", PHPVersions: []string{"8.3", "8.2"}},
	{Name: "Magento 2.4.6-p7", Version: "2.4.6-p7", Package: "magento/project-community-edition", PHPVersions: []string{"8.2", "8.1"}},
	{Name: "Magento 2.4.6-p6", Version: "2.4.6-p6", Package: "magento/project-community-edition", PHPVersions: []string{"8.2", "8.1"}},
	{Name: "Magento 2.4.5-p9", Version: "2.4.5-p9", Package: "magento/project-community-edition", PHPVersions: []string{"8.1"}},
}

// Available MageOS versions
var mageosVersions = []MagentoVersion{
	{Name: "MageOS 1.0.4 (Latest)", Version: "1.0.4", Package: "mage-os/project-community-edition", PHPVersions: []string{"8.3", "8.2"}, Default: true},
	{Name: "MageOS 1.0.3", Version: "1.0.3", Package: "mage-os/project-community-edition", PHPVersions: []string{"8.3", "8.2"}},
	{Name: "MageOS 1.0.2", Version: "1.0.2", Package: "mage-os/project-community-edition", PHPVersions: []string{"8.3", "8.2"}},
	{Name: "MageOS 1.0.1", Version: "1.0.1", Package: "mage-os/project-community-edition", PHPVersions: []string{"8.2", "8.1"}},
}

// findRealComposer finds the real composer binary, skipping our wrapper
func findRealComposer(p *platform.Platform) (string, error) {
	// Our wrapper is in ~/.magebox/bin/composer - we need to skip it
	wrapperDir := filepath.Join(p.MageBoxDir(), "bin")

	// Check common locations for composer.phar first
	pharLocations := []string{
		"/usr/local/bin/composer.phar",
		"/opt/homebrew/bin/composer.phar",
		filepath.Join(os.Getenv("HOME"), ".composer", "composer.phar"),
		filepath.Join(os.Getenv("HOME"), "composer.phar"),
	}

	for _, loc := range pharLocations {
		if _, err := os.Stat(loc); err == nil {
			return loc, nil
		}
	}

	// Search PATH for composer, but skip our wrapper directory
	pathEnv := os.Getenv("PATH")
	paths := strings.Split(pathEnv, string(os.PathListSeparator))

	for _, dir := range paths {
		// Skip our wrapper directory
		if dir == wrapperDir {
			continue
		}

		composerPath := filepath.Join(dir, "composer")
		if info, err := os.Stat(composerPath); err == nil && info.Mode()&0111 != 0 {
			// Check if it's a PHP file (not our bash wrapper)
			// Read first few bytes to check for PHP shebang or <?php
			content, err := os.ReadFile(composerPath)
			if err == nil && len(content) > 10 {
				header := string(content[:100])
				// Skip bash scripts
				if strings.HasPrefix(header, "#!/bin/bash") || strings.Contains(header, "# MageBox") {
					continue
				}
				// It's likely a PHP script or phar
				return composerPath, nil
			}
		}

		// Also check for composer.phar
		pharPath := filepath.Join(dir, "composer.phar")
		if _, err := os.Stat(pharPath); err == nil {
			return pharPath, nil
		}
	}

	return "", fmt.Errorf("composer not found in PATH (excluding MageBox wrapper)")
}

// runNew creates a new Magento/MageOS project with interactive setup
func runNew(cmd *cobra.Command, args []string) error {
	targetDir := args[0]
	reader := bufio.NewReader(os.Stdin)

	p, err := getPlatform()
	if err != nil {
		return err
	}

	cli.PrintLogoSmall(version)
	fmt.Println()

	// Check prerequisites
	if !platform.CommandExists("composer") {
		cli.PrintError("Composer is not installed!")
		fmt.Println()
		cli.PrintInfo("Install Composer first:")
		fmt.Println("  curl -sS https://getcomposer.org/installer | php")
		fmt.Println("  sudo mv composer.phar /usr/local/bin/composer")
		return nil
	}

	// Quick mode - skip all questions, use sensible defaults
	if newQuick {
		return runNewQuick(targetDir, p)
	}

	cli.PrintTitle("Create New Magento/MageOS Project")
	fmt.Println()

	// Step 1: Choose distribution
	fmt.Println(cli.Header("Step 1: Choose Distribution"))
	fmt.Println()
	fmt.Println("  [1] Magento Open Source (Adobe)")
	fmt.Println("  [2] MageOS (Community Fork)")
	fmt.Println()
	fmt.Print("Select distribution [1]: ")

	distChoice, _ := reader.ReadString('\n')
	distChoice = strings.TrimSpace(distChoice)
	if distChoice == "" {
		distChoice = "1"
	}

	var distribution string
	var versions []MagentoVersion
	if distChoice == "2" {
		distribution = DistMageOS
		versions = mageosVersions
		fmt.Println("  → MageOS selected")
	} else {
		distribution = DistMagento
		versions = magentoVersions
		fmt.Println("  → Magento Open Source selected")
	}
	fmt.Println()

	// Step 2: Choose version
	fmt.Println(cli.Header("Step 2: Choose Version"))
	fmt.Println()
	defaultIdx := 0
	for i, v := range versions {
		marker := "  "
		if v.Default {
			marker = "→ "
			defaultIdx = i + 1
		}
		fmt.Printf("  [%d] %s%s\n", i+1, marker, v.Name)
	}
	fmt.Println()
	fmt.Printf("Select version [%d]: ", defaultIdx)

	versionChoice, _ := reader.ReadString('\n')
	versionChoice = strings.TrimSpace(versionChoice)
	if versionChoice == "" {
		versionChoice = fmt.Sprintf("%d", defaultIdx)
	}

	versionIdx := 0
	_, _ = fmt.Sscanf(versionChoice, "%d", &versionIdx)
	if versionIdx < 1 || versionIdx > len(versions) {
		versionIdx = defaultIdx
	}
	selectedVersion := versions[versionIdx-1]
	fmt.Printf("  → %s selected\n", selectedVersion.Name)
	fmt.Println()

	// Step 3: PHP Version
	fmt.Println(cli.Header("Step 3: PHP Version"))
	fmt.Println()
	fmt.Printf("  Compatible versions: %s\n", strings.Join(selectedVersion.PHPVersions, ", "))
	fmt.Println()
	for i, phpV := range selectedVersion.PHPVersions {
		marker := "  "
		if i == 0 {
			marker = "→ "
		}
		fmt.Printf("  [%d] %sPHP %s\n", i+1, marker, phpV)
	}
	fmt.Println()
	fmt.Print("Select PHP version [1]: ")

	phpChoice, _ := reader.ReadString('\n')
	phpChoice = strings.TrimSpace(phpChoice)
	if phpChoice == "" {
		phpChoice = "1"
	}

	phpIdx := 0
	_, _ = fmt.Sscanf(phpChoice, "%d", &phpIdx)
	if phpIdx < 1 || phpIdx > len(selectedVersion.PHPVersions) {
		phpIdx = 1
	}
	selectedPHP := selectedVersion.PHPVersions[phpIdx-1]
	fmt.Printf("  → PHP %s selected\n", selectedPHP)
	fmt.Println()

	// Check if PHP version is installed
	detector := php.NewDetector(p)
	phpVersion := detector.Detect(selectedPHP)
	if !phpVersion.Installed {
		cli.PrintWarning("PHP %s is not installed!", selectedPHP)
		fmt.Printf("  Install: %s\n", cli.Command(p.PHPInstallCommand(selectedPHP)))
		fmt.Println()
		fmt.Print("Continue anyway? [y/N]: ")
		continueChoice, _ := reader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(continueChoice)) != "y" {
			return nil
		}
	}

	// Step 4: Composer Authentication (for Magento only)
	var composerUser, composerPass string
	if distribution == DistMagento {
		fmt.Println(cli.Header("Step 4: Composer Authentication"))
		fmt.Println()
		fmt.Println("  Magento requires authentication keys from marketplace.magento.com")
		fmt.Println("  Get your keys at: " + cli.URL("https://marketplace.magento.com/customer/accessKeys/"))
		fmt.Println()

		// Check for existing auth.json
		homeDir, _ := os.UserHomeDir()
		authFile := filepath.Join(homeDir, ".composer", "auth.json")
		hasAuth := false
		if _, err := os.Stat(authFile); err == nil {
			// Check if repo.magento.com exists in auth.json
			authContent, _ := os.ReadFile(authFile)
			if strings.Contains(string(authContent), "repo.magento.com") {
				hasAuth = true
				fmt.Println("  " + cli.Success("✓") + " Found existing Composer authentication")
			}
		}

		if !hasAuth {
			fmt.Print("Public Key (username): ")
			composerUser, _ = reader.ReadString('\n')
			composerUser = strings.TrimSpace(composerUser)

			fmt.Print("Private Key (password): ")
			composerPass, _ = reader.ReadString('\n')
			composerPass = strings.TrimSpace(composerPass)

			if composerUser == "" || composerPass == "" {
				cli.PrintError("Composer keys are required for Magento installation")
				return nil
			}
		}
		fmt.Println()
	} else {
		fmt.Println(cli.Header("Step 4: Composer Authentication"))
		fmt.Println()
		fmt.Println("  " + cli.Success("✓") + " MageOS does not require authentication keys")
		fmt.Println()
	}

	// Step 5: Services
	fmt.Println(cli.Header("Step 5: Database & Services"))
	fmt.Println()

	// MySQL version
	fmt.Println("  Database:")
	fmt.Println("  [1] → MySQL 8.0 (recommended)")
	fmt.Println("  [2]   MySQL 8.4")
	fmt.Println("  [3]   MariaDB 10.6")
	fmt.Println("  [4]   MariaDB 11.4")
	fmt.Println()
	fmt.Print("Select database [1]: ")

	dbChoice, _ := reader.ReadString('\n')
	dbChoice = strings.TrimSpace(dbChoice)
	if dbChoice == "" {
		dbChoice = "1"
	}

	var dbService, dbVersion string
	switch dbChoice {
	case "2":
		dbService, dbVersion = "mysql", "8.4"
	case "3":
		dbService, dbVersion = "mariadb", "10.6"
	case "4":
		dbService, dbVersion = "mariadb", "11.4"
	default:
		dbService, dbVersion = "mysql", "8.0"
	}
	fmt.Printf("  → %s %s selected\n", titleCase(dbService), dbVersion)
	fmt.Println()

	// Search engine
	fmt.Println("  Search Engine:")
	fmt.Println("  [1] → OpenSearch 2.19 (recommended)")
	fmt.Println("  [2]   Elasticsearch 8.11")
	fmt.Println("  [3]   None (use MySQL for catalog search)")
	fmt.Println()
	fmt.Print("Select search engine [1]: ")

	searchChoice, _ := reader.ReadString('\n')
	searchChoice = strings.TrimSpace(searchChoice)
	if searchChoice == "" {
		searchChoice = "1"
	}

	var searchEngine, searchVersion string
	switch searchChoice {
	case "2":
		searchEngine, searchVersion = "elasticsearch", "8.11"
	case "3":
		searchEngine, searchVersion = "", ""
	default:
		searchEngine, searchVersion = "opensearch", "2.19.4"
	}
	if searchEngine != "" {
		fmt.Printf("  → %s %s selected\n", titleCase(searchEngine), searchVersion)
	} else {
		fmt.Println("  → No search engine (MySQL search)")
	}
	fmt.Println()

	// Additional services
	fmt.Println("  Additional Services:")
	fmt.Print("  Enable Redis cache? [Y/n]: ")
	redisChoice, _ := reader.ReadString('\n')
	enableRedis := strings.ToLower(strings.TrimSpace(redisChoice)) != "n"

	fmt.Print("  Enable RabbitMQ? [y/N]: ")
	rabbitChoice, _ := reader.ReadString('\n')
	enableRabbitMQ := strings.ToLower(strings.TrimSpace(rabbitChoice)) == "y"

	// Mailpit is always enabled by default for email testing
	enableMailpit := true
	fmt.Println("  Mailpit (email testing): enabled by default")

	// Step 6: Sample Data
	fmt.Println(cli.Header("Step 6: Sample Data"))
	fmt.Println()
	fmt.Println("  Sample data includes demo products, categories, and CMS content.")
	fmt.Print("  Install sample data? [y/N]: ")
	sampleChoice, _ := reader.ReadString('\n')
	installSampleData := strings.ToLower(strings.TrimSpace(sampleChoice)) == "y"
	fmt.Println()

	// Step 7: Project Details
	fmt.Println(cli.Header("Step 7: Project Details"))
	fmt.Println()

	// Determine project name from directory
	var projectDir string
	if targetDir == "." {
		projectDir, _ = os.Getwd()
	} else {
		if filepath.IsAbs(targetDir) {
			projectDir = targetDir
		} else {
			cwd, _ := os.Getwd()
			projectDir = filepath.Join(cwd, targetDir)
		}
	}
	projectName := filepath.Base(projectDir)

	fmt.Printf("  Project directory: %s\n", cli.Highlight(projectDir))
	fmt.Printf("  Project name [%s]: ", projectName)
	nameInput, _ := reader.ReadString('\n')
	nameInput = strings.TrimSpace(nameInput)
	if nameInput != "" {
		projectName = nameInput
	}

	// Domain
	defaultDomain := projectName + ".test"
	fmt.Printf("  Domain [%s]: ", defaultDomain)
	domainInput, _ := reader.ReadString('\n')
	domainInput = strings.TrimSpace(domainInput)
	if domainInput == "" {
		domainInput = defaultDomain
	}
	fmt.Println()

	// Summary
	fmt.Println(cli.Header("Summary"))
	fmt.Println()
	fmt.Printf("  Distribution:    %s\n", cli.Highlight(distribution))
	fmt.Printf("  Version:         %s\n", cli.Highlight(selectedVersion.Version))
	fmt.Printf("  PHP:             %s\n", cli.Highlight(selectedPHP))
	fmt.Printf("  Database:        %s %s\n", cli.Highlight(dbService), cli.Highlight(dbVersion))
	if searchEngine != "" {
		fmt.Printf("  Search:          %s %s\n", cli.Highlight(searchEngine), cli.Highlight(searchVersion))
	}
	fmt.Printf("  Redis:           %s\n", cli.Status(enableRedis))
	fmt.Printf("  RabbitMQ:        %s\n", cli.Status(enableRabbitMQ))
	fmt.Printf("  Mailpit:         %s\n", cli.Status(enableMailpit))
	fmt.Printf("  Sample Data:     %s\n", cli.Status(installSampleData))
	fmt.Printf("  Project:         %s\n", cli.Highlight(projectName))
	fmt.Printf("  Domain:          %s\n", cli.URL("https://"+domainInput))
	fmt.Println()

	fmt.Print("Proceed with installation? [Y/n]: ")
	proceedChoice, _ := reader.ReadString('\n')
	if strings.ToLower(strings.TrimSpace(proceedChoice)) == "n" {
		fmt.Println("Installation canceled.")
		return nil
	}
	fmt.Println()

	// Get the correct PHP binary and composer path for this version
	phpBin := p.PHPBinary(selectedPHP)
	composerPath, err := findRealComposer(p)
	if err != nil {
		return fmt.Errorf("composer not found: %w", err)
	}

	// Set up Composer auth if needed (use explicit PHP to avoid shebang issues)
	if composerUser != "" && composerPass != "" {
		cli.PrintInfo("Configuring Composer authentication...")
		authCmd := exec.Command(phpBin, composerPath, "config", "--global", "http-basic.repo.magento.com", composerUser, composerPass)
		if err := authCmd.Run(); err != nil {
			cli.PrintWarning("Failed to configure Composer auth: %v", err)
		}
	}

	// Create project directory and .magebox.yaml first so our wrapper uses correct PHP
	cli.PrintTitle("Installing " + selectedVersion.Name)
	fmt.Println()

	// Step 1: Create directory and .magebox.yaml
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Create .magebox.yaml with PHP version so our wrapper uses it
	cli.PrintInfo("Creating MageBox configuration...")

	mageboxConfig := fmt.Sprintf(`name: %s
domains:
  - host: %s
    root: pub
    ssl: true
php: "%s"
services:
`, projectName, domainInput, selectedPHP)

	// Add database
	if dbService == "mysql" {
		mageboxConfig += fmt.Sprintf("  mysql: \"%s\"\n", dbVersion)
	} else {
		mageboxConfig += fmt.Sprintf("  mariadb: \"%s\"\n", dbVersion)
	}

	// Add search
	if searchEngine == "opensearch" {
		mageboxConfig += fmt.Sprintf("  opensearch:\n    version: \"%s\"\n    memory: \"2g\"\n", searchVersion)
	} else if searchEngine == "elasticsearch" {
		mageboxConfig += fmt.Sprintf("  elasticsearch:\n    version: \"%s\"\n    memory: \"2g\"\n", searchVersion)
	}

	// Add other services
	if enableRedis {
		mageboxConfig += "  redis: true\n"
	}
	if enableRabbitMQ {
		mageboxConfig += "  rabbitmq: true\n"
	}
	if enableMailpit {
		mageboxConfig += "  mailpit: true\n"
	}

	// Add common commands
	mageboxConfig += `
commands:
  setup:
    description: "Run Magento setup"
    run: "php bin/magento setup:upgrade && php bin/magento cache:flush"
  reindex:
    description: "Reindex all indexes"
    run: "php bin/magento indexer:reindex"
  deploy:
    description: "Deploy static content"
    run: "php bin/magento setup:static-content:deploy -f"
  cache:
    description: "Flush all caches"
    run: "php bin/magento cache:flush"
`

	mageboxFile := filepath.Join(projectDir, config.ConfigFileName)
	if err := os.WriteFile(mageboxFile, []byte(mageboxConfig), 0644); err != nil {
		cli.PrintWarning("Failed to create %s file: %v", config.ConfigFileName, err)
	} else {
		fmt.Printf("  Created %s\n", cli.Highlight(config.ConfigFileName))
	}

	// Step 2: Initialize composer.json and install Magento
	fmt.Println()
	cli.PrintInfo("Installing Magento via Composer...")
	fmt.Printf("  Using PHP %s: %s\n", selectedPHP, phpBin)

	// Create composer.json from proper template
	var composerJSON []byte
	if distribution == DistMageOS {
		composerJSON, err = templates.GenerateMageOSComposerJSON(projectName, selectedVersion.Version)
	} else {
		composerJSON, err = templates.GenerateMagentoComposerJSON(projectName, selectedVersion.Version)
	}
	if err != nil {
		return fmt.Errorf("failed to generate composer.json: %w", err)
	}

	composerJSONFile := filepath.Join(projectDir, "composer.json")
	if err := os.WriteFile(composerJSONFile, composerJSON, 0644); err != nil {
		return fmt.Errorf("failed to create composer.json: %w", err)
	}

	// Run composer install using our wrapper (which reads .magebox.yaml for PHP version)
	wrapperPath := filepath.Join(p.MageBoxDir(), "bin", "composer")
	compInstallCmd := exec.Command(wrapperPath, "install")
	compInstallCmd.Dir = projectDir
	compInstallCmd.Stdout = os.Stdout
	compInstallCmd.Stderr = os.Stderr
	compInstallCmd.Stdin = os.Stdin

	if err := compInstallCmd.Run(); err != nil {
		cli.PrintError("Composer install failed: %v", err)
		return err
	}
	fmt.Println()

	// Install sample data if requested
	if installSampleData {
		fmt.Println()
		cli.PrintInfo("Installing sample data...")

		// Use our wrapper which reads .magebox.yaml for PHP version
		sampleCmd := exec.Command(wrapperPath, "require",
			"magento/module-bundle-sample-data",
			"magento/module-catalog-sample-data", "magento/module-catalog-rule-sample-data",
			"magento/module-cms-sample-data", "magento/module-configurable-sample-data",
			"magento/module-customer-sample-data", "magento/module-downloadable-sample-data",
			"magento/module-grouped-sample-data", "magento/module-msrp-sample-data",
			"magento/module-offline-shipping-sample-data", "magento/module-product-links-sample-data",
			"magento/module-review-sample-data", "magento/module-sales-rule-sample-data",
			"magento/module-sales-sample-data", "magento/module-swatches-sample-data",
			"magento/module-tax-sample-data", "magento/module-theme-sample-data",
			"magento/module-widget-sample-data", "magento/module-wishlist-sample-data",
			"magento/sample-data-media", "--no-update")
		sampleCmd.Dir = projectDir
		sampleCmd.Stdout = os.Stdout
		sampleCmd.Stderr = os.Stderr

		if err := sampleCmd.Run(); err != nil {
			cli.PrintWarning("Sample data require failed: %v", err)
		}

		// Run composer update
		updateCmd := exec.Command(wrapperPath, "update")
		updateCmd.Dir = projectDir
		updateCmd.Stdout = os.Stdout
		updateCmd.Stderr = os.Stderr
		_ = updateCmd.Run()
	}

	// Success!
	fmt.Println()
	cli.PrintTitle("Installation Complete!")
	fmt.Println()
	cli.PrintSuccess("Project created successfully!")
	fmt.Println()

	// Determine database port based on service and version
	var dbPort string
	if dbService == "mysql" {
		switch dbVersion {
		case "8.4":
			dbPort = "33084"
		default:
			dbPort = "33080"
		}
	} else {
		switch dbVersion {
		case "10.4":
			dbPort = "33104"
		case "10.5":
			dbPort = "33105"
		case "11.0":
			dbPort = "33110"
		case "11.4":
			dbPort = "33114"
		default:
			dbPort = "33106"
		}
	}

	fmt.Println("Next steps:")
	fmt.Println()
	fmt.Println(cli.Bullet("1. Start services:"))
	fmt.Println("      cd " + cli.Highlight(projectDir))
	fmt.Println("      " + cli.Command("magebox start"))
	fmt.Println()
	fmt.Println(cli.Bullet("2. Install Magento:"))

	// Build setup:install command based on selected services
	installCmd := fmt.Sprintf(`php bin/magento setup:install \
    --base-url=https://%s \
    --backend-frontname=admin \
    --db-host=127.0.0.1:%s \
    --db-name=%s \
    --db-user=root \
    --db-password=magebox`, domainInput, dbPort, projectName)

	// Add search engine config
	if searchEngine == "opensearch" {
		installCmd += ` \
    --search-engine=opensearch \
    --opensearch-host=127.0.0.1 \
    --opensearch-port=9200 \
    --opensearch-index-prefix=magento2 \
    --opensearch-timeout=15`
	} else if searchEngine == "elasticsearch" {
		installCmd += ` \
    --search-engine=elasticsearch7 \
    --elasticsearch-host=127.0.0.1 \
    --elasticsearch-port=9200 \
    --elasticsearch-index-prefix=magento2 \
    --elasticsearch-timeout=15`
	}

	// Add Redis config
	if enableRedis {
		installCmd += ` \
    --session-save=redis \
    --session-save-redis-host=127.0.0.1 \
    --session-save-redis-port=6379 \
    --session-save-redis-db=2 \
    --cache-backend=redis \
    --cache-backend-redis-server=127.0.0.1 \
    --cache-backend-redis-port=6379 \
    --cache-backend-redis-db=0 \
    --page-cache=redis \
    --page-cache-redis-server=127.0.0.1 \
    --page-cache-redis-port=6379 \
    --page-cache-redis-db=1`
	}

	// Add RabbitMQ config
	if enableRabbitMQ {
		installCmd += ` \
    --amqp-host=127.0.0.1 \
    --amqp-port=5672 \
    --amqp-user=guest \
    --amqp-password=guest`
	}

	fmt.Println("      " + cli.Command(installCmd))
	fmt.Println()

	if installSampleData {
		fmt.Println(cli.Bullet("3. Deploy sample data:"))
		fmt.Println("      " + cli.Command("php bin/magento sampledata:deploy"))
		fmt.Println("      " + cli.Command("php bin/magento setup:upgrade"))
		fmt.Println("      " + cli.Command("php bin/magento indexer:reindex"))
		fmt.Println("      " + cli.Command("php bin/magento cache:flush"))
		fmt.Println()
	}

	fmt.Println("After setup, access your store at: " + cli.URL("https://"+domainInput))
	fmt.Println("Admin panel: " + cli.URL("https://"+domainInput+"/admin"))
	fmt.Println()

	return nil
}

// runNewQuick creates a new MageOS project with sensible defaults (no questions)
func runNewQuick(targetDir string, p *platform.Platform) error {
	cli.PrintTitle("Quick Install - MageOS with Sample Data")
	fmt.Println()

	// Defaults for quick mode
	selectedVersion := mageosVersions[0] // MageOS 1.0.4 (latest)
	selectedPHP := DefaultPHPVersion
	dbVersion := DefaultMySQLVersion
	searchVersion := DefaultOpenSearchVersion

	// Determine project name and directory
	var projectName string
	var projectDir string

	if targetDir == "." {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		projectDir = cwd
		projectName = filepath.Base(cwd)
	} else {
		projectDir, _ = filepath.Abs(targetDir)
		projectName = filepath.Base(targetDir)
	}

	// Domain from project name
	domainInput := projectName + ".test"

	// Check PHP availability
	detector := php.NewDetector(p)
	installedVersions := detector.DetectInstalled()

	phpFound := false
	for _, v := range installedVersions {
		if v.Version == selectedPHP {
			phpFound = true
			break
		}
	}

	if !phpFound {
		// Try to find any compatible version
		for _, compatiblePHP := range selectedVersion.PHPVersions {
			for _, v := range installedVersions {
				if v.Version == compatiblePHP {
					selectedPHP = compatiblePHP
					phpFound = true
					cli.PrintWarning("PHP 8.3 not found, using PHP %s instead", selectedPHP)
					break
				}
			}
			if phpFound {
				break
			}
		}
	}

	if !phpFound {
		cli.PrintError("No compatible PHP version found!")
		fmt.Println()
		cli.PrintInfo("Install PHP 8.3 first:")
		fmt.Println("  macOS:  brew install php@8.3")
		fmt.Println("  Ubuntu: sudo apt install php8.3-fpm php8.3-cli ...")
		return nil
	}

	// Show what we're going to install
	fmt.Println("Configuration:")
	fmt.Println(cli.Bullet("Distribution: " + cli.Highlight("MageOS")))
	fmt.Println(cli.Bullet("Version:      " + cli.Highlight(selectedVersion.Name)))
	fmt.Println(cli.Bullet("PHP:          " + cli.Highlight(selectedPHP)))
	fmt.Println(cli.Bullet("Database:     " + cli.Highlight("MySQL "+dbVersion)))
	fmt.Println(cli.Bullet("Search:       " + cli.Highlight("OpenSearch "+searchVersion)))
	fmt.Println(cli.Bullet("Services:     " + cli.Highlight("Redis, RabbitMQ, Mailpit")))
	fmt.Println(cli.Bullet("Sample Data:  " + cli.Highlight("Yes")))
	fmt.Println(cli.Bullet("Directory:    " + cli.Highlight(projectDir)))
	fmt.Println(cli.Bullet("Domain:       " + cli.Highlight(domainInput)))
	fmt.Println()

	// Create project directory and config first
	cli.PrintTitle("Installing " + selectedVersion.Name)
	fmt.Println()

	// Get the correct PHP binary for this version
	phpBin := p.PHPBinary(selectedPHP)

	// Step 1: Create directory
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Step 2: Create .magebox.yaml first so our wrapper uses correct PHP
	cli.PrintInfo("Creating MageBox configuration...")

	mageboxConfig := fmt.Sprintf(`name: %s
domains:
  - host: %s
    root: pub
    ssl: true
php: "%s"
services:
  mysql: "%s"
  opensearch:
    version: "%s"
    memory: "2g"
  redis: true
  rabbitmq: true
  mailpit: true

commands:
  setup:
    description: "Run Magento setup"
    run: "php bin/magento setup:upgrade && php bin/magento cache:flush"
  reindex:
    description: "Reindex all indexes"
    run: "php bin/magento indexer:reindex"
  deploy:
    description: "Deploy static content"
    run: "php bin/magento setup:static-content:deploy -f"
  cache:
    description: "Flush all caches"
    run: "php bin/magento cache:flush"
`, projectName, domainInput, selectedPHP, dbVersion, searchVersion)

	mageboxFile := filepath.Join(projectDir, config.ConfigFileName)
	if err := os.WriteFile(mageboxFile, []byte(mageboxConfig), 0644); err != nil {
		cli.PrintWarning("Failed to create %s file: %v", config.ConfigFileName, err)
	} else {
		fmt.Printf("  Created %s\n", cli.Highlight(config.ConfigFileName))
	}

	// Step 3: Create composer.json and install MageOS
	fmt.Println()
	cli.PrintInfo("Installing MageOS via Composer...")
	fmt.Printf("  Using PHP %s: %s\n", selectedPHP, phpBin)

	// Create composer.json from proper template
	composerJSON, err := templates.GenerateMageOSComposerJSON(projectName, selectedVersion.Version)
	if err != nil {
		return fmt.Errorf("failed to generate composer.json: %w", err)
	}

	composerJSONFile := filepath.Join(projectDir, "composer.json")
	if err := os.WriteFile(composerJSONFile, composerJSON, 0644); err != nil {
		return fmt.Errorf("failed to create composer.json: %w", err)
	}

	// Run composer install using our wrapper
	wrapperPath := filepath.Join(p.MageBoxDir(), "bin", "composer")
	compInstallCmd := exec.Command(wrapperPath, "install")
	compInstallCmd.Dir = projectDir
	compInstallCmd.Stdout = os.Stdout
	compInstallCmd.Stderr = os.Stderr
	compInstallCmd.Stdin = os.Stdin

	if err := compInstallCmd.Run(); err != nil {
		cli.PrintError("Composer install failed: %v", err)
		return err
	}

	// Step 4: Start MageBox services
	fmt.Println()
	cli.PrintInfo("Starting MageBox services...")

	// Use project manager to start all services (nginx, php-fpm, docker)
	projectMgr := project.NewManager(p)
	startResult, err := projectMgr.Start(projectDir)
	if err != nil {
		cli.PrintWarning("Failed to start services: %v", err)
	} else {
		for _, svc := range startResult.Services {
			fmt.Printf("  %s %s\n", cli.Success("✓"), svc)
		}
		for _, warn := range startResult.Warnings {
			cli.PrintWarning("%s", warn)
		}
	}

	// Reload nginx to pick up new vhost
	nginxCtrl := nginx.NewController(p)
	if err := nginxCtrl.Reload(); err != nil {
		cli.PrintWarning("Failed to reload nginx: %v", err)
	}

	fmt.Println("  Services started " + cli.Success("✓"))

	// Wait for OpenSearch to be ready
	fmt.Println()
	cli.PrintInfo("Waiting for OpenSearch to be ready...")
	dbPort := "33080" // MySQL 8.0 default port
	opensearchURL := fmt.Sprintf("http://127.0.0.1:%d", OpenSearchDefaultPort)
	for i := 0; i < OpenSearchReadinessMaxRetries; i++ {
		checkCmd := exec.Command("curl", "-s", "-o", "/dev/null", "-w", "%{http_code}", opensearchURL)
		output, err := checkCmd.Output()
		if err == nil && string(output) == "200" {
			fmt.Println("  OpenSearch ready " + cli.Success("✓"))
			break
		}
		if i == OpenSearchReadinessMaxRetries-1 {
			cli.PrintWarning("OpenSearch may not be ready, continuing anyway...")
		}
		time.Sleep(OpenSearchReadinessRetryInterval)
	}

	// Step 6: Run Magento setup:install
	fmt.Println()
	cli.PrintInfo("Running Magento setup:install (this may take several minutes)...")

	setupArgs := []string{
		"bin/magento", "setup:install",
		"--base-url=https://" + domainInput,
		"--backend-frontname=admin",
		"--db-host=127.0.0.1:" + dbPort,
		"--db-name=" + projectName,
		fmt.Sprintf("--db-user=%s", DefaultDBUser),
		fmt.Sprintf("--db-password=%s", DefaultDBPassword),
		"--admin-firstname=Admin",
		"--admin-lastname=User",
		fmt.Sprintf("--admin-email=%s", DefaultAdminEmail),
		fmt.Sprintf("--admin-user=%s", DefaultAdminUser),
		fmt.Sprintf("--admin-password=%s", DefaultAdminPassword),
		"--language=en_US",
		"--currency=USD",
		"--timezone=America/New_York",
		"--use-rewrites=1",
		"--search-engine=opensearch",
		"--opensearch-host=127.0.0.1",
		fmt.Sprintf("--opensearch-port=%d", OpenSearchDefaultPort),
		"--opensearch-index-prefix=magento2",
		"--opensearch-timeout=15",
		"--session-save=redis",
		"--session-save-redis-host=127.0.0.1",
		fmt.Sprintf("--session-save-redis-port=%d", RedisDefaultPort),
		fmt.Sprintf("--session-save-redis-db=%d", RedisSessionDB),
		"--cache-backend=redis",
		"--cache-backend-redis-server=127.0.0.1",
		fmt.Sprintf("--cache-backend-redis-port=%d", RedisDefaultPort),
		fmt.Sprintf("--cache-backend-redis-db=%d", RedisCacheDB),
		"--page-cache=redis",
		"--page-cache-redis-server=127.0.0.1",
		fmt.Sprintf("--page-cache-redis-port=%d", RedisDefaultPort),
		fmt.Sprintf("--page-cache-redis-db=%d", RedisFullPageCacheDB),
		"--amqp-host=127.0.0.1",
		fmt.Sprintf("--amqp-port=%d", RabbitMQDefaultPort),
		fmt.Sprintf("--amqp-user=%s", RabbitMQDefaultUser),
		fmt.Sprintf("--amqp-password=%s", RabbitMQDefaultPass),
	}

	setupCmd := exec.Command(wrapperPath, setupArgs...)
	setupCmd.Dir = projectDir
	setupCmd.Stdout = os.Stdout
	setupCmd.Stderr = os.Stderr
	setupCmd.Stdin = os.Stdin

	if err := setupCmd.Run(); err != nil {
		cli.PrintError("Magento setup:install failed: %v", err)
		fmt.Println()
		fmt.Println("You can retry manually with:")
		fmt.Println("  cd " + cli.Highlight(projectDir))
		fmt.Println("  " + cli.Command("php "+strings.Join(setupArgs, " ")))
		return err
	}

	fmt.Println("  Magento installed " + cli.Success("✓"))

	// Step 7: Deploy sample data
	fmt.Println()
	cli.PrintInfo("Deploying sample data...")

	sampleDataCmd := exec.Command(wrapperPath, "bin/magento", "sampledata:deploy")
	sampleDataCmd.Dir = projectDir
	sampleDataCmd.Stdout = os.Stdout
	sampleDataCmd.Stderr = os.Stderr
	sampleDataCmd.Stdin = os.Stdin
	if err := sampleDataCmd.Run(); err != nil {
		cli.PrintWarning("sampledata:deploy failed: %v", err)
	}

	// Step 8: Run setup:upgrade
	fmt.Println()
	cli.PrintInfo("Running setup:upgrade...")

	upgradeCmd := exec.Command(wrapperPath, "bin/magento", "setup:upgrade")
	upgradeCmd.Dir = projectDir
	upgradeCmd.Stdout = os.Stdout
	upgradeCmd.Stderr = os.Stderr
	if err := upgradeCmd.Run(); err != nil {
		cli.PrintWarning("setup:upgrade failed: %v", err)
	}

	// Step 9: Reindex
	fmt.Println()
	cli.PrintInfo("Running indexer:reindex...")

	reindexCmd := exec.Command(wrapperPath, "bin/magento", "indexer:reindex")
	reindexCmd.Dir = projectDir
	reindexCmd.Stdout = os.Stdout
	reindexCmd.Stderr = os.Stderr
	if err := reindexCmd.Run(); err != nil {
		cli.PrintWarning("indexer:reindex failed: %v", err)
	}

	// Step 10: Flush cache
	fmt.Println()
	cli.PrintInfo("Flushing cache...")

	cacheCmd := exec.Command(wrapperPath, "bin/magento", "cache:flush")
	cacheCmd.Dir = projectDir
	cacheCmd.Stdout = os.Stdout
	cacheCmd.Stderr = os.Stderr
	if err := cacheCmd.Run(); err != nil {
		cli.PrintWarning("cache:flush failed: %v", err)
	}

	// Success!
	fmt.Println()
	cli.PrintTitle("Installation Complete!")
	fmt.Println()
	cli.PrintSuccess("MageOS project installed successfully!")
	fmt.Println()
	fmt.Println("Your store is ready at: " + cli.URL("https://"+domainInput))
	fmt.Println("Admin panel: " + cli.URL("https://"+domainInput+"/admin"))
	fmt.Println()
	fmt.Println("Admin credentials:")
	fmt.Println("  Username: " + cli.Highlight(DefaultAdminUser))
	fmt.Println("  Password: " + cli.Highlight(DefaultAdminPassword))
	fmt.Println()

	return nil
}

// titleCase converts a string to title case (first letter uppercase)
// Used instead of deprecated strings.Title
func titleCase(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

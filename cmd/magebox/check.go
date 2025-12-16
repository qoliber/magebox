// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/qoliber/magebox/internal/cli"
	"github.com/qoliber/magebox/internal/config"
	"github.com/qoliber/magebox/internal/docker"
	"github.com/qoliber/magebox/internal/nginx"
	"github.com/qoliber/magebox/internal/php"
	"github.com/qoliber/magebox/internal/platform"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check project health and dependencies",
	Long: `Performs health checks on the current MageBox project.

Checks include:
  - PHP version availability and extensions
  - Nginx configuration and status
  - Docker services (MySQL, Redis, OpenSearch, etc.)
  - SSL certificates
  - Database connectivity
  - Project configuration

Run this command to diagnose issues with your development environment.`,
	RunE: runCheck,
}

func init() {
	rootCmd.AddCommand(checkCmd)
}

type checkResult struct {
	name    string
	status  string // "ok", "warning", "error"
	message string
}

func runCheck(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	p, err := getPlatform()
	if err != nil {
		return err
	}

	cli.PrintTitle("MageBox Health Check")
	fmt.Println()

	results := make([]checkResult, 0)

	// Check 1: Project configuration
	fmt.Println(cli.Header("Project Configuration"))
	cfg, cfgErr := config.LoadFromPath(cwd)
	if cfgErr != nil {
		results = append(results, checkResult{
			name:    "Project Config",
			status:  "error",
			message: fmt.Sprintf("No %s found - run 'magebox init' first", config.ConfigFileName),
		})
		printCheckResult(results[len(results)-1])
	} else {
		results = append(results, checkResult{
			name:    "Project Config",
			status:  "ok",
			message: fmt.Sprintf("Found %s (project: %s)", config.ConfigFileName, cfg.Name),
		})
		printCheckResult(results[len(results)-1])

		// Check domains
		if len(cfg.Domains) > 0 {
			domains := make([]string, 0)
			for _, d := range cfg.Domains {
				domains = append(domains, d.Host)
			}
			results = append(results, checkResult{
				name:    "Domains",
				status:  "ok",
				message: strings.Join(domains, ", "),
			})
		} else {
			results = append(results, checkResult{
				name:    "Domains",
				status:  "warning",
				message: "No domains configured",
			})
		}
		printCheckResult(results[len(results)-1])
	}
	fmt.Println()

	// Check 2: PHP
	fmt.Println(cli.Header("PHP"))
	detector := php.NewDetector(p)

	if cfg != nil && cfg.PHP != "" {
		phpVersion := detector.Detect(cfg.PHP)
		if phpVersion.Installed {
			results = append(results, checkResult{
				name:    fmt.Sprintf("PHP %s", cfg.PHP),
				status:  "ok",
				message: phpVersion.PHPBinary,
			})
		} else {
			results = append(results, checkResult{
				name:    fmt.Sprintf("PHP %s", cfg.PHP),
				status:  "error",
				message: fmt.Sprintf("Not installed - run 'brew install php@%s'", cfg.PHP),
			})
		}
		printCheckResult(results[len(results)-1])

		// Check PHP-FPM
		if phpVersion.Installed {
			fpmCtrl := php.NewFPMController(p, cfg.PHP)
			if fpmCtrl.IsRunning() {
				results = append(results, checkResult{
					name:    "PHP-FPM",
					status:  "ok",
					message: "Running",
				})
			} else {
				results = append(results, checkResult{
					name:    "PHP-FPM",
					status:  "warning",
					message: "Not running - will start on 'magebox start'",
				})
			}
			printCheckResult(results[len(results)-1])
		}

		// Check required extensions
		results = append(results, checkPHPExtensions(p, cfg.PHP)...)
	} else {
		// Check installed versions
		installed := detector.DetectInstalled()
		if len(installed) > 0 {
			versions := make([]string, 0)
			for _, v := range installed {
				versions = append(versions, v.Version)
			}
			results = append(results, checkResult{
				name:    "PHP Versions",
				status:  "ok",
				message: strings.Join(versions, ", "),
			})
		} else {
			results = append(results, checkResult{
				name:    "PHP",
				status:  "error",
				message: "No PHP versions installed",
			})
		}
		printCheckResult(results[len(results)-1])
	}
	fmt.Println()

	// Check 3: Nginx
	fmt.Println(cli.Header("Nginx"))
	nginxCtrl := nginx.NewController(p)

	if nginxCtrl.IsRunning() {
		results = append(results, checkResult{
			name:    "Nginx",
			status:  "ok",
			message: "Running",
		})
	} else {
		results = append(results, checkResult{
			name:    "Nginx",
			status:  "warning",
			message: "Not running - run 'brew services start nginx'",
		})
	}
	printCheckResult(results[len(results)-1])

	// Check nginx config
	if err := nginxCtrl.Test(); err != nil {
		results = append(results, checkResult{
			name:    "Nginx Config",
			status:  "error",
			message: "Configuration test failed",
		})
	} else {
		results = append(results, checkResult{
			name:    "Nginx Config",
			status:  "ok",
			message: "Valid",
		})
	}
	printCheckResult(results[len(results)-1])

	// Check vhost exists (check for upstream file which is unique per project)
	if cfg != nil {
		upstreamPath := filepath.Join(p.MageBoxDir(), "nginx", "vhosts", cfg.Name+"-upstream.conf")
		if _, err := os.Stat(upstreamPath); err == nil {
			results = append(results, checkResult{
				name:    "Project Vhost",
				status:  "ok",
				message: filepath.Join(p.MageBoxDir(), "nginx", "vhosts", cfg.Name+"-*.conf"),
			})
		} else {
			results = append(results, checkResult{
				name:    "Project Vhost",
				status:  "warning",
				message: "Not found - run 'magebox start'",
			})
		}
		printCheckResult(results[len(results)-1])
	}
	fmt.Println()

	// Check 4: Docker Services
	fmt.Println(cli.Header("Docker Services"))

	// Check Docker running
	dockerCmd := exec.Command("docker", "info")
	if dockerCmd.Run() != nil {
		results = append(results, checkResult{
			name:    "Docker",
			status:  "error",
			message: "Not running - start Docker Desktop",
		})
		printCheckResult(results[len(results)-1])
	} else {
		results = append(results, checkResult{
			name:    "Docker",
			status:  "ok",
			message: "Running",
		})
		printCheckResult(results[len(results)-1])

		// Check individual services
		composeGen := docker.NewComposeGenerator(p)
		dockerCtrl := docker.NewDockerController(composeGen.ComposeFilePath())

		services := []struct {
			name        string
			serviceName string
			port        int
		}{
			{"MySQL 8.0", "mysql80", 33080},
			{"Redis", "redis", 6379},
			{"Mailpit", "mailpit", 8025},
		}

		for _, svc := range services {
			if dockerCtrl.IsServiceRunning(svc.serviceName) {
				results = append(results, checkResult{
					name:    svc.name,
					status:  "ok",
					message: fmt.Sprintf("Running (port %d)", svc.port),
				})
			} else {
				results = append(results, checkResult{
					name:    svc.name,
					status:  "warning",
					message: "Not running",
				})
			}
			printCheckResult(results[len(results)-1])
		}

		// Check project-specific services
		if cfg != nil {
			if cfg.Services.HasOpenSearch() {
				if dockerCtrl.IsServiceRunning("opensearch") {
					results = append(results, checkResult{
						name:    "OpenSearch",
						status:  "ok",
						message: "Running (port 9200)",
					})
				} else {
					results = append(results, checkResult{
						name:    "OpenSearch",
						status:  "warning",
						message: "Not running",
					})
				}
				printCheckResult(results[len(results)-1])
			}

			if cfg.Services.HasRabbitMQ() {
				if dockerCtrl.IsServiceRunning("rabbitmq") {
					results = append(results, checkResult{
						name:    "RabbitMQ",
						status:  "ok",
						message: "Running (port 5672)",
					})
				} else {
					results = append(results, checkResult{
						name:    "RabbitMQ",
						status:  "warning",
						message: "Not running",
					})
				}
				printCheckResult(results[len(results)-1])
			}
		}
	}
	fmt.Println()

	// Check 5: Database connectivity
	if cfg != nil && cfg.Services.HasMySQL() {
		fmt.Println(cli.Header("Database"))
		dbResult := checkDatabaseConnection(p, cfg)
		results = append(results, dbResult)
		printCheckResult(dbResult)
		fmt.Println()
	}

	// Check 6: SSL Certificates
	fmt.Println(cli.Header("SSL Certificates"))

	// Check mkcert installation
	mkcertInstalled := platform.CommandExists("mkcert")
	if mkcertInstalled {
		results = append(results, checkResult{
			name:    "mkcert",
			status:  "ok",
			message: "Installed",
		})
	} else {
		results = append(results, checkResult{
			name:    "mkcert",
			status:  "error",
			message: fmt.Sprintf("Not installed - run '%s'", p.MkcertInstallCommand()),
		})
	}
	printCheckResult(results[len(results)-1])

	// Check if local CA is installed
	if mkcertInstalled {
		caRoot := getMkcertCARoot()
		if caRoot != "" {
			rootCAPath := filepath.Join(caRoot, "rootCA.pem")
			if _, err := os.Stat(rootCAPath); err == nil {
				results = append(results, checkResult{
					name:    "Local CA",
					status:  "ok",
					message: "Installed and trusted",
				})
			} else {
				results = append(results, checkResult{
					name:    "Local CA",
					status:  "warning",
					message: "Not installed - run 'mkcert -install'",
				})
			}
			printCheckResult(results[len(results)-1])
		}
	}

	// Check domain certificates
	if cfg != nil && len(cfg.Domains) > 0 {
		for _, domain := range cfg.Domains {
			if domain.IsSSLEnabled() {
				// Certificates are stored in ~/.magebox/certs/{domain}/cert.pem
				certPath := filepath.Join(p.MageBoxDir(), "certs", domain.Host, "cert.pem")
				if _, err := os.Stat(certPath); err == nil {
					results = append(results, checkResult{
						name:    domain.Host,
						status:  "ok",
						message: "Certificate exists",
					})
				} else {
					results = append(results, checkResult{
						name:    domain.Host,
						status:  "warning",
						message: "Certificate missing - will be created on 'magebox start'",
					})
				}
				printCheckResult(results[len(results)-1])
			}
		}
	}
	fmt.Println()

	// Check 7: DNS Resolution
	if cfg != nil && len(cfg.Domains) > 0 {
		fmt.Println(cli.Header("DNS Resolution"))
		for _, domain := range cfg.Domains {
			addrs, err := net.LookupHost(domain.Host)
			if err == nil && len(addrs) > 0 {
				if addrs[0] == "127.0.0.1" || addrs[0] == "::1" {
					results = append(results, checkResult{
						name:    domain.Host,
						status:  "ok",
						message: "Resolves to localhost",
					})
				} else {
					results = append(results, checkResult{
						name:    domain.Host,
						status:  "warning",
						message: fmt.Sprintf("Resolves to %s (expected 127.0.0.1)", addrs[0]),
					})
				}
			} else {
				results = append(results, checkResult{
					name:    domain.Host,
					status:  "error",
					message: "Does not resolve - check /etc/hosts or dnsmasq",
				})
			}
			printCheckResult(results[len(results)-1])
		}
		fmt.Println()
	}

	// Summary
	printCheckSummary(results)

	return nil
}

func printCheckResult(r checkResult) {
	var icon string
	switch r.status {
	case "ok":
		icon = cli.Success("✓")
	case "warning":
		icon = cli.Warning("!")
	case "error":
		icon = cli.Error("✗")
	}
	fmt.Printf("  %s %-20s %s\n", icon, r.name+":", r.message)
}

func printCheckSummary(results []checkResult) {
	var okCount, warnCount, errCount int
	for _, r := range results {
		switch r.status {
		case "ok":
			okCount++
		case "warning":
			warnCount++
		case "error":
			errCount++
		}
	}

	fmt.Println(cli.Header("Summary"))
	fmt.Printf("  %s %d passed\n", cli.Success("✓"), okCount)
	if warnCount > 0 {
		fmt.Printf("  %s %d warnings\n", cli.Warning("!"), warnCount)
	}
	if errCount > 0 {
		fmt.Printf("  %s %d errors\n", cli.Error("✗"), errCount)
	}
	fmt.Println()

	if errCount > 0 {
		cli.PrintError("Some checks failed. Run 'magebox start' to fix most issues.")
	} else if warnCount > 0 {
		cli.PrintWarning("Some warnings found. Run 'magebox start' to fix.")
	} else {
		cli.PrintSuccess("All checks passed!")
	}
}

func checkPHPExtensions(p *platform.Platform, version string) []checkResult {
	results := make([]checkResult, 0)

	// Required extensions for Magento
	requiredExtensions := []string{
		"bcmath", "ctype", "curl", "dom", "gd", "hash", "iconv",
		"intl", "mbstring", "openssl", "pdo_mysql", "simplexml",
		"soap", "xsl", "zip", "sockets",
	}

	phpBin := p.PHPBinary(version)
	if !platform.BinaryExists(phpBin) {
		return results
	}

	// Get list of loaded extensions
	cmd := exec.Command(phpBin, "-m")
	output, err := cmd.Output()
	if err != nil {
		return results
	}

	loadedExtensions := strings.ToLower(string(output))
	missing := make([]string, 0)

	for _, ext := range requiredExtensions {
		if !strings.Contains(loadedExtensions, strings.ToLower(ext)) {
			missing = append(missing, ext)
		}
	}

	if len(missing) == 0 {
		results = append(results, checkResult{
			name:    "PHP Extensions",
			status:  "ok",
			message: "All required extensions loaded",
		})
	} else {
		results = append(results, checkResult{
			name:    "PHP Extensions",
			status:  "warning",
			message: fmt.Sprintf("Missing: %s", strings.Join(missing, ", ")),
		})
	}
	printCheckResult(results[len(results)-1])

	return results
}

// getMkcertCARoot returns the mkcert CA root directory
func getMkcertCARoot() string {
	cmd := exec.Command("mkcert", "-CAROOT")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func checkDatabaseConnection(p *platform.Platform, cfg *config.Config) checkResult {
	composeGen := docker.NewComposeGenerator(p)
	composeFile := composeGen.ComposeFilePath()

	var serviceName string
	var port string

	if cfg.Services.HasMySQL() {
		serviceName = fmt.Sprintf("mysql%s", strings.ReplaceAll(cfg.Services.MySQL.Version, ".", ""))
		switch cfg.Services.MySQL.Version {
		case "8.4":
			port = "33084"
		default:
			port = "33080"
		}
	} else if cfg.Services.HasMariaDB() {
		serviceName = fmt.Sprintf("mariadb%s", strings.ReplaceAll(cfg.Services.MariaDB.Version, ".", ""))
		port = "33106"
	}

	// Try to connect via TCP
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%s", port), 2*time.Second)
	if err != nil {
		return checkResult{
			name:    "Database",
			status:  "error",
			message: fmt.Sprintf("Cannot connect to port %s", port),
		}
	}
	conn.Close()

	// Check if database exists
	dockerCtrl := docker.NewDockerController(composeFile)
	if dockerCtrl.DatabaseExists(serviceName, cfg.Name) {
		return checkResult{
			name:    "Database",
			status:  "ok",
			message: fmt.Sprintf("Connected, database '%s' exists", cfg.Name),
		}
	}

	return checkResult{
		name:    "Database",
		status:  "warning",
		message: fmt.Sprintf("Connected, but database '%s' not found", cfg.Name),
	}
}

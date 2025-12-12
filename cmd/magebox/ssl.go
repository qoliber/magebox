package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/qoliber/magebox/internal/ssl"
)

var sslCmd = &cobra.Command{
	Use:   "ssl",
	Short: "SSL certificate management",
	Long:  "Manage SSL certificates for project domains",
}

var sslTrustCmd = &cobra.Command{
	Use:   "trust",
	Short: "Trust local CA",
	Long:  "Installs and trusts the local certificate authority",
	RunE:  runSslTrust,
}

var sslGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate certificates",
	Long:  "Generates SSL certificates for project domains",
	RunE:  runSslGenerate,
}

func init() {
	sslCmd.AddCommand(sslTrustCmd)
	sslCmd.AddCommand(sslGenerateCmd)
	rootCmd.AddCommand(sslCmd)
}

func runSslTrust(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	sslMgr := ssl.NewManager(p)

	if !sslMgr.IsMkcertInstalled() {
		fmt.Println("mkcert is not installed")
		fmt.Println()
		fmt.Println("Install it with:")
		fmt.Printf("  %s\n", p.MkcertInstallCommand())
		return nil
	}

	fmt.Println("Installing and trusting local CA...")

	if err := sslMgr.EnsureCAInstalled(); err != nil {
		return err
	}

	fmt.Println("Local CA is now trusted!")
	fmt.Println("SSL certificates will be automatically generated when you run 'magebox start'")

	return nil
}

func runSslGenerate(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	p, err := getPlatform()
	if err != nil {
		return err
	}

	cfg, ok := loadProjectConfig(cwd)
	if !ok {
		return nil
	}

	sslMgr := ssl.NewManager(p)

	if !sslMgr.IsMkcertInstalled() {
		fmt.Println("mkcert is not installed")
		fmt.Println()
		fmt.Println("Install it with:")
		fmt.Printf("  %s\n", p.MkcertInstallCommand())
		fmt.Println()
		fmt.Println("Then run: magebox ssl:trust")
		return nil
	}

	fmt.Println("Generating SSL certificates...")

	for _, domain := range cfg.Domains {
		if domain.IsSSLEnabled() {
			baseDomain := ssl.ExtractBaseDomain(domain.Host)
			fmt.Printf("  %s... ", baseDomain)
			cert, err := sslMgr.GenerateCert(baseDomain)
			if err != nil {
				fmt.Printf("failed: %v\n", err)
				continue
			}
			fmt.Printf("done\n")
			fmt.Printf("    Cert: %s\n", cert.CertFile)
			fmt.Printf("    Key:  %s\n", cert.KeyFile)
		}
	}

	fmt.Println("\nSSL certificates generated!")
	return nil
}

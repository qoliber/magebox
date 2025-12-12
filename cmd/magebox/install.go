package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/qoliber/magebox/internal/cli"
	"github.com/qoliber/magebox/internal/php"
	"github.com/qoliber/magebox/internal/platform"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install MageBox dependencies",
	Long:  "Checks and installs required dependencies for MageBox",
	RunE:  runInstall,
}

func init() {
	rootCmd.AddCommand(installCmd)
}

func runInstall(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	cli.PrintTitle("MageBox Installation Check")
	fmt.Println()

	allOk := true

	// Check Docker
	dockerInstalled := platform.CommandExists("docker")
	fmt.Printf("%-12s %s\n", "Docker:", cli.StatusInstalled(dockerInstalled))
	if !dockerInstalled {
		fmt.Printf("  Install: %s\n", cli.Command(p.DockerInstallCommand()))
		allOk = false
	}

	// Check Nginx
	nginxInstalled := p.IsNginxInstalled()
	fmt.Printf("%-12s %s\n", "Nginx:", cli.StatusInstalled(nginxInstalled))
	if !nginxInstalled {
		fmt.Printf("  Install: %s\n", cli.Command(p.NginxInstallCommand()))
		allOk = false
	}

	// Check mkcert
	mkcertInstalled := platform.CommandExists("mkcert")
	fmt.Printf("%-12s %s\n", "mkcert:", cli.StatusInstalled(mkcertInstalled))
	if !mkcertInstalled {
		fmt.Printf("  Install: %s\n", cli.Command(p.MkcertInstallCommand()))
		allOk = false
	}

	// Check PHP versions
	fmt.Println(cli.Header("PHP Versions"))
	detector := php.NewDetector(p)
	installedPHP := false
	for _, v := range php.SupportedVersions {
		version := detector.Detect(v)
		if version.Installed {
			fmt.Printf("  PHP %s: %s\n", v, cli.StatusInstalled(true))
			installedPHP = true
		}
	}
	if !installedPHP {
		cli.PrintWarning("No PHP versions installed!")
		fmt.Printf("  Install at least one version:\n")
		fmt.Printf("    %s\n", cli.Command(p.PHPInstallCommand("8.2")))
		allOk = false
	}

	fmt.Println()
	if allOk {
		cli.PrintSuccess("All dependencies are installed!")
		fmt.Println()
		cli.PrintInfo("Next steps:")
		fmt.Println(cli.Bullet("Run " + cli.Command("magebox ssl trust") + " to set up SSL"))
		fmt.Println(cli.Bullet("Run " + cli.Command("magebox init") + " in your project directory"))
		fmt.Println(cli.Bullet("Run " + cli.Command("magebox start") + " to start your project"))
	} else {
		cli.PrintWarning("Some dependencies are missing. Install them and run '%s' again.", cli.Command("magebox install"))
	}

	return nil
}

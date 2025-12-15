package testing

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Installer handles installation of testing tools via composer
type Installer struct {
	manager *Manager
}

// NewInstaller creates a new installer
func NewInstaller(m *Manager) *Installer {
	return &Installer{manager: m}
}

// InstallPHPUnit installs PHPUnit via composer
func (i *Installer) InstallPHPUnit() error {
	return i.installPackages("phpunit", ComposerPackages["phpunit"])
}

// InstallPHPStan installs PHPStan and Magento extension via composer
func (i *Installer) InstallPHPStan() error {
	return i.installPackages("phpstan", ComposerPackages["phpstan"])
}

// InstallPHPCS installs PHP_CodeSniffer and Magento Coding Standard via composer
func (i *Installer) InstallPHPCS() error {
	return i.installPackages("phpcs", ComposerPackages["phpcs"])
}

// InstallPHPMD installs PHP Mess Detector via composer
func (i *Installer) InstallPHPMD() error {
	return i.installPackages("phpmd", ComposerPackages["phpmd"])
}

// InstallAll installs all testing tools
func (i *Installer) InstallAll() error {
	allPackages := []string{}
	for _, packages := range ComposerPackages {
		allPackages = append(allPackages, packages...)
	}
	return i.installPackages("all testing tools", allPackages)
}

// InstallSelected installs selected tools based on options
func (i *Installer) InstallSelected(opts SetupOptions) error {
	packages := []string{}

	if opts.InstallPHPUnit {
		packages = append(packages, ComposerPackages["phpunit"]...)
	}
	if opts.InstallPHPStan {
		packages = append(packages, ComposerPackages["phpstan"]...)
	}
	if opts.InstallPHPCS {
		packages = append(packages, ComposerPackages["phpcs"]...)
	}
	if opts.InstallPHPMD {
		packages = append(packages, ComposerPackages["phpmd"]...)
	}

	if len(packages) == 0 {
		return fmt.Errorf("no packages selected for installation")
	}

	return i.installPackages("selected tools", packages)
}

// installPackages installs the given composer packages
func (i *Installer) installPackages(toolName string, packages []string) error {
	if len(packages) == 0 {
		return nil
	}

	// Build composer require command
	args := []string{i.manager.GetComposerBinary(), "require", "--dev"}
	args = append(args, packages...)

	fmt.Printf("Installing %s...\n", toolName)
	fmt.Printf("Running: composer require --dev %s\n", strings.Join(packages, " "))

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = i.manager.projectPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Set PHP version for composer
	phpBin := i.manager.GetPHPBinary()
	cmd.Env = append(os.Environ(), fmt.Sprintf("PHP_BINARY=%s", phpBin))

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install %s: %w", toolName, err)
	}

	return nil
}

// UninstallPHPUnit removes PHPUnit via composer
func (i *Installer) UninstallPHPUnit() error {
	return i.uninstallPackages("phpunit", ComposerPackages["phpunit"])
}

// UninstallPHPStan removes PHPStan via composer
func (i *Installer) UninstallPHPStan() error {
	return i.uninstallPackages("phpstan", ComposerPackages["phpstan"])
}

// UninstallPHPCS removes PHP_CodeSniffer via composer
func (i *Installer) UninstallPHPCS() error {
	return i.uninstallPackages("phpcs", ComposerPackages["phpcs"])
}

// UninstallPHPMD removes PHP Mess Detector via composer
func (i *Installer) UninstallPHPMD() error {
	return i.uninstallPackages("phpmd", ComposerPackages["phpmd"])
}

// uninstallPackages removes the given composer packages
func (i *Installer) uninstallPackages(toolName string, packages []string) error {
	if len(packages) == 0 {
		return nil
	}

	args := []string{i.manager.GetComposerBinary(), "remove", "--dev"}
	args = append(args, packages...)

	fmt.Printf("Removing %s...\n", toolName)

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = i.manager.projectPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove %s: %w", toolName, err)
	}

	return nil
}

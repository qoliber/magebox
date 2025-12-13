// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package bootstrap

import (
	"fmt"
	"os/exec"

	"github.com/qoliber/magebox/internal/bootstrap/installer"
	"github.com/qoliber/magebox/internal/platform"
)

// Bootstrapper orchestrates the MageBox bootstrap process
type Bootstrapper struct {
	platform  *platform.Platform
	installer installer.Installer
	progress  *installer.BootstrapProgress
}

// NewBootstrapper creates a new bootstrapper for the detected platform
func NewBootstrapper(p *platform.Platform) (*Bootstrapper, error) {
	var inst installer.Installer

	switch p.Type {
	case platform.Darwin:
		inst = installer.NewDarwinInstaller(p)
	case platform.Linux:
		switch p.LinuxDistro {
		case platform.DistroFedora:
			inst = installer.NewFedoraInstaller(p)
		case platform.DistroDebian:
			inst = installer.NewUbuntuInstaller(p)
		case platform.DistroArch:
			inst = installer.NewArchInstaller(p)
		default:
			return nil, fmt.Errorf("unsupported Linux distribution: %s", p.LinuxDistro)
		}
	default:
		return nil, fmt.Errorf("unsupported platform: %s", p.Type)
	}

	return &Bootstrapper{
		platform:  p,
		installer: inst,
		progress:  installer.NewProgress(10), // 10 main bootstrap steps
	}, nil
}

// ValidateOS validates the OS version is supported
func (b *Bootstrapper) ValidateOS() (installer.OSVersionInfo, error) {
	return b.installer.ValidateOSVersion()
}

// GetInstaller returns the platform-specific installer
func (b *Bootstrapper) GetInstaller() installer.Installer {
	return b.installer
}

// GetProgress returns the current bootstrap progress
func (b *Bootstrapper) GetProgress() *installer.BootstrapProgress {
	return b.progress
}

// CheckDependency checks if a command is available
func (b *Bootstrapper) CheckDependency(name string) bool {
	return platform.CommandExists(name)
}

// CheckDockerRunning checks if Docker daemon is running
func (b *Bootstrapper) CheckDockerRunning() bool {
	cmd := exec.Command("docker", "info")
	return cmd.Run() == nil
}

// InstallDependency installs a single dependency
func (b *Bootstrapper) InstallDependency(name string) error {
	switch name {
	case "nginx":
		return b.installer.InstallNginx()
	case "mkcert":
		return b.installer.InstallMkcert()
	case "dnsmasq":
		return b.installer.InstallDnsmasq()
	default:
		return fmt.Errorf("unknown dependency: %s", name)
	}
}

// InstallPHP installs a specific PHP version
func (b *Bootstrapper) InstallPHP(version string) error {
	if err := b.installer.InstallPHP(version); err != nil {
		return err
	}
	b.progress.PHPInstalled = append(b.progress.PHPInstalled, version)
	return nil
}

// ConfigurePHPFPM configures PHP-FPM for installed versions
func (b *Bootstrapper) ConfigurePHPFPM(versions []string) error {
	return b.installer.ConfigurePHPFPM(versions)
}

// ConfigureNginx configures Nginx for MageBox
func (b *Bootstrapper) ConfigureNginx() error {
	return b.installer.ConfigureNginx()
}

// ConfigureSudoers sets up passwordless sudo (Linux only)
func (b *Bootstrapper) ConfigureSudoers() error {
	return b.installer.ConfigureSudoers()
}

// SetupDNS configures DNS for .test domains
func (b *Bootstrapper) SetupDNS() error {
	return b.installer.SetupDNS()
}

// DockerInstallInstructions returns Docker installation instructions
func (b *Bootstrapper) DockerInstallInstructions() string {
	return b.installer.InstallDocker()
}

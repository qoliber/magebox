// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package installer

import (
	"fmt"
	"os"
	"os/user"
	"strings"

	libconfig "qoliber/magebox/internal/lib/config"
	"qoliber/magebox/internal/platform"
)

// GenericInstaller is a YAML-config driven installer
type GenericInstaller struct {
	BaseInstaller
	config     *libconfig.InstallerConfig
	loader     *libconfig.Loader
	platformID string // "fedora", "ubuntu", "arch", "darwin"
}

// NewGenericInstaller creates a new config-driven installer
func NewGenericInstaller(p *platform.Platform, loader *libconfig.Loader, platformID string) (*GenericInstaller, error) {
	config, err := loader.LoadInstaller(platformID)
	if err != nil {
		return nil, fmt.Errorf("failed to load installer config for %s: %w", platformID, err)
	}

	// Setup OS-level variables
	loader.SetupOSVariables()

	return &GenericInstaller{
		BaseInstaller: BaseInstaller{Platform: p},
		config:        config,
		loader:        loader,
		platformID:    platformID,
	}, nil
}

// Platform returns the platform type
func (g *GenericInstaller) Platform() platform.Type {
	if g.config.Meta.Platform == "darwin" {
		return platform.Darwin
	}
	return platform.Linux
}

// Distro returns the Linux distribution
func (g *GenericInstaller) Distro() platform.LinuxDistro {
	switch g.config.Meta.Distro {
	case "fedora":
		return platform.DistroFedora
	case "ubuntu", "debian":
		return platform.DistroDebian
	case "arch":
		return platform.DistroArch
	default:
		return platform.DistroUnknown
	}
}

// ValidateOSVersion checks if the current OS version is supported
func (g *GenericInstaller) ValidateOSVersion() (OSVersionInfo, error) {
	info := OSVersionInfo{
		Name: g.config.Meta.DisplayName,
	}

	// Get current version
	info.Version = g.loader.Variables().Get("osVersion")

	// Check if supported
	for _, v := range g.config.Meta.GetSupportedVersions() {
		if info.Version == v || v == "rolling" {
			info.Supported = true
			break
		}
	}

	if !info.Supported && info.Version != "" {
		info.Message = fmt.Sprintf("%s %s may work but is not officially tested", info.Name, info.Version)
	}

	return info, nil
}

// PackageManager returns the package manager name
func (g *GenericInstaller) PackageManager() string {
	return g.config.PackageManager.Name
}

// InstallCommand returns the install command format
func (g *GenericInstaller) InstallCommand(packages ...string) string {
	cmd := g.loader.Expand(g.config.PackageManager.Install)
	return cmd + " " + strings.Join(packages, " ")
}

// InstallPrerequisites installs system prerequisites
func (g *GenericInstaller) InstallPrerequisites() error {
	// Install prerequisite packages
	if len(g.config.Prerequisites.Packages) > 0 {
		packages := g.loader.ExpandSlice(g.config.Prerequisites.Packages)
		installCmd := g.config.PackageManager.Install
		args := strings.Fields(installCmd)
		args = append(args, packages...)

		// Remove "sudo" prefix if present - we'll add it ourselves
		if args[0] == "sudo" {
			args = args[1:]
		}

		if err := g.RunSudo(args...); err != nil {
			return fmt.Errorf("failed to install prerequisites: %w", err)
		}
	}

	// Install required repositories
	for _, repo := range g.config.Repositories {
		checkPath := g.loader.Expand(repo.Check)
		if !g.FileExists(checkPath) {
			installCmd := g.loader.Expand(repo.Install)
			if err := g.RunCommand(installCmd); err != nil {
				return fmt.Errorf("failed to install repository %s: %w", repo.Name, err)
			}
		}
	}

	return nil
}

// InstallPHP installs a specific PHP version
func (g *GenericInstaller) InstallPHP(version string) error {
	packages := g.loader.GetPHPPackages(g.config, version)
	if len(packages) == 0 {
		// macOS: single formula
		if g.config.PHP.Packages.Formula != "" {
			g.loader.SetupPHPVariables(g.config, version)
			formula := g.loader.Expand(g.config.PHP.Packages.Formula)
			packages = []string{formula}
		}
	}

	if len(packages) == 0 {
		return fmt.Errorf("no packages defined for PHP %s", version)
	}

	installCmd := g.config.PackageManager.Install
	args := strings.Fields(installCmd)
	args = append(args, packages...)

	// Remove "sudo" prefix if present
	if args[0] == "sudo" {
		args = args[1:]
		return g.RunSudo(args...)
	}

	return g.RunCommand(strings.Join(args, " "))
}

// InstallNginx installs Nginx
func (g *GenericInstaller) InstallNginx() error {
	packages := g.config.Nginx.Packages
	if len(packages) == 0 {
		return nil
	}

	installCmd := g.config.PackageManager.Install
	args := strings.Fields(installCmd)
	args = append(args, packages...)

	if args[0] == "sudo" {
		args = args[1:]
		return g.RunSudo(args...)
	}

	return g.RunCommand(strings.Join(args, " "))
}

// InstallMkcert installs mkcert
func (g *GenericInstaller) InstallMkcert() error {
	packages := g.config.Tools.Mkcert.Packages
	if len(packages) == 0 {
		return nil
	}

	installCmd := g.config.PackageManager.Install
	args := strings.Fields(installCmd)
	args = append(args, packages...)

	if args[0] == "sudo" {
		args = args[1:]
		return g.RunSudo(args...)
	}

	return g.RunCommand(strings.Join(args, " "))
}

// InstallDocker returns Docker installation instructions
func (g *GenericInstaller) InstallDocker() string {
	return g.loader.Expand(g.config.Tools.Docker.InstallInstructions)
}

// InstallDnsmasq installs dnsmasq
func (g *GenericInstaller) InstallDnsmasq() error {
	packages := g.config.Tools.Dnsmasq.Packages
	if len(packages) == 0 {
		return nil
	}

	installCmd := g.config.PackageManager.Install
	args := strings.Fields(installCmd)
	args = append(args, packages...)

	if args[0] == "sudo" {
		args = args[1:]
		return g.RunSudo(args...)
	}

	return g.RunCommand(strings.Join(args, " "))
}

// InstallMultitail installs multitail
func (g *GenericInstaller) InstallMultitail() error {
	packages := g.config.Tools.Multitail.Packages
	if len(packages) == 0 {
		return nil
	}

	installCmd := g.config.PackageManager.Install
	args := strings.Fields(installCmd)
	args = append(args, packages...)

	if args[0] == "sudo" {
		args = args[1:]
		return g.RunSudo(args...)
	}

	return g.RunCommand(strings.Join(args, " "))
}

// InstallXdebug installs Xdebug for a specific PHP version
func (g *GenericInstaller) InstallXdebug(version string) error {
	g.loader.SetupPHPVariables(g.config, version)

	optional := g.config.PHP.Optional
	if optional == nil {
		return nil
	}

	xdebug, ok := optional["xdebug"]
	if !ok {
		return nil
	}

	// Handle string value (package name)
	if pkg, ok := xdebug.(string); ok {
		expandedPkg := g.loader.Expand(pkg)
		installCmd := g.config.PackageManager.Install
		args := strings.Fields(installCmd)
		args = append(args, expandedPkg)

		if args[0] == "sudo" {
			args = args[1:]
			return g.RunSudo(args...)
		}
		return g.RunCommand(strings.Join(args, " "))
	}

	return nil
}

// InstallImagick installs ImageMagick PHP extension
func (g *GenericInstaller) InstallImagick(version string) error {
	g.loader.SetupPHPVariables(g.config, version)

	optional := g.config.PHP.Optional
	if optional == nil {
		return nil
	}

	imagick, ok := optional["imagick"]
	if !ok {
		return nil
	}

	// Handle string value (package name)
	if pkg, ok := imagick.(string); ok {
		expandedPkg := g.loader.Expand(pkg)
		installCmd := g.config.PackageManager.Install
		args := strings.Fields(installCmd)
		args = append(args, expandedPkg)

		if args[0] == "sudo" {
			args = args[1:]
			return g.RunSudo(args...)
		}
		return g.RunCommand(strings.Join(args, " "))
	}

	return nil
}

// InstallSodium installs sodium PHP extension
func (g *GenericInstaller) InstallSodium(version string) error {
	g.loader.SetupPHPVariables(g.config, version)

	optional := g.config.PHP.Optional
	if optional == nil {
		return nil
	}

	sodium, ok := optional["sodium"]
	if !ok {
		return nil
	}

	// Handle string value (package name)
	if pkg, ok := sodium.(string); ok {
		expandedPkg := g.loader.Expand(pkg)
		installCmd := g.config.PackageManager.Install
		args := strings.Fields(installCmd)
		args = append(args, expandedPkg)

		if args[0] == "sudo" {
			args = args[1:]
			return g.RunSudo(args...)
		}
		return g.RunCommand(strings.Join(args, " "))
	}

	return nil
}

// InstallBlackfire installs Blackfire profiler
func (g *GenericInstaller) InstallBlackfire(versions []string) error {
	bf := g.config.Profiling.Blackfire

	// Import GPG key if specified
	if bf.GPGImport != "" {
		if err := g.RunCommand(g.loader.Expand(bf.GPGImport)); err != nil {
			return fmt.Errorf("failed to import Blackfire GPG key: %w", err)
		}
	}

	// Install repository if specified
	if bf.Repository.Install != "" {
		if err := g.RunCommand(g.loader.Expand(bf.Repository.Install)); err != nil {
			return fmt.Errorf("failed to install Blackfire repository: %w", err)
		}
	}

	// macOS: tap homebrew
	if bf.TapInstall != "" {
		if err := g.RunCommand(g.loader.Expand(bf.TapInstall)); err != nil {
			return fmt.Errorf("failed to tap Blackfire: %w", err)
		}
	}

	// Install packages
	packages := bf.GetPackages()
	if len(packages) > 0 {
		installCmd := g.config.PackageManager.Install
		args := strings.Fields(installCmd)
		args = append(args, packages...)

		if args[0] == "sudo" {
			args = args[1:]
			if err := g.RunSudo(args...); err != nil {
				return err
			}
		} else {
			if err := g.RunCommand(strings.Join(args, " ")); err != nil {
				return err
			}
		}
	}

	return nil
}

// InstallTideways installs Tideways profiler
func (g *GenericInstaller) InstallTideways(versions []string) error {
	tw := g.config.Profiling.Tideways

	// Import GPG key if specified
	if tw.GPGImport != "" {
		if err := g.RunCommand(g.loader.Expand(tw.GPGImport)); err != nil {
			return fmt.Errorf("failed to import Tideways GPG key: %w", err)
		}
	}

	// Install packages
	if tw.Install != "" {
		var allPackages []string
		for _, pkg := range tw.Packages {
			allPackages = append(allPackages, pkg)
		}
		g.loader.Variables().SetPackages(strings.Join(allPackages, " "))
		installCmd := g.loader.Expand(tw.Install)
		if err := g.RunCommand(installCmd); err != nil {
			return err
		}
	}

	return nil
}

// ConfigurePHPFPM configures PHP-FPM for the platform
func (g *GenericInstaller) ConfigurePHPFPM(versions []string) error {
	// This is typically handled by the bootstrap process using templates
	// The GenericInstaller provides the paths and service commands
	return nil
}

// ConfigureNginx configures Nginx for MageBox
func (g *GenericInstaller) ConfigureNginx() error {
	// Set nginx user if command is specified
	if g.config.Nginx.Configuration.SetUser != "" {
		currentUser, _ := user.Current()
		if currentUser != nil {
			g.loader.Variables().Set("user", currentUser.Username)
		}
		cmd := g.loader.Expand(g.config.Nginx.Configuration.SetUser)
		_ = g.RunCommand(cmd) // Ignore errors, user might already be set
	}

	// Apply fixes
	for name := range g.config.Nginx.Fixes {
		commands := g.loader.GetNginxFixCommands(g.config, name)
		for _, cmd := range commands {
			_ = g.RunCommand(cmd) // Best effort
		}
	}

	return nil
}

// ConfigureSudoers sets up passwordless sudo for services
func (g *GenericInstaller) ConfigureSudoers() error {
	if !g.config.Sudoers.Enabled {
		return nil
	}

	rules := g.loader.GetSudoersRules(g.config)
	if len(rules) == 0 {
		return nil
	}

	content := "# MageBox sudoers configuration\n"
	content += "# Generated by MageBox - do not edit manually\n\n"
	for _, rule := range rules {
		content += rule + "\n"
	}

	sudoersFile := g.loader.Expand(g.config.Sudoers.File)
	return g.WriteFile(sudoersFile, content)
}

// ConfigureSELinux configures SELinux for nginx
func (g *GenericInstaller) ConfigureSELinux() error {
	if !g.config.SELinux.Enabled {
		return nil
	}

	// Check if SELinux is available
	if g.config.SELinux.Check != "" {
		if err := g.RunCommandSilent(g.config.SELinux.Check); err != nil {
			return nil // SELinux not available
		}
	}

	// Set booleans
	for _, boolean := range g.config.SELinux.Booleans {
		cmd := g.loader.Expand(boolean.Command)
		_ = g.RunCommand(cmd) // Best effort
	}

	// Set contexts
	contexts := g.loader.GetSELinuxContexts(g.config)
	for _, ctx := range contexts {
		g.loader.Variables().SetSELinuxContext(ctx.Path, ctx.Type, ctx.Pattern)

		// Create directory if needed
		if !g.FileExists(ctx.Path) {
			_ = os.MkdirAll(ctx.Path, 0755)
		}

		// Apply semanage
		semanageCmd := g.loader.Expand(g.config.SELinux.Commands.Semanage)
		_ = g.RunCommand(semanageCmd)

		// Apply restorecon
		restoreconCmd := g.loader.Expand(g.config.SELinux.Commands.Restorecon)
		if err := g.RunCommand(restoreconCmd); err != nil {
			// Try chcon fallback
			chconCmd := g.loader.Expand(g.config.SELinux.Commands.ChconFallback)
			_ = g.RunCommand(chconCmd)
		}
	}

	return nil
}

// SetupDNS configures DNS resolution for .test domains
func (g *GenericInstaller) SetupDNS() error {
	// Run dnsmasq setup commands
	for _, step := range g.config.DNS.Dnsmasq.Setup {
		cmd := g.loader.Expand(step.Command)
		_ = g.RunCommand(cmd)
	}

	// Write dnsmasq config if template exists
	if g.config.DNS.Dnsmasq.ConfigTemplate != "" {
		configContent := g.loader.Expand(g.config.DNS.Dnsmasq.ConfigTemplate)
		configDir := g.loader.Expand(g.config.DNS.Dnsmasq.ConfigDir)
		configFile := configDir + "/magebox.conf"

		_ = os.MkdirAll(configDir, 0755)
		if err := g.WriteFile(configFile, configContent); err != nil {
			return err
		}
	}

	// Configure systemd-resolved if available
	if g.config.DNS.SystemdResolved.Check != "" {
		if err := g.RunCommandSilent(g.config.DNS.SystemdResolved.Check); err == nil {
			// systemd-resolved is active
			if g.config.DNS.SystemdResolved.ConfigTemplate != "" {
				configContent := g.loader.Expand(g.config.DNS.SystemdResolved.ConfigTemplate)
				configDir := g.loader.Expand(g.config.DNS.SystemdResolved.ConfigDir)

				_ = g.RunSudo("mkdir", "-p", configDir)
				configFile := configDir + "/magebox.conf"
				if err := g.WriteFile(configFile, configContent); err != nil {
					return err
				}

				// Restart systemd-resolved
				if g.config.DNS.SystemdResolved.Restart != "" {
					_ = g.RunCommand(g.config.DNS.SystemdResolved.Restart)
				}
			}
		}
	}

	// Restart dnsmasq
	if g.config.DNS.Dnsmasq.Services.Restart != "" {
		_ = g.RunCommand(g.loader.Expand(g.config.DNS.Dnsmasq.Services.Restart))
	}

	return nil
}

// ConfigurePHPINI sets Magento-friendly PHP INI defaults
func (g *GenericInstaller) ConfigurePHPINI(versions []string) error {
	for _, version := range versions {
		g.loader.SetupPHPVariables(g.config, version)
		iniPath := g.loader.Expand(g.config.PHP.Paths.INI)

		if !g.FileExists(iniPath) {
			continue
		}

		for key, value := range g.config.PHP.INISettings {
			// Use sed to update INI settings
			sedCmd := fmt.Sprintf("sed -i 's/^%s\\s*=.*/%s = %s/' %s", key, key, value, iniPath)
			_ = g.RunSudo("sh", "-c", sedCmd)
		}
	}

	return nil
}

// GetConfig returns the underlying installer config
func (g *GenericInstaller) GetConfig() *libconfig.InstallerConfig {
	return g.config
}

// GetLoader returns the config loader
func (g *GenericInstaller) GetLoader() *libconfig.Loader {
	return g.loader
}

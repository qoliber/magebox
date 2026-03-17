// Copyright (c) qoliber

package php

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"qoliber/magebox/internal/platform"
)

// ExtensionMapping defines how an extension name maps to platform-specific packages
type ExtensionMapping struct {
	Ubuntu   string // e.g., "php${v}-redis" where ${v} is "8.3"
	Fedora   string // e.g., "php${vnd}-php-pecl-redis" where ${vnd} is "83"
	Arch     string // e.g., "php-redis" (single version, no prefix)
	PeclName string // fallback for macOS (always pecl) and platforms without a package
}

// knownExtensions maps canonical extension names to platform-specific package names.
// Template variables: ${v} = "8.3" (dotted), ${vnd} = "83" (no dot).
var knownExtensions = map[string]ExtensionMapping{
	"redis":     {Ubuntu: "php${v}-redis", Fedora: "php${vnd}-php-pecl-redis", Arch: "php-redis", PeclName: "redis"},
	"xdebug":    {Ubuntu: "php${v}-xdebug", Fedora: "php${vnd}-php-xdebug", Arch: "xdebug", PeclName: "xdebug"},
	"imagick":   {Ubuntu: "php${v}-imagick", Fedora: "php${vnd}-php-pecl-imagick-im7", Arch: "php-imagick", PeclName: "imagick"},
	"memcached": {Ubuntu: "php${v}-memcached", Fedora: "php${vnd}-php-pecl-memcached", Arch: "php-memcached", PeclName: "memcached"},
	"apcu":      {Ubuntu: "php${v}-apcu", Fedora: "php${vnd}-php-pecl-apcu", Arch: "php-apcu", PeclName: "apcu"},
	"mongodb":   {Ubuntu: "php${v}-mongodb", Fedora: "php${vnd}-php-pecl-mongodb", Arch: "", PeclName: "mongodb"},
	"mailparse": {Ubuntu: "php${v}-mailparse", Fedora: "php${vnd}-php-pecl-mailparse", Arch: "", PeclName: "mailparse"},
	"pgsql":     {Ubuntu: "php${v}-pgsql", Fedora: "php${vnd}-php-pgsql", Arch: "php-pgsql", PeclName: "pgsql"},
	"sqlite3":   {Ubuntu: "php${v}-sqlite3", Fedora: "php${vnd}-php-sqlite3", Arch: "php-sqlite", PeclName: "sqlite3"},
	"gd":        {Ubuntu: "php${v}-gd", Fedora: "php${vnd}-php-gd", Arch: "php-gd", PeclName: ""},
	"intl":      {Ubuntu: "php${v}-intl", Fedora: "php${vnd}-php-intl", Arch: "php-intl", PeclName: ""},
	"bcmath":    {Ubuntu: "php${v}-bcmath", Fedora: "php${vnd}-php-bcmath", Arch: "", PeclName: ""},
	"soap":      {Ubuntu: "php${v}-soap", Fedora: "php${vnd}-php-soap", Arch: "", PeclName: ""},
	"zip":       {Ubuntu: "php${v}-zip", Fedora: "php${vnd}-php-zip", Arch: "", PeclName: ""},
	"curl":      {Ubuntu: "php${v}-curl", Fedora: "php${vnd}-php-curl", Arch: "", PeclName: ""},
	"mbstring":  {Ubuntu: "php${v}-mbstring", Fedora: "php${vnd}-php-mbstring", Arch: "", PeclName: ""},
	"xml":       {Ubuntu: "php${v}-xml", Fedora: "php${vnd}-php-xml", Arch: "", PeclName: ""},
	"sodium":    {Ubuntu: "php${v}-sodium", Fedora: "php${vnd}-php-sodium", Arch: "php-sodium", PeclName: ""},
	"opcache":   {Ubuntu: "php${v}-opcache", Fedora: "php${vnd}-php-opcache", Arch: "", PeclName: ""},
	"mysql":     {Ubuntu: "php${v}-mysql", Fedora: "php${vnd}-php-mysqlnd", Arch: "", PeclName: ""},
	"amqp":      {Ubuntu: "php${v}-amqp", Fedora: "php${vnd}-php-pecl-amqp", Arch: "", PeclName: "amqp"},
	"grpc":      {Ubuntu: "php${v}-grpc", Fedora: "php${vnd}-php-pecl-grpc", Arch: "", PeclName: "grpc"},
	"excimer":   {Ubuntu: "php${v}-excimer", Fedora: "php${vnd}-php-pecl-excimer", Arch: "", PeclName: "excimer"},
	"tidy":      {Ubuntu: "php${v}-tidy", Fedora: "php${vnd}-php-tidy", Arch: "php-tidy", PeclName: ""},
	"ldap":      {Ubuntu: "php${v}-ldap", Fedora: "php${vnd}-php-ldap", Arch: "", PeclName: ""},
}

// IsPIEPackage returns true if the extension name looks like a PIE/Composer
// vendor/package identifier (e.g., "noisebynorthwest/php-spx").
func IsPIEPackage(name string) bool {
	return strings.Contains(name, "/")
}

// ExtensionManager handles PHP extension installation and management
type ExtensionManager struct {
	platform *platform.Platform
}

// NewExtensionManager creates a new extension manager
func NewExtensionManager(p *platform.Platform) *ExtensionManager {
	return &ExtensionManager{platform: p}
}

// IsPIEInstalled checks whether the PIE binary is available
func (m *ExtensionManager) IsPIEInstalled() bool {
	_, err := exec.LookPath("pie")
	return err == nil
}

// PIEInstallHint returns platform-specific instructions for installing PIE
func (m *ExtensionManager) PIEInstallHint() string {
	return "curl -fsSL https://github.com/php/pie/releases/latest/download/pie.phar -o pie.phar && sudo mv pie.phar /usr/local/bin/pie && sudo chmod +x /usr/local/bin/pie"
}

// InstallPIE downloads and installs PIE to /usr/local/bin
func (m *ExtensionManager) InstallPIE() error {
	// Download pie.phar
	cmd := exec.Command("curl", "-fsSL",
		"https://github.com/php/pie/releases/latest/download/pie.phar",
		"-o", "/tmp/pie.phar")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to download PIE: %s", strings.TrimSpace(string(output)))
	}

	// Move to /usr/local/bin
	cmd = exec.Command("sudo", "mv", "/tmp/pie.phar", "/usr/local/bin/pie")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to install PIE: %s", strings.TrimSpace(string(output)))
	}

	// Make executable
	cmd = exec.Command("sudo", "chmod", "+x", "/usr/local/bin/pie")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to make PIE executable: %s", strings.TrimSpace(string(output)))
	}

	return nil
}

// EnsurePHPDev ensures the PHP development package (phpize, php-config) is
// installed for the given PHP version. PIE requires these to compile extensions.
func (m *ExtensionManager) EnsurePHPDev(phpVersion string) error {
	// Check if php-config is already available
	phpConfigBin := m.phpConfigBinary(phpVersion)
	if _, err := exec.LookPath(phpConfigBin); err == nil {
		return nil
	}
	// Also check the full path
	if platform.BinaryExists(phpConfigBin) {
		return nil
	}

	v := normalizeVersion(phpVersion)
	vnd := strings.ReplaceAll(v, ".", "")

	switch m.platform.Type {
	case platform.Darwin:
		// Homebrew PHP includes php-config by default
		return nil
	case platform.Linux:
		switch m.platform.LinuxDistro {
		case platform.DistroDebian:
			return m.installViaPackageManager(fmt.Sprintf("php%s-dev", v))
		case platform.DistroFedora:
			return m.installViaPackageManager(fmt.Sprintf("php%s-php-devel", vnd))
		case platform.DistroArch:
			// Arch includes phpize/php-config in the base php package
			return nil
		}
	}
	return nil
}

// InstallViaPIE installs an extension using PIE for the given PHP version.
// The package must be in vendor/name format (e.g., "noisebynorthwest/php-spx").
func (m *ExtensionManager) InstallViaPIE(pkg, phpVersion string) error {
	phpConfigBin := m.phpConfigBinary(phpVersion)

	cmd := exec.Command("sudo", "pie", "install", "--with-php-config="+phpConfigBin, pkg)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pie install failed: %w", err)
	}
	return nil
}

// RemoveViaPIE removes a PIE-installed extension for the given PHP version.
// PIE has no remove command, so we manually remove the .so, INI file, and conf.d symlinks.
func (m *ExtensionManager) RemoveViaPIE(pkg, phpVersion string) error {
	// Derive the extension name from vendor/package (e.g., "noisebynorthwest/php-spx" -> "spx")
	extName := m.pieExtensionName(pkg)
	if extName == "" {
		return fmt.Errorf("could not determine extension name from package %s", pkg)
	}

	v := normalizeVersion(phpVersion)
	phpConfigBin := m.phpConfigBinary(phpVersion)

	// Get extension directory from php-config
	extDir := ""
	if cmd, err := exec.Command(phpConfigBin, "--extension-dir").Output(); err == nil {
		extDir = strings.TrimSpace(string(cmd))
	}

	var errors []string

	// Remove the .so file
	if extDir != "" {
		soFile := filepath.Join(extDir, extName+".so")
		if _, err := os.Stat(soFile); err == nil {
			if err := runSudo("rm", "-f", soFile); err != nil {
				errors = append(errors, fmt.Sprintf("failed to remove %s: %v", soFile, err))
			}
		}
	}

	// Platform-specific INI cleanup
	switch m.platform.Type {
	case platform.Darwin:
		// macOS Homebrew: INI files in the conf.d directory
		base := "/usr/local"
		if m.platform.IsAppleSilicon {
			base = "/opt/homebrew"
		}
		confDir := filepath.Join(base, "etc", "php", v, "conf.d")
		m.removeINIFiles(confDir, extName, &errors)

	case platform.Linux:
		switch m.platform.LinuxDistro {
		case platform.DistroDebian:
			// Ondrej: mods-available + conf.d symlinks per SAPI
			modsAvail := fmt.Sprintf("/etc/php/%s/mods-available", v)
			m.removeINIFiles(modsAvail, extName, &errors)
			// Remove symlinks from all SAPIs (cli, fpm, apache2, etc.)
			sapiDirs, _ := filepath.Glob(fmt.Sprintf("/etc/php/%s/*/conf.d", v))
			for _, confDir := range sapiDirs {
				m.removeINIFiles(confDir, extName, &errors)
			}
		case platform.DistroFedora:
			vnd := strings.ReplaceAll(v, ".", "")
			confDir := fmt.Sprintf("/etc/opt/remi/php%s/php.d", vnd)
			m.removeINIFiles(confDir, extName, &errors)
		case platform.DistroArch:
			confDir := "/etc/php/conf.d"
			m.removeINIFiles(confDir, extName, &errors)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("removal incomplete: %s", strings.Join(errors, "; "))
	}
	return nil
}

// removeINIFiles removes INI files (and symlinks) matching an extension name in a directory.
func (m *ExtensionManager) removeINIFiles(dir, extName string, errors *[]string) {
	// Match patterns like "80-spx.ini", "spx.ini", "20-spx.ini"
	patterns := []string{
		filepath.Join(dir, fmt.Sprintf("*-%s.ini", extName)),
		filepath.Join(dir, fmt.Sprintf("%s.ini", extName)),
	}
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		for _, match := range matches {
			if err := runSudo("rm", "-f", match); err != nil {
				*errors = append(*errors, fmt.Sprintf("failed to remove %s: %v", match, err))
			}
		}
	}
}

// pieExtensionName derives the PHP extension name from a PIE vendor/package string.
// e.g., "noisebynorthwest/php-spx" -> "spx", "openswoole/openswoole" -> "openswoole"
func (m *ExtensionManager) pieExtensionName(pkg string) string {
	parts := strings.SplitN(pkg, "/", 2)
	if len(parts) != 2 {
		return ""
	}
	name := parts[1]
	// Strip common "php-" prefix
	name = strings.TrimPrefix(name, "php-")
	return name
}

// runSudo runs a command with sudo.
func runSudo(args ...string) error {
	cmd := exec.Command("sudo", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// PIEInstallCommand returns the command string for installing via PIE.
func (m *ExtensionManager) PIEInstallCommand(pkg, phpVersion string) string {
	phpConfigBin := m.phpConfigBinary(phpVersion)
	return fmt.Sprintf("sudo pie install --with-php-config=%s %s", phpConfigBin, pkg)
}

// phpConfigBinary returns the path to the php-config binary for a specific version.
func (m *ExtensionManager) phpConfigBinary(phpVersion string) string {
	v := normalizeVersion(phpVersion)

	switch m.platform.Type {
	case platform.Darwin:
		base := "/usr/local"
		if m.platform.IsAppleSilicon {
			base = "/opt/homebrew"
		}
		return fmt.Sprintf("%s/opt/php@%s/bin/php-config", base, v)
	case platform.Linux:
		switch m.platform.LinuxDistro {
		case platform.DistroFedora:
			vnd := strings.ReplaceAll(v, ".", "")
			return fmt.Sprintf("/opt/remi/php%s/root/usr/bin/php-config", vnd)
		case platform.DistroArch:
			return "/usr/bin/php-config"
		default: // Debian/Ubuntu (Ondrej)
			return fmt.Sprintf("/usr/bin/php-config%s", v)
		}
	}
	return "php-config"
}

// Install installs one or more PHP extensions for the given PHP version
func (m *ExtensionManager) Install(extensions []string, phpVersion string) []error {
	var errs []error
	for _, ext := range extensions {
		if err := m.installOne(ext, phpVersion); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", ext, err))
		}
	}
	return errs
}

// Remove removes one or more PHP extensions for the given PHP version
func (m *ExtensionManager) Remove(extensions []string, phpVersion string) []error {
	var errs []error
	for _, ext := range extensions {
		if err := m.removeOne(ext, phpVersion); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", ext, err))
		}
	}
	return errs
}

// List returns all loaded PHP extensions for the given version
func (m *ExtensionManager) List(phpVersion string) ([]string, error) {
	binary := m.platform.PHPBinary(phpVersion)

	cmd := exec.Command(binary, "-m")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run php -m: %w", err)
	}

	var extensions []string
	for _, line := range strings.Split(string(output), "\n") {
		ext := strings.TrimSpace(line)
		if ext != "" && !strings.HasPrefix(ext, "[") {
			extensions = append(extensions, ext)
		}
	}
	return extensions, nil
}

// Search searches for available extension packages matching the query
func (m *ExtensionManager) Search(query, phpVersion string) ([]string, error) {
	v := normalizeVersion(phpVersion)
	vnd := strings.ReplaceAll(v, ".", "")

	var cmd *exec.Cmd
	switch m.platform.Type {
	case platform.Darwin:
		cmd = exec.Command("pecl", "search", query)
	case platform.Linux:
		switch m.platform.LinuxDistro {
		case platform.DistroFedora:
			cmd = exec.Command("dnf", "search", fmt.Sprintf("php%s-php-*%s*", vnd, query))
		case platform.DistroArch:
			cmd = exec.Command("pacman", "-Ss", fmt.Sprintf("php-%s", query))
		default:
			cmd = exec.Command("apt-cache", "search", fmt.Sprintf("php%s-%s", v, query))
		}
	default:
		return nil, fmt.Errorf("unsupported platform")
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("search failed: %s", strings.TrimSpace(string(output)))
	}

	var results []string
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			results = append(results, line)
		}
	}
	return results, nil
}

// ResolvePackageName returns the platform-specific package name for an extension.
// Returns the package name and whether pecl should be used instead.
func (m *ExtensionManager) ResolvePackageName(extName, phpVersion string) (packageName string, usePecl bool) {
	v := normalizeVersion(phpVersion)
	vnd := strings.ReplaceAll(v, ".", "")
	ext := strings.ToLower(extName)

	mapping, known := knownExtensions[ext]

	switch m.platform.Type {
	case platform.Darwin:
		// macOS always uses pecl
		if known && mapping.PeclName != "" {
			return mapping.PeclName, true
		}
		// Bundled extensions (gd, intl, etc.) don't need pecl
		if known && mapping.PeclName == "" {
			return ext, false
		}
		return ext, true

	case platform.Linux:
		switch m.platform.LinuxDistro {
		case platform.DistroFedora:
			if known && mapping.Fedora != "" {
				pkg := strings.ReplaceAll(mapping.Fedora, "${vnd}", vnd)
				return pkg, false
			}
			// Fallback: try convention
			return fmt.Sprintf("php%s-php-%s", vnd, ext), false

		case platform.DistroArch:
			if known && mapping.Arch != "" {
				return mapping.Arch, false
			}
			// Arch fallback to pecl if no package known
			if known && mapping.PeclName != "" {
				return mapping.PeclName, true
			}
			return fmt.Sprintf("php-%s", ext), false

		default: // Debian/Ubuntu
			if known && mapping.Ubuntu != "" {
				pkg := strings.ReplaceAll(mapping.Ubuntu, "${v}", v)
				return pkg, false
			}
			// Fallback: try convention
			return fmt.Sprintf("php%s-%s", v, ext), false
		}
	}

	return ext, true
}

// InstallCommand returns the shell command that would be used to install an extension.
// Useful for showing the user what will happen.
func (m *ExtensionManager) InstallCommand(extName, phpVersion string) string {
	pkg, usePecl := m.ResolvePackageName(extName, phpVersion)

	if usePecl {
		peclBin := m.peclBinary(phpVersion)
		return fmt.Sprintf("%s install %s", peclBin, pkg)
	}

	switch m.platform.Type {
	case platform.Linux:
		switch m.platform.LinuxDistro {
		case platform.DistroFedora:
			return fmt.Sprintf("sudo dnf install -y %s", pkg)
		case platform.DistroArch:
			return fmt.Sprintf("sudo pacman -S --noconfirm %s", pkg)
		default:
			return fmt.Sprintf("sudo apt install -y %s", pkg)
		}
	default:
		return fmt.Sprintf("pecl install %s", pkg)
	}
}

func (m *ExtensionManager) installOne(extName, phpVersion string) error {
	pkg, usePecl := m.ResolvePackageName(extName, phpVersion)

	if usePecl {
		return m.installViaPecl(pkg, phpVersion)
	}

	switch m.platform.Type {
	case platform.Darwin:
		// Bundled extension, nothing to install
		return nil
	case platform.Linux:
		return m.installViaPackageManager(pkg)
	default:
		return fmt.Errorf("unsupported platform")
	}
}

func (m *ExtensionManager) removeOne(extName, phpVersion string) error {
	pkg, usePecl := m.ResolvePackageName(extName, phpVersion)

	if usePecl {
		return m.removeViaPecl(pkg, phpVersion)
	}

	switch m.platform.Type {
	case platform.Linux:
		return m.removeViaPackageManager(pkg)
	default:
		return fmt.Errorf("removal not supported on this platform for package %s", pkg)
	}
}

func (m *ExtensionManager) installViaPackageManager(pkg string) error {
	var cmd *exec.Cmd
	switch m.platform.LinuxDistro {
	case platform.DistroFedora:
		cmd = exec.Command("sudo", "dnf", "install", "-y", pkg)
	case platform.DistroArch:
		cmd = exec.Command("sudo", "pacman", "-S", "--noconfirm", pkg)
	default:
		cmd = exec.Command("sudo", "apt", "install", "-y", pkg)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("package install failed: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

func (m *ExtensionManager) removeViaPackageManager(pkg string) error {
	var cmd *exec.Cmd
	switch m.platform.LinuxDistro {
	case platform.DistroFedora:
		cmd = exec.Command("sudo", "dnf", "remove", "-y", pkg)
	case platform.DistroArch:
		cmd = exec.Command("sudo", "pacman", "-Rns", "--noconfirm", pkg)
	default:
		cmd = exec.Command("sudo", "apt", "remove", "-y", pkg)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("package removal failed: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

func (m *ExtensionManager) installViaPecl(extName, phpVersion string) error {
	peclBin := m.peclBinary(phpVersion)

	cmd := exec.Command(peclBin, "install", extName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		outStr := strings.TrimSpace(string(output))
		// pecl returns error if already installed, that's OK
		if strings.Contains(outStr, "already installed") {
			return nil
		}
		return fmt.Errorf("pecl install failed: %s", outStr)
	}
	return nil
}

func (m *ExtensionManager) removeViaPecl(extName, phpVersion string) error {
	peclBin := m.peclBinary(phpVersion)

	cmd := exec.Command(peclBin, "uninstall", extName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pecl uninstall failed: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

func (m *ExtensionManager) peclBinary(phpVersion string) string {
	v := normalizeVersion(phpVersion)

	if m.platform.Type == platform.Darwin {
		base := "/usr/local"
		if m.platform.IsAppleSilicon {
			base = "/opt/homebrew"
		}
		return fmt.Sprintf("%s/opt/php@%s/bin/pecl", base, v)
	}

	// Linux: pecl is typically in PATH
	return "pecl"
}

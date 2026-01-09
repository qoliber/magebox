package php

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"qoliber/magebox/internal/platform"
)

// PHPINISystemSettings is the list of PHP_INI_SYSTEM settings that can only be set globally
// These cannot be set per-pool via php_admin_value, only in php.ini
var PHPINISystemSettings = map[string]bool{
	// OPcache system-level settings
	"opcache.enable_cli":                    true,
	"opcache.memory_consumption":            true,
	"opcache.interned_strings_buffer":       true,
	"opcache.max_accelerated_files":         true,
	"opcache.max_wasted_percentage":         true,
	"opcache.force_restart_timeout":         true,
	"opcache.log_verbosity_level":           true,
	"opcache.preferred_memory_model":        true,
	"opcache.protect_memory":                true,
	"opcache.mmap_base":                     true,
	"opcache.restrict_api":                  true,
	"opcache.file_update_protection":        true,
	"opcache.huge_code_pages":               true,
	"opcache.lockfile_path":                 true,
	"opcache.opt_debug_level":               true,
	"opcache.file_cache":                    true,
	"opcache.file_cache_only":               true,
	"opcache.file_cache_consistency_checks": true,
	"opcache.file_cache_fallback":           true,

	// OPcache preloading (PHP 7.4+)
	"opcache.preload":      true,
	"opcache.preload_user": true,

	// JIT settings (PHP 8.0+)
	"opcache.jit":                       true,
	"opcache.jit_buffer_size":           true,
	"opcache.jit_debug":                 true,
	"opcache.jit_bisect_limit":          true,
	"opcache.jit_prof_threshold":        true,
	"opcache.jit_max_root_traces":       true,
	"opcache.jit_max_side_traces":       true,
	"opcache.jit_max_exit_counters":     true,
	"opcache.jit_hot_loop":              true,
	"opcache.jit_hot_func":              true,
	"opcache.jit_hot_return":            true,
	"opcache.jit_hot_side_exit":         true,
	"opcache.jit_blacklist_root_trace":  true,
	"opcache.jit_blacklist_side_trace":  true,
	"opcache.jit_max_loop_unrolls":      true,
	"opcache.jit_max_recursive_calls":   true,
	"opcache.jit_max_recursive_returns": true,
	"opcache.jit_max_polymorphic_calls": true,
}

// SystemINIOwner tracks which project owns the system INI settings
type SystemINIOwner struct {
	ProjectName string            `json:"project_name"`
	ProjectPath string            `json:"project_path"`
	PHPVersion  string            `json:"php_version"`
	Settings    map[string]string `json:"settings"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// SystemINIManager manages PHP_INI_SYSTEM settings
type SystemINIManager struct {
	platform *platform.Platform
	phpDir   string
}

// NewSystemINIManager creates a new system INI manager
func NewSystemINIManager(p *platform.Platform) *SystemINIManager {
	return &SystemINIManager{
		platform: p,
		phpDir:   filepath.Join(p.MageBoxDir(), "php"),
	}
}

// IsSystemSetting returns true if the setting is a PHP_INI_SYSTEM setting
func IsSystemSetting(key string) bool {
	return PHPINISystemSettings[key]
}

// SeparateSettings separates PHP INI settings into system and pool settings
func SeparateSettings(settings map[string]string) (system, pool map[string]string) {
	system = make(map[string]string)
	pool = make(map[string]string)

	for key, value := range settings {
		if IsSystemSetting(key) {
			system[key] = value
		} else {
			pool[key] = value
		}
	}

	return system, pool
}

// GetSystemINIPath returns the path to the system INI file for a PHP version
func (m *SystemINIManager) GetSystemINIPath(phpVersion string) string {
	return filepath.Join(m.phpDir, fmt.Sprintf("php-system-%s.ini", phpVersion))
}

// GetOwnerPath returns the path to the owner tracking file for a PHP version
func (m *SystemINIManager) GetOwnerPath(phpVersion string) string {
	return filepath.Join(m.phpDir, fmt.Sprintf("php-system-%s.owner.json", phpVersion))
}

// GetCurrentOwner returns the current owner of system settings for a PHP version
func (m *SystemINIManager) GetCurrentOwner(phpVersion string) (*SystemINIOwner, error) {
	ownerPath := m.GetOwnerPath(phpVersion)
	data, err := os.ReadFile(ownerPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var owner SystemINIOwner
	if err := json.Unmarshal(data, &owner); err != nil {
		return nil, err
	}

	return &owner, nil
}

// WriteSystemINI writes system INI settings and updates ownership
// Returns the previous owner if settings were overwritten from another project
func (m *SystemINIManager) WriteSystemINI(phpVersion, projectName, projectPath string, settings map[string]string) (*SystemINIOwner, error) {
	if len(settings) == 0 {
		return nil, nil
	}

	// Ensure directory exists
	if err := os.MkdirAll(m.phpDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create php directory: %w", err)
	}

	// Get current owner
	previousOwner, _ := m.GetCurrentOwner(phpVersion)

	// Generate INI content
	var content strings.Builder
	content.WriteString(fmt.Sprintf("; MageBox PHP System Settings for PHP %s\n", phpVersion))
	content.WriteString(fmt.Sprintf("; Owner: %s\n", projectName))
	content.WriteString(fmt.Sprintf("; Path: %s\n", projectPath))
	content.WriteString(fmt.Sprintf("; Updated: %s\n", time.Now().Format(time.RFC3339)))
	content.WriteString(";\n")
	content.WriteString("; These are PHP_INI_SYSTEM settings that apply to ALL projects using this PHP version.\n")
	content.WriteString("; They can only be changed by restarting PHP-FPM with a project that defines them.\n")
	content.WriteString(";\n\n")

	// Sort keys for consistent output
	keys := make([]string, 0, len(settings))
	for k := range settings {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		content.WriteString(fmt.Sprintf("%s = %s\n", key, settings[key]))
	}

	// Write INI file
	iniPath := m.GetSystemINIPath(phpVersion)
	if err := os.WriteFile(iniPath, []byte(content.String()), 0644); err != nil {
		return nil, fmt.Errorf("failed to write system INI: %w", err)
	}

	// Write owner file
	owner := SystemINIOwner{
		ProjectName: projectName,
		ProjectPath: projectPath,
		PHPVersion:  phpVersion,
		Settings:    settings,
		UpdatedAt:   time.Now(),
	}
	ownerData, err := json.MarshalIndent(owner, "", "  ")
	if err != nil {
		return nil, err
	}
	ownerPath := m.GetOwnerPath(phpVersion)
	if err := os.WriteFile(ownerPath, ownerData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write owner file: %w", err)
	}

	// Return previous owner if different project
	if previousOwner != nil && previousOwner.ProjectName != projectName {
		return previousOwner, nil
	}

	return nil, nil
}

// ClearSystemINI removes system INI settings if owned by the specified project
func (m *SystemINIManager) ClearSystemINI(phpVersion, projectName string) error {
	owner, err := m.GetCurrentOwner(phpVersion)
	if err != nil {
		return err
	}

	// Only clear if this project owns it
	if owner == nil || owner.ProjectName != projectName {
		return nil
	}

	iniPath := m.GetSystemINIPath(phpVersion)
	ownerPath := m.GetOwnerPath(phpVersion)

	// Remove files (ignore errors if they don't exist)
	_ = os.Remove(iniPath)
	_ = os.Remove(ownerPath)

	return nil
}

// GetSystemSettingsList returns a sorted list of all PHP_INI_SYSTEM setting names
func GetSystemSettingsList() []string {
	keys := make([]string, 0, len(PHPINISystemSettings))
	for k := range PHPINISystemSettings {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// FormatOwnerWarning formats a warning message when system settings are being overwritten
func FormatOwnerWarning(previous *SystemINIOwner, newProject string, newSettings map[string]string) string {
	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("\n⚠  PHP system settings (PHP_INI_SYSTEM) are being changed\n"))
	msg.WriteString(fmt.Sprintf("   Previous owner: %s\n", previous.ProjectName))
	msg.WriteString(fmt.Sprintf("   Previous path:  %s\n", previous.ProjectPath))
	msg.WriteString(fmt.Sprintf("   New owner:      %s\n", newProject))
	msg.WriteString("\n   Settings being overwritten:\n")

	// Show what's changing
	allKeys := make(map[string]bool)
	for k := range previous.Settings {
		allKeys[k] = true
	}
	for k := range newSettings {
		allKeys[k] = true
	}

	keys := make([]string, 0, len(allKeys))
	for k := range allKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		oldVal, hadOld := previous.Settings[key]
		newVal, hasNew := newSettings[key]
		if hadOld && hasNew {
			if oldVal != newVal {
				msg.WriteString(fmt.Sprintf("   - %s: %s → %s\n", key, oldVal, newVal))
			}
		} else if hadOld {
			msg.WriteString(fmt.Sprintf("   - %s: %s → (removed)\n", key, oldVal))
		} else {
			msg.WriteString(fmt.Sprintf("   - %s: (new) %s\n", key, newVal))
		}
	}

	msg.WriteString("\n   Note: These settings apply to ALL projects using this PHP version.\n")
	msg.WriteString("   Run 'mbox php system' to see current system settings.\n")

	return msg.String()
}

// FormatSystemSettingsInfo formats information about current system settings for display
func (m *SystemINIManager) FormatSystemSettingsInfo(phpVersion string) string {
	owner, err := m.GetCurrentOwner(phpVersion)
	if err != nil || owner == nil {
		return fmt.Sprintf("PHP %s: No system settings configured", phpVersion)
	}

	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("PHP %s System Settings (PHP_INI_SYSTEM)\n", phpVersion))
	msg.WriteString(strings.Repeat("-", 50) + "\n")
	msg.WriteString(fmt.Sprintf("Owner:   %s\n", owner.ProjectName))
	msg.WriteString(fmt.Sprintf("Path:    %s\n", owner.ProjectPath))
	msg.WriteString(fmt.Sprintf("Updated: %s\n", owner.UpdatedAt.Format("2006-01-02 15:04:05")))
	msg.WriteString("\nSettings:\n")

	keys := make([]string, 0, len(owner.Settings))
	for k := range owner.Settings {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		msg.WriteString(fmt.Sprintf("  %s = %s\n", key, owner.Settings[key]))
	}

	return msg.String()
}

// GetPHPScanDir returns the PHP INI scan directory for a given PHP version
func (m *SystemINIManager) GetPHPScanDir(phpVersion string) string {
	switch m.platform.Type {
	case platform.Darwin:
		// macOS with Homebrew
		return fmt.Sprintf("/opt/homebrew/etc/php/%s/conf.d", phpVersion)
	case platform.Linux:
		switch m.platform.LinuxDistro {
		case platform.DistroFedora:
			// Remi PHP on Fedora/RHEL
			shortVersion := strings.ReplaceAll(phpVersion, ".", "")
			return fmt.Sprintf("/etc/opt/remi/php%s/php.d", shortVersion)
		case platform.DistroDebian:
			// Ondrej PPA on Ubuntu/Debian (Debian-based distros)
			return fmt.Sprintf("/etc/php/%s/fpm/conf.d", phpVersion)
		case platform.DistroArch:
			// Arch Linux
			return "/etc/php/conf.d"
		}
	}
	return ""
}

// GetSymlinkPath returns the path where the system INI should be symlinked
func (m *SystemINIManager) GetSymlinkPath(phpVersion string) string {
	scanDir := m.GetPHPScanDir(phpVersion)
	if scanDir == "" {
		return ""
	}
	return filepath.Join(scanDir, "99-magebox-system.ini")
}

// IsSymlinkActive checks if the system INI is symlinked to the PHP scan directory
func (m *SystemINIManager) IsSymlinkActive(phpVersion string) bool {
	symlinkPath := m.GetSymlinkPath(phpVersion)
	if symlinkPath == "" {
		return false
	}

	target, err := os.Readlink(symlinkPath)
	if err != nil {
		return false
	}

	expectedTarget := m.GetSystemINIPath(phpVersion)
	return target == expectedTarget
}

// GetEnableCommand returns the command to enable system INI settings
func (m *SystemINIManager) GetEnableCommand(phpVersion string) string {
	iniPath := m.GetSystemINIPath(phpVersion)
	symlinkPath := m.GetSymlinkPath(phpVersion)
	if symlinkPath == "" {
		return "# Unable to determine PHP scan directory for your platform"
	}
	return fmt.Sprintf("sudo ln -sf %s %s", iniPath, symlinkPath)
}

// GetDisableCommand returns the command to disable system INI settings
func (m *SystemINIManager) GetDisableCommand(phpVersion string) string {
	symlinkPath := m.GetSymlinkPath(phpVersion)
	if symlinkPath == "" {
		return "# Unable to determine PHP scan directory for your platform"
	}
	return fmt.Sprintf("sudo rm -f %s", symlinkPath)
}

// FormatActivationInstructions returns instructions for activating system INI settings
func (m *SystemINIManager) FormatActivationInstructions(phpVersion string, settings map[string]string) string {
	if len(settings) == 0 {
		return ""
	}

	var msg strings.Builder
	msg.WriteString("\n")
	msg.WriteString("┌─────────────────────────────────────────────────────────────────┐\n")
	msg.WriteString("│  PHP System Settings (PHP_INI_SYSTEM) Detected                  │\n")
	msg.WriteString("└─────────────────────────────────────────────────────────────────┘\n")
	msg.WriteString("\n")
	msg.WriteString("The following settings require system-level PHP configuration:\n\n")

	keys := make([]string, 0, len(settings))
	for k := range settings {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		msg.WriteString(fmt.Sprintf("  • %s = %s\n", key, settings[key]))
	}

	msg.WriteString("\n")
	msg.WriteString(fmt.Sprintf("Config file: %s\n", m.GetSystemINIPath(phpVersion)))
	msg.WriteString("\n")

	if m.IsSymlinkActive(phpVersion) {
		msg.WriteString("Status: ✓ Active (symlink exists)\n")
	} else {
		msg.WriteString("Status: ✗ Not active (requires symlink)\n")
		msg.WriteString("\n")
		msg.WriteString("To activate these settings, run:\n")
		msg.WriteString(fmt.Sprintf("  %s\n", m.GetEnableCommand(phpVersion)))
		msg.WriteString("\n")
		msg.WriteString("Then restart PHP-FPM:\n")
		msg.WriteString("  mbox restart php\n")
	}

	msg.WriteString("\n")
	msg.WriteString("Note: These settings apply to ALL projects using PHP " + phpVersion + "\n")

	return msg.String()
}

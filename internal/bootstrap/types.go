// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

// Package bootstrap provides the main bootstrap functionality for MageBox.
// Platform-specific installers are in the installer subpackage.
package bootstrap

import (
	"github.com/qoliber/magebox/internal/bootstrap/installer"
)

// Re-export types from installer package for convenience
type (
	// OSVersionInfo contains OS version details
	OSVersionInfo = installer.OSVersionInfo

	// Installer is the interface for platform-specific installation
	Installer = installer.Installer

	// InstallResult tracks the result of an installation step
	InstallResult = installer.InstallResult

	// BootstrapProgress tracks overall bootstrap progress
	BootstrapProgress = installer.BootstrapProgress
)

// Re-export variables from installer package
var (
	// SupportedVersions defines the OS versions supported by MageBox
	SupportedVersions = installer.SupportedVersions

	// PHPVersions defines PHP versions to install for Magento compatibility
	PHPVersions = installer.PHPVersions

	// RequiredPHPExtensions defines required PHP extensions for Magento
	RequiredPHPExtensions = installer.RequiredPHPExtensions

	// NewProgress creates a new bootstrap progress tracker
	NewProgress = installer.NewProgress
)

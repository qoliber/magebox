// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package templates

import (
	"encoding/json"
	"fmt"
)

// ComposerJSON represents a composer.json structure
type ComposerJSON struct {
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	Type             string                 `json:"type"`
	License          []string               `json:"license"`
	Version          string                 `json:"version"`
	Config           ComposerConfig         `json:"config"`
	Require          map[string]string      `json:"require"`
	RequireDev       map[string]string      `json:"require-dev,omitempty"`
	Conflict         map[string]string      `json:"conflict,omitempty"`
	Autoload         ComposerAutoload       `json:"autoload"`
	AutoloadDev      *ComposerAutoloadDev   `json:"autoload-dev,omitempty"`
	MinimumStability string                 `json:"minimum-stability"`
	PreferStable     bool                   `json:"prefer-stable"`
	Repositories     []ComposerRepository   `json:"repositories"`
	Extra            map[string]interface{} `json:"extra,omitempty"`
}

// ComposerConfig represents the config section
type ComposerConfig struct {
	AllowPlugins     map[string]bool `json:"allow-plugins"`
	PreferredInstall string          `json:"preferred-install"`
	SortPackages     bool            `json:"sort-packages"`
	Audit            *ComposerAudit  `json:"audit,omitempty"`
}

// ComposerAudit represents the audit section in config
type ComposerAudit struct {
	BlockInsecure bool `json:"block-insecure"`
}

// ComposerAutoload represents the autoload section
type ComposerAutoload struct {
	PSR4                map[string]string   `json:"psr-4,omitempty"`
	PSR0                map[string][]string `json:"psr-0,omitempty"`
	Files               []string            `json:"files,omitempty"`
	ExcludeFromClassmap []string            `json:"exclude-from-classmap,omitempty"`
}

// ComposerAutoloadDev represents the autoload-dev section
type ComposerAutoloadDev struct {
	PSR4 map[string]string `json:"psr-4,omitempty"`
}

// ComposerRepository represents a composer repository
type ComposerRepository struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

// MagentoVersion represents a Magento version with its requirements
type MagentoVersion struct {
	Version            string
	ProductVersion     string
	RootUpdatePlugin   string
	VersionAuditPlugin string
}

// MageOSVersion represents a MageOS version with its requirements
type MageOSVersion struct {
	Version            string
	ProductVersion     string
	RootUpdatePlugin   string
	VersionAuditPlugin string
}

// GetMagentoVersions returns available Magento versions
func GetMagentoVersions() map[string]MagentoVersion {
	return map[string]MagentoVersion{
		"2.4.7-p3": {
			Version:            "2.4.7-p3",
			ProductVersion:     "2.4.7-p3",
			RootUpdatePlugin:   "^2.0.4",
			VersionAuditPlugin: "~0.1",
		},
		"2.4.7-p2": {
			Version:            "2.4.7-p2",
			ProductVersion:     "2.4.7-p2",
			RootUpdatePlugin:   "^2.0.4",
			VersionAuditPlugin: "~0.1",
		},
		"2.4.7-p1": {
			Version:            "2.4.7-p1",
			ProductVersion:     "2.4.7-p1",
			RootUpdatePlugin:   "^2.0.4",
			VersionAuditPlugin: "~0.1",
		},
		"2.4.7": {
			Version:            "2.4.7",
			ProductVersion:     "2.4.7",
			RootUpdatePlugin:   "^2.0.4",
			VersionAuditPlugin: "~0.1",
		},
		"2.4.6-p7": {
			Version:            "2.4.6-p7",
			ProductVersion:     "2.4.6-p7",
			RootUpdatePlugin:   "^2.0.4",
			VersionAuditPlugin: "~0.1",
		},
		"2.4.6-p6": {
			Version:            "2.4.6-p6",
			ProductVersion:     "2.4.6-p6",
			RootUpdatePlugin:   "^2.0.4",
			VersionAuditPlugin: "~0.1",
		},
		"2.4.5-p9": {
			Version:            "2.4.5-p9",
			ProductVersion:     "2.4.5-p9",
			RootUpdatePlugin:   "^2.0.4",
			VersionAuditPlugin: "~0.1",
		},
	}
}

// GetMageOSVersions returns available MageOS versions
func GetMageOSVersions() map[string]MageOSVersion {
	return map[string]MageOSVersion{
		"2.0.0": {
			Version:            "2.0.0",
			ProductVersion:     "2.0.0",
			RootUpdatePlugin:   "2.0.0",
			VersionAuditPlugin: "2.0.0",
		},
		"1.1.0": {
			Version:            "1.1.0",
			ProductVersion:     "1.1.0",
			RootUpdatePlugin:   "1.1.0",
			VersionAuditPlugin: "1.1.0",
		},
		"1.0.5": {
			Version:            "1.0.5",
			ProductVersion:     "1.0.5",
			RootUpdatePlugin:   "1.0.5",
			VersionAuditPlugin: "1.0.5",
		},
		"1.0.3": {
			Version:            "1.0.3",
			ProductVersion:     "1.0.3",
			RootUpdatePlugin:   "1.0.3",
			VersionAuditPlugin: "1.0.3",
		},
		"1.0.2": {
			Version:            "1.0.2",
			ProductVersion:     "1.0.2",
			RootUpdatePlugin:   "1.0.2",
			VersionAuditPlugin: "1.0.2",
		},
		"1.0.1": {
			Version:            "1.0.1",
			ProductVersion:     "1.0.1",
			RootUpdatePlugin:   "1.0.1",
			VersionAuditPlugin: "1.0.1",
		},
		"1.0.0": {
			Version:            "1.0.0",
			ProductVersion:     "1.0.0",
			RootUpdatePlugin:   "1.0.0",
			VersionAuditPlugin: "1.0.0",
		},
	}
}

// GetLatestMagentoVersion returns the latest Magento version
func GetLatestMagentoVersion() string {
	return "2.4.7-p3"
}

// GetLatestMageOSVersion returns the latest MageOS version
func GetLatestMageOSVersion() string {
	return "2.0.0"
}

// GenerateMagentoComposerJSON generates a composer.json for Magento
func GenerateMagentoComposerJSON(projectName, version string) ([]byte, error) {
	versions := GetMagentoVersions()
	v, ok := versions[version]
	if !ok {
		return nil, fmt.Errorf("unsupported Magento version: %s", version)
	}

	composer := ComposerJSON{
		Name:        fmt.Sprintf("magebox/%s", projectName),
		Description: "Magento 2 project created with MageBox",
		Type:        "project",
		License:     []string{"OSL-3.0", "AFL-3.0"},
		Version:     v.Version,
		Config: ComposerConfig{
			AllowPlugins: map[string]bool{
				"dealerdirect/phpcodesniffer-composer-installer": true,
				"laminas/laminas-dependency-plugin":              true,
				"magento/*":                                      true,
				"php-http/discovery":                             true,
			},
			PreferredInstall: "dist",
			SortPackages:     true,
			Audit: &ComposerAudit{
				BlockInsecure: false,
			},
		},
		Require: map[string]string{
			"magento/product-community-edition":                v.ProductVersion,
			"magento/composer-root-update-plugin":              v.RootUpdatePlugin,
			"magento/composer-dependency-version-audit-plugin": v.VersionAuditPlugin,
		},
		Autoload: ComposerAutoload{
			PSR4: map[string]string{
				"Magento\\Setup\\": "setup/src/Magento/Setup/",
			},
			PSR0: map[string][]string{
				"": {"app/code/", "generated/code/"},
			},
			Files: []string{
				"app/etc/NonComposerComponentRegistration.php",
			},
			ExcludeFromClassmap: []string{
				"**/dev/**",
				"**/update/**",
				"**/Test/**",
			},
		},
		Conflict: map[string]string{
			"gene/bluefoot": "*",
		},
		MinimumStability: "stable",
		PreferStable:     true,
		Repositories: []ComposerRepository{
			{
				Type: "composer",
				URL:  "https://repo.magento.com/",
			},
		},
		Extra: map[string]interface{}{
			"magento-force": "override",
		},
	}

	return json.MarshalIndent(composer, "", "    ")
}

// GenerateMageOSComposerJSON generates a composer.json for MageOS
func GenerateMageOSComposerJSON(projectName, version string) ([]byte, error) {
	versions := GetMageOSVersions()
	v, ok := versions[version]
	if !ok {
		return nil, fmt.Errorf("unsupported MageOS version: %s", version)
	}

	composer := ComposerJSON{
		Name:        fmt.Sprintf("magebox/%s", projectName),
		Description: "MageOS project created with MageBox",
		Type:        "project",
		License:     []string{"OSL-3.0", "AFL-3.0"},
		Version:     v.Version,
		Config: ComposerConfig{
			AllowPlugins: map[string]bool{
				"dealerdirect/phpcodesniffer-composer-installer": true,
				"laminas/laminas-dependency-plugin":              true,
				"mage-os/*":                                      true,
				"php-http/discovery":                             true,
			},
			PreferredInstall: "dist",
			SortPackages:     true,
			Audit: &ComposerAudit{
				BlockInsecure: false,
			},
		},
		Require: map[string]string{
			"mage-os/product-community-edition":                v.ProductVersion,
			"mage-os/composer-root-update-plugin":              v.RootUpdatePlugin,
			"mage-os/composer-dependency-version-audit-plugin": v.VersionAuditPlugin,
		},
		Autoload: ComposerAutoload{
			PSR4: map[string]string{
				"Magento\\Framework\\": "lib/internal/Magento/Framework/",
				"Magento\\Setup\\":     "setup/src/Magento/Setup/",
				"Magento\\":            "app/code/Magento/",
			},
			PSR0: map[string][]string{
				"": {"app/code/", "generated/code/"},
			},
			Files: []string{
				"app/etc/NonComposerComponentRegistration.php",
			},
			ExcludeFromClassmap: []string{
				"**/dev/**",
				"**/update/**",
				"*/*/Test/**/*Test",
			},
		},
		Conflict: map[string]string{
			"gene/bluefoot": "*",
		},
		MinimumStability: "stable",
		PreferStable:     true,
		Repositories: []ComposerRepository{
			{
				Type: "composer",
				URL:  "https://repo.mage-os.org/",
			},
		},
		Extra: map[string]interface{}{
			"magento-force": "override",
		},
	}

	return json.MarshalIndent(composer, "", "    ")
}

// GetAvailableMagentoVersions returns a list of available Magento versions
func GetAvailableMagentoVersions() []string {
	return []string{
		"2.4.7-p3",
		"2.4.7-p2",
		"2.4.7-p1",
		"2.4.7",
		"2.4.6-p7",
		"2.4.6-p6",
		"2.4.5-p9",
	}
}

// GetAvailableMageOSVersions returns a list of available MageOS versions
func GetAvailableMageOSVersions() []string {
	return []string{
		"2.0.0",
		"1.1.0",
		"1.0.5",
		"1.0.3",
		"1.0.2",
		"1.0.1",
		"1.0.0",
	}
}

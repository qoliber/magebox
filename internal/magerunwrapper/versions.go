package magerunwrapper

import (
	"strconv"
	"strings"
)

// magentoPackageNames lists the Composer package names used by Magento distributions.
var magentoPackageNames = []string{
	"magento/product-community-edition",
	"mage-os/mage-os-community-edition",
	"magento/magento2-community-edition",
}

const (
	// legacyVersion is the last n98-magerun2 release compatible with Magento < 2.4.5.
	legacyVersion = "7.5.0"
	// legacyMinorCutoff is the first 2.4.x minor that works with the modern magerun2 series.
	legacyMinorCutoff = 5
)

// needsLegacy reports whether magentoVersion requires the 7.x magerun2 series.
// Returns false (use latest) when version is empty or not a 2.4.x string.
func needsLegacy(magentoVersion string) bool {
	minor, ok := parseMagentoMinor(magentoVersion)
	return ok && minor < legacyMinorCutoff
}

// parseMagentoMinor extracts the patch number from a 2.4.x version string such
// as "2.4.8" or "2.4.5-p1". Returns (0, false) for any non-2.4.x input.
func parseMagentoMinor(version string) (int, bool) {
	version = strings.TrimPrefix(version, "v")
	if idx := strings.IndexByte(version, '-'); idx >= 0 {
		version = version[:idx]
	}
	parts := strings.SplitN(version, ".", 3)
	if len(parts) != 3 || parts[0] != "2" || parts[1] != "4" {
		return 0, false
	}
	minor, err := strconv.Atoi(parts[2])
	if err != nil {
		return 0, false
	}
	return minor, true
}

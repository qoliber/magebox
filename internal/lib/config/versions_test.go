package config

import (
	"os"
	"strings"
	"testing"
)

// MageOS 3.x requires PHP ~8.3 || ~8.4 || ~8.5 upstream
// (mage-os/mageos-magento2 composer.json). Advertising 8.2 sends people into a
// build that cannot install, and omitting 8.5 hides a supported version.
func TestMageOS3PHPVersionsMatchUpstream(t *testing.T) {
	cfg, err := LoadEmbeddedVersions()
	if err != nil {
		t.Fatalf("LoadEmbeddedVersions() failed: %v", err)
	}

	supported := map[string]bool{"8.3": true, "8.4": true, "8.5": true}

	for _, entry := range cfg.GetMageOSVersions() {
		if !strings.HasPrefix(entry.Version, "3.") {
			continue
		}
		t.Run(entry.Version, func(t *testing.T) {
			if len(entry.PHP) == 0 {
				t.Fatalf("MageOS %s lists no PHP versions", entry.Version)
			}
			for _, php := range entry.PHP {
				if !supported[php] {
					t.Errorf("MageOS %s advertises PHP %s, which upstream does not support", entry.Version, php)
				}
			}
		})
	}
}

// The registry exists twice: embedded in the binary, and shipped in lib/ for
// users who override it. They must not drift apart.
func TestVersionRegistryCopiesAreIdentical(t *testing.T) {
	embedded, err := os.ReadFile("versions.yaml")
	if err != nil {
		t.Fatalf("failed to read embedded registry: %v", err)
	}
	shipped, err := os.ReadFile("../../../lib/templates/config/versions.yaml")
	if err != nil {
		t.Fatalf("failed to read shipped registry: %v", err)
	}

	if string(embedded) != string(shipped) {
		t.Error("internal/lib/config/versions.yaml and lib/templates/config/versions.yaml differ; update both")
	}
}

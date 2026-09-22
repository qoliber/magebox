// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package templates

import (
	"encoding/json"
	"testing"
)

// TestGenerateMageOSComposerJSONAutoload guards the autoload roots against drifting
// from the upstream mage-os/project-community-edition template. Since MageOS 3.2.0
// the shipped setup/ code references MageOS\Installer\Console\Command\InstallCommand
// from a class-constant array in Magento\Setup\Console\CommandLoader, so a missing
// psr-4 root makes every bin/magento call fatal.
func TestGenerateMageOSComposerJSONAutoload(t *testing.T) {
	raw, err := GenerateMageOSComposerJSON("shop", "3.4.0")
	if err != nil {
		t.Fatalf("GenerateMageOSComposerJSON returned error: %v", err)
	}

	var composer ComposerJSON
	if err := json.Unmarshal(raw, &composer); err != nil {
		t.Fatalf("generated composer.json is not valid JSON: %v", err)
	}

	want := map[string]string{
		"Magento\\":            "app/code/Magento/",
		"Magento\\Framework\\": "lib/internal/Magento/Framework/",
		"Magento\\Setup\\":     "setup/src/Magento/Setup/",
		"MageOS\\Installer\\":  "setup/src/MageOS/Installer/",
	}

	for namespace, path := range want {
		got, ok := composer.Autoload.PSR4[namespace]
		if !ok {
			t.Errorf("psr-4 autoload is missing namespace %q", namespace)
			continue
		}
		if got != path {
			t.Errorf("psr-4 autoload for %q = %q, want %q", namespace, got, path)
		}
	}
}

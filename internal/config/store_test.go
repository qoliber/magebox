package config

import "testing"

// A store type other than store or website silently produces a MAGE_RUN_TYPE
// Magento does not understand, so a typo must fail validation.
func TestConfigValidatesStoreType(t *testing.T) {
	base := func(storeType string) *Config {
		return &Config{
			Name:    "mystore",
			PHP:     "8.3",
			Domains: []Domain{{Host: "mystore.test", StoreCode: "nl", StoreType: storeType}},
		}
	}

	tests := []struct {
		name      string
		storeType string
		wantError bool
	}{
		{name: "unset defaults to store", storeType: "", wantError: false},
		{name: "store", storeType: "store", wantError: false},
		{name: "website", storeType: "website", wantError: false},
		{name: "typo", storeType: "websites", wantError: true},
		{name: "store view", storeType: "store_view", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := base(tt.storeType).Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// Projects written for earlier MageBox releases use mage_run_code and
// mage_run_type. The loader ignores unknown keys, so without a migration those
// projects would quietly lose their store codes and serve the wrong store.
func TestMigrateDeprecatedStoreKeys(t *testing.T) {
	cfg := &Config{
		Name: "mystore",
		Domains: []Domain{
			{Host: "mystore.test", MageRunCode: "nl", MageRunType: "website"},
			{Host: "de.mystore.test", StoreCode: "de", MageRunCode: "ignored"},
			{Host: "plain.test"},
		},
	}

	warnings := cfg.MigrateDeprecatedKeys()

	if cfg.Domains[0].StoreCode != "nl" || cfg.Domains[0].StoreType != "website" {
		t.Errorf("legacy keys were not migrated: %+v", cfg.Domains[0])
	}
	if cfg.Domains[1].StoreCode != "de" {
		t.Errorf("an explicit store_code must win over the legacy key: %+v", cfg.Domains[1])
	}
	if len(warnings) == 0 {
		t.Error("migrating a legacy key should report a deprecation warning")
	}
	if cfg.Domains[2].StoreCode != "" {
		t.Errorf("a domain without either key must stay empty: %+v", cfg.Domains[2])
	}
}

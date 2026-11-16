package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"qoliber/magebox/internal/config"
)

func TestProjectConfigJSON(t *testing.T) {
	tests := []struct {
		name         string
		main         string
		local        string
		database     *projectEndpoint
		cache        *projectEndpoint
		search       *projectEndpoint
		projectName  string
		databaseName string
	}{
		{
			name:         "local overrides and includes",
			main:         "include_config: [included.yaml]\nname: original\nphp: '8.2'\ndomains:\n  - host: original.test\nservices:\n  mysql: '8.0'\n  redis: true\n",
			local:        "name: my-shop\nphp: '8.3'\ndomains:\n  - host: shop.test\n    mage_run_code: french\n    store_type: website\nservices:\n  mysql: false\n  mariadb: '11.4'\n  opensearch: '2.0'\n",
			database:     &projectEndpoint{Service: "mariadb", Version: "11.4", Host: "127.0.0.1", Port: 33114},
			cache:        &projectEndpoint{Service: "redis", Host: "127.0.0.1", Port: 6379},
			search:       &projectEndpoint{Service: "opensearch", Version: "2.0", Host: "127.0.0.1", Port: 9200},
			projectName:  "my-shop",
			databaseName: "my_shop",
		},
		{
			name:         "absent database and alternative services",
			main:         "name: example\nphp: '8.2'\ndomains:\n  - host: example.test\nservices:\n  valkey: true\n  elasticsearch: '8.0'\n",
			cache:        &projectEndpoint{Service: "valkey", Host: "127.0.0.1", Port: 6379},
			search:       &projectEndpoint{Service: "elasticsearch", Version: "8.0", Host: "127.0.0.1", Port: 9500},
			projectName:  "example",
			databaseName: "example",
		},
	}

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldDir) })
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, content := range map[string]string{
				config.ConfigFileName: tt.main, config.LocalConfigFileName: tt.local,
				"included.yaml": "env:\n  INCLUDED: yes\n",
			} {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.Chdir(dir); err != nil {
				t.Fatal(err)
			}
			var output bytes.Buffer
			configCmd.SetOut(&output)
			t.Cleanup(func() { configCmd.SetOut(nil) })
			if err := configCmd.Flags().Set("json", "true"); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = configCmd.Flags().Set("json", "false") })
			if err := runConfigProject(configCmd, nil); err != nil {
				t.Fatal(err)
			}
			var got projectConfigJSON
			if err := json.Unmarshal(output.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if got.Config["name"] != tt.projectName || got.Derived.DatabaseName != tt.databaseName {
				t.Errorf("unexpected project name or database name: %+v", got)
			}
			for name, pair := range map[string][2]*projectEndpoint{
				"database": {got.Derived.Database, tt.database},
				"cache":    {got.Derived.Cache, tt.cache},
				"search":   {got.Derived.Search, tt.search},
			} {
				if (pair[0] == nil) != (pair[1] == nil) || (pair[0] != nil && *pair[0] != *pair[1]) {
					t.Errorf("%s = %+v, want %+v", name, pair[0], pair[1])
				}
			}
			if tt.name == "local overrides and includes" {
				env, ok := got.Config["env"].(map[string]interface{})
				if got.Config["php"] != "8.3" || !ok || env["INCLUDED"] != "yes" {
					t.Errorf("merged config = %#v", got.Config)
				}
				if d := got.Derived.Domains[0]; d.Host != "shop.test" || d.StoreCode != "french" || d.StoreType != "website" || d.Root != "pub" || !d.SSL {
					t.Errorf("domain = %+v", d)
				}
			}
		})
	}
}

func TestProjectConfigJSONMissingConfig(t *testing.T) {
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldDir) })
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if err := configCmd.Flags().Set("json", "true"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = configCmd.Flags().Set("json", "false") })
	var output bytes.Buffer
	configCmd.SetOut(&output)
	t.Cleanup(func() { configCmd.SetOut(nil) })
	if err := runConfigProject(configCmd, nil); err == nil {
		t.Fatal("expected an error for a missing config")
	}
	if output.Len() != 0 {
		t.Fatalf("unexpected output: %s", output.String())
	}
}

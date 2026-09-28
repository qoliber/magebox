package nginx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"qoliber/magebox/internal/config"
)

func project(name string, domains ...config.Domain) *config.Config {
	return &config.Config{Name: name, Domains: domains}
}

// nginx allows one map per variable. A map block per project puts several in the
// shared vhosts directory, which nginx rejects with `duplicate "MAGE_RUN_CODE"
// variable`, and then no project on the machine serves at all.
func TestRenderStoreMapHasOneMapPerVariableForManyProjects(t *testing.T) {
	projects := []*config.Config{
		project("shop", config.Domain{Host: "shop.test", StoreCode: "nl"}, config.Domain{Host: "de.shop.test", StoreCode: "de", StoreType: "website"}),
		project("blog", config.Domain{Host: "blog.test", StoreCode: "blog"}),
		project("plain", config.Domain{Host: "plain.test"}),
	}

	content, err := RenderStoreMap(StoreMapEntries(projects))
	if err != nil {
		t.Fatalf("RenderStoreMap failed: %v", err)
	}

	if got := strings.Count(content, "map $host $MAGE_RUN_CODE"); got != 1 {
		t.Errorf("found %d MAGE_RUN_CODE maps, want exactly 1:\n%s", got, content)
	}
	if got := strings.Count(content, "map $host $MAGE_RUN_TYPE"); got != 1 {
		t.Errorf("found %d MAGE_RUN_TYPE maps, want exactly 1:\n%s", got, content)
	}
	for _, want := range []string{"shop.test", "de.shop.test", "blog.test"} {
		if !strings.Contains(content, want) {
			t.Errorf("map is missing host %q:\n%s", want, content)
		}
	}
	if strings.Contains(content, "plain.test") {
		t.Errorf("a domain without a store code must not appear:\n%s", content)
	}
}

// A leading dot with `hostnames` matches every subdomain, so a deliberately
// code-less admin subdomain would inherit its parent's store code.
func TestRenderStoreMapMatchesExactHostsOnly(t *testing.T) {
	content, err := RenderStoreMap(StoreMapEntries([]*config.Config{
		project("shop", config.Domain{Host: "shop.test", StoreCode: "nl"}),
	}))
	if err != nil {
		t.Fatalf("RenderStoreMap failed: %v", err)
	}

	if strings.Contains(content, "hostnames") {
		t.Errorf("the map must not use wildcard hostname matching:\n%s", content)
	}
	if strings.Contains(content, ".shop.test") {
		t.Errorf("hosts must be exact, not dot-prefixed:\n%s", content)
	}
}

func TestStoreMapEntries(t *testing.T) {
	entries := StoreMapEntries([]*config.Config{
		project("b", config.Domain{Host: "b.test", StoreCode: "b"}),
		project("a", config.Domain{Host: "a.test", StoreCode: "a", StoreType: "website"}, config.Domain{Host: "nocode.test"}),
	})

	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2: %+v", len(entries), entries)
	}
	// Sorted, so the generated file does not churn between runs.
	if entries[0].Host != "a.test" || entries[1].Host != "b.test" {
		t.Errorf("entries are not sorted by host: %+v", entries)
	}
	if entries[0].Type != "website" {
		t.Errorf("explicit store type lost: %+v", entries[0])
	}
	if entries[1].Type != "store" {
		t.Errorf("missing store type should default to store: %+v", entries[1])
	}
}

// Upgrading from the per-project map must not leave those files behind, or the
// duplicate variable they cause keeps nginx broken.
func TestEnsureStoreMapRemovesLegacyPerProjectFiles(t *testing.T) {
	dir := t.TempDir()
	legacy := filepath.Join(dir, "shop-map.conf")
	if err := os.WriteFile(legacy, []byte("map $host $MAGE_RUN_CODE {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := WriteStoreMap(dir, StoreMapEntries([]*config.Config{
		project("shop", config.Domain{Host: "shop.test", StoreCode: "nl"}),
	})); err != nil {
		t.Fatalf("WriteStoreMap failed: %v", err)
	}

	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Error("the legacy per-project map file was not removed")
	}
	if _, err := os.Stat(filepath.Join(dir, StoreMapFile)); err != nil {
		t.Errorf("the shared map file was not written: %v", err)
	}
}

// With no store codes anywhere the file must go, so no vhost references a
// variable nginx does not know.
func TestWriteStoreMapRemovesFileWhenNoStoreCodes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, StoreMapFile)
	if err := os.WriteFile(path, []byte("map $host $MAGE_RUN_CODE {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := WriteStoreMap(dir, StoreMapEntries([]*config.Config{project("plain", config.Domain{Host: "plain.test"})})); err != nil {
		t.Fatalf("WriteStoreMap failed: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("the shared map file should be removed when nothing uses store codes")
	}
}

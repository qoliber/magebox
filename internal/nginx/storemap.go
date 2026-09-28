package nginx

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"qoliber/magebox/internal/config"
)

// StoreMapFile holds the host to store mapping for every project.
//
// nginx accepts one map per variable, and every file in the vhosts directory is
// included into the same http block. A map per project therefore makes nginx
// reject the whole configuration with `duplicate "MAGE_RUN_CODE" variable`,
// taking down every project on the machine, so all projects share this file.
// The numeric prefix keeps it ahead of the vhosts that read the variables.
const StoreMapFile = "000-magebox-store-map.conf"

// storeMapTemplate renders the shared map. Hosts are matched exactly: nginx's
// `hostnames` mode with a leading dot would also match every subdomain, so a
// deliberately code-less admin host would inherit its parent's store code.
const storeMapTemplate = `# MageBox store map for every project with store codes
# Do not edit manually - regenerated on magebox start

map $host $MAGE_RUN_CODE {
    default "";
{{- range .}}
    {{.Host}}    {{.Code}};
{{- end}}
}

map $host $MAGE_RUN_TYPE {
    default "";
{{- range .}}
    {{.Host}}    {{.Type}};
{{- end}}
}
`

// StoreMapEntry is one host in the shared map.
type StoreMapEntry struct {
	Host string
	Code string
	Type string
}

// StoreMapEntries collects the mapping from every project, sorted by host so
// the generated file does not churn between runs.
func StoreMapEntries(projects []*config.Config) []StoreMapEntry {
	var entries []StoreMapEntry
	for _, project := range projects {
		if project == nil {
			continue
		}
		for _, domain := range project.Domains {
			if domain.StoreCode == "" {
				continue
			}
			entries = append(entries, StoreMapEntry{
				Host: domain.Host,
				Code: domain.StoreCode,
				Type: domain.GetStoreType(),
			})
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Host < entries[j].Host })
	return entries
}

// RenderStoreMap renders the shared map file.
func RenderStoreMap(entries []StoreMapEntry) (string, error) {
	tmpl, err := template.New("storemap").Parse(storeMapTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse the store map template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, entries); err != nil {
		return "", fmt.Errorf("failed to render the store map: %w", err)
	}
	return buf.String(), nil
}

// WriteStoreMap installs the shared map, or removes it when no project uses
// store codes, and clears the per-project map files older releases wrote.
func WriteStoreMap(vhostsDir string, entries []StoreMapEntry) error {
	if err := removeLegacyStoreMaps(vhostsDir); err != nil {
		return err
	}

	path := filepath.Join(vhostsDir, StoreMapFile)
	if len(entries) == 0 {
		// No vhost references the variables, and an empty map left behind would
		// only be one more file to explain.
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove %s: %w", path, err)
		}
		return nil
	}

	content, err := RenderStoreMap(entries)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(vhostsDir, 0755); err != nil {
		return fmt.Errorf("failed to create %s: %w", vhostsDir, err)
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// removeLegacyStoreMaps deletes the <project>-map.conf files written before the
// map was shared. Left in place they reintroduce the duplicate variable.
func removeLegacyStoreMaps(vhostsDir string) error {
	matches, err := filepath.Glob(filepath.Join(vhostsDir, "*-map.conf"))
	if err != nil {
		return err
	}
	for _, match := range matches {
		if strings.HasSuffix(match, StoreMapFile) {
			continue
		}
		if err := os.Remove(match); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove the legacy map %s: %w", match, err)
		}
	}
	return nil
}

// EnsureStoreMap rebuilds the shared map from every project MageBox knows about.
//
// It takes all projects rather than one, because a single file has to describe
// them all: nginx permits one map per variable.
func (g *VhostGenerator) EnsureStoreMap(projects []*config.Config) error {
	return WriteStoreMap(g.vhostsDir, StoreMapEntries(projects))
}

package installer

import (
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// The YAML installer definitions drive the generic installer, while the Go
// specs drive the per-distro ones. They describe the same machine, so they must
// not drift — a wildcard surviving in either place breaks sudo on Ubuntu 26.04.
func TestInstallerYAMLSudoersMatchGeneratedRules(t *testing.T) {
	tests := []struct {
		name string
		file string
		spec SudoersSpec
	}{
		{name: "ubuntu", file: "../../../lib/templates/installers/ubuntu.yaml", spec: UbuntuSudoersSpec()},
		{name: "fedora", file: "../../../lib/templates/installers/fedora.yaml", spec: FedoraSudoersSpec()},
		{name: "arch", file: "../../../lib/templates/installers/arch.yaml", spec: ArchSudoersSpec()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, err := os.ReadFile(tt.file)
			if err != nil {
				t.Fatalf("failed to read %s: %v", tt.file, err)
			}

			var doc struct {
				Sudoers struct {
					Rules []string `yaml:"rules"`
				} `yaml:"sudoers"`
			}
			if err := yaml.Unmarshal(raw, &doc); err != nil {
				t.Fatalf("failed to parse %s: %v", tt.file, err)
			}

			got := make([]string, 0, len(doc.Sudoers.Rules))
			for _, rule := range doc.Sudoers.Rules {
				got = append(got, strings.ReplaceAll(rule, "${user}", "jakub"))
			}

			want := SudoersRules("jakub", tt.spec)
			if len(got) != len(want) {
				t.Fatalf("%s lists %d rules, generator produces %d", tt.file, len(got), len(want))
			}
			for i := range want {
				if got[i] != want[i] {
					t.Errorf("rule %d differs:\n yaml: %s\n  got: %s", i, got[i], want[i])
				}
			}
		})
	}
}

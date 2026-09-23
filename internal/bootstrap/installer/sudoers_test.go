package installer

import (
	"strings"
	"testing"
)

// sudo-rs, the default sudo on Ubuntu 26.04, refuses to parse wildcards in
// command arguments. A single such rule makes the whole file invalid, so every
// MageBox rule breaks and the user is asked for a password on every operation.
func TestSudoersRulesHaveNoWildcards(t *testing.T) {
	specs := map[string]SudoersSpec{
		"ubuntu": UbuntuSudoersSpec(),
		"fedora": FedoraSudoersSpec(),
		"arch":   ArchSudoersSpec(),
	}

	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			for _, rule := range SudoersRules("jakub", spec) {
				if strings.Contains(rule, "*") {
					t.Errorf("rule contains a wildcard, which sudo-rs rejects: %q", rule)
				}
			}
		})
	}
}

// Wildcards were how one rule covered every PHP version. Without them each
// version needs its own line, or MageBox asks for a password when it starts,
// stops or reloads that version's pool.
func TestSudoersRulesCoverEveryPHPVersionAndAction(t *testing.T) {
	tests := []struct {
		name        string
		spec        SudoersSpec
		wantService func(version string) string
	}{
		{name: "ubuntu names the unit per version", spec: UbuntuSudoersSpec(), wantService: func(v string) string { return "php" + v + "-fpm" }},
		{name: "fedora drops the dot", spec: FedoraSudoersSpec(), wantService: func(v string) string { return "php" + strings.ReplaceAll(v, ".", "") + "-php-fpm" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules := strings.Join(SudoersRules("jakub", tt.spec), "\n")
			for _, version := range PHPVersions {
				for _, action := range []string{"start", "stop", "reload", "restart"} {
					want := "/usr/bin/systemctl " + action + " " + tt.wantService(version)
					if !strings.Contains(rules, want) {
						t.Errorf("missing rule for %q", want)
					}
				}
			}
		})
	}
}

// Arch ships a single php-fpm unit, so per-version lines would just repeat.
func TestArchSudoersRulesUseOneFPMUnit(t *testing.T) {
	rules := SudoersRules("jakub", ArchSudoersSpec())

	count := 0
	for _, rule := range rules {
		if strings.HasSuffix(rule, "/usr/bin/systemctl restart php-fpm") {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly one restart rule for the single php-fpm unit, got %d", count)
	}
}

func TestSudoersRulesCoverNginxAndBlackfire(t *testing.T) {
	rules := strings.Join(SudoersRules("jakub", UbuntuSudoersSpec()), "\n")

	for _, want := range []string{
		"/usr/bin/systemctl reload nginx",
		"/usr/bin/systemctl restart nginx",
		"/usr/sbin/nginx -s reload",
		"/usr/sbin/nginx -t",
		"/usr/bin/systemctl restart blackfire-agent",
	} {
		if !strings.Contains(rules, want) {
			t.Errorf("missing rule for %q", want)
		}
	}
}

func TestSudoersRulesNameTheUser(t *testing.T) {
	for _, rule := range SudoersRules("someuser", UbuntuSudoersSpec()) {
		if !strings.HasPrefix(rule, "someuser ALL=(ALL) NOPASSWD: ") {
			t.Errorf("rule does not grant to the user: %q", rule)
		}
	}
}

// An existing file is not proof of a working one: every user upgrading from an
// older MageBox carries a file full of wildcards that sudo-rs rejects.
func TestNeedsSudoersRewrite(t *testing.T) {
	tests := []struct {
		name    string
		exists  bool
		valid   bool
		current bool
		want    bool
	}{
		{name: "missing file", exists: false, valid: false, current: false, want: true},
		{name: "present but rejected by sudo", exists: true, valid: false, current: false, want: true},
		{name: "valid but stale content", exists: true, valid: true, current: false, want: true},
		{name: "valid and current", exists: true, valid: true, current: true, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := needsSudoersRewrite(tt.exists, tt.valid, tt.current); got != tt.want {
				t.Errorf("needsSudoersRewrite(%v, %v, %v) = %v, want %v", tt.exists, tt.valid, tt.current, got, tt.want)
			}
		})
	}
}

// Bootstrap is often started with sudo, where USER is root. Granting rules to
// root would be useless: the rules must name the human who invoked it.
func TestSudoersUserFromEnv(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{name: "plain invocation", env: map[string]string{"USER": "jakub"}, want: "jakub"},
		{name: "run under sudo", env: map[string]string{"SUDO_USER": "jakub", "USER": "root", "LOGNAME": "root"}, want: "jakub"},
		{name: "falls back to logname", env: map[string]string{"LOGNAME": "jakub"}, want: "jakub"},
		{name: "nothing usable", env: map[string]string{"USER": "root", "LOGNAME": "root"}, want: ""},
		{name: "empty environment", env: map[string]string{}, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lookup := func(key string) string { return tt.env[key] }
			if got := sudoersUserFromEnv(lookup); got != tt.want {
				t.Errorf("sudoersUserFromEnv() = %q, want %q", got, tt.want)
			}
		})
	}
}

package installer

import "testing"

func publishing(suites ...string) func(string) bool {
	set := make(map[string]bool, len(suites))
	for _, s := range suites {
		set[s] = true
	}
	return func(suite string) bool { return set[suite] }
}

// Ondrej's PPA stops at 24.04 and is being folded into packages.sury.org, which
// does publish for 26.04. Pinning the PPA's older suite instead produces
// packages whose dependencies (libxml2, libicu74, libzip4t64) cannot be
// satisfied on the newer release, so PHP still fails to install.
func TestPickPHPRepository(t *testing.T) {
	tests := []struct {
		name     string
		codename string
		ppa      func(string) bool
		sury     func(string) bool
		want     PHPRepository
	}{
		{
			name:     "PPA covers this release",
			codename: "noble",
			ppa:      publishing("noble"),
			sury:     publishing("noble"),
			want:     PHPRepository{Kind: PHPRepoPPA, Suite: "noble"},
		},
		{
			name:     "only sury covers this release",
			codename: "resolute",
			ppa:      publishing("noble"),
			sury:     publishing("resolute", "noble"),
			want:     PHPRepository{Kind: PHPRepoSury, Suite: "resolute"},
		},
		{
			name:     "neither covers it",
			codename: "vivacious",
			ppa:      publishing("noble"),
			sury:     publishing("noble"),
			want:     PHPRepository{Kind: PHPRepoNone},
		},
		{
			name:     "unknown codename",
			codename: "",
			ppa:      publishing("noble"),
			sury:     publishing("noble"),
			want:     PHPRepository{Kind: PHPRepoNone},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PickPHPRepository(tt.codename, tt.ppa, tt.sury)
			if got != tt.want {
				t.Errorf("PickPHPRepository(%q) = %+v, want %+v", tt.codename, got, tt.want)
			}
		})
	}
}

// Running bootstrap as root writes the whole environment into /root: config,
// certificates, nginx includes and pool directories nginx cannot even read.
func TestShouldRefuseRootBootstrap(t *testing.T) {
	tests := []struct {
		name string
		uid  int
		want bool
	}{
		{name: "regular user", uid: 1000, want: false},
		{name: "root", uid: 0, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShouldRefuseRootBootstrap(tt.uid); got != tt.want {
				t.Errorf("ShouldRefuseRootBootstrap(%d) = %v, want %v", tt.uid, got, tt.want)
			}
		})
	}
}

package installer

import "testing"

// A package left half-configured makes every later apt install exit 100, no
// matter what it is asked to install. Bootstrap then gave up before repairing
// the repository, so nothing could be installed at all.
func TestDpkgAuditReportsBroken(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   bool
	}{
		{name: "nothing broken", output: "", want: false},
		{name: "only whitespace", output: "\n \n", want: false},
		{
			name: "half-configured package",
			output: `The following packages are only half configured, probably due to problems
configuring them the first time.  The configuration should be retried using
dpkg --configure <package> or the configure menu option in dselect:
 php8.5-fpm       server-side, HTML-embedded scripting language (FPM-CGI binary)`,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dpkgAuditReportsBroken(tt.output); got != tt.want {
				t.Errorf("dpkgAuditReportsBroken() = %v, want %v", got, tt.want)
			}
		})
	}
}

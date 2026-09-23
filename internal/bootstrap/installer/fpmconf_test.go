package installer

import "testing"

// php-fpm.conf accumulated one MageBox include per bootstrap run, including one
// pointing into /root after a run as root. Duplicates define the same pool
// twice and the /root path matches nothing, so PHP-FPM refuses to start, which
// in turn breaks every dpkg operation touching the package.
func TestNormalizeFPMIncludes(t *testing.T) {
	const want = "include=/home/jakub/.magebox/php/pools/8.5/*.conf"

	tests := []struct {
		name        string
		content     string
		wantContent string
		wantChanged bool
	}{
		{
			name:        "adds a missing include",
			content:     "[global]\ninclude=/etc/php/8.5/fpm/pool.d/*.conf\n",
			wantContent: "[global]\ninclude=/etc/php/8.5/fpm/pool.d/*.conf\n" + want + "\n",
			wantChanged: true,
		},
		{
			name:        "leaves a correct file alone",
			content:     "[global]\n" + want + "\n",
			wantContent: "[global]\n" + want + "\n",
			wantChanged: false,
		},
		{
			name:        "collapses duplicates",
			content:     "[global]\n" + want + "\n" + want + "\n",
			wantContent: "[global]\n" + want + "\n",
			wantChanged: true,
		},
		{
			name:        "drops an include for another home",
			content:     "[global]\ninclude=/root/.magebox/php/pools/8.5/*.conf\n" + want + "\n",
			wantContent: "[global]\n" + want + "\n",
			wantChanged: true,
		},
		{
			name:        "keeps unrelated includes",
			content:     "[global]\ninclude=/etc/php/8.5/fpm/pool.d/*.conf\ninclude=/root/.magebox/php/pools/8.5/*.conf\n",
			wantContent: "[global]\ninclude=/etc/php/8.5/fpm/pool.d/*.conf\n" + want + "\n",
			wantChanged: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, changed := NormalizeFPMIncludes(tt.content, want)
			if got != tt.wantContent {
				t.Errorf("content =\n%q\nwant\n%q", got, tt.wantContent)
			}
			if changed != tt.wantChanged {
				t.Errorf("changed = %v, want %v", changed, tt.wantChanged)
			}
		})
	}
}

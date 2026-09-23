package installer

import "testing"

// apt-cache answers "unknown package" the same way whether the package really
// is missing or apt could not read a sources file. Treating an unreadable
// source as "package unavailable" made bootstrap skip every PHP package and
// report success, leaving the machine with only the distribution's PHP.
func TestAptCacheAnswerReliable(t *testing.T) {
	tests := []struct {
		name   string
		stderr string
		want   bool
	}{
		{name: "clean run", stderr: "", want: true},
		{
			name:   "sources file not readable",
			stderr: "W: Unable to read /etc/apt/sources.list.d/magebox-php.list - open (13: Permission denied)",
			want:   false,
		},
		{
			name:   "lists directory not readable",
			stderr: "E: Could not open file /var/lib/apt/lists/… - open (13: Permission denied)",
			want:   false,
		},
		{
			name:   "ordinary warning about a missing package",
			stderr: "N: Unable to locate package php9.9-fpm",
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := aptCacheAnswerReliable(tt.stderr); got != tt.want {
				t.Errorf("aptCacheAnswerReliable(%q) = %v, want %v", tt.stderr, got, tt.want)
			}
		})
	}
}

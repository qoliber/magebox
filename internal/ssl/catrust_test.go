package ssl

import "testing"

// mkcert names its authority "mkcert development CA <serial in decimal>", so a
// listing from the browser trust store shows which authority is trusted. A
// machine carried over from another install trusts a stale one, and MageBox
// reported the CA as installed because the file existed on disk, while every
// HTTPS site showed a warning.
func TestNSSListsCASerial(t *testing.T) {
	const listing = `mkcert development CA 244346547749281684915500604760826213949 C,,
Some Other CA                                                    ,,
`

	tests := []struct {
		name   string
		output string
		serial string
		want   bool
	}{
		{name: "serial is trusted", output: listing, serial: "244346547749281684915500604760826213949", want: true},
		{name: "a different mkcert CA is trusted", output: listing, serial: "271385820751692510796476016112517704270", want: false},
		{name: "empty store", output: "", serial: "244346547749281684915500604760826213949", want: false},
		{name: "no serial known", output: listing, serial: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nssListsCASerial(tt.output, tt.serial); got != tt.want {
				t.Errorf("nssListsCASerial(serial %q) = %v, want %v", tt.serial, got, tt.want)
			}
		})
	}
}

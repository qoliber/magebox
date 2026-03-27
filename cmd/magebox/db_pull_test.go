package main

import (
	"bytes"
	"testing"
	"time"
)

func TestWarningFilter(t *testing.T) {
	tests := []struct {
		name  string
		input []string // multiple writes
		want  string
	}{
		{
			name:  "passes normal output",
			input: []string{"hello world\n"},
			want:  "hello world\n",
		},
		{
			name:  "filters password warning",
			input: []string{"mysql: [Warning] Using a password on the command line interface can be insecure.\n"},
			want:  "",
		},
		{
			name:  "filters mysqldump warning",
			input: []string{"mysqldump: [Warning] Using a password on the command line interface can be insecure.\n"},
			want:  "",
		},
		{
			name: "keeps real errors, filters warnings",
			input: []string{
				"mysqldump: [Warning] Using a password on the command line interface can be insecure.\n",
				"ERROR 1045 (28000): Access denied\n",
			},
			want: "ERROR 1045 (28000): Access denied\n",
		},
		{
			name:  "handles warning split across writes",
			input: []string{"mysqldump: [Warning] Using a password on the command line", " interface can be insecure.\n"},
			want:  "",
		},
		{
			name:  "handles mixed content",
			input: []string{"mysqldump: [Warning] Using a password on the command line interface can be insecure.\nreal output here\n"},
			want:  "real output here\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			f := newWarningFilter(&buf)

			for _, s := range tt.input {
				_, err := f.Write([]byte(s))
				if err != nil {
					t.Fatalf("Write() error: %v", err)
				}
			}
			_ = f.Flush()

			if got := buf.String(); got != tt.want {
				t.Errorf("output = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFilterWarnings(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "only warnings",
			input: "mysql: [Warning] Using a password on the command line interface can be insecure.\n",
			want:  "",
		},
		{
			name:  "real error preserved",
			input: "Using a password on the command line interface can be insecure.\nERROR 1045: Access denied\n",
			want:  "ERROR 1045: Access denied",
		},
		{
			name:  "no warnings",
			input: "ERROR 1045: Access denied\nERROR 1146: Table not found",
			want:  "ERROR 1045: Access denied\nERROR 1146: Table not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filterWarnings(tt.input); got != tt.want {
				t.Errorf("filterWarnings() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatElapsed(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{"seconds only", 11 * time.Second, "11.0s"},
		{"sub-second", 500 * time.Millisecond, "0.5s"},
		{"one minute", 60 * time.Second, "1m0s"},
		{"minutes and seconds", 4*time.Minute + 32*time.Second, "4m32s"},
		{"exact minutes", 5 * time.Minute, "5m0s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatElapsed(tt.duration); got != tt.want {
				t.Errorf("formatElapsed() = %q, want %q", got, tt.want)
			}
		})
	}
}

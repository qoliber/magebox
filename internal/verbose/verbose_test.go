package verbose

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestSetLevel(t *testing.T) {
	// Reset to quiet after test
	defer SetLevel(LevelQuiet)

	tests := []struct {
		name     string
		level    Level
		expected Level
	}{
		{"quiet", LevelQuiet, LevelQuiet},
		{"basic", LevelBasic, LevelBasic},
		{"detailed", LevelDetailed, LevelDetailed},
		{"debug", LevelDebug, LevelDebug},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetLevel(tt.level)
			if got := GetLevel(); got != tt.expected {
				t.Errorf("GetLevel() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsEnabled(t *testing.T) {
	defer SetLevel(LevelQuiet)

	tests := []struct {
		name       string
		setLevel   Level
		checkLevel Level
		shouldBeOn bool
	}{
		{"quiet_check_quiet", LevelQuiet, LevelQuiet, true},
		{"quiet_check_basic", LevelQuiet, LevelBasic, false},
		{"basic_check_basic", LevelBasic, LevelBasic, true},
		{"basic_check_detailed", LevelBasic, LevelDetailed, false},
		{"detailed_check_basic", LevelDetailed, LevelBasic, true},
		{"detailed_check_detailed", LevelDetailed, LevelDetailed, true},
		{"debug_check_all", LevelDebug, LevelBasic, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetLevel(tt.setLevel)
			if got := IsEnabled(tt.checkLevel); got != tt.shouldBeOn {
				t.Errorf("IsEnabled(%v) with level %v = %v, want %v",
					tt.checkLevel, tt.setLevel, got, tt.shouldBeOn)
			}
		})
	}
}

func TestPrintf(t *testing.T) {
	defer SetLevel(LevelQuiet)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	SetLevel(LevelDebug)
	Printf(LevelDebug, "test message %s", "arg")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "test message arg") {
		t.Errorf("Printf output = %q, should contain 'test message arg'", output)
	}
	if !strings.Contains(output, "[trace]") {
		t.Errorf("Printf output = %q, should contain '[trace]' prefix", output)
	}
}

func TestPrintf_NotEnabled(t *testing.T) {
	defer SetLevel(LevelQuiet)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	SetLevel(LevelQuiet)
	Printf(LevelDebug, "should not appear")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output != "" {
		t.Errorf("Printf should not output when level is quiet, got %q", output)
	}
}

func TestCommand(t *testing.T) {
	defer SetLevel(LevelQuiet)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	SetLevel(LevelBasic)
	Command("docker", "compose", "up", "-d")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "$ docker compose up -d") {
		t.Errorf("Command output = %q, should contain '$ docker compose up -d'", output)
	}
}

func TestDebug(t *testing.T) {
	defer SetLevel(LevelQuiet)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	SetLevel(LevelDebug)
	Debug("debug info: %d", 42)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "debug info: 42") {
		t.Errorf("Debug output = %q, should contain 'debug info: 42'", output)
	}
}

func TestInfo(t *testing.T) {
	defer SetLevel(LevelQuiet)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	SetLevel(LevelBasic)
	Info("info message")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "info message") {
		t.Errorf("Info output = %q, should contain 'info message'", output)
	}
	if !strings.Contains(output, "[verbose]") {
		t.Errorf("Info output = %q, should contain '[verbose]' prefix", output)
	}
}

func TestDetail(t *testing.T) {
	defer SetLevel(LevelQuiet)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	SetLevel(LevelDetailed)
	Detail("detailed message")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "detailed message") {
		t.Errorf("Detail output = %q, should contain 'detailed message'", output)
	}
	if !strings.Contains(output, "[debug]") {
		t.Errorf("Detail output = %q, should contain '[debug]' prefix", output)
	}
}

func TestSection(t *testing.T) {
	defer SetLevel(LevelQuiet)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	SetLevel(LevelBasic)
	Section("Test Section")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "=== Test Section ===") {
		t.Errorf("Section output = %q, should contain '=== Test Section ==='", output)
	}
}

func TestCommandOutput(t *testing.T) {
	defer SetLevel(LevelQuiet)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	SetLevel(LevelDetailed)
	CommandOutput("line1\nline2\nline3")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "> line1") {
		t.Errorf("CommandOutput should contain '> line1', got %q", output)
	}
	if !strings.Contains(output, "> line2") {
		t.Errorf("CommandOutput should contain '> line2', got %q", output)
	}
}

func TestLevelConstants(t *testing.T) {
	if LevelQuiet != 0 {
		t.Error("LevelQuiet should be 0")
	}
	if LevelBasic != 1 {
		t.Error("LevelBasic should be 1")
	}
	if LevelDetailed != 2 {
		t.Error("LevelDetailed should be 2")
	}
	if LevelDebug != 3 {
		t.Error("LevelDebug should be 3")
	}
}

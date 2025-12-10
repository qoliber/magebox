package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewLogTailer(t *testing.T) {
	tailer := NewLogTailer("/var/log", "*.log", true, 10)

	if tailer == nil {
		t.Fatal("NewLogTailer should not return nil")
	}
	if tailer.logDir != "/var/log" {
		t.Errorf("logDir = %v, want /var/log", tailer.logDir)
	}
	if tailer.pattern != "*.log" {
		t.Errorf("pattern = %v, want *.log", tailer.pattern)
	}
	if !tailer.follow {
		t.Error("follow should be true")
	}
	if tailer.lines != 10 {
		t.Errorf("lines = %v, want 10", tailer.lines)
	}
}

func TestLogTailer_FindLogFiles(t *testing.T) {
	// Create temp directory with log files
	tmpDir := t.TempDir()

	// Create some log files
	os.WriteFile(filepath.Join(tmpDir, "system.log"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "debug.log"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "exception.log"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "other.txt"), []byte("test"), 0644)

	tailer := NewLogTailer(tmpDir, "*.log", false, 10)
	files, err := tailer.findLogFiles()

	if err != nil {
		t.Fatalf("findLogFiles failed: %v", err)
	}

	if len(files) != 3 {
		t.Errorf("Found %d files, want 3", len(files))
	}

	// Verify all .log files were found
	foundSystem := false
	foundDebug := false
	foundException := false

	for _, f := range files {
		switch filepath.Base(f) {
		case "system.log":
			foundSystem = true
		case "debug.log":
			foundDebug = true
		case "exception.log":
			foundException = true
		}
	}

	if !foundSystem {
		t.Error("system.log not found")
	}
	if !foundDebug {
		t.Error("debug.log not found")
	}
	if !foundException {
		t.Error("exception.log not found")
	}
}

func TestLogTailer_FindLogFilesWithPattern(t *testing.T) {
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, "system.log"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "debug.log"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "system.log.1"), []byte("test"), 0644)

	// Only match files ending in .log
	tailer := NewLogTailer(tmpDir, "*.log", false, 10)
	files, _ := tailer.findLogFiles()

	if len(files) != 2 {
		t.Errorf("Found %d files, want 2", len(files))
	}
}

func TestLogTailer_ReadLastLines(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	// Create log file with multiple lines
	content := "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10\n"
	os.WriteFile(logFile, []byte(content), 0644)

	tailer := NewLogTailer(tmpDir, "*.log", false, 5)

	file, _ := os.Open(logFile)
	defer file.Close()

	lines := tailer.readLastLines(file, 5)

	if len(lines) != 5 {
		t.Errorf("Got %d lines, want 5", len(lines))
	}

	// Should be the last 5 lines
	expected := []string{"line6", "line7", "line8", "line9", "line10"}
	for i, line := range lines {
		if line != expected[i] {
			t.Errorf("lines[%d] = %q, want %q", i, line, expected[i])
		}
	}
}

func TestLogTailer_ReadLastLinesSmallFile(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	// Create log file with fewer lines than requested
	content := "line1\nline2\nline3\n"
	os.WriteFile(logFile, []byte(content), 0644)

	tailer := NewLogTailer(tmpDir, "*.log", false, 10)

	file, _ := os.Open(logFile)
	defer file.Close()

	lines := tailer.readLastLines(file, 10)

	if len(lines) != 3 {
		t.Errorf("Got %d lines, want 3", len(lines))
	}
}

func TestLogTailer_ReadLastLinesEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	// Create empty log file
	os.WriteFile(logFile, []byte(""), 0644)

	tailer := NewLogTailer(tmpDir, "*.log", false, 10)

	file, _ := os.Open(logFile)
	defer file.Close()

	lines := tailer.readLastLines(file, 10)

	if len(lines) != 0 {
		t.Errorf("Got %d lines, want 0 for empty file", len(lines))
	}
}

func TestLogTailer_NoMatchingFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create non-log files
	os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("test"), 0644)

	tailer := NewLogTailer(tmpDir, "*.log", false, 10)
	err := tailer.Start()

	if err == nil {
		t.Error("Expected error when no matching files")
	}

	if !strings.Contains(err.Error(), "no log files found") {
		t.Errorf("Error should mention 'no log files found', got: %v", err)
	}
}

func TestLogTailer_PrintLogLine(t *testing.T) {
	EnableColors()

	tailer := NewLogTailer("/tmp", "*.log", false, 10)

	// Test with Magento log format
	line := "[2024-01-15T10:30:45.123456+00:00] main.CRITICAL: Error message"
	matches := tailer.levelRegex.FindStringSubmatch(line)

	if len(matches) < 2 {
		t.Error("Should match log level")
	}
	if matches[1] != "CRITICAL" {
		t.Errorf("Level = %v, want CRITICAL", matches[1])
	}
}

func TestLogTailer_Stop(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "test.log"), []byte("line1\n"), 0644)

	tailer := NewLogTailer(tmpDir, "*.log", false, 10)

	// Should not panic
	tailer.Stop()
}

func TestLogLevelRegex(t *testing.T) {
	tailer := NewLogTailer("/tmp", "*.log", false, 10)

	tests := []struct {
		line          string
		expectedLevel string
	}{
		{"[2024-01-15T10:30:45.123456+00:00] main.DEBUG: Debug message", "DEBUG"},
		{"[2024-01-15T10:30:45.123456+00:00] main.INFO: Info message", "INFO"},
		{"[2024-01-15T10:30:45.123456+00:00] main.WARNING: Warning message", "WARNING"},
		{"[2024-01-15T10:30:45.123456+00:00] main.ERROR: Error message", "ERROR"},
		{"[2024-01-15T10:30:45.123456+00:00] main.CRITICAL: Critical message", "CRITICAL"},
	}

	for _, tt := range tests {
		t.Run(tt.expectedLevel, func(t *testing.T) {
			matches := tailer.levelRegex.FindStringSubmatch(tt.line)
			if len(matches) < 2 {
				t.Errorf("Failed to match level in: %s", tt.line)
				return
			}
			if matches[1] != tt.expectedLevel {
				t.Errorf("Level = %v, want %v", matches[1], tt.expectedLevel)
			}
		})
	}
}

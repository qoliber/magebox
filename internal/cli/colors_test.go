package cli

import (
	"strings"
	"testing"
)

func TestColorize(t *testing.T) {
	// Enable colors for testing
	EnableColors()

	t.Run("Success", func(t *testing.T) {
		result := Success("test message")
		if !strings.Contains(result, "test message") {
			t.Error("Success should contain message")
		}
		if !strings.Contains(result, Green) {
			t.Error("Success should contain green color code")
		}
		if !strings.Contains(result, SymbolCheck) {
			t.Error("Success should contain checkmark symbol")
		}
	})

	t.Run("Error", func(t *testing.T) {
		result := Error("error message")
		if !strings.Contains(result, "error message") {
			t.Error("Error should contain message")
		}
		if !strings.Contains(result, Red) {
			t.Error("Error should contain red color code")
		}
		if !strings.Contains(result, SymbolCross) {
			t.Error("Error should contain cross symbol")
		}
	})

	t.Run("Warning", func(t *testing.T) {
		result := Warning("warning message")
		if !strings.Contains(result, "warning message") {
			t.Error("Warning should contain message")
		}
		if !strings.Contains(result, Yellow) {
			t.Error("Warning should contain yellow color code")
		}
	})

	t.Run("Info", func(t *testing.T) {
		result := Info("info message")
		if !strings.Contains(result, "info message") {
			t.Error("Info should contain message")
		}
		if !strings.Contains(result, Cyan) {
			t.Error("Info should contain cyan color code")
		}
	})
}

func TestColorsDisabled(t *testing.T) {
	DisableColors()
	defer EnableColors()

	t.Run("Success without colors", func(t *testing.T) {
		result := Success("test")
		if strings.Contains(result, "\033[") {
			t.Error("Should not contain ANSI codes when colors disabled")
		}
		if !strings.Contains(result, "[OK]") {
			t.Error("Should contain [OK] prefix when colors disabled")
		}
	})

	t.Run("Error without colors", func(t *testing.T) {
		result := Error("test")
		if strings.Contains(result, "\033[") {
			t.Error("Should not contain ANSI codes when colors disabled")
		}
		if !strings.Contains(result, "[ERROR]") {
			t.Error("Should contain [ERROR] prefix when colors disabled")
		}
	})

	t.Run("Warning without colors", func(t *testing.T) {
		result := Warning("test")
		if strings.Contains(result, "\033[") {
			t.Error("Should not contain ANSI codes when colors disabled")
		}
		if !strings.Contains(result, "[WARN]") {
			t.Error("Should contain [WARN] prefix when colors disabled")
		}
	})
}

func TestStatus(t *testing.T) {
	EnableColors()

	t.Run("Running status", func(t *testing.T) {
		result := Status(true)
		if !strings.Contains(result, "running") {
			t.Error("Should contain 'running'")
		}
		if !strings.Contains(result, Green) {
			t.Error("Running should be green")
		}
	})

	t.Run("Stopped status", func(t *testing.T) {
		result := Status(false)
		if !strings.Contains(result, "stopped") {
			t.Error("Should contain 'stopped'")
		}
		if !strings.Contains(result, Red) {
			t.Error("Stopped should be red")
		}
	})
}

func TestStatusInstalled(t *testing.T) {
	EnableColors()

	t.Run("Installed", func(t *testing.T) {
		result := StatusInstalled(true)
		if !strings.Contains(result, "installed") {
			t.Error("Should contain 'installed'")
		}
		if !strings.Contains(result, Green) {
			t.Error("Installed should be green")
		}
	})

	t.Run("Not installed", func(t *testing.T) {
		result := StatusInstalled(false)
		if !strings.Contains(result, "not installed") {
			t.Error("Should contain 'not installed'")
		}
		if !strings.Contains(result, Red) {
			t.Error("Not installed should be red")
		}
	})
}

func TestLogLevel(t *testing.T) {
	EnableColors()

	tests := []struct {
		level    string
		expected string
	}{
		{"DEBUG", Dim},
		{"INFO", Blue},
		{"NOTICE", Cyan},
		{"WARNING", Yellow},
		{"ERROR", Red},
		{"CRITICAL", Bold + Red},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			result := LogLevel(tt.level)
			if !strings.Contains(result, tt.expected) {
				t.Errorf("LogLevel(%s) should contain %q", tt.level, tt.expected)
			}
		})
	}
}

func TestTitle(t *testing.T) {
	EnableColors()

	result := Title("Test Title")
	if !strings.Contains(result, "Test Title") {
		t.Error("Title should contain text")
	}
	if !strings.Contains(result, Bold) {
		t.Error("Title should be bold")
	}
}

func TestHeader(t *testing.T) {
	EnableColors()

	result := Header("Test Header")
	if !strings.Contains(result, "Test Header") {
		t.Error("Header should contain text")
	}
	// Should have a separator line
	if !strings.Contains(result, "─") {
		t.Error("Header should have separator")
	}
}

func TestHighlight(t *testing.T) {
	EnableColors()

	result := Highlight("highlighted")
	if !strings.Contains(result, "highlighted") {
		t.Error("Should contain text")
	}
	if !strings.Contains(result, Cyan) {
		t.Error("Should be cyan")
	}
	if !strings.Contains(result, Bold) {
		t.Error("Should be bold")
	}
}

func TestURL(t *testing.T) {
	EnableColors()

	result := URL("https://example.com")
	if !strings.Contains(result, "https://example.com") {
		t.Error("Should contain URL")
	}
	if !strings.Contains(result, Underline) {
		t.Error("URL should be underlined")
	}
}

func TestRepeatChar(t *testing.T) {
	result := repeatChar('-', 5)
	if result != "-----" {
		t.Errorf("repeatChar('-', 5) = %q, want '-----'", result)
	}

	result = repeatChar('=', 0)
	if result != "" {
		t.Errorf("repeatChar('=', 0) = %q, want ''", result)
	}
}

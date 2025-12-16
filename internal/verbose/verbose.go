// Package verbose provides verbosity-controlled logging for MageBox
package verbose

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// Level represents the verbosity level
type Level int

const (
	// LevelQuiet - no extra output
	LevelQuiet Level = 0
	// LevelBasic - show commands being run (-v)
	LevelBasic Level = 1
	// LevelDetailed - show command output (-vv)
	LevelDetailed Level = 2
	// LevelDebug - show full debug info (-vvv)
	LevelDebug Level = 3
)

var (
	currentLevel Level
	mu           sync.RWMutex
)

// SetLevel sets the global verbosity level
func SetLevel(level Level) {
	mu.Lock()
	defer mu.Unlock()
	currentLevel = level
}

// GetLevel returns the current verbosity level
func GetLevel() Level {
	mu.RLock()
	defer mu.RUnlock()
	return currentLevel
}

// IsEnabled returns true if the given level is enabled
func IsEnabled(level Level) bool {
	return GetLevel() >= level
}

// Printf prints a message at the given verbosity level
func Printf(level Level, format string, args ...interface{}) {
	if IsEnabled(level) {
		prefix := ""
		switch level {
		case LevelBasic:
			prefix = "\033[36m[verbose]\033[0m "
		case LevelDetailed:
			prefix = "\033[33m[debug]\033[0m "
		case LevelDebug:
			prefix = "\033[35m[trace]\033[0m "
		}
		fmt.Printf(prefix+format+"\n", args...)
	}
}

// Println prints a message at the given verbosity level
func Println(level Level, message string) {
	Printf(level, "%s", message)
}

// Command logs a command being executed (level 1)
func Command(name string, args ...string) {
	if IsEnabled(LevelBasic) {
		cmd := name
		if len(args) > 0 {
			cmd += " " + strings.Join(args, " ")
		}
		Printf(LevelBasic, "$ %s", cmd)
	}
}

// CommandOutput logs command output (level 2)
func CommandOutput(output string) {
	if IsEnabled(LevelDetailed) && output != "" {
		for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
			Printf(LevelDetailed, "  > %s", line)
		}
	}
}

// Debug logs debug information (level 3)
func Debug(format string, args ...interface{}) {
	Printf(LevelDebug, format, args...)
}

// Info logs info at basic level
func Info(format string, args ...interface{}) {
	Printf(LevelBasic, format, args...)
}

// Detail logs detailed info
func Detail(format string, args ...interface{}) {
	Printf(LevelDetailed, format, args...)
}

// RunCommand executes a command with verbose logging
func RunCommand(cmd *exec.Cmd) error {
	Command(cmd.Path, cmd.Args[1:]...)

	if IsEnabled(LevelDetailed) {
		output, err := cmd.CombinedOutput()
		CommandOutput(string(output))
		return err
	}

	return cmd.Run()
}

// RunCommandOutput executes a command and returns output with verbose logging
func RunCommandOutput(cmd *exec.Cmd) ([]byte, error) {
	Command(cmd.Path, cmd.Args[1:]...)

	output, err := cmd.CombinedOutput()
	if IsEnabled(LevelDetailed) {
		CommandOutput(string(output))
	}

	return output, err
}

// SystemInfo logs system information at debug level
func SystemInfo(key, value string) {
	Debug("System: %s = %s", key, value)
}

// Section logs a section header at basic level
func Section(name string) {
	if IsEnabled(LevelBasic) {
		fmt.Printf("\033[36m[verbose]\033[0m === %s ===\n", name)
	}
}

// Error logs an error with details at debug level
func Error(err error, context string) {
	if IsEnabled(LevelDebug) && err != nil {
		Printf(LevelDebug, "Error in %s: %v", context, err)
	}
}

// Env logs environment info at debug level
func Env() {
	if IsEnabled(LevelDebug) {
		Debug("Environment:")
		for _, env := range os.Environ() {
			if strings.HasPrefix(env, "MAGEBOX_") ||
				strings.HasPrefix(env, "DOCKER_") ||
				strings.HasPrefix(env, "PATH=") ||
				strings.HasPrefix(env, "HOME=") {
				Debug("  %s", env)
			}
		}
	}
}

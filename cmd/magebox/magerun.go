package main

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"qoliber/magebox/internal/config"
	"qoliber/magebox/internal/php"
)

// magerunBinaries lists the known magerun2 executable names to look for in PATH.
var magerunBinaries = []string{"n98-magerun2", "n98-magerun2.phar", "magerun2"}

// findMagerun returns the path to a magerun2 binary if one is available in PATH.
func findMagerun() (string, bool) {
	for _, name := range magerunBinaries {
		if path, err := exec.LookPath(name); err == nil {
			return path, true
		}
	}
	return "", false
}

// magerunEnv builds the environment for invoking magerun2, prepending the
// project's PHP to PATH (when a project config is available) so magerun2's
// "#!/usr/bin/env php" shebang picks up the right PHP version.
func magerunEnv(cwd string) []string {
	env := os.Environ()
	cfg, err := config.LoadFromPath(cwd)
	if err != nil {
		return env
	}
	if p, err := getPlatform(); err == nil {
		detector := php.NewDetector(p)
		version := detector.Detect(cfg.PHP)
		if version.Installed {
			phpDir := filepath.Dir(version.PHPBinary)
			env = append(env, "PATH="+phpDir+string(os.PathListSeparator)+os.Getenv("PATH"))
		}
	}
	for key, value := range cfg.Env {
		env = append(env, key+"="+value)
	}
	return env
}

// magerunHasCommand reports whether magerun2 knows about the given command
// name. It invokes `magerun2 list --raw --no-ansi` and parses the output,
// where each line begins with a command name followed by whitespace and a
// description.
func magerunHasCommand(binary, name string) bool {
	cwd, _ := os.Getwd()

	listCmd := exec.Command(binary, "list", "--raw", "--no-ansi")
	listCmd.Dir = cwd
	listCmd.Env = magerunEnv(cwd)

	var out bytes.Buffer
	listCmd.Stdout = &out
	// Discard stderr — we only care about the command list.
	listCmd.Stderr = &bytes.Buffer{}

	if err := listCmd.Run(); err != nil {
		return false
	}

	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		cmdName := strings.Fields(line)[0]
		if cmdName == name {
			return true
		}
	}
	return false
}

// runMagerun executes magerun2 with the given args, using the project's PHP
// version (if a project config is available) so the correct PHP is on PATH.
// It returns the exit code from magerun2, or -1 if it could not be started.
func runMagerun(binary string, args []string) int {
	cwd, _ := os.Getwd()

	shellCmd := exec.Command(binary, args...)
	shellCmd.Dir = cwd
	shellCmd.Stdin = os.Stdin
	shellCmd.Stdout = os.Stdout
	shellCmd.Stderr = os.Stderr
	shellCmd.Env = magerunEnv(cwd)

	if err := shellCmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		return -1
	}
	return 0
}

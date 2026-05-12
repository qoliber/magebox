package telemetry

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"qoliber/magebox/internal/config"
)

// consentPrompt is the exact text shown to the user on first run. It is
// intentionally verbose: we would rather the prompt be skipped because the
// user read it and said no than have them opt in without understanding.
const consentPrompt = `MageBox can send anonymous usage statistics to help prioritize improvements.

  What is sent:   command name, exit code, duration, MageBox version, OS, arch
  What is NOT:    flag values, arguments, paths, hostnames, IPs, project names

Everything is opt-in. You can review the last events with:
  magebox telemetry show

The public dashboard lives at: https://telemetry.magebox.dev
You can disable at any time with: magebox telemetry disable

Enable telemetry? [y/N] `

// ShouldPrompt reports whether the first-run prompt should be shown. It
// returns false once the user has been prompted once (regardless of answer)
// so we never ask twice.
func ShouldPrompt(cfg *config.GlobalConfig) bool {
	if cfg == nil || cfg.Telemetry == nil {
		return true
	}
	return !cfg.Telemetry.Prompted
}

// StdinIsInteractive reports whether stdin is attached to a terminal. The
// consent prompt should be suppressed in non-interactive contexts (CI, pipes)
// because blocking on stdin.Read would hang the command forever.
func StdinIsInteractive() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// MaybePrompt runs the first-run telemetry consent flow if appropriate. It is
// a no-op when:
//   - telemetry has already been prompted,
//   - stdin is not a terminal,
//   - the command being run is itself a telemetry management command (to avoid
//     surprising users running `magebox telemetry disable`).
//
// The user's choice is persisted to the global config. Returns the enabled
// state the caller should use for the remainder of the process.
func MaybePrompt(homeDir string, cfg *config.GlobalConfig, currentCommand string, in io.Reader, out io.Writer) (bool, error) {
	if cfg.Telemetry == nil {
		cfg.Telemetry = &config.TelemetryConfig{}
	}

	// Skip for meta commands and telemetry subcommand itself.
	if ShouldSkipCommand(currentCommand) {
		return cfg.Telemetry.Enabled, nil
	}
	if !ShouldPrompt(cfg) {
		return cfg.Telemetry.Enabled, nil
	}
	if !StdinIsInteractive() {
		// Mark prompted so we don't re-ask next time, but leave disabled.
		cfg.Telemetry.Prompted = true
		_ = config.SaveGlobalConfig(homeDir, cfg)
		return false, nil
	}

	_, _ = fmt.Fprint(out, "\n"+consentPrompt)
	reader := bufio.NewReader(in)
	answer, _ := reader.ReadString('\n')
	enabled := isYes(answer)

	cfg.Telemetry.Enabled = enabled
	cfg.Telemetry.Prompted = true
	if err := config.SaveGlobalConfig(homeDir, cfg); err != nil {
		return enabled, err
	}

	if enabled {
		_, _ = fmt.Fprintln(out, "Telemetry enabled. Thanks — you can disable any time with `magebox telemetry disable`.")
	} else {
		_, _ = fmt.Fprintln(out, "Telemetry disabled. Enable later with `magebox telemetry enable`.")
	}
	_, _ = fmt.Fprintln(out)
	return enabled, nil
}

func isYes(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	return s == "y" || s == "yes"
}

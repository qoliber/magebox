// Package telemetry provides opt-in, anonymous usage reporting for MageBox.
//
// Design goals:
//   - Opt-in only. Nothing is ever sent unless the user has explicitly enabled
//     telemetry through the first-run prompt or `magebox telemetry enable`.
//   - No PII. Only the command name, exit code, duration, MageBox version and
//     OS/arch are collected. Positional arguments and flag values are never
//     captured. See Event for the exact schema.
//   - Transparent. Every event recorded locally is also written to a rolling
//     log at ~/.magebox/telemetry-log.jsonl so users can verify with
//     `magebox telemetry show` exactly what has been or would be sent.
//   - Non-blocking. Record spools to disk synchronously (cheap) and then
//     hands the HTTP send off to a fully detached child process. A broken
//     network or a hung ingestion server cannot delay the user's CLI exit.
package telemetry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"

	"qoliber/magebox/internal/config"
)

// SchemaVersion is the version of the event schema. Bump whenever fields are
// added, removed or renamed so the ingestion server can route appropriately.
const SchemaVersion = 1

// DefaultEndpoint is where events are posted unless overridden in config.
const DefaultEndpoint = "https://telemetry.magebox.dev/v1/event"

// maxCommandLen is a belt-and-suspenders cap on the command field length so a
// stray very long command path can never bloat the payload.
const maxCommandLen = 64

// Event is the exact payload sent to the ingestion server. Every field here is
// considered public — if you add a field, document it in docs/telemetry.md and
// bump SchemaVersion.
type Event struct {
	SchemaVersion int    `json:"schema_version"`
	EventID       string `json:"event_id"`
	AnonID        string `json:"anon_id"`
	Command       string `json:"command"`
	ExitCode      int    `json:"exit_code"`
	DurationMs    int64  `json:"duration_ms"`
	MBVersion     string `json:"mb_version"`
	OS            string `json:"os"`
	Arch          string `json:"arch"`
	Timestamp     string `json:"ts"`
}

// RecordInput holds the parameters the CLI passes to Record. Keeping it as a
// struct means hook-site code reads clearly and adding future fields doesn't
// break callers.
type RecordInput struct {
	HomeDir   string
	MBVersion string
	Command   string
	ExitCode  int
	Duration  time.Duration
}

// Record builds an event and writes it to the local spool and rolling log,
// then hands the HTTP send off to a fully detached child process via
// flusher. It is safe to call from any command exit path and never blocks
// the caller on the network.
//
// If telemetry is disabled in the global config, Record returns immediately
// without touching disk.
func Record(in RecordInput) {
	if in.HomeDir == "" || in.Command == "" {
		return
	}

	cfg, err := config.LoadGlobalConfig(in.HomeDir)
	if err != nil || cfg.Telemetry == nil || !cfg.Telemetry.Enabled {
		return
	}

	anonID, err := loadOrCreateAnonID(in.HomeDir)
	if err != nil {
		return
	}

	ev := Event{
		SchemaVersion: SchemaVersion,
		EventID:       uuid.NewString(),
		AnonID:        anonID,
		Command:       truncate(in.Command, maxCommandLen),
		ExitCode:      in.ExitCode,
		DurationMs:    in.Duration.Milliseconds(),
		MBVersion:     in.MBVersion,
		OS:            runtime.GOOS,
		Arch:          runtime.GOARCH,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
	}

	// Always append to the rolling log so `telemetry show` can display it.
	_ = appendLog(in.HomeDir, ev)

	// Spool to disk so a failed send survives until next invocation.
	if err := appendSpool(in.HomeDir, ev); err != nil {
		return
	}

	endpoint := cfg.Telemetry.Endpoint
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}

	// Fire-and-forget: hand the HTTP send off to a detached child process
	// and return. Tests override flusher with a synchronous implementation
	// so they can assert against an httptest server in-process.
	flusher(in.HomeDir, endpoint)
}

// CanonicalCommand turns a Cobra cmd.CommandPath() into the canonical form we
// record: the path with the leading "magebox" root stripped. The bare root
// command (no subcommand) canonicalizes to the empty string, which
// ShouldSkipCommand then filters out. For custom project commands handled by
// the delegation in main.go, the caller should pass the literal string "run"
// so that no user-defined command name leaks.
func CanonicalCommand(commandPath string) string {
	cmd := strings.TrimSpace(commandPath)
	if cmd == "magebox" {
		return ""
	}
	cmd = strings.TrimPrefix(cmd, "magebox ")
	return strings.TrimSpace(cmd)
}

// ShouldSkipCommand returns true for commands that should never be recorded,
// either because they are telemetry-management commands themselves, or because
// recording them would be noise (help, version, completion).
func ShouldSkipCommand(cmd string) bool {
	if cmd == "" {
		return true
	}
	// Meta / self-referential
	switch cmd {
	case "help", "completion", "version", "self-update":
		return true
	}
	if strings.HasPrefix(cmd, "telemetry") {
		return true
	}
	if strings.HasPrefix(cmd, "help ") || strings.HasPrefix(cmd, "completion ") {
		return true
	}
	return false
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

// PathsFor returns the on-disk paths used by the telemetry package, rooted at
// the given home directory. Exposed for tests and the `telemetry show` /
// `telemetry purge` commands.
type Paths struct {
	Dir   string
	ID    string
	Spool string
	Log   string
}

func PathsFor(homeDir string) Paths {
	base := filepath.Join(homeDir, ".magebox")
	return Paths{
		Dir:   base,
		ID:    filepath.Join(base, "telemetry.id"),
		Spool: filepath.Join(base, "telemetry-spool.jsonl"),
		Log:   filepath.Join(base, "telemetry-log.jsonl"),
	}
}

// ReadLog returns events in the local rolling log in chronological order. It
// returns an empty slice (not an error) if the log doesn't exist yet.
func ReadLog(homeDir string) ([]Event, error) {
	p := PathsFor(homeDir).Log
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return decodeJSONL(data), nil
}

// Purge removes all telemetry state (id, spool, log) from disk. Used by
// `magebox telemetry purge`.
func Purge(homeDir string) error {
	p := PathsFor(homeDir)
	for _, path := range []string{p.ID, p.Spool, p.Log} {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func decodeJSONL(data []byte) []Event {
	var out []Event
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var ev Event
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}
		out = append(out, ev)
	}
	return out
}

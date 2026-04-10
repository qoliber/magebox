package telemetry

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"qoliber/magebox/internal/config"
)

func TestCanonicalCommand(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"root only", "magebox", ""},
		{"top-level", "magebox start", "start"},
		{"nested", "magebox service redis", "service redis"},
		{"no prefix", "start", "start"},
		{"trailing space", "magebox start ", "start"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := CanonicalCommand(tc.in); got != tc.want {
				t.Errorf("CanonicalCommand(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestShouldSkipCommand(t *testing.T) {
	skip := []string{
		"",
		"help",
		"help start",
		"completion",
		"completion bash",
		"version",
		"self-update",
		"telemetry",
		"telemetry enable",
	}
	for _, c := range skip {
		if !ShouldSkipCommand(c) {
			t.Errorf("ShouldSkipCommand(%q) = false, want true", c)
		}
	}

	keep := []string{"start", "stop", "service redis", "run", "db import"}
	for _, c := range keep {
		if ShouldSkipCommand(c) {
			t.Errorf("ShouldSkipCommand(%q) = true, want false", c)
		}
	}
}

func TestLoadOrCreateAnonID_PersistsAndReuses(t *testing.T) {
	home := t.TempDir()

	first, err := loadOrCreateAnonID(home)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if _, err := uuid.Parse(first); err != nil {
		t.Fatalf("first id is not a UUID: %v", err)
	}

	second, err := loadOrCreateAnonID(home)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if first != second {
		t.Errorf("id was regenerated on second call: %q != %q", first, second)
	}
}

func TestLoadOrCreateAnonID_RegeneratesOnCorrupt(t *testing.T) {
	home := t.TempDir()
	p := PathsFor(home)
	if err := os.MkdirAll(p.Dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.ID, []byte("not a uuid\n"), 0644); err != nil {
		t.Fatal(err)
	}

	id, err := loadOrCreateAnonID(home)
	if err != nil {
		t.Fatalf("loadOrCreateAnonID: %v", err)
	}
	if _, err := uuid.Parse(id); err != nil {
		t.Errorf("expected regenerated uuid, got %q", id)
	}
}

func TestResetAnonID(t *testing.T) {
	home := t.TempDir()
	first, err := loadOrCreateAnonID(home)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ResetAnonID(home)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Error("ResetAnonID returned the same id")
	}
	read := ReadAnonID(home)
	if read != second {
		t.Errorf("ReadAnonID = %q, want %q", read, second)
	}
}

func TestAppendCapped_RingBuffer(t *testing.T) {
	home := t.TempDir()
	p := PathsFor(home)
	if err := os.MkdirAll(p.Dir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write logMax+5 events; we should end up with exactly logMax.
	for i := 0; i < logMax+5; i++ {
		ev := Event{EventID: uuid.NewString(), Command: "start", Timestamp: time.Now().UTC().Format(time.RFC3339)}
		if err := appendLog(home, ev); err != nil {
			t.Fatalf("appendLog: %v", err)
		}
	}

	events, err := ReadLog(home)
	if err != nil {
		t.Fatalf("ReadLog: %v", err)
	}
	if len(events) != logMax {
		t.Errorf("log length = %d, want %d", len(events), logMax)
	}
}

func TestRecord_DisabledIsNoop(t *testing.T) {
	home := t.TempDir()

	// No telemetry config saved → disabled by default.
	Record(RecordInput{
		HomeDir:   home,
		MBVersion: "test",
		Command:   "start",
		ExitCode:  0,
		Duration:  100 * time.Millisecond,
	})

	// Nothing should have been written.
	p := PathsFor(home)
	for _, path := range []string{p.ID, p.Spool, p.Log} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("expected %s to not exist when disabled, got err=%v", filepath.Base(path), err)
		}
	}
}

func TestRecord_EndToEnd_SendsAndClearsSpool(t *testing.T) {
	// Swap the detached flusher for the synchronous one so we can assert
	// against the httptest server in-process.
	origFlusher := flusher
	flusher = flushSpool
	defer func() { flusher = origFlusher }()

	home := t.TempDir()

	var received atomic.Int32
	var gotPayload batchPayload
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotPayload)
		received.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	cfg := &config.GlobalConfig{
		Telemetry: &config.TelemetryConfig{
			Enabled:  true,
			Endpoint: srv.URL,
			Prompted: true,
		},
	}
	if err := config.SaveGlobalConfig(home, cfg); err != nil {
		t.Fatal(err)
	}

	Record(RecordInput{
		HomeDir:   home,
		MBVersion: "1.14.0",
		Command:   "start",
		ExitCode:  0,
		Duration:  250 * time.Millisecond,
	})

	if got := received.Load(); got != 1 {
		t.Errorf("received count = %d, want 1", got)
	}

	if len(gotPayload.Events) != 1 {
		t.Fatalf("payload events = %d, want 1", len(gotPayload.Events))
	}
	ev := gotPayload.Events[0]
	if ev.Command != "start" {
		t.Errorf("command = %q, want start", ev.Command)
	}
	if ev.DurationMs != 250 {
		t.Errorf("duration_ms = %d, want 250", ev.DurationMs)
	}
	if ev.MBVersion != "1.14.0" {
		t.Errorf("mb_version = %q, want 1.14.0", ev.MBVersion)
	}
	if _, err := uuid.Parse(ev.AnonID); err != nil {
		t.Errorf("anon_id is not a uuid: %q", ev.AnonID)
	}

	// Spool should be empty after successful flush.
	if data, _ := os.ReadFile(PathsFor(home).Spool); len(strings.TrimSpace(string(data))) != 0 {
		t.Errorf("spool not cleared after send: %q", string(data))
	}

	// But log should contain the event.
	logged, err := ReadLog(home)
	if err != nil {
		t.Fatal(err)
	}
	if len(logged) != 1 {
		t.Errorf("log length = %d, want 1", len(logged))
	}
}

func TestRecord_FailedSendKeepsSpool(t *testing.T) {
	// Use the synchronous flusher so the failed send is observable before
	// the test asserts on the surviving spool.
	origFlusher := flusher
	flusher = flushSpool
	defer func() { flusher = origFlusher }()

	home := t.TempDir()

	// Server always errors — spool should survive for the next run.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	cfg := &config.GlobalConfig{
		Telemetry: &config.TelemetryConfig{
			Enabled:  true,
			Endpoint: srv.URL,
			Prompted: true,
		},
	}
	if err := config.SaveGlobalConfig(home, cfg); err != nil {
		t.Fatal(err)
	}

	Record(RecordInput{
		HomeDir:   home,
		MBVersion: "1.14.0",
		Command:   "stop",
		ExitCode:  1,
		Duration:  10 * time.Millisecond,
	})

	spooled, err := readSpool(home)
	if err != nil {
		t.Fatal(err)
	}
	if len(spooled) != 1 {
		t.Errorf("spool length = %d, want 1 (should survive failed send)", len(spooled))
	}
	if len(spooled) == 1 && spooled[0].ExitCode != 1 {
		t.Errorf("spooled exit_code = %d, want 1", spooled[0].ExitCode)
	}
}

func TestPurge_RemovesAllState(t *testing.T) {
	home := t.TempDir()
	p := PathsFor(home)
	if err := os.MkdirAll(p.Dir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{p.ID, p.Spool, p.Log} {
		if err := os.WriteFile(f, []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	if err := Purge(home); err != nil {
		t.Fatalf("Purge: %v", err)
	}
	for _, f := range []string{p.ID, p.Spool, p.Log} {
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			t.Errorf("%s still exists after Purge", filepath.Base(f))
		}
	}
}

func TestShouldPrompt(t *testing.T) {
	if !ShouldPrompt(nil) {
		t.Error("ShouldPrompt(nil) = false, want true")
	}
	cfg := &config.GlobalConfig{}
	if !ShouldPrompt(cfg) {
		t.Error("ShouldPrompt(empty) = false, want true")
	}
	cfg.Telemetry = &config.TelemetryConfig{Prompted: false}
	if !ShouldPrompt(cfg) {
		t.Error("ShouldPrompt(prompted=false) = false, want true")
	}
	cfg.Telemetry.Prompted = true
	if ShouldPrompt(cfg) {
		t.Error("ShouldPrompt(prompted=true) = true, want false")
	}
}

func TestMaybePrompt_SkipsForTelemetrySubcommand(t *testing.T) {
	home := t.TempDir()
	cfg := &config.GlobalConfig{}
	enabled, err := MaybePrompt(home, cfg, "telemetry enable", strings.NewReader("y\n"), io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if enabled {
		t.Error("enabled = true, want false (should have been skipped)")
	}
	if cfg.Telemetry != nil && cfg.Telemetry.Prompted {
		t.Error("Prompted was set even though prompt should have been skipped")
	}
}

func TestTruncate(t *testing.T) {
	if truncate("hello", 64) != "hello" {
		t.Error("short string should be unchanged")
	}
	long := strings.Repeat("a", 100)
	if got := truncate(long, 64); len(got) != 64 {
		t.Errorf("truncate cap = %d, want 64", len(got))
	}
}

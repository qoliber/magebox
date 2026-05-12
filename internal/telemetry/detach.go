package telemetry

import (
	"os"
	"os/exec"
	"syscall"
)

// Environment variables used to signal a detached child process to perform
// the spool flush. When the MageBox binary sees envFlushTrigger=1 at startup
// it runs the flush and exits without executing any other CLI code.
const (
	envFlushTrigger  = "MAGEBOX_TELEMETRY_FLUSH"
	envFlushHome     = "MAGEBOX_TELEMETRY_HOME"
	envFlushEndpoint = "MAGEBOX_TELEMETRY_ENDPOINT"
)

// flusher is the function Record uses to hand spooled events off to the
// network. In production it spawns a fully detached child process so the
// parent can exit immediately regardless of network state. Tests override it
// with flushSpool for synchronous assertions against an httptest server.
var flusher = detachFlush

// detachFlush re-invokes the current binary with a marker env var that tells
// the child to run the HTTP send and exit. It returns immediately after
// Start(); a slow or dead ingestion server cannot delay the parent at all.
//
// The child is placed in a new session (Setsid) so it outlives its parent's
// process group, and all three stdio streams are routed to /dev/null so it
// can never write to the parent's TTY. When the parent exits, the child is
// reparented to init/systemd, which handles reaping.
//
// All failure modes are silent: a missing executable path, a failed
// os.OpenFile on /dev/null, or a failed Start() all leave the event on the
// local spool for the next invocation to retry.
func detachFlush(homeDir, endpoint string) {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	cmd := exec.Command(exe)
	cmd.Env = append(os.Environ(),
		envFlushTrigger+"=1",
		envFlushHome+"="+homeDir,
		envFlushEndpoint+"="+endpoint,
	)
	if devNull, derr := os.OpenFile(os.DevNull, os.O_RDWR, 0); derr == nil {
		cmd.Stdin = devNull
		cmd.Stdout = devNull
		cmd.Stderr = devNull
		defer func() { _ = devNull.Close() }()
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	_ = cmd.Start()
	// Intentionally no Wait — the child is fully detached and will be
	// reaped by init once the parent exits.
}

// FlushFromEnv reports whether this process was spawned by detachFlush. When
// it returns true the caller (main) must exit immediately — the flush has
// already been performed and no CLI command should run. Returns false for
// normal invocations.
func FlushFromEnv() bool {
	if os.Getenv(envFlushTrigger) != "1" {
		return false
	}
	home := os.Getenv(envFlushHome)
	endpoint := os.Getenv(envFlushEndpoint)
	if home != "" && endpoint != "" {
		flushSpool(home, endpoint)
	}
	return true
}

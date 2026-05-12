package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// flushBudget caps how long the detached flush child spends attempting to
// send spooled events. It can be generous: the child runs out-of-band from
// the user's CLI invocation, so the parent has already exited by the time
// this matters. Tests use this same budget against an in-process httptest
// server, where responses come back in microseconds.
const flushBudget = 10 * time.Second

// batchPayload is the body shape the ingestion server expects. Keeping the
// batch wrapper even for single-event sends means the server has one code path.
type batchPayload struct {
	SchemaVersion int     `json:"schema_version"`
	Events        []Event `json:"events"`
}

// flushSpool reads the spool, attempts to POST it to endpoint within
// flushBudget, and clears the spool on success. On failure it leaves the
// spool intact for the next invocation. It is used by the detached flush
// child (via FlushFromEnv in detach.go) and swapped in directly by tests.
func flushSpool(homeDir, endpoint string) {
	events, err := readSpool(homeDir)
	if err != nil || len(events) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), flushBudget)
	defer cancel()

	if err := send(ctx, endpoint, events); err != nil {
		return
	}

	_ = clearSpool(homeDir)
}

func send(ctx context.Context, endpoint string, events []Event) error {
	payload := batchPayload{
		SchemaVersion: SchemaVersion,
		Events:        events,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "MageBox-Telemetry")

	client := &http.Client{Timeout: flushBudget}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	return &httpError{Status: resp.StatusCode}
}

type httpError struct {
	Status int
}

func (e *httpError) Error() string {
	return http.StatusText(e.Status)
}

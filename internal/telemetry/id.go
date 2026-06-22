package telemetry

import (
	"os"
	"strings"

	"github.com/google/uuid"
)

// loadOrCreateAnonID returns the anonymous installation ID stored at
// ~/.magebox/telemetry.id, creating it the first time it's requested. The ID
// is a v4 UUID — it contains no data about the user, host or filesystem and
// can be safely regenerated with `magebox telemetry reset-id`.
func loadOrCreateAnonID(homeDir string) (string, error) {
	p := PathsFor(homeDir)
	if data, err := os.ReadFile(p.ID); err == nil {
		id := strings.TrimSpace(string(data))
		if _, err := uuid.Parse(id); err == nil {
			return id, nil
		}
		// Fall through on corrupt content — regenerate.
	}

	if err := os.MkdirAll(p.Dir, 0755); err != nil {
		return "", err
	}

	id := uuid.NewString()
	if err := os.WriteFile(p.ID, []byte(id+"\n"), 0644); err != nil {
		return "", err
	}
	return id, nil
}

// ResetAnonID generates and persists a new anonymous ID, returning it. Used by
// `magebox telemetry reset-id`.
func ResetAnonID(homeDir string) (string, error) {
	p := PathsFor(homeDir)
	if err := os.MkdirAll(p.Dir, 0755); err != nil {
		return "", err
	}
	id := uuid.NewString()
	if err := os.WriteFile(p.ID, []byte(id+"\n"), 0644); err != nil {
		return "", err
	}
	return id, nil
}

// ReadAnonID returns the current anonymous ID, or empty string if none exists.
// Unlike loadOrCreateAnonID it never creates the file — used by status/show.
func ReadAnonID(homeDir string) string {
	data, err := os.ReadFile(PathsFor(homeDir).ID)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

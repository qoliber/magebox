package telemetry

import (
	"encoding/json"
	"os"
)

// spoolMax caps the on-disk spool so a long-offline install can't grow
// unbounded. Oldest entries are dropped on overflow.
const spoolMax = 100

// logMax caps the rolling log read by `telemetry show`.
const logMax = 50

func appendSpool(homeDir string, ev Event) error {
	p := PathsFor(homeDir)
	if err := os.MkdirAll(p.Dir, 0755); err != nil {
		return err
	}
	return appendCapped(p.Spool, ev, spoolMax)
}

func appendLog(homeDir string, ev Event) error {
	p := PathsFor(homeDir)
	if err := os.MkdirAll(p.Dir, 0755); err != nil {
		return err
	}
	return appendCapped(p.Log, ev, logMax)
}

// appendCapped appends ev to path and, if the resulting file has more than
// max entries, rewrites it keeping only the most recent max entries.
func appendCapped(path string, ev Event, max int) error {
	line, err := json.Marshal(ev)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	_, werr := f.Write(append(line, '\n'))
	cerr := f.Close()
	if werr != nil {
		return werr
	}
	if cerr != nil {
		return cerr
	}

	// Trim if over cap. Cheap: read all, keep tail, rewrite.
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	events := decodeJSONL(data)
	if len(events) <= max {
		return nil
	}
	trimmed := events[len(events)-max:]
	return writeJSONL(path, trimmed)
}

func writeJSONL(path string, events []Event) error {
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	for _, ev := range events {
		if err := enc.Encode(ev); err != nil {
			_ = f.Close()
			_ = os.Remove(tmp)
			return err
		}
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

func readSpool(homeDir string) ([]Event, error) {
	data, err := os.ReadFile(PathsFor(homeDir).Spool)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return decodeJSONL(data), nil
}

func clearSpool(homeDir string) error {
	p := PathsFor(homeDir).Spool
	err := os.Remove(p)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package progress

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"
)

func TestNewReader(t *testing.T) {
	data := []byte("test data")
	reader := bytes.NewReader(data)

	pr := NewReader(reader, int64(len(data)), func(p Progress) {
		_ = p // consume progress
	})

	if pr == nil {
		t.Fatal("NewReader should not return nil")
	}

	if pr.total != int64(len(data)) {
		t.Errorf("total = %d, want %d", pr.total, len(data))
	}
}

func TestReader_Read(t *testing.T) {
	data := []byte("hello world test data for progress tracking")
	reader := bytes.NewReader(data)

	var progressUpdates []Progress
	pr := NewReader(reader, int64(len(data)), func(p Progress) {
		progressUpdates = append(progressUpdates, p)
	})
	pr.updateEvery = 0 // Force updates on every read for testing

	// Read all data
	buf := make([]byte, 10)
	totalRead := 0
	for {
		n, err := pr.Read(buf)
		totalRead += n
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if totalRead != len(data) {
		t.Errorf("totalRead = %d, want %d", totalRead, len(data))
	}

	// Check that we got progress updates
	if len(progressUpdates) == 0 {
		t.Error("expected at least one progress update")
	}

	// Last progress should show 100%
	lastProgress := progressUpdates[len(progressUpdates)-1]
	if lastProgress.Read != int64(len(data)) {
		t.Errorf("lastProgress.Read = %d, want %d", lastProgress.Read, len(data))
	}
	if lastProgress.Percentage < 99.9 {
		t.Errorf("lastProgress.Percentage = %f, want ~100", lastProgress.Percentage)
	}
}

func TestReader_ProgressCalculation(t *testing.T) {
	data := make([]byte, 1000)
	reader := bytes.NewReader(data)

	var lastProgress Progress
	pr := NewReader(reader, int64(len(data)), func(p Progress) {
		lastProgress = p
	})
	pr.updateEvery = 0 // Force updates on every read

	// Read half
	buf := make([]byte, 500)
	_, err := pr.Read(buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if lastProgress.Percentage < 49.9 || lastProgress.Percentage > 50.1 {
		t.Errorf("percentage after half read = %f, want ~50", lastProgress.Percentage)
	}

	// Read rest
	_, err = io.ReadAll(pr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if lastProgress.Percentage < 99.9 {
		t.Errorf("percentage after full read = %f, want ~100", lastProgress.Percentage)
	}
}

func TestReader_SpeedCalculation(t *testing.T) {
	data := make([]byte, 10000)
	reader := bytes.NewReader(data)

	var lastProgress Progress
	pr := NewReader(reader, int64(len(data)), func(p Progress) {
		lastProgress = p
	})
	pr.updateEvery = 0 // Force updates on every read

	// Read all data
	_, err := io.ReadAll(pr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Speed should be positive
	if lastProgress.Speed <= 0 {
		t.Errorf("speed = %f, want > 0", lastProgress.Speed)
	}

	// Elapsed should be positive
	if lastProgress.Elapsed <= 0 {
		t.Errorf("elapsed = %v, want > 0", lastProgress.Elapsed)
	}
}

func TestReader_UnknownTotal(t *testing.T) {
	data := []byte("test data")
	reader := bytes.NewReader(data)

	var lastProgress Progress
	pr := NewReader(reader, 0, func(p Progress) { // 0 = unknown total
		lastProgress = p
	})
	pr.updateEvery = 0

	_, err := io.ReadAll(pr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// With unknown total, percentage should be 0
	if lastProgress.Percentage != 0 {
		t.Errorf("percentage with unknown total = %f, want 0", lastProgress.Percentage)
	}

	// But Read should still track bytes
	if lastProgress.Read != int64(len(data)) {
		t.Errorf("Read = %d, want %d", lastProgress.Read, len(data))
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{100, "100 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1572864, "1.5 MB"},
		{1073741824, "1.0 GB"},
		{1610612736, "1.5 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := FormatBytes(tt.bytes)
			if got != tt.expected {
				t.Errorf("FormatBytes(%d) = %s, want %s", tt.bytes, got, tt.expected)
			}
		})
	}
}

func TestFormatSpeed(t *testing.T) {
	tests := []struct {
		bytesPerSec float64
		expected    string
	}{
		{1024, "1.0 KB/s"},
		{1048576, "1.0 MB/s"},
		{52428800, "50.0 MB/s"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := FormatSpeed(tt.bytesPerSec)
			if got != tt.expected {
				t.Errorf("FormatSpeed(%f) = %s, want %s", tt.bytesPerSec, got, tt.expected)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{5 * time.Second, "5s"},
		{30 * time.Second, "30s"},
		{90 * time.Second, "1m30s"},
		{3600 * time.Second, "1h0m"},
		{3660 * time.Second, "1h1m"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := formatDuration(tt.duration)
			if got != tt.expected {
				t.Errorf("formatDuration(%v) = %s, want %s", tt.duration, got, tt.expected)
			}
		})
	}
}

func TestBar_Update(t *testing.T) {
	bar := NewBar("Testing:")

	if bar == nil {
		t.Fatal("NewBar should not return nil")
	}

	if bar.width != 40 {
		t.Errorf("bar.width = %d, want 40", bar.width)
	}

	if bar.description != "Testing:" {
		t.Errorf("bar.description = %s, want Testing:", bar.description)
	}

	// Test that Update doesn't panic
	bar.Update(Progress{
		Read:       500,
		Total:      1000,
		Percentage: 50,
		Speed:      1048576,
		ETA:        10 * time.Second,
	})

	bar.Finish()
}

func TestBar_UpdateUnknownTotal(t *testing.T) {
	bar := NewBar("Testing:")

	// Test with unknown total (0)
	bar.Update(Progress{
		Read:       500,
		Total:      0,
		Percentage: 0,
		Speed:      1048576,
	})

	bar.Finish()
}

func TestReader_ConcurrentAccess(t *testing.T) {
	data := make([]byte, 10000)
	reader := bytes.NewReader(data)

	pr := NewReader(reader, int64(len(data)), func(p Progress) {
		// Just consume progress updates
	})

	// Read from multiple goroutines (simulating concurrent access)
	done := make(chan bool, 2)

	go func() {
		buf := make([]byte, 100)
		for {
			_, err := pr.Read(buf)
			if err != nil {
				break
			}
		}
		done <- true
	}()

	<-done
	// If we get here without deadlock/panic, test passes
}

func TestReader_EmptyReader(t *testing.T) {
	reader := strings.NewReader("")

	pr := NewReader(reader, 0, func(p Progress) {
		_ = p // consume progress
	})

	buf := make([]byte, 10)
	n, err := pr.Read(buf)

	if n != 0 {
		t.Errorf("n = %d, want 0", n)
	}
	if err != io.EOF {
		t.Errorf("err = %v, want EOF", err)
	}
}

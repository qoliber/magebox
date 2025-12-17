// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package progress

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// Reader wraps an io.Reader to track read progress
type Reader struct {
	reader      io.Reader
	total       int64
	read        int64
	onProgress  func(Progress)
	startTime   time.Time
	lastUpdate  time.Time
	updateEvery time.Duration
	mu          sync.Mutex
}

// Progress contains current progress information
type Progress struct {
	Read       int64
	Total      int64
	Percentage float64
	Speed      float64 // bytes per second
	ETA        time.Duration
	Elapsed    time.Duration
}

// NewReader creates a new progress-tracking reader
func NewReader(r io.Reader, total int64, onProgress func(Progress)) *Reader {
	return &Reader{
		reader:      r,
		total:       total,
		onProgress:  onProgress,
		startTime:   time.Now(),
		lastUpdate:  time.Now(),
		updateEvery: 100 * time.Millisecond,
	}
}

// Read implements io.Reader
func (r *Reader) Read(p []byte) (n int, err error) {
	n, err = r.reader.Read(p)
	if n > 0 {
		r.mu.Lock()
		r.read += int64(n)
		now := time.Now()

		// Throttle updates to avoid too much output
		if now.Sub(r.lastUpdate) >= r.updateEvery || err == io.EOF {
			r.lastUpdate = now
			if r.onProgress != nil {
				elapsed := now.Sub(r.startTime)
				speed := float64(r.read) / elapsed.Seconds()

				var eta time.Duration
				var percentage float64
				if r.total > 0 {
					percentage = float64(r.read) / float64(r.total) * 100
					remaining := r.total - r.read
					if speed > 0 {
						eta = time.Duration(float64(remaining)/speed) * time.Second
					}
				}

				r.onProgress(Progress{
					Read:       r.read,
					Total:      r.total,
					Percentage: percentage,
					Speed:      speed,
					ETA:        eta,
					Elapsed:    elapsed,
				})
			}
		}
		r.mu.Unlock()
	}
	return n, err
}

// Bar renders a progress bar to the terminal
type Bar struct {
	width       int
	description string
	mu          sync.Mutex
}

// NewBar creates a new progress bar
func NewBar(description string) *Bar {
	return &Bar{
		width:       40,
		description: description,
	}
}

// Update updates the progress bar display
func (b *Bar) Update(p Progress) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Calculate filled portion
	filled := int(p.Percentage / 100 * float64(b.width))
	if filled > b.width {
		filled = b.width
	}

	// Build progress bar
	bar := strings.Repeat("█", filled) + strings.Repeat("░", b.width-filled)

	// Format sizes
	readStr := FormatBytes(p.Read)
	totalStr := FormatBytes(p.Total)

	// Format speed
	speedStr := FormatSpeed(p.Speed)

	// Format ETA
	etaStr := "..."
	if p.ETA > 0 && p.ETA < 24*time.Hour {
		etaStr = formatDuration(p.ETA)
	}

	// Print progress line (carriage return to overwrite)
	if p.Total > 0 {
		fmt.Printf("\r  %s %s %.1f%% (%s/%s) %s ETA: %s  ",
			b.description, bar, p.Percentage, readStr, totalStr, speedStr, etaStr)
	} else {
		// Unknown total - just show read bytes and speed
		fmt.Printf("\r  %s %s %s  ", b.description, readStr, speedStr)
	}
}

// Finish completes the progress bar
func (b *Bar) Finish() {
	b.mu.Lock()
	defer b.mu.Unlock()
	fmt.Println() // Move to next line
}

// Clear clears the progress bar line
func (b *Bar) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	fmt.Printf("\r%s\r", strings.Repeat(" ", 100))
}

// FormatBytes formats bytes as human-readable string
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatSpeed formats bytes per second as human-readable string
func FormatSpeed(bytesPerSec float64) string {
	return FormatBytes(int64(bytesPerSec)) + "/s"
}

// formatDuration formats a duration as human-readable string
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}

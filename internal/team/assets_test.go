// Copyright (c) qoliber
// Author: Jakub Winkler <jwinkler@qoliber.com>

package team

import (
	"testing"
	"time"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{
			name:     "bytes",
			bytes:    500,
			expected: "500 B",
		},
		{
			name:     "kilobytes",
			bytes:    1536,
			expected: "1.50 KB",
		},
		{
			name:     "megabytes",
			bytes:    1572864,
			expected: "1.50 MB",
		},
		{
			name:     "gigabytes",
			bytes:    1610612736,
			expected: "1.50 GB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("FormatBytes(%d) = %q, want %q", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestFormatSpeed(t *testing.T) {
	tests := []struct {
		name     string
		speed    float64
		expected string
	}{
		{
			name:     "bytes per second",
			speed:    500,
			expected: "500 B/s",
		},
		{
			name:     "kilobytes per second",
			speed:    1536,
			expected: "1.50 KB/s",
		},
		{
			name:     "megabytes per second",
			speed:    1572864,
			expected: "1.50 MB/s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatSpeed(tt.speed)
			if result != tt.expected {
				t.Errorf("FormatSpeed(%f) = %q, want %q", tt.speed, result, tt.expected)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "seconds only",
			duration: 45 * time.Second,
			expected: "45s",
		},
		{
			name:     "minutes and seconds",
			duration: 5*time.Minute + 30*time.Second,
			expected: "5m 30s",
		},
		{
			name:     "hours and minutes",
			duration: 2*time.Hour + 15*time.Minute,
			expected: "2h 15m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, result, tt.expected)
			}
		})
	}
}

func TestNewAssetClient(t *testing.T) {
	team := &Team{
		Name: "testteam",
		Assets: AssetConfig{
			Provider: AssetSFTP,
			Host:     "backup.example.com",
			Port:     22,
			Path:     "/backups",
			Username: "deploy",
		},
	}

	progressFn := func(p DownloadProgress) {
		// Progress callback
		_ = p
	}

	client := NewAssetClient(team, progressFn)
	if client == nil {
		t.Fatal("NewAssetClient returned nil")
	}
	if client.team != team {
		t.Error("AssetClient team reference mismatch")
	}
	if client.progress == nil {
		t.Error("AssetClient progress function not set")
	}
}

func TestDownloadProgress(t *testing.T) {
	progress := DownloadProgress{
		Filename:   "test.sql.gz",
		TotalBytes: 1000000,
		Downloaded: 500000,
		Speed:      100000,
		Percentage: 50.0,
		ETA:        "5s",
	}

	if progress.Filename != "test.sql.gz" {
		t.Errorf("Expected filename test.sql.gz, got %s", progress.Filename)
	}
	if progress.Percentage != 50.0 {
		t.Errorf("Expected percentage 50.0, got %f", progress.Percentage)
	}
}

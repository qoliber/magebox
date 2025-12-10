package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// LogTailer watches and tails multiple log files
type LogTailer struct {
	logDir     string
	pattern    string
	follow     bool
	lines      int
	watcher    *fsnotify.Watcher
	files      map[string]*tailedFile
	mutex      sync.Mutex
	stopChan   chan struct{}
	levelRegex *regexp.Regexp
}

// tailedFile represents a file being tailed
type tailedFile struct {
	path   string
	file   *os.File
	offset int64
	name   string
}

// NewLogTailer creates a new log tailer
func NewLogTailer(logDir string, pattern string, follow bool, lines int) *LogTailer {
	// Regex to extract log level from Magento log format
	// Format: [2024-01-15T10:30:45.123456+00:00] main.CRITICAL: ...
	levelRegex := regexp.MustCompile(`\]\s+\w+\.(\w+):`)

	return &LogTailer{
		logDir:     logDir,
		pattern:    pattern,
		follow:     follow,
		lines:      lines,
		files:      make(map[string]*tailedFile),
		stopChan:   make(chan struct{}),
		levelRegex: levelRegex,
	}
}

// Start begins tailing log files
func (t *LogTailer) Start() error {
	// Find matching log files
	files, err := t.findLogFiles()
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return fmt.Errorf("no log files found matching pattern '%s' in %s", t.pattern, t.logDir)
	}

	// Print initial lines from each file
	for _, path := range files {
		if err := t.initFile(path); err != nil {
			PrintWarning("Could not read %s: %v", filepath.Base(path), err)
			continue
		}
	}

	// If not following, we're done
	if !t.follow {
		return nil
	}

	// Set up file watcher for follow mode
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	t.watcher = watcher

	// Watch the log directory for new files
	if err := watcher.Add(t.logDir); err != nil {
		return fmt.Errorf("failed to watch directory: %w", err)
	}

	// Watch each log file
	for _, path := range files {
		if err := watcher.Add(path); err != nil {
			PrintWarning("Could not watch %s: %v", filepath.Base(path), err)
		}
	}

	fmt.Println()
	PrintInfo("Watching for changes... (Ctrl+C to stop)")
	fmt.Println()

	// Start watching
	go t.watchLoop()

	// Wait for stop signal
	<-t.stopChan
	return nil
}

// Stop stops the log tailer
func (t *LogTailer) Stop() {
	close(t.stopChan)
	if t.watcher != nil {
		t.watcher.Close()
	}

	// Close all open files
	t.mutex.Lock()
	defer t.mutex.Unlock()
	for _, tf := range t.files {
		if tf.file != nil {
			tf.file.Close()
		}
	}
}

// findLogFiles finds all log files matching the pattern
func (t *LogTailer) findLogFiles() ([]string, error) {
	var files []string

	pattern := t.pattern
	if pattern == "" {
		pattern = "*.log"
	}

	// Walk the log directory
	err := filepath.Walk(t.logDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		if info.IsDir() {
			return nil
		}

		// Match against pattern
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err != nil {
			return nil
		}

		if matched {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// initFile initializes a file for tailing
func (t *LogTailer) initFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}

	name := filepath.Base(path)

	// Get file info for size
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return err
	}

	tf := &tailedFile{
		path:   path,
		file:   file,
		name:   name,
		offset: 0,
	}

	// If we want to show last N lines, seek to appropriate position
	if t.lines > 0 && info.Size() > 0 {
		lines := t.readLastLines(file, t.lines)
		if len(lines) > 0 {
			fmt.Println(Header(name))
			for _, line := range lines {
				t.printLogLine(name, line)
			}
		}
		// Set offset to end of file for follow mode
		tf.offset, _ = file.Seek(0, io.SeekEnd)
	} else {
		tf.offset = info.Size()
	}

	t.mutex.Lock()
	t.files[path] = tf
	t.mutex.Unlock()

	return nil
}

// readLastLines reads the last n lines from a file
func (t *LogTailer) readLastLines(file *os.File, n int) []string {
	// Seek to end
	info, err := file.Stat()
	if err != nil {
		return nil
	}

	size := info.Size()
	if size == 0 {
		return nil
	}

	// Read chunks from the end to find lines
	var lines []string
	chunkSize := int64(8192)
	offset := size

	for offset > 0 && len(lines) < n {
		// Calculate chunk start
		start := offset - chunkSize
		if start < 0 {
			start = 0
		}

		// Read chunk
		chunk := make([]byte, offset-start)
		_, err := file.ReadAt(chunk, start)
		if err != nil && err != io.EOF {
			break
		}

		// Split into lines (in reverse)
		chunkLines := strings.Split(string(chunk), "\n")

		// Prepend to lines (they're in reverse order)
		for i := len(chunkLines) - 1; i >= 0; i-- {
			line := strings.TrimRight(chunkLines[i], "\r")
			if line != "" {
				lines = append([]string{line}, lines...)
			}
			if len(lines) >= n {
				break
			}
		}

		offset = start
	}

	// Return only last n lines
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}

	return lines
}

// watchLoop watches for file changes
func (t *LogTailer) watchLoop() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-t.stopChan:
			return

		case event, ok := <-t.watcher.Events:
			if !ok {
				return
			}

			if event.Has(fsnotify.Write) {
				t.handleFileChange(event.Name)
			} else if event.Has(fsnotify.Create) {
				// New file created - check if it matches our pattern
				if matched, _ := filepath.Match(t.pattern, filepath.Base(event.Name)); matched {
					_ = t.initFile(event.Name)
					_ = t.watcher.Add(event.Name)
				}
			}

		case err, ok := <-t.watcher.Errors:
			if !ok {
				return
			}
			PrintWarning("Watcher error: %v", err)

		case <-ticker.C:
			// Periodic check for changes (backup for fsnotify)
			t.checkAllFiles()
		}
	}
}

// handleFileChange handles a file modification event
func (t *LogTailer) handleFileChange(path string) {
	t.mutex.Lock()
	tf, ok := t.files[path]
	t.mutex.Unlock()

	if !ok {
		return
	}

	t.readNewContent(tf)
}

// checkAllFiles checks all files for new content
func (t *LogTailer) checkAllFiles() {
	t.mutex.Lock()
	files := make([]*tailedFile, 0, len(t.files))
	for _, tf := range t.files {
		files = append(files, tf)
	}
	t.mutex.Unlock()

	for _, tf := range files {
		t.readNewContent(tf)
	}
}

// readNewContent reads new content from a file
func (t *LogTailer) readNewContent(tf *tailedFile) {
	// Check current file size
	info, err := tf.file.Stat()
	if err != nil {
		return
	}

	currentSize := info.Size()

	// File was truncated (log rotation)
	if currentSize < tf.offset {
		tf.offset = 0
	}

	// No new content
	if currentSize <= tf.offset {
		return
	}

	// Seek to last position
	_, _ = tf.file.Seek(tf.offset, io.SeekStart)

	// Read new content
	reader := bufio.NewReader(tf.file)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				break
			}
			// Partial line - don't print yet, wait for newline
			if line != "" {
				// Seek back to before the partial line
				_, _ = tf.file.Seek(tf.offset, io.SeekStart)
				return
			}
			break
		}

		line = strings.TrimRight(line, "\n\r")
		if line != "" {
			t.printLogLine(tf.name, line)
		}
	}

	// Update offset
	tf.offset, _ = tf.file.Seek(0, io.SeekCurrent)
}

// printLogLine prints a formatted log line
func (t *LogTailer) printLogLine(filename, line string) {
	// Extract and colorize log level
	coloredLine := line

	// Try to match Magento log format and colorize level
	if matches := t.levelRegex.FindStringSubmatch(line); len(matches) > 1 {
		level := matches[1]
		coloredLevel := LogLevel(level)
		coloredLine = strings.Replace(line, "."+level+":", "."+coloredLevel+":", 1)
	}

	// Print with filename prefix
	fmt.Printf("%s %s\n", LogFile(fmt.Sprintf("[%s]", filename)), coloredLine)
}

// TailLogs is a convenience function to tail logs
func TailLogs(logDir, pattern string, follow bool, lines int) error {
	tailer := NewLogTailer(logDir, pattern, follow, lines)

	// Handle interrupt
	go func() {
		// This would need signal handling in main
	}()

	return tailer.Start()
}

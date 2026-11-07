package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// DailyWriter is a thread-safe io.WriteCloser that writes log messages
// to daily-partitioned files (e.g. desknotify-2026-11-07.log) and automatically
// cleans up old logs exceeding maxDays retention.
type DailyWriter struct {
	mu          sync.Mutex
	dir         string
	prefix      string
	currentDate string
	file        *os.File
	stdout      io.Writer
	maxDays     int
}

// NewDailyWriter creates a new DailyWriter writing into dir with the specified prefix.
// If stdout is non-nil, all writes will be teed/mirrored to stdout as well.
func NewDailyWriter(dir, prefix string, maxDays int, stdout io.Writer) (*DailyWriter, error) {
	if dir == "" {
		return nil, fmt.Errorf("log directory cannot be empty")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory %q: %w", dir, err)
	}
	if maxDays <= 0 {
		maxDays = 30
	}
	if prefix == "" {
		prefix = "desknotify"
	}

	dw := &DailyWriter{
		dir:     dir,
		prefix:  prefix,
		maxDays: maxDays,
		stdout:  stdout,
	}

	if err := dw.rotate(time.Now()); err != nil {
		return nil, err
	}

	return dw, nil
}

// rotate checks if the date has changed and opens a new log file if needed.
// Must be called with dw.mu held or before concurrent access.
func (dw *DailyWriter) rotate(now time.Time) error {
	dateStr := now.Format("2006-01-02")
	if dw.file != nil && dw.currentDate == dateStr {
		return nil
	}

	if dw.file != nil {
		_ = dw.file.Close()
		dw.file = nil
	}

	if err := os.MkdirAll(dw.dir, 0755); err != nil {
		return fmt.Errorf("failed to ensure log directory %q: %w", dw.dir, err)
	}

	fileName := fmt.Sprintf("%s-%s.log", dw.prefix, dateStr)
	filePath := filepath.Join(dw.dir, fileName)

	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file %q: %w", filePath, err)
	}

	dw.file = f
	dw.currentDate = dateStr

	// Clean up old log files asynchronously
	go dw.cleanOldLogs(now)

	return nil
}

// Write writes the given bytes to the current day's log file and optional stdout.
func (dw *DailyWriter) Write(p []byte) (n int, err error) {
	dw.mu.Lock()
	defer dw.mu.Unlock()

	now := time.Now()
	if err := dw.rotate(now); err != nil {
		if dw.stdout != nil {
			return dw.stdout.Write(p)
		}
		return 0, err
	}

	n, err = dw.file.Write(p)
	if dw.stdout != nil {
		_, _ = dw.stdout.Write(p)
	}
	return n, err
}

// Close closes the currently opened log file.
func (dw *DailyWriter) Close() error {
	dw.mu.Lock()
	defer dw.mu.Unlock()

	if dw.file != nil {
		err := dw.file.Close()
		dw.file = nil
		return err
	}
	return nil
}

// CurrentFilePath returns the path of the current day's log file.
func (dw *DailyWriter) CurrentFilePath() string {
	dw.mu.Lock()
	defer dw.mu.Unlock()
	if dw.currentDate == "" {
		return ""
	}
	return filepath.Join(dw.dir, fmt.Sprintf("%s-%s.log", dw.prefix, dw.currentDate))
}

// cleanOldLogs scans the log directory and removes files older than maxDays.
func (dw *DailyWriter) cleanOldLogs(now time.Time) {
	entries, err := os.ReadDir(dw.dir)
	if err != nil {
		return
	}

	prefix := dw.prefix + "-"
	suffix := ".log"
	cutoff := now.AddDate(0, 0, -dw.maxDays)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, suffix) {
			continue
		}

		datePart := strings.TrimSuffix(strings.TrimPrefix(name, prefix), suffix)
		fileDate, err := time.Parse("2006-01-02", datePart)
		if err != nil {
			continue
		}

		if fileDate.Before(cutoff) {
			_ = os.Remove(filepath.Join(dw.dir, name))
		}
	}
}

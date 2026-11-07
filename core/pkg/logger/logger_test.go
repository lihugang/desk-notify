package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestDailyWriter_Basic(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "desk_notify_logger_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dw, err := NewDailyWriter(tempDir, "desknotify", 30, nil)
	if err != nil {
		t.Fatalf("failed to create daily writer: %v", err)
	}
	defer dw.Close()

	testMsg := "Hello DeskNotify Logging Test\n"
	n, err := dw.Write([]byte(testMsg))
	if err != nil {
		t.Fatalf("failed to write to daily writer: %v", err)
	}
	if n != len(testMsg) {
		t.Fatalf("expected to write %d bytes, wrote %d", len(testMsg), n)
	}

	today := time.Now().Format("2006-01-02")
	expectedFile := filepath.Join(tempDir, fmt.Sprintf("desknotify-%s.log", today))

	content, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("failed to read expected log file %s: %v", expectedFile, err)
	}
	if string(content) != testMsg {
		t.Fatalf("expected content %q, got %q", testMsg, string(content))
	}
}

func TestDailyWriter_CleanOldLogs(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "desk_notify_logger_clean_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create an old log file (40 days ago) and a recent log file (5 days ago)
	now := time.Now()
	oldDate := now.AddDate(0, 0, -40).Format("2006-01-02")
	recentDate := now.AddDate(0, 0, -5).Format("2006-01-02")

	oldFilePath := filepath.Join(tempDir, fmt.Sprintf("desknotify-%s.log", oldDate))
	recentFilePath := filepath.Join(tempDir, fmt.Sprintf("desknotify-%s.log", recentDate))

	_ = os.WriteFile(oldFilePath, []byte("old log content"), 0644)
	_ = os.WriteFile(recentFilePath, []byte("recent log content"), 0644)

	dw, err := NewDailyWriter(tempDir, "desknotify", 30, nil)
	if err != nil {
		t.Fatalf("failed to create daily writer: %v", err)
	}
	defer dw.Close()

	// Trigger cleanOldLogs
	dw.cleanOldLogs(now)

	// Old file should be deleted
	if _, err := os.Stat(oldFilePath); !os.IsNotExist(err) {
		t.Errorf("expected old log file %s to be deleted, but it still exists", oldFilePath)
	}

	// Recent file should be preserved
	if _, err := os.Stat(recentFilePath); err != nil {
		t.Errorf("expected recent log file %s to exist, but got err: %v", recentFilePath, err)
	}
}

func TestLogger_InitAndLevels(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "desk_notify_logger_levels_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	if err := Init(tempDir); err != nil {
		t.Fatalf("failed to init logger: %v", err)
	}
	defer Close()

	Infof("This is info: %s", "test1")
	Warnf("This is warn: %d", 123)
	Errorf("This is error: %v", "failure")
	Println("This is println")

	logFile := GetCurrentLogFile()
	if logFile == "" {
		t.Fatalf("expected non-empty log file path")
	}

	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	text := string(data)

	if !strings.Contains(text, "[INFO] This is info: test1") {
		t.Errorf("missing expected INFO message in log:\n%s", text)
	}
	if !strings.Contains(text, "[WARN] This is warn: 123") {
		t.Errorf("missing expected WARN message in log:\n%s", text)
	}
	if !strings.Contains(text, "[ERROR] This is error: failure") {
		t.Errorf("missing expected ERROR message in log:\n%s", text)
	}
	if !strings.Contains(text, "[INFO] This is println") {
		t.Errorf("missing expected Println message in log:\n%s", text)
	}
}

func TestLogger_Concurrency(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "desk_notify_logger_concurrency_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	if err := Init(tempDir); err != nil {
		t.Fatalf("failed to init logger: %v", err)
	}
	defer Close()

	var wg sync.WaitGroup
	concurrency := 10
	iterations := 50

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				Infof("worker %d iteration %d", id, j)
			}
		}(i)
	}

	wg.Wait()

	data, err := os.ReadFile(GetCurrentLogFile())
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	expectedCount := concurrency * iterations
	if len(lines) != expectedCount {
		t.Fatalf("expected %d log lines, got %d", expectedCount, len(lines))
	}
}

package config

import (
	"os"
	"testing"
)

func TestFirstRunDetectionAndCompletion(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "desk_notify_firstrun_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 1. First initialization: config.json does not exist -> IsFirstRun must be true
	mgr, err := NewManagerWithDir(tempDir)
	if err != nil {
		t.Fatalf("failed to create config manager: %v", err)
	}

	if !mgr.IsFirstRun() {
		t.Fatalf("expected IsFirstRun() to be true on brand new installation")
	}
	if mgr.Get().FirstRunDone {
		t.Fatalf("expected FirstRunDone in config to be false on initial creation")
	}

	// 2. Mark first run complete
	if err := mgr.MarkFirstRunComplete(); err != nil {
		t.Fatalf("failed to mark first run complete: %v", err)
	}

	if mgr.IsFirstRun() {
		t.Fatalf("expected IsFirstRun() to be false after MarkFirstRunComplete()")
	}
	if !mgr.Get().FirstRunDone {
		t.Fatalf("expected FirstRunDone in config to be true after MarkFirstRunComplete()")
	}

	// 3. Second launch (reload from existing config.json) -> IsFirstRun must be false
	mgr2, err := NewManagerWithDir(tempDir)
	if err != nil {
		t.Fatalf("failed to reload config manager: %v", err)
	}

	if mgr2.IsFirstRun() {
		t.Fatalf("expected IsFirstRun() to be false on subsequent app launch")
	}
	if !mgr2.Get().FirstRunDone {
		t.Fatalf("expected FirstRunDone to persist as true across launches")
	}
}

func TestSanitizeClassNameAndDocumentsDir(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"", "班级通知"},
		{"   ", "班级通知"},
		{"2907", "2907"},
		{"高三(1)班", "高三(1)班"},
		{"高三/1班", "高三_1班"},
		{"初二:3*班?|", "初二_3_班__"},
	}

	for _, c := range cases {
		got := SanitizeClassName(c.input)
		if got != c.expected {
			t.Errorf("SanitizeClassName(%q) = %q; want %q", c.input, got, c.expected)
		}
	}

	dir := GetDocumentsDir("2907")
	if dir == "" {
		t.Fatalf("expected non-empty documents dir")
	}
}

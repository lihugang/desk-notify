package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

func TestConfigAESEncryption(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "desk_notify_aes_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr, err := NewManagerWithDir(tempDir)
	if err != nil {
		t.Fatalf("failed to create config manager: %v", err)
	}

	cfg := mgr.Get()
	cfg.ClassName = "TestClassAES"
	cfg.Port = 9999
	cfg.PasswordHash = "custom_hash_123"

	if err := mgr.Save(cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	configFilePath := filepath.Join(tempDir, "config.json")
	fileBytes, err := os.ReadFile(configFilePath)
	if err != nil {
		t.Fatalf("failed to read config file from disk: %v", err)
	}

	// 1. Verify file content on disk is encrypted (NOT valid JSON or plain text)
	var rawJSON map[string]interface{}
	if err := json.Unmarshal(fileBytes, &rawJSON); err == nil {
		t.Fatalf("disk config file should be encrypted, but successfully unmarshaled as plain JSON: %s", string(fileBytes))
	}
	if strings.Contains(string(fileBytes), "TestClassAES") {
		t.Fatalf("disk config file should not contain plain text 'TestClassAES'")
	}

	// 2. Load into a new manager and verify decrypted data matches
	mgr2, err := NewManagerWithDir(tempDir)
	if err != nil {
		t.Fatalf("failed to load manager from encrypted file: %v", err)
	}

	loadedCfg := mgr2.Get()
	if loadedCfg.ClassName != "TestClassAES" {
		t.Errorf("expected ClassName 'TestClassAES', got %q", loadedCfg.ClassName)
	}
	if loadedCfg.Port != 9999 {
		t.Errorf("expected Port 9999, got %d", loadedCfg.Port)
	}
	if loadedCfg.PasswordHash != "custom_hash_123" {
		t.Errorf("expected PasswordHash 'custom_hash_123', got %q", loadedCfg.PasswordHash)
	}

	// 3. Verify corrupted ciphertext returns error
	corrupted := append([]byte(nil), fileBytes...)
	corrupted[len(corrupted)-1] ^= 0xFF
	if err := os.WriteFile(configFilePath, corrupted, 0644); err != nil {
		t.Fatalf("failed to write corrupted file: %v", err)
	}
	if err := mgr.Load(); err == nil {
		t.Fatalf("expected error when loading corrupted ciphertext, got nil")
	}

	// 4. Verify plain JSON file cannot be loaded directly (no legacy format compatibility)
	plainJSON := []byte(`{"className": "plain", "port": 8888}`)
	if err := os.WriteFile(configFilePath, plainJSON, 0644); err != nil {
		t.Fatalf("failed to write plain json: %v", err)
	}
	if err := mgr.Load(); err == nil {
		t.Fatalf("expected error when loading unencrypted plain JSON file, got nil")
	}
}

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lihugang/desk-notify/core/pkg/config"
	"github.com/lihugang/desk-notify/core/pkg/db"
	"github.com/lihugang/desk-notify/core/pkg/desktop"
	"github.com/lihugang/desk-notify/core/pkg/storage"
	"github.com/lihugang/desk-notify/core/pkg/tunnel"
)

type mockDesktopManager struct {
	openedFile   string
	revealedFile string
}

var _ desktop.Manager = (*mockDesktopManager)(nil)

func (m *mockDesktopManager) Run(cfgMgr *config.Manager) error { return nil }
func (m *mockDesktopManager) Notify(n *db.Notification)        {}
func (m *mockDesktopManager) ShowCard()                        {}
func (m *mockDesktopManager) HideCard()                        {}
func (m *mockDesktopManager) MinimizeCard()                    {}
func (m *mockDesktopManager) StopFlash()                       {}
func (m *mockDesktopManager) SetCardHeight(height int)          {}
func (m *mockDesktopManager) OpenSettings()                    {}
func (m *mockDesktopManager) HideSettings()                    {}
func (m *mockDesktopManager) OpenHistory()                     {}
func (m *mockDesktopManager) HideHistory()                     {}
func (m *mockDesktopManager) OpenArchiveDir() error            { return nil }
func (m *mockDesktopManager) OpenLogsDir() error               { return nil }
func (m *mockDesktopManager) OpenFile(filePath string) error {
	m.openedFile = filePath
	return nil
}
func (m *mockDesktopManager) RevealFile(filePath string) error {
	m.revealedFile = filePath
	return nil
}
func (m *mockDesktopManager) PinToTaskbar() error { return nil }
func (m *mockDesktopManager) CreateDesktopShortcut() error { return nil }
func (m *mockDesktopManager) SetAutoStart(enabled bool) error { return nil }
func (m *mockDesktopManager) IsAutoStartEnabled() bool { return false }
func (m *mockDesktopManager) Quit() {}

func TestFileOpenAndReveal(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "desk_notify_files_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfgMgr, err := config.NewManagerWithDir(tempDir)
	if err != nil {
		t.Fatalf("failed to init config manager: %v", err)
	}
	store, err := db.NewStore(tempDir)
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer store.Close()

	fileStorage := storage.NewStorage(tempDir)
	tun := tunnel.NewMockTunnel()
	mockDesk := &mockDesktopManager{}
	srv := NewServer(cfgMgr, store, fileStorage, tun, mockDesk, nil)

	// Create a dummy attachment file
	dummyFile := filepath.Join(tempDir, "test_doc.pptx")
	_ = os.WriteFile(dummyFile, []byte("dummy pptx content"), 0644)

	notif := &db.Notification{
		ID:        "notif_test_1",
		Content:   "这是课件通知",
		Pinned:    true,
		Duration:  0,
		Sender:    "老师",
		CreatedAt: time.Now(),
	}
	att := &db.Attachment{
		ID:        "att_pptx_1",
		NotifyID:  notif.ID,
		Name:      "数学第1课.pptx",
		Size:      100,
		MimeType:  "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		LocalPath: dummyFile,
		DocPath:   dummyFile,
		IsImage:   false,
	}
	if err := store.SaveNotification(notif, []*db.Attachment{att}); err != nil {
		t.Fatalf("failed to save notification: %v", err)
	}

	// 1. Test POST /api/files/att_pptx_1/open
	reqOpen := httptest.NewRequest(http.MethodPost, "/api/files/att_pptx_1/open", nil)
	reqOpen.RemoteAddr = "127.0.0.1:12345"
	wOpen := httptest.NewRecorder()
	srv.handleFiles(wOpen, reqOpen)

	if wOpen.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from file open, got %d: %s", wOpen.Code, wOpen.Body.String())
	}
	if mockDesk.openedFile != dummyFile {
		t.Fatalf("expected opened file %s, got %s", dummyFile, mockDesk.openedFile)
	}

	var openResp map[string]any
	if err := json.Unmarshal(wOpen.Body.Bytes(), &openResp); err != nil || openResp["ok"] != true {
		t.Fatalf("invalid json response from open: %v", err)
	}

	// 2. Test POST /api/files/att_pptx_1/reveal
	reqReveal := httptest.NewRequest(http.MethodPost, "/api/files/att_pptx_1/reveal", nil)
	reqReveal.RemoteAddr = "127.0.0.1:12345"
	wReveal := httptest.NewRecorder()
	srv.handleFiles(wReveal, reqReveal)

	if wReveal.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from file reveal, got %d: %s", wReveal.Code, wReveal.Body.String())
	}
	if mockDesk.revealedFile != dummyFile {
		t.Fatalf("expected revealed file %s, got %s", dummyFile, mockDesk.revealedFile)
	}

	// 3. Test POST /api/files/open with JSON body
	reqOpenJSON := httptest.NewRequest(http.MethodPost, "/api/files/open", strings.NewReader(`{"id":"att_pptx_1"}`))
	reqOpenJSON.RemoteAddr = "127.0.0.1:12345"
	reqOpenJSON.Header.Set("Content-Type", "application/json")
	wOpenJSON := httptest.NewRecorder()
	srv.handleFileOpen(wOpenJSON, reqOpenJSON)

	if wOpenJSON.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from /api/files/open with json, got %d: %s", wOpenJSON.Code, wOpenJSON.Body.String())
	}

	// 4. Test POST /api/files/reveal with query param
	reqRevealQuery := httptest.NewRequest(http.MethodPost, "/api/files/reveal?id=att_pptx_1", nil)
	reqRevealQuery.RemoteAddr = "127.0.0.1:12345"
	wRevealQuery := httptest.NewRecorder()
	srv.handleFileReveal(wRevealQuery, reqRevealQuery)

	if wRevealQuery.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from /api/files/reveal with query, got %d: %s", wRevealQuery.Code, wRevealQuery.Body.String())
	}

	// 5. Test nonexistent file ID
	reqNotFound := httptest.NewRequest(http.MethodPost, "/api/files/nonexistent/open", nil)
	reqNotFound.RemoteAddr = "127.0.0.1:12345"
	wNotFound := httptest.NewRecorder()
	srv.handleFiles(wNotFound, reqNotFound)

	if wNotFound.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", wNotFound.Code)
	}
}

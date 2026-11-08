package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lihugang/desk-notify/core/pkg/auth"
	"github.com/lihugang/desk-notify/core/pkg/config"
	"github.com/lihugang/desk-notify/core/pkg/db"
	"github.com/lihugang/desk-notify/core/pkg/desktop"
	"github.com/lihugang/desk-notify/core/pkg/storage"
	"github.com/lihugang/desk-notify/core/pkg/tunnel"
)

func TestHistoryPaginationAndDownload(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "desk_notify_history_test_*")
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
	desk := desktop.NewManager()
	srv := NewServer(cfgMgr, store, fileStorage, tun, desk, nil)

	// Create a dummy attachment file
	dummyFile := filepath.Join(tempDir, "test_doc.docx")
	_ = os.WriteFile(dummyFile, []byte("dummy file content"), 0644)

	// Insert 5 notifications
	for i := 1; i <= 5; i++ {
		notif := &db.Notification{
			ID:        fmt.Sprintf("notif_%d", i),
			Content:   fmt.Sprintf("## 历史通知 %d\n这是第 %d 条完整的测试通知内容", i, i),
			Pinned:    i%2 == 0,
			Duration:  30,
			Sender:    "张老师",
			CreatedAt: time.Now().Add(time.Duration(i) * time.Minute),
		}
		att := &db.Attachment{
			ID:        fmt.Sprintf("att_%d", i),
			NotifyID:  notif.ID,
			Name:      fmt.Sprintf("复习课件_%d.docx", i),
			Size:      18,
			MimeType:  "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			LocalPath: dummyFile,
			DocPath:   dummyFile,
			IsImage:   false,
		}
		if err := store.SaveNotification(notif, []*db.Attachment{att}); err != nil {
			t.Fatalf("failed to save notification: %v", err)
		}
	}

	// 1. Test Page 1: limit=2, offset=0
	req1 := httptest.NewRequest("GET", "/api/history?limit=2&offset=0", nil)
	req1.RemoteAddr = "192.168.1.50:12345" // LAN
	w1 := httptest.NewRecorder()
	srv.handleHistory(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w1.Code)
	}

	var resp1 struct {
		Items   []*db.Notification `json:"items"`
		Total   int                `json:"total"`
		Limit   int                `json:"limit"`
		Offset  int                `json:"offset"`
		HasMore bool               `json:"hasMore"`
	}
	if err := json.Unmarshal(w1.Body.Bytes(), &resp1); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp1.Total != 5 {
		t.Fatalf("expected total 5, got %d", resp1.Total)
	}
	if len(resp1.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp1.Items))
	}
	if !resp1.HasMore {
		t.Fatalf("expected hasMore to be true")
	}
	if resp1.Limit != 2 || resp1.Offset != 0 {
		t.Fatalf("expected limit=2, offset=0, got limit=%d, offset=%d", resp1.Limit, resp1.Offset)
	}
	// Verify attachments populated
	if len(resp1.Items[0].Attachments) != 1 {
		t.Fatalf("expected attachment to be populated in item, got %d", len(resp1.Items[0].Attachments))
	}

	// 2. Test Page 3: limit=2, offset=4 (last item)
	req2 := httptest.NewRequest("GET", "/api/history?limit=2&offset=4", nil)
	req2.RemoteAddr = "192.168.1.50:12345"
	w2 := httptest.NewRecorder()
	srv.handleHistory(w2, req2)

	var resp2 struct {
		Items   []*db.Notification `json:"items"`
		Total   int                `json:"total"`
		HasMore bool               `json:"hasMore"`
	}
	_ = json.Unmarshal(w2.Body.Bytes(), &resp2)
	if len(resp2.Items) != 1 {
		t.Fatalf("expected 1 item on last page, got %d", len(resp2.Items))
	}
	if resp2.HasMore {
		t.Fatalf("expected hasMore to be false on last page")
	}

	// 3. Test File Download with ?download=1
	req3 := httptest.NewRequest("GET", "/api/files/att_1?download=1", nil)
	w3 := httptest.NewRecorder()
	srv.handleFiles(w3, req3)

	if w3.Code != http.StatusOK {
		t.Fatalf("expected status 200 for download, got %d", w3.Code)
	}
	cd := w3.Header().Get("Content-Disposition")
	if !strings.Contains(cd, "attachment") {
		t.Fatalf("expected Content-Disposition to contain attachment, got %s", cd)
	}
	if !strings.Contains(cd, "filename=") {
		t.Fatalf("expected Content-Disposition to contain filename, got %s", cd)
	}
}

func TestHistoryWithLanAuthRequired(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "desk_notify_history_auth_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfgMgr, err := config.NewManagerWithDir(tempDir)
	if err != nil {
		t.Fatalf("failed to init config manager: %v", err)
	}
	cfg := cfgMgr.Get()
	cfg.LanAuthRequired = true
	_ = cfgMgr.Save(cfg)

	store, err := db.NewStore(tempDir)
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer store.Close()

	fileStorage := storage.NewStorage(tempDir)
	tun := tunnel.NewMockTunnel()
	desk := desktop.NewManager()
	srv := NewServer(cfgMgr, store, fileStorage, tun, desk, nil)

	// 1. Loopback request (Desktop app) when LanAuthRequired=true should succeed with 200 OK
	reqLoopback := httptest.NewRequest("GET", "/api/history?limit=10&offset=0", nil)
	reqLoopback.RemoteAddr = "127.0.0.1:54321"
	wLoopback := httptest.NewRecorder()
	srv.handleHistory(wLoopback, reqLoopback)
	if wLoopback.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for loopback history request when LanAuthRequired=true, got %d", wLoopback.Code)
	}

	// 2. LAN request (e.g. mobile app on campus Wi-Fi) when LanAuthRequired=true should succeed with 200 OK
	reqLAN := httptest.NewRequest("GET", "/api/history?limit=10&offset=0", nil)
	reqLAN.RemoteAddr = "192.168.1.100:54321"
	wLAN := httptest.NewRecorder()
	srv.handleHistory(wLAN, reqLAN)
	if wLAN.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for LAN history request when LanAuthRequired=true, got %d", wLAN.Code)
	}

	// 3. Latest notify endpoint when LanAuthRequired=true should also succeed with 200 OK
	reqLatestLAN := httptest.NewRequest("GET", "/api/notify/latest", nil)
	reqLatestLAN.RemoteAddr = "192.168.1.100:54321"
	wLatestLAN := httptest.NewRecorder()
	srv.handleLatestNotify(wLatestLAN, reqLatestLAN)
	if wLatestLAN.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for latest notify request when LanAuthRequired=true, got %d", wLatestLAN.Code)
	}

	// 4. Public request without token when password configured should return 401
	reqPublic := httptest.NewRequest("GET", "/api/history?limit=10&offset=0", nil)
	reqPublic.RemoteAddr = "127.0.0.1:54321"
	reqPublic.Header.Set("X-Forwarded-For", "203.0.113.195") // public frp
	wPublic := httptest.NewRecorder()
	srv.handleHistory(wPublic, reqPublic)
	if wPublic.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for unauthenticated public history request, got %d", wPublic.Code)
	}

	// 5. Public request with valid token should return 200 OK
	token, _ := auth.GenerateToken("123456")
	reqPublicAuth := httptest.NewRequest("GET", "/api/history?limit=10&offset=0", nil)
	reqPublicAuth.RemoteAddr = "127.0.0.1:54321"
	reqPublicAuth.Header.Set("X-Forwarded-For", "203.0.113.195")
	reqPublicAuth.Header.Set("Authorization", "Bearer "+token)
	wPublicAuth := httptest.NewRecorder()
	srv.handleHistory(wPublicAuth, reqPublicAuth)
	if wPublicAuth.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for authenticated public history request, got %d", wPublicAuth.Code)
	}
}

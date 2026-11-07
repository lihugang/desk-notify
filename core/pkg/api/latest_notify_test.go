package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/lihugang/desk-notify/core/pkg/config"
	"github.com/lihugang/desk-notify/core/pkg/db"
	"github.com/lihugang/desk-notify/core/pkg/desktop"
	"github.com/lihugang/desk-notify/core/pkg/storage"
	"github.com/lihugang/desk-notify/core/pkg/tunnel"
)

func TestLatestNotifyEndpoint(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "desk_notify_latest_test_*")
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

	// 1. Test when database has NO notifications
	req1 := httptest.NewRequest(http.MethodGet, "/api/notify/latest", nil)
	rec1 := httptest.NewRecorder()
	srv.handleLatestNotify(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec1.Code)
	}

	var res1 struct {
		HasNotification bool   `json:"hasNotification"`
		ClassName       string `json:"className"`
	}
	if err := json.Unmarshal(rec1.Body.Bytes(), &res1); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}
	if res1.HasNotification {
		t.Fatalf("expected hasNotification=false when db is empty")
	}

	// 2. Insert a notification
	n := &db.Notification{
		ID:        "latest_test_1",
		Content:   "这是一条最新测试通知",
		Pinned:    true,
		Duration:  0,
		Sender:    "127.0.0.1",
		CreatedAt: time.Now(),
	}
	if err := store.SaveNotification(n, nil); err != nil {
		t.Fatalf("failed to save notification: %v", err)
	}

	// 3. Test when database has notification
	req2 := httptest.NewRequest(http.MethodGet, "/api/notify/latest", nil)
	rec2 := httptest.NewRecorder()
	srv.handleLatestNotify(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec2.Code)
	}

	var res2 struct {
		HasNotification bool               `json:"hasNotification"`
		ClassName       string             `json:"className"`
		Notification    *db.Notification   `json:"notification"`
		Notifications   []*db.Notification `json:"notifications"`
		SoundEnabled    bool               `json:"soundEnabled"`
	}
	if err := json.Unmarshal(rec2.Body.Bytes(), &res2); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}
	if !res2.HasNotification {
		t.Fatalf("expected hasNotification=true after inserting notification")
	}
	if res2.Notification == nil || res2.Notification.ID != "latest_test_1" {
		t.Fatalf("expected notification id 'latest_test_1', got %+v", res2.Notification)
	}
	if len(res2.Notifications) != 1 || res2.Notifications[0].ID != "latest_test_1" {
		t.Fatalf("expected 1 notification in notifications list, got %d", len(res2.Notifications))
	}
	if !res2.Notification.Pinned {
		t.Fatalf("expected pinned=true, got %v", res2.Notification.Pinned)
	}

	// 3.5 Insert an expired notification (100 mins ago) and a fresh notification (30 mins ago)
	oldN := &db.Notification{
		ID:        "latest_test_old",
		Content:   "这是100分钟前的过期通知",
		Pinned:    false,
		Duration:  0,
		Sender:    "127.0.0.1",
		CreatedAt: time.Now().Add(-100 * time.Minute),
	}
	_ = store.SaveNotification(oldN, nil)

	freshN := &db.Notification{
		ID:        "latest_test_30m",
		Content:   "这是30分钟前的一般通知",
		Pinned:    false,
		Duration:  0,
		Sender:    "127.0.0.1",
		CreatedAt: time.Now().Add(-30 * time.Minute),
	}
	_ = store.SaveNotification(freshN, nil)

	req3 := httptest.NewRequest(http.MethodGet, "/api/notify/latest", nil)
	rec3 := httptest.NewRecorder()
	srv.handleLatestNotify(rec3, req3)
	var res3 struct {
		HasNotification bool               `json:"hasNotification"`
		Notifications   []*db.Notification `json:"notifications"`
	}
	_ = json.Unmarshal(rec3.Body.Bytes(), &res3)
	if !res3.HasNotification {
		t.Fatalf("expected hasNotification=true")
	}
	if len(res3.Notifications) != 2 {
		t.Fatalf("expected exactly 2 notifications within 90 mins (excluding the 100m old one), got %d", len(res3.Notifications))
	}
	if res3.Notifications[0].ID != "latest_test_1" || res3.Notifications[1].ID != "latest_test_30m" {
		t.Fatalf("unexpected notifications list order or ids: %+v", res3.Notifications)
	}

	// 4. Test window minimize and stop-flash endpoints
	reqMin := httptest.NewRequest(http.MethodPost, "/api/window/minimize", nil)
	recMin := httptest.NewRecorder()
	srv.handleWindowMinimize(recMin, reqMin)
	if recMin.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for minimize, got %d", recMin.Code)
	}

	reqFlash := httptest.NewRequest(http.MethodPost, "/api/window/stop-flash", nil)
	recFlash := httptest.NewRecorder()
	srv.handleWindowStopFlash(recFlash, reqFlash)
	if recFlash.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for stop-flash, got %d", recFlash.Code)
	}
}

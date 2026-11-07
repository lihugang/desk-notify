package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"testing/fstest"

	"github.com/lihugang/desk-notify/core/pkg/auth"
	"github.com/lihugang/desk-notify/core/pkg/config"
	"github.com/lihugang/desk-notify/core/pkg/db"
	"github.com/lihugang/desk-notify/core/pkg/desktop"
	"github.com/lihugang/desk-notify/core/pkg/storage"
	"github.com/lihugang/desk-notify/core/pkg/tunnel"
)

func TestFilterLanIPs(t *testing.T) {
	// Case 1: Contains 192.x address -> only 192.x should be returned, others hidden
	ipsWith192 := []string{"10.0.0.5", "192.168.1.100", "172.16.2.3", "192.168.31.20"}
	filtered1 := FilterLanIPs(ipsWith192)
	if len(filtered1) != 2 || filtered1[0] != "192.168.1.100" || filtered1[1] != "192.168.31.20" {
		t.Fatalf("expected only 192.* IPs, got %v", filtered1)
	}

	// Case 2: Does NOT contain 192.x address -> all addresses displayed
	ipsWithout192 := []string{"10.0.0.5", "172.16.2.3"}
	filtered2 := FilterLanIPs(ipsWithout192)
	if len(filtered2) != 2 || filtered2[0] != "10.0.0.5" || filtered2[1] != "172.16.2.3" {
		t.Fatalf("expected all IPs to be returned when no 192.* exists, got %v", filtered2)
	}
}

func TestAdminVerifyAndTwoTierConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "desk_notify_cfg_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfgMgr, err := config.NewManagerWithDir(tempDir)
	if err != nil {
		t.Fatalf("failed to init config manager: %v", err)
	}
	store, _ := db.NewStore(tempDir)
	defer store.Close()
	fileStorage := storage.NewStorage(tempDir)
	tun := tunnel.NewMockTunnel()
	deskMgr := desktop.NewManager()

	srv := NewServer(cfgMgr, store, fileStorage, tun, deskMgr, fstest.MapFS{})

	// 1. Verify admin password with default "123456"
	adminVerifyBody, _ := json.Marshal(map[string]string{"password": "123456"})
	req := httptest.NewRequest(http.MethodPost, "/api/config/verify-admin", bytes.NewReader(adminVerifyBody))
	req.RemoteAddr = "127.0.0.1:1234"
	w := httptest.NewRecorder()
	srv.handleAdminVerify(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from admin verify, got %d", resp.StatusCode)
	}

	var verifyRes map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&verifyRes)
	adminToken, ok := verifyRes["adminToken"].(string)
	if !ok || adminToken == "" {
		t.Fatalf("expected adminToken in verify response, got %v", verifyRes)
	}

	// 2. Try modifying Level 2 FRP/PublicPassword settings WITHOUT admin token -> should return 403 Forbidden
	frpEnable := true
	unauthBody, _ := json.Marshal(map[string]interface{}{
		"frp": map[string]interface{}{
			"enabled": frpEnable,
		},
	})
	reqUnauth := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewReader(unauthBody))
	reqUnauth.RemoteAddr = "127.0.0.1:1234"
	wUnauth := httptest.NewRecorder()
	srv.handleConfig(wUnauth, reqUnauth)
	if wUnauth.Result().StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden without admin token, got %d", wUnauth.Result().StatusCode)
	}

	// 3. Modify Level 1 setting (className) WITHOUT admin token -> should succeed
	newClassName := "2907"
	level1Body, _ := json.Marshal(map[string]interface{}{
		"className": newClassName,
	})
	reqL1 := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewReader(level1Body))
	reqL1.RemoteAddr = "127.0.0.1:1234"
	wL1 := httptest.NewRecorder()
	srv.handleConfig(wL1, reqL1)
	if wL1.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for Level 1 config update, got %d", wL1.Result().StatusCode)
	}
	if cfgMgr.Get().ClassName != "2907" {
		t.Fatalf("expected className to be 2907, got %s", cfgMgr.Get().ClassName)
	}

	// 4. Modify Level 2 setting (FRP & PublicPassword) WITH admin token -> should succeed and bcrypt hash should be stored
	newPublicPwd := "myNewTeacherSecret"
	level2Body, _ := json.Marshal(map[string]interface{}{
		"newPublicPassword": newPublicPwd,
		"frp": map[string]interface{}{
			"enabled":    true,
			"serverAddr": "frp.freefrp.net",
			"serverPort": 7000,
			"token":      "freefrp.net",
			"remotePort": 28888,
		},
	})
	reqL2 := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewReader(level2Body))
	reqL2.RemoteAddr = "127.0.0.1:1234"
	reqL2.Header.Set("Authorization", "Bearer "+adminToken)
	wL2 := httptest.NewRecorder()
	srv.handleConfig(wL2, reqL2)
	if wL2.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for Level 2 config update with admin token, got %d", wL2.Result().StatusCode)
	}

	updatedCfg := cfgMgr.Get()
	if updatedCfg.PublicPassword != "" {
		t.Fatalf("PublicPassword should NOT be stored in plain text, got %s", updatedCfg.PublicPassword)
	}
	if !auth.CheckPassword(updatedCfg.PublicPasswordHash, newPublicPwd) {
		t.Fatalf("PublicPasswordHash failed bcrypt verification against %s", newPublicPwd)
	}
	if !updatedCfg.FRP.Enabled || updatedCfg.FRP.RemotePort != 28888 {
		t.Fatalf("FRP settings not correctly updated: %+v", updatedCfg.FRP)
	}
}

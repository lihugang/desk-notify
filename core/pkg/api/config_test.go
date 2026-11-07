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

	// 0. Verify default FRP is enabled
	if !cfgMgr.Get().FRP.Enabled {
		t.Fatalf("expected FRP to be enabled by default")
	}

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

	// 2. Try modifying Level 2 FRP/Security settings WITHOUT admin token -> should return 403 Forbidden
	lanAuth := true
	unauthBody, _ := json.Marshal(map[string]interface{}{
		"lanAuthRequired": lanAuth,
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

	// 4. Modify Level 2 setting (FRP, lanAuthRequired & PublicPassword) WITH admin token -> should succeed
	newPublicPwd := "myNewTeacherSecret"
	level2Body, _ := json.Marshal(map[string]interface{}{
		"newPublicPassword": newPublicPwd,
		"lanAuthRequired":   true,
		"frp": map[string]interface{}{
			"enabled":    false,
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
	if !auth.CheckPassword(updatedCfg.GetEffectivePasswordHash(), newPublicPwd) {
		t.Fatalf("Effective password failed bcrypt verification against %s", newPublicPwd)
	}
	if !auth.CheckPassword(updatedCfg.PublicPasswordHash, newPublicPwd) {
		t.Fatalf("PublicPasswordHash failed bcrypt verification against %s", newPublicPwd)
	}
	if !auth.CheckPassword(updatedCfg.AdminPasswordHash, newPublicPwd) {
		t.Fatalf("AdminPasswordHash failed bcrypt verification against %s", newPublicPwd)
	}
	if updatedCfg.FRP.Enabled || updatedCfg.FRP.RemotePort != 28888 {
		t.Fatalf("FRP settings not correctly updated: %+v", updatedCfg.FRP)
	}
	if !updatedCfg.LanAuthRequired {
		t.Fatalf("LanAuthRequired should be true, got %v", updatedCfg.LanAuthRequired)
	}
}

func TestUnifiedPassword(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "desk_notify_unified_pwd_test_*")
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

	// 1. Initial password "123456" works for both admin verify and login
	adminBody, _ := json.Marshal(map[string]string{"password": "123456"})
	reqAdmin := httptest.NewRequest(http.MethodPost, "/api/config/verify-admin", bytes.NewReader(adminBody))
	wAdmin := httptest.NewRecorder()
	srv.handleAdminVerify(wAdmin, reqAdmin)
	if wAdmin.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from admin verify with default password, got %d", wAdmin.Result().StatusCode)
	}
	var res map[string]interface{}
	_ = json.NewDecoder(wAdmin.Result().Body).Decode(&res)
	adminToken := res["adminToken"].(string)

	loginBody, _ := json.Marshal(map[string]string{"password": "123456"})
	reqLogin := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader(loginBody))
	wLogin := httptest.NewRecorder()
	srv.handleLogin(wLogin, reqLogin)
	if wLogin.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from login with default password, got %d", wLogin.Result().StatusCode)
	}

	// 2. Change unified password to "UnifiedSecret999" via newPassword
	newPwd := "UnifiedSecret999"
	updateBody, _ := json.Marshal(map[string]interface{}{
		"newPassword": newPwd,
	})
	reqUpdate := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewReader(updateBody))
	reqUpdate.Header.Set("Authorization", "Bearer "+adminToken)
	wUpdate := httptest.NewRecorder()
	srv.handleConfig(wUpdate, reqUpdate)
	if wUpdate.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from config update, got %d", wUpdate.Result().StatusCode)
	}

	// 3. Old password "123456" must fail for BOTH admin verify and login
	reqAdminOld := httptest.NewRequest(http.MethodPost, "/api/config/verify-admin", bytes.NewReader(adminBody))
	wAdminOld := httptest.NewRecorder()
	srv.handleAdminVerify(wAdminOld, reqAdminOld)
	if wAdminOld.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for admin verify with old password, got %d", wAdminOld.Result().StatusCode)
	}

	reqLoginOld := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader(loginBody))
	wLoginOld := httptest.NewRecorder()
	srv.handleLogin(wLoginOld, reqLoginOld)
	if wLoginOld.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for login with old password, got %d", wLoginOld.Result().StatusCode)
	}

	// 4. New unified password works for BOTH admin verify and login
	newAdminBody, _ := json.Marshal(map[string]string{"password": newPwd})
	reqAdminNew := httptest.NewRequest(http.MethodPost, "/api/config/verify-admin", bytes.NewReader(newAdminBody))
	wAdminNew := httptest.NewRecorder()
	srv.handleAdminVerify(wAdminNew, reqAdminNew)
	if wAdminNew.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from admin verify with new password, got %d", wAdminNew.Result().StatusCode)
	}

	newLoginBody, _ := json.Marshal(map[string]string{"password": newPwd})
	reqLoginNew := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader(newLoginBody))
	wLoginNew := httptest.NewRecorder()
	srv.handleLogin(wLoginNew, reqLoginNew)
	if wLoginNew.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from login with new password, got %d", wLoginNew.Result().StatusCode)
	}
}

func TestFirstRunAPI(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "desk_notify_api_firstrun_*")
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

	// 1. Initial GET /api/config should report isFirstRun = true
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	w := httptest.NewRecorder()
	srv.handleConfig(w, req)

	var getRes map[string]interface{}
	if err := json.NewDecoder(w.Result().Body).Decode(&getRes); err != nil {
		t.Fatalf("failed to decode GET /api/config: %v", err)
	}

	if isFirstRun, ok := getRes["isFirstRun"].(bool); !ok || !isFirstRun {
		t.Fatalf("expected isFirstRun == true on initial GET /api/config, got %v", getRes["isFirstRun"])
	}

	// 2. Hide settings window -> should mark first run complete
	reqHide := httptest.NewRequest(http.MethodGet, "/api/window/settings-hide", nil)
	wHide := httptest.NewRecorder()
	srv.handleSettingsHide(wHide, reqHide)

	if wHide.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from handleSettingsHide, got %d", wHide.Result().StatusCode)
	}

	if cfgMgr.IsFirstRun() {
		t.Fatalf("expected cfgMgr.IsFirstRun() == false after closing settings window")
	}

	// 3. Second GET /api/config should report isFirstRun = false
	w2 := httptest.NewRecorder()
	srv.handleConfig(w2, req)
	var getRes2 map[string]interface{}
	_ = json.NewDecoder(w2.Result().Body).Decode(&getRes2)

	if isFirstRun, ok := getRes2["isFirstRun"].(bool); !ok || isFirstRun {
		t.Fatalf("expected isFirstRun == false after completing first run, got %v", getRes2["isFirstRun"])
	}
}

func TestOpenArchiveDirAPI(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "desk_notify_archive_test_*")
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

	// 1. GET /api/config should contain documentsDir
	reqConfig := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	reqConfig.RemoteAddr = "127.0.0.1:1234"
	wConfig := httptest.NewRecorder()
	srv.handleConfig(wConfig, reqConfig)

	var configRes map[string]interface{}
	if err := json.NewDecoder(wConfig.Result().Body).Decode(&configRes); err != nil {
		t.Fatalf("failed to decode config response: %v", err)
	}
	cfgMap := configRes["config"].(map[string]interface{})
	if docDir, ok := cfgMap["documentsDir"].(string); !ok || docDir == "" {
		t.Fatalf("expected non-empty documentsDir in config, got %v", cfgMap["documentsDir"])
	}

	// 2. POST /api/archive/open should succeed and return ok: true and documentsDir
	reqOpen := httptest.NewRequest(http.MethodPost, "/api/archive/open", nil)
	reqOpen.RemoteAddr = "127.0.0.1:1234"
	wOpen := httptest.NewRecorder()
	srv.handleOpenArchiveDir(wOpen, reqOpen)

	if wOpen.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK from handleOpenArchiveDir, got %d", wOpen.Result().StatusCode)
	}
	var openRes map[string]interface{}
	if err := json.NewDecoder(wOpen.Result().Body).Decode(&openRes); err != nil {
		t.Fatalf("failed to decode open response: %v", err)
	}
	if ok, _ := openRes["ok"].(bool); !ok {
		t.Fatalf("expected ok: true in open response, got %v", openRes)
	}
	if docDir, ok := openRes["documentsDir"].(string); !ok || docDir == "" {
		t.Fatalf("expected non-empty documentsDir in open response, got %v", openRes["documentsDir"])
	}
}

func TestOpenLogsDirAPI(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "desk_notify_logs_api_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfgMgr, _ := config.NewManagerWithDir(tempDir)
	store, _ := db.NewStore(tempDir)
	defer store.Close()
	fileStorage := storage.NewStorage(tempDir)
	tun := tunnel.NewMockTunnel()
	mockDesktop := &mockDesktopManager{}
	srv := NewServer(cfgMgr, store, fileStorage, tun, mockDesktop, nil)

	// 1. GET /api/config should return logsDir in config
	reqConfig := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	reqConfig.RemoteAddr = "127.0.0.1:1234"
	wConfig := httptest.NewRecorder()
	srv.handleConfig(wConfig, reqConfig)

	var configRes map[string]interface{}
	if err := json.NewDecoder(wConfig.Result().Body).Decode(&configRes); err != nil {
		t.Fatalf("failed to decode config response: %v", err)
	}
	cfgMap := configRes["config"].(map[string]interface{})
	if logsDir, ok := cfgMap["logsDir"].(string); !ok || logsDir == "" {
		t.Fatalf("expected non-empty logsDir in config, got %v", cfgMap["logsDir"])
	}

	// 2. POST /api/logs/open should succeed and return ok: true and logsDir
	reqOpen := httptest.NewRequest(http.MethodPost, "/api/logs/open", nil)
	reqOpen.RemoteAddr = "127.0.0.1:1234"
	wOpen := httptest.NewRecorder()
	srv.handleOpenLogsDir(wOpen, reqOpen)

	if wOpen.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK from handleOpenLogsDir, got %d", wOpen.Result().StatusCode)
	}
	var openRes map[string]interface{}
	if err := json.NewDecoder(wOpen.Result().Body).Decode(&openRes); err != nil {
		t.Fatalf("failed to decode open response: %v", err)
	}
	if ok, _ := openRes["ok"].(bool); !ok {
		t.Fatalf("expected ok: true in open response, got %v", openRes)
	}
	if logsDir, ok := openRes["logsDir"].(string); !ok || logsDir == "" {
		t.Fatalf("expected non-empty logsDir in open response, got %v", openRes["logsDir"])
	}
}

func TestSettingsHideAndLockAdmin(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "desk_notify_lock_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfgMgr, _ := config.NewManagerWithDir(tempDir)
	store, _ := db.NewStore(tempDir)
	defer store.Close()
	fileStorage := storage.NewStorage(tempDir)
	tun := tunnel.NewMockTunnel()
	deskMgr := desktop.NewManager()

	srv := NewServer(cfgMgr, store, fileStorage, tun, deskMgr, fstest.MapFS{})

	// 1. Verify admin password -> should set cookie
	adminVerifyBody, _ := json.Marshal(map[string]string{"password": "123456"})
	req := httptest.NewRequest(http.MethodPost, "/api/config/verify-admin", bytes.NewReader(adminVerifyBody))
	w := httptest.NewRecorder()
	srv.handleAdminVerify(w, req)

	cookies := w.Result().Cookies()
	var foundCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "dn_admin_token" {
			foundCookie = c
			break
		}
	}
	if foundCookie == nil || foundCookie.MaxAge <= 0 {
		t.Fatalf("expected valid dn_admin_token cookie after verify-admin, got %v", foundCookie)
	}

	// 2. Hide settings window -> should clear dn_admin_token cookie
	wHide := httptest.NewRecorder()
	reqHide := httptest.NewRequest(http.MethodPost, "/api/window/settings-hide", nil)
	srv.handleSettingsHide(wHide, reqHide)

	cookiesHide := wHide.Result().Cookies()
	var clearedHideCookie *http.Cookie
	for _, c := range cookiesHide {
		if c.Name == "dn_admin_token" {
			clearedHideCookie = c
			break
		}
	}
	if clearedHideCookie == nil || clearedHideCookie.MaxAge >= 0 {
		t.Fatalf("expected dn_admin_token to be expired after handleSettingsHide, got %v", clearedHideCookie)
	}

	// 3. Explicit POST /api/config/lock-admin -> should clear dn_admin_token cookie
	wLock := httptest.NewRecorder()
	reqLock := httptest.NewRequest(http.MethodPost, "/api/config/lock-admin", nil)
	srv.handleAdminLock(wLock, reqLock)

	cookiesLock := wLock.Result().Cookies()
	var clearedLockCookie *http.Cookie
	for _, c := range cookiesLock {
		if c.Name == "dn_admin_token" {
			clearedLockCookie = c
			break
		}
	}
	if clearedLockCookie == nil || clearedLockCookie.MaxAge >= 0 {
		t.Fatalf("expected dn_admin_token to be expired after handleAdminLock, got %v", clearedLockCookie)
	}
}

func TestSecurityConfigHotReload(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "desk_notify_hotreload_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfgMgr, _ := config.NewManagerWithDir(tempDir)
	store, _ := db.NewStore(tempDir)
	defer store.Close()
	fileStorage := storage.NewStorage(tempDir)
	tun := tunnel.NewMockTunnel()
	deskMgr := desktop.NewManager()

	srv := NewServer(cfgMgr, store, fileStorage, tun, deskMgr, fstest.MapFS{})

	// 1. Verify admin password
	adminVerifyBody, _ := json.Marshal(map[string]string{"password": "123456"})
	req := httptest.NewRequest(http.MethodPost, "/api/config/verify-admin", bytes.NewReader(adminVerifyBody))
	w := httptest.NewRecorder()
	srv.handleAdminVerify(w, req)
	var verifyRes map[string]interface{}
	_ = json.NewDecoder(w.Result().Body).Decode(&verifyRes)
	adminToken := verifyRes["adminToken"].(string)

	// 2. Save security config with updated FRP settings
	hotReloadFRPBody, _ := json.Marshal(map[string]interface{}{
		"frp": map[string]interface{}{
			"enabled":    true,
			"serverAddr": "reload.freefrp.net",
			"serverPort": 7000,
			"token":      "reload-token",
			"remotePort": 39999,
		},
		"lanAuthRequired": true,
	})
	reqFRP := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewReader(hotReloadFRPBody))
	reqFRP.Header.Set("Authorization", "Bearer "+adminToken)
	wFRP := httptest.NewRecorder()
	srv.handleConfig(wFRP, reqFRP)

	if wFRP.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK from saving security config, got %d", wFRP.Result().StatusCode)
	}

	var frpRes map[string]interface{}
	if err := json.NewDecoder(wFRP.Result().Body).Decode(&frpRes); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if ok, _ := frpRes["ok"].(bool); !ok {
		t.Fatalf("expected ok: true, got %v", frpRes)
	}

	// 3. Verify tunnel was hot-reloaded with the new configuration
	if tun.Status() != "connected" {
		t.Fatalf("expected tunnel status to be connected after hot reload, got %s", tun.Status())
	}
	expectedURL := "http://reload.freefrp.net:39999"
	if tun.PublicURL() != expectedURL {
		t.Fatalf("expected tunnel public URL %s, got %s", expectedURL, tun.PublicURL())
	}

	// 4. Change unified password and verify new adminToken is returned
	newPwdBody, _ := json.Marshal(map[string]interface{}{
		"newPassword": "brandNewSecret456",
		"frp": map[string]interface{}{
			"enabled":    true,
			"serverAddr": "reload.freefrp.net",
			"serverPort": 7000,
			"token":      "reload-token",
			"remotePort": 39999,
		},
	})
	reqPwd := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewReader(newPwdBody))
	reqPwd.Header.Set("Authorization", "Bearer "+adminToken)
	wPwd := httptest.NewRecorder()
	srv.handleConfig(wPwd, reqPwd)

	if wPwd.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK when updating password, got %d", wPwd.Result().StatusCode)
	}
	var pwdRes map[string]interface{}
	_ = json.NewDecoder(wPwd.Result().Body).Decode(&pwdRes)
	newAdminToken, ok := pwdRes["adminToken"].(string)
	if !ok || newAdminToken == "" {
		t.Fatalf("expected new adminToken in response after changing password, got %v", pwdRes)
	}
	if !auth.ValidateAdminToken(newAdminToken) {
		t.Fatalf("expected returned newAdminToken to be valid")
	}
}

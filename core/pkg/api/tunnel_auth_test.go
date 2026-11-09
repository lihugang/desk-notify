package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/lihugang/desk-notify/core/pkg/auth"
	"github.com/lihugang/desk-notify/core/pkg/config"
	"github.com/lihugang/desk-notify/core/pkg/db"
	"github.com/lihugang/desk-notify/core/pkg/desktop"
	"github.com/lihugang/desk-notify/core/pkg/storage"
	"github.com/lihugang/desk-notify/core/pkg/tunnel"
)

func TestServerTunnelAndHostAuthEndToEnd(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "desk_notify_tunnel_e2e_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfgMgr, err := config.NewManagerWithDir(tempDir)
	if err != nil {
		t.Fatalf("failed to init config manager: %v", err)
	}

	cfg := cfgMgr.Get()
	cfg.Port = 28920 // dedicated test port
	cfg.LanAuthRequired = false
	cfg.FRP.Enabled = true
	cfg.FRP.ServerAddr = "kr-nc-bgp-1.ofalias.net"
	cfg.FRP.RemotePort = 39980
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

	ready := make(chan struct{})
	go func() {
		_ = srv.StartWithReady(ready)
	}()
	defer func() {
		_ = srv.Stop()
	}()

	select {
	case <-ready:
	case <-time.After(3 * time.Second):
		t.Fatalf("server failed to start within 3s")
	}

	// Give a small moment for listeners to accept connections
	time.Sleep(50 * time.Millisecond)

	client := &http.Client{Timeout: 2 * time.Second}

	// 1. Local loopback request to main port with localhost Host -> should succeed with 200 OK
	reqLocal, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/api/status", cfg.Port), nil)
	reqLocal.Host = fmt.Sprintf("localhost:%d", cfg.Port)
	respLocal, err := client.Do(reqLocal)
	if err != nil {
		t.Fatalf("failed to request local api/status: %v", err)
	}
	defer respLocal.Body.Close()
	if respLocal.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for local loopback request, got %d", respLocal.StatusCode)
	}

	// 2. Request to main port with public domain Host (simulating external frp/proxy to main port) -> should return 401
	reqPublicHost, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/api/status", cfg.Port), nil)
	reqPublicHost.Host = "kr-nc-bgp-1.ofalias.net:39980"
	respPublicHost, err := client.Do(reqPublicHost)
	if err != nil {
		t.Fatalf("failed to request public host api/status: %v", err)
	}
	defer respPublicHost.Body.Close()
	if respPublicHost.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for public domain Host, got %d", respPublicHost.StatusCode)
	}

	// 3. Request arriving at dedicated FRP tunnel port without token -> should return 401
	if srv.tunnelPort <= 0 {
		t.Fatalf("expected srv.tunnelPort > 0, got %d", srv.tunnelPort)
	}
	reqTunnel, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/api/status", srv.tunnelPort), nil)
	// Even if attacker spoofs Host to localhost on tunnel port
	reqTunnel.Host = "localhost"
	respTunnel, err := client.Do(reqTunnel)
	if err != nil {
		t.Fatalf("failed to request tunnel port api/status: %v", err)
	}
	defer respTunnel.Body.Close()
	if respTunnel.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for tunnel port request even with spoofed Host, got %d", respTunnel.StatusCode)
	}

	// 4. Request arriving at dedicated FRP tunnel port WITH token -> should return 200 OK
	token, _ := auth.GenerateToken("123456")
	reqTunnelAuth, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/api/status", srv.tunnelPort), nil)
	reqTunnelAuth.Header.Set("Authorization", "Bearer "+token)
	respTunnelAuth, err := client.Do(reqTunnelAuth)
	if err != nil {
		t.Fatalf("failed to request tunnel port with auth: %v", err)
	}
	defer respTunnelAuth.Body.Close()
	if respTunnelAuth.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for authenticated tunnel port request, got %d", respTunnelAuth.StatusCode)
	}

	var statusData map[string]interface{}
	if err := json.NewDecoder(respTunnelAuth.Body).Decode(&statusData); err != nil {
		t.Fatalf("failed to parse status json: %v", err)
	}
	if statusData["className"] != "2907" {
		t.Fatalf("expected className 2907, got %v", statusData["className"])
	}
}

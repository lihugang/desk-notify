package api

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/lihugang/desk-notify/core/pkg/config"
	"github.com/lihugang/desk-notify/core/pkg/db"
	"github.com/lihugang/desk-notify/core/pkg/desktop"
	"github.com/lihugang/desk-notify/core/pkg/storage"
	"github.com/lihugang/desk-notify/core/pkg/tunnel"
)

func setupTestServer(t *testing.T) (*Server, string) {
	tempDir, err := os.MkdirTemp("", "desk_notify_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Fake staticFS
	mockFS := fstest.MapFS{
		"alert.mp3": {
			Data: []byte("FAKE_DEFAULT_ALERT_AUDIO_DATA"),
		},
		"sounds/chime.mp3": {
			Data: []byte("FAKE_CHIME_PRESET_AUDIO_DATA"),
		},
	}

	cfgMgr, err := config.NewManager()
	if err != nil {
		t.Fatalf("failed to init config manager: %v", err)
	}

	// Override data dir for testing
	cfg := cfgMgr.Get()
	cfg.PublicPassword = ""
	cfg.LanAuthRequired = false
	cfg.SoundEnabled = true
	cfg.CustomSound = ""
	_ = cfgMgr.Save(cfg)

	store, err := db.NewStore(tempDir)
	if err != nil {
		t.Fatalf("failed to init db store: %v", err)
	}

	fileStorage := storage.NewStorage(tempDir)
	tun := tunnel.NewMockTunnel()
	deskMgr := desktop.NewManager()

	srv := NewServer(cfgMgr, store, fileStorage, tun, deskMgr, mockFS)
	return srv, tempDir
}

func TestDefaultEmbeddedSound(t *testing.T) {
	srv, tempDir := setupTestServer(t)
	defer os.RemoveAll(tempDir)
	defer srv.store.Close()

	req := httptest.NewRequest(http.MethodGet, "/alert.mp3", nil)
	w := httptest.NewRecorder()

	srv.handleCurrentSound(w, req)
	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "FAKE_DEFAULT_ALERT_AUDIO_DATA" {
		t.Fatalf("expected embedded alert data, got %s", string(body))
	}
}

func TestPresetSound(t *testing.T) {
	srv, tempDir := setupTestServer(t)
	defer os.RemoveAll(tempDir)
	defer srv.store.Close()

	cfg := srv.cfgMgr.Get()
	cfg.CustomSound = "chime.mp3"
	_ = srv.cfgMgr.Save(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/sound/current", nil)
	w := httptest.NewRecorder()

	srv.handleCurrentSound(w, req)
	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "FAKE_CHIME_PRESET_AUDIO_DATA" {
		t.Fatalf("expected preset chime data, got %s", string(body))
	}
}

func TestSoundDisabled(t *testing.T) {
	srv, tempDir := setupTestServer(t)
	defer os.RemoveAll(tempDir)
	defer srv.store.Close()

	cfg := srv.cfgMgr.Get()
	cfg.SoundEnabled = false
	_ = srv.cfgMgr.Save(cfg)

	req := httptest.NewRequest(http.MethodGet, "/alert.mp3", nil)
	w := httptest.NewRecorder()

	srv.handleCurrentSound(w, req)
	resp := w.Result()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 No Content when sound is disabled, got %d", resp.StatusCode)
	}
}

func TestUploadCustomSound(t *testing.T) {
	srv, tempDir := setupTestServer(t)
	defer os.RemoveAll(tempDir)
	defer srv.store.Close()

	bodyBuf := &bytes.Buffer{}
	writer := multipart.NewWriter(bodyBuf)

	part, err := writer.CreateFormFile("audio", "my_bell.mp3")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	_, _ = part.Write([]byte("CUSTOM_UPLOADED_AUDIO_BYTES"))
	_ = writer.Close()

	uploadReq := httptest.NewRequest(http.MethodPost, "/api/sound/upload", bodyBuf)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	srv.handleSoundUpload(w, uploadReq)
	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK from upload, got %d", resp.StatusCode)
	}

	var res map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&res)
	if res["ok"] != true {
		t.Fatalf("expected ok=true, got %v", res)
	}

	customName, ok := res["customSound"].(string)
	if !ok || customName == "" {
		t.Fatalf("expected customSound name in response, got %v", res)
	}

	// Verify that custom sound is now served by handleCurrentSound
	req2 := httptest.NewRequest(http.MethodGet, "/api/sound/current", nil)
	w2 := httptest.NewRecorder()

	srv.handleCurrentSound(w2, req2)
	resp2 := w2.Result()

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for custom sound, got %d", resp2.StatusCode)
	}

	soundBody, _ := io.ReadAll(resp2.Body)
	if string(soundBody) != "CUSTOM_UPLOADED_AUDIO_BYTES" {
		t.Fatalf("expected custom audio content, got %s", string(soundBody))
	}

	// Clean up uploaded file
	uploadedFile := filepath.Join(srv.cfgMgr.GetDataDir(), "sounds", customName)
	_ = os.Remove(uploadedFile)
}

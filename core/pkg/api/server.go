package api

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/lihugang/desk-notify/core/pkg/auth"
	"github.com/lihugang/desk-notify/core/pkg/cert"
	"github.com/lihugang/desk-notify/core/pkg/config"
	"github.com/lihugang/desk-notify/core/pkg/db"
	"github.com/lihugang/desk-notify/core/pkg/desktop"
	"github.com/lihugang/desk-notify/core/pkg/storage"
	"github.com/lihugang/desk-notify/core/pkg/tunnel"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow local & frp origins
	},
}

type Server struct {
	cfgMgr     *config.Manager
	store      *db.Store
	storage    *storage.Storage
	tunnel     tunnel.Tunnel
	desktop    desktop.Manager
	staticFS   fs.FS
	httpServer *http.Server
	httpsServer *http.Server

	mu      sync.RWMutex
	clients map[*websocket.Conn]bool
}

func NewServer(
	cfgMgr *config.Manager,
	store *db.Store,
	storage *storage.Storage,
	tun tunnel.Tunnel,
	desktop desktop.Manager,
	staticFS fs.FS,
) *Server {
	return &Server{
		cfgMgr:   cfgMgr,
		store:    store,
		storage:  storage,
		tunnel:   tun,
		desktop:  desktop,
		staticFS: staticFS,
		clients:  make(map[*websocket.Conn]bool),
	}
}

func (s *Server) Start() error {
	cfg := s.cfgMgr.Get()

	// Start tunnel
	_ = s.tunnel.Start(tunnel.Config{
		Enabled:    cfg.FRP.Enabled,
		ServerAddr: cfg.FRP.ServerAddr,
		ServerPort: cfg.FRP.ServerPort,
		Token:      cfg.FRP.Token,
		RemotePort: cfg.FRP.RemotePort,
		LocalPort:  cfg.Port,
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/config/verify-admin", s.handleAdminVerify)
	mux.HandleFunc("/api/login", s.handleLogin)
	mux.HandleFunc("/api/notify", s.handleNotify)
	mux.HandleFunc("/api/history", s.handleHistory)
	mux.HandleFunc("/api/files/", s.handleFiles)
	mux.HandleFunc("/api/sound/current", s.handleCurrentSound)
	mux.HandleFunc("/alert.mp3", s.handleCurrentSound)
	mux.HandleFunc("/api/sound/upload", s.handleSoundUpload)
	mux.HandleFunc("/api/window/hide", s.handleWindowHide)
	mux.HandleFunc("/api/window/show", s.handleWindowShow)
	mux.HandleFunc("/api/window/settings-hide", s.handleSettingsHide)
	mux.HandleFunc("/api/window/settings-show", s.handleSettingsShow)
	mux.HandleFunc("/ws", s.handleWS)

	// Root CA Certificate download endpoints
	handleCACert := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-x509-ca-cert")
		w.Header().Set("Content-Disposition", `attachment; filename="Lihugang-CA.cer"`)
		_, _ = w.Write(cert.GetRootCACertDER())
	}
	mux.HandleFunc("/Lihugang-CA.cer", handleCACert)
	mux.HandleFunc("/DeskNotify-CA.cer", handleCACert)
	mux.HandleFunc("/api/cert/ca.cer", handleCACert)

	// Static frontend router
	if s.staticFS != nil {
		fileServer := http.FileServer(http.FS(s.staticFS))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			if path == "/card" || path == "/card/" {
				r.URL.Path = "/card.html"
			} else if path == "/settings" || path == "/settings/" {
				r.URL.Path = "/settings.html"
			}
			fileServer.ServeHTTP(w, r)
		})
	}

	addr := fmt.Sprintf("0.0.0.0:%d", cfg.Port)
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// 1. Initialize local TLS Server Certificate (signed by embedded DeskNotify Root CA)
	dataDir := s.cfgMgr.GetDataDir()
	certFile, keyFile, err := cert.EnsureServerCertificate(dataDir)
	if err == nil {
		httpsPort := cfg.Port + 1
		httpsAddr := fmt.Sprintf("0.0.0.0:%d", httpsPort)
		s.httpsServer = &http.Server{
			Addr:    httpsAddr,
			Handler: mux,
		}
		go func() {
			fmt.Printf("[DeskNotify] HTTPS 服务已就绪: https://0.0.0.0:%d (CA 信任模式已启用)\n", httpsPort)
			if err := s.httpsServer.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
				fmt.Printf("[DeskNotify] HTTPS 服务异常: %v\n", err)
			}
		}()
	} else {
		fmt.Printf("[DeskNotify] 警告: 初始化 TLS 证书失败: %v\n", err)
	}

	fmt.Printf("[DeskNotify] HTTP Web server listening on %s\n", addr)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Stop() error {
	_ = s.tunnel.Stop()
	if s.httpsServer != nil {
		_ = s.httpsServer.Close()
	}
	if s.httpServer != nil {
		return s.httpServer.Close()
	}
	return nil
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	cfg := s.cfgMgr.Get()
	if !auth.CheckAuth(r, cfg) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	ips := FilterLanIPs(getLocalIPs())
	frpUrl := s.tunnel.PublicURL()
	if frpUrl == "" && cfg.FRP.Enabled && cfg.FRP.ServerAddr != "" && cfg.FRP.RemotePort > 0 {
		frpUrl = fmt.Sprintf("http://%s:%d", cfg.FRP.ServerAddr, cfg.FRP.RemotePort)
	}

	resp := map[string]interface{}{
		"className":    cfg.ClassName,
		"lanIPs":       ips,
		"port":         cfg.Port,
		"httpsPort":    cfg.Port + 1,
		"frpStatus":    s.tunnel.Status(),
		"frpPublicUrl": frpUrl,
		"frpEnabled":   cfg.FRP.Enabled,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	cfg := s.cfgMgr.Get()
	if !auth.CheckAuth(r, cfg) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method == http.MethodGet {
		lanIPs := FilterLanIPs(getLocalIPs())
		frpUrl := s.tunnel.PublicURL()
		if frpUrl == "" && cfg.FRP.Enabled && cfg.FRP.ServerAddr != "" && cfg.FRP.RemotePort > 0 {
			frpUrl = fmt.Sprintf("http://%s:%d", cfg.FRP.ServerAddr, cfg.FRP.RemotePort)
		}

		safeConfig := map[string]interface{}{
			"className":         cfg.ClassName,
			"port":              cfg.Port,
			"hasPublicPassword": cfg.PublicPasswordHash != "" || cfg.PublicPassword != "",
			"lanAuthRequired":   cfg.LanAuthRequired,
			"defaultDuration":   cfg.DefaultDuration,
			"soundEnabled":      cfg.SoundEnabled,
			"customSound":       cfg.CustomSound,
			"autoStart":         cfg.AutoStart,
			"frp":               cfg.FRP,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"config":       safeConfig,
			"lanIPs":       lanIPs,
			"frpPublicUrl": frpUrl,
			"frpEnabled":   cfg.FRP.Enabled,
		})
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			ClassName         *string           `json:"className"`
			DefaultDuration   *int              `json:"defaultDuration"`
			SoundEnabled      *bool             `json:"soundEnabled"`
			CustomSound       *string           `json:"customSound"`
			AutoStart         *bool             `json:"autoStart"`
			NewPublicPassword *string           `json:"newPublicPassword"`
			NewAdminPassword  *string           `json:"newAdminPassword"`
			FRP               *config.FRPConfig `json:"frp"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		isAdmin := auth.CheckAdminAuth(r)

		// Level 1: Basic Parameters
		if req.ClassName != nil {
			cfg.ClassName = *req.ClassName
		}
		if req.DefaultDuration != nil {
			cfg.DefaultDuration = *req.DefaultDuration
		}
		if req.SoundEnabled != nil {
			cfg.SoundEnabled = *req.SoundEnabled
		}
		if req.CustomSound != nil {
			cfg.CustomSound = *req.CustomSound
		}
		if req.AutoStart != nil {
			cfg.AutoStart = *req.AutoStart
		}

		// Level 2: Advanced & FRP Parameters (Requires admin authentication)
		frpChanged := false
		if (req.NewPublicPassword != nil && *req.NewPublicPassword != "") ||
			(req.NewAdminPassword != nil && *req.NewAdminPassword != "") ||
			req.FRP != nil {
			if !isAdmin {
				http.Error(w, "修改公网投递口令或暴露服务到公网设置需要管理员密码", http.StatusForbidden)
				return
			}

			if req.NewPublicPassword != nil && *req.NewPublicPassword != "" {
				hash, err := auth.HashPassword(*req.NewPublicPassword)
				if err == nil {
					cfg.PublicPasswordHash = hash
					cfg.PublicPassword = ""
				}
			}

			if req.NewAdminPassword != nil && *req.NewAdminPassword != "" {
				hash, err := auth.HashPassword(*req.NewAdminPassword)
				if err == nil {
					cfg.AdminPasswordHash = hash
				}
			}

			if req.FRP != nil {
				if req.FRP.RemotePort <= 0 {
					req.FRP.RemotePort = cfg.FRP.RemotePort
				}
				if req.FRP.ServerAddr == "" {
					req.FRP.ServerAddr = "frp.freefrp.net"
				}
				if req.FRP.ServerPort <= 0 {
					req.FRP.ServerPort = 7000
				}
				if req.FRP.Token == "" {
					req.FRP.Token = "freefrp.net"
				}
				cfg.FRP = *req.FRP
				frpChanged = true
			}
		}

		if err := s.cfgMgr.Save(cfg); err != nil {
			http.Error(w, "Failed to save config", http.StatusInternalServerError)
			return
		}

		if frpChanged {
			_ = s.tunnel.Stop()
			_ = s.tunnel.Start(tunnel.Config{
				Enabled:    cfg.FRP.Enabled,
				ServerAddr: cfg.FRP.ServerAddr,
				ServerPort: cfg.FRP.ServerPort,
				Token:      cfg.FRP.Token,
				RemotePort: cfg.FRP.RemotePort,
				LocalPort:  cfg.Port,
			})
		}

		w.WriteHeader(http.StatusOK)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (s *Server) handleAdminVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	cfg := s.cfgMgr.Get()
	valid := false
	if cfg.AdminPasswordHash != "" {
		valid = auth.CheckPassword(cfg.AdminPasswordHash, req.Password)
	} else {
		valid = req.Password == "123456"
	}

	if !valid {
		http.Error(w, "管理密码错误，请重新输入", http.StatusUnauthorized)
		return
	}

	token, err := auth.GenerateAdminToken()
	if err != nil {
		http.Error(w, "Token generation error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "dn_admin_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   24 * 3600,
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "adminToken": token})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	cfg := s.cfgMgr.Get()
	valid := false
	if cfg.PublicPasswordHash != "" {
		valid = auth.CheckPassword(cfg.PublicPasswordHash, req.Password)
	} else if cfg.PublicPassword != "" {
		valid = req.Password == cfg.PublicPassword
	} else {
		valid = true
	}

	if !valid {
		http.Error(w, "Invalid password", http.StatusUnauthorized)
		return
	}

	token, err := auth.GenerateToken(req.Password)
	if err != nil {
		http.Error(w, "Token error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "dn_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   7 * 24 * 3600,
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "token": token})
}

func (s *Server) handleNotify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cfg := s.cfgMgr.Get()
	if !auth.CheckAuth(r, cfg) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 100MB limit for multipart uploads
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		http.Error(w, "File upload size exceeded", http.StatusBadRequest)
		return
	}

	content := r.FormValue("content")
	level := r.FormValue("level")
	if level == "" {
		level = "info"
	}
	durationStr := r.FormValue("duration")
	duration, _ := strconv.Atoi(durationStr)
	if duration <= 0 {
		duration = cfg.DefaultDuration
	}

	notifyID := uuid.New().String()
	var attachments []*db.Attachment

	// Handle Image uploads
	if r.MultipartForm != nil && r.MultipartForm.File != nil {
		if imageFiles, ok := r.MultipartForm.File["images"]; ok {
			for _, fh := range imageFiles {
				file, err := fh.Open()
				if err != nil {
					continue
				}
				attID := uuid.New().String()
				localPath, docPath, err := s.storage.SaveFile(notifyID, fh.Filename, file, cfg.ClassName)
				file.Close()
				if err == nil {
					attachments = append(attachments, &db.Attachment{
						ID:        attID,
						NotifyID:  notifyID,
						Name:      fh.Filename,
						Size:      fh.Size,
						MimeType:  mime.TypeByExtension(filepath.Ext(fh.Filename)),
						LocalPath: localPath,
						DocPath:   docPath,
						IsImage:   true,
						URL:       fmt.Sprintf("/api/files/%s", attID),
					})
				}
			}
		}

		// Handle Attachment uploads
		if docFiles, ok := r.MultipartForm.File["attachments"]; ok {
			for _, fh := range docFiles {
				file, err := fh.Open()
				if err != nil {
					continue
				}
				attID := uuid.New().String()
				localPath, docPath, err := s.storage.SaveFile(notifyID, fh.Filename, file, cfg.ClassName)
				file.Close()
				if err == nil {
					attachments = append(attachments, &db.Attachment{
						ID:        attID,
						NotifyID:  notifyID,
						Name:      fh.Filename,
						Size:      fh.Size,
						MimeType:  mime.TypeByExtension(filepath.Ext(fh.Filename)),
						LocalPath: localPath,
						DocPath:   docPath,
						IsImage:   false,
						URL:       fmt.Sprintf("/api/files/%s", attID),
					})
				}
			}
		}
	}

	n := &db.Notification{
		ID:        notifyID,
		Content:   content,
		Level:     level,
		Duration:  duration,
		Sender:    r.RemoteAddr,
		CreatedAt: time.Now(),
	}

	// Save to SQLite
	if err := s.store.SaveNotification(n, attachments); err != nil {
		http.Error(w, "Failed to persist notification", http.StatusInternalServerError)
		return
	}

	// Broadcast via WebSocket to desktop card window
	for _, a := range attachments {
		if a.IsImage {
			n.Images = append(n.Images, a.URL)
		} else {
			n.Attachments = append(n.Attachments, a)
		}
	}
	s.broadcastNotification(n)

	// Trigger Desktop Runtime: System Notification and Pop-up Card Window
	if s.desktop != nil {
		s.desktop.Notify(n)
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "id": notifyID})
}

func (s *Server) handleWindowHide(w http.ResponseWriter, r *http.Request) {
	if s.desktop != nil {
		s.desktop.HideCard()
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleWindowShow(w http.ResponseWriter, r *http.Request) {
	if s.desktop != nil {
		s.desktop.ShowCard()
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	cfg := s.cfgMgr.Get()
	if !auth.CheckAuth(r, cfg) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	list, err := s.store.GetRecent(20)
	if err != nil {
		http.Error(w, "Failed to fetch history", http.StatusInternalServerError)
		return
	}
	if list == nil {
		list = make([]*db.Notification, 0)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(list)
}

func (s *Server) handleFiles(w http.ResponseWriter, r *http.Request) {
	attID := strings.TrimPrefix(r.URL.Path, "/api/files/")
	if attID == "" {
		http.NotFound(w, r)
		return
	}

	att, err := s.store.GetAttachment(attID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	f, err := os.Open(att.LocalPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()

	if att.MimeType != "" {
		w.Header().Set("Content-Type", att.MimeType)
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", att.Name))
	_, _ = io.Copy(w, f)
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	s.mu.Lock()
	s.clients[conn] = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.clients, conn)
		s.mu.Unlock()
		_ = conn.Close()
	}()

	// Keep alive read loop
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

func (s *Server) broadcastNotification(n *db.Notification) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cfg := s.cfgMgr.Get()

	// Broadcast notification enriched with sound configuration
	msg := struct {
		*db.Notification
		SoundEnabled bool   `json:"soundEnabled"`
		SoundURL     string `json:"soundUrl"`
	}{
		Notification: n,
		SoundEnabled: cfg.SoundEnabled,
		SoundURL:     fmt.Sprintf("/api/sound/current?t=%d", time.Now().UnixMilli()),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	for client := range s.clients {
		_ = client.WriteMessage(websocket.TextMessage, data)
	}
}

func (s *Server) handleCurrentSound(w http.ResponseWriter, r *http.Request) {
	cfg := s.cfgMgr.Get()

	if !cfg.SoundEnabled {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	custom := strings.TrimSpace(cfg.CustomSound)
	if custom != "" && custom != "default" {
		// 1. External HTTP/HTTPS URL
		if strings.HasPrefix(custom, "http://") || strings.HasPrefix(custom, "https://") {
			http.Redirect(w, r, custom, http.StatusFound)
			return
		}

		// 2. Uploaded sound in AppData/sounds
		uploadedPath := filepath.Join(s.cfgMgr.GetDataDir(), "sounds", filepath.Base(custom))
		if fi, err := os.Stat(uploadedPath); err == nil && !fi.IsDir() {
			w.Header().Set("Cache-Control", "no-cache, must-revalidate")
			http.ServeFile(w, r, uploadedPath)
			return
		}

		// 3. Local filesystem path
		if fi, err := os.Stat(custom); err == nil && !fi.IsDir() {
			w.Header().Set("Cache-Control", "no-cache, must-revalidate")
			http.ServeFile(w, r, custom)
			return
		}

		// 4. Built-in preset sound in staticFS (e.g. "sounds/chime.mp3" or "chime.mp3")
		if s.staticFS != nil {
			presetPath := custom
			if !strings.HasPrefix(presetPath, "sounds/") {
				presetPath = "sounds/" + presetPath
			}
			if f, err := s.staticFS.Open(presetPath); err == nil {
				defer f.Close()
				contentType := mime.TypeByExtension(filepath.Ext(presetPath))
				if contentType == "" {
					contentType = "audio/mpeg"
				}
				w.Header().Set("Content-Type", contentType)
				w.Header().Set("Cache-Control", "public, max-age=86400")
				_, _ = io.Copy(w, f)
				return
			}
		}
	}

	// Default fallback: embedded alert.mp3
	if s.staticFS != nil {
		f, err := s.staticFS.Open("alert.mp3")
		if err == nil {
			defer f.Close()
			w.Header().Set("Content-Type", "audio/mpeg")
			w.Header().Set("Cache-Control", "public, max-age=86400")
			_, _ = io.Copy(w, f)
			return
		}
	}

	http.NotFound(w, r)
}

func (s *Server) handleSoundUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cfg := s.cfgMgr.Get()
	if !auth.CheckAuth(r, cfg) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Max 10MB audio file
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Audio file too large", http.StatusBadRequest)
		return
	}

	file, fh, err := r.FormFile("audio")
	if err != nil {
		// Also check "file" field name
		var err2 error
		file, fh, err2 = r.FormFile("file")
		if err2 != nil {
			http.Error(w, "Missing audio file in form", http.StatusBadRequest)
			return
		}
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(fh.Filename))
	if ext != ".mp3" && ext != ".wav" && ext != ".ogg" && ext != ".m4a" && ext != ".aac" && ext != ".flac" {
		http.Error(w, "Unsupported audio format. Please use .mp3, .wav, or .ogg", http.StatusBadRequest)
		return
	}

	soundsDir := filepath.Join(s.cfgMgr.GetDataDir(), "sounds")
	if err := os.MkdirAll(soundsDir, 0755); err != nil {
		http.Error(w, "Failed to create sounds directory", http.StatusInternalServerError)
		return
	}

	cleanFileName := fmt.Sprintf("custom_sound_%d%s", time.Now().Unix(), ext)
	targetPath := filepath.Join(soundsDir, cleanFileName)

	out, err := os.Create(targetPath)
	if err != nil {
		http.Error(w, "Failed to create target file", http.StatusInternalServerError)
		return
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		http.Error(w, "Failed to save audio content", http.StatusInternalServerError)
		return
	}

	// Update configuration
	cfg.CustomSound = cleanFileName
	if err := s.cfgMgr.Save(cfg); err != nil {
		http.Error(w, "Failed to persist configuration", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":           true,
		"customSound":  cleanFileName,
		"originalName": fh.Filename,
		"url":          "/api/sound/current",
	})
}

func (s *Server) handleSettingsHide(w http.ResponseWriter, r *http.Request) {
	if s.desktop != nil {
		s.desktop.HideSettings()
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleSettingsShow(w http.ResponseWriter, r *http.Request) {
	if s.desktop != nil {
		s.desktop.OpenSettings()
	}
	w.WriteHeader(http.StatusOK)
}

func getLocalIPs() []string {
	var ips []string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return []string{"127.0.0.1"}
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ips = append(ips, ipnet.IP.String())
			}
		}
	}
	if len(ips) == 0 {
		return []string{"127.0.0.1"}
	}
	return ips
}

// FilterLanIPs filters LAN IP addresses according to requirements:
// If there are any IP addresses starting with "192.", use and return only those starting with 192, and hide others.
// If none start with "192.", display all of them.
func FilterLanIPs(ips []string) []string {
	var ips192 []string
	for _, ip := range ips {
		if strings.HasPrefix(ip, "192.") {
			ips192 = append(ips192, ip)
		}
	}
	if len(ips192) > 0 {
		return ips192
	}
	return ips
}

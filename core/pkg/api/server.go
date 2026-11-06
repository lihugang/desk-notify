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
	"github.com/lihugang/desk-notify/core/pkg/config"
	"github.com/lihugang/desk-notify/core/pkg/db"
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
	staticFS   fs.FS
	httpServer *http.Server

	mu      sync.RWMutex
	clients map[*websocket.Conn]bool
}

func NewServer(
	cfgMgr *config.Manager,
	store *db.Store,
	storage *storage.Storage,
	tun tunnel.Tunnel,
	staticFS fs.FS,
) *Server {
	return &Server{
		cfgMgr:   cfgMgr,
		store:    store,
		storage:  storage,
		tunnel:   tun,
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
	mux.HandleFunc("/api/login", s.handleLogin)
	mux.HandleFunc("/api/notify", s.handleNotify)
	mux.HandleFunc("/api/history", s.handleHistory)
	mux.HandleFunc("/api/files/", s.handleFiles)
	mux.HandleFunc("/ws", s.handleWS)

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

	fmt.Printf("[DeskNotify] Web server listening on %s\n", addr)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Stop() error {
	_ = s.tunnel.Stop()
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

	ips := getLocalIPs()
	resp := map[string]interface{}{
		"className":   cfg.ClassName,
		"lanIPs":      ips,
		"port":        cfg.Port,
		"frpStatus":   s.tunnel.Status(),
		"frpPublicUrl": s.tunnel.PublicURL(),
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
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"config":       cfg,
			"lanIPs":       getLocalIPs(),
			"frpPublicUrl": s.tunnel.PublicURL(),
		})
		return
	}

	if r.Method == http.MethodPost {
		var newCfg config.AppConfig
		if err := json.NewDecoder(r.Body).Decode(&newCfg); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		if err := s.cfgMgr.Save(newCfg); err != nil {
			http.Error(w, "Failed to save config", http.StatusInternalServerError)
			return
		}

		// Reload tunnel
		_ = s.tunnel.Stop()
		_ = s.tunnel.Start(tunnel.Config{
			Enabled:    newCfg.FRP.Enabled,
			ServerAddr: newCfg.FRP.ServerAddr,
			ServerPort: newCfg.FRP.ServerPort,
			Token:      newCfg.FRP.Token,
			RemotePort: newCfg.FRP.RemotePort,
			LocalPort:  newCfg.Port,
		})

		w.WriteHeader(http.StatusOK)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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
	if cfg.PublicPassword != "" && req.Password != cfg.PublicPassword {
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
		}
		n.Attachments = append(n.Attachments, a)
	}
	s.broadcastNotification(n)

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "id": notifyID})
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

	data, err := json.Marshal(n)
	if err != nil {
		return
	}

	for client := range s.clients {
		_ = client.WriteMessage(websocket.TextMessage, data)
	}
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

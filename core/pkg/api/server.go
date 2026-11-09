package api

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net"
	"net/http"
	"net/url"
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
	"github.com/lihugang/desk-notify/core/pkg/logger"
	"github.com/lihugang/desk-notify/core/pkg/storage"
	"github.com/lihugang/desk-notify/core/pkg/tunnel"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow local & frp origins
	},
}

type Server struct {
	cfgMgr         *config.Manager
	store          *db.Store
	storage        *storage.Storage
	tunnel         tunnel.Tunnel
	desktop        desktop.Manager
	staticFS       fs.FS
	httpServer     *http.Server
	httpsServer    *http.Server
	tunnelServer   *http.Server
	tunnelListener net.Listener
	tunnelPort     int

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

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (s *Server) Start() error {
	return s.StartWithReady(nil)
}

func (s *Server) StartWithReady(ready chan<- struct{}) error {
	cfg := s.cfgMgr.Get()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/config/verify-admin", s.handleAdminVerify)
	mux.HandleFunc("/api/login", s.handleLogin)
	mux.HandleFunc("/api/notify/latest", s.handleLatestNotify)
	mux.HandleFunc("/api/notify", s.handleNotify)
	mux.HandleFunc("/api/history", s.handleHistory)
	mux.HandleFunc("/api/files/open", s.handleFileOpen)
	mux.HandleFunc("/api/files/reveal", s.handleFileReveal)
	mux.HandleFunc("/api/files/", s.handleFiles)
	mux.HandleFunc("/api/sound/current", s.handleCurrentSound)
	mux.HandleFunc("/alert.mp3", s.handleCurrentSound)
	mux.HandleFunc("/api/sound/upload", s.handleSoundUpload)
	mux.HandleFunc("/api/window/hide", s.handleWindowHide)
	mux.HandleFunc("/api/window/show", s.handleWindowShow)
	mux.HandleFunc("/api/window/minimize", s.handleWindowMinimize)
	mux.HandleFunc("/api/window/stop-flash", s.handleWindowStopFlash)
	mux.HandleFunc("/api/window/card-height", s.handleCardHeight)
	mux.HandleFunc("/api/window/settings-hide", s.handleSettingsHide)
	mux.HandleFunc("/api/window/settings-show", s.handleSettingsShow)
	mux.HandleFunc("/api/config/lock-admin", s.handleAdminLock)
	mux.HandleFunc("/api/window/history-hide", s.handleHistoryHide)
	mux.HandleFunc("/api/window/history-show", s.handleHistoryShow)
	mux.HandleFunc("/api/archive/open", s.handleOpenArchiveDir)
	mux.HandleFunc("/api/open-archive-dir", s.handleOpenArchiveDir)
	mux.HandleFunc("/api/logs/open", s.handleOpenLogsDir)
	mux.HandleFunc("/api/open-logs-dir", s.handleOpenLogsDir)
	mux.HandleFunc("/api/window/pin-taskbar", s.handlePinTaskbar)
	mux.HandleFunc("/api/window/desktop-shortcut", s.handleDesktopShortcut)
	mux.HandleFunc("/api/cert/status", s.handleCertStatus)
	mux.HandleFunc("/api/cert/trust", s.handleCertTrust)
	mux.HandleFunc("/ws", s.handleWS)

	// Root CA Certificate download endpoints
	handleCACert := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-x509-ca-cert")
		w.Header().Set("Content-Disposition", `attachment; filename="DeskNotify-CA.cer"`)
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
			} else if path == "/history" || path == "/history/" {
				r.URL.Path = "/history.html"
			}
			fileServer.ServeHTTP(w, r)
		})
	}

	loggingHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ws" {
			mux.ServeHTTP(w, r)
			return
		}
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		mux.ServeHTTP(rec, r)
		duration := time.Since(start)

		if strings.HasPrefix(r.URL.Path, "/api/") || rec.status >= 400 {
			logger.Infof("[HTTP] %s %s from %s -> %d (%v)", r.Method, r.URL.Path, r.RemoteAddr, rec.status, duration)
		}
	})

	// 1. Initialize dedicated internal tunnel listener for FRP on loopback
	tunnelLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err == nil {
		s.tunnelListener = tunnelLn
		s.tunnelPort = tunnelLn.Addr().(*net.TCPAddr).Port
		tunnelHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := auth.WithTunnelContext(r.Context())
			loggingHandler.ServeHTTP(w, r.WithContext(ctx))
		})
		s.tunnelServer = &http.Server{
			Handler: tunnelHandler,
		}
		go func() {
			if err := s.tunnelServer.Serve(tunnelLn); err != nil && err != http.ErrServerClosed {
				logger.Errorf("[DeskNotify] 内部 FRP 专用穿透转发服务异常: %v", err)
			}
		}()
		logger.Infof("[DeskNotify] 内部 FRP 专用穿透端口已就绪: 127.0.0.1:%d", s.tunnelPort)
	} else {
		logger.Warnf("[DeskNotify] 启动内部 FRP 专用穿透监听失败: %v", err)
	}

	// 2. Start FRP tunnel (forwarding to dedicated tunnel port)
	_ = s.tunnel.Start(s.toTunnelConfig(cfg))

	addr := fmt.Sprintf("0.0.0.0:%d", cfg.Port)
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: loggingHandler,
	}

	// 3. Initialize local TLS Server Certificate (signed by embedded DeskNotify Root CA)
	dataDir := s.cfgMgr.GetDataDir()
	certFile, keyFile, err := cert.EnsureServerCertificate(dataDir)
	if err == nil {
		httpsPort := cfg.Port + 1
		httpsAddr := fmt.Sprintf("0.0.0.0:%d", httpsPort)
		s.httpsServer = &http.Server{
			Addr:    httpsAddr,
			Handler: loggingHandler,
		}
		go func() {
			logger.Infof("[DeskNotify] HTTPS 服务已就绪: https://0.0.0.0:%d (CA 信任模式已启用)", httpsPort)
			if err := s.httpsServer.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
				logger.Errorf("[DeskNotify] HTTPS 服务异常: %v", err)
			}
		}()
	} else {
		logger.Warnf("[DeskNotify] 警告: 初始化 TLS 证书失败: %v", err)
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	if ready != nil {
		close(ready)
	}

	logger.Infof("[DeskNotify] HTTP Web server listening on %s", addr)
	return s.httpServer.Serve(ln)
}

func (s *Server) Stop() error {
	_ = s.tunnel.Stop()
	if s.tunnelServer != nil {
		_ = s.tunnelServer.Close()
	}
	if s.tunnelListener != nil {
		_ = s.tunnelListener.Close()
	}
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

		hasPassword := cfg.GetEffectivePasswordHash() != "" || cfg.PublicPassword != ""
		safeConfig := map[string]interface{}{
			"className":         cfg.ClassName,
			"port":              cfg.Port,
			"hasPassword":       hasPassword,
			"hasPublicPassword": hasPassword,
			"lanAuthRequired":   cfg.LanAuthRequired,
			"soundEnabled":      cfg.SoundEnabled,
			"customSound":       cfg.CustomSound,
			"fontSize":          cfg.FontSize,
			"autoStart":         cfg.AutoStart,
			"firstRunDone":      cfg.FirstRunDone,
			"frp":               cfg.FRP,
			"documentsDir":      config.GetDocumentsDir(cfg.ClassName),
			"logsDir":           config.GetLogsDir(),
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"config":       safeConfig,
			"lanIPs":       lanIPs,
			"frpPublicUrl": frpUrl,
			"frpStatus":    s.tunnel.Status(),
			"frpEnabled":   cfg.FRP.Enabled,
			"isFirstRun":   s.cfgMgr.IsFirstRun(),
			"caInstalled":  cert.IsRootCAInstalled(),
		})
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			ClassName         *string           `json:"className"`
			SoundEnabled      *bool             `json:"soundEnabled"`
			CustomSound       *string           `json:"customSound"`
			FontSize          *string           `json:"fontSize"`
			AutoStart         *bool             `json:"autoStart"`
			LanAuthRequired   *bool             `json:"lanAuthRequired"`
			NewPassword       *string           `json:"newPassword"`
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
		if req.SoundEnabled != nil {
			cfg.SoundEnabled = *req.SoundEnabled
		}
		if req.CustomSound != nil {
			cfg.CustomSound = *req.CustomSound
		}
		if req.FontSize != nil && *req.FontSize != "" {
			cfg.FontSize = *req.FontSize
		}
		if req.AutoStart != nil {
			cfg.AutoStart = *req.AutoStart
			if s.desktop != nil {
				_ = s.desktop.SetAutoStart(*req.AutoStart)
			}
		}

		// Level 2: Advanced & FRP & Security Parameters (Requires admin authentication)
		var newPasswordVal string
		if req.NewPassword != nil && *req.NewPassword != "" {
			newPasswordVal = *req.NewPassword
		} else if req.NewAdminPassword != nil && *req.NewAdminPassword != "" {
			newPasswordVal = *req.NewAdminPassword
		} else if req.NewPublicPassword != nil && *req.NewPublicPassword != "" {
			newPasswordVal = *req.NewPublicPassword
		}

		frpChanged := false
		if newPasswordVal != "" ||
			req.LanAuthRequired != nil ||
			req.FRP != nil {
			if !isAdmin {
				http.Error(w, "修改安全与公网设置需要管理员密码", http.StatusForbidden)
				return
			}

			if req.LanAuthRequired != nil {
				cfg.LanAuthRequired = *req.LanAuthRequired
			}

			if newPasswordVal != "" {
				hash, err := auth.HashPassword(newPasswordVal)
				if err == nil {
					cfg.PasswordHash = hash
					cfg.PublicPasswordHash = hash
					cfg.AdminPasswordHash = hash
					cfg.PublicPassword = ""
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

		cfg.FirstRunDone = true
		if err := s.cfgMgr.Save(cfg); err != nil {
			logger.Errorf("[Config] 保存系统配置失败: %v", err)
			http.Error(w, "Failed to save config", http.StatusInternalServerError)
			return
		}
		logger.Infof("[Config] 系统配置已更新并保存 (班级: %s, 端口: %d, 来源: %s)", cfg.ClassName, cfg.Port, r.RemoteAddr)
		s.broadcastConfig(cfg)

		// Hot reload tunnel / FRP whenever FRP config or security parameters are saved
		if frpChanged || (isAdmin && (req.FRP != nil || req.LanAuthRequired != nil || newPasswordVal != "")) {
			if s.tunnel != nil {
				_ = s.tunnel.Stop()
				_ = s.tunnel.Start(s.toTunnelConfig(cfg))
			}
		}

		var respAdminToken string
		var respUserToken string
		if newPasswordVal != "" {
			// If unified password changed, generate new admin & user tokens so current session stays authenticated
			if newToken, err := auth.GenerateAdminToken(); err == nil {
				respAdminToken = newToken
				http.SetCookie(w, &http.Cookie{
					Name:     "dn_admin_token",
					Value:    newToken,
					Path:     "/",
					HttpOnly: true,
					MaxAge:   7 * 24 * 3600,
				})
			}
			if uToken, err := auth.GenerateToken(newPasswordVal); err == nil {
				respUserToken = uToken
				http.SetCookie(w, &http.Cookie{
					Name:     "dn_token",
					Value:    uToken,
					Path:     "/",
					HttpOnly: true,
					MaxAge:   7 * 24 * 3600,
				})
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":           true,
			"adminToken":   respAdminToken,
			"token":        respUserToken,
			"frpStatus":    s.tunnel.Status(),
			"frpPublicUrl": s.tunnel.PublicURL(),
		})
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
	pwdHash := cfg.GetEffectivePasswordHash()
	if pwdHash != "" {
		valid = auth.CheckPassword(pwdHash, req.Password)
	} else {
		valid = req.Password == "123456"
	}

	if !valid {
		logger.Warnf("[Auth] 管理员身份验证失败: 密码错误 (来源: %s)", r.RemoteAddr)
		http.Error(w, "密码错误，请重新输入", http.StatusUnauthorized)
		return
	}

	token, err := auth.GenerateAdminToken()
	if err != nil {
		logger.Errorf("[Auth] 生成管理员 Token 失败: %v", err)
		http.Error(w, "Token generation error", http.StatusInternalServerError)
		return
	}

	logger.Infof("[Auth] 管理员身份验证通过 (来源: %s)", r.RemoteAddr)

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
	pwdHash := cfg.GetEffectivePasswordHash()
	if pwdHash != "" {
		valid = auth.CheckPassword(pwdHash, req.Password)
	} else if cfg.PublicPassword != "" {
		valid = req.Password == cfg.PublicPassword
	} else {
		valid = true
	}

	if !valid {
		logger.Warnf("[Auth] 教师端口令验证失败: 密码错误 (来源: %s)", r.RemoteAddr)
		http.Error(w, "密码错误，请重新输入", http.StatusUnauthorized)
		return
	}

	token, err := auth.GenerateToken(req.Password)
	if err != nil {
		logger.Errorf("[Auth] 生成教师端 Token 失败: %v", err)
		http.Error(w, "Token error", http.StatusInternalServerError)
		return
	}

	adminToken, err := auth.GenerateAdminToken()
	if err != nil {
		logger.Warnf("[Auth] 生成管理员 Token 失败: %v", err)
	}

	logger.Infof("[Auth] 教师端口令验证通过 (来源: %s)", r.RemoteAddr)

	http.SetCookie(w, &http.Cookie{
		Name:     "dn_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   7 * 24 * 3600,
	})
	if adminToken != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     "dn_admin_token",
			Value:    adminToken,
			Path:     "/",
			HttpOnly: true,
			MaxAge:   7 * 24 * 3600,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":         true,
		"token":      token,
		"adminToken": adminToken,
	})
}

func (s *Server) handleLatestNotify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cfg := s.cfgMgr.Get()
	list, err := s.store.GetRecentWithinMinutes(90)
	if err != nil || len(list) == 0 {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"hasNotification": false,
			"className":       cfg.ClassName,
			"fontSize":        cfg.FontSize,
			"notifications":   []*db.Notification{},
		})
		return
	}

	n := list[0]
	type latestResponse struct {
		HasNotification bool               `json:"hasNotification"`
		ClassName       string             `json:"className"`
		Notification    *db.Notification   `json:"notification"`
		Notifications   []*db.Notification `json:"notifications"`
		SoundEnabled    bool               `json:"soundEnabled"`
		SoundURL        string             `json:"soundUrl"`
		FontSize        string             `json:"fontSize,omitempty"`
	}

	resp := latestResponse{
		HasNotification: true,
		ClassName:       cfg.ClassName,
		Notification:    n,
		Notifications:   list,
		SoundEnabled:    cfg.SoundEnabled,
		SoundURL:        fmt.Sprintf("/api/sound/current?t=%d", time.Now().UnixMilli()),
		FontSize:        cfg.FontSize,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
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
	pinnedStr := r.FormValue("pinned")
	pinned := true
	if pinnedStr == "false" || pinnedStr == "0" {
		pinned = false
	}
	// 弹窗卡片常驻，需手动点击关闭 (duration = 0)
	duration := 0

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

	sender := r.RemoteAddr
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 && strings.TrimSpace(parts[0]) != "" {
			sender = strings.TrimSpace(parts[0])
		}
	} else if auth.IsTunnelRequest(r) || auth.IsPublicRequest(r) {
		sender = "公网穿透"
		if r.Host != "" {
			sender = fmt.Sprintf("公网穿透 (%s)", r.Host)
		}
	}

	n := &db.Notification{
		ID:        notifyID,
		Content:   content,
		Pinned:    pinned,
		Duration:  duration,
		Sender:    sender,
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

	logger.Infof("[Notify] 成功推送新通知 (ID: %s, 置顶: %v, 附件数: %d, 来源: %s)", notifyID, pinned, len(attachments), r.RemoteAddr)

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

func (s *Server) handleWindowMinimize(w http.ResponseWriter, r *http.Request) {
	if s.desktop != nil {
		s.desktop.MinimizeCard()
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleWindowStopFlash(w http.ResponseWriter, r *http.Request) {
	if s.desktop != nil {
		s.desktop.StopFlash()
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleCardHeight(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Height int `json:"height"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Height < 200 || req.Height > 3000 {
		http.Error(w, "Invalid height", http.StatusBadRequest)
		return
	}
	if s.desktop != nil {
		s.desktop.SetCardHeight(req.Height)
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cfg := s.cfgMgr.Get()
	// In LAN or localhost, history notifications can be viewed freely (consistent with /api/notify/latest;
	// lanAuthRequired is only for posting notifications and admin config).
	// Public requests via FRP still require authentication if password is configured.
	if auth.IsPublicRequest(r) && !auth.CheckAuth(r, cfg) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	limit := 10
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = val
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val >= 0 {
			offset = val
		}
	}

	list, total, err := s.store.GetHistory(limit, offset)
	if err != nil {
		http.Error(w, "Failed to fetch history", http.StatusInternalServerError)
		return
	}
	if list == nil {
		list = make([]*db.Notification, 0)
	}

	hasMore := offset+len(list) < total

	resp := struct {
		Items   []*db.Notification `json:"items"`
		Total   int                `json:"total"`
		Limit   int                `json:"limit"`
		Offset  int                `json:"offset"`
		HasMore bool               `json:"hasMore"`
	}{
		Items:   list,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		HasMore: hasMore,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleFiles(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/open") {
		attID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/files/"), "/open")
		s.processFileOpen(w, r, attID)
		return
	}
	if strings.HasSuffix(r.URL.Path, "/reveal") || strings.HasSuffix(r.URL.Path, "/locate") {
		attID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/files/"), "/reveal")
		attID = strings.TrimSuffix(attID, "/locate")
		s.processFileReveal(w, r, attID)
		return
	}

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

	disposition := "inline"
	if r.URL.Query().Get("download") == "1" || !att.IsImage {
		disposition = "attachment"
	}
	encodedName := url.PathEscape(att.Name)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`%s; filename=%q; filename*=UTF-8''%s`, disposition, att.Name, encodedName))
	_, _ = io.Copy(w, f)
}

type fileActionReq struct {
	ID string `json:"id"`
}

func (s *Server) handleFileOpen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cfg := s.cfgMgr.Get()
	if !auth.CheckAuth(r, cfg) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" && r.Body != nil {
		var req fileActionReq
		_ = json.NewDecoder(r.Body).Decode(&req)
		id = req.ID
	}
	if id == "" {
		http.Error(w, "Missing file ID", http.StatusBadRequest)
		return
	}
	s.processFileOpen(w, r, id)
}

func (s *Server) handleFileReveal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cfg := s.cfgMgr.Get()
	if !auth.CheckAuth(r, cfg) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" && r.Body != nil {
		var req fileActionReq
		_ = json.NewDecoder(r.Body).Decode(&req)
		id = req.ID
	}
	if id == "" {
		http.Error(w, "Missing file ID", http.StatusBadRequest)
		return
	}
	s.processFileReveal(w, r, id)
}

func (s *Server) resolveAttachmentDocPath(att *db.Attachment) (string, error) {
	if att.DocPath != "" {
		if _, err := os.Stat(att.DocPath); err == nil {
			return att.DocPath, nil
		}
	}

	// If DocPath doesn't exist, but LocalPath exists, mirror it to DocPath
	if att.LocalPath != "" {
		if _, err := os.Stat(att.LocalPath); err == nil {
			dest := att.DocPath
			if dest == "" {
				cfg := s.cfgMgr.Get()
				docBase := config.GetDocumentsDir(cfg.ClassName)
				dateDir := time.Now().Format("2006-01-02")
				docTargetDir := filepath.Join(docBase, dateDir)
				_ = os.MkdirAll(docTargetDir, 0755)
				dest = filepath.Join(docTargetDir, att.Name)
			} else {
				_ = os.MkdirAll(filepath.Dir(dest), 0755)
			}

			// Try link or copy
			if err := os.Link(att.LocalPath, dest); err != nil {
				_ = storage.CopyFile(att.LocalPath, dest)
			}
			if _, err := os.Stat(dest); err == nil {
				return dest, nil
			}
			return att.LocalPath, nil
		}
	}

	if att.DocPath != "" {
		return att.DocPath, nil
	}
	return "", fmt.Errorf("文件在磁盘上不存在")
}

func (s *Server) processFileOpen(w http.ResponseWriter, r *http.Request, attID string) {
	cfg := s.cfgMgr.Get()
	if !auth.CheckAuth(r, cfg) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	att, err := s.store.GetAttachment(attID)
	if err != nil || att == nil {
		http.Error(w, "附件不存在", http.StatusNotFound)
		return
	}

	targetPath, err := s.resolveAttachmentDocPath(att)
	if err != nil {
		http.Error(w, fmt.Sprintf("无法定位文件: %v", err), http.StatusInternalServerError)
		return
	}

	var openErr error
	if s.desktop != nil {
		openErr = s.desktop.OpenFile(targetPath)
	} else {
		openErr = desktop.OpenFile(targetPath)
	}

	if openErr != nil {
		http.Error(w, fmt.Sprintf("打开文件失败: %v", openErr), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":   true,
		"path": targetPath,
	})
}

func (s *Server) processFileReveal(w http.ResponseWriter, r *http.Request, attID string) {
	cfg := s.cfgMgr.Get()
	if !auth.CheckAuth(r, cfg) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	att, err := s.store.GetAttachment(attID)
	if err != nil || att == nil {
		http.Error(w, "附件不存在", http.StatusNotFound)
		return
	}

	targetPath, err := s.resolveAttachmentDocPath(att)
	if err != nil {
		// Fallback to opening the archive directory
		docDir := config.GetDocumentsDir(cfg.ClassName)
		if s.desktop != nil {
			_ = s.desktop.OpenArchiveDir()
		} else {
			_ = desktop.RevealFile(docDir)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":   true,
			"path": docDir,
		})
		return
	}

	var revealErr error
	if s.desktop != nil {
		revealErr = s.desktop.RevealFile(targetPath)
	} else {
		revealErr = desktop.RevealFile(targetPath)
	}

	if revealErr != nil {
		http.Error(w, fmt.Sprintf("定位文件失败: %v", revealErr), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":   true,
		"path": targetPath,
	})
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	cfg := s.cfgMgr.Get()
	if auth.IsPublicRequest(r) && !auth.CheckAuth(r, cfg) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Warnf("[WS] 升级 WebSocket 失败 (%s): %v", r.RemoteAddr, err)
		return
	}

	s.mu.Lock()
	s.clients[conn] = true
	clientCount := len(s.clients)
	s.mu.Unlock()

	logger.Infof("[WS] 客户端已连接 (%s, 当前在线客户端: %d)", r.RemoteAddr, clientCount)

	defer func() {
		s.mu.Lock()
		delete(s.clients, conn)
		remaining := len(s.clients)
		s.mu.Unlock()
		_ = conn.Close()
		logger.Infof("[WS] 客户端已断开 (%s, 剩余在线客户端: %d)", r.RemoteAddr, remaining)
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

	var recent []*db.Notification
	if s.store != nil {
		recent, _ = s.store.GetRecentWithinMinutes(90)
	}
	if len(recent) == 0 {
		recent = []*db.Notification{n}
	}

	// Broadcast notification enriched with sound configuration, font size and recent 90-min list
	msg := struct {
		*db.Notification
		SoundEnabled  bool               `json:"soundEnabled"`
		SoundURL      string             `json:"soundUrl"`
		Notifications []*db.Notification `json:"notifications"`
		FontSize      string             `json:"fontSize,omitempty"`
	}{
		Notification:  n,
		SoundEnabled:  cfg.SoundEnabled,
		SoundURL:      fmt.Sprintf("/api/sound/current?t=%d", time.Now().UnixMilli()),
		Notifications: recent,
		FontSize:      cfg.FontSize,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	for client := range s.clients {
		_ = client.WriteMessage(websocket.TextMessage, data)
	}
}

func (s *Server) broadcastConfig(cfg config.AppConfig) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	msg := map[string]interface{}{
		"type":         "config",
		"className":    cfg.ClassName,
		"fontSize":     cfg.FontSize,
		"soundEnabled": cfg.SoundEnabled,
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
	_ = s.cfgMgr.MarkFirstRunComplete()
	http.SetCookie(w, &http.Cookie{
		Name:     "dn_admin_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleAdminLock(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "dn_admin_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
}

func (s *Server) handleSettingsShow(w http.ResponseWriter, r *http.Request) {
	if s.desktop != nil {
		s.desktop.OpenSettings()
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handlePinTaskbar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var msg string
	pinned := true
	if s.desktop != nil {
		if err := s.desktop.PinToTaskbar(true); err != nil {
			pinned = false
			msg = err.Error()
		} else {
			msg = "已成功将 DeskNotify 传纸条固定到底部任务栏！"
		}
	} else {
		msg = "非桌面环境，已跳过固定任务栏"
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":      true,
		"pinned":  pinned,
		"message": msg,
	})
}

func (s *Server) handleDesktopShortcut(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var createErr error
	if s.desktop != nil {
		createErr = s.desktop.CreateDesktopShortcut()
	}
	w.Header().Set("Content-Type", "application/json")
	if createErr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":    false,
			"error": createErr.Error(),
		})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":      true,
		"message": "已成功在桌面创建【传纸条】快捷方式",
	})
}

func (s *Server) handleCertStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	installed := cert.IsRootCAInstalled()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"installed": installed,
	})
}

func (s *Server) handleCertTrust(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	err := cert.EnsureRootCAInstalled()
	installed := cert.IsRootCAInstalled()
	w.Header().Set("Content-Type", "application/json")
	if err != nil && !installed {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":        false,
			"installed": false,
			"error":     err.Error(),
		})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":        true,
		"installed": installed,
		"message":   "根证书已成功信任安装",
	})
}

func (s *Server) handleHistoryHide(w http.ResponseWriter, r *http.Request) {
	if s.desktop != nil {
		s.desktop.HideHistory()
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleHistoryShow(w http.ResponseWriter, r *http.Request) {
	if s.desktop != nil {
		s.desktop.OpenHistory()
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleOpenArchiveDir(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cfg := s.cfgMgr.Get()
	if !auth.CheckAuth(r, cfg) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if s.desktop != nil {
		if err := s.desktop.OpenArchiveDir(); err != nil {
			logger.Errorf("[DeskNotify] 打开通知存档目录失败: %v", err)
			http.Error(w, fmt.Sprintf("打开存档目录失败: %v", err), http.StatusInternalServerError)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":           true,
		"documentsDir": config.GetDocumentsDir(cfg.ClassName),
	})
}

func (s *Server) handleOpenLogsDir(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cfg := s.cfgMgr.Get()
	if !auth.CheckAuth(r, cfg) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if s.desktop != nil {
		if err := s.desktop.OpenLogsDir(); err != nil {
			logger.Errorf("[DeskNotify] 打开运行日志目录失败: %v", err)
			http.Error(w, fmt.Sprintf("打开日志目录失败: %v", err), http.StatusInternalServerError)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":      true,
		"logsDir": config.GetLogsDir(),
	})
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

func (s *Server) toTunnelConfig(cfg config.AppConfig) tunnel.Config {
	localPort := cfg.Port
	if s != nil && s.tunnelPort > 0 {
		localPort = s.tunnelPort
	}
	return tunnel.Config{
		Enabled:                   cfg.FRP.Enabled,
		ServerAddr:                cfg.FRP.ServerAddr,
		ServerPort:                cfg.FRP.ServerPort,
		Token:                     cfg.FRP.Token,
		RemotePort:                cfg.FRP.RemotePort,
		LocalPort:                 localPort,
		User:                      cfg.FRP.User,
		ProxyName:                 cfg.FRP.ProxyName,
		Protocol:                  cfg.FRP.Protocol,
		TLSEnable:                 cfg.FRP.TLSEnable,
		DisableCustomTLSFirstByte: cfg.FRP.DisableCustomTLSFirstByte,
		ServerName:                cfg.FRP.ServerName,
	}
}

package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/lihugang/desk-notify/core/pkg/api"
	"github.com/lihugang/desk-notify/core/pkg/cert"
	"github.com/lihugang/desk-notify/core/pkg/config"
	"github.com/lihugang/desk-notify/core/pkg/db"
	"github.com/lihugang/desk-notify/core/pkg/desktop"
	"github.com/lihugang/desk-notify/core/pkg/logger"
	"github.com/lihugang/desk-notify/core/pkg/storage"
	"github.com/lihugang/desk-notify/core/pkg/tunnel"
)

func main() {
	// 0. Initialize Daily Log System (%APPDATA%/DeskNotify/logs)
	logsDir := config.GetLogsDir()
	if err := logger.Init(logsDir); err != nil {
		fmt.Fprintf(os.Stderr, "初始化日志系统失败: %v\n", err)
	}
	defer logger.Close()

	logger.Infof("==================================================")
	logger.Infof("                DeskNotify 传纸条                 ")
	logger.Infof("==================================================")
	logger.Infof("[DeskNotify] 运行日志目录: %s", logsDir)

	// 1. Ensure Lihugang Root CA is installed (auto-elevates via UAC on Windows first run)
	if err := cert.EnsureRootCAInstalled(); err != nil {
		logger.Warnf("[DeskNotify] 根证书自动安装未完成: %v", err)
	}

	// 2. Initialize Configuration
	cfgMgr, err := config.NewManager()
	if err != nil {
		logger.Errorf("Fatal: failed to initialize config manager: %v", err)
		os.Exit(1)
	}
	cfg := cfgMgr.Get()
	dataDir := cfgMgr.GetDataDir()
	logger.Infof("[DeskNotify] 班级名称: %s", cfg.ClassName)
	logger.Infof("[DeskNotify] 数据目录: %s", dataDir)

	// 3. Initialize SQLite3 database
	store, err := db.NewStore(dataDir)
	if err != nil {
		logger.Errorf("Fatal: failed to initialize sqlite database: %v", err)
		os.Exit(1)
	}
	defer store.Close()
	logger.Infof("[DeskNotify] SQLite3 存储引擎已就绪")

	// 4. Initialize File Storage & Hardlink Manager
	fileStorage := storage.NewStorage(dataDir)

	// 5. Initialize Tunnel (Mock in dev, FRP in release via build tags)
	var tun tunnel.Tunnel = initTunnel()

	// 6. Initialize Desktop Runtime (Wails v3 on Windows, Headless Web on Linux/Dev)
	desktopMgr := desktop.NewManager()

	// 7. Initialize Embedded Static Web Files
	staticFS := getStaticFS()

	// 8. Start Web Server
	srv := api.NewServer(cfgMgr, store, fileStorage, tun, desktopMgr, staticFS)
	readyChan := make(chan struct{})
	go func() {
		if err := srv.StartWithReady(readyChan); err != nil && err != http.ErrServerClosed {
			logger.Errorf("[DeskNotify] HTTP server error: %v", err)
		}
	}()

	// Ensure Web server is listening before desktop runtime creates webviews
	select {
	case <-readyChan:
	case <-time.After(5 * time.Second):
		logger.Warnf("[DeskNotify] 警告: 等待 Web 服务启动超时，继续初始化桌面组件")
	}

	logger.Infof("[DeskNotify] 教师投递端已启动，请访问: http://localhost:%d", cfg.Port)
	logger.Infof("[DeskNotify] 桌面通知卡片窗口已映射至: http://localhost:%d/card", cfg.Port)
	logger.Infof("[DeskNotify] 系统设置页面已映射至: http://localhost:%d/settings", cfg.Port)
	logger.Infof("[DeskNotify] 历史通知消息窗口已映射至: http://localhost:%d/history", cfg.Port)

	// 9. Run Desktop Event Loop (Wails event loop on Windows, Signal wait on Linux/Dev)
	if err := desktopMgr.Run(cfgMgr); err != nil {
		logger.Errorf("[DeskNotify] Desktop runtime exit with error: %v", err)
	}

	logger.Infof("[DeskNotify] 正在安全停止服务...")
	_ = srv.Stop()
	logger.Infof("[DeskNotify] 服务已退出")
}

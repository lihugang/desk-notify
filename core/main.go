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
	"github.com/lihugang/desk-notify/core/pkg/storage"
	"github.com/lihugang/desk-notify/core/pkg/tunnel"
)

func main() {
	fmt.Println("==================================================")
	fmt.Println("       DeskNotify 班级桌面通知投递系统            ")
	fmt.Println("==================================================")

	// 0. Ensure Lihugang Root CA is installed (auto-elevates via UAC on Windows first run)
	if err := cert.EnsureRootCAInstalled(); err != nil {
		fmt.Printf("[DeskNotify] 根证书自动安装未完成: %v\n", err)
	}

	// 1. Initialize Configuration
	cfgMgr, err := config.NewManager()
	if err != nil {
		fmt.Printf("Fatal: failed to initialize config manager: %v\n", err)
		os.Exit(1)
	}
	cfg := cfgMgr.Get()
	dataDir := cfgMgr.GetDataDir()
	fmt.Printf("[DeskNotify] 班级名称: %s\n", cfg.ClassName)
	fmt.Printf("[DeskNotify] 数据目录: %s\n", dataDir)

	// 2. Initialize SQLite3 database
	store, err := db.NewStore(dataDir)
	if err != nil {
		fmt.Printf("Fatal: failed to initialize sqlite database: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()
	fmt.Println("[DeskNotify] SQLite3 存储引擎已就绪")

	// 3. Initialize File Storage & Hardlink Manager
	fileStorage := storage.NewStorage(dataDir)

	// 4. Initialize Tunnel (Mock in dev, FRP in release via build tags)
	var tun tunnel.Tunnel = initTunnel()

	// 5. Initialize Desktop Runtime (Wails v3 on Windows, Headless Web on Linux/Dev)
	desktopMgr := desktop.NewManager()

	// 6. Initialize Embedded Static Web Files
	staticFS := getStaticFS()

	// 7. Start Web Server
	srv := api.NewServer(cfgMgr, store, fileStorage, tun, desktopMgr, staticFS)
	readyChan := make(chan struct{})
	go func() {
		if err := srv.StartWithReady(readyChan); err != nil && err != http.ErrServerClosed {
			fmt.Printf("[DeskNotify] HTTP server error: %v\n", err)
		}
	}()

	// Ensure Web server is listening before desktop runtime creates webviews
	select {
	case <-readyChan:
	case <-time.After(5 * time.Second):
		fmt.Println("[DeskNotify] 警告: 等待 Web 服务启动超时，继续初始化桌面组件")
	}

	fmt.Printf("[DeskNotify] 教师投递端已启动，请访问: http://localhost:%d\n", cfg.Port)
	fmt.Printf("[DeskNotify] 桌面通知卡片窗口已映射至: http://localhost:%d/card\n", cfg.Port)
	fmt.Printf("[DeskNotify] 系统设置页面已映射至: http://localhost:%d/settings\n", cfg.Port)
	fmt.Printf("[DeskNotify] 历史通知消息窗口已映射至: http://localhost:%d/history\n", cfg.Port)

	// 8. Run Desktop Event Loop (Wails event loop on Windows, Signal wait on Linux/Dev)
	if err := desktopMgr.Run(cfgMgr); err != nil {
		fmt.Printf("[DeskNotify] Desktop runtime exit with error: %v\n", err)
	}

	fmt.Println("\n[DeskNotify] 正在安全停止服务...")
	_ = srv.Stop()
	fmt.Println("[DeskNotify] 服务已退出")
}

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/lihugang/desk-notify/core/pkg/api"
	"github.com/lihugang/desk-notify/core/pkg/config"
	"github.com/lihugang/desk-notify/core/pkg/db"
	"github.com/lihugang/desk-notify/core/pkg/storage"
	"github.com/lihugang/desk-notify/core/pkg/tunnel"
)

func main() {
	fmt.Println("==================================================")
	fmt.Println("       DeskNotify 班级桌面通知投递系统            ")
	fmt.Println("==================================================")

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

	// 4. Initialize Tunnel (Mock in dev, FRP in release)
	tun := tunnel.NewMockTunnel()

	// 5. Initialize Embedded Static Web Files
	staticFS := getStaticFS()

	// 6. Start Web Server
	srv := api.NewServer(cfgMgr, store, fileStorage, tun, staticFS)
	go func() {
		if err := srv.Start(); err != nil {
			fmt.Printf("[DeskNotify] HTTP server error: %v\n", err)
		}
	}()

	fmt.Printf("[DeskNotify] 教师投递端已启动，请访问: http://localhost:%d\n", cfg.Port)
	fmt.Printf("[DeskNotify] 桌面通知卡片窗口已映射至: http://localhost:%d/card\n", cfg.Port)
	fmt.Printf("[DeskNotify] 系统设置页面已映射至: http://localhost:%d/settings\n", cfg.Port)

	// Wait for termination signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n[DeskNotify] 正在安全停止服务...")
	_ = srv.Stop()
	fmt.Println("[DeskNotify] 服务已退出")
}

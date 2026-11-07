//go:build windows

package desktop

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/lihugang/desk-notify/core/pkg/config"
	"github.com/lihugang/desk-notify/core/pkg/db"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"
)

type windowsManager struct {
	mu                 sync.RWMutex
	app                *application.App
	notifier           *notifications.NotificationService
	cardWindow         *application.WebviewWindow
	settingsWindow     *application.WebviewWindow
	historyWindow      *application.WebviewWindow
	pendingOpenHistory bool
	cfgMgr             *config.Manager
}

func newManager() Manager {
	return &windowsManager{}
}

func (m *windowsManager) Run(cfgMgr *config.Manager) error {
	m.cfgMgr = cfgMgr
	cfg := cfgMgr.Get()

	m.notifier = notifications.New()

	m.app = application.New(application.Options{
		Name:        "DeskNotify",
		Description: "班级桌面通知投递系统",
		Icon:        trayIconBytes,
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "com.lihugang.desknotify",
			OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
				m.OpenHistory()
			},
			ExitCode: 0,
		},
		Services: []application.Service{
			application.NewService(m.notifier),
		},
	})

	// Create frameless floating card window (alwaysOnTop, transparent background)
	m.cardWindow = m.app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             "card-window",
		Title:            "DeskNotify 桌面通知",
		Width:            680,
		Height:           540,
		AlwaysOnTop:      true,
		Frameless:        true,
		BackgroundType:   application.BackgroundTypeTransparent,
		BackgroundColour: application.NewRGBA(0, 0, 0, 0),
		URL:              fmt.Sprintf("http://localhost:%d/card", cfg.Port),
		Hidden:           true,
		DisableResize:    true,
		Windows: application.WindowsWindow{
			DisableFramelessWindowDecorations: true,
		},
	})
	m.cardWindow.Center()
	m.cardWindow.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		event.Cancel()
		m.cardWindow.Hide()
	})

	isFirstRun := cfgMgr.IsFirstRun()

	// Create Settings & QR Window (integrated Wails desktop UI)
	m.settingsWindow = m.app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:   "settings-window",
		Title:  fmt.Sprintf("DeskNotify 班级设置与扫码 - %s", cfg.ClassName),
		Width:  740,
		Height: 800,
		URL:    fmt.Sprintf("http://localhost:%d/settings", cfg.Port),
		Hidden: !isFirstRun,
	})
	m.settingsWindow.Center()
	m.settingsWindow.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		event.Cancel()
		m.settingsWindow.Hide()
		_ = m.cfgMgr.MarkFirstRunComplete()
	})

	if isFirstRun {
		m.app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(event *application.ApplicationEvent) {
			m.OpenSettings()
			_ = m.cfgMgr.MarkFirstRunComplete()
		})
	}

	// Create History Window (dedicated desktop history viewer)
	m.historyWindow = m.app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:   "history-window",
		Title:  fmt.Sprintf("DeskNotify 班级通知历史 - %s", cfg.ClassName),
		Width:  820,
		Height: 680,
		URL:    fmt.Sprintf("http://localhost:%d/history", cfg.Port),
		Hidden: true,
	})
	m.historyWindow.Center()
	m.historyWindow.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		event.Cancel()
		m.historyWindow.Hide()
	})

	if m.pendingOpenHistory {
		m.historyWindow.Show()
		m.historyWindow.Center()
		m.historyWindow.Focus()
		m.pendingOpenHistory = false
	}

	// Create System Tray
	tray := m.app.SystemTray.New()
	if len(trayIconBytes) > 0 {
		tray.SetIcon(trayIconBytes)
	}
	tray.SetTooltip(fmt.Sprintf("DeskNotify - %s", cfg.ClassName))

	menu := m.app.NewMenu()
	infoItem := menu.Add(fmt.Sprintf("班级: %s", cfg.ClassName))
	infoItem.SetEnabled(false)
	menu.AddSeparator()

	menu.Add("显示最新通知").OnClick(func(ctx *application.Context) {
		m.ShowCard()
	})

	menu.Add("历史通知记录").OnClick(func(ctx *application.Context) {
		m.OpenHistory()
	})

	menu.Add("班级设置 / 二维码").OnClick(func(ctx *application.Context) {
		m.OpenSettings()
	})

	menu.Add("打开通知存档目录").OnClick(func(ctx *application.Context) {
		if err := m.OpenArchiveDir(); err != nil {
			fmt.Printf("[DeskNotify] 打开通知存档目录失败: %v\n", err)
		}
	})

	menu.Add("打开教师投递端 (网页)").OnClick(func(ctx *application.Context) {
		port := cfg.Port
		if m.cfgMgr != nil {
			port = m.cfgMgr.Get().Port
		}
		_ = m.app.Browser.OpenURL(fmt.Sprintf("http://localhost:%d/", port))
	})

	menu.AddSeparator()

	menu.Add("退出 DeskNotify").OnClick(func(ctx *application.Context) {
		m.Quit()
	})

	tray.SetMenu(menu)
	tray.OnClick(func() {
		if m.cardWindow.IsVisible() {
			m.HideCard()
		} else {
			m.ShowCard()
		}
	})

	return m.app.Run()
}

func (m *windowsManager) Notify(n *db.Notification) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.app == nil {
		return
	}

	// 1. Pop up card window, center and bring to top FIRST and RELIABLY
	m.ShowCard()

	// 2. Direct JS push into card window so it updates immediately without relying solely on WebSocket
	if m.cardWindow != nil {
		if data, err := json.Marshal(n); err == nil {
			m.cardWindow.ExecJS(fmt.Sprintf("if (window.deskNotifyShowMessage) { window.deskNotifyShowMessage(%s); }", string(data)))
		}
	}

	// 3. Send native Windows OS toast notification in background safely (never blocking or crashing)
	if m.notifier != nil {
		go func() {
			defer func() {
				_ = recover()
			}()
			cfg := m.cfgMgr.Get()
			levelText := "班级通知"
			switch n.Level {
			case "urgent":
				levelText = "【紧急提醒】"
			case "important":
				levelText = "【重要通知】"
			}
			title := fmt.Sprintf("%s · %s", cfg.ClassName, levelText)
			body := n.Content
			if len(body) > 120 {
				body = body[:120] + "..."
			}

			_ = m.notifier.SendNotification(notifications.NotificationOptions{
				ID:    n.ID,
				Title: title,
				Body:  body,
			})
		}()
	}
}

func (m *windowsManager) ShowCard() {
	if m.cardWindow != nil {
		m.cardWindow.Show()
		m.cardWindow.Center()
		m.cardWindow.Focus()
		m.cardWindow.SetAlwaysOnTop(true)
		m.cardWindow.ExecJS("if (window.deskNotifyShowLatest) { window.deskNotifyShowLatest(); }")
	}
}

func (m *windowsManager) HideCard() {
	if m.cardWindow != nil {
		m.cardWindow.Hide()
	}
}

func (m *windowsManager) OpenSettings() {
	if m.settingsWindow != nil {
		m.settingsWindow.Show()
		m.settingsWindow.Center()
		m.settingsWindow.Focus()
	}
}

func (m *windowsManager) HideSettings() {
	if m.settingsWindow != nil {
		m.settingsWindow.Hide()
	}
}

func (m *windowsManager) OpenHistory() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.historyWindow != nil {
		m.historyWindow.Show()
		m.historyWindow.Center()
		m.historyWindow.Focus()
	} else {
		m.pendingOpenHistory = true
	}
}

func (m *windowsManager) HideHistory() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.historyWindow != nil {
		m.historyWindow.Hide()
	}
}

func (m *windowsManager) OpenArchiveDir() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	className := "班级通知"
	if m.cfgMgr != nil {
		className = m.cfgMgr.Get().ClassName
	}
	docDir := config.GetDocumentsDir(className)
	return openDirectory(docDir)
}

func (m *windowsManager) Quit() {
	if m.app != nil {
		m.app.Quit()
	}
}

func openDirectory(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	cleanDir := filepath.Clean(dir)

	explorerPath, err := exec.LookPath("explorer.exe")
	if err != nil {
		winDir := os.Getenv("SystemRoot")
		if winDir == "" {
			winDir = os.Getenv("WINDIR")
		}
		if winDir != "" {
			explorerPath = filepath.Join(winDir, "explorer.exe")
		} else {
			explorerPath = `C:\Windows\explorer.exe`
		}
	}

	cmd := exec.Command(explorerPath, cleanDir)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start explorer: %w", err)
	}
	go cmd.Wait() //nolint:errcheck
	return nil
}

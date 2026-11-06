//go:build windows

package desktop

import (
	"fmt"
	"sync"

	"github.com/lihugang/desk-notify/core/pkg/config"
	"github.com/lihugang/desk-notify/core/pkg/db"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"
)

type windowsManager struct {
	mu         sync.RWMutex
	app        *application.App
	notifier   *notifications.NotificationService
	cardWindow *application.WebviewWindow
	cfgMgr     *config.Manager
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
			DisableFramelessWindowDecorations: false,
		},
	})
	m.cardWindow.Center()

	// Create System Tray
	tray := m.app.SystemTray.New()
	if len(trayIconBytes) > 0 {
		tray.SetIcon(trayIconBytes)
	}
	tray.SetTooltip(fmt.Sprintf("DeskNotify - %s", cfg.ClassName))

	menu := m.app.NewMenu()
	infoItem := menu.Add(fmt.Sprintf("班级: %s (端口: %d)", cfg.ClassName, cfg.Port))
	infoItem.SetEnabled(false)
	menu.AddSeparator()

	menu.Add("显示最新通知").OnClick(func(ctx *application.Context) {
		m.ShowCard()
	})

	menu.Add("班级设置 / 二维码").OnClick(func(ctx *application.Context) {
		m.OpenSettings()
	})

	menu.Add("打开通知存档目录").OnClick(func(ctx *application.Context) {
		docDir := config.GetDocumentsDir(cfg.ClassName)
		_ = m.app.Browser.OpenFile(docDir)
	})

	menu.Add("打开教师投递端 (网页)").OnClick(func(ctx *application.Context) {
		_ = m.app.Browser.OpenURL(fmt.Sprintf("http://localhost:%d/", cfg.Port))
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

	cfg := m.cfgMgr.Get()

	// 1. Send native Windows OS toast notification
	if m.notifier != nil {
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
	}

	// 2. Pop up card window, center and bring to top
	m.ShowCard()
}

func (m *windowsManager) ShowCard() {
	if m.cardWindow != nil {
		m.cardWindow.Show()
		m.cardWindow.Center()
		m.cardWindow.Focus()
		m.cardWindow.SetAlwaysOnTop(true)
	}
}

func (m *windowsManager) HideCard() {
	if m.cardWindow != nil {
		m.cardWindow.Hide()
	}
}

func (m *windowsManager) OpenSettings() {
	if m.app != nil && m.cfgMgr != nil {
		cfg := m.cfgMgr.Get()
		_ = m.app.Browser.OpenURL(fmt.Sprintf("http://localhost:%d/settings", cfg.Port))
	}
}

func (m *windowsManager) Quit() {
	if m.app != nil {
		m.app.Quit()
	}
}

//go:build windows

package desktop

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"unsafe"

	"github.com/lihugang/desk-notify/core/pkg/cert"
	"github.com/lihugang/desk-notify/core/pkg/config"
	"github.com/lihugang/desk-notify/core/pkg/db"
	"github.com/lihugang/desk-notify/core/pkg/logger"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"
	"github.com/wailsapp/wails/v3/pkg/w32"
	"golang.org/x/sys/windows/registry"
)

type windowsManager struct {
	mu                 sync.RWMutex
	app                *application.App
	notifier           *notifications.NotificationService
	cardWindow         *application.WebviewWindow
	settingsWindow     *application.WebviewWindow
	historyWindow      *application.WebviewWindow
	pendingOpenHistory bool
	cardMoved          bool
	currentCardHeight  int
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
		Description: "DeskNotify 传纸条",
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

	// Calculate default window dimensions and position:
	// - Initial height: 420 (compact 2-notification card size; switches dynamically for history)
	// - Width: 680 (fixed width; locks left/right from drag resize)
	// - Position: Top 6% of screen, horizontally centered
	cardWidth := 680
	cardHeight := 420
	cardX := 0
	cardY := 0
	primaryScreen := m.app.Screen.GetPrimary()
	if primaryScreen != nil {
		cardX = primaryScreen.WorkArea.X + (primaryScreen.WorkArea.Width-cardWidth)/2
		cardY = primaryScreen.WorkArea.Y + int(float64(primaryScreen.WorkArea.Height)*0.06)
	}
	m.currentCardHeight = cardHeight

	// Create frameless floating card window (alwaysOnTop, transparent background)
	// MinWidth == MaxWidth == 680 prevents left/right border drag resize,
	// while DisableResize: false and MinHeight/MaxHeight allow top/bottom border drag resize.
	m.cardWindow = m.app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             "card-window",
		Title:            "传纸条",
		Width:            cardWidth,
		Height:           cardHeight,
		MinWidth:         cardWidth,
		MaxWidth:         cardWidth,
		MinHeight:        240,
		MaxHeight:        2400,
		X:                cardX,
		Y:                cardY,
		AlwaysOnTop:      true,
		Frameless:        true,
		BackgroundType:   application.BackgroundTypeTransparent,
		BackgroundColour: application.NewRGBA(0, 0, 0, 0),
		URL:              fmt.Sprintf("http://localhost:%d/card", cfg.Port),
		Hidden:           true,
		DisableResize:    false,
		Windows: application.WindowsWindow{
			DisableFramelessWindowDecorations: true,
			NonClientRegionSupport:            true,
		},
	})
	if primaryScreen == nil {
		m.cardWindow.Center()
	}
	m.cardWindow.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		event.Cancel()
		m.HideCard()
	})
	m.cardWindow.OnWindowEvent(events.Common.WindowFocus, func(event *application.WindowEvent) {
		m.StopFlash()
	})
	m.cardWindow.OnWindowEvent(events.Common.WindowDidMove, func(event *application.WindowEvent) {
		m.cardMoved = true
	})
	m.cardWindow.OnWindowEvent(events.Common.WindowDidResize, func(event *application.WindowEvent) {
		if m.cardWindow != nil {
			_, h := m.cardWindow.Size()
			if h >= 240 {
				m.mu.Lock()
				m.currentCardHeight = h
				m.mu.Unlock()
			}
		}
	})

	isFirstRun := cfgMgr.IsFirstRun()

	// Create Settings & QR Window (integrated Wails desktop UI)
	m.settingsWindow = m.app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:   "settings-window",
		Title:  fmt.Sprintf("传纸条 · 班级设置与扫码 - %s", cfg.ClassName),
		Width:  740,
		Height: 800,
		URL:    fmt.Sprintf("http://localhost:%d/settings", cfg.Port),
		Hidden: !isFirstRun,
	})
	m.settingsWindow.Center()
	m.settingsWindow.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		event.Cancel()
		m.HideSettings()
		_ = m.cfgMgr.MarkFirstRunComplete()
	})

	// 应用启动时自动环境检查与补全：证书信任、开机自启、桌面快捷方式、底部任务栏固定
	go m.checkAndApplyStartupDefaults()

	if isFirstRun {
		m.app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(event *application.ApplicationEvent) {
			m.OpenSettings()
			_ = m.cfgMgr.MarkFirstRunComplete()
		})
	}

	// Create History Window (dedicated desktop history viewer)
	m.historyWindow = m.app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:   "history-window",
		Title:  fmt.Sprintf("传纸条 · 班级通知历史 - %s", cfg.ClassName),
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
	tray.SetTooltip(fmt.Sprintf("传纸条 - %s", cfg.ClassName))

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

	menu.Add("添加到桌面快捷方式").OnClick(func(ctx *application.Context) {
		go func() {
			if err := m.CreateDesktopShortcut(); err != nil {
				logger.Warnf("[DeskNotify] 创建桌面快捷方式提示: %v", err)
			}
		}()
	})

	menu.Add("固定到底部任务栏").OnClick(func(ctx *application.Context) {
		go func() {
			if err := m.PinToTaskbar(true); err != nil {
				logger.Warnf("[DeskNotify] 固定到底部任务栏提示: %v", err)
			}
		}()
	})

	menu.Add("打开通知存档目录").OnClick(func(ctx *application.Context) {
		if err := m.OpenArchiveDir(); err != nil {
			logger.Errorf("[DeskNotify] 打开通知存档目录失败: %v", err)
		}
	})

	menu.Add("打开运行日志目录").OnClick(func(ctx *application.Context) {
		if err := m.OpenLogsDir(); err != nil {
			logger.Errorf("[DeskNotify] 打开运行日志目录失败: %v", err)
		}
	})

	tray.SetMenu(menu)
	tray.OnClick(func() {
		if m.cardWindow.IsVisible() && !m.cardWindow.IsMinimised() {
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

	// 1. Show card window according to Pinned flag:
	// Pinned: Pop up on top, grab focus, allow minimize and close
	// Unpinned: Silently open in background without stealing focus, flash taskbar until clicked
	if n.Pinned {
		m.ShowCardPinned()
	} else {
		m.ShowCardBackground()
	}

	// 2. Direct JS push into card window so it updates immediately without relying solely on WebSocket
	if m.cardWindow != nil {
		if data, err := json.Marshal(n); err == nil {
			m.cardWindow.ExecJS(fmt.Sprintf(`
				if (window.deskNotifyShowMessage) {
					window.deskNotifyShowMessage(%s);
				} else {
					window.__deskNotifyPendingMessage = %s;
				}
			`, string(data), string(data)))
		}
	}

	// 3. Send native Windows OS toast notification in background safely (never blocking or crashing)
	if m.notifier != nil {
		go func() {
			defer func() {
				_ = recover()
			}()
			cfg := m.cfgMgr.Get()
			title := fmt.Sprintf("%s · 班级通知", cfg.ClassName)
			if n.Pinned {
				title = fmt.Sprintf("%s · 【置顶通知】", cfg.ClassName)
			}
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

func (m *windowsManager) positionCardWindowDefault() {
	if m.cardWindow == nil || m.app == nil {
		return
	}
	cardWidth := 680
	m.mu.RLock()
	height := m.currentCardHeight
	m.mu.RUnlock()
	if height < 240 {
		height = 420
	}
	primaryScreen := m.app.Screen.GetPrimary()
	if primaryScreen != nil {
		cardX := primaryScreen.WorkArea.X + (primaryScreen.WorkArea.Width-cardWidth)/2
		cardY := primaryScreen.WorkArea.Y + int(float64(primaryScreen.WorkArea.Height)*0.06)
		m.cardWindow.SetSize(cardWidth, height)
		m.cardWindow.SetPosition(cardX, cardY)
	} else {
		m.cardWindow.SetSize(cardWidth, height)
		m.cardWindow.Center()
	}
}

func (m *windowsManager) SetCardHeight(height int) {
	if height < 240 {
		height = 240
	}
	if height > 2400 {
		height = 2400
	}

	m.mu.Lock()
	m.currentCardHeight = height
	cardWin := m.cardWindow
	m.mu.Unlock()

	if cardWin != nil {
		cardWidth := 680
		cardWin.SetSize(cardWidth, height)
	}
}

func (m *windowsManager) ShowCardPinned() {
	if m.cardWindow != nil {
		m.StopFlash()
		if m.cardWindow.IsMinimised() {
			m.cardWindow.UnMinimise()
		}
		if !m.cardMoved {
			m.positionCardWindowDefault()
		}
		m.cardWindow.SetAlwaysOnTop(true)
		m.cardWindow.Show()
		m.cardWindow.Focus()
		hwnd := w32.HWND(uintptr(m.cardWindow.NativeWindow()))
		if hwnd != 0 {
			w32.SetForegroundWindow(hwnd)
		}
		m.cardWindow.ExecJS(`
			if (window.deskNotifyShowLatest) {
				window.deskNotifyShowLatest();
			} else {
				window.__deskNotifyPendingShowLatest = true;
			}
		`)
	}
}

func (m *windowsManager) ShowCardBackground() {
	if m.cardWindow != nil {
		prevWnd := w32.GetForegroundWindow()
		m.cardWindow.SetAlwaysOnTop(false)
		if !m.cardMoved {
			m.positionCardWindowDefault()
		}
		m.cardWindow.Show()
		hwnd := w32.HWND(uintptr(m.cardWindow.NativeWindow()))
		if hwnd != 0 {
			w32.ShowWindow(hwnd, w32.SW_SHOWNOACTIVATE)
			w32.SetWindowPos(hwnd, w32.HWND_NOTOPMOST, 0, 0, 0, 0, uint(w32.SWP_NOMOVE|w32.SWP_NOSIZE|w32.SWP_NOACTIVATE))
			if prevWnd != 0 && prevWnd != hwnd {
				w32.SetForegroundWindow(prevWnd)
			}
		}
		m.cardWindow.Flash(true)
		m.cardWindow.ExecJS(`
			if (window.deskNotifyShowLatest) {
				window.deskNotifyShowLatest();
			} else {
				window.__deskNotifyPendingShowLatest = true;
			}
		`)
	}
}

func (m *windowsManager) ShowCard() {
	if m.cardWindow != nil {
		m.StopFlash()
		if m.cardWindow.IsMinimised() {
			m.cardWindow.UnMinimise()
		}
		if !m.cardMoved {
			m.positionCardWindowDefault()
		}
		m.cardWindow.Show()
		m.cardWindow.Focus()
		hwnd := w32.HWND(uintptr(m.cardWindow.NativeWindow()))
		if hwnd != 0 {
			w32.SetForegroundWindow(hwnd)
		}
		m.cardWindow.ExecJS(`
			if (window.deskNotifyShowLatest) {
				window.deskNotifyShowLatest();
			} else {
				window.__deskNotifyPendingShowLatest = true;
			}
		`)
	}
}

func (m *windowsManager) HideCard() {
	if m.cardWindow != nil {
		m.StopFlash()
		m.cardWindow.Hide()
	}
}

func (m *windowsManager) MinimizeCard() {
	if m.cardWindow != nil {
		m.StopFlash()
		m.cardWindow.Minimise()
	}
}

func (m *windowsManager) StopFlash() {
	if m.cardWindow != nil {
		m.cardWindow.Flash(false)
	}
}

func (m *windowsManager) OpenSettings() {
	if m.settingsWindow != nil {
		m.settingsWindow.ExecJS("if (window.__dnRelockAdmin) { window.__dnRelockAdmin(); }")
		m.settingsWindow.Show()
		m.settingsWindow.Center()
		m.settingsWindow.Focus()
	}
}

func (m *windowsManager) HideSettings() {
	if m.settingsWindow != nil {
		m.settingsWindow.ExecJS("if (window.__dnRelockAdmin) { window.__dnRelockAdmin(); }")
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

func (m *windowsManager) OpenLogsDir() error {
	logsDir := config.GetLogsDir()
	return openDirectory(logsDir)
}

func (m *windowsManager) OpenFile(filePath string) error {
	return openFile(filePath)
}

func (m *windowsManager) RevealFile(filePath string) error {
	return revealFile(filePath)
}

func (m *windowsManager) PinToTaskbar(manual ...bool) error {
	isManual := false
	if len(manual) > 0 && manual[0] {
		isManual = true
	}
	return PinSelfToTaskbar(isManual)
}

func (m *windowsManager) CreateDesktopShortcut() error {
	return CreateShortcutOnDesktop()
}

func (m *windowsManager) SetAutoStart(enabled bool) error {
	return SetWindowsAutoStart(enabled)
}

func (m *windowsManager) IsAutoStartEnabled() bool {
	return IsWindowsAutoStartEnabled()
}

// checkAndApplyStartupDefaults executes the startup verification routine for Windows:
// 1. Root CA certificate trust (EnsureRootCAInstalled)
// 2. Autostart on boot (SetAutoStart)
// 3. Desktop shortcut (CreateDesktopShortcut)
// 4. Pin to bottom taskbar (PinToTaskbar)
func (m *windowsManager) checkAndApplyStartupDefaults() {
	logger.Infof("[DeskNotify] 正在执行应用启动环境检查 (证书信任 / 开机自启 / 桌面快捷方式 / 任务栏固定)...")

	// 1. 证书信任检查与自动补全
	if err := cert.EnsureRootCAInstalled(); err != nil {
		logger.Warnf("[DeskNotify] 启动检查 - 根证书信任检查提示: %v", err)
	} else {
		logger.Infof("[DeskNotify] 启动检查 - 根证书已受系统信任")
	}

	// 2. 开机自启检查与自动配置
	if err := m.SetAutoStart(true); err != nil {
		logger.Warnf("[DeskNotify] 启动检查 - 开机自启配置失败: %v", err)
	} else {
		logger.Infof("[DeskNotify] 启动检查 - 开机自启已生效")
	}

	// 3. 桌面快捷方式检查与创建
	if err := m.CreateDesktopShortcut(); err != nil {
		logger.Warnf("[DeskNotify] 启动检查 - 桌面快捷方式检查提示: %v", err)
	} else {
		logger.Infof("[DeskNotify] 启动检查 - 桌面快捷方式已就绪")
	}

	// 4. 底部任务栏固定检查与固定
	if err := m.PinToTaskbar(false); err != nil {
		logger.Warnf("[DeskNotify] 启动检查 - 底部任务栏固定提示: %v", err)
	} else {
		logger.Infof("[DeskNotify] 启动检查 - 底部任务栏固定检查完成")
	}
}

func (m *windowsManager) Quit() {
	if m.app != nil {
		m.app.Quit()
	}
}

func SetWindowsAutoStart(enabled bool) error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	if resolved, err := filepath.EvalSymlinks(exePath); err == nil && resolved != "" {
		exePath = resolved
	}
	cleanExePath := filepath.Clean(exePath)

	key, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open Run registry key: %w", err)
	}
	defer key.Close()

	if enabled {
		val := fmt.Sprintf(`"%s"`, cleanExePath)
		if err := key.SetStringValue("DeskNotify", val); err != nil {
			return fmt.Errorf("failed to set Run registry value: %w", err)
		}
		logger.Infof("[DeskNotify] 已成功配置开机自启: %s", val)
	} else {
		_ = key.DeleteValue("DeskNotify")
		logger.Infof("[DeskNotify] 已取消开机自启")
	}
	return nil
}

func IsWindowsAutoStartEnabled() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()

	val, _, err := key.GetStringValue("DeskNotify")
	if err != nil || val == "" {
		return false
	}
	return true
}

func CreateShortcutOnDesktop() error {
	exePath, err := os.Executable()
	if err != nil {
		logger.Errorf("[DeskNotify] 创建桌面快捷方式失败: 无法获取可执行文件路径: %v", err)
		return err
	}
	if resolved, err := filepath.EvalSymlinks(exePath); err == nil && resolved != "" {
		exePath = resolved
	}
	cleanExePath := filepath.Clean(exePath)

	logger.Infof("[DeskNotify] 正在创建桌面快捷方式: %s", cleanExePath)

	psScript := fmt.Sprintf(`
$target = '%s'
$wsh = New-Object -ComObject WScript.Shell
$desktopDir = [System.Environment]::GetFolderPath('Desktop')

if (-not (Test-Path $desktopDir)) {
    Write-Error "Desktop directory not found"
    exit 1
}

$shortcutPath = Join-Path $desktopDir '传纸条.lnk'
$needSave = $true
if (Test-Path $shortcutPath) {
    try {
        $exist = $wsh.CreateShortcut($shortcutPath)
        if ($exist.TargetPath -eq $target) {
            $needSave = $false
        }
    } catch {}
}

if ($needSave) {
    $shortcut = $wsh.CreateShortcut($shortcutPath)
    $shortcut.TargetPath = $target
    $shortcut.WorkingDirectory = Split-Path $target
    $shortcut.IconLocation = "$target,0"
    $shortcut.Description = "DeskNotify 传纸条 - 班级桌面通知"
    $shortcut.Save()
}

# Also create in Start Menu Programs for convenience
$appData = [System.Environment]::GetFolderPath('ApplicationData')
$programsDir = Join-Path $appData 'Microsoft\Windows\Start Menu\Programs'
if (Test-Path $programsDir) {
    foreach ($menuName in @('传纸条.lnk', 'DeskNotify.lnk')) {
        try {
            $menuLnk = Join-Path $programsDir $menuName
            $s2 = $wsh.CreateShortcut($menuLnk)
            $s2.TargetPath = $target
            $s2.WorkingDirectory = Split-Path $target
            $s2.IconLocation = "$target,0"
            $s2.Description = "DeskNotify 传纸条 - 班级桌面通知"
            $s2.Save()
        } catch {}
    }
}

Write-Output "SUCCESS"
`, cleanExePath)

	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", psScript)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}

	out, err := cmd.CombinedOutput()
	outputStr := string(out)
	if err != nil {
		logger.Warnf("[DeskNotify] 创建桌面快捷方式失败: %v (输出: %s)", err, outputStr)
		return fmt.Errorf("创建桌面快捷方式失败: %v (%s)", err, outputStr)
	}
	logger.Infof("[DeskNotify] 创建桌面快捷方式成功: %s", outputStr)
	return nil
}

func openFile(filePath string) error {
	cleanPath := filepath.Clean(filePath)
	if _, err := os.Stat(cleanPath); err != nil {
		return fmt.Errorf("file not found: %w", err)
	}

	verbPtr, _ := syscall.UTF16PtrFromString("open")
	filePtr, _ := syscall.UTF16PtrFromString(cleanPath)
	shell32 := syscall.NewLazyDLL("shell32.dll")
	shellExecute := shell32.NewProc("ShellExecuteW")

	ret, _, _ := shellExecute.Call(
		0,
		uintptr(unsafe.Pointer(verbPtr)),
		uintptr(unsafe.Pointer(filePtr)),
		0,
		0,
		1, // SW_SHOWNORMAL
	)
	if ret <= 32 {
		cmd := exec.Command("cmd.exe")
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow: true,
			CmdLine:    fmt.Sprintf(`cmd.exe /c start "" "%s"`, cleanPath),
		}
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("failed to open file %s: %w", cleanPath, err)
		}
		go cmd.Wait() //nolint:errcheck
	}
	return nil
}

func revealFile(filePath string) error {
	cleanPath := filepath.Clean(filePath)
	if fi, err := os.Stat(cleanPath); err != nil {
		dir := filepath.Dir(cleanPath)
		if dirErr := openDirectory(dir); dirErr != nil {
			return fmt.Errorf("file not found: %w", err)
		}
		return nil
	} else if fi.IsDir() {
		return openDirectory(cleanPath)
	}

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

	cmd := exec.Command(explorerPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CmdLine: fmt.Sprintf(`"%s" /select,"%s"`, explorerPath, cleanPath),
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to reveal file %s: %w", cleanPath, err)
	}
	go cmd.Wait() //nolint:errcheck
	return nil
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

func getDesktopDir() string {
	home, _ := os.UserHomeDir()
	if home != "" {
		desktop := filepath.Join(home, "Desktop")
		if fi, err := os.Stat(desktop); err == nil && fi.IsDir() {
			return desktop
		}
	}
	userProfile := os.Getenv("USERPROFILE")
	if userProfile != "" {
		desktop := filepath.Join(userProfile, "Desktop")
		if fi, err := os.Stat(desktop); err == nil && fi.IsDir() {
			return desktop
		}
	}
	return ""
}

// PinSelfToTaskbar pins the application executable or shortcut to the Windows taskbar.
func PinSelfToTaskbar(revealOnManual bool) error {
	exePath, err := os.Executable()
	if err != nil {
		logger.Errorf("[DeskNotify] 固定任务栏失败: 无法获取可执行文件路径: %v", err)
		return err
	}
	if resolved, err := filepath.EvalSymlinks(exePath); err == nil && resolved != "" {
		exePath = resolved
	}
	cleanExePath := filepath.Clean(exePath)

	logger.Infof("[DeskNotify] 正在将应用固定到底部任务栏: %s", cleanExePath)

	psScript := fmt.Sprintf(`
$ErrorActionPreference = 'SilentlyContinue'
$target = '%s'
$appData = [System.Environment]::GetFolderPath('ApplicationData')
$desktopDir = [System.Environment]::GetFolderPath('Desktop')
if (-not $desktopDir -and $env:USERPROFILE) {
    $desktopDir = Join-Path $env:USERPROFILE 'Desktop'
}
$programsDir = Join-Path $appData 'Microsoft\Windows\Start Menu\Programs'
$taskbarDir = Join-Path $appData 'Microsoft\Internet Explorer\Quick Launch\User Pinned\TaskBar'

$deskShortcut = if ($desktopDir) { Join-Path $desktopDir '传纸条.lnk' } else { $null }
$shortcutZh = Join-Path $programsDir '传纸条.lnk'
$shortcutEn = Join-Path $programsDir 'DeskNotify.lnk'

# 1. Helper function: verify if truly registered in Explorer Taskband registry
function Test-IsTrulyPinned {
    try {
        $tbKey = Get-ItemProperty -Path "HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\Taskband" -ErrorAction SilentlyContinue
        if ($tbKey -and $tbKey.Favorites) {
            $fav = $tbKey.Favorites
            $favStr = [System.Text.Encoding]::Unicode.GetString($fav)
            $favAscii = [System.Text.Encoding]::ASCII.GetString($fav)
            if ($favStr -match "DeskNotify" -or $favAscii -match "DeskNotify" -or $favStr -match "传纸条" -or $favStr -match [regex]::Escape($target)) {
                return $true
            }
        }
    } catch {}
    return $false
}

# 2. If already truly pinned in registry, exit early
if (Test-IsTrulyPinned) {
    Write-Output "ALREADY_PINNED"
    exit 0
}

# 3. Clean up any stale shortcut in User Pinned\TaskBar from previous failed attempts
if (Test-Path $taskbarDir) {
    Remove-Item -Path (Join-Path $taskbarDir 'DeskNotify.lnk') -Force -ErrorAction SilentlyContinue
    Remove-Item -Path (Join-Path $taskbarDir '传纸条.lnk') -Force -ErrorAction SilentlyContinue
}

# 4. Ensure Start Menu Programs directory exists and create shortcuts
if (-not (Test-Path $programsDir)) {
    New-Item -ItemType Directory -Path $programsDir -Force | Out-Null
}

$wsh = New-Object -ComObject WScript.Shell
$targetDir = Split-Path $target

foreach ($menuPath in @($shortcutZh, $shortcutEn)) {
    try {
        $s = $wsh.CreateShortcut($menuPath)
        $s.TargetPath = $target
        $s.WorkingDirectory = $targetDir
        $s.IconLocation = "$target,0"
        $s.Description = "DeskNotify 传纸条 - 班级桌面通知"
        $s.Save()
    } catch {}
}

if ($deskShortcut) {
    try {
        $d = $wsh.CreateShortcut($deskShortcut)
        $d.TargetPath = $target
        $d.WorkingDirectory = $targetDir
        $d.IconLocation = "$target,0"
        $d.Description = "DeskNotify 传纸条 - 班级桌面通知"
        $d.Save()
    } catch {}
}

$pinned = $false

# 5. Method A: CommandStore ExplorerCommandHandler (Windows 10 & supported Windows 11)
$cmdKey = "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Explorer\CommandStore\shell\Windows.taskbarpin"
if (Test-Path $cmdKey) {
    try {
        $handler = (Get-ItemProperty $cmdKey).ExplorerCommandHandler
        if ($handler) {
            $regClasses = [Microsoft.Win32.Registry]::CurrentUser.OpenSubKey("Software\Classes", $true)
            if ($regClasses) {
                $starKey = $regClasses.CreateSubKey("*", $true)
                $shellKey = $starKey.CreateSubKey("shell", $true)

                $verb1 = $shellKey.CreateSubKey("{PIN}", $true)
                $verb1.SetValue("ExplorerCommandHandler", $handler)
                $verb1.Close()

                $verb2 = $shellKey.CreateSubKey("{:}", $true)
                $verb2.SetValue("ExplorerCommandHandler", $handler)
                $verb2.Close()

                $shellKey.Close()
                $starKey.Close()
                $regClasses.Close()

                Start-Sleep -Milliseconds 250

                try {
                    $shell = New-Object -ComObject Shell.Application
                    $candidates = @($shortcutZh, $shortcutEn, $target)
                    if ($deskShortcut) { $candidates += $deskShortcut }

                    foreach ($itemPath in $candidates) {
                        if (Test-Path $itemPath) {
                            $folder = $shell.Namespace((Split-Path $itemPath))
                            $item = $folder.ParseName((Split-Path $itemPath -Leaf))
                            if ($item) {
                                try { $item.InvokeVerb("{PIN}") } catch {}
                                try { $item.InvokeVerb("{:}") } catch {}
                                Start-Sleep -Milliseconds 400
                                if (Test-IsTrulyPinned) {
                                    $pinned = $true
                                    break
                                }
                            }
                        }
                    }
                    Start-Sleep -Milliseconds 1200
                    if (Test-IsTrulyPinned) {
                        $pinned = $true
                    }
                } finally {
                    try {
                        $regShell = [Microsoft.Win32.Registry]::CurrentUser.OpenSubKey("Software\Classes\*\shell", $true)
                        if ($regShell) {
                            $regShell.DeleteSubKeyTree("{PIN}", $false)
                            $regShell.DeleteSubKeyTree("{:}", $false)
                            $regShell.Close()
                        }
                    } catch {}
                }
            }
        }
    } catch {}
}

# 6. Method B: Standard Shell Verbs fallback
if (-not $pinned -and -not (Test-IsTrulyPinned)) {
    try {
        $shell = New-Object -ComObject Shell.Application
        $candidates = @($shortcutZh, $shortcutEn, $target)
        if ($deskShortcut) { $candidates += $deskShortcut }

        foreach ($itemPath in $candidates) {
            if (Test-Path $itemPath) {
                $folder = $shell.Namespace((Split-Path $itemPath))
                $item = $folder.ParseName((Split-Path $itemPath -Leaf))
                if ($item) {
                    foreach ($verb in $item.Verbs()) {
                        $vname = $verb.Name -replace '&', ''
                        if ($vname -notmatch "unpin|从任务栏|取消|解锁" -and $vname -match "固定到任务栏|锁定到任务栏|Pin to taskbar") {
                            try {
                                $verb.DoIt()
                                Start-Sleep -Milliseconds 1000
                                if (Test-IsTrulyPinned) {
                                    $pinned = $true
                                    break
                                }
                            } catch {}
                        }
                    }
                }
                if ($pinned -or (Test-IsTrulyPinned)) { break }
            }
        }
    } catch {}
}

# 7. Method C: shell:AppsFolder virtual namespace fallback
if (-not $pinned -and -not (Test-IsTrulyPinned)) {
    try {
        $shell = New-Object -ComObject Shell.Application
        $appsFolder = $shell.Namespace('shell:AppsFolder')
        if ($appsFolder) {
            foreach ($appItem in $appsFolder.Items()) {
                if ($appItem.Name -match "传纸条|DeskNotify") {
                    foreach ($verb in $appItem.Verbs()) {
                        $vname = $verb.Name -replace '&', ''
                        if ($vname -notmatch "unpin|从任务栏|取消|解锁" -and $vname -match "固定到任务栏|锁定到任务栏|Pin to taskbar") {
                            try {
                                $verb.DoIt()
                                Start-Sleep -Milliseconds 1000
                                if (Test-IsTrulyPinned) {
                                    $pinned = $true
                                    break
                                }
                            } catch {}
                        }
                    }
                }
                if ($pinned -or (Test-IsTrulyPinned)) { break }
            }
        }
    } catch {}
}

# 8. Report final status
if ($pinned -or (Test-IsTrulyPinned)) {
    Write-Output "SUCCESS"
} else {
    Write-Output "NEED_MANUAL_PIN"
}
`, cleanExePath)

	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", psScript)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}

	out, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(out))
	if err != nil {
		logger.Warnf("[DeskNotify] 固定任务栏脚本执行提示: %v (输出: %s)", err, outputStr)
	}

	if strings.Contains(outputStr, "ALREADY_PINNED") {
		logger.Infof("[DeskNotify] 应用已固定在底部任务栏，无需重复固定")
		return nil
	} else if strings.Contains(outputStr, "SUCCESS") {
		logger.Infof("[DeskNotify] 已成功将应用固定到底部任务栏")
		return nil
	}

	logger.Warnf("[DeskNotify] 任务栏未自动固定 (当前系统安全策略限制静默固定)，已创建桌面与开始菜单快捷方式")
	if revealOnManual {
		desktopDir := getDesktopDir()
		if desktopDir != "" {
			shortcutPath := filepath.Join(desktopDir, "传纸条.lnk")
			if fi, statErr := os.Stat(shortcutPath); statErr == nil && !fi.IsDir() {
				_ = revealFile(shortcutPath)
			}
		}
		return fmt.Errorf("当前 Windows 系统安全策略限制了自动静默固定任务栏。已在桌面为您创建并高亮选中【传纸条】快捷方式，请直接在选中的图标上右键选择【固定到任务栏】")
	}

	return fmt.Errorf("系统安全策略限制了静默固定任务栏，已准备好桌面与开始菜单快捷方式")
}

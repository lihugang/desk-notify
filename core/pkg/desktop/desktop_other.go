//go:build !windows

package desktop

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/lihugang/desk-notify/core/pkg/config"
	"github.com/lihugang/desk-notify/core/pkg/db"
	"github.com/lihugang/desk-notify/core/pkg/logger"
)

type headlessManager struct {
	cfgMgr  *config.Manager
	sigChan chan os.Signal
}

func newManager() Manager {
	return &headlessManager{
		sigChan: make(chan os.Signal, 1),
	}
}

func (m *headlessManager) Run(cfgMgr *config.Manager) error {
	m.cfgMgr = cfgMgr
	signal.Notify(m.sigChan, os.Interrupt, syscall.SIGTERM)
	logger.Infof("[DeskNotify] 运行在 Web Server 渲染模式 (测试/开发环境)")
	if cfgMgr.IsFirstRun() {
		logger.Infof("[DeskNotify] 首次启动应用，自动固定到任务栏并弹出设置页: http://localhost:%d/settings", cfgMgr.Get().Port)
		_ = m.PinToTaskbar()
		m.OpenSettings()
		_ = cfgMgr.MarkFirstRunComplete()
	}
	<-m.sigChan
	return nil
}

func (m *headlessManager) Notify(n *db.Notification) {
	pinnedText := "后台静默 (任务栏闪烁)"
	if n.Pinned {
		pinnedText = "置顶强提醒 (抢走焦点)"
	}
	logger.Infof("[DeskNotify 系统通知] 模式: %s, 内容: %.60s...", pinnedText, n.Content)
	logger.Infof("[DeskNotify 桌面弹窗] 窗口已打开 (测试环境请在浏览器中打开 /card 查看)")
}

func (m *headlessManager) ShowCard() {
	logger.Infof("[DeskNotify 桌面弹窗] 显示卡片窗口")
}

func (m *headlessManager) HideCard() {
	logger.Infof("[DeskNotify 桌面弹窗] 隐藏卡片窗口")
}

func (m *headlessManager) MinimizeCard() {
	logger.Infof("[DeskNotify 桌面弹窗] 最小化卡片窗口")
}

func (m *headlessManager) StopFlash() {
	logger.Infof("[DeskNotify 桌面弹窗] 停止任务栏闪烁")
}

func (m *headlessManager) OpenSettings() {
	if m.cfgMgr != nil {
		url := fmt.Sprintf("http://localhost:%d/settings", m.cfgMgr.Get().Port)
		logger.Infof("[DeskNotify] 请访问设置: %s", url)
		openBrowser(url)
	}
}

func (m *headlessManager) HideSettings() {
	logger.Infof("[DeskNotify] 隐藏设置窗口")
}

func (m *headlessManager) OpenHistory() {
	if m.cfgMgr != nil {
		url := fmt.Sprintf("http://localhost:%d/history", m.cfgMgr.Get().Port)
		logger.Infof("[DeskNotify] 请访问历史通知: %s", url)
		openBrowser(url)
	}
}

func (m *headlessManager) HideHistory() {
	logger.Infof("[DeskNotify] 隐藏历史通知窗口")
}

func (m *headlessManager) OpenArchiveDir() error {
	className := "班级通知"
	if m.cfgMgr != nil {
		className = m.cfgMgr.Get().ClassName
	}
	docDir := config.GetDocumentsDir(className)
	logger.Infof("[DeskNotify] 打开通知存档目录: %s", docDir)
	return openDirectory(docDir)
}

func (m *headlessManager) OpenLogsDir() error {
	logsDir := config.GetLogsDir()
	logger.Infof("[DeskNotify] 打开日志目录: %s", logsDir)
	return openDirectory(logsDir)
}

func (m *headlessManager) OpenFile(filePath string) error {
	return openFile(filePath)
}

func (m *headlessManager) RevealFile(filePath string) error {
	return revealFile(filePath)
}

func (m *headlessManager) PinToTaskbar() error {
	logger.Infof("[DeskNotify] 固定到任务栏: 非 Windows 系统环境跳过")
	return nil
}

func (m *headlessManager) Quit() {
	select {
	case m.sigChan <- syscall.SIGTERM:
	default:
	}
}

func openFile(filePath string) error {
	cleanPath := filepath.Clean(filePath)
	if _, err := os.Stat(cleanPath); err != nil {
		return fmt.Errorf("file not found: %w", err)
	}
	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		cmd = exec.Command("open", cleanPath)
	} else {
		cmd = exec.Command("xdg-open", cleanPath)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	go cmd.Wait() //nolint:errcheck
	return nil
}

func revealFile(filePath string) error {
	cleanPath := filepath.Clean(filePath)
	if runtime.GOOS == "darwin" {
		cmd := exec.Command("open", "-R", cleanPath)
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("failed to reveal file: %w", err)
		}
		go cmd.Wait() //nolint:errcheck
		return nil
	}

	dir := filepath.Dir(cleanPath)
	if fi, err := os.Stat(cleanPath); err == nil && fi.IsDir() {
		dir = cleanPath
	}
	return openDirectory(dir)
}

func openDirectory(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	cleanDir := filepath.Clean(dir)

	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		cmd = exec.Command("open", cleanDir)
	} else {
		cmd = exec.Command("xdg-open", cleanDir)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open directory: %w", err)
	}
	go cmd.Wait() //nolint:errcheck
	return nil
}

func openBrowser(targetURL string) {
	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		cmd = exec.Command("open", targetURL)
	} else {
		cmd = exec.Command("xdg-open", targetURL)
	}
	if err := cmd.Start(); err == nil {
		go cmd.Wait() //nolint:errcheck
	}
}

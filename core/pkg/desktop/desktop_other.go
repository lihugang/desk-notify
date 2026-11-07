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
	fmt.Println("[DeskNotify] 运行在 Web Server 渲染模式 (测试/开发环境)")
	if cfgMgr.IsFirstRun() {
		fmt.Printf("[DeskNotify] 首次启动应用，自动弹出设置页: http://localhost:%d/settings\n", cfgMgr.Get().Port)
		m.OpenSettings()
		_ = cfgMgr.MarkFirstRunComplete()
	}
	<-m.sigChan
	return nil
}

func (m *headlessManager) Notify(n *db.Notification) {
	fmt.Printf("[DeskNotify 系统通知] 级别: %s, 内容: %.60s...\n", n.Level, n.Content)
	fmt.Printf("[DeskNotify 桌面弹窗] 窗口已弹出 (测试环境请在浏览器中打开 /card 查看)\n")
}

func (m *headlessManager) ShowCard() {
	fmt.Println("[DeskNotify 桌面弹窗] 显示卡片窗口")
}

func (m *headlessManager) HideCard() {
	fmt.Println("[DeskNotify 桌面弹窗] 隐藏卡片窗口")
}

func (m *headlessManager) OpenSettings() {
	if m.cfgMgr != nil {
		url := fmt.Sprintf("http://localhost:%d/settings", m.cfgMgr.Get().Port)
		fmt.Printf("[DeskNotify] 请访问设置: %s\n", url)
		openBrowser(url)
	}
}

func (m *headlessManager) HideSettings() {
	fmt.Println("[DeskNotify] 隐藏设置窗口")
}

func (m *headlessManager) OpenHistory() {
	if m.cfgMgr != nil {
		url := fmt.Sprintf("http://localhost:%d/history", m.cfgMgr.Get().Port)
		fmt.Printf("[DeskNotify] 请访问历史通知: %s\n", url)
		openBrowser(url)
	}
}

func (m *headlessManager) HideHistory() {
	fmt.Println("[DeskNotify] 隐藏历史通知窗口")
}

func (m *headlessManager) OpenArchiveDir() error {
	className := "班级通知"
	if m.cfgMgr != nil {
		className = m.cfgMgr.Get().ClassName
	}
	docDir := config.GetDocumentsDir(className)
	fmt.Printf("[DeskNotify] 打开通知存档目录: %s\n", docDir)
	return openDirectory(docDir)
}

func (m *headlessManager) Quit() {
	select {
	case m.sigChan <- syscall.SIGTERM:
	default:
	}
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

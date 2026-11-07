//go:build !windows

package desktop

import (
	"fmt"
	"os"
	"os/signal"
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
		fmt.Printf("[DeskNotify] 请访问设置: http://localhost:%d/settings\n", m.cfgMgr.Get().Port)
	}
}

func (m *headlessManager) HideSettings() {
	fmt.Println("[DeskNotify] 隐藏设置窗口")
}

func (m *headlessManager) Quit() {
	select {
	case m.sigChan <- syscall.SIGTERM:
	default:
	}
}

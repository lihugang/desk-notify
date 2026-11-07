package frptunnel

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fatedier/frp/client"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/lihugang/desk-notify/core/pkg/tunnel"
)

type FRPTunnel struct {
	mu     sync.RWMutex
	cfg    tunnel.Config
	status string
	cancel context.CancelFunc
	svr    *client.Service
}

func NewFRPTunnel() *FRPTunnel {
	return &FRPTunnel{
		status: "disabled",
	}
}

func (f *FRPTunnel) Start(cfg tunnel.Config) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Stop previous running instance if any
	if f.cancel != nil {
		f.cancel()
		f.cancel = nil
	}
	if f.svr != nil {
		f.svr.Close()
		f.svr = nil
	}

	if !cfg.Enabled {
		f.status = "disabled"
		return nil
	}

	if cfg.ServerAddr == "" || cfg.RemotePort == 0 {
		f.status = "disconnected"
		return fmt.Errorf("invalid frp configuration")
	}

	ctx, cancel := context.WithCancel(context.Background())
	f.cancel = cancel
	f.cfg = cfg
	f.status = "connecting"

	go func() {
		// Wait a small delay if restarting to let TCP socket release cleanly
		time.Sleep(100 * time.Millisecond)

		clientCfg := v1.ClientCommonConfig{}
		clientCfg.ServerAddr = cfg.ServerAddr
		clientCfg.ServerPort = cfg.ServerPort
		clientCfg.Auth.Token = cfg.Token
		loginFailExit := false
		clientCfg.LoginFailExit = &loginFailExit
		clientCfg.Complete()

		proxyCfg := &v1.TCPProxyConfig{}
		proxyCfg.Name = fmt.Sprintf("desk-notify-%d", cfg.RemotePort)
		proxyCfg.Type = "tcp"
		proxyCfg.LocalIP = "127.0.0.1"
		proxyCfg.LocalPort = cfg.LocalPort
		proxyCfg.RemotePort = cfg.RemotePort
		proxyCfg.Complete("")

		svr, err := client.NewService(client.ServiceOptions{
			Common: &clientCfg,
			ProxyCfgs: []v1.ProxyConfigurer{
				proxyCfg,
			},
		})
		if err != nil {
			f.mu.Lock()
			f.status = "error"
			f.mu.Unlock()
			fmt.Printf("[DeskNotify FRP] 初始化失败: %v\n", err)
			return
		}

		f.mu.Lock()
		f.svr = svr
		f.status = "connected"
		f.mu.Unlock()
		fmt.Printf("[DeskNotify FRP] 穿透服务启动中: 远程端口 %d -> 本地端口 %d (服务器: %s:%d)\n",
			cfg.RemotePort, cfg.LocalPort, cfg.ServerAddr, cfg.ServerPort)

		if err := svr.Run(ctx); err != nil {
			f.mu.Lock()
			f.status = "disconnected"
			f.mu.Unlock()
			fmt.Printf("[DeskNotify FRP] 穿透服务已停止: %v\n", err)
		} else {
			f.mu.Lock()
			f.status = "disconnected"
			f.mu.Unlock()
			fmt.Println("[DeskNotify FRP] 穿透服务已退出")
		}
	}()

	return nil
}

func (f *FRPTunnel) Stop() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.cancel != nil {
		f.cancel()
		f.cancel = nil
	}
	if f.svr != nil {
		f.svr.Close()
		f.svr = nil
	}
	f.status = "disconnected"
	return nil
}

func (f *FRPTunnel) Status() string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.status
}

func (f *FRPTunnel) PublicURL() string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if f.status != "connected" || !f.cfg.Enabled {
		return ""
	}
	return fmt.Sprintf("http://%s:%d", f.cfg.ServerAddr, f.cfg.RemotePort)
}

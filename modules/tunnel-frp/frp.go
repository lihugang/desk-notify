package frptunnel

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/fatedier/frp/client"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/lihugang/desk-notify/core/pkg/logger"
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
		if cfg.User != "" {
			clientCfg.User = cfg.User
		}
		if cfg.Protocol != "" {
			clientCfg.Transport.Protocol = cfg.Protocol
		}
		if cfg.TLSEnable != nil {
			clientCfg.Transport.TLS.Enable = cfg.TLSEnable
		}
		if cfg.DisableCustomTLSFirstByte != nil {
			clientCfg.Transport.TLS.DisableCustomTLSFirstByte = cfg.DisableCustomTLSFirstByte
		}
		if cfg.ServerName != "" {
			clientCfg.Transport.TLS.ServerName = cfg.ServerName
		}

		loginFailExit := false
		clientCfg.LoginFailExit = &loginFailExit
		clientCfg.Complete()

		proxyName := cfg.ProxyName
		if proxyName == "" {
			proxyName = fmt.Sprintf("desk-notify-%d", cfg.RemotePort)
		}

		proxyCfg := &v1.TCPProxyConfig{}
		proxyCfg.Name = proxyName
		proxyCfg.Type = "tcp"
		proxyCfg.LocalIP = "127.0.0.1"
		proxyCfg.LocalPort = cfg.LocalPort
		proxyCfg.RemotePort = cfg.RemotePort

		prefix := clientCfg.User
		if prefix != "" && strings.HasPrefix(proxyName, prefix+".") {
			prefix = ""
		}
		proxyCfg.Complete(prefix)

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
			logger.Errorf("[DeskNotify FRP] 初始化失败: %v", err)
			return
		}

		f.mu.Lock()
		f.svr = svr
		f.status = "connected"
		f.mu.Unlock()

		tlsStatus := "关闭"
		if clientCfg.Transport.TLS.Enable != nil && *clientCfg.Transport.TLS.Enable {
			if cfg.ServerName != "" {
				tlsStatus = fmt.Sprintf("开启 (SNI: %s)", cfg.ServerName)
			} else {
				tlsStatus = "开启"
			}
		}
		logger.Infof("[DeskNotify FRP] 穿透服务启动中: 隧道 [%s] 远程端口 %d -> 本地端口 %d (服务器: %s:%d, TLS: %s)",
			proxyCfg.Name, cfg.RemotePort, cfg.LocalPort, cfg.ServerAddr, cfg.ServerPort, tlsStatus)

		if err := svr.Run(ctx); err != nil {
			f.mu.Lock()
			f.status = "disconnected"
			f.mu.Unlock()
			logger.Warnf("[DeskNotify FRP] 穿透服务已停止: %v", err)
		} else {
			f.mu.Lock()
			f.status = "disconnected"
			f.mu.Unlock()
			logger.Infof("[DeskNotify FRP] 穿透服务已退出")
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

package frptunnel

import (
	"context"
	"fmt"
	"sync"

	"github.com/fatedier/frp/pkg/config"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/client"
	"github.com/lihugang/desk-notify/core/pkg/tunnel"
)

type FRPTunnel struct {
	mu     sync.RWMutex
	cfg    tunnel.Config
	status string
	cancel context.CancelFunc
}

func NewFRPTunnel() *FRPTunnel {
	return &FRPTunnel{
		status: "disabled",
	}
}

func (f *FRPTunnel) Start(cfg tunnel.Config) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !cfg.Enabled {
		f.status = "disabled"
		return nil
	}

	if cfg.ServerAddr == "" || cfg.RemotePort == 0 {
		f.status = "disconnected"
		return fmt.Errorf("invalid frp configuration")
	}

	if f.cancel != nil {
		f.cancel()
		f.cancel = nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	f.cancel = cancel
	f.cfg = cfg
	f.status = "connecting"

	go func() {
		clientCfg := v1.ClientCommonConfig{}
		clientCfg.ServerAddr = cfg.ServerAddr
		clientCfg.ServerPort = cfg.ServerPort
		clientCfg.Auth.Token = cfg.Token

		proxyCfg := &v1.TCPProxyConfig{}
		proxyCfg.Name = fmt.Sprintf("desk-notify-%d", cfg.RemotePort)
		proxyCfg.LocalIP = "127.0.0.1"
		proxyCfg.LocalPort = cfg.LocalPort
		proxyCfg.RemotePort = cfg.RemotePort

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
			return
		}

		f.mu.Lock()
		f.status = "connected"
		f.mu.Unlock()

		if err := svr.Run(ctx); err != nil {
			f.mu.Lock()
			f.status = "disconnected"
			f.mu.Unlock()
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

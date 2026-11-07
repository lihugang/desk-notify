//go:build !(frp || release)
// +build !frp,!release

package main

import (
	"github.com/lihugang/desk-notify/core/pkg/logger"
	"github.com/lihugang/desk-notify/core/pkg/tunnel"
)

// initTunnel 初始化本地开发模拟穿透通道 (Mock Tunnel)
func initTunnel() tunnel.Tunnel {
	logger.Infof("[DeskNotify] 运行模式: 本地开发模式 (使用 Mock Tunnel，未链接 FRP SDK)")
	return tunnel.NewMockTunnel()
}

//go:build frp || release
// +build frp release

package main

import (
	"fmt"

	"github.com/lihugang/desk-notify/core/pkg/tunnel"
	frptunnel "github.com/lihugang/desk-notify/modules/tunnel-frp"
)

// initTunnel 初始化正式发布原生 FRP 穿透通道 (FRP Tunnel)
func initTunnel() tunnel.Tunnel {
	fmt.Println("[DeskNotify] 运行模式: 正式发布模式 (链接 FRP 原生 SDK，支持内网穿透)")
	return frptunnel.NewFRPTunnel()
}

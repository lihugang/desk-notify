package tunnel

import (
	"fmt"
	"sync"
)

type MockTunnel struct {
	mu     sync.RWMutex
	cfg    Config
	status string
}

func NewMockTunnel() *MockTunnel {
	return &MockTunnel{
		status: "disabled",
	}
}

func (m *MockTunnel) Start(cfg Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cfg = cfg
	if !cfg.Enabled {
		m.status = "disabled"
		return nil
	}

	if cfg.ServerAddr == "" {
		m.status = "disconnected"
		return fmt.Errorf("server address is empty")
	}

	m.status = "connected"
	return nil
}

func (m *MockTunnel) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status = "disconnected"
	return nil
}

func (m *MockTunnel) Status() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

func (m *MockTunnel) PublicURL() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.status != "connected" || !m.cfg.Enabled {
		return ""
	}
	return fmt.Sprintf("http://%s:%d", m.cfg.ServerAddr, m.cfg.RemotePort)
}

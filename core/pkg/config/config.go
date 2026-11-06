package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

type FRPConfig struct {
	Enabled    bool   `json:"enabled"`
	ServerAddr string `json:"serverAddr"`
	ServerPort int    `json:"serverPort"`
	Token      string `json:"token"`
	RemotePort int    `json:"remotePort"`
	Subdomain  string `json:"subdomain,omitempty"`
}

type AppConfig struct {
	ClassName       string    `json:"className"`
	Port            int       `json:"port"`
	PublicPassword  string    `json:"publicPassword"`
	LanAuthRequired bool      `json:"lanAuthRequired"`
	DefaultDuration int       `json:"defaultDuration"`
	SoundEnabled    bool      `json:"soundEnabled"`
	CustomSound     string    `json:"customSound"`
	AutoStart       bool      `json:"autoStart"`
	FRP             FRPConfig `json:"frp"`
}

type Manager struct {
	mu         sync.RWMutex
	cfg        AppConfig
	configPath string
	dataDir    string
}

func DefaultConfig() AppConfig {
	return AppConfig{
		ClassName:       "三年二班",
		Port:            8920,
		PublicPassword:  "teacher888",
		LanAuthRequired: false,
		DefaultDuration: 30,
		SoundEnabled:    true,
		CustomSound:     "",
		AutoStart:       true,
		FRP: FRPConfig{
			Enabled:    false,
			ServerAddr: "",
			ServerPort: 7000,
			Token:      "",
			RemotePort: 8920,
		},
	}
}

// GetDataDir returns the directory for storing databases and configs.
func GetDataDir() string {
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData != "" {
			return filepath.Join(appData, "DeskNotify")
		}
	}
	home, err := os.UserHomeDir()
	if err == nil {
		return filepath.Join(home, ".local", "share", "desk-notify")
	}
	return "./dev_data"
}

// GetDocumentsDir returns the Documents directory for archiving files.
func GetDocumentsDir(className string) string {
	if className == "" {
		className = "班级通知"
	}
	if runtime.GOOS == "windows" {
		userProfile := os.Getenv("USERPROFILE")
		if userProfile != "" {
			return filepath.Join(userProfile, "Documents", "DeskNotify", className)
		}
	}
	home, err := os.UserHomeDir()
	if err == nil {
		return filepath.Join(home, "Documents", "DeskNotify", className)
	}
	return filepath.Join("./dev_data", "Documents", className)
}

func NewManager() (*Manager, error) {
	dataDir := GetDataDir()
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	configPath := filepath.Join(dataDir, "config.json")
	mgr := &Manager{
		cfg:        DefaultConfig(),
		configPath: configPath,
		dataDir:    dataDir,
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		_ = mgr.Save(mgr.cfg)
	} else {
		_ = mgr.Load()
	}

	return mgr, nil
}

func (m *Manager) Get() AppConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg
}

func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &m.cfg)
}

func (m *Manager) Save(cfg AppConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cfg = cfg
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.configPath, data, 0644)
}

func (m *Manager) GetDataDir() string {
	return m.dataDir
}

package config

import (
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
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
	ClassName          string    `json:"className"`
	Port               int       `json:"port"`
	PublicPassword     string    `json:"publicPassword,omitempty"`
	PublicPasswordHash string    `json:"publicPasswordHash,omitempty"`
	AdminPasswordHash  string    `json:"adminPasswordHash,omitempty"`
	LanAuthRequired    bool      `json:"lanAuthRequired"`
	DefaultDuration    int       `json:"defaultDuration"`
	SoundEnabled       bool      `json:"soundEnabled"`
	CustomSound        string    `json:"customSound"`
	AutoStart          bool      `json:"autoStart"`
	FRP                FRPConfig `json:"frp"`
}

type Manager struct {
	mu         sync.RWMutex
	cfg        AppConfig
	configPath string
	dataDir    string
}

func hashBcrypt(password string) string {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return ""
	}
	return string(bytes)
}

func generateRandomFRPPort() int {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return 10001 + r.Intn(39999) // 10001 - 50000
}

func DefaultConfig() AppConfig {
	return AppConfig{
		ClassName:          "2907",
		Port:               8920,
		PublicPassword:     "",
		PublicPasswordHash: hashBcrypt("123456"),
		AdminPasswordHash:  hashBcrypt("123456"),
		LanAuthRequired:    false,
		DefaultDuration:    30,
		SoundEnabled:       true,
		CustomSound:        "",
		AutoStart:          true,
		FRP: FRPConfig{
			Enabled:    false,
			ServerAddr: "frp.freefrp.net",
			ServerPort: 7000,
			Token:      "freefrp.net",
			RemotePort: generateRandomFRPPort(),
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
	return NewManagerWithDir(GetDataDir())
}

func NewManagerWithDir(dataDir string) (*Manager, error) {
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
		modified := false
		if mgr.cfg.ClassName == "三年二班" || mgr.cfg.ClassName == "" {
			mgr.cfg.ClassName = "2907"
			modified = true
		}
		if mgr.cfg.FRP.ServerAddr == "" {
			mgr.cfg.FRP.ServerAddr = "frp.freefrp.net"
			mgr.cfg.FRP.ServerPort = 7000
			mgr.cfg.FRP.Token = "freefrp.net"
			modified = true
		}
		if mgr.cfg.FRP.RemotePort <= 0 {
			mgr.cfg.FRP.RemotePort = generateRandomFRPPort()
			modified = true
		}
		if mgr.cfg.AdminPasswordHash == "" {
			mgr.cfg.AdminPasswordHash = hashBcrypt("123456")
			modified = true
		}
		if mgr.cfg.PublicPasswordHash == "" {
			if mgr.cfg.PublicPassword != "" {
				mgr.cfg.PublicPasswordHash = hashBcrypt(mgr.cfg.PublicPassword)
				mgr.cfg.PublicPassword = ""
			} else {
				mgr.cfg.PublicPasswordHash = hashBcrypt("123456")
			}
			modified = true
		} else if mgr.cfg.PublicPassword != "" {
			mgr.cfg.PublicPassword = ""
			modified = true
		}
		if modified {
			_ = mgr.Save(mgr.cfg)
		}
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

package config

import (
	"crypto/aes"
	"crypto/cipher"
	cryptorand "crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// configKey is the 32-byte AES-256 key embedded directly in the application binary for config encryption.
var configKey = []byte("DeskNotify_Encrypted_Config_2026")

// encryptAES encrypts plaintext using AES-GCM with the embedded key.
func encryptAES(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(configKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(cryptorand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// decryptAES decrypts ciphertext using AES-GCM with the embedded key.
func decryptAES(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(configKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, encryptedData := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, encryptedData, nil)
}

type FRPConfig struct {
	Enabled                   bool   `json:"enabled"`
	ServerAddr                string `json:"serverAddr"`
	ServerPort                int    `json:"serverPort"`
	Token                     string `json:"token"`
	RemotePort                int    `json:"remotePort"`
	Subdomain                 string `json:"subdomain,omitempty"`
	User                      string `json:"user,omitempty"`
	ProxyName                 string `json:"proxyName,omitempty"`
	Protocol                  string `json:"protocol,omitempty"`
	TLSEnable                 *bool  `json:"tlsEnable,omitempty"`
	DisableCustomTLSFirstByte *bool  `json:"disableCustomTLSFirstByte,omitempty"`
	ServerName                string `json:"serverName,omitempty"`
}

type AppConfig struct {
	ClassName          string    `json:"className"`
	Port               int       `json:"port"`
	PasswordHash       string    `json:"passwordHash,omitempty"`
	PublicPassword     string    `json:"publicPassword,omitempty"`
	PublicPasswordHash string    `json:"publicPasswordHash,omitempty"`
	AdminPasswordHash  string    `json:"adminPasswordHash,omitempty"`
	LanAuthRequired    bool      `json:"lanAuthRequired"`
	SoundEnabled       bool      `json:"soundEnabled"`
	CustomSound        string    `json:"customSound"`
	FontSize           string    `json:"fontSize,omitempty"`
	AutoStart          bool      `json:"autoStart"`
	FirstRunDone       bool      `json:"firstRunDone"`
	FRP                FRPConfig `json:"frp"`
}

type Manager struct {
	mu         sync.RWMutex
	cfg        AppConfig
	configPath string
	dataDir    string
	isFirstRun bool
}

// GetEffectivePasswordHash returns the unified password hash.
func (c *AppConfig) GetEffectivePasswordHash() string {
	if c.PasswordHash != "" {
		return c.PasswordHash
	}
	if c.AdminPasswordHash != "" {
		return c.AdminPasswordHash
	}
	if c.PublicPasswordHash != "" {
		return c.PublicPasswordHash
	}
	return ""
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
	defaultHash := hashBcrypt("123456")
	return AppConfig{
		ClassName:          "2907",
		Port:               8920,
		PasswordHash:       defaultHash,
		PublicPassword:     "",
		PublicPasswordHash: defaultHash,
		AdminPasswordHash:  defaultHash,
		LanAuthRequired:    false,
		SoundEnabled:       true,
		CustomSound:        "",
		FontSize:           "16px",
		AutoStart:          true,
		FirstRunDone:       false,
		FRP: FRPConfig{
			Enabled:    true,
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

// GetLogsDir returns the directory for storing daily log files (%APPDATA%/DeskNotify/logs on Windows).
func GetLogsDir() string {
	return filepath.Join(GetDataDir(), "logs")
}

// GetLogsDir returns the manager's data directory logs path.
func (m *Manager) GetLogsDir() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return filepath.Join(m.dataDir, "logs")
}

// SanitizeClassName sanitizes a class name for use in filesystem paths.
func SanitizeClassName(name string) string {
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	res := name
	for _, c := range invalidChars {
		res = strings.ReplaceAll(res, c, "_")
	}
	res = strings.TrimSpace(res)
	if res == "" {
		res = "班级通知"
	}
	return res
}

// GetDocumentsDir returns the Documents directory for archiving files.
func GetDocumentsDir(className string) string {
	className = SanitizeClassName(className)
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
		mgr.isFirstRun = true
		_ = mgr.Save(mgr.cfg)
	} else {
		_ = mgr.Load()
		if !mgr.cfg.FirstRunDone {
			mgr.isFirstRun = true
		}
		modified := false
		if mgr.cfg.FontSize == "" {
			mgr.cfg.FontSize = "16px"
			modified = true
		}
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
		// Unify password configuration
		if mgr.cfg.PasswordHash == "" {
			if mgr.cfg.AdminPasswordHash != "" {
				mgr.cfg.PasswordHash = mgr.cfg.AdminPasswordHash
			} else if mgr.cfg.PublicPasswordHash != "" {
				mgr.cfg.PasswordHash = mgr.cfg.PublicPasswordHash
			} else if mgr.cfg.PublicPassword != "" {
				mgr.cfg.PasswordHash = hashBcrypt(mgr.cfg.PublicPassword)
			} else {
				mgr.cfg.PasswordHash = hashBcrypt("123456")
			}
			modified = true
		}
		if mgr.cfg.AdminPasswordHash != mgr.cfg.PasswordHash {
			mgr.cfg.AdminPasswordHash = mgr.cfg.PasswordHash
			modified = true
		}
		if mgr.cfg.PublicPasswordHash != mgr.cfg.PasswordHash {
			mgr.cfg.PublicPasswordHash = mgr.cfg.PasswordHash
			modified = true
		}
		if mgr.cfg.PublicPassword != "" {
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
	decrypted, err := decryptAES(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(decrypted, &m.cfg)
}

func (m *Manager) Save(cfg AppConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cfg = cfg
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	encrypted, err := encryptAES(data)
	if err != nil {
		return err
	}
	return os.WriteFile(m.configPath, encrypted, 0644)
}

func (m *Manager) GetDataDir() string {
	return m.dataDir
}

// IsFirstRun returns true if this is the first time the app is launched.
func (m *Manager) IsFirstRun() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isFirstRun
}

// MarkFirstRunComplete marks the first run as completed and persists this to configuration.
func (m *Manager) MarkFirstRunComplete() error {
	m.mu.Lock()
	m.isFirstRun = false
	m.cfg.FirstRunDone = true
	cfg := m.cfg
	m.mu.Unlock()
	return m.Save(cfg)
}

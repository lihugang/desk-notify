package desktop

import (
	"github.com/lihugang/desk-notify/core/pkg/config"
	"github.com/lihugang/desk-notify/core/pkg/db"
)

type Manager interface {
	Run(cfgMgr *config.Manager) error
	Notify(n *db.Notification)
	ShowCard()
	HideCard()
	MinimizeCard()
	StopFlash()
	OpenSettings()
	HideSettings()
	OpenHistory()
	HideHistory()
	OpenArchiveDir() error
	OpenLogsDir() error
	OpenFile(filePath string) error
	RevealFile(filePath string) error
	PinToTaskbar() error
	Quit()
}

func NewManager() Manager {
	return newManager()
}

func OpenFile(filePath string) error {
	return openFile(filePath)
}

func RevealFile(filePath string) error {
	return revealFile(filePath)
}

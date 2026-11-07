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
	OpenSettings()
	HideSettings()
	Quit()
}

func NewManager() Manager {
	return newManager()
}

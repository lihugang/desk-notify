package tunnel

type Config struct {
	Enabled                   bool   `json:"enabled"`
	ServerAddr                string `json:"serverAddr"`
	ServerPort                int    `json:"serverPort"`
	Token                     string `json:"token"`
	RemotePort                int    `json:"remotePort"`
	LocalPort                 int    `json:"localPort"`
	User                      string `json:"user,omitempty"`
	ProxyName                 string `json:"proxyName,omitempty"`
	Protocol                  string `json:"protocol,omitempty"`
	TLSEnable                 *bool  `json:"tlsEnable,omitempty"`
	DisableCustomTLSFirstByte *bool  `json:"disableCustomTLSFirstByte,omitempty"`
	ServerName                string `json:"serverName,omitempty"`
}

type Tunnel interface {
	Start(cfg Config) error
	Stop() error
	Status() string
	PublicURL() string
}

package tunnel

type Config struct {
	Enabled    bool   `json:"enabled"`
	ServerAddr string `json:"serverAddr"`
	ServerPort int    `json:"serverPort"`
	Token      string `json:"token"`
	RemotePort int    `json:"remotePort"`
	LocalPort  int    `json:"localPort"`
}

type Tunnel interface {
	Start(cfg Config) error
	Stop() error
	Status() string
	PublicURL() string
}

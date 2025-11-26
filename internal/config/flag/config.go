package config

type ServerConfig struct {
	AddrServer string
	AddrURL    string
}

func NewServerConfig(addr, url string) *ServerConfig {
	return &ServerConfig{
		AddrServer: addr,
		AddrURL:    url,
	}
}

package config

type ServerConfig struct {
	AddrServer string
	AddrUrl    string
}

func NewServerConfig(addr, url string) *ServerConfig {
	return &ServerConfig{
		AddrServer: addr,
		AddrUrl:    url,
	}
}

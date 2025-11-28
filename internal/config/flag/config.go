package config

type ServerConfig struct {
	AddrServer string
	AddrURL    string
	Env        string
}

func NewServerConfig(addr, url, env string) *ServerConfig {
	return &ServerConfig{
		AddrServer: addr,
		AddrURL:    url,
		Env:        env,
	}
}

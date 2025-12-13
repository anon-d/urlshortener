package config

import (
	"flag"
	"os"
	"sync"
)

type ServerConfig struct {
	AddrServer string `env:"SERVER_ADDRESS"`
	AddrURL    string `env:"BASE_URL"`
	Env        string `env:"ENV"`
}

var (
	addrServer *string
	addrURL    *string
	envValue   *string
	flagsOnce  sync.Once
)

func initFlags() {
	addrServer = flag.String("a", ":8080", "address to listen on")
	addrURL = flag.String("b", "http://localhost:8080", "base URL for short URLs")
	envValue = flag.String("e", "dev", "environment")
}

func NewServerConfig() *ServerConfig {
	flagsOnce.Do(initFlags)

	if !flag.Parsed() {
		flag.Parse()
	}

	cfg := &ServerConfig{}

	if envAddr := os.Getenv("SERVER_ADDRESS"); envAddr != "" {
		cfg.AddrServer = envAddr
	} else {
		cfg.AddrServer = *addrServer
	}

	if envURL := os.Getenv("BASE_URL"); envURL != "" {
		cfg.AddrURL = envURL
	} else {
		cfg.AddrURL = *addrURL
	}

	if envEnv := os.Getenv("ENV"); envEnv != "" {
		cfg.Env = envEnv
	} else {
		cfg.Env = *envValue
	}

	return cfg
}

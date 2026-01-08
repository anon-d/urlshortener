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
	File       string `env:"FILE_STORAGE_PATH"`
	DSN        string `env:"DATABASE_DSN"`
}

var (
	addrServer *string
	addrURL    *string
	envValue   *string
	fileValue  *string
	dsnValue   *string
	flagsOnce  sync.Once
)

func initFlags() {
	addrServer = flag.String("a", ":8080", "address to listen on")
	addrURL = flag.String("b", "http://localhost:8080", "base URL for short URLs")
	envValue = flag.String("e", "dev", "environment")
	fileValue = flag.String("f", "data.json", "file to store data")
	dsnValue = flag.String("d", "", "database DSN")
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

	if envFile := os.Getenv("FILE_STORAGE_PATH"); envFile != "" {
		cfg.File = envFile
	} else {
		cfg.File = *fileValue
	}

	if envDSN := os.Getenv("DATABASE_DSN"); envDSN != "" {
		cfg.DSN = envDSN
	} else {
		cfg.DSN = *dsnValue
	}

	return cfg
}

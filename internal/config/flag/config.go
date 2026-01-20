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

	if envAddr, ok := os.LookupEnv("SERVER_ADDRESS"); ok {
		cfg.AddrServer = envAddr
	} else {
		cfg.AddrServer = *addrServer
	}

	if envURL, ok := os.LookupEnv("BASE_URL"); ok {
		cfg.AddrURL = envURL
	} else {
		cfg.AddrURL = *addrURL
	}

	if envEnv, ok := os.LookupEnv("ENV"); ok {
		cfg.Env = envEnv
	} else {
		cfg.Env = *envValue
	}

	if envFile, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		cfg.File = envFile
	} else {
		cfg.File = *fileValue
	}

	if envDSN, ok := os.LookupEnv("DATABASE_DSN"); ok {
		cfg.DSN = envDSN
	} else {
		cfg.DSN = *dsnValue
	}

	return cfg
}

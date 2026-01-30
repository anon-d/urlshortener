package config

import (
	"flag"
	"os"
	"strconv"
	"sync"
)

type ServerConfig struct {
	AddrServer         string `env:"SERVER_ADDRESS"`
	AddrURL            string `env:"BASE_URL"`
	Env                string `env:"ENV"`
	File               string `env:"FILE_STORAGE_PATH"`
	DSN                string `env:"DATABASE_DSN"`
	DeleteWorkerCount  int    `env:"DELETE_WORKER_COUNT"`
	DeleteChannelSize  int    `env:"DELETE_CHANNEL_SIZE"`
	SecretKey          string `env:"SECRET_KEY"`
}

var (
	addrServer         *string
	addrURL            *string
	envValue           *string
	fileValue          *string
	dsnValue           *string
	deleteWorkerCount  *int
	deleteChannelSize  *int
	secretKey          *string
	flagsOnce          sync.Once
)

func initFlags() {
	addrServer = flag.String("a", ":8080", "address to listen on")
	addrURL = flag.String("b", "http://localhost:8080", "base URL for short URLs")
	envValue = flag.String("e", "dev", "environment")
	fileValue = flag.String("f", "data.json", "file to store data")
	dsnValue = flag.String("d", "", "database DSN")
	deleteWorkerCount = flag.Int("w", 2, "number of delete worker channels")
	deleteChannelSize = flag.Int("c", 1000, "size of each delete channel buffer")
	secretKey = flag.String("s", "my-super-secret-key-change-in-production", "secret key for signing cookies")
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

	if envWorkerCount, ok := os.LookupEnv("DELETE_WORKER_COUNT"); ok {
		if count, err := parseIntFromEnv(envWorkerCount); err == nil {
			cfg.DeleteWorkerCount = count
		} else {
			cfg.DeleteWorkerCount = *deleteWorkerCount
		}
	} else {
		cfg.DeleteWorkerCount = *deleteWorkerCount
	}

	if envChannelSize, ok := os.LookupEnv("DELETE_CHANNEL_SIZE"); ok {
		if size, err := parseIntFromEnv(envChannelSize); err == nil {
			cfg.DeleteChannelSize = size
		} else {
			cfg.DeleteChannelSize = *deleteChannelSize
		}
	} else {
		cfg.DeleteChannelSize = *deleteChannelSize
	}

	if envSecretKey, ok := os.LookupEnv("SECRET_KEY"); ok {
		cfg.SecretKey = envSecretKey
	} else {
		cfg.SecretKey = *secretKey
	}

	return cfg
}

func parseIntFromEnv(s string) (int, error) {
	return strconv.Atoi(s)
}

// Package config обеспечивает загрузку конфигурации из флагов командной строки,
// переменных окружения и JSON-файла конфигурации.
// Приоритет: переменные окружения > флаги > JSON-файл.
package config

import (
	"encoding/json"
	"flag"
	"os"
	"strconv"
	"sync"
)

// ServerConfig — конфигурация сервера.
type ServerConfig struct {
	AddrServer        string `env:"SERVER_ADDRESS"`
	AddrURL           string `env:"BASE_URL"`
	Env               string `env:"ENV"`
	File              string `env:"FILE_STORAGE_PATH"`
	DSN               string `env:"DATABASE_DSN"`
	DeleteWorkerCount int    `env:"DELETE_WORKER_COUNT"`
	DeleteChannelSize int    `env:"DELETE_CHANNEL_SIZE"`
	SecretKey         string `env:"SECRET_KEY"`
	AuditFile         string `env:"AUDIT_FILE"`
	AuditURL          string `env:"AUDIT_URL"`
	Enable_HTTPS      bool   `env:"ENABLE_HTTPS"`
	ConfigJSON        string `env:"CONFIG"`
}

// JSONFileConfig — структура JSON-файла конфигурации.
type JSONFileConfig struct {
	ServerAddress   string `json:"server_address"`
	BaseURL         string `json:"base_url"`
	FileStoragePath string `json:"file_storage_path"`
	DatabaseDSN     string `json:"database_dsn"`
	EnableHTTPS     *bool  `json:"enable_https"`
}

var (
	addrServer        *string
	addrURL           *string
	envValue          *string
	fileValue         *string
	dsnValue          *string
	deleteWorkerCount *int
	deleteChannelSize *int
	secretKey         *string
	auditFile         *string
	auditURL          *string
	enableHTTPS       *bool
	flagsOnce         sync.Once
	jsonConfig        *string
	jsonConfigLong    *string
)

func initFlags() {
	addrServer = flag.String("a", ":8080", "address to listen on")
	addrURL = flag.String("b", "http://localhost:8080", "base URL for short URLs")
	envValue = flag.String("e", "dev", "environment")
	fileValue = flag.String("f", "data.json", "file to store data")
	dsnValue = flag.String("d", "", "database DSN")
	deleteWorkerCount = flag.Int("wc", 2, "number of delete worker channels")
	deleteChannelSize = flag.Int("cs", 1000, "size of each delete channel buffer")
	secretKey = flag.String("sk", "my-super-secret-key-change-in-production", "secret key for signing cookies")
	auditFile = flag.String("audit-file", "", "path to audit log file")
	auditURL = flag.String("audit-url", "", "URL of remote audit server")
	enableHTTPS = flag.Bool("s", false, "enable HTTPS")
	jsonConfig = flag.String("c", "", "path to JSON config file")
	jsonConfigLong = flag.String("config", "", "path to JSON config file")
}

// loadJSONConfig загружает конфигурацию из JSON-файла.
func loadJSONConfig(path string) (*JSONFileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg JSONFileConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// NewServerConfig создаёт конфигурацию, читая флаги, JSON-файл и переменные окружения.
// Приоритет: переменная окружения > явно переданный флаг > JSON-конфиг > значение флага по умолчанию.
func NewServerConfig() *ServerConfig {
	flagsOnce.Do(initFlags)

	if !flag.Parsed() {
		flag.Parse()
	}

	// Определяем, какие флаги были явно переданы
	flagSet := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		flagSet[f.Name] = true
	})

	// Определяем путь к JSON-конфигурации: ENV CONFIG > флаг -c/-config
	configPath := ""
	if v, ok := os.LookupEnv("CONFIG"); ok {
		configPath = v
	} else if *jsonConfig != "" {
		configPath = *jsonConfig
	} else if *jsonConfigLong != "" {
		configPath = *jsonConfigLong
	}

	// Загружаем JSON-конфиг, если путь задан
	var jcfg *JSONFileConfig
	if configPath != "" {
		if loaded, err := loadJSONConfig(configPath); err == nil {
			jcfg = loaded
		}
	}

	cfg := &ServerConfig{
		ConfigJSON: configPath,
	}

	// --- Поля с поддержкой JSON-конфига ---

	// AddrServer
	cfg.AddrServer = *addrServer
	if jcfg != nil && jcfg.ServerAddress != "" {
		cfg.AddrServer = jcfg.ServerAddress
	}
	if flagSet["a"] {
		cfg.AddrServer = *addrServer
	}
	if v, ok := os.LookupEnv("SERVER_ADDRESS"); ok {
		cfg.AddrServer = v
	}

	// AddrURL
	cfg.AddrURL = *addrURL
	if jcfg != nil && jcfg.BaseURL != "" {
		cfg.AddrURL = jcfg.BaseURL
	}
	if flagSet["b"] {
		cfg.AddrURL = *addrURL
	}
	if v, ok := os.LookupEnv("BASE_URL"); ok {
		cfg.AddrURL = v
	}

	// File
	cfg.File = *fileValue
	if jcfg != nil && jcfg.FileStoragePath != "" {
		cfg.File = jcfg.FileStoragePath
	}
	if flagSet["f"] {
		cfg.File = *fileValue
	}
	if v, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		cfg.File = v
	}

	// DSN
	cfg.DSN = *dsnValue
	if jcfg != nil && jcfg.DatabaseDSN != "" {
		cfg.DSN = jcfg.DatabaseDSN
	}
	if flagSet["d"] {
		cfg.DSN = *dsnValue
	}
	if v, ok := os.LookupEnv("DATABASE_DSN"); ok {
		cfg.DSN = v
	}

	// Enable_HTTPS
	cfg.Enable_HTTPS = *enableHTTPS
	if jcfg != nil && jcfg.EnableHTTPS != nil {
		cfg.Enable_HTTPS = *jcfg.EnableHTTPS
	}
	if flagSet["s"] {
		cfg.Enable_HTTPS = *enableHTTPS
	}
	if v, ok := os.LookupEnv("ENABLE_HTTPS"); ok {
		if val, err := strconv.ParseBool(v); err == nil {
			cfg.Enable_HTTPS = val
		}
	}

	// --- Поля без JSON-конфига ---

	if v, ok := os.LookupEnv("ENV"); ok {
		cfg.Env = v
	} else {
		cfg.Env = *envValue
	}

	if v, ok := os.LookupEnv("DELETE_WORKER_COUNT"); ok {
		if count, err := parseIntFromEnv(v); err == nil {
			cfg.DeleteWorkerCount = count
		} else {
			cfg.DeleteWorkerCount = *deleteWorkerCount
		}
	} else {
		cfg.DeleteWorkerCount = *deleteWorkerCount
	}

	if v, ok := os.LookupEnv("DELETE_CHANNEL_SIZE"); ok {
		if size, err := parseIntFromEnv(v); err == nil {
			cfg.DeleteChannelSize = size
		} else {
			cfg.DeleteChannelSize = *deleteChannelSize
		}
	} else {
		cfg.DeleteChannelSize = *deleteChannelSize
	}

	if v, ok := os.LookupEnv("SECRET_KEY"); ok {
		cfg.SecretKey = v
	} else {
		cfg.SecretKey = *secretKey
	}

	if v, ok := os.LookupEnv("AUDIT_FILE"); ok {
		cfg.AuditFile = v
	} else {
		cfg.AuditFile = *auditFile
	}

	if v, ok := os.LookupEnv("AUDIT_URL"); ok {
		cfg.AuditURL = v
	} else {
		cfg.AuditURL = *auditURL
	}

	return cfg
}

func parseIntFromEnv(s string) (int, error) {
	return strconv.Atoi(s)
}

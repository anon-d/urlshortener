// Package config обеспечивает загрузку конфигурации с помощью Viper.
// Источники читаются в следующем приоритете (от высшего к низшему):
// явно переданный флаг > переменная окружения > JSON-файл > значение по умолчанию.
package config

import (
	"os"
	"strconv"
	"sync"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
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
	CertFile          string `env:"CERT_FILE"`
	KeyFile           string `env:"KEY_FILE"`
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
	fs        *pflag.FlagSet
	flagsOnce sync.Once
)

func initFlags() {
	fs = pflag.NewFlagSet("shortener", pflag.ContinueOnError)
	fs.String("a", ":8080", "address to listen on")
	fs.String("b", "http://localhost:8080", "base URL for short URLs")
	fs.String("e", "dev", "environment")
	fs.String("f", "data.json", "file to store data")
	fs.String("d", "", "database DSN")
	fs.Int("wc", 2, "number of delete worker channels")
	fs.Int("cs", 1000, "size of each delete channel buffer")
	fs.String("sk", "my-super-secret-key-change-in-production", "secret key for signing cookies")
	fs.String("audit-file", "", "path to audit log file")
	fs.String("audit-url", "", "URL of remote audit server")
	fs.Bool("s", false, "enable HTTPS")
	fs.String("cert", "cert.pem", "path to TLS certificate file")
	fs.String("key", "key.pem", "path to TLS private key file")
	fs.StringP("config", "c", "", "path to JSON config file")
	// Ошибки разбора флагов (например, неизвестные флаги go test) намеренно игнорируются.
	_ = fs.Parse(os.Args[1:])
}

// loadJSONConfig загружает конфигурацию из JSON-файла.
func loadJSONConfig(path string) (*JSONFileConfig, error) {
	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}
	cfg := &JSONFileConfig{
		ServerAddress:   v.GetString("server_address"),
		BaseURL:         v.GetString("base_url"),
		FileStoragePath: v.GetString("file_storage_path"),
		DatabaseDSN:     v.GetString("database_dsn"),
	}
	if v.IsSet("enable_https") {
		b := v.GetBool("enable_https")
		cfg.EnableHTTPS = &b
	}
	return cfg, nil
}

// NewServerConfig создаёт конфигурацию с помощью Viper.
// Приоритет: явно переданный флаг > переменная окружения > JSON-конфиг > значение по умолчанию.
func NewServerConfig() *ServerConfig {
	flagsOnce.Do(initFlags)

	v := viper.New()

	// Значения по умолчанию
	v.SetDefault("server_address", ":8080")
	v.SetDefault("base_url", "http://localhost:8080")
	v.SetDefault("env", "dev")
	v.SetDefault("file_storage_path", "data.json")
	v.SetDefault("database_dsn", "")
	v.SetDefault("delete_worker_count", 2)
	v.SetDefault("delete_channel_size", 1000)
	v.SetDefault("secret_key", "my-super-secret-key-change-in-production")
	v.SetDefault("audit_file", "")
	v.SetDefault("audit_url", "")
	v.SetDefault("enable_https", false)
	v.SetDefault("cert_file", "cert.pem")
	v.SetDefault("key_file", "key.pem")

	// Привязка флагов командной строки к ключам Viper
	_ = v.BindPFlag("server_address", fs.Lookup("a"))
	_ = v.BindPFlag("base_url", fs.Lookup("b"))
	_ = v.BindPFlag("env", fs.Lookup("e"))
	_ = v.BindPFlag("file_storage_path", fs.Lookup("f"))
	_ = v.BindPFlag("database_dsn", fs.Lookup("d"))
	_ = v.BindPFlag("delete_worker_count", fs.Lookup("wc"))
	_ = v.BindPFlag("delete_channel_size", fs.Lookup("cs"))
	_ = v.BindPFlag("secret_key", fs.Lookup("sk"))
	_ = v.BindPFlag("audit_file", fs.Lookup("audit-file"))
	_ = v.BindPFlag("audit_url", fs.Lookup("audit-url"))
	_ = v.BindPFlag("enable_https", fs.Lookup("s"))
	_ = v.BindPFlag("cert_file", fs.Lookup("cert"))
	_ = v.BindPFlag("key_file", fs.Lookup("key"))
	_ = v.BindPFlag("config", fs.Lookup("config"))

	// Переменные окружения: ключ автоматически преобразуется в верхний регистр
	// (server_address → SERVER_ADDRESS, base_url → BASE_URL и т.д.)
	v.AutomaticEnv()

	// Загрузка JSON-конфига, если путь задан через флаг --config/-c или переменную CONFIG
	configPath := v.GetString("config")
	if configPath != "" {
		v.SetConfigFile(configPath)
		_ = v.ReadInConfig()
	}

	return &ServerConfig{
		AddrServer:        v.GetString("server_address"),
		AddrURL:           v.GetString("base_url"),
		Env:               v.GetString("env"),
		File:              v.GetString("file_storage_path"),
		DSN:               v.GetString("database_dsn"),
		DeleteWorkerCount: getIntWithFallback(v, "delete_worker_count", 2),
		DeleteChannelSize: getIntWithFallback(v, "delete_channel_size", 1000),
		SecretKey:         v.GetString("secret_key"),
		AuditFile:         v.GetString("audit_file"),
		AuditURL:          v.GetString("audit_url"),
		Enable_HTTPS:      v.GetBool("enable_https"),
		CertFile:          v.GetString("cert_file"),
		KeyFile:           v.GetString("key_file"),
		ConfigJSON:        configPath,
	}
}

// getIntWithFallback возвращает целочисленное значение ключа из Viper.
// Если строковое представление нельзя преобразовать в int (например, невалидная строка в env),
// возвращает fallback вместо нуля.
func getIntWithFallback(v *viper.Viper, key string, fallback int) int {
	if n, err := strconv.Atoi(v.GetString(key)); err == nil {
		return n
	}
	return fallback
}

// parseIntFromEnv преобразует строку переменной окружения в int.
func parseIntFromEnv(s string) (int, error) {
	return strconv.Atoi(s)
}

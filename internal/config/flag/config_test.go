package config

import (
	"testing"
)

const (
	ServerAddr = ":8081"
	EnvVar     = "prod"
	URL        = "http://default-host:8081"
	File       = "data.json"
	DSN        = "postgres://user:password@localhost:5432/dbname"
)

// TestNewServerConfig проводит табличные тесты для функции NewServerConfig.
// 1 - все переменные окружения переданы;
// 2 - переданы все переменные окружения, кроме адреса сервера;
// 3 - переменные окружения не переданы, используются флаги
// 4 - отсутствуют и переменные окружения и флаги. Используются значения по умолчанию
func TestNewServerConfig(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		want    ServerConfig
	}{
		{
			name: "All environments setup",
			envVars: map[string]string{
				"SERVER_ADDRESS":    ServerAddr,
				"BASE_URL":          URL,
				"ENV":               EnvVar,
				"FILE_STORAGE_PATH": File,
				"DATABASE_DSN":      DSN,
			},
			want: ServerConfig{
				AddrServer: ServerAddr,
				AddrURL:    URL,
				Env:        EnvVar,
				File:       File,
				DSN:        DSN,
			},
		},
		{
			name: "Missing server address",
			envVars: map[string]string{
				"BASE_URL":     URL,
				"ENV":          EnvVar,
				"DATABASE_DSN": DSN,
			},
			want: ServerConfig{
				AddrServer: ":8080",
				AddrURL:    URL,
				Env:        EnvVar,
				File:       File,
				DSN:        DSN,
			},
		},
		{
			name:    "Empty environments - use flags",
			envVars: map[string]string{},
			want: ServerConfig{
				AddrServer: ":8080",
				AddrURL:    "http://localhost:8080",
				Env:        "dev",
				File:       "data.json",
				DSN:        "postgres://user:password@localhost:5432/dbname",
			},
		},
		// по сути... тоже самое что и 3 тест,
		// потому что используются значения по умолчанию, которые и так указаны во флаге
		{
			name:    "Default parameters",
			envVars: map[string]string{},
			want: ServerConfig{
				AddrServer: ":8080",
				AddrURL:    "http://localhost:8080",
				Env:        "dev",
				File:       "data.json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

		for key, val := range tt.envVars {
			t.Setenv(key, val)
		}

			got := NewServerConfig()

			if got.AddrServer != tt.want.AddrServer {
				t.Errorf("AddrServer = %v, want %v", got.AddrServer, tt.want.AddrServer)
			}
			if got.AddrURL != tt.want.AddrURL {
				t.Errorf("AddrURL = %v, want %v", got.AddrURL, tt.want.AddrURL)
			}
			if got.Env != tt.want.Env {
				t.Errorf("Env = %v, want %v", got.Env, tt.want.Env)
			}
			if got.File != tt.want.File {
				t.Errorf("File = %v, want %v", got.File, tt.want.File)
			}
		})
	}
}

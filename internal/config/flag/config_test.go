package config

import (
	"os"
	"path/filepath"
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

func TestLoadJSONConfig(t *testing.T) {
	t.Run("valid JSON file", func(t *testing.T) {
		content := `{"server_address":"localhost:9090","base_url":"http://example.com","file_storage_path":"/tmp/test.db","database_dsn":"postgres://test","enable_https":true}`
		tmpFile := filepath.Join(t.TempDir(), "config.json")
		if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := loadJSONConfig(tmpFile)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.ServerAddress != "localhost:9090" {
			t.Errorf("ServerAddress = %v, want localhost:9090", cfg.ServerAddress)
		}
		if cfg.BaseURL != "http://example.com" {
			t.Errorf("BaseURL = %v, want http://example.com", cfg.BaseURL)
		}
		if cfg.FileStoragePath != "/tmp/test.db" {
			t.Errorf("FileStoragePath = %v, want /tmp/test.db", cfg.FileStoragePath)
		}
		if cfg.DatabaseDSN != "postgres://test" {
			t.Errorf("DatabaseDSN = %v, want postgres://test", cfg.DatabaseDSN)
		}
		if cfg.EnableHTTPS == nil || !*cfg.EnableHTTPS {
			t.Errorf("EnableHTTPS = %v, want true", cfg.EnableHTTPS)
		}
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := loadJSONConfig("/nonexistent/config.json")
		if err == nil {
			t.Error("expected error for missing file")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		tmpFile := filepath.Join(t.TempDir(), "bad.json")
		if err := os.WriteFile(tmpFile, []byte("{invalid"), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := loadJSONConfig(tmpFile)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

func TestParseIntFromEnv(t *testing.T) {
	if v, err := parseIntFromEnv("42"); err != nil || v != 42 {
		t.Errorf("parseIntFromEnv(\"42\") = %v, %v; want 42, nil", v, err)
	}
	if _, err := parseIntFromEnv("abc"); err == nil {
		t.Error("expected error for non-numeric string")
	}
}

func TestNewServerConfigWithJSONFile(t *testing.T) {
	content := `{"server_address":"json-host:3000","base_url":"http://json-url","file_storage_path":"/json/path.db","database_dsn":"json-dsn","enable_https":true}`
	tmpFile := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("JSON config used as base", func(t *testing.T) {
		t.Setenv("CONFIG", tmpFile)

		got := NewServerConfig()

		if got.AddrServer != "json-host:3000" {
			t.Errorf("AddrServer = %v, want json-host:3000", got.AddrServer)
		}
		if got.AddrURL != "http://json-url" {
			t.Errorf("AddrURL = %v, want http://json-url", got.AddrURL)
		}
		if got.File != "/json/path.db" {
			t.Errorf("File = %v, want /json/path.db", got.File)
		}
		if got.DSN != "json-dsn" {
			t.Errorf("DSN = %v, want json-dsn", got.DSN)
		}
		if !got.Enable_HTTPS {
			t.Errorf("Enable_HTTPS = %v, want true", got.Enable_HTTPS)
		}
	})

	t.Run("env vars override JSON config", func(t *testing.T) {
		t.Setenv("CONFIG", tmpFile)
		t.Setenv("SERVER_ADDRESS", ":9999")
		t.Setenv("BASE_URL", "http://env-url")
		t.Setenv("FILE_STORAGE_PATH", "/env/path.db")
		t.Setenv("DATABASE_DSN", "env-dsn")
		t.Setenv("ENABLE_HTTPS", "false")
		t.Setenv("DELETE_WORKER_COUNT", "5")
		t.Setenv("DELETE_CHANNEL_SIZE", "500")
		t.Setenv("SECRET_KEY", "env-secret")
		t.Setenv("AUDIT_FILE", "/env/audit.log")
		t.Setenv("AUDIT_URL", "http://env-audit")

		got := NewServerConfig()

		if got.AddrServer != ":9999" {
			t.Errorf("AddrServer = %v, want :9999", got.AddrServer)
		}
		if got.AddrURL != "http://env-url" {
			t.Errorf("AddrURL = %v, want http://env-url", got.AddrURL)
		}
		if got.File != "/env/path.db" {
			t.Errorf("File = %v, want /env/path.db", got.File)
		}
		if got.DSN != "env-dsn" {
			t.Errorf("DSN = %v, want env-dsn", got.DSN)
		}
		if got.Enable_HTTPS {
			t.Errorf("Enable_HTTPS = %v, want false", got.Enable_HTTPS)
		}
		if got.DeleteWorkerCount != 5 {
			t.Errorf("DeleteWorkerCount = %v, want 5", got.DeleteWorkerCount)
		}
		if got.DeleteChannelSize != 500 {
			t.Errorf("DeleteChannelSize = %v, want 500", got.DeleteChannelSize)
		}
		if got.SecretKey != "env-secret" {
			t.Errorf("SecretKey = %v, want env-secret", got.SecretKey)
		}
		if got.AuditFile != "/env/audit.log" {
			t.Errorf("AuditFile = %v, want /env/audit.log", got.AuditFile)
		}
		if got.AuditURL != "http://env-audit" {
			t.Errorf("AuditURL = %v, want http://env-audit", got.AuditURL)
		}
	})

	t.Run("invalid env ints fallback to flag defaults", func(t *testing.T) {
		t.Setenv("DELETE_WORKER_COUNT", "not-a-number")
		t.Setenv("DELETE_CHANNEL_SIZE", "not-a-number")

		got := NewServerConfig()

		if got.DeleteWorkerCount != 2 {
			t.Errorf("DeleteWorkerCount = %v, want 2 (flag default)", got.DeleteWorkerCount)
		}
		if got.DeleteChannelSize != 1000 {
			t.Errorf("DeleteChannelSize = %v, want 1000 (flag default)", got.DeleteChannelSize)
		}
	})
}

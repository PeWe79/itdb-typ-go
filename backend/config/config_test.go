package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// clearTestEnv 清理测试涉及的 ITDB 环境变量，避免外部环境干扰断言
func clearTestEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{"ITDB_SERVER_ADDR", "ADDR", "PORT", "ITDB_DB_PATH", "ITDB_UPLOAD_DIR", "ITDB_JWT_SECRET", "ITDB_HISTORY_LIMIT", "ITDB_SESSION_TTL_HOURS", "ITDB_CORS_ORIGINS"} {
		t.Setenv(key, "")
		os.Unsetenv(key)
	}
}

// TestLoadDefaults 校验无覆盖与无环境变量时使用默认配置
func TestLoadDefaults(t *testing.T) {
	clearTestEnv(t)
	cfg := LoadWithOverrides(nil)
	if cfg.ServerAddr != "127.0.0.1:8080" {
		t.Fatalf("ServerAddr = %q, want 127.0.0.1:8080", cfg.ServerAddr)
	}
	if cfg.DBPath != "./data/itdb.db" {
		t.Fatalf("DBPath = %q, want ./data/itdb.db", cfg.DBPath)
	}
	if cfg.UploadDir != "./data/files" {
		t.Fatalf("UploadDir = %q, want ./data/files", cfg.UploadDir)
	}
	if cfg.JWTSecret != "" {
		t.Fatalf("JWTSecret = %q, want empty", cfg.JWTSecret)
	}
	if cfg.HistoryLimit != 1000 {
		t.Fatalf("HistoryLimit = %d, want 1000", cfg.HistoryLimit)
	}
	if cfg.SessionTTL != 24*time.Hour {
		t.Fatalf("SessionTTL = %s, want 24h", cfg.SessionTTL)
	}
	if len(cfg.CORSOrigins) != 1 || cfg.CORSOrigins[0] != "*" {
		t.Fatalf("CORSOrigins = %v, want [*]", cfg.CORSOrigins)
	}
}

// TestLoadFromEnvironment 校验环境变量配置生效
func TestLoadFromEnvironment(t *testing.T) {
	clearTestEnv(t)
	t.Setenv("ITDB_SERVER_ADDR", "0.0.0.0:9000")
	t.Setenv("ITDB_DB_PATH", "/env/itdb.db")
	t.Setenv("ITDB_SESSION_TTL_HOURS", "48")
	t.Setenv("ITDB_CORS_ORIGINS", "https://a.example.com, https://b.example.com")
	cfg := LoadWithOverrides(nil)
	if cfg.ServerAddr != "0.0.0.0:9000" {
		t.Fatalf("ServerAddr = %q, want 0.0.0.0:9000", cfg.ServerAddr)
	}
	if cfg.DBPath != "/env/itdb.db" {
		t.Fatalf("DBPath = %q, want /env/itdb.db", cfg.DBPath)
	}
	if cfg.SessionTTL != 48*time.Hour {
		t.Fatalf("SessionTTL = %s, want 48h", cfg.SessionTTL)
	}
	if len(cfg.CORSOrigins) != 2 || cfg.CORSOrigins[0] != "https://a.example.com" {
		t.Fatalf("CORSOrigins = %v, want two origins", cfg.CORSOrigins)
	}
}

// TestLoadWithOverridesPriority 校验覆盖值优先于环境变量
func TestLoadWithOverridesPriority(t *testing.T) {
	clearTestEnv(t)
	t.Setenv("ITDB_DB_PATH", "/env/itdb.db")
	t.Setenv("ITDB_HISTORY_LIMIT", "2000")
	overrides := map[string]string{"db": "/cli/itdb.db", "history_limit": "500"}
	cfg := LoadWithOverrides(overrides)
	if cfg.DBPath != "/cli/itdb.db" {
		t.Fatalf("DBPath = %q, want /cli/itdb.db", cfg.DBPath)
	}
	if cfg.HistoryLimit != 500 {
		t.Fatalf("HistoryLimit = %d, want 500", cfg.HistoryLimit)
	}
}

// TestLoadInvalidNumbersIgnored 校验无效数字回退默认值
func TestLoadInvalidNumbersIgnored(t *testing.T) {
	clearTestEnv(t)
	t.Setenv("ITDB_SESSION_TTL_HOURS", "abc")
	t.Setenv("ITDB_HISTORY_LIMIT", "-5")
	cfg := LoadWithOverrides(nil)
	if cfg.SessionTTL != 24*time.Hour {
		t.Fatalf("SessionTTL = %s, want 24h", cfg.SessionTTL)
	}
	if cfg.HistoryLimit != 1000 {
		t.Fatalf("HistoryLimit = %d, want 1000", cfg.HistoryLimit)
	}
}

// TestLoadPortFallback 校验未设置完整地址时按 PORT 回退
func TestLoadPortFallback(t *testing.T) {
	clearTestEnv(t)
	t.Setenv("PORT", "9090")
	cfg := LoadWithOverrides(nil)
	if cfg.ServerAddr != "127.0.0.1:9090" {
		t.Fatalf("ServerAddr = %q, want 127.0.0.1:9090", cfg.ServerAddr)
	}
}

// TestLoadCustomEnvFile 校验覆盖键 env 指定的 .env 文件生效且不覆盖已有环境变量
func TestLoadCustomEnvFile(t *testing.T) {
	clearTestEnv(t)
	envFile := filepath.Join(t.TempDir(), "custom.env")
	content := "ITDB_DB_PATH=/envfile/itdb.db\nITDB_UPLOAD_DIR=/envfile/files\n"
	if err := os.WriteFile(envFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write env file: %v", err)
	}
	t.Setenv("ITDB_UPLOAD_DIR", "/existing/files")
	cfg := LoadWithOverrides(map[string]string{"env": envFile})
	if cfg.DBPath != "/envfile/itdb.db" {
		t.Fatalf("DBPath = %q, want /envfile/itdb.db", cfg.DBPath)
	}
	if cfg.UploadDir != "/existing/files" {
		t.Fatalf("UploadDir = %q, want /existing/files", cfg.UploadDir)
	}
}

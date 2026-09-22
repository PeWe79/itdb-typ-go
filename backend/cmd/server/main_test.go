package main

import (
	"strings"
	"testing"

	"itdb-backend/internal/buildinfo"
)

// TestVersionText 校验版本文本包含版本与构建信息
func TestVersionText(t *testing.T) {
	text := versionText()
	if !strings.HasPrefix(text, "itdb ") {
		t.Fatalf("version text prefix missing: %q", text)
	}
	if !strings.Contains(text, buildinfo.Version) || !strings.Contains(text, buildinfo.Commit) || !strings.Contains(text, buildinfo.BuildDate) {
		t.Fatalf("version text missing build info: %q", text)
	}
}

// TestOverridesFromFlags 校验显式参数映射为覆盖键值
func TestOverridesFromFlags(t *testing.T) {
	overrides := overridesFromFlags(flagValues{
		envPath:      "/opt/itdb/.env",
		addr:         "0.0.0.0:9000",
		dbPath:       "/tmp/itdb.db",
		uploadDir:    "/tmp/files",
		jwtSecret:    "secret",
		historyLimit: 500,
		sessionTTL:   12,
		corsOrigins:  "https://a.example.com",
	})
	want := map[string]string{
		"env":           "/opt/itdb/.env",
		"addr":          "0.0.0.0:9000",
		"db":            "/tmp/itdb.db",
		"upload_dir":    "/tmp/files",
		"jwt_secret":    "secret",
		"history_limit": "500",
		"session_ttl":   "12",
		"cors_origins":  "https://a.example.com",
	}
	for key, value := range want {
		if overrides[key] != value {
			t.Fatalf("overrides[%s] = %q, want %q", key, overrides[key], value)
		}
	}
}

// TestOverridesFromFlagsSkipsZero 校验零值参数不产生覆盖项
func TestOverridesFromFlagsSkipsZero(t *testing.T) {
	overrides := overridesFromFlags(flagValues{historyLimit: 0, sessionTTL: 0})
	if len(overrides) != 0 {
		t.Fatalf("overrides = %v, want empty", overrides)
	}
}

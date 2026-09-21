package auth

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"itdb-backend/internal/domain"
	"itdb-backend/internal/repository"
	"itdb-backend/internal/security"
	"itdb-backend/internal/service"
	"itdb-backend/router/settings"
)

func newLoginThrottleTestRouter(t *testing.T) *Router {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := settings.EnsureResourceSchema(db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, username TEXT NOT NULL, pass TEXT NOT NULL DEFAULT '', usertype INTEGER NOT NULL DEFAULT 1, userdesc TEXT NOT NULL DEFAULT '')"); err != nil {
		t.Fatal(err)
	}
	hashed, err := security.HashPassword("correct-password")
	if err != nil {
		t.Fatal(err)
	}
	for _, seed := range []string{
		"INSERT INTO users (id,username,pass,usertype,userdesc) VALUES (1,'alice','" + hashed + "',1,'Alice')",
		"INSERT INTO users (id,username,pass,usertype,userdesc) VALUES (2,'admin','" + hashed + "',0,'Administrator')",
	} {
		if _, err := db.Exec(seed); err != nil {
			t.Fatal(err)
		}
	}
	authRepo := repository.NewAuthRepository(repository.NewStore(db))
	return &Router{
		db:           db,
		domains:      service.NewDomainServices(repository.NewStore(db)),
		authWorkflow: service.NewAuthWorkflow(authRepo, "jwt-test-secret", nil, 24*time.Hour),
	}
}

func loginBody(username, password string) []byte {
	body, _ := json.Marshal(map[string]string{"username": username, "password": password, "mode": "local"})
	return body
}

// TestLoginLocksAfterConfiguredFailures 验证连续密码失败达到阈值后锁定，admin 不受限
func TestLoginLocksAfterConfiguredFailures(t *testing.T) {
	app := newLoginThrottleTestRouter(t)
	for i := 0; i < 5; i++ {
		rec := httptest.NewRecorder()
		app.handleLogin(rec, httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody("alice", "wrong-password"))))
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("failure %d status = %d, want 401", i+1, rec.Code)
		}
	}
	rec := httptest.NewRecorder()
	app.handleLogin(rec, httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody("alice", "correct-password"))))
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("locked status = %d, want 429, body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "密码连续错误次数过多") {
		t.Fatalf("locked body missing message: %s", rec.Body.String())
	}

	adminRec := httptest.NewRecorder()
	app.handleLogin(adminRec, httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody("admin", "correct-password"))))
	if adminRec.Code != http.StatusOK {
		t.Fatalf("admin login status = %d, want 200, body=%s", adminRec.Code, adminRec.Body.String())
	}
}

// TestLoginLockoutExpiresAndClearsOnSuccess 验证锁定窗口过期后可重新登录且成功后计数清零
func TestLoginLockoutExpiresAndClearsOnSuccess(t *testing.T) {
	app := newLoginThrottleTestRouter(t)
	stale := time.Now().Unix() - 3600
	for i := 0; i < 5; i++ {
		if _, err := app.db.Exec("INSERT INTO login_failure_log(username,failed_at) VALUES('alice',?)", stale); err != nil {
			t.Fatal(err)
		}
	}
	rec := httptest.NewRecorder()
	app.handleLogin(rec, httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody("alice", "correct-password"))))
	if rec.Code != http.StatusOK {
		t.Fatalf("login after lockout window status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var count int
	if err := app.db.QueryRow("SELECT COUNT(*) FROM login_failure_log WHERE username='alice'").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("login_failure_log count = %d, want 0", count)
	}
}

// TestLoginLockoutConfigValidation 验证锁定配置的归一化与校验范围
func TestLoginLockoutConfigValidation(t *testing.T) {
	config := domain.NormalizeSystemBaseConfig(domain.SystemBaseConfig{})
	if config.LoginMaxFailures != 5 || config.LoginLockoutMinutes != 2 {
		t.Fatalf("defaults = %d/%d, want 5/2", config.LoginMaxFailures, config.LoginLockoutMinutes)
	}
	config.LoginMaxFailures = 2
	config.LoginLockoutMinutes = 11
	if err := domain.ValidateSystemBaseConfig(config); err == nil {
		t.Fatal("ValidateSystemBaseConfig should reject maxFailures=2 and lockMinutes=11")
	}
}

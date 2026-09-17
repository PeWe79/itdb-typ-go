package auth

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"itdb-backend/router/settings"
)

func newResetTestDB(t *testing.T, dsn string) *Router {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+dsn+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := settings.EnsureResourceSchema(db); err != nil {
		t.Fatal(err)
	}
	return &Router{db: db}
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func TestPasswordResetSendRateLimit(t *testing.T) {
	app := newResetTestDB(t, "password-reset-send-limit")
	now := time.Now().Unix()
	if _, err := app.db.Exec("INSERT INTO password_reset_requests(token_hash,user_id,email,expires_at,used,last_sent_at,created_at) VALUES(?,?,?,?,?,?,?)", hashToken("verification"), 1, "alice@example.com", now+600, 0, now-3600, now-3600); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < defaultResetRateLimitMax; i++ {
		if _, err := app.db.Exec("INSERT INTO password_reset_send_log(email,sent_at) VALUES(?,?)", "alice@example.com", now-int64(i)*60); err != nil {
			t.Fatal(err)
		}
	}
	payload := []byte(`{"verificationToken":"verification","channel":"email","verifyEmail":"alice@example.com"}`)
	response := httptest.NewRecorder()
	app.handlePasswordResetSend(response, httptest.NewRequest(http.MethodPost, "/api/auth/password-reset/send", bytes.NewReader(payload)))
	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("rate limited status = %d, want 429, body=%s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "验证码请求过于频繁，请于 1 分钟后再试") {
		t.Fatalf("rate limited body missing countdown message: %s", response.Body.String())
	}
}

func TestPasswordResetCaptchaUsesConfiguredTTL(t *testing.T) {
	app := newResetTestDB(t, "password-reset-send-limit")
	raw, err := json.Marshal(map[string]any{"resetCaptchaTtlMinutes": 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.db.Exec("UPDATE settings_base SET config=? WHERE id=1", string(raw)); err != nil {
		t.Fatal(err)
	}
	before := time.Now().Unix()
	response := httptest.NewRecorder()
	app.handlePasswordResetCaptcha(response, httptest.NewRequest(http.MethodGet, "/api/auth/password-reset/captcha", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("captcha status = %d", response.Code)
	}
	var body struct {
		ExpiresAt string `json:"expiresAt"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	expiresAt, err := time.Parse(time.RFC3339, body.ExpiresAt)
	if err != nil {
		t.Fatal(err)
	}
	delta := expiresAt.Unix() - before
	if delta < 55 || delta > 65 {
		t.Fatalf("captcha expires delta = %d, want about 60", delta)
	}
}

// TestAuthUserResponseIncludesEffectivePermissions 验证用户响应聚合角色权限
func TestAuthUserResponseIncludesEffectivePermissions(t *testing.T) {
	app := newResetTestDB(t, "password-reset-send-limit")
	if _, err := app.db.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, username TEXT, userdesc TEXT, pass TEXT, usertype INTEGER)"); err != nil {
		t.Fatal(err)
	}
	if _, err := app.db.Exec("INSERT INTO users(id,username,userdesc,usertype) VALUES(1,'alice','Alice',1)"); err != nil {
		t.Fatal(err)
	}
	if _, err := app.db.Exec("UPDATE settings_roles SET permissions='[\"assets.items.read\"]' WHERE key='operator'"); err != nil {
		t.Fatal(err)
	}
	if _, err := app.db.Exec("INSERT INTO settings_user_roles(user_id,role_id) SELECT 1,id FROM settings_roles WHERE key='operator'"); err != nil {
		t.Fatal(err)
	}
	response, err := app.authUserResponse(context.Background(), SessionUser{ID: 1, Username: "alice", UserType: 1})
	if err != nil {
		t.Fatal(err)
	}
	permissions, ok := response["permissions"].([]string)
	if !ok || !containsString(permissions, "assets.items.read") {
		t.Fatalf("permissions = %#v", response["permissions"])
	}
	if response["role"] != "operator" {
		t.Fatalf("role = %#v", response["role"])
	}
}

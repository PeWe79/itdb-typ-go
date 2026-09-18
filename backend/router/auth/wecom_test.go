package auth

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"itdb-backend/config"
	"itdb-backend/internal/domain"
	"itdb-backend/internal/repository"
	"itdb-backend/internal/security"
	"itdb-backend/internal/service"
	"itdb-backend/router/common"
	"itdb-backend/router/settings"

	_ "modernc.org/sqlite"
)

// newWecomTestRouter 构建内存库测试路由：启用企业微信配置并使用可替换的授权码换取实现
func newWecomTestRouter(t *testing.T) *Router {
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
	encrypted, err := security.EncryptSettingsSecret("test-secret")
	if err != nil {
		t.Fatal(err)
	}
	providerConfig, err := json.Marshal(map[string]string{
		"corpid": "ww123", "agentid": "1000002",
		"redirectPrefix": "https://itdb.example.com", "secret": encrypted,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, seed := range []string{
		"INSERT INTO users (id,username,pass,usertype,userdesc) VALUES (1,'alice','x',1,'Alice')",
		"INSERT INTO users (id,username,pass,usertype,userdesc) VALUES (2,'bob','x',1,'Bob')",
	} {
		if _, err := db.Exec(seed); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec("UPDATE settings_auth_providers SET enabled=1, config=? WHERE id='wecom'", string(providerConfig)); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{JWTSecret: "jwt-test-secret"}
	authRepo := repository.NewAuthRepository(repository.NewStore(db))
	return &Router{
		db:           db,
		domains:      service.NewDomainServices(repository.NewStore(db)),
		authWorkflow: service.NewAuthWorkflow(authRepo, cfg.JWTSecret, nil),
		cfg:          cfg,
		wecomExchange: func(_ context.Context, _ *service.WecomProvider, code string) (string, error) {
			if code == "good-code" {
				return "zhangsan-wecom-id", nil
			}
			return "", errors.New("企业微信授权码无效或已使用，请重新扫码")
		},
	}
}

func wecomUserContext(userID int64, username string) context.Context {
	return common.WithUser(context.Background(), domain.SessionUser{ID: userID, Username: username})
}

// TestWecomBindAndCallbackFlow 校验绑定、扫码登录、冲突拦截与解绑的完整流程。
func TestWecomBindAndCallbackFlow(t *testing.T) {
	app := newWecomTestRouter(t)

	bindState, err := service.SignWecomState("jwt-test-secret", service.WecomState{Purpose: service.WecomBindPurpose, UserID: 1, Exp: time.Now().Add(time.Minute).Unix()})
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	app.handleWecomBind(rec, httptest.NewRequest(http.MethodPost, "/api/auth/wecom/bind", bytes.NewReader(wecomBody("good-code", bindState))).WithContext(wecomUserContext(1, "alice")))
	if rec.Code != http.StatusOK {
		t.Fatalf("bind status=%d body=%s", rec.Code, rec.Body.String())
	}
	var bindCount int
	if err := app.db.QueryRow("SELECT COUNT(*) FROM settings_user_wecom WHERE user_id=1 AND wecom_userid='zhangsan-wecom-id'").Scan(&bindCount); err != nil {
		t.Fatal(err)
	}
	if bindCount != 1 {
		t.Fatalf("bindCount=%d", bindCount)
	}

	loginState, err := service.SignWecomState("jwt-test-secret", service.WecomState{Purpose: service.WecomLoginPurpose, Exp: time.Now().Add(time.Minute).Unix()})
	if err != nil {
		t.Fatal(err)
	}
	rec = httptest.NewRecorder()
	app.handleWecomCallback(rec, httptest.NewRequest(http.MethodPost, "/api/auth/wecom/callback", bytes.NewReader(wecomBody("good-code", loginState))))
	if rec.Code != http.StatusOK {
		t.Fatalf("callback status=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Token string `json:"token"`
		User  struct {
			Username   string `json:"username"`
			Source     string `json:"source"`
			WecomBound bool   `json:"wecomBound"`
		} `json:"user"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.User.Username != "alice" || payload.User.Source != "wecom" || payload.Token == "" {
		t.Fatalf("payload=%+v", payload)
	}
	if !payload.User.WecomBound {
		t.Fatal("wecomBound should be true for scan-login user")
	}

	conflictState, err := service.SignWecomState("jwt-test-secret", service.WecomState{Purpose: service.WecomBindPurpose, UserID: 2, Exp: time.Now().Add(time.Minute).Unix()})
	if err != nil {
		t.Fatal(err)
	}
	rec = httptest.NewRecorder()
	app.handleWecomBind(rec, httptest.NewRequest(http.MethodPost, "/api/auth/wecom/bind", bytes.NewReader(wecomBody("good-code", conflictState))).WithContext(wecomUserContext(2, "bob")))
	if rec.Code != http.StatusConflict {
		t.Fatalf("conflict status=%d body=%s", rec.Code, rec.Body.String())
	}

	unboundState, err := service.SignWecomState("jwt-test-secret", service.WecomState{Purpose: service.WecomLoginPurpose, Exp: time.Now().Add(time.Minute).Unix()})
	if err != nil {
		t.Fatal(err)
	}
	rec = httptest.NewRecorder()
	app.handleWecomCallback(rec, httptest.NewRequest(http.MethodPost, "/api/auth/wecom/callback", bytes.NewReader(wecomBody("other-code", unboundState))))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unbound status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	app.handleWecomUnbind(rec, httptest.NewRequest(http.MethodDelete, "/api/auth/wecom/bind", nil).WithContext(wecomUserContext(1, "alice")))
	if rec.Code != http.StatusOK {
		t.Fatalf("unbind status=%d body=%s", rec.Code, rec.Body.String())
	}
	var remain int
	if err := app.db.QueryRow("SELECT COUNT(*) FROM settings_user_wecom").Scan(&remain); err != nil {
		t.Fatal(err)
	}
	if remain != 0 {
		t.Fatalf("remain=%d", remain)
	}
}

// TestWecomCallbackRejectsBindState 校验登录回调拒绝绑定用途的 state，避免跨用途混用。
func TestWecomCallbackRejectsBindState(t *testing.T) {
	app := newWecomTestRouter(t)
	bindState, err := service.SignWecomState("jwt-test-secret", service.WecomState{Purpose: service.WecomBindPurpose, UserID: 1, Exp: time.Now().Add(time.Minute).Unix()})
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	app.handleWecomCallback(rec, httptest.NewRequest(http.MethodPost, "/api/auth/wecom/callback", bytes.NewReader(wecomBody("good-code", bindState))))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func wecomBody(code, state string) []byte {
	body, _ := json.Marshal(map[string]string{"code": code, "state": state})
	return body
}

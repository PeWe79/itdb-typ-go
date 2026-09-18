package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"itdb-backend/internal/security"
	"itdb-backend/internal/service"
)

// newWecomSSOTestRouter 在直连测试路由基础上切换为统一认证模式并注入 ticket 校验实现
func newWecomSSOTestRouter(t *testing.T) *Router {
	t.Helper()
	app := newWecomTestRouter(t)
	encrypted, err := security.EncryptSettingsSecret("sso-secret")
	if err != nil {
		t.Fatal(err)
	}
	providerConfig, err := json.Marshal(map[string]string{
		"authMode": "sso", "ssoBaseUrl": "https://auth.example.com", "ssoAppID": "itdb", "ssoAppSecret": encrypted,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.db.Exec("UPDATE settings_auth_providers SET enabled=1, config=? WHERE id='wecom'", string(providerConfig)); err != nil {
		t.Fatal(err)
	}
	app.wecomSSOVerify = func(_ context.Context, _ *service.WecomProvider, ticket string) (string, error) {
		switch ticket {
		case "good-ticket":
			return "zhangsan-wecom-id", nil
		case "other-ticket":
			return "wang-unbound-id", nil
		default:
			return "", errors.New("登录凭证无效或已过期，请重新发起登录")
		}
	}
	return app
}

func wecomSSOBody(ticket string) []byte {
	body, _ := json.Marshal(map[string]string{"ticket": ticket})
	return body
}

// TestWecomSSOBindAndLoginFlow 校验统一认证模式下绑定、扫码登录、冲突拦截与解绑的完整流程。
func TestWecomSSOBindAndLoginFlow(t *testing.T) {
	app := newWecomSSOTestRouter(t)

	rec := httptest.NewRecorder()
	app.handleWecomSSOBind(rec, httptest.NewRequest(http.MethodPost, "/api/auth/wecom/sso/bind", bytes.NewReader(wecomSSOBody("good-ticket"))).WithContext(wecomUserContext(1, "alice")))
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

	rec = httptest.NewRecorder()
	app.handleWecomSSOCallback(rec, httptest.NewRequest(http.MethodPost, "/api/auth/wecom/sso/callback", bytes.NewReader(wecomSSOBody("good-ticket"))))
	if rec.Code != http.StatusOK {
		t.Fatalf("callback status=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Token string `json:"token"`
		User  struct {
			Username string `json:"username"`
			Source   string `json:"source"`
		} `json:"user"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.User.Username != "alice" || payload.User.Source != "wecom" || payload.Token == "" {
		t.Fatalf("payload=%+v", payload)
	}

	rec = httptest.NewRecorder()
	app.handleWecomSSOBind(rec, httptest.NewRequest(http.MethodPost, "/api/auth/wecom/sso/bind", bytes.NewReader(wecomSSOBody("good-ticket"))).WithContext(wecomUserContext(2, "bob")))
	if rec.Code != http.StatusConflict {
		t.Fatalf("conflict status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	app.handleWecomSSOCallback(rec, httptest.NewRequest(http.MethodPost, "/api/auth/wecom/sso/callback", bytes.NewReader(wecomSSOBody("other-ticket"))))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unbound status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	app.handleWecomSSOCallback(rec, httptest.NewRequest(http.MethodPost, "/api/auth/wecom/sso/callback", bytes.NewReader(wecomSSOBody("bad-ticket"))))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("invalid ticket status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	app.handleWecomUnbind(rec, httptest.NewRequest(http.MethodDelete, "/api/auth/wecom/bind", nil).WithContext(wecomUserContext(1, "alice")))
	if rec.Code != http.StatusOK {
		t.Fatalf("unbind status=%d body=%s", rec.Code, rec.Body.String())
	}
}

// TestWecomSSOAuthorizeURL 校验统一认证模式下扫码登录地址跳转认证中心。
func TestWecomSSOAuthorizeURL(t *testing.T) {
	app := newWecomSSOTestRouter(t)
	rec := httptest.NewRecorder()
	app.handleWecomAuthorize(rec, httptest.NewRequest(http.MethodGet, "/api/auth/wecom/authorize", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("authorize status=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.URL != "https://auth.example.com/login?app=itdb" {
		t.Fatalf("url=%q", payload.URL)
	}
}

// TestWecomSSOEndpointsRejectedInDirectMode 校验直连模式下统一认证端点被拒绝。
func TestWecomSSOEndpointsRejectedInDirectMode(t *testing.T) {
	app := newWecomTestRouter(t)
	rec := httptest.NewRecorder()
	app.handleWecomSSOCallback(rec, httptest.NewRequest(http.MethodPost, "/api/auth/wecom/sso/callback", bytes.NewReader(wecomSSOBody("good-ticket"))))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("callback status=%d body=%s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	app.handleWecomSSOBind(rec, httptest.NewRequest(http.MethodPost, "/api/auth/wecom/sso/bind", bytes.NewReader(wecomSSOBody("good-ticket"))).WithContext(wecomUserContext(1, "alice")))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bind status=%d body=%s", rec.Code, rec.Body.String())
	}
}

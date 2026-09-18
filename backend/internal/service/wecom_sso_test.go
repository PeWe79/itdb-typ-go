package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func ssoTestProvider(baseURL string) *WecomProvider {
	return &WecomProvider{
		AuthMode:     WecomAuthModeSSO,
		SSOBaseURL:   baseURL,
		SSOAppID:     "itdb",
		SSOAppSecret: "sso-secret-32-bytes-aaaaaaaaaaaa",
	}
}

// TestWecomSSOVerifyTicketSuccess 校验 verify 请求拼装、HMAC 签名与 userid 提取。
func TestWecomSSOVerifyTicketSuccess(t *testing.T) {
	frozen := time.Unix(1726650000, 0)
	originalNow := wecomNowFunc
	wecomNowFunc = func() time.Time { return frozen }
	t.Cleanup(func() { wecomNowFunc = originalNow })

	provider := ssoTestProvider("http://auth.example.com")
	var gotBody struct {
		App    string `json:"app"`
		Ticket string `json:"ticket"`
		TS     int64  `json:"ts"`
		Sign   string `json:"sign"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/verify" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode body: %v", err)
		}
		mac := hmac.New(sha256.New, []byte(provider.SSOAppSecret))
		fmt.Fprintf(mac, "%s\n%s\n%d", gotBody.App, gotBody.Ticket, gotBody.TS)
		if gotBody.Sign != hex.EncodeToString(mac.Sum(nil)) {
			t.Error("signature mismatch")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"userid":"zhangsan","name":"张三"}`))
	}))
	defer server.Close()
	provider.SSOBaseURL = server.URL

	userid, err := provider.WecomSSOVerifyTicket(context.Background(), " ticket-1 ")
	if err != nil {
		t.Fatal(err)
	}
	if userid != "zhangsan" {
		t.Fatalf("userid=%q", userid)
	}
	if gotBody.App != "itdb" || gotBody.Ticket != "ticket-1" || gotBody.TS != frozen.Unix() {
		t.Fatalf("body=%+v", gotBody)
	}
}

// TestWecomSSOVerifyTicketErrorMapping 校验认证中心错误码到中文提示的转译。
func TestWecomSSOVerifyTicketErrorMapping(t *testing.T) {
	cases := map[string]string{
		"invalid_app":    "统一认证中心未注册本系统的应用标识，请检查应用标识配置",
		"invalid_sign":   "统一认证中心签名校验失败，请检查应用密钥配置",
		"expired_ts":     "与统一认证中心服务器时间偏差过大，请校准系统时钟后重试",
		"invalid_ticket": "登录凭证无效或已过期，请重新发起登录",
		"unknown_code":   "统一认证中心登录校验失败",
	}
	for code, expected := range cases {
		code, expected := code, expected
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": code})
		}))
		provider := ssoTestProvider(server.URL)
		_, err := provider.WecomSSOVerifyTicket(context.Background(), "ticket-1")
		server.Close()
		if err == nil || err.Error() != expected {
			t.Fatalf("code=%s err=%v", code, err)
		}
	}
}

// TestWecomSSOVerifyTicketEdgeCases 校验空 ticket、空 userid 与登录入口地址拼装。
func TestWecomSSOVerifyTicketEdgeCases(t *testing.T) {
	provider := ssoTestProvider("http://auth.example.com")
	if _, err := provider.WecomSSOVerifyTicket(context.Background(), "  "); err == nil {
		t.Fatal("expected error for empty ticket")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"userid":"","name":""}`))
	}))
	defer server.Close()
	provider.SSOBaseURL = server.URL
	if _, err := provider.WecomSSOVerifyTicket(context.Background(), "ticket-1"); err == nil {
		t.Fatal("expected error for empty userid")
	}

	if got, want := ssoTestProvider("https://auth.example.com/").WecomSSOLoginURL(), "https://auth.example.com/login?app=itdb"; got != want {
		t.Fatalf("loginURL=%q want=%q", got, want)
	}
}

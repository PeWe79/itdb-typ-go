package service

import (
	"strings"
	"testing"
	"time"
)

// TestWecomStateRoundTrip 校验 state 签发与校验的往返一致性及篡改、过期、用途不匹配拦截。
func TestWecomStateRoundTrip(t *testing.T) {
	signed, err := SignWecomState("test-secret", WecomState{Purpose: WecomBindPurpose, UserID: 7, Exp: time.Now().Add(time.Minute).Unix()})
	if err != nil {
		t.Fatal(err)
	}
	state, err := VerifyWecomState("test-secret", signed, WecomBindPurpose)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if state.UserID != 7 || state.Purpose != WecomBindPurpose {
		t.Fatalf("state=%+v", state)
	}

	if _, err := VerifyWecomState("test-secret", signed, WecomLoginPurpose); err == nil {
		t.Fatal("expected purpose mismatch to fail")
	}
	if _, err := VerifyWecomState("other-secret", signed, WecomBindPurpose); err == nil {
		t.Fatal("expected secret mismatch to fail")
	}
	tampered := signed[:len(signed)-2] + "zz"
	if _, err := VerifyWecomState("test-secret", tampered, WecomBindPurpose); err == nil {
		t.Fatal("expected tampered state to fail")
	}
	expired, err := SignWecomState("test-secret", WecomState{Purpose: WecomLoginPurpose, Exp: time.Now().Add(-time.Second).Unix()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyWecomState("test-secret", expired, WecomLoginPurpose); err == nil {
		t.Fatal("expected expired state to fail")
	}
}

// TestWecomAuthorizeURL 校验扫码登录地址的参数拼装与回调地址规范化。
func TestWecomAuthorizeURL(t *testing.T) {
	provider := &WecomProvider{
		CorpID: "ww1234567890", AgentID: "1000002", Secret: "s3cret",
		RedirectPrefix: "https://itdb.example.com/",
	}
	target := provider.WecomAuthorizeURL("state-abc")
	for _, part := range []string{
		"login_type=CorpApp", "appid=ww1234567890", "agentid=1000002",
		"redirect_uri=https%3A%2F%2Fitdb.example.com%2Flogin", "state=state-abc",
	} {
		if !strings.Contains(target, part) {
			t.Fatalf("url %s missing %s", target, part)
		}
	}
	if provider.WecomRedirectURI() != "https://itdb.example.com/login" {
		t.Fatalf("redirect=%s", provider.WecomRedirectURI())
	}
	if !provider.HasWecomCredential() {
		t.Fatal("expected credential complete")
	}
	incomplete := &WecomProvider{CorpID: "ww1"}
	if incomplete.HasWecomCredential() {
		t.Fatal("expected credential incomplete")
	}
}

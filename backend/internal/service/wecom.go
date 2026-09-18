package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"itdb-backend/internal/security"
)

// 企业微信认证：配置存于 settings_auth_providers 表 id='wecom' 行，
// Secret 以 enc:v1 加密保存；扫码登录遵循企业微信 OAuth 授权码流程
const (
	WecomProviderID   = "wecom"
	WecomLoginPurpose = "login"
	WecomBindPurpose  = "bind"

	// WecomStateTTL 授权 state 的有效期，超时需重新发起扫码
	WecomStateTTL = 5 * time.Minute

	wecomAuthorizeURL = "https://login.work.weixin.qq.com/wwlogin/sso/login"
	wecomTokenURL     = "https://qyapi.weixin.qq.com/cgi-bin/gettoken"
	wecomUserInfoURL  = "https://qyapi.weixin.qq.com/cgi-bin/auth/getuserinfo"
)

var ErrWecomNotBound = errors.New("该企业微信账号尚未绑定系统用户，请先使用账号密码登录后在右上角绑定企微")

// WecomProvider 企业微信认证提供者配置快照
type WecomProvider struct {
	Name           string
	Enabled        bool
	CorpID         string
	AgentID        string
	Secret         string
	RedirectPrefix string
}

// LoadWecomProvider 读取企业微信认证配置并解密 Secret，未配置时返回 nil
func LoadWecomProvider(ctx context.Context, db *sql.DB) (*WecomProvider, error) {
	var name, config string
	var enabled int64
	err := db.QueryRowContext(ctx,
		"SELECT name,enabled,COALESCE(config,'') FROM settings_auth_providers WHERE id=?", WecomProviderID,
	).Scan(&name, &enabled, &config)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	provider := &WecomProvider{Name: name, Enabled: enabled != 0}
	if strings.TrimSpace(config) == "" {
		return provider, nil
	}
	var stored map[string]any
	if err := json.Unmarshal([]byte(config), &stored); err != nil {
		return provider, nil
	}
	provider.CorpID = wecomConfigString(stored, "corpid")
	provider.AgentID = wecomConfigString(stored, "agentid")
	provider.RedirectPrefix = wecomConfigString(stored, "redirectPrefix")
	if secret := wecomConfigString(stored, "secret"); secret != "" {
		plain, err := security.DecryptSettingsSecret(secret, "")
		if err != nil {
			return nil, fmt.Errorf("企业微信 Secret 解密失败: %w", err)
		}
		provider.Secret = plain
	}
	return provider, nil
}

// HasWecomCredential 配置是否具备发起扫码登录所需的完整凭据（回调地址前缀可留空按访问地址推断）
func (p *WecomProvider) HasWecomCredential() bool {
	return p != nil && p.CorpID != "" && p.AgentID != "" && p.Secret != ""
}

// WecomRedirectURI 回调落地页固定为前端登录路由：配置了前缀用前缀，否则按当前访问地址推断
func (p *WecomProvider) WecomRedirectURI(baseURL string) string {
	if p != nil && p.RedirectPrefix != "" {
		return strings.TrimRight(p.RedirectPrefix, "/") + "/login"
	}
	return strings.TrimRight(baseURL, "/") + "/login"
}

// WecomAuthorizeURL 构造企业微信 Web 扫码登录页地址
func (p *WecomProvider) WecomAuthorizeURL(baseURL, state string) string {
	query := url.Values{}
	query.Set("login_type", "CorpApp")
	query.Set("appid", p.CorpID)
	query.Set("agentid", p.AgentID)
	query.Set("redirect_uri", p.WecomRedirectURI(baseURL))
	query.Set("state", state)
	return wecomAuthorizeURL + "?" + query.Encode()
}

// WecomRequestBaseURL 从当前请求推断站点外部访问地址：优先取反向代理的 X-Forwarded-Proto
func WecomRequestBaseURL(r *http.Request) string {
	scheme := "http"
	if proto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); proto != "" {
		scheme = proto
	} else if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

// WecomState OAuth state 载荷：防 CSRF 的签名数据，区分登录与绑定用途
type WecomState struct {
	Purpose string `json:"p"`
	UserID  int64  `json:"u,omitempty"`
	Exp     int64  `json:"e"`
	Nonce   string `json:"n"`
}

// SignWecomState 签发带过期时间的 HMAC 签名 state
func SignWecomState(secret string, state WecomState) (string, error) {
	if strings.TrimSpace(secret) == "" {
		return "", errors.New("missing signing secret")
	}
	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	state.Nonce = hex.EncodeToString(nonce)
	payload, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	body := base64.RawURLEncoding.EncodeToString(payload)
	return body + "." + wecomHMAC(secret, body), nil
}

// VerifyWecomState 校验 state 签名与有效期，并要求用途匹配
func VerifyWecomState(secret, state, purpose string) (*WecomState, error) {
	body, signature, found := strings.Cut(state, ".")
	if !found {
		return nil, errors.New("授权状态无效，请重新扫码")
	}
	if !hmac.Equal([]byte(wecomHMAC(secret, body)), []byte(signature)) {
		return nil, errors.New("授权状态无效，请重新扫码")
	}
	payload, err := base64.RawURLEncoding.DecodeString(body)
	if err != nil {
		return nil, errors.New("授权状态无效，请重新扫码")
	}
	var decoded WecomState
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, errors.New("授权状态无效，请重新扫码")
	}
	if decoded.Purpose != purpose {
		return nil, errors.New("授权用途不匹配，请重新发起")
	}
	if time.Now().Unix() > decoded.Exp {
		return nil, errors.New("授权已过期，请重新扫码")
	}
	return &decoded, nil
}

func wecomHMAC(secret, body string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	return hex.EncodeToString(mac.Sum(nil))
}

type wecomCachedToken struct {
	token     string
	expiresAt time.Time
}

var (
	wecomTokenMu     sync.Mutex
	wecomTokenCache  = map[string]wecomCachedToken{}
	wecomHTTPClient  = &http.Client{Timeout: 10 * time.Second}
	wecomNowFunc     = time.Now
	wecomTokenExpiry = 7000 * time.Second
)

// WecomExchangeCode 用授权码换取企业微信成员 userid
func (p *WecomProvider) WecomExchangeCode(ctx context.Context, code string) (string, error) {
	userid, err := p.exchangeCode(ctx, code)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(userid) == "" {
		return "", errors.New("企业微信接口未返回成员信息")
	}
	return userid, nil
}

func (p *WecomProvider) exchangeCode(ctx context.Context, code string) (string, error) {
	token, err := p.cachedAccessToken(ctx)
	if err != nil {
		return "", err
	}
	userid, werr := p.requestUserInfo(ctx, token, code)
	if werr != nil && errors.Is(werr, errWecomTokenInvalid) {
		token, terr := p.fetchAccessToken(ctx)
		if terr != nil {
			return "", terr
		}
		return p.requestUserInfo(ctx, token, code)
	}
	return userid, werr
}

func (p *WecomProvider) cachedAccessToken(ctx context.Context) (string, error) {
	cacheKey := wecomCredentialKey(p.CorpID, p.Secret)
	wecomTokenMu.Lock()
	cached, ok := wecomTokenCache[cacheKey]
	wecomTokenMu.Unlock()
	if ok && wecomNowFunc().Before(cached.expiresAt) {
		return cached.token, nil
	}
	token, err := p.fetchAccessToken(ctx)
	if err != nil {
		return "", err
	}
	wecomTokenMu.Lock()
	wecomTokenCache[cacheKey] = wecomCachedToken{token: token, expiresAt: wecomNowFunc().Add(wecomTokenExpiry)}
	wecomTokenMu.Unlock()
	return token, nil
}

func (p *WecomProvider) fetchAccessToken(ctx context.Context) (string, error) {
	query := url.Values{}
	query.Set("corpid", p.CorpID)
	query.Set("corpsecret", p.Secret)
	var payload struct {
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
		AccessToken string `json:"access_token"`
	}
	if err := wecomGetJSON(ctx, wecomTokenURL+"?"+query.Encode(), &payload); err != nil {
		return "", err
	}
	if payload.ErrCode != 0 || payload.AccessToken == "" {
		return "", fmt.Errorf("获取企业微信访问凭证失败: %s", wecomErrMessage(payload.ErrCode, payload.ErrMsg))
	}
	return payload.AccessToken, nil
}

var errWecomTokenInvalid = errors.New("wecom access token invalid")

func (p *WecomProvider) requestUserInfo(ctx context.Context, token, code string) (string, error) {
	query := url.Values{}
	query.Set("access_token", token)
	query.Set("code", code)
	var payload struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
		UserID  string `json:"userid"`
	}
	if err := wecomGetJSON(ctx, wecomUserInfoURL+"?"+query.Encode(), &payload); err != nil {
		return "", err
	}
	if payload.ErrCode == 40014 || payload.ErrCode == 42001 {
		wecomTokenMu.Lock()
		delete(wecomTokenCache, wecomCredentialKey(p.CorpID, p.Secret))
		wecomTokenMu.Unlock()
		return "", errWecomTokenInvalid
	}
	if payload.ErrCode != 0 {
		return "", fmt.Errorf("%s", wecomErrMessage(payload.ErrCode, payload.ErrMsg))
	}
	return payload.UserID, nil
}

func wecomGetJSON(ctx context.Context, target string, payload any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return fmt.Errorf("构造企业微信请求失败: %w", err)
	}
	response, err := wecomHTTPClient.Do(request)
	if err != nil {
		return fmt.Errorf("连接企业微信接口失败: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("读取企业微信接口响应失败: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("企业微信接口响应异常: HTTP %d", response.StatusCode)
	}
	if err := json.Unmarshal(body, payload); err != nil {
		return fmt.Errorf("解析企业微信接口响应失败: %w", err)
	}
	return nil
}

// wecomErrMessage 将常见企业微信错误码转译为用户可读的提示
func wecomErrMessage(code int, message string) string {
	if text := strings.TrimSpace(message); text != "" && code != 40029 {
		return fmt.Sprintf("%s（错误码 %d）", text, code)
	}
	switch code {
	case 40029:
		return "企业微信授权码无效或已使用，请重新扫码"
	case 60020:
		return "企业微信应用 IP 不在可信域名内，请检查应用配置"
	default:
		return fmt.Sprintf("企业微信接口返回错误（错误码 %d）", code)
	}
}

func wecomCredentialKey(corpID, secret string) string {
	digest := sha256.Sum256([]byte(corpID + "\x00" + secret))
	return hex.EncodeToString(digest[:])
}

func wecomConfigString(stored map[string]any, key string) string {
	value, _ := stored[key].(string)
	return strings.TrimSpace(value)
}

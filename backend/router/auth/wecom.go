package auth

import (
	"context"
	"errors"
	"net/http"
	"time"

	"itdb-backend/internal/service"
	"itdb-backend/router/common"
	"itdb-backend/router/settings"
)

// handleWecomAuthorize 签发企业微信扫码登录跳转地址（公开接口，登录页使用）
func (a *Router) handleWecomAuthorize(w http.ResponseWriter, r *http.Request) {
	provider, err := a.wecomEnabledProvider(r.Context())
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	target, err := a.wecomAuthorizeURL(r, provider, service.WecomLoginPurpose, 0)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]string{"url": target})
}

// handleWecomBindURL 签发企业微信绑定扫码跳转地址（需登录，state 绑定当前用户）
func (a *Router) handleWecomBindURL(w http.ResponseWriter, r *http.Request) {
	operator, err := common.CurrentUser(r.Context())
	if err != nil {
		common.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	provider, err := a.wecomEnabledProvider(r.Context())
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	target, err := a.wecomAuthorizeURL(r, provider, service.WecomBindPurpose, operator.ID)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]string{"url": target})
}

// handleWecomCallback 接收企业微信扫码回调：校验 state 后按绑定关系登录并签发会话
func (a *Router) handleWecomCallback(w http.ResponseWriter, r *http.Request) {
	var req wecomCallbackRequest
	if !common.DecodeJSON(w, r, &req) {
		return
	}
	operator, _ := common.CurrentUser(r.Context())
	if _, err := service.VerifyWecomState(a.cfg.JWTSecret, req.State, service.WecomLoginPurpose); err != nil {
		common.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	provider, err := a.wecomEnabledProvider(r.Context())
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	userid, err := a.exchangeWecomCode(r.Context(), provider, req.Code)
	if err != nil {
		a.recordAuditEvent(r.Context(), operator.Username, common.ClientIP(r), service.AuditModuleAuth, "用户登录", loginAuditTarget(operator.Username), "企业微信 登录失败："+err.Error(), service.AuditResultFailure)
		common.WriteError(w, http.StatusUnauthorized, err.Error())
		return
	}
	response, err := a.authWorkflow.LoginByWecom(r.Context(), userid)
	if err != nil {
		a.recordAuditEvent(r.Context(), "-", common.ClientIP(r), service.AuditModuleAuth, "用户登录", "-", "企业微信 登录失败："+wecomLoginFailureMessage(err), service.AuditResultFailure)
		if errors.Is(err, service.ErrUserNotProvisioned) {
			common.WriteError(w, http.StatusUnauthorized, "用户未在平台中启用")
			return
		}
		common.WriteError(w, http.StatusUnauthorized, err.Error())
		return
	}
	userResponse, err := a.authUserResponse(r.Context(), response.User)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := settings.RecordSettingsUserLogin(r.Context(), a.db, response.User.ID); err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	a.recordAuditEvent(r.Context(), response.User.Username, common.ClientIP(r), service.AuditModuleAuth, "用户登录", response.User.Username, authLoginDetail(response.User.Source, response.User.Username, "登录"), service.AuditResultSuccess)
	common.WriteJSON(w, http.StatusOK, map[string]any{"token": response.Token, "user": userResponse})
}

// handleWecomBind 完成当前登录用户与企业微信账号的绑定
func (a *Router) handleWecomBind(w http.ResponseWriter, r *http.Request) {
	var req wecomCallbackRequest
	if !common.DecodeJSON(w, r, &req) {
		return
	}
	operator, err := common.CurrentUser(r.Context())
	if err != nil {
		common.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	state, err := service.VerifyWecomState(a.cfg.JWTSecret, req.State, service.WecomBindPurpose)
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if state.UserID != operator.ID {
		common.WriteError(w, http.StatusForbidden, "绑定状态与当前用户不符，请重新发起绑定")
		return
	}
	provider, err := a.wecomEnabledProvider(r.Context())
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	userid, err := a.exchangeWecomCode(r.Context(), provider, req.Code)
	if err != nil {
		common.WriteError(w, http.StatusUnauthorized, err.Error())
		return
	}
	var boundUser int64
	bindErr := a.db.QueryRowContext(r.Context(), "SELECT user_id FROM settings_user_wecom WHERE wecom_userid=?", userid).Scan(&boundUser)
	if bindErr == nil && boundUser != operator.ID {
		common.WriteError(w, http.StatusConflict, "该企业微信账号已绑定其他用户，请先在对方账号解绑")
		return
	}
	if _, err := a.db.ExecContext(r.Context(),
		"INSERT INTO settings_user_wecom(user_id,wecom_userid,bound_at) VALUES(?,?,strftime('%s','now')) ON CONFLICT(user_id) DO UPDATE SET wecom_userid=excluded.wecom_userid, bound_at=excluded.bound_at",
		operator.ID, userid,
	); err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	a.recordAuditEvent(r.Context(), operator.Username, common.ClientIP(r), service.AuditModuleAuth, "绑定企业微信", operator.Username, "绑定企业微信账号 "+userid, service.AuditResultSuccess)
	common.WriteJSON(w, http.StatusOK, map[string]any{"ok": true, "wecomUserid": userid})
}

// handleWecomUnbind 解除当前登录用户的企业微信绑定
func (a *Router) handleWecomUnbind(w http.ResponseWriter, r *http.Request) {
	operator, err := common.CurrentUser(r.Context())
	if err != nil {
		common.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	if _, err := a.db.ExecContext(r.Context(), "DELETE FROM settings_user_wecom WHERE user_id=?", operator.ID); err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	a.recordAuditEvent(r.Context(), operator.Username, common.ClientIP(r), service.AuditModuleAuth, "解绑企业微信", operator.Username, "已解除企业微信账号绑定", service.AuditResultSuccess)
	common.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// wecomCallbackRequest 扫码回调与绑定共用的请求体
type wecomCallbackRequest struct {
	// 企业微信回调携带的授权码
	Code string `json:"code"`
	// 发起跳转时签发的防伪状态串
	State string `json:"state"`
}

// wecomEnabledProvider 读取启用且凭据完整的企业微信配置，未就绪时返回错误提示
func (a *Router) wecomEnabledProvider(ctx context.Context) (*service.WecomProvider, error) {
	provider, err := service.LoadWecomProvider(ctx, a.db)
	if err != nil {
		return nil, errors.New("读取企业微信认证配置失败")
	}
	if provider == nil || !provider.Enabled || !provider.HasWecomCredential() {
		return nil, errors.New("企业微信认证未启用或配置不完整")
	}
	return provider, nil
}

// wecomAuthorizeURL 签发指定用途的 state 并拼装扫码跳转地址
func (a *Router) wecomAuthorizeURL(r *http.Request, provider *service.WecomProvider, purpose string, userID int64) (string, error) {
	state, err := service.SignWecomState(a.cfg.JWTSecret, service.WecomState{Purpose: purpose, UserID: userID, Exp: time.Now().Add(service.WecomStateTTL).Unix()})
	if err != nil {
		return "", err
	}
	return provider.WecomAuthorizeURL(service.WecomRequestBaseURL(r), state), nil
}

// exchangeWecomCode 授权码换取企业微信成员 userid，测试可替换实现
func (a *Router) exchangeWecomCode(ctx context.Context, provider *service.WecomProvider, code string) (string, error) {
	if a.wecomExchange != nil {
		return a.wecomExchange(ctx, provider, code)
	}
	return provider.WecomExchangeCode(ctx, code)
}

// wecomLoginFailureMessage 将企微登录失败归类为可读提示
func wecomLoginFailureMessage(err error) string {
	switch {
	case errors.Is(err, service.ErrWecomNotBound):
		return service.ErrWecomNotBound.Error()
	case errors.Is(err, service.ErrUserNotProvisioned):
		return "用户未在平台中启用"
	default:
		return err.Error()
	}
}

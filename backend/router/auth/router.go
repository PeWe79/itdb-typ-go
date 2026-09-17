package auth

import (
	"database/sql"

	"github.com/go-chi/chi/v5"

	"itdb-backend/config"
	"itdb-backend/internal/service"
)

type Router struct {
	db           *sql.DB
	domains      *service.DomainServices
	audit        *service.AuditService
	authWorkflow *service.AuthWorkflow
	cfg          config.Config
}

func New(db *sql.DB, domains *service.DomainServices, audit *service.AuditService, authWorkflow *service.AuthWorkflow, cfg config.Config) *Router {
	return &Router{db: db, domains: domains, audit: audit, authWorkflow: authWorkflow, cfg: cfg}
}

// RegisterPublic 挂载无需认证的认证入口与找回密码接口
func (a *Router) RegisterPublic(r chi.Router) {
	r.Post("/api/auth/login", a.handleLogin)
	r.Get("/api/auth/providers", a.handlePublicAuthProviders)
	r.Get("/api/auth/password-reset/captcha", a.handlePasswordResetCaptcha)
	r.Post("/api/auth/password-reset/verify", a.handlePasswordResetVerify)
	r.Post("/api/auth/password-reset/send", a.handlePasswordResetSend)
	r.Post("/api/auth/password-reset/confirm", a.handlePasswordResetConfirm)
}

// Register 挂载需认证的会话接口，修改密码对所有登录用户开放
func (a *Router) Register(r chi.Router) {
	r.Get("/api/auth/me", a.handleMe)
	r.Post("/api/auth/logout", a.handleLogout)
	r.Post("/api/auth/change-password", a.handleChangePassword)
}

package settings

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"

	"itdb-backend/config"
	"itdb-backend/internal/repository"
	"itdb-backend/internal/service"
)

type Router struct {
	db           *sql.DB
	sql          *service.SQLService
	queries      *repository.SQL
	audit        *service.AuditService
	userWorkflow *service.UserWorkflow
	cfg          config.Config
}

func New(db *sql.DB, sqlSvc *service.SQLService, queries *repository.SQL, audit *service.AuditService, userWorkflow *service.UserWorkflow, cfg config.Config) *Router {
	return &Router{db: db, sql: sqlSvc, queries: queries, audit: audit, userWorkflow: userWorkflow, cfg: cfg}
}

// RegisterPublic 挂载无需认证的公开接口
func (a *Router) RegisterPublic(r chi.Router) {
	r.Get("/api/public/base", a.handlePublicBase)
}

func (a *Router) Register(r chi.Router, requirePermission func(string) func(http.Handler) http.Handler) {

	r.With(requirePermission("settings.base.read")).Get("/api/settings/base", a.handleSettingsBase)
	r.With(requirePermission("settings.base.manage")).Put("/api/settings/base", a.handleSettingsBase)
	r.With(requirePermission("settings.users.read")).Get("/api/settings/permissions", a.handleSettingsPermissions)
	r.With(requirePermission("settings.users.read")).Get("/api/settings/roles", a.handleSettingsRoles)
	r.With(requirePermission("settings.users.manage")).Post("/api/settings/roles", a.handleSettingsRoles)
	r.With(requirePermission("settings.users.manage")).Put("/api/settings/roles/{id}", a.handleSettingsRoles)
	r.With(requirePermission("settings.users.manage")).Post("/api/settings/roles/{id}/disabled", a.handleSettingsRoleDisabled)
	r.With(requirePermission("settings.users.manage")).Delete("/api/settings/roles/{id}", a.handleSettingsRoles)
	r.With(requirePermission("settings.users.read")).Get("/api/settings/user-groups", a.handleSettingsGroups)
	r.With(requirePermission("settings.users.manage")).Post("/api/settings/user-groups", a.handleSettingsGroups)
	r.With(requirePermission("settings.users.manage")).Put("/api/settings/user-groups/{id}", a.handleSettingsGroups)
	r.With(requirePermission("settings.users.manage")).Delete("/api/settings/user-groups/{id}", a.handleSettingsGroups)
	r.With(requirePermission("settings.users.read")).Get("/api/settings/users", a.handleSettingsUsers)
	r.With(requirePermission("settings.users.manage")).Post("/api/settings/users", a.handleSettingsUsers)
	r.With(requirePermission("settings.users.manage")).Put("/api/settings/users/{id}", a.handleSettingsUsers)
	r.With(requirePermission("settings.users.manage")).Delete("/api/settings/users/{id}", a.handleSettingsUsers)
	r.With(requirePermission("settings.users.manage")).Post("/api/settings/users/{id}/disabled", a.handleSettingsUserDisabled)
	r.With(requirePermission("settings.auth.read")).Get("/api/settings/auth-provider", a.handleSettingsAuthProvider)
	r.With(requirePermission("settings.auth.manage")).Put("/api/settings/auth-provider", a.handleSettingsAuthProvider)
	r.With(requirePermission("settings.auth.manage")).Post("/api/settings/auth-provider/test", a.handleSettingsAuthProvider)
	r.With(requirePermission("settings.notifications.read")).Get("/api/settings/email", a.handleSettingsEmail)
	r.With(requirePermission("settings.notifications.manage")).Put("/api/settings/email", a.handleSettingsEmail)
	r.With(requirePermission("settings.notifications.manage")).Post("/api/settings/email/test", a.handleSettingsEmail)
}

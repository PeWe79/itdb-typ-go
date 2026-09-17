package router

import (
	"itdb-backend/router/common"
	"net/http"
	"time"

	_ "itdb-backend/docs"
	"itdb-backend/router/assets"
	"itdb-backend/router/auth"
	"itdb-backend/router/settings"
	"itdb-backend/router/system"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	_ "modernc.org/sqlite"
)

func (a *App) routes() http.Handler {
	r := chi.NewRouter()
	r.Use(a.corsMiddleware)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		common.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		common.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Get("/swagger/*", httpSwagger.Handler(httpSwagger.URL("/swagger/doc.json")))

	a.settingsR = settings.New(a.db, a.sql, a.queries, a.audit, a.userWorkflow, a.cfg)
	a.authR = auth.New(a.db, a.domains, a.audit, a.authWorkflow, a.cfg)
	a.systemR = system.New(a.db, a.sql, a.queries, a.domains, a.audit, a.backupWorkflow, a.cfg)
	a.systemR.SetOnDatabaseReplaced(a.onDatabaseReplaced)
	a.assetsR = assets.New(a.assetsDeps())

	a.authR.RegisterPublic(r)
	a.settingsR.RegisterPublic(r)
	anyReadPermission := a.requireAnyPermission(settings.ReadPermissionKeys()...)
	r.Group(func(r chi.Router) {
		r.Use(a.authMiddleware)
		a.authR.Register(r)

		r.With(anyReadPermission).Get("/api/bootstrap", a.handleBootstrap)
		r.With(anyReadPermission).Get("/api/dashboard/summary", a.handleDashboardSummary)
		a.systemR.Register(r, a.requirePermission)

		a.settingsR.Register(r, a.requirePermission)

		a.assetsR.Register(r, a.requirePermission, a.currentUserPermissions)
	})

	return r
}

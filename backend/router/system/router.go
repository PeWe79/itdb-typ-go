package system

import (
	"database/sql"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"itdb-backend/config"
	"itdb-backend/internal/repository"
	"itdb-backend/internal/service"
)

type Router struct {
	db                 *sql.DB
	sql                *service.SQLService
	queries            *repository.SQL
	domains            *service.DomainServices
	audit              *service.AuditService
	backupWorkflow     *service.BackupService
	dbMu               sync.Mutex
	cfg                config.Config
	onDatabaseReplaced func(*sql.DB)
}

func New(db *sql.DB, sqlSvc *service.SQLService, queries *repository.SQL, domains *service.DomainServices, audit *service.AuditService, backupWorkflow *service.BackupService, cfg config.Config) *Router {
	return &Router{db: db, sql: sqlSvc, queries: queries, domains: domains, audit: audit, backupWorkflow: backupWorkflow, cfg: cfg}
}

// Register 挂载系统域路由：审计日志按查看权限，数据库备份与导入按基础配置管理权限
func (a *Router) Register(r chi.Router, requirePermission func(string) func(http.Handler) http.Handler) {
	r.With(requirePermission("audit.read")).Get("/api/history", a.handleHistory)
	r.With(requirePermission("settings.base.manage")).Get("/api/backups/database", a.handleDownloadDatabaseBackup)
	r.With(requirePermission("settings.base.manage"), middleware.Timeout(300*time.Second)).Post("/api/import/database", a.handleImportDatabase)
}

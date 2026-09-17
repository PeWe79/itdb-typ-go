package system

import (
	"database/sql"

	"itdb-backend/config"
	"itdb-backend/internal/repository"
	"itdb-backend/internal/service"
)

// SetOnDatabaseReplaced 注册数据库导入完成后的回调，由根路由负责整体刷新服务依赖
func (a *Router) SetOnDatabaseReplaced(fn func(newDB *sql.DB)) {
	a.onDatabaseReplaced = fn
}

// Reset 原地更新服务依赖（数据库导入后由根路由调用），保留互斥锁与回调状态
func (a *Router) Reset(db *sql.DB, sqlSvc *service.SQLService, queries *repository.SQL, domains *service.DomainServices, audit *service.AuditService, backupWorkflow *service.BackupService, cfg config.Config) {
	a.db = db
	a.sql = sqlSvc
	a.queries = queries
	a.domains = domains
	a.audit = audit
	a.backupWorkflow = backupWorkflow
	a.cfg = cfg
}

func (a *Router) notifyDatabaseReplaced(newDB *sql.DB) {
	if a.onDatabaseReplaced != nil {
		a.onDatabaseReplaced(newDB)
	}
}

package system

import (
	"context"

	"itdb-backend/internal/service"
	"itdb-backend/pkg/database"
)

func (a *Router) auditService() *service.AuditService {
	if a.audit == nil {
		a.audit = service.NewAuditService(a.db, a.cfg.HistoryLimit)
	}
	return a.audit
}

// recordAuditEvent 写入一条与 SQL 无关的显式审计事件（登录、备份、导入、连接测试等）
func (a *Router) recordAuditEvent(ctx context.Context, username, ip, module, action, target, detail, result string) {
	event := service.AuditEvent{Module: module, Action: action, Target: target, Detail: detail, Result: result}
	_ = a.auditService().RecordEvent(ctx, username, ip, event)
}

// AuditBackupFailureDetail 备份/导入失败的审计详情
func AuditBackupFailureDetail(err error) string {
	if err == nil {
		return "操作失败"
	}
	return "操作失败：" + err.Error()
}

func (a *Router) fetchRows(query string, args ...interface{}) ([]map[string]interface{}, error) {
	if a.sql != nil {
		return a.queries.QueryRows(context.Background(), query, args...)
	}
	rows, err := a.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return database.RowsToMaps(rows)
}

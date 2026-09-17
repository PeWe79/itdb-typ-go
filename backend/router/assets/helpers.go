package assets

import (
	"context"
	"errors"

	"itdb-backend/internal/service"
	"itdb-backend/pkg/database"
)

// newServiceConflict 提取业务冲突错误中的中文提示；非冲突错误返回空串
func newServiceConflict(err error) string {
	var conflict *service.ConflictError
	if errors.As(err, &conflict) {
		return conflict.Message
	}
	return ""
}

func (a *Router) auditService() *service.AuditService {
	if a.audit == nil {
		a.audit = service.NewAuditService(a.db, a.cfg.HistoryLimit)
	}
	return a.audit
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

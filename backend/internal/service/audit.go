package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"itdb-backend/internal/repository"
)

// AuditService 负责执行业务写入并记录结构化审计日志：除保留原始 SQL 外，
// 同时归类出模块、操作、目标与结果，供审计日志页面展示与筛选。
type AuditService struct {
	db           repository.Executor
	historyLimit int64
}

func NewAuditService(db repository.Executor, historyLimit int64) *AuditService {
	return &AuditService{db: db, historyLimit: historyLimit}
}

// auditExecWriter 兼容 *sql.DB 与 *sql.Tx 的最小写入接口
type auditExecWriter interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

// RecordEvent 记录一条与 SQL 无关的显式审计事件（如登录、备份、导入、连接测试）
func (s *AuditService) RecordEvent(ctx context.Context, username, ip string, event AuditEvent) error {
	return s.recordEvent(ctx, s.db, username, ip, event)
}

// RecordEventTx 与 RecordEvent 相同，但在调用方事务内写入，保证业务数据与审计同生共死
func (s *AuditService) RecordEventTx(ctx context.Context, tx *sql.Tx, username, ip string, event AuditEvent) error {
	return s.recordEvent(ctx, tx, username, ip, event)
}

func (s *AuditService) recordEvent(ctx context.Context, writer auditExecWriter, username, ip string, event AuditEvent) error {
	if strings.TrimSpace(event.Result) == "" {
		event.Result = AuditResultSuccess
	}
	if strings.TrimSpace(event.Target) == "" {
		event.Target = "-"
	}
	return s.insertHistory(ctx, writer, username, ip, "", event)
}

func (s *AuditService) RecordTx(ctx context.Context, tx *sql.Tx, username, ip, query string, args ...interface{}) error {
	return s.recordWithTarget(ctx, tx, username, ip, "", query, args)
}

func (s *AuditService) Record(ctx context.Context, username, ip, query string, args ...interface{}) error {
	return s.recordWithTarget(ctx, s.db, username, ip, "", query, args)
}

// RecordTxWithTarget 与 RecordTx 相同，但允许调用方显式指定目标（如自增 ID）
func (s *AuditService) RecordTxWithTarget(ctx context.Context, tx *sql.Tx, username, ip, target, query string, args ...interface{}) error {
	return s.recordWithTarget(ctx, tx, username, ip, target, query, args)
}

// RecordWithTarget 与 Record 相同，但允许调用方显式指定目标（如自增 ID）
func (s *AuditService) RecordWithTarget(ctx context.Context, username, ip, target, query string, args ...interface{}) error {
	return s.recordWithTarget(ctx, s.db, username, ip, target, query, args)
}

func (s *AuditService) recordWithTarget(ctx context.Context, writer auditExecWriter, username, ip, target, query string, args []interface{}) error {
	event, ok := ClassifyAuditSQL(username, query, args)
	if !ok {
		return nil
	}
	if strings.TrimSpace(target) != "" {
		event.Target = target
	}
	event.Result = AuditResultSuccess
	event.Detail = auditDetailText(query, args)
	return s.insertHistory(ctx, writer, username, ip, query, event)
}

func (s *AuditService) Execute(ctx context.Context, username, ip, query string, args ...interface{}) (sql.Result, error) {
	res, err := s.db.ExecContext(ctx, query, args...)
	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "SELECT") {
		return res, nil
	}
	event, ok := ClassifyAuditSQL(username, query, args)
	if !ok {
		return res, err
	}
	if err != nil {
		event.Result = AuditResultFailure
		event.Detail = auditDetailText(query, args) + " | error: " + err.Error()
		_ = s.insertHistory(ctx, s.db, username, ip, query, event)
		return nil, err
	}
	if strings.TrimSpace(event.Target) == "" || event.Target == "-" {
		if id, idErr := res.LastInsertId(); idErr == nil && id > 0 {
			event.Target = fmt.Sprintf("#%d", id)
		}
	}
	event.Result = AuditResultSuccess
	event.Detail = auditDetailText(query, args)
	_ = s.insertHistory(ctx, s.db, username, ip, query, event)
	return res, nil
}

func (s *AuditService) insertHistory(ctx context.Context, writer auditExecWriter, username, ip, query string, event AuditEvent) error {
	if strings.TrimSpace(username) == "" {
		username = "unknown"
	}
	if strings.TrimSpace(event.Result) == "" {
		event.Result = AuditResultSuccess
	}
	_, err := writer.ExecContext(ctx,
		`INSERT INTO history (date, sql, authuser, ip, module, action, target, detail, result) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		time.Now().Unix(), query, username, ip, event.Module, event.Action, event.Target, event.Detail, event.Result,
	)
	if err != nil {
		return err
	}
	return s.applyHistoryLimit(ctx, writer)
}

func (s *AuditService) applyHistoryLimit(ctx context.Context, writer auditExecWriter) error {
	if s.historyLimit > 0 {
		_, err := writer.ExecContext(ctx, `DELETE FROM history WHERE id < (SELECT COALESCE(MAX(id), 0) - ? FROM history)`, s.historyLimit)
		return err
	}
	return nil
}

func auditDetailText(query string, args []interface{}) string {
	if len(args) > 0 {
		return fmt.Sprintf("%s | args=%v", query, args)
	}
	return query
}

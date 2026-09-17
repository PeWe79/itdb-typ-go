package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"itdb-backend/internal/repository"
)

type TagWorkflow struct {
	repo  repository.Repository
	audit *AuditService
}

func NewTagWorkflow(repo repository.Repository, audit *AuditService) *TagWorkflow {
	return &TagWorkflow{repo: repo, audit: audit}
}

// NextTagID 返回下一个可用的标记编号（tags 为 AUTOINCREMENT，优先读 sqlite_sequence），
// 供标记关联页签的预新增行显示未来真实编号
func (s *TagWorkflow) NextTagID(ctx context.Context) (int64, error) {
	var next sql.NullInt64
	err := s.repo.QueryRowContext(ctx, `
		SELECT COALESCE(
			(SELECT seq FROM sqlite_sequence WHERE name = 'tags'),
			(SELECT MAX(id) FROM tags)
		) + 1`).Scan(&next)
	if err == nil {
		return next.Int64, nil
	}
	var maxID sql.NullInt64
	if err := s.repo.QueryRowContext(ctx, `SELECT MAX(id) FROM tags`).Scan(&maxID); err != nil {
		return 0, err
	}
	return maxID.Int64 + 1, nil
}

// EnsureTagLogged 确保标签存在（不存在时创建并记录审计），供标签关联变更复用
func (s *TagWorkflow) EnsureTagLogged(ctx context.Context, actor UserActor, name string) (int64, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, errors.New("name is required")
	}
	var id sql.NullInt64
	err := s.repo.QueryRowContext(ctx, `SELECT id FROM tags WHERE LOWER(name) = LOWER(?) LIMIT 1`, name).Scan(&id)
	if err == nil && id.Valid {
		return id.Int64, nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	res, err := s.repo.ExecContext(ctx, `INSERT INTO tags (name) VALUES (?)`, name)
	if err != nil {
		return 0, err
	}
	newID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	event := AuditEvent{
		Module: AuditModuleCatalog,
		Action: "新增标记",
		Target: strings.TrimSpace(name),
		Detail: FormatSimpleName(name, newID, true) + " 已创建",
		Result: AuditResultSuccess,
	}
	return newID, s.audit.RecordEvent(ctx, actor.Username, actor.IP, event)
}

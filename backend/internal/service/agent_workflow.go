package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"itdb-backend/internal/domain"
	"itdb-backend/internal/repository"
)

type AgentWorkflow struct {
	repo  repository.AgentRepository
	audit *AuditService
}

func NewAgentWorkflow(repo repository.AgentRepository, audit *AuditService) *AgentWorkflow {
	return &AgentWorkflow{repo: repo, audit: audit}
}
func (s *AgentWorkflow) List(ctx context.Context, search string) (*sql.Rows, error) {
	return s.repo.ListAgents(ctx, strings.TrimSpace(search))
}
func (s *AgentWorkflow) Get(ctx context.Context, id int64) (*sql.Rows, error) {
	return s.repo.GetAgent(ctx, id)
}

func (s *AgentWorkflow) Create(ctx context.Context, actor UserActor, req domain.AgentPayload) (int64, error) {
	values, err := normalizeAgent(req)
	if err != nil {
		return 0, err
	}
	var existing int64
	if err := s.repo.QueryRowContext(ctx, `SELECT id FROM agents WHERE LOWER(TRIM(COALESCE(title,''))) = LOWER(TRIM(?)) LIMIT 1`, strings.TrimSpace(req.Title)).Scan(&existing); err == nil {
		return 0, NewConflictError("代理名称 %s 已存在", strings.TrimSpace(req.Title))
	} else if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	res, err := s.repo.CreateAgent(ctx, values[0].(int64), values[1].(string), values[2].(string), values[3].(string), values[4].(string))
	if err != nil {
		s.recordAgentAuditFailure(ctx, actor, AuditOutcomeCreate, 0, req, err)
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	event := AuditEvent{
		Module: AuditModuleAssets,
		Action: AssetOutcomeAction(AuditOutcomeCreate, "代理"),
		Target: formatAssetTarget("代理", id),
		Detail: AuditEntityDetail(FormatSimpleName(req.Title, id, true), AuditOutcomeCreate, true, "", ""),
		Result: AuditResultSuccess,
	}
	return id, s.audit.RecordEvent(ctx, actor.Username, actor.IP, event)
}

func (s *AgentWorkflow) Update(ctx context.Context, actor UserActor, id int64, req domain.AgentPayload) error {
	values, err := normalizeAgent(req)
	if err != nil {
		return err
	}
	if err := enforceAgentTypeRemovalRules(
		func(query string, args ...interface{}) *sql.Row {
			return s.repo.QueryRowContext(ctx, query, args...)
		}, id, agentTypeMask(req),
	); err != nil {
		s.recordAgentAuditFailure(ctx, actor, AuditOutcomeUpdate, id, req, err)
		return err
	}
	var oldTitle string
	if err := s.repo.QueryRowContext(ctx, `SELECT COALESCE(title,'') FROM agents WHERE id = ?`, id).Scan(&oldTitle); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	} else if err == nil && strings.TrimSpace(req.Title) != strings.TrimSpace(oldTitle) {
		var existing int64
		if err := s.repo.QueryRowContext(ctx, `SELECT id FROM agents WHERE LOWER(TRIM(COALESCE(title,''))) = LOWER(TRIM(?)) AND id <> ? LIMIT 1`, strings.TrimSpace(req.Title), id).Scan(&existing); err == nil {
			err := NewConflictError("代理名称 %s 已存在", strings.TrimSpace(req.Title))
			s.recordAgentAuditFailure(ctx, actor, AuditOutcomeUpdate, id, req, err)
			return err
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	}
	if _, err := s.repo.UpdateAgent(ctx, id, values[0].(int64), values[1].(string), values[2].(string), values[3].(string), values[4].(string)); err != nil {
		s.recordAgentAuditFailure(ctx, actor, AuditOutcomeUpdate, id, req, err)
		return err
	}
	if strings.TrimSpace(req.ChangeNote) == "" {
		return nil
	}
	event := AuditEvent{
		Module: AuditModuleAssets,
		Action: AssetOutcomeAction(AuditOutcomeUpdate, "代理"),
		Target: formatAssetTarget("代理", id),
		Detail: AuditEntityDetail(FormatSimpleName(req.Title, id, true), AuditOutcomeUpdate, true, req.ChangeNote, ""),
		Result: AuditResultSuccess,
	}
	return s.audit.RecordEvent(ctx, actor.Username, actor.IP, event)
}

func (s *AgentWorkflow) Delete(ctx context.Context, actor UserActor, id int64) error {
	tx, err := s.repo.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := enforceAgentDeleteRules(tx, id); err != nil {
		return err
	}
	if err := s.recordAgentAudit(ctx, tx, actor, AuditOutcomeDelete, id, true, ""); err != nil {
		return err
	}
	if err := s.repo.DeleteAgentReferences(ctx, tx, id); err != nil {
		return err
	}
	return tx.Commit()
}

// recordAgentAudit 在事务内写入代理显式审计事件（删除按库中名称）
func (s *AgentWorkflow) recordAgentAudit(ctx context.Context, tx *sql.Tx, actor UserActor, outcome AuditOutcome, id int64, success bool, changes string) error {
	var title string
	_ = tx.QueryRowContext(ctx, `SELECT COALESCE(title, '') FROM agents WHERE id = ?`, id).Scan(&title)
	event := AuditEvent{
		Module: AuditModuleAssets,
		Action: AssetOutcomeAction(outcome, "代理"),
		Target: formatAssetTarget("代理", id),
		Detail: AuditEntityDetail(FormatSimpleName(title, id, true), outcome, success, changes, ""),
		Result: AuditResultSuccess,
	}
	return s.audit.RecordEventTx(ctx, tx, actor.Username, actor.IP, event)
}

// recordAgentAuditFailure 失败路径在事务外补记失败审计事件
func (s *AgentWorkflow) recordAgentAuditFailure(ctx context.Context, actor UserActor, outcome AuditOutcome, id int64, req domain.AgentPayload, auditErr error) {
	title := req.Title
	if title == "" {
		_ = s.repo.QueryRowContext(ctx, `SELECT COALESCE(title, '') FROM agents WHERE id = ?`, id).Scan(&title)
	}
	event := AuditEvent{
		Module: AuditModuleAssets,
		Action: AssetOutcomeAction(outcome, "代理"),
		Target: formatAssetTarget("代理", id),
		Detail: AuditEntityDetail(FormatSimpleName(title, id, id > 0), outcome, false, "", ConflictReason(auditErr)),
		Result: AuditResultFailure,
	}
	_ = s.audit.RecordEvent(ctx, actor.Username, actor.IP, event)
}

// agentTypeUsageChecks 代理各类型（硬件厂商/软件厂商/供应商/采购方/承包方）的引用占用检查表
var agentTypeUsageChecks = []struct {
	mask  int64
	label string
	query string
	noun  string
}{
	{8, "硬件厂商", `SELECT COUNT(id) FROM items WHERE manufacturerid = ?`, "硬件"},
	{2, "软件厂商", `SELECT COUNT(id) FROM software WHERE manufacturerid = ?`, "软件"},
	{4, "供应商", `SELECT COUNT(id) FROM invoices WHERE vendorid = ?`, "单据"},
	{4, "供应商", `SELECT COUNT(id) FROM items WHERE TRIM(origin) != '' AND TRIM(origin) = (SELECT TRIM(title) FROM agents WHERE id = ?)`, "硬件"},
	{1, "采购方", `SELECT COUNT(id) FROM invoices WHERE buyerid = ?`, "单据"},
	{16, "承包方", `SELECT COUNT(id) FROM contracts WHERE contractorid = ?`, "合同"},
}

// collectAgentTypeUsage 检查 typesMask 中每个类型的引用占用，
// 返回全部被引用类型的完整消息（suffix 区分“无法删除/无法保存”场景），供聚合提示
func collectAgentTypeUsage(queryRow func(query string, args ...interface{}) *sql.Row, typesMask, agentID int64, suffix string) ([]string, error) {
	var messages []string
	for _, check := range agentTypeUsageChecks {
		if typesMask&check.mask != check.mask {
			continue
		}
		var count int64
		if err := queryRow(check.query, agentID).Scan(&count); err != nil {
			return nil, err
		}
		if count > 0 {
			messages = append(messages, fmt.Sprintf("该代理的%s类型已被 %d 条%s记录使用%s", check.label, count, check.noun, suffix))
		}
	}
	return messages, nil
}

// enforceAgentDeleteRules 删除前检查代理各类型的引用占用，
// 全部被引用的类型聚合为一条多行冲突错误（前端拆分为多条提示）
func enforceAgentDeleteRules(tx *sql.Tx, id int64) error {
	var types int64
	if err := tx.QueryRow(`SELECT COALESCE(type, 0) FROM agents WHERE id = ?`, id).Scan(&types); err != nil {
		return err
	}
	messages, err := collectAgentTypeUsage(tx.QueryRow, types, id, "，无法删除")
	if err != nil {
		return err
	}
	if len(messages) > 0 {
		return NewConflictError("%s", strings.Join(messages, "\n"))
	}
	return nil
}

// enforceAgentTypeRemovalRules 编辑保存前检查被取消勾选的类型是否仍被引用，
// 任一被取消的类型被引用即整次保存拦截，全部被引用的类型聚合为一条多行冲突错误
func enforceAgentTypeRemovalRules(queryRow func(query string, args ...interface{}) *sql.Row, agentID, newMask int64) error {
	var oldMask int64
	if err := queryRow(`SELECT COALESCE(type, 0) FROM agents WHERE id = ?`, agentID).Scan(&oldMask); err != nil {
		return err
	}
	removed := oldMask & ^newMask
	if removed == 0 {
		return nil
	}
	messages, err := collectAgentTypeUsage(queryRow, removed, agentID, "，无法保存")
	if err != nil {
		return err
	}
	if len(messages) > 0 {
		return NewConflictError("%s", strings.Join(messages, "\n"))
	}
	return nil
}

func normalizeAgent(req domain.AgentPayload) ([]interface{}, error) {
	if err := domain.ValidateAgentTitle(req.Title); err != nil {
		return nil, err
	}
	return []interface{}{agentTypeMask(req), req.Title, req.ContactInfo, encodeAgentContacts(req.Contacts), encodeAgentURLs(req.URLs)}, nil
}
func agentTypeMask(req domain.AgentPayload) int64 {
	if req.Type != nil {
		return *req.Type
	}
	var mask int64
	for _, value := range req.Types {
		mask += value
	}
	return mask
}
func sanitizeAgentValue(value string) string {
	return strings.TrimSpace(strings.NewReplacer("|", " ", "#", " ").Replace(value))
}
func encodeAgentContacts(items []domain.AgentContact) string {
	rows := make([]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, strings.Join([]string{sanitizeAgentValue(item.Name), sanitizeAgentValue(item.Phones), sanitizeAgentValue(item.Email), sanitizeAgentValue(item.Role), sanitizeAgentValue(item.Comments)}, "#"))
	}
	return strings.Join(rows, "|")
}
func encodeAgentURLs(items []domain.AgentURL) string {
	rows := make([]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, strings.Join([]string{sanitizeAgentValue(item.Description), sanitizeAgentValue(item.URL)}, "#"))
	}
	return strings.Join(rows, "|")
}

package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"itdb-backend/internal/common/primitives"
	"itdb-backend/internal/domain"
	"itdb-backend/internal/repository"
	"strings"
)

type ContractWorkflow struct {
	repo      repository.ContractRepository
	relations *RelationService
	audit     *AuditService
}

func NewContractWorkflow(repo repository.ContractRepository, rel *RelationService, audit *AuditService) *ContractWorkflow {
	return &ContractWorkflow{repo: repo, relations: rel, audit: audit}
}
func (s *ContractWorkflow) List(ctx context.Context, search string) (*sql.Rows, error) {
	return s.repo.ListContracts(ctx, strings.TrimSpace(search))
}
func (s *ContractWorkflow) Get(ctx context.Context, id int64) (*sql.Rows, error) {
	return s.repo.GetContract(ctx, id)
}
func contractArgs(req domain.ContractPayload) ([]interface{}, error) {
	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Number) == "" || req.TypeID == 0 || req.ContractorID == 0 || strings.TrimSpace(req.StartDate) == "" || strings.TrimSpace(req.CurrentEnd) == "" {
		return nil, fmt.Errorf("missing mandatory fields")
	}
	sd, e := primitives.ParseDateInput(req.StartDate)
	if e != nil {
		return nil, fmt.Errorf("invalid startDate")
	}
	ed, e := primitives.ParseDateInput(req.CurrentEnd)
	if e != nil {
		return nil, fmt.Errorf("invalid currentEndDate")
	}
	return []interface{}{req.TypeID, req.SubTypeID, primitives.NullableInt(req.ParentID), req.Title, req.Number, req.Description, req.Comments, req.TotalCost, req.ContractorID, sd, ed, req.Renewals}, nil
}
func (s *ContractWorkflow) Create(ctx context.Context, actor UserActor, req domain.ContractPayload) (int64, error) {
	a, e := contractArgs(req)
	if e != nil {
		return 0, e
	}
	var existing int64
	if err := s.repo.QueryRowContext(ctx, `SELECT id FROM contracts WHERE LOWER(TRIM(COALESCE(title,''))) = LOWER(TRIM(?)) LIMIT 1`, strings.TrimSpace(req.Title)).Scan(&existing); err == nil {
		return 0, NewConflictError("合同标题 %s 已存在", strings.TrimSpace(req.Title))
	} else if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	id, e := s.createTx(ctx, actor, req, a)
	if e != nil {
		s.recordContractAuditFailure(ctx, actor, AuditOutcomeCreate, 0, req, e)
		return 0, e
	}
	return id, nil
}

func (s *ContractWorkflow) createTx(ctx context.Context, actor UserActor, req domain.ContractPayload, a []interface{}) (int64, error) {
	tx, e := s.repo.BeginTx(ctx, nil)
	if e != nil {
		return 0, e
	}
	defer tx.Rollback()
	res, e := s.repo.InsertContract(ctx, tx, a...)
	if e != nil {
		return 0, e
	}
	id, e := res.LastInsertId()
	if e != nil {
		return 0, e
	}
	if e = s.replace(ctx, tx, id, req, nil); e != nil {
		return 0, e
	}
	if e = s.recordContractAudit(ctx, tx, actor, AuditOutcomeCreate, id, req, true, ""); e != nil {
		return 0, e
	}
	if e = tx.Commit(); e != nil {
		return 0, e
	}
	return id, nil
}

func (s *ContractWorkflow) Update(ctx context.Context, actor UserActor, id int64, req domain.ContractPayload) error {
	a, e := contractArgs(req)
	if e != nil {
		return e
	}
	var oldTitle string
	if err := s.repo.QueryRowContext(ctx, `SELECT COALESCE(title,'') FROM contracts WHERE id = ?`, id).Scan(&oldTitle); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	} else if err == nil && strings.TrimSpace(req.Title) != strings.TrimSpace(oldTitle) {
		var existing int64
		if err := s.repo.QueryRowContext(ctx, `SELECT id FROM contracts WHERE LOWER(TRIM(COALESCE(title,''))) = LOWER(TRIM(?)) AND id <> ? LIMIT 1`, strings.TrimSpace(req.Title), id).Scan(&existing); err == nil {
			err := NewConflictError("合同标题 %s 已存在", strings.TrimSpace(req.Title))
			s.recordContractAuditFailure(ctx, actor, AuditOutcomeUpdate, id, req, err)
			return err
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	}
	if e = s.updateTx(ctx, actor, id, req, a); e != nil {
		s.recordContractAuditFailure(ctx, actor, AuditOutcomeUpdate, id, req, e)
		return e
	}
	return nil
}

func (s *ContractWorkflow) updateTx(ctx context.Context, actor UserActor, id int64, req domain.ContractPayload, a []interface{}) error {
	tx, e := s.repo.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	old, e := s.repo.ContractFileIDs(ctx, tx, id)
	if e != nil {
		return e
	}
	if _, e = s.repo.UpdateContract(ctx, tx, id, a...); e != nil {
		return e
	}
	if e = s.replace(ctx, tx, id, req, old); e != nil {
		return e
	}
	if strings.TrimSpace(req.ChangeNote) != "" {
		if e = s.recordContractAudit(ctx, tx, actor, AuditOutcomeUpdate, id, req, true, req.ChangeNote); e != nil {
			return e
		}
	}
	return tx.Commit()
}

func (s *ContractWorkflow) Delete(ctx context.Context, actor UserActor, id int64) error {
	tx, e := s.repo.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if e = enforceContractDeleteRules(tx, id); e != nil {
		return e
	}
	if e = s.recordContractAudit(ctx, tx, actor, AuditOutcomeDelete, id, domain.ContractPayload{}, true, ""); e != nil {
		return e
	}
	if e = s.repo.DeleteContractReferences(ctx, tx, id); e != nil {
		return e
	}
	return tx.Commit()
}

// recordContractAudit 在事务内写入合同显式审计事件
func (s *ContractWorkflow) recordContractAudit(ctx context.Context, tx *sql.Tx, actor UserActor, outcome AuditOutcome, id int64, req domain.ContractPayload, success bool, changes string) error {
	title := req.Title
	if title == "" {
		_ = tx.QueryRowContext(ctx, `SELECT COALESCE(title, '') FROM contracts WHERE id = ?`, id).Scan(&title)
	}
	event := AuditEvent{
		Module: AuditModuleAssets,
		Action: AssetOutcomeAction(outcome, "合同"),
		Target: formatAssetTarget("合同", id),
		Detail: AuditEntityDetail(FormatSimpleName(title, id, id > 0), outcome, success, changes, ""),
		Result: AuditResultSuccess,
	}
	return s.audit.RecordEventTx(ctx, tx, actor.Username, actor.IP, event)
}

// recordContractAuditFailure 失败路径在事务外补记失败审计事件
func (s *ContractWorkflow) recordContractAuditFailure(ctx context.Context, actor UserActor, outcome AuditOutcome, id int64, req domain.ContractPayload, auditErr error) {
	title := req.Title
	if title == "" {
		_ = s.repo.QueryRowContext(ctx, `SELECT COALESCE(title, '') FROM contracts WHERE id = ?`, id).Scan(&title)
	}
	event := AuditEvent{
		Module: AuditModuleAssets,
		Action: AssetOutcomeAction(outcome, "合同"),
		Target: formatAssetTarget("合同", id),
		Detail: AuditEntityDetail(FormatSimpleName(title, id, id > 0), outcome, false, "", ConflictReason(auditErr)),
		Result: AuditResultFailure,
	}
	_ = s.audit.RecordEvent(ctx, actor.Username, actor.IP, event)
}

// enforceContractDeleteRules 检查合同被其他合同作为上级合同引用的占用，被引用即返回冲突错误
func enforceContractDeleteRules(tx *sql.Tx, id int64) error {
	var count int64
	if err := tx.QueryRow(`SELECT COUNT(id) FROM contracts WHERE parentid = ?`, id).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return NewConflictError("该合同已被 %d 条合同记录使用，无法删除", count)
	}
	return nil
}
func (s *ContractWorkflow) replace(ctx context.Context, tx *sql.Tx, id int64, req domain.ContractPayload, old []int64) error {
	for _, v := range [][3]interface{}{{`DELETE FROM contract2item WHERE contractid=?`, `INSERT INTO contract2item (contractid,itemid) VALUES (?,?)`, req.ItemLinks}, {`DELETE FROM contract2soft WHERE contractid=?`, `INSERT INTO contract2soft (contractid,softid) VALUES (?,?)`, req.SoftwareLinks}, {`DELETE FROM contract2inv WHERE contractid=?`, `INSERT INTO contract2inv (contractid,invid) VALUES (?,?)`, req.InvoiceLinks}, {`DELETE FROM contract2file WHERE contractid=?`, `INSERT INTO contract2file (contractid,fileid) VALUES (?,?)`, req.FileLinks}} {
		if e := s.relations.ReplaceIDLinks(tx, v[0].(string), v[1].(string), id, v[2].([]int64)); e != nil {
			return e
		}
	}
	if len(old) > 0 {
		return s.relations.CleanupRemovedFileLinks(tx, old, req.FileLinks, req.CleanupFileLinks)
	}
	return nil
}

type ContractEventWorkflow struct {
	repo  repository.ContractRepository
	audit *AuditService
}

func NewContractEventWorkflow(repo repository.ContractRepository, audit *AuditService) *ContractEventWorkflow {
	return &ContractEventWorkflow{repo: repo, audit: audit}
}
func (s *ContractEventWorkflow) List(ctx context.Context, id int64) (*sql.Rows, error) {
	return s.repo.ListContractEvents(ctx, id)
}

// NextEventID returns the next available contractevent id so pre-added event rows
// can display their future real id. The table is AUTOINCREMENT, so the
// sqlite_sequence counter takes precedence over MAX(id).
func (s *ContractEventWorkflow) NextEventID(ctx context.Context) (int64, error) {
	var next sql.NullInt64
	e := s.repo.QueryRowContext(ctx, `
		SELECT COALESCE(
			(SELECT seq FROM sqlite_sequence WHERE name = 'contractevents'),
			(SELECT MAX(id) FROM contractevents)
		) + 1`).Scan(&next)
	if e == nil {
		return next.Int64, nil
	}
	var maxID sql.NullInt64
	if e := s.repo.QueryRowContext(ctx, `SELECT MAX(id) FROM contractevents`).Scan(&maxID); e != nil {
		return 0, e
	}
	return maxID.Int64 + 1, nil
}
func parseContractEvent(req domain.ContractEventPayload) (int64, int64, error) {
	if strings.TrimSpace(req.Description) == "" {
		return 0, 0, fmt.Errorf("description is required")
	}
	sd, e := primitives.ParseDateInput(req.StartDate)
	if e != nil {
		return 0, 0, fmt.Errorf("invalid startDate")
	}
	ed, e := primitives.ParseDateInput(req.EndDate)
	if e != nil {
		return 0, 0, fmt.Errorf("invalid endDate")
	}
	return sd, ed, nil
}
func (s *ContractEventWorkflow) Create(ctx context.Context, actor UserActor, contractID int64, req domain.ContractEventPayload) (int64, error) {
	sd, ed, e := parseContractEvent(req)
	if e != nil {
		return 0, e
	}
	r, e := s.repo.CreateContractEvent(ctx, contractID, req.SiblingID, sd, ed, req.Description)
	if e != nil {
		return 0, e
	}
	return r.LastInsertId()
}
func (s *ContractEventWorkflow) Update(ctx context.Context, actor UserActor, eventID int64, req domain.ContractEventPayload) error {
	sd, ed, e := parseContractEvent(req)
	if e != nil {
		return e
	}
	r, e := s.repo.UpdateContractEvent(ctx, eventID, req.SiblingID, sd, ed, req.Description)
	if e != nil {
		return e
	}
	n, e := r.RowsAffected()
	if e != nil {
		return e
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
func (s *ContractEventWorkflow) Delete(ctx context.Context, actor UserActor, eventID int64) error {
	r, e := s.repo.DeleteContractEvent(ctx, eventID)
	if e != nil {
		return e
	}
	n, e := r.RowsAffected()
	if e != nil {
		return e
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// eventContractID 查询事件历史所属的合同编号
func (s *ContractEventWorkflow) eventContractID(ctx context.Context, eventID int64) (int64, error) {
	var contractID int64
	err := s.repo.QueryRowContext(ctx, `SELECT COALESCE(contractid, 0) FROM contractevents WHERE id = ?`, eventID).Scan(&contractID)
	return contractID, err
}

// recordContractEventAudit 合同事件历史变更以所属合同为审计目标
func (s *ContractEventWorkflow) recordContractEventAudit(ctx context.Context, actor UserActor, outcome AuditOutcome, contractID int64, description string) error {
	var title string
	_ = s.repo.QueryRowContext(ctx, `SELECT COALESCE(title, '') FROM contracts WHERE id = ?`, contractID).Scan(&title)
	verb := "新增事件历史"
	if outcome == AuditOutcomeUpdate {
		verb = "更新事件历史"
	}
	if outcome == AuditOutcomeDelete {
		verb = "删除事件历史"
	}
	if description = strings.TrimSpace(description); description != "" && outcome != AuditOutcomeDelete {
		verb += "：" + description
	}
	event := AuditEvent{
		Module: AuditModuleAssets,
		Action: AssetOutcomeAction(outcome, "合同事件"),
		Target: formatAssetTarget("合同", contractID),
		Detail: fmt.Sprintf("%s %s", FormatSimpleName(title, contractID, true), verb),
		Result: AuditResultSuccess,
	}
	return s.audit.RecordEvent(ctx, actor.Username, actor.IP, event)
}

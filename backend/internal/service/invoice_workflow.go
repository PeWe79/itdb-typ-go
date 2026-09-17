package service

import (
	"context"
	"database/sql"
	"fmt"
	"itdb-backend/internal/common/primitives"
	"itdb-backend/internal/domain"
	"itdb-backend/internal/repository"
	"strings"
)

type InvoiceWorkflow struct {
	repo      repository.InvoiceRepository
	relations *RelationService
	audit     *AuditService
}

func NewInvoiceWorkflow(repo repository.InvoiceRepository, rel *RelationService, audit *AuditService) *InvoiceWorkflow {
	return &InvoiceWorkflow{repo: repo, relations: rel, audit: audit}
}
func (s *InvoiceWorkflow) List(ctx context.Context, search string, limit, offset int64) (*sql.Rows, error) {
	return s.repo.ListInvoices(ctx, strings.TrimSpace(search), limit, offset)
}
func (s *InvoiceWorkflow) Get(ctx context.Context, id int64) (*sql.Rows, error) {
	return s.repo.GetInvoice(ctx, id)
}
func (s *InvoiceWorkflow) Create(ctx context.Context, actor UserActor, req domain.InvoicePayload) (int64, error) {
	args, e := invoiceArgs(req)
	if e != nil {
		return 0, e
	}
	id, e := s.createTx(ctx, actor, req, args)
	if e != nil {
		s.recordInvoiceAuditFailure(ctx, actor, AuditOutcomeCreate, 0, req, e)
		return 0, e
	}
	return id, nil
}

func (s *InvoiceWorkflow) createTx(ctx context.Context, actor UserActor, req domain.InvoicePayload, args []interface{}) (int64, error) {
	tx, e := s.repo.BeginTx(ctx, nil)
	if e != nil {
		return 0, e
	}
	defer tx.Rollback()
	res, e := s.repo.InsertInvoice(ctx, tx, args...)
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
	if e = s.recordInvoiceAudit(ctx, tx, actor, AuditOutcomeCreate, id, req, true, ""); e != nil {
		return 0, e
	}
	if e = tx.Commit(); e != nil {
		return 0, e
	}
	return id, nil
}

func (s *InvoiceWorkflow) Update(ctx context.Context, actor UserActor, id int64, req domain.InvoicePayload) error {
	args, e := invoiceArgs(req)
	if e != nil {
		return e
	}
	if e = s.updateTx(ctx, actor, id, req, args); e != nil {
		s.recordInvoiceAuditFailure(ctx, actor, AuditOutcomeUpdate, id, req, e)
		return e
	}
	return nil
}

func (s *InvoiceWorkflow) updateTx(ctx context.Context, actor UserActor, id int64, req domain.InvoicePayload, args []interface{}) error {
	tx, e := s.repo.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	old, e := s.repo.InvoiceFileIDs(ctx, tx, id)
	if e != nil {
		return e
	}
	if _, e = s.repo.UpdateInvoice(ctx, tx, id, args...); e != nil {
		return e
	}
	if e = s.replace(ctx, tx, id, req, old); e != nil {
		return e
	}
	if strings.TrimSpace(req.ChangeNote) != "" {
		if e = s.recordInvoiceAudit(ctx, tx, actor, AuditOutcomeUpdate, id, req, true, req.ChangeNote); e != nil {
			return e
		}
	}
	return tx.Commit()
}

func (s *InvoiceWorkflow) Delete(ctx context.Context, actor UserActor, id int64) error {
	tx, e := s.repo.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if e = s.recordInvoiceAudit(ctx, tx, actor, AuditOutcomeDelete, id, domain.InvoicePayload{}, true, ""); e != nil {
		return e
	}
	if e = s.repo.DeleteInvoiceReferences(ctx, tx, id); e != nil {
		return e
	}
	return tx.Commit()
}

// invoiceName 组装单据显示名：删除按库中数据，新增/更新按请求参数
func (s *InvoiceWorkflow) invoiceName(ctx context.Context, exec repository.Executor, req domain.InvoicePayload, id int64, withID bool) string {
	if req.Number == "" {
		if name, err := LoadSimpleName(ctx, exec, "invoices", "number", id); err == nil {
			return name
		}
		return FormatSimpleName("", id, withID)
	}
	return FormatSimpleName(req.Number, id, withID)
}

// recordInvoiceAudit 在事务内写入单据显式审计事件
func (s *InvoiceWorkflow) recordInvoiceAudit(ctx context.Context, tx *sql.Tx, actor UserActor, outcome AuditOutcome, id int64, req domain.InvoicePayload, success bool, changes string) error {
	event := AuditEvent{
		Module: AuditModuleAssets,
		Action: AssetOutcomeAction(outcome, "单据"),
		Target: formatAssetTarget("单据", id),
		Detail: AuditEntityDetail(s.invoiceName(ctx, tx, req, id, id > 0), outcome, success, changes, ""),
		Result: AuditResultSuccess,
	}
	return s.audit.RecordEventTx(ctx, tx, actor.Username, actor.IP, event)
}

// recordInvoiceAuditFailure 失败路径在事务外补记失败审计事件
func (s *InvoiceWorkflow) recordInvoiceAuditFailure(ctx context.Context, actor UserActor, outcome AuditOutcome, id int64, req domain.InvoicePayload, auditErr error) {
	event := AuditEvent{
		Module: AuditModuleAssets,
		Action: AssetOutcomeAction(outcome, "单据"),
		Target: formatAssetTarget("单据", id),
		Detail: AuditEntityDetail(s.invoiceName(ctx, s.repo, req, id, id > 0), outcome, false, "", ConflictReason(auditErr)),
		Result: AuditResultFailure,
	}
	_ = s.audit.RecordEvent(ctx, actor.Username, actor.IP, event)
}
func (s *InvoiceWorkflow) replace(ctx context.Context, tx *sql.Tx, id int64, req domain.InvoicePayload, old []int64) error {
	for _, v := range [][3]interface{}{{`DELETE FROM item2inv WHERE invid=?`, `INSERT INTO item2inv (invid,itemid) VALUES (?,?)`, req.ItemLinks}, {`DELETE FROM soft2inv WHERE invid=?`, `INSERT INTO soft2inv (invid,softid) VALUES (?,?)`, req.SoftwareLinks}, {`DELETE FROM contract2inv WHERE invid=?`, `INSERT INTO contract2inv (invid,contractid) VALUES (?,?)`, req.ContractLinks}, {`DELETE FROM invoice2file WHERE invoiceid=?`, `INSERT INTO invoice2file (invoiceid,fileid) VALUES (?,?)`, req.FileLinks}} {
		if e := s.relations.ReplaceIDLinks(tx, v[0].(string), v[1].(string), id, v[2].([]int64)); e != nil {
			return e
		}
	}
	if len(old) > 0 {
		return s.relations.CleanupRemovedFileLinks(tx, old, req.FileLinks, req.CleanupFileLinks)
	}
	return nil
}
func invoiceArgs(req domain.InvoicePayload) ([]interface{}, error) {
	if req.VendorID == 0 || req.BuyerID == 0 || strings.TrimSpace(req.Number) == "" || strings.TrimSpace(req.Date) == "" {
		return nil, fmt.Errorf("vendorId, buyerId, number and date are required")
	}
	d, e := primitives.ParseDateInput(req.Date)
	if e != nil {
		return nil, fmt.Errorf("invalid date")
	}
	return []interface{}{req.VendorID, req.BuyerID, req.Number, req.Description, d}, nil
}

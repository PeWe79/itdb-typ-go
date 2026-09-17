package service

import (
	"context"
	"database/sql"
	"fmt"
	"itdb-backend/internal/domain"
	"itdb-backend/internal/repository"
	"strings"
)

type RackWorkflow struct {
	repo  repository.InfrastructureRepository
	audit *AuditService
}

func NewRackWorkflow(repo repository.InfrastructureRepository, audit *AuditService) *RackWorkflow {
	return &RackWorkflow{repo: repo, audit: audit}
}
func (s *RackWorkflow) List(ctx context.Context, search string) (*sql.Rows, error) {
	return s.repo.ListRacks(ctx, strings.TrimSpace(search))
}
func (s *RackWorkflow) Get(ctx context.Context, id int64) (*sql.Rows, error) {
	return s.repo.GetRack(ctx, id)
}
func rackArgs(req domain.RackPayload) ([]interface{}, error) {
	if req.USize == 0 || req.Depth == 0 {
		return nil, fmt.Errorf("uSize and depth are required")
	}
	var area interface{}
	if req.LocAreaID != nil && *req.LocAreaID != 0 {
		area = *req.LocAreaID
	}
	return []interface{}{req.LocationID, req.USize, req.Depth, req.Comments, req.Model, req.Label, req.RevNums, area}, nil
}
func (s *RackWorkflow) Create(ctx context.Context, actor UserActor, req domain.RackPayload) (int64, error) {
	a, e := rackArgs(req)
	if e != nil {
		return 0, e
	}
	r, e := s.repo.InsertRack(ctx, a...)
	if e != nil {
		s.recordRackAuditFailure(ctx, actor, AuditOutcomeCreate, 0, req, e)
		return 0, e
	}
	id, e := r.LastInsertId()
	if e != nil {
		return 0, e
	}
	name := s.rackName(ctx, s.repo, req, id, true)
	event := AuditEvent{
		Module: AuditModuleAssets,
		Action: AssetOutcomeAction(AuditOutcomeCreate, "机架"),
		Target: formatAssetTarget("机架", id),
		Detail: AuditEntityDetail(name, AuditOutcomeCreate, true, "", ""),
		Result: AuditResultSuccess,
	}
	return id, s.audit.RecordEvent(ctx, actor.Username, actor.IP, event)
}

func (s *RackWorkflow) Update(ctx context.Context, actor UserActor, id int64, req domain.RackPayload) error {
	a, e := rackArgs(req)
	if e != nil {
		return e
	}
	tx, e := s.repo.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if _, e = s.repo.UpdateRack(ctx, tx, id, a...); e != nil {
		return e
	}
	if e = s.repo.UpdateRackItems(ctx, tx, id, req.LocationID, a[7]); e != nil {
		return e
	}
	if strings.TrimSpace(req.ChangeNote) != "" {
		name := s.rackName(ctx, tx, req, id, true)
		event := AuditEvent{
			Module: AuditModuleAssets,
			Action: AssetOutcomeAction(AuditOutcomeUpdate, "机架"),
			Target: formatAssetTarget("机架", id),
			Detail: AuditEntityDetail(name, AuditOutcomeUpdate, true, req.ChangeNote, ""),
			Result: AuditResultSuccess,
		}
		if e = s.audit.RecordEventTx(ctx, tx, actor.Username, actor.IP, event); e != nil {
			return e
		}
	}
	return tx.Commit()
}

func (s *RackWorkflow) Delete(ctx context.Context, actor UserActor, id int64) error {
	n, e := s.repo.RackItemCount(ctx, id)
	if e != nil {
		return e
	}
	if n > 0 {
		return NewConflictError("该机架已被 %d 条硬件记录使用，无法删除", n)
	}
	name, e := LoadRackName(ctx, s.repo, id)
	if e != nil {
		return e
	}
	tx, e := s.repo.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if e = s.repo.DeleteRack(ctx, tx, id); e != nil {
		return e
	}
	event := AuditEvent{
		Module: AuditModuleAssets,
		Action: AssetOutcomeAction(AuditOutcomeDelete, "机架"),
		Target: formatAssetTarget("机架", id),
		Detail: AuditEntityDetail(name, AuditOutcomeDelete, true, "", ""),
		Result: AuditResultSuccess,
	}
	if e = s.audit.RecordEventTx(ctx, tx, actor.Username, actor.IP, event); e != nil {
		return e
	}
	return tx.Commit()
}

// rackName 组装机架显示名：删除按库中数据，新增/更新按请求参数 + 地点/区域名称查询；
// “<地点>”对应机架编辑里的地点选择项内容
func (s *RackWorkflow) rackName(ctx context.Context, exec repository.Executor, req domain.RackPayload, id int64, withID bool) string {
	if req.Label == "" && req.LocationID == 0 && (req.LocAreaID == nil || *req.LocAreaID == 0) {
		if name, err := LoadRackName(ctx, s.repo, id); err == nil {
			return name
		}
		return FormatRackName("", "", "", id, withID)
	}
	var locationName, locationFloor, area string
	_ = exec.QueryRowContext(ctx, `SELECT COALESCE(name, ''), COALESCE(floor, '') FROM locations WHERE id = ?`, req.LocationID).Scan(&locationName, &locationFloor)
	location := FormatRackLocationText(locationName, locationFloor)
	if req.LocAreaID != nil && *req.LocAreaID != 0 {
		_ = exec.QueryRowContext(ctx, `SELECT COALESCE(areaname, '') FROM locareas WHERE id = ?`, *req.LocAreaID).Scan(&area)
	}
	return FormatRackName(req.Label, location, area, id, withID)
}

// recordRackAuditFailure 失败路径在事务外补记失败审计事件
func (s *RackWorkflow) recordRackAuditFailure(ctx context.Context, actor UserActor, outcome AuditOutcome, id int64, req domain.RackPayload, auditErr error) {
	name := req.Label
	if name == "" {
		if loaded, err := LoadRackName(ctx, s.repo, id); err == nil {
			name = loaded
		}
	}
	event := AuditEvent{
		Module: AuditModuleAssets,
		Action: AssetOutcomeAction(outcome, "机架"),
		Target: formatAssetTarget("机架", id),
		Detail: AuditEntityDetail(FormatSimpleName(name, id, id > 0), outcome, false, "", ConflictReason(auditErr)),
		Result: AuditResultFailure,
	}
	_ = s.audit.RecordEvent(ctx, actor.Username, actor.IP, event)
}

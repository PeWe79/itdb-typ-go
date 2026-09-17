package service

import (
	"context"
	"database/sql"
	"fmt"
	"itdb-backend/internal/domain"
	"itdb-backend/internal/repository"
	"strings"
)

type LocationWorkflow struct {
	repo  repository.InfrastructureRepository
	audit *AuditService
}

func NewLocationWorkflow(repo repository.InfrastructureRepository, audit *AuditService) *LocationWorkflow {
	return &LocationWorkflow{repo: repo, audit: audit}
}
func (s *LocationWorkflow) List(ctx context.Context, search string) (*sql.Rows, error) {
	return s.repo.ListLocations(ctx, strings.TrimSpace(search))
}
func (s *LocationWorkflow) Get(ctx context.Context, id int64) (*sql.Rows, error) {
	return s.repo.GetLocation(ctx, id)
}
func (s *LocationWorkflow) Areas(ctx context.Context, id int64) (*sql.Rows, error) {
	return s.repo.ListAreas(ctx, id)
}

// NextAreaID returns the next available locarea id so pre-added area rows can
// display their future real id without relying on cached lookups. The table is
// AUTOINCREMENT, so the sqlite_sequence counter (which never decreases after
// deletes) takes precedence over MAX(id).
func (s *LocationWorkflow) NextAreaID(ctx context.Context) (int64, error) {
	var next sql.NullInt64
	e := s.repo.QueryRowContext(ctx, `
		SELECT COALESCE(
			(SELECT seq FROM sqlite_sequence WHERE name = 'locareas'),
			(SELECT MAX(id) FROM locareas)
		) + 1`).Scan(&next)
	if e == nil {
		return next.Int64, nil
	}
	var maxID sql.NullInt64
	if e := s.repo.QueryRowContext(ctx, `SELECT MAX(id) FROM locareas`).Scan(&maxID); e != nil {
		return 0, e
	}
	return maxID.Int64 + 1, nil
}
func validateLocation(req domain.LocationPayload) error {
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Floor) == "" {
		return fmt.Errorf("name and floor are required")
	}
	return nil
}
func (s *LocationWorkflow) CreateArea(ctx context.Context, actor UserActor, locationID int64, req domain.LocAreaPayload) (int64, error) {
	if strings.TrimSpace(req.AreaName) == "" {
		return 0, fmt.Errorf("areaName is required")
	}
	r, e := s.repo.ExecContext(ctx, `INSERT INTO locareas (locationid,areaname) VALUES (?,?)`, locationID, strings.TrimSpace(req.AreaName))
	if e != nil {
		return 0, e
	}
	return r.LastInsertId()
}
func (s *LocationWorkflow) UpdateArea(ctx context.Context, actor UserActor, areaID, locationID int64, req domain.LocAreaPayload) error {
	if strings.TrimSpace(req.AreaName) == "" {
		return fmt.Errorf("areaName is required")
	}
	r, e := s.repo.ExecContext(ctx, `UPDATE locareas SET locationid=?,areaname=? WHERE id=?`, locationID, strings.TrimSpace(req.AreaName), areaID)
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
func (s *LocationWorkflow) DeleteArea(ctx context.Context, actor UserActor, id int64) error {
	var messages []string
	racks, e := s.repo.AreaRackCount(ctx, id)
	if e != nil {
		return e
	}
	if racks > 0 {
		messages = append(messages, fmt.Sprintf("该区域已被 %d 条机架记录使用，无法删除", racks))
	}
	items, e := s.repo.AreaItemCount(ctx, id)
	if e != nil {
		return e
	}
	if items > 0 {
		messages = append(messages, fmt.Sprintf("该区域已被 %d 条硬件记录使用，无法删除", items))
	}
	if len(messages) > 0 {
		return NewConflictError("%s", strings.Join(messages, "\n"))
	}
	r, e := s.repo.ExecContext(ctx, `DELETE FROM locareas WHERE id=?`, id)
	if e != nil {
		return e
	}
	affected, e := r.RowsAffected()
	if e != nil {
		return e
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *LocationWorkflow) CreateWithFloorplan(ctx context.Context, actor UserActor, req domain.LocationPayload, floorplan string) (int64, error) {
	if e := validateLocation(req); e != nil {
		return 0, e
	}
	name, floor := strings.TrimSpace(req.Name), strings.TrimSpace(req.Floor)
	r, e := s.repo.CreateLocation(ctx, name, floor, floorplan)
	if e != nil {
		s.recordLocationAuditFailure(ctx, actor, AuditOutcomeCreate, 0, req, e)
		return 0, e
	}
	id, e := r.LastInsertId()
	if e != nil {
		return 0, e
	}
	areas := req.Areas
	if areas == nil {
		areas = []string{}
	}
	event := AuditEvent{
		Module: AuditModuleAssets,
		Action: AssetOutcomeAction(AuditOutcomeCreate, "地点"),
		Target: formatAssetTarget("地点", id),
		Detail: AuditEntityDetail(FormatLocationName(req.Name, areas, req.Floor, id, true), AuditOutcomeCreate, true, "", ""),
		Result: AuditResultSuccess,
	}
	return id, s.audit.RecordEvent(ctx, actor.Username, actor.IP, event)
}
func (s *LocationWorkflow) UpdateWithFloorplan(ctx context.Context, actor UserActor, id int64, req domain.LocationPayload, floorplan *string) (string, error) {
	if e := validateLocation(req); e != nil {
		return "", e
	}
	old, _, e := s.repo.LocationFloorplan(ctx, id)
	if e != nil {
		return "", e
	}
	tx, e := s.repo.BeginTx(ctx, nil)
	if e != nil {
		return "", e
	}
	defer tx.Rollback()
	if _, e = s.repo.UpdateLocation(ctx, tx, id, strings.TrimSpace(req.Name), strings.TrimSpace(req.Floor), floorplan); e != nil {
		return "", e
	}
	if strings.TrimSpace(req.ChangeNote) != "" {
		areas := req.Areas
		if areas == nil {
			areas, e = loadLocationAreas(ctx, tx, id)
			if e != nil {
				return "", e
			}
		}
		event := AuditEvent{
			Module: AuditModuleAssets,
			Action: AssetOutcomeAction(AuditOutcomeUpdate, "地点"),
			Target: formatAssetTarget("地点", id),
			Detail: AuditEntityDetail(FormatLocationName(req.Name, areas, req.Floor, id, true), AuditOutcomeUpdate, true, req.ChangeNote, ""),
			Result: AuditResultSuccess,
		}
		if e = s.audit.RecordEventTx(ctx, tx, actor.Username, actor.IP, event); e != nil {
			return "", e
		}
	}
	if e = tx.Commit(); e != nil {
		return "", e
	}
	return old, nil
}
func (s *LocationWorkflow) Delete(ctx context.Context, actor UserActor, id int64) (string, error) {
	var messages []string
	racks, e := s.repo.LocationRackCount(ctx, id)
	if e != nil {
		return "", e
	}
	if racks > 0 {
		messages = append(messages, fmt.Sprintf("该地点已被 %d 条机架记录使用，无法删除", racks))
	}
	hardware, e := s.repo.LocationItemCount(ctx, id)
	if e != nil {
		return "", e
	}
	if hardware > 0 {
		messages = append(messages, fmt.Sprintf("该地点已被 %d 条硬件记录使用，无法删除", hardware))
	}
	if len(messages) > 0 {
		return "", NewConflictError("%s", strings.Join(messages, "\n"))
	}
	old, _, e := s.repo.LocationFloorplan(ctx, id)
	if e != nil {
		return "", e
	}
	name, e := LoadLocationName(ctx, s.repo, id)
	if e != nil {
		return "", e
	}
	tx, e := s.repo.BeginTx(ctx, nil)
	if e != nil {
		return "", e
	}
	defer tx.Rollback()
	if e = s.repo.DeleteLocation(ctx, tx, id); e != nil {
		return "", e
	}
	event := AuditEvent{
		Module: AuditModuleAssets,
		Action: AssetOutcomeAction(AuditOutcomeDelete, "地点"),
		Target: formatAssetTarget("地点", id),
		Detail: AuditEntityDetail(name, AuditOutcomeDelete, true, "", ""),
		Result: AuditResultSuccess,
	}
	if e = s.audit.RecordEventTx(ctx, tx, actor.Username, actor.IP, event); e != nil {
		return "", e
	}
	if e = tx.Commit(); e != nil {
		return "", e
	}
	return old, nil
}

// recordLocationAuditFailure 失败路径在事务外补记失败审计事件
func (s *LocationWorkflow) recordLocationAuditFailure(ctx context.Context, actor UserActor, outcome AuditOutcome, id int64, req domain.LocationPayload, auditErr error) {
	name := req.Name
	if name == "" {
		if loaded, err := LoadLocationName(ctx, s.repo, id); err == nil {
			name = loaded
		}
	}
	event := AuditEvent{
		Module: AuditModuleAssets,
		Action: AssetOutcomeAction(outcome, "地点"),
		Target: formatAssetTarget("地点", id),
		Detail: AuditEntityDetail(FormatSimpleName(name, id, id > 0), outcome, false, "", ConflictReason(auditErr)),
		Result: AuditResultFailure,
	}
	_ = s.audit.RecordEvent(ctx, actor.Username, actor.IP, event)
}

func (s *LocationWorkflow) Floorplan(ctx context.Context, id int64) (string, string, error) {
	return s.repo.LocationFloorplan(ctx, id)
}

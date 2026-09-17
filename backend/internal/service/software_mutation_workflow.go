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

type SoftwareMutationWorkflow struct {
	repo      repository.SoftwareRepository
	relations *RelationService
	audit     *AuditService
}

func NewSoftwareMutationWorkflow(repo repository.SoftwareRepository, relations *RelationService, audit *AuditService) *SoftwareMutationWorkflow {
	return &SoftwareMutationWorkflow{repo: repo, relations: relations, audit: audit}
}

// enforceSoftwareLicenseCount 软件保存时校验授权：
// 按CPU类型时关联的硬件必须已配置 CPU 数量，按核心类型时还须配置每CPU核心数（缺失即报错）；
// 配置齐全后按口径统计总消耗，超过授权数量同样拦截
func enforceSoftwareLicenseCount(ctx context.Context, tx *sql.Tx, licensed *int64, licenseType int64, itemLinks []int64) error {
	seen := make(map[int64]bool)
	ordered := make([]int64, 0, len(itemLinks))
	for _, itemID := range itemLinks {
		if itemID > 0 && !seen[itemID] {
			seen[itemID] = true
			ordered = append(ordered, itemID)
		}
	}
	if licensed == nil {
		if len(ordered) > 0 {
			return NewConflictError("该软件未配置授权数量，请先进行配置")
		}
		return nil
	}
	var messages []string
	total := int64(0)
	for _, itemID := range ordered {
		var cpuNo, coresPerCPU sql.NullString
		if err := tx.QueryRowContext(ctx, `SELECT CAST(cpuno AS TEXT), CAST(corespercpu AS TEXT) FROM items WHERE id = ?`, itemID).Scan(&cpuNo, &coresPerCPU); err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			return err
		}
		cpuValue := licenseUsageNumber(cpuNo.String)
		coresValue := licenseUsageNumber(coresPerCPU.String)
		switch {
		case licenseType == 1 && cpuValue <= 0:
			messages = append(messages, fmt.Sprintf("硬件 %d 未配置 CPU 数量，无法关联按 CPU 授权的软件", itemID))
		case licenseType == 2 && (cpuValue <= 0 || coresValue <= 0):
			messages = append(messages, fmt.Sprintf("硬件 %d 未配置 CPU 数量或每 CPU 核心数，无法关联按核心授权的软件", itemID))
		}
		switch licenseType {
		case 1:
			total += cpuValue
		case 2:
			total += cpuValue * coresValue
		default:
			total++
		}
	}
	if total > *licensed {
		messages = append(messages, "该软件授权已达上限，请先扩容授权")
	}
	if len(messages) > 0 {
		return NewConflictError("%s", strings.Join(messages, "\n"))
	}
	return nil
}

func (s *SoftwareMutationWorkflow) Create(ctx context.Context, actor UserActor, req domain.SoftwarePayload) (int64, error) {
	args, err := softwareArgs(req)
	if err != nil {
		return 0, err
	}
	id, err := s.createTx(ctx, actor, req, args)
	if err != nil {
		if !IsNameConflictError(err) {
			s.recordSoftwareAuditFailure(ctx, actor, AuditOutcomeCreate, 0, req, err)
		}
		return 0, err
	}
	return id, nil
}

func (s *SoftwareMutationWorkflow) createTx(ctx context.Context, actor UserActor, req domain.SoftwarePayload, args []interface{}) (int64, error) {
	tx, err := s.repo.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if err := enforceSoftwareUniqueTitleVersion(ctx, tx, req.Title, req.Version, 0); err != nil {
		return 0, err
	}
	if err := enforceSoftwareLicenseCount(ctx, tx, req.LicenseQty, req.LicenseType, req.ItemLinks); err != nil {
		return 0, err
	}
	res, err := s.repo.InsertSoftware(ctx, tx, args...)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if err := s.replace(ctx, tx, id, req, nil); err != nil {
		return 0, err
	}
	if err := s.recordSoftwareAudit(ctx, tx, actor, AuditOutcomeCreate, id, req, true, ""); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *SoftwareMutationWorkflow) Update(ctx context.Context, actor UserActor, id int64, req domain.SoftwarePayload) error {
	args, err := softwareArgs(req)
	if err != nil {
		return err
	}
	if err := s.updateTx(ctx, actor, id, req, args); err != nil {
		s.recordSoftwareAuditFailure(ctx, actor, AuditOutcomeUpdate, id, req, err)
		return err
	}
	return nil
}

func (s *SoftwareMutationWorkflow) updateTx(ctx context.Context, actor UserActor, id int64, req domain.SoftwarePayload, args []interface{}) error {
	tx, err := s.repo.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := enforceSoftwareUniqueTitleVersion(ctx, tx, req.Title, req.Version, id); err != nil {
		return err
	}
	if req.LicenseQty == nil && s.softwareLicenseInUse(ctx, tx, id) {
		return NewConflictError("该软件授权已被使用中，授权数量不能为空")
	}
	if err := enforceSoftwareLicenseCount(ctx, tx, req.LicenseQty, req.LicenseType, req.ItemLinks); err != nil {
		return err
	}
	old, err := s.repo.SoftwareFileIDs(ctx, tx, id)
	if err != nil {
		return err
	}
	if _, err := s.repo.UpdateSoftware(ctx, tx, id, args...); err != nil {
		return err
	}
	if err := s.replace(ctx, tx, id, req, old); err != nil {
		return err
	}
	if strings.TrimSpace(req.ChangeNote) != "" {
		if err := s.recordSoftwareAudit(ctx, tx, actor, AuditOutcomeUpdate, id, req, true, req.ChangeNote); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *SoftwareMutationWorkflow) Delete(ctx context.Context, actor UserActor, id int64) error {
	tx, err := s.repo.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := s.recordSoftwareAudit(ctx, tx, actor, AuditOutcomeDelete, id, domain.SoftwarePayload{}, true, ""); err != nil {
		return err
	}
	if err := s.recordTagDetachAudit(ctx, tx, actor, id); err != nil {
		return err
	}
	if err := s.repo.DeleteSoftwareReferences(ctx, tx, id); err != nil {
		return err
	}
	return tx.Commit()
}

// softwareName 组装软件显示名：删除按库中数据，新增/更新按请求参数 + 厂商名称查询
func (s *SoftwareMutationWorkflow) softwareName(ctx context.Context, exec repository.Executor, req domain.SoftwarePayload, id int64, withID bool) string {
	if req.Title == "" && req.Version == "" && req.Manufacturer == 0 {
		if name, err := LoadSoftwareName(ctx, exec, id); err == nil {
			return name
		}
		return FormatSoftwareName("", "", "", id, withID)
	}
	var manufacturer string
	_ = exec.QueryRowContext(ctx, `SELECT COALESCE(title, '') FROM agents WHERE id = ?`, req.Manufacturer).Scan(&manufacturer)
	return FormatSoftwareName(manufacturer, req.Title, req.Version, id, withID)
}

// recordSoftwareAudit 在事务内写入软件显式审计事件
func (s *SoftwareMutationWorkflow) recordSoftwareAudit(ctx context.Context, tx *sql.Tx, actor UserActor, outcome AuditOutcome, id int64, req domain.SoftwarePayload, success bool, changes string) error {
	event := AuditEvent{
		Module: AuditModuleAssets,
		Action: AssetOutcomeAction(outcome, "软件"),
		Target: formatAssetTarget("软件", id),
		Detail: AuditEntityDetail(s.softwareName(ctx, tx, req, id, id > 0), outcome, success, changes, ""),
		Result: AuditResultSuccess,
	}
	return s.audit.RecordEventTx(ctx, tx, actor.Username, actor.IP, event)
}

// recordSoftwareAuditFailure 失败路径在事务外补记失败审计事件
func (s *SoftwareMutationWorkflow) recordSoftwareAuditFailure(ctx context.Context, actor UserActor, outcome AuditOutcome, id int64, req domain.SoftwarePayload, auditErr error) {
	event := AuditEvent{
		Module: AuditModuleAssets,
		Action: AssetOutcomeAction(outcome, "软件"),
		Target: formatAssetTarget("软件", id),
		Detail: AuditEntityDetail(s.softwareName(ctx, s.repo, req, id, id > 0), outcome, false, "", ConflictReason(auditErr)),
		Result: AuditResultFailure,
	}
	_ = s.audit.RecordEvent(ctx, actor.Username, actor.IP, event)
}

// recordTagDetachAudit 软件删除时对每个关联标记写“删除标记关联”审计（解除关联软件）
func (s *SoftwareMutationWorkflow) recordTagDetachAudit(ctx context.Context, tx *sql.Tx, actor UserActor, id int64) error {
	rows, err := tx.QueryContext(ctx, `SELECT t.name FROM tag2software ts JOIN tags t ON t.id = ts.tagid WHERE ts.softwareid = ? ORDER BY t.id`, id)
	if err != nil {
		return err
	}
	tagNames := make([]string, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return err
		}
		if trimmed := strings.TrimSpace(name); trimmed != "" {
			tagNames = append(tagNames, trimmed)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	if len(tagNames) == 0 {
		return nil
	}
	softwareName, err := LoadSoftwareName(ctx, tx, id)
	if err != nil {
		return err
	}
	for _, tag := range tagNames {
		event := AuditEvent{
			Module: AuditModuleCatalog,
			Action: "删除标记关联",
			Target: tag,
			Detail: tag + " 已解除关联软件：" + softwareName,
			Result: AuditResultSuccess,
		}
		if err := s.audit.RecordEventTx(ctx, tx, actor.Username, actor.IP, event); err != nil {
			return err
		}
	}
	return nil
}

// softwareLicenseTypeLabel 授权类型的中文标签（CPU 因拉丁字母与“授权”之间补空格）
func softwareLicenseTypeLabel(licenseType int64) string {
	switch licenseType {
	case 1:
		return "CPU "
	case 2:
		return "核心"
	default:
		return "设备"
	}
}

// enforceSoftwareUniqueTitleVersion 同一标题+版本的软件视为重复，新增与编辑（排除自身）统一拦截
func enforceSoftwareUniqueTitleVersion(ctx context.Context, tx *sql.Tx, title, version string, excludeID int64) error {
	trimmedTitle := strings.TrimSpace(title)
	trimmedVersion := strings.TrimSpace(version)
	var count int64
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(id) FROM software WHERE TRIM(stitle) = ? AND TRIM(sversion) = ? AND id <> ?`, trimmedTitle, trimmedVersion, excludeID).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return NewConflictError("软件名称 %s %s 已存在", trimmedTitle, trimmedVersion)
	}
	return nil
}

// softwareLicenseInUse 软件是否已有关联硬件（即授权已投入使用）
func (s *SoftwareMutationWorkflow) softwareLicenseInUse(ctx context.Context, tx *sql.Tx, id int64) bool {
	var linked int64
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(itemid) FROM item2soft WHERE softid = ?`, id).Scan(&linked); err != nil {
		return false
	}
	return linked > 0
}

func (s *SoftwareMutationWorkflow) replace(ctx context.Context, tx *sql.Tx, id int64, req domain.SoftwarePayload, old []int64) error {
	for _, v := range [][3]interface{}{{`DELETE FROM item2soft WHERE softid = ?`, `INSERT INTO item2soft (softid,itemid) VALUES (?, ?)`, req.ItemLinks}, {`DELETE FROM soft2inv WHERE softid = ?`, `INSERT INTO soft2inv (softid,invid) VALUES (?, ?)`, req.InvoiceLinks}, {`DELETE FROM contract2soft WHERE softid = ?`, `INSERT INTO contract2soft (softid,contractid) VALUES (?, ?)`, req.ContractLinks}, {`DELETE FROM software2file WHERE softwareid = ?`, `INSERT INTO software2file (softwareid,fileid) VALUES (?, ?)`, req.FileLinks}} {
		if err := s.relations.ReplaceIDLinks(tx, v[0].(string), v[1].(string), id, v[2].([]int64)); err != nil {
			return err
		}
	}
	if len(old) > 0 {
		return s.relations.CleanupRemovedFileLinks(tx, old, req.FileLinks, req.CleanupFileLinks)
	}
	return nil
}
func softwareArgs(req domain.SoftwarePayload) ([]interface{}, error) {
	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Version) == "" || req.Manufacturer == 0 {
		return nil, fmt.Errorf("title, version and manufacturerId are required")
	}
	if req.LicenseQty != nil && *req.LicenseQty < 0 {
		return nil, fmt.Errorf("licenseQty cannot be negative")
	}
	var date interface{}
	if strings.TrimSpace(req.PurchaseDate) != "" {
		v, err := primitives.ParseDateInput(req.PurchaseDate)
		if err != nil {
			return nil, fmt.Errorf("invalid purchaseDate")
		}
		date = primitives.NullableInt64Value(v)
	}
	var licenseQty interface{}
	if req.LicenseQty != nil {
		licenseQty = *req.LicenseQty
	}
	return []interface{}{primitives.NullableInt(req.InvoiceID), req.SLicenseInfo, req.Manufacturer, req.Title, req.Version, req.Info, date, licenseQty, req.LicenseType}, nil
}

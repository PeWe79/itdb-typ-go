package service

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"itdb-backend/internal/common/primitives"
	"itdb-backend/internal/domain"
	"itdb-backend/internal/repository"
)

type ItemMutationWorkflow struct {
	repo      repository.ItemRepository
	relations *RelationService
	audit     *AuditService
}

func NewItemMutationWorkflow(repo repository.ItemRepository, relations *RelationService, audit *AuditService) *ItemMutationWorkflow {
	return &ItemMutationWorkflow{repo: repo, relations: relations, audit: audit}
}

func (s *ItemMutationWorkflow) Create(ctx context.Context, actor UserActor, req domain.ItemPayload) (int64, error) {
	args, err := itemCreateArgs(req)
	if err != nil {
		return 0, err
	}
	id, err := s.createTx(ctx, actor, req, args)
	if err != nil {
		s.recordItemAuditFailure(ctx, actor, AuditOutcomeCreate, 0, req, err)
		return 0, err
	}
	return id, nil
}

func (s *ItemMutationWorkflow) createTx(ctx context.Context, actor UserActor, req domain.ItemPayload, args []interface{}) (int64, error) {
	tx, err := s.repo.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if err := enforceRackPlacement(ctx, tx, req.RackID, req.RackPosition, req.USize, 0); err != nil {
		return 0, err
	}
	if err := validateItemPorts(req); err != nil {
		return 0, err
	}
	if err := enforceSoftwareLicenseLimits(ctx, tx, 0, req); err != nil {
		return 0, err
	}
	result, err := s.repo.InsertItem(ctx, tx, args...)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	if err := s.replaceRelations(ctx, tx, id, req, nil); err != nil {
		return 0, err
	}
	now := time.Now().Unix()
	if _, err := tx.ExecContext(ctx, `INSERT INTO actions (itemid, actiondate, description, invoiceinfo, isauto) VALUES (?, ?, ?, '成功', 1)`, id, now, fmt.Sprintf("由 %s 用户创建", actor.Username)); err != nil {
		return 0, err
	}
	if err := s.recordItemAudit(ctx, tx, actor, AuditOutcomeCreate, id, req, true, ""); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *ItemMutationWorkflow) Update(ctx context.Context, actor UserActor, id int64, req domain.ItemPayload) error {
	note := strings.TrimSpace(req.ChangeNote)
	updateDesc := ""
	if note != "" {
		updateDesc = fmt.Sprintf("由 %s 用户更新：%s配置", actor.Username, note)
	}
	if err := s.update(ctx, actor, id, req, updateDesc); err != nil {
		if updateDesc != "" {
			_ = s.repo.RecordItemAction(ctx, id, updateDesc, "失败")
		}
		s.recordItemAuditFailure(ctx, actor, AuditOutcomeUpdate, id, req, err)
		return err
	}
	return nil
}

func (s *ItemMutationWorkflow) update(ctx context.Context, actor UserActor, id int64, req domain.ItemPayload, updateDesc string) error {
	args, err := itemUpdateArgs(req)
	if err != nil {
		return err
	}
	tx, err := s.repo.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := enforceRackPlacement(ctx, tx, req.RackID, req.RackPosition, req.USize, id); err != nil {
		return err
	}
	if err := validateItemPorts(req); err != nil {
		return err
	}
	if err := enforceSoftwareLicenseLimits(ctx, tx, id, req); err != nil {
		return err
	}
	oldFiles, err := s.repo.ItemFileIDs(ctx, tx, id)
	if err != nil {
		return err
	}
	if _, err := s.repo.UpdateItem(ctx, tx, id, args...); err != nil {
		return err
	}
	if updateDesc != "" {
		now := time.Now().Unix()
		var lastDesc string
		var lastAction sql.NullInt64
		_ = tx.QueryRowContext(ctx, `SELECT description, actiondate FROM actions WHERE itemid = ? ORDER BY id DESC LIMIT 1`, id).Scan(&lastDesc, &lastAction)
		if lastDesc != updateDesc || !sameDayValue(lastAction.Int64, now) {
			if _, err := tx.ExecContext(ctx, `INSERT INTO actions (itemid, actiondate, description, invoiceinfo, isauto) VALUES (?, ?, ?, '成功', 1)`, id, now, updateDesc); err != nil {
				return err
			}
		}
	}
	if err := s.replaceRelations(ctx, tx, id, req, oldFiles); err != nil {
		return err
	}
	if strings.TrimSpace(req.ChangeNote) != "" {
		if err := s.recordItemAudit(ctx, tx, actor, AuditOutcomeUpdate, id, req, true, req.ChangeNote); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *ItemMutationWorkflow) Delete(ctx context.Context, actor UserActor, id int64) error {
	tx, err := s.repo.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var switchCount int64
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM items WHERE switchid = ? AND id <> ?`, id, id).Scan(&switchCount); err != nil {
		return err
	}
	if switchCount > 0 {
		return NewConflictError("该交换机类型硬件已被 %d 条硬件记录使用，无法删除", switchCount)
	}
	if err := s.recordItemAudit(ctx, tx, actor, AuditOutcomeDelete, id, domain.ItemPayload{}, true, ""); err != nil {
		return err
	}
	if err := s.recordTagDetachAudit(ctx, tx, actor, id); err != nil {
		return err
	}
	if err := s.repo.DeleteItemReferences(ctx, tx, id); err != nil {
		return err
	}
	return tx.Commit()
}

// itemName 按优先级组装硬件显示名：删除按库中数据，新增/更新按请求参数 + 关联名称查询
func (s *ItemMutationWorkflow) itemName(ctx context.Context, exec repository.Executor, req domain.ItemPayload, id int64, withID bool) string {
	if req.Model == "" && req.ManufacturerID == 0 {
		if name, err := LoadItemName(ctx, exec, id); err == nil {
			return name
		}
	}
	var manufacturer, itemType string
	_ = exec.QueryRowContext(ctx, `SELECT COALESCE(title, '') FROM agents WHERE id = ?`, req.ManufacturerID).Scan(&manufacturer)
	_ = exec.QueryRowContext(ctx, `SELECT COALESCE(typedesc, '') FROM itemtypes WHERE id = ?`, req.ItemTypeID).Scan(&itemType)
	return FormatHardwareName(manufacturer, req.Model, itemType, id, withID)
}

// recordItemAudit 在事务内写入硬件显式审计事件（含原始 SQL 便于追溯主语句）
func (s *ItemMutationWorkflow) recordItemAudit(ctx context.Context, tx *sql.Tx, actor UserActor, outcome AuditOutcome, id int64, req domain.ItemPayload, success bool, changes string) error {
	name := s.itemName(ctx, tx, req, id, id > 0)
	target := formatAssetTarget("硬件", id)
	if outcome == AuditOutcomeCreate && id <= 0 {
		target = "-"
	}
	event := AuditEvent{
		Module: AuditModuleAssets,
		Action: AssetOutcomeAction(outcome, "硬件"),
		Target: target,
		Detail: AuditEntityDetail(name, outcome, success, changes, ""),
		Result: AuditResultSuccess,
	}
	return s.audit.RecordEventTx(ctx, tx, actor.Username, actor.IP, event)
}

// recordTagDetachAudit 硬件删除时对每个关联标记写“删除标记关联”审计（解除关联硬件）
func (s *ItemMutationWorkflow) recordTagDetachAudit(ctx context.Context, tx *sql.Tx, actor UserActor, id int64) error {
	tagNames, err := loadItemTagNames(ctx, tx, id)
	if err != nil {
		return err
	}
	if len(tagNames) == 0 {
		return nil
	}
	itemName, err := LoadItemName(ctx, tx, id)
	if err != nil {
		return err
	}
	for _, tag := range tagNames {
		event := AuditEvent{
			Module: AuditModuleCatalog,
			Action: "删除标记关联",
			Target: tag,
			Detail: tag + " 已解除关联硬件：" + itemName,
			Result: AuditResultSuccess,
		}
		if err := s.audit.RecordEventTx(ctx, tx, actor.Username, actor.IP, event); err != nil {
			return err
		}
	}
	return nil
}

// loadItemTagNames 查询硬件关联的全部标记名称（按标记编号排序）
func loadItemTagNames(ctx context.Context, exec repository.Executor, itemID int64) ([]string, error) {
	rows, err := exec.QueryContext(ctx, `SELECT t.name FROM tag2item ti JOIN tags t ON t.id = ti.tagid WHERE ti.itemid = ? ORDER BY t.id`, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	names := make([]string, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		if strings.TrimSpace(name) != "" {
			names = append(names, strings.TrimSpace(name))
		}
	}
	return names, rows.Err()
}

// recordItemAuditFailure 失败路径（事务已回滚）在事务外补记失败审计事件
func (s *ItemMutationWorkflow) recordItemAuditFailure(ctx context.Context, actor UserActor, outcome AuditOutcome, id int64, req domain.ItemPayload, auditErr error) {
	name := s.itemName(ctx, s.repo, req, id, id > 0)
	event := AuditEvent{
		Module: AuditModuleAssets,
		Action: AssetOutcomeAction(outcome, "硬件"),
		Target: formatAssetTarget("硬件", id),
		Detail: AuditEntityDetail(name, outcome, false, "", ConflictReason(auditErr)),
		Result: AuditResultFailure,
	}
	_ = s.audit.RecordEvent(ctx, actor.Username, actor.IP, event)
}

func (s *ItemMutationWorkflow) replaceRelations(ctx context.Context, tx *sql.Tx, id int64, req domain.ItemPayload, oldFiles []int64) error {
	if err := s.relations.ReplaceUndirectedItemLinks(tx, id, req.ItemLinks); err != nil {
		return err
	}
	for _, pair := range [][3]string{{`DELETE FROM item2inv WHERE itemid = ?`, `INSERT INTO item2inv (itemid, invid) VALUES (?, ?)`, `invoice`}, {`DELETE FROM item2soft WHERE itemid = ?`, `INSERT INTO item2soft (itemid, softid) VALUES (?, ?)`, `software`}, {`DELETE FROM contract2item WHERE itemid = ?`, `INSERT INTO contract2item (itemid, contractid) VALUES (?, ?)`, `contract`}, {`DELETE FROM item2file WHERE itemid = ?`, `INSERT INTO item2file (itemid, fileid) VALUES (?, ?)`, `file`}} {
		var links []int64
		switch pair[2] {
		case "invoice":
			links = req.InvoiceLinks
		case "software":
			links = req.SoftwareLinks
		case "contract":
			links = req.ContractLinks
		case "file":
			links = req.FileLinks
		}
		if err := s.relations.ReplaceIDLinks(tx, pair[0], pair[1], id, links); err != nil {
			return err
		}
	}
	if len(oldFiles) > 0 {
		return s.relations.CleanupRemovedFileLinks(tx, oldFiles, req.FileLinks, req.CleanupFileLinks)
	}
	return nil
}

func itemCreateArgs(req domain.ItemPayload) ([]interface{}, error) { return itemCommonArgs(req) }
func itemUpdateArgs(req domain.ItemPayload) ([]interface{}, error) {
	if err := domain.ValidateItem(req); err != nil {
		return nil, err
	}
	purchaseDate, err := parsedDate(req.PurchaseDate)
	if err != nil {
		return nil, err
	}
	return []interface{}{req.ItemTypeID, req.Function, req.ManufacturerID, req.Label, req.WarrInfo, req.Model, req.SN, req.SN2, req.SN3, nullable(req.LocationID), nullable(req.LocAreaID), req.Origin, nullable(req.WarrantyMonths), purchaseDate, req.PurchPrice, req.DNSName, nullable(req.UserID), nullable(req.DptID), req.Principal, req.Comments, req.MaintenanceInfo, req.IsPart, req.HD, req.CPU, req.CPUNo, req.CoresPerCPU, req.RAM, req.Raid, req.RaidConfig, req.RackMountable, nullable(req.RackID), nullable(req.RackPosition), nullable(req.RackPosDepth), nullable(req.USize), req.Status, req.MACs, req.IPv4, req.IPv6, req.RemAdmIP, req.PanelPort, nullable(req.SwitchID), req.SwitchPort, req.Ports}, nil
}
func itemCommonArgs(req domain.ItemPayload) ([]interface{}, error) {
	if err := domain.ValidateItem(req); err != nil {
		return nil, err
	}
	purchaseDate, err := parsedDate(req.PurchaseDate)
	if err != nil {
		return nil, err
	}
	return []interface{}{req.Label, req.ItemTypeID, req.Function, req.ManufacturerID, req.WarrInfo, req.Model, req.SN, req.SN2, req.SN3, req.Origin, nullable(req.WarrantyMonths), purchaseDate, req.PurchPrice, req.DNSName, nullable(req.DptID), req.Principal, nullable(req.LocationID), nullable(req.LocAreaID), nullable(req.UserID), req.MaintenanceInfo, req.Comments, req.IsPart, nullable(req.RackID), nullable(req.RackPosition), nullable(req.RackPosDepth), req.RackMountable, nullable(req.USize), req.Status, req.MACs, req.IPv4, req.IPv6, req.RemAdmIP, req.HD, req.CPU, req.CPUNo, req.CoresPerCPU, req.RAM, req.Raid, req.RaidConfig, req.PanelPort, nullable(req.SwitchID), req.SwitchPort, req.Ports}, nil
}
func nullable(v *int64) interface{} { return primitives.NullableInt(v) }
func parsedDate(raw string) (interface{}, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	v, err := primitives.ParseDateInput(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid purchaseDate")
	}
	return primitives.NullableInt64Value(v), nil
}
func sameDayValue(a, b int64) bool {
	if a == 0 {
		return false
	}
	ta, tb := time.Unix(a, 0), time.Unix(b, 0)
	return ta.Year() == tb.Year() && ta.YearDay() == tb.YearDay()
}

// validateItemPorts 校验网络端口格式：留空合法，填写时必须为 0-65535 的整数
func validateItemPorts(req domain.ItemPayload) error {
	trimmed := strings.TrimSpace(req.Ports)
	if trimmed == "" {
		return nil
	}
	port, err := strconv.Atoi(trimmed)
	if err != nil || port < 0 || port > 65535 {
		return fmt.Errorf("invalid ports")
	}
	return nil
}

// enforceSoftwareLicenseLimits 关联按CPU/按核心类型的软件前，校验本硬件的 CPU 数量、
// 每CPU核心数已按口径填写（缺失即报错，不再隐式按 0 计），再校验授权上限
func enforceSoftwareLicenseLimits(ctx context.Context, tx *sql.Tx, itemID int64, req domain.ItemPayload) error {
	if len(req.SoftwareLinks) == 0 {
		return nil
	}
	var messages []string
	seen := make(map[int64]bool)
	cpuNo := licenseUsageNumber(req.CPUNo)
	coresPerCPU := licenseUsageNumber(req.CoresPerCPU)
	for _, softwareID := range req.SoftwareLinks {
		if softwareID <= 0 || seen[softwareID] {
			continue
		}
		seen[softwareID] = true
		var licensed sql.NullInt64
		var licenseType int64
		var title, version string
		if err := tx.QueryRowContext(ctx, `SELECT licqty, COALESCE(lictype, 0), COALESCE(NULLIF(TRIM(stitle), ''), '未知'), COALESCE(NULLIF(TRIM(sversion), ''), '-') FROM software WHERE id = ?`, softwareID).Scan(&licensed, &licenseType, &title, &version); err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			return err
		}
		if !licensed.Valid {
			messages = append(messages, fmt.Sprintf("软件 %s %s 未配置授权数量，请先在软件中进行配置", title, version))
			continue
		}
		if licenseType == 1 && cpuNo <= 0 {
			messages = append(messages, fmt.Sprintf("软件 %s %s 采用按 CPU 授权，需填写 CPU 数量", title, version))
		}
		if licenseType == 2 && (cpuNo <= 0 || coresPerCPU <= 0) {
			messages = append(messages, fmt.Sprintf("软件 %s %s 采用按核心授权，需填写 CPU 数量及单 CPU 核心数", title, version))
		}
	}
	if len(messages) > 0 {
		return NewConflictError("%s", strings.Join(messages, "\n"))
	}
	for _, softwareID := range req.SoftwareLinks {
		if softwareID <= 0 {
			continue
		}
		var licensed sql.NullInt64
		var licenseType int64
		var title, version string
		if err := tx.QueryRowContext(ctx, `SELECT licqty, COALESCE(lictype, 0), COALESCE(NULLIF(TRIM(stitle), ''), '未知'), COALESCE(NULLIF(TRIM(sversion), ''), '-') FROM software WHERE id = ?`, softwareID).Scan(&licensed, &licenseType, &title, &version); err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			return err
		}
		if !licensed.Valid {
			continue
		}
		var others int64
		if err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(CASE ?
				WHEN 1 THEN COALESCE(CAST(items.cpuno AS INTEGER), 0)
				WHEN 2 THEN COALESCE(CAST(items.cpuno AS INTEGER), 0) * COALESCE(CAST(items.corespercpu AS INTEGER), 0)
				ELSE 1
			END), 0)
			FROM item2soft
			JOIN items ON items.id = item2soft.itemid
			WHERE item2soft.softid = ? AND items.id <> ?`, licenseType, softwareID, itemID).Scan(&others); err != nil {
			return err
		}
		if others+itemLicenseUsage(licenseType, req.CPUNo, req.CoresPerCPU) > licensed.Int64 {
			messages = append(messages, fmt.Sprintf("软件 %s %s %s授权已达上限，请先扩容授权", title, version, softwareLicenseTypeLabel(licenseType)))
		}
	}
	if len(messages) > 0 {
		return NewConflictError("%s", strings.Join(messages, "\n"))
	}
	return nil
}

// itemLicenseUsage 计算单台硬件按授权类型的授权消耗
func itemLicenseUsage(licenseType int64, cpuNo, coresPerCPU string) int64 {
	cpu := licenseUsageNumber(cpuNo)
	cores := licenseUsageNumber(coresPerCPU)
	switch licenseType {
	case 1:
		return cpu
	case 2:
		return cpu * cores
	default:
		return 1
	}
}

func licenseUsageNumber(raw string) int64 {
	v, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || v < 0 {
		return 0
	}
	return int64(v)
}

// enforceRackPlacement 硬件选择了机架且位置、大小(U)齐备时，校验机架放置不越界、不冲突；
// 放置信息不完整时交由前端必填提示处理
func enforceRackPlacement(ctx context.Context, tx *sql.Tx, rackID, position, uSize *int64, excludeItemID int64) error {
	if rackID == nil || *rackID <= 0 || position == nil || *position <= 0 || uSize == nil || *uSize <= 0 {
		return nil
	}
	message, err := ValidateRackPlacement(ctx, tx, *rackID, excludeItemID, *position, *uSize)
	if err != nil {
		return err
	}
	if message != "" {
		return NewConflictError("%s", message)
	}
	return nil
}

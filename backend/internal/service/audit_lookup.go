package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"itdb-backend/internal/repository"
)

// 本文件提供审计详情所需的实体显示名查询：按主键加载各实体的名称组合，供工作流在记录显式审计事件（含失败回滚后补记）时复用。

// LoadItemName 按编号加载硬件显示名（厂商 型号[硬件类型, ID: n]）
func LoadItemName(ctx context.Context, exec repository.Executor, id int64) (string, error) {
	var manufacturer, model, itemType sql.NullString
	err := exec.QueryRowContext(ctx,
		`SELECT COALESCE(a.title, ''), i.model, COALESCE(t.typedesc, '')
		FROM items i
		LEFT JOIN agents a ON a.id = i.manufacturerid
		LEFT JOIN itemtypes t ON t.id = i.itemtypeid
		WHERE i.id = ?`, id).Scan(&manufacturer, &model, &itemType)
	if err != nil {
		return "", err
	}
	return FormatHardwareName(manufacturer.String, model.String, itemType.String, id, true), nil
}

// LoadSoftwareName 按编号加载软件显示名（厂商 标题 版本[ID: n]）
func LoadSoftwareName(ctx context.Context, exec repository.Executor, id int64) (string, error) {
	var manufacturer, title, version sql.NullString
	err := exec.QueryRowContext(ctx,
		`SELECT COALESCE(a.title, ''), COALESCE(NULLIF(TRIM(s.stitle), ''), '未知'), COALESCE(NULLIF(TRIM(s.sversion), ''), '-')
		FROM software s
		LEFT JOIN agents a ON a.id = s.manufacturerid
		WHERE s.id = ?`, id).Scan(&manufacturer, &title, &version)
	if err != nil {
		return "", err
	}
	return FormatSoftwareName(manufacturer.String, title.String, version.String, id, true), nil
}

// LoadSimpleName 加载单名称实体（单据/代理/文件/合同）的显示名
func LoadSimpleName(ctx context.Context, exec repository.Executor, table, column string, id int64) (string, error) {
	if !auditSafeIdentifier(table) || !auditSafeIdentifier(column) {
		return "", fmt.Errorf("invalid audit identifier")
	}
	var name sql.NullString
	err := exec.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT COALESCE(%s, '') FROM %s WHERE id = ?`, column, table), id).Scan(&name)
	if err != nil {
		return "", err
	}
	return FormatSimpleName(name.String, id, true), nil
}

// LoadLocationName 按编号加载地点显示名（名称-区域(楼层: x)[ID: n]，区域多个以顿号合并）
func LoadLocationName(ctx context.Context, exec repository.Executor, id int64) (string, error) {
	var name, floor sql.NullString
	if err := exec.QueryRowContext(ctx,
		`SELECT COALESCE(name, ''), COALESCE(floor, '') FROM locations WHERE id = ?`, id).Scan(&name, &floor); err != nil {
		return "", err
	}
	areas, err := loadLocationAreas(ctx, exec, id)
	if err != nil {
		return "", err
	}
	return FormatLocationName(name.String, areas, floor.String, id, true), nil
}

// LoadRackName 按编号加载机架显示名（标签(地点, 区域: x)[ID: n]）
func LoadRackName(ctx context.Context, exec repository.Executor, id int64) (string, error) {
	var label, locationName, locationFloor, area sql.NullString
	err := exec.QueryRowContext(ctx,
		`SELECT r.label, COALESCE(l.name, ''), COALESCE(l.floor, ''), COALESCE(a.areaname, '')
		FROM racks r
		LEFT JOIN locations l ON l.id = r.locationid
		LEFT JOIN locareas a ON a.id = r.locareaid
		WHERE r.id = ?`, id).Scan(&label, &locationName, &locationFloor, &area)
	if err != nil {
		return "", err
	}
	location := FormatRackLocationText(locationName.String, locationFloor.String)
	return FormatRackName(label.String, location, area.String, id, true), nil
}

// loadLocationAreas 加载地点下的全部区域名称
func loadLocationAreas(ctx context.Context, exec repository.Executor, locationID int64) ([]string, error) {
	rows, err := exec.QueryContext(ctx, `SELECT COALESCE(areaname, '') FROM locareas WHERE locationid = ? ORDER BY id`, locationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	areas := make([]string, 0)
	for rows.Next() {
		var area string
		if err := rows.Scan(&area); err != nil {
			return nil, err
		}
		if trimmed := strings.TrimSpace(area); trimmed != "" {
			areas = append(areas, trimmed)
		}
	}
	return areas, rows.Err()
}

// auditSafeIdentifier 限制表名/列名只能来自内置白名单式的标识符
func auditSafeIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
		default:
			return false
		}
	}
	return true
}

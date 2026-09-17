package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"itdb-backend/internal/common/primitives"
	"itdb-backend/internal/common/statustypes"
	"itdb-backend/internal/repository"
)

// 本文件提供资料管理各类字典（硬件类型、合同类型、标记等）的审计事件构建：目标列、新增/更新/删除详情格式集中在此，供资产域路由在写入后记录。

type DictionaryAuditMeta struct {
	Table       string
	Label       string
	Column      string
	HasSoftware bool
	HasColor    bool
	HasParent   bool
}

// DictionaryAuditMetaByName 各字典的审计元信息
func DictionaryAuditMetaByName(name string) (DictionaryAuditMeta, error) {
	switch name {
	case "itemtypes":
		return DictionaryAuditMeta{Table: name, Label: "硬件类型", Column: "typedesc", HasSoftware: true}, nil
	case "filetypes":
		return DictionaryAuditMeta{Table: name, Label: "文件类型", Column: "typedesc"}, nil
	case "statustypes":
		return DictionaryAuditMeta{Table: name, Label: "状态类型", Column: "statusdesc", HasColor: true}, nil
	case "dpttypes":
		return DictionaryAuditMeta{Table: name, Label: "部门", Column: "dptname"}, nil
	case "contracttypes":
		return DictionaryAuditMeta{Table: name, Label: "合同类型", Column: "name"}, nil
	case "contractsubtypes":
		return DictionaryAuditMeta{Table: name, Label: "合同子类型", Column: "name", HasParent: true}, nil
	case "tags":
		return DictionaryAuditMeta{Table: name, Label: "标记", Column: "name"}, nil
	default:
		return DictionaryAuditMeta{}, fmt.Errorf("不支持的字典类型")
	}
}

// LoadDictionaryAuditValues 读取字典行的审计相关值：名称、支持软件、颜色、父类型编号
func LoadDictionaryAuditValues(ctx context.Context, exec repository.Executor, meta DictionaryAuditMeta, id int64) (name, color string, hasSoftware bool, parentID int64, err error) {
	columns := []string{fmt.Sprintf("COALESCE(%s, '')", meta.Column)}
	if meta.HasSoftware {
		columns = append(columns, "COALESCE(hassoftware, 0)")
	}
	if meta.HasColor {
		columns = append(columns, "COALESCE(color, '')")
	}
	if meta.HasParent {
		columns = append(columns, "COALESCE(contypeid, 0)")
	}
	dests := []interface{}{&name}
	var softwareValue int64
	if meta.HasSoftware {
		dests = append(dests, &softwareValue)
	}
	var colorHolder sql.NullString
	if meta.HasColor {
		dests = append(dests, &colorHolder)
	}
	if meta.HasParent {
		dests = append(dests, &parentID)
	}
	query := fmt.Sprintf(`SELECT %s FROM %s WHERE id = ?`, strings.Join(columns, ", "), dictionaryTableName(meta))
	if err = exec.QueryRowContext(ctx, query, id).Scan(dests...); err != nil {
		return "", "", false, 0, err
	}
	hasSoftware = softwareValue == 1
	color = colorHolder.String
	return name, color, hasSoftware, parentID, nil
}

// DictionaryTargetText 字典审计目标列内容：统一为名称本身（合同子类型也只显示子类型名称）
func DictionaryTargetText(name string) string {
	return strings.TrimSpace(name)
}

// DictionaryParentName 查询合同类型的名称，供合同子类型详情的“父合同”标注
func DictionaryParentName(ctx context.Context, exec repository.Executor, parentID int64) string {
	if parentID <= 0 {
		return ""
	}
	var parentName string
	_ = exec.QueryRowContext(ctx, `SELECT COALESCE(name, '') FROM contracttypes WHERE id = ?`, parentID).Scan(&parentName)
	return strings.TrimSpace(parentName)
}

// DictionaryCreatedDetail 新增详情：“<名称>(支持软件: 是) 已创建”“<子类型>(父合同: <类型>) 已创建”等
func DictionaryCreatedDetail(meta DictionaryAuditMeta, name, color string, hasSoftware bool, parentName string) string {
	return dictionaryEntityText(meta, name, color, hasSoftware, parentName) + " 已创建"
}

// DictionaryDeletedDetail 删除详情
func DictionaryDeletedDetail(meta DictionaryAuditMeta, name, color string, hasSoftware bool, parentName string) string {
	return dictionaryEntityText(meta, name, color, hasSoftware, parentName) + " 已删除"
}

// DictionaryUpdatedDetail 更新详情：“<旧名称> 已更新：名称已改为 <新名称>，支持软件由 “是” 改为 “否””；
// 无任何变化时 changed 为 false
func DictionaryUpdatedDetail(ctx context.Context, exec repository.Executor, meta DictionaryAuditMeta, id int64, oldName, oldColor string, oldSoftware bool, oldParent int64, body map[string]interface{}) (detail string, changed bool) {
	newName := strings.TrimSpace(primitives.AsString(body[meta.Column]))
	parts := make([]string, 0, 2)
	if newName != "" && newName != strings.TrimSpace(oldName) {
		parts = append(parts, fmt.Sprintf("名称已改为 %s", newName))
	}
	if meta.HasSoftware {
		newSoftware := primitives.AsInt64(body["hassoftware"]) == 1
		if newSoftware != oldSoftware {
			parts = append(parts, fmt.Sprintf("支持软件由 %q 改为 %q", softwareYesNo(oldSoftware), softwareYesNo(newSoftware)))
		}
	}
	if meta.HasColor {
		newColor := statustypes.NormalizeHexColor(primitives.AsString(body["color"]))
		if fixedColor, ok := statustypes.FixedStatusTypeColor(newName); ok {
			newColor = fixedColor
		}
		if !strings.EqualFold(strings.TrimSpace(newColor), strings.TrimSpace(oldColor)) {
			parts = append(parts, fmt.Sprintf("状态颜色由 %q 改为 %q", strings.TrimSpace(oldColor), strings.TrimSpace(newColor)))
		}
	}
	if meta.HasParent {
		newParent := primitives.AsInt64(body["contypeid"])
		if newParent != oldParent {
			var parentName string
			_ = exec.QueryRowContext(ctx, `SELECT COALESCE(name, '') FROM contracttypes WHERE id = ?`, newParent).Scan(&parentName)
			if parent := strings.TrimSpace(parentName); parent != "" {
				parts = append(parts, fmt.Sprintf("父合同已改为 %s", parent))
			}
		}
	}
	if len(parts) == 0 {
		return "", false
	}
	subject := dictionaryEntityText(meta, strings.TrimSpace(oldName), oldColor, oldSoftware, DictionaryParentName(ctx, exec, oldParent))
	return subject + " 已更新：" + strings.Join(parts, "，"), true
}

// DictionaryFailureDetail 字典操作失败详情：“<名称> 创建失败：<原因>”
func DictionaryFailureDetail(base string, outcome AuditOutcome, reason string) string {
	return AuditFailureDetail(base+" "+outcome.phrase(false), reason)
}

// dictionaryEntityText 字典实体展示文本：硬件类型带支持软件、状态类型带状态颜色、
// 合同子类型带父合同标注（父名称缺失时省略括注）
func dictionaryEntityText(meta DictionaryAuditMeta, name, color string, hasSoftware bool, parentName string) string {
	name = strings.TrimSpace(name)
	if meta.HasParent {
		if parent := strings.TrimSpace(parentName); parent != "" {
			return fmt.Sprintf("%s(父合同: %s)", name, parent)
		}
		return name
	}
	if meta.HasSoftware {
		return fmt.Sprintf("%s(支持软件: %s)", name, softwareYesNo(hasSoftware))
	}
	if meta.HasColor {
		return fmt.Sprintf("%s(状态颜色: %s)", name, strings.TrimSpace(color))
	}
	return name
}

func dictionaryTableName(meta DictionaryAuditMeta) string {
	return meta.Table
}

func softwareYesNo(value bool) string {
	if value {
		return "是"
	}
	return "否"
}

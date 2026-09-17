package service

import (
	"errors"
	"fmt"
	"strings"

	"itdb-backend/internal/common/localizer"
)

// 本文件集中维护审计日志的中文展示格式：各实体的目标名称、操作结果短语，供各工作流在写入显式审计事件时复用，保证操作列与详情弹窗文案一致。

// formatAssetTarget 生成“xx编号 N”形式的资产目标列内容
func formatAssetTarget(prefix string, id int64) string {
	if id > 0 {
		return fmt.Sprintf("%s编号 %d", prefix, id)
	}
	return prefix + "编号 -"
}

// auditEntityName 返回不带 ID 段的实体显示名；空段按各实体的规则裁剪
func joinNonEmpty(parts ...string) string {
	trimmed := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			trimmed = append(trimmed, value)
		}
	}
	return strings.Join(trimmed, " ")
}

// FormatHardwareName 硬件名称："<厂商> <型号>[<硬件类型>, ID: <编号>]"；
// withID 为 false 时省略 ID 段（创建失败尚无编号时使用）
func FormatHardwareName(manufacturer, model, itemType string, id int64, withID bool) string {
	head := joinNonEmpty(manufacturer, model)
	inner := strings.TrimSpace(itemType)
	if withID && id > 0 {
		if inner != "" {
			inner += ", "
		}
		inner += fmt.Sprintf("ID: %d", id)
	}
	if inner == "" {
		return head
	}
	return fmt.Sprintf("%s[%s]", head, inner)
}

// FormatSoftwareName 软件名称：“<厂商> <标题> <版本>[ID: <编号>]”
func FormatSoftwareName(manufacturer, title, version string, id int64, withID bool) string {
	head := joinNonEmpty(manufacturer, title, version)
	if !withID || id <= 0 {
		return head
	}
	return fmt.Sprintf("%s[ID: %d]", head, id)
}

// FormatSimpleName 单名称实体（单据/代理/文件/合同）：“<名称>[ID: <编号>]”
func FormatSimpleName(name string, id int64, withID bool) string {
	name = strings.TrimSpace(name)
	if !withID || id <= 0 {
		return name
	}
	return fmt.Sprintf("%s[ID: %d]", name, id)
}

// FormatLocationName 地点名称：“<地点名称>(区域: <区域/办公室>, 楼层: <楼层>)[ID: <编号>]”，
// 区域多项以顿号合并，未配置区域时省略“区域”段
func FormatLocationName(name string, areas []string, floor string, id int64, withID bool) string {
	head := strings.TrimSpace(name)
	if region := strings.Join(nonEmpty(areas), "、"); region != "" {
		head = fmt.Sprintf("%s(区域: %s, 楼层: %s)", head, region, strings.TrimSpace(floor))
	} else {
		head = fmt.Sprintf("%s(楼层: %s)", head, strings.TrimSpace(floor))
	}
	if !withID || id <= 0 {
		return head
	}
	return fmt.Sprintf("%s[ID: %d]", head, id)
}

// FormatRackName 机架名称：“<标签>(<地点>, 区域: <区域>)[ID: <编号>]”，区域为空时去掉“, 区域: <区域>”
func FormatRackName(label, location, area string, id int64, withID bool) string {
	head := strings.TrimSpace(label)
	if area = strings.TrimSpace(area); area != "" {
		head = fmt.Sprintf("%s(%s, 区域: %s)", head, strings.TrimSpace(location), area)
	} else {
		head = fmt.Sprintf("%s(%s)", head, strings.TrimSpace(location))
	}
	if !withID || id <= 0 {
		return head
	}
	return fmt.Sprintf("%s[ID: %d]", head, id)
}

// AuditOutcomeAction 生成详情动作短语：成功为“已创建/已更新/已删除/已导出/已导入”，
// 失败统一替换为“创建失败/更新失败/删除失败/导出失败/导入失败”
type AuditOutcome int

const (
	AuditOutcomeCreate AuditOutcome = iota
	AuditOutcomeUpdate
	AuditOutcomeDelete
	AuditOutcomeExport
	AuditOutcomeImport
)

func (o AuditOutcome) phrase(success bool) string {
	if success {
		switch o {
		case AuditOutcomeCreate:
			return "已创建"
		case AuditOutcomeUpdate:
			return "已更新"
		case AuditOutcomeDelete:
			return "已删除"
		case AuditOutcomeExport:
			return "已导出"
		case AuditOutcomeImport:
			return "已导入"
		}
		return "已操作"
	}
	switch o {
	case AuditOutcomeCreate:
		return "创建失败"
	case AuditOutcomeUpdate:
		return "更新失败"
	case AuditOutcomeDelete:
		return "删除失败"
	case AuditOutcomeExport:
		return "导出失败"
	case AuditOutcomeImport:
		return "导入失败"
	}
	return "操作失败"
}

// AuditEntityDetail 生成实体详情：成功时“<名称>[ 已更新：<变更项>配置]”，
// 失败时“<名称> 更新失败：<原因>”
func AuditEntityDetail(name string, outcome AuditOutcome, success bool, changes, reason string) string {
	phrase := outcome.phrase(success)
	if !success {
		name = fmt.Sprintf("%s %s", name, phrase)
		if reason = strings.TrimSpace(reason); reason != "" {
			return name + "：" + reason
		}
		return name
	}
	if changes = strings.TrimSpace(changes); changes != "" && outcome == AuditOutcomeUpdate {
		return fmt.Sprintf("%s %s：%s配置", name, phrase, changes)
	}
	return fmt.Sprintf("%s %s", name, phrase)
}

// AuditFailureDetail 失败详情：在基础短语后追加以“：”连接的失败原因
func AuditFailureDetail(base, reason string) string {
	if reason = strings.TrimSpace(reason); reason != "" {
		return base + "：" + reason
	}
	return base
}

// FormatRackLocationText 机架的地点文本与机架编辑窗口地点选项一致：
// “<地点>, 楼层:<楼层>”，未配置楼层时仅显示地点名称
func FormatRackLocationText(name, floor string) string {
	name = strings.TrimSpace(name)
	if floor = strings.TrimSpace(floor); floor != "" {
		return fmt.Sprintf("%s, 楼层:%s", name, floor)
	}
	return name
}

// AssetOutcomeAction 生成资产管理操作名：新增硬件/更新硬件/删除硬件/导出硬件等
func AssetOutcomeAction(outcome AuditOutcome, entity string) string {
	switch outcome {
	case AuditOutcomeCreate:
		return "新增" + entity
	case AuditOutcomeUpdate:
		return "更新" + entity
	case AuditOutcomeDelete:
		return "删除" + entity
	case AuditOutcomeExport:
		return "导出" + entity
	case AuditOutcomeImport:
		return "导入" + entity
	}
	return "操作" + entity
}

// ConflictReason 提取审计失败详情的中文原因：业务冲突错误取其消息，其余错误经本地化词典转换
func ConflictReason(err error) string {
	if err == nil {
		return ""
	}
	var conflict *ConflictError
	if errors.As(err, &conflict) {
		return conflict.Message
	}
	return localizer.LocalizeMessage(err.Error())
}

// IsNameConflictError 判断错误是否为名称已存在类冲突；此类创建尝试对照资料管理口径不写入审计
func IsNameConflictError(err error) bool {
	var conflict *ConflictError
	if errors.As(err, &conflict) {
		return strings.Contains(conflict.Message, "已存在")
	}
	return false
}

// SplitAuditTargetNames 按顶层顿号拆分目标文本（括号内的顿号不计），
// 用于导入类事件目标列的截断展示
func SplitAuditTargetNames(target string) []string {
	var parts []string
	depth := 0
	current := strings.Builder{}
	for _, r := range target {
		switch {
		case r == '(' || r == '（':
			depth++
			current.WriteRune(r)
		case r == ')' || r == '）':
			if depth > 0 {
				depth--
			}
			current.WriteRune(r)
		case r == '、' && depth == 0:
			parts = append(parts, current.String())
			current.Reset()
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// TruncateAuditTarget 目标列截断展示：超过 maxCount 个名称时仅保留前 maxCount 个并以“...”结尾
func TruncateAuditTarget(target string, maxCount int) (display string, truncated bool) {
	if maxCount <= 0 {
		return target, false
	}
	parts := SplitAuditTargetNames(target)
	if len(parts) <= maxCount {
		return target, false
	}
	return strings.Join(parts[:maxCount], "、") + "...", true
}

func nonEmpty(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

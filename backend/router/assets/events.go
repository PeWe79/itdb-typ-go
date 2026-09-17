package assets

import (
	"fmt"
	"itdb-backend/internal/service"
	"itdb-backend/router/common"
	"net/http"
	"strconv"
	"strings"
)

// 本文件提供前端操作的审计上报端点：数据导出、字典导入与标签打印由前端执行，通过白名单事件类型在服务端校验权限并生成统一格式的审计记录。

// trackedEventRequest 前端上报的审计事件载荷
type trackedEventRequest struct {
	Type     string   `json:"type"`
	Names    []string `json:"names"`
	Existing []string `json:"existing"`
	Target   string   `json:"target"`
	Count    int      `json:"count"`
	Result   string   `json:"result"`
}

// trackedEventSpec 白名单事件类型的模块、实体名、操作名、详情与权限
type trackedEventSpec struct {
	module     string
	action     string
	detail     string
	permission string
}

// trackedEventSpecs 前端可上报的事件类型白名单
func trackedEventSpecs() map[string]trackedEventSpec {
	specs := make(map[string]trackedEventSpec)
	assetExports := map[string]string{
		"items": "硬件", "software": "软件", "invoices": "单据", "agents": "代理",
		"files": "文件", "contracts": "合同", "locations": "地点", "racks": "机架",
	}
	for resource, entity := range assetExports {
		detail := "所有" + entity + "数据已导出"
		specs["export:"+resource] = trackedEventSpec{
			module: service.AuditModuleAssets,
			action: "导出" + entity, detail: detail,
			permission: assetPermission(resource, "read"),
		}
	}
	specs["export:items-maintenance"] = trackedEventSpec{
		module:     service.AuditModuleAssets,
		action:     "导出硬件维护日志",
		permission: assetPermission("items", "read"),
	}
	dictionaryExports := map[string]string{
		"itemtypes": "硬件类型", "filetypes": "文件类型", "statustypes": "状态类型",
		"dpttypes": "部门", "contracttypes": "合同类型", "tags": "标记",
	}
	for dictionary, entity := range dictionaryExports {
		detail := "所有" + entity + "已导出"
		if dictionary == "contracttypes" {
			detail = "所有合同类型及合同子类型已导出"
		}
		if dictionary == "tags" {
			detail = "所有标记及关联的硬软件已导出"
		}
		specs["export:"+dictionary] = trackedEventSpec{
			module: service.AuditModuleCatalog,
			action: "导出" + entity, detail: detail,
			permission: dictionaryPermission(dictionary, "read"),
		}
		specs["import:"+dictionary] = trackedEventSpec{
			module:     service.AuditModuleCatalog,
			action:     "导入" + entity,
			permission: dictionaryPermission(dictionary, "manage"),
		}
	}
	specs["print:labels"] = trackedEventSpec{
		module: service.AuditModuleLabels,
		action: "打印标签", permission: "labels.preview",
	}
	specs["export:reports"] = trackedEventSpec{
		module:     service.AuditModuleReports,
		action:     "导出报表",
		detail:     "所有报表数据已导出",
		permission: "reports.manage",
	}
	return specs
}

// handleTrackAuditEvent 接收前端导出/导入/打印操作的审计上报
// @Summary 上报前端审计事件
	// @Description 前端执行数据导出（资产、字典、统计报表）、硬件维护日志导出、资料字典导入或标签打印后上报审计事件；事件类型白名单校验并按所属资源校验权限
// @Tags 审计日志
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body trackedEventRequest true "审计事件载荷"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /api/history/events [post]
func (a *Router) handleTrackAuditEvent(w http.ResponseWriter, r *http.Request) {
	operator, _ := common.CurrentUser(r.Context())
	var body trackedEventRequest
	if !common.DecodeJSON(w, r, &body) {
		return
	}
	spec, ok := trackedEventSpecs()[strings.TrimSpace(body.Type)]
	if !ok {
		common.WriteError(w, http.StatusBadRequest, "不支持的事件类型")
		return
	}
	if !a.hasPermission(r, spec.permission) {
		common.WriteError(w, http.StatusForbidden, "permission denied")
		return
	}
	result := service.AuditResultSuccess
	if strings.TrimSpace(body.Result) == service.AuditResultFailure {
		result = service.AuditResultFailure
	}
	event := service.AuditEvent{Module: spec.module, Action: spec.action, Result: result}
	switch {
	case body.Type == "export:items-maintenance":
		itemID, err := strconv.ParseInt(strings.TrimSpace(body.Target), 10, 64)
		if err != nil || itemID <= 0 {
			common.WriteError(w, http.StatusBadRequest, "硬件编号无效")
			return
		}
		itemName, err := service.LoadItemName(r.Context(), a.db, itemID)
		if err != nil {
			common.WriteError(w, http.StatusBadRequest, "硬件不存在")
			return
		}
		event.Target = trackedExportTarget(body.Count)
		event.Detail = trackedOutcomeText(itemName+" 维护日志所有数据已导出", result, itemName+" 维护日志导出失败")
	case body.Type == "export:reports":
		reportName := strings.TrimSpace(body.Target)
		if reportName == "" {
			reportName = "-"
		}
		event.Target = trackedExportTarget(body.Count)
		event.Detail = trackedOutcomeText(
			"\""+reportName+"\" 的所有报表数据已导出",
			result,
			"\""+reportName+"\" 的所有报表数据导出失败",
		)
	case strings.HasPrefix(body.Type, "export:"):
		event.Target = trackedExportTarget(body.Count)
		event.Detail = trackedOutcomeText(spec.detail, result, strings.Replace(spec.detail, "已导出", "导出失败", 1))
	case body.Type == "print:labels":
		preset := strings.TrimSpace(body.Target)
		if preset == "" {
			preset = "-"
		}
		event.Target = preset
		if result == service.AuditResultFailure {
			event.Detail = "打印标签失败"
		} else {
			event.Detail = fmt.Sprintf("使用标签预设 %s 打印 %d 张标签（支持导出 PDF）", preset, body.Count)
		}
	case strings.HasPrefix(body.Type, "import:"):
		imported := nonEmptyStrings(body.Names)
		existing := nonEmptyStrings(body.Existing)
		merged := mergeDistinctNames(imported, existing)
		if len(merged) == 0 {
			common.WriteError(w, http.StatusBadRequest, "导入名称不能为空")
			return
		}
		event.Target = strings.Join(merged, "、")
		if result == service.AuditResultFailure {
			if len(merged) == 1 {
				event.Detail = merged[0] + " 已存在"
			} else {
				event.Detail = strings.Join(merged, "、") + " 均已存在"
			}
			break
		}
		event.Detail = strings.Join(imported, "、") + " 已导入"
		if len(existing) > 0 {
			event.Detail += "，其中 " + strings.Join(existing, "、") + " 已存在"
		}
	default:
		common.WriteError(w, http.StatusBadRequest, "不支持的事件类型")
		return
	}
	_ = a.auditService().RecordEvent(r.Context(), operator.Username, common.ClientIP(r), event)
	common.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// trackedExportTarget 导出事件的目标列显示导出行数，行数缺失时保持占位符
func trackedExportTarget(count int) string {
	if count <= 0 {
		return "-"
	}
	return fmt.Sprintf("%d 条", count)
}

// trackedOutcomeText 成功返回成功短语，失败替换为失败短语
func trackedOutcomeText(successPhrase, result, failurePhrase string) string {
	if result == service.AuditResultFailure {
		return failurePhrase
	}
	return successPhrase
}

// mergeDistinctNames 合并已导入与已存在名称并去重，保持先后顺序，作为导入事件的目标全量
func mergeDistinctNames(imported, existing []string) []string {
	seen := map[string]bool{}
	merged := make([]string, 0, len(imported)+len(existing))
	for _, group := range [][]string{imported, existing} {
		for _, name := range group {
			if !seen[name] {
				seen[name] = true
				merged = append(merged, name)
			}
		}
	}
	return merged
}

func nonEmptyStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

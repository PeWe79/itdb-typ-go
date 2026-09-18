package service

import (
	"fmt"
	"regexp"
	"strings"
)

// 审计事件结构：所有写入操作在记录原始 SQL 的同时，归类出模块、操作与目标，
// 供审计日志页面按模块筛选与展示。
type AuditEvent struct {
	Module string
	Action string
	Target string
	Detail string
	Result string
}

const (
	AuditResultSuccess = "success"
	AuditResultFailure = "failure"

	AuditModuleAuth     = "auth"
	AuditModuleAssets   = "assets"
	AuditModuleCatalog  = "catalog"
	AuditModuleSettings = "settings"
	AuditModuleBackup   = "backup"
	AuditModuleLabels   = "labels"
	AuditModuleReports  = "reports"
	AuditModuleAudit    = "audit"
)

// 审计忽略表：审计记录自身与页面浏览轨迹不进入审计
var auditIgnoredTables = map[string]bool{
	"history":  true,
	"viewhist": true,
}

type auditTableMeta struct {
	module string
	label  string
}

// 表 → 模块与实体名称
var auditTableRegistry = map[string]auditTableMeta{
	"items":                       {AuditModuleAssets, "资产"},
	"actions":                     {AuditModuleAssets, "资产操作"},
	"item2soft":                   {AuditModuleAssets, "资产软件关联"},
	"item2file":                   {AuditModuleAssets, "资产附件关联"},
	"item2inv":                    {AuditModuleAssets, "资产单据关联"},
	"itemlink":                    {AuditModuleAssets, "资产关联"},
	"software":                    {AuditModuleAssets, "软件"},
	"contracts":                   {AuditModuleAssets, "合同"},
	"contractevents":              {AuditModuleAssets, "合同事件"},
	"invoices":                    {AuditModuleAssets, "发票"},
	"files":                       {AuditModuleAssets, "附件"},
	"agents":                      {AuditModuleAssets, "厂商"},
	"locations":                   {AuditModuleAssets, "位置"},
	"locareas":                    {AuditModuleAssets, "机位区域"},
	"racks":                       {AuditModuleAssets, "机架"},
	"itemtypes":                   {AuditModuleCatalog, "资产类型"},
	"statustypes":                 {AuditModuleCatalog, "状态类型"},
	"dpttypes":                    {AuditModuleCatalog, "部门"},
	"filetypes":                   {AuditModuleCatalog, "附件类型"},
	"contracttypes":               {AuditModuleCatalog, "合同类型"},
	"contractsubtypes":            {AuditModuleCatalog, "合同子类型"},
	"tags":                        {AuditModuleCatalog, "标记"},
	"tag2item":                    {AuditModuleCatalog, "标记关联"},
	"tag2software":                {AuditModuleCatalog, "标记关联"},
	"labelpapers":                 {AuditModuleLabels, "标签预设"},
	"users":                       {AuditModuleSettings, "用户"},
	"settings_user_profiles":      {AuditModuleSettings, "用户档案"},
	"settings_roles":              {AuditModuleSettings, "用户角色"},
	"settings_role_status":        {AuditModuleSettings, "用户角色状态"},
	"settings_user_groups":        {AuditModuleSettings, "用户群组"},
	"settings_user_group_members": {AuditModuleSettings, "群组成员"},
	"settings_user_group_roles":   {AuditModuleSettings, "群组角色"},
	"settings_user_roles":         {AuditModuleSettings, "用户角色分配"},
	"settings":                    {AuditModuleSettings, "认证配置"},
	"settings_base":               {AuditModuleSettings, "基础配置"},
	"settings_email":              {AuditModuleSettings, "邮件配置"},
	"settings_auth_providers":     {AuditModuleSettings, "认证配置"},
	"system_secrets":              {AuditModuleSettings, "系统密钥"},
	"password_reset_requests":     {AuditModuleAuth, "找回密码请求"},
}

// INSERT 语句中可作为目标展示的名称列，按优先级排列
var auditNameColumns = []string{
	"username", "title", "stitle", "name", "number", "model", "sn",
	"statusdesc", "typedesc", "dptname", "fname", "areaname",
}

var (
	auditVerbRe    = regexp.MustCompile(`(?is)^\s*(INSERT\s+INTO|UPDATE|DELETE\s+FROM)\s+([a-zA-Z_][a-zA-Z0-9_]*)`)
	auditColumnsRe = regexp.MustCompile(`(?is)^\s*INSERT\s+INTO\s+\w+\s*\(([^)]*)\)`)
	auditSetPassRe = regexp.MustCompile(`(?is)^\s*UPDATE\s+users\s+SET\s+pass\s*=\s*\?\s+WHERE\s+id\s*=\s*\?`)
	auditWhereIDRe = regexp.MustCompile(`(?is)\bid\s*=\s*\?`)
)

const auditTargetEmpty = "-"

// ClassifyAuditSQL 将 SQL 写操作归类为审计事件；ok 为 false 表示该语句不进入审计
// （查询语句、审计表与浏览轨迹表）。targetFromID 用于调用方已拿到自增 ID 时覆盖目标。
func ClassifyAuditSQL(username, query string, args []interface{}) (event AuditEvent, ok bool) {
	verb, table, matched := parseAuditVerbTable(query)
	if !matched {
		return event, false
	}
	if auditIgnoredTables[table] {
		return event, false
	}

	event.Result = AuditResultSuccess
	if strings.EqualFold(strings.TrimSpace(username), "password-reset") && table == "users" {
		event.Module = AuditModuleAuth
		event.Action = "重置密码"
		event.Target = auditTargetFromWhereID(query, args)
		return event, true
	}
	if table == "users" && auditSetPassRe.MatchString(query) {
		event.Module = AuditModuleAuth
		event.Action = "修改密码"
		event.Target = auditTargetFromWhereID(query, args)
		return event, true
	}

	meta, known := auditTableRegistry[table]
	if !known {
		meta = auditTableMeta{module: AuditModuleSettings, label: table}
	}
	event.Module = meta.module
	event.Action = auditVerbLabel(verb) + meta.label
	if target := auditTargetFromInsertColumns(query, args); target != auditTargetEmpty {
		event.Target = target
	} else {
		event.Target = auditTargetFromWhereID(query, args)
	}
	return event, true
}

// ClassifyAuditLabels 供历史数据读取时回填模块与操作标签（无参数上下文，仅解析语句）
func ClassifyAuditLabels(username, query string) (module, action string) {
	event, ok := ClassifyAuditSQL(username, query, nil)
	if !ok {
		return "", ""
	}
	return event.Module, event.Action
}

func parseAuditVerbTable(query string) (verb, table string, ok bool) {
	match := auditVerbRe.FindStringSubmatch(query)
	if match == nil {
		return "", "", false
	}
	return match[1], strings.ToLower(match[2]), true
}

func auditVerbLabel(verb string) string {
	normalized := strings.ToUpper(strings.Join(strings.Fields(verb), " "))
	switch {
	case strings.HasPrefix(normalized, "INSERT"):
		return "新增"
	case strings.HasPrefix(normalized, "UPDATE"):
		return "更新"
	case strings.HasPrefix(normalized, "DELETE"):
		return "删除"
	default:
		return "操作"
	}
}

// auditTargetFromInsertColumns 从 INSERT 的列清单与参数中提取名称类目标
func auditTargetFromInsertColumns(query string, args []interface{}) string {
	if len(args) == 0 {
		return auditTargetEmpty
	}
	columnsMatch := auditColumnsRe.FindStringSubmatch(query)
	if columnsMatch == nil {
		return auditTargetEmpty
	}
	columns := strings.Split(columnsMatch[1], ",")
	for _, nameColumn := range auditNameColumns {
		for index, column := range columns {
			if strings.ToLower(strings.TrimSpace(column)) != nameColumn {
				continue
			}
			if index >= len(args) {
				return auditTargetEmpty
			}
			return truncateAuditTarget(fmt.Sprintf("%v", args[index]))
		}
	}
	return auditTargetEmpty
}

// auditTargetFromWhereID 从 WHERE 条件的 id 占位符对应的参数中提取 "#ID" 形式目标
func auditTargetFromWhereID(query string, args []interface{}) string {
	if len(args) == 0 {
		return auditTargetEmpty
	}
	location := auditWhereIDRe.FindStringIndex(query)
	if location == nil {
		return auditTargetEmpty
	}
	placeholderIndex := strings.Count(query[:location[1]], "?") - 1
	if placeholderIndex < 0 || placeholderIndex >= len(args) {
		return auditTargetEmpty
	}
	return truncateAuditTarget(fmt.Sprintf("#%v", args[placeholderIndex]))
}

func truncateAuditTarget(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return auditTargetEmpty
	}
	runes := []rune(value)
	if len(runes) > 64 {
		return string(runes[:64]) + "…"
	}
	return value
}

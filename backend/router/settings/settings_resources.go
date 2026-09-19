package settings

import (
	"net/http"
	"strings"

	"itdb-backend/router/common"
)

// Permission 角色权限目录条目，Implied 字段用于角色编辑勾选联动与服务端闭包补全
type Permission struct {
	Key                   string   `json:"key"`
	Name                  string   `json:"name"`
	Description           string   `json:"description"`
	Category              string   `json:"category"`
	ImpliedReadPermission string   `json:"impliedReadPermission,omitempty"`
	ImpliedPermissions    []string `json:"impliedPermissions,omitempty"`
}

var permissionCatalog = []Permission{
	{Key: "assets.items.read", Name: "查看硬件", Description: "查看硬件资产列表与详情", Category: "资产管理"},
	{Key: "assets.items.manage", Name: "管理硬件", Description: "创建、编辑和删除硬件资产", Category: "资产管理", ImpliedReadPermission: "assets.items.read"},
	{Key: "assets.software.read", Name: "查看软件", Description: "查看软件资产列表与详情", Category: "资产管理"},
	{Key: "assets.software.manage", Name: "管理软件", Description: "创建、编辑和删除软件资产", Category: "资产管理", ImpliedReadPermission: "assets.software.read"},
	{Key: "assets.invoices.read", Name: "查看单据", Description: "查看单据列表与详情", Category: "资产管理"},
	{Key: "assets.invoices.manage", Name: "管理单据", Description: "创建、编辑和删除单据", Category: "资产管理", ImpliedReadPermission: "assets.invoices.read"},
	{Key: "assets.agents.read", Name: "查看代理", Description: "查看供应商与厂商代理信息", Category: "资产管理"},
	{Key: "assets.agents.manage", Name: "管理代理", Description: "创建、编辑和删除代理", Category: "资产管理", ImpliedReadPermission: "assets.agents.read"},
	{Key: "assets.files.read", Name: "查看文件", Description: "查看资产文件与下载附件", Category: "资产管理"},
	{Key: "assets.files.manage", Name: "管理文件", Description: "上传、编辑和删除资产文件", Category: "资产管理", ImpliedReadPermission: "assets.files.read"},
	{Key: "assets.contracts.read", Name: "查看合同", Description: "查看合同与合同事件", Category: "资产管理"},
	{Key: "assets.contracts.manage", Name: "管理合同", Description: "创建、编辑和删除合同与合同事件", Category: "资产管理", ImpliedReadPermission: "assets.contracts.read"},
	{Key: "assets.locations.read", Name: "查看地点", Description: "查看地点、楼层平面与区域", Category: "资产管理"},
	{Key: "assets.locations.manage", Name: "管理地点", Description: "创建、编辑和删除地点与区域", Category: "资产管理", ImpliedReadPermission: "assets.locations.read"},
	{Key: "assets.racks.read", Name: "查看机架", Description: "查看机架与设备布局", Category: "资产管理"},
	{Key: "assets.racks.manage", Name: "管理机架", Description: "创建、编辑和删除机架", Category: "资产管理", ImpliedReadPermission: "assets.racks.read"},

	{Key: "dictionaries.itemtypes.read", Name: "查看硬件类型", Description: "查看硬件类型字典", Category: "资料管理"},
	{Key: "dictionaries.itemtypes.manage", Name: "管理硬件类型", Description: "创建、编辑和删除硬件类型", Category: "资料管理", ImpliedReadPermission: "dictionaries.itemtypes.read"},
	{Key: "dictionaries.contracttypes.read", Name: "查看合同类型", Description: "查看合同类型字典", Category: "资料管理"},
	{Key: "dictionaries.contracttypes.manage", Name: "管理合同类型", Description: "创建、编辑和删除合同类型与子类型", Category: "资料管理", ImpliedReadPermission: "dictionaries.contracttypes.read"},
	{Key: "dictionaries.statustypes.read", Name: "查看状态类型", Description: "查看状态类型字典", Category: "资料管理"},
	{Key: "dictionaries.statustypes.manage", Name: "管理状态类型", Description: "创建、编辑和删除状态类型", Category: "资料管理", ImpliedReadPermission: "dictionaries.statustypes.read"},
	{Key: "dictionaries.filetypes.read", Name: "查看文件类型", Description: "查看文件类型字典", Category: "资料管理"},
	{Key: "dictionaries.filetypes.manage", Name: "管理文件类型", Description: "创建、编辑和删除文件类型", Category: "资料管理", ImpliedReadPermission: "dictionaries.filetypes.read"},
	{Key: "dictionaries.dpttypes.read", Name: "查看所属部门", Description: "查看所属部门字典", Category: "资料管理"},
	{Key: "dictionaries.dpttypes.manage", Name: "管理所属部门", Description: "创建、编辑和删除所属部门", Category: "资料管理", ImpliedReadPermission: "dictionaries.dpttypes.read"},
	{Key: "dictionaries.tags.read", Name: "查看标记", Description: "查看标记字典与关联资产", Category: "资料管理"},
	{Key: "dictionaries.tags.manage", Name: "管理标记", Description: "创建、编辑和删除标记", Category: "资料管理", ImpliedReadPermission: "dictionaries.tags.read"},

	{Key: "labels.preview", Name: "预览标签", Description: "选择硬件并生成标签预览", Category: "打印标签", ImpliedReadPermission: "assets.items.read"},
	{Key: "labels.print", Name: "打印标签", Description: "打印或导出已生成的标签预览", Category: "打印标签", ImpliedReadPermission: "labels.preview"},
	{Key: "labels.manage", Name: "编辑标签属性", Description: "保存、修改和删除标签预设", Category: "打印标签", ImpliedReadPermission: "labels.preview"},

	{Key: "reports.read", Name: "查看统计报表", Description: "查看运行资产统计报表", Category: "统计报表"},
	{Key: "reports.manage", Name: "管理统计报表", Description: "导出运行资产统计报表", Category: "统计报表", ImpliedReadPermission: "reports.read"},
	{Key: "browse.read", Name: "查看资产导航", Description: "按维度逐层定位资产", Category: "资产导航"},

	{Key: "audit.read", Name: "查看审计", Description: "查看操作审计日志", Category: "审计日志"},
	{Key: "audit.manage", Name: "管理审计", Description: "导出、清空操作审计日志", Category: "审计日志", ImpliedReadPermission: "audit.read"},

	{Key: "settings.base.read", Name: "查看基础配置", Description: "查看品牌标识、登录展示、安全时效与数据备份", Category: "系统设置"},
	{Key: "settings.base.manage", Name: "管理基础配置", Description: "维护品牌标识、登录展示、安全时效与数据备份", Category: "系统设置", ImpliedReadPermission: "settings.base.read"},
	{Key: "settings.users.read", Name: "查看用户配置", Description: "查看用户、用户群组和角色", Category: "系统设置"},
	{Key: "settings.users.manage", Name: "管理用户配置", Description: "维护用户、用户群组和角色", Category: "系统设置", ImpliedReadPermission: "settings.users.read"},
	{Key: "settings.auth.read", Name: "查看认证配置", Description: "查看 AD/LDAP 与企业微信认证配置", Category: "系统设置"},
	{Key: "settings.auth.manage", Name: "管理认证配置", Description: "维护并测试 AD/LDAP 与企业微信认证配置", Category: "系统设置", ImpliedReadPermission: "settings.auth.read"},
	{Key: "settings.notifications.read", Name: "查看通知配置", Description: "查看找回密码邮件媒介", Category: "系统设置"},
	{Key: "settings.notifications.manage", Name: "管理通知配置", Description: "维护并测试找回密码邮件媒介", Category: "系统设置", ImpliedReadPermission: "settings.notifications.read"},
}

func (a *Router) handleSettingsPermissions(w http.ResponseWriter, r *http.Request) {
	common.WriteJSON(w, http.StatusOK, map[string]any{"items": permissionCatalog, "total": len(permissionCatalog)})
}

func boolInt(v bool) int64 {
	if v {
		return 1
	}
	return 0
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item), target) {
			return true
		}
	}
	return false
}

func uniqueStrings(items []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" && !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

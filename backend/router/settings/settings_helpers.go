package settings

import (
	"encoding/json"
	"strings"
)

// permissionKeys 返回权限集合：admin 为全量权限，普通账号为只读类权限
func permissionKeys(admin bool) []string {
	if admin {
		return AllPermissionKeys()
	}
	return ReadPermissionKeys()
}
func stringConfig(config map[string]any, key string) string {
	value, _ := config[key].(string)
	return strings.TrimSpace(value)
}
func mapBool(config map[string]any, key string) bool { value, _ := config[key].(bool); return value }
func mustJSON(value any) string                      { raw, _ := json.Marshal(value); return string(raw) }

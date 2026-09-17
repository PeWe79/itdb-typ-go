package settings

import (
	"database/sql"
	"encoding/json"
	"log"
	"sort"
	"strings"
)

// normalizePermissions 过滤未知权限 Key，并按 Implied 规则补全隐含权限，返回排序去重后的集合
func normalizePermissions(keys []string) []string {
	catalog := map[string]Permission{}
	for _, item := range permissionCatalog {
		catalog[item.Key] = item
	}
	expanded := map[string]bool{}
	queue := make([]string, 0, len(keys))
	for _, key := range uniqueStrings(keys) {
		if _, ok := catalog[key]; ok {
			queue = append(queue, key)
		}
	}
	for len(queue) > 0 {
		key := queue[0]
		queue = queue[1:]
		if expanded[key] {
			continue
		}
		expanded[key] = true
		item := catalog[key]
		if item.ImpliedReadPermission != "" {
			queue = append(queue, item.ImpliedReadPermission)
		}
		queue = append(queue, item.ImpliedPermissions...)
	}
	result := make([]string, 0, len(expanded))
	for key := range expanded {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

// isReadPermission 判断是否为只读类权限，只读类权限不含任何写操作能力
func isReadPermission(key string) bool {
	return strings.HasSuffix(key, ".read") || key == "labels.preview"
}

// AllPermissionKeys 返回权限目录中的全量权限 Key
func AllPermissionKeys() []string {
	keys := make([]string, 0, len(permissionCatalog))
	for _, item := range permissionCatalog {
		keys = append(keys, item.Key)
	}
	return keys
}

// ReadPermissionKeys 返回全部只读类权限 Key，用于聚合入口的最小权限校验
func ReadPermissionKeys() []string {
	keys := make([]string, 0, len(permissionCatalog))
	for _, item := range permissionCatalog {
		if isReadPermission(item.Key) {
			keys = append(keys, item.Key)
		}
	}
	return keys
}

// legacyAssetReadPermissions 旧版 assets.read 对应的新版只读权限集合
func legacyAssetReadPermissions() []string {
	return []string{
		"assets.items.read", "assets.software.read", "assets.invoices.read", "assets.agents.read",
		"assets.files.read", "assets.contracts.read", "assets.locations.read", "assets.racks.read",
		"dictionaries.itemtypes.read", "dictionaries.contracttypes.read", "dictionaries.statustypes.read",
		"dictionaries.filetypes.read", "dictionaries.dpttypes.read", "dictionaries.tags.read",
		"labels.preview", "reports.read", "browse.read", "audit.read",
	}
}

// legacyAssetManagePermissions 旧版 assets.manage 额外对应的新版管理权限集合
func legacyAssetManagePermissions() []string {
	return []string{
		"assets.items.manage", "assets.software.manage", "assets.invoices.manage", "assets.agents.manage",
		"assets.files.manage", "assets.contracts.manage", "assets.locations.manage", "assets.racks.manage",
		"dictionaries.itemtypes.manage", "dictionaries.contracttypes.manage", "dictionaries.statustypes.manage",
		"dictionaries.filetypes.manage", "dictionaries.dpttypes.manage", "dictionaries.tags.manage",
		"labels.print", "labels.manage",
	}
}

// builtinRolePermissions 内置角色的默认权限集合，返回值已按隐含规则补全
func builtinRolePermissions(key string) []string {
	switch key {
	case "admin":
		return normalizePermissions(AllPermissionKeys())
	case "operator":
		return normalizePermissions([]string{
			"assets.items.manage", "assets.software.manage", "assets.invoices.manage", "assets.agents.manage",
			"assets.files.manage", "assets.contracts.manage", "assets.locations.manage", "assets.racks.manage",
			"dictionaries.itemtypes.manage", "dictionaries.contracttypes.manage", "dictionaries.statustypes.manage",
			"dictionaries.filetypes.manage", "dictionaries.dpttypes.manage", "dictionaries.tags.manage",
			"labels.preview", "labels.print", "labels.manage",
			"reports.manage", "reports.read", "browse.read", "audit.read",
			"settings.base.read", "settings.users.read", "settings.auth.read", "settings.notifications.read",
		})
	case "viewer":
		return normalizePermissions([]string{
			"assets.items.read", "assets.software.read", "assets.invoices.read", "assets.agents.read",
			"assets.files.read", "assets.contracts.read", "assets.locations.read", "assets.racks.read",
			"dictionaries.itemtypes.read", "dictionaries.contracttypes.read", "dictionaries.statustypes.read",
			"dictionaries.filetypes.read", "dictionaries.dpttypes.read", "dictionaries.tags.read",
			"labels.preview", "reports.read", "browse.read", "audit.read",
			"settings.base.read", "settings.users.read", "settings.auth.read", "settings.notifications.read",
		})
	}
	return nil
}

// migrateLegacyRolePermissions 将存量角色中的旧版权限 Key 重写为新版细粒度权限
func migrateLegacyRolePermissions(db *sql.DB) error {
	rows, err := db.Query("SELECT id, permissions FROM settings_roles")
	if err != nil {
		return err
	}
	type roleUpdate struct {
		id  int64
		raw string
	}
	updates := make([]roleUpdate, 0)
	for rows.Next() {
		var id int64
		var raw string
		if err := rows.Scan(&id, &raw); err != nil {
			rows.Close()
			return err
		}
		var permissions []string
		if err := json.Unmarshal([]byte(raw), &permissions); err != nil || len(permissions) == 0 {
			continue
		}
		migrated := make([]string, 0, len(permissions)+8)
		changed := false
		for _, key := range permissions {
			if replacement, ok := legacyReplacements()[key]; ok {
				migrated = append(migrated, replacement...)
				changed = true
				continue
			}
			migrated = append(migrated, key)
		}
		if !changed {
			continue
		}
		updates = append(updates, roleUpdate{id: id, raw: mustJSON(normalizePermissions(migrated))})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	for _, update := range updates {
		if _, err := db.Exec("UPDATE settings_roles SET permissions=?, updated_at=strftime('%s','now') WHERE id=?", update.raw, update.id); err != nil {
			return err
		}
	}
	if len(updates) > 0 {
		log.Printf("Migrated legacy permissions for %d roles", len(updates))
	}
	return nil
}

// legacyReplacements 返回旧版权限 Key 到新版权限集合的迁移映射
func legacyReplacements() map[string][]string {
	read := legacyAssetReadPermissions()
	manage := append(legacyAssetReadPermissions(), legacyAssetManagePermissions()...)
	return map[string][]string{
		"assets.read":   read,
		"assets.manage": manage,
	}
}

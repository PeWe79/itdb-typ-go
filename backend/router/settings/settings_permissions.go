package settings

import (
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

func settingsRoleMap(id int64, key, name, description, raw string, builtin, disabled int64, createdAt, updatedAt int64) map[string]any {
	permissions := []string{}
	if err := json.Unmarshal([]byte(raw), &permissions); err != nil || permissions == nil {
		permissions = []string{}
	}
	return map[string]any{
		"id":          strconv.FormatInt(id, 10),
		"key":         key,
		"name":        name,
		"description": description,
		"permissions": normalizePermissions(permissions),
		"builtin":     builtin != 0,
		"disabled":    disabled != 0,
		"createdAt":   time.Unix(createdAt, 0).UTC().Format(time.RFC3339),
		"updatedAt":   time.Unix(updatedAt, 0).UTC().Format(time.RFC3339),
	}
}

func UserAccess(db *sql.DB, userID, userType int64) (directRoles, effectiveRoles []map[string]any, permissions []string, err error) {
	if userType == 0 {
		role := settingsRoleMap(1, "admin", "admin", "系统配置、资产管理等所有操作", mustJSON(permissionKeys(true)), 1, 0, 0, 0)
		return []map[string]any{role}, []map[string]any{role}, permissionKeys(true), nil
	}

	directRoles, err = querySettingsRoles(db, `
		SELECT r.id,r.key,r.name,r.description,r.permissions,r.builtin,
		       COALESCE(s.disabled,0),r.created_at,r.updated_at
		FROM settings_user_roles ur
		JOIN settings_roles r ON r.id=ur.role_id
		LEFT JOIN settings_role_status s ON s.role_id=r.id
		WHERE ur.user_id=? ORDER BY r.name`, userID)
	if err != nil {
		return nil, nil, nil, err
	}

	effectiveRoles, err = querySettingsRoles(db, `
		SELECT r.id,r.key,r.name,r.description,r.permissions,r.builtin,
		       COALESCE(s.disabled,0),r.created_at,r.updated_at
		FROM settings_user_roles ur
		JOIN settings_roles r ON r.id=ur.role_id
		LEFT JOIN settings_role_status s ON s.role_id=r.id
		WHERE ur.user_id=? AND COALESCE(s.disabled,0)=0
		UNION
		SELECT r.id,r.key,r.name,r.description,r.permissions,r.builtin,
		       COALESCE(s.disabled,0),r.created_at,r.updated_at
		FROM settings_user_group_members gm
		JOIN settings_user_groups g ON g.id=gm.group_id AND COALESCE(g.disabled,0)=0
		JOIN settings_user_group_roles gr ON gr.group_id=g.id
		JOIN settings_roles r ON r.id=gr.role_id
		LEFT JOIN settings_role_status s ON s.role_id=r.id
		WHERE gm.user_id=? AND COALESCE(s.disabled,0)=0
		ORDER BY name`, userID, userID)
	if err != nil {
		return nil, nil, nil, err
	}

	permissionSet := map[string]struct{}{}
	for _, role := range effectiveRoles {
		for _, permission := range role["permissions"].([]string) {
			permissionSet[permission] = struct{}{}
		}
	}
	permissions = make([]string, 0, len(permissionSet))
	for permission := range permissionSet {
		permissions = append(permissions, permission)
	}
	return directRoles, effectiveRoles, normalizePermissions(permissions), nil
}

func querySettingsRoles(db *sql.DB, query string, args ...any) ([]map[string]any, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	roles := make([]map[string]any, 0)
	for rows.Next() {
		var id, builtin, disabled, createdAt, updatedAt int64
		var key, name, description, raw string
		if err := rows.Scan(&id, &key, &name, &description, &raw, &builtin, &disabled, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		roles = append(roles, settingsRoleMap(id, key, name, description, raw, builtin, disabled, createdAt, updatedAt))
	}
	return roles, rows.Err()
}

func HasPermission(db *sql.DB, user SessionUser, permission string) bool {
	if user.UserType == 0 || strings.EqualFold(user.Username, "admin") {
		return true
	}
	_, _, permissions, err := UserAccess(db, user.ID, user.UserType)
	if err != nil {
		return false
	}
	for _, item := range permissions {
		if item == permission {
			return true
		}
	}
	return false
}

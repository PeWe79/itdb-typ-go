package settings

import (
	"context"
	"database/sql"
	"errors"
	"github.com/go-chi/chi/v5"
	"itdb-backend/internal/domain"
	"itdb-backend/internal/security"
	"itdb-backend/internal/service"
	"itdb-backend/router/common"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"
)

func replaceSettingsUserRoles(db *sql.DB, userID int64, roleKeys []string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec("DELETE FROM settings_user_roles WHERE user_id=?", userID); err != nil {
		return err
	}
	for _, key := range uniqueStrings(roleKeys) {
		var roleID, disabled int64
		if err = tx.QueryRow("SELECT r.id,COALESCE(s.disabled,0) FROM settings_roles r LEFT JOIN settings_role_status s ON s.role_id=r.id WHERE r.key=?", key).Scan(&roleID, &disabled); err != nil {
			return errors.New("选择的角色不存在")
		}
		if disabled != 0 {
			return errors.New("角色已禁用，不能分配")
		}
		if _, err = tx.Exec("INSERT OR IGNORE INTO settings_user_roles(user_id,role_id) VALUES(?,?)", userID, roleID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

type settingsUserBody struct {
	Username    string
	Email       string
	Password    string
	DisplayName string
	RoleKeys    []string
	Disabled    bool
}

func (a *Router) handleSettingsUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		rows, err := a.db.QueryContext(r.Context(), `
			SELECT u.id,u.username,u.userdesc,u.usertype,
			       COALESCE(p.email,''),COALESCE(p.disabled,0),
			       COALESCE(p.created_at,0),COALESCE(p.last_login_at,0)
			FROM users u
			LEFT JOIN settings_user_profiles p ON p.user_id=u.id
			ORDER BY u.username ASC`)
		if err != nil {
			common.WriteError(w, 500, err.Error())
			return
		}
		type settingsUserRow struct {
			id, userType, disabled, createdAt, lastLoginAt int64
			username, description, email                   string
		}
		userRows := make([]settingsUserRow, 0)
		for rows.Next() {
			var item settingsUserRow
			if err := rows.Scan(&item.id, &item.username, &item.description, &item.userType, &item.email, &item.disabled, &item.createdAt, &item.lastLoginAt); err != nil {
				rows.Close()
				common.WriteError(w, 500, err.Error())
				return
			}
			userRows = append(userRows, item)
		}
		if err := rows.Close(); err != nil {
			common.WriteError(w, 500, err.Error())
			return
		}
		if err := rows.Err(); err != nil {
			common.WriteError(w, 500, err.Error())
			return
		}
		items := []map[string]any{}
		for _, item := range userRows {
			row, err := a.buildSettingsUserItem(r.Context(), item.id, item.username, item.description, item.email, item.userType, item.disabled, item.createdAt, item.lastLoginAt)
			if err != nil {
				common.WriteError(w, 500, err.Error())
				return
			}
			items = append(items, row)
		}
		if err := rows.Err(); err != nil {
			common.WriteError(w, 500, err.Error())
			return
		}
		common.WriteJSON(w, 200, map[string]any{"items": items, "total": len(items)})
		return
	}
	actor, _ := common.CurrentUser(r.Context())
	if r.Method == http.MethodDelete {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			common.WriteError(w, 400, "invalid id")
			return
		}
		if actor.ID == id {
			common.WriteError(w, 400, "不能删除当前登录用户")
			return
		}
		var targetDisabled int64
		var deletedUsername string
		if err = a.db.QueryRowContext(r.Context(), "SELECT COALESCE(p.disabled,0), u.username FROM users u LEFT JOIN settings_user_profiles p ON p.user_id=u.id WHERE u.id=?", id).Scan(&targetDisabled, &deletedUsername); err != nil {
			common.WriteError(w, 404, "用户不存在")
			return
		}
		if targetDisabled == 0 {
			common.WriteError(w, 400, "请先禁用用户再删除")
			return
		}
		if err = a.userWorkflow.Delete(r.Context(), service.UserActor{Username: actor.Username, IP: common.ClientIP(r)}, id); err != nil {
			WriteUserWorkflowError(w, err)
			return
		}
		a.recordAuditEvent(r.Context(), actor.Username, common.ClientIP(r), service.AuditModuleSettings, "删除用户", deletedUsername, "用户 "+deletedUsername+" 已删除", service.AuditResultSuccess)
		for _, query := range []string{
			"DELETE FROM settings_user_profiles WHERE user_id=?",
			"DELETE FROM settings_user_roles WHERE user_id=?",
			"DELETE FROM settings_user_group_members WHERE user_id=?",
		} {
			if _, cleanupErr := a.db.ExecContext(r.Context(), query, id); cleanupErr != nil {
				common.WriteError(w, http.StatusInternalServerError, cleanupErr.Error())
				return
			}
		}
		common.WriteJSON(w, 200, map[string]bool{"ok": true})
		return
	}
	var body settingsUserBody
	if !common.DecodeJSON(w, r, &body) {
		return
	}
	username := strings.TrimSpace(body.Username)
	email := strings.TrimSpace(body.Email)
	displayName := strings.TrimSpace(body.DisplayName)
	if username == "" {
		common.WriteError(w, 400, "用户名不能为空")
		return
	}
	if _, err := mail.ParseAddress(email); err != nil {
		common.WriteError(w, 400, "请输入有效的邮箱地址")
		return
	}
	if displayName == "" {
		displayName = username
	}
	if len(body.Password) < 6 && (r.Method == http.MethodPost || body.Password != "") {
		common.WriteError(w, 400, "密码至少 6 个字符")
		return
	}
	if len(body.RoleKeys) == 0 {
		body.RoleKeys = []string{"viewer"}
	}
	ut := int64(1)
	if containsString(body.RoleKeys, "admin") {
		ut = 0
	}
	if r.Method == http.MethodPost {
		var existing int64
		if err := a.db.QueryRowContext(r.Context(), "SELECT COUNT(id) FROM users WHERE LOWER(username)=LOWER(?)", username).Scan(&existing); err != nil {
			common.WriteError(w, 500, err.Error())
			return
		}
		if existing > 0 {
			common.WriteError(w, 409, "用户名已存在")
			return
		}
		hash, err := security.HashPassword(body.Password)
		if err != nil {
			common.WriteError(w, 500, err.Error())
			return
		}
		res, err := a.db.ExecContext(r.Context(), "INSERT INTO users (username,userdesc,pass,usertype) VALUES (?,?,?,?)", username, displayName, hash, ut)
		if err != nil {
			WriteUserWorkflowError(w, err)
			return
		}
		id, _ := res.LastInsertId()
		now := time.Now().Unix()
		if _, err = a.db.ExecContext(r.Context(), "INSERT INTO settings_user_profiles(user_id,email,disabled,source,created_at,updated_at) VALUES(?,?,?,?,?,?)", id, email, boolInt(body.Disabled), "local", now, now); err != nil {
			_, _ = a.db.ExecContext(r.Context(), "DELETE FROM users WHERE id=?", id)
			common.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if err := replaceSettingsUserRoles(a.db, id, body.RoleKeys); err != nil {
			_, _ = a.db.ExecContext(r.Context(), "DELETE FROM settings_user_profiles WHERE user_id=?", id)
			_, _ = a.db.ExecContext(r.Context(), "DELETE FROM users WHERE id=?", id)
			common.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		a.recordAuditEvent(r.Context(), actor.Username, common.ClientIP(r), service.AuditModuleSettings, "创建用户", username, "本地用户 "+username+" 已创建", service.AuditResultSuccess)
		item, err := a.settingsUserResponse(r.Context(), id)
		if err != nil {
			common.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		common.WriteJSON(w, 201, item)
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		common.WriteError(w, 400, "invalid id")
		return
	}
	if r.Method == http.MethodPut {
		if body.Disabled && actor.ID == id {
			common.WriteError(w, 400, "不能禁用当前登录用户")
			return
		}
		var oldUsername, oldDisplayName string
		if err := a.db.QueryRowContext(r.Context(), "SELECT username,COALESCE(userdesc,'') FROM users WHERE id=?", id).Scan(&oldUsername, &oldDisplayName); err != nil {
			common.WriteError(w, 404, "用户不存在")
			return
		}
		var oldEmail string
		_ = a.db.QueryRowContext(r.Context(), "SELECT COALESCE(email,'') FROM settings_user_profiles WHERE user_id=?", id).Scan(&oldEmail)
		oldRoles := queryRoleKeys(a.db, "SELECT r.key FROM settings_user_roles ur JOIN settings_roles r ON r.id=ur.role_id WHERE ur.user_id=?", id)
		if strings.EqualFold(oldUsername, "admin") && (body.Disabled || !strings.EqualFold(username, oldUsername)) {
			common.WriteError(w, 400, "默认管理员不能禁用、删除或改名")
			return
		}
		err = a.userWorkflow.Update(r.Context(), service.UserActor{Username: actor.Username, IP: common.ClientIP(r)}, id, domain.UserPayload{Username: username, UserDesc: displayName, Password: body.Password, UserType: ut})
		if err != nil {
			WriteUserWorkflowError(w, err)
			return
		}
		if _, err = a.db.ExecContext(r.Context(), "INSERT INTO settings_user_profiles(user_id,email,disabled,source,created_at,updated_at) VALUES(?,?,?,?,?,?) ON CONFLICT(user_id) DO UPDATE SET email=excluded.email,disabled=excluded.disabled,updated_at=excluded.updated_at", id, email, boolInt(body.Disabled), "local", time.Now().Unix(), time.Now().Unix()); err != nil {
			common.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if err = replaceSettingsUserRoles(a.db, id, body.RoleKeys); err != nil {
			common.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if detail := userChangeDetail(username, oldEmail, oldDisplayName, oldRoles, email, displayName, body.Password, body.RoleKeys); detail != "" {
			a.recordAuditEvent(r.Context(), actor.Username, common.ClientIP(r), service.AuditModuleSettings, "更新用户", username, detail, service.AuditResultSuccess)
		}
		item, err := a.settingsUserResponse(r.Context(), id)
		if err != nil {
			common.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		common.WriteJSON(w, 200, item)
		return
	}
	common.WriteError(w, 405, "method not allowed")
}

// userChangeDetail 生成“更新用户”审计详情：变更项按 邮箱、显示名称、密码、角色 顺序合并，
// 如“用户 alice 已更新：邮箱、显示名称、密码、角色已修改”；无任何修改时返回空串（不记审计）
func userChangeDetail(username, oldEmail, oldDisplayName string, oldRoles []string, newEmail, newDisplayName, newPassword string, newRoles []string) string {
	changes := make([]string, 0, 4)
	if strings.TrimSpace(newEmail) != strings.TrimSpace(oldEmail) {
		changes = append(changes, "邮箱")
	}
	if strings.TrimSpace(newDisplayName) != strings.TrimSpace(oldDisplayName) {
		changes = append(changes, "显示名称")
	}
	if strings.TrimSpace(newPassword) != "" {
		changes = append(changes, "密码")
	}
	if !equalStringSets(oldRoles, uniqueStrings(newRoles)) {
		changes = append(changes, "角色")
	}
	if len(changes) == 0 {
		return ""
	}
	return "用户 " + username + " 已更新：" + mergeChangeItems(changes)
}

func (a *Router) handleSettingsUserDisabled(w http.ResponseWriter, r *http.Request) {
	actor, _ := common.CurrentUser(r.Context())
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var body struct{ Disabled bool }
	if !common.DecodeJSON(w, r, &body) {
		return
	}
	var username string
	var userType int64
	if err := a.db.QueryRowContext(r.Context(), "SELECT username,usertype FROM users WHERE id=?", id).Scan(&username, &userType); err != nil {
		common.WriteError(w, http.StatusNotFound, "user not found")
		return
	}
	if strings.EqualFold(username, "admin") {
		common.WriteError(w, http.StatusBadRequest, "默认管理员不可禁用")
		return
	}
	if actor.ID == id {
		common.WriteError(w, http.StatusBadRequest, "不能禁用当前登录用户")
		return
	}
	now := time.Now().Unix()
	_, err = a.db.ExecContext(r.Context(), "INSERT INTO settings_user_profiles(user_id,disabled,source,created_at,updated_at) VALUES(?,?,'local',?,?) ON CONFLICT(user_id) DO UPDATE SET disabled=excluded.disabled,updated_at=excluded.updated_at", id, boolInt(body.Disabled), now, now)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	a.recordSettingsStatusEvent(r.Context(), actor.Username, common.ClientIP(r), "用户", username, body.Disabled)
	item, err := a.settingsUserResponse(r.Context(), id)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.WriteJSON(w, http.StatusOK, item)
}

func (a *Router) settingsUserResponse(ctx context.Context, id int64) (map[string]any, error) {
	var username, description, email string
	var userType, disabled, createdAt, lastLoginAt int64
	err := a.db.QueryRowContext(ctx, `
		SELECT u.username,u.userdesc,u.usertype,
		       COALESCE(p.email,''),COALESCE(p.disabled,0),
		       COALESCE(p.created_at,0),COALESCE(p.last_login_at,0)
		FROM users u
		LEFT JOIN settings_user_profiles p ON p.user_id=u.id
		WHERE u.id=?`, id).Scan(&username, &description, &userType, &email, &disabled, &createdAt, &lastLoginAt)
	if err != nil {
		return nil, err
	}
	return a.buildSettingsUserItem(ctx, id, username, description, email, userType, disabled, createdAt, lastLoginAt)
}

func (a *Router) buildSettingsUserItem(ctx context.Context, id int64, username, description, email string, userType, disabled, createdAt, lastLoginAt int64) (map[string]any, error) {
	directRoles, effectiveRoles, permissions, err := UserAccess(a.db, id, userType)
	if err != nil {
		return nil, err
	}
	roleKey := "viewer"
	if userType == 0 {
		roleKey = "admin"
	} else if len(effectiveRoles) > 0 {
		roleKey, _ = effectiveRoles[0]["key"].(string)
	}
	return map[string]any{
		"id": strconv.FormatInt(id, 10), "username": username, "displayName": description,
		"email": email, "role": roleKey, "source": "local", "roles": effectiveRoles,
		"directRoles": directRoles, "effectiveUserRoles": effectiveRoles,
		"disabled": disabled != 0, "permissions": permissions,
		"effectiveUserPermissions": permissions,
		"createdAt":                formatUnixTime(createdAt),
		"lastLoginAt":              formatUnixTime(lastLoginAt),
	}, nil
}

func formatUnixTime(value int64) string {
	if value <= 0 {
		return ""
	}
	return time.Unix(value, 0).UTC().Format(time.RFC3339)
}

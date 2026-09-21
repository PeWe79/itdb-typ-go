package settings

import (
	"database/sql"
	"fmt"
	"itdb-backend/router/common"
	"log"
	"strings"
)

func EnsureResourceSchema(db *sql.DB) error {
	statements := []string{
		"CREATE TABLE IF NOT EXISTS settings_user_profiles (user_id INTEGER PRIMARY KEY, email TEXT NOT NULL DEFAULT '', disabled INTEGER NOT NULL DEFAULT 0, source TEXT NOT NULL DEFAULT 'local', created_at INTEGER NOT NULL DEFAULT 0, updated_at INTEGER NOT NULL DEFAULT 0, last_login_at INTEGER NOT NULL DEFAULT 0)",
		"CREATE TABLE IF NOT EXISTS settings_roles (id INTEGER PRIMARY KEY AUTOINCREMENT, key TEXT NOT NULL UNIQUE, name TEXT NOT NULL, description TEXT NOT NULL DEFAULT '', permissions TEXT NOT NULL DEFAULT '[]', builtin INTEGER NOT NULL DEFAULT 0, created_at INTEGER NOT NULL DEFAULT 0, updated_at INTEGER NOT NULL DEFAULT 0)",
		"CREATE TABLE IF NOT EXISTS settings_user_groups (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE, description TEXT NOT NULL DEFAULT '', disabled INTEGER NOT NULL DEFAULT 0, created_at INTEGER NOT NULL DEFAULT 0, updated_at INTEGER NOT NULL DEFAULT 0)",
		"CREATE TABLE IF NOT EXISTS settings_user_group_members (group_id INTEGER NOT NULL, user_id INTEGER NOT NULL, PRIMARY KEY (group_id,user_id))",
		"CREATE TABLE IF NOT EXISTS settings_user_group_roles (group_id INTEGER NOT NULL, role_id INTEGER NOT NULL, PRIMARY KEY (group_id,role_id))",
		"CREATE TABLE IF NOT EXISTS settings_user_roles (user_id INTEGER NOT NULL, role_id INTEGER NOT NULL, PRIMARY KEY (user_id,role_id))",
		"CREATE TABLE IF NOT EXISTS settings_role_status (role_id INTEGER PRIMARY KEY, disabled INTEGER NOT NULL DEFAULT 0)",
		"CREATE TABLE IF NOT EXISTS password_reset_captchas (token_hash TEXT PRIMARY KEY, question TEXT NOT NULL, answer_hash TEXT NOT NULL, expires_at INTEGER NOT NULL, created_at INTEGER NOT NULL)",
		"CREATE TABLE IF NOT EXISTS password_reset_requests (token_hash TEXT PRIMARY KEY, user_id INTEGER NOT NULL, code_hash TEXT NOT NULL DEFAULT '', email TEXT NOT NULL DEFAULT '', expires_at INTEGER NOT NULL, used INTEGER NOT NULL DEFAULT 0, last_sent_at INTEGER NOT NULL DEFAULT 0, created_at INTEGER NOT NULL)",
		"CREATE TABLE IF NOT EXISTS settings_auth_providers (id TEXT PRIMARY KEY, name TEXT NOT NULL, enabled INTEGER NOT NULL DEFAULT 0, config TEXT NOT NULL DEFAULT '{}', updated_at INTEGER NOT NULL DEFAULT 0)",
		"CREATE TABLE IF NOT EXISTS settings_user_wecom (user_id INTEGER PRIMARY KEY, wecom_userid TEXT NOT NULL UNIQUE, bound_at INTEGER NOT NULL DEFAULT 0)",
		"CREATE TABLE IF NOT EXISTS settings_email (id INTEGER PRIMARY KEY CHECK (id = 1), name TEXT NOT NULL DEFAULT '邮件通知', enabled INTEGER NOT NULL DEFAULT 0, password_reset_enabled INTEGER NOT NULL DEFAULT 0, config TEXT NOT NULL DEFAULT '{}', updated_at INTEGER NOT NULL DEFAULT 0)",
		"CREATE TABLE IF NOT EXISTS settings_base (id INTEGER PRIMARY KEY CHECK (id = 1), config TEXT NOT NULL DEFAULT '{}', updated_at INTEGER NOT NULL DEFAULT 0)",
		"CREATE TABLE IF NOT EXISTS password_reset_send_log (id INTEGER PRIMARY KEY AUTOINCREMENT, email TEXT NOT NULL, sent_at INTEGER NOT NULL)",
		"CREATE INDEX IF NOT EXISTS idx_password_reset_send_log_email ON password_reset_send_log(email,sent_at)",
		"CREATE TABLE IF NOT EXISTS login_failure_log (id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT NOT NULL, failed_at INTEGER NOT NULL)",
		"CREATE INDEX IF NOT EXISTS idx_login_failure_log_username ON login_failure_log(username,failed_at)",
		"INSERT OR IGNORE INTO settings_auth_providers(id,name,enabled,config,updated_at) VALUES ('ldap','AD/LDAP',0,'{}',strftime('%s','now'))",
		"INSERT OR IGNORE INTO settings_auth_providers(id,name,enabled,config,updated_at) VALUES ('wecom','企业微信',0,'{}',strftime('%s','now'))",
		"INSERT OR IGNORE INTO settings_email(id,name,enabled,password_reset_enabled,config,updated_at) VALUES (1,'邮件通知',0,0,'{}',strftime('%s','now'))",
		"INSERT OR IGNORE INTO settings_base(id,config,updated_at) VALUES (1,'{}',strftime('%s','now'))",
		"CREATE TABLE IF NOT EXISTS system_secrets (id INTEGER PRIMARY KEY CHECK (id = 1), jwt_secret TEXT NOT NULL, updated_at INTEGER NOT NULL DEFAULT 0)",
	}
	statements = append(statements, builtinRoleSeedStatements()...)
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return err
		}
	}
	if err := migrateLegacyRolePermissions(db); err != nil {
		return err
	}
	return ensureSettingsUserProfilesBackfill(db)
}

// builtinRoleSeedStatements 内置角色种子语句，权限集合随权限目录自动刷新
func builtinRoleSeedStatements() []string {
	roles := []struct{ key, name, description string }{
		{key: "admin", name: "admin", description: "系统配置、资产管理等所有操作"},
		{key: "operator", name: "operator", description: "资产资源和日常操作，不能修改系统配置"},
		{key: "viewer", name: "viewer", description: "只读查看资产资源和系统配置"},
	}
	statements := make([]string, 0, len(roles))
	for _, role := range roles {
		statements = append(statements, "INSERT INTO settings_roles(key,name,description,permissions,builtin,created_at,updated_at) VALUES ('"+role.key+"','"+role.name+"','"+role.description+"','"+mustJSON(builtinRolePermissions(role.key))+"',1,strftime('%s','now'),strftime('%s','now')) ON CONFLICT(key) DO UPDATE SET name=excluded.name, description=excluded.description, permissions=excluded.permissions, builtin=1, updated_at=excluded.updated_at")
	}
	return statements
}

// ensureSettingsUserProfilesBackfill 为缺失档案的用户补建 profile 行，
// 使存量账号在用户配置中也能展示创建时间；同时把默认 admin 的显示名归一为 admin。
func ensureSettingsUserProfilesBackfill(db *sql.DB) error {
	usersExists, err := common.SQLiteTableExists(db, "users")
	if err != nil {
		return err
	}
	if !usersExists {
		return nil
	}
	if _, err := db.Exec("UPDATE users SET userdesc='admin' WHERE username='admin' AND userdesc='administrator'"); err != nil {
		return err
	}
	hasDisabled, err := common.SQLiteColumnExists(db, "settings_user_profiles", "disabled")
	if err != nil {
		return err
	}
	if !hasDisabled {
		return nil
	}
	_, err = db.Exec("INSERT INTO settings_user_profiles(user_id,email,disabled,source,created_at,updated_at) SELECT u.id,'',0,'local',strftime('%s','now'),strftime('%s','now') FROM users u WHERE NOT EXISTS (SELECT 1 FROM settings_user_profiles p WHERE p.user_id=u.id)")
	return err
}

// EnsureUserProfileLoginSchema 为存量库的 settings_user_profiles 补充 last_login_at 列。
func EnsureUserProfileLoginSchema(db *sql.DB, dbPath string) error {
	tableExists, err := common.SQLiteTableExists(db, "settings_user_profiles")
	if err != nil {
		return err
	}
	if !tableExists {
		return nil
	}
	hasColumn, err := common.SQLiteColumnExists(db, "settings_user_profiles", "last_login_at")
	if err != nil {
		return err
	}
	if hasColumn {
		return nil
	}
	trimmedDBPath := strings.TrimSpace(dbPath)
	if trimmedDBPath != "" && !strings.EqualFold(trimmedDBPath, ":memory:") {
		backupPath, err := common.BackupDatabaseBeforeAlter(db, dbPath, "user-profile-last-login-at")
		if err != nil {
			return err
		}
		log.Printf("Database backup completed: %s", backupPath)
	}
	if _, err := db.Exec(`ALTER TABLE settings_user_profiles ADD COLUMN last_login_at INTEGER NOT NULL DEFAULT 0`); err != nil {
		return fmt.Errorf("add user profile last login column failed: %w", err)
	}
	return nil
}

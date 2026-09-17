package common

import (
	"database/sql"
	"errors"
	"fmt"
	"itdb-backend/internal/security"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "itdb-backend/docs"
	"itdb-backend/internal/common/statustypes"

	_ "modernc.org/sqlite"
)

func SetupSQLite(db *sql.DB) error {
	pragmas := []string{
		"PRAGMA busy_timeout = 5000;",
		"PRAGMA journal_mode = WAL;",
		"PRAGMA synchronous = NORMAL;",
		"PRAGMA case_sensitive_like = 0;",
		"PRAGMA encoding = \"UTF-8\";",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return err
		}
	}
	return nil
}

func EnsureStatusTypeColorSchema(db *sql.DB, dbPath string) error {
	hasColor, err := SQLiteColumnExists(db, "statustypes", "color")
	if err != nil {
		return err
	}
	if !hasColor {
		backupPath, err := BackupDatabaseBeforeAlter(db, dbPath, "statustypes-color")
		if err != nil {
			return err
		}
		log.Printf("Database backup completed: %s", backupPath)

		if _, err := db.Exec(`ALTER TABLE statustypes ADD COLUMN color TEXT`); err != nil {
			return err
		}
	}

	for desc, color := range statustypes.FixedStatusTypeColors() {
		if _, err := db.Exec(`UPDATE statustypes SET color = ? WHERE TRIM(statusdesc) = ?`, color, desc); err != nil {
			return err
		}
	}
	return nil
}

// EnsureLabelPapersSchema 为存量 labelpapers 表补充 labelskip 列（顶部跳过标签数）
func EnsureLabelPapersSchema(db *sql.DB, dbPath string) error {
	tableExists, err := SQLiteTableExists(db, "labelpapers")
	if err != nil {
		return err
	}
	if !tableExists {
		return nil
	}
	hasSkip, err := SQLiteColumnExists(db, "labelpapers", "labelskip")
	if err != nil {
		return err
	}
	if hasSkip {
		return nil
	}
	trimmedDBPath := strings.TrimSpace(dbPath)
	if trimmedDBPath != "" && !strings.EqualFold(trimmedDBPath, ":memory:") {
		backupPath, err := BackupDatabaseBeforeAlter(db, dbPath, "labelpapers-labelskip")
		if err != nil {
			return err
		}
		log.Printf("Database backup completed: %s", backupPath)
	}
	if _, err := db.Exec(`ALTER TABLE labelpapers ADD COLUMN labelskip integer DEFAULT 0`); err != nil {
		return err
	}
	log.Printf("Label papers labelskip column added with default 0")
	return nil
}

// EnsureActionsSchema 为存量 actions 表移除 entrydate 列（与 actiondate 恒等，已停用）
func EnsureActionsSchema(db *sql.DB, dbPath string) error {
	tableExists, err := SQLiteTableExists(db, "actions")
	if err != nil {
		return err
	}
	if !tableExists {
		return nil
	}
	hasEntryDate, err := SQLiteColumnExists(db, "actions", "entrydate")
	if err != nil {
		return err
	}
	if !hasEntryDate {
		return nil
	}
	trimmedDBPath := strings.TrimSpace(dbPath)
	if trimmedDBPath != "" && !strings.EqualFold(trimmedDBPath, ":memory:") {
		backupPath, err := BackupDatabaseBeforeAlter(db, dbPath, "actions-drop-entrydate")
		if err != nil {
			return err
		}
		log.Printf("Database backup completed: %s", backupPath)
	}
	if _, err := db.Exec(`ALTER TABLE actions DROP COLUMN entrydate`); err != nil {
		return err
	}
	log.Printf("Actions entrydate column dropped")
	return nil
}

func EnsureSettingsSchema(db *sql.DB, dbPath string) error {
	legacyColumns := []string{"companytitle", "dateformat", "currency", "lang", "version", "timezone", "dbversion"}
	currentColumns := []string{"useldap", "ldap_server", "ldap_dn", "ldap_bind_dn", "ldap_bind_password", "ldap_getusers", "ldap_getusers_filter"}
	tableExists, err := SQLiteTableExists(db, "settings")
	if err != nil {
		return err
	}

	hasColumns := map[string]bool{}
	needsMigration := false
	if tableExists {
		for _, columnName := range append(append([]string{}, legacyColumns...), currentColumns...) {
			hasColumn, err := SQLiteColumnExists(db, "settings", columnName)
			if err != nil {
				return err
			}
			hasColumns[columnName] = hasColumn
		}
		for _, columnName := range legacyColumns {
			if hasColumns[columnName] {
				needsMigration = true
				break
			}
		}
		if !needsMigration {
			for _, columnName := range currentColumns {
				if !hasColumns[columnName] {
					needsMigration = true
					break
				}
			}
		}
	}

	if tableExists && needsMigration {
		trimmedDBPath := strings.TrimSpace(dbPath)
		if trimmedDBPath != "" && !strings.EqualFold(trimmedDBPath, ":memory:") {
			backupPath, err := BackupDatabaseBeforeAlter(db, dbPath, "settings-schema")
			if err != nil {
				return err
			}
			log.Printf("Database backup completed: %s", backupPath)
		}

		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		if _, err := tx.Exec(`CREATE TABLE settings_new (useldap integer default 0, ldap_server, ldap_dn, ldap_bind_dn, ldap_bind_password, ldap_getusers, ldap_getusers_filter)`); err != nil {
			return err
		}

		useLDAPExpr := "0"
		if hasColumns["useldap"] {
			useLDAPExpr = "COALESCE(useldap, 0)"
		}
		ldapServerExpr := "''"
		if hasColumns["ldap_server"] {
			ldapServerExpr = "COALESCE(ldap_server, '')"
		}
		ldapDNExpr := "''"
		if hasColumns["ldap_dn"] {
			ldapDNExpr = "COALESCE(ldap_dn, '')"
		}
		ldapBindDNExpr := "''"
		if hasColumns["ldap_bind_dn"] {
			ldapBindDNExpr = "COALESCE(ldap_bind_dn, '')"
		}
		ldapBindPasswordExpr := "''"
		if hasColumns["ldap_bind_password"] {
			ldapBindPasswordExpr = "COALESCE(ldap_bind_password, '')"
		}
		ldapGetUsersExpr := "''"
		if hasColumns["ldap_getusers"] {
			ldapGetUsersExpr = "COALESCE(ldap_getusers, '')"
		}
		ldapGetUsersFilterExpr := "''"
		if hasColumns["ldap_getusers_filter"] {
			ldapGetUsersFilterExpr = "COALESCE(ldap_getusers_filter, '')"
		}

		copyQuery := fmt.Sprintf(`INSERT INTO settings_new (useldap, ldap_server, ldap_dn, ldap_bind_dn, ldap_bind_password, ldap_getusers, ldap_getusers_filter)
			SELECT %s, %s, %s, %s, %s, %s, %s
			FROM settings
			LIMIT 1`, useLDAPExpr, ldapServerExpr, ldapDNExpr, ldapBindDNExpr, ldapBindPasswordExpr, ldapGetUsersExpr, ldapGetUsersFilterExpr)
		if _, err := tx.Exec(copyQuery); err != nil {
			return err
		}
		if _, err := tx.Exec(`DROP TABLE settings`); err != nil {
			return err
		}
		if _, err := tx.Exec(`ALTER TABLE settings_new RENAME TO settings`); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}

	return EnsureSettingsRow(db)
}

func EnsureSettingsRow(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS settings (useldap integer default 0, ldap_server, ldap_dn, ldap_bind_dn, ldap_bind_password, ldap_getusers, ldap_getusers_filter)`); err != nil {
		return err
	}

	var count int64
	if err := db.QueryRow(`SELECT COUNT(*) FROM settings`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	_, err := db.Exec(`INSERT INTO settings (useldap, ldap_server, ldap_dn, ldap_bind_dn, ldap_bind_password, ldap_getusers, ldap_getusers_filter) VALUES (0, '', '', '', '', '', '')`)
	return err
}

// EnsureHistoryAuditSchema 为 history 表补充审计日志结构化列（老库增量升级）
func EnsureHistoryAuditSchema(db *sql.DB) error {
	columns := map[string]string{
		"module": `TEXT DEFAULT ''`,
		"action": `TEXT DEFAULT ''`,
		"target": `TEXT DEFAULT ''`,
		"detail": `TEXT DEFAULT ''`,
		"result": `TEXT DEFAULT 'success'`,
	}
	tableExists, err := SQLiteTableExists(db, "history")
	if err != nil {
		return err
	}
	if !tableExists {
		return nil
	}
	for name, definition := range columns {
		exists, err := SQLiteColumnExists(db, "history", name)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		if _, err := db.Exec(fmt.Sprintf(`ALTER TABLE history ADD COLUMN %s %s`, name, definition)); err != nil {
			return fmt.Errorf("add history audit column %s failed: %w", name, err)
		}
	}
	return nil
}

// EnsureTagLinkOrphanCleanup 清理删除硬件/软件后残留的孤儿标记关联行：
// 历史版本删除硬件时将 itemid 置空、删除软件时未清理关联，导致标记关联计数虚高。
func EnsureTagLinkOrphanCleanup(db *sql.DB) error {
	for table, entityTable := range map[string]string{"tag2item": "items", "tag2software": "software"} {
		tableExists, err := SQLiteTableExists(db, table)
		if err != nil {
			return err
		}
		entityExists, err := SQLiteTableExists(db, entityTable)
		if err != nil {
			return err
		}
		if !tableExists || !entityExists {
			continue
		}
		column := "itemid"
		if table == "tag2software" {
			column = "softwareid"
		}
		if _, err := db.Exec(fmt.Sprintf("DELETE FROM %s WHERE %s IS NULL OR %s NOT IN (SELECT id FROM %s)", table, column, column, entityTable)); err != nil {
			return fmt.Errorf("cleanup orphan tag links on %s failed: %w", table, err)
		}
	}
	return nil
}

func EnsurePasswordResetSchema(db *sql.DB, dbPath string) error {
	tableExists, err := SQLiteTableExists(db, "password_reset_requests")
	if err != nil {
		return err
	}
	if err != nil {
		return err
	}
	if !tableExists {
		return nil
	}

	hasLastSentAt, err := SQLiteColumnExists(db, "password_reset_requests", "last_sent_at")
	if err != nil {
		return err
	}
	if hasLastSentAt {
		return nil
	}

	trimmedDBPath := strings.TrimSpace(dbPath)
	if trimmedDBPath != "" && !strings.EqualFold(trimmedDBPath, ":memory:") {
		backupPath, err := BackupDatabaseBeforeAlter(db, dbPath, "password-reset-last-sent-at")
		if err != nil {
			return err
		}
		log.Printf("Database backup completed: %s", backupPath)
	}

	if _, err := db.Exec(`ALTER TABLE password_reset_requests ADD COLUMN last_sent_at INTEGER NOT NULL DEFAULT 0`); err != nil {
		return fmt.Errorf("add password reset send timestamp column failed: %w", err)
	}
	return nil
}
func EnsureSettingsSecretsEncrypted(db *sql.DB, cipherKey string) error {
	var (
		rowID       int64
		bindPassRaw sql.NullString
	)
	err := db.QueryRow(`SELECT rowid, ldap_bind_password FROM settings LIMIT 1`).Scan(&rowID, &bindPassRaw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	encrypted, changed, err := security.EncryptSettingsSecretIfNeeded(bindPassRaw.String, cipherKey)
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}

	_, err = db.Exec(`UPDATE settings SET ldap_bind_password = ? WHERE rowid = ?`, encrypted, rowID)
	return err
}

func SQLiteTableExists(db *sql.DB, tableName string) (bool, error) {
	var name string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ? LIMIT 1`, tableName).Scan(&name)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return false, err
}

func SQLiteColumnExists(db *sql.DB, tableName, columnName string) (bool, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid      int
			name     string
			colType  string
			notNull  int
			defaultV sql.NullString
			pk       int
		)
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultV, &pk); err != nil {
			return false, err
		}
		if strings.EqualFold(name, columnName) {
			return true, nil
		}
	}
	return false, rows.Err()
}

func BackupDatabaseBeforeAlter(db *sql.DB, dbPath, tag string) (string, error) {
	if strings.TrimSpace(dbPath) == "" || strings.EqualFold(strings.TrimSpace(dbPath), ":memory:") {
		return "", errors.New("in-memory database cannot create file backup")
	}

	absPath, err := filepath.Abs(dbPath)
	if err != nil {
		return "", err
	}
	baseName := strings.TrimSuffix(filepath.Base(absPath), filepath.Ext(absPath))
	if baseName == "" {
		baseName = "itdb"
	}
	backupDir := filepath.Join(filepath.Dir(absPath), "backups")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return "", err
	}

	stamp := time.Now().Format("20060102-150405")
	backupPath := filepath.Join(backupDir, fmt.Sprintf("%s-before-%s-%s.db", baseName, tag, stamp))
	escapedBackupPath := strings.ReplaceAll(backupPath, "'", "''")
	backupSQL := fmt.Sprintf("VACUUM INTO '%s'", escapedBackupPath)
	if _, err := db.Exec(backupSQL); err != nil {
		return "", err
	}
	return backupPath, nil
}

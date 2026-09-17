// 旧版数据库导入适配：新建当前结构空库后仅拷贝资产管理、资料管理与用户数据，系统配置、审计历史、标签预设等保持当前项目默认。
package system

import (
	"database/sql"
	"errors"
	"fmt"
	"itdb-backend/internal/security"
	"itdb-backend/internal/service"
	"itdb-backend/router/common"
	"itdb-backend/router/settings"
	"log"
	"os"
	"strings"
	"time"
)

// legacyImportTables 旧库迁移的数据范围：资料管理全部字典、资产管理全部菜单数据与用户表
var legacyImportTables = []string{
	"itemtypes", "filetypes", "statustypes", "dpttypes", "contracttypes", "contractsubtypes", "tags", "tag2Item", "tag2software",
	"items", "item2soft", "item2inv", "item2file", "itemlink", "actions",
	"software", "soft2inv", "software2file",
	"invoices", "invoice2file",
	"agents",
	"files",
	"contracts", "contract2item", "contract2inv", "contract2soft", "contract2file", "contractevents",
	"locations", "locareas", "racks",
	"users",
}

// legacySeededTables 当前项目内置默认数据的字典表：仅当旧库存在数据时才整表替换种子行
var legacySeededTables = map[string]bool{
	"itemtypes": true, "filetypes": true, "statustypes": true, "contracttypes": true,
}

// syncRuntimeJWTSecret 将当前运行时签名密钥写入导入后的库，保证服务重启后已签发令牌仍有效
func syncRuntimeJWTSecret(db *sql.DB, secret string) {
	if strings.TrimSpace(secret) == "" {
		return
	}
	if _, err := db.Exec(`INSERT INTO system_secrets (id, jwt_secret, updated_at) VALUES (1, ?, ?)
		ON CONFLICT(id) DO UPDATE SET jwt_secret = excluded.jwt_secret, updated_at = excluded.updated_at`,
		secret, time.Now().Unix()); err != nil {
		log.Printf("Sync runtime jwt secret into imported database failed: %v", err)
	}
}

// isCurrentProjectDatabase 通过 settings_user_profiles 表判断是否为当前项目导出的数据库
func isCurrentProjectDatabase(db *sql.DB) (bool, error) {
	return common.SQLiteTableExists(db, "settings_user_profiles")
}

// prepareImportDatabaseFile 判断上传库类型：当前项目库直接安装，旧版库转换出新库文件后安装
func prepareImportDatabaseFile(sourcePath string) (string, error) {
	probe, err := sql.Open("sqlite", sourcePath)
	if err != nil {
		return "", errors.New("数据库文件解析失败")
	}
	current, err := isCurrentProjectDatabase(probe)
	closeErr := probe.Close()
	if err != nil || closeErr != nil {
		return "", errors.New("数据库文件解析失败")
	}
	if current {
		return sourcePath, nil
	}
	log.Printf("Legacy database detected, starting migration: %s", sourcePath)
	migratedPath, err := migrateLegacyDatabaseFile(sourcePath)
	if err != nil {
		log.Printf("Legacy database migration failed: %v", err)
		return "", errors.New("旧版数据库转换失败：" + err.Error())
	}
	log.Printf("Legacy database migration completed: %s", migratedPath)
	return migratedPath, nil
}

// migrateLegacyDatabaseFile 将旧版库转换为当前结构库文件并返回新文件路径：
// 先按当前项目初始化全新结构与默认数据，再仅拷贝迁移范围内的表
func migrateLegacyDatabaseFile(sourcePath string) (string, error) {
	migratedPath := sourcePath + "-migrated.db"
	if err := os.Remove(migratedPath); err != nil && !os.IsNotExist(err) {
		return "", err
	}
	db, err := sql.Open("sqlite", migratedPath)
	if err != nil {
		return "", err
	}
	defer db.Close()
	if err := common.SetupSQLite(db); err != nil {
		return "", err
	}
	if err := initLegacyTargetSchema(db); err != nil {
		return "", err
	}
	escaped := strings.ReplaceAll(sourcePath, "'", "''")
	if _, err := db.Exec(fmt.Sprintf(`ATTACH DATABASE '%s' AS legacy`, escaped)); err != nil {
		return "", err
	}
	for _, table := range legacyImportTables {
		if err := copyLegacyTable(db, table); err != nil {
			_, _ = db.Exec(`DETACH DATABASE legacy`)
			return "", fmt.Errorf("copy legacy table %s failed: %w", table, err)
		}
	}
	if _, err := db.Exec(`DETACH DATABASE legacy`); err != nil {
		return "", err
	}
	if err := normalizeLegacyBuiltinDictionaries(db); err != nil {
		return "", err
	}
	if err := ensureLegacyAdminFallback(db); err != nil {
		return "", err
	}
	if err := db.Close(); err != nil {
		return "", err
	}
	if err := service.ValidateSQLiteFile(migratedPath); err != nil {
		return "", err
	}
	return migratedPath, nil
}

// initLegacyTargetSchema 在全新空库上执行当前项目的建表语句与系统资源表初始化
func initLegacyTargetSchema(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	for i, statement := range databaseBootstrapSQL {
		if _, err := tx.Exec(statement); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("execute bootstrap sql #%d failed: %w", i+1, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return settings.EnsureResourceSchema(db)
}

// copyLegacyTable 按两库共有列拷贝整表数据：源表缺失时跳过，内置种子的字典表仅在有数据时整表替换
func copyLegacyTable(db *sql.DB, table string) error {
	sourceCols, err := schemaColumns(db, "legacy", table)
	if err != nil {
		return err
	}
	if len(sourceCols) == 0 {
		log.Printf("Legacy import skipped missing table: %s", table)
		return nil
	}
	targetCols, err := schemaColumns(db, "main", table)
	if err != nil {
		return err
	}
	shared := sharedColumns(targetCols, sourceCols)
	if len(shared) == 0 {
		log.Printf("Legacy import skipped table without shared columns: %s", table)
		return nil
	}
	if legacySeededTables[table] {
		var count int
		if err := db.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM legacy.%s`, quoteIdentifier(table))).Scan(&count); err != nil {
			return err
		}
		if count == 0 {
			return nil
		}
		if _, err := db.Exec(fmt.Sprintf(`DELETE FROM main.%s`, quoteIdentifier(table))); err != nil {
			return err
		}
	}
	columns := joinQuotedIdentifiers(shared)
	_, err = db.Exec(fmt.Sprintf(`INSERT INTO main.%s (%s) SELECT %s FROM legacy.%s`,
		quoteIdentifier(table), columns, columns, quoteIdentifier(table)))
	return err
}

// normalizeLegacyBuiltinDictionaries 迁移后统一内置字典：状态类型编号从 1 起、补齐缺失的内置硬件类型、英文内置名转中文
func normalizeLegacyBuiltinDictionaries(db *sql.DB) error {
	if err := renumberLegacyStatusTypes(db); err != nil {
		return err
	}
	if err := supplementBuiltinItemTypes(db); err != nil {
		return err
	}
	return translateLegacyBuiltinNames(db)
}

// renumberLegacyStatusTypes 旧库状态类型编号可能从 0 开始，迁移后统一按原顺序从 1 连续编号，并同步硬件记录的状态引用
func renumberLegacyStatusTypes(db *sql.DB) error {
	rows, err := db.Query(`SELECT id FROM main.statustypes ORDER BY id`)
	if err != nil {
		return err
	}
	oldIDs := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		oldIDs = append(oldIDs, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	needsRenumber := false
	for index, oldID := range oldIDs {
		if oldID != int64(index+1) {
			needsRenumber = true
			break
		}
	}
	if !needsRenumber {
		return nil
	}

	if _, err := db.Exec(`ALTER TABLE main.statustypes RENAME TO statustypes_renumber`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE TABLE main.statustypes (id INTEGER PRIMARY KEY AUTOINCREMENT, statusdesc, color TEXT)`); err != nil {
		return err
	}
	if _, err := db.Exec(`INSERT INTO main.statustypes (id, statusdesc, color) SELECT ROW_NUMBER() OVER (ORDER BY id), statusdesc, COALESCE(color, '') FROM main.statustypes_renumber`); err != nil {
		return err
	}
	if _, err := db.Exec(`DROP TABLE main.statustypes_renumber`); err != nil {
		return err
	}

	const offset = int64(1000000)
	for _, oldID := range oldIDs {
		if _, err := db.Exec(`UPDATE main.items SET status = ? WHERE status = ?`, oldID+offset, oldID); err != nil {
			return err
		}
	}
	for index, oldID := range oldIDs {
		if _, err := db.Exec(`UPDATE main.items SET status = ? WHERE status = ?`, int64(index+1), oldID+offset); err != nil {
			return err
		}
	}
	log.Printf("Legacy import renumbered statustypes from 1: rows=%d", len(oldIDs))
	return nil
}

// supplementBuiltinItemTypes 旧项目的五个内置硬件类型可删除，迁移后按名称补齐缺失项
func supplementBuiltinItemTypes(db *sql.DB) error {
	for _, name := range service.BuiltinDictionaryNames("itemtypes") {
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM main.itemtypes WHERE LOWER(TRIM(COALESCE(typedesc, ''))) = LOWER(?)`, name).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		if _, err := db.Exec(`INSERT INTO main.itemtypes (typedesc, hassoftware) VALUES (?, 1)`, name); err != nil {
			return err
		}
		log.Printf("Legacy import supplemented missing builtin item type: %s", name)
	}
	return nil
}

// translateLegacyBuiltinNames 旧库内置文件类型与合同类型可能为英文名，统一转换为当前项目中文名（非内置名称不动）
func translateLegacyBuiltinNames(db *sql.DB) error {
	tables := []struct {
		table      string
		column     string
		dictionary string
	}{
		{"filetypes", "typedesc", "filetypes"},
		{"contracttypes", "name", "contracttypes"},
	}
	for _, item := range tables {
		for english, chinese := range service.LegacyBuiltinNameTranslations(item.dictionary) {
			if _, err := db.Exec(fmt.Sprintf(
				`UPDATE main.%s SET %s = ? WHERE LOWER(TRIM(COALESCE(%s, ''))) = LOWER(?)`,
				quoteIdentifier(item.table), quoteIdentifier(item.column), quoteIdentifier(item.column),
			), chinese, english); err != nil {
				return err
			}
		}
	}
	return nil
}

// ensureLegacyAdminFallback 旧库无任何用户时补默认管理员，避免迁移后无法登录
func ensureLegacyAdminFallback(db *sql.DB) error {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	adminPass, err := security.HashPassword("admin123")
	if err != nil {
		return err
	}
	_, err = db.Exec(`INSERT INTO users (username, userdesc, pass, usertype) VALUES (?, ?, ?, ?)`,
		"admin", "admin", adminPass, 0)
	return err
}

// schemaColumns 返回指定 schema 下表的列名，表不存在时返回空切片
func schemaColumns(db *sql.DB, schema, table string) ([]string, error) {
	rows, err := db.Query(fmt.Sprintf(`PRAGMA %s.table_info(%s)`, schema, quoteIdentifier(table)))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns := make([]string, 0)
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull int
		var defaultValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultValue, &pk); err != nil {
			return nil, err
		}
		columns = append(columns, name)
	}
	return columns, rows.Err()
}

// sharedColumns 取目标列顺序下两库共有的列
func sharedColumns(target, source []string) []string {
	sourceSet := make(map[string]bool, len(source))
	for _, name := range source {
		sourceSet[name] = true
	}
	shared := make([]string, 0, len(target))
	for _, name := range target {
		if sourceSet[name] {
			shared = append(shared, name)
		}
	}
	return shared
}

// quoteIdentifier 对 SQLite 标识符加双引号转义
func quoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// joinQuotedIdentifiers 以逗号连接已转义的标识符列表
func joinQuotedIdentifiers(names []string) string {
	quoted := make([]string, 0, len(names))
	for _, name := range names {
		quoted = append(quoted, quoteIdentifier(name))
	}
	return strings.Join(quoted, ", ")
}

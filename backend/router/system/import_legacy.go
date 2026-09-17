// 旧版数据库导入适配：新建当前结构空库后仅拷贝资产管理与资料管理数据；用户按用户名合并，
// 系统配置与审计历史从当前运行库恢复，硬件维护日志不迁移，标签预设等保持当前项目默认。
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

// legacyImportTables 旧库迁移的数据范围：资料管理全部字典与资产管理全部菜单数据（用户、硬件维护日志 actions 不在此列）
var legacyImportTables = []string{
	"itemtypes", "filetypes", "statustypes", "dpttypes", "contracttypes", "contractsubtypes", "tags", "tag2Item", "tag2software",
	"items", "item2soft", "item2inv", "item2file", "itemlink",
	"software", "soft2inv", "software2file",
	"invoices", "invoice2file",
	"agents",
	"files",
	"contracts", "contract2item", "contract2inv", "contract2soft", "contract2file", "contractevents",
	"locations", "locareas", "racks",
}

// legacyPreservedCurrentTables 旧库迁移完成后从当前运行库恢复的系统表：系统配置、LDAP 认证开关、
// 用户角色档案与审计历史（settings 表承载 LDAP 启用状态、服务器与加密绑定凭据）
var legacyPreservedCurrentTables = []string{
	"settings_base", "settings_email", "settings_auth_providers",
	"settings_roles", "settings_role_status",
	"settings_user_groups", "settings_user_group_members", "settings_user_group_roles",
	"settings_user_roles", "settings_user_profiles",
	"settings",
	"history",
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

// prepareImportDatabaseFile 判断上传库类型：当前项目库直接安装，旧版库转换出新库文件后安装；
// currentDB 为当前运行库连接，用于旧库迁移时恢复系统配置、合并用户并保留审计历史
func prepareImportDatabaseFile(sourcePath string, currentDB *sql.DB) (string, error) {
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
	migratedPath, err := migrateLegacyDatabaseFile(sourcePath, currentDB)
	if err != nil {
		log.Printf("Legacy database migration failed: %v", err)
		return "", errors.New("旧版数据库转换失败：" + err.Error())
	}
	log.Printf("Legacy database migration completed: %s", migratedPath)
	return migratedPath, nil
}

// migrateLegacyDatabaseFile 将旧版库转换为当前结构库文件并返回新文件路径：
// 先按当前项目初始化全新结构与默认数据，再拷贝迁移范围内的表，
// 用户按用户名与当前运行库合并，系统配置与审计历史从当前运行库恢复
func migrateLegacyDatabaseFile(sourcePath string, currentDB *sql.DB) (string, error) {
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
	var importedUsers []legacyImportedUser
	for _, table := range legacyImportTables {
		if err := copyLegacyTable(db, table); err != nil {
			_, _ = db.Exec(`DETACH DATABASE legacy`)
			return "", fmt.Errorf("copy legacy table %s failed: %w", table, err)
		}
	}
	importedUsers, err = mergeLegacyUsers(db, currentDB)
	if err != nil {
		_, _ = db.Exec(`DETACH DATABASE legacy`)
		return "", fmt.Errorf("merge legacy users failed: %w", err)
	}
	if _, err := db.Exec(`DETACH DATABASE legacy`); err != nil {
		return "", err
	}
	if err := normalizeLegacyBuiltinDictionaries(db); err != nil {
		return "", err
	}
	if err := restoreCurrentSystemData(db, currentDB); err != nil {
		return "", err
	}
	assignLegacyImportedUserRoles(db, importedUsers)
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

// normalizeLegacyBuiltinDictionaries 迁移后统一内置字典：状态类型编号从 1 起、硬件类型按当前默认重建、英文内置名转中文
func normalizeLegacyBuiltinDictionaries(db *sql.DB) error {
	if err := renumberLegacyStatusTypes(db); err != nil {
		return err
	}
	if err := rebuildLegacyItemTypes(db); err != nil {
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

// legacyBuiltinItemTypeDef 内置硬件类型定义及其默认软件支持配置
type legacyBuiltinItemTypeDef struct {
	name        string
	hasSoftware int64
}

// legacyBuiltinItemTypeDefs 内置硬件类型默认清单：编号固定 1-5，与数据库种子一致，仅服务器默认支持软件
func legacyBuiltinItemTypeDefs() []legacyBuiltinItemTypeDef {
	return []legacyBuiltinItemTypeDef{
		{"服务器", 1}, {"存储", 0}, {"交换机", 0}, {"电话", 0}, {"安防", 0},
	}
}

// rebuildLegacyItemTypes 迁移后重建硬件类型：编号 1-5 固定为当前内置默认，
// 旧库其余类型按原编号顺序从 6 连续追加，名称与内置或已追加类型重复的忽略，
// 并把硬件记录的类型编号同步映射到新编号
func rebuildLegacyItemTypes(db *sql.DB) error {
	defs := legacyBuiltinItemTypeDefs()
	rows, err := db.Query(`SELECT id, TRIM(COALESCE(typedesc, '')), CASE WHEN COALESCE(hassoftware, 0) = 1 THEN 1 ELSE 0 END FROM main.itemtypes ORDER BY id`)
	if err != nil {
		return err
	}
	type legacyRow struct {
		id          int64
		name        string
		hasSoftware int64
	}
	oldRows := make([]legacyRow, 0)
	for rows.Next() {
		var item legacyRow
		if err := rows.Scan(&item.id, &item.name, &item.hasSoftware); err != nil {
			rows.Close()
			return err
		}
		oldRows = append(oldRows, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	builtinKeys := make(map[string]int64, len(defs))
	for index, def := range defs {
		builtinKeys[normalizedDictionaryKey(def.name)] = int64(index + 1)
	}

	idMap := make(map[int64]int64, len(oldRows))
	appended := make(map[string]int64)
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM main.itemtypes`); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.Exec(`DELETE FROM sqlite_sequence WHERE name = 'itemtypes'`); err != nil {
		_ = tx.Rollback()
		return err
	}
	for index, def := range defs {
		if _, err := tx.Exec(`INSERT INTO main.itemtypes (id, typedesc, hassoftware) VALUES (?, ?, ?)`, index+1, def.name, def.hasSoftware); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	nextID := int64(len(defs)) + 1
	for _, item := range oldRows {
		key := normalizedDictionaryKey(item.name)
		if key == "" {
			continue
		}
		mapped, duplicated := builtinKeys[key]
		if !duplicated {
			if existing, ok := appended[key]; ok {
				mapped, duplicated = existing, true
			}
		}
		if !duplicated {
			mapped = nextID
			appended[key] = nextID
			nextID++
			if _, err := tx.Exec(`INSERT INTO main.itemtypes (id, typedesc, hassoftware) VALUES (?, ?, ?)`, mapped, item.name, item.hasSoftware); err != nil {
				_ = tx.Rollback()
				return err
			}
		}
		idMap[item.id] = mapped
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return remapLegacyItemTypeReferences(db, idMap)
}

// remapLegacyItemTypeReferences 按新旧编号映射同步硬件记录的类型编号，借助偏移量避免映射过程中的中间态冲突
func remapLegacyItemTypeReferences(db *sql.DB, idMap map[int64]int64) error {
	const offset = int64(1000000)
	for oldID, newID := range idMap {
		if oldID == newID {
			continue
		}
		if _, err := db.Exec(`UPDATE main.items SET itemtypeid = ? WHERE itemtypeid = ?`, oldID+offset, oldID); err != nil {
			return err
		}
	}
	for oldID, newID := range idMap {
		if oldID == newID {
			continue
		}
		if _, err := db.Exec(`UPDATE main.items SET itemtypeid = ? WHERE itemtypeid = ?`, newID, oldID+offset); err != nil {
			return err
		}
	}
	return nil
}

// normalizedDictionaryKey 生成字典名称的比对键：忽略首尾空格与大小写
func normalizedDictionaryKey(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
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

// legacyImportedUser 记录本次旧库导入的新用户编号与其旧库用户类型
type legacyImportedUser struct {
	id       int64
	userType int64
}

// mergeLegacyUsers 旧库用户按用户名合并：当前库已有的同名用户保留现状不导入，
// 其余用户随旧库数据导入并保持原编号（编号与已存在用户冲突时改用自动分配的新编号），
// 返回本次导入的新用户清单
func mergeLegacyUsers(migrated *sql.DB, currentDB *sql.DB) ([]legacyImportedUser, error) {
	if currentDB == nil {
		return nil, copyLegacyTable(migrated, "users")
	}
	if _, err := migrated.Exec(`DELETE FROM main.users`); err != nil {
		return nil, err
	}
	if err := copyTableRowsAcross(currentDB, migrated, "users"); err != nil {
		return nil, err
	}
	legacyColumns, err := schemaColumns(migrated, "legacy", "users")
	if err != nil {
		return nil, err
	}
	if len(legacyColumns) == 0 {
		log.Printf("Legacy import skipped missing users table")
		return nil, nil
	}
	targetColumns, err := columnsOf(migrated, "users")
	if err != nil {
		return nil, err
	}
	shared := sharedColumns(targetColumns, legacyColumns)
	if len(shared) == 0 {
		return nil, nil
	}
	rows, err := migrated.Query(fmt.Sprintf(`SELECT %s FROM legacy.users`, joinQuotedIdentifiers(shared)))
	if err != nil {
		return nil, err
	}
	currentNames, err := collectCurrentUserNames(currentDB)
	if err != nil {
		rows.Close()
		return nil, err
	}
	usedIDs, err := collectUsedUserIDs(migrated)
	if err != nil {
		rows.Close()
		return nil, err
	}
	values := make([]interface{}, len(shared))
	pointers := make([]interface{}, len(values))
	for i := range values {
		pointers[i] = &values[i]
	}
	idIndex := -1
	nameIndex := -1
	typeIndex := -1
	for i, column := range shared {
		switch strings.ToLower(column) {
		case "id":
			idIndex = i
		case "username":
			nameIndex = i
		case "usertype":
			typeIndex = i
		}
	}
	var importedUsers []legacyImportedUser
	var legacyRows [][]interface{}
	for rows.Next() {
		if err := rows.Scan(pointers...); err != nil {
			rows.Close()
			return nil, err
		}
		row := make([]interface{}, len(shared))
		copy(row, values)
		legacyRows = append(legacyRows, row)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	inserted, skipped := 0, 0
	for _, row := range legacyRows {
		name := normalizedDictionaryKey(asText(row[nameIndex]))
		if nameIndex >= 0 && currentNames[name] {
			skipped++
			continue
		}
		keepID := idIndex >= 0 && row[idIndex] != nil
		if keepID {
			if _, conflict := usedIDs[asInt64(row[idIndex])]; conflict {
				keepID = false
			}
		}
		columns := shared
		args := row
		if !keepID {
			columns = make([]string, 0, len(shared)-1)
			args = make([]interface{}, 0, len(shared)-1)
			for i := range shared {
				if i == idIndex {
					continue
				}
				columns = append(columns, shared[i])
				args = append(args, row[i])
			}
		}
		insertSQL := fmt.Sprintf(`INSERT INTO main.users (%s) VALUES (%s)`,
			joinQuotedIdentifiers(columns), strings.TrimSuffix(strings.Repeat("?, ", len(columns)), ", "))
		result, err := migrated.Exec(insertSQL, args...)
		if err != nil {
			return nil, err
		}
		newID := asInt64(row[idIndex])
		if !keepID {
			newID, err = result.LastInsertId()
			if err != nil {
				return nil, err
			}
		}
		importedUsers = append(importedUsers, legacyImportedUser{id: newID, userType: asInt64(row[typeIndex])})
		if keepID {
			usedIDs[asInt64(row[idIndex])] = true
		}
		inserted++
	}
	log.Printf("Legacy users merged: inserted=%d skipped=%d", inserted, skipped)
	return importedUsers, nil
}

// assignLegacyImportedUserRoles 按旧库用户类型为本次导入的新用户补充内置角色关联：
// 用户类型 0（管理员）关联 admin 角色，其余关联 viewer 只读角色；仅在当前恢复后的
// 角色体系中缺失该关联时补齐，已有用户不受影响
func assignLegacyImportedUserRoles(migrated *sql.DB, imported []legacyImportedUser) {
	for _, user := range imported {
		roleKey := "viewer"
		if user.userType == 0 {
			roleKey = "admin"
		}
		var roleID int64
		if err := migrated.QueryRow(`SELECT id FROM settings_roles WHERE key = ? AND builtin = 1`, roleKey).Scan(&roleID); err != nil {
			log.Printf("Legacy imported user role lookup failed: user=%d role=%s err=%v", user.id, roleKey, err)
			continue
		}
		if _, err := migrated.Exec(`INSERT INTO settings_user_roles (user_id, role_id) VALUES (?, ?) ON CONFLICT (user_id, role_id) DO NOTHING`, user.id, roleID); err != nil {
			log.Printf("Legacy imported user role assign failed: user=%d role=%s err=%v", user.id, roleKey, err)
		}
	}
}

// restoreCurrentSystemData 旧库迁移完成后从当前运行库恢复系统配置、用户角色档案与审计历史，避免重新导入后丢失
func restoreCurrentSystemData(migrated *sql.DB, currentDB *sql.DB) error {
	if currentDB == nil {
		return nil
	}
	for _, table := range legacyPreservedCurrentTables {
		exists, err := common.SQLiteTableExists(currentDB, table)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
		if err := copyTableRowsAcross(currentDB, migrated, table); err != nil {
			return fmt.Errorf("restore current table %s failed: %w", table, err)
		}
	}
	return nil
}

// copyTableRowsAcross 将源连接上指定表的全部行复制到目标连接的同名表，目标表先清空
func copyTableRowsAcross(source *sql.DB, dest *sql.DB, table string) error {
	columns, err := columnsOf(source, table)
	if err != nil {
		return err
	}
	if len(columns) == 0 {
		return nil
	}
	if _, err := dest.Exec(fmt.Sprintf(`DELETE FROM %s`, quoteIdentifier(table))); err != nil {
		return err
	}
	rows, err := source.Query(fmt.Sprintf(`SELECT %s FROM %s`, joinQuotedIdentifiers(columns), quoteIdentifier(table)))
	if err != nil {
		return err
	}
	defer rows.Close()
	insertSQL := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s)`,
		quoteIdentifier(table), joinQuotedIdentifiers(columns), strings.TrimSuffix(strings.Repeat("?, ", len(columns)), ", "))
	tx, err := dest.Begin()
	if err != nil {
		return err
	}
	values := make([]interface{}, len(columns))
	pointers := make([]interface{}, len(values))
	for i := range values {
		pointers[i] = &values[i]
	}
	for rows.Next() {
		if err := rows.Scan(pointers...); err != nil {
			_ = tx.Rollback()
			return err
		}
		if _, err := tx.Exec(insertSQL, values...); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	if err := rows.Err(); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := rows.Close(); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// columnsOf 返回连接上指定表的列名，表不存在时返回空切片
func columnsOf(db *sql.DB, table string) ([]string, error) {
	rows, err := db.Query(fmt.Sprintf(`PRAGMA table_info(%s)`, quoteIdentifier(table)))
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

// collectCurrentUserNames 收集当前库用户名集合，键为忽略大小写与首尾空格的比对键
func collectCurrentUserNames(currentDB *sql.DB) (map[string]bool, error) {
	rows, err := currentDB.Query(`SELECT username FROM users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	names := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names[normalizedDictionaryKey(name)] = true
	}
	return names, rows.Err()
}

// collectUsedUserIDs 收集目标库已占用的用户编号集合
func collectUsedUserIDs(migrated *sql.DB) (map[int64]bool, error) {
	rows, err := migrated.Query(`SELECT id FROM main.users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make(map[int64]bool)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids[id] = true
	}
	return ids, rows.Err()
}

// asText 把动态扫描值转为比对文本，空值按空字符串处理
func asText(value interface{}) string {
	if text, ok := value.(string); ok {
		return text
	}
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%v", value)
}

// asInt64 把动态扫描值转为整数，无法转换时返回 0
func asInt64(value interface{}) int64 {
	switch typed := value.(type) {
	case int64:
		return typed
	case int:
		return int64(typed)
	case float64:
		return int64(typed)
	}
	return 0
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

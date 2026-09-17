package system

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

// newLegacyDatabaseFixture 构造模拟旧平台结构的数据库文件：settings 为旧格式、statustypes 无 color、
// actions 带 entrydate、history 为旧结构、settings_* 系列表缺失、含多余的 viewhist 表
func newLegacyDatabaseFixture(t *testing.T, path string, withUsers bool) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	statements := []string{
		`CREATE TABLE items (id INTEGER PRIMARY KEY AUTOINCREMENT, itemtypeid integer, model, label, status, maintenanceinfo)`,
		`CREATE TABLE agents (id INTEGER PRIMARY KEY AUTOINCREMENT, type integer, title, contactinfo, contacts, urls)`,
		`CREATE TABLE actions (id INTEGER PRIMARY KEY AUTOINCREMENT, itemid INTEGER, actiondate integer, description, invoiceinfo, isauto, entrydate)`,
		`CREATE TABLE statustypes (id INTEGER PRIMARY KEY AUTOINCREMENT, statusdesc)`,
		`CREATE TABLE itemtypes (id INTEGER PRIMARY KEY AUTOINCREMENT, typeid, typedesc, hassoftware integer)`,
		`CREATE TABLE filetypes (id INTEGER PRIMARY KEY AUTOINCREMENT, typedesc)`,
		`CREATE TABLE contracttypes (id INTEGER PRIMARY KEY AUTOINCREMENT, name)`,
		`CREATE TABLE tags (id INTEGER PRIMARY KEY AUTOINCREMENT, name)`,
		`CREATE TABLE tag2Item (itemid integer, tagid integer)`,
		`CREATE TABLE history (id INTEGER PRIMARY KEY AUTOINCREMENT, date integer, sql, authuser, ip)`,
		`CREATE TABLE settings (companytitle, dateformat, currency, lang, version, timezone, dbversion, useldap integer default 0, ldap_server, ldap_dn, ldap_getusers, ldap_getusers_filter)`,
		`CREATE TABLE viewhist (id INTEGER PRIMARY KEY AUTOINCREMENT, url, description)`,
		`INSERT INTO agents (id, title) VALUES (1, '联想')`,
		`INSERT INTO items (id, itemtypeid, model, label, status, maintenanceinfo) VALUES (16, 1, 'TaiShan 2280V2', '2102315PAM10RA100004121', 0, '每季度除尘')`,
		`INSERT INTO items (id, itemtypeid, model, label, status, maintenanceinfo) VALUES (17, 10, 'AR6120', 'HW-AR6120', 0, '升级固件')`,
		`INSERT INTO items (id, itemtypeid, model, label, status, maintenanceinfo) VALUES (18, 11, 'DS4231', 'NAS-01', 0, '')`,
		`INSERT INTO actions (id, itemid, actiondate, description, invoiceinfo, isauto, entrydate) VALUES (1, 16, 1700000000, '更换内存', '成功', 0, '2023-11-14')`,
		`INSERT INTO statustypes (id, statusdesc) VALUES (0, '使用中'), (1, '库存'), (2, '有故障'), (3, '报废')`,
		`INSERT INTO itemtypes (id, typeid, typedesc, hassoftware) VALUES (1, NULL, '服务器', 0)`,
		`INSERT INTO itemtypes (id, typeid, typedesc, hassoftware) VALUES (10, NULL, '路由器', 1)`,
		`INSERT INTO itemtypes (id, typeid, typedesc, hassoftware) VALUES (11, NULL, ' 存储 ', 1)`,
		`INSERT INTO filetypes (id, typedesc) VALUES (1, 'photo'), (2, 'manual'), (3, '检测报告')`,
		`INSERT INTO contracttypes (id, name) VALUES (1, 'Support & Maintenance')`,
		`INSERT INTO tags (id, name) VALUES (1, '核心设备')`,
		`INSERT INTO tag2Item (itemid, tagid) VALUES (16, 1)`,
		`INSERT INTO history (id, date, sql, authuser, ip) VALUES (1, 1700000000, 'UPDATE items SET model=1', 'admin', '127.0.0.1')`,
		`INSERT INTO settings (companytitle, useldap, ldap_server) VALUES ('旧平台', 0, '')`,
	}
		if withUsers {
			statements = append(statements,
				`CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, username, userdesc, pass, cookie1, usertype integer)`,
				`INSERT INTO users (id, username, userdesc, pass, cookie1, usertype) VALUES (2, 'wangyq', '王玉荃', 'Sunline2023', '', 0)`,
				`INSERT INTO users (id, username, userdesc, pass, cookie1, usertype) VALUES (5, 'zhangsan', '张三', 'pass5', '', 1)`,
				`INSERT INTO users (id, username, userdesc, pass, cookie1, usertype) VALUES (6, 'lisi', '李四', 'pass6', '', 0)`,
			)
		}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
}

// newCurrentDatabaseFixture 构造带 settings_user_profiles 的当前项目结构库文件
func newCurrentDatabaseFixture(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE settings_user_profiles (user_id INTEGER PRIMARY KEY, email TEXT)`); err != nil {
		t.Fatal(err)
	}
}

// newCurrentLiveDatabaseFixture 构造已完成配置的当前项目运行库：含用户、系统配置、用户档案与审计历史
func newCurrentLiveDatabaseFixture(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	statements := []string{
		`CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, username, userdesc, pass, cookie1, usertype integer)`,
		`CREATE TABLE history (id INTEGER PRIMARY KEY AUTOINCREMENT, date integer, sql, authuser, ip, module TEXT DEFAULT '', action TEXT DEFAULT '', target TEXT DEFAULT '', detail TEXT DEFAULT '', result TEXT DEFAULT 'success')`,
		`CREATE TABLE settings_base (id INTEGER PRIMARY KEY CHECK (id = 1), config TEXT NOT NULL DEFAULT '{}', updated_at INTEGER NOT NULL DEFAULT 0)`,
		`CREATE TABLE settings_user_profiles (user_id INTEGER PRIMARY KEY, email TEXT NOT NULL DEFAULT '', disabled INTEGER NOT NULL DEFAULT 0, source TEXT NOT NULL DEFAULT 'local', created_at INTEGER NOT NULL DEFAULT 0, updated_at INTEGER NOT NULL DEFAULT 0, last_login_at INTEGER NOT NULL DEFAULT 0)`,
		`CREATE TABLE settings_roles (id INTEGER PRIMARY KEY AUTOINCREMENT, key TEXT NOT NULL UNIQUE, name TEXT NOT NULL, description TEXT NOT NULL DEFAULT '', permissions TEXT NOT NULL DEFAULT '[]', builtin INTEGER NOT NULL DEFAULT 0, created_at INTEGER NOT NULL DEFAULT 0, updated_at INTEGER NOT NULL DEFAULT 0)`,
		`CREATE TABLE settings_user_roles (user_id INTEGER NOT NULL, role_id INTEGER NOT NULL, PRIMARY KEY (user_id, role_id))`,
		`INSERT INTO users (id, username, userdesc, pass, cookie1, usertype) VALUES (1, 'admin', '管理员', 'currentadmin', '', 0)`,
		`INSERT INTO users (id, username, userdesc, pass, cookie1, usertype) VALUES (2, 'wangyq', '王玉荃', 'currentpass', '', 0)`,
		`INSERT INTO history (id, date, sql, authuser, ip, module, action, target, detail, result) VALUES (1, 1700000000, 'UPDATE items', 'admin', '127.0.0.1', '资产管理', '新增', '硬件 16', '成功', 'success')`,
		`INSERT INTO settings_base (id, config, updated_at) VALUES (1, '{"brand":"当前公司"}', 100)`,
		`INSERT INTO settings_user_profiles (user_id, email, disabled, source) VALUES (2, 'wangyq@corp.com', 0, 'local')`,
		`INSERT INTO settings_roles (id, key, name, permissions, builtin) VALUES (1, 'operator', '操作员', '[]', 1)`,
		`INSERT INTO settings_roles (id, key, name, permissions, builtin) VALUES (2, 'admin', 'admin', '[]', 1)`,
		`INSERT INTO settings_roles (id, key, name, permissions, builtin) VALUES (3, 'viewer', 'viewer', '[]', 1)`,
		`INSERT INTO settings_user_roles (user_id, role_id) VALUES (2, 1)`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
}

func openVerified(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestIsCurrentProjectDatabase(t *testing.T) {
	legacyPath := t.TempDir() + "/legacy.db"
	newLegacyDatabaseFixture(t, legacyPath, true)
	legacy, err := isCurrentProjectDatabase(openVerified(t, legacyPath))
	if err != nil {
		t.Fatal(err)
	}
	if legacy {
		t.Fatal("legacy fixture should not be detected as current project database")
	}

	currentPath := t.TempDir() + "/current.db"
	newCurrentDatabaseFixture(t, currentPath)
	current, err := isCurrentProjectDatabase(openVerified(t, currentPath))
	if err != nil {
		t.Fatal(err)
	}
	if !current {
		t.Fatal("current fixture should be detected as current project database")
	}
}

// TestMigrateLegacyDatabaseFile 锁定旧库迁移行为：范围内数据带 ID 拷贝、范围外保持当前默认、
// 用户按用户名与当前库合并、系统配置与审计历史从当前库恢复、历史记录与多余表不迁移
func TestMigrateLegacyDatabaseFile(t *testing.T) {
	legacyPath := t.TempDir() + "/legacy.db"
	newLegacyDatabaseFixture(t, legacyPath, true)
	currentPath := t.TempDir() + "/current.db"
	newCurrentLiveDatabaseFixture(t, currentPath)
	live := openVerified(t, currentPath)
	migratedPath, err := migrateLegacyDatabaseFile(legacyPath, live)
	if err != nil {
		t.Fatal(err)
	}
	db := openVerified(t, migratedPath)

	current, err := isCurrentProjectDatabase(db)
	if err != nil {
		t.Fatal(err)
	}
	if !current {
		t.Fatal("migrated database should contain settings_user_profiles")
	}

	var count int
	assertCount := func(want int, query string) {
		t.Helper()
		if err := db.QueryRow(query).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != want {
			t.Fatalf("query %q rows=%d, want %d", query, count, want)
		}
	}

	assertCount(1, `SELECT COUNT(*) FROM items WHERE id=16 AND model='TaiShan 2280V2' AND label='2102315PAM10RA100004121'`)
	assertCount(0, `SELECT COUNT(*) FROM actions`)
	assertCount(1, `SELECT COUNT(*) FROM tag2Item WHERE itemid=16 AND tagid=1`)
	assertCount(4, `SELECT COUNT(*) FROM statustypes WHERE id IN (1, 2, 3, 4)`)
	assertCount(1, `SELECT COUNT(*) FROM statustypes WHERE id=1 AND statusdesc='使用中'`)
	assertCount(1, `SELECT COUNT(*) FROM items WHERE id=16 AND status=1`)
	assertCount(6, `SELECT COUNT(*) FROM itemtypes`)
	assertCount(5, `SELECT COUNT(*) FROM itemtypes WHERE id IN (1, 2, 3, 4, 5) AND typedesc IN ('服务器', '存储', '交换机', '电话', '安防')`)
	assertCount(1, `SELECT COUNT(*) FROM itemtypes WHERE id=1 AND typedesc='服务器' AND hassoftware=1`)
	assertCount(4, `SELECT COUNT(*) FROM itemtypes WHERE id IN (2, 3, 4, 5) AND hassoftware=0`)
	assertCount(1, `SELECT COUNT(*) FROM itemtypes WHERE id=6 AND typedesc='路由器' AND hassoftware=1`)
	assertCount(0, `SELECT COUNT(*) FROM itemtypes WHERE typedesc='存储' AND hassoftware=1`)
	assertCount(1, `SELECT COUNT(*) FROM items WHERE id=16 AND itemtypeid=1`)
	assertCount(1, `SELECT COUNT(*) FROM items WHERE id=17 AND itemtypeid=6`)
	assertCount(1, `SELECT COUNT(*) FROM items WHERE id=18 AND itemtypeid=2`)
	assertCount(1, `SELECT COUNT(*) FROM items WHERE id=16 AND maintenanceinfo='每季度除尘'`)
	assertCount(1, `SELECT COUNT(*) FROM items WHERE id=17 AND maintenanceinfo='升级固件'`)
	assertCount(3, `SELECT COUNT(*) FROM filetypes`)
	assertCount(2, `SELECT COUNT(*) FROM filetypes WHERE typedesc IN ('照片', '手册')`)
	assertCount(0, `SELECT COUNT(*) FROM filetypes WHERE typedesc='photo'`)
	assertCount(1, `SELECT COUNT(*) FROM contracttypes WHERE name='支持 & 维护'`)
	assertCount(3, `SELECT COUNT(*) FROM labelpapers`)
	assertCount(0, `SELECT COUNT(*) FROM actions`)
	assertCount(1, `SELECT COUNT(*) FROM users WHERE id=1 AND username='admin' AND pass='currentadmin'`)
	assertCount(1, `SELECT COUNT(*) FROM users WHERE id=2 AND username='wangyq' AND userdesc='王玉荃' AND pass='currentpass'`)
	assertCount(1, `SELECT COUNT(*) FROM users WHERE id=5 AND username='zhangsan' AND userdesc='张三'`)
	assertCount(1, `SELECT COUNT(*) FROM users WHERE id=6 AND username='lisi' AND userdesc='李四'`)
	assertCount(4, `SELECT COUNT(*) FROM users`)
	assertCount(1, `SELECT COUNT(*) FROM history WHERE id=1 AND module='资产管理' AND target='硬件 16'`)
	assertCount(1, `SELECT COUNT(*) FROM settings_base WHERE config='{"brand":"当前公司"}'`)
	assertCount(1, `SELECT COUNT(*) FROM settings_user_profiles WHERE user_id=2 AND email='wangyq@corp.com'`)
	assertCount(1, `SELECT COUNT(*) FROM settings_user_roles WHERE user_id=2 AND role_id=1`)
	assertCount(1, `SELECT COUNT(*) FROM settings_base`)
	assertCount(1, `SELECT COUNT(*) FROM settings_user_roles ur JOIN settings_roles r ON r.id=ur.role_id WHERE ur.user_id=5 AND r.key='viewer'`)
	assertCount(1, `SELECT COUNT(*) FROM settings_user_roles ur JOIN settings_roles r ON r.id=ur.role_id WHERE ur.user_id=6 AND r.key='admin'`)

	if _, err := db.Query(`SELECT * FROM viewhist`); err == nil {
		t.Fatal("viewhist table should not exist in migrated database")
	}

	var color sql.NullString
	if err := db.QueryRow(`SELECT color FROM statustypes WHERE id=1`).Scan(&color); err != nil {
		t.Fatal(err)
	}
}

// TestMigrateLegacyDatabaseFileAdminFallback 迁移来源无当前库时按旧整表方式合并用户，旧库无用户则补默认管理员
func TestMigrateLegacyDatabaseFileAdminFallback(t *testing.T) {
	legacyPath := t.TempDir() + "/legacy.db"
	newLegacyDatabaseFixture(t, legacyPath, false)
	migratedPath, err := migrateLegacyDatabaseFile(legacyPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	db := openVerified(t, migratedPath)

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM users WHERE username='admin' AND usertype=0`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("admin fallback rows=%d, want 1", count)
	}
}

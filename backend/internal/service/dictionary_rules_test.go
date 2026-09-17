package service

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestEnforceDictionaryDeleteRulesBlocksReferencedSubtypes(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, q := range []string{
		`CREATE TABLE contracttypes (id INTEGER PRIMARY KEY, name TEXT)`,
		`CREATE TABLE contractsubtypes (id INTEGER PRIMARY KEY, contypeid INTEGER, name TEXT)`,
		`CREATE TABLE contracts (id INTEGER PRIMARY KEY, type INTEGER, subtype INTEGER)`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`INSERT INTO contracttypes VALUES (1,'内置'),(4,'类型甲'),(5,'类型乙'),(6,'类型丙');
		INSERT INTO contractsubtypes VALUES (8,1,'内置子类型'),(9,4,'子类型甲'),(10,5,'子类型乙'),(11,6,'子类型丙');
		-- 合同选择的是“类型乙”，但子类型单独选了“类型甲”名下的子类型
		INSERT INTO contracts VALUES (100, 5, 9);`); err != nil {
		t.Fatal(err)
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	err = EnforceDictionaryDeleteRules(tx, "contractsubtypes", 9)
	if err == nil || err.Error() != "该合同子类型已被 1 条合同记录使用，无法删除" {
		t.Fatalf("subtype delete err=%v", err)
	}

	err = EnforceDictionaryDeleteRules(tx, "contracttypes", 4)
	if err == nil || err.Error() != "该合同类型下的合同子类型已被 1 条合同记录使用，无法删除" {
		t.Fatalf("contract type delete err=%v", err)
	}

	if err := EnforceDictionaryDeleteRules(tx, "contracttypes", 6); err != nil {
		t.Fatalf("unused type should pass, got %v", err)
	}

	if err := EnforceDictionaryDeleteRules(tx, "contractsubtypes", 10); err != nil {
		t.Fatalf("unused subtype should pass, got %v", err)
	}
}

// TestEnforceDictionaryRulesProtectBuiltinByName 锁定编号错位（如旧库迁移后）时内置保护仍按名称生效
func TestEnforceDictionaryRulesProtectBuiltinByName(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, q := range []string{
		`CREATE TABLE itemtypes (id INTEGER PRIMARY KEY, typedesc TEXT, hassoftware INTEGER)`,
		`CREATE TABLE items (id INTEGER PRIMARY KEY, itemtypeid INTEGER)`,
		`CREATE TABLE filetypes (id INTEGER PRIMARY KEY, typedesc TEXT)`,
		`CREATE TABLE files (id INTEGER PRIMARY KEY, type INTEGER)`,
		`CREATE TABLE contracttypes (id INTEGER PRIMARY KEY, name TEXT)`,
		`CREATE TABLE contracts (id INTEGER PRIMARY KEY, type INTEGER, subtype INTEGER)`,
		`CREATE TABLE contractsubtypes (id INTEGER PRIMARY KEY, contypeid INTEGER, name TEXT)`,
		`INSERT INTO itemtypes VALUES (9, '电话', 1), (12, '笔记本', 1)`,
		`INSERT INTO filetypes VALUES (11, '照片'), (12, '其他资料')`,
		`INSERT INTO contracttypes VALUES (2, '支持 & 维护'), (3, '租赁')`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	expectBlocked := func(dictionary string, id int64, deleteErr, updateErr string) {
		t.Helper()
		if err := EnforceDictionaryDeleteRules(tx, dictionary, id); err == nil || err.Error() != deleteErr {
			t.Fatalf("%s delete id=%d err=%v", dictionary, id, err)
		}
		if err := EnforceDictionaryUpdateRules(tx, dictionary, id); err == nil || err.Error() != updateErr {
			t.Fatalf("%s update id=%d err=%v", dictionary, id, err)
		}
	}
	expectBlocked("itemtypes", 9, "内置硬件类型不可删除", "内置硬件类型不可编辑")
	expectBlocked("filetypes", 11, "内置文件类型不可删除", "内置文件类型不可编辑")
	expectBlocked("contracttypes", 2, "内置合同类型不可删除", "内置合同类型不可编辑")

	if err := EnforceDictionaryDeleteRules(tx, "itemtypes", 12); err != nil {
		t.Fatalf("non-builtin item type should pass, got %v", err)
	}
	if err := EnforceDictionaryDeleteRules(tx, "filetypes", 12); err != nil {
		t.Fatalf("non-builtin file type should pass, got %v", err)
	}
	if err := EnforceDictionaryDeleteRules(tx, "contracttypes", 3); err != nil {
		t.Fatalf("non-builtin contract type should pass, got %v", err)
	}
}

package service

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func newContractSubtypeAuditDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec("CREATE TABLE contracttypes (id INTEGER PRIMARY KEY, name TEXT)"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO contracttypes (id, name) VALUES (5, '租赁合同')"); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestContractSubtypeAuditFormats(t *testing.T) {
	db := newContractSubtypeAuditDB(t)
	ctx := context.Background()
	meta := DictionaryAuditMeta{Table: "contractsubtypes", Label: "合同子类型", Column: "name", HasParent: true}

	if got := DictionaryTargetText(" 服务器租赁 "); got != "服务器租赁" {
		t.Fatalf("target=%q", got)
	}
	if got := DictionaryCreatedDetail(meta, "服务器租赁", "", false, DictionaryParentName(ctx, db, 5)); got != "服务器租赁(父合同: 租赁合同) 已创建" {
		t.Fatalf("created=%q", got)
	}
	if got := DictionaryDeletedDetail(meta, "服务器租赁", "", false, DictionaryParentName(ctx, db, 5)); got != "服务器租赁(父合同: 租赁合同) 已删除" {
		t.Fatalf("deleted=%q", got)
	}

	detail, changed := DictionaryUpdatedDetail(ctx, db, meta, 9, "服务器租赁", "", false, 5, map[string]interface{}{"name": "存储租赁"})
	if !changed || detail != "服务器租赁(父合同: 租赁合同) 已更新：名称已改为 存储租赁" {
		t.Fatalf("updated=%q changed=%v", detail, changed)
	}

	if _, changed := DictionaryUpdatedDetail(ctx, db, meta, 9, "服务器租赁", "", false, 5, map[string]interface{}{"name": "服务器租赁", "contypeid": int64(5)}); changed {
		t.Fatal("no-change should not be reported")
	}
}

func TestContractSubtypeTargetWithoutParentFallsBackToName(t *testing.T) {
	if got := DictionaryCreatedDetail(DictionaryAuditMeta{Table: "contractsubtypes", Label: "合同子类型", Column: "name", HasParent: true}, "孤儿子类型", "", false, ""); got != "孤儿子类型 已创建" {
		t.Fatalf("fallback created=%q", got)
	}
}

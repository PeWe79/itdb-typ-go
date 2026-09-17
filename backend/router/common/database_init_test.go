package common

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestEnsureActionsSchemaDropsEntryDate(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE actions (id INTEGER PRIMARY KEY AUTOINCREMENT,itemid INTEGER, actiondate integer, description, invoiceinfo, isauto, entrydate integer)`); err != nil {
		t.Fatalf("create actions table failed: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO actions (itemid, actiondate, description, invoiceinfo, isauto, entrydate) VALUES (1, 1700000000, '由 admin 用户创建', '成功', 1, 1700000000)`); err != nil {
		t.Fatalf("seed action row failed: %v", err)
	}

	if err := EnsureActionsSchema(db, ":memory:"); err != nil {
		t.Fatalf("EnsureActionsSchema failed: %v", err)
	}

	hasEntryDate, err := SQLiteColumnExists(db, "actions", "entrydate")
	if err != nil {
		t.Fatalf("check entrydate column failed: %v", err)
	}
	if hasEntryDate {
		t.Fatal("entrydate column should be dropped")
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM actions WHERE description = '由 admin 用户创建'`).Scan(&count); err != nil {
		t.Fatalf("query actions failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("existing action rows should survive migration, got %d", count)
	}

	if err := EnsureActionsSchema(db, ":memory:"); err != nil {
		t.Fatalf("EnsureActionsSchema should be idempotent: %v", err)
	}
}

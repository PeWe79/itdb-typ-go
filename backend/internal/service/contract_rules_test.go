package service

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestEnforceContractDeleteRulesBlocksParentReferences(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE contracts (id INTEGER PRIMARY KEY, parentid INTEGER)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO contracts VALUES (1, NULL),(2, 1),(3, 1),(4, NULL)`); err != nil {
		t.Fatal(err)
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	err = enforceContractDeleteRules(tx, 1)
	if err == nil || err.Error() != "该合同已被 2 条合同记录使用，无法删除" {
		t.Fatalf("parent referenced delete err=%v", err)
	}

	if err := enforceContractDeleteRules(tx, 4); err != nil {
		t.Fatalf("unreferenced contract should pass, got %v", err)
	}

	if err := enforceContractDeleteRules(tx, 99); err != nil {
		t.Fatalf("missing contract should pass, got %v", err)
	}
}

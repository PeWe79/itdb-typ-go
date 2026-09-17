package service

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestEnforceAgentDeleteRulesBlocksUsedTypes(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, q := range []string{
		`CREATE TABLE agents (id INTEGER PRIMARY KEY, type INTEGER, title TEXT)`,
		`CREATE TABLE items (id INTEGER PRIMARY KEY, manufacturerid INTEGER, origin TEXT)`,
		`CREATE TABLE software (id INTEGER PRIMARY KEY, manufacturerid INTEGER)`,
		`CREATE TABLE invoices (id INTEGER PRIMARY KEY, vendorid INTEGER, buyerid INTEGER)`,
		`CREATE TABLE contracts (id INTEGER PRIMARY KEY, contractorid INTEGER)`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`INSERT INTO agents VALUES (1, 8, '甲'),(2, 31, '乙'),(3, 16, '丙'),(4, 0, '丁');
		INSERT INTO items VALUES (10, 1, ''),(11, 1, ''),(12, 1, '');
		INSERT INTO invoices VALUES (20, NULL, 2),(21, NULL, 2);
		INSERT INTO contracts VALUES (30, 2);`); err != nil {
		t.Fatal(err)
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	err = enforceAgentDeleteRules(tx, 1)
	if err == nil || err.Error() != "该代理的硬件厂商类型已被 3 条硬件记录使用，无法删除" {
		t.Fatalf("hardware vendor delete err=%v", err)
	}

	err = enforceAgentDeleteRules(tx, 2)
	want := "该代理的采购方类型已被 2 条单据记录使用，无法删除\n该代理的承包方类型已被 1 条合同记录使用，无法删除"
	if err == nil || err.Error() != want {
		t.Fatalf("multi-type delete err=%v, want %q", err, want)
	}

	if err := enforceAgentDeleteRules(tx, 3); err != nil {
		t.Fatalf("unused contractor should pass, got %v", err)
	}

	if err := enforceAgentDeleteRules(tx, 4); err != nil {
		t.Fatalf("typeless agent should pass, got %v", err)
	}
}

func TestEnforceAgentTypeRemovalRulesBlocksReferencedRemovedTypes(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, q := range []string{
		`CREATE TABLE agents (id INTEGER PRIMARY KEY, type INTEGER, title TEXT)`,
		`CREATE TABLE items (id INTEGER PRIMARY KEY, manufacturerid INTEGER, origin TEXT)`,
		`CREATE TABLE invoices (id INTEGER PRIMARY KEY, vendorid INTEGER, buyerid INTEGER)`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`INSERT INTO agents VALUES (2, 5, '戊');
		INSERT INTO invoices VALUES (20, 2, NULL)`); err != nil {
		t.Fatal(err)
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	err = enforceAgentTypeRemovalRules(tx.QueryRow, 2, 1)
	if err == nil || err.Error() != "该代理的供应商类型已被 1 条单据记录使用，无法保存" {
		t.Fatalf("vendor removal err=%v", err)
	}

	if err := enforceAgentTypeRemovalRules(tx.QueryRow, 2, 5); err != nil {
		t.Fatalf("keeping types should pass, got %v", err)
	}

	if err := enforceAgentTypeRemovalRules(tx.QueryRow, 2, 4); err != nil {
		t.Fatalf("removing unused buyer should pass, got %v", err)
	}
}

func TestEnforceAgentDeleteRulesMissingAgent(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE agents (id INTEGER PRIMARY KEY, type INTEGER, title TEXT)`); err != nil {
		t.Fatal(err)
	}
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err := enforceAgentDeleteRules(tx, 99); !func() bool {
		return err != nil && err.Error() == "sql: no rows in result set"
	}() {
		t.Fatalf("missing agent err=%v", err)
	}
}

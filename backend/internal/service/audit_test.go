package service

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

const auditHistoryTestDDL = "CREATE TABLE history (id INTEGER PRIMARY KEY AUTOINCREMENT, date INTEGER, sql TEXT, authuser TEXT, ip TEXT, module TEXT DEFAULT '', action TEXT DEFAULT '', target TEXT DEFAULT '', detail TEXT DEFAULT '', result TEXT DEFAULT 'success')"

func TestAuditServiceRecordsWritesAndAppliesLimit(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE sample (id INTEGER PRIMARY KEY, value TEXT); ` + auditHistoryTestDDL); err != nil {
		t.Fatal(err)
	}
	audit := NewAuditService(db, 2)
	for i := 0; i < 4; i++ {
		if _, err := audit.Execute(context.Background(), "admin", "127.0.0.1", `INSERT INTO sample (value) VALUES (?)`, i); err != nil {
			t.Fatal(err)
		}
	}
	var writes, history int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sample`).Scan(&writes); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM history`).Scan(&history); err != nil {
		t.Fatal(err)
	}
	if writes != 4 || history != 3 {
		t.Fatalf("writes=%d history=%d", writes, history)
	}
}

func TestAuditServiceRecordAppliesLimit(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE sample (id INTEGER PRIMARY KEY, value TEXT); ` + auditHistoryTestDDL); err != nil {
		t.Fatal(err)
	}
	audit := NewAuditService(db, 1)
	for i := 0; i < 3; i++ {
		if err := audit.Record(context.Background(), "admin", "127.0.0.1", `UPDATE sample SET value = ? WHERE id = ?`, i, i); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM history`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("history=%d", count)
	}
}

func TestClassifyAuditSQL(t *testing.T) {
	cases := []struct {
		name         string
		username     string
		query        string
		args         []interface{}
		expectOK     bool
		expectModule string
		expectAction string
		expectTarget string
	}{
		{name: "asset insert with stub columns", query: "INSERT INTO items (...)", args: []interface{}{1, "model-a"}, expectOK: true, expectModule: AuditModuleAssets, expectAction: "新增资产", expectTarget: "-"},
		{name: "agent insert derives title", query: "INSERT INTO agents (type, title, contactinfo, contacts, urls) VALUES (?, ?, ?, ?, ?)", args: []interface{}{1, "华为", "", "", ""}, expectOK: true, expectModule: AuditModuleAssets, expectAction: "新增厂商", expectTarget: "华为"},
		{name: "tag update", query: "UPDATE tags SET name = ? WHERE id = ?", args: []interface{}{"服务器", 7}, expectOK: true, expectModule: AuditModuleCatalog, expectAction: "更新标记", expectTarget: "#7"},
		{name: "user delete merges into settings module", query: "DELETE FROM users WHERE id = ?", args: []interface{}{12}, expectOK: true, expectModule: AuditModuleSettings, expectAction: "删除用户", expectTarget: "#12"},
		{name: "password only change", query: "UPDATE users SET pass=? WHERE id=?", args: []interface{}{"hash", 3}, expectOK: true, expectModule: AuditModuleAuth, expectAction: "修改密码", expectTarget: "#3"},
		{name: "password reset actor", username: "password-reset", query: "UPDATE users SET pass=? WHERE id=?", args: []interface{}{"hash", 3}, expectOK: true, expectModule: AuditModuleAuth, expectAction: "重置密码", expectTarget: "#3"},
		{name: "user profile edit is not password change", query: "UPDATE users SET username = ?, userdesc = ?, pass = ?, usertype = ? WHERE id = ?", args: []interface{}{"alice", "Alice", "hash", 1, 3}, expectOK: true, expectModule: AuditModuleSettings, expectAction: "更新用户", expectTarget: "#3"},
		{name: "auth provider save", query: "UPDATE settings_auth_providers SET config = ? WHERE id = ?", args: []interface{}{"{}", "ldap"}, expectOK: true, expectModule: AuditModuleSettings, expectAction: "更新认证配置", expectTarget: "#ldap"},
		{name: "audit table skipped", query: "INSERT INTO history (date, sql) VALUES (?, ?)", args: []interface{}{1, "x"}, expectOK: false},
		{name: "view history skipped", query: "INSERT INTO viewhist (url, description) VALUES (?, ?)", args: []interface{}{"/", "仪表盘"}, expectOK: false},
		{name: "select skipped", query: "SELECT * FROM items WHERE id = ?", args: []interface{}{1}, expectOK: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			event, ok := ClassifyAuditSQL(tc.username, tc.query, tc.args)
			if ok != tc.expectOK {
				t.Fatalf("ok=%v want %v", ok, tc.expectOK)
			}
			if !ok {
				return
			}
			if event.Module != tc.expectModule || event.Action != tc.expectAction || event.Target != tc.expectTarget {
				t.Fatalf("event=%+v want module=%s action=%s target=%s", event, tc.expectModule, tc.expectAction, tc.expectTarget)
			}
		})
	}
}

func TestAuditServiceExecuteCapturesInsertTarget(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE agents (id INTEGER PRIMARY KEY AUTOINCREMENT, type INTEGER, title TEXT); ` + auditHistoryTestDDL); err != nil {
		t.Fatal(err)
	}
	audit := NewAuditService(db, 0)
	if _, err := audit.Execute(context.Background(), "admin", "127.0.0.1", `INSERT INTO agents (type, title) VALUES (?, ?)`, 1, "联想"); err != nil {
		t.Fatal(err)
	}
	var module, action, target, result string
	if err := db.QueryRow(`SELECT module, action, target, result FROM history LIMIT 1`).Scan(&module, &action, &target, &result); err != nil {
		t.Fatal(err)
	}
	if module != AuditModuleAssets || action != "新增厂商" || target != "联想" || result != AuditResultSuccess {
		t.Fatalf("module=%s action=%s target=%s result=%s", module, action, target, result)
	}
}

func TestAuditServiceExecuteRecordsFailure(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(auditHistoryTestDDL); err != nil {
		t.Fatal(err)
	}
	audit := NewAuditService(db, 0)
	if _, err := audit.Execute(context.Background(), "admin", "127.0.0.1", `UPDATE missing_table SET value = 1`); err == nil {
		t.Fatal("expected business error")
	}
	var action, result, detail string
	if err := db.QueryRow(`SELECT action, result, detail FROM history LIMIT 1`).Scan(&action, &result, &detail); err != nil {
		t.Fatal(err)
	}
	if action != "更新missing_table" || result != AuditResultFailure || detail == "" {
		t.Fatalf("action=%s result=%s detail=%q", action, result, detail)
	}
}

func TestAuditServiceRecordEvent(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(auditHistoryTestDDL); err != nil {
		t.Fatal(err)
	}
	audit := NewAuditService(db, 0)
	event := AuditEvent{Module: AuditModuleAuth, Action: "用户登录", Target: "admin", Detail: "登录成功（本地账号）", Result: AuditResultSuccess}
	if err := audit.RecordEvent(context.Background(), "admin", "10.0.0.1", event); err != nil {
		t.Fatal(err)
	}
	var module, action, target, sqlText, result string
	if err := db.QueryRow(`SELECT module, action, target, sql, result FROM history LIMIT 1`).Scan(&module, &action, &target, &sqlText, &result); err != nil {
		t.Fatal(err)
	}
	if module != AuditModuleAuth || action != "用户登录" || target != "admin" || sqlText != "" || result != AuditResultSuccess {
		t.Fatalf("module=%s action=%s target=%s sql=%q result=%s", module, action, target, sqlText, result)
	}
}

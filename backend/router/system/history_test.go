package system

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"itdb-backend/internal/repository"
	"itdb-backend/internal/service"

	_ "modernc.org/sqlite"
)

// TestAuditHistoryHandler 校验审计日志接口的结构化响应与历史数据回填。
func TestAuditHistoryHandler(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("CREATE TABLE history (id INTEGER PRIMARY KEY AUTOINCREMENT, date INTEGER, sql TEXT, authuser TEXT, ip TEXT, module TEXT DEFAULT '', action TEXT DEFAULT '', target TEXT DEFAULT '', detail TEXT DEFAULT '', result TEXT DEFAULT 'success')"); err != nil {
		t.Fatal(err)
	}
	seed := []string{
		"INSERT INTO history (date, sql, authuser, ip, module, action, target, detail, result) VALUES (1700000002, '', 'admin', '10.0.0.2', 'auth', '用户登录', 'admin', '登录成功（本地账号）', 'success')",
		"INSERT INTO history (date, sql, authuser, ip, module, action, target, detail, result) VALUES (1700000001, 'UPDATE tags SET name = ? WHERE id = ? | args=[生产 5]', 'alice', '10.0.0.1', 'catalog', '更新标记', '#5', 'UPDATE tags SET name = ? WHERE id = ? | args=[生产 5]', 'success')",
		"INSERT INTO history (date, sql, authuser, ip) VALUES (1700000000, 'DELETE FROM users WHERE id = ? | args=[9]', 'bob', '10.0.0.3')",
		"INSERT INTO history (date, sql, authuser, ip, module, action, target, detail, result) VALUES (1699999999, 'UPDATE missing SET v = 1', 'admin', '10.0.0.2', 'assets', '更新资产', '-', 'UPDATE missing SET v = 1 | error: no such table', 'failure')",
	}
	for _, query := range seed {
		if _, err := db.Exec(query); err != nil {
			t.Fatal(err)
		}
	}

	app := &Router{db: db, domains: service.NewDomainServices(repository.NewStore(db))}
	rec := httptest.NewRecorder()
	app.handleHistory(rec, httptest.NewRequest(http.MethodGet, "/api/history?limit=500", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Items []struct {
			ID        int64  `json:"id"`
			Timestamp int64  `json:"timestamp"`
			Username  string `json:"username"`
			Module    string `json:"module"`
			Action    string `json:"action"`
			Target    string `json:"target"`
			IP        string `json:"ip"`
			Result    string `json:"result"`
			Detail    string `json:"detail"`
		} `json:"items"`
		Total int64 `json:"total"`
		Limit int   `json:"limit"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Total != 4 || payload.Limit != 500 || len(payload.Items) != 4 {
		t.Fatalf("total=%d limit=%d items=%d", payload.Total, payload.Limit, len(payload.Items))
	}
	newest := payload.Items[0]
	if newest.Action != "更新资产" || newest.Result != "failure" || newest.Target != "-" {
		t.Fatalf("newest=%+v", newest)
	}
	legacy := payload.Items[1]
	if legacy.Module != "settings" || legacy.Action != "删除用户" || legacy.Result != "success" {
		t.Fatalf("legacy=%+v", legacy)
	}
	login := payload.Items[3]
	if login.Action != "用户登录" || login.Module != "auth" || login.Target != "admin" {
		t.Fatalf("login=%+v", login)
	}
}

// TestAuditHistoryClearHandler 校验清空接口按编号删除、去重与清空后补记审计事件。
func TestAuditHistoryClearHandler(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("CREATE TABLE history (id INTEGER PRIMARY KEY AUTOINCREMENT, date INTEGER, sql TEXT, authuser TEXT, ip TEXT, module TEXT DEFAULT '', action TEXT DEFAULT '', target TEXT DEFAULT '', detail TEXT DEFAULT '', result TEXT DEFAULT 'success')"); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 5; i++ {
		if _, err := db.Exec(
			"INSERT INTO history (date, authuser, module, action, result) VALUES (?, 'admin', 'auth', '用户登录', 'success')", i,
		); err != nil {
			t.Fatal(err)
		}
	}

	app := &Router{
		db:      db,
		domains: service.NewDomainServices(repository.NewStore(db)),
		audit:   service.NewAuditService(db, 0),
	}
	body := []byte(`{"ids":[5,2,2,0,-1,5]}`)
	rec := httptest.NewRecorder()
	app.handleHistoryClear(rec, httptest.NewRequest(http.MethodPost, "/api/history/clear", bytes.NewReader(body)))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Deleted int64 `json:"deleted"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Deleted != 2 {
		t.Fatalf("deleted=%d", payload.Deleted)
	}
	var remain int
	if err := db.QueryRow("SELECT COUNT(*) FROM history").Scan(&remain); err != nil {
		t.Fatal(err)
	}
	if remain != 4 {
		t.Fatalf("remain=%d", remain)
	}
	var kept int
	if err := db.QueryRow("SELECT COUNT(*) FROM history WHERE id IN (1,3,4)").Scan(&kept); err != nil {
		t.Fatal(err)
	}
	if kept != 3 {
		t.Fatalf("kept=%d", kept)
	}
	var clearEvent int
	if err := db.QueryRow("SELECT COUNT(*) FROM history WHERE module = 'audit' AND action = '清空审计日志' AND target = '2 条' AND result = 'success'").Scan(&clearEvent); err != nil {
		t.Fatal(err)
	}
	if clearEvent != 1 {
		t.Fatalf("clearEvent=%d", clearEvent)
	}

	emptyRec := httptest.NewRecorder()
	app.handleHistoryClear(emptyRec, httptest.NewRequest(http.MethodPost, "/api/history/clear", bytes.NewReader([]byte(`{"ids":[]}`))))
	if emptyRec.Code != http.StatusBadRequest {
		t.Fatalf("empty status=%d body=%s", emptyRec.Code, emptyRec.Body.String())
	}
}

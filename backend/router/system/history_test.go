package system

import (
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

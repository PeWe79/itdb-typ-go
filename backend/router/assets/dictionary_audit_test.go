package assets

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"itdb-backend/internal/domain"
	"itdb-backend/internal/repository"
	"itdb-backend/internal/service"
	"itdb-backend/router/common"

	"github.com/go-chi/chi/v5"
	_ "modernc.org/sqlite"
)

// newDictionaryAuditTestApp 构造最小可用的资产域路由：只建字典与审计相关表，
// 当前用户持有全部字典管理权限，用于锁定字典写入的审计单条性。
func newDictionaryAuditTestApp(t *testing.T) *Router {
	t.Helper()
	db, err := sql.Open("sqlite", "file:dictionary-audit-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	for _, ddl := range []string{
		"CREATE TABLE dpttypes (id INTEGER PRIMARY KEY AUTOINCREMENT, dptname TEXT)",
		"CREATE TABLE history (id INTEGER PRIMARY KEY AUTOINCREMENT, date INTEGER, sql TEXT, authuser TEXT, ip TEXT, module TEXT DEFAULT '', action TEXT DEFAULT '', target TEXT DEFAULT '', detail TEXT DEFAULT '', result TEXT DEFAULT 'success')",
	} {
		if _, err := db.Exec(ddl); err != nil {
			t.Fatal(err)
		}
	}
	return &Router{
		db:      db,
		domains: service.NewDomainServices(repository.NewStore(db)),
		audit:   service.NewAuditService(db, 0),
		currentUserPermissions: func(*http.Request) []string {
			return []string{"dictionaries.dpttypes.manage"}
		},
	}
}

// TestDictionaryCreateWritesSingleAuditEvent 锁定字典创建只写一条中文显式审计，
// 防止 SQL 分类器与显式事件双写回归
func TestDictionaryCreateWritesSingleAuditEvent(t *testing.T) {
	app := newDictionaryAuditTestApp(t)
	userContext := common.WithUser(context.Background(), domain.SessionUser{ID: 1, Username: "admin"})
	request := httptest.NewRequest(http.MethodPost, "/api/dictionaries/dpttypes", bytes.NewReader([]byte(`{"dptname":"信息运维部"}`))).WithContext(userContext)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("name", "dpttypes")
	request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
	response := httptest.NewRecorder()
	app.handleCreateDictionaryRow(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body=%s", response.Code, response.Body.String())
	}

	var total int
	var action, target, detail string
	if err := app.db.QueryRow("SELECT COUNT(*) FROM history").Scan(&total); err != nil {
		t.Fatal(err)
	}
	if err := app.db.QueryRow("SELECT action, target, detail FROM history ORDER BY id DESC LIMIT 1").Scan(&action, &target, &detail); err != nil {
		t.Fatal(err)
	}
	if total != 1 || action != "新增部门" || target != "信息运维部" || detail != "信息运维部 已创建" {
		t.Fatalf("audit rows=%d action=%q target=%q detail=%q", total, action, target, detail)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["id"].(float64) <= 0 {
		t.Fatalf("created id missing: %v", payload)
	}
}

// TestTrackAuditEventImportAllExisting 锁定“导入全部已存在”写一条失败审计，
// 多条时详情为“均已存在”，单条时去掉“均”字
func TestTrackAuditEventImportAllExisting(t *testing.T) {
	app := newDictionaryAuditTestApp(t)
	userContext := common.WithUser(context.Background(), domain.SessionUser{ID: 1, Username: "admin"})
	body := `{"type":"import:dpttypes","names":["服务器","存储","交换机"],"existing":["服务器","存储","交换机"],"result":"failure"}`
	request := httptest.NewRequest(http.MethodPost, "/api/history/events", bytes.NewReader([]byte(body))).WithContext(userContext)
	response := httptest.NewRecorder()
	app.handleTrackAuditEvent(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("track status = %d, body=%s", response.Code, response.Body.String())
	}

	var total int
	var action, target, detail, result string
	if err := app.db.QueryRow("SELECT COUNT(*) FROM history").Scan(&total); err != nil {
		t.Fatal(err)
	}
	if err := app.db.QueryRow("SELECT action, target, detail, result FROM history ORDER BY id DESC LIMIT 1").Scan(&action, &target, &detail, &result); err != nil {
		t.Fatal(err)
	}
	expectDetail := "服务器、存储、交换机 均已存在"
	if total != 1 || action != "导入部门" || target != "服务器、存储、交换机" || detail != expectDetail || result != "failure" {
		t.Fatalf("audit rows=%d action=%q target=%q detail=%q result=%q", total, action, target, detail, result)
	}

	single := httptest.NewRequest(http.MethodPost, "/api/history/events", bytes.NewReader([]byte(`{"type":"import:dpttypes","names":["服务器"],"existing":["服务器"],"result":"failure"}`))).WithContext(userContext)
	singleResponse := httptest.NewRecorder()
	app.handleTrackAuditEvent(singleResponse, single)
	if singleResponse.Code != http.StatusOK {
		t.Fatalf("single track status = %d", singleResponse.Code)
	}
	if err := app.db.QueryRow("SELECT detail FROM history ORDER BY id DESC LIMIT 1").Scan(&detail); err != nil {
		t.Fatal(err)
	}
	if detail != "服务器 已存在" {
		t.Fatalf("single detail=%q, want 服务器 已存在", detail)
	}
}

// newMaintenanceExportTestApp 构造最小可用的资产域路由：建硬件、厂商、硬件类型与审计表，
// 当前用户持有硬件查看权限，用于锁定维护日志导出的审计格式
func newMaintenanceExportTestApp(t *testing.T) *Router {
	t.Helper()
	db, err := sql.Open("sqlite", "file:maintenance-export-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	for _, ddl := range []string{
		"CREATE TABLE agents (id INTEGER PRIMARY KEY AUTOINCREMENT, title TEXT)",
		"CREATE TABLE itemtypes (id INTEGER PRIMARY KEY AUTOINCREMENT, typedesc TEXT)",
		"CREATE TABLE items (id INTEGER PRIMARY KEY AUTOINCREMENT, label TEXT, model TEXT, manufacturerid INTEGER, itemtypeid INTEGER)",
		"CREATE TABLE history (id INTEGER PRIMARY KEY AUTOINCREMENT, date INTEGER, sql TEXT, authuser TEXT, ip TEXT, module TEXT DEFAULT '', action TEXT DEFAULT '', target TEXT DEFAULT '', detail TEXT DEFAULT '', result TEXT DEFAULT 'success')",
	} {
		if _, err := db.Exec(ddl); err != nil {
			t.Fatal(err)
		}
	}
	return &Router{
		db:      db,
		domains: service.NewDomainServices(repository.NewStore(db)),
		audit:   service.NewAuditService(db, 0),
		currentUserPermissions: func(*http.Request) []string {
			return []string{"assets.items.read"}
		},
	}
}

// TestTrackAuditEventMaintenanceLogExport 锁定硬件维护日志导出审计：操作为「导出硬件维护日志」，
// 目标为导出行数，详情为「<厂商> <型号>[<硬件类型>, ID: <编号>] 维护日志所有数据已导出」（不含标签前缀）
func TestTrackAuditEventMaintenanceLogExport(t *testing.T) {
	app := newMaintenanceExportTestApp(t)
	for _, seed := range []string{
		"INSERT INTO agents (id, title) VALUES (1, '联想')",
		"INSERT INTO itemtypes (id, typedesc) VALUES (1, '笔记本')",
		"INSERT INTO items (id, label, model, manufacturerid, itemtypeid) VALUES (2, 'TL-001', 'ThinkPad', 1, 1)",
	} {
		if _, err := app.db.Exec(seed); err != nil {
			t.Fatal(err)
		}
	}
	userContext := common.WithUser(context.Background(), domain.SessionUser{ID: 1, Username: "admin"})
	request := httptest.NewRequest(http.MethodPost, "/api/history/events",
		bytes.NewReader([]byte(`{"type":"export:items-maintenance","target":"2","count":5}`))).WithContext(userContext)
	response := httptest.NewRecorder()
	app.handleTrackAuditEvent(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("track status = %d, body=%s", response.Code, response.Body.String())
	}

	var total int
	var action, target, detail, result string
	if err := app.db.QueryRow("SELECT COUNT(*) FROM history").Scan(&total); err != nil {
		t.Fatal(err)
	}
	if err := app.db.QueryRow("SELECT action, target, detail, result FROM history ORDER BY id DESC LIMIT 1").Scan(&action, &target, &detail, &result); err != nil {
		t.Fatal(err)
	}
	expectDetail := "联想 ThinkPad[笔记本, ID: 2] 维护日志所有数据已导出"
	if total != 1 || action != "导出硬件维护日志" || target != "5 条" || detail != expectDetail || result != "success" {
		t.Fatalf("audit rows=%d action=%q target=%q detail=%q result=%q", total, action, target, detail, result)
	}

	missing := httptest.NewRequest(http.MethodPost, "/api/history/events",
		bytes.NewReader([]byte(`{"type":"export:items-maintenance","target":"999","count":5}`))).WithContext(userContext)
	missingResponse := httptest.NewRecorder()
	app.handleTrackAuditEvent(missingResponse, missing)
	if missingResponse.Code != http.StatusBadRequest {
		t.Fatalf("missing track status = %d", missingResponse.Code)
	}
	if err := app.db.QueryRow("SELECT COUNT(*) FROM history").Scan(&total); err != nil {
		t.Fatal(err)
	}
	if total != 1 {
		t.Fatalf("audit rows after invalid report = %d, want 1", total)
	}
}

// TestTrackAuditEventImportTargetCoversExisting 锁定部分成功场景的目标包含已导入与已存在全部名称
func TestTrackAuditEventImportTargetCoversExisting(t *testing.T) {
	app := newDictionaryAuditTestApp(t)
	userContext := common.WithUser(context.Background(), domain.SessionUser{ID: 1, Username: "admin"})
	body := `{"type":"import:dpttypes","names":["信息运维部1"],"existing":["信息运维部"]}`
	request := httptest.NewRequest(http.MethodPost, "/api/history/events", bytes.NewReader([]byte(body))).WithContext(userContext)
	response := httptest.NewRecorder()
	app.handleTrackAuditEvent(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("track status = %d, body=%s", response.Code, response.Body.String())
	}

	var total int
	var target, detail, result string
	if err := app.db.QueryRow("SELECT COUNT(*) FROM history").Scan(&total); err != nil {
		t.Fatal(err)
	}
	if err := app.db.QueryRow("SELECT target, detail, result FROM history ORDER BY id DESC LIMIT 1").Scan(&target, &detail, &result); err != nil {
		t.Fatal(err)
	}
	expectTarget := "信息运维部1、信息运维部"
	expectDetail := "信息运维部1 已导入，其中 信息运维部 已存在"
	if total != 1 || target != expectTarget || detail != expectDetail || result != "success" {
		t.Fatalf("audit rows=%d target=%q detail=%q result=%q", total, target, detail, result)
	}
}

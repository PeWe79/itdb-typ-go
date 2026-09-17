package settings

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"itdb-backend/internal/service"
	_ "modernc.org/sqlite"
)

func newSettingsUsersTestApp(t *testing.T) *Router {
	t.Helper()
	db, err := sql.Open("sqlite", "file:settings-users-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT, userdesc TEXT, pass TEXT, cookie1 TEXT, usertype INTEGER)"); err != nil {
		t.Fatal(err)
	}
	if err := EnsureResourceSchema(db); err != nil {
		t.Fatal(err)
	}
	if err := EnsureUserProfileLoginSchema(db, ":memory:"); err != nil {
		t.Fatal(err)
	}
	return &Router{db: db}
}

func TestSettingsUserDisabledReturnsFullUser(t *testing.T) {
	app := newSettingsUsersTestApp(t)
	now := time.Now().Unix()
	if _, err := app.db.Exec("INSERT INTO users(id,username,userdesc,usertype) VALUES(2,'alice','Alice',1)"); err != nil {
		t.Fatal(err)
	}
	if _, err := app.db.Exec("INSERT INTO settings_user_profiles(user_id,email,source,created_at,updated_at,last_login_at) VALUES(2,'alice@example.com','local',?, ?, ?)", now, now, now); err != nil {
		t.Fatal(err)
	}
	payload := []byte(`{"disabled":true}`)
	response := httptest.NewRecorder()
	request := settingsUserDisabledRequest(2, payload)
	app.handleSettingsUserDisabled(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("disabled status = %d, body=%s", response.Code, response.Body.String())
	}
	var user map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &user); err != nil {
		t.Fatal(err)
	}
	if user["disabled"] != true {
		t.Fatalf("disabled flag mismatch: %v", user)
	}
	if _, ok := user["directRoles"]; !ok {
		t.Fatalf("directRoles missing, frontend expects array: %v", user)
	}
	if _, ok := user["permissions"]; !ok {
		t.Fatalf("permissions missing: %v", user)
	}
	createdAt, _ := user["createdAt"].(string)
	if createdAt == "" {
		t.Fatalf("createdAt missing: %v", user)
	}
	lastLoginAt, _ := user["lastLoginAt"].(string)
	if lastLoginAt == "" {
		t.Fatalf("lastLoginAt missing: %v", user)
	}
}

func TestSettingsUserDisabledProtectsAdminAndSelf(t *testing.T) {
	app := newSettingsUsersTestApp(t)
	if _, err := app.db.Exec("INSERT INTO users(id,username,userdesc,usertype) VALUES(1,'admin','Administrator',0)"); err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	request := settingsUserDisabledRequest(1, []byte(`{"disabled":true}`))
	app.handleSettingsUserDisabled(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("admin disabled status = %d, want 400", response.Code)
	}
}

// settingsUserDisabledRequest 构造带 chi 路由参数的禁用请求。
func settingsUserDisabledRequest(id int64, payload []byte) *http.Request {
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("id", strconv.FormatInt(id, 10))
	request := httptest.NewRequest(http.MethodPost, "/api/settings/users/{id}/disabled", bytes.NewReader(payload))
	return request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeCtx))
}

func TestSettingsUserListReturnsTimestamps(t *testing.T) {
	app := newSettingsUsersTestApp(t)
	now := time.Now().Unix()
	if _, err := app.db.Exec("INSERT INTO users(id,username,userdesc,usertype) VALUES(3,'bob','Bob',1)"); err != nil {
		t.Fatal(err)
	}
	if _, err := app.db.Exec("INSERT INTO settings_user_profiles(user_id,email,source,created_at,updated_at,last_login_at) VALUES(3,'bob@example.com','local',?,?,?)", now, now, now); err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	app.handleSettingsUsers(response, httptest.NewRequest(http.MethodGet, "/api/settings/users", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("list status = %d", response.Code)
	}
	var body struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 1 {
		t.Fatalf("items = %d", len(body.Items))
	}
	if body.Items[0]["createdAt"] == "" || body.Items[0]["lastLoginAt"] == "" {
		t.Fatalf("timestamps missing: %v", body.Items[0])
	}
}

func TestSettingsUserCreateRejectsDuplicateUsername(t *testing.T) {
	app := newSettingsUsersTestApp(t)
	app.audit = service.NewAuditService(app.db, 100)
	if _, err := app.db.Exec("INSERT INTO users(id,username,userdesc,usertype) VALUES(2,'alice','Alice',1)"); err != nil {
		t.Fatal(err)
	}
	create := func(username string) *httptest.ResponseRecorder {
		payload, _ := json.Marshal(map[string]any{
			"username": username, "password": "123456", "email": "alice@example.com",
			"displayName": "Alice", "roleKeys": []string{"viewer"},
		})
		response := httptest.NewRecorder()
		app.handleSettingsUsers(response, httptest.NewRequest(http.MethodPost, "/api/settings/users", bytes.NewReader(payload)))
		return response
	}
	for _, username := range []string{"alice", "ALICE"} {
		response := create(username)
		if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), "用户名已存在") {
			t.Fatalf("create %s status = %d, body=%s", username, response.Code, response.Body.String())
		}
	}
}

func TestSettingsUserProfileLoginSchemaBackfillsProfiles(t *testing.T) {
	db, err := sql.Open("sqlite", "file:user-profile-schema-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT, userdesc TEXT)"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO users(id,username) VALUES(9,'legacy')"); err != nil {
		t.Fatal(err)
	}
	if err := EnsureResourceSchema(db); err != nil {
		t.Fatal(err)
	}
	if err := EnsureUserProfileLoginSchema(db, ":memory:"); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM settings_user_profiles WHERE user_id=9").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("legacy profile rows = %d, want 1", count)
	}
	var hasColumn int
	if err := db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('settings_user_profiles') WHERE name='last_login_at'").Scan(&hasColumn); err != nil {
		t.Fatal(err)
	}
	if hasColumn != 1 {
		t.Fatalf("last_login_at column missing")
	}
}

// TestSettingsUserProfileLoginSchemaAltersLegacyTable 模拟存量库的无列表结构，验证迁移补列。
func TestSettingsUserProfileLoginSchemaAltersLegacyTable(t *testing.T) {
	db, err := sql.Open("sqlite", "file:user-profile-legacy-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("CREATE TABLE settings_user_profiles (user_id INTEGER PRIMARY KEY, email TEXT NOT NULL DEFAULT '', disabled INTEGER NOT NULL DEFAULT 0, source TEXT NOT NULL DEFAULT 'local', created_at INTEGER NOT NULL DEFAULT 0, updated_at INTEGER NOT NULL DEFAULT 0)"); err != nil {
		t.Fatal(err)
	}
	if err := EnsureUserProfileLoginSchema(db, ":memory:"); err != nil {
		t.Fatal(err)
	}
	var hasColumn int
	if err := db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('settings_user_profiles') WHERE name='last_login_at'").Scan(&hasColumn); err != nil {
		t.Fatal(err)
	}
	if hasColumn != 1 {
		t.Fatalf("last_login_at column missing after alter")
	}
}

func TestRecordSettingsUserLogin(t *testing.T) {
	app := newSettingsUsersTestApp(t)
	if _, err := app.db.Exec("INSERT INTO users(id,username,userdesc,usertype) VALUES(4,'carol','Carol',1)"); err != nil {
		t.Fatal(err)
	}
	if err := RecordSettingsUserLogin(context.Background(), app.db, 4); err != nil {
		t.Fatal(err)
	}
	var lastLoginAt int64
	if err := app.db.QueryRow("SELECT last_login_at FROM settings_user_profiles WHERE user_id=4").Scan(&lastLoginAt); err != nil {
		t.Fatal(err)
	}
	if lastLoginAt <= 0 {
		t.Fatalf("last_login_at = %d", lastLoginAt)
	}
	if err := RecordSettingsUserLogin(context.Background(), app.db, 4); err != nil {
		t.Fatal(err)
	}
}

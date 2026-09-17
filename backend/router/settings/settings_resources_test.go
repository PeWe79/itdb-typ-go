package settings

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"itdb-backend/internal/repository"
	"itdb-backend/internal/service"
	_ "modernc.org/sqlite"
)

func TestEnsureSettingsResourceSchema(t *testing.T) {
	db, err := sql.Open("sqlite", "file:settings-resource-schema?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := EnsureResourceSchema(db); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"settings_user_profiles", "settings_roles", "settings_user_groups", "settings_auth_providers", "settings_email"} {
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("table %s was not created", table)
		}
	}
	var roleCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM settings_roles").Scan(&roleCount); err != nil {
		t.Fatal(err)
	}
	if roleCount != 3 {
		t.Fatalf("expected 3 built-in roles, got %d", roleCount)
	}
	for _, key := range []string{"admin", "operator", "viewer"} {
		var name string
		if err := db.QueryRow("SELECT name FROM settings_roles WHERE key=?", key).Scan(&name); err != nil {
			t.Fatal(err)
		}
		if name != key {
			t.Fatalf("built-in role %s name = %s, want English key", key, name)
		}
	}
}

func TestSettingsUserAccessUsesEffectiveRoles(t *testing.T) {
	db, err := sql.Open("sqlite", "file:settings-user-access?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, username TEXT, userdesc TEXT, pass TEXT, usertype INTEGER)"); err != nil {
		t.Fatal(err)
	}
	if err := EnsureResourceSchema(db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO users(id,username,userdesc,usertype) VALUES(1,'alice','Alice',1)"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO settings_roles(key,name,permissions,builtin) VALUES('direct','Direct','[\"assets.items.read\"]',0),('group','Group','[\"settings.users.read\"]',0),('disabled','Disabled','[\"assets.items.manage\"]',0)"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO settings_user_roles(user_id,role_id) SELECT 1,id FROM settings_roles WHERE key IN ('direct','disabled')"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO settings_user_groups(id,name) VALUES(1,'Operators')"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO settings_user_group_members(group_id,user_id) VALUES(1,1)"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO settings_user_group_roles(group_id,role_id) SELECT 1,id FROM settings_roles WHERE key='group'"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO settings_role_status(role_id,disabled) SELECT id,1 FROM settings_roles WHERE key='disabled'"); err != nil {
		t.Fatal(err)
	}

	direct, effective, permissions, err := UserAccess(db, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(direct) != 2 || len(effective) != 2 {
		t.Fatalf("direct/effective role counts = %d/%d, want 2/2", len(direct), len(effective))
	}
	if containsString(permissions, "assets.items.manage") || !containsString(permissions, "assets.items.read") || !containsString(permissions, "settings.users.read") {
		t.Fatalf("unexpected effective permissions: %#v", permissions)
	}
}

func TestSettingsRelationsRejectUnknownReferencesWithoutChangingExisting(t *testing.T) {
	db, err := sql.Open("sqlite", "file:settings-relation-validation?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, username TEXT, userdesc TEXT, pass TEXT, usertype INTEGER)"); err != nil {
		t.Fatal(err)
	}
	if err := EnsureResourceSchema(db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO users(id,username) VALUES(1,'alice')"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO settings_user_groups(id,name) VALUES(1,'ops')"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO settings_roles(key,name,permissions) VALUES('reader','Reader','[]'),('disabled','Disabled','[]')"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO settings_user_group_members(group_id,user_id) VALUES(1,1)"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO settings_user_group_roles(group_id,role_id) SELECT 1,id FROM settings_roles WHERE key='reader'"); err != nil {
		t.Fatal(err)
	}

	if err := replaceSettingsGroupRelations(db, 1, []string{"999"}, []string{"reader"}); err == nil || err.Error() != "成员用户不存在" {
		t.Fatalf("unknown member error = %v", err)
	}
	assertRelationCounts(t, db, 1, 1)
	if err := replaceSettingsGroupRelations(db, 1, []string{"1"}, []string{"missing"}); err == nil || err.Error() != "选择的角色不存在" {
		t.Fatalf("unknown role error = %v", err)
	}
	assertRelationCounts(t, db, 1, 1)
	if _, err := db.Exec("INSERT INTO settings_role_status(role_id,disabled) SELECT id,1 FROM settings_roles WHERE key='disabled'"); err != nil {
		t.Fatal(err)
	}
	if err := replaceSettingsGroupRelations(db, 1, []string{"1"}, []string{"disabled"}); err == nil || err.Error() != "角色已禁用，不能分配" {
		t.Fatalf("disabled role error = %v", err)
	}
	assertRelationCounts(t, db, 1, 1)
}

// settingsResourceDeleteRequest 构造带 chi 路由参数、无请求体的 DELETE 请求。
func settingsResourceDeleteRequest(path, id string) *http.Request {
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("id", id)
	request := httptest.NewRequest(http.MethodDelete, path, nil)
	return request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeCtx))
}

// TestSettingsResourceDeleteWithoutBody 三个模块的删除均无请求体，不应被 JSON 解码拦截。
func TestSettingsResourceDeleteWithoutBody(t *testing.T) {
	app := newSettingsUsersTestApp(t)
	app.audit = service.NewAuditService(app.db, 100)
	app.userWorkflow = service.NewUserWorkflow(repository.NewUserRepository(repository.NewStore(app.db)), app.audit)
	now := time.Now().Unix()
	if _, err := app.db.Exec("CREATE TABLE items (id INTEGER PRIMARY KEY AUTOINCREMENT, userid INTEGER)"); err != nil {
		t.Fatal(err)
	}
	if _, err := app.db.Exec("CREATE TABLE contracts (id INTEGER PRIMARY KEY AUTOINCREMENT, renewals TEXT)"); err != nil {
		t.Fatal(err)
	}
	if _, err := app.db.Exec("CREATE TABLE history (id INTEGER PRIMARY KEY AUTOINCREMENT, date INTEGER, sql TEXT, authuser TEXT, ip TEXT, module TEXT DEFAULT '', action TEXT DEFAULT '', target TEXT DEFAULT '', detail TEXT DEFAULT '', result TEXT DEFAULT 'success')"); err != nil {
		t.Fatal(err)
	}
	if _, err := app.db.Exec("INSERT INTO users(id,username,userdesc,pass,usertype) VALUES(1,'admin','Administrator','x',0)"); err != nil {
		t.Fatal(err)
	}
	if _, err := app.db.Exec("INSERT INTO users(id,username,userdesc,usertype) VALUES(2,'alice','Alice',1)"); err != nil {
		t.Fatal(err)
	}
	if _, err := app.db.Exec("INSERT INTO settings_user_profiles(user_id,email,disabled,source,created_at,updated_at) VALUES(2,'alice@example.com',1,'local',?,?)", now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := app.db.Exec("INSERT INTO users(id,username,userdesc,usertype) VALUES(3,'bob','Bob',1)"); err != nil {
		t.Fatal(err)
	}
	if _, err := app.db.Exec("INSERT INTO settings_user_profiles(user_id,email,disabled,source,created_at,updated_at) VALUES(3,'bob@example.com',0,'local',?,?)", now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := app.db.Exec("INSERT INTO settings_roles(key,name,permissions,builtin) VALUES('auditor','Auditor','[]',0)"); err != nil {
		t.Fatal(err)
	}
	if _, err := app.db.Exec("INSERT INTO settings_user_groups(name) VALUES('ops')"); err != nil {
		t.Fatal(err)
	}
	var auditorRoleID string
	if err := app.db.QueryRow("SELECT id FROM settings_roles WHERE key='auditor'").Scan(&auditorRoleID); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	app.handleSettingsRoles(rec, settingsResourceDeleteRequest("/api/settings/roles/{id}", auditorRoleID))
	if rec.Code != http.StatusOK {
		t.Fatalf("role delete status = %d, body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	app.handleSettingsGroups(rec, settingsResourceDeleteRequest("/api/settings/user-groups/{id}", "1"))
	if rec.Code != http.StatusOK {
		t.Fatalf("group delete status = %d, body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	app.handleSettingsUsers(rec, settingsResourceDeleteRequest("/api/settings/users/{id}", "3"))
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "请先禁用用户再删除") {
		t.Fatalf("user delete guard status = %d, body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	app.handleSettingsUsers(rec, settingsResourceDeleteRequest("/api/settings/users/{id}", "2"))
	if rec.Code != http.StatusOK {
		t.Fatalf("user delete status = %d, body=%s", rec.Code, rec.Body.String())
	}
}

func assertRelationCounts(t *testing.T, db *sql.DB, groupID int64, want int) {
	t.Helper()
	var members, roles int
	if err := db.QueryRow("SELECT COUNT(*) FROM settings_user_group_members WHERE group_id=?", groupID).Scan(&members); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM settings_user_group_roles WHERE group_id=?", groupID).Scan(&roles); err != nil {
		t.Fatal(err)
	}
	if members != want || roles != want {
		t.Fatalf("relation counts = %d/%d, want %d/%d", members, roles, want, want)
	}
}

func TestSettingsEmailPreservesPasswordWhenBlank(t *testing.T) {
	db, err := sql.Open("sqlite", "file:settings-email-password?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := EnsureResourceSchema(db); err != nil {
		t.Fatal(err)
	}
	oldPassword := "enc:v1:existing-password"
	if _, err := db.Exec("UPDATE settings_email SET config=? WHERE id=1", mustJSON(map[string]any{"smtpHost": "localhost", "smtpPort": "25", "username": "sender@example.com", "password": oldPassword, "from": "sender@example.com"})); err != nil {
		t.Fatal(err)
	}
	app := &Router{db: db}
	req := httptest.NewRequest(http.MethodPut, "/api/settings/email", strings.NewReader(`{"passwordResetEnabled":true,"config":{"smtpHost":"localhost","smtpPort":"25","username":"sender@example.com","from":"sender@example.com"}}`))
	response := httptest.NewRecorder()
	app.handleSettingsEmail(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("save email status = %d, body=%s", response.Code, response.Body.String())
	}
	var raw string
	if err := db.QueryRow("SELECT config FROM settings_email WHERE id=1").Scan(&raw); err != nil {
		t.Fatal(err)
	}
	stored := decodeSettingsMap(raw)
	if stored["password"] != oldPassword {
		t.Fatalf("stored password = %#v, want original ciphertext", stored["password"])
	}
	if strings.Contains(response.Body.String(), oldPassword) {
		t.Fatal("response leaked encrypted password")
	}
}
func TestSettingsAuthProviderPreservesPasswordAndSyncsPublicProvider(t *testing.T) {
	db, err := sql.Open("sqlite", "file:settings-auth-sync?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("CREATE TABLE settings (useldap INTEGER, ldap_server TEXT, ldap_dn TEXT, ldap_bind_dn TEXT, ldap_bind_password TEXT, ldap_getusers TEXT, ldap_getusers_filter TEXT)"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO settings VALUES(0,'','','','enc:v1:ldap-password','','')"); err != nil {
		t.Fatal(err)
	}
	if err := EnsureResourceSchema(db); err != nil {
		t.Fatal(err)
	}
	app := &Router{db: db}
	body := `{"name":"企业 LDAP","enabled":true,"config":{"host":"ldap.example.com","port":389,"baseDN":"DC=example,DC=com","bindDN":"CN=bind,DC=example,DC=com","userFilter":"sAMAccountName","groupFilter":"","hasPassword":true}}`
	response := httptest.NewRecorder()
	app.handleSettingsAuthProvider(response, httptest.NewRequest(http.MethodPut, "/api/settings/auth-provider", strings.NewReader(body)))
	if response.Code != http.StatusOK {
		t.Fatalf("save LDAP status = %d, body=%s", response.Code, response.Body.String())
	}
	var password string
	if err := db.QueryRow("SELECT ldap_bind_password FROM settings").Scan(&password); err != nil {
		t.Fatal(err)
	}
	if password != "enc:v1:ldap-password" {
		t.Fatalf("LDAP password = %q, want preserved ciphertext", password)
	}
	var enabled int64
	var name string
	if err := db.QueryRow("SELECT enabled,name FROM settings_auth_providers WHERE id='ldap'").Scan(&enabled, &name); err != nil {
		t.Fatal(err)
	}
	if enabled != 1 || name != "企业 LDAP" {
		t.Fatalf("public provider = enabled:%d name:%q", enabled, name)
	}
	if strings.Contains(response.Body.String(), "ldap-password") {
		t.Fatal("LDAP password leaked in response")
	}
	getResponse := httptest.NewRecorder()
	app.handleSettingsAuthProvider(getResponse, httptest.NewRequest(http.MethodGet, "/api/settings/auth-provider", nil))
	if getResponse.Code != http.StatusOK {
		t.Fatalf("get LDAP status = %d, body=%s", getResponse.Code, getResponse.Body.String())
	}
	var payload struct {
		Name   string         `json:"name"`
		Config map[string]any `json:"config"`
	}
	if err := json.Unmarshal(getResponse.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Name != "企业 LDAP" {
		t.Fatalf("GET name = %q, want saved display name", payload.Name)
	}
	if payload.Config["port"] != float64(389) {
		t.Fatalf("GET config port = %v, want 389", payload.Config["port"])
	}
	if payload.Config["hasBindPassword"] != true {
		t.Fatalf("GET config hasBindPassword = %v, want true", payload.Config["hasBindPassword"])
	}
	disableBody := `{"name":"企业 LDAP","enabled":false,"config":{"host":"ldap.example.com","port":389,"baseDN":"DC=example,DC=com","bindDN":"CN=bind,DC=example,DC=com","userFilter":"sAMAccountName"}}`
	disableResponse := httptest.NewRecorder()
	app.handleSettingsAuthProvider(disableResponse, httptest.NewRequest(http.MethodPut, "/api/settings/auth-provider", strings.NewReader(disableBody)))
	if disableResponse.Code != http.StatusOK {
		t.Fatalf("disable LDAP status = %d, body=%s", disableResponse.Code, disableResponse.Body.String())
	}
	if err := db.QueryRow("SELECT ldap_bind_password FROM settings").Scan(&password); err != nil {
		t.Fatal(err)
	}
	if password != "enc:v1:ldap-password" {
		t.Fatalf("LDAP password = %q, want preserved after disabled save", password)
	}
}

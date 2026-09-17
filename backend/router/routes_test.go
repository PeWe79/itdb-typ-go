package router

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	_ "modernc.org/sqlite"

	"itdb-backend/router/settings"
)

func settingsTestToken(t *testing.T, secret string, userID int64, username string, userType int64) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, AuthClaims{
		UserID: userID, Username: username, UserType: userType,
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
	})
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	return signed
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func TestSettingsHTTPUsesRolePermissions(t *testing.T) {
	db, err := sql.Open("sqlite", "file:settings-http?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, username TEXT, userdesc TEXT, pass TEXT, usertype INTEGER)"); err != nil {
		t.Fatal(err)
	}
	if err := settings.EnsureResourceSchema(db); err != nil {
		t.Fatal(err)
	}
	if err := settings.EnsureUserProfileLoginSchema(db, ":memory:"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO users(id,username,userdesc,usertype) VALUES(1,'alice','Alice',1)"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO settings_user_profiles(user_id,email) VALUES(1,'alice@example.com')"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO settings_roles(key,name,permissions,builtin) VALUES('reader','Reader','[\"settings.users.read\"]',0),('manager','Manager','[\"settings.users.manage\"]',0)"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO settings_user_roles(user_id,role_id) SELECT 1,id FROM settings_roles WHERE key='reader'"); err != nil {
		t.Fatal(err)
	}

	app := &App{db: db, cfg: Config{JWTSecret: "test-secret"}}
	handler := app.routes()
	readToken := settingsTestToken(t, app.cfg.JWTSecret, 1, "alice", 1)
	request := httptest.NewRequest(http.MethodGet, "/api/settings/users", nil)
	request.Header.Set("Authorization", "Bearer "+readToken)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("GET users status = %d, body=%s", response.Code, response.Body.String())
	}
	var body struct {
		Items []struct {
			Permissions []string `json:"effectiveUserPermissions"`
		} `json:"items"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 1 || !containsString(body.Items[0].Permissions, "settings.users.read") {
		t.Fatalf("unexpected user permissions: %#v", body.Items)
	}
	if _, err := db.Exec("INSERT INTO settings_user_groups(id,name) VALUES(1,'Operators')"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO settings_user_group_members(group_id,user_id) VALUES(1,1)"); err != nil {
		t.Fatal(err)
	}
	groupRequest := httptest.NewRequest(http.MethodGet, "/api/settings/user-groups", nil)
	groupRequest.Header.Set("Authorization", "Bearer "+readToken)
	groupResponse := httptest.NewRecorder()
	handler.ServeHTTP(groupResponse, groupRequest)
	if groupResponse.Code != http.StatusOK {
		t.Fatalf("GET user groups status = %d, body=%s", groupResponse.Code, groupResponse.Body.String())
	}

	create := httptest.NewRequest(http.MethodPost, "/api/settings/roles", strings.NewReader(`{"key":"blocked","name":"Blocked"}`))
	create.Header.Set("Authorization", "Bearer "+readToken)
	createResponse := httptest.NewRecorder()
	handler.ServeHTTP(createResponse, create)
	if createResponse.Code != http.StatusForbidden {
		t.Fatalf("POST role with read permission status = %d, want 403", createResponse.Code)
	}

	if _, err := db.Exec("INSERT INTO settings_user_roles(user_id,role_id) SELECT 1,id FROM settings_roles WHERE key='manager'"); err != nil {
		t.Fatal(err)
	}
	writeToken := settingsTestToken(t, app.cfg.JWTSecret, 1, "alice", 1)
	create = httptest.NewRequest(http.MethodPost, "/api/settings/roles", strings.NewReader(`{"key":"auditor","name":"Auditor"}`))
	create.Header.Set("Authorization", "Bearer "+writeToken)
	createResponse = httptest.NewRecorder()
	handler.ServeHTTP(createResponse, create)
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("POST role with manage permission status = %d, body=%s", createResponse.Code, createResponse.Body.String())
	}
}

func TestPublicAuthProvidersReturnsLDAPConfiguration(t *testing.T) {
	db, err := sql.Open("sqlite", "file:public-auth-providers?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := settings.EnsureResourceSchema(db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("UPDATE settings_auth_providers SET enabled=1,name='企业 LDAP' WHERE id='ldap'"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("UPDATE settings_email SET password_reset_enabled=1 WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	app := &App{db: db}
	response := httptest.NewRecorder()
	app.routes().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/auth/providers", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("providers status = %d, body=%s", response.Code, response.Body.String())
	}
	var body struct {
		Items []struct {
			ID      string `json:"id"`
			Type    string `json:"type"`
			Enabled bool   `json:"enabled"`
		} `json:"items"`
		PasswordResetEnabled bool `json:"passwordResetEnabled"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 1 || body.Items[0].ID != "ldap" || body.Items[0].Type != "ldap" || !body.Items[0].Enabled || !body.PasswordResetEnabled {
		t.Fatalf("unexpected public auth response: %#v", body)
	}
}

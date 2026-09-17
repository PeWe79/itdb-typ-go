package settings

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"itdb-backend/internal/domain"
	"itdb-backend/internal/service"
	"itdb-backend/router/common"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "modernc.org/sqlite"
)

func newSettingsBaseTestApp(t *testing.T) *Router {
	t.Helper()
	db, err := sql.Open("sqlite", "file:settings-base-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := EnsureResourceSchema(db); err != nil {
		t.Fatal(err)
	}
	return &Router{db: db}
}

func TestSettingsBaseReturnsDefaults(t *testing.T) {
	app := newSettingsBaseTestApp(t)
	response := httptest.NewRecorder()
	app.handleSettingsBase(response, httptest.NewRequest(http.MethodGet, "/api/settings/base", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("GET status = %d, body=%s", response.Code, response.Body.String())
	}
	var config map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &config); err != nil {
		t.Fatal(err)
	}
	if config["siteName"] != "ITDB" || config["appSubtitle"] != "IT Asset Management" {
		t.Fatalf("unexpected defaults: %v", config)
	}
	if config["resetCodeTtlMinutes"] != float64(10) || config["resetCaptchaTtlMinutes"] != float64(1) {
		t.Fatalf("unexpected security defaults: %v", config)
	}
}

func TestSettingsBaseUpdateValidatesAndPersists(t *testing.T) {
	app := newSettingsBaseTestApp(t)
	payload := []byte(`{"siteName":"资产中枢","loginName":"资产中枢","appName":"资产中枢","appSubtitle":"IT Asset Management","iconData":"/favicon.svg","resetCodeTtlMinutes":30,"resetCaptchaTtlMinutes":2,"passwordResetSendCooldownMinutes":0.5,"passwordResetRateLimitMinutes":5}`)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/api/settings/base", bytes.NewReader(payload))
	app.handleSettingsBase(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, body=%s", response.Code, response.Body.String())
	}

	updated := httptest.NewRecorder()
	app.handleSettingsBase(updated, httptest.NewRequest(http.MethodGet, "/api/settings/base", nil))
	var config map[string]any
	if err := json.Unmarshal(updated.Body.Bytes(), &config); err != nil {
		t.Fatal(err)
	}
	if config["siteName"] != "资产中枢" || config["resetCodeTtlMinutes"] != float64(30) || config["passwordResetSendCooldownMinutes"] != float64(0.5) {
		t.Fatalf("persisted config mismatch: %v", config)
	}

	public := httptest.NewRecorder()
	app.handlePublicBase(public, httptest.NewRequest(http.MethodGet, "/api/public/base", nil))
	if public.Code != http.StatusOK {
		t.Fatalf("public status = %d", public.Code)
	}
	var branding map[string]any
	if err := json.Unmarshal(public.Body.Bytes(), &branding); err != nil {
		t.Fatal(err)
	}
	if branding["siteName"] != "资产中枢" {
		t.Fatalf("public branding mismatch: %v", branding)
	}
	if _, exists := branding["resetCodeTtlMinutes"]; exists {
		t.Fatalf("public branding must not expose security timings: %v", branding)
	}
}

func TestSettingsBaseUpdateRejectsOutOfRange(t *testing.T) {
	app := newSettingsBaseTestApp(t)
	cases := map[string]string{
		"reset code ttl":    `{"resetCodeTtlMinutes":999}`,
		"captcha ttl":       `{"resetCaptchaTtlMinutes":99}`,
		"cooldown step":     `{"passwordResetSendCooldownMinutes":1.3}`,
		"rate limit window": `{"passwordResetRateLimitMinutes":60}`,
		"icon data":         `{"iconData":"http://evil.example/logo.png"}`,
		"site name length":  `{"siteName":"` + string(make([]rune, 61)) + `"}`,
	}
	for name, payload := range cases {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPut, "/api/settings/base", bytes.NewReader([]byte(payload)))
		app.handleSettingsBase(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d, want 400, body=%s", name, response.Code, response.Body.String())
		}
	}
}

func TestSettingsBaseSectionUpdateKeepsOtherSections(t *testing.T) {
	app := newSettingsBaseTestApp(t)
	seed := []byte(`{"siteName":"资产中枢","loginName":"资产中枢","appName":"资产中枢","appSubtitle":"IT Asset Management","iconData":"/favicon.svg","resetCodeTtlMinutes":10,"resetCaptchaTtlMinutes":1,"passwordResetSendCooldownMinutes":0.5,"passwordResetRateLimitMinutes":5,"backupRetentionDays":30}`)
	response := httptest.NewRecorder()
	app.handleSettingsBase(response, httptest.NewRequest(http.MethodPut, "/api/settings/base", bytes.NewReader(seed)))
	if response.Code != http.StatusOK {
		t.Fatalf("seed PUT status = %d, body=%s", response.Code, response.Body.String())
	}

	patch := []byte(`{"resetCodeTtlMinutes":45,"section":"security"}`)
	response = httptest.NewRecorder()
	app.handleSettingsBase(response, httptest.NewRequest(http.MethodPut, "/api/settings/base", bytes.NewReader(patch)))
	if response.Code != http.StatusOK {
		t.Fatalf("section PUT status = %d, body=%s", response.Code, response.Body.String())
	}

	updated := httptest.NewRecorder()
	app.handleSettingsBase(updated, httptest.NewRequest(http.MethodGet, "/api/settings/base", nil))
	var config map[string]any
	if err := json.Unmarshal(updated.Body.Bytes(), &config); err != nil {
		t.Fatal(err)
	}
	if config["resetCodeTtlMinutes"] != float64(45) {
		t.Fatalf("section patch not applied: %v", config)
	}
	if config["siteName"] != "资产中枢" || config["backupRetentionDays"] != float64(30) {
		t.Fatalf("other sections must stay untouched: %v", config)
	}

	invalid := httptest.NewRecorder()
	app.handleSettingsBase(invalid, httptest.NewRequest(http.MethodPut, "/api/settings/base", bytes.NewReader([]byte(`{"siteName":"X","section":"nope"}`))))
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid section status = %d", invalid.Code)
	}
}

func TestSettingsBaseAuditSkipsUnchangedSave(t *testing.T) {
	app := newSettingsBaseTestApp(t)
	app.audit = service.NewAuditService(app.db, 0)
	if _, err := app.db.Exec("CREATE TABLE history (id INTEGER PRIMARY KEY AUTOINCREMENT, date INTEGER, sql TEXT, authuser TEXT, ip TEXT, module TEXT DEFAULT '', action TEXT DEFAULT '', target TEXT DEFAULT '', detail TEXT DEFAULT '', result TEXT DEFAULT 'success')"); err != nil {
		t.Fatal(err)
	}
	user := common.WithUser(context.Background(), domain.SessionUser{ID: 1, Username: "admin"})

	first := httptest.NewRecorder()
	firstRequest := httptest.NewRequest(http.MethodPut, "/api/settings/base", bytes.NewReader([]byte(`{"siteName":"资产中枢","section":"brand"}`))).WithContext(user)
	app.handleSettingsBase(first, firstRequest)
	if first.Code != http.StatusOK {
		t.Fatalf("changed save status = %d, body=%s", first.Code, first.Body.String())
	}

	second := httptest.NewRecorder()
	secondRequest := httptest.NewRequest(http.MethodPut, "/api/settings/base", bytes.NewReader([]byte(`{"siteName":"资产中枢","section":"brand"}`))).WithContext(user)
	app.handleSettingsBase(second, secondRequest)
	if second.Code != http.StatusOK {
		t.Fatalf("unchanged save status = %d, body=%s", second.Code, second.Body.String())
	}

	var total int
	var target, detail string
	if err := app.db.QueryRow("SELECT COUNT(*) FROM history").Scan(&total); err != nil {
		t.Fatal(err)
	}
	if err := app.db.QueryRow("SELECT target, detail FROM history ORDER BY id DESC LIMIT 1").Scan(&target, &detail); err != nil {
		t.Fatal(err)
	}
	if total != 1 || target != "品牌标识" || detail != "品牌标识配置已更新" {
		t.Fatalf("audit rows=%d target=%q detail=%q, want single 品牌标识 record", total, target, detail)
	}
}

func TestSettingsBaseEmptyStringsFallBackToDefaults(t *testing.T) {
	app := newSettingsBaseTestApp(t)
	payload := []byte(`{"siteName":"  ","iconData":""}`)
	response := httptest.NewRecorder()
	app.handleSettingsBase(response, httptest.NewRequest(http.MethodPut, "/api/settings/base", bytes.NewReader(payload)))
	if response.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, body=%s", response.Code, response.Body.String())
	}
	var config map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &config); err != nil {
		t.Fatal(err)
	}
	if config["siteName"] != "ITDB" || config["iconData"] != "/favicon.svg" {
		t.Fatalf("empty values should fall back to defaults: %v", config)
	}
}

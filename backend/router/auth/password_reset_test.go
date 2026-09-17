package auth

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"itdb-backend/router/settings"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestPasswordResetCaptchaAndVerify(t *testing.T) {
	db, err := sql.Open("sqlite", "file:password-reset-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, query := range []string{
		"CREATE TABLE users (id INTEGER PRIMARY KEY, username TEXT, userdesc TEXT, pass TEXT, usertype INTEGER)",
		"CREATE TABLE settings_user_profiles (user_id INTEGER PRIMARY KEY, email TEXT, disabled INTEGER, source TEXT, created_at INTEGER, updated_at INTEGER)",
	} {
		if _, err := db.Exec(query); err != nil {
			t.Fatal(err)
		}
	}
	if err := settings.EnsureResourceSchema(db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO users(id,username,userdesc,pass,usertype) VALUES(1,'alice','Alice','hash',1)"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO settings_user_profiles(user_id,email) VALUES(1,'alice@example.com')"); err != nil {
		t.Fatal(err)
	}
	app := &Router{db: db}
	captcha := httptest.NewRecorder()
	app.handlePasswordResetCaptcha(captcha, httptest.NewRequest(http.MethodGet, "/api/auth/password-reset/captcha", nil))
	if captcha.Code != http.StatusOK {
		t.Fatalf("captcha status = %d", captcha.Code)
	}
	var captchaBody struct {
		Token    string `json:"token"`
		Question string `json:"question"`
	}
	if err := json.Unmarshal(captcha.Body.Bytes(), &captchaBody); err != nil {
		t.Fatal(err)
	}
	match := regexp.MustCompile(`(\d+) \+ (\d+)`).FindStringSubmatch(captchaBody.Question)
	if len(match) != 3 {
		t.Fatalf("unexpected captcha question: %s", captchaBody.Question)
	}
	first, _ := strconv.Atoi(match[1])
	second, _ := strconv.Atoi(match[2])
	payload, _ := json.Marshal(map[string]string{"username": "alice", "captchaToken": captchaBody.Token, "captchaAnswer": strconv.Itoa(first + second)})
	verify := httptest.NewRecorder()
	app.handlePasswordResetVerify(verify, httptest.NewRequest(http.MethodPost, "/api/auth/password-reset/verify", bytes.NewReader(payload)))
	if verify.Code != http.StatusOK {
		t.Fatalf("verify status = %d, body=%s", verify.Code, verify.Body.String())
	}
	secondVerify := httptest.NewRecorder()
	app.handlePasswordResetVerify(secondVerify, httptest.NewRequest(http.MethodPost, "/api/auth/password-reset/verify", bytes.NewReader(payload)))
	if secondVerify.Code != http.StatusBadRequest {
		t.Fatalf("reused captcha status = %d, want 400", secondVerify.Code)
	}
}

func TestPasswordResetSendRejectsCooldownBeforeSMTP(t *testing.T) {
	db, err := sql.Open("sqlite", "file:password-reset-cooldown?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, query := range []string{
		"CREATE TABLE users (id INTEGER PRIMARY KEY, username TEXT, userdesc TEXT, pass TEXT, usertype INTEGER)",
		"CREATE TABLE settings_user_profiles (user_id INTEGER PRIMARY KEY, email TEXT)",
	} {
		if _, err := db.Exec(query); err != nil {
			t.Fatal(err)
		}
	}
	if err := settings.EnsureResourceSchema(db); err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	if _, err := db.Exec("INSERT INTO users(id,username) VALUES(1,'alice')"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO password_reset_requests(token_hash,user_id,email,expires_at,used,last_sent_at,created_at) VALUES(?,?,?,?,?,?,?)", hashToken("verification"), 1, "alice@example.com", now+600, 0, now, now); err != nil {
		t.Fatal(err)
	}
	app := &Router{db: db}
	payload := []byte(`{"verificationToken":"verification","channel":"email","verifyEmail":"alice@example.com"}`)
	response := httptest.NewRecorder()
	app.handlePasswordResetSend(response, httptest.NewRequest(http.MethodPost, "/api/auth/password-reset/send", bytes.NewReader(payload)))
	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("cooldown status = %d, want 429", response.Code)
	}
}

func TestPasswordResetVerifyMessagesAndConfirmSplit(t *testing.T) {
	db, err := sql.Open("sqlite", "file:password-reset-messages?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, query := range []string{
		"CREATE TABLE users (id INTEGER PRIMARY KEY, username TEXT, userdesc TEXT, pass TEXT, usertype INTEGER)",
		"CREATE TABLE settings_user_profiles (user_id INTEGER PRIMARY KEY, email TEXT, disabled INTEGER, source TEXT, created_at INTEGER, updated_at INTEGER)",
	} {
		if _, err := db.Exec(query); err != nil {
			t.Fatal(err)
		}
	}
	for _, seed := range []string{
		"INSERT INTO users(id,username) VALUES(1,'alice')",
		"INSERT INTO users(id,username) VALUES(2,'bob')",
		"INSERT INTO users(id,username) VALUES(3,'carol')",
	} {
		if _, err := db.Exec(seed); err != nil {
			t.Fatal(err)
		}
	}
	if err := settings.EnsureResourceSchema(db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("UPDATE settings_user_profiles SET email='alice@example.com' WHERE user_id=1"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("UPDATE settings_user_profiles SET email='bob@example.com',disabled=1 WHERE user_id=2"); err != nil {
		t.Fatal(err)
	}
	app := &Router{db: db}
	for _, username := range []string{"ghost", "bob", "carol"} {
		captcha := createResetCaptchaForTest(t, app)
		payload, _ := json.Marshal(map[string]string{"username": username, "captchaToken": captcha.token, "captchaAnswer": captcha.answer})
		verify := httptest.NewRecorder()
		app.handlePasswordResetVerify(verify, httptest.NewRequest(http.MethodPost, "/api/auth/password-reset/verify", bytes.NewReader(payload)))
		if verify.Code != http.StatusBadRequest || !strings.Contains(verify.Body.String(), "当前账号无法找回密码") {
			t.Fatalf("verify %s status = %d, body=%s", username, verify.Code, verify.Body.String())
		}
	}
	now := time.Now().Unix()
	if _, err := db.Exec("INSERT INTO password_reset_requests(token_hash,user_id,email,code_hash,expires_at,used,created_at) VALUES(?,?,?,?,?,0,?)", hashToken("vtoken"), 1, "alice@example.com", hashToken("123456"), now+600, now); err != nil {
		t.Fatal(err)
	}
	confirm := func(token, code string) *httptest.ResponseRecorder {
		payload, _ := json.Marshal(map[string]string{"username": "alice", "verificationToken": token, "code": code, "newPassword": "newpass123", "confirmPassword": "newpass123"})
		rec := httptest.NewRecorder()
		app.handlePasswordResetConfirm(rec, httptest.NewRequest(http.MethodPost, "/api/auth/password-reset/confirm", bytes.NewReader(payload)))
		return rec
	}
	if rec := confirm("bad-token", "123456"); rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "请重新完成用户名和图形验证码校验") {
		t.Fatalf("confirm bad token status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if rec := confirm("vtoken", "000000"); rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "验证码不正确") {
		t.Fatalf("confirm wrong code status = %d, body=%s", rec.Code, rec.Body.String())
	}
}

type resetCaptchaForTest struct {
	token  string
	answer string
}

func createResetCaptchaForTest(t *testing.T, app *Router) resetCaptchaForTest {
	t.Helper()
	rec := httptest.NewRecorder()
	app.handlePasswordResetCaptcha(rec, httptest.NewRequest(http.MethodGet, "/api/auth/password-reset/captcha", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("captcha status = %d", rec.Code)
	}
	var body struct {
		Token    string `json:"token"`
		Question string `json:"question"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	match := regexp.MustCompile(`(\d+) \+ (\d+)`).FindStringSubmatch(body.Question)
	if len(match) != 3 {
		t.Fatalf("unexpected captcha question: %s", body.Question)
	}
	first, _ := strconv.Atoi(match[1])
	second, _ := strconv.Atoi(match[2])
	return resetCaptchaForTest{token: body.Token, answer: strconv.Itoa(first + second)}
}

package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"itdb-backend/internal/domain"
	"itdb-backend/internal/repository"
	"itdb-backend/internal/security"

	_ "modernc.org/sqlite"
)

func TestAuthWorkflowLocalLoginAndLegacyUpgrade(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY, username TEXT, pass TEXT, usertype INTEGER, userdesc TEXT); INSERT INTO users VALUES (1,'admin','legacy-pass',1,'administrator')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE settings_user_profiles (user_id INTEGER PRIMARY KEY, disabled INTEGER NOT NULL DEFAULT 0)`); err != nil {
		t.Fatal(err)
	}
	workflow := NewAuthWorkflow(repository.NewAuthRepository(repository.NewStore(db)), "test-secret", nil, 24*time.Hour)
	response, err := workflow.Login(context.Background(), domain.AuthLoginRequest{Username: " admin ", Password: "legacy-pass", Mode: "local"})
	if err != nil {
		t.Fatal(err)
	}
	if response.Token == "" || response.User.UserType != 0 {
		t.Fatalf("unexpected login response: %+v", response)
	}
	var stored string
	if err := db.QueryRow(`SELECT pass FROM users WHERE id=1`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if !security.IsBcryptHash(stored) {
		t.Fatalf("legacy password was not upgraded: %q", stored)
	}
	me, err := workflow.Me(context.Background(), response.User)
	if err != nil {
		t.Fatal(err)
	}
	if me["userDesc"] != "administrator" {
		t.Fatalf("unexpected me response: %#v", me)
	}
}

func TestAuthWorkflowRejectsInvalidPassword(t *testing.T) {
	db, _ := sql.Open("sqlite", ":memory:")
	defer db.Close()
	_, _ = db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY, username TEXT, pass TEXT, usertype INTEGER, userdesc TEXT); INSERT INTO users VALUES (1,'alice','secret',1,'')`)
	_, _ = db.Exec(`CREATE TABLE settings_user_profiles (user_id INTEGER PRIMARY KEY, disabled INTEGER NOT NULL DEFAULT 0)`)
	workflow := NewAuthWorkflow(repository.NewAuthRepository(repository.NewStore(db)), "test-secret", nil, 24*time.Hour)
	_, err := workflow.Login(context.Background(), domain.AuthLoginRequest{Username: "alice", Password: "wrong"})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
}

func TestAuthWorkflowLocalDisabledUserChecksPasswordFirst(t *testing.T) {
	db, _ := sql.Open("sqlite", ":memory:")
	defer db.Close()
	hashed, _ := security.HashPassword("secret")
	_, _ = db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY, username TEXT, pass TEXT, usertype INTEGER, userdesc TEXT)`)
	_, _ = db.Exec(`CREATE TABLE settings_user_profiles (user_id INTEGER PRIMARY KEY, disabled INTEGER NOT NULL DEFAULT 0)`)
	_, _ = db.Exec(`INSERT INTO users (id, username, pass, usertype) VALUES (1, 'alice', ?, 1)`, hashed)
	_, _ = db.Exec(`INSERT INTO settings_user_profiles (user_id, disabled) VALUES (1, 1)`)
	workflow := NewAuthWorkflow(repository.NewAuthRepository(repository.NewStore(db)), "test-secret", nil, 24*time.Hour)

	_, err := workflow.Login(context.Background(), domain.AuthLoginRequest{Username: "ghost", Password: "whatever", Mode: "local"})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
	_, err = workflow.Login(context.Background(), domain.AuthLoginRequest{Username: "alice", Password: "secret", Mode: "local"})
	if !errors.Is(err, ErrUserNotProvisioned) {
		t.Fatalf("expected user not provisioned, got %v", err)
	}
}

func TestAuthWorkflowUnknownLocalUserReturnsInvalidCredentials(t *testing.T) {
	db, _ := sql.Open("sqlite", ":memory:")
	defer db.Close()
	_, _ = db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY, username TEXT, pass TEXT, usertype INTEGER, userdesc TEXT)`)
	_, _ = db.Exec(`CREATE TABLE settings_user_profiles (user_id INTEGER PRIMARY KEY, disabled INTEGER NOT NULL DEFAULT 0)`)
	workflow := NewAuthWorkflow(repository.NewAuthRepository(repository.NewStore(db)), "test-secret", nil, 24*time.Hour)
	_, err := workflow.Login(context.Background(), domain.AuthLoginRequest{Username: "ghost", Password: "whatever", Mode: "local"})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
}

// TestSessionTTLHonored 校验会话有效期由构造参数决定且零值回退默认 24 小时。
func TestSessionTTLHonored(t *testing.T) {
	newDB := func() *sql.DB {
		db, _ := sql.Open("sqlite", ":memory:")
		db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY, username TEXT, pass TEXT, usertype INTEGER, userdesc TEXT); INSERT INTO users VALUES (1,'alice','pw',1,'')`)
		db.Exec(`CREATE TABLE settings_user_profiles (user_id INTEGER PRIMARY KEY, disabled INTEGER NOT NULL DEFAULT 0)`)
		return db
	}
	ttlFor := func(ttl time.Duration) time.Duration {
		db := newDB()
		defer db.Close()
		workflow := NewAuthWorkflow(repository.NewAuthRepository(repository.NewStore(db)), "test-secret", nil, ttl)
		response, err := workflow.Login(context.Background(), domain.AuthLoginRequest{Username: "alice", Password: "pw", Mode: "local"})
		if err != nil {
			t.Fatal(err)
		}
		parsed, _, err := jwt.NewParser().ParseUnverified(response.Token, &authClaims{})
		if err != nil {
			t.Fatal(err)
		}
		claims := parsed.Claims.(*authClaims)
		return claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time)
	}
	if got := ttlFor(2 * time.Hour); got != 2*time.Hour {
		t.Fatalf("ttl = %v, want 2h", got)
	}
	if got := ttlFor(0); got != 24*time.Hour {
		t.Fatalf("default ttl = %v, want 24h", got)
	}
}

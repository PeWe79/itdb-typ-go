package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"itdb-backend/internal/domain"
	"itdb-backend/internal/repository"
	"itdb-backend/internal/security"

	_ "modernc.org/sqlite"
)

func newUserWorkflowTest(t *testing.T) (*UserWorkflow, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{
		`CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT, userdesc TEXT, pass TEXT, cookie1 TEXT, usertype INTEGER)`,
		`CREATE TABLE items (id INTEGER PRIMARY KEY AUTOINCREMENT, userid INTEGER)`,
		`CREATE TABLE contracts (id INTEGER PRIMARY KEY AUTOINCREMENT, renewals TEXT)`,
		auditHistoryTestDDL,
	} {
		if _, err := db.Exec(query); err != nil {
			db.Close()
			t.Fatal(err)
		}
	}
	store := repository.NewStore(db)
	audit := NewAuditService(store, 100)
	return NewUserWorkflow(repository.NewUserRepository(store), audit), db
}

func TestUserWorkflowUpdate(t *testing.T) {
	workflow, db := newUserWorkflowTest(t)
	defer db.Close()
	if _, err := db.Exec(`INSERT INTO users (username, userdesc, pass, usertype) VALUES ('admin', 'ops', 'old', 1)`); err != nil {
		t.Fatal(err)
	}

	if err := workflow.Update(context.Background(), UserActor{Username: "admin", IP: "127.0.0.1"}, 1, domain.UserPayload{Username: "admin", UserDesc: "ops", Password: "new-secret", UserType: 1}); err != nil {
		t.Fatal(err)
	}
	var userType int64
	var password string
	if err := db.QueryRow(`SELECT usertype, pass FROM users WHERE id = 1`).Scan(&userType, &password); err != nil {
		t.Fatal(err)
	}
	if userType != 0 {
		t.Fatalf("admin user type was not forced to 0: %d", userType)
	}
	if !security.IsBcryptHash(password) {
		t.Fatalf("password was not hashed: %q", password)
	}
	if ok, _ := security.VerifyPassword(password, "new-secret"); !ok {
		t.Fatal("stored password does not verify")
	}
}

func TestUserWorkflowRejectsDuplicateUsername(t *testing.T) {
	workflow, db := newUserWorkflowTest(t)
	defer db.Close()
	if _, err := db.Exec(`INSERT INTO users (username) VALUES (?)`, "alice"); err != nil {
		t.Fatal(err)
	}
	if err := workflow.Update(context.Background(), UserActor{}, 99, domain.UserPayload{Username: "alice"}); !errors.Is(err, ErrUsernameExists) {
		t.Fatalf("expected duplicate username error, got %v", err)
	}
}

func TestUserWorkflowDeleteBlocksReferencedUser(t *testing.T) {
	workflow, db := newUserWorkflowTest(t)
	defer db.Close()
	if _, err := db.Exec(`INSERT INTO users (id, username) VALUES (1, 'admin'), (2, 'alice')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO items (userid) VALUES (2), (2), (1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO contracts (id, renewals) VALUES (1, '#### 2026-01-01#2')`); err != nil {
		t.Fatal(err)
	}
	err := workflow.Delete(context.Background(), UserActor{Username: "admin", IP: "10.0.0.1"}, 2)
	var conflict *ConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("expected conflict error, got %v", err)
	}
	expected := "该用户已被 2 条硬件记录使用，无法删除\n该用户已被 1 条合同记录使用，无法删除"
	if conflict.Message != expected {
		t.Fatalf("unexpected conflict message: %q", conflict.Message)
	}
	var remaining int
	if err := db.QueryRow(`SELECT COUNT(*) FROM users WHERE id = 2`).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if remaining != 1 {
		t.Fatalf("referenced user should not be deleted")
	}
}

func TestUserWorkflowDeleteUnreferencedUserAuditsAtomically(t *testing.T) {
	workflow, db := newUserWorkflowTest(t)
	defer db.Close()
	if _, err := db.Exec(`INSERT INTO users (id, username) VALUES (1, 'admin'), (2, 'alice')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO items (userid) VALUES (1)`); err != nil {
		t.Fatal(err)
	}
	if err := workflow.Delete(context.Background(), UserActor{Username: "admin", IP: "10.0.0.1"}, 2); err != nil {
		t.Fatal(err)
	}
	var remaining, reassigned int
	if err := db.QueryRow(`SELECT COUNT(*) FROM users WHERE id = 2`).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM items WHERE userid = 1`).Scan(&reassigned); err != nil {
		t.Fatal(err)
	}
	if remaining != 0 || reassigned != 1 {
		t.Fatalf("delete did not reassign correctly: remaining=%d reassigned=%d", remaining, reassigned)
	}
	var historyCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM history`).Scan(&historyCount); err != nil {
		t.Fatal(err)
	}
	if historyCount != 0 {
		t.Fatalf("workflow delete should not write audit rows (routing layer records explicit event), got %d", historyCount)
	}
}

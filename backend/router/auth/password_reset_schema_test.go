package auth

import (
	"database/sql"
	"itdb-backend/router/common"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestEnsurePasswordResetSchemaAddsLastSentAt(t *testing.T) {
	db, err := sql.Open("sqlite", "file:password-reset-schema?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE password_reset_requests (
		token_hash TEXT PRIMARY KEY,
		user_id INTEGER NOT NULL,
		code_hash TEXT NOT NULL DEFAULT '',
		email TEXT NOT NULL DEFAULT '',
		expires_at INTEGER NOT NULL,
		used INTEGER NOT NULL DEFAULT 0,
		created_at INTEGER NOT NULL
	)`); err != nil {
		t.Fatal(err)
	}

	if err := common.EnsurePasswordResetSchema(db, ":memory:"); err != nil {
		t.Fatal(err)
	}
	assertPasswordResetLastSentAt(t, db)

	if err := common.EnsurePasswordResetSchema(db, ":memory:"); err != nil {
		t.Fatal(err)
	}
}

func TestEnsurePasswordResetSchemaBacksUpFileDatabase(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "itdb.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE password_reset_requests (
		token_hash TEXT PRIMARY KEY,
		user_id INTEGER NOT NULL,
		code_hash TEXT NOT NULL DEFAULT '',
		email TEXT NOT NULL DEFAULT '',
		expires_at INTEGER NOT NULL,
		used INTEGER NOT NULL DEFAULT 0,
		created_at INTEGER NOT NULL
	)`); err != nil {
		db.Close()
		t.Fatal(err)
	}

	if err := common.EnsurePasswordResetSchema(db, dbPath); err != nil {
		db.Close()
		t.Fatal(err)
	}
	assertPasswordResetLastSentAt(t, db)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	backups, err := filepath.Glob(filepath.Join(dir, "backups", "itdb-before-password-reset-last-sent-at-*.db"))
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 1 {
		t.Fatalf("expected one database backup, got %d", len(backups))
	}
	if _, err := os.Stat(backups[0]); err != nil {
		t.Fatalf("backup file is missing: %v", err)
	}
}

func assertPasswordResetLastSentAt(t *testing.T, db *sql.DB) {
	t.Helper()
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('password_reset_requests') WHERE name = 'last_sent_at'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("last_sent_at column count = %d", count)
	}
	var defaultValue int
	if err := db.QueryRow(`SELECT COALESCE(last_sent_at, -1) FROM password_reset_requests LIMIT 1`).Scan(&defaultValue); err != sql.ErrNoRows && err != nil {
		t.Fatal(err)
	}
}

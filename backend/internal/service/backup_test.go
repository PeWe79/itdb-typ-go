package service

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestBackupServiceCreatesTemporarySnapshot(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "itdb.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE items (id INTEGER PRIMARY KEY); INSERT INTO items VALUES (1)`); err != nil {
		t.Fatal(err)
	}
	service := NewBackupService(db, dbPath)
	backupPath, cleanup, err := service.CreateTemporaryDatabaseBackup(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if err := ValidateSQLiteFile(backupPath); err != nil {
		t.Fatal(err)
	}
}

func TestValidateSQLiteFileRejectsDatabaseWithoutItems(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE users (id INTEGER)`); err != nil {
		t.Fatal(err)
	}
	db.Close()
	if err := ValidateSQLiteFile(path); err == nil {
		t.Fatal("expected missing items table error")
	}
}

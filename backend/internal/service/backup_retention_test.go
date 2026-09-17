package service

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestCleanupExpiredScheduledBackups(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "itdb.db")
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	svc := NewBackupService(db, dbPath)

	dir := filepath.Join(tmp, "backups")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 14, 12, 0, 0, 0, time.Local)
	files := map[string]bool{
		"itdb-20260901-120000.db":  true,
		"itdb-20260901-120000.zip": true,
		"itdb-20260910-120000.db":  true,
		"itdb-old.db":              true,
		"other-20260901-120000.db": false,
		"notadb.txt":               false,
	}
	for name := range files {
		f, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
	}

	removed, err := svc.CleanupExpiredScheduledBackups(now, 30)
	if err != nil || removed != 0 {
		t.Fatalf("retention 30 should remove nothing, removed=%d err=%v", removed, err)
	}
	removed, err = svc.CleanupExpiredScheduledBackups(now, 7)
	if err != nil || removed != 2 {
		t.Fatalf("retention 7 should remove 2 files, removed=%d err=%v", removed, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "itdb-20260901-120000.db")); !os.IsNotExist(err) {
		t.Fatalf("expired backup should be removed, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "itdb-20260901-120000.zip")); !os.IsNotExist(err) {
		t.Fatalf("expired zip backup should be removed, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "itdb-20260910-120000.db")); err != nil {
		t.Fatalf("recent backup should be kept, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "itdb-old.db")); err != nil {
		t.Fatalf("unparseable file should be skipped")
	}
	removed, err = svc.CleanupExpiredScheduledBackups(now, 0)
	if err != nil || removed != 0 {
		t.Fatalf("retention 0 means keep forever, removed=%d err=%v", removed, err)
	}
}

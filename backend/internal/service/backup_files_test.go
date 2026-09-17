package service

import (
	"archive/zip"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// newScheduledBackupFixture 构造内存库（含文件引用表）、上传目录与定时备份服务
func newScheduledBackupFixture(t *testing.T, fixtureSQL []string) (*BackupService, string, string) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	for _, statement := range fixtureSQL {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	dir := t.TempDir()
	uploadDir := filepath.Join(dir, "uploads")
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		t.Fatal(err)
	}
	return NewBackupService(db, filepath.Join(dir, "itdb.db")), dir, uploadDir
}

func writeUploadFile(t *testing.T, uploadDir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(uploadDir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestCreateScheduledBackupPacksReferencedFiles 引用文件部分存在时打包 zip：
// 包内含备份库与 files 目录下的存在文件，缺失文件名单独返回
func TestCreateScheduledBackupPacksReferencedFiles(t *testing.T) {
	fixture := []string{
		`CREATE TABLE files (id INTEGER PRIMARY KEY AUTOINCREMENT, fname)`,
		`CREATE TABLE locations (id INTEGER PRIMARY KEY AUTOINCREMENT, floorplanfn)`,
		`INSERT INTO files (fname) VALUES ('a.png'), ('c.png')`,
		`INSERT INTO locations (floorplanfn) VALUES ('b.png')`,
	}
	svc, dir, uploadDir := newScheduledBackupFixture(t, fixture)
	writeUploadFile(t, uploadDir, "a.png", "A")
	writeUploadFile(t, uploadDir, "c.png", "C")

	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.Local)
	path, missing, err := svc.CreateScheduledBackup(context.Background(), now, uploadDir)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "itdb-20260917-120000.zip" {
		t.Fatalf("backup path=%s, want zip", path)
	}
	if len(missing) != 1 || missing[0] != "b.png" {
		t.Fatalf("missing=%v, want [b.png]", missing)
	}
	if _, err := os.Stat(filepath.Join(dir, "backups", "itdb-20260917-120000.zip")); err != nil {
		t.Fatal(err)
	}

	zipReader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer zipReader.Close()
	names := make(map[string]bool)
	for _, entry := range zipReader.File {
		names[entry.Name] = true
	}
	for _, expected := range []string{"itdb-20260917-120000.db", "files/a.png", "files/c.png"} {
		if !names[expected] {
			t.Fatalf("zip entry %s missing, got %v", expected, names)
		}
	}
	if names["files/b.png"] {
		t.Fatal("zip should not contain missing file b.png")
	}
}

// TestCreateScheduledBackupPlainWhenNoReference 未引用任何文件时仅导出 db 文件
func TestCreateScheduledBackupPlainWhenNoReference(t *testing.T) {
	fixture := []string{
		`CREATE TABLE files (id INTEGER PRIMARY KEY AUTOINCREMENT, fname)`,
		`CREATE TABLE locations (id INTEGER PRIMARY KEY AUTOINCREMENT, floorplanfn)`,
	}
	svc, dir, uploadDir := newScheduledBackupFixture(t, fixture)

	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.Local)
	path, missing, err := svc.CreateScheduledBackup(context.Background(), now, uploadDir)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "itdb-20260917-120000.db" || len(missing) != 0 {
		t.Fatalf("path=%s missing=%v, want plain db without missing", path, missing)
	}
	if _, err := os.Stat(filepath.Join(dir, "backups", "itdb-20260917-120000.db")); err != nil {
		t.Fatal(err)
	}
}

// TestCreateScheduledBackupPlainWhenAllMissing 引用文件全部不存在时仅导出 db 文件并列出缺失清单
func TestCreateScheduledBackupPlainWhenAllMissing(t *testing.T) {
	fixture := []string{
		`CREATE TABLE files (id INTEGER PRIMARY KEY AUTOINCREMENT, fname)`,
		`CREATE TABLE locations (id INTEGER PRIMARY KEY AUTOINCREMENT, floorplanfn)`,
		`INSERT INTO files (fname) VALUES ('a.png')`,
		`INSERT INTO locations (floorplanfn) VALUES ('b.png')`,
	}
	svc, _, uploadDir := newScheduledBackupFixture(t, fixture)

	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.Local)
	path, missing, err := svc.CreateScheduledBackup(context.Background(), now, uploadDir)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "itdb-20260917-120000.db" {
		t.Fatalf("backup path=%s, want plain db", path)
	}
	if len(missing) != 2 || missing[0] != "a.png" || missing[1] != "b.png" {
		t.Fatalf("missing=%v, want [a.png b.png]", missing)
	}
}

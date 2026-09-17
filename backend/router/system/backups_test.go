package system

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"itdb-backend/config"
	"itdb-backend/internal/repository"
	"itdb-backend/internal/service"

	_ "modernc.org/sqlite"
)

const backupHandlerHistoryDDL = "CREATE TABLE history (id INTEGER PRIMARY KEY AUTOINCREMENT, date INTEGER, sql TEXT, authuser TEXT, ip TEXT, module TEXT DEFAULT '', action TEXT DEFAULT '', target TEXT DEFAULT '', detail TEXT DEFAULT '', result TEXT DEFAULT 'success')"

// newBackupHandlerTestApp 构造手动备份处理器测试实例：内存库 + 临时上传目录
func newBackupHandlerTestApp(t *testing.T, fixtureSQL []string) (*Router, string, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	statements := append([]string{
		backupHandlerHistoryDDL,
		`CREATE TABLE files (id INTEGER PRIMARY KEY AUTOINCREMENT, fname)`,
		`CREATE TABLE locations (id INTEGER PRIMARY KEY AUTOINCREMENT, floorplanfn)`,
	}, fixtureSQL...)
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	dir := t.TempDir()
	uploadDir := filepath.Join(dir, "uploads")
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{UploadDir: uploadDir, DBPath: filepath.Join(dir, "itdb.db")}
	app := &Router{
		db:             db,
		backupWorkflow: service.NewBackupService(repository.NewStore(db), cfg.DBPath),
		cfg:            cfg,
	}
	return app, uploadDir, db
}

func downloadBackup(app *Router, query string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, "/api/backups/database"+query, nil)
	response := httptest.NewRecorder()
	app.handleDownloadDatabaseBackup(response, request)
	return response
}

func lastBackupAudit(t *testing.T, db *sql.DB) (target, detail string) {
	t.Helper()
	if err := db.QueryRow(`SELECT target, detail FROM history ORDER BY id DESC LIMIT 1`).Scan(&target, &detail); err != nil {
		t.Fatal(err)
	}
	return target, detail
}

// TestDownloadBackupPlainWhenAllMissing 勾选包含文件但引用文件全部不存在时应返回 db 文件并注明缺失
func TestDownloadBackupPlainWhenAllMissing(t *testing.T) {
	app, uploadDir, db := newBackupHandlerTestApp(t, []string{
		`INSERT INTO files (fname) VALUES ('test.png')`,
		`INSERT INTO locations (floorplanfn) VALUES ('test2.png')`,
	})
	_ = uploadDir

	response := downloadBackup(app, "?files=1")
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	disposition := response.Header().Get("Content-Disposition")
	if !strings.Contains(disposition, ".db") || strings.Contains(disposition, ".zip") {
		t.Fatalf("disposition=%q, want db filename", disposition)
	}
	target, detail := lastBackupAudit(t, db)
	if detail != "已手动执行数据库备份，其中 test.png、test2.png 文件不存在" {
		t.Fatalf("detail=%q", detail)
	}
	if target[:5] != "itdb-" || filepath.Ext(target) != ".db" {
		t.Fatalf("target=%q", target)
	}
}

// TestDownloadBackupZipWhenFilesExist 引用文件存在时返回 zip 包
func TestDownloadBackupZipWhenFilesExist(t *testing.T) {
	app, uploadDir, db := newBackupHandlerTestApp(t, []string{
		`INSERT INTO files (fname) VALUES ('test.png')`,
	})
	if err := os.WriteFile(filepath.Join(uploadDir, "test.png"), []byte("P"), 0o644); err != nil {
		t.Fatal(err)
	}

	response := downloadBackup(app, "?files=1")
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if disposition := response.Header().Get("Content-Disposition"); !strings.Contains(disposition, ".zip") {
		t.Fatalf("disposition=%q, want zip filename", disposition)
	}
	_, detail := lastBackupAudit(t, db)
	if detail != "已手动执行数据库备份（包含上传文件一并打包）" {
		t.Fatalf("detail=%q", detail)
	}
}

// TestDownloadBackupPlainWithoutFilesFlag 未勾选包含文件时仅返回 db 文件
func TestDownloadBackupPlainWithoutFilesFlag(t *testing.T) {
	app, _, db := newBackupHandlerTestApp(t, []string{
		`INSERT INTO files (fname) VALUES ('test.png')`,
	})

	response := downloadBackup(app, "?files=0")
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if disposition := response.Header().Get("Content-Disposition"); !strings.Contains(disposition, ".db") {
		t.Fatalf("disposition=%q, want db filename", disposition)
	}
	_, detail := lastBackupAudit(t, db)
	if detail != "已手动执行数据库备份" {
		t.Fatalf("detail=%q", detail)
	}
}

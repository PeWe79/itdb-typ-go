package system

import (
	"archive/zip"
	"database/sql"
	"errors"
	"io"
	"itdb-backend/router/common"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"itdb-backend/internal/service"
	_ "modernc.org/sqlite"
)

// handleImportDatabase 导入数据库文件（.db）或包含数据库与上传文件的压缩包（.zip），
// 直接替换当前数据库，不做额外数据处理。
func (a *Router) handleImportDatabase(w http.ResponseWriter, r *http.Request) {
	dbPath := strings.TrimSpace(a.cfg.DBPath)
	if dbPath == "" || strings.EqualFold(dbPath, ":memory:") {
		common.WriteError(w, http.StatusBadRequest, "内存数据库不支持导入操作")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 500<<20)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		common.WriteError(w, http.StatusBadRequest, "文件过大或格式错误")
		return
	}
	uploaded, header, err := r.FormFile("file")
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, "未找到上传文件")
		return
	}
	defer uploaded.Close()

	tmpFile, err := os.CreateTemp("", "itdb-import-*.db")
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "创建临时文件失败")
		return
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)
	if _, err := io.Copy(tmpFile, uploaded); err != nil {
		_ = tmpFile.Close()
		common.WriteError(w, http.StatusInternalServerError, "保存上传文件失败")
		return
	}
	if err := tmpFile.Close(); err != nil {
		common.WriteError(w, http.StatusInternalServerError, "保存上传文件失败")
		return
	}

	var exportFiles []zipExportFile
	if strings.EqualFold(filepath.Ext(header.Filename), ".zip") || isZipFile(tmpPath) {
		dbPathInZip, files, zipErr := extractImportZip(tmpPath)
		if zipErr != nil {
			common.WriteError(w, http.StatusBadRequest, zipErr.Error())
			return
		}
		if dbPathInZip == "" {
			common.WriteError(w, http.StatusBadRequest, "压缩包内未找到 .db 数据库文件")
			return
		}
		if err := copyFile(dbPathInZip, tmpPath); err != nil {
			common.WriteError(w, http.StatusInternalServerError, "解析压缩包失败")
			return
		}
		defer func() {
			_ = os.RemoveAll(filepath.Dir(dbPathInZip))
			for _, item := range files {
				_ = os.RemoveAll(filepath.Dir(item.source))
			}
		}()
		exportFiles = files
	}
	if err := service.ValidateSQLiteFile(tmpPath); err != nil {
		common.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	installPath, err := prepareImportDatabaseFile(tmpPath)
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if installPath != tmpPath {
		defer os.Remove(installPath)
	}

	a.dbMu.Lock()
	defer a.dbMu.Unlock()
	absPath, err := filepath.Abs(dbPath)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "解析数据库路径失败")
		return
	}
	backupPath, err := common.BackupDatabaseBeforeAlter(a.db, dbPath, "import-database")
	if err != nil {
		log.Printf("Pre-import backup failed: %v", err)
		common.WriteError(w, http.StatusInternalServerError, "备份当前数据库失败，导入已取消")
		return
	}
	log.Printf("Pre-import backup completed: %s", backupPath)

	oldDB := a.db
	oldDB.Close()
	removeSQLiteSidecars(absPath)
	if err := probeDatabaseFileLocked(absPath); err != nil {
		log.Printf("Database file appears locked by external process: %v", err)
		if reopenErr := a.reopenOriginalDatabase(dbPath); reopenErr != nil {
			log.Printf("Reopen original database failed: %v", reopenErr)
		}
		common.WriteError(w, http.StatusConflict, "当前数据库文件正被外部工具使用，请先关闭后再导入")
		return
	}
	if err := service.CopyFile(installPath, absPath); err != nil {
		log.Printf("Copy imported database file failed: %v", err)
		if restoreErr := a.restoreImportedDatabase(dbPath, absPath, backupPath, nil); restoreErr != nil {
			log.Printf("Database restore failed: %v", restoreErr)
		}
		common.WriteError(w, http.StatusInternalServerError, "导入失败，已恢复原数据库：" + err.Error())
		return
	}

	newDB, err := openImportedDatabase(dbPath)
	if err != nil {
		log.Printf("Open imported database failed: %v", err)
		if restoreErr := a.restoreImportedDatabase(dbPath, absPath, backupPath, nil); restoreErr != nil {
			log.Printf("Database restore failed: %v", restoreErr)
		}
		common.WriteError(w, http.StatusInternalServerError, "导入失败，已恢复原数据库：" + err.Error())
		return
	}
	if err := common.SetupSQLite(newDB); err != nil {
		log.Printf("Configure imported database failed: %v", err)
		_ = newDB.Close()
		if restoreErr := a.restoreImportedDatabase(dbPath, absPath, backupPath, newDB); restoreErr != nil {
			log.Printf("Database restore failed: %v", restoreErr)
		}
		common.WriteError(w, http.StatusInternalServerError, "导入失败，已恢复原数据库：" + err.Error())
		return
	}
	if err := EnsureRuntimeSchema(newDB, dbPath); err != nil {
		log.Printf("Upgrade imported database schema failed: %v", err)
		_ = newDB.Close()
		if restoreErr := a.restoreImportedDatabase(dbPath, absPath, backupPath, newDB); restoreErr != nil {
			log.Printf("Database restore failed: %v", restoreErr)
		}
		common.WriteError(w, http.StatusInternalServerError, "导入失败，已恢复原数据库：" + err.Error())
		return
	}
	configureDBLimits(newDB)
	syncRuntimeJWTSecret(newDB, a.cfg.JWTSecret)
	a.notifyDatabaseReplaced(newDB)

	operator, _ := common.CurrentUser(r.Context())
	if len(exportFiles) > 0 {
		if err := extractImportFiles(exportFiles, strings.TrimSpace(a.cfg.UploadDir)); err != nil {
			log.Printf("Extract imported upload files failed: %v", err)
			a.recordAuditEvent(r.Context(), operator.Username, common.ClientIP(r), service.AuditModuleBackup, "导入数据库", filepath.Base(header.Filename), "数据库导入失败：压缩包内的上传文件恢复失败", service.AuditResultFailure)
			common.WriteError(w, http.StatusInternalServerError, "数据库已导入，但压缩包内的上传文件恢复失败")
			return
		}
	}
	log.Printf("Database import completed, backup file: %s", backupPath)
	a.recordAuditEvent(r.Context(), operator.Username, common.ClientIP(r), service.AuditModuleBackup, "导入数据库", filepath.Base(header.Filename), "已手动执行数据库导入（导入前已自动备份原数据库）", service.AuditResultSuccess)
	common.WriteJSON(w, http.StatusOK, map[string]any{"ok": true, "message": "数据库导入成功"})
}

// zipExportFile 记录压缩包内上传文件的临时路径与其在包内的相对路径
type zipExportFile struct {
	source string
	rel    string
}

func extractImportZip(zipPath string) (string, []zipExportFile, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", nil, errors.New("压缩包解析失败")
	}
	defer reader.Close()
	extractDir, err := os.MkdirTemp("", "itdb-import-files-*")
	if err != nil {
		return "", nil, err
	}
	dbPath := ""
	files := make([]zipExportFile, 0)
	for _, entry := range reader.File {
		name := strings.ReplaceAll(entry.Name, "\\", "/")
		if entry.FileInfo().IsDir() || strings.Contains(name, "__MACOSX") {
			continue
		}
		if strings.Contains(name, "..") {
			continue
		}
		if strings.HasSuffix(strings.ToLower(name), ".db") && dbPath == "" {
			target := filepath.Join(extractDir, "import.db")
			if err := extractZipEntry(entry, target); err != nil {
				return "", nil, err
			}
			dbPath = target
			continue
		}
		target := filepath.Join(extractDir, "files", filepath.FromSlash(name))
		if err := extractZipEntry(entry, target); err != nil {
			return "", nil, err
		}
		files = append(files, zipExportFile{
			source: target,
			rel:    filepath.Base(name),
		})
	}
	return dbPath, files, nil
}

func extractZipEntry(entry *zip.File, target string) error {
	source, err := entry.Open()
	if err != nil {
		return err
	}
	defer source.Close()
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	targetFile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer targetFile.Close()
	_, err = io.Copy(targetFile, source)
	return err
}

func extractImportFiles(files []zipExportFile, uploadDir string) error {
	if uploadDir == "" {
		uploadDir = filepath.Join("data", "files")
	}
	for _, item := range files {
		target := filepath.Join(uploadDir, filepath.FromSlash(item.rel))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := copyFile(item.source, target); err != nil {
			return err
		}
	}
	return nil
}

func isZipFile(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	magic := make([]byte, 4)
	if _, err := io.ReadFull(file, magic); err != nil {
		return false
	}
	return string(magic) == "PK\x03\x04"
}

// probeDatabaseFileLocked 通过临时重命名探测数据库文件是否被外部进程占用：
// 外部工具打开的文件句柄会阻止重命名（Windows 共享冲突），导入覆盖前必须先释放
func probeDatabaseFileLocked(absPath string) error {
	probePath := absPath + "-import-lock-probe"
	if err := os.Rename(absPath, probePath); err != nil {
		return err
	}
	if err := os.Rename(probePath, absPath); err != nil {
		log.Printf("Rename database back after lock probe failed: %v", err)
	}
	return nil
}

// reopenOriginalDatabase 导入前的占用检查失败后重新打开原数据库文件，恢复服务依赖
func (a *Router) reopenOriginalDatabase(dbPath string) error {
	reopened, err := openImportedDatabase(dbPath)
	if err != nil {
		return err
	}
	if err := common.SetupSQLite(reopened); err != nil {
		_ = reopened.Close()
		return err
	}
	configureDBLimits(reopened)
	a.notifyDatabaseReplaced(reopened)
	return nil
}

func (a *Router) restoreImportedDatabase(dbPath, absPath, backupPath string, current *sql.DB) error {
	if current != nil {
		_ = current.Close()
	}
	removeSQLiteSidecars(absPath)
	if err := service.CopyFile(backupPath, absPath); err != nil {
		return err
	}
	restored, err := openImportedDatabase(dbPath)
	if err != nil {
		return err
	}
	if err := common.SetupSQLite(restored); err != nil {
		_ = restored.Close()
		return err
	}
	if err := EnsureRuntimeSchema(restored, dbPath); err != nil {
		_ = restored.Close()
		return err
	}
	configureDBLimits(restored)
	a.notifyDatabaseReplaced(restored)
	return nil
}

func openImportedDatabase(dbPath string) (*sql.DB, error) {
	db, e := sql.Open("sqlite", dbPath)
	if e != nil {
		return nil, e
	}
	if e = db.Ping(); e != nil {
		_ = db.Close()
		return nil, e
	}
	return db, nil
}
func configureDBLimits(db *sql.DB) {
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)
	db.SetConnMaxIdleTime(0)
}
func removeSQLiteSidecars(absPath string) {
	_ = os.Remove(absPath + "-wal")
	_ = os.Remove(absPath + "-shm")
}

func copyFile(src, dst string) error { return service.CopyFile(src, dst) }

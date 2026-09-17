package service

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"itdb-backend/internal/repository"
)

type BackupService struct {
	db     repository.Executor
	dbPath string
}

func NewBackupService(db repository.Executor, dbPath string) *BackupService {
	return &BackupService{db: db, dbPath: strings.TrimSpace(dbPath)}
}
func (s *BackupService) Available() bool {
	return s.dbPath != "" && !strings.EqualFold(s.dbPath, ":memory:")
}
func (s *BackupService) CreateTemporaryDatabaseBackup(ctx context.Context) (string, func(), error) {
	if !s.Available() {
		return "", func() {}, errors.New("database backup is unavailable for in-memory database")
	}
	dir, e := os.MkdirTemp("", "itdb-db-backup-*")
	if e != nil {
		return "", func() {}, e
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	target := filepath.Join(dir, "itdb.db")
	if e = s.vacuumInto(ctx, target); e != nil {
		cleanup()
		return "", func() {}, e
	}
	return target, cleanup, nil
}

// CreateScheduledBackup 创建定时备份：数据库引用的上传文件存在时打包为 zip（文件统一在 files 目录下），
// 未引用文件或引用文件全部不存在时仅导出 db 文件；返回备份路径与不存在的引用文件名
func (s *BackupService) CreateScheduledBackup(ctx context.Context, now time.Time, uploadDir string) (string, []string, error) {
	if !s.Available() {
		return "", nil, errors.New("database backup is unavailable for in-memory database")
	}
	abs, e := filepath.Abs(s.dbPath)
	if e != nil {
		return "", nil, e
	}
	dir := filepath.Join(filepath.Dir(abs), "backups")
	if e = os.MkdirAll(dir, 0o755); e != nil {
		return "", nil, e
	}
	stamp := now.Format("20060102-150405")
	existing, missing, e := s.ReferencedUploadFiles(ctx, uploadDir)
	if e != nil {
		return "", nil, e
	}
	tmpDir, e := os.MkdirTemp("", "itdb-scheduled-backup-*")
	if e != nil {
		return "", nil, e
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()
	dumpPath := filepath.Join(tmpDir, fmt.Sprintf("itdb-%s.db", stamp))
	if e = s.vacuumInto(ctx, dumpPath); e != nil {
		return "", nil, e
	}
	if len(existing) == 0 {
		target := filepath.Join(dir, fmt.Sprintf("itdb-%s.db", stamp))
		if e = CopyFile(dumpPath, target); e != nil {
			return "", nil, e
		}
		return target, missing, nil
	}
	target := filepath.Join(dir, fmt.Sprintf("itdb-%s.zip", stamp))
	if e = writeScheduledBackupZip(target, dumpPath, stamp, uploadDir, existing); e != nil {
		return "", nil, e
	}
	return target, missing, nil
}

func (s *BackupService) vacuumInto(ctx context.Context, target string) error {
	escaped := strings.ReplaceAll(target, "'", "''")
	_, e := s.db.ExecContext(ctx, fmt.Sprintf("VACUUM INTO '%s'", escaped))
	return e
}

// ReferencedUploadFiles 按上传目录检查数据库引用文件（文件附件与地点平面图）的存在性，
// 返回存在与不存在的文件名列表
func (s *BackupService) ReferencedUploadFiles(ctx context.Context, uploadDir string) (existing, missing []string, err error) {
	referenced, err := s.collectReferencedUploadFiles(ctx)
	if err != nil {
		return nil, nil, err
	}
	existing, missing = splitExistingFiles(uploadDir, referenced)
	return existing, missing, nil
}

// collectReferencedUploadFiles 收集数据库引用的上传文件名：文件附件与地点平面图，去空后返回
func (s *BackupService) collectReferencedUploadFiles(ctx context.Context) ([]string, error) {
	queries := []string{
		`SELECT DISTINCT TRIM(fname) FROM files WHERE TRIM(COALESCE(fname, '')) <> ''`,
		`SELECT DISTINCT TRIM(floorplanfn) FROM locations WHERE TRIM(COALESCE(floorplanfn, '')) <> ''`,
	}
	names := make([]string, 0)
	for _, query := range queries {
		rows, e := s.db.QueryContext(ctx, query)
		if e != nil {
			return nil, e
		}
		for rows.Next() {
			var name string
			if e = rows.Scan(&name); e != nil {
				rows.Close()
				return nil, e
			}
			names = append(names, name)
		}
		if e = rows.Err(); e != nil {
			rows.Close()
			return nil, e
		}
		rows.Close()
	}
	return names, nil
}

// splitExistingFiles 按上传目录检查引用文件的存在性，返回存在与不存在的文件名（去重）
func splitExistingFiles(uploadDir string, names []string) (existing, missing []string) {
	seen := map[string]bool{}
	for _, name := range names {
		name = filepath.Base(filepath.ToSlash(strings.TrimSpace(name)))
		if name == "" || name == "." || name == ".." || seen[name] {
			continue
		}
		seen[name] = true
		if uploadDir == "" {
			missing = append(missing, name)
			continue
		}
		if _, e := os.Stat(filepath.Join(uploadDir, name)); e != nil {
			missing = append(missing, name)
			continue
		}
		existing = append(existing, name)
	}
	return existing, missing
}

// writeScheduledBackupZip 将备份库与引用文件打包为 zip，库在包根目录、文件统一放在 files 目录下
func writeScheduledBackupZip(target, dumpPath, stamp, uploadDir string, files []string) error {
	zipFile, e := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if e != nil {
		return e
	}
	defer func() { _ = zipFile.Close() }()
	zipWriter := zip.NewWriter(zipFile)
	defer func() { _ = zipWriter.Close() }()
	if e = AddZipEntry(zipWriter, dumpPath, fmt.Sprintf("itdb-%s.db", stamp)); e != nil {
		return e
	}
	for _, name := range files {
		if e = AddZipEntry(zipWriter, filepath.Join(uploadDir, name), "files/"+filepath.ToSlash(name)); e != nil {
			return e
		}
	}
	return zipWriter.Close()
}

// AddZipEntry 将文件以指定名称写入 zip 包
func AddZipEntry(zipWriter *zip.Writer, path, name string) error {
	file, e := os.Open(path)
	if e != nil {
		return e
	}
	defer file.Close()
	info, e := file.Stat()
	if e != nil {
		return e
	}
	header, e := zip.FileInfoHeader(info)
	if e != nil {
		return e
	}
	header.Name = name
	header.Method = zip.Deflate
	writer, e := zipWriter.CreateHeader(header)
	if e != nil {
		return e
	}
	_, e = io.Copy(writer, file)
	return e
}

// CleanupExpiredScheduledBackups 删除备份目录中创建时间超过保留天数的定时备份文件（db 与 zip），
// 返回清理数量；保留天数小于等于 0 表示永久保留。
func (s *BackupService) CleanupExpiredScheduledBackups(now time.Time, retentionDays int) (int, error) {
	if retentionDays <= 0 || !s.Available() {
		return 0, nil
	}
	abs, e := filepath.Abs(s.dbPath)
	if e != nil {
		return 0, e
	}
	dir := filepath.Join(filepath.Dir(abs), "backups")
	entries, e := os.ReadDir(dir)
	if e != nil {
		if os.IsNotExist(e) {
			return 0, nil
		}
		return 0, e
	}
	cutoff := now.AddDate(0, 0, -retentionDays)
	removed := 0
	for _, entry := range entries {
		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if entry.IsDir() || !strings.HasPrefix(name, "itdb-") || (ext != ".db" && ext != ".zip") {
			continue
		}
		stamp := strings.TrimSuffix(strings.TrimPrefix(name, "itdb-"), filepath.Ext(name))
		created, err := time.ParseInLocation("20060102-150405", stamp, time.Local)
		if err != nil {
			continue
		}
		if created.Before(cutoff) {
			if os.Remove(filepath.Join(dir, name)) == nil {
				removed++
			}
		}
	}
	return removed, nil
}

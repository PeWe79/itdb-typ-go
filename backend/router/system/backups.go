package system

import (
	"archive/zip"
	"fmt"
	"itdb-backend/router/common"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"itdb-backend/internal/service"
)

// handleDownloadDatabaseBackup 下载数据库备份：
// 默认仅包含数据库文件 itdb-YYYYMMDD.db；携带 files 参数时，
// 将上传目录文件一并打包为 itdb-YYYYMMDD.zip。
func (a *Router) handleDownloadDatabaseBackup(w http.ResponseWriter, r *http.Request) {
	includeFiles := r.URL.Query().Get("files") == "1" || strings.EqualFold(r.URL.Query().Get("files"), "true")
	operator, _ := common.CurrentUser(r.Context())
	backupPath, cleanup, e := a.backupWorkflow.CreateTemporaryDatabaseBackup(r.Context())
	if e != nil {
		a.recordAuditEvent(r.Context(), operator.Username, common.ClientIP(r), service.AuditModuleBackup, "手动备份数据库", "-", AuditBackupFailureDetail(e), service.AuditResultFailure)
		status := http.StatusInternalServerError
		if strings.Contains(e.Error(), "in-memory") {
			status = http.StatusBadRequest
		}
		common.WriteError(w, status, e.Error())
		return
	}
	defer cleanup()

	stamp := time.Now().Format("20060102")
	if !includeFiles {
		name := fmt.Sprintf("itdb-%s.db", stamp)
		a.recordAuditEvent(r.Context(), operator.Username, common.ClientIP(r), service.AuditModuleBackup, "手动备份数据库", name, "已手动执行数据库备份", service.AuditResultSuccess)
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", name))
		serveFileContent(w, r, backupPath, name)
		return
	}

	uploadDir := strings.TrimSpace(a.cfg.UploadDir)
	existing, missing, e := a.backupWorkflow.ReferencedUploadFiles(r.Context(), uploadDir)
	if e != nil {
		a.recordAuditEvent(r.Context(), operator.Username, common.ClientIP(r), service.AuditModuleBackup, "手动备份数据库", "-", AuditBackupFailureDetail(e), service.AuditResultFailure)
		common.WriteError(w, http.StatusInternalServerError, e.Error())
		return
	}
	if len(existing) == 0 {
		name := fmt.Sprintf("itdb-%s.db", stamp)
		detail := appendMissingFilesDetail("已手动执行数据库备份", missing, "、")
		a.recordAuditEvent(r.Context(), operator.Username, common.ClientIP(r), service.AuditModuleBackup, "手动备份数据库", name, detail, service.AuditResultSuccess)
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", name))
		serveFileContent(w, r, backupPath, name)
		return
	}

	name := fmt.Sprintf("itdb-%s.zip", stamp)
	detail := appendMissingFilesDetail("已手动执行数据库备份（包含上传文件一并打包）", missing, "、")
	a.recordAuditEvent(r.Context(), operator.Username, common.ClientIP(r), service.AuditModuleBackup, "手动备份数据库", name, detail, service.AuditResultSuccess)
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", name))
	zipWriter := zip.NewWriter(w)
	defer func() { _ = zipWriter.Close() }()

	if err := service.AddZipEntry(zipWriter, backupPath, fmt.Sprintf("itdb-%s.db", stamp)); err != nil {
		log.Printf("Backup zip database entry failed: %v", err)
		return
	}
	for _, fileName := range existing {
		if err := service.AddZipEntry(zipWriter, filepath.Join(uploadDir, fileName), "files/"+filepath.ToSlash(fileName)); err != nil {
			log.Printf("Backup zip upload file %s failed: %v", fileName, err)
			return
		}
	}
}

// appendMissingFilesDetail 在备份详情后追加以指定分隔符合并的缺失文件说明
func appendMissingFilesDetail(detail string, missing []string, separator string) string {
	if len(missing) == 0 {
		return detail
	}
	return detail + "，其中 " + strings.Join(missing, separator) + " 文件不存在"
}

func serveFileContent(w http.ResponseWriter, r *http.Request, path, name string) {
	file, err := os.Open(path)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", name))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))
	http.ServeContent(w, r, name, info.ModTime(), file)
}

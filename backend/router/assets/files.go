package assets

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"itdb-backend/internal/common/primitives"
	"itdb-backend/internal/service"
	"itdb-backend/router/common"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

func (a *Router) handleListFiles(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("search"))

	baseQuery := `SELECT files.id, files.title, files.fname, files.type, files.date, filetypes.typedesc,
        (SELECT COUNT(*) FROM software2file WHERE fileid = files.id) +
        (SELECT COUNT(*) FROM invoice2file WHERE fileid = files.id) +
        (SELECT COUNT(*) FROM item2file WHERE fileid = files.id) +
        (SELECT COUNT(*) FROM contract2file WHERE fileid = files.id) AS links
        FROM files
        LEFT JOIN filetypes ON filetypes.id = files.type`

	where := ""
	args := []interface{}{}
	if search != "" {
		where = `WHERE (
            CAST(file_view.id AS TEXT) LIKE ? OR
            file_view.typedesc LIKE ? OR
            file_view.title LIKE ? OR
            file_view.fname LIKE ? OR
            (CASE WHEN COALESCE(file_view.date, 0) > 0 THEN date(file_view.date, 'unixepoch') ELSE '' END) LIKE ? OR
            CAST(file_view.links AS TEXT) LIKE ?
        )`
		q := "%" + search + "%"
		args = append(args, q, q, q, q, q, q)
	}

	query := `SELECT * FROM (` + baseQuery + `) AS file_view
        ` + where + `
        ORDER BY file_view.id DESC`
	rows, err := a.fetchRows(query, args...)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.WriteJSON(w, http.StatusOK, rows)
}

func (a *Router) handleGetFile(w http.ResponseWriter, r *http.Request) {
	id, err := primitives.IntParam(chi.URLParam(r, "id"))
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}

	rows, err := a.fetchRows(`SELECT * FROM files WHERE id = ?`, id)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(rows) == 0 {
		common.WriteError(w, http.StatusNotFound, "file not found")
		return
	}

	payload := rows[0]
	relations, err := a.detailRelations.Load(r.Context(), "files", id)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for key, value := range relations {
		payload[key] = value
	}

	common.WriteJSON(w, http.StatusOK, payload)
}

func (a *Router) handleDownloadFile(w http.ResponseWriter, r *http.Request) {
	id, err := primitives.IntParam(chi.URLParam(r, "id"))
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if !a.hasPermission(r, assetPermission("files", "read")) {
		if !a.hasPermission(r, assetPermission("invoices", "read")) {
			common.WriteError(w, http.StatusForbidden, "permission denied")
			return
		}
		var linked int
		if err := a.domains.Files.QueryRow(`SELECT COUNT(*) FROM invoice2file WHERE fileid = ?`, id).Scan(&linked); err != nil {
			common.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if linked == 0 {
			common.WriteError(w, http.StatusForbidden, "permission denied")
			return
		}
	}

	var storedName, title sql.NullString
	if err := a.domains.Files.QueryRow(`SELECT fname, title FROM files WHERE id = ?`, id).Scan(&storedName, &title); err != nil {
		if err == sql.ErrNoRows {
			common.WriteError(w, http.StatusNotFound, "file not found")
			return
		}
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	name := strings.TrimSpace(storedName.String)
	if name == "" {
		common.WriteError(w, http.StatusNotFound, "file not found")
		return
	}
	path := filepath.Join(a.cfg.UploadDir, name)
	if _, err := os.Stat(path); err != nil {
		common.WriteError(w, http.StatusNotFound, "file not found")
		return
	}

	downloadName := name
	if t := strings.TrimSpace(title.String); t != "" {
		downloadName = fmt.Sprintf("%s%s", t, filepath.Ext(name))
	}
	common.SetContentDispositionHeader(w, "attachment", downloadName)
	http.ServeFile(w, r, path)
}

func (a *Router) handleCreateFile(w http.ResponseWriter, r *http.Request) {
	user, _ := common.CurrentUser(r.Context())
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	typeID, _ := primitives.IntParam(strings.TrimSpace(r.FormValue("typeId")))
	dateRaw := strings.TrimSpace(r.FormValue("date"))
	fileHeader, err := common.GetMultipartFileHeader(r.MultipartForm, "file")
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if title == "" || typeID <= 0 {
		common.WriteError(w, http.StatusBadRequest, "title and typeId are required")
		return
	}

	var existing int64
	if err := a.domains.Files.QueryRowContext(r.Context(), `SELECT id FROM files WHERE LOWER(TRIM(COALESCE(title,''))) = LOWER(TRIM(?)) LIMIT 1`, title).Scan(&existing); err == nil {
		common.WriteError(w, http.StatusConflict, fmt.Sprintf("文件标题 %s 已存在", title))
		return
	} else if !errors.Is(err, sql.ErrNoRows) {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	dateValue, err := primitives.ParseDateInput(dateRaw)
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid date")
		return
	}

	storedName, err := a.storeUploadedFile(fileHeader, typeID, title)
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	tx, err := a.domains.Files.BeginTx(r.Context(), nil)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()

	res, err := tx.Exec(`INSERT INTO files (title, type, fname, uploader, uploaddate, date) VALUES (?, ?, ?, ?, ?, ?)`,
		title, typeID, storedName, user.Username, time.Now().Unix(), dateValue)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	newID, _ := res.LastInsertId()

	itemLinks := primitives.ParseIDCSV(r.FormValue("itemLinks"))
	softwareLinks := primitives.ParseIDCSV(r.FormValue("softwareLinks"))
	invoiceLinks := primitives.ParseIDCSV(r.FormValue("invoiceLinks"))
	contractLinks := primitives.ParseIDCSV(r.FormValue("contractLinks"))

	if err := a.relations.ReplaceIDLinks(tx, `DELETE FROM item2file WHERE fileid = ?`, `INSERT INTO item2file (fileid, itemid) VALUES (?, ?)`, newID, itemLinks); err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.relations.ReplaceIDLinks(tx, `DELETE FROM software2file WHERE fileid = ?`, `INSERT INTO software2file (fileid, softwareid) VALUES (?, ?)`, newID, softwareLinks); err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.relations.ReplaceIDLinks(tx, `DELETE FROM invoice2file WHERE fileid = ?`, `INSERT INTO invoice2file (fileid, invoiceid) VALUES (?, ?)`, newID, invoiceLinks); err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.relations.ReplaceIDLinks(tx, `DELETE FROM contract2file WHERE fileid = ?`, `INSERT INTO contract2file (fileid, contractid) VALUES (?, ?)`, newID, contractLinks); err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	a.recordFileAudit(r.Context(), user, common.ClientIP(r), "新增文件", "已创建", newID, title)
	common.WriteJSON(w, http.StatusCreated, map[string]int64{"id": newID})
}

func (a *Router) handleUpdateFile(w http.ResponseWriter, r *http.Request) {
	user, _ := common.CurrentUser(r.Context())
	id, err := primitives.IntParam(chi.URLParam(r, "id"))
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	typeID, _ := primitives.IntParam(strings.TrimSpace(r.FormValue("typeId")))
	dateRaw := strings.TrimSpace(r.FormValue("date"))
	if title == "" || typeID <= 0 {
		common.WriteError(w, http.StatusBadRequest, "title and typeId are required")
		return
	}

	var oldTitle string
	if err := a.domains.Files.QueryRowContext(r.Context(), `SELECT COALESCE(title,'') FROM files WHERE id = ?`, id).Scan(&oldTitle); err != nil && !errors.Is(err, sql.ErrNoRows) {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	} else if err == nil && title != strings.TrimSpace(oldTitle) {
		var existing int64
		if err := a.domains.Files.QueryRowContext(r.Context(), `SELECT id FROM files WHERE LOWER(TRIM(COALESCE(title,''))) = LOWER(TRIM(?)) AND id <> ? LIMIT 1`, title, id).Scan(&existing); err == nil {
			common.WriteError(w, http.StatusConflict, fmt.Sprintf("文件标题 %s 已存在", title))
			return
		} else if !errors.Is(err, sql.ErrNoRows) {
			common.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	dateValue, err := primitives.ParseDateInput(dateRaw)
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid date")
		return
	}

	fileHeader, _ := common.GetMultipartFileHeader(r.MultipartForm, "file")
	var oldName string
	_ = a.domains.Files.QueryRow(`SELECT fname FROM files WHERE id = ?`, id).Scan(&oldName)
	var storedName string
	if fileHeader != nil {
		storedName, err = a.storeUploadedFile(fileHeader, typeID, title)
		if err != nil {
			common.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	tx, err := a.domains.Files.BeginTx(r.Context(), nil)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()

	if storedName != "" {
		_, err = tx.Exec(`UPDATE files SET title = ?, type = ?, fname = ?, uploader = ?, uploaddate = ?, date = ? WHERE id = ?`,
			title, typeID, storedName, user.Username, time.Now().Unix(), dateValue, id)
	} else {
		_, err = tx.Exec(`UPDATE files SET title = ?, type = ?, uploader = ?, uploaddate = ?, date = ? WHERE id = ?`,
			title, typeID, user.Username, time.Now().Unix(), dateValue, id)
	}
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	itemLinks := primitives.ParseIDCSV(r.FormValue("itemLinks"))
	softwareLinks := primitives.ParseIDCSV(r.FormValue("softwareLinks"))
	invoiceLinks := primitives.ParseIDCSV(r.FormValue("invoiceLinks"))
	contractLinks := primitives.ParseIDCSV(r.FormValue("contractLinks"))
	if err := a.relations.ReplaceIDLinks(tx, `DELETE FROM item2file WHERE fileid = ?`, `INSERT INTO item2file (fileid, itemid) VALUES (?, ?)`, id, itemLinks); err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.relations.ReplaceIDLinks(tx, `DELETE FROM software2file WHERE fileid = ?`, `INSERT INTO software2file (fileid, softwareid) VALUES (?, ?)`, id, softwareLinks); err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.relations.ReplaceIDLinks(tx, `DELETE FROM invoice2file WHERE fileid = ?`, `INSERT INTO invoice2file (fileid, invoiceid) VALUES (?, ?)`, id, invoiceLinks); err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.relations.ReplaceIDLinks(tx, `DELETE FROM contract2file WHERE fileid = ?`, `INSERT INTO contract2file (fileid, contractid) VALUES (?, ?)`, id, contractLinks); err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if storedName != "" && strings.TrimSpace(oldName) != "" && oldName != storedName {
		_ = os.Remove(filepath.Join(a.cfg.UploadDir, oldName))
	}

	changeNote := strings.TrimSpace(r.FormValue("changeNote"))
	if changeNote != "" {
		a.recordFileAudit(r.Context(), user, common.ClientIP(r), "更新文件", "已更新："+changeNote+"配置", id, title)
	}
	common.WriteJSON(w, http.StatusOK, map[string]int64{"id": id})
}

func (a *Router) handleDeleteFile(w http.ResponseWriter, r *http.Request) {
	user, _ := common.CurrentUser(r.Context())
	id, err := primitives.IntParam(chi.URLParam(r, "id"))
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var fileTitle string
	_ = a.domains.Files.QueryRowContext(r.Context(), `SELECT COALESCE(title,'') FROM files WHERE id = ?`, id).Scan(&fileTitle)

	tx, err := a.domains.Files.BeginTx(r.Context(), nil)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()

	linkSources := []struct {
		table string
		noun  string
	}{
		{"item2file", "硬件"},
		{"software2file", "软件"},
		{"invoice2file", "单据"},
		{"contract2file", "合同"},
	}
	var messages []string
	for _, source := range linkSources {
		var count int64
		if err := tx.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM `+source.table+` WHERE fileid = ?`, id).Scan(&count); err != nil {
			common.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if count > 0 {
			messages = append(messages, fmt.Sprintf("该文件已被 %d 条%s记录使用，无法删除", count, source.noun))
		}
	}
	if len(messages) > 0 {
		common.WriteError(w, http.StatusConflict, strings.Join(messages, "\n"))
		return
	}
	if err := a.relations.DeleteFile(tx, id); err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	a.recordFileAudit(r.Context(), user, common.ClientIP(r), "删除文件", "已删除", id, fileTitle)
	common.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// recordFileAudit 记录文件实体的显式审计事件
func (a *Router) recordFileAudit(ctx context.Context, user SessionUser, ip, action, phrase string, id int64, title string) {
	_ = a.auditService().RecordEvent(ctx, user.Username, ip, service.AuditEvent{
		Module: service.AuditModuleAssets,
		Action: action,
		Target: formatFileTarget(id),
		Detail: fmt.Sprintf("%s[ID: %d] %s", strings.TrimSpace(title), id, phrase),
		Result: service.AuditResultSuccess,
	})
}

// formatFileTarget 文件实体的目标列内容
func formatFileTarget(id int64) string {
	return fmt.Sprintf("文件编号 %d", id)
}

func (a *Router) storeUploadedFile(h *multipart.FileHeader, fileTypeID int64, title string) (string, error) {
	if a.files == nil {
		return "", errors.New("file storage service is unavailable")
	}
	return a.files.StoreUploadedFile(h, fileTypeID, title)
}

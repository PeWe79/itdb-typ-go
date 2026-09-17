package assets

import (
	"context"
	"encoding/json"
	"itdb-backend/internal/common/primitives"
	"itdb-backend/internal/service"
	"itdb-backend/router/common"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

func (a *Router) handleListDictionaries(w http.ResponseWriter, r *http.Request) {
	payload := map[string]interface{}{}
	tables := map[string]string{
		"itemtypes":        `SELECT id, typedesc, hassoftware FROM itemtypes ORDER BY id ASC`,
		"filetypes":        `SELECT id, typedesc FROM filetypes ORDER BY id`,
		"statustypes":      `SELECT id, statusdesc, color FROM statustypes ORDER BY id`,
		"dpttypes":         `SELECT id, dptname FROM dpttypes ORDER BY id ASC`,
		"contracttypes":    `SELECT id, name FROM contracttypes ORDER BY id`,
		"contractsubtypes": `SELECT id, contypeid, name FROM contractsubtypes ORDER BY id ASC`,
		"tags": `SELECT
            tags.id,
            tags.name,
            (SELECT COUNT(*) FROM tag2item JOIN items ON items.id = tag2item.itemid WHERE tag2item.tagid = tags.id) AS itemCount,
            (SELECT COUNT(*) FROM tag2software JOIN software ON software.id = tag2software.softwareid WHERE tag2software.tagid = tags.id) AS softwareCount
        FROM tags ORDER BY id ASC`,
	}
	for name, query := range tables {
		if !a.hasPermission(r, dictionaryPermission(name, "read")) {
			continue
		}
		rows, err := a.fetchRows(query)
		if err != nil {
			common.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		payload[name] = rows
	}
	common.WriteJSON(w, http.StatusOK, payload)
}

// recordDictionaryAudit 记录资料管理字典的显式审计事件
func (a *Router) recordDictionaryAudit(ctx context.Context, user SessionUser, ip string, event service.AuditEvent) {
	event.Module = service.AuditModuleCatalog
	_ = a.auditService().RecordEvent(ctx, user.Username, ip, event)
}

// dictionaryCreateEvent 依据创建请求体构建新增审计事件
func (a *Router) dictionaryCreateEvent(ctx context.Context, name string, id int64, body map[string]interface{}) (service.AuditEvent, bool) {
	meta, err := service.DictionaryAuditMetaByName(name)
	if err != nil {
		return service.AuditEvent{}, false
	}
	rowName := strings.TrimSpace(primitives.AsString(body[meta.Column]))
	color := strings.TrimSpace(primitives.AsString(body["color"]))
	hasSoftware := primitives.AsInt64(body["hassoftware"]) == 1
	parentName := service.DictionaryParentName(ctx, a.db, primitives.AsInt64(body["contypeid"]))
	return service.AuditEvent{
		Action: "新增" + meta.Label,
		Target: service.DictionaryTargetText(rowName),
		Detail: service.DictionaryCreatedDetail(meta, rowName, color, hasSoftware, parentName),
		Result: service.AuditResultSuccess,
	}, true
}

// dictionaryDeleteEvent 依据库中旧行构建删除审计事件
func (a *Router) dictionaryDeleteEvent(ctx context.Context, name string, id int64) (service.AuditEvent, bool) {
	meta, err := service.DictionaryAuditMetaByName(name)
	if err != nil {
		return service.AuditEvent{}, false
	}
	rowName, color, hasSoftware, parentID, loadErr := service.LoadDictionaryAuditValues(ctx, a.db, meta, id)
	if loadErr != nil {
		return service.AuditEvent{}, false
	}
	return service.AuditEvent{
		Action: "删除" + meta.Label,
		Target: service.DictionaryTargetText(rowName),
		Detail: service.DictionaryDeletedDetail(meta, rowName, color, hasSoftware, service.DictionaryParentName(ctx, a.db, parentID)),
		Result: service.AuditResultSuccess,
	}, true
}

// recordDictionaryFailure 字典操作失败时补记失败审计事件
func (a *Router) recordDictionaryFailure(r *http.Request, user SessionUser, name string, outcome service.AuditOutcome, id int64, body map[string]interface{}, auditErr error) {
	meta, err := service.DictionaryAuditMetaByName(name)
	if err != nil {
		return
	}
	rowName := strings.TrimSpace(primitives.AsString(body[meta.Column]))
	target := rowName
	if rowName == "" || id > 0 {
		if oldName, _, _, _, loadErr := service.LoadDictionaryAuditValues(r.Context(), a.db, meta, id); loadErr == nil && oldName != "" {
			target = service.DictionaryTargetText(oldName)
			if rowName == "" {
				rowName = oldName
			}
		}
	}
	a.recordDictionaryAudit(r.Context(), user, common.ClientIP(r), service.AuditEvent{
		Action: service.AssetOutcomeAction(outcome, meta.Label),
		Target: target,
		Detail: service.DictionaryFailureDetail(rowName, outcome, service.ConflictReason(auditErr)),
		Result: service.AuditResultFailure,
	})
}

func (a *Router) handleCreateDictionaryRow(w http.ResponseWriter, r *http.Request) {
	user, _ := common.CurrentUser(r.Context())
	name := chi.URLParam(r, "name")
	if !a.hasPermission(r, dictionaryPermission(name, "manage")) {
		common.WriteError(w, http.StatusForbidden, "permission denied")
		return
	}
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	query, args, err := service.DictionaryInsert(name, body)
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	tx, err := a.domains.Dictionaries.BeginTx(r.Context(), nil)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()

	if err := service.EnforceDictionaryUniqueText(tx, name, body, 0); err != nil {
		common.WriteError(w, http.StatusConflict, err.Error())
		return
	}

	res, err := tx.ExecContext(r.Context(), query, args...)
	if err != nil {
		_ = tx.Rollback()
		if !silentAudit(r) {
			a.recordDictionaryFailure(r, user, name, service.AuditOutcomeCreate, 0, body, err)
		}
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		if !silentAudit(r) {
			a.recordDictionaryFailure(r, user, name, service.AuditOutcomeCreate, 0, body, err)
		}
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	id, _ := res.LastInsertId()
	if event, ok := a.dictionaryCreateEvent(r.Context(), name, id, body); ok && !silentAudit(r) {
		a.recordDictionaryAudit(r.Context(), user, common.ClientIP(r), event)
	}
	common.WriteJSON(w, http.StatusCreated, map[string]int64{"id": id})
}

// silentAudit 批量导入期间的逐行写入不单独记录审计，由导入完成后统一上报
func silentAudit(r *http.Request) bool {
	return r.URL.Query().Get("audit") == "0"
}

func (a *Router) handleUpdateDictionaryRow(w http.ResponseWriter, r *http.Request) {
	user, _ := common.CurrentUser(r.Context())
	name := chi.URLParam(r, "name")
	if !a.hasPermission(r, dictionaryPermission(name, "manage")) {
		common.WriteError(w, http.StatusForbidden, "permission denied")
		return
	}
	id, err := primitives.IntParam(chi.URLParam(r, "id"))
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	meta, err := service.DictionaryAuditMetaByName(name)
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	oldName, oldColor, oldSoftware, oldParent, loadErr := service.LoadDictionaryAuditValues(r.Context(), a.domains.Dictionaries, meta, id)
	if loadErr != nil {
		common.WriteError(w, http.StatusNotFound, loadErr.Error())
		return
	}
	tx, err := a.domains.Dictionaries.BeginTx(r.Context(), nil)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()

	if err := service.EnforceDictionaryUpdateRules(tx, name, id); err != nil {
		common.WriteError(w, http.StatusConflict, err.Error())
		return
	}
	if err := service.EnforceDictionaryUniqueText(tx, name, body, id); err != nil {
		common.WriteError(w, http.StatusConflict, err.Error())
		return
	}
	query, args, err := service.DictionaryUpdate(name, id, body)
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, err := tx.ExecContext(r.Context(), query, args...); err != nil {
		_ = tx.Rollback()
		a.recordDictionaryFailure(r, user, name, service.AuditOutcomeUpdate, id, body, err)
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		a.recordDictionaryFailure(r, user, name, service.AuditOutcomeUpdate, id, body, err)
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	detail, changed := service.DictionaryUpdatedDetail(r.Context(), a.db, meta, id, oldName, oldColor, oldSoftware, oldParent, body)
	if changed {
		a.recordDictionaryAudit(r.Context(), user, common.ClientIP(r), service.AuditEvent{
			Action: "更新" + meta.Label,
			Target: service.DictionaryTargetText(oldName),
			Detail: detail,
			Result: service.AuditResultSuccess,
		})
	}
	common.WriteJSON(w, http.StatusOK, map[string]int64{"id": id})
}

func (a *Router) handleDeleteDictionaryRow(w http.ResponseWriter, r *http.Request) {
	user, _ := common.CurrentUser(r.Context())
	name := chi.URLParam(r, "name")
	if !a.hasPermission(r, dictionaryPermission(name, "manage")) {
		common.WriteError(w, http.StatusForbidden, "permission denied")
		return
	}
	id, err := primitives.IntParam(chi.URLParam(r, "id"))
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	event, eventOK := a.dictionaryDeleteEvent(r.Context(), name, id)
	tx, err := a.domains.Dictionaries.BeginTx(r.Context(), nil)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()

	if err := service.EnforceDictionaryDeleteRules(tx, name, id); err != nil {
		common.WriteError(w, http.StatusConflict, err.Error())
		return
	}
	query, err := service.DictionaryDelete(name)
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, err := tx.Exec(query, id); err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if eventOK {
		a.recordDictionaryAudit(r.Context(), user, common.ClientIP(r), event)
	}
	common.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

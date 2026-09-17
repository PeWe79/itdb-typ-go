package assets

import (
	"database/sql"
	"encoding/json"
	"itdb-backend/internal/common/primitives"
	"itdb-backend/pkg/database"
	"itdb-backend/router/common"
	"strings"

	"fmt"
	"itdb-backend/internal/domain"
	"net/http"

	"errors"
	"itdb-backend/internal/service"

	"github.com/go-chi/chi/v5"
)

func (a *Router) handleListLabelItems(w http.ResponseWriter, r *http.Request) {
	limit := primitives.IntParamDefault(r.URL.Query().Get("limit"), 1000)
	offset := primitives.IntParamDefault(r.URL.Query().Get("offset"), 0)
	rows, e := a.labelWorkflow.Items(r.Context(), r.URL.Query().Get("search"), r.URL.Query().Get("orderBy"), limit, offset)
	if e != nil {
		common.WriteError(w, http.StatusInternalServerError, e.Error())
		return
	}
	defer rows.Close()
	out, e := database.RowsToMaps(rows)
	if e != nil {
		common.WriteError(w, http.StatusInternalServerError, e.Error())
		return
	}
	for i := range out {
		id := primitives.AsInt64(out[i]["id"])
		typ := primitives.AsString(out[i]["itemtype"])
		man := primitives.AsString(out[i]["manufacturer"])
		model := primitives.AsString(out[i]["model"])
		sn := primitives.AsString(out[i]["sn"])
		if sn == "" {
			sn = primitives.AsString(out[i]["sn3"])
		}
		label := primitives.AsString(out[i]["label"])
		suffix := ""
		if label != "" {
			suffix = "-" + label
		}
		out[i]["text"] = fmt.Sprintf("%04d-%s | %s-%s-%s%s", id, typ, man, model, sn, suffix)
	}
	common.WriteJSON(w, http.StatusOK, out)
}

func (a *Router) handleListLabelPresets(w http.ResponseWriter, r *http.Request) {
	rows, e := a.labelPresetWorkflow.List(r.Context())
	if e != nil {
		common.WriteError(w, http.StatusInternalServerError, e.Error())
		return
	}
	defer rows.Close()
	out, e := database.RowsToMaps(rows)
	if e != nil {
		common.WriteError(w, http.StatusInternalServerError, e.Error())
		return
	}
	common.WriteJSON(w, http.StatusOK, out)
}
func (a *Router) handleCreateLabelPreset(w http.ResponseWriter, r *http.Request) {
	var body map[string]interface{}
	if e := json.NewDecoder(r.Body).Decode(&body); e != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	u, _ := common.CurrentUser(r.Context())
	id, created, e := a.labelPresetWorkflow.Create(r.Context(), service.UserActor{Username: u.Username, IP: common.ClientIP(r)}, body)
	if e != nil {
		if e.Error() == "name is required" {
			common.WriteError(w, http.StatusBadRequest, e.Error())
		} else {
			common.WriteError(w, http.StatusInternalServerError, e.Error())
		}
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	common.WriteJSON(w, status, map[string]int64{"id": id})
}
func (a *Router) handleDeleteLabelPreset(w http.ResponseWriter, r *http.Request) {
	id, e := primitives.IntParam(chi.URLParam(r, "id"))
	if e != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	u, _ := common.CurrentUser(r.Context())
	if e = a.labelPresetWorkflow.Delete(r.Context(), service.UserActor{Username: u.Username, IP: common.ClientIP(r)}, id); e != nil {
		if errors.Is(e, sql.ErrNoRows) {
			common.WriteError(w, http.StatusNotFound, "label preset not found")
		} else {
			common.WriteError(w, http.StatusInternalServerError, e.Error())
		}
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

type labelPreviewRequest = domain.LabelPreviewRequest

func (a *Router) handlePreviewLabels(w http.ResponseWriter, r *http.Request) {
	var req labelPreviewRequest
	if e := json.NewDecoder(r.Body).Decode(&req); e != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	out, e := a.labelWorkflow.Preview(r.Context(), req)
	if e != nil {
		common.WriteError(w, http.StatusInternalServerError, e.Error())
		return
	}
	if u, uerr := common.CurrentUser(r.Context()); uerr == nil {
		preset := strings.TrimSpace(req.PresetName)
		if preset == "" {
			preset = "-"
		}
		a.auditService().RecordEvent(r.Context(), u.Username, common.ClientIP(r), service.AuditEvent{
			Module: service.AuditModuleLabels,
			Action: "生成标签预览",
			Target: preset,
			Detail: fmt.Sprintf("使用标签预设 %s 生成 %d 张标签预览", strings.TrimSpace(req.PresetName), len(out)),
			Result: service.AuditResultSuccess,
		})
	}
	common.WriteJSON(w, http.StatusOK, out)
}

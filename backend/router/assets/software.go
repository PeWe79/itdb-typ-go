package assets

import (
	"encoding/json"
	"itdb-backend/internal/common/primitives"
	"itdb-backend/pkg/database"
	"itdb-backend/router/common"
	"net/http"

	"itdb-backend/internal/service"

	"github.com/go-chi/chi/v5"
)

func (a *Router) handleListSoftware(w http.ResponseWriter, r *http.Request) {
	limit := primitives.ListLimitParam(r.URL.Query().Get("limit"), 50)
	offset := primitives.IntParamDefault(r.URL.Query().Get("offset"), 0)
	rows, err := a.softwareWorkflow.List(r.Context(), r.URL.Query().Get("search"), limit, offset)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	result, err := database.RowsToMaps(rows)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.WriteJSON(w, http.StatusOK, result)
}
func (a *Router) handleGetSoftware(w http.ResponseWriter, r *http.Request) {
	id, err := primitives.IntParam(chi.URLParam(r, "id"))
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	softwareRows, err := a.softwareWorkflow.Get(r.Context(), id)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer softwareRows.Close()
	rows, err := database.RowsToMaps(softwareRows)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(rows) == 0 {
		common.WriteError(w, http.StatusNotFound, "software not found")
		return
	}
	payload := rows[0]
	relations, err := a.detailRelations.Load(r.Context(), "software", id)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for key, value := range relations {
		payload[key] = value
	}
	common.WriteJSON(w, http.StatusOK, payload)
}

func (a *Router) handleCreateSoftware(w http.ResponseWriter, r *http.Request) {
	var req softwarePayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	user, _ := common.CurrentUser(r.Context())
	id, err := a.softwareMutationWorkflow.Create(r.Context(), service.UserActor{Username: user.Username, IP: common.ClientIP(r)}, req)
	if err != nil {
		writeSoftwareMutationError(w, err)
		return
	}
	common.WriteJSON(w, http.StatusCreated, map[string]int64{"id": id})
}
func (a *Router) handleUpdateSoftware(w http.ResponseWriter, r *http.Request) {
	id, err := primitives.IntParam(chi.URLParam(r, "id"))
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req softwarePayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	user, _ := common.CurrentUser(r.Context())
	if err := a.softwareMutationWorkflow.Update(r.Context(), service.UserActor{Username: user.Username, IP: common.ClientIP(r)}, id, req); err != nil {
		writeSoftwareMutationError(w, err)
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]int64{"id": id})
}
func (a *Router) handleDeleteSoftware(w http.ResponseWriter, r *http.Request) {
	id, err := primitives.IntParam(chi.URLParam(r, "id"))
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	user, _ := common.CurrentUser(r.Context())
	if err := a.softwareMutationWorkflow.Delete(r.Context(), service.UserActor{Username: user.Username, IP: common.ClientIP(r)}, id); err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
func writeSoftwareMutationError(w http.ResponseWriter, err error) {
	if conflict := newServiceConflict(err); conflict != "" {
		common.WriteError(w, http.StatusConflict, conflict)
		return
	}
	if err.Error() == "title, version and manufacturerId are required" || err.Error() == "invalid purchaseDate" || err.Error() == "licenseQty cannot be negative" {
		common.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	common.WriteError(w, http.StatusInternalServerError, err.Error())
}

func (a *Router) handleMutateSoftwareTag(w http.ResponseWriter, r *http.Request) {
	id, err := primitives.IntParam(chi.URLParam(r, "id"))
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid software id")
		return
	}
	var req tagMutationPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	user, _ := common.CurrentUser(r.Context())
	if err := a.softwareTagWorkflow.Mutate(r.Context(), service.UserActor{Username: user.Username, IP: common.ClientIP(r)}, id, req.Name, req.Action); err != nil {
		if err.Error() == "name is required" || err.Error() == "action must be add or remove" {
			common.WriteError(w, http.StatusBadRequest, err.Error())
		} else {
			common.WriteError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

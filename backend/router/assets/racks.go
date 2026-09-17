package assets

import (
	"database/sql"
	"encoding/json"
	"errors"
	"itdb-backend/internal/common/primitives"
	"itdb-backend/pkg/database"
	"itdb-backend/router/common"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"itdb-backend/internal/service"
)

func (a *Router) handleListRacks(w http.ResponseWriter, r *http.Request) {
	rows, e := a.rackWorkflow.List(r.Context(), r.URL.Query().Get("search"))
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
func (a *Router) handleGetRack(w http.ResponseWriter, r *http.Request) {
	id, e := primitives.IntParam(chi.URLParam(r, "id"))
	if e != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	rr, e := a.rackWorkflow.Get(r.Context(), id)
	if e != nil {
		common.WriteError(w, http.StatusInternalServerError, e.Error())
		return
	}
	defer rr.Close()
	out, e := database.RowsToMaps(rr)
	if e != nil {
		common.WriteError(w, http.StatusInternalServerError, e.Error())
		return
	}
	if len(out) == 0 {
		common.WriteError(w, http.StatusNotFound, "rack not found")
		return
	}
	common.WriteJSON(w, http.StatusOK, out[0])
}
func (a *Router) handleCreateRack(w http.ResponseWriter, r *http.Request) {
	var req rackPayload
	if e := json.NewDecoder(r.Body).Decode(&req); e != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	u, _ := common.CurrentUser(r.Context())
	id, e := a.rackWorkflow.Create(r.Context(), service.UserActor{Username: u.Username, IP: common.ClientIP(r)}, req)
	if e != nil {
		writeRackError(w, e)
		return
	}
	common.WriteJSON(w, http.StatusCreated, map[string]int64{"id": id})
}
func (a *Router) handleUpdateRack(w http.ResponseWriter, r *http.Request) {
	id, e := primitives.IntParam(chi.URLParam(r, "id"))
	if e != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req rackPayload
	if e = json.NewDecoder(r.Body).Decode(&req); e != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	u, _ := common.CurrentUser(r.Context())
	if e = a.rackWorkflow.Update(r.Context(), service.UserActor{Username: u.Username, IP: common.ClientIP(r)}, id, req); e != nil {
		writeRackError(w, e)
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]int64{"id": id})
}
func (a *Router) handleDeleteRack(w http.ResponseWriter, r *http.Request) {
	id, e := primitives.IntParam(chi.URLParam(r, "id"))
	if e != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	u, _ := common.CurrentUser(r.Context())
	if e = a.rackWorkflow.Delete(r.Context(), service.UserActor{Username: u.Username, IP: common.ClientIP(r)}, id); e != nil {
		if conflict := newServiceConflict(e); conflict != "" {
			common.WriteError(w, http.StatusConflict, conflict)
		} else {
			common.WriteError(w, http.StatusInternalServerError, e.Error())
		}
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
func writeRackError(w http.ResponseWriter, e error) {
	if e.Error() == "uSize and depth are required" || errors.Is(e, sql.ErrNoRows) || strings.Contains(e.Error(), "超出机架边界") {
		common.WriteError(w, http.StatusBadRequest, e.Error())
		return
	}
	common.WriteError(w, http.StatusInternalServerError, e.Error())
}

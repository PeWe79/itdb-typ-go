package assets

import (
	"database/sql"
	"encoding/json"
	"itdb-backend/internal/common/primitives"
	"itdb-backend/pkg/database"
	"itdb-backend/router/common"
	"net/http"

	"errors"
	"itdb-backend/internal/service"

	"github.com/go-chi/chi/v5"
)

func (a *Router) handleListContracts(w http.ResponseWriter, r *http.Request) {
	rows, e := a.contractWorkflow.List(r.Context(), r.URL.Query().Get("search"))
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
func (a *Router) handleGetContract(w http.ResponseWriter, r *http.Request) {
	id, e := primitives.IntParam(chi.URLParam(r, "id"))
	if e != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	cr, e := a.contractWorkflow.Get(r.Context(), id)
	if e != nil {
		common.WriteError(w, http.StatusInternalServerError, e.Error())
		return
	}
	defer cr.Close()
	rows, e := database.RowsToMaps(cr)
	if e != nil {
		common.WriteError(w, http.StatusInternalServerError, e.Error())
		return
	}
	if len(rows) == 0 {
		common.WriteError(w, http.StatusNotFound, "contract not found")
		return
	}
	p := rows[0]
	relations, err := a.detailRelations.Load(r.Context(), "contracts", id)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for key, value := range relations {
		p[key] = value
	}
	common.WriteJSON(w, http.StatusOK, p)
}
func (a *Router) handleCreateContract(w http.ResponseWriter, r *http.Request) {
	var req contractPayload
	if e := json.NewDecoder(r.Body).Decode(&req); e != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	u, _ := common.CurrentUser(r.Context())
	id, e := a.contractWorkflow.Create(r.Context(), service.UserActor{Username: u.Username, IP: common.ClientIP(r)}, req)
	if e != nil {
		writeContractError(w, e)
		return
	}
	common.WriteJSON(w, http.StatusCreated, map[string]int64{"id": id})
}
func (a *Router) handleUpdateContract(w http.ResponseWriter, r *http.Request) {
	id, e := primitives.IntParam(chi.URLParam(r, "id"))
	if e != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req contractPayload
	if e = json.NewDecoder(r.Body).Decode(&req); e != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	u, _ := common.CurrentUser(r.Context())
	if e = a.contractWorkflow.Update(r.Context(), service.UserActor{Username: u.Username, IP: common.ClientIP(r)}, id, req); e != nil {
		writeContractError(w, e)
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]int64{"id": id})
}
func (a *Router) handleDeleteContract(w http.ResponseWriter, r *http.Request) {
	id, e := primitives.IntParam(chi.URLParam(r, "id"))
	if e != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	u, _ := common.CurrentUser(r.Context())
	if e = a.contractWorkflow.Delete(r.Context(), service.UserActor{Username: u.Username, IP: common.ClientIP(r)}, id); e != nil {
		var conflict *service.ConflictError
		if errors.As(e, &conflict) {
			common.WriteError(w, http.StatusConflict, conflict.Message)
		} else {
			common.WriteError(w, http.StatusInternalServerError, e.Error())
		}
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
func writeContractError(w http.ResponseWriter, e error) {
	var conflict *service.ConflictError
	if errors.As(e, &conflict) {
		common.WriteError(w, http.StatusConflict, conflict.Message)
		return
	}
	if e.Error() == "missing mandatory fields" || e.Error() == "invalid startDate" || e.Error() == "invalid currentEndDate" {
		common.WriteError(w, http.StatusBadRequest, e.Error())
		return
	}
	common.WriteError(w, http.StatusInternalServerError, e.Error())
}

func (a *Router) handleNextContractEventID(w http.ResponseWriter, r *http.Request) {
	nextID, e := a.contractEventWorkflow.NextEventID(r.Context())
	if e != nil {
		common.WriteError(w, http.StatusInternalServerError, e.Error())
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]int64{"nextId": nextID})
}
func (a *Router) handleListContractEvents(w http.ResponseWriter, r *http.Request) {
	id, e := primitives.IntParam(chi.URLParam(r, "id"))
	if e != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid contract id")
		return
	}
	rows, e := a.contractEventWorkflow.List(r.Context(), id)
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
func (a *Router) handleCreateContractEvent(w http.ResponseWriter, r *http.Request) {
	id, e := primitives.IntParam(chi.URLParam(r, "id"))
	if e != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid contract id")
		return
	}
	var req contractEventPayload
	if e = json.NewDecoder(r.Body).Decode(&req); e != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	u, _ := common.CurrentUser(r.Context())
	eventID, e := a.contractEventWorkflow.Create(r.Context(), service.UserActor{Username: u.Username, IP: common.ClientIP(r)}, id, req)
	if e != nil {
		writeContractEventError(w, e)
		return
	}
	common.WriteJSON(w, http.StatusCreated, map[string]int64{"id": eventID})
}
func (a *Router) handleUpdateContractEvent(w http.ResponseWriter, r *http.Request) {
	eventID, e := primitives.IntParam(chi.URLParam(r, "eventId"))
	if e != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid event id")
		return
	}
	var req contractEventPayload
	if e = json.NewDecoder(r.Body).Decode(&req); e != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	u, _ := common.CurrentUser(r.Context())
	if e = a.contractEventWorkflow.Update(r.Context(), service.UserActor{Username: u.Username, IP: common.ClientIP(r)}, eventID, req); e != nil {
		if errors.Is(e, sql.ErrNoRows) {
			common.WriteError(w, http.StatusNotFound, "event not found")
		} else {
			writeContractEventError(w, e)
		}
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]int64{"id": eventID})
}
func (a *Router) handleDeleteContractEvent(w http.ResponseWriter, r *http.Request) {
	eventID, e := primitives.IntParam(chi.URLParam(r, "eventId"))
	if e != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid event id")
		return
	}
	u, _ := common.CurrentUser(r.Context())
	if e = a.contractEventWorkflow.Delete(r.Context(), service.UserActor{Username: u.Username, IP: common.ClientIP(r)}, eventID); e != nil {
		if errors.Is(e, sql.ErrNoRows) {
			common.WriteError(w, http.StatusNotFound, "event not found")
		} else {
			common.WriteError(w, http.StatusInternalServerError, e.Error())
		}
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
func writeContractEventError(w http.ResponseWriter, e error) {
	if e.Error() == "invalid startDate" || e.Error() == "invalid endDate" || e.Error() == "description is required" {
		common.WriteError(w, http.StatusBadRequest, e.Error())
		return
	}
	common.WriteError(w, http.StatusInternalServerError, e.Error())
}

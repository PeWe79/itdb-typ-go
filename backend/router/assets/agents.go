package assets

import (
	"database/sql"
	"encoding/json"
	"errors"
	"itdb-backend/internal/common/primitives"
	"itdb-backend/pkg/database"
	"itdb-backend/router/common"
	"net/http"

	"itdb-backend/internal/domain"
	"itdb-backend/internal/service"

	"github.com/go-chi/chi/v5"
)

func (a *Router) handleListAgents(w http.ResponseWriter, r *http.Request) {
	rows, err := a.agentWorkflow.List(r.Context(), r.URL.Query().Get("search"))
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
func (a *Router) handleGetAgent(w http.ResponseWriter, r *http.Request) {
	id, err := primitives.IntParam(chi.URLParam(r, "id"))
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	rows, err := a.agentWorkflow.Get(r.Context(), id)
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
	if len(result) == 0 {
		common.WriteError(w, http.StatusNotFound, "agent not found")
		return
	}
	common.WriteJSON(w, http.StatusOK, result[0])
}
func (a *Router) handleCreateAgent(w http.ResponseWriter, r *http.Request) {
	var req domain.AgentPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	user, _ := common.CurrentUser(r.Context())
	id, err := a.agentWorkflow.Create(r.Context(), service.UserActor{Username: user.Username, IP: common.ClientIP(r)}, req)
	if err != nil {
		var conflict *service.ConflictError
		switch {
		case errors.As(err, &conflict):
			common.WriteError(w, http.StatusConflict, conflict.Message)
		case err.Error() == "title is required":
			common.WriteError(w, http.StatusBadRequest, err.Error())
		default:
			common.WriteError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	common.WriteJSON(w, http.StatusCreated, map[string]int64{"id": id})
}
func (a *Router) handleUpdateAgent(w http.ResponseWriter, r *http.Request) {
	id, err := primitives.IntParam(chi.URLParam(r, "id"))
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req domain.AgentPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	user, _ := common.CurrentUser(r.Context())
	if err := a.agentWorkflow.Update(r.Context(), service.UserActor{Username: user.Username, IP: common.ClientIP(r)}, id, req); err != nil {
		var conflict *service.ConflictError
		switch {
		case errors.As(err, &conflict):
			common.WriteError(w, http.StatusConflict, conflict.Message)
		case err.Error() == "title is required":
			common.WriteError(w, http.StatusBadRequest, err.Error())
		default:
			common.WriteError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]int64{"id": id})
}
func (a *Router) handleDeleteAgent(w http.ResponseWriter, r *http.Request) {
	id, err := primitives.IntParam(chi.URLParam(r, "id"))
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	user, _ := common.CurrentUser(r.Context())
	if err := a.agentWorkflow.Delete(r.Context(), service.UserActor{Username: user.Username, IP: common.ClientIP(r)}, id); err != nil {
		var conflict *service.ConflictError
		switch {
		case errors.As(err, &conflict):
			common.WriteError(w, http.StatusConflict, conflict.Message)
		case errors.Is(err, sql.ErrNoRows):
			common.WriteError(w, http.StatusNotFound, "agent not found")
		default:
			common.WriteError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

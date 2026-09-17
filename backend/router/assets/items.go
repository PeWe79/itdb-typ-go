package assets

import (
	"encoding/json"
	"itdb-backend/internal/common/primitives"
	"itdb-backend/pkg/database"
	"itdb-backend/router/common"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"itdb-backend/internal/service"
)

func (a *Router) handleListItems(w http.ResponseWriter, r *http.Request) {
	limit := primitives.ListLimitParam(r.URL.Query().Get("limit"), 50)
	offset := primitives.IntParamDefault(r.URL.Query().Get("offset"), 0)
	rows, err := a.itemWorkflow.List(r.Context(), r.URL.Query().Get("search"), limit, offset)
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

func (a *Router) handleGetItem(w http.ResponseWriter, r *http.Request) {
	id, err := primitives.IntParam(chi.URLParam(r, "id"))
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	itemRows, err := a.itemWorkflow.Get(r.Context(), id)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer itemRows.Close()
	rows, err := database.RowsToMaps(itemRows)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(rows) == 0 {
		common.WriteError(w, http.StatusNotFound, "item not found")
		return
	}
	payload := rows[0]
	relations, err := a.detailRelations.Load(r.Context(), "items", id)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for key, value := range relations {
		payload[key] = value
	}
	common.WriteJSON(w, http.StatusOK, payload)
}

func (a *Router) handleCreateItem(w http.ResponseWriter, r *http.Request) {
	var req itemPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	user, _ := common.CurrentUser(r.Context())
	id, err := a.itemMutationWorkflow.Create(r.Context(), service.UserActor{Username: user.Username, IP: common.ClientIP(r)}, req)
	if err != nil {
		writeItemMutationError(w, err)
		return
	}
	common.WriteJSON(w, http.StatusCreated, map[string]int64{"id": id})
}
func (a *Router) handleUpdateItem(w http.ResponseWriter, r *http.Request) {
	id, err := primitives.IntParam(chi.URLParam(r, "id"))
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req itemPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	user, _ := common.CurrentUser(r.Context())
	if err := a.itemMutationWorkflow.Update(r.Context(), service.UserActor{Username: user.Username, IP: common.ClientIP(r)}, id, req); err != nil {
		writeItemMutationError(w, err)
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]int64{"id": id})
}
func (a *Router) handleDeleteItem(w http.ResponseWriter, r *http.Request) {
	id, err := primitives.IntParam(chi.URLParam(r, "id"))
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	user, _ := common.CurrentUser(r.Context())
	if err := a.itemMutationWorkflow.Delete(r.Context(), service.UserActor{Username: user.Username, IP: common.ClientIP(r)}, id); err != nil {
		writeItemMutationError(w, err)
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
func writeItemMutationError(w http.ResponseWriter, err error) {
	if conflict := newServiceConflict(err); conflict != "" {
		common.WriteError(w, http.StatusConflict, conflict)
		return
	}
	if strings.HasSuffix(err.Error(), "is required") || err.Error() == "invalid purchaseDate" || err.Error() == "invalid ports" {
		common.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	common.WriteError(w, http.StatusInternalServerError, err.Error())
}

func (a *Router) handleMutateItemTag(w http.ResponseWriter, r *http.Request) {
	itemID, err := primitives.IntParam(chi.URLParam(r, "id"))
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid item id")
		return
	}
	var req tagMutationPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	user, _ := common.CurrentUser(r.Context())
	if err := a.itemTagWorkflow.Mutate(r.Context(), service.UserActor{Username: user.Username, IP: common.ClientIP(r)}, itemID, req.Name, req.Action); err != nil {
		if err.Error() == "name is required" || err.Error() == "action must be add or remove" {
			common.WriteError(w, http.StatusBadRequest, err.Error())
		} else {
			common.WriteError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

package assets

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"itdb-backend/internal/common/primitives"
	"itdb-backend/pkg/database"
	"itdb-backend/router/common"
)

func (a *Router) handleListItemActions(w http.ResponseWriter, r *http.Request) {
	itemID, err := primitives.IntParam(chi.URLParam(r, "id"))
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid item id")
		return
	}
	rows, err := a.itemActionWorkflow.List(r.Context(), itemID)
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

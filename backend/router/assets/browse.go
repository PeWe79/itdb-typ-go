package assets

import (
	"itdb-backend/internal/domain"
	"itdb-backend/router/common"
	"net/http"
)

type browseNode = domain.BrowseNode

func (a *Router) handleBrowseTree(w http.ResponseWriter, r *http.Request) {
	out, e := a.browseWorkflow.List(r.Context(), r.URL.Query().Get("id"))
	if e != nil {
		if e.Error() == "invalid browse node id" {
			common.WriteError(w, http.StatusBadRequest, e.Error())
		} else {
			common.WriteError(w, http.StatusInternalServerError, e.Error())
		}
		return
	}
	common.WriteJSON(w, http.StatusOK, out)
}

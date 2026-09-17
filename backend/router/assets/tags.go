package assets

import (
	"itdb-backend/internal/common/primitives"
	"itdb-backend/router/common"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (a *Router) handleNextTagID(w http.ResponseWriter, r *http.Request) {
	nextID, err := a.tagWorkflow.NextTagID(r.Context())
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]int64{"nextId": nextID})
}

func (a *Router) handleListTagItems(w http.ResponseWriter, r *http.Request) {
	tagID, err := primitives.IntParam(chi.URLParam(r, "id"))
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid tag id")
		return
	}
	rows, err := a.fetchRows(`SELECT items.id, agents.title || ' ' || items.model || ' [' || itemtypes.typedesc || ', ID:' || items.id || ']' AS txt
        FROM agents, items, itemtypes
        WHERE agents.id = items.manufacturerid
          AND items.itemtypeid = itemtypes.id
          AND items.id IN (SELECT itemid FROM tag2item WHERE tagid = ?)`, tagID)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.WriteJSON(w, http.StatusOK, rows)
}

func (a *Router) handleListTagSoftware(w http.ResponseWriter, r *http.Request) {
	tagID, err := primitives.IntParam(chi.URLParam(r, "id"))
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid tag id")
		return
	}
	rows, err := a.fetchRows(`SELECT software.id, agents.title || ' ' || software.stitle || ' ' || software.sversion || ' [ID:' || software.id || ']' AS txt
        FROM agents, software
        WHERE agents.id = software.manufacturerid
          AND software.id IN (SELECT softwareid FROM tag2software WHERE tagid = ?)`, tagID)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.WriteJSON(w, http.StatusOK, rows)
}

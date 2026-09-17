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

func (a *Router) handleListInvoices(w http.ResponseWriter, r *http.Request) {
	limit := primitives.ListLimitParam(r.URL.Query().Get("limit"), 50)
	offset := primitives.IntParamDefault(r.URL.Query().Get("offset"), 0)
	rows, e := a.invoiceWorkflow.List(r.Context(), r.URL.Query().Get("search"), limit, offset)
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
func (a *Router) handleGetInvoice(w http.ResponseWriter, r *http.Request) {
	id, e := primitives.IntParam(chi.URLParam(r, "id"))
	if e != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	ir, e := a.invoiceWorkflow.Get(r.Context(), id)
	if e != nil {
		common.WriteError(w, http.StatusInternalServerError, e.Error())
		return
	}
	defer ir.Close()
	rows, e := database.RowsToMaps(ir)
	if e != nil {
		common.WriteError(w, http.StatusInternalServerError, e.Error())
		return
	}
	if len(rows) == 0 {
		common.WriteError(w, http.StatusNotFound, "invoice not found")
		return
	}
	p := rows[0]
	relations, err := a.detailRelations.Load(r.Context(), "invoices", id)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for key, value := range relations {
		p[key] = value
	}
	common.WriteJSON(w, http.StatusOK, p)
}

func (a *Router) handleCreateInvoice(w http.ResponseWriter, r *http.Request) {
	var req invoicePayload
	if e := json.NewDecoder(r.Body).Decode(&req); e != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	u, _ := common.CurrentUser(r.Context())
	id, e := a.invoiceWorkflow.Create(r.Context(), service.UserActor{Username: u.Username, IP: common.ClientIP(r)}, req)
	if e != nil {
		writeInvoiceError(w, e)
		return
	}
	common.WriteJSON(w, http.StatusCreated, map[string]int64{"id": id})
}
func (a *Router) handleUpdateInvoice(w http.ResponseWriter, r *http.Request) {
	id, e := primitives.IntParam(chi.URLParam(r, "id"))
	if e != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req invoicePayload
	if e = json.NewDecoder(r.Body).Decode(&req); e != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	u, _ := common.CurrentUser(r.Context())
	if e = a.invoiceWorkflow.Update(r.Context(), service.UserActor{Username: u.Username, IP: common.ClientIP(r)}, id, req); e != nil {
		writeInvoiceError(w, e)
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]int64{"id": id})
}
func (a *Router) handleDeleteInvoice(w http.ResponseWriter, r *http.Request) {
	id, e := primitives.IntParam(chi.URLParam(r, "id"))
	if e != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	u, _ := common.CurrentUser(r.Context())
	if e = a.invoiceWorkflow.Delete(r.Context(), service.UserActor{Username: u.Username, IP: common.ClientIP(r)}, id); e != nil {
		common.WriteError(w, http.StatusInternalServerError, e.Error())
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
func writeInvoiceError(w http.ResponseWriter, e error) {
	if e.Error() == "vendorId, buyerId, number and date are required" || e.Error() == "invalid date" {
		common.WriteError(w, http.StatusBadRequest, e.Error())
		return
	}
	common.WriteError(w, http.StatusInternalServerError, e.Error())
}

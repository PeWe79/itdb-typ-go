package assets

import (
	"github.com/go-chi/chi/v5"
	"itdb-backend/internal/common/primitives"
	"itdb-backend/internal/domain"
	"itdb-backend/internal/service"
	"itdb-backend/pkg/database"
	"itdb-backend/router/common"
	"net/http"
	"strings"
)

type reportDefinition = domain.ReportDefinition

func (a *Router) handleListReports(w http.ResponseWriter, r *http.Request) {
	common.WriteJSON(w, http.StatusOK, a.reportWorkflow.List())
}
func (a *Router) handleRunReport(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(chi.URLParam(r, "name"))
	limit := primitives.IntParamDefault(r.URL.Query().Get("limit"), 1000)
	report, rows, e := a.reportWorkflow.Run(r.Context(), name, limit)
	if e != nil {
		if e.Error() == "report not found" {
			common.WriteError(w, http.StatusNotFound, e.Error())
		} else {
			common.WriteError(w, http.StatusInternalServerError, e.Error())
		}
		return
	}
	defer rows.Close()
	data, e := database.RowsToMaps(rows)
	if e != nil {
		common.WriteError(w, http.StatusInternalServerError, e.Error())
		return
	}
	chart := []map[string]interface{}{}
	if report.ChartType == "pie" && report.ChartX != "" && report.ChartY != "" {
		max := report.ChartLimit
		if max <= 0 {
			max = 15
		}
		for _, row := range data {
			if len(chart) >= max {
				break
			}
			if strings.EqualFold(primitives.AsString(row[report.ChartX]), "Total") {
				continue
			}
			chart = append(chart, map[string]interface{}{"x": primitives.AsString(row[report.ChartX]), "y": primitives.AsInt64(row[report.ChartY])})
		}
	}
	common.WriteJSON(w, http.StatusOK, map[string]interface{}{"meta": map[string]interface{}{"name": report.Name, "description": report.Description, "chartType": report.ChartType, "chartX": report.ChartX, "chartY": report.ChartY, "chartLimit": report.ChartLimit}, "rows": data, "chart": chart})
}

var _ *service.ReportWorkflow

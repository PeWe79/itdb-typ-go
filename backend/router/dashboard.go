package router

import (
	"itdb-backend/router/common"
	"net/http"
)

func (a *App) handleDashboardSummary(w http.ResponseWriter, r *http.Request) {
	stats := map[string]int64{}
	tables := []string{"items", "software", "invoices", "contracts", "files", "agents", "users", "locations", "racks"}

	for _, table := range tables {
		query := "SELECT COUNT(*) FROM " + table
		var count int64
		if err := a.domains.System.QueryRow(query).Scan(&count); err != nil {
			common.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		stats[table] = count
	}

	common.WriteJSON(w, http.StatusOK, map[string]interface{}{"counts": stats})
}

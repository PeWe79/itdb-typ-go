package router

import (
	"context"

	"itdb-backend/pkg/database"
)

func (a *App) fetchRows(query string, args ...interface{}) ([]map[string]interface{}, error) {
	if a.sql != nil {
		return a.queries.QueryRows(context.Background(), query, args...)
	}
	rows, err := a.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return database.RowsToMaps(rows)
}
